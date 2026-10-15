package server

// ══════════════════════════════════════════════════════════════════════
// **تذكيرُ مصاريف الشهر المتكرّرة** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **أوّلَ كلّ شهرٍ بيوم دمشق** يصل المالكَ ومن يملك `finance.manage` إشعارٌ
// بأبواب «مصروف شهري متكرر» (`expense_categories.monthly_recurring`):
// «تذكير: مصاريف الشهر المتكررة — إيجار، رواتب، …». **والمالكُ يسجّلها بيده** —
// التذكيرُ لا يكتب مصروفاً.
//
// **ومرّةً للشهر**: صفٌّ في `expense_monthly_reminders` بمفتاح أوّل الشهر، **فإعادةُ
// تشغيل الخادم أو جولتان في اليوم نفسِه لا تكرّران التذكير.**

import (
	"context"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

const (
	expenseReminderTitle = "تذكير: مصاريف الشهر المتكررة"
	// expenseReminderSep فاصلُ أسماء الأبواب في المتن.
	expenseReminderSep = "، "
)

// remindRecurringExpenses **جولةٌ واحدة** — تذكّر إن كان اليومُ أوّلَ الشهر بيوم دمشق
// ولم يُذكَّر بعد. ويردّ `true` إن أُرسل التذكيرُ في هذه الجولة.
func (s *Server) remindRecurringExpenses(ctx context.Context, now time.Time) (bool, error) {
	day := now.In(damascusLoc())
	if day.Day() != 1 {
		return false, nil
	}
	month := day.Format("2006-01") + "-01"
	rows, err := s.pg.Query(ctx, `
		SELECT name FROM expense_categories
		WHERE monthly_recurring AND active
		ORDER BY sort_order, name`)
	if err != nil {
		return false, err
	}
	var names []string
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return false, err
		}
		names = append(names, n)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return false, err
	}
	// **ولا تذكيرَ بلا باب** — ولا يُختم الشهر، فبابٌ يُعلَّم اليومَ يُذكَّر به اليومَ نفسَه.
	if len(names) == 0 {
		return false, nil
	}
	tag, err := s.pg.Exec(ctx, `
		INSERT INTO expense_monthly_reminders (month) VALUES ($1::date)
		ON CONFLICT (month) DO NOTHING`, month)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	s.notify.NotifyCapsRoles(ctx, []string{string(authz.FinanceManage)},
		[]string{authz.RoleOwnerSuperAdmin}, notifications.Input{
			Kind:  notifications.KindWallet,
			Title: expenseReminderTitle,
			Body:  expenseReminderTitle + " — " + strings.Join(names, expenseReminderSep),
			Href:  "/dashboard/expenses",
		})
	return true, nil
}

// RunExpenseReminder **جولةُ التذكير كلَّ ساعة** — والختمُ يمنع التكرار.
func (s *Server) RunExpenseReminder(ctx context.Context, interval time.Duration) {
	_, _ = s.remindRecurringExpenses(ctx, time.Now())
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.remindRecurringExpenses(ctx, time.Now()); err != nil {
				s.logger.Error("تذكيرُ المصاريف المتكرّرة", "error", err)
			}
		}
	}
}
