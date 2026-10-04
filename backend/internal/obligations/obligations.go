// ══════════════════════════════════════════════════════════════════════
// **الالتزاماتُ الماليّة — نشأةً وتسويةً وإغلاقاً**
// ══════════════════════════════════════════════════════════════════════
//
// (`XG-31` — دورةُ إصلاحٍ ٣، ٢٠٢٦-٠٩-٠٦.)
//
// # ما تحلّه
//
// **كان الالتزامُ عمودَين يُزادان وينقصان** — `merchants.debt` و
// `users.commission_debt`. **ومن رأى ديناً قدرُه ألفٌ ومئتان لم يستطع
// أن يقول من أين.**
//
// **فصار لكلّ التزامٍ واقعةُ نشأةٍ** تحمل الطرفَ والمبلغَ والسببَ
// والطلبَ والوقتَ والمنشئ، **ولكلّ اقتطاعٍ سطرُ تسويةٍ** يحمل ما اقتُطع
// ومن أيّ طلبٍ وما بقي وقيدَه في الدفتر.
//
// # وأيُّهما الحقّ
//
// **`financial_obligations` هي الحقيقة.** **والعمودان صورةٌ محفوظةٌ
// للقراءة** — لا يُكتبان إلّا من هنا، ويحرسهما ثابتٌ دائم.
package obligations

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// أسبابُ النشأة — **وهي نفسُها قيدُ `CHECK` في القاعدة.**
const (
	CauseRefundMerchant = "refund_merchant_earning"
	CauseRefundRep      = "refund_rep_commission"
	CauseReturnedGoods  = "returned_goods"
	CauseLegacy         = "legacy_opening"
	// CauseDeliveryFee أجرةُ «لدي توصيلة» لم تحملها محفظةُ المتجر (هجرة `0170`).
	CauseDeliveryFee = "merchant_delivery_fee"
)

// طرقُ التسوية — **وهي قيدُ `CHECK` في `obligation_settlements.method`** (هجرة `0340`).
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الديون: «يتفرّق بوضوح ما انسدّ من مستحقّ وما
// انشطب لأنّ الطلب ما صار وما دُفع بالمكتب».)
const (
	MethodEarning    = "earning"      // اقتُطع من مستحقٍّ أو عمولةٍ جديدة
	MethodVoided     = "voided"       // أُسقط لأنّ الطلب لم يقع — لا مالَ تحرّك
	MethodOfficeCash = "office_cash"  // دفعه نقداً بالمكتب — دخل صندوقَ المكتب
	MethodTopup      = "wallet_topup" // اقتُطع من شحنِ محفظته — الشحنُ يسدّ الدينَ أوّلاً
	MethodWrittenOff = "written_off"  // شُطب بموافقة مدير المنصّة — خسارةٌ على المنصّة
)

// causeWords **السببُ كما يُقرأ في كشف المحفظة** — بلغة المكتب البسيطة.
var causeWords = map[string]string{
	CauseRefundMerchant: "استرجاع طلب بعد صرف مستحقه",
	CauseRefundRep:      "استرجاع طلب بعد صرف عمولته",
	CauseReturnedGoods:  "بضاعة رجعت",
	CauseLegacy:         "دين سابق",
	CauseDeliveryFee:    "أجرة توصيلة",
}

// CauseWords السببُ بالعربيّ — ومجهولُه «دين».
func CauseWords(cause string) string {
	if w, ok := causeWords[cause]; ok {
		return w
	}
	return "دين"
}

// OffsetNote **وصفُ الاقتطاع في كشف المحفظة — من أسباب ما سيُسدَّد فعلاً.**
//
// (فحصُ قسم الديون ٢٠٢٦-١٠-٠٤، المشكلة ٥: كان الوصفُ ثابتاً «عن بضاعةٍ رُدّت»
// ولو كان الدينُ أجرةَ توصيلة — **فيقرأ المتجرُ في كشفه سبباً لم يقع.**)
//
// **يقرأ الالتزاماتِ المفتوحةَ بالترتيب نفسِه الذي يسدّها به `Settle`** (الأقدمُ
// أوّلاً) حتّى يبلغ `take`، ويجمع أسبابَها بلا تكرار.
func OffsetNote(ctx context.Context, q Querier, partyKind, partyID string, take int64) (string, error) {
	rows, err := q.Query(ctx, `
		SELECT cause, amount - settled FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2 AND closed_at IS NULL
		 ORDER BY created_at, id`, partyKind, partyID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var words []string
	seen := map[string]bool{}
	for rows.Next() && take > 0 {
		var cause string
		var rest int64
		if err := rows.Scan(&cause, &rest); err != nil {
			return "", err
		}
		take -= rest
		w := CauseWords(cause)
		if !seen[w] {
			seen[w] = true
			words = append(words, w)
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(words) == 0 {
		return "اقتطاع دين", nil
	}
	return "اقتطاع دين — " + strings.Join(words, "، "), nil
}

// الأطراف.
const (
	PartyMerchant = "merchant"
	PartyRep      = "rep"
)

// Querier ما تحتاجه الحزمةُ من معاملة — **وهو عقدُ المحفظة نفسُه.**
//
// **ولا واجهةَ ثانيةٌ تُعرَّف**: النشأةُ والتسويةُ تجريان في معاملة
// الاسترداد نفسِها (البند ٥)، **فمن حملها إلى المحفظة يحملها إلى هنا.**
type Querier = wallet.Querier

// Create يقيّد نشأةَ التزام — **ويحدّث الصورةَ المحفوظة في المعاملة نفسِها.**
//
// **والنشأةُ والاسترداد في معاملةٍ واحدة** (البند ٥): **إن سقط أحدُهما
// سقط الآخر**، **فلا يقع دَينٌ بلا أصلٍ ولا أصلٌ بلا دَين.**
//
// **وإعادةُ النداء لا تُنشئ ثانياً** — **يمنعها فهرسٌ فريدٌ في القاعدة
// لا ترتيبُ الشيفرة** (البند ٦). ويُرجع `false` إن كان قائماً.
func Create(ctx context.Context, q Querier, partyKind, partyID string,
	amount int64, cause, orderID string, actorID *string) (bool, error) {
	if amount <= 0 {
		return false, errors.New("التزامٌ بمبلغٍ غيرِ موجب")
	}
	var order any
	if orderID != "" {
		order = orderID
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO financial_obligations
		       (party_kind, party_id, amount, cause, order_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT DO NOTHING
		RETURNING id::text`,
		partyKind, partyID, amount, cause, order, actorID).Scan(&id)
	if err != nil {
		// **لا صفَّ راجعٌ = التزامٌ قائمٌ لهذا الطلب بهذا السبب.**
		// **وإعادةُ نداءٍ لا تزيد ديناً.**
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, bumpCache(ctx, q, partyKind, partyID, amount)
}

// Settle يقتطع من الالتزامات القائمة بما يحتمله المتاح — **الأقدمُ أوّلاً.**
//
// # ولماذا الأقدمُ أوّلاً
//
// **لا سياسةَ تخصيصٍ معتمدةٌ في المنصّة** (البند ٨)، **فتُختار أبسطُ
// سياسةٍ عادلةٍ ومفهومة**: **من نشأ أوّلاً يُسدَّد أوّلاً.** **ولا
// أولويّاتٌ تُخترَع** — قاعدةٌ لا يفهمها صاحبُها لا يقبلها.
//
// **ويُكتب سطرُ تسويةٍ لكلّ التزامٍ مسّه الاقتطاع** — **ولو اقتُطع من
// ثلاثةٍ في نداءٍ واحد.**
//
// يُرجع مجموعَ ما اقتُطع.
func Settle(ctx context.Context, q Querier, partyKind, partyID string,
	available int64, orderID string, ledgerTx int64, actorID *string) (int64, error) {
	return SettleBy(ctx, q, partyKind, partyID, available, orderID, ledgerTx, actorID,
		MethodEarning, "")
}

// SettleBy كـ`Settle` **بطريقةِ تسويةٍ ومرجعٍ صريحين** — شحنُ المحفظة مثلاً
// (`MethodTopup` ومرجعُه طلبُ الشحن).
func SettleBy(ctx context.Context, q Querier, partyKind, partyID string,
	available int64, orderID string, ledgerTx int64, actorID *string,
	method, ref string) (int64, error) {
	if available <= 0 {
		return 0, nil
	}
	rows, err := q.Query(ctx, `
		SELECT id::text, amount - settled
		  FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2 AND closed_at IS NULL
		 ORDER BY created_at, id
		 FOR UPDATE`, partyKind, partyID)
	if err != nil {
		return 0, err
	}
	type open struct {
		id        string
		remaining int64
	}
	var list []open
	for rows.Next() {
		var o open
		if err := rows.Scan(&o.id, &o.remaining); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	var taken int64
	for _, o := range list {
		if available <= 0 {
			break
		}
		take := o.remaining
		if available < take {
			take = available
		}
		remaining := o.remaining - take
		var order any
		if orderID != "" {
			order = orderID
		}
		var tx any
		if ledgerTx > 0 {
			tx = ledgerTx
		}
		if _, err := q.Exec(ctx, `
			INSERT INTO obligation_settlements
			       (obligation_id, amount, order_id, ledger_tx_id, remaining, created_by,
			        method, ref)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			o.id, take, order, tx, remaining, actorID, method, ref); err != nil {
			return 0, err
		}
		// **والإغلاقُ لحظةَ بلوغِ الصفر** — **ولا يُمحى السطر.**
		if _, err := q.Exec(ctx, `
			UPDATE financial_obligations
			   SET settled = settled + $2,
			       closed_at = CASE WHEN settled + $2 >= amount THEN now() END
			 WHERE id = $1`, o.id, take); err != nil {
			return 0, err
		}
		available -= take
		taken += take
	}
	if taken == 0 {
		return 0, nil
	}
	return taken, bumpCache(ctx, q, partyKind, partyID, -taken)
}

// VoidForOrder **يُسقط التزاماً نشأ من طلبٍ لم يقع** — «لدي توصيلة»، ٢٠٢٦-١٠-٠١.
//
// **دينُ أجرةِ توصيلةٍ أُلغيت قبل التسليم باطل** — ما أُخذ عنه شيء. **فيُغلق
// بباقيه ويُكتب سطرُ تسويةٍ بلا قيدِ دفتر** (`ledger_tx_id` فارغ): **يُقرأ في
// السجلّ «أُسقط» لا «سُدِّد»**، ولا مالَ يتحرّك.
//
// **ولا يُمحى الصفّ** — كما لا يُمحى في `Settle`. **ولا أثرَ لنداءٍ ثانٍ**:
// المغلقُ لا يُختار.
func VoidForOrder(ctx context.Context, q Querier, partyKind, partyID, orderID, cause string,
	actorID *string) error {
	var id string
	var remaining int64
	err := q.QueryRow(ctx, `
		SELECT id::text, amount - settled FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2 AND order_id = $3 AND cause = $4
		   AND closed_at IS NULL
		 FOR UPDATE`, partyKind, partyID, orderID, cause).Scan(&id, &remaining)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if remaining <= 0 {
		return nil
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO obligation_settlements
		       (obligation_id, amount, order_id, ledger_tx_id, remaining, created_by, method)
		VALUES ($1, $2, $3, NULL, 0, $4, $5)`, id, remaining, orderID, actorID,
		MethodVoided); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `
		UPDATE financial_obligations SET settled = amount, closed_at = now()
		 WHERE id = $1`, id); err != nil {
		return err
	}
	return bumpCache(ctx, q, partyKind, partyID, -remaining)
}

// ErrClosed **الالتزامُ مغلقٌ أو لا باقيَ يكفي** — سُدّ بين الاقتراح والموافقة.
var ErrClosed = errors.New("obligation closed or short")

// SettleOne **يسدّ التزاماً بعينه بمبلغٍ بعينه** — دفعةُ المكتب والشطب.
//
// **ولا يمسّ غيرَه**: دفعةٌ نقديّةٌ على دينٍ سمّاه الموظّفُ لا تُوزَّع على
// الأقدم. **ولا تتجاوز الباقي** — `ErrClosed` إن كان أقلَّ من المبلغ.
//
// يُرجع الطرفَ وما بقي بعد السداد.
func SettleOne(ctx context.Context, q Querier, obligationID string, amount int64,
	method, ref string, ledgerTx int64, actorID *string) (partyKind, partyID string, remaining int64, err error) {
	if amount <= 0 {
		return "", "", 0, errors.New("سدادٌ بمبلغٍ غيرِ موجب")
	}
	var rest int64
	err = q.QueryRow(ctx, `
		SELECT party_kind, party_id::text, amount - settled FROM financial_obligations
		 WHERE id = $1 AND closed_at IS NULL FOR UPDATE`, obligationID).
		Scan(&partyKind, &partyID, &rest)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", 0, ErrClosed
	}
	if err != nil {
		return "", "", 0, err
	}
	if amount > rest {
		return "", "", 0, ErrClosed
	}
	remaining = rest - amount
	var tx any
	if ledgerTx > 0 {
		tx = ledgerTx
	}
	if _, err = q.Exec(ctx, `
		INSERT INTO obligation_settlements
		       (obligation_id, amount, order_id, ledger_tx_id, remaining, created_by, method, ref)
		VALUES ($1, $2, NULL, $3, $4, $5, $6, $7)`,
		obligationID, amount, tx, remaining, actorID, method, ref); err != nil {
		return "", "", 0, err
	}
	if _, err = q.Exec(ctx, `
		UPDATE financial_obligations
		   SET settled = settled + $2,
		       closed_at = CASE WHEN settled + $2 >= amount THEN now() END
		 WHERE id = $1`, obligationID, amount); err != nil {
		return "", "", 0, err
	}
	return partyKind, partyID, remaining, bumpCache(ctx, q, partyKind, partyID, -amount)
}

// Balance الباقي على طرفٍ — **من الوقائع لا من الصورة.**
func Balance(ctx context.Context, q Querier, partyKind, partyID string) (int64, error) {
	var v int64
	err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2 AND closed_at IS NULL`,
		partyKind, partyID).Scan(&v)
	return v, err
}

// bumpCache يحدّث الصورةَ المحفوظة — **ولا يكتبها أحدٌ سواه.**
func bumpCache(ctx context.Context, q Querier, partyKind, partyID string, delta int64) error {
	var stmt string
	switch partyKind {
	case PartyMerchant:
		stmt = `UPDATE merchants SET debt = debt + $2 WHERE id = $1`
	case PartyRep:
		stmt = `UPDATE users SET commission_debt = commission_debt + $2 WHERE id = $1`
	default:
		return fmt.Errorf("طرفٌ مجهول: %q", partyKind)
	}
	_, err := q.Exec(ctx, stmt, partyID, delta)
	return err
}
