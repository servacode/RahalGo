package authz

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الأدوار والصلاحيّات — قرارات المالك ٢٠٢٦-١٠-٠٤** (حرّاسُ المنطق)
// ══════════════════════════════════════════════════════════════════════

// TestROLES_AccountTypeRolesTakeNoAdminCapability **ولا قدرةَ إداريّةَ لأدوار الحسابات.**
func TestROLES_AccountTypeRolesTakeNoAdminCapability(t *testing.T) {
	for _, role := range []string{"customer", "driver", "merchant", "sales"} {
		for _, c := range All() {
			if got := CanGrantCapability(role, c); got != CapBlockAccountType {
				t.Errorf("**%s يقبل %s** (%q) — وصار كلُّ حاملٍ له موظّفاً", role, c, got)
			}
		}
	}
}

// TestROLES_OwnerRoleCapabilitiesFrozen **ولا أحدَ يحرّر قدرات المالك الأعلى.**
func TestROLES_OwnerRoleCapabilitiesFrozen(t *testing.T) {
	for _, c := range All() {
		if CanGrantCapability(RoleOwnerSuperAdmin, c) != CapBlockProtected {
			t.Errorf("**يُمنَح %s لدور المالك الأعلى**", c)
		}
	}
	if CanRevokeCapability(RoleOwnerSuperAdmin) != CapBlockProtected {
		t.Error("**يُنزَع من دور المالك الأعلى**")
	}
}

// TestROLES_RolesManageOnlyForAdminAndOwner **`roles.manage` لـ`admin` والمالك وحدَهما.**
func TestROLES_RolesManageOnlyForAdminAndOwner(t *testing.T) {
	for _, role := range []string{"customer_support", "finance", "operations", "zz_custom"} {
		if CanGrantCapability(role, RolesManage) != CapBlockRolesManageScope {
			t.Errorf("**`roles.manage` تُمنَح لـ%s**", role)
		}
	}
	if CanGrantCapability(RoleAdmin, RolesManage) != CapOK {
		t.Error("**`roles.manage` لا تُمنَح لمدير المنصّة**")
	}
	if !GrantCapabilityNeedsOwner(RolesManage) || GrantCapabilityNeedsOwner(FinanceRead) {
		t.Error("**سلطةُ منح `roles.manage` ليست للمالك وحدَه**")
	}
	// **وبقيّةُ القدرات تُمنَح للموظّفين عاديّاً.**
	if CanGrantCapability("finance", FinanceManage) != CapOK {
		t.Error("**قدرةٌ عاديّةٌ مُنعت عن دور موظّفين**")
	}
}

// TestROLES_DeletableOnlyCustomAndLegacy **الحذفُ للمخصَّص والقديم وحدَهما.**
func TestROLES_DeletableOnlyCustomAndLegacy(t *testing.T) {
	for _, r := range []string{"zz_custom", "s1_ops_viewer", "ops"} {
		if !RoleDeletable(r) {
			t.Errorf("**%s لا يُحذَف**", r)
		}
	}
	for _, r := range []string{"admin", RoleOwnerSuperAdmin, "finance", "customer", "driver"} {
		if RoleDeletable(r) {
			t.Errorf("**%s يُحذَف** — وهو كانونيّ", r)
		}
	}
}

// TestROLES_CapabilityDescriptionsPlainAndAccurate **أوصافٌ بسيطةٌ صادقة.**
//
// بلا تشكيلٍ ثقيل · «إعدادات عامة» = المناطق والمدن والدوام · «إدارة المتاجر»
// بلا تعليق · `roles.manage` أقوى صلاحيّة.
func TestROLES_CapabilityDescriptionsPlainAndAccurate(t *testing.T) {
	for _, c := range All() {
		d := Describe(c)
		for _, r := range d {
			if r >= 0x064B && r <= 0x0652 {
				t.Errorf("**وصفُ %s مشكول**: %q", c, d)
				break
			}
		}
	}
	gen := Describe(SettingsGeneralManage)
	if strings.Contains(gen, "محتوى") || !strings.Contains(gen, "المناطق") {
		t.Errorf("**«إعدادات عامة» لا تقول ما تفتحه**: %q", gen)
	}
	if !strings.Contains(Describe(MerchantsManage), "بدون تعليق") {
		t.Errorf("**«إدارة المتاجر» توحي بالتعليق**: %q", Describe(MerchantsManage))
	}
	if !strings.Contains(Describe(RolesManage), "أقوى") {
		t.Errorf("**`roles.manage` لا تُوصَف بأنّها أقوى صلاحيّة**: %q", Describe(RolesManage))
	}
}

// TestROLES_EveryCapabilityHasAGroup **كلُّ قدرةٍ في مجموعةٍ من السبع.**
func TestROLES_EveryCapabilityHasAGroup(t *testing.T) {
	known := map[Group]bool{}
	for _, g := range Groups {
		known[g] = true
	}
	if len(Groups) != 7 {
		t.Fatalf("**المجموعاتُ %d لا سبع**", len(Groups))
	}
	for _, c := range All() {
		g, ok := groupOf[c]
		if !ok || !known[g] {
			t.Errorf("**%s بلا مجموعة**", c)
		}
	}
	// **والمالُ والسلطةُ معلَّمان.**
	for _, c := range []Capability{FinanceManage, PayoutsDecide, RolesManage} {
		if RiskOf(c) == RiskNone {
			t.Errorf("**%s غيرُ معلَّمٍ بالأحمر**", c)
		}
	}
}
