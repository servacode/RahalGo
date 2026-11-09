package orders

// ══════════════════════════════════════════════════════════════════════
// **الكابتن يطلب تعديل ثمن البضاعة — والزبون يوافق أو يرفض** (قرار المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// بطلب المشتريات، بعد ما يشتري الكابتن ويستلم، ممكن يطلع السعر الحقيقي غير اللي انكتب.
// **كان الكابتن ممنوع يعدّل بعد الاستلام** (`custom_locked`) والمكتب وحده بيعدّل
// (`AdminOverrideCustomQuote`). صار الكابتن يطلب، **والزبون يقرّر**:
//
//	طلب الكابتن  ←  نسخة سعر جديدة بانتظار الزبون (`applyCustomQuoteTx` بعلَم needApproval)،
//	                 والمؤكَّد القديم بحجزه و`cash_due` تبعه ما بيتحرّكوا، والتسليم موقوف
//	                 (`ErrQuoteReconfirmPending` بـ`transitions.go`).
//	موافقة       ←  `confirm-quote` القائم نفسه (التطبيقات القديمة شغّالة) — حجز/مستحق جديد.
//	رفض          ←  `reject-quote` القائم: **السعر القديم برجع** والكابتن بيسلّم فيه، والمكتب
//	                 والكابتن بينخبروا، وعلامة الرفض بتضل سطر أحمر باللوحة.
//	مهلة         ←  الراصد بينذر المكتب مرّة وحدة (`SweepPriceRequestsOnce`).
//
// **والدفتر ما بينمس هون أبداً** — التسوية القائمة وقت التسليم بتقيّد المبلغ المؤكَّد.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrPriceRequestStage **مو وقته أو مو نوعه** — بس مشتريات، بعد الاستلام وقبل النهاية،
	// وثمنها موثّق ومؤكَّد.
	ErrPriceRequestStage = httpx.NewError(http.StatusConflict,
		"price_request_stage", "errors.price_request_stage")
	// ErrPriceRequestPending **في سعر جديد لسا بانتظار الزبون** — طلب واحد بس بنفس الوقت.
	ErrPriceRequestPending = httpx.NewError(http.StatusConflict,
		"price_request_pending", "errors.price_request_pending")
	// ErrPriceRequestSame **نفس الثمن الحالي** — ما في شي يتعدّل.
	ErrPriceRequestSame = httpx.NewError(http.StatusUnprocessableEntity,
		"price_request_same", "errors.price_request_same")
	// ErrPriceRequestOff **الإعداد مقفول** (النسبة صفر) — المكتب بيعدّل.
	ErrPriceRequestOff = httpx.NewError(http.StatusConflict,
		"price_request_off", "errors.price_request_off")
	// ErrPriceRequestCurrency **عملة الطلب غير عملة الثمن المؤكَّد** (٢٠٢٦-١١-٠٩) — الكابتن بيعدّل
	// المبلغ بنفس العملة، وتبديل العملة بعد الاستلام للمكتب.
	ErrPriceRequestCurrency = httpx.NewError(http.StatusConflict,
		"price_request_currency", "errors.price_request_currency")
	// ErrPriceRequestReason **السبب إلزامي** — الزبون لازم يعرف ليش.
	ErrPriceRequestReason = httpx.NewError(http.StatusBadRequest,
		"price_request_reason", "errors.price_request_reason")
)

// ErrPriceRequestTooFar **التغيير أكبر من السقف** — والنسبة بالتفصيل ليقولها التطبيق.
func ErrPriceRequestTooFar(pct int64) error {
	return &httpx.AppError{
		Status: http.StatusUnprocessableEntity, Code: "price_request_too_far",
		MessageKey: "errors.price_request_too_far",
		Details:    map[string]any{"max_percent": strconv.FormatInt(pct, 10)},
	}
}

// DriverPriceEditDefaultPercent وDriverPriceEditDefaultTimeout **افتراضا الإعدادين** —
// لخدمة بلا إعدادات (اختبار) وللكتالوج نفسه.
const (
	DriverPriceEditDefaultPercent int64 = 50
	DriverPriceEditDefaultTimeout int64 = 10
)

const priceRequestReasonMax = 200

func (s *Service) driverPriceEditPercent(ctx context.Context) int64 {
	if s.settings == nil {
		return DriverPriceEditDefaultPercent
	}
	return s.settings.GetInt(ctx, "delivery.driver_price_edit_max_percent")
}

func (s *Service) driverPriceEditTimeout(ctx context.Context) int64 {
	if s.settings == nil {
		return DriverPriceEditDefaultTimeout
	}
	if n := s.settings.GetInt(ctx, "delivery.driver_price_edit_timeout_min"); n > 0 {
		return n
	}
	return DriverPriceEditDefaultTimeout
}

// RequestDriverPrice **الكابتن يطلب ثمن بضاعة جديد** — بعد الاستلام، والزبون يوافق.
func (s *Service) RequestDriverPrice(ctx context.Context, orderID, driverID string,
	goods int64, reason string) (*Order, error) {
	return s.RequestDriverPriceCurrency(ctx, orderID, driverID, goods, "", reason)
}

// RequestDriverPriceCurrency **وبعملة الثمن** (٢٠٢٦-١١-٠٩) — `goods` بالسنتات للدولار، و«» ليرة
// (التطبيق القديم). **ولازم تطابق عملة الثمن المؤكَّد** (`ErrPriceRequestCurrency`).
func (s *Service) RequestDriverPriceCurrency(ctx context.Context, orderID, driverID string,
	goods int64, currency, reason string) (*Order, error) {
	if currency == "" {
		currency = CurrencySYP
	}
	if currency != CurrencySYP && currency != CurrencyUSD {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, ErrPriceRequestReason
	}
	if r := []rune(reason); len(r) > priceRequestReasonMax {
		reason = string(r[:priceRequestReasonMax])
	}
	if goods < 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if currency == CurrencySYP && goods > CustomGoodsMax {
		return nil, ErrCustomGoodsTooHigh()
	}
	if currency == CurrencyUSD && goods > CustomGoodsUSDMax {
		return nil, ErrCustomGoodsUSDTooHigh()
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := s.lockCustomRow(ctx, tx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if row.kind != KindCustom {
		return nil, ErrNotCustom
	}
	if row.driverID == nil || *row.driverID != driverID {
		return nil, httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	var mode string
	var goodsPending bool
	if err := tx.QueryRow(ctx, `SELECT custom_mode, custom_goods_pending FROM orders WHERE id = $1`,
		orderID).Scan(&mode, &goodsPending); err != nil {
		return nil, err
	}
	// **مشتريات بس، بعد الاستلام وقبل النهاية، وثمنها موثّق ومؤكَّد** — قبل الاستلام بابه
	// «وثّق ثمن البضاعة» القائم، والأمانة ما إلها ثمن بضاعة.
	if mode != CustomModePurchase || row.pickedUpAt == nil || terminal(row.status) ||
		!row.hasGoods || goodsPending || row.confirmedAt == nil || row.confirmedVersion == nil {
		return nil, ErrPriceRequestStage
	}
	// **طلب واحد بنفس الوقت** — نسخة بانتظار الزبون (منه أو من المكتب) بتسكّر الباب.
	if *row.confirmedVersion != row.quoteVersion {
		return nil, ErrPriceRequestPending
	}
	var open bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM order_price_requests
		WHERE order_id = $1 AND status = 'pending')`, orderID).Scan(&open); err != nil {
		return nil, err
	}
	if open {
		return nil, ErrPriceRequestPending
	}
	// **والمبلغ بعملة الثمن المؤكَّد** — والمقارنة والسقف بنفس العملة.
	if (row.goodsUSD != nil) != (currency == CurrencyUSD) {
		return nil, ErrPriceRequestCurrency
	}
	curGoods := row.goods
	var newUSD *int64
	if row.goodsUSD != nil {
		curGoods = *row.goodsUSD
		if goods <= 0 {
			return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
		}
		newUSD = &goods
	}
	if goods == curGoods {
		return nil, ErrPriceRequestSame
	}
	// **السقف بالنسبة من الثمن المؤكَّد** — والصفر بيقفل الباب.
	pct := s.driverPriceEditPercent(ctx)
	if pct <= 0 {
		return nil, ErrPriceRequestOff
	}
	diff := goods - curGoods
	if diff < 0 {
		diff = -diff
	}
	if diff*100 > pct*curGoods {
		return nil, ErrPriceRequestTooFar(pct)
	}

	oldTotal := row.goods + row.fee - row.waived
	if _, err := s.applyCustomQuoteTx(ctx, tx, row, goods, row.fee, newUSD, quoteMutator{
		actorID: driverID, role: "driver", source: "driver_price_request", reason: reason,
		needApproval: true,
	}); err != nil {
		return nil, err
	}
	var newTotal, version int64
	if err := tx.QueryRow(ctx, `SELECT total, quote_version FROM orders WHERE id = $1`, orderID).
		Scan(&newTotal, &version); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_price_requests (order_id, driver_id, old_goods, new_goods,
		                                  old_total, new_total, quote_version, reason)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		orderID, driverID, curGoods, goods, oldTotal, newTotal, version, reason); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	s.publishWalletsOf(ctx, orderID)
	line := fmt.Sprintf(
		"طلبتُ تعديل ثمن البضاعة من %s إلى %s ل.س — السبب: %s. المجموع الجديد %s ل.س، وافق أو ارفض.",
		groupDigits(row.goods), groupDigits(goods), reason, groupDigits(newTotal))
	if newUSD != nil {
		line = fmt.Sprintf(
			"طلبتُ تعديل ثمن البضاعة من %s إلى %s — السبب: %s. أجرة التوصيل %s ل.س، وافق أو ارفض.",
			AmountText(curGoods, CurrencyUSD), AmountText(goods, CurrencyUSD), reason, groupDigits(newTotal))
	}
	s.quoteLine(ctx, orderID, driverID, "driver", line)
	body := fmt.Sprintf("#%d — كان %s صار %s ل.س — السبب: %s",
		o.Number, groupDigits(oldTotal), groupDigits(newTotal), reason)
	if newUSD != nil {
		body = fmt.Sprintf("#%d — ثمن البضاعة كان %s صار %s — السبب: %s",
			o.Number, AmountText(curGoods, CurrencyUSD), AmountText(goods, CurrencyUSD), reason)
	}
	if s.notify != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: o.CustomerID, Kind: notifications.KindOrder,
			Title:  t.priceRequestCustomer,
			Body:   body,
			Entity: "order", EntityID: o.ID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		})
	}
	return o, nil
}

// pendingPriceRequest **طلب الكابتن المعلّق لهالنسخة** — والفاضي ما في.
type pendingPriceRequest struct {
	id                 string
	oldGoods, newGoods int64
	oldTotal, newTotal int64
	reason             string
}

func pendingPriceRequestTx(ctx context.Context, q wallet.Querier, orderID string,
	version int64) (*pendingPriceRequest, error) {
	var p pendingPriceRequest
	err := q.QueryRow(ctx, `
		SELECT id::text, old_goods, new_goods, old_total, new_total, reason
		  FROM order_price_requests
		 WHERE order_id = $1 AND status = 'pending' AND quote_version = $2
		 FOR UPDATE`, orderID, version).
		Scan(&p.id, &p.oldGoods, &p.newGoods, &p.oldTotal, &p.newTotal, &p.reason)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// closePriceRequestsTx **يسكّر طلبات الكابتن المعلّقة بحالة** — موافقة الزبون (`approved`)
// أو تعديل المكتب (`superseded`).
func closePriceRequestsTx(ctx context.Context, q wallet.Querier, orderID, status string) error {
	_, err := q.Exec(ctx, `
		UPDATE order_price_requests SET status = $2, decided_at = now()
		 WHERE order_id = $1 AND status = 'pending'`, orderID, status)
	return err
}

// revertPriceRequestTx **الزبون رفض طلب الكابتن ⇒ السعر القديم برجع** (قرار المالك ٢٠٢٦-١١-٠٦:
// «الرفض ما بيغيّر شي»).
//
// **نسخة جديدة مؤكَّدة بنفس المبلغ المؤكَّد** — الحجز و`cash_due` ما تحرّكوا أصلاً، فالسعر
// القديم برجع كامل والكابتن بيقدر يسلّم فيه. والأجرة ما تغيّرت (الطلب ما بيمسّها).
// **وإذا المؤكَّد ما بيطابق القديم** (ما لازم يصير) ما برجع شي ويضل السلوك القديم.
func revertPriceRequestTx(ctx context.Context, q wallet.Querier, row customQuoteRow,
	p *pendingPriceRequest) (bool, error) {
	if row.confirmedTotal == nil || *row.confirmedTotal != p.oldTotal {
		return false, nil
	}
	// **وثمنٌ بالدولار يرجع بعملته** (٢٠٢٦-١١-٠٩) — المجموع الأجرة وحدها فما بيتغيّر.
	if row.goodsUSD != nil {
		tag, err := q.Exec(ctx, `
			UPDATE orders SET custom_goods_usd = $2,
			       quote_version = quote_version + 1, quote_confirmed_version = quote_version + 1,
			       updated_at = now()
			 WHERE id = $1 AND custom_goods_usd IS NOT NULL AND total = $3`,
			row.id, p.oldGoods, p.oldTotal)
		if err != nil {
			return false, err
		}
		if tag.RowsAffected() != 1 {
			return false, nil
		}
		_, err = q.Exec(ctx, `
			UPDATE order_price_requests SET status = 'rejected', decided_at = now() WHERE id = $1`, p.id)
		return err == nil, err
	}
	tag, err := q.Exec(ctx, `
		UPDATE orders SET custom_goods_amount = $2, subtotal = $2,
		       total = $2::bigint + custom_fee - promo_delivery_waived,
		       quote_version = quote_version + 1, quote_confirmed_version = quote_version + 1,
		       updated_at = now()
		 WHERE id = $1 AND $2::bigint + custom_fee - promo_delivery_waived = $3`,
		row.id, p.oldGoods, p.oldTotal)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() != 1 {
		return false, nil
	}
	if _, err := q.Exec(ctx, `
		UPDATE order_price_requests SET status = 'rejected', decided_at = now() WHERE id = $1`,
		p.id); err != nil {
		return false, err
	}
	return true, nil
}

// notifyPriceRequestRejected **الرفض بيوصل المكتب ليتدخّل والكابتن ليعرف إنّ القديم رجع.**
func (s *Service) notifyPriceRequestRejected(ctx context.Context, o *Order, p *pendingPriceRequest) {
	if s.notify == nil || o == nil {
		return
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind:  notifications.KindOrder,
		Title: t.priceRequestRejectedOps,
		Body: fmt.Sprintf("#%d — الكابتن طلب %s بدل %s ل.س (السبب: %s) — رجع السعر القديم",
			o.Number, groupDigits(p.newTotal), groupDigits(p.oldTotal), p.reason),
		Entity: "order", EntityID: o.ID, Href: "/dashboard/orders",
	})
	if o.DriverID != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *o.DriverID, Kind: notifications.KindOrder,
			Title:  t.priceRequestRejectedDriver,
			Body:   fmt.Sprintf("#%d — المجموع %s ل.س", o.Number, groupDigits(o.Total)),
			Entity: "order", EntityID: o.ID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		})
	}
}

// SweepPriceRequestsOnce **جولة الراصد على طلبات الكابتن** — يسكّر طلبات الطلبات المنتهية،
// **وينذر المكتب مرّة وحدة** بكل طلب عدّى المهلة والزبون ما ردّ. بيرجّع عدد الإنذارات.
//
// **والعلامة قبل الإشعار** (`alerted_at` بشرط فاضي) — متل `escalate`: إشعار ضايع أهون من
// إشعار بيتكرّر كل نبضة.
func (s *Service) SweepPriceRequestsOnce(ctx context.Context) int {
	// **وطلبات تعديل الكابتن المعلّقة على طلبات انتهت** (`order_edit_request.go`).
	s.closeEndedEditRequests(ctx)
	if _, err := s.db.Exec(ctx, `
		UPDATE order_price_requests r SET status = 'closed', decided_at = now()
		  FROM orders o
		 WHERE o.id = r.order_id AND r.status = 'pending'
		   AND (o.closed_at IS NOT NULL OR o.status IN ('delivered','rejected','cancelled','failed','refunded'))`); err != nil {
		s.logger.Warn("طلبات سعر الكابتن: تعذّر تسكير المنتهي", "error", err)
	}
	mins := s.driverPriceEditTimeout(ctx)
	rows, err := s.db.Query(ctx, `
		UPDATE order_price_requests r SET alerted_at = now()
		  FROM orders o
		 WHERE o.id = r.order_id AND r.status = 'pending' AND r.alerted_at IS NULL
		   AND r.created_at < now() - make_interval(mins => $1::int)
		RETURNING o.id::text, o.number, r.old_total, r.new_total`, mins)
	if err != nil {
		s.logger.Warn("طلبات سعر الكابتن: تعذّرت قراءة المتأخّر", "error", err)
		return 0
	}
	type late struct {
		id         string
		number     int64
		old, newer int64
	}
	var list []late
	for rows.Next() {
		var l late
		if err := rows.Scan(&l.id, &l.number, &l.old, &l.newer); err == nil {
			list = append(list, l)
		}
	}
	rows.Close()
	if s.notify == nil {
		return len(list)
	}
	for _, l := range list {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind:  notifications.KindOrder,
			Title: t.priceRequestLateOps,
			Body: fmt.Sprintf("#%d — كان %s صار %s ل.س — الزبون ما ردّ من %d دقيقة",
				l.number, groupDigits(l.old), groupDigits(l.newer), mins),
			Entity: "order", EntityID: l.id, Href: "/dashboard/orders",
		})
	}
	return len(list)
}
