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
		number, stops                                           int64
		subtotal, deliveryFee, discount, total, walletPaid      int64
		waived, sourcesFee, driverFee, sharePct, shareFixed     int64
		driverID                                                *string
	}
	err = tx.QueryRow(ctx, `
		SELECT status, kind, COALESCE(customer_id::text, ''), payment_method,
		       COALESCE(promo_code, ''), COALESCE(merchant_id::text, ''), number,
		       subtotal, delivery_fee, discount, total, wallet_paid,
		       promo_delivery_waived, sources_fee, driver_fee,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, driver_id, stops
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.status, &o.kind, &o.customerID, &o.method, &o.promoCode, &o.merchantID, &o.number,
			&o.subtotal, &o.deliveryFee, &o.discount, &o.total, &o.walletPaid,
			&o.waived, &o.sourcesFee, &o.driverFee, &o.sharePct, &o.shareFixed, &o.driverID, &o.stops)
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
	// **ولا بعد الاستلام ولو عاد الطلبُ إلى الطابور** (PD-8): مستحقُّ المتجر مقيَّدٌ منذ الاستلام،
	// **وحذفٌ يغيّره في الطلب لا في صفّ التسوية** — وتصحيحُه بابُه «تعديل الطلب» (`EditOrder`).
	if left, err := GoodsLeftStore(ctx, tx, orderID); err != nil {
		return nil, res, err
	} else if left {
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
	// ══════════════════════════════════════════════════════════════════
	// **وفي الطلب بعدّة محطّات: صنفُ متجرٍ استُلم لا يُحذف، وآخرُ صنفٍ لمتجرٍ يشيل قسمَه** (§6.2)
	// ══════════════════════════════════════════════════════════════════
	//
	// **مستحقُّ المتجر المستلَم مقيَّد** — وتصحيحُه بابُه «تعديل الطلب». **وصنفٌ هو آخرُ ما للمتجر في
	// الطلب يُخرج المتجرَ كلَّه** — فيُشال قسمُه (ذنبُ المنصّة) بالباب نفسِه الذي يشيل به المكتبُ قسماً:
	// لقطتُه في قسمه، وخبرُ الزبون والسائق، **وإعادةُ الحساب الواحدة.**
	if o.stops > 1 {
		if collected, err := partCollectedTx(ctx, tx, orderID, lineMerchant); err != nil {
			return nil, res, err
		} else if collected {
			return nil, res, ErrPartCollected
		}
		var storeLeft int
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(sum(qty) FILTER (WHERE NOT ($4 AND drink_for IS NOT DISTINCT FROM $2::uuid)
			                                  AND drink_for IS NULL), 0)
			FROM order_items WHERE order_id = $1 AND COALESCE(merchant_id::text, $3) = $5`,
			orderID, itemID, o.merchantID, res.RemovedQty == lineQty, lineMerchant).Scan(&storeLeft); err != nil {
			return nil, res, err
		}
		if res.RemovedQty == lineQty {
			var lineIsDrink bool
			if err := tx.QueryRow(ctx, `SELECT drink_for IS NOT NULL FROM order_items WHERE id = $1`, itemID).
				Scan(&lineIsDrink); err != nil {
				return nil, res, err
			}
			if !lineIsDrink && storeLeft-res.RemovedQty <= 0 {
				_ = tx.Rollback(ctx)
				return s.itemUnavailableRemovesPart(ctx, actorID, orderID, lineMerchant, res, hook)
			}
		}
	}
	if res.RemovedQty == lineQty {
		_, err = tx.Exec(ctx, `DELETE FROM order_items WHERE id = $1`, itemID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE order_items SET qty = qty - $2 WHERE id = $1`, itemID, res.RemovedQty)
	}
	if err != nil {
		return nil, res, err
	}

	// ── ٣…٦ · إعادةُ الحساب — الدالّةُ المشتركةُ مع إزالة قسمٍ من طلبٍ بعدّة محطّات ──
	rc, err := unit.recomputeAfterRemovalTx(ctx, tx, orderID, lineMerchant, actorID,
		fmt.Sprintf("صنف غير متوفر: %s", res.ItemName))
	if err != nil {
		return nil, res, err
	}
	res.SourceGone, res.OldTotal, res.NewTotal, res.Refunded = rc.sourceGone, rc.oldTotal, rc.newTotal, rc.refunded
	total := rc.newTotal

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

// itemUnavailableRemovesPart **آخرُ صنفٍ لمتجرٍ في طلبٍ بعدّة محطّات ⇒ قسمُه يُشال** (ذنبُ المنصّة).
//
// **والتدقيقُ في معاملة الشيل نفسِها** — كما في الحذف.
func (s *Service) itemUnavailableRemovesPart(ctx context.Context, actorID, orderID, store string,
	res ItemUnavailableResult, hook func(context.Context, dbtx.Querier, ItemUnavailableResult) error) (*Order, ItemUnavailableResult, error) {
	res.SourceGone = true
	o, err := s.removePart(ctx, actorID, []string{"ops"}, orderID, store,
		"صنف غير متوفر: "+res.ItemName, FaultPlatform,
		func(ctx context.Context, q dbtx.Querier, r partRemoval) error {
			res.OldTotal, res.NewTotal, res.Refunded = r.money.oldTotal, r.money.newTotal, r.money.refunded
			if hook != nil {
				return hook(ctx, q, res)
			}
			return nil
		})
	return o, res, err
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
