package qa

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **الرصدُ العميقُ المحميّ — دورةُ ٧٠أ**
// ══════════════════════════════════════════════════════════════════════
//
// **وبابُ تشخيصٍ بلا حارسٍ خريطةُ حملٍ تُقرأ من خارج** — ومن عرف متى
// يمتلئ المسبحُ عرف متى يدفع. **فالحراسةُ أوّلُ ما يُثبَت، لا آخرُه.**

// **ولا توازيَ في هذه الحزمة** — `XG-41C`: **مِسنَدُها يملك صفّاً
// عامّاً، وفحصان متداخلان يمحو أحدُهما عملَ الآخر.** **وعدّاداتُ
// `obs` مشتركةٌ في العمليّة كذلك**، فالقراءةُ المتوازيةُ تقرأ عملَ
// غيرها.

const opsHealthPath = "/api/v1/admin/ops/health"

// TestOps70A_A1_ProtectedByCapability **ثلاثُ حالاتٍ لبابٍ واحد.**
//
// **ولا يكفي «ردَّ ٢٠٠ لمن يملك»**: **بابٌ يردّ ٢٠٠ للجميع يردُّه
// لمالك القدرة أيضاً.** **فالمنعُ هو ما يُثبَت.**
func TestOps70A_A1_ProtectedByCapability(t *testing.T) {
	hh := New(t)

	// **بلا جلسةٍ أصلاً.**
	if r := hh.GET(opsHealthPath, ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("**بلا جلسةٍ يجب ٤٠١** — وردَّ %d", r.Code)
	}

	// **بجلسةٍ إداريّةٍ تملك قدرةً أخرى ولا تملك هذه.**
	capRole(t, hh, "qa-ops-blind", authz.OrdersRead)
	_, blind := capUser(t, hh, "qa-ops-blind")
	if r := hh.GET(opsHealthPath, blind); r.Code != http.StatusForbidden {
		t.Fatalf("**قدرةٌ أخرى لا تفتح البابَ** — وردَّ %d", r.Code)
	}

	// **وبالقدرة نفسِها.**
	capRole(t, hh, "qa-ops-seeing", authz.ObservabilityRead)
	_, seeing := capUser(t, hh, "qa-ops-seeing")
	r := hh.GET(opsHealthPath, seeing)
	if r.Code != http.StatusOK {
		t.Fatalf("**مالكُ القدرة يُقرئ** — وردَّ %d: %s", r.Code, r.Body)
	}
	t.Logf("A1: بلا جلسة=٤٠١ · بقدرةٍ أخرى=٤٠٣ · بالقدرة=٢٠٠")
}

// TestOps70A_A2_AnalyticsIsNotObservability **قدرةُ التحليلات لا تفتحه.**
//
// **وهي أقربُ ما يُخلَط به**: **`analytics.read` أرقامُ عملٍ، وهذا
// صحّةُ بنية.** **ومن جمعهما أعطى محلّلَ المبيعات بابَ التشخيص.**
func TestOps70A_A2_AnalyticsIsNotObservability(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-analyst", authz.AnalyticsRead)
	_, tok := capUser(t, hh, "qa-ops-analyst")
	if r := hh.GET(opsHealthPath, tok); r.Code != http.StatusForbidden {
		t.Fatalf("**`analytics.read` ليست `observability.read`** — وردَّ %d", r.Code)
	}
}

// TestOps70A_A3_NoSecretsInBody **الجسدُ تجميعٌ لا تجسّس.**
//
// **ولا يُفحَص بحقلٍ حقلاً**: **حقلٌ يُضاف غداً لا يمرّ على مراجعة.**
// **فيُفحَص النصُّ كلُّه بأنماطٍ تدلّ.**
func TestOps70A_A3_NoSecretsInBody(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-priv", authz.ObservabilityRead)
	u, tok := capUser(t, hh, "qa-ops-priv")

	r := hh.GET(opsHealthPath, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**البابُ لا يُجيب** — %d: %s", r.Code, r.Body)
	}
	body := string(r.Body)

	// **ورمزُ الجلسة نفسُه أوضحُ ما يُبحث عنه** — ومعه معرّفُ صاحبها.
	banned := map[string]string{
		"رمزُ الوصول":     tok,
		"معرّفُ المستخدم": u.ID,
		"هاتفُ المستخدم":  u.Phone,
	}
	for what, needle := range banned {
		if needle != "" && strings.Contains(body, needle) {
			t.Errorf("**%s ظهر في جسد التشخيص**", what)
		}
	}
	// **وأنماطٌ تدلّ على تسريبٍ بنيويّ** — نصُّ `SQL`، سرٌّ، مسارُ قرص.
	for _, pat := range []string{
		"SELECT ", "INSERT ", "UPDATE ", "DELETE ",
		"password", "secret", "Bearer", "eyJ",
		"postgres://", "redis://", "user=", "dbname=",
		"session_id", "refresh", "@",
	} {
		if strings.Contains(body, pat) {
			t.Errorf("**نمطٌ ممنوعٌ في جسد التشخيص**: %q", pat)
		}
	}
	t.Logf("A3: صفرُ تسريبٍ في %d بايتاً", len(body))
}

// TestOps70A_A4_ReportsRealPoolAndDB **الأرقامُ من المصدر لا مخترَعة.**
//
// **وحقلٌ يردّ صفراً أبداً يمرّ في كلّ اختبارٍ ولا يقيس شيئاً** —
// **فيُشترط أن يكون المسبحُ مأهولاً والقاعدةُ مُجيبة.**
func TestOps70A_A4_ReportsRealPoolAndDB(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-real", authz.ObservabilityRead)
	_, tok := capUser(t, hh, "qa-ops-real")

	// **والردُّ ملفوفٌ بـ`data`** — عقدُ `httpx.JSON` في كلّ المنصّة.
	var env struct {
		Data struct {
			Status string `json:"status"`
			Pool   struct {
				Max      int32 `json:"max"`
				Total    int32 `json:"total"`
				Acquired int32 `json:"acquired"`
			} `json:"pg_pool"`
			PG struct {
				Reachable bool `json:"reachable"`
				Backends  int  `json:"backends"`
			} `json:"pg"`
			Redis struct {
				Reachable bool `json:"reachable"`
			} `json:"redis"`
			Runtime struct {
				Goroutines int `json:"goroutines"`
			} `json:"runtime"`
		} `json:"data"`
	}
	r := hh.GET(opsHealthPath, tok)
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**الجسدُ ليس JSON**: %v", err)
	}
	got := env.Data
	if got.Status != "ok" {
		t.Errorf("**الحالُ في بيئةٍ سليمةٍ يجب `ok`** — وهو %q", got.Status)
	}
	if got.Pool.Max <= 0 {
		t.Errorf("**سقفُ المسبح يجب أن يُقرأ** — %d", got.Pool.Max)
	}
	if got.Pool.Total <= 0 {
		t.Errorf("**والمسبحُ مأهولٌ أثناء نداءٍ حيّ** — %d", got.Pool.Total)
	}
	if !got.PG.Reachable || !got.Redis.Reachable {
		t.Errorf("**القاعدةُ والذاكرةُ مُجيبتان هنا** — pg=%v redis=%v",
			got.PG.Reachable, got.Redis.Reachable)
	}
	if got.PG.Backends <= 0 {
		t.Errorf("**ولا بدَّ من جلسةٍ واحدةٍ على الأقلّ** — %d", got.PG.Backends)
	}
	if got.Runtime.Goroutines <= 0 {
		t.Errorf("**عددُ الخيوط يجب أن يُقرأ** — %d", got.Runtime.Goroutines)
	}
	t.Logf("A4: المسبحُ %d/%d مُعارٌ %d · جلساتُ القاعدة %d · خيوطٌ %d",
		got.Pool.Total, got.Pool.Max, got.Pool.Acquired,
		got.PG.Backends, got.Runtime.Goroutines)
}

// TestOps70A_A5_PublicHealthStaysMinimal **العامُّ لا يُصبح تشخيصاً.**
//
// **وحارسٌ دائمٌ لا ملاحظةٌ في مراجعة** — **ومن وسّعه غداً يسقط هنا.**
func TestOps70A_A5_PublicHealthStaysMinimal(t *testing.T) {
	hh := New(t)
	r := hh.GET("/healthz", "")
	if r.Code != http.StatusOK {
		t.Fatalf("**`/healthz` يجب ٢٠٠ في بيئةٍ سليمة** — %d", r.Code)
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**ليس JSON**: %v", err)
	}
	m := env.Data
	// **ثلاثةُ حقولٍ لا أكثر** — والرابعُ يحتاج قراراً لا صمتاً.
	want := map[string]bool{"status": true, "postgres": true, "redis": true}
	for k := range m {
		if !want[k] {
			t.Errorf("**حقلٌ زائدٌ في البابِ العامّ**: %q", k)
		}
	}
	for _, k := range []string{"status", "postgres", "redis"} {
		if _, ok := m[k]; !ok {
			t.Errorf("**حقلٌ اختفى من عقدِ البوّابة**: %q", k)
		}
	}
	// **ولا قيمةَ خارج المعجم** — **ونصُّ الخطأ كان يحمل المضيفَ
	// واسمَ القاعدة.**
	for _, k := range []string{"postgres", "redis"} {
		v, _ := m[k].(string)
		if v != "ok" && v != "down" {
			t.Errorf("**قيمةٌ خارج المعجم في %q**: %q", k, v)
		}
	}
	for _, pat := range []string{"postgres://", "redis://", "user=", "dbname=",
		"127.0.0.1", "localhost", "dial tcp", "password"} {
		if strings.Contains(string(r.Body), pat) {
			t.Errorf("**البابُ العامُّ يرسم البنية**: %q", pat)
		}
	}
	t.Logf("A5: `/healthz` = %s", string(r.Body))
}
