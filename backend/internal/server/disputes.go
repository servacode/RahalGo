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
//   - **`charged`** — يُخصم من محفظة الطرف ويعود إلى الخزينة. **مالٌ تحرّك.**
//   - **`waived`**  — تتحمّله المنصة. **قرارٌ لا حركة**، ولا قيدَ له.
//
// **وخلطُهما يجعل تقريرَ الخسائر يقرأ إسقاطاً تحصيلاً.**

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errClaimSettled = httpx.NewError(http.StatusConflict,
		"claim_already_settled", "errors.claim_already_settled")
	errDisputeNoWallet = httpx.NewError(http.StatusConflict,
		"dispute_party_has_no_wallet", "errors.dispute_party_has_no_wallet")
)

// disputeParties الأدوارُ التي يجوز أن نتنازع معها — **مرآةُ قيد القاعدة.**
var disputeParties = map[string]bool{
	"merchant": true, "driver": true, "sales": true, "customer": true,
}

type disputeRow struct {
	ID         string    `json:"id"`
	PartyRole  string    `json:"party_role"`
	PartyID    string    `json:"party_id"`
	PartyName  string    `json:"party_name"`
	PartyPhone string    `json:"party_phone"`
	Reason     string    `json:"reason"`
	Note       string    `json:"note"`
	Amount     int64     `json:"amount"`
	Status     string    `json:"status"`
	Settlement *string   `json:"settlement"`
	OrderNo    *int64    `json:"order_number"`
	CreatedAt  time.Time `json:"created_at"`
}

// disputeSelect **اسمُ الطرف من مصدره** — المتجرُ من `merchants` والباقي من
// `users`. ولا عمودَ اسمٍ محفوظٌ في النزاع: **اسمٌ يُنسخ يوم النزاع يبقى قديماً
// بعد أن يُصحَّح صاحبُه.**
const disputeSelect = `
	SELECT d.id::text, d.party_role,
	       COALESCE(d.merchant_id::text, d.party_user_id::text),
	       COALESCE(mm.name, uu.full_name, ''),
	       COALESCE(mm.phone::text, uu.phone::text, ''),
	       d.reason, d.note, d.amount, d.status, d.settlement, o.number, d.created_at
	FROM disputes d
	LEFT JOIN merchants mm ON mm.id = d.merchant_id
	LEFT JOIN users uu     ON uu.id = d.party_user_id
	LEFT JOIN orders o     ON o.id = d.order_id`

// handleListDisputes النزاعاتُ مجموعةً — **بتبويب الطرف وبحالته.**
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

	rows, err := s.pg.Query(r.Context(), disputeSelect+`
		WHERE ($1 = '' OR d.party_role = $1)
		  AND ($2 = '' OR d.status = $2)
		ORDER BY d.created_at DESC LIMIT 200`, party, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []disputeRow{}
	// **والمجموعُ للمفتوح وحدَه** — «كم لنا عند الناس» سؤالٌ عن الدَّين لا عن
	// التاريخ، **وجمعُ المحسومِ معه يضخّمه بما استُرِدّ فعلاً.**
	var total int64
	for rows.Next() {
		var x disputeRow
		if err := rows.Scan(&x.ID, &x.PartyRole, &x.PartyID, &x.PartyName, &x.PartyPhone,
			&x.Reason, &x.Note, &x.Amount, &x.Status, &x.Settlement,
			&x.OrderNo, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		if x.Status == "open" {
			total += x.Amount
		}
		out = append(out, x)
	}

	// **وعدُّ المفتوح لكلّ طرفٍ يُقرأ على التبويبات نفسِها** — فمن فتح القسم
	// عرف أين العمل قبل أن ينقر.
	counts := map[string]int{}
	crows, err := s.pg.Query(r.Context(),
		`SELECT party_role, count(*) FROM disputes WHERE status = 'open' GROUP BY party_role`)
	if err == nil {
		defer crows.Close()
		for crows.Next() {
			var k string
			var n int
			if crows.Scan(&k, &n) == nil {
				counts[k] = n
			}
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"disputes": out, "total": total, "open_counts": counts,
	})
}

// handleCreateDispute يفتح نزاعاً بيد إنسان — **لما لا يُولد من طلب.**
//
// صندوقُ سائقٍ لم يُسلَّم، وعمولةُ مندوبٍ على متجرٍ لم يعمل، **ولا واقعةَ في
// المحرّك تُنتجهما.** فيُفتح بيدٍ ويُسجَّل من فتحه.
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
	if !disputeParties[req.PartyRole] || req.PartyID == "" || req.Amount <= 0 || reason == "" {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)

	var id string
	// **والطرفُ يُكتب في عموده** — المتجرُ كيانٌ والباقي أشخاص.
	col := "party_user_id"
	if req.PartyRole == "merchant" {
		col = "merchant_id"
	}
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO disputes (party_role, `+col+`, reason, amount, note, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id::text`,
		req.PartyRole, req.PartyID, reason, req.Amount,
		strings.TrimSpace(req.Note), actor).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "ops.dispute_opened", "dispute", id, map[string]any{
		"party": req.PartyRole, "party_id": req.PartyID, "amount": req.Amount, "reason": reason,
	})
	s.touch("dispute", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// handleSettleDispute يحسم نزاعاً: خصماً أو إسقاطاً.
func (s *Server) handleSettleDispute(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Settlement string `json:"settlement"` // charged | waived
		Note       string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Settlement != "charged" && req.Settlement != "waived" {
		s.respondErr(w, errValidation)
		return
	}
	// **والسببُ إلزاميّ في الحالين.** إسقاطٌ بلا كلمةٍ لا يُراجَع، **وخصمٌ بلا
	// كلمةٍ يجده صاحبُه في محفظته ولا يعرف عمّاذا.**
	note := strings.TrimSpace(req.Note)
	if note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	id := chi.URLParam(r, "id")
	actor := userIDFrom(r)

	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	var amount int64
	var role, status string
	var walletUser *string
	// **القفلُ داخل المعاملة**: ضغطتان متزامنتان تخصمان مرّتين.
	//
	// **ومحفظةُ الطرف من مصدره**: المتجرُ يُخصم من صاحبه، والباقي من نفسه.
	if err := tx.QueryRow(r.Context(), `
		SELECT d.amount, d.party_role, d.status,
		       COALESCE(mm.owner_user_id::text, d.party_user_id::text)
		FROM disputes d
		LEFT JOIN merchants mm ON mm.id = d.merchant_id
		WHERE d.id = $1 FOR UPDATE OF d`, id).
		Scan(&amount, &role, &status, &walletUser); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if status != "open" {
		s.respondErr(w, errClaimSettled)
		return
	}

	if req.Settlement == "charged" {
		// **ومتجرٌ بلا صاحبٍ لا يُخصم منه** — ولا يُقال «سُوّي» عمّا لم يقع.
		if walletUser == nil {
			s.respondErr(w, errDisputeNoWallet)
			return
		}
		if _, err := s.wallet.ApplyTx(r.Context(), tx, *walletUser, -amount,
			"adjustment", id, "مطالبةُ المنصة — "+note, &actor); err != nil {
			s.respondErr(w, err)
			return
		}
		// **ويعود إلى الخزينة ما خرج منها.**
		//
		// خصمٌ بلا قيدٍ مقابلٍ في الخزينة **يجعل المنصةَ تبدو خاسرةً وقد
		// استُرِدّ لها** — والربحُ الذي لا يعرف ما عاد إليه ليس ربحاً.
		if err := s.orders.CreditTreasuryDirect(r.Context(), tx, amount, id,
			"استردادُ مطالبةٍ", actor); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	newStatus := "settled"
	if req.Settlement == "waived" {
		newStatus = "waived"
	}
	if _, err := tx.Exec(r.Context(), `
		UPDATE disputes
		SET status = $2, settlement = $3, settled_at = now(), settled_by = $4,
		    note = CASE WHEN note = '' THEN $5 ELSE note || ' · ' || $5 END
		WHERE id = $1`, id, newStatus, req.Settlement, actor, note); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "ops.dispute_settled", "dispute", id, map[string]any{
		"party": role, "settlement": req.Settlement, "amount": amount, "note": note,
	})
	s.touch("dispute", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"settled": req.Settlement})
}
