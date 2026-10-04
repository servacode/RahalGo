package server

// ══════════════════════════════════════════════════════════════════════
// **صندوقُ المكتب والإغلاقُ اليوميّ** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الخزينة)
// ══════════════════════════════════════════════════════════════════════
//
//	الصندوق     ما دخل الدرجَ نقداً وما خرج منه — سطرٌ لكلّ حركةٍ بمرجع قيدها
//	              (`officecash.Record` في معاملة القيد نفسِها)
//	الإغلاق     موظّفٌ يعدّ ويسجّل «المعدود»، والنظامُ يقارنه بالمتوقَّع ويُظهر الفرق،
//	              **وشخصٌ ثانٍ يراجع** (`approval.Check`)
//	النقص       يُسجَّل عند المراجعة، ويُحَلّ إن وُجد المال، **وإن بقي يوماً صار
//	              خسارةً على المنصّة بموافقة مدير المنصّة** — قيدُ `platform_expense`
//	              في الخزينة بمرجع النقص.
//
// **والمتوقَّعُ = معدودُ آخر إغلاقٍ مُراجَع + ما دخل − ما خرج من سطورٍ لم يُغلَق
// عليها بعد.** وكلُّ سطرٍ يُوسَم بإغلاقه (`close_id`) فلا يُحسب مرّتين ولا يُفوَّت.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/officecash"
)

var (
	errCashClosePending = httpx.NewError(http.StatusConflict,
		"cash_close_pending", "errors.cash_close_pending")
	errCashCloseDayDone = httpx.NewError(http.StatusConflict,
		"cash_close_day_done", "errors.cash_close_day_done")
	errShortfallNotDue = httpx.NewError(http.StatusConflict,
		"shortfall_not_due", "errors.shortfall_not_due")
	errNoteRequired = httpx.NewError(http.StatusBadRequest,
		"note_required", "errors.note_required")
)

// cashPosition **موضعُ الدرج الآن** — ما يُنتظر أن يكون فيه.
type cashPosition struct {
	Opening   int64      `json:"opening"`    // معدودُ آخر إغلاقٍ مُراجَع
	OpenedAt  *time.Time `json:"opened_at"`  // متى أُغلق
	CashIn    int64      `json:"cash_in"`    // ما دخل بعده
	CashOut   int64      `json:"cash_out"`   // ما خرج بعده
	Expected  int64      `json:"expected"`   // = الأوّل + الداخل − الخارج
	Book      int64      `json:"book"`       // مجموعُ الصندوق منذ أوّل سطر
	OpenLines int        `json:"open_lines"` // سطورٌ لم يُغلَق عليها
}

func cashboxPosition(ctx context.Context, q dbtx.Querier) (cashPosition, error) {
	var p cashPosition
	err := q.QueryRow(ctx, `
		SELECT counted, decided_at FROM office_cash_closes
		 WHERE status = 'approved' ORDER BY decided_at DESC, created_at DESC LIMIT 1`).
		Scan(&p.Opening, &p.OpenedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return p, err
	}
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount) FILTER (WHERE direction = 'in'), 0)::bigint,
		       COALESCE(sum(amount) FILTER (WHERE direction = 'out'), 0)::bigint,
		       count(*)::int
		  FROM office_cash_entries WHERE close_id IS NULL`).
		Scan(&p.CashIn, &p.CashOut, &p.OpenLines); err != nil {
		return p, err
	}
	p.Expected = p.Opening + p.CashIn - p.CashOut
	b, err := officecash.Book(ctx, q)
	p.Book = b
	return p, err
}

type cashEntryRow struct {
	ID         string    `json:"id"`
	Direction  string    `json:"direction"`
	Amount     int64     `json:"amount"`
	Source     string    `json:"source"`
	Ref        string    `json:"ref"`
	UserName   string    `json:"user_name"`
	RecordedBy string    `json:"recorded_by"`
	Note       string    `json:"note"`
	Closed     bool      `json:"closed"`
	CreatedAt  time.Time `json:"created_at"`
}

// handleCashbox **الصندوقُ الآن**: موضعُه، وسطورُه (المفتوحةُ افتراضاً أو كلُّها)،
// والإغلاقُ المعلَّق إن وُجد.
func (s *Server) handleCashbox(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	pos, err := cashboxPosition(ctx, s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	all := r.URL.Query().Get("all") == "1"
	pg := pagingOf(r, 50)
	var total int
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*) FROM office_cash_entries WHERE $1 OR close_id IS NULL`, all).
		Scan(&total); err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT e.id::text, e.direction, e.amount, e.source, e.ref,
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''),
		       COALESCE(NULLIF(b.full_name, ''), b.phone::text, ''),
		       e.note, e.close_id IS NOT NULL, e.created_at
		  FROM office_cash_entries e
		  LEFT JOIN users u ON u.id = e.user_id
		  LEFT JOIN users b ON b.id = e.recorded_by
		 WHERE $1 OR e.close_id IS NULL
		 ORDER BY e.created_at DESC, e.id
		 LIMIT $2 OFFSET $3`, all, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []cashEntryRow{}
	for rows.Next() {
		var e cashEntryRow
		if err := rows.Scan(&e.ID, &e.Direction, &e.Amount, &e.Source, &e.Ref, &e.UserName,
			&e.RecordedBy, &e.Note, &e.Closed, &e.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	pending, err := s.closesWhere(ctx, s.pg, `c.status = 'pending'`, 1)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("entries", out, total, pg)
	res["position"] = pos
	if len(pending) > 0 {
		res["pending_close"] = pending[0]
	} else {
		res["pending_close"] = nil
	}
	httpx.JSON(w, http.StatusOK, res)
}

type cashCloseRow struct {
	ID           string     `json:"id"`
	Day          string     `json:"day"`
	Opening      int64      `json:"opening"`
	CashIn       int64      `json:"cash_in"`
	CashOut      int64      `json:"cash_out"`
	Expected     int64      `json:"expected"`
	Counted      int64      `json:"counted"`
	Difference   int64      `json:"difference"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	DeciderName  *string    `json:"decider_name"`
	DecidedAt    *time.Time `json:"decided_at"`
	DecisionNote string     `json:"decision_note"`
	SelfApproved bool       `json:"self_approved"`
	CreatedAt    time.Time  `json:"created_at"`
	// Shortfall **نقصُ هذا الإغلاق** إن سُجّل — وحالُه.
	ShortfallID     *string    `json:"shortfall_id"`
	ShortfallStatus *string    `json:"shortfall_status"`
	ShortfallDueAt  *time.Time `json:"shortfall_due_at"`
}

func (s *Server) closesWhere(ctx context.Context, q dbtx.Querier, where string, limit int, args ...any) ([]cashCloseRow, error) {
	args = append(args, limit)
	rows, err := q.Query(ctx, `
		SELECT c.id::text, c.day::text, c.opening, c.cash_in, c.cash_out, c.expected, c.counted,
		       c.difference, c.note, c.status, c.proposed_by::text,
		       COALESCE(NULLIF(p.full_name, ''), p.phone::text, ''),
		       NULLIF(COALESCE(NULLIF(d.full_name, ''), d.phone::text, ''), ''),
		       c.decided_at, c.decision_note, c.self_approved, c.created_at,
		       sf.id::text, sf.status, sf.eligible_at
		  FROM office_cash_closes c
		  JOIN users p ON p.id = c.proposed_by
		  LEFT JOIN users d ON d.id = c.decided_by
		  LEFT JOIN office_cash_shortfalls sf ON sf.close_id = c.id
		 WHERE `+where+`
		 ORDER BY c.created_at DESC
		 LIMIT $`+strconv.Itoa(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []cashCloseRow{}
	for rows.Next() {
		var c cashCloseRow
		if err := rows.Scan(&c.ID, &c.Day, &c.Opening, &c.CashIn, &c.CashOut, &c.Expected,
			&c.Counted, &c.Difference, &c.Note, &c.Status, &c.ProposedBy, &c.ProposerName,
			&c.DeciderName, &c.DecidedAt, &c.DecisionNote, &c.SelfApproved, &c.CreatedAt,
			&c.ShortfallID, &c.ShortfallStatus, &c.ShortfallDueAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// handleListCashCloses **سجلُّ الإغلاقات** — آخرُها أوّلاً.
func (s *Server) handleListCashCloses(w http.ResponseWriter, r *http.Request) {
	out, err := s.closesWhere(r.Context(), s.pg, `true`, 90)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"closes": out})
}

// handleCreateCashClose **«إغلاقُ اليوم»** — الموظّفُ يكتب ما عدّه، والنظامُ يحسب
// المتوقَّعَ والفرق ويوسم سطورَ الفترة بهذا الإغلاق. **وينتظر مراجعةَ شخصٍ ثانٍ.**
func (s *Server) handleCreateCashClose(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Counted *int64 `json:"counted"`
		Note    string `json:"note"`
	}](r)
	if err != nil || req.Counted == nil || *req.Counted < 0 {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)
	var row cashCloseRow
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		// **إغلاقٌ واحدٌ في كلّ مرّة** — يُقفَل الجدولُ لكتابةٍ حتّى تُحسب الفترةُ وتُوسَم.
		if _, err := q.Exec(ctx, `LOCK TABLE office_cash_closes IN SHARE ROW EXCLUSIVE MODE`); err != nil {
			return err
		}
		var pending, dayDone bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS (SELECT 1 FROM office_cash_closes WHERE status = 'pending'),
			       EXISTS (SELECT 1 FROM office_cash_closes
			                WHERE status = 'approved'
			                  AND day = (now() AT TIME ZONE 'Asia/Damascus')::date)`).
			Scan(&pending, &dayDone); err != nil {
			return err
		}
		if pending {
			return errCashClosePending
		}
		if dayDone {
			return errCashCloseDayDone
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
			       (day, opening, cash_in, cash_out, expected, counted, difference, amount, note, proposed_by)
			VALUES ((now() AT TIME ZONE 'Asia/Damascus')::date, $1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING id::text`,
			pos.Opening, pos.CashIn, pos.CashOut, pos.Expected, *req.Counted, diff, abs,
			clip(strings.TrimSpace(req.Note), 500), actor).Scan(&id); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE office_cash_entries SET close_id = $1 WHERE close_id IS NULL`, id); err != nil {
			return err
		}
		if err := s.auditTx(ctx, q, r, "finance.cash_close", "office_cash_close", id,
			map[string]any{"expected": pos.Expected, "counted": *req.Counted, "difference": diff}); err != nil {
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

// handleDecideCashClose **مراجعةُ الإغلاق** — شخصٌ غيرُ من عدّ.
//
// **الموافقةُ تثبّت المعدودَ أساساً للإغلاق التالي**، وإن كان ناقصاً سُجّل نقصٌ
// ينتظر: يُحَلّ إن وُجد المال، أو يصير خسارةً بعد يومٍ بموافقة مدير المنصّة.
// **والرفضُ يفكّ وسمَ السطور** فتعود إلى الفترة المفتوحة ويُعاد العدّ.
func (s *Server) handleDecideCashClose(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if !isUUID(id) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		req, err := decode[struct {
			Note string `json:"note"`
		}](r)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		note := clip(strings.TrimSpace(req.Note), 500)
		if !approve && note == "" {
			s.respondErr(w, errNoteRequired)
			return
		}
		actor := userIDFrom(r)
		var status string
		var self bool
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			var proposedBy, cur, day string
			var diff int64
			if err := q.QueryRow(ctx, `
				SELECT proposed_by::text, status, difference, day::text
				  FROM office_cash_closes WHERE id = $1 FOR UPDATE`, id).
				Scan(&proposedBy, &cur, &diff, &day); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return httpx.ErrNotFound
				}
				return err
			}
			if cur != "pending" {
				return errRequestDecided
			}
			var verdict approval.Verdict
			if approve {
				v, err := approval.Check(ctx, q, approval.Request{
					ProposedBy: proposedBy, Actor: actor, Capability: authz.FinanceManage})
				if err != nil {
					return err
				}
				verdict = v
			}
			self = verdict.SelfApproved
			status = "rejected"
			if approve {
				status = "approved"
			}
			if _, err := q.Exec(ctx, `
				UPDATE office_cash_closes
				   SET status = $2, decided_by = $3, decided_at = now(),
				       decision_note = $4, self_approved = $5
				 WHERE id = $1`, id, status, actor, note, self); err != nil {
				return err
			}
			if !approve {
				if _, err := q.Exec(ctx,
					`UPDATE office_cash_entries SET close_id = NULL WHERE close_id = $1`, id); err != nil {
					return err
				}
			}
			details := map[string]any{"difference": diff, "proposed_by": proposedBy, "note": note}
			if approve && diff < 0 {
				// **النقصُ يُسجَّل ولا يُقيَّد** — ينتظر يوماً: يُحَلّ أو يصير خسارة.
				var sid string
				if err := q.QueryRow(ctx, `
					INSERT INTO office_cash_shortfalls (close_id, amount, note, proposed_by, eligible_at)
					VALUES ($1, $2, $3, $4, now() + interval '1 day')
					RETURNING id::text`, id, -diff, "نقصُ صندوق المكتب — إغلاقُ يوم "+day, proposedBy).
					Scan(&sid); err != nil {
					return err
				}
				details["shortfall_id"] = sid
			}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			action := "finance.cash_close_rejected"
			if approve {
				action = "finance.cash_close_approved"
			}
			return s.auditTx(ctx, q, r, action, "office_cash_close", id, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": status, "self_approved": self})
	}
}

// handleResolveShortfall **وُجد المالُ الناقص** — يدخل الدرجَ سطراً، ويُغلَق النقص.
func (s *Server) handleResolveShortfall(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 500)
	if note == "" {
		s.respondErr(w, errNoteRequired)
		return
	}
	actor := userIDFrom(r)
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var amount int64
		var status string
		if err := q.QueryRow(ctx, `
			SELECT amount, status FROM office_cash_shortfalls WHERE id = $1 FOR UPDATE`, id).
			Scan(&amount, &status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			return err
		}
		if status != "pending" {
			return errRequestDecided
		}
		if err := officecash.Record(ctx, q, officecash.Entry{
			Direction: officecash.In, Amount: amount, Source: officecash.SourceShortfallFound,
			Ref: id, Actor: actor, Note: note,
		}); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE office_cash_shortfalls
			   SET status = 'resolved', decided_by = $2, decided_at = now(), decision_note = $3
			 WHERE id = $1`, id, actor, note); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.cash_shortfall_resolved", "office_cash_shortfall", id,
			map[string]any{"amount": amount, "note": note})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"status": "resolved"})
}

// handleDecideShortfall **نقصٌ لم يُحَلّ يومَه يصير خسارةً — بموافقة مدير المنصّة.**
//
// **الموافقةُ قيدُ `platform_expense` في الخزينة بمرجع النقص** — فتراه صفحةُ
// الخسائر والأرباح. **والرفضُ لا يقيّد شيئاً**: يبقى النقصُ مسجَّلاً على من أغلق.
func (s *Server) handleDecideShortfall(approve bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		if !isUUID(id) {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		req, err := decode[struct {
			Note string `json:"note"`
		}](r)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		note := clip(strings.TrimSpace(req.Note), 500)
		actor := userIDFrom(r)
		var status string
		var self bool
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			var amount int64
			var cur, proposedBy, srcNote string
			var due bool
			if err := q.QueryRow(ctx, `
				SELECT amount, status, proposed_by::text, note, eligible_at <= now()
				  FROM office_cash_shortfalls WHERE id = $1 FOR UPDATE`, id).
				Scan(&amount, &cur, &proposedBy, &srcNote, &due); err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return httpx.ErrNotFound
				}
				return err
			}
			if cur != "pending" {
				return errRequestDecided
			}
			if !due {
				return errShortfallNotDue
			}
			var verdict approval.Verdict
			status = "rejected"
			if approve {
				v, err := approval.Check(ctx, q, approval.Request{
					ProposedBy: proposedBy, Actor: actor, Capability: authz.TreasuryManage})
				if err != nil {
					return err
				}
				verdict = v
				status = "approved"
				if s.orders.TreasuryID(ctx) == "" {
					return errTreasuryMissing
				}
				if err := s.orders.DebitTreasury(ctx, q, amount, id, srcNote, actor); err != nil {
					return err
				}
			}
			self = verdict.SelfApproved
			if _, err := q.Exec(ctx, `
				UPDATE office_cash_shortfalls
				   SET status = $2, decided_by = $3, decided_at = now(),
				       decision_note = $4, self_approved = $5
				 WHERE id = $1`, id, status, actor, note, self); err != nil {
				return err
			}
			details := map[string]any{"amount": amount, "proposed_by": proposedBy, "note": note}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			action := "finance.cash_shortfall_rejected"
			if approve {
				action = "finance.cash_shortfall_loss"
			}
			return s.auditTx(ctx, q, r, action, "office_cash_shortfall", id, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": status, "self_approved": self})
	}
}
