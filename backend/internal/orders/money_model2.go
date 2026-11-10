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

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/obligations"
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
	delta := (paid - refunded - toParties) - posted
	if delta == 0 {
		return nil
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, delta, "platform_profit",
		orderID, "تسويةُ نصيب المنصة", &actorID)
	return err
}
