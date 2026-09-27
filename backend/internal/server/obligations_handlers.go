package server

// عرضُ الالتزامات الماليّة — قراءةٌ فقط.
//
// **الجدولُ `financial_obligations` هو الحقيقة** (هجرة 0130): لكلّ التزامٍ
// طرفٌ (متجرٌ أو مندوب) ومبلغٌ ومسدَّدٌ وسببٌ وطلبٌ ووقت. **والباقي = المبلغُ
// ناقصَ المسدَّد**، والحالُ مفتوحٌ ما دام `closed_at` فارغاً.
//
// **ولا فعلَ هنا** — لا عفوَ ولا تسويةَ ولا تعديل. **يُقرأ ولا يُكتب**، فمن
// أراد أن يسوّي فعلها من بابها (استرداد أو نزاع). وهذا الباب يُري الدَّينَ
// كلَّه: على من، وكم، ومن أين، وكم بقي — مع سطور تسويته إن وُجدت.

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// obligationSettlement سطرُ تسويةٍ واحد — **كم اقتُطع، وكم بقي، ومتى.**
type obligationSettlement struct {
	Amount    int64     `json:"amount"`
	Remaining int64     `json:"remaining"`
	OrderNo   *int64    `json:"order_number"`
	CreatedAt time.Time `json:"created_at"`
}

// obligationRow التزامٌ واحدٌ كما يُعرَض — **بلا هاتفٍ ولا سرٍّ**، اسمُ الطرف
// ومعرّفُه يكفيان.
type obligationRow struct {
	ID          string                 `json:"id"`
	PartyKind   string                 `json:"party_kind"`
	PartyID     string                 `json:"party_id"`
	PartyName   string                 `json:"party_name"`
	Amount      int64                  `json:"amount"`
	Outstanding int64                  `json:"outstanding"`
	Cause       string                 `json:"cause"`
	OrderNo     *int64                 `json:"order_number"`
	CreatedAt   time.Time              `json:"created_at"`
	State       string                 `json:"state"`
	Settlements []obligationSettlement `json:"settlements"`
}

// handleListObligations الالتزاماتُ الماليّة — ترشيحٌ بالطرف والحالة، وصفحةٌ محدودة.
//
// **واسمُ الطرف من مصدره**: المتجرُ من `merchants` والمندوبُ من `users` —
// **ولا عمودَ اسمٍ محفوظٌ هنا**، فاسمٌ يُنسخ يوم النشأة يشيخ بعد أن يُصحَّح صاحبُه.
func (s *Server) handleListObligations(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// **الطرفُ إمّا متجرٌ أو مندوبٌ أو الكلّ** — ومجهولُه رفضٌ لا تجاهل.
	partyKind := q.Get("party_kind")
	if partyKind != "" && partyKind != obligations.PartyMerchant && partyKind != obligations.PartyRep {
		s.respondErr(w, errValidation)
		return
	}
	// **والحالُ مفتوحٌ أو مغلقٌ أو الكلّ.**
	state := q.Get("state")
	if state != "" && state != "open" && state != "closed" {
		s.respondErr(w, errValidation)
		return
	}
	// **ومعرّفُ الطرف يُطابَق نصّاً** — فلا يُكسَر النداءُ بقيمةٍ ليست معرّفاً.
	partyID := q.Get("party_id")

	pg := pagingOf(r, 25)

	// **والشرطُ الواحدُ يُكتب مرّةً** — للعدّ وللقائمة وللمجموع، فلا يفترق ثلاثتُها.
	const where = `
		WHERE ($1 = '' OR o.party_kind = $1)
		  AND ($2 = '' OR o.party_id::text = $2)
		  AND ($3 = '' OR ($3 = 'open'  AND o.closed_at IS NULL)
		               OR ($3 = 'closed' AND o.closed_at IS NOT NULL))`

	var count int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM financial_obligations o`+where,
		partyKind, partyID, state).Scan(&count); err != nil {
		s.respondErr(w, err)
		return
	}

	// **ومجموعُ الباقي على المفتوح وحدَه** — «كم لنا عند الناس الآن»، ولا يتبع
	// ترشيحَ الحالة: مجموعٌ يتبع «مغلق» يقول صفراً. **ويُحسب على كلّ المفتوح لا
	// على الصفحة المعروضة**، فلا ينقص كلّما قُلّب الترقيم.
	var outstandingTotal int64
	if err := s.pg.QueryRow(r.Context(), `
		SELECT COALESCE(sum(o.amount - o.settled), 0)
		FROM financial_obligations o
		WHERE ($1 = '' OR o.party_kind = $1)
		  AND ($2 = '' OR o.party_id::text = $2)
		  AND o.closed_at IS NULL`,
		partyKind, partyID).Scan(&outstandingTotal); err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT o.id::text, o.party_kind, o.party_id::text,
		       COALESCE(mm.name, uu.full_name, ''),
		       o.amount, o.amount - o.settled, o.cause,
		       ord.number, o.created_at,
		       CASE WHEN o.closed_at IS NULL THEN 'open' ELSE 'closed' END
		FROM financial_obligations o
		LEFT JOIN merchants mm ON o.party_kind = 'merchant' AND mm.id = o.party_id
		LEFT JOIN users     uu ON o.party_kind = 'rep'      AND uu.id = o.party_id
		LEFT JOIN orders   ord ON ord.id = o.order_id`+where+`
		ORDER BY o.created_at DESC, o.id
		LIMIT $4 OFFSET $5`,
		partyKind, partyID, state, pg.PerPage, pg.Offset)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []obligationRow{}
	byID := map[string]int{}
	for rows.Next() {
		var x obligationRow
		x.Settlements = []obligationSettlement{}
		if err := rows.Scan(&x.ID, &x.PartyKind, &x.PartyID, &x.PartyName,
			&x.Amount, &x.Outstanding, &x.Cause, &x.OrderNo, &x.CreatedAt,
			&x.State); err != nil {
			s.respondErr(w, err)
			return
		}
		byID[x.ID] = len(out)
		out = append(out, x)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	// **وسطورُ التسوية لهذه الصفحة وحدَها** — نداءٌ واحدٌ لكلّ التزامات الصفحة،
	// **فلا نداءٌ لكلّ صفّ.** ومن لا تسويةَ له يبقى بمصفوفةٍ فارغة.
	if len(out) > 0 {
		ids := make([]string, 0, len(out))
		for _, x := range out {
			ids = append(ids, x.ID)
		}
		srows, err := s.pg.Query(r.Context(), `
			SELECT st.obligation_id::text, st.amount, st.remaining, ord.number, st.created_at
			FROM obligation_settlements st
			LEFT JOIN orders ord ON ord.id = st.order_id
			WHERE st.obligation_id::text = ANY($1)
			ORDER BY st.created_at, st.id`, ids)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		defer srows.Close()
		for srows.Next() {
			var oid string
			var st obligationSettlement
			if err := srows.Scan(&oid, &st.Amount, &st.Remaining, &st.OrderNo, &st.CreatedAt); err != nil {
				s.respondErr(w, err)
				return
			}
			if i, ok := byID[oid]; ok {
				out[i].Settlements = append(out[i].Settlements, st)
			}
		}
		if err := srows.Err(); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	res := paged("obligations", out, count, pg)
	res["outstanding_total"] = outstandingTotal
	httpx.JSON(w, http.StatusOK, res)
}
