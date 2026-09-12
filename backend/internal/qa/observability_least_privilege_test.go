package qa

// ══════════════════════════════════════════════════════════════════════
//  **أقلُّ امتيازٍ لحساب الرصد — بابٌ واحدٌ يُقرأ ولا غير** (`OLP-1`)
// ══════════════════════════════════════════════════════════════════════
//
// # حالُ الإنتاج التي تُحاكى هنا
//
// **قِيس على الإنتاج ٢٠٢٦-٠٩-١٢** (قراءةٌ محضة):
//
//	الدورُ `observability`  قدراتُه = **observability.read وحدَها**
//	حسابُ الرصد            أدوارُه = customer, observability
//	قدراتُه الفاعلة        = **observability.read وحدَها**
//
// **فيُبنى الشكلُ نفسُه هنا** — دورٌ بقدرةٍ واحدةٍ ومعه `customer` —
// **ويُسأل كلُّ بابٍ إداريٍّ في المنصّة.**
//
// # ولماذا فحصٌ لا نداءٌ في الإنتاج
//
// **لا كلمةَ مرورٍ لحساب الرصد عندي، ولا تُبدَّل** (قرارُ المالك:
// لا تبديلَ في الإنتاج). **وإخفاءُ زرٍّ ليس حدّاً** — **والحدُّ في
// جدول السياسة**، وهو ما يُسأل هنا بالموجّه الحقيقيّ.
//
// # وما يُحرَس
//
//	١ · **`GET /ops/health` ⇒ ٢٠٠** — والبابُ المقصودُ يعمل
//	٢ · **وكلُّ بابٍ آخرَ ⇒ ٤٠٣** — إدارةٌ ومالٌ وطلباتٌ ومتاجرُ
//	    وسائقونَ وسلامةٌ وإعداداتٌ وأدوار
//	٣ · **والنداءُ المباشرُ لا يتجاوز شيئاً** — لا واجهةَ في الطريق
//	٤ · **ولا قدرةً ثانيةً تُكتسَب** من `customer`

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// obsAccount **حسابُ رصدٍ بشكل الإنتاج** — قدرةٌ واحدةٌ ومعه الزبون.
func obsAccount(t *testing.T, hh *Harness) (*User, string) {
	t.Helper()
	capRole(t, hh, "qa_obs_prod", authz.ObservabilityRead)
	return capUser(t, hh, "qa_obs_prod")
}

// TestOLP1_ObservabilityReadOpensOnlyOpsHealth **بابٌ واحدٌ ولا غير.**
func TestOLP1_ObservabilityReadOpensOnlyOpsHealth(t *testing.T) {
	hh := New(t)
	u, tok := obsAccount(t, hh)

	// **وقدراتُه الفاعلةُ واحدةٌ** — كما في الإنتاج.
	var caps []string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(array_agg(DISTINCT rc.capability_code
		                          ORDER BY rc.capability_code), '{}')
		  FROM user_roles ur JOIN role_capabilities rc ON rc.role_code = ur.role_code
		 WHERE ur.user_id = $1::uuid`, u.ID).Scan(&caps); err != nil {
		t.Fatalf("قراءةُ القدرات: %v", err)
	}
	t.Logf("OLP-1: قدراتٌ فاعلةٌ = %v", caps)
	if len(caps) != 1 || caps[0] != string(authz.ObservabilityRead) {
		t.Fatalf("**شكلُ الحساب لا يحاكي الإنتاج** — %v", caps)
	}

	// ── ١ · البابُ المقصود ──────────────────────────────────────────
	if r := hh.GET(opsHealthPath, tok); r.Code != http.StatusOK {
		t.Fatalf("**بابُ الرصد لا يُقرأ لمالك قدرته** — %d: %s", r.Code, r.Body)
	}

	// ── ٢ · **وكلُّ بابٍ آخرَ يُردّ** ───────────────────────────────
	//
	// **والمسارُ من جدول السياسة لا مُختَرَعٌ** — وقدرتُه المطلوبةُ
	// مكتوبةٌ بجانبه ليُقرأ **لماذا** يُردّ.
	victim, _ := capUser(t, hh, "customer")
	id := victim.ID
	denials := []struct {
		domain string
		p      probe
		need   authz.Capability
	}{
		{"إدارةُ الحسابات", probe{Method: "GET", Path: "/api/v1/admin/users"}, authz.UsersRead},
		{"إدارةُ الحسابات", probe{Method: "GET", Path: "/api/v1/admin/users/" + id}, authz.UsersRead},
		{"تصديرُ الحسابات", probe{Method: "GET", Path: "/api/v1/admin/users/export"}, authz.UsersExport},
		{"الأدوارُ والقدرات", probe{Method: "GET", Path: "/api/v1/admin/roles"}, authz.RolesManage},
		{"الأدوارُ والقدرات", probe{Method: "GET", Path: "/api/v1/admin/capabilities"}, authz.RolesManage},
		{"منحُ دور", probe{Method: "POST", Path: grantPath(id),
			Body: map[string]any{"role": "finance", "reason": "OLP"}}, authz.RolesManage},
		{"المال", probe{Method: "GET", Path: "/api/v1/admin/payouts"}, authz.FinanceRead},
		{"المال", probe{Method: "GET", Path: "/api/v1/admin/profits"}, authz.FinanceRead},
		{"المال", probe{Method: "GET", Path: "/api/v1/admin/reports/losses"}, authz.FinanceRead},
		{"المحافظ", probe{Method: "GET", Path: "/api/v1/admin/users/" + id + "/wallet"}, authz.FinanceRead},
		{"المحافظ", probe{Method: "POST", Path: "/api/v1/admin/users/" + id + "/wallet",
			Body: walletBody(100)}, authz.FinanceManage},
		{"الصناديق", probe{Method: "GET", Path: "/api/v1/admin/cash/outstanding"}, authz.FinanceRead},
		{"الصناديق", probe{Method: "GET", Path: "/api/v1/admin/drivers/" + id + "/cash"}, authz.FinanceRead},
		{"الطلبات", probe{Method: "GET", Path: "/api/v1/admin/orders"}, authz.OrdersRead},
		{"تدخّلُ الطلبات", probe{Method: "POST", Path: "/api/v1/admin/orders/" + id + "/assign",
			Body: map[string]any{"driver_id": id}}, authz.OrdersIntervene},
		{"المتاجر", probe{Method: "GET", Path: "/api/v1/admin/merchants"}, authz.MerchantsManage},
		{"السائقون", probe{Method: "GET", Path: "/api/v1/admin/drivers"}, authz.DriversRead},
		{"الثقةُ والسلامة", probe{Method: "GET", Path: "/api/v1/admin/users/" + id + "/warnings"}, authz.SafetyManage},
		{"الثقةُ والسلامة", probe{Method: "GET", Path: "/api/v1/admin/merchants/" + id + "/violations"}, authz.SafetyManage},
		{"الدعم", probe{Method: "GET", Path: "/api/v1/admin/tickets"}, authz.SupportManage},
		{"الإعدادات", probe{Method: "GET", Path: "/api/v1/admin/settings"}, authz.SettingsGeneralManage},
		{"سجلُّ التدقيق", probe{Method: "GET", Path: "/api/v1/admin/audit"}, authz.AuditRead},
	}

	bad := 0
	for _, d := range denials {
		no, code := denied(t, hh, tok, d.p)
		status := "٤٠٣"
		if !no {
			status = fmt.Sprintf("**%d**", code)
			bad++
		}
		t.Logf("   %-18s %-6s %-52s ⇒ %s   (يلزمه %s)",
			d.domain, d.p.Method, shortPath(d.p.Path), status, d.need)
	}
	if bad > 0 {
		t.Errorf("**%d باباً لم يُردّ** — **وقدرةُ الرصد فتحت ما ليس لها**", bad)
	}
}

// TestOLP2_NoUIInThePath **والنداءُ المباشرُ لا يتجاوز شيئاً.**
//
// **وإخفاءُ زرٍّ ليس حدّاً** — **والحدُّ في الوسيط، يُسأل بلا واجهة:**
// **لا متصفّحَ ولا صفحةَ ولا زرّ، نداءٌ خامٌّ على السلك.**
func TestOLP2_NoUIInThePath(t *testing.T) {
	hh := New(t)
	_, tok := obsAccount(t, hh)

	// **ولا مسارٌ ملتوٍ يمرّ**: نقاطٌ وشُرَطٌ وحروفٌ كبيرة.
	for _, p := range []string{
		"/api/v1/admin/roles",
		"/api/v1/admin/roles/",
		"/api/v1/admin/users",
		"/api/v1/admin/settings",
	} {
		if r := hh.GET(p, tok); r.Code != http.StatusForbidden && r.Code != http.StatusNotFound {
			t.Errorf("**%s ⇒ %d** — والمنتظَرُ منعاً", p, r.Code)
		}
	}
	// **والبابُ المقصودُ يبقى مفتوحاً** — فحارسٌ يمنع كلَّ شيءٍ لا يُقاس.
	if r := hh.GET(opsHealthPath, tok); r.Code != http.StatusOK {
		t.Errorf("**بابُ الرصد أُغلق** — %d", r.Code)
	}
}

// shortPath **يقصّ المعرّفاتَ الطويلةَ من المسار** — ليُقرأ السطر.
func shortPath(p string) string {
	out := ""
	for _, seg := range splitCSVBy(p, '/') {
		if len(seg) == 36 {
			seg = "{id}"
		}
		out += "/" + seg
	}
	return out
}

func splitCSVBy(s string, sep rune) []string {
	out := []string{}
	cur := ""
	for _, r := range s {
		if r == sep {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
