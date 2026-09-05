package failmap

import (
	"os"
	"strings"
	"testing"
)

// TestNineFlowsMapped **`PARTIAL-FAILURE FLOWS = 9`** — ولا واحدَ ينقص.
func TestNineFlowsMapped(t *testing.T) {
	if len(All) != 9 {
		t.Fatalf("الخريطةُ فيها %d — والعددُ المجمَّد 9", len(All))
	}
	seen := map[string]bool{}
	for _, f := range All {
		if seen[f.ID] {
			t.Errorf("معرّفٌ مكرَّر: %s", f.ID)
		}
		seen[f.ID] = true
		for _, s := range []struct{ n, v string }{
			{"Title", f.Title}, {"AtomicBoundary", f.AtomicBoundary},
			{"Expected", f.Expected}, {"Observed", f.Observed},
			{"UserVisible", f.UserVisible}, {"Recovery", f.Recovery},
			// **ولا نتيجةَ بلا دليلٍ مطبوع** — وخريطةٌ تسبق القياسَ تكذب.
			{"Evidence", f.Evidence},
		} {
			if s.v == "" {
				t.Errorf("%s: %s ناقص", f.ID, s.n)
			}
		}
		if len(f.Steps) == 0 {
			t.Errorf("%s: بلا خطوات", f.ID)
		}
		if len(f.Tests) == 0 {
			t.Errorf("%s: بلا اختبارٍ يحرسه", f.ID)
		}
		if len(f.Failpoints) == 0 {
			t.Errorf("%s: بلا نقطةِ فشل", f.ID)
		}
		if f.AdminVisible == "" || f.Result == "" || f.Where == "" {
			t.Errorf("%s: حقلُ تصنيفٍ ناقص", f.ID)
		}
	}
	t.Logf("PARTIAL-FAILURE FLOWS MAPPED = %d/9", len(All))
}

// TestEveryMappedTestExists **خريطةٌ تشير إلى اختبارٍ لا وجودَ له تكذب.**
func TestEveryMappedTestExists(t *testing.T) {
	var body strings.Builder
	entries, err := os.ReadDir("../qa")
	if err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		b, err := os.ReadFile("../qa/" + e.Name())
		if err != nil {
			t.Fatalf("قراءة %s: %v", e.Name(), err)
		}
		body.Write(b)
	}
	src := body.String()
	for _, n := range Tests() {
		if !strings.Contains(src, "func "+n+"(") {
			t.Errorf("اختبارٌ لا وجودَ له: %s", n)
		}
	}
	t.Logf("MAPPED TESTS = %d · وFAILPOINTS = %d", len(Tests()), len(Failpoints()))
}

// TestNoFalsePass **ولا `PASS` لِما لا يُثبَت محلّيّاً.**
func TestNoFalsePass(t *testing.T) {
	for _, f := range All {
		if f.Where != Local && (f.Result == Pass || f.Result == DefectReproduced) {
			t.Errorf("%s: نتيجةٌ %s وموضعُه %s — **ادّعاءُ إثباتٍ لم يقع**",
				f.ID, f.Result, f.Where)
		}
	}
	c := Snapshot()["counts"].(Counts)
	t.Logf("PROVEN LOCALLY = %d · REQUIRES P-0 = %d · REQUIRES P-7 = %d · REQUIRES P-8 = %d",
		c.Local, c.RequiresP0, c.RequiresP7, c.RequiresP8)
	t.Logf("VISIBLE = %d · PARTIAL = %d · INVISIBLE = %d",
		c.Visible, c.PartialVisible, c.Invisible)
}
