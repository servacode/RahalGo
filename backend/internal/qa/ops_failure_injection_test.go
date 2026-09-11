package qa

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **ورصدٌ لا يرى العطبَ زينةٌ** — حقنُ الأعطال (دورةُ ٧٠أ · `Phase 16`)
// ══════════════════════════════════════════════════════════════════════
//
// **وبابٌ يقول «سليم» أبداً يمرّ في كلّ اختبارٍ ولا يقيس شيئاً.**
// **فيُسقَط ما يراقبه ويُقرأ ما يقول.**

// opsStatus **يقرأ الحالَ وحالَ التبعيّتين من الباب المحميّ.**
func opsStatus(t *testing.T, hh *Harness, tok string) (status string, pg, redis bool) {
	t.Helper()
	r := hh.GET(opsHealthPath, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الرصد لا يُجيب** — %d: %s", r.Code, r.Body)
	}
	var env struct {
		Data struct {
			Status string `json:"status"`
			PG     struct {
				Reachable bool `json:"reachable"`
			} `json:"pg"`
			Redis struct {
				Reachable bool `json:"reachable"`
			} `json:"redis"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**الجسدُ ليس JSON**: %v", err)
	}
	return env.Data.Status, env.Data.PG.Reachable, env.Data.Redis.Reachable
}

// TestOps70A_C1_RedisDownIsReported **ذاكرةٌ ساقطةٌ تُقرأ في الباب.**
//
// **ولا تُسقَط ذاكرةُ الإنتاج ولا تُفرَّغ** — **يُغلَق العميلُ في هذا
// المِسند وحدَه**، فتصير كلُّ نداءاته خطأً، **وهو عينُ ما يقع حين
// تموت الحاوية.**
func TestOps70A_C1_RedisDownIsReported(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-c1", authz.ObservabilityRead)
	_, tok := capUser(t, hh, "qa-ops-c1")

	// **قبلُ: سليم.**
	if st, pg, rd := opsStatus(t, hh, tok); st != "ok" || !pg || !rd {
		t.Fatalf("**المِسندُ غيرُ سليمٍ قبل الحقن** — %s pg=%v redis=%v", st, pg, rd)
	}

	// **والإسقاطُ بإغلاق عميل هذا المِسند.**
	if err := hh.Redis().Close(); err != nil {
		t.Fatalf("إغلاقُ عميل الذاكرة: %v", err)
	}

	st, pg, rd := opsStatus(t, hh, tok)
	if rd {
		t.Errorf("**الذاكرةُ ساقطةٌ والبابُ يقول إنّها تُجيب**")
	}
	if st != "degraded" {
		t.Errorf("**والحالُ يجب `degraded`** — وهو %q", st)
	}
	if !pg {
		t.Errorf("**والقاعدةُ سليمةٌ ولا تُجرّ مع الذاكرة** — pg=%v", pg)
	}
	t.Logf("C1: حالٌ=%q · pg=%v · redis=%v", st, pg, rd)
}

// TestOps70A_C2_PublicHealthAlsoFlipsButStaysSilent **والعامُّ يقول
// «ساقطة» ولا يقول لماذا.**
//
// **وهذا هو العقدُ كلُّه**: **يتبدّل فيُعرف أنّ ثمّة عطباً، ولا يرسم
// البنيةَ لمن سأله بلا اسم.**
func TestOps70A_C2_PublicHealthAlsoFlipsButStaysSilent(t *testing.T) {
	hh := New(t)
	if err := hh.Redis().Close(); err != nil {
		t.Fatalf("إغلاقُ عميل الذاكرة: %v", err)
	}
	r := hh.GET("/healthz", "")
	if r.Code != http.StatusServiceUnavailable {
		t.Fatalf("**المنتظَرُ ٥٠٣** — %d: %s", r.Code, r.Body)
	}
	var env struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**ليس JSON**: %v", err)
	}
	if env.Data["redis"] != "down" {
		t.Errorf("**المنتظَرُ `down` من معجمه** — %v", env.Data["redis"])
	}
	// **ونصُّ الخطأ كان يُطبع هنا** — والمضيفُ والمنفذُ معه.
	for _, pat := range []string{"dial tcp", "closed", "connection", "127.0.0.1",
		"localhost", "6379", "6380", "redis://"} {
		if strings.Contains(string(r.Body), pat) {
			t.Errorf("**البابُ العامُّ يرسم البنية**: %q في %s", pat, r.Body)
		}
	}
	t.Logf("C2: ٥٠٣ · %s", r.Body)
}
