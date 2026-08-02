package server

// **نزاعاتُ المتاجر** — ما دفعته المنصةُ بسببهم وتُطالِبهم به.
//
// # القرار
//
// **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»** (المالك،
// ٢٠٢٦-٠٨-٠٣)
//
// فالسائقُ يُعوَّض **لحظتَها**، والمطالبةُ تُفتح وتُحسم في مسارها. **ولا يقف
// أحدٌ ثالثٌ على نزاعٍ بين طرفين.**
//
// # ولماذا لا تُخصم آلياً
//
// **الخصمُ قرارُ إنسانٍ بعد أن يسمع المتجر.** وقد يكون العذرُ حقّاً: انقطعت
// الكهرباءُ، أو جاءه الطلبُ ولم يُبلَّغ. **ومالٌ يخرج من محفظةِ متجرٍ قبل أن
// يُسأل نزاعٌ خُسر قبل أن يُفتح** — يبقى المالُ ويذهب المتجر.
//
// **والإعفاءُ يُسجَّل لا يمرّ صمتاً**: من أعفى متجراً مرّةً يُسأل عن الثانية،
// **ومطالبةٌ تُنسى تبدو كأنّها لم تكن.**

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errClaimSettled = httpx.NewError(http.StatusConflict,
	"claim_already_settled", "errors.claim_already_settled")

type claimRow struct {
	ID           string    `json:"id"`
	MerchantID   string    `json:"merchant_id"`
	MerchantName string    `json:"merchant_name"`
	Reason       string    `json:"reason"`
	Amount       int64     `json:"claim_amount"`
	OrderNumber  *int64    `json:"order_number"`
	CreatedAt    time.Time `json:"created_at"`
}

// handleOpenClaims المطالباتُ المفتوحة — **مجموعةً في مكانٍ واحد.**
//
// **ومطالبةٌ لا تُرى مجموعةً لا تُتابَع**: من أراد أن يعرف كم لنا عند المتاجر
// فتح صفحةَ كلِّ متجرٍ على حدة — **فلا يفعل، فلا يعرف.** وهي علّةُ «أموالٌ لم
// تُستلم» نفسُها في بابٍ آخر.
func (s *Server) handleOpenClaims(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT cw.id, cw.merchant_id::text, m.name, cw.reason, cw.claim_amount,
		       o.number, cw.created_at
		FROM merchant_warnings cw
		JOIN merchants m ON m.id = cw.merchant_id
		LEFT JOIN orders o ON o.id = cw.order_id
		WHERE cw.claim_amount > 0 AND cw.settlement IS NULL
		ORDER BY cw.created_at DESC LIMIT 200`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []claimRow{}
	var total int64
	for rows.Next() {
		var x claimRow
		if err := rows.Scan(&x.ID, &x.MerchantID, &x.MerchantName, &x.Reason,
			&x.Amount, &x.OrderNumber, &x.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		total += x.Amount
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"claims": out, "total": total})
}

// handleSettleClaim يحسم مطالبةً: خصماً أو إعفاءً.
func (s *Server) handleSettleClaim(w http.ResponseWriter, r *http.Request) {
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
	// **والسببُ إلزاميّ في الحالين.** إعفاءٌ بلا كلمةٍ لا يُراجَع، **وخصمٌ بلا
	// كلمةٍ يجده المتجرُ في محفظته ولا يعرف عمّاذا.**
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
	var merchantID string
	var ownerID *string
	var settled *string
	// **القفلُ داخل المعاملة**: ضغطتان متزامنتان تخصمان مرّتين.
	if err := tx.QueryRow(r.Context(), `
		SELECT cw.claim_amount, cw.merchant_id::text, m.owner_user_id::text, cw.settlement
		FROM merchant_warnings cw JOIN merchants m ON m.id = cw.merchant_id
		WHERE cw.id = $1 FOR UPDATE OF cw`, id).
		Scan(&amount, &merchantID, &ownerID, &settled); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if settled != nil {
		s.respondErr(w, errClaimSettled)
		return
	}

	if req.Settlement == "charged" && amount > 0 && ownerID != nil {
		// **ويعود إلى الخزينة ما خرج منها.**
		//
		// خصمٌ من المتجر بلا قيدٍ مقابلٍ في الخزينة **يجعل المنصةَ تبدو خاسرةً
		// وقد استُرِدّ لها** — والربحُ الذي لا يعرف ما عاد إليه ليس ربحاً.
		if _, err := s.wallet.ApplyTx(r.Context(), tx, *ownerID, -amount,
			"adjustment", id, "مطالبةُ المنصة — "+note, &actor); err != nil {
			s.respondErr(w, err)
			return
		}
		if err := s.orders.CreditTreasuryDirect(r.Context(), tx, amount, id,
			"استردادُ مطالبةٍ من متجر", actor); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	if _, err := tx.Exec(r.Context(), `
		UPDATE merchant_warnings
		SET settlement = $2, settled_at = now(), settled_by = $3,
		    note = CASE WHEN note = '' THEN $4 ELSE note || ' · ' || $4 END
		WHERE id = $1`, id, req.Settlement, actor, note); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "ops.claim_settled", "merchant", merchantID, map[string]any{
		"claim": id, "settlement": req.Settlement, "amount": amount, "note": note,
	})
	s.touch("merchant", "ops")
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"settled": req.Settlement})
}
