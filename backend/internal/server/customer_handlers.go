package server

import (
	"context"
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

// platformLogo مسارُ شعار المنصة — **وفارغٌ يعني أنّ الحرفَ يبقى.**
//
// **ولا يُحذف حرفُ العلامة**: منصّةٌ لم تَرفع شعاراً يجب أن تبقى تعمل،
// **وشريطٌ علويٌّ بمربّعٍ فارغٍ أسوأُ من حرف.**
func (s *Server) platformLogo(r *http.Request) *string {
	id := s.settings.GetString(r.Context(), "platform.logo")
	if id == "" {
		return nil
	}
	var path *string
	if s.pg.QueryRow(r.Context(), `SELECT path FROM media WHERE id = $1`, id).Scan(&path) != nil {
		return nil
	}
	return media.URLForPtr(path)
}

// handlePublicHome بيانات الصفحة الأولى: لافتاتٌ وتصنيفاتٌ وأقسامُ سوق.
//
// **ولا متاجرَ فيها** — انظر الشرحَ عند الأقسام أدناه.
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

	// **ولا متاجرَ في الرئيسية.**
	//
	// كان يُستعلَم هنا عن كلّ متجرٍ فعّالٍ باسمِه ووصفِه وشعارِه **ويُرسل في
	// كلّ فتحةِ صفحةٍ أولى** — **ولا أحدَ يقرؤه**: الرئيسيةُ تتصفّح أقساماً،
	// وخريطةُ الموقع وحدَها كانت تأخذه لتنشر `/m/{id}` لغوغل.
	//
	// **والمتاجرُ مخفيّةٌ عن الزبون بالكامل**: المنصةُ سوقٌ يجلب منها، **وهو
	// يشتري «من رحّال» لا «من مطعم فلان».** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
	//
	// **وحقلٌ يصل المتصفّحَ يُقرأ في أدوات المطوّر**: حجبٌ في الشاشة ولا يفرضه
	// المحرّك ليس حجباً — **وهي عائلةُ الخلل التي تكرّرت في هذه المنصة.**

	// رقم الدعم يُحمَّل مع الصفحة الأولى لا بنداءٍ ثانٍ: هو سطرٌ واحد في
	// التذييل، ونداءٌ مستقلٌّ له تكلفةُ رحلةٍ كاملة لسطر.
	//
	// وكان الرقم لا وجود له أصلاً: الشكوى تذهب إلى التذاكر وحدها، ومن لا يعرف
	// التذاكر لا يجد باباً. ويبقى فارغاً حتى يكتبه المالك، فتُخفيه الواجهة.
	// **والأقسامُ تُرسل مع الرئيسية.**
	//
	// **ونداءٌ ثانٍ من الصفحة الأولى نداءٌ يُرى تأخيراً**: الرئيسيةُ تُقدَّم من
	// الخادم، **فما لم يصل معها يظهر بعد ومضةٍ فارغة.**
	// **ومن مصدرٍ واحدٍ مع نقطة الأقسام** — لا باستعلامٍ ثانٍ يشبهه.
	//
	// كان مكتوباً هنا بيده، **فأُضيفت صورةُ القسم في تلك ولم تُضف في هذه**:
	// نقطةُ الأقسام تُخرجها والرئيسيةُ لا. **والرئيسيةُ هي ما يفتحه الزبون**،
	// فبقيت الصورُ لا تظهر بعد أن رُفعت وأُصلحت روابطُها.
	sections, err := s.publicSections(r)
	if err != nil {
		sections = []publicSection{}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"banners": active, "categories": categories,
		"sections":      sections,
		"support_phone": s.settings.GetString(r.Context(), "platform.support_phone"),

		// **وإعداداتُ جولةِ الأقسام تصل مع الصفحة.**
		//
		// **ونداءٌ ثانٍ لمفتاحين تأخيرٌ يُرى**: صفحةُ التسوّق تجلب هذه الردّةَ
		// أصلاً، **والجولةُ تبدأ مع أوّل رسم** — فلو انتظرت نداءً ثانياً
		// لَبدأت ساكنةً ثمّ تحرّكت فجأة.
		//
		// **والثواني تُحوَّل إلى ملّي هنا لا في الشاشة**: الإعدادُ يُقرأ
		// بالثانية لأنّ من يضبطه إنسان، **والمؤقّتُ يعمل بالملّي** — والتحويلُ
		// في موضعٍ واحدٍ لا في كلّ من يقرؤه.
		"rail_auto":     s.settings.GetBool(r.Context(), "shop.rail_auto"),
		"rail_every_ms": s.settings.GetInt(r.Context(), "shop.rail_seconds") * 1000,

		// **وهويّةُ المنصة تصل مع الصفحة الأولى.**
		//
		// **والاسمُ فارغٌ يعني «خذ من المعجم»** — لا يُفرض على المالك أن
		// يملأه ليعمل الموقع.
		//
		// **والشعارُ مسارٌ لا معرّف**: الشاشةُ ترسم صورةً، **ومن أرسل إليها
		// معرّفاً أجبرها على نداءٍ ثانٍ لتعرف أين هي.**
		"platform_name": s.settings.GetString(r.Context(), "platform.name"),
		"platform_logo": s.platformLogo(r),
	})
}

// **نقطةُ «متجرٌ واحدٌ بقائمته» حُذفت.**
//
// كانت تردّ اسمَ المتجر ووصفَه وشعارَه وقائمتَه كاملةً لأيّ زائرٍ يعرف
// المعرّف — **بلا حسابٍ ولا حدّ.** وصفحتُها (`‎/m/{id}`) حُذفت معها.
//
// **والمتاجرُ مخفيّةٌ عن الزبون بالكامل**: المنصةُ سوقٌ يجلب منها، **وهو
// يشتري «من رحّال» لا «من مطعم فلان»** — يتصفّح أقساماً وأصنافاً، **ولا
// شاشةَ في المنصة تربط إلى متجرٍ بعينه.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
//
// **ونقطةٌ بلا شاشةٍ تبقى مفتوحة**: من قرأ معرّفَ متجرٍ يوماً فتحها من
// الطرفيّة — **والحجبُ الذي يتوقّف عند الشاشة ليس حجباً.**
//
// **ومن يحتاج القائمةَ يقرؤها من قسمِها**: `‎/public/sections/{id}/items`
// **تُخرج الأصنافَ بلا مصدرِها** — وهي ما يتصفّحه الزبونُ أصلاً.

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
	// **وطلبٌ كبيرٌ لا ينتظر يداً** — يُحوَّل تلقائياً إن بلغ العتبة، **ولا
	// يُوسَم مُرسَلاً ما لم يُرسَل.** (انظر `auto_transfer.go`)
	//
	// **وبعد الردّ لا قبله**: التحويلُ يُبلّغ طرفاً خارجياً وقد يتعثّر،
	// **وزبونٌ ينتظر شاشتَه بينما نُرسل رسالةً إلى مطعمٍ يقرأ بطئاً لا نجاحاً.**
	go s.autoTransfer(context.WithoutCancel(r.Context()), o.ID, userIDFrom(r))

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
	redactAllForCustomer(res.Orders)

	// **ومهلةُ الإلغاء مع كلّ طلبٍ في القائمة.**
	//
	// كانت تُرسَل في صفحة الطلب وحدَها — **وقد حُذفت**، وصار الإلغاءُ في
	// البطاقة. **وزرٌّ بلا مهلةٍ إمّا يظهر دائماً فيعتذر، أو لا يظهر أبداً
	// فيُحبس الزبونُ في طلبٍ لم يبدأ.**
	//
	// **والرقمُ من الخادم لا من حسابٍ في الشاشة**: المهلةُ إعدادٌ يملك المالكُ
	// تغييرَه، **ورقمٌ محسوبٌ في المتصفّح يخالفه بعد أوّل تعديل.**
	type withWindow struct {
		orders.Order
		CancelSecondsLeft int `json:"cancel_seconds_left"`
	}
	out := make([]withWindow, len(res.Orders))
	for i := range res.Orders {
		out[i] = withWindow{
			Order:             res.Orders[i],
			CancelSecondsLeft: s.orders.CancelSecondsLeft(r.Context(), &res.Orders[i]),
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"orders": out, "total": res.Total, "page": res.Page, "per_page": res.PerPage,
	})
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
	redactForCustomer(o)
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
