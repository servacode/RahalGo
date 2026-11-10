package orders

// ══════════════════════════════════════════════════════════════════════
// **النموذج ٢: البضاعةُ خارجَ كلّ حساب** — خطّةُ المال ٢٠٢٦-١١-١٠ (الدفعة B4)
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ `money_model = 2`: **الكابتن يدفع للمتجر من جيبه عند الاستلام** (الثمنُ ناقصَ العمولة)
// **ويقبض من الزبون عند التسليم.** والمنصّةُ تلمس الأجرةَ والعمولةَ (مع الهامش) وحدَهما:
//
//	صفُّ التسوية   سجلٌّ بلا دفتر (`off_ledger`): «الكابتن دفع للمتجر X» — لا احتباسَ ولا دفعَ
//	               (`merchant_cash_accrued` / `merchant_cash_paid`) ولا مستحقَّ محفظة.
//	صندوقُ الكابتن يقبض `cash_due − ما دفعه للمتجر` (الأجرة + العمولة + الهامش) — ولا سلفةَ درج.
//	الخزينة        = المدفوعُ من المحفظة + ما قبضه الصندوقُ − المردود − أجرُ الكابتن − نصيبُ المندوب
//	               (`creditTreasuryGoodsOutside`) — **ولا حدَّ للبضاعة فيها**، فلا تهبط تحت الصفر
//	               بين الاستلام والتسليم.
//
// **والنموذج ١ لا يُمسّ** — كلُّ فرعٍ هنا يُسأل بعمود الطلب، والقديمُ على مساره حرفاً.

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/pricing"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// orderMoneyModel **نموذجُ مال الطلب** كما خُتم عند إنشائه.
func orderMoneyModel(ctx context.Context, q wallet.Querier, orderID string) (int16, error) {
	var m int16
	err := q.QueryRow(ctx, `SELECT money_model FROM orders WHERE id = $1`, orderID).Scan(&m)
	if errors.Is(err, pgx.ErrNoRows) {
		return MoneyModelLegacy, nil
	}
	return m, err
}

// goodsOutside **أهو طلبٌ من النموذج ٢؟** — والخطأُ يُعامَل كالقديم (لا يُغيّر سلوكَ مال).
func goodsOutside(ctx context.Context, q wallet.Querier, orderID string) (bool, error) {
	m, err := orderMoneyModel(ctx, q, orderID)
	return m == MoneyModelGoodsOutside, err
}

// settleMerchantRecord **صفُّ تسويةٍ سجلٌّ بلا دفتر** — النموذج ٢ (انظر رأسَ الملفّ).
//
//   - يُنشأ نقداً `cash_due` بلا قيد، **ودينُ متجرٍ قديمٍ يُقتطع منه** (قرارُ المالك: الاقتطاعُ
//     التلقائيُّ عند الاستلام يستمرّ حتّى يُسدَّد الدينُ القديم) — سطرُ عكسٍ بلا قيد وسطرُ
//     تسويةِ التزام. **فيدفع الكابتنُ للمتجر أقلَّ ويقبض صندوقُه الفرقَ للمكتب.**
//   - **والكابتنُ دفع الباقي الآن** ⇒ `cash_paid` بجهة `driver` — سجلٌّ يقرؤه تطبيقُه وصندوقُه.
//
// **ومُعادُ النداء لا يكرّر** (هُويّةُ `(order_id, merchant_id)`) — ويُتحقَّق مبلغُه.
func (s *Service) settleMerchantRecord(ctx context.Context, q wallet.Querier, orderID string,
	sh sourceShare, driverID *string, actorID string) error {
	var settleID string
	err := q.QueryRow(ctx, `
		INSERT INTO merchant_settlements
		       (order_id, merchant_id, owner_user_id, method, amount, state, commission_percent, off_ledger)
		VALUES ($1, $2, $3, 'cash', $4, 'cash_due', $5, true)
		ON CONFLICT (order_id, merchant_id) DO NOTHING
		RETURNING id::text`,
		orderID, sh.merchantID, *sh.ownerID, sh.due, sh.pct).Scan(&settleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return validateSettlementRecord(ctx, q, orderID, sh.merchantID, sh.due)
	}
	if err != nil {
		return err
	}

	// ── دينُ المتجر القديم يُقتطع من حقّه (بلا قيد) ──
	debt, err := obligations.Balance(ctx, q, obligations.PartyMerchant, sh.merchantID)
	if err != nil {
		return err
	}
	take := min(max(debt, 0), sh.due)
	if take > 0 {
		if _, err := q.Exec(ctx, `
			INSERT INTO merchant_settlement_reversals (settlement_id, tx_id, amount, cause, event_ref)
			VALUES ($1, NULL, $2, 'debt_offset', $3)`, settleID, take, "debt:"+orderID); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE merchant_settlements
			   SET reversed_amount = reversed_amount + $2,
			       state = CASE WHEN reversed_amount + $2 >= amount THEN 'cash_reversed' ELSE state END
			 WHERE id = $1`, settleID, take); err != nil {
			return err
		}
		applied, err := obligations.Settle(ctx, q, obligations.PartyMerchant,
			sh.merchantID, take, orderID, 0, &actorID)
		if err != nil {
			return err
		}
		if applied != take {
			return fmt.Errorf("تسويةُ دَينِ متجرٍ (سجلّ) لم تُطابق: اقتُطع %d وسُوّي %d", take, applied)
		}
	}

	// ── والكابتنُ دفع الباقي من جيبه ──
	if driverID != nil && sh.due-take > 0 {
		if _, err := q.Exec(ctx, `
			UPDATE merchant_settlements ms
			   SET state = 'cash_paid', paid_by = $2, paid_via = 'driver', paid_at = now(),
			       paid_owner_user_id = (SELECT owner_user_id FROM merchants WHERE id = ms.merchant_id),
			       note = 'دفعه الكابتن من جيبه عند الاستلام'
			 WHERE id = $1 AND state = 'cash_due'`, settleID, *driverID); err != nil {
			return err
		}
	}
	return nil
}

// validateSettlementRecord **صفُّ سجلٍّ قائم يُطابَق ولا يُتجاوَز** — كنظيره `validateSettlement`.
func validateSettlementRecord(ctx context.Context, q wallet.Querier, orderID, merchantID string, due int64) error {
	var offLedger bool
	var amount int64
	if err := q.QueryRow(ctx, `
		SELECT off_ledger, amount + adjusted_amount FROM merchant_settlements
		WHERE order_id = $1 AND merchant_id = $2 FOR UPDATE`, orderID, merchantID).
		Scan(&offLedger, &amount); err != nil {
		return err
	}
	if !offLedger {
		return fmt.Errorf("XS-1 صفُّ تسويةٍ بالدفتر على طلبٍ من النموذج ٢ (طلب %s متجر %s)", orderID, merchantID)
	}
	if amount != due {
		return fmt.Errorf("XS-1 صفُّ سجلٍّ بمبلغٍ مخالف (طلب %s متجر %s): %d ≠ %d", orderID, merchantID, amount, due)
	}
	return nil
}

// creditTreasuryGoodsOutside **نصيبُ المنصّة من طلبٍ في النموذج ٢** — من الوقائع وحدَها:
//
//	المدفوعُ  = قيودُ الدفع من المحفظة (`order_payment`) + ما قبضه صندوقُ الكابتن عنه
//	            (`order_collection` + `commission_waived` + `collection_reversed`)
//	الربحُ    = المدفوع − المردود − أجرُ الكابتن − نصيبُ المندوب − ما قُيّد سابقاً
//
// **ولا حدَّ للبضاعة**: لا `merchant_*` ولا `custom_goods_return` ولا خسارةَ ولا معادلَ دولار —
// البضاعةُ لم تدخل حساباً. **وبالفرق** كأختها: نداءٌ ثانٍ بلا تغيّرٍ لا يكتب شيئاً.
func (s *Service) creditTreasuryGoodsOutside(ctx context.Context, q wallet.Querier, tid, orderID, actorID string) error {
	var paid, toParties, refunded, posted int64
	if err := q.QueryRow(ctx, `
		SELECT
			COALESCE((SELECT -sum(amount) FROM wallet_transactions
			           WHERE ref = $1 AND kind = 'order_payment'), 0)
			+ COALESCE((SELECT sum(amount) FROM driver_cash_entries
			             WHERE ref = $1 AND kind IN ('order_collection', 'commission_waived',
			                                         'collection_reversed')), 0),
			COALESCE((SELECT sum(amount) FROM wallet_transactions
			           WHERE ref = $1 AND kind IN ('driver_earning', 'commission')), 0),
			COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'refund'), 0),
			COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'platform_profit'), 0)`,
		orderID).Scan(&paid, &toParties, &refunded, &posted); err != nil {
		return err
	}
	// **وما ردّه الدرجُ للكابتن من تحصيلٍ لم يقع** (تصحيحُ تسليمٍ خاطئ بعد أن سلّم نقدَه للمكتب) —
	// نقدٌ خرج من المنصّة فيُطرح من المقبوض.
	var drawerBack int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		 WHERE ref = $1 AND source = 'undelivered_return'`, orderID).Scan(&drawerBack); err != nil {
		return err
	}
	paid -= drawerBack
	delta := (paid - refunded - toParties) - posted
	if delta == 0 {
		return nil
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, delta, "platform_profit",
		orderID, "تسويةُ نصيب المنصة", &actorID)
	return err
}

// ══════════════════════════════════════════════════════════════════════
// **البضاعةُ الراجعةُ بعد الاستلام — النموذج ٢** (القرار ٤، الدفعة B5)
// ══════════════════════════════════════════════════════════════════════
//
//	إلى المتجر   صفوفُ السجلّ `goods_returned` — والمتجرُ يردّ للكابتن ما دفعه (برّا الحسابات).
//	إلى المكتب   **المتجرُ رفض** (أو بضاعةٌ من عدّة متاجر) — صفوفُ السجلّ `held_by_driver`،
//	             و`stuck_goods_amount` = ما دفعه الكابتنُ ولم يرجع له: «مصاري الكابتن عالقة»،
//	             والمكتبُ يقرّر (`DecideStuckGoodsTx`).
//
// **ولا خزينةَ ولا خسارةَ ولا دينَ بضاعة** — المنصّةُ لم تشترِ شيئاً.
func (s *Service) settleGoodsOutside(ctx context.Context, q wallet.Querier, orderID, to string) error {
	state := "goods_returned"
	if to == GoodsToOffice {
		state = "held_by_driver"
	}
	if _, err := q.Exec(ctx, `
		UPDATE merchant_settlements SET state = $2
		 WHERE order_id = $1 AND off_ledger AND state IN ('cash_due', 'cash_paid')`,
		orderID, state); err != nil {
		return err
	}
	stuck := int64(0)
	if to == GoodsToOffice {
		// **ما دفعه الكابتنُ للمتجر** — صفوفُ السجلّ (العاديّ) أو ما دفعه في المشتريات.
		if err := q.QueryRow(ctx, `
			SELECT COALESCE((SELECT sum(ms.amount + ms.adjusted_amount - ms.reversed_amount)
			                   FROM merchant_settlements ms
			                  WHERE ms.order_id = o.id AND ms.off_ledger AND ms.paid_via = 'driver'), 0)
			       + CASE WHEN o.kind = 'custom' THEN COALESCE(o.custom_store_paid, 0) ELSE 0 END
			  FROM orders o WHERE o.id = $1`, orderID).Scan(&stuck); err != nil {
			return err
		}
		if stuck < 0 {
			stuck = 0
		}
	}
	returned := "NULL"
	if to == GoodsToMerchant {
		returned = "now()"
	}
	_, err := q.Exec(ctx, `UPDATE orders SET goods_settled_to = $2, stuck_goods_amount = $3, returned_at = `+
		returned+` WHERE id = $1`, orderID, to, stuck)
	return err
}

// قرارا المكتب على «مصاري الكابتن عالقة».
const (
	// StuckByPlatform **المنصّةُ تعوّض الكابتن** — من الخزينة إلى محفظته.
	StuckByPlatform = "platform"
	// StuckByCustomer **دينٌ على الزبون** — سجلٌّ يُقرأ في تقرير البضاعة، ولا مالَ يتحرّك الآن.
	StuckByCustomer = "customer"
)

// StuckGoodsResult ما وقع — للردّ والتدقيق.
type StuckGoodsResult struct {
	OrderID  string `json:"order_id"`
	Number   int64  `json:"number"`
	Amount   int64  `json:"amount"`
	Decision string `json:"decision"`
	DriverID string `json:"-"`
}

var (
	// ErrStuckGoodsNone لا مصاري عالقة للكابتن على هذا الطلب.
	ErrStuckGoodsNone = httpx.NewError(http.StatusConflict, "stuck_goods_none", "errors.stuck_goods_none")
	// ErrStuckGoodsDecided قُرّر سابقاً — والثاني لا يدفع مرّتين.
	ErrStuckGoodsDecided = httpx.NewError(http.StatusConflict, "stuck_goods_decided", "errors.stuck_goods_decided")
	// ErrStuckGoodsBadDecision القرارُ واحدٌ من اثنين.
	ErrStuckGoodsBadDecision = httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
)

// DecideStuckGoodsTx **المكتبُ يقرّر على مصاري الكابتن العالقة** — في معاملة المستدعي.
//
//	platform  تعويضُ الكابتن من الخزينة (`compensation` له و`platform_expense` على الخزينة،
//	          بمرجع `stuck:<الطلب>` — فلا يُخلط بتعويض مشواره).
//	customer  سجلٌّ: الزبونُ مدينٌ بالمبلغ — لا مالَ يتحرّك (يظهر في تقرير البضاعة).
//
// **ومرّةً واحدة** — القرارُ يُقفل الطلب.
func (s *Service) DecideStuckGoodsTx(ctx context.Context, q wallet.Querier, orderID, decision, actorID string) (StuckGoodsResult, error) {
	res := StuckGoodsResult{OrderID: orderID, Decision: decision}
	if decision != StuckByPlatform && decision != StuckByCustomer {
		return res, ErrStuckGoodsBadDecision
	}
	var driverID, decided *string
	err := q.QueryRow(ctx, `
		SELECT number, stuck_goods_amount, stuck_goods_decision, driver_id::text
		  FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&res.Number, &res.Amount, &decided, &driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		return res, httpx.ErrNotFound
	}
	if err != nil {
		return res, err
	}
	if decided != nil {
		return res, ErrStuckGoodsDecided
	}
	if res.Amount <= 0 || driverID == nil {
		return res, ErrStuckGoodsNone
	}
	res.DriverID = *driverID
	if decision == StuckByPlatform {
		ref := "stuck:" + orderID
		if _, err := s.wallet.ApplyTx(ctx, q, *driverID, res.Amount, "compensation", ref,
			"تعويض المنصّة عن بضاعة رفض المتجر يرجّع ثمنها", &actorID); err != nil {
			return res, err
		}
		if err := s.DebitTreasury(ctx, q, res.Amount, ref,
			"مصاري الكابتن العالقة — بضاعة رفض المتجر يرجّع ثمنها", actorID); err != nil {
			return res, err
		}
	}
	_, err = q.Exec(ctx, `
		UPDATE orders SET stuck_goods_decision = $2, stuck_goods_decided_at = now(),
		       stuck_goods_decided_by = NULLIF($3, '')::uuid
		 WHERE id = $1`, orderID, decision, actorID)
	return res, err
}

// refundDeliveredFeeOnly **استرجاعُ طلبٍ عاديٍّ مُسلَّمٍ في النموذج ٢** (القرار ٦، الدفعة B6).
//
//	محفظةُ الزبون  ← ما دفعه للتوصيل فعلاً: `delivery_fee` (بعد كود الخصم والتوصيل المجانيّ)،
//	                 مقصوراً على ما دفعه أصلاً — **وصفرٌ لا يُقيَّد**
//	ثمنُ البضاعة   ← لا يُردّ من التطبيق — بين الزبون والمتجر نقداً برّا الحسابات
//	المتجر         ← لا دينَ عليه (`reverseCashSettlements` لا يُنادى) — صفوفُه سجلّ
//	المندوب        ← عمولتُه تبقى (`reverseCommissions` كما في كلّ استرجاع — لا تمسّ إلّا ما قُيّد)
//	الكابتن        ← أجرُه وصندوقُه كما هما
//	الخزينة        ← تتحمّل المردود (`creditTreasury`)
func (s *Service) refundDeliveredFeeOnly(ctx context.Context, q wallet.Querier, in settlement, out *settled) error {
	var fee int64
	var already bool
	if err := q.QueryRow(ctx, `
		SELECT LEAST(GREATEST(delivery_fee, 0), wallet_paid + cash_due),
		       EXISTS(SELECT 1 FROM wallet_transactions
		               WHERE ref = orders.id::text AND kind = 'refund'
		                 AND created_at >= COALESCE(orders.delivered_at, orders.created_at))
		FROM orders WHERE id = $1 FOR UPDATE`, in.orderID).Scan(&fee, &already); err != nil {
		return err
	}
	if fee > 0 && !already {
		if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, fee, "refund",
			in.orderID, "استرجاع أجرة توصيل طلب مُسلَّم", &in.actorID); err != nil {
			return err
		}
		out.credit(in.customerID, fee, t.refunded2, notifications.AppCustomer)
	}
	if err := s.reverseCommissions(ctx, q, in.orderID, in.actorID); err != nil {
		return err
	}
	return s.creditTreasury(ctx, q, in.orderID, in.actorID)
}

// ══════════════════════════════════════════════════════════════════════
// **طلبُ الواتساب في النموذج ٢: العمولةُ عبر الصندوق** (القرار ٢، الدفعة B7)
// ══════════════════════════════════════════════════════════════════════
//
// **لا خصمَ من محفظة متجرنا ولا دينَ عليه**: الكابتنُ يدفع للمتجر الثمنَ ناقصَ العمولة عند
// الاستلام (`payWhatsAppStoreFor` يقول له كم)، ويقبض من المستلِم الثمنَ والأجرة، **فصندوقُه يقبض
// الأجرةَ والعمولةَ** عند التسليم. وثمنٌ بالدولار: يدفع للمتجر دولاراً ناقصَ عمولته بالسنتات،
// **والعمولةُ بالليرة في الصندوق من معادله** بالسعر الملقوط.

// whatsAppCut **ثمنُ طلب الواتساب وعمولةُ متجرنا عليه** — بلا أثر (حسابٌ وحدَه).
type whatsAppCut struct {
	merchantID, merchantName, currency string
	goods                              int64 // بعملته: ليرةٌ أو سنتات
	commission                         int64 // بالليرة
	pct                                *pricing.Percent2
}

func (s *Service) whatsAppCutFor(ctx context.Context, q wallet.Querier, orderID string) (*whatsAppCut, error) {
	var c whatsAppCut
	var goods *int64
	var override *pricing.Percent2
	var rate *float64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(o.pickup_merchant_id::text, ''), COALESCE(m.name, ''), o.collect_amount,
		       o.amount_currency, m.commission_percent, COALESCE(o.goods_fx_rate, m.exchange_rate)::float8
		  FROM orders o LEFT JOIN merchants m ON m.id = o.pickup_merchant_id
		 WHERE o.id = $1 AND o.origin = 'whatsapp'`, orderID).
		Scan(&c.merchantID, &c.merchantName, &goods, &c.currency, &override, &rate)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if c.merchantID == "" || goods == nil || *goods <= 0 {
		return nil, nil
	}
	c.goods = *goods
	if c.currency == CurrencyUSD && (rate == nil || *rate <= 0) {
		if ref := s.usdReferenceRate(ctx, q); ref > 0 {
			rate = &ref
		}
	}
	base, ok := whatsAppGoodsSYP(c.goods, c.currency, rate)
	if !ok {
		return nil, nil
	}
	snap, err := s.snapshotOf(ctx, q, orderID)
	if err != nil {
		return nil, err
	}
	c.pct = orderPct(override, snap)
	c.commission = pricing.MerchantCommission(ctx, nil, c.pct).Of(base)
	return &c, nil
}

// collectWhatsAppGoodsOutside **تسليمُ طلب واتساب في النموذج ٢** — الأجرةُ والعمولةُ في قيدِ
// تحصيلٍ واحد (`order_collection` مرّةً للطلب)، و`platform_commission` = العمولة.
func (s *Service) collectWhatsAppGoodsOutside(ctx context.Context, q wallet.Querier, in settlement) error {
	cut, err := s.whatsAppCutFor(ctx, q, in.orderID)
	if err != nil {
		return err
	}
	if cut != nil && cut.commission > 0 {
		if _, err := q.Exec(ctx, `UPDATE orders SET platform_commission = $2 WHERE id = $1`,
			in.orderID, cut.commission); err != nil {
			return err
		}
		in.cashDue += cut.commission
	}
	return s.collectMerchantDeliveryCash(ctx, q, in)
}

// payWhatsAppStoreFor **ما يدفعه الكابتنُ لمتجرنا في طلب واتساب من النموذج ٢** — الثمنُ ناقصَ العمولة.
func (s *Service) payWhatsAppStoreFor(ctx context.Context, q dbtx.Querier, orderID string) ([]StorePay, error) {
	var model int16
	var origin string
	if err := q.QueryRow(ctx, `SELECT money_model, origin FROM orders WHERE id = $1`, orderID).
		Scan(&model, &origin); err != nil {
		return nil, err
	}
	if model != MoneyModelGoodsOutside || origin != "whatsapp" {
		return nil, nil
	}
	cut, err := s.whatsAppCutFor(ctx, q, orderID)
	if err != nil || cut == nil {
		return nil, err
	}
	p := StorePay{MerchantID: cut.merchantID, MerchantName: cut.merchantName, Amount: cut.goods - cut.commission}
	if cut.currency == CurrencyUSD {
		p.Currency = CurrencyUSD
		p.Amount = cut.goods - pricing.MerchantCommission(ctx, nil, cut.pct).Of(cut.goods)
	}
	if p.Amount <= 0 {
		return nil, nil
	}
	return []StorePay{p}, nil
}
