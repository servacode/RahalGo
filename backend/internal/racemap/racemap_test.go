package racemap

import (
	"os"
	"strings"
	"testing"
)

// TestTenFlowsMapped **`CONCURRENCY-SENSITIVE FLOWS = 11`** — ولا واحدَ ينقص.
//
// **وكان العددُ عشرةً** — **وصار أحدَ عشرَ في دورةِ ١٨** بإضافة `C-11`
// (إعادةُ كلمةٍ مقابلَ تجديد). **والتجميدُ يمنع اختفاءَ تدفّقٍ لا
// اكتشافَ واحد.**
func TestTenFlowsMapped(t *testing.T) {
	if len(All) != 11 {
		t.Fatalf("الخريطةُ فيها %d — والعددُ المجمَّد 11", len(All))
	}
	seen := map[string]bool{}
	for _, r := range All {
		if seen[r.ID] {
			t.Errorf("معرّفٌ مكرَّر: %s", r.ID)
		}
		seen[r.ID] = true
		if r.Title == "" || r.Shared == "" || r.Window == "" || r.Invariant == "" {
			t.Errorf("%s: حقلٌ ناقص", r.ID)
		}
		if len(r.Actors) == 0 {
			t.Errorf("%s: بلا ممثّلين", r.ID)
		}
		if len(r.Flows) == 0 {
			t.Errorf("%s: بلا تدفّقٍ مرتبط", r.ID)
		}
		if len(r.Tests) == 0 {
			t.Errorf("%s: بلا اختبارٍ يحرسه — **وخريطةٌ بلا اختبارٍ ورقة**", r.ID)
		}
		if r.Result == "" || r.Where == "" {
			t.Errorf("%s: بلا نتيجةٍ أو بلا موضعِ إثبات", r.ID)
		}
		if r.Evidence == "" {
			t.Errorf("%s: بلا دليلٍ مطبوع — **ونتيجةٌ بلا دليلٍ ادّعاء**", r.ID)
		}
		for _, f := range r.Flows {
			if !strings.HasPrefix(f, "F-") || len(f) != 4 {
				t.Errorf("%s: تدفّقٌ غريبُ الشكل %q", r.ID, f)
			}
		}
	}
	t.Logf("CONCURRENCY-SENSITIVE FLOWS MAPPED = %d/10", len(All))
}

// TestEveryMappedTestExists **كلُّ اختبارٍ مذكورٍ موجودٌ في الشجرة.**
//
// **ومرجعٌ إلى اختبارٍ لا وجودَ له يجعل الخريطةَ تكذب** — وهي تُقرأ في
// `P-9` و`P-10` بوصفها حقيقة.
func TestEveryMappedTestExists(t *testing.T) {
	var body strings.Builder
	for _, dir := range []string{"../qa"} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("قراءةُ %s: %v", dir, err)
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			b, err := os.ReadFile(dir + "/" + e.Name())
			if err != nil {
				t.Fatalf("قراءةُ %s: %v", e.Name(), err)
			}
			body.Write(b)
		}
	}
	src := body.String()
	for _, name := range Tests() {
		if !strings.Contains(src, "func "+name+"(") {
			t.Errorf("الخريطةُ تشير إلى اختبارٍ لا وجودَ له: %s", name)
		}
	}
	t.Logf("MAPPED TESTS = %d · كلُّها موجودة", len(Tests()))
}

// TestNoFalsePass **ولا `PASS` لِما لا يُثبَت محلّيّاً.**
func TestNoFalsePass(t *testing.T) {
	for _, r := range All {
		if r.Where != Local && r.Result == Pass {
			t.Errorf("%s: نتيجةٌ PASS وموضعُ إثباته %s — **ادّعاءُ اكتمالٍ لم يقع**",
				r.ID, r.Where)
		}
	}
	c := Snapshot()["counts"].(Counts)
	if c.Local+c.RequiresP0+c.DeferredP8 != c.Total {
		t.Errorf("مجموعُ مواضع الإثبات %d لا يساوي %d",
			c.Local+c.RequiresP0+c.DeferredP8, c.Total)
	}
	t.Logf("PROVEN LOCALLY = %d · REQUIRES P-0 = %d · DEFERRED TO P-8 = %d",
		c.Local, c.RequiresP0, c.DeferredP8)
}
