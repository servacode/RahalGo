package qa

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **أيُّ نسخةٍ تطرق البابَ — تجميعاً** (دورةُ ٧٠أ · `Phase 13`)
// ══════════════════════════════════════════════════════════════════════
//
// **ودورةُ ٦٩ج انتهت إلى `CLIENT VERSION ATTRIBUTION = NOT POSSIBLE`**
// — **لأنّ سجلَّ البوّابة يحذف الترويسات جملةً، وهو صواب.** **فيُعدَّ
// في المحرّك حيث الترويسةُ مقروءةٌ أصلاً**، ولا يُضعَّف سجلُّ البوّابة.

func opsClients(t *testing.T, hh *Harness, tok string) (map[string]int64, int64) {
	t.Helper()
	r := hh.GET(opsHealthPath, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الرصد لا يُجيب** — %d: %s", r.Code, r.Body)
	}
	var env struct {
		Data struct {
			Clients        map[string]int64 `json:"clients"`
			ClientsDropped int64            `json:"clients_dropped"`
		} `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**الجسدُ ليس JSON**: %v", err)
	}
	return env.Data.Clients, env.Data.ClientsDropped
}

// TestOps70A_E1_ClientVersionIsCounted **نوعٌ ونسخةٌ يُعدّان.**
func TestOps70A_E1_ClientVersionIsCounted(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-ver", authz.ObservabilityRead)
	_, tok := capUser(t, hh, "qa-ops-ver")

	before, _ := opsClients(t, hh, tok)
	head := map[string]string{
		"X-RahalGo-Client":  "rahalgo-customer",
		"X-RahalGo-Version": "11",
	}
	for i := 0; i < 3; i++ {
		if r := hh.Call("GET", "/api/v1/public/platform", "", nil, head); r.Code != http.StatusOK {
			t.Fatalf("**مسارٌ عامٌّ يجب ٢٠٠** — %d", r.Code)
		}
	}
	after, dropped := opsClients(t, hh, tok)

	if after["customer:11"] != before["customer:11"]+3 {
		t.Fatalf("**`customer:11` لم يُعدّ ثلاثاً** — %d ← %d",
			before["customer:11"], after["customer:11"])
	}
	if dropped != 0 {
		t.Errorf("**لا يُهمَل شيءٌ عند مفتاحٍ واحد** — %d", dropped)
	}
	t.Logf("E1: customer:11 = %d · مُهمَلٌ = %d", after["customer:11"], dropped)
}

// TestOps70A_E2_UnknownClientIsNotCounted **المجهولُ لا يُعدّ.**
//
// **وهو ما يمنع انفجارَ الخريطة**: **مفتاحٌ يأتي من عميلٍ يُصدَّق
// بلا معجمٍ يبني خريطةً بلا قعر.**
func TestOps70A_E2_UnknownClientIsNotCounted(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-ver2", authz.ObservabilityRead)
	_, tok := capUser(t, hh, "qa-ops-ver2")

	before, _ := opsClients(t, hh, tok)
	for _, h := range []map[string]string{
		{"X-RahalGo-Client": "rahalgo-hacker", "X-RahalGo-Version": "11"},
		{"X-RahalGo-Client": "rahalgo-customer", "X-RahalGo-Version": "abc"},
		{"X-RahalGo-Client": "rahalgo-customer", "X-RahalGo-Version": "-5"},
	} {
		hh.Call("GET", "/api/v1/public/platform", "", nil, h)
	}
	after, dropped := opsClients(t, hh, tok)

	if len(after) != len(before) {
		t.Errorf("**مفتاحٌ مجهولٌ دخل الخريطة** — %v ← %v", before, after)
	}
	if dropped != 0 {
		t.Errorf("**والمجهولُ يُرَدّ لا يُهمَل بعد السقف** — %d", dropped)
	}
	t.Logf("E2: الخريطةُ %d مفتاحاً كما كانت", len(after))
}

// TestOps70A_E3_NoDeviceOrUserIdentityInClients **ولا هويّةَ في المفاتيح.**
//
// **والمفتاحُ `نوع:رقم` لا غير** — **ومن أضاف جهازاً أو حساباً بنى
// تتبّعاً باسم الرصد.**
func TestOps70A_E3_NoDeviceOrUserIdentityInClients(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa-ops-ver3", authz.ObservabilityRead)
	u, tok := capUser(t, hh, "qa-ops-ver3")

	hh.Call("GET", "/api/v1/public/platform", "", nil, map[string]string{
		"X-RahalGo-Client":  "rahalgo-driver",
		"X-RahalGo-Version": "9",
	})
	clients, _ := opsClients(t, hh, tok)
	for k := range clients {
		if k == u.ID || k == u.Phone {
			t.Fatalf("**هويّةٌ في مفتاح النسخ**")
		}
		kind, ver, ok := splitKey(k)
		if !ok {
			t.Errorf("**مفتاحٌ خارج الشكل `نوع:رقم`**: %q", k)
			continue
		}
		if !map[string]bool{"customer": true, "driver": true,
			"merchant": true, "rep": true}[kind] {
			t.Errorf("**نوعٌ خارج المعجم**: %q", kind)
		}
		if ver <= 0 {
			t.Errorf("**نسخةٌ غيرُ موجبة**: %q", k)
		}
	}
	t.Logf("E3: %d مفتاحاً كلُّها `نوع:رقم`", len(clients))
}

func splitKey(k string) (kind string, ver int, ok bool) {
	for i := 0; i < len(k); i++ {
		if k[i] == ':' {
			kind = k[:i]
			n := 0
			for _, c := range k[i+1:] {
				if c < '0' || c > '9' {
					return kind, 0, false
				}
				n = n*10 + int(c-'0')
			}
			return kind, n, kind != "" && n > 0
		}
	}
	return "", 0, false
}
