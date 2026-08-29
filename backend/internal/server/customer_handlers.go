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
	return s.settingMedia(r, "platform.logo")
}

// settingMedia مسارُ وسيطٍ مخزَّنٍ في الإعدادات — **واحدةٌ لكلّ مفتاحٍ من نوع
// `media`.** كانت خاصّةً بالشعار، **فلمّا جاءت خلفيّةُ الدخول كان الحلُّ
// نسخَها باسمٍ ثانٍ** — ونسختان تفترقان بلا صوت.
func (s *Server) settingMedia(r *http.Request, key string) *string {
	id := s.settings.GetString(r.Context(), key)
	if id == "" {
		return nil
	}
	var path *string
	if s.pg.QueryRow(r.Context(), `SELECT path FROM media WHERE id = $1`, id).Scan(&path) != nil {
		return nil
	}
	return media.URLForPtr(path)
}

// settingMediaBlur **لمحةُ وسيطٍ مخزَّنٍ في الإعدادات** — وفارغٌ يعني
// «لا لمحة»: **صفوفٌ رُفعت قبل أن تُولَّد اللمحاتُ تبقى تعمل بلا واحدة.**
func (s *Server) settingMediaBlur(r *http.Request, key string) string {
	id := s.settings.GetString(r.Context(), key)
	if id == "" {
		return ""
	}
	var blur string
	if s.pg.QueryRow(r.Context(),
		`SELECT COALESCE(blur, '') FROM media WHERE id = $1`, id).Scan(&blur) != nil {
		return ""
	}
	return blur
}

// handlePublicPlatform هويّةُ المنصة — **الاسمُ والشعارُ من الإعدادات وحدَها.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا أريد أيَّ مكانٍ يُكتب فيه اسم المنصة بشكلٍ
//
//	جامد، يجب أن يأتي من الإعدادات فقط».)
//
// # ولماذا نقطةٌ مستقلّةٌ ولم تكفِ `/public/home`
//
// **الهويّةُ كانت تُسلَّم في `/public/home` وحدَها** — وتلك تجلب اللافتاتِ
// والأقسامَ بعددِ أصنافها، **ثلاثةُ استعلاماتٍ لِمن أراد اسماً وصورة.**
//
// **ولوحاتُ التحكّم لا تناديها أصلاً**: لا لافتاتٍ في لوحة الإدارة ولا
// أقسامَ سوقٍ في بوّابة السائق. **فبقيت اللوحاتُ الأربعُ تكتب اسمَها في
// شيفرتها** — وأربعةُ أسماءٍ مختلفة: «رحّال غو» و«بوابة المتجر» وعنوانا
// دخولِ السائق والمندوب.
//
// # ومفتوحةٌ بلا توثيق
//
// **شاشةُ الدخول تحتاجها قبل أن يكون هناك حساب** — وهي أوّلُ ما يُرى.
func (s *Server) handlePublicPlatform(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, map[string]any{
		"name": s.settings.GetString(r.Context(), "platform.name"),
		"logo": s.platformLogo(r),
		// **وما تحتاجه الشاشةُ قبل أن يكون هناك حساب** — لا الهويّةَ وحدَها.
		// **وشاشةُ الدخول لا تعرف أيَّ أبوابٍ تعرض حتّى تسأل**، ونداءٌ ثانٍ
		// لسطرٍ واحدٍ رحلةٌ زائدةٌ في أوّل ما يُفتح.
		"otp_login": s.settings.GetBool(r.Context(), "auth.otp_login"),
		// **وأيُطلب رمزٌ عند إنشاء الحساب؟** — (قرارُ المالك 2026-08-25).
		//
		// **وتطبيقٌ يعرض خطوةَ رمزٍ والمنصّةُ أطفأتها يحبس صاحبَه على
		// شاشةٍ لا مخرجَ منها** — كما وقع في تبويب الرمز قبله.
		"signup_verify": s.settings.GetBool(r.Context(), "auth.signup_verify"),
		// **وبابا الموقع** — (طلبُ المالك ٢٠٢٦-٠٨-١٧).
		//
		// **ومعهما لا في نداءٍ ثانٍ**: الشريطُ العلويُّ يُرسم في كلّ صفحة،
		// **ونداءٌ ثانٍ لسطرين يجعل الزرَّ يظهر ثمّ يختفي** أمام عين
		// الزائر — وهو أسوأُ من ظهوره.
		"show_login": s.settings.GetBool(r.Context(), "site.show_login"),
		"show_shop":  s.settings.GetBool(r.Context(), "site.show_shop"),
		// **وبابُ الانضمام** — (قرارُ المالك ٢٠٢٦-٠٨-١٧). **ومغلقٌ حتّى
		// يُفتح**: بابٌ يستقبل طلباتٍ لا أحدَ يراجعها أسوأُ من بابٍ مغلق.
		"join_open": s.settings.GetBool(r.Context(), "site.join_open"),
		// **وطولُ كلمة المرور — رقمٌ واحدٌ تقرؤه كلُّ شاشةٍ تسأل عنها.**
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «طولُ كلمة المرور يجب أن تكون موحّدةً
		//  بكلّ البرنامج».)
		//
		// **وكان مكتوباً ستَّ مرّاتٍ في الويب** — بوّابةُ الكلمة والتسجيلُ
		// وإنشاءُ المتجر وثلاثةُ حقولٍ في الإدارة. **ورقمٌ يُكرَّر ستّاً
		// يفترق**، والمحرّكُ يرفع الحدَّ فتقبل الشاشةُ ما يردّه هو —
		// **فيُرفض المستخدمُ بعد أن قيل له إنّ كلمتَه صالحة.**
		"password_min_length": s.minPasswordLen(r.Context()),
		// **ورابطُ التطبيق هنا لا في نداءٍ ثانٍ** — زرُّه بجانب زرِّ الدخول،
		// **فيُقرأ مع ما تُقرأ به الشاشةُ أوّلَ مرّة.**
		"app_url": s.appHref(r),
		// **ورقمُ الدعم مع الهويّة لا في نداءٍ ثالث.**
		//
		// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «بالفاتورة لازم نضيف رقم هاتف الدعم
		//  للشكاوى».)
		//
		// **الفاتورةُ تُطبع وتُسلَّم ثمّ تُقرأ بعيداً عن التطبيق** — ومن وجد
		// فيها خطأً لا يجد أين يشتكي. **ورقمٌ على الورق هو البابُ الوحيد
		// حينَها.**
		//
		// وكان في `/public/contact` وحدَه — **نداءٌ لا تعرفه الورقة**، فيُضمّ
		// إلى ما تقرؤه كلُّ لوحةٍ أوّلَ مرّة.
		"support_phone": s.settings.GetString(r.Context(), "platform.support_phone"),
		// **وحساباتُ التواصل مع الهويّة** — التذييلُ يُرسم في كلّ صفحة،
		// **ونداءٌ ثانٍ له رحلةٌ في كلّ فتحة.**
		// **وعنوانُ المكتب وموقعُه** — صفحةُ التواصل ترسمهما.
		"address":  s.settings.GetString(r.Context(), "platform.address"),
		"location": s.settings.GetString(r.Context(), "platform.location"),
		"social": map[string]string{
			"facebook":  s.settings.GetString(r.Context(), "platform.facebook"),
			"instagram": s.settings.GetString(r.Context(), "platform.instagram"),
			"telegram":  s.settings.GetString(r.Context(), "platform.telegram"),
			"whatsapp":  s.settings.GetString(r.Context(), "platform.whatsapp"),
		},
		// **وصورتان لكلّ خلفيّة** — عريضةٌ للشاشة وطوليّةٌ للجوّال.
		// (قرارُ المالك 2026-08-09.) **وفارغةُ الجوّال تسقط إلى العريضة.**
		"auth_bg":        s.settingMedia(r, "auth.background"),
		"auth_bg_mobile": s.settingMedia(r, "auth.background_mobile"),
		"site_bg":        s.settingMedia(r, "platform.background"),
		"site_bg_mobile": s.settingMedia(r, "platform.background_mobile"),
		"site_bg_dim":    s.settings.GetInt(r.Context(), "platform.background_dim"),
		// **ولمحةُ الخلفيّة معها** — (طلبُ المالك ٢٠٢٦-٠٨-١٧):
		// **تُرسم لوناً في أوّل رسمةٍ ثمّ تحلّ الصورةُ محلَّها**، فلا
		// يُرى تدرّجٌ عارٍ ثمّ تقفز الصورةُ فوقه.
		"site_bg_blur": s.settingMediaBlur(r, "platform.background"),
		"auth_bg_dim":  s.settings.GetInt(r.Context(), "auth.background_dim"),
	})
}

// handlePublicBanners **لافتاتُ موضعٍ بعينه — للرئيسيّة.**
//
// (تصحيحُ المالك ٢٠٢٦-٠٨-١٧: «بانرات صفحة التسوّق مختلفة برأيي عن
//
//	الرئيسيّة».)
//
// **ونقطةٌ خفيفةٌ لا `/public/home`**: تلك تجلب التصنيفاتِ والأقسامَ بعددِ
// أصنافها — **ثلاثةُ استعلاماتٍ لصفحةٍ لا تريد إلّا صوراً.**
//
// **ومفتوحةٌ بلا توثيق** — الرئيسيّةُ تُفتح قبل أن يكون حساب.
func (s *Server) handlePublicBanners(w http.ResponseWriter, r *http.Request) {
	at := r.URL.Query().Get("at")
	if at != "shop" && at != "home" {
		at = "home"
	}
	banners, err := s.catalog.ListBanners(r.Context(), at)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والمطفأةُ وبلا صورةٍ لا تُرسل** — **ولافتةٌ بلا صورةٍ فراغٌ في
	// سلايدر**، والزائرُ يسحب فلا يجد شيئاً.
	active := banners[:0]
	for _, b := range banners {
		if b.Active && b.ImageURL != nil {
			active = append(active, b)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"banners": active})
}

// handlePublicHome بيانات الصفحة الأولى: لافتاتٌ وتصنيفاتٌ وأقسامُ سوق.
//
// **ولا متاجرَ فيها** — انظر الشرحَ عند الأقسام أدناه.
func (s *Server) handlePublicHome(w http.ResponseWriter, r *http.Request) {
	// **وعدّادُ الفتحات هنا** — هذه أوّلُ نقطةٍ يناديها التطبيقُ عند
	// الإقلاع، **فهي أقربُ ما يكون إلى «فُتح التطبيق».** انظر
	// `app_opens.go`.
	s.countOpen(r)

	// ══════════════════════════════════════════════════════════════════
	// **وسلايدرٌ واحدٌ للمنصّة كلِّها**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٨: «أريد حذفَ سلايدر التسوّق وربطَ التطبيق
	//  بسلايدر الرئيسيّة».)
	//
	// **فصلَهما ٢٠٢٦-٠٨-١٧ ثمّ قاسه بالعمل**: يرفع الصورةَ مرّتين
	// ويحذفها مرّتين، **ولافتةٌ تُحدَّث في موضعٍ وتُنسى في آخرَ تُقرأ
	// عرضاً منتهياً في نصف المنصّة.**
	//
	// **وهذه النقطةُ يقرؤها التطبيقُ وصفحتا التسوّق والعروض** — فصارت
	// كلُّها ترى ما تراه الرئيسيّة.
	banners, err := s.catalog.ListBanners(r.Context(), "home")
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
		// **ومهلةُ السلايدر كمهلة الجولة** — تصل مع الصفحة لا بنداءٍ ثانٍ.
		// **ومهلةُ الرئيسيّة هي المهلة** — سلايدرٌ واحدٌ ومفتاحٌ واحد:
		// **مفتاحان لدورانٍ واحدٍ يفترقان فيدور في شاشةٍ ويسكن في مثلها.**
		"banner_auto":     s.settings.GetBool(r.Context(), "home.banner_auto"),
		"banner_every_ms": s.settings.GetInt(r.Context(), "home.banner_seconds") * 1000,
		"rail_auto":       s.settings.GetBool(r.Context(), "shop.rail_auto"),
		"rail_every_ms":   s.settings.GetInt(r.Context(), "shop.rail_seconds") * 1000,

		// **وهويّةُ المنصة تصل مع الصفحة الأولى.**
		//
		// **والاسمُ فارغٌ يعني «خذ من المعجم»** — لا يُفرض على المالك أن
		// يملأه ليعمل الموقع.
		//
		// **والشعارُ مسارٌ لا معرّف**: الشاشةُ ترسم صورةً، **ومن أرسل إليها
		// معرّفاً أجبرها على نداءٍ ثانٍ لتعرف أين هي.**
		// **ولا هويّةَ هنا** — لها نقطتُها (`/public/platform`)، **وحقلٌ يُسلَّم
		// في موضعين يفترق بلا صوت.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)
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
	// ══════════════════════════════════════════════════════════════════
	// **والأجرةُ من الإعدادات — لا من عمود المنطقة الميّت**
	// ══════════════════════════════════════════════════════════════════
	//
	// (شهده المالك ٢٠٢٦-٠٨-١١: «يقول التوصيلُ مجّانيٌّ وأنا حاطط أجور ١٠
	//  آلاف».)
	//
	// **`delivery_zones.delivery_fee` عمودٌ لم يعد يُقرأ** — بقي في القاعدة
	// بعد أن صارت الأجرةُ رقماً مقطوعاً واحداً في الإعدادات (قرارُ المالك
	// ٢٠٢٦-٠٨-٠٤: «قيمُ التوصيل يجب أن تأتي من مكانٍ واحدٍ بكلّ المشروع»).
	//
	// **وهذه النقطةُ وحدَها بقيت تقرؤه** — فتردّ صفراً أبداً، **وشاشةُ السلّة
	// تكتب «مجّاني» فوق أجرةٍ مضبوطة.** ومن رآها ظنّ الإعدادَ لا يعمل.
	//
	// **وأخطرُ ما فيه أنّ الطلبَ يُحاسَب بالصحيح**: الزبونُ يقرأ «مجّاني»
	// ويُخصم منه، **وهو أسوأُ ما يقع في شاشة دفع.**
	//
	// **و`DeliveryAt` هي مصدرُ الحقيقة** — تناديها التسعيرةُ والإنشاء معاً.
	d, err := s.orders.DeliveryAt(r.Context(), lat, lng)
	if err != nil {
		s.respondErr(w, orders.ErrOutOfZone)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"name": d.Name, "delivery_fee": d.Fee, "min_order": d.MinOrder,
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
		// **ومسارُه بأوقاته** — (قرارُ المالك ٢٠٢٦-٠٨-١٢): يعرف متى
		// قُبل ومتى استلمه سائقُه ومتى وصل، **بلا أن يسأل أحداً.**
		Timeline []TimelineStep `json:"timeline"`
	}{
		o,
		s.orders.CancelSecondsLeft(r.Context(), o),
		s.timeline(r.Context(), o.ID, false),
	})
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

// appHref وجهةُ زرّ «حمّل التطبيق» — رابطٌ أو ملفٌّ أو لا شيء.
//
// (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «إذا كان الموجودُ رابطاً يذهب إلى غوغل بلاي،
//
//	وإذا ملفّاً ينزل بشكلٍ مباشر».)
//
// **والرابطُ يسبق**: من رفع تطبيقَه إلى المتجر فالمتجرُ أولى — يُحدِّث
// نفسَه ويُطمئن من ينزّله. **والملفُّ لمن لم يُقبل بعد.**
//
// **ويُحلّ هنا لا في الواجهة**: خمسُ بوّاباتٍ تعرض الزرّ، **وقاعدةُ
// أولويّةٍ تُكتب خمسَ مرّاتٍ تفترق في الرابعة.**
func (s *Server) appHref(r *http.Request) string {
	if link := s.settings.GetString(r.Context(), "platform.app_url"); link != "" {
		return link
	}
	if s.settings.GetString(r.Context(), appFileSetting) != "" {
		return "/api/v1/public/app"
	}
	return ""
}
