package server

// عرضُ الديون — «الديون» في الخزينة (قراراتُ المالك ٢٠٢٦-١٠-٠٤).
//
// **الجدولُ `financial_obligations` هو الحقيقة** (هجرة 0130): لكلّ دينٍ طرفٌ
// (متجرٌ أو مندوب) ومبلغٌ ومسدَّدٌ وسببٌ وطلبٌ ووقت. **والباقي = المبلغُ ناقصَ
// المسدَّد.**
//
// **والحالُ أربعة لا اثنان**: مفتوح · انسدّ · انشطب لأنّ الطلب ما صار ·
// شطبته الإدارة. **كان الملغى يُعرض «مسدَّداً» كأنّ المتجر دفع** — والفرقُ
// يُقرأ من طريقة التسوية (`method`، هجرة 0340).
//
// والأفعالُ في `obligations_actions.go`.

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// obligationSettlement سطرُ تسويةٍ واحد — **كم، وبأيّ طريقة، وكم بقي، ومتى.**
type obligationSettlement struct {
	Amount    int64     `json:"amount"`
	Remaining int64     `json:"remaining"`
	OrderNo   *int64    `json:"order_number"`
	Method    string    `json:"method"`
	CreatedAt time.Time `json:"created_at"`
}

// obligationPending طلبُ دفعٍ أو شطبٍ ينتظر الموافقة على هذا الدين.
type obligationPending struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	Note   string `json:"note"`
	// ProposedBy مقترحُ الطلب — **فلا يُعرَض له زرُّ موافقةٍ يردّه المحرّك**
	// (فحصُ المال ٢٠٢٦-١٠-٠٥: كان المقترحُ يرى «موافقة» فيُسأل كلمةَ سرّه ثمّ يُردّ ٤٠٣).
	ProposedBy string `json:"proposed_by"`
}

// obligationRow دينٌ واحدٌ كما يُعرَض — **بلا هاتفٍ ولا سرّ.**
type obligationRow struct {
	ID          string                 `json:"id"`
	PartyKind   string                 `json:"party_kind"`
	PartyID     string                 `json:"party_id"`
	PartyUserID string                 `json:"party_user_id"`
	PartyName   string                 `json:"party_name"`
	Amount      int64                  `json:"amount"`
	Outstanding int64                  `json:"outstanding"`
	Cause       string                 `json:"cause"`
	OrderID     *string                `json:"order_id"`
	OrderNo     *int64                 `json:"order_number"`
	CreatedAt   time.Time              `json:"created_at"`
	AgeDays     int64                  `json:"age_days"`
	State       string                 `json:"state"`
	Pending     *obligationPending     `json:"pending"`
	Settlements []obligationSettlement `json:"settlements"`
}

// obligationsSummary **مربّعاتُ الأعلى** — على كلّ المفتوح لا على الصفحة.
type obligationsSummary struct {
	OutstandingTotal int64 `json:"outstanding_total"`
	Merchants        int64 `json:"merchants_total"`
	Reps             int64 `json:"reps_total"`
	Debtors          int64 `json:"debtors"`
	NearLimit        int64 `json:"near_limit"`
	Overdue          int64 `json:"overdue"`
	PendingRequests  int64 `json:"pending_requests"`
}

// oblStateSQL **الحالُ المشتقّ** — مفتوح · انسدّ · انشطب لأنّ الطلب ما صار · شطبته الإدارة.
const oblStateSQL = `
	CASE WHEN o.closed_at IS NULL THEN 'open'
	     WHEN EXISTS (SELECT 1 FROM obligation_settlements x
	                   WHERE x.obligation_id = o.id AND x.method = 'written_off') THEN 'written_off'
	     WHEN EXISTS (SELECT 1 FROM obligation_settlements x
	                   WHERE x.obligation_id = o.id AND x.method = 'voided') THEN 'voided'
	     ELSE 'paid' END`

// oblFrom الجدولُ وصلاتُه — للعدّ وللقائمة وللتصدير.
const oblFrom = `
	FROM financial_obligations o
	LEFT JOIN merchants mm ON o.party_kind = 'merchant' AND mm.id = o.party_id
	LEFT JOIN users     uu ON o.party_kind = 'rep'      AND uu.id = o.party_id
	LEFT JOIN orders   ord ON ord.id = o.order_id`

// oblWhere **الشرطُ الواحدُ يُكتب مرّةً** — للعدّ وللقائمة وللتصدير.
//
//	$1 نوعُ الطرف · $2 الطرف · $3 الحال · $4 السبب · $5 البحث · $6 من · $7 إلى ·
//	$8 أيّامُ التأخّر (صفرٌ = بلا شرط)
const oblWhere = `
	WHERE ($1 = '' OR o.party_kind = $1)
	  AND ($2 = '' OR o.party_id::text = $2)
	  AND ($3 = '' OR ($3 = 'closed' AND o.closed_at IS NOT NULL) OR (` + oblStateSQL + `) = $3)
	  AND ($4 = '' OR o.cause = $4)
	  AND ($5 = '' OR COALESCE(mm.name, uu.full_name, '') ILIKE '%' || $5 || '%'
	               OR ord.number::text = $5)
	  AND ($6::date IS NULL OR o.created_at >= $6::date)
	  AND ($7::date IS NULL OR o.created_at < $7::date + 1)
	  AND ($8 = 0 OR (o.closed_at IS NULL AND o.created_at < now() - make_interval(days => $8)))`

// oblFilter المرشّحاتُ مقروءةً ومتحقَّقاً منها.
type oblFilter struct {
	kind, party, state, cause, query string
	from, to                         *string
	overdue                          int
}

var oblStates = map[string]bool{"": true, "open": true, "closed": true, "paid": true,
	"voided": true, "written_off": true}

func (s *Server) readOblFilter(r *http.Request) (oblFilter, bool) {
	q := r.URL.Query()
	f := oblFilter{kind: q.Get("party_kind"), party: q.Get("party_id"), state: q.Get("state"),
		cause: q.Get("cause"), query: strings.TrimSpace(q.Get("q"))}
	if f.kind != "" && f.kind != obligations.PartyMerchant && f.kind != obligations.PartyRep {
		return f, false
	}
	if !oblStates[f.state] {
		return f, false
	}
	if f.party != "" && !isUUID(f.party) {
		return f, false
	}
	for _, p := range []struct {
		key string
		dst **string
	}{{"from", &f.from}, {"to", &f.to}} {
		if v := q.Get(p.key); v != "" {
			if _, err := time.Parse("2006-01-02", v); err != nil {
				return f, false
			}
			vv := v
			*p.dst = &vv
		}
	}
	if q.Get("overdue") == "1" {
		f.overdue = s.obligationAlertDays(r.Context())
		if f.overdue <= 0 {
			f.overdue = 1 << 20 // **التنبيهُ مطفأ** ⇒ لا متأخّر.
		}
	}
	return f, true
}

func (f oblFilter) args() []any {
	return []any{f.kind, f.party, f.state, f.cause, f.query, f.from, f.to, f.overdue}
}

// obligationAlertDays **بعد كم يوماً ينبّه الدينُ المفتوح** — من الإعدادات، وصفرُه بلا تنبيه.
func (s *Server) obligationAlertDays(ctx context.Context) int {
	return int(s.settings.GetNum(ctx, "finance.obligation_alert_days", 30))
}

// handleListObligations الديون — ترشيحٌ وبحثٌ ومجاميعُ وصفحةٌ محدودة.
//
// **واسمُ الطرف من مصدره**: المتجرُ من `merchants` والمندوبُ من `users`.
func (s *Server) handleListObligations(w http.ResponseWriter, r *http.Request) {
	f, ok := s.readOblFilter(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	pg := pagingOf(r, 25)
	ctx := r.Context()

	var count int
	if err := s.pg.QueryRow(ctx, `SELECT count(*) `+oblFrom+oblWhere, f.args()...).
		Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}
	out, err := s.queryObligations(ctx, f, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	sum, err := s.obligationsSummary(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res := paged("obligations", out, count, pg)
	res["outstanding_total"] = sum.OutstandingTotal
	res["summary"] = sum
	res["alert_days"] = s.obligationAlertDays(ctx)
	res["can_manage"] = s.hasCapability(r, authz.FinanceManage)
	res["can_approve_writeoff"] = s.hasCapability(r, authz.FinanceWriteoffApprove)
	res["can_export"] = s.hasCapability(r, authz.FinanceExport)
	httpx.JSON(w, http.StatusOK, res)
}

// obligationsExportCap **سقفُ الملفّ** — ومن تجاوزه ضيّق المرشّحات.
const obligationsExportCap = 5000

// handleExportObligations الديون بمرشّحات الصفحة نفسِها — ملفّاً (`finance.export`).
func (s *Server) handleExportObligations(w http.ResponseWriter, r *http.Request) {
	f, ok := s.readOblFilter(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	out, err := s.queryObligations(r.Context(), f, obligationsExportCap+1, 0)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	truncated := len(out) > obligationsExportCap
	if truncated {
		out = out[:obligationsExportCap]
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"obligations": out,
		"truncated": truncated, "cap": obligationsExportCap})
}

func (s *Server) obligationsSummary(ctx context.Context) (obligationsSummary, error) {
	var x obligationsSummary
	days := s.obligationAlertDays(ctx)
	err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(sum(amount - settled), 0)::bigint,
		       COALESCE(sum(amount - settled) FILTER (WHERE party_kind = 'merchant'), 0)::bigint,
		       COALESCE(sum(amount - settled) FILTER (WHERE party_kind = 'rep'), 0)::bigint,
		       count(DISTINCT (party_kind, party_id)),
		       count(*) FILTER (WHERE $1 > 0 AND created_at < now() - make_interval(days => $1)),
		       (SELECT count(*) FROM merchants m
		         WHERE m.delivery_credit_limit > 0 AND m.debt * 5 >= m.delivery_credit_limit * 4),
		       (SELECT count(*) FROM obligation_requests WHERE status = 'pending')
		  FROM financial_obligations WHERE closed_at IS NULL`, days).
		Scan(&x.OutstandingTotal, &x.Merchants, &x.Reps, &x.Debtors, &x.Overdue,
			&x.NearLimit, &x.PendingRequests)
	return x, err
}

func (s *Server) queryObligations(ctx context.Context, f oblFilter, limit, offset int) ([]obligationRow, error) {
	args := append(f.args(), limit, offset)
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text, o.party_kind, o.party_id::text,
		       COALESCE(CASE WHEN o.party_kind = 'merchant' THEN mm.owner_user_id::text
		                     ELSE o.party_id::text END, ''),
		       COALESCE(mm.name, uu.full_name, ''),
		       o.amount, o.amount - o.settled, o.cause,
		       o.order_id::text, ord.number, o.created_at,
		       (extract(epoch FROM (COALESCE(o.closed_at, now()) - o.created_at)) / 86400)::bigint,
		       `+oblStateSQL+`,
		       rq.id::text, rq.kind, rq.amount, rq.note, rq.proposed_by::text
		`+oblFrom+`
		LEFT JOIN obligation_requests rq ON rq.obligation_id = o.id AND rq.status = 'pending'`+
		oblWhere+`
		ORDER BY o.created_at DESC, o.id
		LIMIT $9 OFFSET $10`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []obligationRow{}
	byID := map[string]int{}
	for rows.Next() {
		var x obligationRow
		var pID, pKind, pNote, pBy *string
		var pAmount *int64
		x.Settlements = []obligationSettlement{}
		if err := rows.Scan(&x.ID, &x.PartyKind, &x.PartyID, &x.PartyUserID, &x.PartyName,
			&x.Amount, &x.Outstanding, &x.Cause, &x.OrderID, &x.OrderNo, &x.CreatedAt,
			&x.AgeDays, &x.State, &pID, &pKind, &pAmount, &pNote, &pBy); err != nil {
			return nil, err
		}
		if pID != nil {
			x.Pending = &obligationPending{ID: *pID, Kind: *pKind, Amount: *pAmount, Note: *pNote}
			if pBy != nil {
				x.Pending.ProposedBy = *pBy
			}
		}
		byID[x.ID] = len(out)
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	if len(out) == 0 {
		return out, nil
	}

	// **وسطورُ التسوية لهذه الصفحة وحدَها** — نداءٌ واحدٌ لكلّ ديونها.
	ids := make([]string, 0, len(out))
	for _, x := range out {
		ids = append(ids, x.ID)
	}
	srows, err := s.pg.Query(ctx, `
		SELECT st.obligation_id::text, st.amount, st.remaining, ord.number, st.method, st.created_at
		FROM obligation_settlements st
		LEFT JOIN orders ord ON ord.id = st.order_id
		WHERE st.obligation_id::text = ANY($1)
		ORDER BY st.created_at, st.id`, ids)
	if err != nil {
		return nil, err
	}
	defer srows.Close()
	for srows.Next() {
		var oid string
		var st obligationSettlement
		if err := srows.Scan(&oid, &st.Amount, &st.Remaining, &st.OrderNo, &st.Method,
			&st.CreatedAt); err != nil {
			return nil, err
		}
		if i, ok := byID[oid]; ok {
			out[i].Settlements = append(out[i].Settlements, st)
		}
	}
	return out, srows.Err()
}
