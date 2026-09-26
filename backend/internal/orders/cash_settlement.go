package orders

// ══════════════════════════════════════════════════════════════════════
// **تسويةُ المتجر نقداً — الاحتباسُ والدفعُ والعكس** — دورةُ ٢٠٢٦-٠٩-٢٧
// ══════════════════════════════════════════════════════════════════════
//
// المتجرُ ذو طريقةِ التسوية «نقد» لا يُقيَّد له في محفظته. **يُقيَّد التزامٌ
// نقديٌّ في محفظةِ الاحتباس النظاميّة** (`is_cash_holding`) بالنوع
// `merchant_cash_accrued` — **تراه الخزينةُ في `toParties` فلا تحسبه ربحاً** —
// ويُنشَأ صفُّ `merchant_settlements(state=cash_due)`. حين يؤكّد الأدمنُ الدفعَ
// نقداً يُخصَم من الاحتباس بالنوع `merchant_cash_paid` وتصير الحالةُ `cash_paid`.
//
// **والهُويّةُ `(order_id, merchant_id)` نفسُها للطريقتين** (محفظةً ونقداً) —
// فالمحفظةُ والنقدُ لا يجتمعان لمصدرٍ واحد (XOR بنيويّ).

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrSettlementNotFound لا تسويةَ بهذا المُعرّف.
	ErrSettlementNotFound = httpx.NewError(http.StatusNotFound, "settlement_not_found", "errors.settlement_not_found")
	// ErrSettlementNotCash التسويةُ ليست نقديّةً — لا تُدفَع نقداً.
	ErrSettlementNotCash = httpx.NewError(http.StatusConflict, "settlement_not_cash", "errors.settlement_not_cash")
	// ErrSettlementNotPayable لا مستحقَّ نقديّاً قائماً لهذه التسوية.
	ErrSettlementNotPayable = httpx.NewError(http.StatusConflict, "settlement_not_payable", "errors.settlement_not_payable")
)

// cashHoldingOn مالكُ محفظةِ الاحتباس النظاميّة — **وهي مبذورةٌ حتماً.**
func (s *Service) cashHoldingOn(ctx context.Context, q wallet.Querier) (string, error) {
	var id string
	err := q.QueryRow(ctx,
		`SELECT user_id::text FROM wallets WHERE is_cash_holding LIMIT 1`).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("XS-2 لا محفظةَ احتباسٍ للمستحقّات النقديّة — يلزم EnsureCashHolding")
	}
	return id, err
}

// initialState الحالةُ الابتدائيّةُ لصفّ التسوية بحسب الطريقة.
func initialState(method string) string {
	if method == "wallet" {
		return "wallet_credited"
	}
	return "cash_due"
}

// validateSettlement **تعارضُ الهُويّة يُتحقَّق ولا يُتجاوَز عمياءَ** (البند ١).
//
// حين يجد `settleMerchant` تسويةً قائمةً لـ(طلب، متجر) — إعادةُ نداءٍ صحيحةٌ —
// **يُقفل الصفُّ ويُطابَق مع الحقيقةِ المُعادِ حسابُها**: الطريقةُ واللقطةُ،
// والمبلغُ الصافي المُعاد، وتماسكُ الطريقة/الحالة، وحضورُ قيدِ الدفتر المطابق
// للحالة. **أيُّ اختلافٍ يُسقط المعاملةَ بخطأ ثابتٍ — لا يُحسَب إعادةً ناجحة.**
func (s *Service) validateSettlement(ctx context.Context, q wallet.Querier,
	orderID, merchantID, method string, due int64) error {
	var exMethod, exState string
	var exAmount int64
	var earningTx, accruedTx, paidTx *int64
	err := q.QueryRow(ctx, `
		SELECT method, state, amount, earning_tx_id, accrued_tx_id, paid_tx_id
		FROM merchant_settlements
		WHERE order_id = $1 AND merchant_id = $2
		FOR UPDATE`, orderID, merchantID).
		Scan(&exMethod, &exState, &exAmount, &earningTx, &accruedTx, &paidTx)
	if err != nil {
		return err
	}
	if exMethod != method {
		return fmt.Errorf("XS-1 تسويةٌ قائمةٌ بطريقةٍ مخالفة (طلب %s متجر %s): %q ≠ %q",
			orderID, merchantID, exMethod, method)
	}
	if exAmount != due {
		return fmt.Errorf("XS-1 تسويةٌ قائمةٌ بمبلغٍ مخالف (طلب %s متجر %s): %d ≠ %d",
			orderID, merchantID, exAmount, due)
	}
	switch exMethod {
	case "wallet":
		if exState != "wallet_credited" && exState != "wallet_reversed" {
			return fmt.Errorf("XS-1 حالةٌ محفظيّةٌ فاسدة: %q", exState)
		}
		if exState == "wallet_credited" && earningTx == nil {
			return fmt.Errorf("XS-1 تسويةٌ محفظيّةٌ بلا قيدِ مستحقّ (طلب %s متجر %s)", orderID, merchantID)
		}
	case "cash":
		if exState != "cash_due" && exState != "cash_paid" && exState != "cash_reversed" {
			return fmt.Errorf("XS-1 حالةٌ نقديّةٌ فاسدة: %q", exState)
		}
		if accruedTx == nil {
			return fmt.Errorf("XS-1 تسويةٌ نقديّةٌ بلا قيدِ احتباس (طلب %s متجر %s)", orderID, merchantID)
		}
		if exState == "cash_paid" && paidTx == nil {
			return fmt.Errorf("XS-1 تسويةٌ مدفوعةٌ بلا قيدِ دفع (طلب %s متجر %s)", orderID, merchantID)
		}
	default:
		return fmt.Errorf("XS-1 طريقةٌ مجهولة: %q", exMethod)
	}
	return nil
}

// accrueCashSettlement يقيّد التزامَ المتجرِ النقديَّ في الاحتباس ثمّ يقتطع دَينَه.
//
// **يُنادى مرّةً واحدةً لكلّ صفِّ تسويةٍ نُشئ حديثاً** (بوّابةُ الهُويّة تضمن ذلك).
func (s *Service) accrueCashSettlement(ctx context.Context, q wallet.Querier,
	settleID, orderID, merchantID, ownerID string, due int64, actorID string) error {
	hid, err := s.cashHoldingOn(ctx, q)
	if err != nil {
		return err
	}
	_, txID, err := s.wallet.ApplyTxID(ctx, q, hid, due, "merchant_cash_accrued",
		orderID, "مستحقٌّ نقديٌّ للمتجر عن بضاعةٍ سُلّمت للسائق", &actorID)
	if err != nil {
		return err
	}
	if _, err := q.Exec(ctx,
		`UPDATE merchant_settlements SET accrued_tx_id = $2 WHERE id = $1`,
		settleID, txID); err != nil {
		return err
	}
	// **اقتطاعُ الدَّينِ من المستحقّ النقديّ** — كنظيرِه المحفظيّ (البند ٢): نصيبٌ
	// أُنقص لا مالٌ اختُلق. الالتزامُ يُسدَّد بقيدٍ سالبٍ في الاحتباس (يتصافى في
	// `toParties` فترتفع الخزينةُ بالمُقتطَع)، **وصفُّ التسوية يُحدَّث في المعاملة
	// نفسِها** فيبقى المستحقُّ القائم = صافي التزامِ الاحتباس (البند الإلزاميّ).
	return s.offsetMerchantDebtCash(ctx, q, settleID, merchantID, hid, due, orderID, actorID)
}

// offsetMerchantDebtCash يقتطع من مستحقٍّ نقديٍّ جديدٍ ما بقي على المتجر من دَين.
func (s *Service) offsetMerchantDebtCash(ctx context.Context, q wallet.Querier,
	settleID, merchantID, hid string, available int64, orderID, actorID string) error {
	debt, err := obligations.Balance(ctx, q, obligations.PartyMerchant, merchantID)
	if err != nil {
		return err
	}
	if debt <= 0 {
		return nil
	}
	take := debt
	if available < take {
		take = available
	}
	if take <= 0 {
		return nil
	}
	txID, err := s.postCashReversal(ctx, q, settleID, hid, take,
		"debt_offset", "debt:"+orderID, orderID, actorID)
	if err != nil {
		return err
	}
	if txID == 0 {
		// **حُوسب سلفاً** — لا اقتطاعَ ثانٍ (الالتزامُ سُوّي في المعاملة الأصليّة).
		return nil
	}
	applied, err := obligations.Settle(ctx, q, obligations.PartyMerchant,
		merchantID, take, orderID, txID, &actorID)
	if err != nil {
		return err
	}
	if applied != take {
		return fmt.Errorf("تسويةُ دَينِ متجرٍ نقداً لم تُطابق: اقتُطع %d وسُوّي %d", take, applied)
	}
	return nil
}

// postCashReversal يقيّد عكساً (سالباً) في الاحتباس بالنوع نفسِه، ويُنشئ سطرَ
// عكسٍ مضافاً، ويحدّث `reversed_amount`/`state` — **في المعاملة نفسِها.**
//
// **مُعادُ الاستدعاءِ لا يُكرّر**: (تسوية، حدث) موجودٌ ⇒ لا شيء (يُرجع 0).
// **ولا يتجاوز القائم**: قيدُ `balance >= 0` على الاحتباس + `reversed <= amount`.
func (s *Service) postCashReversal(ctx context.Context, q wallet.Querier,
	settleID, hid string, amount int64, cause, eventRef, orderID, actorID string) (int64, error) {
	if amount <= 0 {
		return 0, nil
	}
	var exists bool
	if err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM merchant_settlement_reversals
		               WHERE settlement_id = $1 AND event_ref = $2)`,
		settleID, eventRef).Scan(&exists); err != nil {
		return 0, err
	}
	if exists {
		return 0, nil
	}
	_, txID, err := s.wallet.ApplyTxID(ctx, q, hid, -amount, "merchant_cash_accrued",
		orderID, cashReversalNote(cause), &actorID)
	if err != nil {
		return 0, err
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO merchant_settlement_reversals (settlement_id, tx_id, amount, cause, event_ref)
		VALUES ($1, $2, $3, $4, $5)`,
		settleID, txID, amount, cause, eventRef); err != nil {
		return 0, err
	}
	if _, err := q.Exec(ctx, `
		UPDATE merchant_settlements
		   SET reversed_amount = reversed_amount + $2,
		       state = CASE WHEN reversed_amount + $2 >= amount THEN 'cash_reversed' ELSE state END
		 WHERE id = $1`, settleID, amount); err != nil {
		return 0, err
	}
	return txID, nil
}

func cashReversalNote(cause string) string {
	switch cause {
	case "debt_offset":
		return "اقتطاعُ دَينٍ سابقٍ من مستحقٍّ نقديّ"
	case "returned_goods":
		return "عكسُ مستحقٍّ نقديّ عن بضاعةٍ رُدّت"
	default:
		return "عكسُ مستحقٍّ نقديّ عن طلبٍ مُسترجَع"
	}
}

// reverseCashSettlements يعكس المستحقّاتِ النقديّةَ عند استرجاعِ طلبٍ مُسلَّم.
//
//   - `cash_due` (لم يُدفَع بعد): يُعكَس القائمُ في الاحتباس، لا التزام. (الحالة C)
//   - `cash_paid` (دُفع نقداً فعلاً): **لا يُعاد كتابةُ الدفع** — يُنشَأ التزامُ
//     متجرٍ للمنصة بالمبلغ المدفوع (يُسترَدّ من مستحقٍّ قادم). (الحالة D)
//   - `cash_reversed`: لا شيء.
func (s *Service) reverseCashSettlements(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	rows, err := q.Query(ctx, `
		SELECT id::text, merchant_id::text, amount - reversed_amount, state
		FROM merchant_settlements
		WHERE order_id = $1 AND method = 'cash'
		FOR UPDATE`, orderID)
	if err != nil {
		return err
	}
	type row struct {
		id          string
		merchantID  string
		outstanding int64
		state       string
	}
	var list []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.merchantID, &r.outstanding, &r.state); err != nil {
			rows.Close()
			return err
		}
		list = append(list, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if len(list) == 0 {
		return nil
	}
	hid, err := s.cashHoldingOn(ctx, q)
	if err != nil {
		return err
	}
	for _, r := range list {
		switch r.state {
		case "cash_due":
			if _, err := s.postCashReversal(ctx, q, r.id, hid, r.outstanding,
				"refund_full", "refund:"+orderID, orderID, actorID); err != nil {
				return err
			}
		case "cash_paid":
			// **المدفوعُ = amount − reversed_amount** — يُنشَأ التزامٌ لا يُعاد كتابة.
			if r.outstanding > 0 {
				if _, err := obligations.Create(ctx, q, obligations.PartyMerchant,
					r.merchantID, r.outstanding, obligations.CauseRefundMerchant,
					orderID, &actorID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ── سطحُ الأدمن: قراءةُ المستحقّات النقديّة وتأكيدُ دفعِها ───────────────

// CashSettlementRow صفٌّ في كشفِ المستحقّات النقديّة لمتجر.
type CashSettlementRow struct {
	ID          string     `json:"id"`
	OrderID     string     `json:"order_id"`
	Amount      int64      `json:"amount"`
	Reversed    int64      `json:"reversed_amount"`
	Outstanding int64      `json:"outstanding"`
	State       string     `json:"state"`
	PaidBy      *string    `json:"paid_by,omitempty"`
	PaidAt      *time.Time `json:"paid_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// MerchantCashSummary مُلخّصُ المستحقّات النقديّة لمتجر — للأدمن.
type MerchantCashSummary struct {
	OutstandingTotal int64               `json:"outstanding_total"`
	Settlements      []CashSettlementRow `json:"settlements"`
}

// MerchantCashSettlements كشفُ المستحقّات النقديّة لمتجرٍ والمجموعُ القائم.
func (s *Service) MerchantCashSettlements(ctx context.Context, merchantID string) (MerchantCashSummary, error) {
	var out MerchantCashSummary
	rows, err := s.db.Query(ctx, `
		SELECT id::text, order_id::text, amount, reversed_amount, state,
		       paid_by::text, paid_at, created_at
		FROM merchant_settlements
		WHERE merchant_id = $1 AND method = 'cash'
		ORDER BY created_at DESC`, merchantID)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var r CashSettlementRow
		if err := rows.Scan(&r.ID, &r.OrderID, &r.Amount, &r.Reversed, &r.State,
			&r.PaidBy, &r.PaidAt, &r.CreatedAt); err != nil {
			return out, err
		}
		r.Outstanding = r.Amount - r.Reversed
		if r.State == "cash_due" {
			out.OutstandingTotal += r.Outstanding
		}
		out.Settlements = append(out.Settlements, r)
	}
	return out, rows.Err()
}

// CashPaidResult نتيجةُ تأكيدِ دفعٍ نقديّ — للتدقيقِ والإشعار في طبقة الخادم.
type CashPaidResult struct {
	SettlementID string
	OrderID      string
	MerchantID   string
	OwnerUserID  string // المالكُ الحاليُّ (من دُفع/يُشعَر)
	Amount       int64  // القائمُ المدفوع
	AlreadyPaid  bool   // إعادةُ تأكيدٍ لمدفوعٍ سلفاً — لا قيدَ ولا دفعةَ ثانية
}

// MarkCashSettlementPaid يؤكّد أنّ الأدمنَ سلّم المتجرَ مستحقَّه نقداً.
//
// **يخصم القائمَ من الاحتباس، ويصير الصفُّ `cash_paid`، ويُسجّل المالكَ الحاليَّ
// لحظةَ الدفع** (قد تكون الملكيّةُ تبدّلت منذ النشأة — فلا يُدفَع/يُشعَر مالكٌ قديم).
// **وإعادةُ التأكيد لمدفوعٍ سلفاً لا تُكرّر شيئاً** (البند ٥).
func (s *Service) MarkCashSettlementPaid(ctx context.Context, settlementID, adminID, note string) (CashPaidResult, error) {
	var res CashPaidResult
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return res, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var orderID, merchantID, method, state string
	var amount, reversed int64
	if err := tx.QueryRow(ctx, `
		SELECT order_id::text, merchant_id::text, method, state, amount, reversed_amount
		FROM merchant_settlements WHERE id = $1 FOR UPDATE`, settlementID).
		Scan(&orderID, &merchantID, &method, &state, &amount, &reversed); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return res, ErrSettlementNotFound
		}
		return res, err
	}
	res.SettlementID, res.OrderID, res.MerchantID = settlementID, orderID, merchantID
	if method != "cash" {
		return res, ErrSettlementNotCash
	}
	if state == "cash_paid" {
		var owner *string
		_ = tx.QueryRow(ctx,
			`SELECT paid_owner_user_id::text FROM merchant_settlements WHERE id = $1`,
			settlementID).Scan(&owner)
		if owner != nil {
			res.OwnerUserID = *owner
		}
		res.Amount = amount - reversed
		res.AlreadyPaid = true
		return res, tx.Commit(ctx)
	}
	if state != "cash_due" {
		return res, ErrSettlementNotPayable // cash_reversed — لا مستحقَّ
	}
	outstanding := amount - reversed
	if outstanding <= 0 {
		return res, ErrSettlementNotPayable
	}
	hid, err := s.cashHoldingOn(ctx, tx)
	if err != nil {
		return res, err
	}
	var currentOwner string
	if err := tx.QueryRow(ctx,
		`SELECT owner_user_id::text FROM merchants WHERE id = $1`, merchantID).
		Scan(&currentOwner); err != nil {
		return res, err
	}
	_, txID, err := s.wallet.ApplyTxID(ctx, tx, hid, -outstanding, "merchant_cash_paid",
		orderID, "تسويةُ مستحقٍّ نقديّ — تأكيدُ دفعٍ من الأدمن", &adminID)
	if err != nil {
		return res, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE merchant_settlements
		   SET state = 'cash_paid', paid_tx_id = $2, paid_by = $3,
		       paid_owner_user_id = $4, paid_at = now(), note = $5
		 WHERE id = $1`, settlementID, txID, adminID, currentOwner, note); err != nil {
		return res, err
	}
	if err := tx.Commit(ctx); err != nil {
		return res, err
	}
	res.OwnerUserID = currentOwner
	res.Amount = outstanding
	return res, nil
}
