package qa

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **أقسامُ موظّف العمليّات — قرارُ المالك ٢٠٢٦-١٠-٠٥** (هجرة `0420`)
// ══════════════════════════════════════════════════════════════════════
//
// **يرى `operations` تسعةَ أقسامٍ لا غير** — ولا الحسابات ولا التقارير (لمدير
// المنصّة وحدَه)، **ولا شيءَ من المال والأدوار والإعدادات والعروض والحملات
// والسجلّ.**
//
// **والقرارُ يُكتب اختباراً** لئلّا يُوسَّع القسمُ غداً بسطرٍ في القائمة الجانبيّة
// أو بقدرةٍ تُمنح في شاشة الأدوار — **ولا يصرخ شيء.**

// opsOwnerSections **الأقسامُ كما سمّاها المالك** — بأبوابها في اللوحة.
var opsOwnerSections = []string{
	"/dashboard/orders",      // الطلبات
	"/dashboard/history",     // سجلّ الطلبات
	"/dashboard/sections",    // السوق
	"/dashboard/tickets",     // الشكاوى والتقييمات
	"/dashboard/emergencies", // الطوارئ
	"/dashboard/leads",       // طلبات الانضمام
	"/dashboard/opsmap",      // خريطة العمليّات
	"/dashboard/expansion",   // طلبات التوسّع
	"/dashboard/ops",         // مراقبة التشغيل
}

// navEntry بندٌ من القائمة الجانبيّة: بابُه والقدراتُ التي تُظهره (أيُّها كفى).
type navEntry struct {
	href string
	caps []string
}

// readDashboardNav **يقرأ `ALL_NAV` من شيفرة اللوحة نفسِها** — لا نسخةً منها.
func readDashboardNav(t *testing.T) []navEntry {
	t.Helper()
	p := filepath.Join("..", "..", "..", "web", "apps", "rahalgo", "src", "app", "dashboard", "layout.tsx")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("قراءةُ القائمة الجانبيّة: %v", err)
	}
	src := string(b)
	start := strings.Index(src, "const ALL_NAV")
	end := strings.Index(src, "function navFor")
	if start < 0 || end < start {
		t.Fatal("لم يوجد `ALL_NAV` في layout.tsx — تغيّر شكلُ الملفّ")
	}
	block := src[start:end]
	re := regexp.MustCompile(`(?s)href:\s*"([^"]+)".*?caps:\s*\[([^\]]*)\]`)
	capRe := regexp.MustCompile(`"([^"]+)"`)
	var out []navEntry
	for _, m := range re.FindAllStringSubmatch(block, -1) {
		e := navEntry{href: m[1]}
		for _, c := range capRe.FindAllStringSubmatch(m[2], -1) {
			e.caps = append(e.caps, c[1])
		}
		out = append(out, e)
	}
	if len(out) < 10 {
		t.Fatalf("قُرئ من القائمة %d بنداً فقط — المحلّلُ لم يعد يطابق الملفّ", len(out))
	}
	return out
}

// TestOpsSections_NavEqualsOwnerList **ما يراه موظّفُ العمليّات في القائمة =
// قائمةُ المالك بالضبط.** والقدراتُ من القاعدة بعد الهجرات، والبنودُ من شيفرة اللوحة.
func TestOpsSections_NavEqualsOwnerList(t *testing.T) {
	hh := New(t)
	caps := map[string]bool{}
	for _, c := range roleCaps(t, hh, "operations") {
		caps[c] = true
	}
	var got []string
	for _, e := range readDashboardNav(t) {
		for _, c := range e.caps {
			if caps[c] {
				got = append(got, e.href)
				break
			}
		}
	}
	want := append([]string(nil), opsOwnerSections...)
	sort.Strings(got)
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("**أقسامُ `operations` فارقت قرارَ المالك**\n   الظاهر  = %s\n   المُقرَّر = %s",
			strings.Join(got, " · "), strings.Join(want, " · "))
	}
	t.Logf("✓ operations ترى %d أقسام — كما قرّر المالك", len(got))
}

// TestOpsSections_PagesWorkAndOthersDenied **كلُّ ما تناديه صفحاتُ أقسامها يُفتح
// لها، وما ليس لها يُردّ ٤٠٣.**
//
// **و`404`/`400`/`409` نجاحُ تخويلٍ وردُّ منطق** — الحكمُ على `403` وحدَها.
func TestOpsSections_PagesWorkAndOthersDenied(t *testing.T) {
	hh := New(t)
	_, tok := roleUser(t, hh, "operations")
	const a = "/api/v1/admin"

	allowed := []string{
		// الطلبات وسجلُّها — اللوحةُ والإسنادُ اليدويّ وقراءةُ المهل.
		a + "/orders?limit=1", a + "/orders/board", a + "/orders/alerts",
		a + "/drivers", a + "/merchants", a + "/settings",
		// السوق.
		a + "/sections", a + "/categories", a + "/market/items", a + "/market/stores",
		a + "/market/quality", a + "/market/new-count",
		// الشكاوى والتقييمات.
		a + "/tickets", a + "/ratings",
		// الطوارئ.
		a + "/emergencies", a + "/emergencies/count", a + "/emergencies/banner", a + "/emergencies/map",
		// طلبات الانضمام — والمحافظاتُ لمرشّحها.
		a + "/leads", a + "/governorates",
		// خريطة العمليّات.
		a + "/ops-map/meta", a + "/ops-map/coverage", a + "/ops-map/branches", a + "/ops-map/areas",
		a + "/ops-map/orders", a + "/ops-map/drivers", a + "/ops-map/merchants",
		a + "/ops-map/demand", a + "/ops-map/opportunities", a + "/ops-map/coverage-requests",
		a + "/ops-map/coverage-demand/places",
		// شريطُ الخريطة وزبائنُها المجمَّعون ومكتبُها (قرارُ المالك ٢٠٢٦-١٠-٠٥).
		a + "/ops-map/summary", a + "/ops-map/customers", a + "/ops-map/office",
		// طلبات التوسّع — بلا `analytics.read`.
		a + "/ops-map/expansion", a + "/ops-map/expansion/reminder",
		// مراقبة التشغيل.
		a + "/ops/monitor", a + "/ops/status",
	}
	for _, p := range allowed {
		if r := hh.GET(p, tok); r.Code == http.StatusForbidden || r.Code == http.StatusUnauthorized {
			t.Errorf("**صفحةٌ لـ`operations` تسقط**: GET %s ← %d", p, r.Code)
		}
	}

	denied := []string{
		a + "/users?limit=1", a + "/users/stats", a + "/reports", a + "/stats", a + "/overview",
		a + "/promos", a + "/offers", a + "/banners", a + "/campaigns", a + "/broadcast/count",
		a + "/roles", a + "/audit", a + "/treasury/overview", a + "/payouts", a + "/obligations",
		a + "/ops/health",
	}
	for _, p := range denied {
		if r := hh.GET(p, tok); r.Code != http.StatusForbidden {
			t.Errorf("**`operations` بلغت ما ليس لها**: GET %s ← %d (أُريد ٤٠٣)", p, r.Code)
		}
	}
	// **وإبلاغُ المنتظرين يبقى لمن يملك التغطية.**
	if r := hh.POST(a+"/ops-map/expansion/notify", tok, map[string]any{}); r.Code != http.StatusForbidden {
		t.Errorf("**`operations` أبلغت المنتظرين**: %d (أُريد ٤٠٣)", r.Code)
	}
}

// TestOpsSections_MarketImageUpload **صورةُ السوق تُرفع بقدرة السوق** — من
// `POST /admin/market/media` بقائمةٍ بيضاء، **والبابُ العامُّ يبقى للمحتوى.**
func TestOpsSections_MarketImageUpload(t *testing.T) {
	hh := New(t)
	_, tok := roleUser(t, hh, "operations")

	upload := func(path, kind string) int {
		t.Helper()
		img := image.NewRGBA(image.Rect(0, 0, 8, 8))
		for x := 0; x < 8; x++ {
			for y := 0; y < 8; y++ {
				img.Set(x, y, color.RGBA{R: 200, G: 40, B: 40, A: 255})
			}
		}
		var pic bytes.Buffer
		if err := png.Encode(&pic, img); err != nil {
			t.Fatalf("ترميزُ الصورة: %v", err)
		}
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		_ = mw.WriteField("kind", kind)
		fw, _ := mw.CreateFormFile("file", "x.png")
		_, _ = fw.Write(pic.Bytes())
		_ = mw.Close()
		req, _ := http.NewRequest("POST", hh.Srv.URL+path, &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+tok)
		resp, err := hh.Srv.Client().Do(req)
		if err != nil {
			t.Fatalf("النداء: %v", err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}

	if c := upload("/api/v1/admin/market/media", "menu_item"); c != http.StatusCreated {
		t.Errorf("**صورةُ صنفٍ من السوق لم تُرفع**: %d (أُريد ٢٠١)", c)
	}
	if c := upload("/api/v1/admin/market/media", "banner"); c != http.StatusCreated {
		t.Errorf("**صورةُ قسمٍ من السوق لم تُرفع**: %d (أُريد ٢٠١)", c)
	}
	// **ولا شعارَ المنصّة من باب السوق.**
	if c := upload("/api/v1/admin/market/media", "platform_logo"); c != http.StatusForbidden {
		t.Errorf("**شعارُ المنصّة رُفع من باب السوق**: %d (أُريد ٤٠٣)", c)
	}
	// **والبابُ العامُّ يبقى بقدرة المحتوى.**
	if c := upload("/api/v1/admin/media", "menu_item"); c != http.StatusForbidden {
		t.Errorf("**`operations` بلغت البابَ العامَّ للوسائط**: %d (أُريد ٤٠٣)", c)
	}
}
