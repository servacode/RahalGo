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

	"github.com/jackc/pgx/v5"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// أسبابُ النشأة — **وهي نفسُها قيدُ `CHECK` في القاعدة.**
const (
	CauseRefundMerchant = "refund_merchant_earning"
	CauseRefundRep      = "refund_rep_commission"
	CauseReturnedGoods  = "returned_goods"
	CauseLegacy         = "legacy_opening"
)

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
			       (obligation_id, amount, order_id, ledger_tx_id, remaining, created_by)
			VALUES ($1, $2, $3, $4, $5, $6)`,
			o.id, take, order, tx, remaining, actorID); err != nil {
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
