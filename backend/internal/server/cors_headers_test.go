package server

// ══════════════════════════════════════════════════════════════════════
// **ترويساتُ العميل ⊆ ترويساتُ CORS المسموحة** (عطبُ إنتاجِ ٢٠٢٦-٠٩-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع
//
// **الإنتاجُ عبرَ أصلين**: الموقعُ `rahalgo.com` والمحرّكُ
// `api.rahalgo.com`. **والنداءُ المُعادُ بإثبات التأكيد يحمل
// `X-Step-Up`** — **وهي لم تكن في المسموح.**
//
// **فردَّ الفحصُ المبدئيُّ ٢٠٠ بلا أيّ ترويسةِ إذن**، فحجب المتصفّحُ
// الإعادةَ **قبل إرسالها**:
//
//	POST /admin/step-up ⇒ 200 · منحةٌ صدرت · consumed_at فارغ
//	OPTIONS /admin/roles ⇒ 200 · **ولا POST بعدها**
//	وقرأ المالكُ «حدث خطأ غير متوقع» — لأنّ `fetch` رُفض بـ`TypeError`
//	لا بـ`ApiError`، فلا `message_key` يُترجَم.
//
// # ولماذا لم يُكشف
//
// **التجهيزُ أصلٌ واحد** (الموقعُ والمحرّكُ على `:8080` نفسِه) — **فـCORS
// لا يعمل فيه إطلاقاً.** **فصنفُ العطب هذا غيرُ مرئيٍّ على التجهيز**،
// ولا يكشفه متصفّحٌ هناك ولو مشى التدفّقَ كلَّه. **وقد مشاه.**
//
// # وما يُحرَس
//
//	١ · فحصٌ مبدئيٌّ حقيقيٌّ بأصل الإنتاج يمرّ ويأذن بـ`X-Step-Up`
//	٢ · وأصلٌ غيرُ مصرَّحٍ به لا يُؤذَن له
//	٣ · **وكلُّ ترويسةٍ يرسلها عميلُ الويب مسموحةٌ** — والقائمةُ تُقرأ من
//	    شيفرة الويب نفسِها لا تُكتب هنا
//
// **والثالثُ هو الذي يمنع تكرارَ هذا**: ترويسةٌ تُضاف في الويب غداً بلا
// إذنٍ في المحرّك تُسقط البناء.

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
)

const prodOrigin = "https://rahalgo.com"

// corsProbe **فحصٌ مبدئيٌّ على الوسيط الحقيقيّ.**
//
// **ولا تُقرأ القائمةُ من الحقل** — **يُركَّب الوسيطُ ويُسأل كما يسأله
// المتصفّح.** فلو تبدّل ترتيبُ الوسائط أو أُسقط أحدُها ظهر هنا.
func corsProbe(t *testing.T, origin, reqHeaders string) *http.Response {
	t.Helper()
	s := srvWith("production", []string{prodOrigin, "https://www.rahalgo.com"})
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.allowedOrigins(),
		AllowedMethods:   corsMethods,
		AllowedHeaders:   corsAllowedHeaders,
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Post("/api/v1/admin/roles", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(201) })

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/admin/roles", nil)
	req.Header.Set("Origin", origin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	if reqHeaders != "" {
		req.Header.Set("Access-Control-Request-Headers", reqHeaders)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec.Result()
}

// TestCORS_ProductionPreflightAllowsStepUp **الفحصُ المبدئيُّ يأذن.**
func TestCORS_ProductionPreflightAllowsStepUp(t *testing.T) {
	res := corsProbe(t, prodOrigin, "content-type,authorization,x-step-up")
	if res.StatusCode >= 400 {
		t.Fatalf("**الفحصُ المبدئيُّ رُدّ** — %d", res.StatusCode)
	}
	origin := res.Header.Get("Access-Control-Allow-Origin")
	if origin != prodOrigin {
		t.Errorf("**لا إذنَ لأصل الإنتاج** — %q (والمنتظَرُ %q)", origin, prodOrigin)
	}
	allow := strings.ToLower(res.Header.Get("Access-Control-Allow-Headers"))
	if allow == "" {
		t.Fatalf("**جوابٌ بلا `Access-Control-Allow-Headers`** — " +
			"**والمتصفّحُ يحجب النداءَ قبل إرساله.** (عطبُ ٢٠٢٦-٠٩-١٢.)")
	}
	if !strings.Contains(allow, "x-step-up") {
		t.Errorf("**`X-Step-Up` غيرُ مأذونٍ بها** — %q — "+
			"**فتُحجب إعادةُ النداء بإثبات التأكيد**، "+
			"**ومنحةٌ تصدر ولا تُستهلك.**", allow)
	}
}

// TestCORS_UnapprovedOriginGetsNoPermission **وأصلٌ غريبٌ لا يُؤذَن له.**
func TestCORS_UnapprovedOriginGetsNoPermission(t *testing.T) {
	for _, bad := range []string{"https://evil.example", "http://rahalgo.com", "https://rahalgo.com.evil.example"} {
		res := corsProbe(t, bad, "content-type,authorization,x-step-up")
		got := res.Header.Get("Access-Control-Allow-Origin")
		if got == bad || got == "*" {
			t.Errorf("**أُذن لأصلٍ غيرِ مصرَّحٍ به**: %s ⇒ %q", bad, got)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
//  **والعقدُ طرفان** — تُقرأ ترويساتُ الويب من شيفرته
// ══════════════════════════════════════════════════════════════════════

// webHeaderRe **ترويسةٌ تُضبَط في شيفرة الويب.**
//
// **وثلاثةُ أشكالٍ يستعملها العميل**: `headers.set("X", …)` ·
// `{ "X": v }` في كائنِ ترويسات · و`X: v` بلا قوسين.
var webHeaderRe = regexp.MustCompile(`(?:headers\.set\(\s*"([A-Za-z][A-Za-z0-9-]*)"|"(X-[A-Za-z0-9-]+)"\s*:|\b(Authorization|Content-Type)\s*:)`)

// TestCORS_WebClientHeadersAreAllowed **كلُّ ما يرسله الويبُ مأذونٌ به.**
//
// **وهذا هو الحارسُ الذي كان غائباً**: ترويسةٌ تُضاف في الويب بلا إذنٍ في
// المحرّك **تُسقط البناء** بدلاً من أن تُكسر اللوحةَ في الإنتاج وحدَه.
func TestCORS_WebClientHeadersAreAllowed(t *testing.T) {
	roots := []string{
		filepath.Join("..", "..", "..", "web", "packages", "auth", "src"),
		filepath.Join("..", "..", "..", "web", "packages", "ui", "src"),
		filepath.Join("..", "..", "..", "web", "apps", "rahalgo", "src"),
	}
	found := map[string]string{} // ترويسة ⇒ الملفّ
	for _, root := range roots {
		if err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil //nolint:nilerr // شجرةٌ ناقصةٌ تُكشف أدناه
			}
			if !strings.HasSuffix(p, ".ts") && !strings.HasSuffix(p, ".tsx") {
				return nil
			}
			b, err := os.ReadFile(filepath.Clean(p))
			if err != nil {
				return nil //nolint:nilerr
			}
			// **ويُطوى التعليقُ قبل الفحص** — **وشرحٌ يذكر ترويسةً ليس
			// إرسالاً لها.**
			src := stripTSComments(string(b))
			for _, m := range webHeaderRe.FindAllStringSubmatch(src, -1) {
				for _, g := range m[1:] {
					if g == "" {
						continue
					}
					// **وترويسةُ جوابٍ ليست ترويسةَ طلب** — `Cache-Control`
					// يُضبَط في مسارٍ يخدم، لا في نداءٍ يخرج.
					if strings.EqualFold(g, "Cache-Control") {
						continue
					}
					if _, seen := found[strings.ToLower(g)]; !seen {
						found[strings.ToLower(g)] = p
					}
				}
			}
			return nil
		}); err != nil {
			t.Fatalf("تعذّر مشيُ شجرة الويب %s: %v", root, err)
		}
	}
	if len(found) == 0 {
		t.Fatal("**لم تُقرأ ترويسةٌ واحدةٌ من شيفرة الويب** — والحارسُ بلا مرجعٍ لا يقيس شيئاً")
	}

	allowed := map[string]bool{}
	for _, h := range corsAllowedHeaders {
		allowed[strings.ToLower(h)] = true
	}
	names := make([]string, 0, len(found))
	for h := range found {
		names = append(names, h)
	}
	sort.Strings(names)
	t.Logf("ترويساتٌ يرسلها الويب = %d: %s", len(names), strings.Join(names, " · "))
	for _, h := range names {
		if !allowed[h] {
			t.Errorf("**ترويسةٌ يرسلها الويبُ ولا يأذن بها CORS**: %q (%s) — "+
				"**يردّ الفحصُ المبدئيُّ بلا إذنٍ فيحجب المتصفّحُ النداءَ قبل إرساله**، "+
				"**ولا يظهر ذلك على التجهيز لأنّه أصلٌ واحد.**", h, found[h])
		}
	}

	// **وكلُّ واحدةٍ تُسأل في فحصٍ مبدئيٍّ حقيقيّ** — لا في القائمة وحدَها.
	for _, h := range names {
		res := corsProbe(t, prodOrigin, h)
		got := strings.ToLower(res.Header.Get("Access-Control-Allow-Headers"))
		if !strings.Contains(got, h) {
			t.Errorf("**الفحصُ المبدئيُّ لا يأذن بـ%q** — الجواب %q", h, got)
		}
	}
}

// stripTSComments **يطوي تعليقَ TypeScript.**
func stripTSComments(src string) string {
	var out strings.Builder
	for i := 0; i < len(src); i++ {
		if src[i] == '/' && i+1 < len(src) {
			switch src[i+1] {
			case '/':
				for i < len(src) && src[i] != '\n' {
					i++
				}
				out.WriteByte('\n')
				continue
			case '*':
				if j := strings.Index(src[i+2:], "*/"); j >= 0 {
					i += 2 + j + 1
					continue
				}
				return out.String()
			}
		}
		out.WriteByte(src[i])
	}
	return out.String()
}
