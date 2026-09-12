package qa

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **شكلُ تحدّي التأكيد — عقدٌ يقرؤه المتصفّح** (عطبُ إنتاجِ ٢٠٢٦-٠٩-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع
//
// **ردَّ المحرّكُ ٤٠٣ بجسمٍ صحيحِ المحتوى مغلَّفٍ في `{"data":…}`** —
// **و`httpx.JSON` غلافُ نجاحٍ لا غلافُ خطأ.** فقرأ العميلُ `error` في
// أعلى الجسم فلم يجده، **فصار الخطأُ «داخليّاً»**، **ولم تُفتح نافذةُ
// كلمةِ المرور قطُّ.**
//
//	POST /api/v1/admin/roles ⇒ 403 · ١٦٤ بايتاً · ثلاثَ مرّات
//	وقرأ المالكُ: «حدث خطأ غير متوقع، حاول مجددا»
//	وصفرُ نداءٍ إلى /admin/step-up في سجلّ الوصول
//
// # ولماذا لم يُكشف
//
// **أُثبت التدفّقُ في ٧٠ب-و١ بـcurl وإثباتٍ صُنع باليد** — **فلم يمرّ
// أحدٌ بالجواب الذي يعتمد عليه المتصفّح.** **وصفرُ اختباراتٍ كانت تلمس
// `step_up_required`.**
//
// **فحصٌ يصنع الإثباتَ بنفسه لا يحرس بابَ الإثبات.**
//
// # وما يُحرَس هنا
//
//	الحالُ ٤٠٣ · و`error.code` و`step_up` في الأعلى · **ولا مفتاحَ
//	`data` إطلاقاً** · ولأفعالٍ من أصنافٍ مختلفة · ومسارٌ مركزيٌّ واحد.

// stepUpChallenge **ينادي فعلاً حسّاساً بلا إثباتٍ ويردّ جسمَ التحدّي.**
//
// **والترويسةُ تُمرَّر فارغةً** — **فلا يصنع المِسنَدُ إثباتاً**، وهذا
// هو ما يفعله المتصفّحُ في أوّل نداء.
func stepUpChallenge(t *testing.T, hh *Harness, token, method, path string, body any) (int, map[string]any) {
	t.Helper()
	res := withGrant(hh, token, method, path, body, "")
	var raw map[string]any
	if err := json.Unmarshal(res.Body, &raw); err != nil {
		t.Fatalf("**جسمُ التحدّي ليس JSON صالحاً**: %v — %s", err, res.Body)
	}
	return res.Code, raw
}

// assertChallengeShape **يقيس العقدَ حرفاً حرفاً.**
func assertChallengeShape(t *testing.T, label string, code int, raw map[string]any, wantAction string) {
	t.Helper()
	if code != http.StatusForbidden {
		t.Errorf("**%s: التحدّي ليس ٤٠٣** — %d", label, code)
	}
	// ── ولا غلافَ نجاحٍ على خطأ ──────────────────────────────────
	if _, wrapped := raw["data"]; wrapped {
		t.Errorf("**%s: التحدّي مغلَّفٌ في `data`** — "+
			"**والعميلُ يقرأ `error` في الأعلى فلا يجده**، "+
			"**فيصير الخطأُ «داخليّاً» ولا تُفتح نافذةُ الكلمة.** "+
			"(عطبُ إنتاجِ ٢٠٢٦-٠٩-١٢.)", label)
	}
	// ── ورمزُ الخطأ في الأعلى ────────────────────────────────────
	errObj, ok := raw["error"].(map[string]any)
	if !ok {
		t.Fatalf("**%s: لا `error` في أعلى الجسم** — %v", label, keysOf(raw))
	}
	if got, _ := errObj["code"].(string); got != "step_up_required" {
		t.Errorf("**%s: رمزُ الخطأ %q لا `step_up_required`**", label, got)
	}
	if got, _ := errObj["message_key"].(string); got != "errors.step_up_required" {
		t.Errorf("**%s: مفتاحُ الرسالة %q**", label, got)
	}
	// ── ووصفُ الفعل في الأعلى كذلك ───────────────────────────────
	su, ok := raw["step_up"].(map[string]any)
	if !ok {
		t.Fatalf("**%s: لا `step_up` في أعلى الجسم** — "+
			"**فلا تعرف اللوحةُ ما تعرضه على من يؤكّد** — %v", label, keysOf(raw))
	}
	if got, _ := su["action"].(string); got != wantAction {
		t.Errorf("**%s: الفعل %q والمنتظَر %q**", label, got, wantAction)
	}
	if _, has := su["target_type"]; !has {
		t.Errorf("**%s: `step_up.target_type` غائب**", label)
	}
	if _, has := su["target_id"]; !has {
		t.Errorf("**%s: `step_up.target_id` غائب**", label)
	}
}

// TestSUC1_ChallengeShapeIsTopLevel **العقدُ على مسارٍ حقيقيٍّ واحد.**
//
// **وهو المسارُ الذي سقط في الإنتاج** — `POST /admin/roles` بلا إثبات.
func TestSUC1_ChallengeShapeIsTopLevel(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	code, raw := stepUpChallenge(t, hh, admin.Token, "POST", "/api/v1/admin/roles",
		map[string]any{"code": "suc_role", "name": "دورُ فحص"})
	t.Logf("جسمُ التحدّي: %s", strings.TrimSpace(string(mustJSON(t, raw))))
	assertChallengeShape(t, "admin.role_create", code, raw, "admin.role_create")
}

// TestSUC2_ChallengeShapeAcrossActionClasses **أصنافٌ لا فعلٌ واحد.**
//
// **وإصلاحٌ مركزيٌّ يُثبَت على أصنافٍ مختلفة**: إدارةُ أدوارٍ · وأمنُ
// حسابٍ · ومالٌ يتحرّك · وإعدادٌ ذو أثر. **ولا تُكرَّر الخمسةَ عشرَ
// شاشةً شاشة** — المسارُ واحدٌ، والأصنافُ تشهد.
func TestSUC2_ChallengeShapeAcrossActionClasses(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")

	cases := []struct {
		label  string
		method string
		path   string
		body   any
		action string
	}{
		{"إنشاءُ دور", "POST", "/api/v1/admin/roles",
			map[string]any{"code": "suc_role2", "name": "دور"}, "admin.role_create"},
		{"منحُ قدرةٍ لدور", "POST", "/api/v1/admin/roles/admin/capabilities",
			map[string]any{"capability": "audit.read"}, "admin.role_capability_grant"},
		{"إسنادُ دورٍ لحساب", "POST", "/api/v1/admin/users/" + victim.ID + "/roles",
			map[string]any{"role": "finance"}, "admin.role_grant"},
		{"استعادةُ كلمةِ حساب", "POST", "/api/v1/admin/users/" + victim.ID + "/password",
			map[string]any{"password": "Zz-9!aaaaaa"}, "admin.password_reset"},
		{"قيدُ محفظةٍ بمبلغ", "POST", walletPath(victim.ID),
			walletBody(500), "finance.wallet_apply"},
		{"إعدادٌ ذو أثرٍ ماليّ", "PUT", "/api/v1/admin/settings/delivery.fee",
			map[string]any{"value": "700"}, "admin.setting_update"},
	}
	for _, c := range cases {
		code, raw := stepUpChallenge(t, hh, admin.Token, c.method, c.path, c.body)
		t.Logf("%-24s ⇒ %d · %v", c.label, code, keysOf(raw))
		assertChallengeShape(t, c.label, code, raw, c.action)
	}
}

// TestSUC3_EveryActionGoesThroughOnePath **مسارٌ مركزيٌّ واحدٌ لا خمسةَ عشر.**
//
// **وإصلاحُ الظرف لا يحمي إلّا ما مرّ به** — **فيُقاس أنّ الجدولَ كلَّه
// يمرّ بالوسيط**: كلُّ صفٍّ يُعثَر عليه بنمطه، **والوسيطُ يُركَّب مرّةً
// واحدةً في الموجّه.**
func TestSUC3_EveryActionGoesThroughOnePath(t *testing.T) {
	all := authz.SensitiveActions()
	if len(all) == 0 {
		t.Fatal("**جدولُ الأفعال الحسّاسة فارغ** — والفاحصُ بلا مرجع")
	}
	for _, act := range all {
		got, ok := authz.LookupSensitive(act.Method, act.Pattern)
		if !ok {
			t.Errorf("**فعلٌ في الجدول لا يجده الوسيط**: %s %s", act.Method, act.Pattern)
			continue
		}
		if got.Action != act.Action {
			t.Errorf("**التماسٌ في الجدول**: %s %s ⇒ %q", act.Method, act.Pattern, got.Action)
		}
	}
	t.Logf("أفعالٌ حسّاسةٌ في الجدول = %d", len(all))

	// **والوسيطُ مرّةً واحدةً** — **ونسخةٌ ثانيةٌ منه تعني ظرفين.**
	src, err := os.ReadFile("../server/server.go")
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ الموجّه: %v", err)
	}
	if n := strings.Count(string(src), "requireStepUp"); n != 1 {
		t.Errorf("**الوسيطُ مذكورٌ %d مرّةً في الموجّه** — "+
			"**والمنتظَرُ مرّةٌ واحدة**: مسارانِ للتحدّي يفترقان يوماً.", n)
	}

	// **وكاتبُ التحدّي واحدٌ كذلك** — ولا يُكتب الشكلُ في موضعٍ ثانٍ.
	up, err := os.ReadFile("../server/stepup.go")
	if err != nil {
		t.Fatalf("تعذّرت قراءةُ البوّابة: %v", err)
	}
	// **والتعليقُ يُطوى قبل العدّ** — **وشرحُ العطب يذكر اسمَ الدالّة
	// و`httpx.JSON` كليهما، فيُسقط الفحصُ نفسَه بشرحه.**
	body := stripGoComments(string(up))
	if n := strings.Count(body, "respondStepUpRequired"); n != 2 {
		t.Errorf("**كاتبُ التحدّي مذكورٌ %d مرّةً** — "+
			"**والمنتظَرُ اثنتان**: تعريفٌ ونداءٌ لا غير.", n)
	}
	// ── ولا غلافَ نجاحٍ داخلَ كاتب التحدّي ──────────────────────
	//
	// **و`httpx.JSON` مشروعةٌ في هذا الملفّ**: **بابُ `/admin/step-up`
	// يردّ منحةً، وذاك نجاحٌ حقّاً.** **فالمنعُ داخلَ الدالّة وحدَها** —
	// **ومنعٌ شاملٌ يُسقط ما لا عيبَ فيه.** (وقع في أوّل تشغيلٍ لهذا
	// الفحص.)
	if fn := funcBody(body, "func (s *Server) respondStepUpRequired"); fn == "" {
		t.Error("**تعذّر العثورُ على كاتب التحدّي** — والفحصُ بلا مرجعٍ لا يقيس")
	} else if strings.Contains(fn, "httpx.JSON(") {
		t.Errorf("**كاتبُ التحدّي يكتب بغلاف النجاح** — " +
			"**`httpx.JSON` تغلّف في `data` فيُدفَن التحدّي.**")
	} else if !strings.Contains(fn, "httpx.ErrorWith(") {
		t.Errorf("**كاتبُ التحدّي لا يستعمل ظرفَ الخطأ المركزيّ**")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("ترميز: %v", err)
	}
	return b
}

// stripGoComments **يطوي تعليقَ Go** — سطريَّه وكتليَّه.
func stripGoComments(src string) string {
	var out strings.Builder
	for i := 0; i < len(src); i++ {
		if src[i] == '/' && i+1 < len(src) {
			if src[i+1] == '/' {
				for i < len(src) && src[i] != '\n' {
					i++
				}
				out.WriteByte('\n')
				continue
			}
			if src[i+1] == '*' {
				if j := strings.Index(src[i+2:], "*/"); j >= 0 {
					i += 2 + j + 1
					continue
				}
				break
			}
		}
		out.WriteByte(src[i])
	}
	return out.String()
}

// funcBody **جسمُ دالّةٍ بعينها** — من ترويستها إلى قوسها الختاميّ.
func funcBody(src, header string) string {
	i := strings.Index(src, header)
	if i < 0 {
		return ""
	}
	rest := src[i:]
	depth := 0
	for j := 0; j < len(rest); j++ {
		switch rest[j] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return rest[:j+1]
			}
		}
	}
	return ""
}
