package qa

// ══════════════════════════════════════════════════════════════════════
// **الإتاحةُ قبل السلّة — أن يُعرَف الردُّ قبل أن يُمشى الطريق** (`PC`)
// ══════════════════════════════════════════════════════════════════════
//
// **وكان أوّلُ خبرٍ يبلغه أنّ عنوانَه خارجَ النطاق يأتيه في السلّة** —
// **بعد أن اختار وأضاف وقرأ الأسعار.** **ومن مشى الطريقَ كلَّه ليُردّ
// في آخره يقرأ الردَّ عقوبةً لا خبرا.**
//
// **والبابُ يقرأ محرّكَ الدفعة الثالثة عينَه** — **ولا محرّكَ ثانٍ
// لشاشةٍ ثانية.**

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// availabilityAt يسأل البابَ عن عنوانٍ بلا سلّة.
func availabilityAt(t *testing.T, h *Harness, lat, lng float64) map[string]any {
	t.Helper()
	r := h.GET(fmt.Sprintf("/api/v1/public/availability?lat=%f&lng=%f", lat, lng), "")
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الإتاحة رُدّ**: %d / %s", r.Code, r.Err())
	}
	return r.JSON()
}

// reasonAt السببُ وحدَه.
func reasonAt(t *testing.T, h *Harness, lat, lng float64) string {
	t.Helper()
	v, _ := availabilityAt(t, h, lat, lng)["reason"].(string)
	return v
}

// ═════════════════ PC-01 — عنوانٌ صالحٌ يُقال له «نعم» ═════════════════

// TestPC01_AvailableAddressIsKnownBeforeCart **ولا سلّةَ في السؤال.**
func TestPC01_AvailableAddressIsKnownBeforeCart(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PC-01")

	got := availabilityAt(t, hh, raqqaLat, raqqaLng)
	if got["available"] != true {
		t.Fatalf("**عنوانٌ مخدومٌ قيل عنه غيرُ متاح**: %v", got)
	}
	if got["reason"] != "service_available" {
		t.Fatalf("**سببٌ غيرُ متوقَّع**: %v", got["reason"])
	}
}

// ═════════════════ PC-03 · PC-04 · PC-05 — الجغرافيا تُعرَف مبكّراً ═════

// TestPC03_PC04_GeographyKnownBeforeCart **والمدينةُ غيرُ المطلقة
// غيرُ العنوان خارجَ الشكل.**
func TestPC03_PC04_GeographyKnownBeforeCart(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PC-03")

	// **PC-04 · عنوانٌ في مدينةٍ مخدومةٍ خارجَ شكل المنطقة.**
	if got := reasonAt(t, hh, raqqaLat+0.08, raqqaLng); got != "address_outside_coverage" {
		t.Fatalf("**خارجَ الشكل قيل عنه**: %q", got)
	}

	// **PC-03 · مدينةٌ مبذورةٌ مُطفأة.**
	if got := reasonAt(t, hh, damA_Lat, damA_Lng); got != "city_not_supported" {
		t.Fatalf("**مدينةٌ لم تُطلَق قيل عنها**: %q", got)
	}

	// **وموضعٌ لا مدينةَ له.**
	if got := reasonAt(t, hh, wildA_Lat, wildA_Lng); got != "area_not_supported" {
		t.Fatalf("**موضعٌ مجهولٌ قيل عنه**: %q", got)
	}
}

// ═════════════════ PC-06 — المنصّةُ خارجَ دوامها ═════════════════

// TestPC06_PlatformClosureVisibleBeforeCart **ويُقال قبل أن يملأ.**
func TestPC06_PlatformClosureVisibleBeforeCart(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PC-06")

	ends := time.Now().Add(2 * time.Hour)
	closure(t, hh, true, "صيانةٌ قصيرة", &ends)
	t.Cleanup(func() { closure(t, hh, false, "", nil) })

	got := availabilityAt(t, hh, raqqaLat, raqqaLng)
	if got["available"] != false {
		t.Fatalf("**المنصّةُ موقوفةٌ والبابُ يقول متاح**: %v", got)
	}
	if got["reason"] != "temporarily_unavailable" {
		t.Fatalf("**سببٌ غيرُ متوقَّع**: %v", got["reason"])
	}
	// **ونصُّ المالك يصل** — **فلا تخترع الشاشةُ سبباً.**
	if got["message"] != "صيانةٌ قصيرة" {
		t.Fatalf("**نصُّ المالك لم يصل**: %v", got["message"])
	}
}

// ═════════════════ PC-07 — المنطقةُ خارجَ وقتها ═════════════════

// TestPC07_ZoneClosureVisibleBeforeCart **ووقتُ المنطقة سببٌ زمنيّ.**
func TestPC07_ZoneClosureVisibleBeforeCart(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ PC-07")

	// **نافذةٌ تبدأ بعد ساعتين** — **فالآنَ خارجَها**، **وبالنصّ
	// القائم لا بنافذةٍ تُخترَع** (`zhShut`).
	zoneHours(t, hh, z.ID, true, zhShut())

	got := availabilityAt(t, hh, raqqaLat, raqqaLng)
	if got["reason"] != "zone_closed_now" {
		t.Fatalf("**منطقةٌ خارجَ وقتها قيل عنها**: %q", got["reason"])
	}
	// **وموعدُ العودة يُقال حين يُعرَف** — **ولا يُخترَع.**
	if got["next_available_at"] == "" {
		t.Fatalf("**وقتٌ معلومٌ ولم يُقَل**: %v", got)
	}
}

// ═════════════════ PC-13 — ولا سلّةَ تُطلَب لتُعرَف الحال ═════════════════

// TestPC13_AvailabilityNeedsNoCart **والسؤالُ عن العنوان لا عن
// البضاعة.**
//
// **وبوّابةُ المتجر تُتخطّى وحدَها** — **ومتجرٌ مغلقٌ لا يمنع أن
// يُقال للعنوان إنّه مخدوم**، **والسلّةُ تسأل عن متجرها حين تُملأ.**
func TestPC13_AvailabilityNeedsNoCart(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ PC-13")

	// **ولا جسمَ ولا أصناف** — **والبابُ قراءةٌ محضة.**
	r := hh.GET(fmt.Sprintf("/api/v1/public/availability?lat=%f&lng=%f",
		raqqaLat, raqqaLng), "")
	if r.Code != http.StatusOK {
		t.Fatalf("**بابٌ بلا سلّةٍ رُدّ**: %d / %s", r.Code, r.Err())
	}
	// **ونقطةٌ مشوَّهةٌ تُقال باسمها.**
	if got := reasonAt(t, hh, 91.0, 200.0); got != "invalid_location" {
		t.Fatalf("**نقطةٌ مشوَّهةٌ قيل عنها**: %q", got)
	}
	// **وحرفٌ مكانَ رقمٍ يُردّ ولا يُخمَّن.**
	if bad := hh.GET("/api/v1/public/availability?lat=abc&lng=1", ""); bad.Code != http.StatusBadRequest {
		t.Fatalf("**إحداثيّةٌ ليست رقماً قُبلت**: %d", bad.Code)
	}
}
