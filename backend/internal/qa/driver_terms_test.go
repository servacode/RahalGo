package qa

// ══════════════════════════════════════════════════════════════════════
// **موافقةُ السائق على شروط العمل قبل فتح دوامه** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ٧)
// ══════════════════════════════════════════════════════════════════════
//
// **التطبيقُ الجديد (بناء ٢٢ فما فوق) لا يفتح الدوامَ قبل الموافقة**، **والقديمُ وما لا
// ترويسةَ له لا يُقفلان.** والموافقةُ بالنسخة الحاليّة وحدَها، مرّةً واحدة، ومعها بصمةُ
// النصّ ونسخةُ التطبيق. **والبندُ في النصّ أبداً** — ولو عدّل المالكُ الشروطَ بلاه.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

func TestDriverTerms_ShiftGateAndAccept(t *testing.T) {
	h := New(t)
	h.Setting("drivers.require_whatsapp", "false")
	h.Setting("drivers.terms_version", "1")
	drv := h.NewUser("driver")
	app := func(build string) map[string]string {
		return map[string]string{"X-RahalGo-Client": "android-driver", "X-RahalGo-Version": build}
	}
	shift := func(on bool, headers map[string]string) Res {
		t.Helper()
		return h.Call("POST", "/api/v1/driver/shift", drv.Token, map[string]any{"on": on}, headers)
	}
	off := func() {
		t.Helper()
		if r := shift(false, nil); r.Code != 200 {
			t.Fatalf("إغلاقُ الدوام: %s", r)
		}
	}

	if me := h.GET("/api/v1/driver/me", drv.Token).JSON(); me["terms_required"] != true || me["terms_version"] != float64(1) {
		t.Fatalf("**/driver/me لا يقول إنّ عليه الموافقة**: %v / %v", me["terms_required"], me["terms_version"])
	}
	// (١) القديمُ وما بلا ترويسةٍ لا يُقفلان.
	if r := shift(true, app("21")); r.Code != 200 {
		t.Fatalf("**تطبيقُ سائقٍ قديمٍ (21) قُفل دوامُه**: %s", r)
	}
	off()
	if r := shift(true, nil); r.Code != 200 {
		t.Fatalf("**نداءٌ بلا ترويسةٍ قُفل**: %s", r)
	}
	off()
	// (٢) الجديدُ يُقفل قبل الموافقة.
	if r := shift(true, app("22")); r.Code != 409 || r.Err() != "driver_terms_required" {
		t.Fatalf("**التطبيقُ الجديد (22) فتح الدوامَ قبل الموافقة**: %s", r)
	}
	// (٣) الموافقةُ بنسخةٍ غير الحاليّة تُردّ.
	if r := h.Call("POST", "/api/v1/driver/terms/accept", drv.Token, map[string]any{"version": 2}, app("22")); r.Code != 409 || r.Err() != "terms_version_changed" {
		t.Fatalf("**موافقةٌ على نسخةٍ غير الحاليّة قُبلت**: %s", r)
	}
	// (٤) الموافقةُ مرّتين ⇒ صفٌّ واحدٌ ببصمةٍ ونسخة تطبيق.
	for i := 0; i < 2; i++ {
		if r := h.Call("POST", "/api/v1/driver/terms/accept", drv.Token, map[string]any{"version": 1}, app("22")); r.Code != 200 {
			t.Fatalf("الموافقة: %s", r)
		}
	}
	var rows int
	var sha string
	var build *int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), max(text_sha256), max(app_build) FROM terms_acceptances
		WHERE user_id = $1::uuid AND role = 'driver' AND version = 1`, drv.ID).Scan(&rows, &sha, &build); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || len(sha) != 64 || build == nil || *build != 22 {
		t.Fatalf("**الموافقةُ**: صفوف=%d بصمة=%q بناء=%v", rows, sha, build)
	}
	// (٥) وبعدها يُفتح الدوام.
	if r := shift(true, app("22")); r.Code != 200 {
		t.Fatalf("**وافق ولم يُفتح دوامُه**: %s", r)
	}
	off()
	if me := h.GET("/api/v1/driver/me", drv.Token).JSON(); me["terms_required"] != false {
		t.Fatalf("**وافق و/driver/me ما زال يطلب الموافقة**")
	}

	// (٦) نصٌّ عدّله المالكُ بلا البند ⇒ البندُ في /driver/terms و/public/contact.
	edited, _ := json.Marshal("عملُك معنا\nنصٌّ عدّله المالكُ بلا البند.")
	h.Setting("page.driver_terms_text", string(edited))
	terms := h.GET("/api/v1/driver/terms", drv.Token).JSON()
	text, _ := terms["text"].(string)
	if !strings.Contains(text, "نصٌّ عدّله المالكُ") || !strings.Contains(text, settings.DriverTermsClause) {
		t.Fatalf("**/driver/terms بلا البند**: %q", text)
	}
	if terms["clause"] != settings.DriverTermsClause {
		t.Errorf("البندُ: %v", terms["clause"])
	}
	contact := h.GET("/api/v1/public/contact", "").JSON()
	if s, _ := contact["driver_terms_text"].(string); !strings.Contains(s, settings.DriverTermsClause) {
		t.Fatalf("**/public/contact بلا البند**: %q", s)
	}
}
