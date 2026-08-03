package server

// **شكاواه هو — وأين وصلت.**
//
// # المسألة
//
// كان الزبونُ يفتح شكوى ثمّ **لا يجد لها أثراً في أيّ شاشة**: التذاكرُ كلُّها
// تحت `/admin` — لمكتب المنصة وحدَه. **وما رآه بعدها سطرٌ في بطاقة الطلب يقول
// رقمَها ولا يقول حالَها.**
//
// **ومن اشتكى ولم يرَ جواباً ظنّ أنّ شكواه ضاعت** — فيشتكي ثانيةً، أو يتّصل،
// **أو يسكت ويذهب.** والسكوتُ أسوأ: نخسر الزبونَ ولا نعرف لماذا.
//
// (شهده المالك ٢٠٢٦-٠٨-٠٣: «الشكاوي ليس لها مكان بحساب المستخدم، أين يرى ما
// حالة الشكوى؟ لا يوجد».)
//
// # وما يُعرض وما لا يُعرض
//
// **حالتُها، وردُّ المنصة، والتعويضُ إن وقع** — وهي ما يخصّه. **ولا يُعرض
// اسمُ المتجر** ولا شيءٌ يدلّ عليه: القاعدةُ نفسُها في `customer_privacy.go`،
// **وبابٌ خلفيٌّ في شاشةِ شكاوى يهدم ما تحرسه شاشةُ الطلبات.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMyTickets شكاوى الزبون نفسِه — الأحدثُ أوّلاً.
func (s *Server) handleMyTickets(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT t.id::text, t.number, o.number, t.subject, COALESCE(t.reason, ''),
		       t.status, COALESCE(t.compensation, 0), COALESCE(t.resolution, ''),
		       t.created_at, t.resolved_at
		FROM tickets t
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE t.customer_id = $1
		ORDER BY t.created_at DESC
		LIMIT 100`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type row struct {
		ID          string `json:"id"`
		Number      int64  `json:"number"`
		OrderNumber *int64 `json:"order_number"`
		Subject     string `json:"subject"`
		Reason      string `json:"reason"`
		Status      string `json:"status"`
		// Compensation ما عُوّض به — **يُعرض لأنّه ماله.**
		Compensation int64 `json:"compensation"`
		// Resolution كلمةُ المنصة عند الإغلاق — **وشكوى تُغلق بلا كلمة تُقرأ
		// تجاهلاً**، ولو كان القرارُ في صالحه.
		Resolution string     `json:"resolution"`
		CreatedAt  time.Time  `json:"created_at"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Number, &x.OrderNumber, &x.Subject, &x.Reason,
			&x.Status, &x.Compensation, &x.Resolution, &x.CreatedAt, &x.ResolvedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"tickets": out})
}
