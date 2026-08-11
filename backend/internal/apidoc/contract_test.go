package apidoc

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حارسُ العقد — يمنع أن يشيخ**
// ══════════════════════════════════════════════════════════════════════
//
// **عقدٌ يشيخ أسوأُ من ألّا يكون عقد.** من قرأه وثق به، **فبنى نموذجَه على
// حقلٍ لم يعد موجوداً** — والشاشةُ تعرض فراغاً بلا خطأٍ في بناءٍ ولا سطرٍ
// في سجلّ.
//
// **فمن غيّر مساراً أو حقلاً ونسي التوليدَ يسقط بناؤه هنا** — كما يفعل
// `TestTruthDocIsCurrent` بوثيقة الحقيقة.

func repoRoot(t *testing.T) string {
	t.Helper()
	// من `backend/internal/apidoc` إلى جذر المستودع.
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("تعذّر حسابُ الجذر: %v", err)
	}
	return root
}

func TestContractIsCurrent(t *testing.T) {
	root := repoRoot(t)
	c, err := Build(filepath.Join(root, "backend", "internal", "server"))
	if err != nil {
		t.Fatalf("تعذّر بناءُ العقد: %v", err)
	}
	want, err := Render(c)
	if err != nil {
		t.Fatalf("تعذّرت الكتابة: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(root, Path))
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ %s — **ولّده بـ`go run ./cmd/apidoc`**: %v", Path, err)
	}
	if !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(want)) {
		t.Fatalf("عقدُ الـAPI شاخ — **ولّده بـ`go run ./cmd/apidoc` وأودِعه**.\n"+
			"وحقلٌ تبدّل في المحرّك ولم يُنقل إلى العقد ينكسر في ثلاثة عملاءَ بصمت.\n"+
			"(الملفّ %s)", Path)
	}
}

// TestContract_CoversEveryRoute **ولا نقطةَ خارجَ العقد.**
//
// **ومسارٌ يُسجَّل ولا يظهر في العقد** يعني أنّ الأداةَ لم تفهم شكلَه —
// **وصمتُها أخطرُ من سقوطها**: العميلُ يبني على عقدٍ ناقصٍ يظنّه كاملاً.
func TestContract_CoversEveryRoute(t *testing.T) {
	root := repoRoot(t)
	dir := filepath.Join(root, "backend", "internal", "server")
	routes, err := Routes(dir)
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ المسارات: %v", err)
	}
	c, err := Build(dir)
	if err != nil {
		t.Fatalf("تعذّر بناءُ العقد: %v", err)
	}
	if len(routes) != len(c.Endpoints) {
		t.Fatalf("المسارات %d والعقدُ %d — **نقطةٌ ضاعت في الطريق**",
			len(routes), len(c.Endpoints))
	}
	// **ولا مسارَ بلا معالِجٍ معروف** — عدا المكتوبةَ في موضعها.
	for _, e := range c.Endpoints {
		if e.Handler == "" {
			t.Fatalf("نقطةٌ بلا معالِج: %s %s", e.Method, e.Path)
		}
	}
}

// TestContract_AuthIsNotUnderstated **وما هو محميٌّ يُقال إنّه محميّ.**
//
// **وأخطرُ ما يكذب فيه عقدٌ هو الصلاحيات**: من قرأ أنّ نقطةً مفتوحةٌ وهي
// محميّةٌ يبني شاشةً لا تعمل، **ومن قرأ أنّها محميّةٌ وهي مفتوحةٌ يظنّ
// المنصّةَ آمنةً وهي ليست.**
//
// **والفحصُ على عيّنةٍ معروفةٍ باليد** — لا على ما تولّده الأداةُ نفسُها،
// **وإلّا صدّقت نفسَها.**
func TestContract_AuthIsNotUnderstated(t *testing.T) {
	routes, err := Routes(filepath.Join(repoRoot(t), "backend", "internal", "server"))
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	byKey := map[string]Route{}
	for _, r := range routes {
		byKey[r.Method+" "+r.Path] = r
	}
	cases := []struct {
		key   string
		auth  bool
		roles []string
	}{
		// **سجلُّ الأحداث للأدمن والمالية دون العمليات** — فيه مبالغُ
		// التعويضات، وموظّفُ العمليات ليس طرفاً في المال.
		{"GET /api/v1/admin/audit", true, []string{"admin", "finance"}},
		{"GET /api/v1/admin/treasury-candidates", true, []string{"admin"}},
		{"GET /api/v1/driver/me", true, []string{"driver"}},
		// **والصحّةُ مفتوحة** — يقرؤها مُوازِنُ الحِمل قبل أن يكون ثمّة توكن.
		{"GET /healthz", false, nil},
	}
	for _, c := range cases {
		got, ok := byKey[c.key]
		if !ok {
			t.Fatalf("نقطةٌ معروفةٌ غابت عن العقد: %s", c.key)
		}
		if got.Auth != c.auth {
			t.Fatalf("%s: المصادقةُ %v والمنتظَر %v", c.key, got.Auth, c.auth)
		}
		if len(got.Roles) != len(c.roles) {
			t.Fatalf("%s: الأدوارُ %v والمنتظَر %v — **وعقدٌ يكذب في الصلاحيات "+
				"أخطرُ من عقدٍ ناقص**", c.key, got.Roles, c.roles)
		}
		for i := range c.roles {
			if got.Roles[i] != c.roles[i] {
				t.Fatalf("%s: الأدوارُ %v والمنتظَر %v", c.key, got.Roles, c.roles)
			}
		}
	}
}
