package server

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// بوابة المتجر — كل نقطة تتحقق أن الكيان مملوك لصاحب الحساب الفاعل:
// المتجر عبر owner_user_id، والطلب/الصنف عبر متجرهما.

// ownsMerchant يتحقق أن المتجر مملوك للمستخدم.
func (s *Server) ownsMerchant(r *http.Request, merchantID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM merchants WHERE id = $1 AND owner_user_id = $2)`,
		merchantID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

type merchantStore struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	CategoryIcon    string  `json:"category_icon"`
	LogoThumbURL    *string `json:"logo_thumb_url"`
	Status          string  `json:"status"`
	EmergencyClosed bool    `json:"emergency_closed"`
	// **وضبطُه الذي يملكه بيده** — كان يُكتب في القاعدة ولا يُقرأ في بابه،
	// **فصفحةُ إعداداتٍ تفتح بحقولٍ فارغةٍ ثمّ يكتب المالكُ فيها ما يظنّه.**
	PrepMinutes int   `json:"default_prep_minutes"`
	MinOrder    int64 `json:"min_order"`
}

// handleMerchantStores متاجر صاحب الحساب.
func (s *Server) handleMerchantStores(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, c.icon, lm.thumb_path, m.status, m.emergency_closed,
		       m.default_prep_minutes, m.min_order
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.owner_user_id = $1 ORDER BY m.created_at`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []merchantStore{}
	for rows.Next() {
		var m merchantStore
		if err := rows.Scan(&m.ID, &m.Name, &m.CategoryIcon, &m.LogoThumbURL,
			&m.Status, &m.EmergencyClosed, &m.PrepMinutes, &m.MinOrder); err != nil {
			s.respondErr(w, err)
			return
		}
		m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
		out = append(out, m)
	}
	// **أيدير المتجرُ طلباته بنفسه؟** إعدادُ منصّةٍ لا يملك المتجرُ قراءته من
	// بابه (`/admin/settings` للإدارة)، ويحتاجه ليعرف أيعرض أزرارَ القبول.
	// فيُمرَّر مع متاجره — **الجوابُ مع السؤال، لا نداءٌ ثانٍ لسطرٍ واحد**.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"stores":             out,
		"self_manage_orders": s.orders.MerchantsSelfManage(r.Context()),
	})
}

// handleMerchantOrders طلبات متجر مملوك.
func (s *Server) handleMerchantOrders(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))

	// **الجاريةُ تُخفى في وضع «المنصة تدير» — والسجلُّ يبقى كاملاً.**
	//
	// قرارُ المالك (٢٠٢٦-٠٨-٠٢): **«غيرُ مجبرٍ على فتح البرنامج، الطلباتُ
	// تصله عبر الواتساب»**. وشاشةُ طلباتٍ جاريةٍ لا يملك فيها زرّاً **تُوهمه
	// أنّ عليه متابعتها**، فيفتحها ويجدها تتحرّك بلا يده — **وشاشةٌ تُشاهَد
	// ولا تُلمَس تُربك أكثرَ ممّا تُفيد.**
	//
	// **والسجلُّ ضرورةٌ لا ترفٌ**: صاحبُ المطعم يسأل «ماذا بعتُ اليومَ وبكم؟»
	// — وذاك حقُّه في كلّ الأوضاع.
	//
	// **والحجبُ في الخادم لا في الشاشة**: من فتح أدوات المتصفّح قرأ الردَّ
	// كما هو.
	f := orders.ListFilter{
		MerchantID: merchantID,
		Status:     q.Get("status"),
		OpenOnly:   q.Get("open_only") == "true",
		Page:       page,
		PerPage:    perPage,
	}
	// ══════════════════════════════════════════════════════════════════
	// **وسجلُّ الطلبات يُطلب صراحةً — ويُجاب في الوضعين**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «أضِف سجلَّ الطلبات بالحالتين — قسمٌ خاصٌّ
	//  بكلّ الطلبات من هذا المتجر… ليعرف المتجرُ ماذا سلّم وماذا أُلغي
	//  منه».)
	//
	// **وصاحبُ المطعم يسأل «ماذا بعتُ وماذا ضاع منّي؟»** — وذاك حقُّه في
	// كلّ الأوضاع. **والجاريةُ وحدَها هي المحجوبةُ في وضع «المنصّة تدير»**،
	// لأنّه لا يملك فيها زرّاً.
	if q.Get("closed_only") == "true" {
		f.ClosedOnly = true
		f.OpenOnly = false
	} else if !s.orders.MerchantsSelfManage(r.Context()) {
		f.ClosedOnly = true
		f.OpenOnly = false
	}
	res, err := s.orders.List(r.Context(), f)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// خصوصيةُ الزبون تبقى عند المنصة — والحجبُ هنا لا في الواجهة
	for i := range res.Orders {
		redactForMerchant(&res.Orders[i])
	}

	// ══════════════════════════════════════════════════════════════════
	// **وعددُ كلّ حالٍ مع السجلّ — لا بنداءٍ لكلّ حال**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «خلّي الحالات كروتاً ذكيّةً — أفضلُ من
	//  القائمة المنسدلة».)
	//
	// **والكرتُ الذكيُّ يقول عددَه** — وإلّا فهو زرٌّ لا كرت. **ومن رأى
	// «ملغاة» بلا رقمٍ لا يعرف أيضغطها أم يمرّ**، فيضغط كلَّ واحدةٍ ليرى.
	//
	// **وستّةُ نداءاتٍ لستّة كروتٍ عبثٌ**: استعلامٌ واحدٌ يجمعها كلَّها،
	// **ولا يتبدّل بالترشيح** — العددُ عن كلّ ما في السجلّ لا عمّا يُعرض.
	//
	// **ولا يُحسب من الصفحة المعروضة**: **رقمٌ يُشتقّ من صفحةٍ وهو عن الكلّ**
	// — وهي عائلةُ العطب التي أمسكتها النزاعاتُ والتقييماتُ والخسائر.
	counts := map[string]int{}
	crows, err := s.pg.Query(r.Context(), `
		SELECT status, count(*) FROM orders
		WHERE merchant_id = $1 AND closed_at IS NOT NULL
		GROUP BY status`, merchantID)
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
		"orders": res.Orders, "total": res.Total,
		"page": res.Page, "per_page": res.PerPage,
		"status_counts": counts,
	})
}

// merchantOwnsOrder يتحقق أن الطلب يتبع متجراً مملوكاً للمستخدم.
func (s *Server) merchantOwnsOrder(r *http.Request, orderID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT EXISTS(
			SELECT 1 FROM orders o JOIN merchants m ON m.id = o.merchant_id
			WHERE o.id = $1 AND m.owner_user_id = $2)`,
		orderID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

func (s *Server) handleMerchantGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.merchantOwnsOrder(r, id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	o, err := s.orders.GetByID(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	redactForMerchant(o)
	httpx.JSON(w, http.StatusOK, o)
}

// handleMerchantTransition قبول/رفض/بدء تحضير — آلة الحالات تضبط المسموح لدور المتجر.
func (s *Server) handleMerchantTransition(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !s.merchantOwnsOrder(r, id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
		// وقت تحضير خاص بهذا الطلب — يُترك فارغاً فيؤخذ افتراض المتجر
		PrepMinutes *int `json:"prep_minutes"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// إلغاءُ المتجر يلزمه سبب: «ضاع طلب» بلا جوابه يترك الزبون والإدارة يخمّنان،
	// ويمنع قياس أي متجرٍ يُكثر الإلغاء.
	if req.To == "cancelled" && strings.TrimSpace(req.Note) == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	// وقت التحضير يُثبَّت لحظة القبول: قبلها لا معنى له، وبعدها يصير تخميناً
	// لأن العدّ يبدأ من القبول لا من الإنشاء.
	if req.To == "accepted" {
		if _, err := s.pg.Exec(r.Context(), `
			UPDATE orders o SET prep_minutes = COALESCE($2,
				(SELECT m.default_prep_minutes FROM merchants m WHERE m.id = o.merchant_id))
			WHERE o.id = $1`, id, req.PrepMinutes); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	o, err := s.orders.Transition(r.Context(), userIDFrom(r), []string{"merchant"}, id, req.To, req.Note)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// وردُّ الانتقال يُعيد الطلب كاملاً — منفذُ تسريبٍ لو نُسي
	redactForMerchant(o)
	httpx.JSON(w, http.StatusOK, o)
}

func (s *Server) handleMerchantMenu(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	menu, err := s.catalog.GetMenu(r.Context(), merchantID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **وسعرُ بيع المنصّة لا يخرج إلى صاحب المتجر**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «هون غلط تُذكر السعر شقد بالمنصّة — الشخصُ
	//  يكتب السعرَ الذي يبيع به المطعمُ بشكلٍ عاديّ».)
	//
	// **هو يضع سعرَه ويقبض عليه** — وما تبيع به المنصّةُ ليس شأنَه.
	// **ورقمٌ ثانٍ بجانبه يدعوه إلى مساءلةٍ لا تخصّ اتّفاقَه**، أو إلى رفع
	// سعره ليقتسم الفرق.
	//
	// **والحجبُ في الخادم لا في الشاشة**: أُخفي في `MenuManager` أيضاً،
	// **لكنّ من فتح أدوات المتصفّح يقرأ الردَّ كما هو** — **وحجبٌ في العرض
	// وحدَه وعدٌ بحجب.** (وهي القاعدةُ نفسُها في إخفاء رقم الزبون عن السائق.)
	for i := range menu {
		for j := range menu[i].Items {
			menu[i].Items[j].Price = menu[i].Items[j].MerchantPrice
		}
	}
	httpx.JSON(w, http.StatusOK, menu)
}

// handleMerchantItemAvailability تشغيل التوفر اليومي — صلاحية المتجر الوحيدة على القائمة
// (إدارة القائمة السيادية للمنصة — قرار 15).
func (s *Server) handleMerchantItemAvailability(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	req, err := decode[struct {
		Available *bool `json:"available"`
	}](r)
	if err != nil || req.Available == nil {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE menu_items i SET available = $3, updated_at = now()
		FROM merchants m
		WHERE i.id = $1 AND m.id = i.merchant_id AND m.owner_user_id = $2`,
		itemID, userIDFrom(r), *req.Available)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.touch("menu", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"available": *req.Available})
}

// handleMerchantEmergency إغلاق/فتح طارئ للمتجر من صاحبه.
func (s *Server) handleMerchantEmergency(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Closed *bool `json:"closed"`
	}](r)
	if err != nil || req.Closed == nil {
		s.respondErr(w, errValidation)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE merchants SET emergency_closed = $3, updated_at = now()
		WHERE id = $1 AND owner_user_id = $2`,
		merchantID, userIDFrom(r), *req.Closed)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, errForbidden)
		return
	}
	// إغلاق متجر وسط الذروة حدث تشغيلي حرج: مكتب المنصة يعرف فوراً، وواجهة
	// الزبون تسقط المتجر من القائمة بلا إعادة تحميل.
	title := notifTitles.storeReopened
	if *req.Closed {
		title = notifTitles.storeClosed
	}
	s.notify.NotifyOps(r.Context(), notifications.Input{
		Kind: notifications.KindAccount, Title: title,
		Body:   s.merchantName(r.Context(), merchantID),
		Entity: "merchant", EntityID: merchantID, Href: "/dashboard/merchants",
	})
	s.touch("merchant", "ops", "merchant:"+merchantID)
	httpx.JSON(w, http.StatusOK, map[string]any{"emergency_closed": *req.Closed})
}

// handleMerchantReports ملخص تشغيلي ومالي لمتجر مملوك بمدى زمني.
func (s *Server) handleMerchantReports(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	to := time.Now()
	from := to.AddDate(0, 0, -6)
	if v := r.URL.Query().Get("from"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			from = t
		}
	}
	if v := r.URL.Query().Get("to"); v != "" {
		if t, err := time.Parse("2006-01-02", v); err == nil {
			to = t
		}
	}
	toEnd := to.AddDate(0, 0, 1)

	// **المبيعاتُ تُقاس بخروج البضاعة لا بوصولها.**
	//
	// كان الشرطُ `status = 'delivered'` — **وهو منطقُ زمنٍ انقضى**: المتجرُ
	// صار يقبض لحظةَ خروج بضاعته من يده، **فطلبٌ استُلم منه ثمّ تعذّر تسليمُه
	// مالُه في محفظته وتقريرُه يقول لم يبع شيئاً.**
	//
	// **ورقمان يختلفان لمعنًى واحد أسوأُ من رقمٍ ناقص**: صاحبُ المتجر يرى
	// رصيدَه أكبرَ من مبيعاته فيظنّ خطأً في أحدهما، **ولا يعرف أيَّهما
	// يصدّق.**
	//
	// **والمرتجعُ يُطرح**: بضاعةٌ عادت إليه وردّ ثمنَها لم تُبَع.
	//
	// و`picked_up_at` هي الفاصل — **لحظةُ خروج البضاعة**، مكتوبةٌ في الطلب
	// لحظةَ وقوعها لا مستنتَجةٌ من حالةٍ نهائية.
	var summary struct {
		Orders    int `json:"orders"`
		Delivered int `json:"delivered"`
		Cancelled int `json:"cancelled"`
		// Sold ما خرج من يده — **وهو ما قبض عليه**.
		Sold int `json:"sold"`
		// Returned ما عاد إليه فرُدّ ثمنُه.
		Returned   int   `json:"returned"`
		Sales      int64 `json:"sales"`
		Commission int64 `json:"platform_commission"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE status = 'delivered'),
		       count(*) FILTER (WHERE status IN ('cancelled','rejected','failed')),
		       count(*) FILTER (WHERE picked_up_at IS NOT NULL AND returned_at IS NULL),
		       count(*) FILTER (WHERE returned_at IS NOT NULL),
		       COALESCE(sum(subtotal) FILTER (WHERE picked_up_at IS NOT NULL
		                                        AND returned_at IS NULL), 0),
		       COALESCE(sum(platform_commission) FILTER (WHERE picked_up_at IS NOT NULL
		                                                  AND returned_at IS NULL), 0)
		FROM orders
		WHERE merchant_id = $1 AND created_at >= $2 AND created_at < $3`,
		merchantID, from, toEnd).
		Scan(&summary.Orders, &summary.Delivered, &summary.Cancelled,
			&summary.Sold, &summary.Returned, &summary.Sales, &summary.Commission)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **والرسمُ البيانيّ بالمقياس نفسِه** — وإلّا لَخالف مجموعُه ملخّصَه فوقه.
	type day struct {
		Date      string `json:"date"`
		Orders    int    `json:"orders"`
		Delivered int    `json:"delivered"`
		Sales     int64  `json:"sales"`
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT d::date::text,
		       COALESCE(o.orders, 0), COALESCE(o.delivered, 0), COALESCE(o.sales, 0)
		FROM generate_series($2::date, $3::date, '1 day') d
		LEFT JOIN (
			SELECT created_at::date AS day, count(*) AS orders,
			       count(*) FILTER (WHERE status = 'delivered') AS delivered,
			       COALESCE(sum(subtotal) FILTER (WHERE picked_up_at IS NOT NULL
			                                        AND returned_at IS NULL), 0) AS sales
			FROM orders WHERE merchant_id = $1 AND created_at >= $2 AND created_at < $4
			GROUP BY 1
		) o ON o.day = d::date`,
		merchantID, from.Format("2006-01-02"), to.Format("2006-01-02"), toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	days := []day{}
	for rows.Next() {
		var d day
		if err := rows.Scan(&d.Date, &d.Orders, &d.Delivered, &d.Sales); err != nil {
			s.respondErr(w, err)
			return
		}
		days = append(days, d)
	}
	// **وتفصيلُ الأصناف — «ماذا بعتُ؟» لا «كم بعتُ؟».**
	//
	// كان التقريرُ مجاميعَ يومية: عددُ طلباتٍ ومبلغٌ. **وصاحبُ المتجر لا يُدير
	// مطبخَه برقمٍ واحد** — يسأل أيُّ صنفٍ يمشي وأيُّه راكد، فيزيد من هذا
	// ويوقف ذاك. **ورقمٌ إجماليٌّ يقول إنّ الأسبوع كان جيّداً ولا يقول لماذا.**
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بالمتجر يجب أن يكون هناك قسم مبيعات ليعرف
	// المتجر ما هي مبيعاته بشكل مفصّل».)
	//
	// **والمقياسُ خروجُ البضاعة كما في المجاميع** — لا حالةُ الطلب النهائية،
	// **ولا يُطرح المرتجَع مرّتين**: يُستثنى هنا كما استُثني هناك.
	//
	// **والسعرُ سعرُه هو** (`merchant_price`) لا ما دفعه الزبون: بينهما هامشُ
	// المنصة، **وتقريرٌ يعرض ما لا يقبضه يجعله يحسب أرباحاً ليست له.**
	type soldItem struct {
		Name string `json:"name"`
		Qty  int    `json:"qty"`
		// Revenue ما استحقّه عن هذا الصنف قبل العمولة.
		Revenue int64 `json:"revenue"`
	}
	irows, err := s.pg.Query(r.Context(), `
		SELECT oi.name, sum(oi.qty)::int, sum(oi.merchant_price * oi.qty)::bigint
		FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.merchant_id = $1
		  AND o.created_at >= $2 AND o.created_at < $3
		  AND o.picked_up_at IS NOT NULL AND o.returned_at IS NULL
		GROUP BY oi.name
		ORDER BY 3 DESC, 2 DESC
		LIMIT 100`, merchantID, from.Format("2006-01-02"), toEnd)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer irows.Close()
	items := []soldItem{}
	for irows.Next() {
		var it soldItem
		if err := irows.Scan(&it.Name, &it.Qty, &it.Revenue); err != nil {
			s.respondErr(w, err)
			return
		}
		items = append(items, it)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"summary": summary, "days": days, "items": items,
	})
}

// merchantName اسم المتجر لنص الإشعار (فارغ عند التعذّر — لا يُفشل العملية).
func (s *Server) merchantName(ctx context.Context, id string) string {
	var name string
	_ = s.pg.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1`, id).Scan(&name)
	return name
}
