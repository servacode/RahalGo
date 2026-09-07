package impact

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **طبقةٌ مشتركةٌ بلا خريطةِ أثرٍ تكذب بصمت**
// ══════════════════════════════════════════════════════════════════════
//
// **وقع ثلاثَ مرّات**: منعُ التكرار قال تدفّقين وهو يمسّ ستّة · والوسائطُ
// قالت تدفّقين · **والتدقيقُ قال صفراً** لتغييرٍ يمسّ ستّةَ أفعالٍ
// ماليّة.
//
// **وثلاثةٌ تعني نمطاً لا سهواً**: **ملفٌّ لا تذكره قاعدةٌ يقع في العامّ
// فيُقرأ أثرُه أقلَّ ممّا هو** — **وتُشغَّل حزمةٌ أضيقُ ممّا يلزم.**
//
// **والأخطرُ صمتُه**: **`FLOWS=[]` تُقرأ «لا أثر» وهي «لا خريطة».**

// sharedLayers **طبقاتٌ يمسّها كلُّ سطحٍ تقريباً** — **ولا يجوز أن
// تُقرأ بلا تدفّقات.**
//
// **والقائمةُ تُوصَف بما هي لا بأسماء ملفّاتٍ تشيخ**: **كلُّ ملفٍّ
// يمرّ به فعلٌ حسّاسٌ أو يُشارَك بين مجالات.**
var sharedLayers = []string{
	"backend/internal/server/audit.go",
	"backend/internal/server/audit_tx.go",
	"backend/internal/server/idempotency.go",
	"backend/internal/server/idempotency_tx.go",
	"backend/internal/server/middleware.go",
	"backend/internal/media/media.go",
}

// TestSharedLayersHaveImpactMapping **ولا طبقةَ مشتركةٌ بلا تدفّقات.**
func TestSharedLayersHaveImpactMapping(t *testing.T) {
	e := loadEngine(t)
	for _, f := range sharedLayers {
		res := e.Analyze("EXPLICIT_FILES", "", "", []string{f})
		t.Logf("%-52s ⇒ تدفّقاتٌ=%d", f, len(res.Flows))
		if len(res.Flows) == 0 {
			t.Errorf("**`%s` طبقةٌ مشتركةٌ بلا تدفّقاتٍ في خريطة الأثر** — "+
				"**فتُقرأ «لا أثر» وهي «لا خريطة»**، وتُشغَّل حزمةٌ أضيقُ "+
				"ممّا يلزم.", f)
		}
	}
}

// TestCriticalAuditActionsAreMapped **وكلُّ فعلٍ حسّاسٍ له تدفّقٌ.**
//
// **ومعجمُ الأفعال الحسّاسة يكبر** (`XG-35`) — **فمن أضاف فعلاً غداً
// وجب أن يكون لطبقته أثرٌ يُقرأ.**
func TestCriticalAuditActionsAreMapped(t *testing.T) {
	root := repoRootOf(t)
	b, err := os.ReadFile(filepath.Join(root, "backend/internal/server/audit_tx.go"))
	if err != nil {
		t.Skipf("لا معجمَ للأفعال الحسّاسة: %v", err)
	}
	// **والعدُّ على النمط الحقيقيّ**: مفاتيحُ المعجم مُصطفّةٌ بمسافات
	// (`"finance.wallet_apply":   true,`) — **فنمطٌ لاصقٌ يقرأ واحداً
	// وهي ستّة**، **وحارسٌ يقيس خطأً أسوأُ من لا حارس.**
	n := len(regexp.MustCompile(`"[a-z_]+\.[a-z_]+":\s*true,`).
		FindAllString(string(b), -1))
	e := loadEngine(t)
	res := e.Analyze("EXPLICIT_FILES", "", "",
		[]string{"backend/internal/server/audit_tx.go"})
	t.Logf("أفعالٌ حسّاسةٌ=%d · تدفّقاتُ طبقتها=%d", n, len(res.Flows))

	if n == 0 {
		t.Fatal("**المعجمُ فارغ** — ولا حارسَ بلا قائمة")
	}
	// **ولا يُشترَط تدفّقٌ لكلّ فعل** — **بل ألّا تكون الطبقةُ عمياء.**
	if len(res.Flows) < 3 {
		t.Errorf("**طبقةُ التدقيق مربوطةٌ بـ%d تدفّقاتٍ و%d أفعالٍ حسّاسة** — "+
			"**والخريطةُ أضيقُ من الواقع.**", len(res.Flows), n)
	}
}

// loadEngine محرّكُ الأثر على جذر المستودع.
func loadEngine(t *testing.T) *Engine {
	t.Helper()
	e, err := Load(repoRootOf(t))
	if err != nil {
		t.Fatalf("تحميلُ محرّك الأثر: %v", err)
	}
	return e
}

func repoRootOf(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("المجلّدُ الحاليّ: %v", err)
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}
