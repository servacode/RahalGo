package authzdoc

import (
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// TestAuthzDocIsCurrent **يُسقط البناءَ إن شاخ العقد.**
func TestAuthzDocIsCurrent(t *testing.T) {
	raw, err := os.ReadFile(DocPath)
	if err != nil {
		t.Fatalf("لم تُقرأ %s: %v", DocPath, err)
	}
	out, err := Render(string(raw))
	if err != nil {
		t.Fatalf("التوليدُ أخطأ: %v", err)
	}
	if out != string(raw) {
		t.Errorf("**%s شاخت** — نادِ: cd backend && go run ./cmd/authzdoc", DocPath)
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
		t.Errorf("**%s شاخ** — نادِ: cd backend && go run ./cmd/authzdoc", JSONPath)
	}
}

// TestEveryCapabilityIsAccountedFor **ولا قدرةٌ تحرس لا باباً ولا حقلاً.**
//
// **وهذا ليس تكراراً لحارس القدرات اليتيمة** (`role_matrix_test.go:476`):
// ذاك يمشي على الموجّه، **وهذا يقيس ما يُخرجه العقدُ نفسُه** — فلو أخطأ
// العدُّ في المُولِّد **خرج عقدٌ يقول إنّ قدرةً بلا حارسٍ وهي محروسة.**
func TestEveryCapabilityIsAccountedFor(t *testing.T) {
	c := Build()
	if len(c.Capabilities) != authz.Count() {
		t.Fatalf("العقدُ فيه %d قدرةً والمعجمُ %d", len(c.Capabilities), authz.Count())
	}
	for _, r := range c.Capabilities {
		if r.Description == "" {
			t.Errorf("قدرةٌ بلا وصف: %s", r.Code)
		}
		if r.GuardsRoutes == 0 && len(r.GuardsFields) == 0 && len(r.GuardsInHandler) == 0 {
			t.Errorf("**القدرةُ %q لا تحرس باباً ولا حقلاً** — يتيمةٌ حقّاً", r.Code)
		}
		if r.FieldScopedOnly && r.GuardsRoutes != 0 {
			t.Errorf("%s: حقليّةٌ ولها مسارات", r.Code)
		}
	}
}

// TestSensitiveActionsAreComplete **وكلُّ فعلٍ حسّاسٍ له فعلٌ وهدفٌ وقدرة.**
func TestSensitiveActionsAreComplete(t *testing.T) {
	c := Build()
	if len(c.Sensitive) != authz.SensitiveCount() {
		t.Fatalf("العقدُ فيه %d فعلاً والجدولُ %d",
			len(c.Sensitive), authz.SensitiveCount())
	}
	seen := map[string]bool{}
	for _, s := range c.Sensitive {
		if s.Action == "" || s.Method == "" || s.Pattern == "" {
			t.Errorf("فعلٌ ناقص: %+v", s)
		}
		if seen[s.Action] {
			t.Errorf("فعلٌ مكرّر: %s", s.Action)
		}
		seen[s.Action] = true
		// **وهدفٌ مُعلَنٌ بلا موضعٍ في المسار لا يُستخرَج.**
		if s.TargetType != "" && s.TargetSeg < 0 && !strings.Contains(s.Pattern, "{") {
			continue // هدفٌ بلا معرّفٍ في المسار — مقبولٌ (إنشاءٌ جديد)
		}
	}
}

// TestStepUpIsPasswordNotPin **والرمزُ السرّيُّ ليس إثباتاً — يُثبَّت نصّاً.**
//
// **لأنّ هذا هو الخطأُ الذي وقعتُ فيه** ووصفتُ النظامَ بغير ما هو. فإن صار
// الإثباتُ يوماً يقبل رمزاً سرّيّاً **فليسقط هذا الفحصُ** ولتُصحَّح الوثيقة،
// **لا أن يبقى العقدُ يقول القديم.**
func TestStepUpIsPasswordNotPin(t *testing.T) {
	c := Build()
	if c.StepUp.Factor != "password" {
		t.Errorf("عاملُ الإثبات %q والمنتظَرُ password", c.StepUp.Factor)
	}
	if c.StepUp.Header != "X-Step-Up" {
		t.Errorf("الترويسةُ %q", c.StepUp.Header)
	}
	if !c.StepUp.SingleUse {
		t.Error("الإذنُ يجب أن يكون لمرّةٍ واحدة")
	}
	if c.StepUp.PinIsNotStepUp == "" {
		t.Error("التصحيحُ يجب أن يبقى مكتوباً في العقد")
	}
}

// TestExemptionsCarryTheirReason **ولا استثناءٌ بلا سبب.**
//
// **وهو ما يضيع في `api/contract.json`** — يطوي الاثنين في «بحسب المفتاح».
func TestExemptionsCarryTheirReason(t *testing.T) {
	c := Build()
	if len(c.Exemptions) == 0 {
		t.Fatal("لا استثناءات — والمنتظَرُ اثنان")
	}
	for _, e := range c.Exemptions {
		if strings.TrimSpace(e.Reason) == "" {
			t.Errorf("استثناءٌ بلا سبب: %s", e.Pattern)
		}
	}
}

// TestRoleClassesCoverEveryClassifiedRole **وكلُّ دورٍ مصنَّفٍ يظهر مرّةً.**
func TestRoleClassesCoverEveryClassifiedRole(t *testing.T) {
	c := Build()
	seen := map[string]string{}
	for _, r := range c.RoleClasses {
		if r.Authority == "" {
			t.Errorf("صنفٌ بلا سلطةِ منح: %s", r.Class)
		}
		for _, role := range r.Roles {
			if prev, dup := seen[role]; dup {
				t.Errorf("الدورُ %q في صنفَين: %s و %s", role, prev, r.Class)
			}
			seen[role] = r.Class
			// **والصنفُ يُسأل من المصدر لا من الجدول** — فلو انحرف
			// الجدولُ عن `ClassOf` خرج عقدٌ يكذب.
			if got := string(authz.ClassOf(role)); got != r.Class {
				t.Errorf("الدورُ %q: العقدُ يقول %s و ClassOf يقول %s",
					role, r.Class, got)
			}
		}
	}
	if len(seen) == 0 {
		t.Fatal("لا أدوارَ مصنَّفة")
	}
}

// TestWebPolicyIsCurrent **صلاحيّاتُ الواجهة لا تشيخ عن جدول المحرّك.**
func TestWebPolicyIsCurrent(t *testing.T) {
	cur, err := os.ReadFile(WebPolicyPath)
	if err != nil {
		t.Fatalf("لم يُقرأ %s: %v", WebPolicyPath, err)
	}
	if string(cur) != WebPolicyTS() {
		t.Errorf("**%s شاخ** — نادِ: cd backend && go run ./cmd/authzdoc", WebPolicyPath)
	}
}
