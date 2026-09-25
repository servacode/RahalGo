package server

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// TestQABoundaryKindsRegistered — نوعا الحدِّ مسجَّلان في السماح، وكلاهما
// حالةٌ (لا يلزمهما هويّةُ زبون QA قبل التبديل — المنطقةُ تُحلّ من عنوانِ QA داخليّاً).
func TestQABoundaryKindsRegistered(t *testing.T) {
	for _, k := range []string{"boundary_arm", "boundary_restore"} {
		if !qaSeedAllowlist[k] {
			t.Fatalf("%s must be in qaSeedAllowlist", k)
		}
		if !qaStateSeed[k] {
			t.Fatalf("%s must be a state seed (no QA uid needed before dispatch)", k)
		}
	}
}

// TestQABoundaryCapabilityGuards **حرّاسُ قدرةِ شاهدِ حدِّ الوقت الحيّ**
// (Batch 5). **بنيةُ اختبارٍ ضيّقةٌ يجب ألّا تصير باباً واسعاً** — تُقاس من
// المصدرِ فلا تُنسى أو يتّسع نطاقُها في تعديلٍ لاحق.
func TestQABoundaryCapabilityGuards(t *testing.T) {
	src, err := os.ReadFile("qa_batch5_witness.go")
	if err != nil {
		t.Fatalf("read source: %v", err)
	}
	s := string(src)

	// 1) كلا المعالجَين يسقط مغلقاً في الإنتاج (حارسٌ ثانٍ بعد المنادي).
	if n := strings.Count(s, "if !s.qaStagingEnabled() {"); n < 2 {
		t.Errorf("**حارسُ qaStagingEnabled ناقصٌ**: وجد %d من 2", n)
	}
	// 2) يمرّ بخدمةِ الأدمن الحقيقيّة (منصّةً ومنطقةً) — لا حقنَ SQL على الجدول.
	for _, m := range []string{"s.platform.SetSchedule(", "s.platform.SetZoneSchedule("} {
		if !strings.Contains(s, m) {
			t.Errorf("**لا يستدعي المسارَ الحقيقيَّ %s**", m)
		}
	}
	// 3) لا كتابةَ خامٍّ على جداولِ الجدول/التغطية — القراءةُ الوحيدةُ لعنوانِ QA.
	for _, bad := range []string{"UPDATE delivery_zones", "UPDATE settings", "INSERT INTO delivery_zones"} {
		if strings.Contains(s, bad) {
			t.Errorf("**كتابةٌ خامّةٌ ممنوعةٌ في القدرة: %s**", bad)
		}
	}
	// 4) يحفظ السابقَ ليُستعاد — لا يُترَك جدولٌ اختباريٌّ نافذاً.
	if !strings.Contains(s, "qaBoundarySaved") {
		t.Error("**لا حفظَ للحالِ السابقةِ (qaBoundarySaved) — لا استعادة**")
	}
	// 5) يبثّ التغيّرَ كبابِ الأدمن ليتجدّدَ التطبيقُ المفتوح ويُسلّح مؤقّتَه.
	if !strings.Contains(s, "s.hub.Publish(realtime.TopicCatalog") {
		t.Error("**لا يبثّ TopicCatalog — التطبيقُ المفتوحُ لن يُسلّح مؤقّتَه**")
	}
	// 6) لا يُصدِر جلسةً/توكن أدمن — الهويّةُ معرّفُ تدقيقٍ فحسب.
	if strings.Contains(s, "IssueForUserID") || strings.Contains(s, "ActiveSessionID") {
		t.Error("**القدرةُ تُصدِر جلسةً/توكن أدمن — ممنوع**")
	}
	// 7) لا يقبل معرّفَ منطقةٍ من الطلب — المنطقةُ من عنوانِ زبونِ QA وحدَه.
	if strings.Contains(s, "req.ZoneID") || strings.Contains(s, "chi.URLParam") {
		t.Error("**القدرةُ تقبل معرّفَ منطقةٍ من الطلب — ممنوع، تُحلّ من عنوانِ QA**")
	}
}

// TestQABoundaryWindows — **المنطقُ الخالصُ لبناءِ النافذتين**: مفتوحٌ الآن،
// يُغلَق بعد الدقائق، يُعيد الفتحَ بعد الإغلاقِ بدقيقتين؛ ويُرفض قربَ منتصفِ الليل.
func TestQABoundaryWindows(t *testing.T) {
	loc := platform.Location()

	// منتصفُ النهار — حالةٌ صالحة.
	now := time.Date(2026, 9, 25, 12, 0, 0, 0, loc)
	ws, closeAt, reopenAt, ok := qaBoundaryWindows(now, 2)
	if !ok {
		t.Fatal("midday arm must succeed")
	}
	if len(ws) != 2 {
		t.Fatalf("want 2 windows, got %d", len(ws))
	}
	// النافذةُ المفتوحةُ تنتهي عند الإغلاق (12:02)، والثانيةُ تفتح بعده بدقيقتين (12:04).
	if int(ws[0].End) != 722 || int(ws[1].Start) != 724 {
		t.Fatalf("boundaries off: openEnd=%d reopenStart=%d", int(ws[0].End), int(ws[1].Start))
	}
	// **الآنُ داخلَ النافذةِ المفتوحة** — أي المنصّةُ/المنطقةُ مفتوحةٌ لحظةَ التسليح.
	mod := now.Hour()*60 + now.Minute()
	if !(int(ws[0].Start) <= mod && mod < int(ws[0].End)) {
		t.Fatalf("arm-time %d must fall inside the open window [%d,%d)", mod, int(ws[0].Start), int(ws[0].End))
	}
	if closeAt.Hour() != 12 || closeAt.Minute() != 2 {
		t.Fatalf("close_at want 12:02, got %02d:%02d", closeAt.Hour(), closeAt.Minute())
	}
	if reopenAt.Hour() != 12 || reopenAt.Minute() != 4 {
		t.Fatalf("reopen_at want 12:04, got %02d:%02d", reopenAt.Hour(), reopenAt.Minute())
	}

	// قربَ منتصفِ الليل — يُرفض (نوافذُ اليومِ لا تعبر).
	if _, _, _, ok := qaBoundaryWindows(time.Date(2026, 9, 25, 23, 59, 0, 0, loc), 2); ok {
		t.Error("near-midnight arm must be refused")
	}
	// أوّلُ الليلِ جدّاً — يُرفض (mod<2).
	if _, _, _, ok := qaBoundaryWindows(time.Date(2026, 9, 25, 0, 0, 0, 0, loc), 2); ok {
		t.Error("just-after-midnight arm must be refused")
	}
}
