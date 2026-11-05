package settings

import (
	"strings"
	"testing"
)

// **بندُ «التعامل برّا المنصّة» في شروط السائق — في الافتراض، وفي كلِّ نصٍّ يعدّله المالك**
// (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧).
func TestDriverTerms_ClauseAlwaysPresent(t *testing.T) {
	if !strings.Contains(DefaultDriverTerms, DriverTermsClause) {
		t.Fatal("**الشروطُ الافتراضيّة بلا بند التعامل برّا المنصّة**")
	}
	if got := EffectiveDriverTerms(DefaultDriverTerms); got != DefaultDriverTerms {
		t.Errorf("**البندُ تكرّر في نصٍّ فيه أصلاً**")
	}
	edited := "عملُك معنا\nنصٌّ عدّله المالك."
	got := EffectiveDriverTerms(edited)
	if !strings.HasPrefix(got, edited) || !strings.HasSuffix(got, DriverTermsClause) {
		t.Errorf("**نصٌّ معدَّلٌ بلا البند لم يُلحَق به**: %q", got)
	}
	if strings.Count(EffectiveDriverTerms(got), DriverTermsClause) != 1 {
		t.Errorf("**البندُ أُلحق مرّتين**")
	}
	if got := EffectiveDriverTerms("  "); !strings.Contains(got, DriverTermsClause) {
		t.Errorf("**نصٌّ فارغٌ بلا البند**: %q", got)
	}
}
