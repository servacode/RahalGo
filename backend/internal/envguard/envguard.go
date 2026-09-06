// Package envguard حرّاسُ البيئة — **ما يمنع أمراً هدّاماً أن يمسّ الإنتاج.**
//
// # المسألةُ التي يحلّها
//
// **بيئةُ التجهيز تُشغَّل فيها أوامرُ تمحو**: إعادةُ بذرِ البيانات،
// وإسقاطُ `Redis`، واستعادةُ نسخةٍ فوق قاعدةٍ قائمة. **ومتغيّرُ بيئةٍ
// واحدٌ خاطئٌ يجعلها كلَّها تمسّ الإنتاج.**
//
// **ولا يكفي `APP_ENV`**: من صدّره خطأً فقد كلَّ شيء. **فخمسةُ حرّاسٍ
// مستقلّةٍ لا واحد** (البند ٦):
//
//	١ · هويّةُ البيئة المعلَنة        `APP_ENV`
//	٢ · اسمُ القاعدة                 لا يكون `rahalgo` المجرَّد
//	٣ · المضيف                       لا يكون مضيفَ إنتاجٍ معروفاً
//	٤ · عنوانُ الـAPI                لا يكون نطاقَ الإنتاج
//	٥ · علامةُ تجهيزٍ صريحة          `RAHALGO_STAGING=1`
//
// **وواحدٌ يسقط ⇒ الأمرُ يُرفَض** — **ولا يُجمَع الحرّاسُ بـ«أو».**
//
// # ولا بابَ خلفيّ
//
// **ولا متغيّرَ `FORCE` ولا `--yes`** (البند ٦: «لا تضف easy bypass»).
// **ومن أراد أن يمسّ الإنتاج يفعله بيده خارجَ هذه الأدوات** — **وحينها
// يعرف أنّه يفعله.**
package envguard

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

// ProductionHosts **مضيفاتُ الإنتاج ونطاقاتُه** — مقروءةٌ من `deploy/`.
//
// **والقائمةُ تُوسَّع ولا تُقصَّر** — **وإضافةُ نطاقٍ إلى الإنتاج تُضاف
// هنا في السطر نفسِه.**
var ProductionHosts = []string{
	"rahalgo.com",
	"www.rahalgo.com",
	"api.rahalgo.com",
	"maps.rahalgo.com",
}

// ProductionDBNames **أسماءُ قواعد الإنتاج** — والمجرَّدُ منها هو الإنتاج.
var ProductionDBNames = []string{"rahalgo"}

// Result حصيلةُ الفحص.
type Result struct {
	// Checks **كلُّ حارسٍ وحكمُه** — **ولا يُقال «رُفض» بلا سبب.**
	Checks []Check `json:"checks"`
	// Safe **أكلُّ الحرّاس راضون؟**
	Safe bool `json:"safe"`
}

// Check حارسٌ واحد.
type Check struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// Env ما يُفحَص — **يُقرأ من البيئة أو يُمرَّر في الاختبار.**
type Env struct {
	AppEnv      string
	DatabaseURL string
	RedisURL    string
	APIURL      string
	StagingFlag string
}

// FromOS يقرأ البيئةَ الحاليّة.
func FromOS() Env {
	return Env{
		AppEnv:      os.Getenv("APP_ENV"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		APIURL:      os.Getenv("NEXT_PUBLIC_API_URL"),
		StagingFlag: os.Getenv("RAHALGO_STAGING"),
	}
}

// Inspect يُجري الحرّاسَ الخمسةَ ويردّ حكمَ كلٍّ منها.
//
// **ولا يتوقّف عند أوّل سقوط** — **من رأى سبباً واحداً أصلحه ثمّ سقط
// بالثاني**، والقائمةُ كاملةً أرحمُ.
func Inspect(e Env) Result {
	var r Result

	// ── ١ · هويّةُ البيئة ────────────────────────────────────────
	env := strings.ToLower(strings.TrimSpace(e.AppEnv))
	r.add("APP_ENV", env == "staging" || env == "test" || env == "development",
		fmt.Sprintf("APP_ENV = %q — **والأمرُ الهدّامُ لا يعمل في `production`**", e.AppEnv))

	// ── ٢ · اسمُ القاعدة ────────────────────────────────────────
	db, dbDetail := dbName(e.DatabaseURL)
	prodDB := false
	for _, p := range ProductionDBNames {
		if db == p {
			prodDB = true
		}
	}
	r.add("DATABASE_NAME", db != "" && !prodDB,
		fmt.Sprintf("قاعدةٌ %q — %s", db, dbDetail))

	// ── ٣ · مضيفُ القاعدة والذاكرة ──────────────────────────────
	badHost := ""
	for _, raw := range []string{e.DatabaseURL, e.RedisURL} {
		if h := hostOf(raw); isProductionHost(h) {
			badHost = h
		}
	}
	r.add("HOST", badHost == "",
		fmt.Sprintf("مضيفٌ %q — **ومضيفُ إنتاجٍ في وصلةٍ يُسقط الأمر**", badHost))

	// ── ٤ · عنوانُ الـAPI ───────────────────────────────────────
	apiHost := hostOf(e.APIURL)
	r.add("API_URL", !isProductionHost(apiHost),
		fmt.Sprintf("API على %q", apiHost))

	// ── ٥ · علامةُ التجهيز الصريحة ──────────────────────────────
	//
	// **وهذه لا تُصدَّر بالغلط** — **ومن نسيها لم يقصد أمراً هدّاماً.**
	r.add("STAGING_MARKER", e.StagingFlag == "1",
		"RAHALGO_STAGING=1 مطلوبةٌ صراحةً — **ولا تُصدَّر بالغلط**")

	r.Safe = true
	for _, c := range r.Checks {
		if !c.Passed {
			r.Safe = false
		}
	}
	return r
}

func (r *Result) add(name string, ok bool, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Passed: ok, Detail: detail})
}

// MustBeSafe يُسقط العمليّةَ إن لم يرضَ الحرّاسُ كلُّهم.
//
// **ورسالةٌ تقول أيُّ حارسٍ سقط ولماذا** — **ومن رأى «رُفض» وحدَها
// بحث عن بابٍ خلفيّ.**
func MustBeSafe(e Env) error {
	r := Inspect(e)
	if r.Safe {
		return nil
	}
	var b strings.Builder
	b.WriteString("حارسُ الإنتاج رفض الأمر — والأسبابُ:\n")
	for _, c := range r.Checks {
		mark := "✓"
		if !c.Passed {
			mark = "✗"
		}
		fmt.Fprintf(&b, "  %s %-16s %s\n", mark, c.Name, c.Detail)
	}
	b.WriteString("\n**ولا بابَ خلفيّ** — صحّح البيئةَ أو لا تُشغّل الأمر.")
	return fmt.Errorf("%s", b.String())
}

// isProductionHost **أمضيفُ إنتاجٍ معروف؟**
//
// **ويُطابَق النطاقُ الفرعيُّ أيضاً** — `x.rahalgo.com` إنتاجٌ ما لم
// يُصرَّح بغيره. **و`staging.rahalgo.com` استثناءٌ صريح.**
func isProductionHost(h string) bool {
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "" {
		return false
	}
	// **ونطاقُ التجهيز يُستثنى صراحةً** — وهو الوحيدُ المستثنى.
	if strings.HasPrefix(h, "staging.") || strings.HasPrefix(h, "stg.") {
		return false
	}
	for _, p := range ProductionHosts {
		if h == p || strings.HasSuffix(h, "."+p) {
			return true
		}
	}
	return false
}

// hostOf مضيفُ وصلةٍ — **ونصٌّ لا يُحلّل يُعامَل مضيفاً خامّاً.**
func hostOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if u, err := url.Parse(raw); err == nil && u.Host != "" {
		return u.Hostname()
	}
	return raw
}

// dbName اسمُ القاعدة من وصلتها.
func dbName(raw string) (string, string) {
	if strings.TrimSpace(raw) == "" {
		return "", "**لا وصلةَ قاعدةٍ أصلاً** — ولا يُخمَّن اسم"
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "وصلةٌ لا تُحلَّل"
	}
	name := strings.TrimPrefix(u.Path, "/")
	if name == "" {
		return "", "وصلةٌ بلا اسمِ قاعدة"
	}
	for _, p := range ProductionDBNames {
		if name == p {
			return name, "**وهو اسمُ قاعدة الإنتاج**"
		}
	}
	return name, "وليس اسمَ الإنتاج"
}

// ══════════════════════════════════════════════════════════════════════
// **هويّةُ البيئة — البند ٥**
// ══════════════════════════════════════════════════════════════════════

// Identity ما تقوله البيئةُ عن نفسها — **بلا سرٍّ واحد.**
type Identity struct {
	Environment string `json:"environment"`
	// SourceCommit **التزامُ الشيفرة** — يُحقَن عند البناء.
	SourceCommit string `json:"source_commit"`
	// BuildID **هويّةُ البناء** — يُحقَن عند البناء.
	BuildID string `json:"build_id"`
	// Migration **آخرُ هجرةٍ طُبِّقت** — تُقرأ من القاعدة.
	Migration string `json:"migration_version"`
	// Staging **علامةٌ صريحةٌ للواجهة** — منها تُرسَم الرايةُ.
	Staging bool `json:"staging"`
}

// **وتُحقَن عند البناء بـ`-ldflags`** — **ولا تُقرأ من متغيّرٍ يُبدَّل
// بعد النشر**: هويّةُ بناءٍ تُغيَّر وقتَ التشغيل ليست هويّة.
var (
	buildCommit = ""
	buildID     = ""
)

// IdentityOf يبني الهويّةَ من البيئة والهجرة.
func IdentityOf(appEnv, migration string) Identity {
	env := strings.ToLower(strings.TrimSpace(appEnv))
	if env == "" {
		env = "development"
	}
	return Identity{
		Environment:  env,
		SourceCommit: buildCommit,
		BuildID:      buildID,
		Migration:    migration,
		Staging:      env == "staging",
	}
}

// BuildInfo التزامُ البناء وهويّتُه — **للاختبار وللتقرير.**
func BuildInfo() (commit, id string) { return buildCommit, buildID }
