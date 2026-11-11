package orders

// ══════════════════════════════════════════════════════════════════════
// **«إحضار»: المبلغُ الحقيقيُّ يكتبه السائق ويوافق عليه المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **ما يكتبه المتجرُ عند الطلب تقدير** (`collect_amount`) — والسائقُ يدفع بمكان الاستلام
// من جيبه **ويكتب ما دفعه فعلاً**، والمتجرُ يوافق أو يرفض بنافذةٍ كنافذة تأكيد سعر الزبون.
// **ولا «استلمت» قبل الموافقة** — والحارسُ في المحرّك (`FetchAmountBlocks`) لا في الشاشة.
//
//	السائقُ يكتب المبلغ  → pending (والنسخةُ تزيد)
//	المتجرُ يوافق        → approved ⇒ يمضي إلى «استلمت»
//	المتجرُ يرفض         → rejected ⇒ يكتب مبلغاً جديداً (نسخةٌ أحدث)
//
// **والنسخةُ تحرس الموافقة** — كنسخة عرض الخاصّ (`expected_quote_version`): موافقةٌ على
// مبلغٍ تبدّل بعدها تُردّ ولا تُقفل المبلغَ الجديد.
//
// **وبلا تقديرٍ من المتجر لا خطوةَ تُفرض** — من قال «ما في مبلغ» لا يُوقَف سائقُه؛ **وإن
// كتب السائقُ مبلغاً مع ذلك فهو بانتظار الموافقة كغيره.**
//
// **وكلُّه عرضٌ خارجَ الدفتر والصندوق** — لا قيدَ ولا رصيد، كالتقدير نفسِه.

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// حالاتُ مبلغ الإحضار.
const (
	FetchAmountPending  = "pending"
	FetchAmountApproved = "approved"
	FetchAmountRejected = "rejected"
)

// MaxFetchAmount **سقفُ ما يُكتب** — رقمٌ فوقه خطأُ إصبعٍ لا مبلغ.
const MaxFetchAmount int64 = 100_000_000

var (
	// ErrFetchAmountStage **ليس وقتَه** — ليست «إحضاراً»، أو ليس بمكان الاستلام، أو وافق المتجرُ.
	ErrFetchAmountStage = httpx.NewError(http.StatusConflict,
		"fetch_amount_stage", "errors.fetch_amount_stage")
	// ErrFetchAmountStale **المبلغُ تبدّل أو قُرّر** — موافقةٌ على نسخةٍ قديمة.
	ErrFetchAmountStale = httpx.NewError(http.StatusConflict,
		"fetch_amount_stale", "errors.fetch_amount_stale")
	// ErrFetchAmountNotApproved **لا «استلمت» والمبلغُ بلا موافقة المتجر.**
	ErrFetchAmountNotApproved = httpx.NewError(http.StatusConflict,
		"fetch_amount_not_approved", "errors.fetch_amount_not_approved")
)

// FetchAmountBlocks **أيوقف مبلغُ الإحضار «استلمت»؟** — دالّةٌ صافيةٌ يقرؤها المحرّك.
//
// **في «إحضار» وحدَه**: مبلغٌ ينتظر أو رُفض يوقف، **وتقديرٌ من المتجر بلا موافقةٍ يوقف.**
// وبلا تقديرٍ ولا مبلغٍ مكتوبٍ يمضي.
func FetchAmountBlocks(mode string, hasEstimate bool, status string) bool {
	if mode != DeliveryModeFetch {
		return false
	}
	switch status {
	case FetchAmountApproved:
		return false
	case FetchAmountPending, FetchAmountRejected:
		return true
	}
	return hasEstimate
}

// requireFetchAmountApproved **حارسُ «استلمت» في الإحضار** — داخلَ معاملة الانتقال.
func requireFetchAmountApproved(ctx context.Context, tx pgx.Tx, orderID string) error {
	var mode, status string
	var hasEstimate bool
	if err := tx.QueryRow(ctx, `
		SELECT delivery_mode, collect_amount IS NOT NULL, COALESCE(fetch_amount_status, '')
		  FROM orders WHERE id = $1`, orderID).Scan(&mode, &hasEstimate, &status); err != nil {
		return err
	}
	if FetchAmountBlocks(mode, hasEstimate, status) {
		return ErrFetchAmountNotApproved
	}
	return nil
}

// ProposeFetchAmount **السائقُ يكتب ما دفعه بمكان الاستلام** — ويُنتظر المتجر.
//
// **ويُعاد ما دام لم يُوافَق**: بعد رفضٍ، أو لتصحيح رقمٍ قبل أن يقرّر المتجر — **وكلُّ
// كتابةٍ نسخةٌ أحدث** فلا تُقفل موافقةٌ قديمةٌ المبلغَ الجديد.
func (s *Service) ProposeFetchAmount(ctx context.Context, orderID, driverID string, amount int64) (*Order, error) {
	return s.ProposeFetchAmountCurrency(ctx, orderID, driverID, amount, "")
}

// ProposeFetchAmountCurrency **وبعملةٍ يختارها السائق لطلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٩) —
// `currency` فارغٌ: عملةُ الطلب كما هي. **وفي «إحضار» لا تُقبل** (العملةُ عملةُ المتجر).
func (s *Service) ProposeFetchAmountCurrency(ctx context.Context, orderID, driverID string, amount int64,
	currency string) (*Order, error) {
	if amount < 0 || amount > MaxFetchAmount {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var kind, mode, status, amountStatus, driver, origin string
	err = tx.QueryRow(ctx, `
		SELECT kind, delivery_mode, status, COALESCE(fetch_amount_status, ''), COALESCE(driver_id::text, ''),
		       origin
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&kind, &mode, &status, &amountStatus, &driver, &origin)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if driver != driverID {
		return nil, httpx.ErrNotFound
	}
	// **وطلبُ الواتساب: ثمنُ الطلب بلا موافقة** (قرارُ المالك ٢٠٢٦-١١-٠٦) — `whatsAppGoodsAmount`.
	if origin == OriginWhatsApp {
		if err := s.whatsAppGoodsAmount(ctx, tx, orderID, driverID, kind, status, amount, currency); err != nil {
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
		return o, nil
	}
	if kind != KindMerchantDelivery || mode != DeliveryModeFetch || status != StAtPickup ||
		amountStatus == FetchAmountApproved {
		return nil, ErrFetchAmountStage
	}
	var version int64
	if err := tx.QueryRow(ctx, `
		UPDATE orders
		   SET fetch_amount = $2, fetch_amount_version = fetch_amount_version + 1,
		       fetch_amount_status = 'pending', fetch_amount_at = now(),
		       fetch_amount_decided_at = NULL, updated_at = now()
		 WHERE id = $1
		RETURNING fetch_amount_version`, orderID, amount).Scan(&version); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $3)`, orderID, StAtPickup, driverID,
		fmt.Sprintf("السائق كتب المبلغ اللي دفعه: %d — بانتظار موافقة المتجر", amount)); err != nil {
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
	s.notifyFetchAmount(ctx, o)
	return o, nil
}

// DecideFetchAmount **المتجرُ يوافق على المبلغ أو يرفضه** — بنسخته.
//
// **ونسخةٌ غيرُ الحاليّة أو مبلغٌ قُرّر تُردّ** (`fetch_amount_stale`) — فلا تُقفل ضغطةٌ على
// نافذةٍ قديمةٍ مبلغاً لم يره صاحبُه.
func (s *Service) DecideFetchAmount(ctx context.Context, orderID, merchantID, actorID string,
	approve bool, expectedVersion int64) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var kind, status, amountStatus, owner string
	var version int64
	var amount *int64
	var driverID *string
	err = tx.QueryRow(ctx, `
		SELECT kind, status, COALESCE(fetch_amount_status, ''), fetch_amount_version, fetch_amount,
		       COALESCE(merchant_id::text, ''), driver_id::text
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&kind, &status, &amountStatus, &version, &amount, &owner, &driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if kind != KindMerchantDelivery || owner != merchantID {
		return nil, httpx.ErrNotFound
	}
	if status != StAtPickup || amountStatus != FetchAmountPending || version != expectedVersion {
		return nil, ErrFetchAmountStale
	}
	next, note := FetchAmountRejected, "المتجر رفض المبلغ"
	if approve {
		next, note = FetchAmountApproved, "المتجر وافق على المبلغ"
	}
	if amount != nil {
		note += fmt.Sprintf(": %d", *amount)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET fetch_amount_status = $2, fetch_amount_decided_at = now(), updated_at = now()
		 WHERE id = $1`, orderID, next); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5)`, orderID, StAtPickup, actorID, note, driverID); err != nil {
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
	if driverID != nil && s.notify != nil {
		title := "المتجر رفض المبلغ — اكتب المبلغ الصحيح"
		if approve {
			title = "المتجر وافق على المبلغ — كمّل الاستلام"
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: *driverID, Kind: notifications.KindOrder, Title: title,
			Body: "#" + itoa(o.Number), Entity: "order", EntityID: orderID,
			Apps: []string{notifications.AppDriver},
		})
	}
	return o, nil
}

// notifyFetchAmount **صاحبُ المتجر يُخبَر أنّ مبلغاً ينتظره** — بعد التثبيت.
func (s *Service) notifyFetchAmount(ctx context.Context, o *Order) {
	if s.notify == nil || o == nil || o.FetchAmount == nil {
		return
	}
	p, err := s.parties(ctx, o.ID)
	if err != nil || p.merchantOwner == nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *p.merchantOwner, Kind: notifications.KindOrder,
		Title:  "السائق دفع مبلغ بمكان الاستلام — بانتظار موافقتك",
		Body:   fmt.Sprintf("#%d — المبلغ %s", o.Number, AmountText(*o.FetchAmount, o.AmountCurrency)),
		Entity: "order", EntityID: o.ID, Href: "/portal/orders",
		Apps: []string{notifications.AppMerchant},
	})
}

// ══════════════════════════════════════════════════════════════════════
// **طلبُ الواتساب: السائقُ يكتب ثمنَ الطلب والزبونُ يُخبَر** (قرارُ المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// «سعر البضاعة يكتبه السائق بتطبيقه وقت يستلم من المحل، يوصل للإدارة، ويوصل للزبون رسالة واتساب
// فيها المبالغ — إعلام بس بلا موافقة». **فبابُ «إحضار» نفسُه** (`/driver/orders/{id}/fetch-amount`)
// **بلا خطوة المتجر**: المبلغُ يُكتب «موافَقاً» في لحظته، **ويُكتب ثمنَ ما يقبضه السائق**
// (`collect_amount`) — فيقرؤه كلُّ قارئٍ قائم: بطاقةُ اللوحة، وسطرُ «اقبض من المستلم» عند السائق
// (القديمُ كذلك).
//
// **والمالُ كما كان**: ثمنُ البضاعة خارجَ الدفتر والصندوق (قرارُ المالك ٢٠٢٦-١١-٠٥: «السائق بيدفع
// للمحل، وبينسجّل بس أجرة التوصيل»)، **والمقيَّدُ الأجرةُ وحدَها عند التسليم.** ولا يوقف «استلمت».
//
// **ويُصحَّح لحدّ التسليم** (قرارُ المالك ٢٠٢٦-١١-١٠: «كلشي على المبالغ الحقيقية») — كتابةٌ ثانيةٌ
// تُبدّل الرقمَ (نسخةٌ أحدث وسطرٌ بالسجلّ) وتُرسل للزبون الرقمَ الصحيح، **وعمولةُ متجرنا تُحسب عند
// التسليم من آخر رقم** (`chargeWhatsAppCommission` تقرأ `collect_amount`) **بسعر لحظة آخر كتابة.**

// whatsAppGoodsStages **متى يكتبه السائق** — من مكان الاستلام لحدّ باب الزبون.
var whatsAppGoodsStages = map[string]bool{StAtPickup: true, StPickedUp: true, StOnTheWay: true, StAtDropoff: true}

func (s *Service) whatsAppGoodsAmount(ctx context.Context, tx pgx.Tx, orderID, driverID, kind, status string,
	amount int64, currency string) error {
	if kind != KindMerchantDelivery || !whatsAppGoodsStages[status] {
		return ErrFetchAmountStage
	}
	// **وصفرٌ = أمانةٌ بلا ثمن بضاعة** (قرارُ المالك ٢٠٢٦-١١-٠٨: «خيار امانه بدون سعر ويقبل رقم صفر») —
	// يُوثَّق ويمضي، **ولا ثمنَ يُقبض** (`collect_amount` فارغ) **ولا رسالةَ مبالغ** ولا عمولة.
	if amount < 0 {
		return httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	// **و«أمانة من شخص» لا ثمنَ فيها** (قرارُ المالك ٢٠٢٦-١١-١١) — صفرُها موثَّقٌ من الإنشاء، **ولا
	// تصل المستلِمَ رسالةُ مبالغ عن شيءٍ لم يُشترَ.**
	if amount > 0 {
		var amanah bool
		if err := tx.QueryRow(ctx, `SELECT wa_amanah FROM orders WHERE id = $1`, orderID).Scan(&amanah); err != nil {
			return err
		}
		if amanah {
			return ErrFetchAmountStage
		}
	}
	var collect *int64
	note := fmt.Sprintf("السائق كتب ثمن الطلب: %d — وصل للزبون برسالة", amount)
	if amount > 0 {
		collect = &amount
	} else {
		note = "السائق وثّقه أمانة — بلا ثمن بضاعة"
	}
	// ══════════════════════════════════════════════════════════════════
	// **والعملةُ يختارها السائق، والسعرُ يُلقط الآن** (قرارُ المالك ٢٠٢٦-١١-٠٩)
	// ══════════════════════════════════════════════════════════════════
	//
	// **فارغةً تبقى عملةُ الطلب** (التطبيقُ القديم). **وثمنٌ بالدولار يُلقط سعرُه لحظةَ كتابته** — سعرُ
	// متجرنا إن ضبطه وإلّا المرجعيّ — **فعمولتُه عند التسليم بسعر اليوم الذي كُتب فيه.**
	// **والتصحيحُ يُقال تصحيحاً بالسجلّ** — بالرقم القديم والجديد.
	var prev *int64
	var prevCurrency string
	var prevVersion int64
	if err := tx.QueryRow(ctx, `SELECT fetch_amount, amount_currency, fetch_amount_version FROM orders WHERE id = $1`,
		orderID).Scan(&prev, &prevCurrency, &prevVersion); err != nil {
		return err
	}
	var rate *float64
	if currency == "" {
		if err := tx.QueryRow(ctx, `SELECT amount_currency FROM orders WHERE id = $1`, orderID).Scan(&currency); err != nil {
			return err
		}
	}
	if currency == CurrencyUSD && amount > 0 {
		r, err := s.goodsRateFor(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if r > 0 {
			rate = &r
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders
		   SET fetch_amount = $2, fetch_amount_version = fetch_amount_version + 1,
		       fetch_amount_status = 'approved', fetch_amount_at = now(),
		       fetch_amount_decided_at = now(), collect_amount = $3, updated_at = now(),
		       amount_currency = $4, goods_fx_rate = $5
		 WHERE id = $1`, orderID, amount, collect, currency, rate); err != nil {
		return err
	}
	if amount > 0 {
		note = fmt.Sprintf("السائق كتب ثمن الطلب: %s — وصل للزبون برسالة", AmountText(amount, currency))
	}
	if prev != nil && prevVersion > 0 && (*prev != amount || prevCurrency != currency) {
		note = fmt.Sprintf("السائق عدّل ثمن الطلب من %s إلى %s — وصل للزبون برسالة",
			AmountText(*prev, prevCurrency), AmountText(amount, currency))
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $3)`, orderID, status, driverID, note)
	return err
}

// WhatsAppGoodsNotice **رسالةُ المبالغ للزبون** — رقمُه ونصُّها، **أو `false`** إن لم يكن طلبَ واتساب
// بثمنٍ مكتوب. «ثمن طلبك X ل.س + أجرة التوصيل Y ل.س = المجموع Z ل.س» (قرارُ المالك ٢٠٢٦-١١-٠٦).
func (s *Service) WhatsAppGoodsNotice(ctx context.Context, orderID string) (phone, text string, ok bool, err error) {
	var origin, currency string
	var goods *int64
	var fee, version int64
	err = s.db.QueryRow(ctx, `
		SELECT origin, COALESCE(recipient_phone, ''), fetch_amount, cash_due, amount_currency, fetch_amount_version
		  FROM orders WHERE id = $1`, orderID).Scan(&origin, &phone, &goods, &fee, &currency, &version)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, err
	}
	if origin != OriginWhatsApp || goods == nil || *goods <= 0 || phone == "" {
		return "", "", false, nil
	}
	text = WhatsAppGoodsText(*goods, fee, currency)
	// **وكتابةٌ تانية تصحيح** (قرارُ المالك ٢٠٢٦-١١-١٠) — الزبون بيعرف إنّ الرقم تغيّر.
	if version > 1 {
		text = "تعدّل المبلغ: " + text
	}
	return phone, text, true, nil
}

// WhatsAppGoodsText **نصُّ رسالة المبالغ** — بفواصل الآلاف، **والمجموعُ بالليرة وحدَها**: ثمنٌ بالدولار
// وأجرةٌ بالليرة لا تُجمعان.
func WhatsAppGoodsText(goods, fee int64, currency string) string {
	if currency == "USD" {
		return fmt.Sprintf("ثمن طلبك %s + أجرة التوصيل %s ل.س", AmountText(goods, "USD"), groupDigits(fee))
	}
	return fmt.Sprintf("ثمن طلبك %s ل.س + أجرة التوصيل %s ل.س = المجموع %s ل.س",
		groupDigits(goods), groupDigits(fee), groupDigits(goods+fee))
}
