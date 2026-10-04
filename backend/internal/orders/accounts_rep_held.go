package orders

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المندوب الموقوف محجوزة** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// «تنحجز وتنصرف إذا رجع فعّال»: **لا تُقيَّد في محفظته** ما دام موقوفاً — فتبقى
// في ربح المنصّة (`creditTreasury` لا يراها نصيبَ طرف). **وإن رجع فعّالاً** قُيّدت
// كلُّها بمراجعها، **وأُعيد حسابُ نصيب المنصّة لكلّ طلبٍ** فينقص ربحُها بقدرها —
// فيبقى دفترُ كلّ طلبٍ متوازناً (`FI-06.a`). **وإن حُظر نهائيّاً سقطت** وبقيت للمنصّة.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// holdRepCommissionIfInactive يحجز العمولةَ إن لم يكن المندوبُ فعّالاً.
func (s *Service) holdRepCommissionIfInactive(ctx context.Context, q wallet.Querier,
	repID, orderID string, amount int64) (bool, error) {
	var status string
	if err := q.QueryRow(ctx, `SELECT status FROM users WHERE id = $1`, repID).Scan(&status); err != nil {
		return false, err
	}
	if status == "active" {
		return false, nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO rep_held_commissions (rep_id, order_id, amount)
		VALUES ($1, $2, $3) ON CONFLICT (order_id) DO NOTHING`, repID, orderID, amount)
	return err == nil, err
}

// ReleaseHeldCommissionsTx **يصرف المحجوزَ لمندوبٍ عاد فعّالاً** — في معاملة المستدعي.
//
// يُرجع عددَ الطلبات ومجموعَ ما صُرف.
func (s *Service) ReleaseHeldCommissionsTx(ctx context.Context, q dbtx.Querier, repID, actorID string) (int, int64, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text, order_id::text, amount FROM rep_held_commissions
		 WHERE rep_id = $1 AND status = 'held'
		 ORDER BY created_at FOR UPDATE`, repID)
	if err != nil {
		return 0, 0, err
	}
	type held struct {
		id, order string
		amount    int64
	}
	var list []held
	for rows.Next() {
		var h held
		if err := rows.Scan(&h.id, &h.order, &h.amount); err != nil {
			rows.Close()
			return 0, 0, err
		}
		list = append(list, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	var total int64
	for _, h := range list {
		if _, err := s.wallet.ApplyTx(ctx, q, repID, h.amount, "commission", h.order,
			"عمولةٌ محجوزةٌ صُرفت بعد إعادة التفعيل", &actorID); err != nil {
			return 0, 0, err
		}
		if err := s.creditTreasury(ctx, q, h.order, actorID); err != nil {
			return 0, 0, err
		}
		if _, err := q.Exec(ctx, `
			UPDATE rep_held_commissions SET status = 'released', decided_at = now(), decided_by = $2
			 WHERE id = $1`, h.id, actorID); err != nil {
			return 0, 0, err
		}
		total += h.amount
	}
	return len(list), total, nil
}

// ForfeitHeldCommissionsTx **حظرٌ نهائيٌّ يُسقط المحجوز** — يبقى للمنصّة.
func (s *Service) ForfeitHeldCommissionsTx(ctx context.Context, q dbtx.Querier, repID, actorID string) (int64, error) {
	tag, err := q.Exec(ctx, `
		UPDATE rep_held_commissions SET status = 'forfeited', decided_at = now(), decided_by = $2
		 WHERE rep_id = $1 AND status = 'held'`, repID, actorID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// HeldCommissionTotal مجموعُ المحجوز القائم لمندوب.
func (s *Service) HeldCommissionTotal(ctx context.Context, repID string) (int64, int) {
	var total int64
	var n int
	_ = s.db.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0), count(*) FROM rep_held_commissions
		 WHERE rep_id = $1 AND status = 'held'`, repID).Scan(&total, &n)
	return total, n
}
