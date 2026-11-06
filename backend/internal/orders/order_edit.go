package orders

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ الطلب العاديّ — بيد المكتب، وبسببٍ مكتوب** (قرارُ المالك ٢٠٢٦-١١-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// «غير متوفر» (`item_unavailable.go`) يحذف صنفاً. **وهذا البابُ أوسع**: كمّيّةٌ تزيد
// أو تنقص، وصنفٌ يُحذف، وصنفٌ يُضاف من متجرٍ في الطلب نفسِه، **وسعرُ صنفٍ غيّره المتجر.**
//
// # ما يُعاد حسابُه — كما يُعيده «غير متوفر»
//
//	البضاعةُ من البنود · خصمُ الكود على البضاعة الجديدة (ولا يسقط) · التوصيلُ المجانيّ
//	· ورسمُ المصدر الإضافيّ إن خرج متجرٌ كاملٌ قبل الاستلام
//
// **والمجموعُ قد يرتفع هنا** (قرارُ المالك ٢٠٢٦-١١-٠٣، البند ١): المكتبُ يقرّر
// والزبونُ يُخبَر — بالعادة هو من طلب الزيادة.
//
// # والدفع
//
//	نقص    ←  المحفظةُ يُردّ لها الفرقُ فوراً
//	زيادة  ←  يُخصم من المحفظة ما يكفيه رصيدُها، **والباقي نقدٌ يقبضه السائق** (البند ٣)
//	نقداً  ←  السائقُ يقبض المجموعَ الجديد
//
// # وسعرُ الصنف
//
// **المكتبُ يكتب سعرَ المتجر الجديد للوحدة**، وسعرُ الزبون يتبعه **بهامش السطر نفسِه**
// (الفرقُ بين السعرين كما كان) — فلا يُعاد تسعيرُ الهامش بإعدادات اليوم.
//
// # وقبل الاستلام لا قيدَ ماليّاً
//
// **مستحقُّ المتجر وعمولتُه يُحسبان عند الاستلام من البنود** — فتعديلُها قبله يجعلهما
// يتبعانها بلا سطرٍ زائد. **وبعد الاستلام قُيّد المال** — وبابُه المرحلةُ الثانية.
//
// # وصنفٌ من خارج القائمة
//
// يكتب المكتبُ اسمَه وسعرَه (`EditCustom`) — **سعرُ المتجر فيه سعرُ الزبون**، ويمرّ بالحسبة
// نفسِها كأيّ سطر.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

var (
	// ErrOrderEditKind **التعديلُ للطلب العاديّ وحدَه** — الخاصُّ بابُه «تعديل عرض السعر».
	ErrOrderEditKind = httpx.NewError(http.StatusConflict,
		"order_edit_not_standard", "errors.order_edit_not_standard")
	// ErrOrderEditClosed **لا تعديلَ بعد التسليم أو الانتهاء** — بابُه الاسترجاعُ أو التعويض.
	ErrOrderEditClosed = httpx.NewError(http.StatusConflict,
		"order_edit_closed", "errors.order_edit_closed")
	// ErrOrderEditAfterPickup **بعد الاستلام قُيّد المال** — ولا يعدّله إلّا من يملك قيدَ تصحيحه
	// (قرارُ المالك ٢٠٢٦-١١-٠٣، البند ٤: المالكُ والأدمن).
	ErrOrderEditAfterPickup = httpx.NewError(http.StatusForbidden,
		"order_edit_after_pickup", "errors.order_edit_after_pickup")
	// ErrOrderEditLast **لا يبقى طلبٌ بلا بضاعة** — يُنهى الطلبُ بدلَ حذف آخر صنف.
	ErrOrderEditLast = httpx.NewError(http.StatusConflict,
		"order_edit_last_item", "errors.order_edit_last_item")
	// ErrOrderEditStore **الصنفُ المضافُ من متجرٍ في الطلب** — متجرٌ جديدٌ يغيّر المشوار.
	ErrOrderEditStore = httpx.NewError(http.StatusConflict,
		"order_edit_store_not_in_order", "errors.order_edit_store_not_in_order")
	// ErrOrderEditNoChange **لا شيءَ تغيّر** — لا حدثَ ولا إشعارَ عن لا شيء.
	ErrOrderEditNoChange = httpx.NewError(http.StatusBadRequest,
		"order_edit_no_change", "errors.order_edit_no_change")
	// ErrOrderEditReason **السببُ إجباريّ** — تعديلٌ بلا سببٍ لا يُراجَع.
	ErrOrderEditReason = httpx.NewError(http.StatusBadRequest,
		"order_edit_reason_required", "errors.order_edit_reason_required")
)

// EditLine **تعديلُ سطرٍ قائم**: كمّيّتُه الجديدة (صفرٌ = حذف)، وسعرُ المتجر الجديد للوحدة إن تغيّر.
type EditLine struct {
	ItemID        string `json:"item_id"`
	Qty           int    `json:"qty"`
	MerchantPrice *int64 `json:"merchant_price,omitempty"`
}

// EditAdd **صنفٌ يُضاف** من قائمة متجرٍ في الطلب — يُسعَّر بسعر اليوم.
type EditAdd struct {
	MenuItemID string   `json:"menu_item_id"`
	Qty        int      `json:"qty"`
	OptionIDs  []string `json:"option_ids,omitempty"`
}

// EditCustom **صنفٌ من خارج القائمة** — اسمُه وسعرُه يكتبهما المكتب (قرارُ المالك ٢٠٢٦-١١-٠٣:
// «صنف مو موجود بالمتجر»).
//
// # قاعدةُ ماله — الأسلمُ والمتّسق
//
//   - **سعرُ المتجر = سعرُ الزبون** (`merchant_price = unit_price`): لا هامشَ منصّةٍ على صنفٍ
//     لم تسعّره المنصّة — فلا يُخترع ربحٌ بلا قاعدة، ولا يُقصّ حقُّ المتجر بلا قاعدة.
//   - **وعمولةُ المتجر تُقتطع منه كأيّ سطر** — يدخل `merchantShares` بسعر شرائه، فحقُّ المتجر
//     = السعرُ ناقصَ نسبته، **والمعادلةُ واحدةٌ قبل الاستلام وبعده** (قيدُ التصحيح نفسُه).
//   - **ومتجرُه من متاجر الطلب** — المذكورُ إن كان في الطلب، وإلّا متجرُ الطلب. وطريقةُ تسويته
//     لقطةُ سطور متجره نفسِها.
//   - **ولا `menu_item_id`** (العمودُ يقبل الفراغ منذ `0011`) — **ولا خياراتٍ ولا عرض.**
//
// ولمن أراد سعرَ متجرٍ أقلّ: يعدّله بعدها بسعر المتجر للسطر (`EditLine.MerchantPrice`)
// — والهامشُ الصفريُّ يبقى صفراً فيتبعه سعرُ الزبون.
type EditCustom struct {
	Name       string `json:"name"`
	Qty        int    `json:"qty"`
	Price      int64  `json:"price"`
	MerchantID string `json:"merchant_id,omitempty"`
}

// EditRequest **طلبُ التعديل كاملاً** — والسببُ إجباريّ.
//
// `StoreRefunded` **للنقص بعد الاستلام وحدَه** (قرارُ المالك ٢٠٢٦-١١-٠٣، البند ٢): افتراضاً
// المتجرُ يأخذ حقَّه كاملاً والمنصّةُ تتحمّل، **وإن رجّع المتجرُ المصاري نقص حقُّه.**
type EditRequest struct {
	Reason string     `json:"reason"`
	Lines  []EditLine `json:"lines"`
	Add    []EditAdd  `json:"add"`
	// Custom **أصنافٌ من خارج القائمة** — انظر `EditCustom`.
	Custom        []EditCustom `json:"custom,omitempty"`
	StoreRefunded bool         `json:"store_refunded"`
	DryRun        bool         `json:"dry_run"`
}

// EditChange **سطرٌ تغيّر** — يُقرأ في الحدث والإشعار والتدقيق.
type EditChange struct {
	Name     string `json:"name"`
	OldQty   int    `json:"old_qty"`
	NewQty   int    `json:"new_qty"`
	OldPrice int64  `json:"old_price"`
	NewPrice int64  `json:"new_price"`
	// Returned **ما رجّعه الزبونُ بعد الاستلام والمنصّةُ تتحمّله** — يبقى للمتجر حقُّه.
	Returned int `json:"returned,omitempty"`
	// merchantID **متجرُ السطر** — داخليٌّ لا يُرسَل: **كلُّ متجرٍ يُخبَر بما تغيّر عنده وحدَه** (G-24).
	merchantID string
}

// EditStoreDelta **فرقُ حقّ متجرٍ بعد الاستلام** — ما كان وما صار.
type EditStoreDelta struct {
	MerchantID string `json:"merchant_id"`
	Name       string `json:"name"`
	Method     string `json:"method"`
	OldDue     int64  `json:"old_due"`
	NewDue     int64  `json:"new_due"`
	// DriverPays **ما يدفعه السائقُ الحاليُّ للمتجر فوق ما دفع** (سالبٌ: ما يردّه المتجرُ له).
	DriverPays int64 `json:"driver_pays"`
	// Debt **ما لم يكفِه رصيدُ متجر «محفظة» فصار ديناً عليه.**
	Debt int64 `json:"debt"`
}

// EditResult **ما وقع** — يُردّ للشاشة ويُكتب في التدقيق.
type EditResult struct {
	AfterPickup bool             `json:"after_pickup"`
	OldTotal    int64            `json:"old_total"`
	NewTotal    int64            `json:"new_total"`
	Refunded    int64            `json:"refunded"`
	Charged     int64            `json:"charged"`
	OldCashDue  int64            `json:"old_cash_due"`
	CashDue     int64            `json:"cash_due"`
	Changes     []EditChange     `json:"changes"`
	Stores      []EditStoreDelta `json:"stores,omitempty"`
	Reason      string           `json:"reason"`
	DryRun      bool             `json:"dry_run"`
}

// editLineRow **سطرٌ قائمٌ مقفول** — ما يُقرأ منه للتعديل.
type editLineRow struct {
	id, name, merchantID string
	qty                  int
	unit, cost           int64
	returned             bool
}

// EditOrder **يعدّل طلباً عاديّاً ويعيد حسابَه** — ذرّيّاً.
//
// `afterPickupAllowed` **قدرةُ من يعدّل بعد الاستلام** (قرارُ المالك، البند ٤) — يقرّرها
// المعالِج. و`hook` **أثرُ التدخّل في المعاملة نفسِها** (التدقيق).
func (s *Service) EditOrder(ctx context.Context, actorID, orderID string, req EditRequest,
	afterPickupAllowed bool,
	hook func(context.Context, dbtx.Querier, EditResult) error) (*Order, EditResult, error) {
	var res EditResult
	res.DryRun = req.DryRun
	req.Reason = strings.TrimSpace(req.Reason)
	res.Reason = req.Reason
	if req.Reason == "" {
		return nil, res, ErrOrderEditReason
	}
	if utf8.RuneCountInString(req.Reason) > 500 {
		return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len(req.Lines) == 0 && len(req.Add) == 0 && len(req.Custom) == 0 {
		return nil, res, ErrOrderEditNoChange
	}
	if _, err := uuid.Parse(orderID); err != nil {
		return nil, res, httpx.ErrNotFound
	}
	seen := map[string]bool{}
	for _, l := range req.Lines {
		if _, err := uuid.Parse(l.ItemID); err != nil || seen[l.ItemID] ||
			l.Qty < 0 || l.Qty > 50 || (l.MerchantPrice != nil && *l.MerchantPrice < 0) {
			return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
		}
		seen[l.ItemID] = true
	}
	for _, a := range req.Add {
		if _, err := uuid.Parse(a.MenuItemID); err != nil || a.Qty < 1 || a.Qty > 50 {
			return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
		}
	}
	for i, c := range req.Custom {
		c.Name = strings.TrimSpace(c.Name)
		req.Custom[i].Name = c.Name
		if c.Name == "" || utf8.RuneCountInString(c.Name) > 120 || c.Qty < 1 || c.Qty > 50 ||
			c.Price < 1 || c.Price > 1_000_000_000 {
			return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
		}
		if c.MerchantID != "" {
			if _, err := uuid.Parse(c.MerchantID); err != nil {
				return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
			}
		}
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
		cashDue, waived, sourcesFee, driverFee                  int64
		sharePct, shareFixed                                    int64
		driverID                                                *string
	}
	err = tx.QueryRow(ctx, `
		SELECT status, kind, COALESCE(customer_id::text, ''), payment_method,
		       COALESCE(promo_code, ''), COALESCE(merchant_id::text, ''), number,
		       subtotal, delivery_fee, discount, total, wallet_paid, cash_due,
		       promo_delivery_waived, sources_fee, driver_fee,
		       snap_platform_delivery_percent, snap_platform_delivery_fixed, driver_id::text
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&o.status, &o.kind, &o.customerID, &o.method, &o.promoCode, &o.merchantID, &o.number,
			&o.subtotal, &o.deliveryFee, &o.discount, &o.total, &o.walletPaid, &o.cashDue,
			&o.waived, &o.sourcesFee, &o.driverFee, &o.sharePct, &o.shareFixed, &o.driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, res, httpx.ErrNotFound
	}
	if err != nil {
		return nil, res, err
	}
	if o.kind != KindStandard {
		return nil, res, ErrOrderEditKind
	}
	if terminal(o.status) {
		return nil, res, ErrOrderEditClosed
	}
	// **«بعد الاستلام» يُعرف بالدفتر لا بالحال**: طلبٌ أُرسل مع سائقٍ آخرَ عاد إلى
	// «بانتظار سائق» ومستحقُّ متجره مقيَّدٌ منذ الاستلام الأوّل.
	var settled bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM merchant_settlements WHERE order_id = $1)
		    OR EXISTS(SELECT 1 FROM wallet_transactions
		               WHERE ref = $1::text AND kind IN ('merchant_earning', 'merchant_cash_accrued'))`,
		orderID).Scan(&settled); err != nil {
		return nil, res, err
	}
	res.AfterPickup = settled || pastPickup[o.status]
	// **وفي الطلب بعدّة محطّات «بعد الاستلام» لكلّ متجر** (§7.1، قرارُ المالك ٣أ): بنودُ متجرٍ
	// استُلم لا يعدّلها إلّا من يملك التصحيح، **وبنودُ متجرٍ لم يُستلم بعدُ يعدّلها المكتب.**
	perStore, collected, err := editCollectedStoresTx(ctx, tx, orderID, o.status)
	if err != nil {
		return nil, res, err
	}
	if perStore {
		res.AfterPickup = false
	}
	if res.AfterPickup && !afterPickupAllowed {
		return nil, res, ErrOrderEditAfterPickup
	}
	if res.AfterPickup {
		// **طلبٌ سُوّي قبل صفوف التسوية** (التراث) لا يُصحَّح آليّاً — لا صفَّ يُقيَّد عليه.
		var rowsExist bool
		if err := tx.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM merchant_settlements WHERE order_id = $1)`, orderID).
			Scan(&rowsExist); err != nil {
			return nil, res, err
		}
		if settled && !rowsExist {
			return nil, res, ErrOrderEditSettlement
		}
	}

	storesBefore, err := orderStoreSet(ctx, tx, orderID, o.merchantID)
	if err != nil {
		return nil, res, err
	}

	// ── ٢ · السطورُ القائمة ────────────────────────────────────────────
	touched := map[string]bool{}
	for _, l := range req.Lines {
		var r editLineRow
		err := tx.QueryRow(ctx, `
			SELECT id::text, name, COALESCE(merchant_id::text, $3), qty, unit_price, merchant_price, returned
			FROM order_items WHERE id = $1 AND order_id = $2 FOR UPDATE`, l.ItemID, orderID, o.merchantID).
			Scan(&r.id, &r.name, &r.merchantID, &r.qty, &r.unit, &r.cost, &r.returned)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, res, httpx.ErrNotFound
		}
		if err != nil {
			return nil, res, err
		}
		// **وسطرٌ رجّعه الزبونُ لا يُعدَّل** — أثرُه قُيّد.
		if r.returned {
			return nil, res, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
		}
		newUnit, newCost := r.unit, r.cost
		if l.MerchantPrice != nil && *l.MerchantPrice != r.cost {
			// **سعرُ الزبون يتبع سعرَ المتجر بهامش السطر نفسِه** — ولا ينزل تحت الصفر.
			newCost = *l.MerchantPrice
			newUnit = newCost + (r.unit - r.cost)
			if newUnit < 0 {
				newUnit = 0
			}
		}
		if l.Qty == r.qty && newUnit == r.unit && newCost == r.cost {
			continue
		}
		change := EditChange{Name: r.name, OldQty: r.qty, NewQty: l.Qty, OldPrice: r.unit, NewPrice: newUnit,
			merchantID: r.merchantID}
		// ══════════════════════════════════════════════════════════════
		// **نقصٌ بعد الاستلام والمتجرُ لم يردّ المال** (قرارُ المالك، البند ٢)
		// ══════════════════════════════════════════════════════════════
		//
		// الزبونُ رجّع الصنفَ بعد أن استلمه السائق: **المتجرُ يأخذ حقَّه كاملاً والمنصّةُ
		// تتحمّل.** فيبقى ما رجّع سطراً بسعر متجره (لا يتغيّر حقُّه ولا عمولتُه) **وسعرُ
		// الزبون فيه صفر** — والمجموعُ ينقص وحدَه، والفرقُ على الخزينة.
		if (res.AfterPickup || collected[r.merchantID]) && !req.StoreRefunded && l.Qty < r.qty {
			back := r.qty - l.Qty
			change.Returned = back
			if l.Qty == 0 {
				if _, err := tx.Exec(ctx, `
					UPDATE order_items SET returned = true, unit_price = 0, offer_cut = 0
					 WHERE id = $1 OR drink_for = $1`, r.id); err != nil {
					return nil, res, err
				}
			} else {
				// **والمشروبُ المرتجعُ يبقى تحت صنفه** (`drink_for` و`drink_option_id`) —
				// فيُعرض بين مشروبات الصنف في الفاتورة (قرارُ المالك ٢٠٢٦-١١-٠٦ «أ»).
				if _, err := tx.Exec(ctx, `
					INSERT INTO order_items (order_id, menu_item_id, name, unit_price, merchant_price,
					                         merchant_id, qty, note, options, merchant_settlement_method,
					                         offer_cut, offer_borne_by, returned, drink_for, drink_option_id)
					SELECT order_id, menu_item_id, name, 0, merchant_price, merchant_id, $2, note, options,
					       merchant_settlement_method, 0, offer_borne_by, true, drink_for, drink_option_id
					FROM order_items WHERE id = $1`, r.id, back); err != nil {
					return nil, res, err
				}
				if _, err := tx.Exec(ctx, `
					UPDATE order_items SET qty = $2, unit_price = $3, merchant_price = $4 WHERE id = $1`,
					r.id, l.Qty, newUnit, newCost); err != nil {
					return nil, res, err
				}
			}
		} else if l.Qty == 0 {
			// **ومشروباتُ الصنف تُمحى بمحوه** (`drink_for ... ON DELETE CASCADE`) —
			// **إلّا ما رجّعه الزبونُ منها**: أثرُه قُيّد والمتجرُ يأخذ حقَّه فيه، فيُفكّ
			// عن صنفه قبل المحو ويبقى (قرارُ المالك ٢٠٢٦-١١-٠٦ «أ»).
			if _, err := tx.Exec(ctx, `
				UPDATE order_items SET drink_for = NULL, drink_option_id = NULL
				 WHERE drink_for = $1 AND returned`, r.id); err != nil {
				return nil, res, err
			}
			if _, err := tx.Exec(ctx, `DELETE FROM order_items WHERE id = $1`, r.id); err != nil {
				return nil, res, err
			}
		} else if _, err := tx.Exec(ctx, `
			UPDATE order_items SET qty = $2, unit_price = $3, merchant_price = $4 WHERE id = $1`,
			r.id, l.Qty, newUnit, newCost); err != nil {
			return nil, res, err
		}
		touched[r.merchantID] = true
		res.Changes = append(res.Changes, change)
	}

	// ── ٣ · الأصنافُ المضافة — بسعر اليوم ومن متاجر الطلب ─────────────────
	if len(req.Add) > 0 {
		inputs := make([]ItemInput, 0, len(req.Add))
		for _, a := range req.Add {
			inputs = append(inputs, ItemInput{MenuItemID: a.MenuItemID, Qty: a.Qty, OptionIDs: a.OptionIDs})
		}
		items, _, err := unit.priceItems(ctx, tx, inputs)
		if err != nil {
			return nil, res, err
		}
		for _, it := range items {
			if !storesBefore[it.MerchantID] {
				return nil, res, ErrOrderEditStore
			}
			var method string
			if err := tx.QueryRow(ctx, `
				SELECT COALESCE(
				         (SELECT max(merchant_settlement_method) FROM order_items
				           WHERE order_id = $1 AND COALESCE(merchant_id::text, $3) = $2),
				         (SELECT settlement_method FROM merchants WHERE id = $2::uuid))`,
				orderID, it.MerchantID, o.merchantID).Scan(&method); err != nil {
				return nil, res, err
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_items (order_id, menu_item_id, name, unit_price,
				                         merchant_price, merchant_id, qty, note, options,
				                         merchant_settlement_method, offer_cut, offer_borne_by)
				VALUES ($1, $2, $3, $4, $5, $6, $7, '', $8, $9, $10, NULLIF($11,''))`,
				orderID, it.MenuItemID, it.Name, it.UnitPrice, it.MerchantPrice,
				it.MerchantID, it.Qty, marshalOptions(it.Options), method,
				it.OfferCut, it.OfferBorneBy); err != nil {
				return nil, res, err
			}
			touched[it.MerchantID] = true
			res.Changes = append(res.Changes, EditChange{
				Name: it.Name, OldQty: 0, NewQty: it.Qty, NewPrice: it.UnitPrice, merchantID: it.MerchantID,
			})
		}
	}
	// ── ٣ب · أصنافٌ من خارج القائمة — سعرُ المتجر سعرُ الزبون (`EditCustom`) ─────
	for _, c := range req.Custom {
		mid := c.MerchantID
		if mid == "" {
			mid = o.merchantID
		}
		if mid == "" || !storesBefore[mid] {
			return nil, res, ErrOrderEditStore
		}
		var method string
		if err := tx.QueryRow(ctx, `
			SELECT COALESCE(
			         (SELECT max(merchant_settlement_method) FROM order_items
			           WHERE order_id = $1 AND COALESCE(merchant_id::text, $3) = $2),
			         (SELECT settlement_method FROM merchants WHERE id = $2::uuid))`,
			orderID, mid, o.merchantID).Scan(&method); err != nil {
			return nil, res, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, name, unit_price,
			                         merchant_price, merchant_id, qty, note, options,
			                         merchant_settlement_method, offer_cut)
			VALUES ($1, NULL, $2, $3, $3, $4::uuid, $5, '', '[]'::jsonb, $6, 0)`,
			orderID, c.Name, c.Price, mid, c.Qty, method); err != nil {
			return nil, res, err
		}
		touched[mid] = true
		res.Changes = append(res.Changes, EditChange{Name: c.Name, OldQty: 0, NewQty: c.Qty, NewPrice: c.Price,
			merchantID: mid})
	}
	if len(res.Changes) == 0 {
		return nil, res, ErrOrderEditNoChange
	}

	// **ومتجرٌ استُلم من بنوده ما تبدّل ⇒ تصحيحٌ بعد الاستلام** — لمن يملكه وحدَه.
	touchedCollected := false
	for m := range touched {
		if collected[m] {
			touchedCollected = true
		}
	}
	if touchedCollected && !afterPickupAllowed {
		return nil, res, ErrOrderEditAfterPickup
	}

	// ── ٤ · لا يبقى طلبٌ بلا بضاعة ──────────────────────────────────────
	var left int
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(sum(qty) FILTER (WHERE NOT returned), 0) FROM order_items WHERE order_id = $1`,
		orderID).Scan(&left); err != nil {
		return nil, res, err
	}
	if left <= 0 {
		return nil, res, ErrOrderEditLast
	}

	// ── ٥ · البضاعةُ والرسومُ والكود ─────────────────────────────────────
	var subtotal int64
	if err := tx.QueryRow(ctx,
		`SELECT COALESCE(sum(unit_price * qty), 0) FROM order_items WHERE order_id = $1`, orderID).
		Scan(&subtotal); err != nil {
		return nil, res, err
	}
	feeBeforePromo := o.deliveryFee + o.waived
	sourcesFee, driverFee, merchantID := o.sourcesFee, o.driverFee, o.merchantID
	storesAfter, err := orderStoreSet(ctx, tx, orderID, o.merchantID)
	if err != nil {
		return nil, res, err
	}
	// **وقبل الاستلام وحدَه: متجرٌ خرج كلُّه يُعيد رسمَ المصادر** — كما يفعل «غير متوفر».
	if !res.AfterPickup && len(storesAfter) < len(storesBefore) {
		remaining, err := remainingSources(ctx, tx, orderID, o.merchantID)
		if err != nil {
			return nil, res, err
		}
		if !storesAfter[o.merchantID] && len(remaining.merchants) > 0 {
			merchantID = remaining.merchants[0]
		}
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
			driverFee = DriverFeeWithShare(feeBeforePromo, o.sharePct, o.shareFixed)
		}
	}

	deliveryFee := feeBeforePromo
	var discount, waived int64
	if o.promoCode != "" {
		var kind string
		var value int64
		var maxDiscount *int64
		err := tx.QueryRow(ctx, `
			SELECT kind, value, max_discount FROM promo_codes WHERE code = $1`, o.promoCode).
			Scan(&kind, &value, &maxDiscount)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
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
	res.OldTotal, res.NewTotal, res.OldCashDue = o.total, total, o.cashDue

	// ── ٦ · الدفع ──────────────────────────────────────────────────────
	walletPaid := o.walletPaid
	note := fmt.Sprintf("تعديل الطلب: %s", req.Reason)
	if total < walletPaid {
		// **نقصٌ دون ما دُفع من المحفظة** — يُردّ الفرقُ فوراً.
		diff := walletPaid - total
		if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, diff, "refund", orderID, note, &actorID); err != nil {
			return nil, res, err
		}
		res.Refunded = diff
		walletPaid = total
	} else if o.method == "wallet" && total > walletPaid {
		// **زيادةٌ على طلب محفظة** — يُخصم ما يكفيه الرصيد، والباقي نقد.
		avail, err := s.wallet.AvailableTx(ctx, tx, o.customerID)
		if err != nil {
			return nil, res, err
		}
		charge := min64(total-walletPaid, avail)
		if charge > 0 {
			if _, err := s.wallet.ApplyTx(ctx, tx, o.customerID, -charge, "order_payment",
				orderID, note, &actorID); err != nil {
				return nil, res, err
			}
			res.Charged = charge
			walletPaid += charge
		}
	}
	cashDue := total - walletPaid
	res.CashDue = cashDue

	if _, err := tx.Exec(ctx, `
		UPDATE orders SET subtotal = $2, delivery_fee = $3, discount = $4, total = $5,
		       wallet_paid = $6, cash_due = $7, promo_delivery_waived = $8,
		       sources_fee = $9, driver_fee = $10, merchant_id = $11::uuid, updated_at = now()
		WHERE id = $1`, orderID, subtotal, deliveryFee, discount, total,
		walletPaid, cashDue, waived, sourcesFee, driverFee, merchantID); err != nil {
		return nil, res, err
	}

	// **ومتجرٌ خرجت بنودُه كلُّها قسمُه «انشال»** — كشيل القسم (رسمُ المحطّة أُعيد فوق).
	if perStore {
		if err := markEmptyPartsTx(ctx, tx, orderID, actorID, req.Reason, o.status, o.driverID); err != nil {
			return nil, res, err
		}
	}
	if touchedCollected {
		res.AfterPickup = true
	}
	// ── ٧ · بعد الاستلام: حقُّ كلِّ متجرٍ يُصحَّح بقيدٍ جديد، ثمّ الخزينة ────────
	if res.AfterPickup {
		stores, err := s.correctSettlementsTx(ctx, tx, orderID, actorID, o.driverID, req.Reason)
		if err != nil {
			return nil, res, err
		}
		res.Stores = stores
		if err := s.creditTreasury(ctx, tx, orderID, actorID); err != nil {
			return nil, res, err
		}
	}

	// ── ٨ · الأثر: حدثٌ وإشعاراتٌ وتدقيق — في المعاملة نفسِها ─────────────
	if err := s.editTrailTx(ctx, tx, actorID, orderID, o.status, o.number, o.customerID,
		o.driverID, touched, res); err != nil {
		return nil, res, err
	}
	if hook != nil {
		if err := hook(ctx, tx, res); err != nil {
			return nil, res, err
		}
	}
	// **والمعاينةُ تُحسب كلُّها ثمّ تُطوى** — الرقمُ الذي يراه المكتبُ قبل التأكيد هو
	// ما سيقع بالضبط، لا تقديرٌ في الشاشة.
	// **وما بقي من المحطّات استلمه السائقُ كلَّه ⇒ `picked_up`** في المعاملة نفسِها (§2.4).
	var follow *appliedTransition
	if perStore && !req.DryRun {
		lo, err := lockOrderTx(ctx, tx, orderID)
		if err != nil {
			return nil, res, err
		}
		if follow, err = unit.stopsDoneFollowTx(ctx, tx, lo, actorID); err != nil {
			return nil, res, err
		}
	}
	if req.DryRun {
		return nil, res, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, res, err
	}
	if follow != nil {
		updated, err := s.afterTransition(ctx, follow)
		if err != nil {
			return nil, res, err
		}
		s.publishWalletsOf(ctx, orderID)
		return updated, res, nil
	}
	updated, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, res, err
	}
	s.publishOrder(updated)
	s.publishWalletsOf(ctx, orderID)
	return updated, res, nil
}

// orderStoreSet **متاجرُ الطلب من بنوده** — وفراغُ المتجر في البند متجرُ الطلب.
func orderStoreSet(ctx context.Context, q dbtx.Querier, orderID, orderMerchant string) (map[string]bool, error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT COALESCE(merchant_id::text, $2) FROM order_items WHERE order_id = $1`,
		orderID, orderMerchant)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

// editChangesText **السطورُ بلغةٍ تُقرأ** — «برغر ×٢←×١، كولا ×٠←×١».
func editChangesText(ch []EditChange) string {
	parts := make([]string, 0, len(ch))
	for _, c := range ch {
		switch {
		case c.NewQty == 0:
			parts = append(parts, fmt.Sprintf("حُذف %s", c.Name))
		case c.OldQty == 0:
			parts = append(parts, fmt.Sprintf("أُضيف %s ×%d", c.Name, c.NewQty))
		case c.OldQty != c.NewQty && c.OldPrice != c.NewPrice:
			parts = append(parts, fmt.Sprintf("%s ×%d←×%d بسعر %s", c.Name, c.OldQty, c.NewQty, groupDigits(c.NewPrice)))
		case c.OldQty != c.NewQty:
			parts = append(parts, fmt.Sprintf("%s ×%d←×%d", c.Name, c.OldQty, c.NewQty))
		default:
			parts = append(parts, fmt.Sprintf("%s بسعر %s بدل %s", c.Name, groupDigits(c.NewPrice), groupDigits(c.OldPrice)))
		}
	}
	return strings.Join(parts, "، ")
}

// editTrailTx **سطرٌ في سجلّ الطلب وإشعاراتُ الأطراف** — الزبونُ بمجموعه الجديد وسببه،
// والسائقُ بما يقبضه، والمتجرُ بما تغيّر في طلبه.
func (s *Service) editTrailTx(ctx context.Context, q dbtx.Querier, actorID, orderID, status string,
	number int64, customerID string, driverID *string, stores map[string]bool, res EditResult) error {
	changes := editChangesText(res.Changes)
	note := fmt.Sprintf("تعديل الطلب: %s — السبب: %s — المجموع %s ← %s ل.س",
		changes, res.Reason, groupDigits(res.OldTotal), groupDigits(res.NewTotal))
	var actor any
	if actorID != "" {
		actor = actorID
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5)`, orderID, status, actor, note, driverID); err != nil {
		return err
	}
	if customerID != "" {
		body := fmt.Sprintf("%s — السبب: %s. صار مجموع طلبك %s ل.س", changes, res.Reason, groupDigits(res.NewTotal))
		if res.Refunded > 0 {
			body += fmt.Sprintf("، ورجع %s ل.س إلى محفظتك", groupDigits(res.Refunded))
		}
		if res.Charged > 0 {
			body += fmt.Sprintf("، وخُصم %s ل.س من محفظتك", groupDigits(res.Charged))
		}
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: customerID, Kind: notifications.KindOrder,
			Title: t.orderEdited, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		}); err != nil {
			return err
		}
	}
	if driverID != nil && *driverID != "" {
		body := fmt.Sprintf("طلب #%d: %s. المبلغ المطلوب من الزبون صار %s ل.س",
			number, changes, groupDigits(res.CashDue))
		for _, st := range res.Stores {
			if st.DriverPays > 0 {
				body += fmt.Sprintf("، وادفع لـ%s فرق %s ل.س", st.Name, groupDigits(st.DriverPays))
			} else if st.DriverPays < 0 {
				body += fmt.Sprintf("، وخذ من %s %s ل.س", st.Name, groupDigits(-st.DriverPays))
			}
		}
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: *driverID, Kind: notifications.KindOrder,
			Title: t.orderEditedDriver, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppDriver},
		}); err != nil {
			return err
		}
	}
	// **والمتجرُ يُخبَر بطلبه** — مالكُ كلِّ متجرٍ في الطلب.
	due := map[string]EditStoreDelta{}
	for _, st := range res.Stores {
		due[st.MerchantID] = st
	}
	for mid := range stores {
		var owner *string
		if err := q.QueryRow(ctx, `SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`, mid).
			Scan(&owner); err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if owner == nil || *owner == "" {
			continue
		}
		body := fmt.Sprintf("طلب #%d: %s", number, changes)
		// **وطلبٌ من أكثرَ من متجر: كلُّ متجرٍ بما تغيّر عنده وحدَه** (G-24) — لا أصنافُ غيره
		// ولا أسعارُها. **ومتجرٌ لم يتغيّر عنده شيءٌ ولا حقُّه لا يُخبَر.**
		if len(stores) > 1 {
			var own []EditChange
			for _, c := range res.Changes {
				if strings.EqualFold(c.merchantID, mid) {
					own = append(own, c)
				}
			}
			d, moved := due[mid]
			if len(own) == 0 && (!moved || d.NewDue == d.OldDue) {
				continue
			}
			body = fmt.Sprintf("طلب #%d", number)
			if len(own) > 0 {
				body += ": " + editChangesText(own)
			}
		}
		if d, ok := due[mid]; ok && d.NewDue != d.OldDue {
			body += fmt.Sprintf(". صار حقك %s ل.س بدل %s", groupDigits(d.NewDue), groupDigits(d.OldDue))
		}
		if err := notifications.NotifyUserTx(ctx, q, notifications.Input{
			UserID: *owner, Kind: notifications.KindOrder,
			Title: t.orderEditedMerchant, Body: body,
			Entity: "order", EntityID: orderID, Href: "/portal",
			Apps: []string{notifications.AppMerchant},
		}); err != nil {
			return err
		}
	}
	return nil
}

// EditCatalogOption خيارٌ في مجموعة — باسمه وفرق سعره.
type EditCatalogOption struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	PriceDelta int64  `json:"price_delta"`
}

// EditCatalogGroup **مجموعةُ خياراتٍ إجباريّة** — والاختياريّةُ لا تُعرض في التعديل.
type EditCatalogGroup struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	Min     int                 `json:"min"`
	Max     int                 `json:"max"`
	Options []EditCatalogOption `json:"options"`
}

// EditCatalogItem صنفٌ يُضاف — **بسعر البيع اليوم** قبل الخيارات والعروض.
type EditCatalogItem struct {
	ID     string             `json:"id"`
	Name   string             `json:"name"`
	Price  int64              `json:"price"`
	Groups []EditCatalogGroup `json:"groups"`
}

// EditCatalogStore متجرٌ في الطلب وأصنافُه المتاحة.
type EditCatalogStore struct {
	MerchantID string            `json:"merchant_id"`
	Name       string            `json:"name"`
	Items      []EditCatalogItem `json:"items"`
}

// EditCatalogLine **سطرٌ قائمٌ بسعريه** — سعرُ المتجر لا يصل الزبونَ، ويحتاجه المكتبُ ليعدّله.
type EditCatalogLine struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	Qty           int     `json:"qty"`
	UnitPrice     int64   `json:"unit_price"`
	MerchantPrice int64   `json:"merchant_price"`
	MerchantID    string  `json:"merchant_id"`
	DrinkFor      *string `json:"drink_for,omitempty"`
}

// EditCatalogResult **ما يحتاجه محرّرُ الطلب**: سطورُه بسعريها، وما يُضاف من متاجره.
type EditCatalogResult struct {
	Lines  []EditCatalogLine  `json:"lines"`
	Stores []EditCatalogStore `json:"stores"`
}

// EditCatalog **ما يُضاف إلى الطلب**: أصنافُ متاجره وحدَها، المتاحةُ والمعتمدة.
// **والسعرُ الدقيقُ يحسبه الخادمُ عند المعاينة** — هذا دليلٌ للاختيار.
func (s *Service) EditCatalog(ctx context.Context, orderID string) (EditCatalogResult, error) {
	res := EditCatalogResult{Lines: []EditCatalogLine{}, Stores: []EditCatalogStore{}}
	if _, err := uuid.Parse(orderID); err != nil {
		return res, httpx.ErrNotFound
	}
	var orderMerchant string
	err := s.db.QueryRow(ctx, `SELECT COALESCE(merchant_id::text, '') FROM orders WHERE id = $1`, orderID).
		Scan(&orderMerchant)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, httpx.ErrNotFound
	}
	if err != nil {
		return res, err
	}
	lrows, err := s.db.Query(ctx, `
		SELECT id::text, name, qty, unit_price, merchant_price, COALESCE(merchant_id::text, $2), drink_for::text
		FROM order_items WHERE order_id = $1 AND NOT returned
		ORDER BY (drink_for IS NOT NULL), name, id`, orderID, orderMerchant)
	if err != nil {
		return res, err
	}
	for lrows.Next() {
		var l EditCatalogLine
		if err := lrows.Scan(&l.ID, &l.Name, &l.Qty, &l.UnitPrice, &l.MerchantPrice, &l.MerchantID, &l.DrinkFor); err != nil {
			lrows.Close()
			return res, err
		}
		res.Lines = append(res.Lines, l)
	}
	lrows.Close()
	if err := lrows.Err(); err != nil {
		return res, err
	}
	stores, err := orderStoreSet(ctx, s.db, orderID, orderMerchant)
	if err != nil {
		return res, err
	}
	rule := pricing.RuleFrom(ctx, s.settings.On(s.db))
	for mid := range stores {
		st := EditCatalogStore{MerchantID: mid, Items: []EditCatalogItem{}}
		if err := s.db.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1::uuid`, mid).
			Scan(&st.Name); err != nil {
			return res, err
		}
		rows, err := s.db.Query(ctx, `
			SELECT mi.id::text, mi.name, mi.merchant_price, mi.margin_override, ps.margin_override
			FROM menu_items mi
			LEFT JOIN menu_sections ms ON ms.id = mi.section_id
			LEFT JOIN platform_sections ps ON ps.id = mi.platform_section_id
			WHERE mi.merchant_id = $1::uuid AND mi.available AND mi.approved
			ORDER BY COALESCE(ms.sort_order, 0), mi.sort_order, mi.name`, mid)
		if err != nil {
			return res, err
		}
		index := map[string]int{}
		for rows.Next() {
			var it EditCatalogItem
			var cost int64
			var itemMargin, sectionMargin *int64
			if err := rows.Scan(&it.ID, &it.Name, &cost, &itemMargin, &sectionMargin); err != nil {
				rows.Close()
				return res, err
			}
			it.Price = rule.SalePrice(cost, itemMargin, sectionMargin)
			it.Groups = []EditCatalogGroup{}
			index[it.ID] = len(st.Items)
			st.Items = append(st.Items, it)
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return res, err
		}
		// **والمجموعاتُ الإجباريّةُ وحدَها** — بلا مشروباتٍ بأعدادها.
		grows, err := s.db.Query(ctx, `
			SELECT g.item_id::text, g.id::text, g.name, g.min_select, g.max_select,
			       o.id::text, o.name, o.price_delta
			FROM modifier_groups g
			JOIN menu_items mi ON mi.id = g.item_id
			JOIN modifier_options o ON o.group_id = g.id AND o.available
			WHERE mi.merchant_id = $1::uuid AND g.min_select > 0
			ORDER BY g.item_id, g.sort_order, g.id, o.sort_order, o.name`, mid)
		if err != nil {
			return res, err
		}
		for grows.Next() {
			var itemID, gid, gname, oid, oname string
			var mn, mx int
			var delta int64
			if err := grows.Scan(&itemID, &gid, &gname, &mn, &mx, &oid, &oname, &delta); err != nil {
				grows.Close()
				return res, err
			}
			i, ok := index[itemID]
			if !ok {
				continue
			}
			gs := st.Items[i].Groups
			if len(gs) == 0 || gs[len(gs)-1].ID != gid {
				gs = append(gs, EditCatalogGroup{ID: gid, Name: gname, Min: mn, Max: mx, Options: []EditCatalogOption{}})
			}
			gs[len(gs)-1].Options = append(gs[len(gs)-1].Options, EditCatalogOption{ID: oid, Name: oname, PriceDelta: delta})
			st.Items[i].Groups = gs
		}
		grows.Close()
		if err := grows.Err(); err != nil {
			return res, err
		}
		res.Stores = append(res.Stores, st)
	}
	return res, nil
}
