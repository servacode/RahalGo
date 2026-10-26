package server

// ══════════════════════════════════════════════════════════════════════
// **درجُ المكتب — قراراتُ المالك ٢٠٢٦-١٠-١٥ (الخزينة)**
// ══════════════════════════════════════════════════════════════════════
//
//	رصيدُ البداية   ما عُدّ في الدرج يومَ بدأ التتبّع — **مرّةً واحدة**، قبل أيّ إغلاقٍ مُعتمَد
//	الإيداع         نقدٌ من جيب المالك يدخل الدرج — **لا قيدَ في الخزينة، فليس ربحاً**
//	السالب          **مسموح**: مصروفٌ كاشٌ أو سحبٌ من الدرج لا يُردّ لأنّ الدرجَ لا يكفي
//	التقرير         يومٌ وأسبوعٌ وشهر: ما دخل وما خرج بمصادره، وفروقُ الإغلاق، والرصيدُ أوّلاً وآخراً
//	المراجعة        `can_approve` من المحرّك — المالكُ وحدَه يراجع إغلاقَه حين لا يوجد غيرُه

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/officecash"
)

var errCashOpeningLate = httpx.NewError(http.StatusConflict,
	"cash_opening_late", "errors.cash_opening_late")

// approvalVerdictFor **أيستطيع `me` أن يوافق على اقتراح `proposedBy`؟** — بالحكم نفسِه الذي
// يطبّقه البابُ (`approval.Check`)، **فالشاشةُ لا تخمّن**. والقدرةُ يسألها المنادي قبله.
func approvalVerdictFor(ctx context.Context, q dbtx.Querier, me, proposedBy string,
	capability authz.Capability) (can, self bool, err error) {
	v, err := approval.Check(ctx, q, approval.Request{ProposedBy: proposedBy, Actor: me, Capability: capability})
	if errors.Is(err, approval.ErrSelfApprove) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, v.SelfApproved, nil
}

// handleCashOpening **«رصيدُ البداية»** — `POST /admin/cashbox/opening` `{counted, note}`.
//
// **ولماذا إغلاقٌ مُعتمَدٌ لا سطرٌ في الصندوق**: المتوقَّعُ يبدأ من معدود آخر إغلاقٍ مُعتمَد
// ويضيف ما لم يُغلَق عليه (`cashboxPosition`). **وسطرٌ داخلٌ بمصدر «بداية» يُضاف فوق ما قبله**
// من سطورٍ كُتبت قبل أن يُعدّ الدرج — فيصير المتوقَّعُ المعدودَ زائداً تاريخاً عدّه المالكُ
// أصلاً، **ويدخل «الداخل» في كلّ تقريرٍ كأنّه نقدٌ وصل.** والإغلاقُ يبتلع تلك السطورَ
// (يُوسمها به) ويثبّت المعدودَ أساساً — **وهو ما يعنيه «بدأنا نعدّ من هنا».**
//
// **ويُحفظ الفرقُ بين المعدود وما ابتلعه** (`difference`) — فرصيدُ الدرج في أيّ لحظةٍ يبقى
// «مجموعَ السطور + فروقَ الإغلاقات المعتمَدة» (`handleCashboxReport`)، **ولا يُعدّ في فروق
// التقرير**: ليس نقصاً ولا زيادة. **ولا يُراجَع**: من يملك `finance.manage` يكتبه مرّةً، ويُسجَّل.
func (s *Server) handleCashOpening(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Counted *int64 `json:"counted"`
		Note    string `json:"note"`
	}](r)
	if err != nil || req.Counted == nil || *req.Counted < 0 {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)
	note := clip(strings.TrimSpace(req.Note), 500)
	var row cashCloseRow
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		if _, err := q.Exec(ctx, `LOCK TABLE office_cash_closes IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		// **مرّةً واحدةً وقبل أيّ إغلاق** — بعد إغلاقٍ مُعتمَدٍ صار للدرج أساسٌ معدود.
		var late bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM office_cash_closes WHERE status IN ('approved', 'pending'))`).
			Scan(&late); err != nil {
			return err
		}
		if late {
			return errCashOpeningLate
		}
		pos, err := cashboxPosition(ctx, q)
		if err != nil {
			return err
		}
		diff := *req.Counted - pos.Expected
		abs := diff
		if abs < 0 {
			abs = -abs
		}
		var id string
		if err := q.QueryRow(ctx, `
			INSERT INTO office_cash_closes
			       (day, kind, opening, cash_in, cash_out, expected, counted, difference, amount, note,
			        status, proposed_by, decided_by, decided_at)
			VALUES (account_day(now()), 'opening', 0, $1, $2, $3, $4, $5, $6, $7,
			        'approved', $8, $8, now())
			RETURNING id::text`,
			pos.CashIn, pos.CashOut, pos.Expected, *req.Counted, diff, abs, note, actor).Scan(&id); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE office_cash_entries SET close_id = $1 WHERE close_id IS NULL`, id); err != nil {
			return err
		}
		if err := s.auditTx(ctx, q, r, "finance.cash_opening", "office_cash_close", id,
			map[string]any{"counted": *req.Counted, "absorbed": pos.Expected, "note": note}); err != nil {
			return err
		}
		rows, err := s.closesWhere(ctx, q, `c.id = $1`, 1, id)
		if err != nil || len(rows) == 0 {
			return err
		}
		row = rows[0]
		return nil
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"close": row})
}

// handleCashDeposit **«إيداعٌ بالدرج»** — `POST /admin/cashbox/deposits` `{amount, note}`.
//
// **المالكُ يضع في الدرج من جيبه**: سطرٌ داخلٌ بمصدر `owner_deposit` **ولا قيدَ في الخزينة**
// — فلا يُقرأ ربحاً ولا دخلاً ولا يمسّ رصيدَها. **ويُؤكَّد بكلمة السرّ كالسحب** (`sensitive.go`)،
// ويُسجَّل في الأثر بالمعاملة نفسِها.
func (s *Server) handleCashDeposit(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil || req.Amount <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 500)
	if note == "" {
		s.respondErr(w, errNoteRequired)
		return
	}
	actor := userIDFrom(r)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var ref string
		if err := q.QueryRow(ctx, `SELECT gen_random_uuid()::text`).Scan(&ref); err != nil {
			return IdempotentBody{}, err
		}
		if err := officecash.Record(ctx, q, officecash.Entry{
			Direction: officecash.In, Amount: req.Amount, Source: officecash.SourceOwnerDeposit,
			Ref: ref, UserID: actor, Actor: actor, Note: note,
		}); err != nil {
			return IdempotentBody{}, err
		}
		if err := s.auditTx(ctx, q, r, "finance.cash_owner_deposit", "office_cash_entry", ref,
			map[string]any{"amount": req.Amount, "note": note}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:      http.StatusCreated,
			Payload:     map[string]any{"ref": ref, "amount": req.Amount},
			AfterCommit: func() { s.touch("wallet", "ops") },
		}, nil
	})
}

// damascusPeriod **مدى «يوم · أسبوع · شهر» بيوم دمشق** — بمسطرة صفحة الأرباح نفسِها
// (`profits/page.tsx`: الأسبوعُ آخرُ سبعة أيّامٍ باليوم، والشهرُ من أوّله إلى اليوم)،
// **فتقول الشاشاتُ الشيءَ نفسَه عن المدّة نفسِها.** و`all` بلا حدّ (فارغان).
func damascusPeriod(period string, now time.Time, startHour int64) (from, to string, ok bool) {
	// **يومُ الحساب يبدأ بساعة فتح الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٦، `platform.day_start_hour`) —
	// ما قبلها لليوم السابق.
	day := now.In(damascusLoc()).Add(-time.Duration(startHour) * time.Hour)
	today := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	to = today.Format("2006-01-02")
	switch period {
	case "", "day":
		return to, to, true
	case "week":
		return today.AddDate(0, 0, -6).Format("2006-01-02"), to, true
	case "month":
		return time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02"), to, true
	case "all":
		return "", "", true
	}
	return "", "", false
}

type cashSourceSum struct {
	Source    string `json:"source"`
	Direction string `json:"direction"`
	Amount    int64  `json:"amount"`
	Count     int    `json:"count"`
}

// handleCashboxReport **تقريرُ الدرج** — `GET /admin/cashbox/report?period=day|week|month`.
//
// **رصيدُ الدرج في لحظةٍ = مجموعُ سطوره قبلها + فروقُ الإغلاقات المعتمَدة قبلها** — فالإغلاقُ
// المعتمَدُ يجعل المعدودَ أساساً (المتوقَّعُ + الفرق)، ورصيدُ البداية فرقُه ما بين المعدود وما
// ابتلعه. **وهو المتوقَّعُ نفسُه** (`cashboxPosition`) ما لم يكن إغلاقٌ معلَّق.
// **وفروقُ الفترة للإغلاقات اليوميّة المعتمَدة بيومها** — ورصيدُ البداية ليس فرقاً.
func (s *Server) handleCashboxReport(w http.ResponseWriter, r *http.Request) {
	period := r.URL.Query().Get("period")
	if period == "all" {
		s.respondErr(w, errValidation)
		return
	}
	from, to, ok := damascusPeriod(period, time.Now(), s.settings.GetInt(r.Context(), "platform.day_start_hour"))
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	if period == "" {
		period = "day"
	}
	ctx := r.Context()
	const bounds = `
		WITH b AS (SELECT account_day_start($1::date) AS t0,
		                  account_day_start($2::date + 1) AS t1)`
	var in, out, entStart, entEnd int64
	if err := s.pg.QueryRow(ctx, bounds+`
		SELECT COALESCE(sum(e.amount) FILTER (WHERE e.direction = 'in'  AND e.created_at >= b.t0 AND e.created_at < b.t1), 0)::bigint,
		       COALESCE(sum(e.amount) FILTER (WHERE e.direction = 'out' AND e.created_at >= b.t0 AND e.created_at < b.t1), 0)::bigint,
		       COALESCE(sum(CASE WHEN e.direction = 'in' THEN e.amount ELSE -e.amount END)
		                FILTER (WHERE e.created_at < b.t0), 0)::bigint,
		       COALESCE(sum(CASE WHEN e.direction = 'in' THEN e.amount ELSE -e.amount END)
		                FILTER (WHERE e.created_at < b.t1), 0)::bigint
		  FROM b LEFT JOIN office_cash_entries e ON true
		 GROUP BY b.t0, b.t1`, from, to).
		Scan(&in, &out, &entStart, &entEnd); err != nil {
		s.respondErr(w, err)
		return
	}
	var diffStart, diffEnd, diff, shortage, excess int64
	var closes int
	if err := s.pg.QueryRow(ctx, bounds+`
		SELECT COALESCE(sum(c.difference) FILTER (WHERE c.decided_at < b.t0), 0)::bigint,
		       COALESCE(sum(c.difference) FILTER (WHERE c.decided_at < b.t1), 0)::bigint,
		       COALESCE(sum(c.difference) FILTER (WHERE c.kind = 'daily' AND c.day BETWEEN $1::date AND $2::date), 0)::bigint,
		       COALESCE(sum(-c.difference) FILTER (WHERE c.kind = 'daily' AND c.day BETWEEN $1::date AND $2::date
		                                             AND c.difference < 0), 0)::bigint,
		       COALESCE(sum(c.difference) FILTER (WHERE c.kind = 'daily' AND c.day BETWEEN $1::date AND $2::date
		                                            AND c.difference > 0), 0)::bigint,
		       count(c.id) FILTER (WHERE c.kind = 'daily' AND c.day BETWEEN $1::date AND $2::date)::int
		  FROM b LEFT JOIN office_cash_closes c ON c.status = 'approved'
		 GROUP BY b.t0, b.t1`, from, to).
		Scan(&diffStart, &diffEnd, &diff, &shortage, &excess, &closes); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, bounds+`
		SELECT e.source, e.direction, sum(e.amount)::bigint, count(*)::int
		  FROM office_cash_entries e, b
		 WHERE e.created_at >= b.t0 AND e.created_at < b.t1
		 GROUP BY e.source, e.direction
		 ORDER BY e.direction, sum(e.amount) DESC, e.source`, from, to)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	bySource := []cashSourceSum{}
	for rows.Next() {
		var x cashSourceSum
		if err := rows.Scan(&x.Source, &x.Direction, &x.Amount, &x.Count); err != nil {
			s.respondErr(w, err)
			return
		}
		bySource = append(bySource, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"period": period, "from": from, "to": to,
		"total_in": in, "total_out": out, "by_source": bySource,
		"close_difference": diff, "shortage": shortage, "excess": excess, "closes": closes,
		"start_balance": entStart + diffStart, "end_balance": entEnd + diffEnd,
	})
}
