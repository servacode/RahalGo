package fininv

import (
	"strings"
	"testing"
)

// TestEveryCheckIsComplete كلُّ فحصٍ يحمل ما يجعله مقروءاً — **ولا فحصَ مبهم.**
func TestEveryCheckIsComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range All {
		if c.ID == "" || c.Name == "" || c.Why == "" || c.SQL == "" {
			t.Errorf("فحصٌ ناقصٌ: %+v", c.ID)
		}
		if seen[c.ID] {
			t.Errorf("معرّفٌ مكرَّر: %s", c.ID)
		}
		seen[c.ID] = true
		if !strings.HasPrefix(c.ID, string(c.Family)+".") {
			t.Errorf("%s لا ينتمي إلى عائلته %s", c.ID, c.Family)
		}
		// **والسببُ ليس إعادةَ صياغةِ الاسم** — من كتب «الرصيدُ خاطئ»
		// في خانة «لماذا» لم يقل شيئاً.
		if c.Why == c.Name {
			t.Errorf("%s: السببُ إعادةُ صياغةٍ للاسم", c.ID)
		}
	}
}

// TestEveryFamilyHasChecks الاثنتا عشرةَ عائلةً التي حسمها المالكُ كلُّها ممثَّلة.
func TestEveryFamilyHasChecks(t *testing.T) {
	want := []Family{FI01, FI02, FI03, FI04, FI05, FI06, FI07, FI08, FI09, FI10, FI11, FI12, FI13}
	have := map[Family]int{}
	for _, c := range All {
		have[c.Family]++
	}
	for _, f := range want {
		if have[f] == 0 {
			t.Errorf("العائلة %s بلا فحصٍ واحد", f)
		}
	}
	if len(have) != len(want) {
		t.Errorf("عوائلُ %d — يُنتظر %d", len(have), len(want))
	}
}

// TestSelectDefaultsToProvable **الافتراضُ لا يشغّل ما لا يُثبَت** — فلا
// يُحسَب `NOT_IMPLEMENTED` نجاحاً لأنّ استعلامَه لم يردّ صفّاً.
func TestSelectDefaultsToProvable(t *testing.T) {
	for _, c := range Select() {
		if c.Status != ProvableNow {
			t.Errorf("%s (%s) شُغّل بلا طلبٍ صريح", c.ID, c.Status)
		}
	}
	if len(Select()) == len(All) {
		t.Fatal("الانتقاءُ لم يستثنِ شيئاً — والحارسُ بلا معنى")
	}
	if got := Select("FI-06"); len(got) != 4 {
		t.Errorf("FI-06 ردّت %d فحصاً — يُنتظر 4", len(got))
	}
	if got := Select("FI-06.a"); len(got) != 1 {
		t.Errorf("انتقاءُ فحصٍ بعينه ردّ %d", len(got))
	}
}

// TestUnprovenChecksAreInert **ما لا يُثبَت لا يدّعي نجاحاً**: استعلامُه
// لا يمسّ جدولاً، فلا يصحّ أن يُقرأ صفرُ صفوفِه سلامةً.
func TestUnprovenChecksAreInert(t *testing.T) {
	for _, c := range All {
		if c.Status == ProvableNow {
			continue
		}
		if c.Ops {
			t.Errorf("%s غيرُ مُثبَتٍ ويُشغَّل في التشغيل", c.ID)
		}
		// **إلّا FI-09.a**: استعلامُه حقيقيٌّ ويكشف XG-10 على قاعدةٍ حيّة،
		// **ولذلك يُنادى صراحةً في الاختبار بوصفه `EXPECTED_FAIL`.**
		if c.ID == "FI-09.a" {
			continue
		}
		if !strings.Contains(c.SQL, "WHERE false") {
			t.Errorf("%s غيرُ مُثبَتٍ واستعلامُه يمسّ بيانات: %s", c.ID, c.SQL)
		}
	}
}

// TestEveryKindHasContract **البندُ ٧** — كلُّ نوعٍ في الخريطة عقدُه مكتمل.
func TestEveryKindHasContract(t *testing.T) {
	for k, c := range Kinds {
		if k != c.Kind {
			t.Errorf("مفتاحٌ %q لعقدِ %q", k, c.Kind)
		}
		if c.Semantics == "" {
			t.Errorf("%s: بلا دلالةٍ ماليّة", k)
		}
		if len(c.Creators) == 0 {
			t.Errorf("%s: بلا موضعِ كتابة", k)
		}
		if len(c.Invariants) == 0 {
			t.Errorf("%s: بلا ثابتٍ يغطّيه", k)
		}
		if c.Sign != "+" && c.Sign != "-" && c.Sign != "±" {
			t.Errorf("%s: اتّجاهٌ غيرُ مفهومٍ %q", k, c.Sign)
		}
		if c.RefRequired && c.RefTarget == "" {
			t.Errorf("%s: يشترط مرجعاً ولا يقول إلى أين", k)
		}
		for _, id := range c.Invariants {
			if len(Select(id)) == 0 {
				t.Errorf("%s: يشير إلى ثابتٍ لا وجودَ له %q", k, id)
			}
		}
	}
}

// TestCreatorSitesExist مواضعُ الكتابةِ المذكورةُ ملفّاتٌ قائمة.
//
// **ولا يُطابَق رقمُ السطر** — يزحف بكلّ تحرير، **وحارسٌ يسقط بلا خللٍ
// حقيقيٍّ يُطفَأ ثمّ لا يُقرأ.** والملفُّ يكفي: **من حذفه نقل المسار.**
func TestCreatorSitesExist(t *testing.T) {
	for k, c := range Kinds {
		for _, site := range c.Creators {
			file := site
			if i := strings.LastIndex(site, ":"); i > 0 {
				file = site[:i]
			}
			if !strings.HasPrefix(file, "internal/") || !strings.HasSuffix(file, ".go") {
				t.Errorf("%s: موضعٌ لا يشبه مساراً %q", k, site)
			}
		}
	}
}

// TestChecksReferenceKnownFlows الإحالاتُ إلى التدفّقات صحيحةُ الشكل.
func TestChecksReferenceKnownFlows(t *testing.T) {
	for _, c := range All {
		for _, f := range c.Flows {
			if !strings.HasPrefix(f, "F-") || len(f) != 4 {
				t.Errorf("%s: تدفّقٌ غريبُ الشكل %q", c.ID, f)
			}
		}
		for _, k := range c.Kinds {
			if k == "*" {
				continue
			}
			if _, ok := Kinds[k]; !ok {
				t.Errorf("%s: نوعٌ بلا عقد %q", c.ID, k)
			}
		}
	}
}

// TestSnapshotCounts الصورةُ الآليّةُ تعدّ ما في المحرّك لا ما يُظنّ فيه.
func TestSnapshotCounts(t *testing.T) {
	e := Snapshot()
	if e.Counts.Checks != len(All) {
		t.Errorf("عدُّ الفحوص %d — والمحرّكُ فيه %d", e.Counts.Checks, len(All))
	}
	if e.Counts.Kinds != len(Kinds) {
		t.Errorf("عدُّ الأنواع %d — والخريطةُ فيها %d", e.Counts.Kinds, len(Kinds))
	}
	sum := e.Counts.ProvableNow + e.Counts.NotImplemented + e.Counts.DeferredP5 + e.Counts.DeferredP6
	if sum != e.Counts.Checks {
		t.Errorf("مجموعُ الحالات %d لا يساوي عددَ الفحوص %d", sum, e.Counts.Checks)
	}
	if len(e.SettingIDs) == 0 {
		t.Fatal("لا إعداداتٍ ماليّةً في الصورة")
	}
}

// TestContractedKindsSQLIsGenerated قائمةُ SQL مولَّدةٌ من الخريطة لا مكتوبةٌ بيد.
func TestContractedKindsSQLIsGenerated(t *testing.T) {
	for k := range Kinds {
		if !strings.Contains(contractedKindsSQL, "'"+k+"'") {
			t.Errorf("النوع %s غائبٌ عن قائمةِ الاستعلام", k)
		}
	}
	if n := strings.Count(contractedKindsSQL, "'") / 2; n != len(Kinds) {
		t.Errorf("قائمةُ الاستعلام فيها %d وفي الخريطة %d", n, len(Kinds))
	}
}

// TestKindDriftDetectsBothDirections **حارسُ الانحراف يُثبَت بالتجربة** —
// نوعٌ جديدٌ في القاعدة يُمسَك، وعقدٌ لنوعٍ زال يُمسَك.
func TestKindDriftDetectsBothDirections(t *testing.T) {
	base := ContractedKinds()

	if m, s := KindDrift(base); len(m) != 0 || len(s) != 0 {
		t.Fatalf("لا انحرافَ ومع ذلك: ناقصٌ %v · شائخٌ %v", m, s)
	}

	added := append(append([]string(nil), base...), "cashback")
	m, s := KindDrift(added)
	if len(m) != 1 || m[0] != "cashback" {
		t.Errorf("نوعٌ جديدٌ بلا عقدٍ لم يُمسَك: %v", m)
	}
	if len(s) != 0 {
		t.Errorf("شائخٌ لا محلَّ له: %v", s)
	}

	removed := base[1:]
	m, s = KindDrift(removed)
	if len(s) != 1 || s[0] != base[0] {
		t.Errorf("عقدٌ لنوعٍ زال لم يُمسَك: %v", s)
	}
	if len(m) != 0 {
		t.Errorf("ناقصٌ لا محلَّ له: %v", m)
	}
}
