package qa

// **لا بابَ انضمامٍ عامّاً في الموجِّه** — `JOIN-0`، ٢٠٢٦-٠٩-٣٠.
//
// (قرارُ المالك: «رابط الدعوة ما له أساس… احذفه ما له داعي».)
//
// # ولماذا يمشي الموجِّهَ لا الدالّة
//
// **كان الفحصُ السابقُ ينادي `handlePublicJoin` مباشرةً** — ويمرّ ولو
// لم يُسجَّل المسارُ في الموجِّه أصلاً. **وما يُحذف يُحذف من حيث يصله
// الغريب**: من العنوان لا من اسم الدالّة.
//
// **فيمشي الموجِّهَ الحقيقيَّ كلَّه** — ومن أعاد المسارَ يوماً سقط هنا.

import (
	"net/http"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestJOIN0_PublicJoinRouteIsGone(t *testing.T) {
	hh := New(t)
	gone := []string{"/api/v1/public/join", "/api/v1/public/invite"}
	var found []string
	err := chi.Walk(hh.API.RouterForWalk(),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			for _, g := range gone {
				if strings.TrimSuffix(route, "/") == g {
					found = append(found, method+" "+route)
				}
			}
			return nil
		})
	if err != nil {
		t.Fatalf("المشّاء: %v", err)
	}
	if len(found) > 0 {
		t.Fatalf("بابُ الانضمام العامّ ما زال في الموجِّه: %v", found)
	}
}
