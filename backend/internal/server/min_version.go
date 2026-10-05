package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/obs"
)

// versionHeader **رقمُ نسخة التطبيق** — يرسله كلُّ نداءٍ من أندرويد.
const versionHeader = "X-RahalGo-Version"

// minVersion **بوّابةُ التحديث — تردّ 426 لمن تخلّف.**
//
// (طلبُ المالك 2026-08-25: «مو معقول التطبيق يتحدّث والمستخدم ما عنده
//
//	خبر بالشي».)
//
// # ولماذا في المحرّك لا في التطبيق
//
// **والتطبيقُ القديمُ لا يعرف أنّه قديم** — ومن سأله «أأنت محدَّث؟»
// سأل من لا يملك الجواب. **والمحرّكُ وحدَه يعرف ما نشرناه.**
//
// # و426 لا 403
//
// **`426 Upgrade Required` رمزٌ قياسيٌّ معناه: بدّل ما تكلّمني به.**
// **ولو رُدّ 403 لَخلطه التطبيقُ بمنعِ صلاحيّة** فأخرج صاحبَه من حسابه.
//
// # ولا تُغلق أبوابُ النجاة — وتُغلق أبوابُ التصفّح (٢٠٢٦-١٠-٠٦)
//
// **كانت `‎/api/v1/public/` كلُّها مفتوحةً للقديم** — **فهاتفٌ لم يسجّل
// دخولاً لا ينادي غيرَها**: الزبونُ يتصفّح والسائقُ ينتظر عند شاشة الدخول،
// **ولا يرى أحدُهما شاشةَ التحديث أبداً.** (شكوى المالك ٢٠٢٦-١٠-٠٦: «رفعتُ
// التطبيقات وحدّدتُ الإصدارات وما طلب منّي جوّالي التحديث».)
//
// **وقُرئ التطبيقُ قبل القرار** (`mobile/shared/.../ApiClient.kt`): **أيُّ
// ٤٢٦ من أيِّ نداءٍ — عامٍّ أو موثَّق — يرفع شاشةَ التحديث**، **وشاشةُ
// التحديث لا تنادي المحرّكَ أبداً** (أزرارُها روابطُ متجرٍ وتنزيلٍ ثابتة).
// **وأوّلُ نداءٍ في التطبيقات الأربعة عند الإقلاع `‎/public/platform`.**
//
// **فصار المفتوحُ للقديم قائمةً مغلقةً** (`updateEscapeHatch`): سجلُّ
// التوزيع وملفّاتُ التنزيل والهويّةُ والصحّةُ والوسائط — **ما يحتاجه من
// يريد أن يحدّث من متصفّح.** **وما سواها يُردّ ٤٢٦** — ومنه
// `‎/public/platform` نفسُه: **هو ما يجعل أوّلَ إقلاعٍ لتطبيقٍ قديمٍ يُظهر
// الشاشةَ ولو لم يُسجَّل دخول.** **والويبُ لا يرسل ترويسةَ النسخة فلا يمسّه
// شيء.**
//
// # والحدُّ من الملفّ المرفوع تلقائيّاً
//
// **والحدُّ أكبرُ اثنين** (`effectiveMinVersion`): رقمٌ يدويٌّ
// (`app.min_version.<app>`) **ورقمُ الملفّ المرفوع** حين يكون
// `release.<app>.auto_force` مشغّلاً (الافتراض). **فرفعُ نسخةٍ أحدث يكفي.**
func (s *Server) minVersion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := r.Header.Get(versionHeader)
		if raw == "" {
			// **وما لا يرسل نسخةً ليس تطبيقاً** — لوحةُ الإدارة وأدواتُ
			// الفحص. **ولا تُحبس بما لا يخصّها.**
			next.ServeHTTP(w, r)
			return
		}
		have, err := strconv.Atoi(raw)
		if err != nil || have <= 0 {
			next.ServeHTTP(w, r)
			return
		}

		kind := ""
		switch {
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-customer"):
			kind = "customer"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-driver"):
			kind = "driver"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-merchant"):
			kind = "merchant"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-rep"):
			kind = "rep"
		default:
			next.ServeHTTP(w, r)
			return
		}

		// ══════════════════════════════════════════════════════════════
		// **وأيُّ نسخةٍ تطرق البابَ — تجميعاً** (دورة ٧٠أ)
		// ══════════════════════════════════════════════════════════════
		//
		// **ودورةُ ٦٩ج لم تستطع أن تقول أيُّ تطبيقٍ يُعيد الوصلَ**، لأنّ
		// سجلَّ البوّابة يحذف الترويسات جملةً **وهو صواب**. **فحُسم
		// الاستدلالُ من إيقاعٍ لا من قياس.**
		//
		// **والترويستان مقروءتان هنا أصلاً لأجل بوّابة التحديث** —
		// **فالعدُّ لا يفتح باباً ولا يقرأ ما لم يكن مقروءاً.**
		//
		// **ولا جهازَ ولا إنسان**: «زبونٌ نسخةُ ١١» رقمٌ للمنصّة كلِّها.
		obs.Client(kind, have)

		want := int(s.effectiveMinVersion(r.Context(), kind))
		if want <= 0 || have >= want {
			next.ServeHTTP(w, r)
			return
		}

		// **وأبوابُ النجاة تبقى** — انظر الشرحَ أعلاه و`updateEscapeHatch`.
		if updateEscapeHatch(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusUpgradeRequired)
		_, _ = w.Write([]byte(
			`{"error":{"code":"update_required","message":"errors.update_required"},` +
				`"min_version":` + strconv.Itoa(want) + `}`))
	})
}

// updateEscapeHatch **ما يبقى مفتوحاً لتطبيقٍ قديم** — قائمةٌ مغلقة.
//
// **ومن أضاف باباً عامّاً جديداً أُغلق على القديم تلقائيّاً** — وهو الصواب:
// **تطبيقٌ قديمٌ يتصفّح بعقدٍ تبدّل ينكسر بصمت.**
func updateEscapeHatch(p string) bool {
	switch {
	case p == "/api/v1/public/releases",
		p == "/api/v1/public/identity",
		p == "/api/v1/public/app",
		strings.HasPrefix(p, "/api/v1/public/app/"),
		strings.HasPrefix(p, "/health"),
		strings.HasPrefix(p, "/media/"):
		return true
	}
	return false
}
