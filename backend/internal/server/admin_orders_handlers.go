package server

import (
	"net/http"
	"strconv"
	"strings"
	"time"

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
		// **وصاحبُ المتجر — طلباتُ متاجره كلِّها** (٢٠٢٦-٠٨-١٦).
		OwnerID:  q.Get("owner_id"),
		Query:    q.Get("query"),
		OpenOnly: q.Get("open") == "1",
		// **والمنتهيةُ وحدَها تُطلب صراحةً.**
		//
		// شاشةُ العمل وشاشةُ السجلّ سؤالان مختلفان: **الأولى «ما الذي يحتاجني
		// الآن؟» والثانية «ماذا جرى لهذا الطلب؟»** — وخلطُهما في قائمةٍ واحدةٍ
		// بمربّعِ اختيارٍ يجعل الأولى تُقرأ سجلّاً والثانية تُقرأ عملاً.
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «يجب أن نفصل بين الطلبات الجديدة والسابقة».)
		ClosedOnly: q.Get("closed") == "1",
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
	// ══════════════════════════════════════════════════════════════════
	// **ومسارُه كاملاً بأوقاته ومن فعله**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الإدارة تشوف المسار … وكلُّ شيءٍ واضح
	//  التوقيت والساعة والدقيقة».)
	//
	// **والبياناتُ محفوظةٌ منذ اليوم الأوّل** في `order_events` — **ولا
	// أحدَ يعرضها**: فلا يُعرف أين ضاع الوقتُ في طلبٍ تأخّر.
	httpx.JSON(w, http.StatusOK, struct {
		*orders.Order
		Timeline []TimelineStep `json:"timeline"`
	}{o, s.timeline(r.Context(), o.ID, true)})
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

// errOrderStillOpen **لا تُعاد تسويةُ طلبٍ لم يُغلق** — تسويتُه ستُنادى في
// مسارها، وقيدٌ يُقحَم في منتصف حياته يُفسد قراءةَ ترتيبه.
var errOrderStillOpen = httpx.NewError(http.StatusConflict,
	"order_still_open", "errors.order_still_open")

func (s *Server) handleOrderTransition(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
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

	roles := rolesFrom(r)
	note := strings.TrimSpace(req.Note)

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
		EntityID: chi.URLParam(r, "id"),
		// **ووجهتُه لوحتُه هو** — كان "/orders"، **وهي صفحةُ طلبات
		// الزبون**: من أُسند إليه طلبٌ فضغط الخبرَ وجد مشترياته هو.
		Href: "/portal", // **لوحتُه أيّاً كانت** — حُذفت `/driver` من الويب ٢٠٢٦-٠٨-٢٣
		// **وتطبيقُ السائق وحدَه يرنّ** — انظر `orders/notify.go`.
		Apps: []string{notifications.AppDriver},
	})
	httpx.JSON(w, http.StatusOK, o)
}

// handleRecomputeSettlement **يُعيد حسابَ نصيب المنصة من طلبٍ بعينه.**
//
// # لماذا يلزم
//
// حسبةُ الخزينة **بالفرق لا بالمجموع**: تحسب النصيبَ كاملاً بحاله الآن، وتطرح
// ما قُيّد سابقاً، وتضع الفرق. **فهي تُصحّح نفسها بمجرّد أن تُنادى.**
//
// **ولا شيءَ ينادِيها على طلبٍ أُغلق.** فإن كُشف خللٌ في المعادلة — كما وقع في
// `#1003` (خسرت الخزينةُ ٣٩٬٨٠٠ على طلبٍ قدرُه ٢٢٬٠٠٠) — **بقي القيدُ الخاطئ في
// الدفتر ولو أُصلح الكود**، ولم يكن أمامنا إلّا تصفيرُ البيانات كلِّها.
//
// # والدفاترُ لا تُعدَّل ولا تُحذف
//
// **تُصحَّح بقيدٍ مقابل** — وهذا ما تفعله المعادلةُ من نفسها: تضع الفرقَ قيداً
// جديداً ويبقى الخطأُ مسطوراً. **ومن قرأ الدفترَ بعد سنةٍ رأى الخطأَ وتصحيحَه
// معاً** — وهو ما يُميّز دفتراً من قاعدة بيانات.
//
// # ولا تُنادى إلّا على مُغلَق
//
// طلبٌ جارٍ ستُنادى تسويتُه في مسارها، **وإعادةُ الحساب عليه تُقحم قيداً في
// منتصف حياته** فيصعب قراءةُ ترتيبه. **والأدمنُ وحدَه**: قيدٌ ماليٌّ يُنشأ بيد.
func (s *Server) handleRecomputeSettlement(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var closed *time.Time
	if err := s.pg.QueryRow(r.Context(),
		`SELECT closed_at FROM orders WHERE id = $1`, id).Scan(&closed); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if closed == nil {
		s.respondErr(w, errOrderStillOpen)
		return
	}

	actor := userIDFrom(r)
	tx, err := s.pg.Begin(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()

	// **الفرقُ يُقاس قبلَ وبعد** — فيُقال للمالك كم صُحِّح، **ولا يُقال «تمّ»
	// عن نداءٍ لم يُغيّر شيئاً.**
	var before int64
	if err := tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'platform_profit'`, id).Scan(&before); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.orders.CreditTreasuryTx(r.Context(), tx, id, actor); err != nil {
		s.respondErr(w, err)
		return
	}
	var after int64
	if err := tx.QueryRow(r.Context(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'platform_profit'`, id).Scan(&after); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		s.respondErr(w, err)
		return
	}

	s.audit(r, "finance.settlement_recomputed", "order", id, map[string]any{
		"before": before, "after": after, "delta": after - before,
	})
	s.touch("wallet", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"before": before, "after": after, "delta": after - before,
	})
}
