package qa

import (
	"sort"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **قرارُ المالك — مصفوفةُ القسمين وتقاعدُ `ops`** (٢٠٢٦-٠٩-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا يُكتب القرارُ اختباراً
//
// **وقرارٌ في رسالةٍ يُنسى، وقرارٌ في تعليقٍ يُقرأ ولا يُفرَض.**
// **ومصفوفةٌ تُصحَّح اليومَ تُوسَّع غداً بسطرٍ في نافذةِ أدوار** — **ولا
// يصرخ شيء.**
//
// # نصُّ القرار
//
//	operations   قدراتٌ بعينها لتسعة أقسام (قرارُ المالك ٢٠٢٦-١٠-٠٥)
//	finance      تسعُ قدراتٍ بعينها — ولا تدخّلَ تشغيليّ
//	ops          إرثٌ متقاعد — **ثمّ حُذف** بقرار المالك ٢٠٢٦-١٠-٠٤ (هجرة 0270)

// opsFinalCaps **قدراتُ العمليّات كما أقرّها المالك** — لا أكثرَ ولا أقلّ.
//
// **و`emergencies.manage` بقرار المالك ٢٠٢٦-١٠-٠٤** (قسمُ «الطلبات»، البند ٧).
//
// **وقرارُ المالك ٢٠٢٦-١٠-٠٥** (هجرة `0420`): تسعةُ أقسامٍ لا غير — فنالت
// `market.manage` (السوق) و`support.manage` (الشكاوى والتقييمات) و`merchants.verify`
// (طلبات الانضمام)، **ونُزعت `users.read` (الحسابات) و`analytics.read` (التقارير).**
var opsFinalCaps = []string{
	"drivers.manage", "drivers.read", "emergencies.manage", "market.manage",
	"merchants.read", "merchants.verify",
	// **و`orders.customer_details.read` بقرار المالك ٢٠٢٦-١٠-٠٤** (سجلُّ الطلبات، البند ٣).
	"orders.customer_details.read",
	"orders.intervene", "orders.read", "settings.read", "support.manage",
	"users.contact.read",
}

// financeFinalCaps **قدراتُ الماليّة كما أقرّها المالك.**
//
// **و`audit.read` بقرار المالك 2026-10-04** (سجلُّ الأحداث — هجرة `0260`).
var financeFinalCaps = []string{
	// **و`disputes.manage`** — الماليّةُ ترى النزاعاتِ التي تحسمها (قرارُ المالك ٢٠٢٦-١٠-٠٤، «الخسائر والنزاعات» البند ١).
	"analytics.read", "audit.read", "disputes.manage", "finance.export", "finance.manage", "finance.read",
	"orders.read", "payouts.decide", "settings.financial.manage",
	"settings.read", "users.read",
}

// roleCaps قدراتُ دورٍ من القاعدة مرتَّبةً.
func roleCaps(t *testing.T, hh *Harness, role string) []string {
	t.Helper()
	rows, err := hh.Pool.Query(ctxBG(),
		`SELECT capability_code FROM role_capabilities WHERE role_code = $1`, role)
	if err != nil {
		t.Fatalf("قراءةُ قدرات %s: %v", role, err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("مسحُ صفّ: %v", err)
		}
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// TestOFM5_DepartmentCapabilitiesAreExactlyApproved **لا زيادةَ ولا نقص.**
//
// **ومساواةٌ تامّةٌ لا احتواء**: **قدرةٌ تُضاف بحسن نيّةٍ توسّع قسماً
// بلا قرار.**
func TestOFM5_DepartmentCapabilitiesAreExactlyApproved(t *testing.T) {
	hh := New(t)
	for _, c := range []struct {
		role string
		want []string
	}{
		{"operations", opsFinalCaps},
		{"finance", financeFinalCaps},
	} {
		got := roleCaps(t, hh, c.role)
		if strings.Join(got, ",") != strings.Join(c.want, ",") {
			t.Errorf("**قدراتُ `%s` فارقت قرارَ المالك**\n   الموجود = %s\n   المُقرَّر = %s",
				c.role, strings.Join(got, " · "), strings.Join(c.want, " · "))
		} else {
			t.Logf("✓ %-12s %d قدرةً — كما أُقرّت", c.role, len(got))
		}
	}

	// **ولا مالَ ولا أدوارَ ولا إعداداتٍ ولا عروضَ ولا حملاتٍ ولا سجلّ** — ولا
	// الحساباتُ ولا التقارير (قرارُ المالك ٢٠٢٦-١٠-٠٥). **وكانت `support.manage`
	// ممنوعةً هنا ثمّ منحها المالكُ لقسم «الشكاوى والتقييمات».**
	for _, forbidden := range []string{
		"users.read", "analytics.read", "content.manage", "merchants.manage",
		"finance.read", "finance.manage", "finance.export", "payouts.decide",
		"roles.manage", "safety.manage", "audit.read", "platform.overview",
		"settings.general.manage", "settings.financial.manage", "settings.security.manage",
		"observability.read",
	} {
		for _, c := range roleCaps(t, hh, "operations") {
			if c == forbidden {
				t.Errorf("**`operations` نالت `%s`** — ومنعَها المالكُ نصّاً.", forbidden)
			}
		}
	}
}

// TestOFM6_OpsRoleIsDeleted **`ops` حُذف** (قرارُ المالك ٢٠٢٦-١٠-٠٤، قسمُ الأدوار، البند ٣).
//
// **كان إرثاً يُقرأ ويُنزَع ولا يُمنَح** (٢٠٢٦-٠٩-١٣) — **وقرّر المالكُ حذفَه**
// بعد نقل حامليه إلى `operations` (هجرةُ `0270`). **فيُقاس**: لا صفَّ له، ولا
// يُمنَح، ولا يُنشَأ به حساب، و`operations` هو الدورُ العامل.
func TestOFM6_OpsRoleIsDeleted(t *testing.T) {
	hh := New(t)

	// ── ١ · صفُّ الدور محذوف ───────────────────────────────────────────
	var exists bool
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT EXISTS (SELECT 1 FROM roles WHERE code = 'ops')`).Scan(&exists); err != nil {
		t.Fatalf("وجودُ الدور: %v", err)
	}
	if exists {
		t.Fatal("**صفُّ `ops` باقٍ** — والمالكُ قرّر حذفَه (هجرة 0270).")
	}

	// ── ٢ · ولا منحَ له — ولو من المالك ────────────────────────────────
	victim := hh.NewUser("customer")
	_, ownerTok := roleUser(t, hh, "owner_super_admin")
	res := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", ownerTok,
		map[string]any{"role": "ops", "reason": "OFM6"})
	if res.Code < 400 {
		t.Errorf("**مُنح `ops` بعد حذفه**: %d / %s", res.Code, res.Err())
	}

	// ── ٣ · ولا حسابٌ يُنشَأ به ─────────────────────────────────────
	create := hh.POST("/api/v1/admin/users", ownerTok, map[string]any{
		"phone": "+963900000931", "full_name": "قياسُ الحذف",
		"password": "Ops#Retired2026", "roles": []string{"ops"},
	})
	if create.Code < 400 {
		t.Errorf("**حسابٌ أُنشئ بدورٍ محذوف** (%d)", create.Code)
	}

	// ── ٤ · و`operations` هو الدورُ العامل ──────────────────────────
	if got := roleCaps(t, hh, "operations"); len(got) == 0 {
		t.Error("**`operations` بلا قدرات** — وهو الدورُ العاملُ للتشغيل.")
	}
}
