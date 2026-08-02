package server

// **أموالٌ لم تُستلم** — ما في أيدي السائقين ولم يبلغ المكتبَ بعد.
//
// # المسألة
//
// النقدُ الذي يقبضه السائقُ **مالُ المنصة يحمله**، لا مالُه. وكان لا يُرى
// مجموعاً في مكان: **من أراد أن يعرف كم في الشارع فتح كشفَ كلِّ سائقٍ على
// حدة** — فلا يفعل، فلا يعرف.
//
// **ومالٌ لا يُرى مجموعاً لا يُطالَب به**: سائقٌ يحمل مئتي ألفٍ منذ ثلاثة
// أيام لا يلفت أحداً، **وسائقان يفعلان ذلك يجعلان نصفَ يومٍ من دخل المنصة
// خارجها** — ولا أحد يعلم.
//
// # ولماذا مدّةُ الحمل بجانب المبلغ
//
// **المبلغُ وحدَه لا يقول شيئاً**: خمسون ألفاً قُبضت قبل ساعة عملٌ يجري،
// وخمسون ألفاً منذ أسبوعٍ مسألةٌ أخرى. **والقِدَمُ هو الإشارة لا المقدار.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

type cashHolder struct {
	DriverID string `json:"driver_id"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Held     int64  `json:"held"`
	// OldestAt أقدمُ قبضٍ لم يُسوَّ بعده — **القِدَمُ هو الإشارة.**
	OldestAt *time.Time `json:"oldest_at"`
	OnShift  bool       `json:"on_shift"`
}

// handleCashOutstanding من يحمل نقداً ولم يورّده — مرتّبين بالأكبر.
func (s *Server) handleCashOutstanding(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id, COALESCE(u.full_name,''), u.phone,
		       COALESCE(sum(e.amount), 0) AS held,
		       -- **أقدمُ قبضٍ بعد آخر تسوية** — ما قبلها سُلّم فلا يُعدّ.
		       min(e.created_at) FILTER (
		           WHERE e.amount > 0
		             AND e.created_at > COALESCE((SELECT max(s.created_at)
		                 FROM driver_cash_entries s
		                 WHERE s.driver_id = u.id AND s.kind = 'settlement'), '-infinity')),
		       COALESCE(u.on_shift, false)
		FROM users u
		JOIN driver_cash_entries e ON e.driver_id = u.id
		GROUP BY u.id, u.full_name, u.phone, u.on_shift
		HAVING COALESCE(sum(e.amount), 0) > 0
		ORDER BY held DESC`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []cashHolder{}
	var total int64
	for rows.Next() {
		var x cashHolder
		if err := rows.Scan(&x.DriverID, &x.Name, &x.Phone, &x.Held,
			&x.OldestAt, &x.OnShift); err != nil {
			s.respondErr(w, err)
			return
		}
		total += x.Held
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"holders": out, "total": total, "limit": s.cashbox.Limit(r.Context()),
	})
}
