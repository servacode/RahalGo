package qa

// ══════════════════════════════════════════════════════════════════════
//  **قدراتُ صاحب الجلسة — بابٌ يُرسَم به لا يُخوَّل** (`MYC-1`)
// ══════════════════════════════════════════════════════════════════════
//
// # لماذا وُجد
//
// **القائمةُ الجانبيّةُ كانت تُبوَّب بأسماء الأدوار** (`roles:
// ["admin"]`) — **فدورٌ جديدٌ يحمل قدرةً لا يرى بابَها** حتّى يُكتب
// اسمُه في الواجهة. **وذاك `ADG-1` في العرض**: حقيقةٌ ثانيةٌ في
// العميل تفترق يوماً، وقد افترقت في الوجود من قبل.
//
// **وقِيس ٢٠٢٦-٠٩-١٢**: لا بابَ يكشف قدراتِ صاحب الجلسة —
// `/auth/capabilities` ⇒ 404 · `/admin/me/capabilities` ⇒ 403.
//
// # وما يُحرَس
//
//	١ · **بلا جلسةٍ ⇒ ٤٠١** — ولا تُقرأ قدراتُ أحدٍ بلا هويّة
//	٢ · **ويردّ ما يملكه صاحبُها بعينه** — لا أكثرَ ولا أقلّ
//	٣ · **ومن لا قدرةَ له يقرأ قائمةً فارغةً** لا `null` ولا خطأً
//	٤ · **ولا تخويلَ فيه**: من قرأ قدراتِه لا يبلغ بها باباً
//	    — **الحدُّ في جدول السياسة كما كان**

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

const myCapsPath = "/api/v1/auth/capabilities"

// myCaps **يقرأ القائمةَ من الجسم.**
func myCaps(t *testing.T, res Res) []string {
	t.Helper()
	var env struct {
		Data struct {
			Capabilities []string `json:"capabilities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("**جسمٌ ليس JSON صالحاً**: %v — %s", err, res.Body)
	}
	out := append([]string(nil), env.Data.Capabilities...)
	sort.Strings(out)
	return out
}

func TestMYC1_OwnCapabilitiesOnly(t *testing.T) {
	hh := New(t)

	// ── ١ · بلا جلسةٍ ⇒ ٤٠١ ─────────────────────────────────────────
	if r := hh.GET(myCapsPath, ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("**بلا جلسةٍ يجب ٤٠١** — وردَّ %d", r.Code)
	}

	// ── ٢ · ويردّ ما يملكه بعينه ────────────────────────────────────
	capRole(t, hh, "qa_myc_obs", authz.ObservabilityRead)
	_, obs := capUser(t, hh, "qa_myc_obs")
	got := myCaps(t, hh.GET(myCapsPath, obs))
	if strings.Join(got, "|") != string(authz.ObservabilityRead) {
		t.Errorf("**قدراتُ حساب الرصد %v** والمنتظَرُ [%s] وحدَها",
			got, authz.ObservabilityRead)
	}
	t.Logf("MYC-1: حسابُ الرصد ⇒ %v", got)

	// **وحسابٌ بقدرتين يقرأ اثنتين** — فالجوابُ يتبع الحقيقةَ لا ثابتاً.
	capRole(t, hh, "qa_myc_two", authz.ObservabilityRead, authz.OrdersRead)
	_, two := capUser(t, hh, "qa_myc_two")
	g2 := myCaps(t, hh.GET(myCapsPath, two))
	want := []string{string(authz.ObservabilityRead), string(authz.OrdersRead)}
	sort.Strings(want)
	if strings.Join(g2, "|") != strings.Join(want, "|") {
		t.Errorf("**حسابٌ بقدرتين ⇒ %v** والمنتظَرُ %v", g2, want)
	}
	t.Logf("MYC-1: حسابٌ بقدرتين ⇒ %v", g2)

	// ── ٣ · ومن لا قدرةَ له يقرأ فراغاً لا `null` ───────────────────
	_, plain := capUser(t, hh, "customer")
	r := hh.GET(myCapsPath, plain)
	if r.Code != http.StatusOK {
		t.Errorf("**زبونٌ يُردّ عن قدراتِه** — %d", r.Code)
	}
	if n := len(myCaps(t, r)); n != 0 {
		t.Errorf("**زبونٌ يقرأ %d قدرةً** — والمنتظَرُ صفر", n)
	}
	if strings.Contains(string(r.Body), "null") {
		t.Errorf("**`null` في جسمٍ يقرؤه عميل** — %s", r.Body)
	}

	// ── ٤ · **ولا تخويلَ فيه** ──────────────────────────────────────
	//
	// **ومن قرأ قدراتِه لا يبلغ بها باباً** — الحدُّ في جدول السياسة.
	if rr := hh.GET("/api/v1/admin/roles", obs); rr.Code != http.StatusForbidden {
		t.Errorf("**بابُ الأدوار انفتح لحساب الرصد** — %d", rr.Code)
	}
	if rr := hh.GET(opsHealthPath, obs); rr.Code != http.StatusOK {
		t.Errorf("**بابُ الرصد أُغلق على مالك قدرته** — %d", rr.Code)
	}
}

// TestMYC2_NoSecretsInBody **ولا يُسرّب الجسمُ شيئاً غيرَ القدرات.**
//
// **ورمزُ الجلسة والهاتفُ والبصمةُ لا شأنَ لها ببابِ قائمة.**
func TestMYC2_NoSecretsInBody(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_myc_priv", authz.ObservabilityRead)
	u, tok := capUser(t, hh, "qa_myc_priv")
	body := string(hh.GET(myCapsPath, tok).Body)

	for _, bad := range []string{
		u.ID, "phone", "+963", "password", "token", "session", "pin", "argon2", "hash",
	} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(bad)) {
			t.Errorf("**الجسمُ يحمل %q** — %s", bad, body)
		}
	}
	// **ومفتاحٌ واحدٌ في الأعلى** — فلا حقلَ يُضاف غداً بلا مراجعة.
	var raw map[string]any
	if err := json.Unmarshal(hh.GET(myCapsPath, tok).Body, &raw); err != nil {
		t.Fatalf("جسمٌ غيرُ صالح: %v", err)
	}
	data, ok := raw["data"].(map[string]any)
	if !ok {
		t.Fatalf("**لا `data` في الجسم** — %v", keysOf(raw))
	}
	if len(data) != 1 {
		t.Errorf("**حقولٌ في الجسم غيرُ القدرات**: %v", keysOf(data))
	}
	t.Logf("MYC-2: مفاتيحُ الجسم = %v", keysOf(data))
}
