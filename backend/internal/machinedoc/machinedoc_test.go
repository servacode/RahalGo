package machinedoc

import (
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestMachineDocIsCurrent **يُسقط البناءَ إن شاخت الوثيقة.**
//
// **وهذا هو الفرقُ كلُّه** بين هذا الملفّ و`ORDER_TRANSITIONS_55.md`: ذاك
// يزعم أنّه مولَّدٌ ويحرسه عدُّ صفوفٍ — **فيبقى أخضرَ ولو بُدِّل كلُّ دورٍ
// في كلّ صفّ.** وهذا يُعيد التوليدَ ويقارن الحرفَ بالحرف.
func TestMachineDocIsCurrent(t *testing.T) {
	raw, err := os.ReadFile(DocPath)
	if err != nil {
		t.Fatalf("لم تُقرأ %s: %v", DocPath, err)
	}
	out, err := Render(string(raw))
	if err != nil {
		t.Fatalf("التوليدُ أخطأ: %v", err)
	}
	if out != string(raw) {
		t.Errorf("**%s شاخت** — نادِ: cd backend && go run ./cmd/machinedoc", DocPath)
	}

	js, err := JSON()
	if err != nil {
		t.Fatalf("JSON أخطأ: %v", err)
	}
	cur, err := os.ReadFile(JSONPath)
	if err != nil {
		t.Fatalf("لم يُقرأ %s: %v", JSONPath, err)
	}
	if string(cur) != js {
		t.Errorf("**%s شاخ** — نادِ: cd backend && go run ./cmd/machinedoc", JSONPath)
	}
}

// TestMachineCoversEveryStatus **ولا حالةٌ تُنسى.**
//
// **فجدولٌ ينقصه صفٌّ لا يُرى ناقصاً** — يُقرأ كاملاً.
func TestMachineCoversEveryStatus(t *testing.T) {
	m := orders.BuildMachine()
	if len(m.Statuses) != 14 {
		t.Errorf("الحالاتُ %d والمنتظَرُ 14", len(m.Statuses))
	}
	if len(m.Terminal) != 5 {
		t.Errorf("النهائيّاتُ %d والمنتظَرُ 5", len(m.Terminal))
	}
	seen := map[string]bool{}
	for _, s := range m.Statuses {
		if s.Label == "" {
			t.Errorf("حالةٌ بلا لفظ: %s", s.Code)
		}
		if seen[s.Code] {
			t.Errorf("حالةٌ مكرّرة: %s", s.Code)
		}
		seen[s.Code] = true
	}
	// **وكلُّ طرَفِ حدٍّ حالةٌ معروفة** — فحدٌّ إلى حالةٍ لا وجودَ لها
	// **خطأُ كتابةٍ لا يُكشف إلّا هكذا.**
	for _, e := range m.Edges {
		if !seen[e.From] {
			t.Errorf("حدٌّ من حالةٍ مجهولة: %s", e.From)
		}
		if !seen[e.To] {
			t.Errorf("حدٌّ إلى حالةٍ مجهولة: %s", e.To)
		}
	}
}

// TestMachineEffectiveIsSubsetOfDeclaredPlusAdmin **التنقيةُ لا تُوسّع.**
//
// **وهذا هو ما يمنع خطأً صامتاً**: لو صار `rolesUnderMode` يُضيف دوراً بدل
// أن يُسقطه — **لَمَلَك الحدَّ من لا يملكه** ولم يشتكِ شيء. فيُقاس: كلُّ
// دورٍ فعليٍّ إمّا مُصرَّحٌ في الخريطة **أو المالك** (وله تجاوزٌ بنصّ
// `statuses.go:330`).
func TestMachineEffectiveIsSubsetOfDeclaredPlusAdmin(t *testing.T) {
	m := orders.BuildMachine()
	for _, e := range m.Edges {
		declared := map[string]bool{"admin": true}
		for _, r := range e.DeclaredRoles {
			declared[r] = true
		}
		for _, mode := range []struct {
			name string
			eff  []string
		}{
			{"platform", e.EffectivePlatform},
			{"merchants", e.EffectiveMerchants},
		} {
			for _, r := range mode.eff {
				if !declared[r] {
					t.Errorf("%s %s→%s في %s: الدورُ %q فعليٌّ وليس مُصرَّحاً",
						e.Kind, e.From, e.To, mode.name, r)
				}
			}
		}
	}
}

// TestMachinePlatformNeverWiderThanMerchants **وضعُ المنصّة لا يزيد.**
//
// **لأنّ `merchants` لا يُنقّي شيئاً** (`modes.go:123`) و`platform` يُنقّي —
// **فما يملكه أحدٌ في المنصّة يملكه في المتاجر ولا عكس.** ومن كسر هذا
// كسر معنى الوضعَين.
func TestMachinePlatformNeverWiderThanMerchants(t *testing.T) {
	m := orders.BuildMachine()
	for _, e := range m.Edges {
		wide := map[string]bool{}
		for _, r := range e.EffectiveMerchants {
			wide[r] = true
		}
		for _, r := range e.EffectivePlatform {
			if !wide[r] {
				t.Errorf("%s %s→%s: %q يملكها في المنصّة ولا يملكها في المتاجر",
					e.Kind, e.From, e.To, r)
			}
		}
	}
}

// TestMachineDeclaredRowsCiteASource **ولا سطرَ مُصرَّحٌ بلا مصدر.**
//
// **فالمُصرَّحُ ما لا يُشتَقّ** — ومن قرأه لا يملك إلّا أن يفتح الملفَّ
// ليتحقّق. **فإن لم يكن للسطر ملفٌّ وسطرٌ لم يكن عقداً بل رأياً.**
func TestMachineDeclaredRowsCiteASource(t *testing.T) {
	m := orders.BuildMachine()
	groups := map[string][]orders.MachineDeclared{
		"mode_rules":     m.ModeRules,
		"guards":         m.Guards,
		"auto_rules":     m.AutoRules,
		"implicit_edges": m.ImplicitEdges,
	}
	for name, rows := range groups {
		if len(rows) == 0 {
			t.Errorf("%s فارغة", name)
		}
		for _, d := range rows {
			if d.ID == "" || d.What == "" {
				t.Errorf("%s: سطرٌ بلا معرّفٍ أو وصف: %+v", name, d)
			}
			if !strings.Contains(d.Source, ".go:") {
				t.Errorf("%s/%s: المصدرُ %q بلا ملفٍّ وسطر", name, d.ID, d.Source)
			}
		}
	}
}
