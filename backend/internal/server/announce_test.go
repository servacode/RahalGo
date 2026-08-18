package server

// **ما تكتبه اللوحةُ يُبَثّ — وما تقرؤه لا.**
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «في مشكلةُ التحديث اللحظيّ بالتطبيق… أيُّ
//
//	تعديلٍ من لوحة الأدمن فوراً يُطبَّق حتّى ولو الزبونُ فاتحٌ التطبيق،
//	ما يلزم يحدّث أو يعيد تشغيل التطبيق».)
//
// **وأخطرُ سطرٍ فيه القراءة**: بثٌّ عند كلّ `GET` يوقظ كلَّ هاتفٍ في
// المنصّة كلَّما فتح موظّفٌ شاشةً — **فيُعاد الجلبُ مئاتِ المرّات في
// الدقيقة**، ويُقرأ التطبيقُ بطيئاً بلا أن يظهر سبب.

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// TestAnnounceWrites يقيس البثَّ من مشتركٍ حقيقيٍّ في المركز.
//
// **ولا مركزَ مزيّفٌ هنا**: المزيّفُ يقيس أنّ الدالّةَ نادت دالّةً،
// **والمشتركُ يقيس أنّ الرسالةَ وصلت** — وهو ما يشتكي المالكُ من غيابه.
func TestAnnounceWrites(t *testing.T) {
	cases := []struct {
		name   string
		method string
		status int
		want   bool
	}{
		{"كتابةٌ ناجحةٌ تُبَثّ", http.MethodPost, http.StatusOK, true},
		{"وتعديلٌ ناجحٌ كذلك", http.MethodPatch, http.StatusNoContent, true},
		{"وحذفٌ ناجحٌ كذلك", http.MethodDelete, http.StatusOK, true},
		{"ومعالجٌ لم يكتب ترويسةً — وهي ٢٠٠", http.MethodPost, 0, true},
		{"والقراءةُ لا تُبَثّ", http.MethodGet, http.StatusOK, false},
		{"وفشلُ التحقّق لا يُبَثّ", http.MethodPost, http.StatusBadRequest, false},
		{"والمنعُ لا يُبَثّ", http.MethodPost, http.StatusForbidden, false},
		{"وعطبُ الخادم لا يُبَثّ", http.MethodPost, http.StatusInternalServerError, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hub := realtime.NewHub(slog.New(slog.NewTextHandler(os.Stderr, nil)))
			ch, done := hub.Subscribe([]string{realtime.TopicCatalog})
			defer done()

			s := &Server{hub: hub}
			h := s.announceWrites(http.HandlerFunc(
				func(w http.ResponseWriter, _ *http.Request) {
					if c.status != 0 {
						w.WriteHeader(c.status)
					}
					_, _ = w.Write([]byte("{}"))
				}))
			h.ServeHTTP(httptest.NewRecorder(),
				httptest.NewRequest(c.method, "/api/v1/admin/settings", nil))

			// **والقناةُ مخزَّنة** — فما بُثّ قبل القراءة ينتظر فيها،
			// ولا يلزم انتظارُ زمنٍ يجعل الاختبارَ متقلّبا.
			var got bool
			select {
			case <-ch:
				got = true
			default:
			}
			if got != c.want {
				t.Fatalf("%s %d: بُثّ %v والمنتظَر %v", c.method, c.status, got, c.want)
			}
		})
	}
}

// TestAnnounceWrites_IsWiredToBothGates **ووسيطٌ صحيحٌ غيرُ موصولٍ لا
// يفعل شيئاً.**
//
// **وهي عائلةُ الخلل التي تتكرّر في هذا المستودع**: يُكتب الحارسُ
// ويُختبر ويُنسى وصلُه، **فتمرّ الفحوصُ خضراءَ والعطبُ قائم.**
//
// **ويُقرأ المصدرُ ولا يُنادى مسار** — كما في `rep_media_route_test.go`:
// تركيبُ خادمٍ كاملٍ هنا يحتاج قاعدةً ومحرّكاً فينهار قبل أن يُبلَغ
// الوسيط، **واختبارٌ يردّ ٥٠٠ حيث يُنتظر بثٌّ لا يقيس شيئا.**
func TestAnnounceWrites_IsWiredToBothGates(t *testing.T) {
	raw, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الموجّه: %v", err)
	}
	src := string(raw)

	for _, gate := range []string{`r.Route("/admin"`, `r.Route("/merchant"`} {
		at := strings.Index(src, gate)
		if at < 0 {
			t.Fatalf("لم تُوجد المجموعة %s", gate)
		}
		// **وضمن رأس المجموعة** — الوسائطُ تُوضع في أوّلها لا في وسطها،
		// **ووسيطٌ يُسجَّل بعد ألفِ سطرٍ يكون داخل مجموعةٍ فرعيّةٍ لا
		// في هذه.**
		end := at + 1200
		if end > len(src) {
			end = len(src)
		}
		if !strings.Contains(src[at:end], "r.Use(s.announceWrites)") {
			t.Errorf("%s بلا `announceWrites` — ما يُكتب فيها لا يصل تطبيقاً مفتوحا", gate)
		}
	}
}
