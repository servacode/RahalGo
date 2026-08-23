package server

// **بلاغاتُه هو — وأين وصلت.**
//
// # المسألة
//
// فُتح للمتجر بابُ البلاغ على السائق (٢٠٢٦-٠٨-١٦) **ولم يُفتح له بابٌ يرى
// فيه ما صار بها.** يُبلّغ فيمضي البلاغُ إلى مكتب المنصة، **ولا شاشةَ تقول
// له: قُرئ · فُصل فيه · هذا جوابُنا.**
//
// **ومن أبلغ ولم يرَ جواباً حسبَ البلاغَ ضائعاً** — فيبلّغ ثانيةً على الطلب
// نفسِه، أو يتّصل، **أو يسكت ويحلّها مع السائق في الشارع.** والأخيرةُ أسوأُ
// ما يقع: خلافٌ خارج المنصة لا نعلم به.
//
// (طلبُ المالك ٢٠٢٦-٠٨-٢٣: «وبالقائمة الجانبيّة يجب أن يكون قسم الشكاوي
//  والبلاغات — غير موجود حاليّاً».)
//
// # ولماذا بابٌ جديدٌ ولم يُستعمل `/my/tickets`
//
// **الخانةُ التي تحمل صاحبَ التذكرة ليست واحدة.** بلاغُ المتجر يُسجَّل
// (`support/merchant_report.go`) هكذا:
//
//	customer_id     = زبونُ الطلب        ← ليُقرأ في ملفّ الزبون
//	created_by      = مالكُ المتجر        ← هو المُبلِّغ
//	against_user_id = سائقُ الطلب         ← المشكوُّ عليه
//
// **و`/my/tickets` يفلتر بـ`customer_id`** — فلو رُبط به المتجرُ لرأى
// **شكاوى الزبائن عليه** بدل بلاغاتِه هو. **وذلك تسريبٌ لا نقصٌ في عرض**:
// شكوى زبونٍ على متجرٍ تُقرأ في مكتب المنصة، لا في جيب المتجر.
//
// # وما لا يُعرض
//
// **لا اسمَ سائقٍ ولا رقمَه** — البلاغُ عند المنصة والمنصةُ تفصل. **واسمُ
// من شُكي عليه في يد الشاكي بابُ ثأرٍ لا بابُ عدل.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// handleMerchantMyReports بلاغاتُ المتجر التي أرسلها — الأحدثُ أوّلاً.
//
// **والاسمُ ليس `handleMerchantReports`** — ذاك تقريرُ مبيعاته
// (`merchant_handlers.go`). **ولفظُ «تقرير» يحمل معنيين في هذا المشروع**:
// حصيلةُ بيعٍ، وبلاغٌ على أحد. **فبابُهما مفترقٌ في المسار كما في الاسم**:
// `/merchant/stores/{id}/reports` مالٌ، و`/merchant/my-reports` بلاغ.
func (s *Server) handleMerchantMyReports(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT t.id::text, t.number, o.number, COALESCE(t.reason, ''),
		       t.status, COALESCE(t.resolution, ''), t.created_at, t.resolved_at
		FROM tickets t
		LEFT JOIN orders o ON o.id = t.order_id
		WHERE t.created_by = $1 AND NOT t.opened_by_customer
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
		Reason      string `json:"reason"`
		Status      string `json:"status"`
		// Resolution كلمةُ المنصة عند الإغلاق — **وبلاغٌ يُغلق بلا كلمة
		// يُقرأ تجاهلاً**، ولو كان القرارُ في صالحه.
		Resolution string     `json:"resolution"`
		CreatedAt  time.Time  `json:"created_at"`
		ResolvedAt *time.Time `json:"resolved_at"`
	}
	out := []row{}
	for rows.Next() {
		var x row
		if err := rows.Scan(&x.ID, &x.Number, &x.OrderNumber, &x.Reason,
			&x.Status, &x.Resolution, &x.CreatedAt, &x.ResolvedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"reports": out})
}
