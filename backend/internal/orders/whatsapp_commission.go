package orders

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ متجرنا على طلب الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٨
// ══════════════════════════════════════════════════════════════════════
//
// طلبُ واتسابٍ من متجرٍ من متاجرنا (`pickup_merchant_id`) **وثمنُه مكتوب** (`collect_amount` — يكتبه
// السائقُ أو الموظّف): **عند التسليم تُحسب عمولةُ المتجر على الثمن** بنسبته (تجاوزُه ثمّ لقطةُ الطلب،
// `orderPct`) **وتُخصم من محفظة صاحبه إلى الخزينة.**
//
//	المتاحُ يكفي     ⇐ `order_payment` سالبٌ بمرجع الطلب — والخزينةُ تقرؤه مدفوعاً (`creditTreasury`)
//	لا يكفي          ⇐ دينٌ على المتجر (`whatsapp_order_commission`) — ولا محفظةَ سالبة
//	متجرٌ غيرُ تابع  ⇐ لا شيء · بلا ثمن ⇐ لا شيء
//	ثمنٌ بالدولار    ⇐ بسعر صرف المتجر، **وبلا سعرٍ لا عمولة** (تحذيرٌ في السجلّ)
//
// **ولا `wallet_paid`**: أجرةُ طلب الواتساب نقدٌ كلُّها (`orders_whatsapp_shape` و`FI-03.c`).
// **والعمولةُ في `platform_commission`** للتقارير. **ولا عمولةَ مندوبٍ عليه** — خارجَ القرار.
//
// **والاسترجاعُ بعد التسليم يردّها**: المأخوذُ من المحفظة وما سُدّ من الدين يعود محفظةً (`refund`)،
// **وباقي الدين يُسقَط.**

import (
	"context"
	"errors"
	"math"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// ملاحظتا الكشف — **صاحبُ المتجر يعرف لماذا تحرّك رصيدُه.**
const (
	whatsAppCommissionNote       = "عمولة طلب واتساب"
	whatsAppCommissionRefundNote = "رجوع عمولة طلب واتساب مسترجع"
)

// whatsAppGoodsSYP **ثمنُ البضاعة بالليرة** — والدولارُ (بالسنتات) بسعر المتجر مدوَّراً لأقرب ٥
// كـ`catalog.USDToSYP` (لا تُستورد: `catalog` تستورد هذه الحزمة). **و`false` بلا سعر.**
func whatsAppGoodsSYP(goods int64, currency string, rate *float64) (int64, bool) {
	if currency != "USD" {
		return goods, true
	}
	if rate == nil || *rate <= 0 {
		return 0, false
	}
	return int64(math.Round(float64(goods)**rate/100/5)) * 5, true
}

// chargeWhatsAppCommission **يقيّد العمولةَ عند التسليم — مرّةً واحدة.**
//
// **والتكرارُ يُعرف من الدفتر**: قيدُ دفعٍ بمرجع الطلب أو دينٌ بسببه ⇒ قُيّدت.
func (s *Service) chargeWhatsAppCommission(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	var merchantID, currency string
	var goods *int64
	var override *pricing.Percent2
	var rate *float64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(o.pickup_merchant_id::text, ''), o.collect_amount, o.amount_currency,
		       m.commission_percent, m.exchange_rate::float8
		  FROM orders o LEFT JOIN merchants m ON m.id = o.pickup_merchant_id
		 WHERE o.id = $1 AND o.origin = 'whatsapp'`, orderID).
		Scan(&merchantID, &goods, &currency, &override, &rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if merchantID == "" || goods == nil || *goods <= 0 {
		return nil
	}
	var booked bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM wallet_transactions WHERE ref = $1 AND kind = 'order_payment')
		    OR EXISTS (SELECT 1 FROM financial_obligations WHERE order_id = $1::uuid AND cause = $2)`,
		orderID, obligations.CauseWhatsAppCommission).Scan(&booked); err != nil {
		return err
	}
	if booked {
		return nil
	}
	base, ok := whatsAppGoodsSYP(*goods, currency, rate)
	if !ok {
		s.logger.Warn("عمولة طلب واتساب: ثمنٌ بالدولار ومتجرٌ بلا سعر صرف — لا عمولة",
			"order", orderID, "merchant", merchantID)
		return nil
	}
	snap, err := s.snapshotOf(ctx, q, orderID)
	if err != nil {
		return err
	}
	// **والنسبةُ لا عدمَ فيها** (`orderPct`) — فلا يُقرأ إعدادٌ من معاملة الانتقال (`XG-48`).
	commission := pricing.MerchantCommission(ctx, nil, orderPct(override, snap)).Of(base)
	if commission <= 0 {
		return nil
	}
	paid, err := s.takeFromMerchantWallet(ctx, q, orderID, merchantID, actorID, commission,
		whatsAppCommissionNote)
	if err != nil {
		return err
	}
	if !paid {
		var actor *string
		if actorID != "" {
			actor = &actorID
		}
		if _, err := obligations.Create(ctx, q, obligations.PartyMerchant, merchantID, commission,
			obligations.CauseWhatsAppCommission, orderID, actor); err != nil {
			return err
		}
	}
	_, err = q.Exec(ctx, `UPDATE orders SET platform_commission = $2 WHERE id = $1`, orderID, commission)
	return err
}

// returnWhatsAppCommission **استرجاعُ طلب واتسابٍ مسلَّم: العمولةُ تعود للمتجر** — مرّةً واحدة.
//
// **ما أُخذ من محفظته وما سدّه من الدين بمالٍ حقيقيّ** (أرباحٌ أو شحنٌ أو نقدٌ بالمكتب) ⇐ `refund`
// لمحفظة صاحبه. **والباقي من الدين يُسقَط**، والمشطوبُ لم يدفعه فلا يُردّ.
func (s *Service) returnWhatsAppCommission(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	var merchantID, ownerID string
	err := q.QueryRow(ctx, `
		SELECT COALESCE(o.pickup_merchant_id::text, ''), COALESCE(m.owner_user_id::text, '')
		  FROM orders o LEFT JOIN merchants m ON m.id = o.pickup_merchant_id
		 WHERE o.id = $1 AND o.origin = 'whatsapp'`, orderID).Scan(&merchantID, &ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if merchantID == "" || ownerID == "" {
		return nil
	}
	var taken, settled int64
	var refunded bool
	if err := q.QueryRow(ctx, `
		SELECT -COALESCE((SELECT sum(amount) FROM wallet_transactions
		                   WHERE ref = $1 AND kind = 'order_payment'), 0),
		       COALESCE((SELECT sum(s.amount) FROM obligation_settlements s
		                   JOIN financial_obligations f ON f.id = s.obligation_id
		                  WHERE f.order_id = $1::uuid AND f.cause = $2
		                    AND s.method IN ($3, $4, $5)), 0),
		       EXISTS (SELECT 1 FROM wallet_transactions WHERE ref = $1 AND kind = 'refund')`,
		orderID, obligations.CauseWhatsAppCommission,
		obligations.MethodEarning, obligations.MethodTopup, obligations.MethodOfficeCash).
		Scan(&taken, &settled, &refunded); err != nil {
		return err
	}
	actor := &actorID
	if actorID == "" {
		actor = nil
	}
	if err := obligations.VoidForOrder(ctx, q, obligations.PartyMerchant, merchantID, orderID,
		obligations.CauseWhatsAppCommission, actor); err != nil {
		return err
	}
	if back := taken + settled; back > 0 && !refunded {
		if _, err := s.wallet.ApplyTx(ctx, q, ownerID, back, "refund", orderID,
			whatsAppCommissionRefundNote, actor); err != nil {
			return err
		}
	}
	if _, err := q.Exec(ctx, `UPDATE orders SET platform_commission = 0 WHERE id = $1`, orderID); err != nil {
		return err
	}
	return s.creditTreasury(ctx, q, orderID, actorID)
}
