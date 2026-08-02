package server

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

func rolesFrom(r *http.Request) []string {
	roles, _ := r.Context().Value(ctxRoles).([]string)
	return roles
}

func (s *Server) handleListOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		Status:     q.Get("status"),
		MerchantID: q.Get("merchant_id"),
		CustomerID: q.Get("customer_id"),
		DriverID:   q.Get("driver_id"),
		Query:      q.Get("query"),
		OpenOnly:   q.Get("open") == "1",
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ولماذا لا يلتقطه أحد — يُقال في البطاقة لا في تنبيهٍ بعد عشر دقائق.**
	//
	// طلبٌ فوق سقف النقد لا يظهر لسائقٍ أبداً، **والعملياتُ ترى «جارٍ إسناد
	// سائق» وتنتظر من لن يأتي.** ويُحسب للطابور وحدَه: **سؤالٌ لكلّ بطاقةٍ
	// في صفحةٍ من عشرين حملٌ بلا حاجة.**
	for i := range res.Orders {
		if res.Orders[i].Status == orders.StDispatching {
			res.Orders[i].BlockedReason = s.orders.NoEligibleReason(r.Context(), res.Orders[i].ID)
		}
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleOrderAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := s.orders.Alerts(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, alerts)
}

func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	o, err := s.orders.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// requiresReason الانتقالاتُ التي لا تُقبل بلا تعليل.
//
// كلُّها تُنهي الطلب أو تعكس مالاً: الرفضُ والإلغاءُ يُرجعان ما دُفع، والفشلُ
// يُغلق بلا تسليم، والاسترجاعُ يعكس تسويةً تمّت. **والسببُ ليس زينةً**: هو ما
// يُقال للزبون، وما يُقاس به متجرٌ يُكثر الرفض أو موظّفٌ يُكثر الإلغاء.
var requiresReason = map[string]bool{
	"rejected":  true,
	"cancelled": true,
	"failed":    true,
	"refunded":  true,
}

func (s *Server) handleOrderTransition(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
		// ManualOverride توقيعُ المالك أنّ ما يفعله **تدخّلٌ يدويّ** لا فعلُ
		// صاحبِ الدور.
		//
		// **يلزم لمراحل السائق**: هاتفٌ نفدت بطاريتُه يترك الطلبَ عالقاً بلا
		// من يُكمله — **وسؤالُ المالك كشفه**: «ربما لن يكون هناك موظفين وفقط
		// مدير المنصة سوف يدير العمل».
		//
		// **والمشكلةُ لم تكن «من ضغط» بل «ماذا يقول السجلّ»**: بالتوقيع يصدق
		// السجلُّ — «أعلنتها المنصةُ» لا «قالها السائق».
		ManualOverride bool `json:"manual_override"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ما لا يُستدرَك يلزمه سبب.**
	//
	// كان السبب إلزامياً على المتجر وحده (`merchant_handlers.go`) ومفتوحاً
	// للعمليات. فيُلغى طلبٌ من اللوحة بلا كلمة، ويبقى الزبون والمتجر يخمّنان
	// — **ولا يُقاس موظّفٌ يُكثر الإلغاء ولا متجرٌ يُكثر الرفض**.
	//
	// والحارسُ هنا لا في الخارطة: الخارطةُ تقول **من يملك** الانتقال، وهذا
	// شرطٌ على **كيف** يُمارَس.
	if requiresReason[req.To] && strings.TrimSpace(req.Note) == "" {
		s.respondErr(w, errReasonRequired)
		return
	}

	// **والتوقيعُ لا يُقبل من غير المالك** — ولا يُقبل بلا كلمةٍ تشرحه.
	//
	// **دورٌ اصطناعيّ يُضاف هنا ولا يُمنح لأحد في القاعدة**: `rolesUnderMode`
	// تقرأ الأدوارَ وحدَها، **ومعاملٌ جديدٌ يمرّ عبر خمس دوالّ ليصل.**
	roles := rolesFrom(r)
	note := strings.TrimSpace(req.Note)
	if req.ManualOverride {
		if !slices.Contains(roles, "admin") {
			s.respondErr(w, errForbidden)
			return
		}
		if note == "" {
			s.respondErr(w, errReasonRequired)
			return
		}
		roles = append(roles, orders.RoleManualOverride)
		// **والسجلُّ يحمل التوقيع** — فمن قرأه بعد شهرٍ عرف أنّ المنصةَ
		// أعلنته، **ولم يقله السائق.**
		note = "تدخّلٌ يدويّ من المنصة — " + note
	}

	o, err := s.orders.Transition(r.Context(), userIDFrom(r), roles,
		chi.URLParam(r, "id"), req.To, note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// موظّفٌ يحرّك طلباً نيابةً عن طرفه: إلغاءٌ أو استرجاعٌ بيده يُطلق تسويات
	// مالية معاكسة. **ولا يُسجَّل انتقالُ الأطراف أنفسهم** — المتجر يقبل مئة
	// طلب في اليوم، وتسجيلُها يُغرق السجلّ فيصير لا يُقرأ.
	s.audit(r, "ops.order_transition", "order", chi.URLParam(r, "id"), map[string]any{
		"to": req.To, "note": req.Note,
	})
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleOrderAssign(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		DriverID string `json:"driver_id"`
		Note     string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	o, err := s.orders.AssignDriver(r.Context(), userIDFrom(r), rolesFrom(r),
		chi.URLParam(r, "id"), req.DriverID, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "ops.order_assign", "order", chi.URLParam(r, "id"), map[string]any{
		"driver_id": req.DriverID, "note": req.Note,
	})

	// السائق يعرف بإسناد الطلب فوراً (يستعمله تطبيقه)
	s.notify.Notify(r.Context(), notifications.Input{
		UserID: req.DriverID, Kind: notifications.KindOrder,
		Title: notifTitles.driverAssigned, Entity: "order",
		EntityID: chi.URLParam(r, "id"), Href: "/orders",
	})
	httpx.JSON(w, http.StatusOK, o)
}
