package qa

import (
	"strings"
	"testing"
)

// TestPLATSW_ManualCloseShowsOwnerReason **سببُ السكّر باليد يصل الزبون** (قرارُ المالك ٢٠٢٦-١١-١٢:
// «وقت نحط مسكرين لازم الرسالة تتغير من خارج الدوام الى سبب اخر نكتبو من لوحة التحكم») —
// والزرُّ مسكّرٌ يقول السبب، **ومفتوحٌ أو بالجدول لا يقوله**، وفارغُه يُبقي النصَّ كما كان.
func TestPLATSW_ManualCloseShowsOwnerReason(t *testing.T) {
	h := New(t)
	const reason = "صيانة طارئة — نرجع بعد شوي"
	h.Setting("hours.platform_switch", `"closed"`)
	h.Setting("hours.platform_switch_reason", `"`+reason+`"`)
	r := h.GET("/api/v1/public/platform", "")
	if !strings.Contains(string(r.Body), reason) || !strings.Contains(string(r.Body), "platform_closed_now") {
		t.Fatalf("**السببُ لم يصل الزبون والمنصّةُ مسكّرة**: %s", r.Body)
	}
	if strings.Contains(string(r.Body), `"next_available_at"`) {
		t.Fatalf("**موعدُ عودةٍ مخترَعٌ لسكّرٍ باليد**: %s", r.Body)
	}
	h.Setting("hours.platform_switch", `"open"`)
	if r := h.GET("/api/v1/public/platform", ""); strings.Contains(string(r.Body), reason) {
		t.Fatalf("**السببُ ظهر والمنصّةُ مفتوحة**: %s", r.Body)
	}
	h.Setting("hours.platform_switch", `"closed"`)
	h.Setting("hours.platform_switch_reason", `""`)
	if r := h.GET("/api/v1/public/platform", ""); strings.Contains(string(r.Body), reason) ||
		!strings.Contains(string(r.Body), "platform_closed_now") {
		t.Fatalf("**فارغُ السبب غيّر الحال**: %s", r.Body)
	}
	h.Setting("hours.platform_switch", `"schedule"`)
}
