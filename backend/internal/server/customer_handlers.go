package server

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/media"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// واجهة الزبون: نقاط عامة للتصفح (بلا حساب) ونقاط الطلب/التتبع/المحفظة
// بحساب الزبون — الطلب حصراً من هنا (قرار 18).

// openNowSQL: المتجر يستقبل الآن؟
//
// **والنصُّ في `orders` لا هنا** — لأن إنشاءَ الطلب يفحصه أيضاً، **ونصّان
// لمعنًى واحد يفترقان**: يُصلَح أحدُهما ويبقى الآخر، فيقول العرضُ «مغلق»
// ويقبل الإنشاءُ الطلب.
const openNowSQL = orders.OpenNowSQL

type publicMerchant struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	CategoryID   string  `json:"category_id"`
	CategoryIcon string  `json:"category_icon"`
	LogoURL      *string `json:"logo_url"`
	LogoThumbURL *string `json:"logo_thumb_url"`
	OpenNow      bool    `json:"open_now"`
}

// handlePublicHome بيانات الصفحة الرئيسية: بانرات وفئات ومتاجر فعالة بحالة فتحها.
func (s *Server) handlePublicHome(w http.ResponseWriter, r *http.Request) {
	banners, err := s.catalog.ListBanners(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	active := banners[:0]
	for _, b := range banners {
		if b.Active && b.ImageURL != nil {
			active = append(active, b)
		}
	}

	categories, err := s.catalog.ListCategories(r.Context(), true)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, m.description, m.category_id, c.icon,
		       lm.path, lm.thumb_path, `+openNowSQL+`
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.status = 'active'
		ORDER BY `+openNowSQL+` DESC, m.created_at`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	merchants := []publicMerchant{}
	for rows.Next() {
		var m publicMerchant
		if err := rows.Scan(&m.ID, &m.Name, &m.Description, &m.CategoryID, &m.CategoryIcon,
			&m.LogoURL, &m.LogoThumbURL, &m.OpenNow); err != nil {
			s.respondErr(w, err)
			return
		}
		m.LogoURL = media.URLForPtr(m.LogoURL)
		m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)
		merchants = append(merchants, m)
	}
	// رقم الدعم يُحمَّل مع الصفحة الأولى لا بنداءٍ ثانٍ: هو سطرٌ واحد في
	// التذييل، ونداءٌ مستقلٌّ له تكلفةُ رحلةٍ كاملة لسطر.
	//
	// وكان الرقم لا وجود له أصلاً: الشكوى تذهب إلى التذاكر وحدها، ومن لا يعرف
	// التذاكر لا يجد باباً. ويبقى فارغاً حتى يكتبه المالك، فتُخفيه الواجهة.
	httpx.JSON(w, http.StatusOK, map[string]any{
		"banners": active, "categories": categories, "merchants": merchants,
		"support_phone": s.settings.GetString(r.Context(), "platform.support_phone", ""),
	})
}

// handlePublicMerchant متجر واحد بقائمته الكاملة (النافد يظهر معطلاً).
func (s *Server) handlePublicMerchant(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var m publicMerchant
	err := s.pg.QueryRow(r.Context(), `
		SELECT m.id, m.name, m.description, m.category_id, c.icon,
		       lm.path, lm.thumb_path, `+openNowSQL+`
		FROM merchants m
		JOIN categories c ON c.id = m.category_id
		LEFT JOIN media lm ON lm.id = m.logo_media_id
		WHERE m.id = $1 AND m.status = 'active'`, id).
		Scan(&m.ID, &m.Name, &m.Description, &m.CategoryID, &m.CategoryIcon,
			&m.LogoURL, &m.LogoThumbURL, &m.OpenNow)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	m.LogoURL = media.URLForPtr(m.LogoURL)
	m.LogoThumbURL = media.URLForPtr(m.LogoThumbURL)

	menu, err := s.catalog.GetMenu(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"merchant": m, "menu": menu})
}

// handlePublicZone معاينة رسوم التوصيل والحد الأدنى لنقطة على الخريطة.
func (s *Server) handlePublicZone(w http.ResponseWriter, r *http.Request) {
	lat, err1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lng, err2 := strconv.ParseFloat(r.URL.Query().Get("lng"), 64)
	if err1 != nil || err2 != nil {
		s.respondErr(w, errValidation)
		return
	}
	z, err := s.catalog.ZoneForPoint(r.Context(), lat, lng)
	if err != nil {
		s.respondErr(w, orders.ErrOutOfZone)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"name": z.Name, "delivery_fee": z.DeliveryFee, "min_order": z.MinOrder,
	})
}

// handleCustomerCreateOrder إنشاء طلب بحساب الزبون نفسه — التسعير خادمي بالكامل.
func (s *Server) handleCustomerCreateOrder(w http.ResponseWriter, r *http.Request) {
	in, err := decode[orders.CreateInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	in.CustomerID = userIDFrom(r) // الطلب باسم صاحب الحساب حصراً
	in.CustomerPhone = ""
	o, err := s.orders.Create(r.Context(), userIDFrom(r), rolesFrom(r), *in, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, o)
}

// handleMyOrders طلبات الزبون نفسه.
func (s *Server) handleMyOrders(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	perPage, _ := strconv.Atoi(q.Get("per_page"))
	res, err := s.orders.List(r.Context(), orders.ListFilter{
		CustomerID: userIDFrom(r),
		OpenOnly:   q.Get("open_only") == "true",
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, res)
}

func (s *Server) handleMyOrder(w http.ResponseWriter, r *http.Request) {
	o, err := s.orders.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if o.CustomerID != userIDFrom(r) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	// **المهلةُ تُرسل مع الطلب لا في نداءٍ ثانٍ.**
	//
	// الشاشةُ تعرض عدّاداً تنازلياً لزرّ الإلغاء، **ورقمُ المهلة إعدادٌ يملك
	// المالكُ تغييره** — فلو كُتب في الشاشة لخالف الخادمَ بعد أوّل تعديل.
	// **وزرٌّ يَعِد بما يرفضه الخادم أسوأُ من زرٍّ لا يظهر.**
	//
	// و`-1` تعني «بلا مهلة» — أي قبل قبول المتجر: يُلغي متى شاء.
	httpx.JSON(w, http.StatusOK, struct {
		*orders.Order
		CancelSecondsLeft int `json:"cancel_seconds_left"`
	}{o, s.orders.CancelSecondsLeft(r.Context(), o)})
}

// handleMyWallet رصيد الزبون وكشف حركاته.
func (s *Server) handleMyWallet(w http.ResponseWriter, r *http.Request) {
	// بلا مدى: لمحة اللوحة (آخر 50). بمدى: كشف حساب كامل قابل للطباعة.
	st, err := s.wallet.Statement(r.Context(), userIDFrom(r), statementRange(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}

// handleCustomerCancelOrder إلغاء الزبون لطلبه.
//
// **كانت الخارطة تسمح له ولا مسار يوصله**: `pending → cancelled` مخوّلة لدور
// الزبون منذ البداية، لكن لا نقطة في الخادم تنادي الانتقال باسمه — فكان الإلغاء
// حقّاً على الورق بلا باب. وهذا نوعٌ من الخلل لا يظهر في قراءة الشيفرة: كلٌّ من
// الطرفين سليم وحده، والوصلة بينهما مفقودة.
//
// والمحرّك هو من يحكم: يسمح ما دام «بانتظار التأكيد»، ويسمح بعد القبول ضمن
// نافذة التدارُك، ويرفض بعدها.
func (s *Server) handleCustomerCancelOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	o, err := s.orders.GetByID(r.Context(), id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if o.CustomerID != userIDFrom(r) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Note string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	updated, err := s.orders.Transition(r.Context(), userIDFrom(r), []string{"customer"},
		id, "cancelled", clip(req.Note, 300))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, updated)
}
