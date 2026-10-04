package qa

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **قسمُ الأدوار والصلاحيّات — حرّاسُ المحرّك** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **كلُّ حارسٍ يُقاس من الباب الحقيقيّ** — نداءٌ عبر الشبكة بجلسةٍ وتأكيد،
// ثمّ قراءةُ القاعدة: **منعٌ يترك أثراً ليس منعاً.** ولا توازيَ في الحزمة.

func roleCapPath(role string) string { return rolesPath + "/" + role + "/capabilities" }

func roleHasCap(t *testing.T, hh *Harness, role string, c authz.Capability) bool {
	t.Helper()
	var has bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM role_capabilities
		                WHERE role_code = $1 AND capability_code = $2)`,
		role, string(c)).Scan(&has); err != nil {
		t.Fatalf("قراءةُ القدرات: %v", err)
	}
	return has
}

func rolesAuditCount(t *testing.T, hh *Harness, action, entityID string) int {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM audit_log WHERE action = $1 AND entity_id = $2`,
		action, entityID).Scan(&n); err != nil {
		t.Fatalf("عدُّ السجلّ: %v", err)
	}
	return n
}

// TestROLES_AccountTypeRolesRejectAdminCapabilities **قيدُ محفظةٍ لدور «الزبون» يُردّ.**
func TestROLES_AccountTypeRolesRejectAdminCapabilities(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	for _, role := range []string{"customer", "driver", "merchant", "sales"} {
		r := hh.POST(roleCapPath(role), owner, map[string]any{
			"capability": string(authz.FinanceManage), "reason": "ROLES-1"})
		if r.Code != http.StatusForbidden || r.Err() != "role_account_type" {
			t.Errorf("**%s قبل قدرةً ماليّة**: %s", role, r)
		}
		if roleHasCap(t, hh, role, authz.FinanceManage) {
			t.Errorf("**مُنع المنحُ ووقع على %s**", role)
		}
	}
}

// TestROLES_OwnerRoleCapabilitiesUntouchable **ولا أحدَ يحرّر دورَ المالك الأعلى — ولا المالك.**
func TestROLES_OwnerRoleCapabilitiesUntouchable(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	rev := hh.DEL(roleCapPath("owner_super_admin")+"/"+string(authz.FinanceManage), owner)
	if rev.Code != http.StatusForbidden || rev.Err() != "role_protected" {
		t.Errorf("**نُزعت قدرةٌ من المالك الأعلى**: %s", rev)
	}
	if !roleHasCap(t, hh, "owner_super_admin", authz.FinanceManage) {
		t.Fatal("**دورُ المالك الأعلى خسر قدرة**")
	}
	gr := hh.POST(roleCapPath("owner_super_admin"), owner, map[string]any{
		"capability": string(authz.FinanceManage), "reason": "ROLES-2"})
	if gr.Code != http.StatusForbidden || gr.Err() != "role_protected" {
		t.Errorf("**حُرّر دورُ المالك الأعلى بالمنح**: %s", gr)
	}
}

// TestROLES_RolesManageOnlyAdminOwnerAndOnlyByOwner **`roles.manage` لا تُبذَر.**
func TestROLES_RolesManageOnlyAdminOwnerAndOnlyByOwner(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	_, admin := capUser(t, hh, "admin")
	capRole(t, hh, "qa_rm_scope")

	// **لدورٍ وظيفيٍّ أو مخصَّص ⇒ يُردّ ولو من المالك.**
	for _, role := range []string{"customer_support", "qa_rm_scope"} {
		r := hh.POST(roleCapPath(role), owner, map[string]any{
			"capability": string(authz.RolesManage), "reason": "ROLES-3"})
		if r.Code != http.StatusForbidden || r.Err() != "roles_manage_scope" {
			t.Errorf("**`roles.manage` مُنحت لـ%s**: %s", role, r)
		}
		if roleHasCap(t, hh, role, authz.RolesManage) {
			t.Errorf("**مُنع ووقع على %s**", role)
		}
	}
	// **ومديرُ المنصّة لا يمنحها — ولو لدور `admin` نفسِه.**
	r := hh.POST(roleCapPath("admin"), admin, map[string]any{
		"capability": string(authz.RolesManage), "reason": "ROLES-3"})
	if r.Code != http.StatusForbidden || r.Err() != "roles_manage_owner_only" {
		t.Errorf("**مديرُ المنصّة منح `roles.manage`**: %s", r)
	}
	// **والمالكُ يمنحها لـ`admin`** (قائمةٌ أصلاً ⇒ بلا تغيير).
	if r := hh.POST(roleCapPath("admin"), owner, map[string]any{
		"capability": string(authz.RolesManage), "reason": "ROLES-3"}); r.Code != http.StatusOK {
		t.Errorf("**المالكُ لا يمنح `roles.manage` لمدير المنصّة**: %s", r)
	}
}

// TestROLES_GrantNeedsReasonAndAuditsOnlyChanges **السببُ إلزاميّ · ولا سجلَّ لما لم يتغيّر.**
func TestROLES_GrantNeedsReasonAndAuditsOnlyChanges(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	capRole(t, hh, "qa_audit_change")
	cap := string(authz.AnalyticsRead)

	if r := hh.POST(roleCapPath("qa_audit_change"), owner, map[string]any{
		"capability": cap}); r.Code != http.StatusBadRequest || r.Err() != "reason_required" {
		t.Errorf("**مُنح بلا سبب**: %s", r)
	}
	if roleHasCap(t, hh, "qa_audit_change", authz.AnalyticsRead) {
		t.Fatal("**مُنع بلا سببٍ ووقع**")
	}

	g1 := hh.POST(roleCapPath("qa_audit_change"), owner, map[string]any{
		"capability": cap, "reason": "سبب المنح الأول"})
	if g1.Code != http.StatusOK {
		t.Fatalf("المنح: %s", g1)
	}
	if n := rolesAuditCount(t, hh, "admin.role_capability_grant", "qa_audit_change"); n != 1 {
		t.Fatalf("**المنحُ الأوّل كتب %d سطراً**", n)
	}
	var reason string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT details->>'reason' FROM audit_log
		 WHERE action = 'admin.role_capability_grant' AND entity_id = 'qa_audit_change'`).
		Scan(&reason); err != nil || reason != "سبب المنح الأول" {
		t.Errorf("**السببُ لم يُحفظ في السجلّ**: %q · %v", reason, err)
	}

	// **منحُ الموجود ⇒ لا سطر.**
	g2 := hh.POST(roleCapPath("qa_audit_change"), owner, map[string]any{
		"capability": cap, "reason": "مكرر"})
	if g2.Code != http.StatusOK {
		t.Fatalf("المنحُ المكرّر: %s", g2)
	}
	if n := rolesAuditCount(t, hh, "admin.role_capability_grant", "qa_audit_change"); n != 1 {
		t.Errorf("**منحُ قدرةٍ موجودةٍ كتب سطراً** (%d)", n)
	}

	// **نزعٌ حقيقيّ ⇒ سطر، ونزعُ الغائب ⇒ لا سطر.**
	for i := 0; i < 2; i++ {
		if r := hh.DEL(roleCapPath("qa_audit_change")+"/"+cap, owner); r.Code != http.StatusOK {
			t.Fatalf("النزع: %s", r)
		}
	}
	if n := rolesAuditCount(t, hh, "admin.role_capability_revoke", "qa_audit_change"); n != 1 {
		t.Errorf("**نزعُ الغائب كتب سطراً** (%d سطر)", n)
	}
}

// TestROLES_DeleteEmptyRoleOnly **«حذف الدور» للمخصَّص الفارغ وحدَه.**
func TestROLES_DeleteEmptyRoleOnly(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")

	// **كانونيٌّ ⇒ لا يُحذف.**
	if r := hh.DEL(rolesPath+"/finance", owner); r.Code != http.StatusForbidden ||
		r.Err() != "role_not_deletable" {
		t.Errorf("**حُذف دورٌ كانونيّ**: %s", r)
	}
	// **مخصَّصٌ عليه حاملٌ ⇒ لا يُحذف.**
	capRole(t, hh, "qa_del_busy")
	capUser(t, hh, "qa_del_busy")
	if r := hh.DEL(rolesPath+"/qa_del_busy", owner); r.Code != http.StatusConflict ||
		r.Err() != "role_not_empty" {
		t.Errorf("**حُذف دورٌ عليه حامل**: %s", r)
	}
	// **مخصَّصٌ فارغ ⇒ يُحذف ويُكتب.**
	capRole(t, hh, "qa_del_empty", authz.AnalyticsRead)
	if r := hh.DEL(rolesPath+"/qa_del_empty", owner); r.Code != http.StatusOK {
		t.Fatalf("**الدورُ الفارغُ لم يُحذف**: %s", r)
	}
	var exists bool
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT EXISTS (SELECT 1 FROM roles WHERE code = 'qa_del_empty')`).Scan(&exists)
	if exists {
		t.Error("**الدورُ باقٍ بعد الحذف**")
	}
	if n := rolesAuditCount(t, hh, "admin.role_delete", "qa_del_empty"); n < 1 {
		t.Error("**الحذفُ بلا سطرٍ في السجلّ**")
	}
}

// TestROLES_MembersAndImpactPreview **مَن يحمل الدور · وكم سيكسب القدرةَ فعلاً.**
func TestROLES_MembersAndImpactPreview(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	capRole(t, hh, "qa_impact")
	capRole(t, hh, "qa_impact_has", authz.FinanceManage)
	capUser(t, hh, "qa_impact")
	capUser(t, hh, "qa_impact", "qa_impact_has") // يملكها أصلاً

	m := hh.GET(rolesPath+"/qa_impact/members", owner)
	if m.Code != http.StatusOK {
		t.Fatalf("الحاملون: %s", m)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(m.Body, &env)
	if len(env.Data) != 2 {
		t.Errorf("**الحاملون %d لا ٢**", len(env.Data))
	}
	imp := hh.GET(rolesPath+"/qa_impact/impact?capability="+string(authz.FinanceManage), owner)
	if imp.Code != http.StatusOK {
		t.Fatalf("الأثر: %s", imp)
	}
	var ie struct {
		Data struct {
			Active int    `json:"active_members"`
			Gain   int    `json:"would_gain"`
			Risk   string `json:"risk"`
		} `json:"data"`
	}
	_ = json.Unmarshal(imp.Body, &ie)
	if ie.Data.Active != 2 || ie.Data.Gain != 1 || ie.Data.Risk != "money" {
		t.Errorf("**أثرُ المنح خطأ**: %+v", ie.Data)
	}
}

// TestROLES_ListCarriesClassAndGroups **القائمةُ بأقسامها والمعجمُ بمجموعاته.**
func TestROLES_ListCarriesClassAndGroups(t *testing.T) {
	hh := New(t)
	_, owner := capUser(t, hh, "owner_super_admin")
	roles := rolesOf(t, hh, owner)
	if len(roles) == 0 || roles[0]["class"] != "protected" {
		t.Errorf("**الإدارةُ العليا ليست أوّلاً**: %v", roles[0])
	}
	for _, r := range roles {
		if r["class"] == "account_type" && r["editable"] != false {
			t.Errorf("**نوعُ حسابٍ قابلٌ للتعديل**: %v", r["code"])
		}
		if r["code"] == "ops" || r["code"] == "s1_ops_viewer" || r["code"] == "s1_orders_only" {
			t.Errorf("**دورٌ قديمٌ/تجريبيٌّ باقٍ**: %v", r["code"])
		}
	}
	c := hh.GET("/api/v1/admin/capabilities", owner)
	var ce struct {
		Data []map[string]any `json:"data"`
	}
	_ = json.Unmarshal(c.Body, &ce)
	for _, x := range ce.Data {
		if x["group"] == "" || x["group"] == nil {
			t.Errorf("**قدرةٌ بلا مجموعة**: %v", x["code"])
		}
	}
}

// TestROLES_OwnerAssignedByMigration **حاملُ مدير المنصّة صار مالكاً أعلى** (هجرة 0270).
func TestROLES_OwnerAssignedByMigration(t *testing.T) {
	hh := New(t)
	src, err := os.ReadFile(filepath.Join("..", "migrate", "migrations",
		"0270_roles_owner_and_cleanup.sql"))
	if err != nil {
		t.Fatalf("الهجرة: %v", err)
	}
	s := string(src)
	for _, must := range []string{
		"NOT EXISTS (SELECT 1 FROM user_roles WHERE role_code = 'owner_super_admin')",
		"WHERE ur.role_code = 'admin' AND u.status = 'active'",
		"DELETE FROM user_roles WHERE role_code = 'ops'",
	} {
		if !strings.Contains(s, must) {
			t.Errorf("**الهجرةُ لا تحمل الشرط**: %s", must)
		}
	}
	var applied bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM schema_migrations
		                WHERE version = '0270_roles_owner_and_cleanup.sql')`).Scan(&applied); err != nil {
		t.Fatalf("سجلُّ الهجرات: %v", err)
	}
	if !applied {
		t.Error("**الهجرةُ 0270 لم تُطبَّق**")
	}
	// **والمالكُ الأعلى لا ينقصه شيءٌ يملكه مدير المنصّة.**
	var missing int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM role_capabilities a
		 WHERE a.role_code = 'admin' AND NOT EXISTS (
		   SELECT 1 FROM role_capabilities o
		    WHERE o.role_code = 'owner_super_admin' AND o.capability_code = a.capability_code)`).
		Scan(&missing); err != nil {
		t.Fatalf("مقارنة: %v", err)
	}
	if missing != 0 {
		t.Errorf("**المالكُ الأعلى ينقصه %d قدرةً يملكها المدير**", missing)
	}
}

// TestROLES_RevokePathsGuardLastRolesManage **حارسُ آخر `roles.manage` موصولٌ بالبابين.**
//
// **والحارسُ نفسُه يُقاس في `rolesguard` داخل معاملةٍ معزولة** — وهنا يُقاس أنّ
// البابين يناديانه (قاعدةُ الاختبار فيها دائماً حاملون كُثر).
func TestROLES_RevokePathsGuardLastRolesManage(t *testing.T) {
	for file, must := range map[string]string{
		filepath.Join("..", "server", "rbac_admin_handlers.go"): "rolesguard.GuardRolesManageRemains(",
		filepath.Join("..", "identity", "admin.go"):             "rolesguard.GuardRolesManageRemains(ctx, q, rolesguard.RolesManageLoss{",
	} {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if !strings.Contains(string(src), must) {
			t.Errorf("**%s لا ينادي حارسَ آخر `roles.manage`**", file)
		}
	}
}
