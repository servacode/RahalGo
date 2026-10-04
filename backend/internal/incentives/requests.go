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
}

const requestCols = `
	SELECT r.id::text, r.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
	       r.kind, r.amount, r.note, r.status, r.source,
	       r.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''),
	       r.decided_by::text, NULLIF(COALESCE(NULLIF(d.full_name, ''), d.phone, ''), ''),
	       r.decided_at, r.decision_note, r.self_approved, r.created_at
	FROM incentive_requests r
	JOIN users u ON u.id = r.user_id
	JOIN users p ON p.id = r.proposed_by
	LEFT JOIN users d ON d.id = r.decided_by`

func scanRequest(row pgx.Row) (Request, error) {
	var x Request
	err := row.Scan(&x.ID, &x.UserID, &x.UserName, &x.Kind, &x.Amount, &x.Note, &x.Status,
		&x.Source, &x.ProposedBy, &x.ProposerName, &x.DecidedBy, &x.DeciderName,
		&x.DecidedAt, &x.DecisionNote, &x.SelfApproved, &x.CreatedAt)
	return x, err
}

// Proposal ما يقترحه الموظّف.
type Proposal struct {
	Actor, UserID, Kind, Note string
	Amount                    int64
	// Source `manual` من الصفحة، أو `alert` استرجاعٌ على تنبيه.
	Source           string
	AlertIncentiveID string
}

// Propose **يكتب الطلبَ ولا يمسّ المال.**
//
// **والعقوبةُ تُفحص هنا أيضاً** على المتاح — فيظهر «رصيده لا يكفي» داخلَ
// النافذة الآن، لا عند الموافقة بعد ساعة. **والموافقةُ تفحصه من جديد.**
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
	var ok bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM users u
		                 JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code IN ('driver', 'sales')
		                WHERE u.id = $1::uuid AND u.status = 'active' AND u.deleted_at IS NULL)`,
		p.UserID).Scan(&ok); err != nil {
		return "", err
	}
	if !ok {
		return "", ErrNotEligible
	}
	if p.Kind == KindPenalty {
		avail, err := AvailableOn(ctx, q, p.UserID)
		if err != nil {
			return "", err
		}
		if avail < p.Amount {
			return "", ErrNoBalance
		}
	}
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
		INSERT INTO incentive_requests (user_id, kind, amount, note, proposed_by, source, alert_incentive_id)
		VALUES ($1::uuid, $2, $3, $4, $5::uuid, $6, $7::uuid)
		RETURNING id::text`, p.UserID, p.Kind, p.Amount, note, p.Actor, src, alert).Scan(&id)
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
		e, err := s.GrantTx(ctx, q, actor, row.UserID, row.Kind, row.Amount, row.Note)
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
