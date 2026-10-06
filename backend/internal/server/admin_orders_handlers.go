package server

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// rolesFrom **فاعلو آلةِ الحالات** — لا أدوارُ الجلسة كما هي.
//
// ══════════════════════════════════════════════════════════════════════
//
//	**وآلةُ الحالات مفرداتُها نطاقٌ لا سجلُّ أدوار**  `BOOK-03`
//
// ══════════════════════════════════════════════════════════════════════
//
// **خريطةُ الانتقالات تعرف خمسةَ فاعلين**: `customer` · `merchant` ·
// `driver` · `ops` · `admin` (`orders/statuses.go`). **وهي مفرداتُ نطاقٍ
// تصف من يفعل ماذا في عمرِ الطلب** — **لا قائمةَ أدوارٍ في القاعدة.**
//
// # والعطبُ الذي كان
//
// **`ops` في الخريطة كان يُطابَق باسم الدور** — **و`ops` دورٌ مُحالٌ إلى
// الإرث** (`authz/roleclass.go`: `ClassLegacy` ⇒ `GrantNever`)، **ودورُ
// العمليّات الحيُّ اسمُه `operations`** — **ولا يظهر في `internal/orders`
// إطلاقاً.**
//
// **فموظّفُ عمليّاتٍ يُنشأ من اللوحة كان لا يستطيع نقلَ حالةِ طلبٍ واحدة**:
// **البابُ يفتح له بالقدرة** (`orders.intervene` في `authz/policy.go`)
// **والآلةُ ترفضه بالاسم.** **ووضعُ المنصّة كلُّه يقوم على أنّ العمليّاتَ
// تقبل وتوزّع** — **فالموظّفُ الذي يُدير المنصّة لا يستطيع أن يُديرها.**
//
// (قِيس ٢٠٢٦-٠٩-٣٠ في تجربة المنصّة. **ولم يُصِب التجربةَ نفسَها** لأنّ
// `canTransition` تستثني `admin` صراحةً — **فالمالكُ يعمل والموظّفُ لا.**)
//
// # والإصلاحُ بالعقد لا بالترقيع
//
// **عقدُ المنصّة `ADG-2`**: «كلُّ بابٍ بقدرته لا باسم دور» — **وحارسٌ
// بأسماء أدوارٍ فوقه يُعطّل دوراً مُنح.** **وإضافةُ `"operations"` إلى
// الخريطة ترقيعٌ يُثبّت اسماً ثانياً** ويُعيد العطبَ لثالثٍ غداً.
//
// **فالفاعلُ `ops` يُشتقّ من القدرة**: من ملك `orders.intervene` فهو فاعلُ
// عمليّاتٍ في الآلة — **أيَّ دورٍ حمل، اليومَ أو غداً.**
//
// **وأنواعُ الحسابات تبقى كما هي** (`customer` · `driver` · `merchant`) —
// **فهي صفةُ الحساب لا قدرةٌ تُمنَح**، والآلةُ تسألها عن الطرف لا عن الإذن.
func rolesFrom(r *http.Request) []string {
	roles, _ := r.Context().Value(ctxRoles).([]string)
	if !s0HasOpsIntervene(r) {
		return roles
	}
	for _, x := range roles {
		if x == "ops" {
			return roles
		}
	}
	// **ولا يُبدَّل المُمرَّرُ للنداء** — نسخةٌ، فلا يُفسد سياقاً مشتركاً.
	out := make([]string, 0, len(roles)+1)
	out = append(out, roles...)
	return append(out, "ops")
}

// s0HasOpsIntervene **أيملك الفاعلُ قدرةَ التدخّل في الطلبات؟**
//
// **ولا يُقرأ دورٌ باسمه** — القدراتُ محسوبةٌ في الحقيقة الموثوقة
// (`ctxCaps`، `ADG-1`).
func s0HasOpsIntervene(r *http.Request) bool {
	caps, _ := r.Context().Value(ctxCaps).([]string)
	for _, c := range caps {
		if c == string(authz.OrdersIntervene) {
			return true
		}
	}
	return false
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
		// **وبلاغٌ ينتظر المكتب** — من بطاقة «بانتظار قرارك» في الرئيسيّة.
		AwaitingOffice: q.Get("awaiting") == "1",
		// **وفلاترُ اللوحة بعدّاداتها** — الشرطُ الذي عدّها بعينه (البند ٣).
		Board: q.Get("filter"),
		// **والأولويّةُ لشاشة العمل** — المُنذَرُ أوّلاً ثمّ المتأخّرُ ثمّ الأقدم (البند ١).
		Priority: q.Get("sort") == "priority",
		// **ومرحلةُ الجاري** — من بطاقات «الآن» في رئيسيّة المدير (٢٠٢٦-١٠-٠٤).
		Stage: q.Get("stage"),
		// **ومدى التاريخ بيوم دمشق** — سجلُّ الطلبات (قرارُ المالك ٢٠٢٦-١٠-٠٣).
		From: q.Get("from"),
		To:   q.Get("to"),
		// **ونوعُ الطلب** — عاديٌّ أو خاصٌّ أو توصيلة (سجلُّ الطلبات ٢٠٢٦-١٠-٠٤).
		Kind: q.Get("kind"),
		// **والمتجرُ أيُّ مصدرٍ شارك** — لا صاحبُ الطلب الأوّلُ وحدَه (البند ٧).
		AnySource: true,
		// **والأعدادُ تتبع البحثَ والفلاتر** — يطلبها السجلّ.
		WithCounts: q.Get("counts") == "1",
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
		// **وعند باب الزبون يقرّر المكتب** — فيرى ما قال السائقُ وكم ينتظر
		// (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢، البند ١، `orders/door_view.go`).
		// **وفي الطريق أيضاً** (٢٠٢٦-١٠-٠٣): الزبونُ يطلب الإلغاءَ والسائقُ ماشٍ.
		if orders.OfficeAnswers(res.Orders[i].Status) {
			res.Orders[i].Door = s.officeDoor(r, res.Orders[i].ID, res.Orders[i].Status)
		}
	}
	// **وما يحتاجه المكتبُ وحدَه** — سببُ العلوق والعرضُ الحيّ وسقفُ التعويض،
	// **في رحلةٍ واحدةٍ للصفحة كلِّها.**
	s.fillBoard(r, res.Orders)
	// **وتفاصيلُ الزبون لثلاثةٍ وحدَهم** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٣).
	s.writeOrdersJSON(w, s.maskOrderDetails(r, res.Orders), res)
}

// fillBoard يملأ `Board` لصفحةٍ من الطلبات — **وتعذّرُه لا يُسقط القائمة**:
// البطاقةُ تُعرض بلا سببِ علوقٍ خيرٌ من شاشةٍ بيضاء.
func (s *Server) fillBoard(r *http.Request, list []orders.Order) {
	ids := make([]string, len(list))
	for i := range list {
		ids[i] = list[i].ID
	}
	info, err := s.orders.BoardInfoOf(r.Context(), ids)
	if err != nil {
		s.logger.Warn("لوحةُ الطلبات: تعذّرت معلوماتُ المكتب", "error", err)
		return
	}
	for i := range list {
		list[i].Board = info[list[i].ID]
	}
}

// boardMeta **إعداداتُ اللوحة التي تقرؤها الشاشة** — من الخادم لا من `/settings`.
//
// (البند ٣٢: دورٌ مخصّصٌ بلا `settings.read` كان يأخذ افتراضاتٍ مكتوبةً في
// الشاشة — مهلةَ ١٠ و«المنصّة تدير» — **وقد تخالف إعداداتِ المالك.**)
type boardMeta struct {
	ManualAssignAfterMin int64  `json:"manual_assign_after_min"`
	OfferTimeoutSec      int64  `json:"offer_timeout_sec"`
	OrdersMode           string `json:"orders_mode"`
	CashBanDays          int64  `json:"cash_ban_days"`
	// StaleLocationMin **متى يُعدّ موضعُ السائق متوقّفاً** — الحدُّ نفسُه الذي يحجب به
	// المحرّكُ الإسنادَ الآليّ، **فلا تقول الشاشةُ «ظاهر» عمّن حجبه المحرّك.**
	StaleLocationMin int `json:"stale_location_minutes"`
}

// handleOrdersBoard **عدّاداتُ لوحة الطلبات وإعداداتُها** — `GET /admin/orders/board`.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٣: «كلُّ عدّادٍ يُضغط فيفتح القائمةَ بالشرط
// الذي عدّه بعينه».) **والعدُّ من `orders.BoardFilterSQL`** — نصُّ القائمة نفسُه.
func (s *Server) handleOrdersBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	counts, err := s.orders.BoardCounts(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	meta := boardMeta{
		ManualAssignAfterMin: s.settings.GetInt(ctx, "orders.manual_assign_after_min"),
		OfferTimeoutSec:      s.settings.GetInt(ctx, "drivers.offer_timeout_sec"),
		OrdersMode:           s.settings.GetString(ctx, "platform.orders_mode"),
		CashBanDays:          s.settings.GetInt(ctx, "customers.cash_ban_days"),
		StaleLocationMin:     staleLocationMinutes,
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"counts":  counts,
		"filters": orders.BoardFilters,
		"meta":    meta,
	})
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
	if orders.OfficeAnswers(o.Status) {
		o.Door = s.officeDoor(r, o.ID, o.Status)
	}
	one := []orders.Order{*o}
	s.fillBoard(r, one)
	strip := s.maskOrderDetails(r, one)
	o = &one[0]
	// ══════════════════════════════════════════════════════════════════
	// **ومسارُه كاملاً بأوقاته ومن فعله**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٢: «الإدارة تشوف المسار … وكلُّ شيءٍ واضح
	//  التوقيت والساعة والدقيقة».)
	//
	// **والبياناتُ محفوظةٌ منذ اليوم الأوّل** في `order_events` — **ولا
	// أحدَ يعرضها**: فلا يُعرف أين ضاع الوقتُ في طلبٍ تأخّر.
	// **وكودُ التسليم لمن يملك التدخّل وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٦) — يقرؤه موظّفُ
	// العمليّات للسائق بعد أن يتحقّق من الزبون بالهاتف. **ولغيره يغيب الحقلُ كلُّه.**
	var code *string
	if s0HasOpsIntervene(r) {
		_ = s.pg.QueryRow(r.Context(),
			`SELECT delivery_code FROM orders WHERE id = $1`, o.ID).Scan(&code)
	}
	s.writeOrdersJSON(w, strip, struct {
		*orders.Order
		Timeline     []TimelineStep `json:"timeline"`
		DeliveryCode *string        `json:"delivery_code,omitempty"`
	}{o, s.timeline(r.Context(), o.ID, true), code})
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

	// **والتدخّلُ وأثرُه في معاملةٍ واحدة** — `XG-20` · `AQ-4`.
	//
	// **ولا انتقالَ ثانٍ يُكتب بيد**: آلةُ الحال واحدةٌ، **والمِعراضُ
	// يمرّر منفّذَها** — فيبقى `order_events` والتسوياتُ والإسنادُ
	// كما هي، **والإشعارُ بعد التثبيت.**
	oid := chi.URLParam(r, "id")
	o, err := s.orders.TransitionAudited(r.Context(), userIDFrom(r), roles,
		oid, req.To, note,
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.order_transition", "order", oid,
				map[string]any{"to": req.To, "note": req.Note})
		})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// موظّفٌ يحرّك طلباً نيابةً عن طرفه: إلغاءٌ أو استرجاعٌ بيده يُطلق تسويات
	// مالية معاكسة. **ولا يُسجَّل انتقالُ الأطراف أنفسهم** — المتجر يقبل مئة
	// طلب في اليوم، وتسجيلُها يُغرق السجلّ فيصير لا يُقرأ.
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
	switch {
	case err == nil:
	case errors.Is(err, orders.ErrDriverOffShift):
		s.respondErr(w, errAssignOffShift)
		return
	case errors.Is(err, orders.ErrDriverExcluded):
		s.respondErr(w, errAssignExcluded)
		return
	case errors.Is(err, orders.ErrOrderTaken):
		s.respondErr(w, errAssignTaken)
		return
	default:
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

// handleOrderSeen **«استلمتها» على طلبٍ جديد** — `POST /admin/orders/{id}/seen`.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٥: «رنينٌ متكرّرٌ للطلب الجديد حتّى يضغط
// موظّفٌ استلمتها».) **والضغطةُ تُكتب في الطلب** فيسكت الرنينُ عند المكتب كلِّه.
func (s *Server) handleOrderSeen(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	changed, err := s.orders.MarkSeen(r.Context(), id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"seen": true, "changed": changed})
}

// assignCandidate سائقٌ يصلح للإسناد اليدويّ — بما يُقرَّر به.
type assignCandidate struct {
	ID         string   `json:"id"`
	FullName   string   `json:"full_name"`
	Phone      string   `json:"phone"`
	DistanceM  *float64 `json:"distance_m"`
	CashHeld   int64    `json:"cash_held"`
	OpenOrders int      `json:"open_orders"`
	// LocationAt **آخرُ موضعٍ وصل منه** — والفارغُ «لم يصل قطّ».
	LocationAt *time.Time `json:"location_at"`
}

// handleAssignCandidates **مرشّحو الإسناد اليدويّ** — `GET /admin/orders/{id}/assign-candidates`.
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٠ — و٣٥: «مرتّبةً بالقرب، فيها المسافةُ
// والنقدُ والطلبات».) **ومن في الدوام وحدَه، ولا من ترك الطلب** — الشرطان
// اللذان يفرضهما المحرّكُ عند الإسناد (`AssignDriver`)، **فلا يُعرض من سيُردّ.**
//
// **والمسافةُ من نقطة الاستلام** (البديلة إن كانت، وإلّا المتجر) **إلى آخر
// موضعٍ للسائق** — وموضعٌ لا يُعرف يقع آخرَ القائمة لا أوّلَها: **الجهلُ ليس قرباً.**
func (s *Server) handleAssignCandidates(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rows, err := s.pg.Query(r.Context(), `
		SELECT u.id::text, COALESCE(u.full_name, ''), COALESCE(u.phone, ''),
		       CASE WHEN u.last_location IS NOT NULL
		             AND COALESCE(o.pickup_override, m.location) IS NOT NULL
		            THEN ST_Distance(u.last_location, COALESCE(o.pickup_override, m.location)) END,
		       COALESCE((SELECT b.held FROM driver_cash_boxes b WHERE b.driver_id = u.id), 0),
		       (SELECT count(*) FROM orders oo WHERE oo.driver_id = u.id AND oo.closed_at IS NULL),
		       u.last_location_at
		FROM orders o
		LEFT JOIN merchants m ON m.id = o.merchant_id
		JOIN users u ON u.on_shift AND u.status = 'active'
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE o.id = $1
		  AND NOT (u.id = ANY(o.excluded_drivers))
		ORDER BY 4 ASC NULLS LAST, 6 ASC, 2`, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	list := []assignCandidate{}
	for rows.Next() {
		var c assignCandidate
		if err := rows.Scan(&c.ID, &c.FullName, &c.Phone, &c.DistanceM, &c.CashHeld,
			&c.OpenOrders, &c.LocationAt); err != nil {
			s.respondErr(w, err)
			return
		}
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"drivers":                list,
		"stale_location_minutes": staleLocationMinutes,
	})
}

// حرّاسُ الإسناد اليدويّ بلغة الإدارة (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ١٠).
var (
	errAssignOffShift = httpx.NewError(http.StatusConflict,
		"driver_off_shift", "errors.driver_off_shift")
	errAssignExcluded = httpx.NewError(http.StatusConflict,
		"driver_excluded", "errors.driver_excluded")
	errAssignTaken = httpx.NewError(http.StatusConflict,
		"order_already_taken", "errors.order_already_taken")
)

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

// officeDoor **لوحةُ ردّ المكتب** — وقبل المتجر لا لوحةَ إلّا ببلاغٍ أو أمر (٢٠٢٦-١٠-٠٣):
// **كلُّ طلبٍ في طريقه إلى المتجر لا يحتاج قراراً.**
func (s *Server) officeDoor(r *http.Request, orderID, status string) *orders.DoorView {
	v := s.orders.DoorViewOf(r.Context(), orderID)
	if status == orders.StAssigned && v.ReportCode == "" && v.Instruction == "" {
		return nil
	}
	return v
}
