package orders

// ══════════════════════════════════════════════════════════════════════
// **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦
// ══════════════════════════════════════════════════════════════════════
//
// المتجرُ «نقداً» يقبض مستحقَّه من السائق عند الاستلام: ثمنَ البضاعة ناقصَ
// عمولتنا (`payCashByDriver`). **وأحياناً يرفض أن يخصمها** فيأخذ الثمنَ كاملاً —
// فالسائقُ لا يحمل العمولةَ التي يُطالَب بها في صندوقه.
//
// فيمحوها المكتبُ من الطلب نهائيّاً، **وكلُّ ما بُني عليها يُصحَّح بقيدٍ جديد**
// (لا يُمحى قيدٌ ولا يُعدَّل):
//
//	السائق    سطرٌ سالبٌ في صندوقه (`commission_waived`) بمقدارها.
//	المتجر    قيدُ احتباسٍ موجب + دفعٌ سالبٌ بمقدار عمولته (`merchant_cash_accrued` ثمّ
//	          `merchant_cash_paid`)، **وصفُّ تسويته يحملهما** (`waived_*_tx_id`) —
//	          فالمكتوبُ أنّه قبض مستحقَّه + العمولةَ من السائق، وهو الواقع.
//	المندوب   نصيبُه من العمولة يُعكَس بالنوع نفسِه سالباً (`commission`)، وما عجزت
//	          عنه محفظتُه دَينٌ (`waived_rep_commission`) — «هو المسؤول عن إقناع المتجر».
//	الطلب     `platform_commission = 0` ومقدارُ المحو ومن ومتى ولماذا.
//	الخزينة   `creditTreasury` تُعيد الحسبةَ بالفرق: تنقص بالعمولة وتسترجع نصيبَ المندوب.

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrWaiverReasonRequired **السببُ إلزاميّ** — مالٌ يُمحى لا يُمحى بلا كلمة.
	ErrWaiverReasonRequired = httpx.NewError(http.StatusBadRequest,
		"commission_waiver_reason_required", "errors.commission_waiver_reason_required")
	// ErrWaiverNotDelivered الطلبُ لم يُسلَّم (أو استُرجع).
	ErrWaiverNotDelivered = httpx.NewError(http.StatusConflict,
		"commission_waiver_not_delivered", "errors.commission_waiver_not_delivered")
	// ErrWaiverNotStandard طلبٌ خاصٌّ أو «لدي توصيلة» — لا عمولةَ متجرٍ فيه.
	ErrWaiverNotStandard = httpx.NewError(http.StatusConflict,
		"commission_waiver_not_standard", "errors.commission_waiver_not_standard")
	// ErrWaiverNoCommission لا عمولةَ على الطلب تُمحى.
	ErrWaiverNoCommission = httpx.NewError(http.StatusConflict,
		"commission_waiver_no_commission", "errors.commission_waiver_no_commission")
	// ErrCommissionAlreadyWaived مُحيت سابقاً — والثانيةُ لا تقيّد شيئاً.
	ErrCommissionAlreadyWaived = httpx.NewError(http.StatusConflict,
		"commission_already_waived", "errors.commission_already_waived")
	// ErrWaiverNotDriverCash السائقُ لم يقبض نقدَ هذا الطلب (أو قبض أقلَّ من العمولة).
	ErrWaiverNotDriverCash = httpx.NewError(http.StatusConflict,
		"commission_waiver_not_driver_cash", "errors.commission_waiver_not_driver_cash")
	// ErrWaiverWalletStore المتجرُ يُسوّى «محفظة» أو لم يدفعه السائقُ عند الاستلام —
	// العمولةُ خُصمت من مستحقّه فلا تقع الحالة.
	ErrWaiverWalletStore = httpx.NewError(http.StatusConflict,
		"commission_waiver_wallet_store", "errors.commission_waiver_wallet_store")
	// ErrWaiverDriverCashShort صندوقُ السائق اليومَ أقلُّ من العمولة — سلّم المالَ للمكتب.
	ErrWaiverDriverCashShort = httpx.NewError(http.StatusConflict,
		"commission_waiver_driver_cash_short", "errors.commission_waiver_driver_cash_short")
)

// WaiverResult نتيجةُ المحو — للردّ والتدقيق.
type WaiverResult struct {
	OrderID    string `json:"order_id"`
	Number     int64  `json:"number"`
	Waived     int64  `json:"waived"`
	DriverHeld int64  `json:"driver_held"`
	// DriverID السائقُ الذي قبض — للإشعار والتدقيق، لا للردّ.
	DriverID string `json:"-"`
	// RepID ومقدارُ ما عُكس من نصيبه، وما صار ديناً عليه.
	RepID       string `json:"-"`
	RepReversed int64  `json:"-"`
	RepDebt     int64  `json:"-"`
}

type waiverShare struct {
	settleID   string
	merchantID string
	commission int64
}

// WaiveCommissionTx **يمحو عمولةَ المنصّة عن طلبٍ مسلَّم** — في معاملة المستدعي
// (والتدقيقُ معه فيها). انظر رأسَ الملفّ.
func (s *Service) WaiveCommissionTx(ctx context.Context, q wallet.Querier,
	orderID, actorID, reason string) (WaiverResult, error) {
	var res WaiverResult
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return res, ErrWaiverReasonRequired
	}

	// (١) **القفلُ على صفّ الطلب** — نداءان متزامنان: الثاني ينتظر ثمّ يجد المحوَ قائماً.
	var status, kind string
	var number, commission, waived int64
	err := q.QueryRow(ctx, `
		SELECT number, status, kind, platform_commission, commission_waived
		FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&number, &status, &kind, &commission, &waived)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, httpx.ErrNotFound
	}
	if err != nil {
		return res, err
	}
	res.OrderID, res.Number = orderID, number
	if waived > 0 {
		return res, ErrCommissionAlreadyWaived
	}
	if kind != KindStandard {
		return res, ErrWaiverNotStandard
	}
	if status != StDelivered {
		return res, ErrWaiverNotDelivered
	}
	if commission <= 0 {
		return res, ErrWaiverNoCommission
	}

	// (٢) **السائقُ الذي قبض نقدَ الطلب** — ويكفي ما قبضه للعمولة.
	var driverID string
	var collected int64
	err = q.QueryRow(ctx, `
		SELECT driver_id::text, sum(amount)::bigint FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'
		 GROUP BY driver_id ORDER BY sum(amount) DESC LIMIT 1`, orderID).Scan(&driverID, &collected)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, ErrWaiverNotDriverCash
	}
	if err != nil {
		return res, err
	}
	if collected < commission {
		return res, ErrWaiverNotDriverCash
	}
	res.DriverID = driverID

	// (٣) **كلُّ متاجر الطلب «نقداً» دفعها السائقُ عند الاستلام** — وعمولةُ كلٍّ منها
	// = كلفتُه − مستحقُّه المقيَّد (صيغةُ `FI-06.b`)، **من الدفتر لا من إعداد اليوم.**
	rows, err := q.Query(ctx, `
		SELECT ms.id::text, ms.merchant_id::text, ms.method, ms.state,
		       COALESCE(ms.paid_via, ''), ms.amount,
		       COALESCE((SELECT sum(oi.merchant_price * oi.qty) FROM order_items oi
		                  JOIN orders o ON o.id = oi.order_id
		                 WHERE oi.order_id = ms.order_id
		                   AND COALESCE(oi.merchant_id, o.merchant_id) = ms.merchant_id), 0)::bigint
		FROM merchant_settlements ms
		WHERE ms.order_id = $1
		ORDER BY ms.created_at, ms.id
		FOR UPDATE OF ms`, orderID)
	if err != nil {
		return res, err
	}
	var shares []waiverShare
	var walletStore bool
	var sum int64
	for rows.Next() {
		var sh waiverShare
		var method, state, via string
		var amount, cost int64
		if err := rows.Scan(&sh.settleID, &sh.merchantID, &method, &state, &via, &amount, &cost); err != nil {
			rows.Close()
			return res, err
		}
		if method != "cash" || state != "cash_paid" || via != paidViaDriver {
			walletStore = true
		}
		sh.commission = cost - amount
		sum += sh.commission
		shares = append(shares, sh)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return res, err
	}
	if len(shares) == 0 || walletStore {
		return res, ErrWaiverWalletStore
	}
	// **طلبٌ بلا بنودٍ** (ما قبل السعرين): الكلفةُ صفر — فالعمولةُ كلُّها على متجره الوحيد.
	if sum <= 0 && len(shares) == 1 {
		shares[0].commission, sum = commission, commission
	}
	if sum != commission {
		return res, fmt.Errorf("محوُ العمولة: مجموعُ عمولات المتاجر %d ≠ عمولةِ الطلب %d (طلب %s)",
			sum, commission, orderID)
	}

	// (٤) **صندوقُ السائق اليومَ يحمل العمولة** — وإلّا سلّمها للمكتب ولا يُنزَل تحت الصفر.
	var held int64
	if err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1), 0)`,
		driverID).Scan(&held); err != nil {
		return res, err
	}
	if held < commission {
		return res, ErrWaiverDriverCashShort
	}

	// (٥) **المندوبُ — بالحسبة قبل تصفير العمولة** (تُقرأ المدخلاتُ نفسُها).
	repID, repReverse, err := s.repShareToReverse(ctx, q, orderID, commission)
	if err != nil {
		return res, err
	}

	// (٦) السائق.
	if res.DriverHeld, err = s.cashbox.WaiveCommissionTx(ctx, q, driverID, commission,
		orderID, reason, actorID); err != nil {
		return res, err
	}

	// (٧) المتجر — قبض العمولةَ نقداً من السائق.
	hid, err := s.cashHoldingOn(ctx, q)
	if err != nil {
		return res, err
	}
	for _, sh := range shares {
		if sh.commission <= 0 {
			continue
		}
		_, accTx, err := s.wallet.ApplyTxID(ctx, q, hid, sh.commission, "merchant_cash_accrued",
			orderID, "عمولةٌ لم يخصمها المتجرُ — أخذها نقداً من السائق", &actorID)
		if err != nil {
			return res, err
		}
		_, paidTx, err := s.wallet.ApplyTxID(ctx, q, hid, -sh.commission, "merchant_cash_paid",
			orderID, "عمولةٌ دفعها السائقُ للمتجر مع مستحقّه عند الاستلام", &actorID)
		if err != nil {
			return res, err
		}
		if _, err := q.Exec(ctx, `
			UPDATE merchant_settlements
			   SET commission_waived = $2, waived_accrued_tx_id = $3, waived_paid_tx_id = $4
			 WHERE id = $1`, sh.settleID, sh.commission, accTx, paidTx); err != nil {
			return res, err
		}
	}

	// (٨) المندوب.
	if repID != "" && repReverse > 0 {
		res.RepID = repID
		if res.RepReversed, res.RepDebt, err = s.reverseRepForWaiver(ctx, q, repID, orderID,
			repReverse, actorID); err != nil {
			return res, err
		}
	}

	// (٩) الطلب.
	if _, err := q.Exec(ctx, `
		UPDATE orders
		   SET platform_commission = 0, commission_waived = $2, commission_waived_at = now(),
		       commission_waived_by = $3, commission_waived_reason = $4
		 WHERE id = $1`, orderID, commission, actorID, reason); err != nil {
		return res, err
	}

	// (١٠) الخزينة — **بعد كلّ ما سبق** فتقرأ ما بقي مقيَّداً.
	if err := s.creditTreasury(ctx, q, orderID, actorID); err != nil {
		return res, err
	}
	res.Waived = commission
	return res, nil
}

// repShareToReverse **كم يُعكَس من نصيب المندوب** — الفرقُ بين ما قبضه وما كان سيقبضه
// لو كانت العمولةُ صفراً (بلقطة الطلب: نسبتُه ومصدرُ قاعدته).
//
// **ولا يتجاوز ما قبضه فعلاً** — ومندوبٌ لم يقبض (بلا تفعيلٍ أو شارٍ من متجره) لا شيء.
// **والمحجوزُ لمندوبٍ موقوفٍ يُنقَص في مكانه** ولا يُكتب في محفظته (يُرجع صفراً للعكس).
func (s *Service) repShareToReverse(ctx context.Context, q wallet.Querier,
	orderID string, commission int64) (string, int64, error) {
	var repID *string
	var promoCost int64
	if err := q.QueryRow(ctx, `
		SELECT m.sales_rep_user_id::text,
		       o.discount + o.promo_delivery_waived
		         - COALESCE((SELECT c.waived FROM free_delivery_claims c WHERE c.order_id = o.id), 0)
		FROM orders o
		JOIN merchants m ON m.id = COALESCE(o.merchant_id, o.partner_merchant_id)
		WHERE o.id = $1`, orderID).Scan(&repID, &promoCost); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", 0, nil
		}
		return "", 0, err
	}
	if repID == nil {
		return "", 0, nil
	}

	// **ما بقي له بعد المحو** — بالقاعدة نفسِها والعمولةُ صفر. وبلا لقطةٍ: صفر (يُعكَس كلُّه).
	var remain int64
	snap, err := s.snapshotOf(ctx, q, orderID)
	switch {
	case errors.Is(err, ErrNoSnapshot):
	case err != nil:
		return "", 0, err
	default:
		var margin int64
		if err := q.QueryRow(ctx, `SELECT `+OrderMarginSQL("$1"), orderID).Scan(&margin); err != nil {
			return "", 0, err
		}
		base := pricing.RepCommissionBase(pricing.CommissionSource(snap.CommissionSource), 0, margin, promoCost)
		remain = pricing.Amount{Mode: "percent", Value: snap.RepCommissionPercent}.Of(base)
	}

	// **المحجوزُ القائم** (مندوبٌ موقوف) — يُنقَص هنا، ولا قيدَ في محفظته.
	var heldID string
	var heldAmount int64
	err = q.QueryRow(ctx, `
		SELECT id::text, amount FROM rep_held_commissions
		 WHERE order_id = $1 AND status = 'held' FOR UPDATE`, orderID).Scan(&heldID, &heldAmount)
	if err == nil {
		cut := heldAmount - remain
		if cut <= 0 {
			return *repID, 0, nil
		}
		if heldAmount-cut <= 0 {
			_, err = q.Exec(ctx, `
				UPDATE rep_held_commissions SET status = 'forfeited', decided_at = now() WHERE id = $1`, heldID)
		} else {
			_, err = q.Exec(ctx, `UPDATE rep_held_commissions SET amount = amount - $2 WHERE id = $1`, heldID, cut)
		}
		return *repID, 0, err
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", 0, err
	}

	// **ما قبضه فعلاً** — العمولةُ الموجبةُ وحدَها (سطورُ اقتطاع الدَّين ليست نقصاً منها).
	var paid int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0)::bigint FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission' AND user_id = $2 AND amount > 0`,
		orderID, *repID).Scan(&paid); err != nil {
		return "", 0, err
	}
	rev := paid - remain
	if rev <= 0 {
		return *repID, 0, nil
	}
	return *repID, rev, nil
}

// reverseRepForWaiver **يعكس نصيبَ المندوب** — بالنوع نفسِه سالباً (فتراه الخزينةُ وكلُّ
// حسبةٍ تجمع `commission`)، **وما لا يحتمله متاحُه دَينٌ عليه** يُقتطَع من عمولةٍ قادمة —
// كعكس مستحقّ المتجر في `reverseCommissions`.
func (s *Service) reverseRepForWaiver(ctx context.Context, q wallet.Querier,
	repID, orderID string, amount int64, actorID string) (taken, debt int64, err error) {
	var available int64
	if err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT balance - reserved FROM wallets WHERE user_id = $1), 0)`,
		repID).Scan(&available); err != nil {
		return 0, 0, err
	}
	taken = amount
	if available < taken {
		taken = available
	}
	if taken < 0 {
		taken = 0
	}
	if taken > 0 {
		if _, err := s.wallet.ApplyTx(ctx, q, repID, -taken, "commission", orderID,
			"عكسُ عمولة — المتجرُ لم يدفع عمولةَ المنصّة على هذا الطلب", &actorID); err != nil {
			return 0, 0, err
		}
	}
	if debt = amount - taken; debt > 0 {
		if _, err := obligations.Create(ctx, q, obligations.PartyRep, repID, debt,
			obligations.CauseWaivedRepCommission, orderID, &actorID); err != nil {
			return 0, 0, err
		}
	}
	return taken, debt, nil
}
