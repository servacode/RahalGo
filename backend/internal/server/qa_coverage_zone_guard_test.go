package server

import (
	"os"
	"strings"
	"testing"
)

// TestQACoverageZoneCapabilityGuards **حرّاسُ قدرةِ شاهدِ التغطية الحيّ**
// (Batch 3d). **بنيةُ اختبارٍ ضيّقةٌ يجب ألّا تصير باباً واسعاً** — تُقاس
// من المصدرِ فلا تُنسى أو يتّسع نطاقُها في تعديلٍ لاحق.
func TestQACoverageZoneCapabilityGuards(t *testing.T) {
	src, err := os.ReadFile("qa_coverage_zone.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(src)

	// 1) اسمٌ ثابتٌ واحدٌ — لا منطقةَ عشوائيّة.
	if !strings.Contains(s, `qaCoverageZoneName   = "QA-3d-coverage-test"`) {
		t.Error("**الاسمُ الثابتُ «QA-3d-coverage-test» غائبٌ أو تغيّر**")
	}
	// 2) كلُّ معالجٍ يسقط مغلقاً في الإنتاج (حارسٌ ثانٍ بعد المنادي).
	if n := strings.Count(s, "if !s.qaStagingEnabled() {"); n < 3 {
		t.Errorf("**حارسُ qaStagingEnabled ناقصٌ**: وجد %d من 3", n)
	}
	// 3) التعديلُ والحذفُ بالاسمِ الثابت لا بمعرّفٍ يأتي من الطلب.
	if !strings.Contains(s, "s.qaCoverageZoneID(ctx)") {
		t.Error("**التعديل/الحذف لا يمرّ بالبحثِ بالاسمِ الثابت**")
	}
	if strings.Contains(s, "chi.URLParam") || strings.Contains(s, `Query().Get("id")`) ||
		strings.Contains(s, `Query().Get("zone_id")`) {
		t.Error("**القدرةُ تقبل معرّفَ منطقةٍ من الطلب — ممنوع**")
	}
	// 4) الإشعارُ الحقيقيّ يمرّ في الإنشاءِ والإعادة (لا شاهدَ مُصطنَع).
	if strings.Count(s, "s.notifyAreaCoverage(ctx)") < 2 {
		t.Error("**لا يمرّ notifyAreaCoverage الحقيقيّ في الإنشاءِ والإعادة**")
	}
	// 5) لا حقنَ SQL خامٍّ على جدولِ المناطق — يمرّ بخدمةِ catalog وحدَها.
	if strings.Contains(s, "delivery_zones") {
		t.Error("**SQL خامٌّ على delivery_zones — يجب أن يمرّ بخدمةِ catalog لا بحقنٍ**")
	}
	// 6) يمرّ بمسارِ خدمةِ الأدمن الحقيقيّ الثلاثيّ.
	for _, m := range []string{"s.catalog.CreateZone(", "s.catalog.UpdateZone(", "s.catalog.DeleteZone("} {
		if !strings.Contains(s, m) {
			t.Errorf("**لا يستدعي المسارَ الحقيقيَّ %s**", m)
		}
	}
	// 7) لا يُصدِر توكن/جلسةَ أدمن — الهويّةُ معرّفُ تدقيقٍ فحسب.
	if strings.Contains(s, "IssueForUserID") || strings.Contains(s, "ActiveSessionID") {
		t.Error("**القدرةُ تُصدِر جلسةً/توكن أدمن — ممنوع**")
	}
}
