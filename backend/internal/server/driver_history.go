package server

// سجلُّ السائق — **ما نفّذه، نجح أم فشل.**
//
// # المسألة
//
// شاشتُه كانت «الآن» وحدَه: مهامٌّ جارية وطابورٌ قادم. **وما انتهى اختفى.**
//
// فلا يعرف كم سلّم أمس، ولا يجد طلباً يتذكّره ليقول فيه شيئاً، **ولا يملك أن
// يُبلّغ عن متجرٍ أوقفه نصفَ ساعةٍ أو زبونٍ أعطاه عنواناً وهمياً** — لأنّ
// الطلبَ لم يعد له وجودٌ في تطبيقه.
//
// **وحسابٌ يُقفل بلا سجلٍّ يُقرأ ظلمٌ صامت**: يُخصم منه في آخر الشهر عن طلبٍ
// لا يستطيع أن يراه.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «لازم يكون عندو قسم يشوف طلباته... في حال النجاح
// أو الفشل».)
//
// # والمغلقةُ وحدَها
//
// الجاريةُ في «مهامّي» — **وشاشتان تعرضان الشيءَ نفسَه تفترقان في زرّ**، ثمّ
// يضغط السائقُ على ما لم يعد قائماً.

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// handleDriverHistory طلباتُه المنتهية — الأحدثُ أوّلاً.
func (s *Server) handleDriverHistory(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		DriverID:   userIDFrom(r),
		ClosedOnly: true,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **وأيُّها قيّمتَه سلفاً** — زرٌّ يُعرض على ما قُيّم يُضغط فيُردّ،
	// **ورفضٌ بعد ضغطةٍ يُقرأ عطباً لا قاعدة.**
	rated := map[string]bool{}
	ids := make([]string, 0, len(res.Orders))
	for i := range res.Orders {
		ids = append(ids, res.Orders[i].ID)
	}
	if len(ids) > 0 {
		rows, err := s.pg.Query(r.Context(),
			`SELECT order_id::text FROM merchant_ratings WHERE order_id = ANY($1::uuid[])`, ids)
		if err == nil {
			for rows.Next() {
				var id string
				if rows.Scan(&id) == nil {
					rated[id] = true
				}
			}
			rows.Close()
		}
	}

	type withRating struct {
		orders.Order
		// MerchantRated أقيّمتُ متجرَه — **ومن قيّم لا يُعرض عليه الزرُّ ثانيةً.**
		MerchantRated bool `json:"merchant_rated"`
		// CanRateMerchant **وقف عند بابه فعلاً** — ومن لم يقف لا رأيَ له فيه.
		CanRateMerchant bool `json:"can_rate_merchant"`
	}
	out := make([]withRating, len(res.Orders))
	for i := range res.Orders {
		o := res.Orders[i]
		out[i] = withRating{
			Order:         o,
			MerchantRated: rated[o.ID],
			// **والشرطُ هو شرطُ الخادم نفسُه** — نسخةٌ ثانيةٌ تفترق فيُعرض
			// زرٌّ يُردّ أو يُخفى زرٌّ يجوز.
			// ══════════════════════════════════════════════════
			// **ولا يُقيَّم متجرٌ لا وجودَ له**
			// ══════════════════════════════════════════════════
			//
			// (سأل المالك ٢٠٢٦-٠٨-١٣: «هل بقي شيءٌ بالطلب الخاصّ لم
			//  نكتشفه؟».)
			//
			// **والطلبُ الخاصُّ بلا متجر** — والشرطُ كان «وقف عنده
			// فعلاً» وحدَه: **والخاصُّ يمرّ بـ`picked_up` عند الشراء**،
			// فيصير مؤهَّلاً لتقييم من لا وجودَ له.
			//
			// **ويُعرض عليه الزرُّ فيضغطه فيُردّ** — بأربعمئةٍ وأربعة
			// لأنّ `merchant_id` فارغ: **رفضٌ بعد ضغطةٍ يُقرأ عطباً لا
			// قاعدة.**
			CanRateMerchant: o.MerchantID != "" &&
				(o.PickedUpAt != nil ||
					(o.Status == "failed" && o.Fault == "merchant")),
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": out, "total": res.Total, "page": res.Page, "per_page": res.PerPage,
	})
}

// handleDriverReportReasons ما يملك السائقُ اختيارَه، وعلى من هو.
//
// **والقائمةُ من الخادم** — لا تُكتب في التطبيق: قائمةٌ في مكانين تفترق حين
// يُضاف سببٌ في أحدهما، **فيرسل التطبيقُ رمزاً لا يعرفه الخادم** ويُردّ عليه
// بلا أن يفهم لماذا.
func (s *Server) handleDriverReportReasons(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{"reasons": support.DriverReportReasons})
}

// handleDriverReport يفتح بلاغَ سائقٍ على متجرٍ أو زبون.
func (s *Server) handleDriverReport(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Reason string `json:"reason"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	t, err := s.support.DriverReport(r.Context(), userIDFrom(r), orderID, req.Reason, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والعملياتُ تُنبَّه** — بلاغٌ لا يراه أحدٌ حتى يفتح الشاشةَ صدفةً بلاغٌ
	// لم يُقدَّم.
	s.touch("ticket", "ops")
	httpx.JSON(w, http.StatusOK, t)
}
