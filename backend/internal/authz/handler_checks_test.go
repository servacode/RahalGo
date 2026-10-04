package authz

import "testing"

// TestHandlerChecksHaveRoutes **وكلُّ قدرةٍ تُسأل داخلَ بابٍ لها بابٌ قائم.**
//
// **وإلّا صار السطرُ عذراً لقدرةٍ يتيمة**: يقول العقدُ إنّها تُسأل في معالِجٍ
// لا مسارَ له في الجدول.
func TestHandlerChecksHaveRoutes(t *testing.T) {
	for _, h := range HandlerChecks() {
		if !Known(h.Need) {
			t.Errorf("قدرةٌ مجهولة: %q", h.Need)
		}
		if _, ok := LookupAdmin(h.Method, h.Pattern); !ok {
			t.Errorf("**%s %s لا سياسةَ له** — والقدرةُ %s تُسأل فيه",
				h.Method, h.Pattern, h.Need)
		}
		if h.Where == "" {
			t.Errorf("%s: بلا موضع", h.Need)
		}
	}
}
