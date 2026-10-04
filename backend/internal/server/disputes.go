package server

// نزاعاتُ المنصة — **مع أربعةٍ لا مع واحد.**
//
// # المسألة
//
// كان النزاعُ مع المتجر وحدَه، ومكانُه صفُّ الإنذار (`merchant_warnings`).
// **ولا مكانَ لنزاعٍ مع سائقٍ أو زبونٍ أو مندوب** — لأنّ هؤلاء لا إنذاراتِ
// لهم، والمالُ كان يسكن في صفّ الإنذار.
//
// وهي نزاعاتٌ تقع فعلاً: **سائقٌ لم يسلّم صندوقَه · ومندوبٌ أخذ عمولةً على
// متجرٍ لم يعمل · وزبونٌ استلم ولم يدفع.** فتُدار بالهاتف وتُنسى، **ولا يعرف
// أحدٌ كم لنا عند الناس مجموعاً.**
//
// قرارُ المالك (٢٠٢٦-٠٨-٠٣): «قسمُ النزاعات يحوي تبويباً للمناديب والمتاجر
// والسائقين والزبائن لنعرف منازعةَ المنصة مع من».
//
// # والحسمُ اتجاهان لا واحد
//
//   - **`charge`** — يُخصم من محفظة الطرف ويعود إلى الخزينة. **مالٌ تحرّك.**
//   - **`waive`**  — تتحمّله المنصة. **قرارٌ لا حركة**، ولا قيدَ له.
//
// **وخلطُهما يجعل تقريرَ الخسائر يقرأ إسقاطاً تحصيلاً.**
//
// # وقراراتُ المالك ٢٠٢٦-١٠-٠٤ («الخسائر والنزاعات»)
//
//  1. **الماليّةُ ترى وتحسم، والدعمُ يرى ويفتح** — قدرةُ `disputes.manage`
//     للعرض والفتح، و`finance.manage` للحسم.
//  2. **الحسمُ اقتراحٌ وموافقةُ شخصٍ آخر** — جدولُ `dispute_resolutions`،
//     والموافقةُ تقرأ `approval.Check` كباقي المال.
//  3. **رصيدٌ لا يكفي: يُخصم الموجودُ ويبقى الباقي مفتوحاً** — عمودُ
//     `recovered`، والمفتوحُ = المبلغ − المسترَدّ.
//  5. **كلُّ خصمٍ بيد إنسانٍ بعد سماع الطرف** — ولا خصمَ آليّاً في المحرّك.

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
)

var (
	errClaimSettled = httpx.NewError(http.StatusConflict,
		"claim_already_settled", "errors.claim_already_settled")
	errDisputeNoWallet = httpx.NewError(http.StatusConflict,
		"dispute_party_has_no_wallet", "errors.dispute_party_has_no_wallet")
	// **الطرفُ المختارُ لا يحمل الدورَ المختار** — سائقٌ لا يُسجَّل زبوناً،
	// **ومتجرٌ يُختار بمعرّفه لا بمعرّف صاحبه.**
	errDisputePartyMismatch = httpx.NewError(http.StatusBadRequest,
		"dispute_party_mismatch", "errors.dispute_party_mismatch")
	// **اقتراحٌ معلَّقٌ واحدٌ لكلّ نزاع** — يُبتّ فيه قبل غيره.
	errDisputeResolutionPending = httpx.NewError(http.StatusConflict,
		"dispute_resolution_pending", "errors.dispute_resolution_pending")
	// **رصيدُ الطرف صفرٌ الآن** — لا شيءَ يُخصم، والنزاعُ يبقى مفتوحاً.
	errDisputeNothingToCharge = httpx.NewError(http.StatusConflict,
		"dispute_nothing_to_charge", "errors.dispute_nothing_to_charge")
)

// disputeParties الأدوارُ التي يجوز أن نتنازع معها — **مرآةُ قيد القاعدة.**
var disputeParties = map[string]bool{
	"merchant": true, "driver": true, "sales": true, "customer": true,
}

// disputePending **اقتراحُ الحسم المعلَّق على النزاع** — إن وُجد.
type disputePending struct {
	ID           string    `json:"id"`
	Action       string    `json:"action"`
	Note         string    `json:"note"`
	ProposedBy   string    `json:"proposed_by"`
	ProposerName string    `json:"proposer_name"`
	CreatedAt    time.Time `json:"created_at"`
}

type disputeRow struct {
	ID        string `json:"id"`
	PartyRole string `json:"party_role"`
	// PartyID **معرّفُ الطرف في عموده** — المتجرُ بمعرّف المتجر، والباقي بمعرّف الحساب.
	PartyID    string  `json:"party_id"`
	PartyName  string  `json:"party_name"`
	PartyPhone string  `json:"party_phone"`
	Reason     string  `json:"reason"`
	Note       string  `json:"note"`
	Amount     int64   `json:"amount"`
	Recovered  int64   `json:"recovered"`
	OpenAmount int64   `json:"open_amount"`
	Status     string  `json:"status"`
	Settlement *string `json:"settlement"`
	OrderID    *string `json:"order_id"`
	OrderNo    *int64  `json:"order_number"`
	// WaivedBefore **«انسقط عنه قبل X مرّات»** — من أعفى طرفاً مرّةً يُسأل عن الثانية.
	WaivedBefore int             `json:"waived_before"`
	Pending      *disputePending `json:"pending"`
	CreatedAt    time.Time       `json:"created_at"`
}

// disputeSelect **اسمُ الطرف من مصدره** — المتجرُ من `merchants` والباقي من
// `users`. ولا عمودَ اسمٍ محفوظٌ في النزاع: **اسمٌ يُنسخ يوم النزاع يبقى قديماً
// بعد أن يُصحَّح صاحبُه.**
const disputeSelect = `
	SELECT d.id::text, d.party_role,
	       COALESCE(d.merchant_id::text, d.party_user_id::text),
	       COALESCE(mm.name, uu.full_name, ''),
	       COALESCE(mm.phone::text, uu.phone::text, ''),
	       d.reason, d.note, d.amount, d.recovered, d.status, d.settlement,
	       d.order_id::text, o.number,
	       (SELECT count(*) FROM disputes x
	         WHERE x.id <> d.id AND x.status = 'waived' AND x.party_role = d.party_role
	           AND (x.merchant_id = d.merchant_id OR x.party_user_id = d.party_user_id))::int,
	       pr.id::text, pr.action, pr.note, pr.proposed_by::text,
	       COALESCE(NULLIF(pu.full_name, ''), pu.phone::text, ''), pr.created_at,
	       d.created_at
	FROM disputes d
	LEFT JOIN merchants mm ON mm.id = d.merchant_id
	LEFT JOIN users uu     ON uu.id = d.party_user_id
	LEFT JOIN orders o     ON o.id = d.order_id
	LEFT JOIN dispute_resolutions pr ON pr.dispute_id = d.id AND pr.status = 'pending'
	LEFT JOIN users pu     ON pu.id = pr.proposed_by`

func scanDispute(row pgx.Row) (disputeRow, error) {
	var x disputeRow
	var pID, pAction, pNote, pBy, pName *string
	var pAt *time.Time
	if err := row.Scan(&x.ID, &x.PartyRole, &x.PartyID, &x.PartyName, &x.PartyPhone,
		&x.Reason, &x.Note, &x.Amount, &x.Recovered, &x.Status, &x.Settlement,
		&x.OrderID, &x.OrderNo, &x.WaivedBefore,
		&pID, &pAction, &pNote, &pBy, &pName, &pAt, &x.CreatedAt); err != nil {
		return x, err
	}
	if x.Status == "open" {
		x.OpenAmount = x.Amount - x.Recovered
	}
	if pID != nil {
		x.Pending = &disputePending{ID: *pID, Action: derefStr(pAction), Note: derefStr(pNote),
			ProposedBy: derefStr(pBy), ProposerName: derefStr(pName)}
		if pAt != nil {
			x.Pending.CreatedAt = *pAt
		}
	}
	return x, nil
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// handleListDisputes النزاعاتُ مجموعةً — **بالطرف وبالحالة.**
//
// **والافتراضُ «المفتوحة»**: القسمُ يُفتح لما لم يُحسم، **وقائمةٌ تعرض المحسومَ
// مع المفتوح تُقرأ عملاً باقياً وهو منتهٍ** — وهي علّةُ «طلبات الانضمام» نفسُها.
func (s *Server) handleListDisputes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	party := q.Get("party")
	if party != "" && !disputeParties[party] {
		s.respondErr(w, errValidation)
		return
	}
	status := q.Get("status")
	if status == "" {
		status = "open"
	}
	if status == "all" {
		status = ""
	}
	if status != "" && status != "open" && status != "settled" && status != "waived" {
		s.respondErr(w, errValidation)
		return
	}

	// **وصفحةٌ محدودةٌ بعدٍّ — لا مئتان بلا كلمة** (قرارُ المالك ٢٠٢٦-٠٨-١٠).
	pg := pagingOf(r, 20)

	// **والعدُّ باستعلامٍ ثانٍ لا بطول الصفحة** — طولُ الصفحة يقول كم عُرض،
	// **والعدُّ يقول كم هناك.**
	var count int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM disputes d
		WHERE ($1 = '' OR d.party_role = $1)
		  AND ($2 = '' OR d.status = $2)`, party, status).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), disputeSelect+`
		WHERE ($1 = '' OR d.party_role = $1)
		  AND ($2 = '' OR d.status = $2)
		ORDER BY d.created_at DESC LIMIT $3 OFFSET $4`,
		party, status, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []disputeRow{}
	for rows.Next() {
		x, err := scanDispute(rows)
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

	// ══════════════════════════════════════════════════════════════════
	// **والكرتان بالترشيح نفسِه** (المشكلة ١٢)
	// ══════════════════════════════════════════════════════════════════
	//
	// كان «نزاعاتٌ مفتوحة» عددَ الكلّ و«مجموعُ المفتوح» بالطرف المختار —
	// **فيُختار «السائقون» فيُقرأ خمسةٌ بمبلغ صفر.** والآن الاثنان بالطرف نفسِه.
	//
	// **والمجموعُ للمفتوح وحدَه وبما بقي منه** — «كم لنا عند الناس» سؤالٌ عن
	// الدَّين، **وما استُرِدّ جزئيّاً لم يعد لنا عندهم.**
	var openCount int
	var total int64
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*), COALESCE(sum(d.amount - d.recovered), 0) FROM disputes d
		WHERE ($1 = '' OR d.party_role = $1) AND d.status = 'open'`,
		party).Scan(&openCount, &total); err != nil {
		s.respondErr(w, err)
		return
	}

	// **وعدُّ المفتوح لكلّ طرفٍ يُقرأ على الترشيح** — فمن فتح القسم عرف أين
	// العمل قبل أن ينقر.
	counts := map[string]int{}
	crows, err := s.pg.Query(r.Context(),
		`SELECT party_role, count(*) FROM disputes WHERE status = 'open' GROUP BY party_role`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer crows.Close()
	for crows.Next() {
		var k string
		var n int
		if err := crows.Scan(&k, &n); err != nil {
			s.respondErr(w, err)
			return
		}
		counts[k] = n
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"disputes": out, "total": total, "open_count": openCount, "open_counts": counts,
		// **و`count` عددُ الصفوف و`total` مجموعُ المال** — اسمان لا يجتمعان.
		"count": count, "page": pg.Page, "per_page": pg.PerPage,
	})
}

// handleDisputeParties **من يُختار طرفاً لنزاعٍ يدويّ — بدوره.**
//
// (المشكلتان ١ و١٥.) كانت الشاشةُ تختار **حساباً** أيّاً كان ثمّ تُخمّن دورَه:
// صاحبُ متجرٍ يُرسَل معرّفُه في خانة المتجر فتردّه القاعدة، **وسائقٌ يُسجَّل
// زبوناً بلا اعتراض.** والآن القائمةُ تُقرأ بالدور: المتاجرُ بمعرّف المتجر،
// **والأشخاصُ ممّن يحملون الدورَ المختار وحدَهم.**
func (s *Server) handleDisputeParties(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if !disputeParties[role] {
		s.respondErr(w, errValidation)
		return
	}
	needle := clip(r.URL.Query().Get("q"), 60)
	like := "%" + needle + "%"
	var rows pgx.Rows
	var err error
	if role == "merchant" {
		rows, err = s.pg.Query(r.Context(), `
			SELECT m.id::text, m.name, COALESCE(m.phone::text, '')
			FROM merchants m
			WHERE ($1 = '' OR m.name ILIKE $2 OR m.phone::text LIKE $2)
			ORDER BY m.name LIMIT 15`, needle, like)
	} else {
		rows, err = s.pg.Query(r.Context(), `
			SELECT u.id::text, COALESCE(NULLIF(u.full_name, ''), u.phone::text), u.phone::text
			FROM users u
			WHERE EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code = $1)
			  AND ($2 = '' OR u.full_name ILIKE $3 OR u.phone::text LIKE $3)
			ORDER BY u.full_name LIMIT 15`, role, needle, like)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type party struct {
		ID    string `json:"id"`
		Name  string `json:"name"`
		Phone string `json:"party_phone"`
	}
	out := []party{}
	for rows.Next() {
		var p party
		if err := rows.Scan(&p.ID, &p.Name, &p.Phone); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"parties": out})
}

// handleCreateDispute يفتح نزاعاً بيد إنسان — **لما لا يُولد من طلب.**
//
// صندوقُ سائقٍ لم يُسلَّم، وعمولةُ مندوبٍ على متجرٍ لم يعمل، **ولا واقعةَ في
// المحرّك تُنتجهما.** فيُفتح بيدٍ ويُسجَّل من فتحه.
//
// **والطرفُ يُتحقَّق من دوره** (المشكلتان ١ و١٥): المتجرُ بمعرّف متجرٍ قائم،
// والشخصُ بحسابٍ يحمل الدورَ المختار — **وإلّا ردٌّ بكلمةٍ لا خطأُ قاعدة.**
func (s *Server) handleCreateDispute(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		PartyRole string `json:"party_role"`
		PartyID   string `json:"party_id"`
		Reason    string `json:"reason"`
		Amount    int64  `json:"amount"`
		Note      string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	reason := strings.TrimSpace(req.Reason)
	if !disputeParties[req.PartyRole] || !isUUID(req.PartyID) || req.Amount <= 0 || reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)

	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		var ok bool
		var check string
		if req.PartyRole == "merchant" {
			check = `SELECT EXISTS (SELECT 1 FROM merchants WHERE id = $1::uuid)`
		} else {
			check = `SELECT EXISTS (SELECT 1 FROM user_roles
			                         WHERE user_id = $1::uuid AND role_code = $2)`
		}
		args := []any{req.PartyID}
		if req.PartyRole != "merchant" {
			args = append(args, req.PartyRole)
		}
		if err := q.QueryRow(ctx, check, args...).Scan(&ok); err != nil {
			return IdempotentBody{}, err
		}
		if !ok {
			return IdempotentBody{}, errDisputePartyMismatch
		}

		// **والطرفُ يُكتب في عموده** — المتجرُ كيانٌ والباقي أشخاص.
		col := "party_user_id"
		if req.PartyRole == "merchant" {
			col = "merchant_id"
		}
		var id string
		if err := q.QueryRow(ctx, `
			INSERT INTO disputes (party_role, `+col+`, reason, amount, note, created_by)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
			req.PartyRole, req.PartyID, clip(reason, 300), req.Amount,
			clip(req.Note, 500), actor).Scan(&id); err != nil {
			return IdempotentBody{}, err
		}
		if err := s.auditTx(ctx, q, r, "ops.dispute_opened", "dispute", id, map[string]any{
			"party": req.PartyRole, "party_id": req.PartyID, "amount": req.Amount, "reason": reason,
		}); err != nil {
			return IdempotentBody{}, err
		}
		return IdempotentBody{
			Status:      http.StatusCreated,
			Payload:     map[string]any{"id": id},
			AfterCommit: func() { s.touch("dispute", "ops") },
		}, nil
	})
}

// handleProposeDisputeResolution **يقترح الحسم** — خصماً أو إسقاطاً، ولا يحرّك مالاً.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٢: «الخصمُ والإسقاطُ كلاهما اقتراحٌ وموافقة».)
// **والسببُ إلزاميّ في الحالين** — إسقاطٌ بلا كلمةٍ لا يُراجَع، **وخصمٌ بلا
// كلمةٍ يجده صاحبُه في محفظته ولا يعرف عمّاذا.**
func (s *Server) handleProposeDisputeResolution(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Action string `json:"action"` // charge | waive
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Action != "charge" && req.Action != "waive" {
		s.respondErr(w, errValidation)
		return
	}
	note := clip(req.Note, 500)
	if note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	actor := userIDFrom(r)
	var resID string
	if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		var status string
		var open int64
		var pending bool
		// **القفلُ على النزاع** — اقتراحان متزامنان لا يمرّان معاً.
		err := q.QueryRow(ctx, `
			SELECT d.status, d.amount - d.recovered,
			       EXISTS (SELECT 1 FROM dispute_resolutions x
			                WHERE x.dispute_id = d.id AND x.status = 'pending')
			FROM disputes d WHERE d.id = $1 FOR UPDATE`, id).Scan(&status, &open, &pending)
		if errors.Is(err, pgx.ErrNoRows) {
			return httpx.ErrNotFound
		}
		if err != nil {
			return err
		}
		if status != "open" {
			return errClaimSettled
		}
		if pending {
			return errDisputeResolutionPending
		}
		if err := q.QueryRow(ctx, `
			INSERT INTO dispute_resolutions (dispute_id, action, amount, note, proposed_by)
			VALUES ($1, $2, $3, $4, $5) RETURNING id::text`,
			id, req.Action, open, note, actor).Scan(&resID); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "finance.dispute_proposed", "dispute", id, map[string]any{
			"resolution_id": resID, "action": req.Action, "amount": open, "note": note,
		})
	}); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("dispute", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"resolution_id": resID, "status": "pending"})
}

type disputeResolutionRow struct {
	ID           string     `json:"id"`
	DisputeID    string     `json:"dispute_id"`
	Action       string     `json:"action"`
	Status       string     `json:"status"`
	Amount       int64      `json:"amount"`
	Charged      *int64     `json:"charged"`
	Note         string     `json:"note"`
	ProposedBy   string     `json:"proposed_by"`
	ProposerName string     `json:"proposer_name"`
	DecidedBy    *string    `json:"decided_by"`
	DecidedAt    *time.Time `json:"decided_at"`
	DecisionNote string     `json:"decision_note"`
	SelfApproved bool       `json:"self_approved"`
	PartyRole    string     `json:"party_role"`
	PartyName    string     `json:"party_name"`
	OrderNo      *int64     `json:"order_number"`
	CreatedAt    time.Time  `json:"created_at"`
}

const disputeResolutionCols = `
	SELECT r.id::text, r.dispute_id::text, r.action, r.status, r.amount, r.charged, r.note,
	       r.proposed_by::text, COALESCE(NULLIF(p.full_name, ''), p.phone::text, ''),
	       r.decided_by::text, r.decided_at, r.decision_note, r.self_approved,
	       d.party_role, COALESCE(mm.name, uu.full_name, ''), o.number, r.created_at
	FROM dispute_resolutions r
	JOIN disputes d       ON d.id = r.dispute_id
	JOIN users p          ON p.id = r.proposed_by
	LEFT JOIN merchants mm ON mm.id = d.merchant_id
	LEFT JOIN users uu     ON uu.id = d.party_user_id
	LEFT JOIN orders o     ON o.id = d.order_id`

func scanDisputeResolution(row pgx.Row) (disputeResolutionRow, error) {
	var x disputeResolutionRow
	err := row.Scan(&x.ID, &x.DisputeID, &x.Action, &x.Status, &x.Amount, &x.Charged, &x.Note,
		&x.ProposedBy, &x.ProposerName, &x.DecidedBy, &x.DecidedAt, &x.DecisionNote,
		&x.SelfApproved, &x.PartyRole, &x.PartyName, &x.OrderNo, &x.CreatedAt)
	return x, err
}

// handleListDisputeResolutions **اقتراحاتُ الحسم** — المعلَّقةُ افتراضاً.
//
// تقرؤها صفحةُ الموافقات الموحَّدة كما تقرأ طلباتِ المحفظة.
func (s *Server) handleListDisputeResolutions(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status == "all" {
		status = ""
	}
	rows, err := s.pg.Query(r.Context(), disputeResolutionCols+`
		WHERE ($1 = '' OR r.status = $1)
		ORDER BY r.created_at DESC LIMIT 200`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []disputeResolutionRow{}
	for rows.Next() {
		x, err := scanDisputeResolution(rows)
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
	httpx.JSON(w, http.StatusOK, map[string]any{"resolutions": out})
}

// handleDecideDisputeResolution **يوافق على اقتراح الحسم أو يرفضه.**
//
// **والموافقةُ لغير المقترِح** (`approval.Check`) — والمالكُ الأعلى وحدَه يوافق
// على نفسه إن لم يكن غيرُه، وتُعلَّم في الطلب والسجلّ.
//
// # والخصمُ بما في المحفظة (القرار ٣)
//
// **يُخصم الأقلُّ من المفتوح ومن الرصيد**، ويعود إلى الخزينة في المعاملة نفسِها.
// فإن غطّى المفتوحَ كلَّه حُسم النزاع، **وإلّا بقي مفتوحاً بما بقي** — يُقترح
// خصمُه ثانيةً حين يصل الطرفَ مال. **ورصيدٌ صفرٌ لا يُقرأ موافقةً**: يُردّ
// بكلمة، والاقتراحُ يبقى معلَّقاً حتّى يُرفض أو يصل مال.
func (s *Server) handleDecideDisputeResolution(approve bool) http.HandlerFunc {
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
		decisionNote := clip(req.Note, 500)
		actor := userIDFrom(r)
		var out disputeResolutionRow
		var remaining int64
		if err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
			res, err := scanDisputeResolution(q.QueryRow(ctx, disputeResolutionCols+`
				WHERE r.id = $1 FOR UPDATE OF r`, id))
			if errors.Is(err, pgx.ErrNoRows) {
				return httpx.ErrNotFound
			}
			if err != nil {
				return err
			}
			if res.Status != "pending" {
				return errRequestDecided
			}
			if !approve {
				if _, err := q.Exec(ctx, `
					UPDATE dispute_resolutions
					   SET status = 'rejected', decided_by = $2, decided_at = now(), decision_note = $3
					 WHERE id = $1`, id, actor, decisionNote); err != nil {
					return err
				}
				res.Status = "rejected"
				out = res
				return s.auditTx(ctx, q, r, "finance.dispute_resolution_rejected", "dispute",
					res.DisputeID, map[string]any{"resolution_id": id, "action": res.Action,
						"proposed_by": res.ProposedBy, "note": decisionNote})
			}

			verdict, err := approval.Check(ctx, q, approval.Request{
				ProposedBy: res.ProposedBy, Actor: actor, Capability: authz.FinanceManage,
			})
			if err != nil {
				return err
			}

			// **القفلُ على النزاع داخل المعاملة** — موافقتان متزامنتان تخصمان مرّتين.
			// **ومحفظةُ الطرف من مصدره**: المتجرُ يُخصم من صاحبه، والباقي من نفسه.
			var amount, recovered int64
			var status string
			var walletUser *string
			if err := q.QueryRow(ctx, `
				SELECT d.amount, d.recovered, d.status,
				       COALESCE(mm.owner_user_id::text, d.party_user_id::text)
				FROM disputes d
				LEFT JOIN merchants mm ON mm.id = d.merchant_id
				WHERE d.id = $1 FOR UPDATE OF d`, res.DisputeID).
				Scan(&amount, &recovered, &status, &walletUser); err != nil {
				return err
			}
			if status != "open" {
				return errClaimSettled
			}
			open := amount - recovered
			// **والملاحظةُ تُقرأ مع القرار** — اقتراحُ المقترِح وكلمةُ الموافق.
			note := res.Note
			var charged int64
			newStatus := "open"
			var settlement any
			if res.Action == "waive" {
				newStatus, settlement = "waived", "waived"
			} else {
				// **ومتجرٌ بلا صاحبٍ لا يُخصم منه** — ولا يُقال «سُوّي» عمّا لم يقع.
				if walletUser == nil {
					return errDisputeNoWallet
				}
				var balance int64
				if err := q.QueryRow(ctx, `
					SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1 FOR UPDATE), 0)`,
					*walletUser).Scan(&balance); err != nil {
					return err
				}
				charged = min(open, balance)
				if charged <= 0 {
					return errDisputeNothingToCharge
				}
				if _, err := s.wallet.ApplyTx(ctx, q, *walletUser, -charged,
					"adjustment", res.DisputeID, "مطالبةُ المنصة — "+note, &actor); err != nil {
					return err
				}
				// **ويعود إلى الخزينة ما خرج منها** — خصمٌ بلا قيدٍ مقابلٍ يجعل
				// المنصةَ تبدو خاسرةً وقد استُرِدّ لها.
				if err := s.orders.CreditTreasuryDirect(ctx, q, charged, res.DisputeID,
					"استردادُ مطالبةٍ", actor); err != nil {
					return err
				}
				if charged == open {
					newStatus, settlement = "settled", "charged"
				}
			}
			// **والباقي ما لم يُخصم ولم يُسقط** — يبقى ديناً مفتوحاً على الطرف.
			remaining = open - charged
			if newStatus == "waived" {
				remaining = 0
			}
			if _, err := q.Exec(ctx, `
				UPDATE disputes
				SET recovered = recovered + $2,
				    status = $3,
				    settlement = COALESCE($4::text, settlement),
				    settled_at = CASE WHEN $3 = 'open' THEN settled_at ELSE now() END,
				    settled_by = CASE WHEN $3 = 'open' THEN settled_by ELSE $5::uuid END,
				    note = CASE WHEN note = '' THEN $6 ELSE note || ' · ' || $6 END
				WHERE id = $1`, res.DisputeID, charged, newStatus, settlement, actor, note); err != nil {
				return err
			}
			if _, err := q.Exec(ctx, `
				UPDATE dispute_resolutions
				   SET status = 'approved', charged = $2, decided_by = $3, decided_at = now(),
				       decision_note = $4, self_approved = $5
				 WHERE id = $1`, id, charged, actor, decisionNote, verdict.SelfApproved); err != nil {
				return err
			}
			res.Status = "approved"
			res.Charged = &charged
			res.SelfApproved = verdict.SelfApproved
			out = res
			// **والأثرُ داخلَ المعاملة** (`AQ-4`/`PF-06`): مالٌ تحرّك أو أُعفي،
			// **فسطرُ تدقيقٍ بعد التثبيت قد يسقط ويترك القرارَ بلا أثر.**
			details := map[string]any{
				"resolution_id": id, "party": res.PartyRole, "action": res.Action,
				"amount": open, "charged": charged, "remaining": remaining,
				"proposed_by": res.ProposedBy, "note": note, "decision_note": decisionNote,
			}
			for k, v := range verdict.AuditFields() {
				details[k] = v
			}
			return s.auditTx(ctx, q, r, "finance.dispute_resolution_approved", "dispute",
				res.DisputeID, details)
		}); err != nil {
			s.respondErr(w, err)
			return
		}
		s.touch("dispute", "ops")
		if approve && out.Action == "charge" {
			s.touch("wallet", "ops")
		}
		httpx.JSON(w, http.StatusOK, map[string]any{
			"status": out.Status, "action": out.Action, "charged": out.Charged,
			"remaining": remaining, "self_approved": out.SelfApproved,
		})
	}
}
