package incentives

// ══════════════════════════════════════════════════════════════════════
// **تنبيهاتُ المكافآت — والماليّةُ تقرّر** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البندان ٣ و٤)
// ══════════════════════════════════════════════════════════════════════
//
//	count_dropped  مكافأةُ مرحلةٍ صُرفت والعدُّ صار تحتها:
//	               سائقٌ استُرجع طلبُه · مندوبٌ حُذف متجرُه أو تبيّن تجريبيّاً
//	grant_failed   مكافأةُ هدفٍ تعثّرت ولم تُعَد بعد
//
// **لا سحبَ آليّ**: الماليّةُ تقرّر «تبقى» أو «تُسترجَع» — والاسترجاعُ طلبُ
// عقوبةٍ يمرّ بالموافقة نفسِها (شخصٌ ثانٍ). **وعددُ المرحلة محفوظٌ على صفّ
// المكافأة** (`target_count`) فلا يتبدّل الحكمُ إن تبدّل الإعدادُ بعدها.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// Alert تنبيهٌ في الصفحة.
type Alert struct {
	Kind        string `json:"kind"`
	IncentiveID string `json:"incentive_id,omitempty"`
	FailureID   string `json:"failure_id,omitempty"`
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Month       string `json:"month"`
	Level       int    `json:"level"`
	TargetCount int    `json:"target_count"`
	Current     int    `json:"current"`
	Amount      int64  `json:"amount"`
	// Refunded كم طلباً مسترجَعاً في ذلك الشهر (للسائق).
	Refunded int `json:"refunded"`
	// FakeStores أسماءُ متاجرَ تجريبيّةٍ انحسبت له (للمندوب).
	FakeStores []string `json:"fake_stores"`
	Error      string   `json:"error,omitempty"`
	Attempts   int      `json:"attempts,omitempty"`
	// At متى صُرفت أو تعثّرت.
	At time.Time `json:"at"`
}

// ErrBadDecision **«تبقى» أو «تُسترجَع» لا غير.**
var ErrBadDecision = httpx.NewError(http.StatusBadRequest, "bad_alert_decision", "errors.validation")

// Alerts **تنبيهاتُ الدور كلُّها** — ما لم يُقرَّر فيه، من آخر سنة.
//
// `testMarker` تعبيرُ «تجريبيّ» المركزيّ (قرارُ المالك: ما في اسمه «تجربة»
// أو «اختبار» أو `test`) — يمرّره من يملكه.
func (s *Service) Alerts(ctx context.Context, role, testMarker string) ([]Alert, error) {
	month := `split_part(i.period, '#', 1)`
	current := doneSQL(role, "i.user_id", month)
	if role == "sales" {
		// **والتجريبيُّ لا يُعدّ** — ومحذوفٌ لم يعد موجوداً أصلاً.
		current = `(SELECT count(*) FROM merchants mm
		             WHERE mm.opened_by_rep_id = i.user_id
		               AND mm.name !~* $2
		               AND mm.created_at >= ` + monthFromSQL(month) + `
		               AND mm.created_at <  ` + monthToSQL(month) + `)`
	}
	extra := `0, ARRAY[]::text[]`
	if role == "sales" {
		extra = `0, ARRAY(SELECT mm.name FROM merchants mm
		                   WHERE mm.opened_by_rep_id = i.user_id AND mm.name ~* $2
		                     AND mm.created_at >= ` + monthFromSQL(month) + `
		                     AND mm.created_at <  ` + monthToSQL(month) + `
		                   ORDER BY mm.created_at LIMIT 20)`
	} else {
		extra = `(SELECT count(*) FROM orders o
		           WHERE o.driver_id = i.user_id AND o.status = 'refunded'
		             AND o.delivered_at >= ` + monthFromSQL(month) + `
		             AND o.delivered_at <  ` + monthToSQL(month) + `)::int,
		         ARRAY[]::text[]`
	}
	rows, err := s.db.Query(ctx, `
		SELECT * FROM (
		  SELECT i.id::text, i.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
		         `+month+`, COALESCE(NULLIF(split_part(i.period, '#', 2), ''), '0')::int,
		         i.target_count, `+current+`::int AS cur, i.amount, `+extra+`, i.created_at
		    FROM incentives i
		    JOIN users u ON u.id = i.user_id
		   WHERE i.for_target AND i.period IS NOT NULL AND i.target_count IS NOT NULL
		     AND i.target_role = $1 AND $2::text IS NOT NULL
		     AND i.created_at > now() - interval '13 months'
		     AND NOT EXISTS (SELECT 1 FROM incentive_alert_decisions d WHERE d.incentive_id = i.id)
		) x
		WHERE x.cur < x.target_count
		ORDER BY x.created_at DESC`, role, testMarker)
	if err != nil {
		return nil, err
	}
	out := []Alert{}
	for rows.Next() {
		a := Alert{Kind: "count_dropped"}
		var fakes []string
		if err := rows.Scan(&a.IncentiveID, &a.UserID, &a.Name, &a.Month, &a.Level,
			&a.TargetCount, &a.Current, &a.Amount, &a.Refunded, &fakes, &a.At); err != nil {
			rows.Close()
			return nil, err
		}
		a.FakeStores = append([]string{}, fakes...)
		out = append(out, a)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	frows, err := s.db.Query(ctx, `
		SELECT f.id::text, f.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
		       f.month, f.last_error, f.attempts, f.updated_at
		  FROM incentive_grant_failures f
		  JOIN users u ON u.id = f.user_id
		 WHERE f.resolved_at IS NULL AND f.role = $1
		 ORDER BY f.updated_at DESC`, role)
	if err != nil {
		return nil, err
	}
	defer frows.Close()
	for frows.Next() {
		a := Alert{Kind: "grant_failed", FakeStores: []string{}}
		if err := frows.Scan(&a.FailureID, &a.UserID, &a.Name, &a.Month, &a.Error,
			&a.Attempts, &a.At); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, frows.Err()
}

// DecideAlert **قرارُ الماليّة على تنبيه** — «تبقى» تُغلقه، و«تُسترجَع» تفتح
// طلبَ عقوبةٍ بمبلغ المكافأة يوافق عليه شخصٌ ثانٍ. يُرجع معرّفَ الطلب إن وُجد.
func (s *Service) DecideAlert(ctx context.Context, q dbtx.Querier, incentiveID, actor,
	decision, note string) (string, error) {
	if decision != "keep" && decision != "clawback" {
		return "", ErrBadDecision
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return "", ErrNeedsReason
	}
	var userID string
	var amount int64
	err := q.QueryRow(ctx, `
		SELECT user_id::text, amount FROM incentives
		 WHERE id = $1::uuid AND for_target AND period IS NOT NULL`, incentiveID).Scan(&userID, &amount)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", httpx.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	var reqID *string
	if decision == "clawback" {
		id, err := s.Propose(ctx, q, Proposal{
			Actor: actor, UserID: userID, Kind: KindPenalty, Amount: amount,
			Note: note, Source: "alert", AlertIncentiveID: incentiveID,
		})
		if err != nil {
			return "", err
		}
		reqID = &id
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO incentive_alert_decisions (incentive_id, decision, note, request_id, decided_by)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5::uuid)
		ON CONFLICT (incentive_id) DO NOTHING`, incentiveID, decision, note, reqID, actor)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", ErrRequestDecided
	}
	if reqID == nil {
		return "", nil
	}
	return *reqID, nil
}
