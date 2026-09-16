package qa

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **ما قبل الافتتاح — أنماطُ حالِ التطبيق** (`PL`، ٢٠٢٦-٠٩-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **والمالكُ لا يقلّب أربعَ راياتٍ بيده** — **بل يختار حالاً**: ما قبلَ
// الافتتاح · واستعراضٌ قبله · ومفتوحٌ للعمل.
//
// **ومن قلّبها بيده نسي واحدةً** — **ففُتح التسجيلُ والسوقُ مقفلٌ**،
// **أو فُتح الطلبُ ولا متجرَ فيه.**
//
// # وما يُقاس هنا
//
//	١ · كلُّ نمطٍ يكتب الأربعَ بأعيانها — لا أقلَّ ولا أكثر
//	٢ · ولا يمسّ أبوابَ العمل الثلاثة
//	٣ · والمنعُ يبقى في المحرّك — **والنمطُ راحةٌ لا سلطان**
//	٤ · والبابُ محروسٌ بقدرةٍ ومُدقَّق
//	٥ · ولا بابَ خلفيٌّ لرقمِ مراجِعٍ بعينه

// presetPath بابُ تطبيق النمط.
const presetPath = "/api/v1/admin/launch/preset"

// launchKeys المفاتيحُ الأربعةُ التي يملكها النمط — **ولا خامسَ لها.**
var launchKeys = []string{
	"launch.customer_signup",
	"launch.customer_browse",
	"launch.customer_orders",
	"launch.customer_custom_orders",
}

// workKeys أبوابُ العمل — **خارجَ النمط عمداً.**
var workKeys = []string{
	"launch.driver_work",
	"launch.merchant_orders",
	"launch.rep_acquisition",
}

// restoreLaunchAfter **يُسجّل استعادةَ الرايات السبع بعد الفحص.**
//
// ====================================================================
// **وقاعدةُ الفحص مشتركةٌ — ورايةٌ تُكتب ولا تُعاد تُسقط ما بعدها**
// ====================================================================
//
// **ووقع ٢٠٢٦-٠٩-١٦**: **أربعةٌ وثمانون فحصاً سقطت بـ`launch_closed`**
// — **تفرّدُ المفاتيح وسباقاتُ السائقين والدفترُ كلُّه** — **لأنّ نمطَ
// «ما قبل الافتتاح» بقي مكتوباً في `app_settings` بعد فحوص `PL`.**
//
// **ولم يكن في الفحوص عيبٌ** — **بل في أثرٍ تركتُه.** **وفحصٌ يكتب في
// حالٍ مشتركةٍ ولا يُعيدها يُفسد جيرانَه لا نفسَه**، **فيُقرأ العطبُ
// في مكانٍ لا علاقةَ له بالسبب.**
//
// **و`Harness.Setting` تلتقط القديمَ وتُعيده في `Cleanup`** — **فتُنادى
// على كلّ مفتاحٍ قبل أن يكتبه النمط.**
func restoreLaunchAfter(t *testing.T, hh *Harness) {
	t.Helper()
	for _, k := range append(append([]string{}, launchKeys...), workKeys...) {
		var cur string
		if err := hh.Pool.QueryRow(ctxBG(),
			`SELECT value::text FROM app_settings WHERE key = $1`, k).Scan(&cur); err != nil {
			// **وغائبٌ يبقى غائباً** — **و`Setting` تحذفه في الاستعادة.**
			cur = "true"
		}
		hh.Setting(k, cur)
	}
}

// ownerToken **مالكٌ يملك قدرةَ الإعدادات.**
func ownerToken(t *testing.T, hh *Harness) string {
	t.Helper()
	_, tok := roleUser(t, hh, "owner_super_admin")
	return tok
}

// applyPreset **يطبّق نمطاً ويُسقط الاختبارَ إن رُدّ.**
func applyPreset(t *testing.T, hh *Harness, tok, name string) Res {
	t.Helper()
	r := hh.POST(presetPath, tok, map[string]any{"preset": name})
	if r.Code != http.StatusOK {
		t.Fatalf("**رُدَّ النمطُ %q**: %d %s", name, r.Code, r.Err())
	}
	return r
}

// flagsNow **الرايةُ كما هي مخزَّنةٌ الآن** — تُقرأ من باب الحال.
func flagsNow(t *testing.T, hh *Harness, tok string) map[string]bool {
	t.Helper()
	r := hh.GET("/api/v1/admin/launch", tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**تعذّرت قراءةُ حال التطبيق**: %d %s", r.Code, r.Err())
	}
	out := map[string]bool{}
	cur, _ := r.JSON()["current"].(map[string]any)
	for k, v := range cur {
		b, _ := v.(bool)
		out[k] = b
	}
	un, _ := r.JSON()["untouched"].(map[string]any)
	for k, v := range un {
		b, _ := v.(bool)
		out[k] = b
	}
	return out
}

// wantFlags **يطابق الأربعَ بأعيانها.**
func wantFlags(t *testing.T, got map[string]bool, signup, browse, orders, custom bool) {
	t.Helper()
	want := map[string]bool{
		"launch.customer_signup":        signup,
		"launch.customer_browse":        browse,
		"launch.customer_orders":        orders,
		"launch.customer_custom_orders": custom,
	}
	for k, w := range want {
		if got[k] != w {
			t.Errorf("**%s = %v والمنتظَرُ %v**", k, got[k], w)
		}
	}
}

// ── PL-01 ─────────────────────────────────────────────────────────────

// TestPL01_PreLaunchClosesAllFourCustomerDoors **ما قبل الافتتاح.**
func TestPL01_PreLaunchClosesAllFourCustomerDoors(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	tok := ownerToken(t, hh)
	for _, k := range launchKeys {
		hh.Setting(k, "true")
	}
	applyPreset(t, hh, tok, "pre_launch")
	wantFlags(t, flagsNow(t, hh, tok), false, false, false, false)
}

// ── PL-02 ─────────────────────────────────────────────────────────────

// TestPL02_BrowseOnlyOpensSignupAndBrowseOnly **استعراضٌ قبل الافتتاح.**
func TestPL02_BrowseOnlyOpensSignupAndBrowseOnly(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	tok := ownerToken(t, hh)
	for _, k := range launchKeys {
		hh.Setting(k, "false")
	}
	applyPreset(t, hh, tok, "browse_only")
	wantFlags(t, flagsNow(t, hh, tok), true, true, false, false)
}

// ── PL-03 ─────────────────────────────────────────────────────────────

// TestPL03_OpenEnablesAllFour **مفتوحٌ للعمل.**
func TestPL03_OpenEnablesAllFour(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	tok := ownerToken(t, hh)
	for _, k := range launchKeys {
		hh.Setting(k, "false")
	}
	applyPreset(t, hh, tok, "open")
	wantFlags(t, flagsNow(t, hh, tok), true, true, true, true)
}

// ── PL-04 · PL-05 · PL-06 ─────────────────────────────────────────────

// TestPL04_06_PresetNeverTouchesWorkDoors **وأبوابُ العمل لا تُمَسّ.**
//
// **وفتحُ السوق للزبائن لا يعني تشغيلَ أسطول** — **ونمطٌ يفتح عملَ
// السائق ضمناً يشغّل الناسَ بلا أن يقصد أحد.**
//
// **وتُقاس الحالتان**: **مفتوحةً فتبقى، ومغلقةً فتبقى** — **واختبارٌ
// يقيس حالاً واحدةً يمرّ على نمطٍ يكتب `true` دائماً.**
func TestPL04_06_PresetNeverTouchesWorkDoors(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	tok := ownerToken(t, hh)

	for _, state := range []string{"true", "false"} {
		for _, k := range workKeys {
			hh.Setting(k, state)
		}
		for _, preset := range []string{"pre_launch", "browse_only", "open"} {
			applyPreset(t, hh, tok, preset)
			got := flagsNow(t, hh, tok)
			for _, k := range workKeys {
				if got[k] != (state == "true") {
					t.Errorf("**النمطُ %q بدّل %s إلى %v** — **وهو خارجَه عمداً.**",
						preset, k, got[k])
				}
			}
		}
	}
}

// ── PL-07 ─────────────────────────────────────────────────────────────

// TestPL07_PublicPlatformExposesLaunchState **الحالُ تُقرأ لا تُستنتَج.**
//
// **وكان التطبيقُ لا يعرف أنّ المنصّةَ لم تُفتح حتّى يطرق باباً فيُردّ**
// — **فيرسم شاشةَ سوقٍ ثمّ يبدّلها رسالةَ خطأ.**
func TestPL07_PublicPlatformExposesLaunchState(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	tok := ownerToken(t, hh)
	applyPreset(t, hh, tok, "pre_launch")
	hh.Setting("launch.notice", `"قريبًا يتم افتتاح رحال غو"`)

	r := hh.GET("/api/v1/public/platform", "")
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ هويّة المنصّة مُنع** (%d) — **وبه تُعرَض الحال.**", r.Code)
	}
	l, ok := r.JSON()["launch"].(map[string]any)
	if !ok {
		t.Fatal("**لا حقلَ `launch` في هويّة المنصّة** — **فلا يعرف التطبيقُ الحالَ إلّا بطرقِ باب.**")
	}
	for _, k := range []string{"customer_signup", "customer_browse", "customer_orders", "customer_custom_orders"} {
		v, has := l[k]
		if !has {
			t.Errorf("**غاب %s عن حقل launch**", k)
			continue
		}
		if b, _ := v.(bool); b {
			t.Errorf("**%s مفتوحٌ في ما قبل الافتتاح**", k)
		}
	}
	if s, _ := l["notice"].(string); !strings.Contains(s, "رحال غو") {
		t.Errorf("**نصُّ المالك لا يصل الشاشةَ**: %q", s)
	}
}

// ── PL-11 · PL-12 · PL-13 ─────────────────────────────────────────────

// TestPL11_13_OrdersBlockedAndNotBypassable **والمنعُ في المحرّك.**
//
// **وزرٌّ مخفيٌّ في أندرويد ليس منعاً** — **ونداءٌ مباشرٌ يتجاوزه.**
// **فيُنادى البابان مباشرةً بعد تطبيق النمط.**
func TestPL11_13_OrdersBlockedAndNotBypassable(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	applyPreset(t, hh, ownerToken(t, hh), "pre_launch")
	_, tok := roleUser(t, hh, "customer")

	normal := hh.POST("/api/v1/orders", tok, map[string]any{"items": []any{}})
	if !isLaunchClosed(normal) {
		t.Errorf("**الطلبُ العاديُّ لم يُردّ في ما قبل الافتتاح**: %d %s",
			normal.Code, normal.Err())
	}
	custom := hh.POST("/api/v1/orders/custom", tok, map[string]any{
		"request": "شيءٌ ما", "address_text": "الرقة",
	})
	if !isLaunchClosed(custom) {
		t.Errorf("**الطلبُ الخاصُّ لم يُردّ في ما قبل الافتتاح**: %d %s",
			custom.Code, custom.Err())
	}
	// **والتسجيلُ كذلك** — **بابٌ ثالثٌ يُنادى مباشرةً.**
	signup := hh.POST("/api/v1/auth/signup", "", map[string]any{
		"phone": "+963900123456", "full_name": "زائر", "password": "Passw0rd!x",
	})
	if signup.Code < 400 {
		t.Errorf("**التسجيلُ نجح في ما قبل الافتتاح** (%d) — **والزرُّ المخفيُّ ليس منعاً.**",
			signup.Code)
	}
}

// ── PL-14 ─────────────────────────────────────────────────────────────

// TestPL14_PresetRequiresSettingsCapability **ولا سلطةَ إعداداتٍ جديدة.**
func TestPL14_PresetRequiresSettingsCapability(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	capRole(t, hh, "pl_obs", authz.ObservabilityRead)
	for _, role := range []string{"operations", "finance", "pl_obs", "customer_support"} {
		_, tok := roleUser(t, hh, role)
		r := hh.POST(presetPath, tok, map[string]any{"preset": "open"})
		if r.Code < 400 {
			t.Errorf("**`%s` طبّق نمطَ افتتاح** (%d) — **ولا سلطةَ إعداداتٍ له.**",
				role, r.Code)
		}
	}
	// **والمالكُ يملكها** — **وبلا مرورٍ واحدٍ يبقى المنعُ منعاً عامّاً.**
	r := hh.POST(presetPath, ownerToken(t, hh), map[string]any{"preset": "pre_launch"})
	if r.Code != http.StatusOK {
		t.Errorf("**المالكُ مُنع من نمطِ الافتتاح**: %d %s", r.Code, r.Err())
	}
}

// ── PL-15 ─────────────────────────────────────────────────────────────

// TestPL15_PresetIsAudited **ونمطٌ طُبِّق بلا سجلٍّ لا يُعرَف من طبّقه.**
func TestPL15_PresetIsAudited(t *testing.T) {
	hh := New(t)
	restoreLaunchAfter(t, hh)
	applyPreset(t, hh, ownerToken(t, hh), "browse_only")

	var n int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM audit_log WHERE action = 'admin.launch_preset'`).Scan(&n); err != nil {
		t.Fatalf("قراءةُ السجلّ: %v", err)
	}
	if n == 0 {
		t.Error("**لا سجلَّ لتطبيق النمط** — **ومن فتح المنصّةَ للناس لا يُعرَف.**")
	}
}

// ── PL-18 ─────────────────────────────────────────────────────────────

// TestPL18_NoReviewerPhoneBypass **ولا بابَ خلفيٌّ لرقمٍ بعينه.**
//
// **ومراجِعُ غوغل حسابٌ محفوظٌ لا استثناء** — **ووضعُ الإطلاق يسري
// عليه كما يسري على غيره.** **ورقمٌ مكتوبٌ في الشيفرة بابٌ لا يُغلق.**
func TestPL18_NoReviewerPhoneBypass(t *testing.T) {
	root := repoRoot(t)
	const reviewer = "963900000099"
	var hits []string
	for _, dir := range []string{"backend", "mobile", "web"} {
		_ = filepath.Walk(filepath.Join(root, dir), func(p string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			switch filepath.Ext(p) {
			case ".go", ".kt", ".ts", ".tsx", ".sql":
			default:
				return nil
			}
			// **وشيفرةُ المنتج وحدَها تُفحَص** — **والبابُ الخلفيُّ
			// يعيش فيما يُشحَن، لا فيما يبقى على المكتب.**
			//
			// **ووقع أوّلَ تشغيلٍ لهذا الحارس**: **فحصُ تهيئةِ مالكٍ
			// قديمٌ يستعمل الرقمَ نفسَه مصادفةً** — **فاحمرَّ على
			// شيفرةٍ سليمة.** **فضُيِّق المدى ولم يُخفَّف الشرط.**
			norm := strings.ReplaceAll(p, string(filepath.Separator), "/")
			if strings.HasSuffix(norm, "_test.go") ||
				strings.Contains(norm, "/src/test/") ||
				strings.Contains(norm, "/src/androidTest/") ||
				strings.Contains(norm, "/__tests__/") {
				return nil
			}
			b, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil
			}
			if strings.Contains(string(b), reviewer) {
				hits = append(hits, strings.TrimPrefix(p, root))
			}
			return nil
		})
	}
	if len(hits) > 0 {
		t.Errorf("**رقمُ المراجِع مكتوبٌ في الشيفرة** — **وهو بابٌ لا يُغلق**:\n  %s",
			strings.Join(hits, "\n  "))
	}
}
