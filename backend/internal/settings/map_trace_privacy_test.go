package settings

import (
	"strings"
	"testing"
)

// **جمعُ نقاط الطريق بلا هويّة لتحسين الخريطة مذكورٌ في خصوصيّة السائق** — في الافتراض، وفي كلِّ
// نصٍّ يعدّله المالك (قرارُ المالك ٢٠٢٦-١١-٠٦). والنقاطُ تُجمع منذ ٢٠٢٦-١٠-٢٠ (الهجرة ٠٤٦٦).
func TestDriverPrivacy_MapTraceLineAlwaysPresent(t *testing.T) {
	if !strings.Contains(DefaultDriverPrivacy, DriverMapTraceLine) {
		t.Fatal("خصوصيّةُ السائق الافتراضيّة لا تذكر جمعَ النقاط لتحسين الخريطة")
	}
	if !strings.Contains(DefaultDriverTerms, DriverMapTraceLine) {
		t.Error("شروطُ السائق الافتراضيّة لا تذكر جمعَ النقاط لتحسين الخريطة")
	}
	if !strings.Contains(DefaultPrivacy, "الخريطة") || !strings.Contains(DefaultPrivacy, "بلا اسم") {
		t.Error("الخصوصيّةُ العامّة (قسمُ السائق) لا تذكر نقاطَ الخريطة بلا اسم")
	}
	// **وبندُ التعامل برّا المنصّة باقٍ في الشروط.**
	if !strings.Contains(DefaultDriverTerms, DriverTermsClause) {
		t.Error("سقط بندُ التعامل برّا المنصّة من الشروط")
	}
	if got := EffectiveDriverPrivacy(DefaultDriverPrivacy); got != DefaultDriverPrivacy {
		t.Error("السطرُ تكرّر في نصٍّ فيه أصلاً")
	}
	edited := "ما نجمعه\nنصٌّ عدّله المالك."
	got := EffectiveDriverPrivacy(edited)
	if !strings.HasPrefix(got, edited) || !strings.HasSuffix(got, DriverMapTraceLine) {
		t.Errorf("نصٌّ معدَّلٌ بلا السطر لم يُلحَق به: %q", got)
	}
	if strings.Count(EffectiveDriverPrivacy(got), DriverMapTraceLine) != 1 {
		t.Error("السطرُ أُلحق مرّتين")
	}
	if got := EffectiveDriverPrivacy(" "); !strings.Contains(got, DriverMapTraceLine) {
		t.Errorf("نصٌّ فارغٌ بلا السطر: %q", got)
	}
}
