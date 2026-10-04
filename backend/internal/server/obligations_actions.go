package server

// ══════════════════════════════════════════════════════════════════════
// **الديون — دفعٌ بالمكتب وشطبٌ وسدادٌ من الشحن** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الدينُ لا يُسدَّد إلّا من مستحقّاتٍ قادمة** — فمتجرٌ لا يبيع لا يسدّ
// أبداً، ومندوبٌ ترك يبقى دينُه للأبد.
//
//	دفع نقدي بالمكتب  الماليّةُ تقترح، وموظّفٌ ماليٌّ آخرُ يوافق ⇒ نقدٌ داخلٌ
//	                   في صندوق المكتب + قيدٌ موجبٌ في الخزينة + سطرُ تسوية
//	الشطب             الماليّةُ تقترح بسببٍ مكتوب، ومديرُ المنصّة يوافق
//	                   (`finance.writeoff.approve`) ⇒ سطرُ تسويةٍ «شُطب» بلا قيد
//	الشحن             شحنُ المحفظة يسدّ الدينَ أوّلاً ⇒ −المحفظة +الخزينة
//
// # ولماذا الشطبُ بلا قيدٍ في الدفتر
//
// **الخزينةُ تحمّلت المالَ يومَ نشأ الدين** — الاستردادُ أو أجرُ السائق خرج
// منها، والدينُ ذمّةٌ خارجَ الدفتر (هجرة `0130`). **فقيدُ خسارةٍ ثانٍ يعدّ
// الخسارةَ مرّتين.** والشطبُ يُقرأ خسارةً من `obligation_requests` المقبولة
// (`kind = 'write_off'`) — صفحتا الخزينة والخسائر تقرآنه من هناك.

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
	"github.com/servacode/rahalgo/backend/internal/obligations"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	errObligationClosed = httpx.NewError(http.StatusConflict,
		"obligation_closed", "errors.obligation_closed")
	errObligationOverAmount = httpx.NewError(http.StatusBadRequest,
		"obligation_over_amount", "errors.obligation_over_amount")
	errObligationNoteRequired = httpx.NewError(http.StatusBadRequest,
		"obligation_note_required", "errors.obligation_note_required")
	errObligationPending = httpx.NewError(http.StatusConflict,
		"obligation_request_pending", "errors.obligation_request_pending")
	errWriteoffNotYours = httpx.NewError(http.StatusForbidden,
		"writeoff_owner_only", "errors.writeoff_owner_only")
)

// أنواعُ الطلب — **وهي قيدُ `CHECK` في `obligation_requests.kind`.**
const (
	oblReqOfficeCash = "office_cash"
	oblReqWriteOff   = "write_off"
)

// obligationParty **صاحبُ محفظة الطرف** — مالكُ المتجر أو المندوبُ نفسُه.
func obligationPartyUser(ctx context.Context, q dbtx.Querier, kind, id string) (string, error) {
	if kind == obligations.PartyRep {
		return id, nil
	}
	var owner string
	err := q.QueryRow(ctx,
		`SELECT COALESCE(owner_user_id::text, '') FROM merchants WHERE id = $1`, id).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return owner, err
}

// recordObligationCash **النقدُ داخلٌ صندوقَ المكتب** — موضعُ النداء الوحيد.
//
// (لوحةُ التنسيق: قسمُ الخزينة ينشر `officecash.Record` بمصدر `obligation_cash`.
// **وحتّى يُدمَج يُكتب السطرُ هنا بالشكل نفسِه** — ومن دمج بدّل هذا السطرَ وحدَه.)
func recordObligationCash(ctx context.Context, q dbtx.Querier, amount int64,
	ref, userID, actor, note string) error {
	var user any
	if userID != "" {
		user = userID
	}
	_, err := q.Exec(ctx, `
		INSERT INTO office_cash_entries (direction, amount, source, ref, user_id, recorded_by, note)
		VALUES ('in', $1, 'obligation_cash', $2, $3, $4, $5)
		ON CONFLICT (source, ref) DO NOTHING`, amount, ref, user, actor, note)
	return err
}

// handleProposeObligationCash «دفع نقدي بالمكتب» — **اقتراحٌ لا قيد.**
func (s *Server) handleProposeObligationCash(w http.ResponseWriter, r *http.Request) {
	s.proposeObligation(w, r, oblReqOfficeCash)
}

// handleProposeObligationWriteoff «شطب الدين» — **بسببٍ مكتوب، ويوافق مديرُ المنصّة.**
func (s *Server) handleProposeObligationWriteoff(w http.ResponseWriter, r *http.Request) {
	s.proposeObligation(w, r, oblReqWriteOff)
}

func (s *Server) proposeObligation(w http.ResponseWriter, r *http.Request, kind string) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.respondErr(w, errObligationNoteRequired)
		return
	}
	actor := userIDFrom(r)
	var reqID string
	var amount int64
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var rest int64
		var partyKind, partyID string
		err := q.QueryRow(ctx, `
			SELECT party_kind, party_id::text, amount - settled FROM financial_obligations
			 WHERE id = $1 AND closed_at IS NULL FOR UPDATE`, id).Scan(&partyKind, &partyID, &rest)
		if errors.Is(err, pgx.ErrNoRows) {
			return errObligationClosed
		}
		if err != nil {
			return err
		}
		// **الشطبُ للباقي كلِّه**، والدفعةُ جزءٌ منه أو كلُّه — ولا تتجاوزه.
		amount = rest
		if kind == oblReqOfficeCash {
			if req.Amount <= 0 {
				return wallet.ErrInvalidAmount
			}
			if req.Amount > rest {
				return errObligationOverAmount
			}
			amount = req.Amount
		}
		if err := q.QueryRow(ctx, `
			INSERT INTO obligation_requests (obligation_id, kind, amount, note, proposed_by)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
			id, kind, amount, clip(note, 500), actor).Scan(&reqID); err != nil {
			if isUniqueViolation(err) {
				return errObligationPending
			}
			return err
		}
		return s.auditTx(ctx, q, r, "finance.obligation_request", "obligation", id,
			map[string]any{"request_id": reqID, "kind": kind, "amount": amount,
				"note": note, "party_kind": partyKind, "party_id": partyID})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"request_id": reqID,
		"status": "pending", "amount": amount})
}

// obligationRequestRow طلبُ دفعٍ أو شطب.
type obligationRequestRow struct {
	ID           string     `json:"id"`
	ObligationID string     `json:"obligation_id"`
	Kind         string     `json:"kind"`
	Amount       int64      `json:"amount"`
	Note         string     `json:"note"`
	Status       string     `json:"status"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	DecidedBy    *string    `json:"decided_by"`
	DecidedAt    *time.Time `json:"decided_at"`
	DecisionNote string     `json:"decision_note"`
	SelfApproved bool       `json:"self_approved"`
	CreatedAt    time.Time  `json:"created_at"`
	PartyKind    string     `json:"party_kind"`
	PartyID      string     `json:"party_id"`
	PartyName    string     `json:"party_name"`
}

const obligationRequestCols = `
	SELECT rq.id::text, rq.obligation_id::text, rq.kind, rq.amount, rq.note, rq.status,
	       rq.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone, ''),
	       rq.decided_by::text, rq.decided_at, rq.decision_note, rq.self_approved, rq.created_at,
	       o.party_kind, o.party_id::text, COALESCE(mm.name, uu.full_name, '')
	FROM obligation_requests rq
	JOIN financial_obligations o ON o.id = rq.obligation_id
	JOIN users p ON p.id = rq.proposed_by
	LEFT JOIN merchants mm ON o.party_kind = 'merchant' AND mm.id = o.party_id
	LEFT JOIN users     uu ON o.party_kind = 'rep'      AND uu.id = o.party_id`

func scanObligationRequest(row pgx.Row) (obligationRequestRow, error) {
	var x obligationRequestRow
	err := row.Scan(&x.ID, &x.ObligationID, &x.Kind, &x.Amount, &x.Note, &x.Status,
		&x.ProposedBy, &x.ProposerName, &x.DecidedBy, &x.DecidedAt, &x.DecisionNote,
		&x.SelfApproved, &x.CreatedAt, &x.PartyKind, &x.PartyID, &x.PartyName)
	return x, err
}

// handleListObligationRequests الطلباتُ — المعلَّقةُ افتراضاً.
func (s *Server) handleListObligationRequests(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "":
		status = "pending"
	case "all":
		status = ""
	case "pending", "approved", "rejected":
	default:
		s.respondErr(w, errValidation)
		return
	}
	rows, err := s.pg.Query(r.Context(), obligationRequestCols+`
		WHERE ($1 = '' OR rq.status = $1)
		ORDER BY rq.created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []obligationRequestRow{}
	for rows.Next() {
		x, err := scanObligationRequest(rows)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"requests": out})
}

// handleDecideObligationRequest يوافق على الطلب أو يرفضه.
//
// **والموافقُ غيرُ المقترِح** (`approval.Check`): الدفعةُ بـ`finance.manage`،
// **والشطبُ بـ`finance.writeoff.approve`** — لمدير المنصّة ومالكها.
func (s *Server) handleDecideObligationRequest(approve bool) http.HandlerFunc {
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
		var row obligationRequestRow
		var partyUser string
		var remaining int64
		err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			x, err := scanObligationRequest(q.QueryRow(ctx, obligationRequestCols+`
				WHERE rq.id = $1 FOR UPDATE OF rq`, id))
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			if err != nil {
				return err
			}
			if x.Status != "pending" {
				return errRequestDecided
			}
			cap := authz.FinanceManage
			if x.Kind == oblReqWriteOff {
				cap = authz.FinanceWriteoffApprove
				if approve && !s.hasCapability(r, cap) {
					return errWriteoffNotYours
				}
			}
			var verdict approval.Verdict
			if approve {
				if verdict, err = approval.Check(ctx, q, approval.Request{
					ProposedBy: x.ProposedBy, Actor: actor, Capability: cap,
				}); err != nil {
					return err
				}
				if partyUser, err = obligationPartyUser(ctx, q, x.PartyKind, x.PartyID); err != nil {
					return err
				}
				if remaining, err = s.postObligationRequest(ctx, q, &x, partyUser, actor); err != nil {
					return err
				}
			}
			status := "rejected"
			if approve {
				status = "approved"
			}
			if _, err := q.Exec(ctx, `
				UPDATE obligation_requests
				   SET status = $2, decided_by = $3, decided_at = now(),
				       decision_note = $4, self_approved = $5, amount = $6
				 WHERE id = $1`, id, status, actor, clip(strings.TrimSpace(req.Note), 500),
				verdict.SelfApproved, x.Amount); err != nil {
				return err
			}
			x.Status = status
			x.SelfApproved = verdict.SelfApproved
			row = x
			details := map[string]any{"request_id": id, "kind": x.Kind, "amount": x.Amount,
				"proposed_by": x.ProposedBy, "note": strings.TrimSpace(req.Note),
				"party_kind": x.PartyKind, "party_id": x.PartyID}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			action := "finance.obligation_request_rejected"
			if approve {
				action = "finance.obligation_request_approved"
			}
			return s.auditTx(ctx, q, r, action, "obligation", x.ObligationID, details)
		})
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if approve && partyUser != "" {
			title, body := obligationNotice(row.Kind, row.Amount, remaining)
			s.notify.Notify(r.Context(), notifications.Input{
				UserID: partyUser, Kind: notifications.KindWallet,
				Title: title, Body: body, Entity: "wallet", Href: "/portal/wallet",
			})
			s.touchUser(partyUser, "wallet")
		}
		s.touch("wallet", "ops")
		httpx.JSON(w, http.StatusOK, map[string]any{"status": row.Status,
			"self_approved": row.SelfApproved, "remaining": remaining})
	}
}

// postObligationRequest **أثرُ الموافقة** — داخلَ معاملتها.
func (s *Server) postObligationRequest(ctx context.Context, q dbtx.Querier,
	x *obligationRequestRow, partyUser, actor string) (int64, error) {
	switch x.Kind {
	case oblReqOfficeCash:
		// **النقدُ دخل المكتب** ⇒ سطرٌ في صندوقه + المالُ عاد إلى الخزينة.
		if err := recordObligationCash(ctx, q, x.Amount, x.ID, partyUser, actor, x.Note); err != nil {
			return 0, err
		}
		txID, err := s.orders.CreditTreasuryDirectID(ctx, q, x.Amount, x.ID,
			"سداد دين نقدا بالمكتب — "+x.Note, actor)
		if err != nil {
			return 0, err
		}
		_, _, rest, err := obligations.SettleOne(ctx, q, x.ObligationID, x.Amount,
			obligations.MethodOfficeCash, x.ID, txID, &actor)
		if errors.Is(err, obligations.ErrClosed) {
			return 0, errObligationClosed
		}
		return rest, err
	default: // write_off — **الباقي كلُّه، كما هو ساعةَ الموافقة.**
		var rest int64
		err := q.QueryRow(ctx, `
			SELECT amount - settled FROM financial_obligations
			 WHERE id = $1 AND closed_at IS NULL FOR UPDATE`, x.ObligationID).Scan(&rest)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && rest <= 0) {
			return 0, errObligationClosed
		}
		if err != nil {
			return 0, err
		}
		x.Amount = rest
		_, _, left, err := obligations.SettleOne(ctx, q, x.ObligationID, rest,
			obligations.MethodWrittenOff, x.ID, 0, &actor)
		return left, err
	}
}

// obligationNotice **ما يصل صاحبَ الدين** — بلغة المكتب.
func obligationNotice(kind string, amount, remaining int64) (string, string) {
	if kind == oblReqWriteOff {
		return "انشطب دين عليك", "شطبت الإدارة دينا عليك بقيمة " + fmtMoneyAr(amount) + "."
	}
	body := "سجلنا دفعتك النقدية بالمكتب " + fmtMoneyAr(amount) + "."
	if remaining > 0 {
		body += " وبقي عليك " + fmtMoneyAr(remaining) + " من هذا الدين."
	}
	return "وصلت دفعتك على الدين", body
}

// payDebtFromTopup **الشحنُ يسدّ الدينَ أوّلاً** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
//
// يُنادى في معاملة موافقة الشحن بعد قيده: يُقتطع من المحفظة ما يحتمله الشحنُ
// والمتاحُ معاً (`adjustment` سالباً بمرجع طلب الشحن)، ويعود إلى الخزينة،
// **ويُكتب سطرُ تسويةٍ «من الشحن»** لكلّ دينٍ مسّه — الأقدمُ أوّلاً.
//
// يُرجع ما اقتُطع — ليُبلَّغ صاحبُه بعد الإيداع.
func (s *Server) payDebtFromTopup(ctx context.Context, q dbtx.Querier,
	userID string, topup int64, ref, actor string) (int64, error) {
	type party struct{ kind, id string }
	var parties []party
	rows, err := q.Query(ctx, `
		SELECT 'rep', $1::text WHERE EXISTS (
			SELECT 1 FROM financial_obligations
			 WHERE party_kind = 'rep' AND party_id = $1::uuid AND closed_at IS NULL)
		UNION ALL
		SELECT 'merchant', m.id::text FROM merchants m
		 WHERE m.owner_user_id = $1::uuid AND m.debt > 0`, userID)
	if err != nil {
		return 0, err
	}
	for rows.Next() {
		var p party
		if err := rows.Scan(&p.kind, &p.id); err != nil {
			rows.Close()
			return 0, err
		}
		parties = append(parties, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	var paid int64
	budget := topup
	for _, p := range parties {
		if budget <= 0 {
			break
		}
		debt, err := obligations.Balance(ctx, q, p.kind, p.id)
		if err != nil {
			return 0, err
		}
		avail, err := s.wallet.AvailableTx(ctx, q, userID)
		if err != nil {
			return 0, err
		}
		take := min(debt, budget, avail)
		if take <= 0 {
			continue
		}
		note, err := obligations.OffsetNote(ctx, q, p.kind, p.id, take)
		if err != nil {
			return 0, err
		}
		note = strings.Replace(note, "اقتطاع دين", "سداد دين من الشحن", 1)
		if _, _, err := s.wallet.ApplyTxID(ctx, q, userID, -take, "adjustment", ref, note, &actor); err != nil {
			return 0, err
		}
		txID, err := s.orders.CreditTreasuryDirectID(ctx, q, take, ref, note, actor)
		if err != nil {
			return 0, err
		}
		applied, err := obligations.SettleBy(ctx, q, p.kind, p.id, take, "", txID, &actor,
			obligations.MethodTopup, ref)
		if err != nil {
			return 0, err
		}
		if applied != take {
			return 0, errors.New("سدادُ الدين من الشحن لم يطابق")
		}
		paid += take
		budget -= take
	}
	return paid, nil
}

// myDebtLine دينٌ مفتوحٌ كما يراه صاحبُه في تطبيقه.
type myDebtLine struct {
	Amount int64  `json:"amount"`
	Cause  string `json:"cause"`
}

// myDebts **ديونُ صاحب الجلسة** — مندوباً أو مالكَ متجر. (قرارُ المالك ٥: «عليك
// كذا، بسبب كذا، وبينقطع من أول أرباح جاية».)
func (s *Server) myDebts(ctx context.Context, userID string) ([]myDebtLine, int64, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT o.cause, sum(o.amount - o.settled)::bigint
		  FROM financial_obligations o
		 WHERE o.closed_at IS NULL
		   AND ((o.party_kind = 'rep' AND o.party_id = $1::uuid)
		     OR (o.party_kind = 'merchant' AND o.party_id IN (
		            SELECT id FROM merchants WHERE owner_user_id = $1::uuid)))
		 GROUP BY o.cause ORDER BY min(o.created_at)`, userID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []myDebtLine{}
	var total int64
	for rows.Next() {
		var d myDebtLine
		if err := rows.Scan(&d.Cause, &d.Amount); err != nil {
			return nil, 0, err
		}
		total += d.Amount
		out = append(out, d)
	}
	return out, total, rows.Err()
}

// walletWithDebts **كشفُ المحفظة ومعه الدين** — والدينُ غائبٌ لمن لا دينَ عليه.
type walletWithDebts struct {
	*wallet.Statement
	Debts     []myDebtLine `json:"debts,omitempty"`
	DebtTotal int64        `json:"debt_total,omitempty"`
}

// statementWithDebts يضمّ ديونَ صاحب الكشف إلى كشفه.
func (s *Server) statementWithDebts(ctx context.Context, userID string, st *wallet.Statement) (walletWithDebts, error) {
	debts, total, err := s.myDebts(ctx, userID)
	if err != nil {
		return walletWithDebts{}, err
	}
	out := walletWithDebts{Statement: st}
	if total > 0 {
		out.Debts, out.DebtTotal = debts, total
	}
	return out, nil
}
