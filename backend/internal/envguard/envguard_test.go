package envguard

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حرّاسُ الإنتاج — `P-0` البند ٦**
// ══════════════════════════════════════════════════════════════════════
//
// **وواحدٌ يسقط ⇒ الأمرُ يُرفَض** — **ولا يُجمَع الحرّاسُ بـ«أو».**

func stagingEnv() Env {
	return Env{
		AppEnv:      "staging",
		DatabaseURL: "postgres://rahalgo:x@localhost:5534/rahalgo_staging?sslmode=disable",
		RedisURL:    "redis://localhost:6580/0",
		APIURL:      "http://localhost:8080",
		StagingFlag: "1",
	}
}

func TestGuard_CleanStagingPasses(t *testing.T) {
	r := Inspect(stagingEnv())
	if !r.Safe {
		for _, c := range r.Checks {
			if !c.Passed {
				t.Errorf("حارسٌ سقط على بيئةٍ سليمة: %s — %s", c.Name, c.Detail)
			}
		}
	}
	if len(r.Checks) != 5 {
		t.Fatalf("حرّاسٌ %d ولا خمسة", len(r.Checks))
	}
}

// TestGuard_EachGuardBlocksAlone **كلُّ حارسٍ يمنع وحدَه** — البند ٦.
//
// **وهذا هو جوهرُ «حرّاسٌ متعدّدة»**: **متغيّرٌ واحدٌ خاطئٌ يكفي.**
func TestGuard_EachGuardBlocksAlone(t *testing.T) {
	cases := []struct {
		name   string
		break_ func(*Env)
		guard  string
	}{
		{"بيئةُ إنتاج", func(e *Env) { e.AppEnv = "production" }, "APP_ENV"},
		{"قاعدةُ الإنتاج", func(e *Env) {
			e.DatabaseURL = "postgres://rahalgo:x@localhost:5432/rahalgo"
		}, "DATABASE_NAME"},
		{"مضيفُ الإنتاج في القاعدة", func(e *Env) {
			e.DatabaseURL = "postgres://rahalgo:x@api.rahalgo.com:5432/rahalgo_staging"
		}, "HOST"},
		{"مضيفُ الإنتاج في الذاكرة", func(e *Env) {
			e.RedisURL = "redis://rahalgo.com:6379/0"
		}, "HOST"},
		{"عنوانُ إنتاجٍ للـAPI", func(e *Env) {
			e.APIURL = "https://api.rahalgo.com"
		}, "API_URL"},
		{"بلا علامةٍ صريحة", func(e *Env) { e.StagingFlag = "" }, "STAGING_MARKER"},
	}
	for _, c := range cases {
		e := stagingEnv()
		c.break_(&e)
		r := Inspect(e)
		if r.Safe {
			t.Errorf("%s: مرَّ — **وحارسٌ واحدٌ يكفي للمنع**", c.name)
			continue
		}
		hit := false
		for _, g := range r.Checks {
			if g.Name == c.guard && !g.Passed {
				hit = true
			}
		}
		if !hit {
			t.Errorf("%s: مُنع لكن لا بالحارس %s", c.name, c.guard)
		}
	}
}

// TestGuard_ProductionSubdomainsBlocked **والنطاقُ الفرعيُّ إنتاجٌ أيضاً.**
func TestGuard_ProductionSubdomainsBlocked(t *testing.T) {
	for _, h := range []string{
		"rahalgo.com", "api.rahalgo.com", "maps.rahalgo.com",
		"db.rahalgo.com", "anything.api.rahalgo.com",
	} {
		if !isProductionHost(h) {
			t.Errorf("%s لم يُعَدَّ مضيفَ إنتاج", h)
		}
	}
	// **ونطاقُ التجهيز مستثنىً صراحةً** — وهو الوحيد.
	for _, h := range []string{"staging.rahalgo.com", "stg.rahalgo.com", "localhost", ""} {
		if isProductionHost(h) {
			t.Errorf("%s عُدَّ مضيفَ إنتاجٍ خطأً", h)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **والتعليقُ ليس قيمة**
// ══════════════════════════════════════════════════════════════════════
//
// **أوّلُ صياغةٍ لهذه الحرّاس مسحت الملفَّ كلَّه نصّاً** — **فسقطت على
// تعليقاتٍ تشرح ما لا يُفعَل**: تعليقٌ يقول «ولا متغيّرَ `FORCE`»
// يُقرأ باباً خلفيّاً، وتعليقٌ يشرح أنّ نمطَ الخريطة يُشارَك للقراءة
// يُقرأ مضيفَ إنتاجٍ في الإعداد.
//
// **والحارسُ لم يُضعَّف بل دُقِّق**: **الخطرُ في القيمة لا في الشرح.**
// **ومضيفُ إنتاجٍ في `DATABASE_URL` يمسّ الإنتاج، وفي تعليقٍ لا يمسّ
// شيئاً** — **ومن أسكت الحارسَ بحذف الشرح خسر الشرحَ وأبقى الخطر.**

// codeOnly ينزع التعليقاتِ ويُبقي ما يُنفَّذ.
func codeOnly(src string, markers ...string) string {
	var b strings.Builder
	for _, line := range strings.Split(src, "\n") {
		cut := line
		for _, m := range markers {
			if i := strings.Index(cut, m); i >= 0 {
				cut = cut[:i]
			}
		}
		b.WriteString(cut)
		b.WriteString("\n")
	}
	return b.String()
}

// TestGuard_NoBypassExists **ولا بابَ خلفيّ** — البند ٦.
//
// **ويُقرأ المصدرُ نفسُه**: **متغيّرُ `FORCE` يُضاف يوماً بحسن نيّة**،
// **ثمّ يُصدَّر في خطّ نشرٍ فيبطل الحرّاسُ كلُّهم.**
func TestGuard_NoBypassExists(t *testing.T) {
	root := mustRepo(t)
	for _, f := range []string{
		"backend/internal/envguard/envguard.go",
		"backend/cmd/stagingctl/main.go",
	} {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		src := codeOnly(string(b), "//")
		for _, bad := range []string{"FORCE", "--force", "SKIP_GUARD", "ALLOW_PRODUCTION", "--yes"} {
			if strings.Contains(src, bad) {
				t.Errorf("%s فيه بابٌ خلفيٌّ محتمل: %q", f, bad)
			}
		}
	}
}

// TestGuard_MustBeSafeNamesEveryFailure **ورسالةٌ تقول أيُّ حارسٍ سقط.**
func TestGuard_MustBeSafeNamesEveryFailure(t *testing.T) {
	e := stagingEnv()
	e.AppEnv = "production"
	e.StagingFlag = ""
	err := MustBeSafe(e)
	if err == nil {
		t.Fatal("بيئةُ إنتاجٍ مرَّت")
	}
	msg := err.Error()
	for _, want := range []string{"APP_ENV", "STAGING_MARKER", "ولا بابَ خلفيّ"} {
		if !strings.Contains(msg, want) {
			t.Errorf("الرسالةُ لا تذكر %q:\n%s", want, msg)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ البيئة — البند ٥**
// ══════════════════════════════════════════════════════════════════════

func TestIdentity_CarriesNoSecret(t *testing.T) {
	id := IdentityOf("staging", "0128_branches_areas.sql")
	if !id.Staging || id.Environment != "staging" {
		t.Fatalf("هويّةٌ خاطئة: %+v", id)
	}
	// **ولا حقلَ فيها يحمل سرّاً** — والبنيةُ نفسُها هي العقد.
	blob := id.Environment + id.SourceCommit + id.BuildID + id.Migration
	for _, secret := range []string{"password", "secret", "postgres://", "redis://", "Bearer"} {
		if strings.Contains(strings.ToLower(blob), secret) {
			t.Errorf("الهويّةُ تحمل %q", secret)
		}
	}
}

func TestIdentity_DefaultsToDevelopmentNotProduction(t *testing.T) {
	// **وفارغٌ يعني تطويراً لا إنتاجاً** — **ومن أخطأ الضبطَ لا يُمنَح
	// صفةَ الإنتاج بالغلط**، وهي تفتح الأوامرَ لا تغلقها.
	if got := IdentityOf("", "").Environment; got != "development" {
		t.Errorf("بيئةٌ فارغةٌ صارت %q", got)
	}
	if IdentityOf("", "").Staging {
		t.Error("بيئةٌ فارغةٌ ادّعت أنّها تجهيز")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **سلامةُ إعداد التجهيز — البنود ٢ و٤٣ و٤٦ و٤٧**
// ══════════════════════════════════════════════════════════════════════

func mustRepo(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// internal/envguard ⇒ backend ⇒ repo
	return filepath.Dir(filepath.Dir(filepath.Dir(wd)))
}

// TestStagingConfigNeverNamesProduction **البندان ٢ و٤٣.**
func TestStagingConfigNeverNamesProduction(t *testing.T) {
	root := mustRepo(t)
	files := []string{
		"deploy/staging/compose.staging.yml",
		"deploy/staging/Caddyfile.staging",
		"deploy/staging/.env.staging.example",
	}
	for _, f := range files {
		b, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		// **والقيمُ وحدَها تُفحَص** — والتعليقُ يشرح ولا يمسّ شيئاً.
		src := codeOnly(string(b), "#")
		for _, host := range ProductionHosts {
			// **و`staging.rahalgo.com` مسموح** — يُنزَع قبل الفحص.
			clean := strings.ReplaceAll(src, "staging."+host, "")
			clean = strings.ReplaceAll(clean, "stg."+host, "")
			if strings.Contains(clean, host) {
				t.Errorf("%s يضع مضيفَ الإنتاج %q في قيمة", f, host)
			}
		}
		// **ولا يُشار إلى قاعدة الإنتاج ولا حجومها** (البند ٤٦).
		for _, bad := range []string{"/rahalgo?", "/rahalgo\"", "POSTGRES_DB: rahalgo\n"} {
			if strings.Contains(src, bad) {
				t.Errorf("%s يشير إلى قاعدة الإنتاج: %q", f, bad)
			}
		}
	}
}

// TestStagingComposeIsFullyIsolated **البنود ٧ و٨ و٩ · `TQ-4`.**
func TestStagingComposeIsFullyIsolated(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(mustRepo(t), "deploy/staging/compose.staging.yml"))
	if err != nil {
		t.Fatal(err)
	}
	src := codeOnly(string(b), "#")
	// **واسمُ المكدّس يحكم أسماءَ الأحجام والشبكات كلَّها** — **وهو
	// الفرقُ بين مشاركةِ قرصٍ وعزلٍ تامّ.**
	if !strings.Contains(src, "name: rahalgo-staging") {
		t.Error("اسمُ المكدّس ليس `rahalgo-staging` — **والأحجامُ ستُشارَك**")
	}
	if !strings.Contains(src, "POSTGRES_DB: rahalgo_staging") {
		t.Error("اسمُ القاعدة ليس `rahalgo_staging`")
	}
	if !strings.Contains(src, "APP_ENV: staging") {
		t.Error("البيئةُ ليست `staging`")
	}
	// **ولا منفذَ إنتاجٍ يُنتزَع** — ٨٠ و٤٤٣ للإنتاج.
	for _, bad := range []string{"\"80:80\"", "\"443:443\""} {
		if strings.Contains(src, bad) {
			t.Errorf("التجهيزُ ينتزع منفذَ الإنتاج %s", bad)
		}
	}
}

// TestStagingProvidersAreSilentByDefault **البنود ١٩–٢٢ و٤٤.**
func TestStagingProvidersAreSilentByDefault(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(mustRepo(t), "deploy/staging/compose.staging.yml"))
	if err != nil {
		t.Fatal(err)
	}
	src := codeOnly(string(b), "#")
	// **ولا رسالةَ تجربةٍ تصل إلى إنسانٍ حقيقيّ.**
	if !strings.Contains(src, "OTP_PROVIDER: dev") {
		t.Error("مزوّدُ الرمز ليس `dev` — **ورسالةُ تجربةٍ قد تصل إلى هاتفِ إنسان**")
	}
	if !strings.Contains(src, `SMS_URL: ""`) {
		t.Error("بوّابةُ الرسائل غيرُ معطَّلة")
	}
	// ══════════════════════════════════════════════════════════════
	// **ودفعُ التجهيز صار جائزاً — بمشروعِ تجهيزٍ لا بمشروع الإنتاج**
	// ══════════════════════════════════════════════════════════════
	//
	// **وكان الشرطُ «لا اعتمادَ دفعٍ ألبتّة»** — **وكان صحيحاً يومَه**:
	// **البيئتان على مشروعِ فايربيس واحدٍ** (`rahalgo-prod`)، **ورمزُ
	// جهازٍ يصلح لمُرسِلٍ واحدٍ لا لبيئة** — **فإشعارُ تجربةٍ قد يقع
	// في هاتفِ زبونٍ حقيقيّ.**
	//
	// **ثمّ أُنشئ للتجهيز مشروعُه** (٢٠٢٦-٠٩-١٦) — **فسقط سببُ المنع**:
	// **مُرسِلان مختلفان لا يبلغ أحدُهما أجهزةَ الآخر** (ويردّ فايربيس
	// `SENDER_ID_MISMATCH`). **وقبولُ `P-8` يلزمه دفعٌ يعمل**، **ولا
	// يُقاس على ما لا يُرسل.**
	//
	// **والحمايةُ تبقى وتشتدّ**: **ليس «لا اعتماد» بل «لا اعتمادَ
	// إنتاجٍ»** — **ومن ركّب سرَّ الإنتاج في التجهيز أعاد الخطرَ عينَه
	// وهو يظنّ أنّه عزل.**
	if strings.Contains(src, "FCM_CREDENTIALS_FILE") {
		// **ومصدرُ السرّ من مجلَّد التجهيز وحدَه.**
		if !strings.Contains(src, "/srv/rahalgo-staging/secrets/fcm.json") {
			t.Error("اعتمادُ الدفع مضبوطٌ ومصدرُه ليس مجلَّدَ التجهيز — " +
				"**ومن ركّب سرَّ الإنتاج هنا أعاد الخطرَ وهو يظنّه عزلاً**")
		}
		// **ولا يُقرَأ سرُّ الإنتاج من مجلَّده ألبتّة.**
		for _, bad := range []string{"/srv/rahalgo/secrets", "rahalgo-prod"} {
			if strings.Contains(src, bad) {
				t.Errorf("تركيبُ التجهيز يشير إلى اعتمادِ الإنتاج: %s", bad)
			}
		}
		// **ويُركَّب للقراءة وحدَها** — **ومحرّكُ تجهيزٍ يكتب في سرٍّ
		// لا يكتب فيه شيئاً، والكتابةُ بابٌ لا حاجةَ إليه.**
		if !strings.Contains(src, "/run/secrets/fcm.json:ro") {
			t.Error("سرُّ الدفع مُركَّبٌ بلا `:ro`")
		}
	}
}

// TestNoSecretsCommitted **البند ٤٧.**
func TestNoSecretsCommitted(t *testing.T) {
	root := mustRepo(t)
	// **والملفُّ الحقيقيُّ لا يدخل المستودع.**
	ignore, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ignore), "deploy/staging/.env.staging") {
		t.Error(".env.staging غيرُ مستثنىً في .gitignore")
	}
	// ══════════════════════════════════════════════════════════════
	// **والقالبُ متتبَّعٌ في git — لا على القرص وحدَه**
	// ══════════════════════════════════════════════════════════════
	//
	// **وهذا ما فات الحارسَ أوّلَ مرّة**: كان يقرأ الملفَّ من الشجرة
	// **فوجده ورضي** — **و`.gitignore` كان يبتلعه بقاعدة `.env.*`.**
	//
	// **فلم يصل الخادمَ، وأخفق `cp` صامتاً، وانهار إقلاعُ التجهيز
	// كلُّه** (قِيس ٢٠٢٦-٠٩-٠٦ على `CX33`). **وقالبٌ لا يُستنسَخ لا
	// ينفع أحداً.**
	tracked, err := exec.Command("git", "-C", root, "ls-files", "--error-unmatch",
		"deploy/staging/.env.staging.example").CombinedOutput()
	if err != nil {
		t.Errorf("القالبُ غيرُ متتبَّعٍ في git — **ومن استنسخ المستودعَ لم يجده**: %s",
			strings.TrimSpace(string(tracked)))
	}

	// **والقالبُ فارغُ القيم** — **وقالبٌ فيه سرٌّ حقيقيٌّ أسوأُ من لا قالب.**
	b, err := os.ReadFile(filepath.Join(root, "deploy/staging/.env.staging.example"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if !strings.Contains(k, "PASSWORD") && !strings.Contains(k, "SECRET") {
			continue
		}
		if strings.TrimSpace(v) != "" {
			t.Errorf("القالبُ يحمل قيمةً لـ%s — **ولا سرَّ في المستودع**", k)
		}
	}
}
