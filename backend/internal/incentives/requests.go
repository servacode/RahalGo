package incentives

// ══════════════════════════════════════════════════════════════════════
// **المكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه شخصٌ ثانٍ**
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ الأهداف، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
// **كان «اعتمد» يُخرج المالَ من الخزينة فوراً بيد شخصٍ واحد** — بلا سقفٍ
// ولا تأكيد. والآن صفٌّ في `incentive_requests` (عقدُ الموافقات الموحّد)،
// **ولا مالَ قبل الموافقة**، وصاحبُ الاقتراح لا يوافق عليه (`approval.Check`).

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	// ErrNotEligible **لا يُكافأ ولا يُعاقب من هنا إلّا سائقٌ أو مندوبٌ فعّال.**
	ErrNotEligible = httpx.NewError(http.StatusConflict,
		"incentive_user_inactive", "errors.incentive_user_inactive")
	// ErrBadRole **دورٌ غيرُ السائق والمندوب.**
	ErrBadRole = httpx.NewError(http.StatusBadRequest, "bad_incentive_role", "errors.validation")
	// ErrRequestDecided **بُتّ فيه من قبل.**
	ErrRequestDecided = httpx.NewError(http.StatusConflict,
		"request_decided", "errors.request_decided")
)

// Request طلبُ مكافأةٍ أو عقوبة.
type Request struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	UserName     string     `json:"user_name"`
	Kind         string     `json:"kind"`
	Amount       int64      `json:"amount"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	Source       string     `json:"source"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	DecidedBy    *string    `json:"decided_by"`
	DeciderName  *string    `json:"decider_name"`
	DecidedAt    *time.Time `json:"decided_at"`
	DecisionNote string     `json:"decision_note"`
	SelfApproved bool       `json:"self_approved"`
	CreatedAt    time.Time  `json:"created_at"`
	// Role **الدورُ الذي اقتُرح له** — ويُعدّ فيه وحدَه (قرارُ المالك ٢٠٢٦-١٠-١٥).
	Role string `json:"role"`
}

const requestCols = `
	SELECT r.id::text, r.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
	       r.kind, r.amount, r.note, r.status, r.source,
	       r.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''),
	       r.decided_by::text, NULLIF(COALESCE(NULLIF(d.full_name, ''), d.phone, ''), ''),
	       r.decided_at, r.decision_note, r.self_approved, r.created_at, COALESCE(r.role, '')
	FROM incentive_requests r
	JOIN users u ON u.id = r.user_id
	JOIN users p ON p.id = r.proposed_by
	LEFT JOIN users d ON d.id = r.decided_by`

func scanRequest(row pgx.Row) (Request, error) {
	var x Request
	err := row.Scan(&x.ID, &x.UserID, &x.UserName, &x.Kind, &x.Amount, &x.Note, &x.Status,
		&x.Source, &x.ProposedBy, &x.ProposerName, &x.DecidedBy, &x.DeciderName,
		&x.DecidedAt, &x.DecisionNote, &x.SelfApproved, &x.CreatedAt, &x.Role)
	return x, err
}

// Proposal ما يقترحه الموظّف.
type Proposal struct {
	Actor, UserID, Kind, Note string
	Amount                    int64
	// Source `manual` من الصفحة، أو `alert` استرجاعٌ على تنبيه.
	Source           string
	AlertIncentiveID string
	// Role **الدورُ الذي يُقترح له** — `driver` أو `sales`. وفراغُه يُستنتج من أدواره
	// (السائقُ أوّلاً)، **وما لا يحمله صاحبُه يُردّ.**
	Role string
}

// Propose **يكتب الطلبَ ولا يمسّ المال.**
//
// **والعقوبةُ لا تُردّ لقلّة الرصيد** (قرارُ المالك ٢٠٢٦-١٠-١٥) — ما زاد على
// المتاح ساعةَ الموافقة يصير ديناً.
func (s *Service) Propose(ctx context.Context, q dbtx.Querier, p Proposal) (string, error) {
	if p.Kind != KindReward && p.Kind != KindPenalty {
		return "", ErrBadKind
	}
	if p.Amount <= 0 {
		return "", ErrBadAmount
	}
	note := strings.TrimSpace(p.Note)
	if note == "" {
		return "", ErrNeedsReason
	}
	if len([]rune(note)) > 500 {
		note = string([]rune(note)[:500])
	}
	if p.Role != "" && p.Role != "driver" && p.Role != "sales" {
		return "", ErrBadRole
	}
	// **والدورُ يُكتب على الطلب** (قرارُ المالك ٢٠٢٦-١٠-١٥) — من له الدوران معاً كان
	// يُعدّ في كروت التبويبين. **وما سُمّي يُشترط أن يحمله**، وفراغُه أوّلُ ما يحمل.
	var role string
	if err := q.QueryRow(ctx, `
		SELECT COALESCE((SELECT ur.role_code FROM users u
		                   JOIN user_roles ur ON ur.user_id = u.id
		                  WHERE u.id = $1::uuid AND u.status = 'active' AND u.deleted_at IS NULL
		                    AND ur.role_code IN ('driver', 'sales')
		                    AND ($2 = '' OR ur.role_code = $2)
		                  ORDER BY ur.role_code LIMIT 1), '')`,
		p.UserID, p.Role).Scan(&role); err != nil {
		return "", err
	}
	if role == "" {
		return "", ErrNotEligible
	}
	// **ولا تُردّ العقوبةُ لقلّة الرصيد** (قرارُ المالك ٢٠٢٦-١٠-١٥): يُؤخذ المتاحُ
	// عند الموافقة ويصير الباقي ديناً (`GrantTx`). **والمقترَحُ عليه سائقٌ أو مندوبٌ
	// حتماً** (الفحصُ أعلاه) — فلدينه طرفٌ دائماً.
	src := p.Source
	if src == "" {
		src = "manual"
	}
	var alert *string
	if p.AlertIncentiveID != "" {
		alert = &p.AlertIncentiveID
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO incentive_requests (user_id, kind, amount, note, proposed_by, source,
		                                alert_incentive_id, role)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid, $6, $7::uuid, $8)
		RETURNING id::text`, p.UserID, p.Kind, p.Amount, note, p.Actor, src, alert, role).Scan(&id)
	return id, err
}

// ListRequests **الطلبات** — المعلَّقةُ افتراضاً، أو طلباتُ حسابٍ بعينه.
func (s *Service) ListRequests(ctx context.Context, status, userID string) ([]Request, error) {
	if status == "all" {
		status = ""
	}
	rows, err := s.db.Query(ctx, requestCols+`
		WHERE ($1 = '' OR r.status = $1)
		  AND ($2 = '' OR r.user_id::text = $2)
		ORDER BY r.created_at DESC LIMIT 200`, status, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Request{}
	for rows.Next() {
		x, err := scanRequest(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// RequestByID **طلبٌ بمعرّفه** — لإشعار من يقرّر فيه.
func (s *Service) RequestByID(ctx context.Context, id string) (Request, bool) {
	x, err := scanRequest(s.db.QueryRow(ctx, requestCols+` WHERE r.id = $1::uuid`, id))
	return x, err == nil
}

// DecideRequest **يوافق أو يرفض — داخلَ المعاملة المُمرَّرة.**
//
// **والموافقةُ تقيّد المالَ بطرفين** (`GrantTx`: المحفظةُ والخزينة) وتربط
// الطلبَ بصفّ الحافز. **وصاحبُ الاقتراح يُردّ** إلّا المالكَ الأعلى وحدَه
// بلا بديل (`approval.Check`) — ويُعلَّم.
func (s *Service) DecideRequest(ctx context.Context, q dbtx.Querier, id, actor string,
	approve bool, note string) (Request, approval.Verdict, error) {
	var verdict approval.Verdict
	row, err := scanRequest(q.QueryRow(ctx, requestCols+` WHERE r.id = $1::uuid FOR UPDATE OF r`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return row, verdict, httpx.ErrNotFound
	}
	if err != nil {
		return row, verdict, err
	}
	if row.Status != "pending" {
		return row, verdict, ErrRequestDecided
	}
	status := "rejected"
	var incID *string
	if approve {
		verdict, err = approval.Check(ctx, q, approval.Request{
			ProposedBy: row.ProposedBy, Actor: actor, Capability: authz.FinanceManage,
		})
		if err != nil {
			return row, verdict, err
		}
		e, err := s.GrantRoleTx(ctx, q, actor, row.UserID, row.Kind, row.Amount, row.Note, row.Role)
		if err != nil {
			return row, verdict, err
		}
		incID = &e.ID
		status = "approved"
	}
	if _, err := q.Exec(ctx, `
		UPDATE incentive_requests
		   SET status = $2, decided_by = $3::uuid, decided_at = now(),
		       decision_note = $4, self_approved = $5, incentive_id = $6::uuid
		 WHERE id = $1::uuid`, id, status, actor, strings.TrimSpace(note),
		verdict.SelfApproved, incID); err != nil {
		return row, verdict, err
	}
	row.Status = status
	row.SelfApproved = verdict.SelfApproved
	return row, verdict, nil
}
