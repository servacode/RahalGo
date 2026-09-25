package server

import (
	"os"
	"strings"
	"testing"
)

// TestQACatalogBridgeGuards **حرّاسُ جسرِ شاهدِ فهرس القِسم** (Batch 5،
// Customer Final). **يجب أن يمرّ بمعالِجاتِ الأدمن الحقيقيّةِ عينِها ووسيطِ
// `announceWrites` عينِه** — لا تطبيقٌ ثانٍ، ولا SQL خامّ، ويسقط مغلقاً في الإنتاج.
func TestQACatalogBridgeGuards(t *testing.T) {
	src, err := os.ReadFile("server.go")
	if err != nil {
		t.Fatalf("read server.go: %v", err)
	}
	s := string(src)

	// 1) الجسرُ يعيد استعمالَ معالِجاتِ الأدمن الحقيقيّةِ عينِها — لا نسخةَ QA.
	for _, m := range []string{
		`r.Post("/qa/catalog/sections", s.handleCreatePlatformSection)`,
		`r.Patch("/qa/catalog/sections/{id}", s.handleUpdatePlatformSection)`,
		`r.Delete("/qa/catalog/sections/{id}", s.handleDeletePlatformSection)`,
	} {
		if !strings.Contains(s, m) {
			t.Errorf("**الجسرُ لا يعيد استعمالَ المعالِجِ الحقيقيّ**: %s", m)
		}
	}

	// 2) الجسرُ ملفوفٌ بـ announceWrites عينِه — نفسُ نشرِ TopicCatalog.
	//    نتحقّق أنّ مجموعةَ /qa/catalog تستعمل announceWrites: نقتطع ما حول المسار.
	i := strings.Index(s, `/qa/catalog/sections`)
	if i < 0 {
		t.Fatal("**مسارُ /qa/catalog/sections غائب**")
	}
	// النافذةُ التي تسبق المسارَ يجب أن تحوي r.Use(s.announceWrites) القريب.
	start := i - 400
	if start < 0 {
		start = 0
	}
	if !strings.Contains(s[start:i], "r.Use(s.announceWrites)") {
		t.Error("**جسرُ الفهرس غيرُ ملفوفٍ بـ announceWrites القريب** — قد لا يُنشَر TopicCatalog")
	}

	// 3) الجسرُ داخلَ حارسِ التجهيز — يسقط مغلقاً في الإنتاج (المسارُ غيرُ مسجَّل).
	guardIdx := strings.LastIndex(s[:i], "if s.qaStagingEnabled() {")
	if guardIdx < 0 {
		t.Error("**جسرُ الفهرس ليس داخلَ if s.qaStagingEnabled()** — قد يُسجَّل في الإنتاج")
	}

	// 4) لا تطبيقٌ ثانٍ للفهرس في ملفّ QA — لا SQL خامّ على platform_sections.
	for _, f := range []string{"qa_batch5b_referral.go", "qa_batch5_witness.go"} {
		if b, e := os.ReadFile(f); e == nil {
			if strings.Contains(string(b), "platform_sections") {
				t.Errorf("**SQL/تطبيقٌ ثانٍ للفهرس في %s — ممنوع، يمرّ بالمعالِج الحقيقيّ**", f)
			}
		}
	}
}
