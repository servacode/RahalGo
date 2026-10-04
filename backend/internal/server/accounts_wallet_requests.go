package server

// ══════════════════════════════════════════════════════════════════════
// **حركةُ المحفظة اليدويّة طلبٌ تقرّره الماليّة** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت قيداً بطرفٍ واحدٍ بكبسة موظّفٍ واحد**: «شحن» أو «تعويض» يزيد رصيدَ
// الشخص ولا ينقص من الخزينة ولا يدخل صندوقَ المكتب — **مالٌ يُخلق من عدم.**
//
// **والآن**: الموظّفُ يقترح (مبلغٌ تحت سقفٍ + ملاحظةٌ إلزاميّة + مفتاحُ عدمِ
// تكرار)، **وغيرُه يوافق** فيقع القيدُ بطرفين في معاملةٍ واحدة:
//
//	topup         نقدٌ دخل المكتب ⇒ +المحفظة · وسطرٌ داخلٌ في صندوق المكتب
//	compensation  ⇒ +المحفظة · −الخزينة (`platform_expense`)
//	adjustment    إيداعٌ ⇒ +المحفظة −الخزينة · خصمٌ ⇒ −المحفظة +الخزينة
//
// **ولا سحبَ من هنا** — السحبُ بابُه صفحةُ السحب وحدَها.
//
// **وصاحبُ الاقتراح لا يوافق عليه** — إلّا المالكُ الأعلى إن لم يكن في المنصّة
// من يملك الموافقةَ غيرُه، **ويُعلَّم ذلك في الطلب والسجلّ** (`approval.Check`).

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/approval"
	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/officecash"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	errWalletNoteRequired = httpx.NewError(http.StatusBadRequest,
		"wallet_note_required", "errors.wallet_note_required")
	errWalletOverCap = httpx.NewError(http.StatusBadRequest,
		"wallet_over_cap", "errors.wallet_over_cap")
	errWalletPayoutHere = httpx.NewError(http.StatusBadRequest,
		"wallet_payout_not_here", "errors.wallet_payout_not_here")
	errRequestDecided = httpx.NewError(http.StatusConflict,
		"request_decided", "errors.request_decided")
)

// walletManualMax سقفُ الحركة اليدويّة الواحدة — من الإعدادات.
func (s *Server) walletManualMax(ctx context.Context) int64 {
	return s.settings.GetNum(ctx, "finance.manual_wallet_max", 500000)
}

type walletRequestRow struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	UserName     string     `json:"user_name"`
	Kind         string     `json:"kind"`
	Debit        bool       `json:"debit"`
	Amount       int64      `json:"amount"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	DecidedBy    *string    `json:"decided_by"`
	DeciderName  *string    `json:"decider_name"`
	DecidedAt    *time.Time `json:"decided_at"`
	DecisionNote string     `json:"decision_note"`
	SelfApproved bool       `json:"self_approved"`
	CreatedAt    time.Time  `json:"created_at"`
}

const walletRequestCols = `
	SELECT wr.id::text, wr.user_id::text, COALESCE(NULLIF(u.full_name, ''), u.phone, ''),
	       wr.kind, wr.debit, wr.amount, wr.note, wr.status,
	       wr.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''),
	       wr.decided_by::text, NULLIF(COALESCE(NULLIF(d.full_name, ''), d.phone, ''), ''),
	       wr.decided_at, wr.decision_note, wr.self_approved, wr.created_at
	FROM wallet_requests wr
	JOIN users u ON u.id = wr.user_id
	JOIN users p ON p.id = wr.proposed_by
	LEFT JOIN users d ON d.id = wr.decided_by`

func scanWalletRequest(row pgx.Row) (walletRequestRow, error) {
	var w walletRequestRow
	err := row.Scan(&w.ID, &w.UserID, &w.UserName, &w.Kind, &w.Debit, &w.Amount, &w.Note,
		&w.Status, &w.ProposedBy, &w.ProposerName, &w.DecidedBy, &w.DeciderName,
		&w.DecidedAt, &w.DecisionNote, &w.SelfApproved, &w.CreatedAt)
	return w, err
}

// handleAdminWalletApply **يقترح حركةً يدويّة** — لا يقيّد شيئاً.
//
// **والاسمُ باقٍ** لأنّ بابَه باقٍ (`POST /admin/users/{id}/wallet`)، **ومعناه
// تبدّل**: الردُّ ٢٠١ بطلبٍ معلَّق، والقيدُ عند الموافقة.
func (s *Server) handleAdminWalletApply(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Kind   string `json:"kind"`
		Debit  bool   `json:"debit"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	target := chi.URLParam(r, "id")
	if !isUUID(target) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **ومفتاحُ عدمِ التكرار ترسله الشاشةُ دائماً** — كبستان سريعتان كانتا تكتبان الشحنَ
	// مرّتين؛ والوسيطُ (`idempotent`) يُعيد الجوابَ الأوّلَ لمفتاحٍ تكرّر.
	switch req.Kind {
	case "payout":
		s.respondErr(w, errWalletPayoutHere)
		return
	case "topup", "compensation", "adjustment":
	default:
		s.respondErr(w, errValidation)
		return
	}
	if req.Amount <= 0 {
		s.respondErr(w, wallet.ErrInvalidAmount)
		return
	}
	if req.Amount > s.walletManualMax(r.Context()) {
		s.respondErr(w, errWalletOverCap)
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.respondErr(w, errWalletNoteRequired)
		return
	}
	debit := req.Kind == "adjustment" && req.Debit
	actor := userIDFrom(r)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var id string
		if err := q.QueryRow(ctx, `
			INSERT INTO wallet_requests (user_id, kind, debit, amount, note, proposed_by)
			SELECT $1, $2, $3, $4, $5, $6 WHERE EXISTS (SELECT 1 FROM users WHERE id = $1)
			RETURNING id::text`, target, req.Kind, debit, req.Amount, clip(note, 500), actor).
			Scan(&id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return IdempotentBody{}, httpx.ErrNotFound
			}
			return IdempotentBody{}, err
		}
		if err := s.auditTx(ctx, q, r, "finance.wallet_request", "user", target,
			map[string]any{"request_id": id, "kind": req.Kind, "debit": debit,
				"amount": req.Amount, "note": note}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"request_id": id, "status": "pending"},
			// **ولا إشعارَ لكلّ مكتب العمليّات عن كلّ اقتراح** — يظهر في لوح الطلبات حيّاً
			// (`touch`)، **وإشعارٌ لكلّ الطاقم عن كلّ حركةٍ يُغرق جرسَهم وطابورَ الدفع.**
			AfterCommit: func() { s.touch("wallet", "ops") },
		}, nil
	})
}

// handleListWalletRequests الطلباتُ — المعلَّقةُ افتراضاً، أو طلباتُ حسابٍ بعينه.
func (s *Server) handleListWalletRequests(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	status := q.Get("status")
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	user := q.Get("user_id")
	if user != "" && !isUUID(user) {
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(), walletRequestCols+`
		WHERE ($1 = '' OR wr.status = $1)
		  AND ($2 = '' OR wr.user_id::text = $2)
		ORDER BY wr.created_at DESC LIMIT 200`, status, user)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []walletRequestRow{}
	for rows.Next() {
		row, err := scanWalletRequest(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": out,
		"cap": s.walletManualMax(r.Context())})
}

// **وحكمُ «المقترحُ غيرُ الموافق» في `approval.Check`** — مصدرٌ واحدٌ لكلّ أقسام
// المال (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الأدوار، البند ٦): الموافقةُ الذاتيّةُ
// للمالك الأعلى وحدَه وحين لا يوجد غيرُه، وتُعلَّم في السجلّ.

// handleDecideWalletRequest يوافق على الطلب أو يرفضه.
func (s *Server) handleDecideWalletRequest(approve bool) http.HandlerFunc {
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
		actor := userIDFrom(r)
		var wr walletRequestRow
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			row, err := scanWalletRequest(q.QueryRow(ctx, walletRequestCols+`
				WHERE wr.id = $1 FOR UPDATE OF wr`, id))
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			if err != nil {
				return err
			}
			if row.Status != "pending" {
				return errRequestDecided
			}
			var verdict approval.Verdict
			if approve {
				v, err := approval.Check(ctx, q, approval.Request{
					ProposedBy: row.ProposedBy, Actor: actor,
					Capability: authz.FinanceManage,
				})
				if err != nil {
					return err
				}
				verdict = v
			}
			self := verdict.SelfApproved
			status := "rejected"
			if approve {
				status = "approved"
				if err := s.postWalletRequest(ctx, q, row, actor); err != nil {
					return err
				}
			}
			if _, err := q.Exec(ctx, `
				UPDATE wallet_requests
				   SET status = $2, decided_by = $3, decided_at = now(),
				       decision_note = $4, self_approved = $5
				 WHERE id = $1`, id, status, actor, clip(strings.TrimSpace(req.Note), 500),
				self && approve); err != nil {
				return err
			}
			action := "finance.wallet_request_rejected"
			if approve {
				action = "finance.wallet_request_approved"
			}
			row.Status = status
			row.SelfApproved = self && approve
			wr = row
			details := map[string]any{
				"request_id": id, "kind": row.Kind, "debit": row.Debit, "amount": row.Amount,
				"proposed_by": row.ProposedBy, "note": strings.TrimSpace(req.Note)}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			return s.auditTx(ctx, q, r, action, "user", row.UserID, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		if approve {
			title := notifTitles.walletCredit
			if wr.Debit {
				title = notifTitles.walletDebit
			}
			s.notify.Notify(r.Context(), notifications.Input{
				UserID: wr.UserID, Kind: notifications.KindWallet,
				Title: title, Body: wr.Note, Entity: "wallet", Href: "/portal/wallet",
			})
			s.touchUser(wr.UserID, "wallet")
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": wr.Status,
			"self_approved": wr.SelfApproved})
	}
}

// postWalletRequest **القيدُ بطرفين** — داخلَ معاملة الموافقة.
//
// **ومرجعُ القيدين معرّفُ الطلب** — فيُقرأ من الدفتر من أين جاء كلُّ طرف.
func (s *Server) postWalletRequest(ctx context.Context, q dbtx.Querier, wr walletRequestRow, actor string) error {
	switch {
	case wr.Kind == "topup":
		if _, err := s.wallet.ApplyTx(ctx, q, wr.UserID, wr.Amount, "topup", wr.ID, wr.Note, &actor); err != nil {
			return err
		}
		// **والنقدُ دخل المكتب** — سطرٌ في صندوقه بالمرجع نفسِه.
		return officecash.Record(ctx, q, officecash.Entry{
			Direction: officecash.In, Amount: wr.Amount, Source: officecash.SourceWalletTopup,
			Ref: wr.ID, UserID: wr.UserID, Actor: actor, Note: wr.Note,
		})
	case wr.Kind == "adjustment" && wr.Debit:
		if _, err := s.wallet.ApplyTx(ctx, q, wr.UserID, -wr.Amount, "adjustment", wr.ID, wr.Note, &actor); err != nil {
			return err
		}
		return s.orders.CreditTreasuryDirect(ctx, q, wr.Amount, wr.ID, "تسويةٌ يدويّة — خصمٌ من محفظة", actor)
	default: // compensation · adjustment إيداعاً
		if _, err := s.wallet.ApplyTx(ctx, q, wr.UserID, wr.Amount, wr.Kind, wr.ID, wr.Note, &actor); err != nil {
			return err
		}
		return s.orders.DebitTreasury(ctx, q, wr.Amount, wr.ID, "حركةٌ يدويّةٌ بموافقة — "+wr.Note, actor)
	}
}
