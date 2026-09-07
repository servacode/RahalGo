package server

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **القرارُ واحدٌ ومركزيّ** — `ADG-1` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # السؤالُ الكانونيّ
//
//	أيملك هذا الفاعلُ هذه القدرة؟
//
// **لا «أدورُه `admin`؟»** — **فمن أراد أن يمنح صلاحيّةً واحدةً منح
// دوراً كاملاً، ومن أراد أن يمنع واحدةً نزع الدورَ كلَّه.**
//
// # ومصدرُ الحقيقة
//
// **القاعدةُ في اللحظة**: أدوارُ الحساب × قدراتُ أدواره —
// **تركب استعلامَ `R16` القائم فلا رحلةَ ثانية** (`SessionRows`).
//
// **ولا ادّعاءَ في رمزٍ يُصدَّق** (`R15`): **سحبُ دورٍ أو نزعُ قدرةٍ
// يسري عند أوّل طلب.**
//
// # والافتراضُ منع
//
// **مجهولُ القدرة يُمنَع** — **فخطأٌ مطبعيٌّ في اسمِ قدرةٍ يُغلق
// البابَ ولا يفتحه.**
//
// **ودورٌ بلا صفوفٍ في `role_capabilities` لا يملك شيئاً** — **ولا
// «كلُّ من دخل بابَ الإدارة يمرّ».**

// errForbiddenCap **مُنع لأنّه لا يملك القدرة** — لا لأنّه مجهول.
//
// **ولا يُقال أيُّ قدرةٍ نقصته**: **من عرف اسمَ القدرة عرف شكلَ
// المعجم** — ويُقرأ الاسمُ في السجلّ لا في الردّ.
var errForbiddenCap = httpx.NewError(http.StatusForbidden,
	"forbidden", "errors.forbidden")

// RequireCapability **الحارسُ الوحيدُ للمسارات المُرحَّلة.**
//
// **ويُنادى بعد `RequireAuth`** — فالقدراتُ توضَع في السياق هناك من
// الحقيقة الموثوقة.
func (s *Server) RequireCapability(need authz.Capability) func(http.Handler) http.Handler {
	// ══════════════════════════════════════════════════════════════
	// **وقدرةٌ غيرُ مسجَّلةٍ تُوقف الإقلاع** — لا تُمرَّر بصمت
	// ══════════════════════════════════════════════════════════════
	//
	// **وحارسٌ يقع عند بناء المسارات** — **فمن كتب اسماً خطأً عرف
	// في الحال، ولا يُكتشَف يومَ يُمنَع موظّفٌ من عملٍ مشروع.**
	if !authz.Known(need) {
		panic("authz: قدرةٌ غيرُ مسجَّلةٍ في المعجم: " + string(need))
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.hasCapability(r, need) {
				s.logger.Info("التخويل: مُنع",
					"capability", string(need),
					"user", userIDFrom(r),
					"roles", rolesFrom(r))
				httpx.Error(w, errForbiddenCap)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// hasCapability **القرارُ نفسُه** — يُنادى من الوسيط ومن البثّ.
//
// **ولا يقرأ دوراً باسمه** — **القدراتُ محسوبةٌ في الحقيقة الموثوقة.**
func (s *Server) hasCapability(r *http.Request, need authz.Capability) bool {
	if !authz.Known(need) {
		return false
	}
	caps, _ := r.Context().Value(ctxCaps).([]string)
	for _, c := range caps {
		if c == string(need) {
			return true
		}
	}
	return false
}

// capabilitiesFrom قدراتُ الفاعل كما حُسبت — **للفحص والتشخيص.**
func capabilitiesFrom(r *http.Request) []string {
	caps, _ := r.Context().Value(ctxCaps).([]string)
	return caps
}

// settingCapability **قدرةُ الإعداد بحسب أثره** — `ADG-1` · `AQ-1`.
//
// **والتصنيفُ من دورةِ ٢١ لا يُخترَع ثانيةً** (`criticalSettingKey`) —
// **وتصنيفان يفترقان يومَ يُضاف مفتاح.**
//
//	security.*                    ⇒ إعداداتُ الأمن
//	fininv.FinancialSettings      ⇒ إعداداتٌ ماليّة
//	ما سواها                       ⇒ عامّة
func settingCapability(key string) authz.Capability {
	switch {
	case strings.HasPrefix(key, "security."):
		return authz.SettingsSecurityManage
	case criticalSettingKey(key):
		return authz.SettingsFinancialManage
	default:
		return authz.SettingsGeneralManage
	}
}

// RequireAnyCapability **بابُ السطح الإداريّ** — `ADG-1`.
//
// ══════════════════════════════════════════════════════════════════════
// **ولماذا لا يبقى «أحدُ ثلاثةِ أدوار»**
// ══════════════════════════════════════════════════════════════════════
//
// **كان البابُ `RequireRoles("admin","ops","finance")`** — **وهو عينُ
// «كلُّ من دخل بابَ الإدارة يمرّ»** الذي منعه العقد.
//
// **ودورٌ جديدٌ يُنشئه الأدمنُ غداً بقدرةٍ واحدةٍ كان يُردّ عند الباب**
// **مهما مُنح** — **فيصير المنحُ من اللوحة بلا أثر.**
//
// # والشرطُ الآن
//
// **أن يملك قدرةً واحدةً مسجَّلةً على الأقلّ.**
//
// **والأدوارُ القديمةُ تمرّ لأنّها مبذورةٌ بقدراتها** (هجرة `0137`) —
// **لا بمحاباةٍ لاسمها.** **فليست سياسةً انتقاليّةً بل نتيجةُ البذر.**
//
// **ومن لا قدرةَ له يُردّ**: `customer` و`driver` و`merchant` و`sales`
// لا صفَّ لها في `role_capabilities`.
//
// **والبابُ لا يُغني عن حارسِ القدرة داخلَه** — **دخولٌ إلى السطح ليس
// إذناً بفعل.**
func (s *Server) RequireAnyCapability(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(capabilitiesFrom(r)) == 0 {
			s.logger.Info("التخويل: لا قدرةَ لهذا الفاعل",
				"user", userIDFrom(r), "roles", rolesFrom(r))
			httpx.Error(w, errForbiddenCap)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ══════════════════════════════════════════════════════════════════════
// **وسياسةُ السطح الإداريّ تُطبَّق مرّةً واحدة** — `ADG-2`
// ══════════════════════════════════════════════════════════════════════
//
// **ومئةٌ وستّةَ عشرَ مساراً** — **وحارسٌ يُكتب عند كلٍّ منها يُنسى
// عند واحد.** **فجدولٌ مركزيٌّ ومشّاءٌ يُثبت أنّ كلَّ مسارٍ مغطّى.**
//
// **والافتراضُ منع**: مسارٌ لا سياسةَ له ولا استثناءَ ⇒ يُمنَع.

// enforceAdminPolicy وسيطُ السطح الإداريّ.
func (s *Server) enforceAdminPolicy(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pattern := adminPattern(r.URL.Path)

		// **والمستثنى يمضي إلى معالِجه** — وهناك يُحسَم بقدرةٍ أدقّ.
		if _, ok := authz.IsExempt(pattern); ok {
			next.ServeHTTP(w, r)
			return
		}
		need, ok := authz.LookupAdmin(r.Method, pattern)
		if !ok {
			// **مسارٌ إداريٌّ بلا سياسة** — **ولا يُقرأ الصمتُ إذناً.**
			s.logger.Warn("التخويل: مسارٌ إداريٌّ بلا سياسة",
				"method", r.Method, "pattern", pattern)
			httpx.Error(w, errForbiddenCap)
			return
		}
		if !s.hasCapability(r, need) {
			s.logger.Info("التخويل: مُنع",
				"capability", string(need), "pattern", pattern,
				"user", userIDFrom(r), "roles", rolesFrom(r))
			httpx.Error(w, errForbiddenCap)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// adminPattern **يحوّل مساراً حيّاً إلى نمطٍ يطابق الجدول.**
//
// **ولا يُقرأ نمطُ الموجِّه هنا**: **وسيطُ المجموعة يعمل قبل أن
// يستقرّ التوجيهُ إلى الورقة** — **فيُبنى النمطُ من الشكل.**
//
// **والمقاطعُ التي تشبه معرّفاً تصير `{}`**: `uuid` أو رقمٌ أو نصٌّ
// طويلٌ بلا معنىً ثابتٍ في الجدول.
func adminPattern(path string) string {
	const prefix = "/api/v1/admin"
	rest := strings.TrimPrefix(path, prefix)
	segs := strings.Split(strings.Trim(rest, "/"), "/")
	out := make([]string, 0, len(segs))
	for _, s := range segs {
		if s == "" {
			continue
		}
		if looksLikeID(s) {
			out = append(out, "{}")
			continue
		}
		out = append(out, s)
	}
	return "/" + strings.Join(out, "/")
}

// looksLikeID **أهذا المقطعُ معرّفٌ لا اسمُ مورد؟**
//
// **ومقاطعُ الجدول كلُّها كلماتٌ لاتينيّةٌ قصيرةٌ بشرطة** — **والمعرّفُ
// `uuid` أو رقمٌ أو مفتاحُ إعدادٍ فيه نقطة.**
func looksLikeID(s string) bool {
	if s == "" {
		return false
	}
	if strings.Count(s, "-") == 4 && len(s) == 36 {
		return true // uuid
	}
	if strings.ContainsAny(s, ".") {
		return true // مفتاحُ إعدادٍ مثل `merchants.commission_percent`
	}
	for _, r := range s {
		if r < 'a' || r > 'z' {
			if r != '-' {
				return true
			}
		}
	}
	return false
}

// RouterForWalk **الموجِّهُ بنوعه** — **لحارس التغطية** (`ADG-2`).
//
// **و`Router()` تُرجع `http.Handler`** فلا يمشي عليها `chi.Walk`.
// **ولا مسارَ جديدٌ يُفتَح**: دالّةُ بناءٍ تُنادى في الفحص.
func (s *Server) RouterForWalk() chi.Routes {
	h := s.Router()
	if r, ok := h.(chi.Routes); ok {
		return r
	}
	return nil
}
