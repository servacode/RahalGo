package config

// **ملفُّ البيئة يُقرأ — ولا يطغى على ما كُتب في البيئة.**
//
// # ولماذا يُحرَس
//
// **الإعدادُ الممرَّرُ في سطر الأوامر يموت مع النافذة.** ومن شغّل المحرّكَ
// بطريقته المعتادة **يعود إلى `OTP_PROVIDER=dev` صامتاً** — فتُطبع رموزُ
// الدخول في السجلّ بدل أن تصل بواتساب، **والمنصّةُ تعمل فلا شيءَ ينبّه.**
//
// **وقارئٌ ينقلب فيطغى على البيئة أسوأُ من لا قارئ**: من مرّر متغيّراً لهذه
// المرّة يجده مُهمَلاً، **فيقلع الاختبارُ على قاعدة التطوير وهو يظنّه على
// قاعدة الاختبار.**

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDotEnv_ReadsFileAndYieldsToEnvironment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	body := "" +
		"# تعليقٌ يُقفز\n" +
		"\n" +
		"RAHALGO_TEST_FROM_FILE=من الملفّ\n" +
		"RAHALGO_TEST_QUOTED=\"بين علامتين\"\n" +
		"RAHALGO_TEST_PRESET=من الملفّ\n" +
		"سطرٌ بلا مساواة\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("تعذّرت كتابةُ الملفّ: %v", err)
	}

	// **وما في البيئة أقوى** — يُضبط قبل القراءة.
	t.Setenv("RAHALGO_TEST_PRESET", "من البيئة")

	loadDotEnv(path)

	if got := os.Getenv("RAHALGO_TEST_FROM_FILE"); got != "من الملفّ" {
		t.Fatalf("لم يُقرأ المفتاحُ من الملفّ: %q — **فيعود المحرّكُ إلى افتراضاته صامتاً**", got)
	}
	if got := os.Getenv("RAHALGO_TEST_QUOTED"); got != "بين علامتين" {
		t.Fatalf("علامتا الاقتباس لم تُنزعا: %q — **فتُقرأ جزءاً من القيمة**", got)
	}
	if got := os.Getenv("RAHALGO_TEST_PRESET"); got != "من البيئة" {
		t.Fatalf("الملفُّ طغى على البيئة: %q — **فمن مرّر متغيّراً لهذه المرّة يجده مُهمَلاً**", got)
	}
	// **وتنظيفٌ لما لا يملكه `t.Setenv`** — المفتاحان كُتبا بيد الدالّة.
	t.Cleanup(func() {
		_ = os.Unsetenv("RAHALGO_TEST_FROM_FILE")
		_ = os.Unsetenv("RAHALGO_TEST_QUOTED")
	})
}

// TestDotEnv_MissingFileIsNotAnError **وملفٌّ مفقودٌ ليس خطأً.**
//
// **والتطويرُ يعمل بلا ملفٍّ أصلاً** — والافتراضاتُ في الشيفرة تكفيه.
func TestDotEnv_MissingFileIsNotAnError(t *testing.T) {
	loadDotEnv(filepath.Join(t.TempDir(), "لا-وجود-له.env"))
}
