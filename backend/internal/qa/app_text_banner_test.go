package qa

// قرارا المالك ٢٠٢٦-١٠-٠٥: **رسائلُ التطبيق تُضبط من اللوحة** (`app_text.*`)
// و**السلايدرُ يُخفى بمفتاح** (`home.banner_enabled`).

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// ── رسالةُ «خارج التغطية» تصل في `details.notice` وتُعدَّل ──────────────
func TestAPPTEXT01_OutOfZoneCarriesEditableNotice(t *testing.T) {
	h := New(t)
	launchOn(h, "launch.customer_browse")
	notice := func() any {
		r := h.GET("/api/v1/public/zone?lat=0&lng=0", "")
		if r.Code != http.StatusBadRequest || r.Err() != "out_of_zone" {
			t.Fatalf("المنتظَرُ ٤٠٠ out_of_zone: %s", r)
		}
		e, _ := r.JSON()["error"].(map[string]any)
		d, _ := e["details"].(map[string]any)
		return d["notice"]
	}

	// **بلا ضبطٍ: النصُّ الأصليّ** — ما كان مكتوباً في التطبيق.
	h.Setting("app_text.out_of_zone", `""`)
	if got := notice(); got != settings.AppTextOutOfZoneDefault {
		t.Errorf("**الافتراضُ لم يصل**: %v", got)
	}
	// **وبنصٍّ: يصل كما ضُبط.**
	const edited = "عنوانك لسّا برّا مناطقنا — قريباً منوصلك"
	h.Setting("app_text.out_of_zone", `"`+edited+`"`)
	if got := notice(); got != edited {
		t.Errorf("**نصُّ المالك لم يصل**: %v", got)
	}
}

// ── الرموزُ الأربعةُ لها مفاتيحُها، وما سواها لا ────────────────────────
func TestAPPTEXT02_EveryCustomerCodeHasItsKey(t *testing.T) {
	for _, c := range []string{"out_of_zone", "coverage_unavailable", "merchant_closed", "item_unavailable"} {
		k := settings.AppTextKeyFor(c)
		if _, ok := settings.Lookup(k); !ok {
			t.Errorf("%s ⇒ %q ليس في الفهرس", c, k)
		}
	}
	if settings.AppTextKeyFor("launch_closed") != "" {
		t.Error("**مفتاحٌ لرمزٍ له نصُّه أصلاً**")
	}
}

// ── السلايدرُ مطفأٌ ⇒ قائمةٌ فارغةٌ في النقطتين ────────────────────────
func TestBANNER01_DisabledSliderReturnsEmptyList(t *testing.T) {
	h := New(t)
	launchOn(h, "launch.customer_browse")
	h.Setting("home.banner_enabled", "false")
	for _, p := range []string{"/api/v1/public/home", "/api/v1/public/banners?at=home"} {
		r := h.GET(p, "")
		if r.Code != http.StatusOK {
			t.Fatalf("%s: %s", p, r)
		}
		b, ok := r.JSON()["banners"].([]any)
		if !ok || len(b) != 0 {
			t.Errorf("**%s أرسل لافتاتٍ والسلايدرُ مطفأ**: %v", p, r.JSON()["banners"])
		}
	}
	h.Setting("home.banner_enabled", "true")
	if _, ok := h.GET("/api/v1/public/banners?at=home", "").JSON()["banners"].([]any); !ok {
		t.Error("**مشعولٌ ولا قائمة**")
	}
}
