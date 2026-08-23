package qa

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ المرحلة ٨أ — ما لا يجوز أن ينزلق**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذه تقرأ المصدر** — فما مضى من عيوبٍ صامتةٍ لا يعود بلا أن يُلاحَظ.

// TestSNAP_010_NoUnlimitedRetry **لا إعادةَ محاولةٍ بلا حدّ.**
//
// **أمرُ المالك (البند ٧) نصّاً**: «ممنوع: restricted radius fails →
// retry without radius. لأن ذلك يعيد العيب الصامت».
func TestSNAP_010_NoUnlimitedRetry(t *testing.T) {
	src := readSource(t, "../routing/osrm.go")

	// **`unlimited` لا تُكتب في المحوّل أصلاً** — موضعُها الوحيد
	// `SnapPolicy.radiusValue`، وتُستعمل في الاختبار لا في الإنتاج.
	if strings.Contains(src, "unlimited") {
		t.Fatal("المحوّلُ يذكر unlimited — وهذا بابُ العيب الصامت")
	}

	// **وجسمُ الطلب يلحق الحدّ دائماً** — وممرٌّ واحدٌ لا غير.
	i := strings.Index(src, "func (c *Client) request(")
	if i < 0 {
		t.Fatal("لم تُوجد request")
	}
	body := src[i:]
	if j := strings.Index(body[10:], "\nfunc "); j > 0 {
		body = body[:j+10]
	}
	if !strings.Contains(body, "&radiuses=") {
		t.Fatal("جسمُ الطلب لا يلحق radiuses")
	}
	if strings.Count(src, "&radiuses=") != 1 {
		t.Fatalf("radiuses تُلحق في %d موضعاً — والممرّ يجب أن يبقى واحداً",
			strings.Count(src, "&radiuses="))
	}
}

// TestSNAP_011_ServerUsesDefaultPolicy **الخادمُ لا يخترع حدّاً.**
//
// **البند ٤**: السياسةُ مركزيّةٌ واحدة. **ومن نادى `NewWithSnap` من
// الخادم** فتح باباً لسياستَين.
func TestSNAP_011_ServerUsesDefaultPolicy(t *testing.T) {
	for _, f := range []string{"../server/server.go", "../server/driver_route.go"} {
		src := readSource(t, f)
		if strings.Contains(src, "NewWithSnap") {
			t.Fatalf("%s ينادي NewWithSnap — والسياسةُ يجب أن تبقى واحدة", f)
		}
		if strings.Contains(src, "SnapPolicy{") {
			t.Fatalf("%s يبني سياسةَ التقاطٍ بنفسه", f)
		}
	}
}

// TestSNAP_012_ClientCannotSetRadius **الهاتفُ لا يحدّد نصفَ القطر.**
//
// **البند ٤ نصّاً**: «الهاتف لا يحدد: snap radius / routing engine /
// profile. هذه سياسة Backend».
func TestSNAP_012_ClientCannotSetRadius(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	for _, bad := range []string{"radius", "radiuses", "snap_radius", "engine="} {
		if strings.Contains(strings.ToLower(src), bad) {
			t.Fatalf("منفذُ المسار يقرأ %q من الطلب", bad)
		}
	}
}

// TestSNAP_013_FailuresAreNotCached **فشلٌ لا يُخزَّن مساراً ناجحاً.**
//
// **البند ١١**: `NoSegment` لا تُحفظ. **والحفظُ يقع بعد فحص الخطأ**
// — فيُقرأ ذلك من ترتيب السطور.
func TestSNAP_013_FailuresAreNotCached(t *testing.T) {
	src := readSource(t, "../server/driver_route.go")
	for _, fn := range []string{"func (s *Server) routeSetCached", "func (s *Server) routeCached"} {
		i := strings.Index(src, fn)
		if i < 0 {
			t.Fatalf("لم تُوجد %s", fn)
		}
		body := src[i:]
		if j := strings.Index(body[10:], "\nfunc "); j > 0 {
			body = body[:j+10]
		}
		errAt := strings.Index(body, "if err != nil {\n\t\treturn nil, err")
		setAt := strings.Index(body, "s.rdb.Set(")
		if errAt < 0 {
			t.Fatalf("%s: لا خروجَ صريحٌ عند الخطأ", fn)
		}
		if setAt < 0 {
			t.Fatalf("%s: لا حفظ", fn)
		}
		if errAt > setAt {
			t.Fatalf("%s: يُحفظ قبل أن يُفحص الخطأ — ففشلٌ يُخزَّن نجاحاً", fn)
		}
	}
}

// TestSNAP_014_BackendIsInterface **الخادمُ يمسك عقداً لا صنفاً.**
//
// **البند ١** — وهو ما يجعل محرّكاً آخرَ ممكناً بلا لمسِ الجوّال.
func TestSNAP_014_BackendIsInterface(t *testing.T) {
	src := readSource(t, "../server/server.go")
	if !regexp.MustCompile(`route\s+routing\.Backend`).MatchString(src) {
		t.Fatal("حقلُ المحرّك ليس routing.Backend")
	}
	if strings.Contains(src, "*routing.Client") {
		t.Fatal("الخادمُ ما زال يمسك *routing.Client")
	}
}

// TestSNAP_015_MobileKnowsNothing **الجوّالُ لا يعلم — البند ٢.**
//
// **ولا كلمةَ محرّكٍ ولا نصفِ قطرٍ في شيفرة أندرويد.**
func TestSNAP_015_MobileKnowsNothing(t *testing.T) {
	roots := []string{
		"../../../mobile/driver-navigation/src/main/kotlin/com/rahalgo/navigation",
		"../../../mobile/shared/src/main/kotlin/com/rahalgo/shared/driver",
	}
	bad := []string{"osrm", "valhalla", "radiuses", "snapradius", "snap_radius"}
	for _, root := range roots {
		if _, err := os.Stat(root); err != nil {
			t.Skipf("لا مجلّد: %s", root)
		}
		entries, err := os.ReadDir(root)
		if err != nil {
			t.Fatalf("قراءة %s: %v", root, err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".kt") {
				continue
			}
			raw, err := os.ReadFile(root + "/" + e.Name())
			if err != nil {
				t.Fatalf("قراءة %s: %v", e.Name(), err)
			}
			// **والتعليقاتُ تُطرح** — ففي `NavRoute.kt` سطرٌ يقول
			// «ولا يعرف OSRM»، **وهو ما نريده لا ما نمنعه.**
			low := strings.ToLower(stripKotlinComments(string(raw)))
			for _, b := range bad {
				if strings.Contains(low, b) {
					t.Fatalf("%s فيه %q — والجوّالُ لا يعرف المحرّك", e.Name(), b)
				}
			}
		}
	}
}

// stripKotlinComments **يحذف `//` و`/* */`** — ليُفحص الكود لا الكلام.
func stripKotlinComments(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if strings.HasPrefix(s[i:], "//") {
			j := strings.IndexByte(s[i:], '\n')
			if j < 0 {
				break
			}
			i += j
			continue
		}
		if strings.HasPrefix(s[i:], "/*") {
			j := strings.Index(s[i:], "*/")
			if j < 0 {
				break
			}
			i += j + 2
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
