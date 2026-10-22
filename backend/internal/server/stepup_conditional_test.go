package server

// **تصنيفُ الفعلِ المشروطِ لخطوة التحقّق** — وحدةٌ صافيةٌ بلا قاعدة.
//
// `sensitiveNow` لا تلمس حالةَ الخادم، فتُختبَر بخادمٍ فارغ. **والعقدُ الجديد**
// (٢٠٢٦-٠٩-٢٧): النقلُ الإداريُّ إلى `refunded` يعكس مالاً فيلزمه تأكيد، وبقيّةُ
// الانتقالات تشغيليّةٌ تمضي بلا تأكيد.

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

func TestSensitiveNow_TransitionRefundNeedsStepUp(t *testing.T) {
	s := &Server{}
	act := authz.Sensitive{Conditional: authz.CondTransitionRefund}
	cases := []struct {
		body string
		want bool
	}{
		{`{"to":"refunded"}`, true},   // **يعكس مالاً ⇒ تأكيد**
		{`{"to":"picked_up"}`, false}, // تشغيليّ
		{`{"to":"delivered"}`, false},
		{`{"to":"cancelled"}`, false},
		{`{"to":"failed"}`, false},
		{`{}`, false},
		{``, false},
	}
	for _, c := range cases {
		if got := s.sensitiveNow("/orders/x/transition", act, []byte(c.body)); got != c.want {
			t.Errorf("جسمٌ %q ⇒ %v، والمتوقّع %v", c.body, got, c.want)
		}
	}
}

// TestSensitiveNow_StatusStrong يثبّت العقدَ القائم (دورةُ ١٧) كي لا ينكسر بجواره.
func TestSensitiveNow_StatusStrong(t *testing.T) {
	s := &Server{}
	act := authz.Sensitive{Conditional: authz.CondStatusIsStrong}
	for body, want := range map[string]bool{
		`{"status":"blocked"}`:   true,
		`{"status":"deleted"}`:   true,
		`{"status":"suspended"}`: false,
		`{"status":"active"}`:    false,
	} {
		if got := s.sensitiveNow("/users/x", act, []byte(body)); got != want {
			t.Errorf("جسمٌ %q ⇒ %v، والمتوقّع %v", body, got, want)
		}
	}
}

// TestStepUp_DisabledPassesSensitive **قرارُ المالك ٢٠٢٦-١٠-٢٢: لا كلمةَ على أيّ فعل** —
// المحرّكُ يُقلع بـ`DisableStepUp`، فالفعلُ الحسّاسُ يمضي بلا إثبات. وبدونه يُردّ.
func TestStepUp_DisabledPassesSensitive(t *testing.T) {
	hit := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hit = true })
	call := func(s *Server) int {
		hit = false
		w := httptest.NewRecorder()
		s.requireStepUp(next).ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/admin/roles", strings.NewReader(`{}`)))
		return w.Code
	}
	if code := call(&Server{}); hit || code != http.StatusForbidden {
		t.Fatalf("التأكيدُ قائمٌ ولم يُطلب: %d · %v", code, hit)
	}
	off := &Server{}
	off.DisableStepUp()
	if call(off); !hit {
		t.Fatal("**التأكيدُ مرفوعٌ والفعلُ الحسّاسُ لم يمضِ**")
	}
}
