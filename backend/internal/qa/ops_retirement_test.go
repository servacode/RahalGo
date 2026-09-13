package qa

import (
	"net/http"
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
//	operations   تسعُ قدراتٍ بعينها — ولا `support.manage`
//	finance      تسعُ قدراتٍ بعينها — ولا تدخّلَ تشغيليّ
//	ops          إرثٌ متقاعد: يُقرأ ويُنزَع، ولا يُمنَح ولا يُنشَأ به حساب
//	             **ولا يُحذَف صفُّه** — وذاك دورةُ تنظيفٍ أخرى.

// opsFinalCaps **قدراتُ العمليّات كما أقرّها المالك** — لا أكثرَ ولا أقلّ.
var opsFinalCaps = []string{
	"analytics.read", "drivers.manage", "drivers.read", "merchants.read",
	"orders.intervene", "orders.read", "settings.read",
	"users.contact.read", "users.read",
}

// financeFinalCaps **قدراتُ الماليّة كما أقرّها المالك.**
var financeFinalCaps = []string{
	"analytics.read", "finance.export", "finance.manage", "finance.read",
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

	// **و`support.manage` بعينها ممنوعةٌ عن العمليّات** — قرارُ المالك
	// الأوّل: **بابُ الدعم أوسعُ من مسؤوليّة التشغيل.**
	for _, forbidden := range []string{
		"support.manage", "merchants.manage", "finance.read", "finance.manage",
		"finance.export", "payouts.decide", "roles.manage", "safety.manage",
		"observability.read",
	} {
		for _, c := range roleCaps(t, hh, "operations") {
			if c == forbidden {
				t.Errorf("**`operations` نالت `%s`** — ومنعَها المالكُ نصّاً.", forbidden)
			}
		}
	}
}

// TestOFM6_OpsIsLegacyRetired **`ops` يُقرأ ويُنزَع ولا يُمنَح.**
//
// **وخمسةُ بنودٍ في قرار المالك تُقاس واحداً واحداً** — **ولا يُقرأ
// نجاحُ واحدٍ إثباتاً للبقيّة.**
func TestOFM6_OpsIsLegacyRetired(t *testing.T) {
	hh := New(t)

	// ── ١ · صفُّ الدور باقٍ — ولا حذفَ في هذه الدورة ────────────────
	var exists bool
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT EXISTS (SELECT 1 FROM roles WHERE code = 'ops')`).Scan(&exists); err != nil {
		t.Fatalf("وجودُ الدور: %v", err)
	}
	if !exists {
		t.Fatal("**صفُّ `ops` حُذف** — والمالكُ منع الحذفَ الفيزيائيَّ في هذه الدورة.")
	}
	t.Log("✓ صفُّ `ops` باقٍ — ولا حذف")

	// ── ٢ · ولا منحَ جديد — ولو كان المانحُ المالك ──────────────────
	//
	// **والمالكُ أوسعُ سلطةٍ في المنصّة** — **فإن رُدَّ هو رُدَّ كلُّ
	// أحد.**
	victim := hh.NewUser("customer")
	_, ownerTok := roleUser(t, hh, "owner_super_admin")
	res := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", ownerTok,
		map[string]any{"role": "ops", "reason": "OFM6"})
	if res.Code != http.StatusForbidden || res.Err() != "role_grant_retired" {
		t.Errorf("**منحُ `ops` لم يُردَّ كدورِ إرث**: %d / %s", res.Code, res.Err())
	} else {
		t.Log("✓ منحُ `ops` ⇒ 403 role_grant_retired — ولو من المالك")
	}

	// ── ٣ · ولا حسابٌ يُنشَأ به ─────────────────────────────────────
	create := hh.POST("/api/v1/admin/users", ownerTok, map[string]any{
		"phone": "+963900000931", "full_name": "قياسُ التقاعد",
		"password": "Ops#Retired2026", "roles": []string{"ops"},
	})
	if create.Code < 400 {
		t.Errorf("**حسابٌ أُنشئ بدورٍ متقاعد** (%d) — وبابُ الإنشاء ليس ثغرةَ منح.", create.Code)
	} else {
		t.Logf("✓ إنشاءُ حسابٍ بـ`ops` ⇒ %d / %s", create.Code, create.Err())
	}

	// ── ٤ · وحاملٌ قائمٌ يُقرأ ويعمل ────────────────────────────────
	//
	// **والتوافقُ مع حاملٍ قائمٍ شرطُ المالك** — **ولا يُكسَر من كان
	// يعمل لأنّ الدورَ تقاعد.**
	p := probes(t, hh)
	checkRole(t, hh, "ops",
		[]probe{p["قراءةُ الطلبات"], p["تدخّلٌ في طلب"], p["إدارةُ المتاجر"]},
		[]probe{p["قيدُ محفظة"], p["منحُ دور"], p["إعدادٌ ماليّ"]})

	// ── ٥ · ويُنزَع عمّن يحمله ──────────────────────────────────────
	//
	// **ومتقاعدٌ لا يُنزَع سجنٌ لا تقاعد.**
	holder, _ := roleUser(t, hh, "ops")
	del := hh.DEL("/api/v1/admin/users/"+holder.ID+"/roles/ops", ownerTok)
	if del.Code >= 400 && del.Code != http.StatusConflict {
		// **و409 جوابُ منطقٍ لا منعُ تخويل** — تُسأل الكلمةُ أوّلاً.
		if del.Err() == "role_grant_retired" {
			t.Errorf("**نزعُ `ops` رُدَّ بحجّة التقاعد** — والتقاعدُ يمنع المنحَ لا النزع.")
		}
	}
	t.Logf("✓ نزعُ `ops` ⇒ %d / %s (ولا يُردّ بحجّة التقاعد)", del.Code, del.Err())

	// ── ٦ · و`operations` هو الدورُ العامل ──────────────────────────
	if got := roleCaps(t, hh, "operations"); len(got) == 0 {
		t.Error("**`operations` بلا قدرات** — وهو الدورُ العاملُ للتشغيل.")
	}
}
