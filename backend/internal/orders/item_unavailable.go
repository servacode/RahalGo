package orders

// ══════════════════════════════════════════════════════════════════════
// **صنفٌ غيرُ متوفّر — المكتبُ يحذفه ويُعاد حسابُ الطلب** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// صنفٌ في طلبٍ عاديٍّ نفد عند المتجر. **فيحذفه المكتبُ قبل الاستلام** — السطرَ كلَّه
// أو بعضَ كمّيّته — **ويُعاد حسابُ كلِّ ما يُشتقّ من البضاعة:**
//
//	المجموعُ الفرعيّ والكلّيّ · المستحقُّ النقديّ · الهامش (من البنود نفسِها)
//	· خصمُ الكود (يُعاد على البضاعة الجديدة ولا يسقط) · ورسمُ المصدر الإضافيّ
//	إن خرج متجرٌ كاملٌ من طلبٍ متعدّد المصادر
//
// **والتوصيلُ كما هو** — إلّا في الحالة الأخيرة. **والمحفظةُ يُردّ لها الفرقُ فوراً.**
//
// # والزبونُ لا يدفع أكثرَ أبداً
//
// (قرارُ المالك ٢٠٢٦-١٠-١٥.) **العيبُ عيبُ المتجر لا الزبون** — فالكودُ يبقى ولو صار
// الطلبُ دون حدّه الأدنى، **ولا يُمَسّ استعمالُه ولا عدّادُه.** فلا مجموعَ يرتفع بحذف
// صنف، **ولا خصمَ زيادةٍ من المحفظة.**
//
// # ولماذا قبل الاستلام وحدَه
//
// **مستحقُّ المتجر وعمولتُه يُحسبان عند الاستلام من `order_items`** (`merchantShares`)
// — **فحذفُ البند قبله يجعلهما يتبعانه بلا سطرٍ زائد**، وكذلك ما يدفعه السائقُ للمتجر
// (`pay_merchant`). **وبعد الاستلام قُيّد المالُ**، والتصحيحُ حينها بابُه آخر.
//
// # والصنفُ الأخيرُ لا يُحذف
//
// **طلبٌ بلا بضاعةٍ ليس طلباً** — والمكتبُ يُلغيه بسببه، **فيُستردّ كلُّه في بابه.**

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

var (
	// ErrItemUnavailableTooLate **لا يُحذف صنفٌ بعد الاستلام أو بعد انتهاء الطلب.**
	ErrItemUnavailableTooLate = httpx.NewError(http.StatusConflict,
		"item_unavailable_too_late", "errors.item_unavailable_too_late")
	// ErrItemUnavailableLast **الصنفُ الأخيرُ لا يُحذف** — يُلغى الطلبُ بدلَه.
	ErrItemUnavailableLast = httpx.NewError(http.StatusConflict,
		"item_unavailable_last_item", "errors.item_unavailable_last_item")
	// ErrItemUnavailableKind **الحذفُ للطلب العاديّ وحدَه** — الخاصُّ والتوصيلةُ بلا أصناف.
	ErrItemUnavailableKind = httpx.NewError(http.StatusConflict,
		"item_unavailable_not_standard", "errors.item_unavailable_not_standard")
)

// itemUnavailableFrom **الحالاتُ التي يُحذف فيها صنف** — كلُّها قبل الاستلام.
var itemUnavailableFrom = map[string]bool{
	StPending: true, StAccepted: true, StPreparing: true,
	StDispatching: true, StAssigned: true, StAtPickup: true,
}

// ItemUnavailableResult **ما وقع** — يُقرأ في التدقيق والردّ.
type ItemUnavailableResult struct {
	ItemName   string `json:"item_name"`
	RemovedQty int    `json:"removed_qty"`
	OldTotal   int64  `json:"old_total"`
	NewTotal   int64  `json:"new_total"`
	Refunded   int64  `json:"refunded"`
	SourceGone bool   `json:"source_gone"`
}

// MarkItemUnavailable **يحذف صنفاً غيرَ متوفّرٍ ويعيد حسابَ الطلب** — ذرّيّاً.
//
// `qty` **ما يُحذف من السطر**: صفرٌ أو أكثرُ من كمّيّته = السطرُ كلُّه.
// و`hook` **أثرُ التدخّل في المعاملة نفسِها** (التدقيق) — كبابِ الانتقال.
func (s *Service) MarkItemUnavailable(ctx context.Context, actorID, orderID, itemID string, qty int,
	hook func(context.Context, dbtx.Querier, ItemUnavailableResult) error) (*Order, ItemUnavailableResult, error) {
	var res ItemUnavailableResult
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, res, httpx.ErrNotFound
	}
	if _, err := uuid.Parse(itemID); err != nil {
		return nil, res, httpx.ErrNotFound
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	unit := s.on(tx)

	// ── ١ · الطلبُ مقفولاً ─────────────────────────────────────────────
	var o struct {
		status, kind, customerID, method, promoCode, merchantID string
		number                                                  int64
		subtotal, deliveryFee, discount, total, walletPaid      int64
		waived, sourcesFee, driverFee, sharePct, shareFixed     int64
		driverID                                                *string
	}
	err = tx.QueryRow(ctx, `
		SELECT status, kind, COALESCE(customer_id::text, ''), payment_method,
		       COALESCE(promo_code, ''), COALESCE(merchant_id::text, ''), number,
		       subtotal, delivery_fee, discount, total, wallet_paid,
		       promo_delivery_waived, sources_fee, driver_fee,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, driver_id
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.status, &o.kind, &o.customerID, &o.method, &o.promoCode, &o.merchantID, &o.number,
			&o.subtotal, &o.deliveryFee, &o.discount, &o.total, &o.walletPaid,
			&o.waived, &o.sourcesFee, &o.driverFee, &o.sharePct, &o.shareFixed, &o.driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, res, httpx.ErrNotFound
	}
	if err != nil {
		return nil, res, err
	}
	if o.kind != KindStandard {
		return nil, res, ErrItemUnavailableKind
	}
	if !itemUnavailableFrom[o.status] {
		return nil, res, ErrItemUnavailableTooLate
	}

	// ── ٢ · البندُ وما يبقى بعده ────────────────────────────────────────
	var lineQty int
	var lineMerchant string
	// **وفراغُ المتجر في البند يعني متجرَ الطلب** — بندٌ قديمٌ بلا مصدرٍ مصدرُه متجرُه.
	err = tx.QueryRow(ctx, `
		SELECT name, qty, COALESCE(merchant_id::text, $3)
		FROM order_items WHERE id = $1 AND order_id = $2 FOR UPDATE`, itemID, orderID, o.merchantID).
		Scan(&res.ItemName, &lineQty, &lineMerchant)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, res, httpx.ErrNotFound
	}
	if err != nil {
		return nil, res, err
	}
	res.RemovedQty = lineQty
	if qty > 0 && qty < lineQty {
		res.RemovedQty = qty
	}
	var allQty int
	// **ومشروباتُ الصنف تُمحى بمحوه** (`drink_for ... ON DELETE CASCADE`) —
	// فلا تُعدّ باقيةً حين يخرج صنفُها كلُّه.
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(sum(qty) FILTER (WHERE NOT ($3 AND drink_for IS NOT DISTINCT FROM $2::uuid)), 0)
		 FROM order_items WHERE order_id = $1`, orderID, itemID, res.RemovedQty == lineQty).
		Scan(&allQty); err != nil {
		return nil, res, err
	}
	if allQty-res.RemovedQty <= 0 {
		return nil, res, ErrItemUnavailableLast
	}
	if res.RemovedQty == lineQty {
		_, err = tx.Exec(ctx, `DELETE FROM order_items WHERE id = $1`, itemID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE order_items SET qty = qty - $2 WHERE id = $1`, itemID, res.RemovedQty)
	}
	if err != nil {
		return nil, res, err
	}

	// ── ٣ · البضاعةُ من البنود — المعادلةُ نفسُها التي سعّرت الطلب ───────
	// (`priceItems`: سعرُ البيع الملقوطُ × الكمّيّة.)
	var subtotal int64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(sum(unit_price * qty), 0) FROM order_items WHERE order_id = $1`, orderID).
		Scan(&subtotal); err != nil {
		return nil, res, err
	}

	// ── ٤ · أخرج متجرٌ كاملٌ من الطلب؟ ───────────────────────────────────
	feeBeforePromo := o.deliveryFee + o.waived
	sourcesFee, driverFee, merchantID := o.sourcesFee, o.driverFee, o.merchantID
	var stillThere bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM order_items
		              WHERE order_id = $1 AND COALESCE(merchant_id::text, $2) = $3)`,
		orderID, o.merchantID, lineMerchant).Scan(&stillThere); err != nil {
		return nil, res, err
	}
	if !stillThere {
		res.SourceGone = true
		remaining, err := remainingSources(ctx, tx, orderID, o.merchantID)
		if err != nil {
			return nil, res, err
		}
		// **ومتجرُ الطلب إن خرج يخلفه أوّلُ الباقين** — وإلّا بقي الطلبُ على متجرٍ لا بضاعةَ له فيه.
		if lineMerchant == o.merchantID && len(remaining.merchants) > 0 {
			merchantID = remaining.merchants[0]
		}
		// **ورسمُ المصادر يُعاد بالمعادلة نفسِها** (`extraSourceFee`) على ما بقي،
		// **ولا يرتفع أبداً** — إعداداتُ اليوم لا تُحمَّل على طلب الأمس.
		if o.sourcesFee > 0 {
			base := feeBeforePromo - o.sourcesFee
			var extra int64
			if len(remaining.items) > 0 {
				src, err := unit.SourcesOf(ctx, tx, remaining.items)
				if err != nil {
					return nil, res, err
				}
				extra = unit.extraSourceFee(ctx, src, base)
			}
			if extra > o.sourcesFee {
				extra = o.sourcesFee
			}
			sourcesFee = extra
			feeBeforePromo = base + extra
			// **وأجرُ السائق يتبع الأجرة** — وقفةٌ أقلّ، **بحصّة المنصّة الملقوطة.**
			driverFee = DriverFeeWithShare(feeBeforePromo, o.sharePct, o.shareFixed)
		}
	}

	// ── ٥ · الكود: يُعاد حسابُه على البضاعة الجديدة — ولا يسقط ─────────────
	//
	// **لا يُسأل عن حدّه الأدنى ولا عن صلاحيّته اليوم**: وُعد به الزبونُ يومَ الطلب،
	// **وحذفُ ما نفد عند المتجر لا ينقض الوعد.** النسبةُ تُعاد بسقفها، والثابتُ يُقصَر على
	// البضاعة (`promoCut`)، **والاستعمالُ وعدّادُه لا يُمسّان.**
	deliveryFee := feeBeforePromo
	var discount, waived int64
	promoCode := o.promoCode
	if promoCode != "" {
		var kind string
		var value int64
		var maxDiscount *int64
		err := tx.QueryRow(ctx, `
			SELECT kind, value, max_discount FROM promo_codes
			WHERE code = $1`, promoCode).Scan(&kind, &value, &maxDiscount)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// **كودٌ مُحي بعد الطلب** — يبقى خصمُه كما وُعد، لا يُخترَع غيرُه.
			discount, waived, deliveryFee = min64(o.discount, subtotal), o.waived, o.deliveryFee
		case err != nil:
			return nil, res, err
		default:
			discount = promoCut(kind, value, maxDiscount, subtotal, &deliveryFee)
			if feeBeforePromo > deliveryFee {
				waived = feeBeforePromo - deliveryFee
			}
		}
	}
	// **والتوصيلُ المجانيّ يبقى وعداً** — يُعاد على الأجرة الجديدة بسقف عرضه.
	if fw, err := s.freeDeliveryReapply(ctx, tx, orderID, deliveryFee); err != nil {
		return nil, res, err
	} else if fw > 0 {
		deliveryFee -= fw
		waived += fw
	}
	total := subtotal - discount + deliveryFee
	if total < 0 {
		total = 0
	}
	// **وحارسٌ بنيويّ**: الحسبةُ أعلاه لا ترفع المجموعَ (البضاعةُ تنقص والخصمُ يتبعها
	// صعوداً ونزولاً)، **فإن ارتفع يوماً بقاعدةٍ تُضاف بعدُ امتصّه الخصمُ** — ولا يدفع
	// الزبونُ أكثرَ ممّا وافق عليه.
	if total > o.total {
		extra := min64(total-o.total, subtotal-discount)
		discount += extra
		total -= extra
	}
	res.OldTotal, res.NewTotal = o.total, total

	// ── ٦ · الدفع: المحفظةُ يُردّ لها الفرقُ فوراً ───────────────────────
	walletPaid, cashDue := int64(0), total
	if o.method == "wallet" {
		walletPaid, cashDue = total, 0
		note := fmt.Sprintf("صنف غير متوفر: %s", res.ItemName)
		if diff := o.walletPaid - total; diff > 0 {
			if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, diff, "refund",
				orderID, note, &actorID); err != nil {
				return nil, res, err
			}
			res.Refunded = diff
		}
	}

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET subtotal = $2, delivery_fee = $3, discount = $4, total = $5,
		       wallet_paid = $6, cash_due = $7, promo_delivery_waived = $8,
		       sources_fee = $9, driver_fee = $10, promo_code = NULLIF($11, ''),
		       merchant_id = $12::uuid, updated_at = now()
		WHERE id = $1`, orderID, subtotal, deliveryFee, discount, total,
		walletPaid, cashDue, waived, sourcesFee, driverFee, promoCode, merchantID); err != nil {
		return nil, res, err
	}

	// ── ٧ · الأثر: حدثٌ وإشعارٌ وتدقيق — في المعاملة نفسِها ───────────────
	note := fmt.Sprintf("صنف غير متوفر: %s ×%d — المجموع الجديد %s ل.س",
		res.ItemName, res.RemovedQty, groupDigits(total))
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5)`, orderID, o.status, actor, note, o.driverID); err != nil {
		return nil, res, err
	}
	if o.customerID != "" {
		body := fmt.Sprintf("الصنف %s غير متوفر — صار طلبك %s ل.س", res.ItemName, groupDigits(total))
		if err := notifications.NotifyUserTx(ctx, tx, notifications.Input{
			UserID: o.customerID, Kind: notifications.KindOrder,
			Title: t.itemUnavailable, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		}); err != nil {
			return nil, res, err
		}
	}
	if hook != nil {
		if err := hook(ctx, tx, res); err != nil {
			return nil, res, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, res, err
	}

	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, res, err
	}
	s.publishOrder(updated)
	s.publishWalletsOf(ctx, orderID)
	return updated, res, nil
}

// orderSources **ما بقي في الطلب من مصادر وأصناف** — بترتيب البنود.
type orderSources struct {
	merchants []string
	items     []ItemInput
}

// remainingSources يقرأ متاجرَ البنود الباقية وأصنافَها — **ليُعاد بها قياسُ المسافة.**
//
// **والمتاجرُ بثقل بضاعتها** — أكبرُها أوّلاً: هو من يخلف متجرَ الطلب إن خرج.
func remainingSources(ctx context.Context, q dbtx.Querier, orderID, orderMerchant string) (orderSources, error) {
	var out orderSources
	rows, err := q.Query(ctx, `
		SELECT COALESCE(merchant_id::text, $2) AS mid,
		       array_remove(array_agg(menu_item_id::text), NULL)
		FROM order_items WHERE order_id = $1
		GROUP BY 1
		ORDER BY sum(unit_price * qty) DESC, 1`, orderID, orderMerchant)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var mid string
		var items []string
		if err := rows.Scan(&mid, &items); err != nil {
			return out, err
		}
		out.merchants = append(out.merchants, mid)
		for _, it := range items {
			out.items = append(out.items, ItemInput{MenuItemID: it, Qty: 1})
		}
	}
	return out, rows.Err()
}
