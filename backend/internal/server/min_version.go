package server

import (
	"net/http"
	"strconv"
	"strings"
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
// # ولا تُغلق أبوابُ النجاة
//
// **والتحديثُ يحتاج شبكةً وقائمةً** — فمن حُبس على شاشةِ تحديثٍ وهو لا
// يستطيع التحديثَ حُبس بلا مخرج. **فالأبوابُ العامّةُ تبقى مفتوحةً**:
// الفهرسُ والإعداداتُ وصفحاتُ النظام، **وتُغلق أبوابُ العمل وحدَها.**
//
// # وصفرٌ يعني لا إجبار
//
// **والافتراضُ صفرٌ** — فلا تُغلق بوّابةٌ حتّى يُرفع الرقمُ بيد.
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

		key := ""
		switch {
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-customer"):
			key = "app.min_version.customer"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-driver"):
			key = "app.min_version.driver"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-merchant"):
			key = "app.min_version.merchant"
		case strings.HasSuffix(r.Header.Get(clientKindHeader), "-rep"):
			key = "app.min_version.rep"
		default:
			next.ServeHTTP(w, r)
			return
		}

		want := int(s.settings.GetInt(r.Context(), key))
		if want <= 0 || have >= want {
			next.ServeHTTP(w, r)
			return
		}

		// **وأبوابُ النجاة تبقى** — انظر الشرحَ أعلاه.
		p := r.URL.Path
		if strings.HasPrefix(p, "/api/v1/public/") ||
			strings.HasPrefix(p, "/health") ||
			strings.HasPrefix(p, "/media/") {
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
