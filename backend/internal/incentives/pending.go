package incentives

// ══════════════════════════════════════════════════════════════════════
// **مكافأةٌ بلغها موقوفٌ — تُصرف يومَ يعود** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// «يظل إذا حقق الشروط». **مندوبٌ اكتمل متجرُه وهو موقوفٌ أو سائقٌ وصّل وهو موقوف**
// يبلغ العتبةَ مرّةً واحدة — **والعدّادُ لا يعود إليها.** فإن لم تُحفظ ضاعت.
//
// **والصفُّ يُكتب عند محاولة الصرف** (`holdIfDue`)، **ويُصرف من بابين**:
//
//	إعادةُ التفعيل  ←  `PayPending` لصاحبه وحدَه، لحظةَ يعود فعّالاً
//	الجولةُ الدوريّة ←  `PayPendingSweep` لكلّ صفٍّ صار صاحبُه فعّالاً
//
// **والصرفُ للشهر الذي بلغها فيه** — لا للشهر الجاري — **والفهرسُ الفريدُ يمنع
// مرحلةً تُدفع مرّتين** ولو التقاها البابان معاً.

import (
	"context"
	"time"
)

// PayPending **يصرف ما حُفظ لصاحبه وهو موقوف** — ثمّ الشهرَ الجاريَ لكلّ دورٍ يحمله.
//
// **والشهرُ الجاري معها**: من بلغ العتبةَ ولم يُحاوَل الصرفُ له (لا تسليمَ ولا متجرَ
// بعدها) لا صفَّ له — **فيُسأل عنه هنا أيضاً.**
func (s *Service) PayPending(ctx context.Context, userID string) []Paid {
	var out []Paid
	for _, row := range s.openPending(ctx, `p.user_id = $1::uuid`, userID) {
		if p, ok := s.payPendingRow(ctx, row); ok {
			out = append(out, p)
		}
	}
	month := PeriodOf(time.Now())
	for _, role := range s.rolesOf(ctx, userID) {
		paid, err := s.grantForMonth(ctx, userID, role, month)
		if err != nil {
			s.RecordFailure(ctx, userID, role, month, err)
			continue
		}
		if paid > 0 {
			out = append(out, Paid{UserID: userID, Role: role, Amount: paid, Month: month})
		}
	}
	return out
}

// PayPendingSweep **جولةٌ على كلّ ما حُفظ وعاد صاحبُه فعّالاً** — من الحلقة الدوريّة.
//
// **ولا يُعتمد على باب التفعيل وحدَه**: حسابٌ يُفعَّل من غير شاشة الحسابات (هجرةٌ،
// أو سكربتُ دعم) يمرّ دون أن يُصرف له — **والجولةُ تلتقطه.**
func (s *Service) PayPendingSweep(ctx context.Context) []Paid {
	var out []Paid
	for _, row := range s.openPending(ctx, `u.status = 'active' AND u.deleted_at IS NULL`) {
		if p, ok := s.payPendingRow(ctx, row); ok {
			out = append(out, p)
		}
	}
	return out
}

type pendingRow struct{ id, userID, role, month string }

// openPending **الصفوفُ المفتوحة** بشرطٍ على الصفّ (`p`) أو صاحبه (`u`).
func (s *Service) openPending(ctx context.Context, cond string, args ...any) []pendingRow {
	rows, err := s.db.Query(ctx, `
		SELECT p.id::text, p.user_id::text, p.role, p.month
		  FROM incentive_pending_grants p
		  JOIN users u ON u.id = p.user_id
		 WHERE p.paid_at IS NULL AND `+cond+`
		 ORDER BY p.created_at LIMIT 200`, args...)
	if err != nil {
		s.logf("قراءةُ المكافآت المحفوظة تعذّرت", "error", err)
		return nil
	}
	defer rows.Close()
	var out []pendingRow
	for rows.Next() {
		var r pendingRow
		if rows.Scan(&r.id, &r.userID, &r.role, &r.month) == nil {
			out = append(out, r)
		}
	}
	return out
}

// payPendingRow **يصرف صفّاً ويُغلقه في معاملةٍ واحدة** — ولا يُغلَق إن بقي موقوفاً.
func (s *Service) payPendingRow(ctx context.Context, r pendingRow) (Paid, bool) {
	p := Paid{UserID: r.userID, Role: r.role, Month: r.month}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return p, false
	}
	defer func() { _ = tx.Rollback(ctx) }()
	state, err := userState(ctx, tx, r.userID)
	if err != nil || state != stateActive {
		return p, false
	}
	paid, err := s.grantForMonthTx(ctx, tx, r.userID, r.role, r.month)
	if err != nil {
		_ = tx.Rollback(ctx)
		s.RecordFailure(ctx, r.userID, r.role, r.month, err)
		return p, false
	}
	if _, err := tx.Exec(ctx, `
		UPDATE incentive_pending_grants SET paid_at = now(), paid_amount = $2
		 WHERE id = $1::uuid`, r.id, paid); err != nil {
		return p, false
	}
	if err := tx.Commit(ctx); err != nil {
		return p, false
	}
	p.Amount = paid
	return p, paid > 0
}

// rolesOf **أدوارُ الحوافز التي يحملها** — سائقٌ ومندوب.
func (s *Service) rolesOf(ctx context.Context, userID string) []string {
	rows, err := s.db.Query(ctx, `
		SELECT role_code FROM user_roles
		 WHERE user_id = $1::uuid AND role_code IN ('driver', 'sales')
		 ORDER BY role_code`, userID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var r string
		if rows.Scan(&r) == nil {
			out = append(out, r)
		}
	}
	return out
}
