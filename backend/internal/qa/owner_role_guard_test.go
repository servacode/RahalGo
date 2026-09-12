package qa

// ══════════════════════════════════════════════════════════════════════
//  **دورُ المالك محميٌّ في المحرّك — لا في اللوحة** (`OWN-1`)
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس قبل الحارس (٢٠٢٦-٠٩-١٢، على التجهيز بالمسار الحقيقيّ)
//
// **أدمنٌ يملك `roles.manage` ولا يملك دورَ المالك**:
//
//	POST   /admin/users/{آخر}/roles  owner_super_admin ⇒ **200 granted**
//	POST   /admin/users/{نفسِه}/roles owner_super_admin ⇒ **200 granted**
//	DELETE /admin/users/{آخر}/roles/owner_super_admin   ⇒ **200 revoked**
//
// **فإخفاءُ الحبّةِ في نافذة الإسناد لم يكن حدّاً** — **والحدُّ هنا.**
//
// # وما يُحرَس
//
//	١ · `roles.manage` **لا تمنح** دورَ المالك
//	٢ · **ولا تنزعه**
//	٣ · **ولا ترقّي صاحبَها** إلى مالك
//	٤ · **ولا يُخلَق حسابٌ به** من باب إنشاء الحساب
//	٥ · ومالكٌ يمنح وينزع — **بتأكيدٍ وقيدِ تدقيق**
//	٦ · **وآخرُ مالكٍ لا يُنزَع** — ولا تُفرَّغ المنصّةُ من مالكها
//	٧ · **ومحاولةٌ مردودةٌ لا تكتب صفّاً ولا أثراً**
//	٨ · **والتأكيدُ باقٍ** — نداءٌ بلا إثباتٍ يُردّ بتحدٍّ
//	٩ · **وأدوارُ العمل تُسند كما كانت** — مراقبةٌ وعملياتٌ وماليّةٌ ودعم

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

const ownerRole = authz.RoleOwnerSuperAdmin

// holdsRole **أيملك هذا الحسابُ هذا الدورَ في القاعدة؟** — لا في جوابٍ.
func holdsRole(t *testing.T, hh *Harness, userID, role string) bool {
	t.Helper()
	var has bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = $2)`,
		userID, role).Scan(&has); err != nil {
		t.Fatalf("قراءةُ الأدوار: %v", err)
	}
	return has
}

// ownerCount **عددُ من يملك دورَ المالك.**
func ownerCount(t *testing.T, hh *Harness) int {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM user_roles WHERE role_code = $1`, ownerRole).Scan(&n); err != nil {
		t.Fatalf("عدُّ المالكين: %v", err)
	}
	return n
}

// grantPath · revokePath
func grantPath(uid string) string { return "/api/v1/admin/users/" + uid + "/roles" }

// **ولا استعلامَ في المسار** — **بصمةُ الإثبات تُحسَب من `r.URL.Path`**
// في الخادم ومن النصّ المُمرَّر في المِسنَد، **فـ`?reason=` يُفرّقهما**
// **فيُردّ الإثباتُ `step_up_invalid`** (قِيس، لم يُفترَض).
func revokePath(uid, role string) string {
	return "/api/v1/admin/users/" + uid + "/roles/" + role
}

// ══════════════════════════════════════════════════════════════════════
//
//	**OWN-1 · `roles.manage` وحدَها لا تبلغ دورَ المالك**
//
// ══════════════════════════════════════════════════════════════════════
func TestOWN1_RolesManageCannotGrantOwner(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_own_mgr", authz.RolesManage, authz.UsersStatusManage)
	actor, tok := capUser(t, hh, "qa_own_mgr")
	victim, _ := capUser(t, hh, "customer")

	before := auditCount(t, hh, "admin.role_grant", victim.ID)

	// ── ١ · منحُه لحسابٍ آخر ──────────────────────────────────────
	res := hh.POST(grantPath(victim.ID), tok, map[string]any{
		"role": ownerRole, "reason": "OWN-1"})
	t.Logf("OWN-1: منحُ دور المالك بـroles.manage ⇒ %d · %s", res.Code, res)
	if res.Code != http.StatusForbidden {
		t.Errorf("**`roles.manage` منحت دورَ المالك** (%d) — "+
			"**فإخفاءُ الحبّةِ في اللوحة هو الحدُّ الوحيد، وهو ليس حدّاً.**", res.Code)
	}
	if got := res.Err(); got != "owner_role_protected" {
		t.Errorf("رمزُ الردّ %q والمنتظَرُ `owner_role_protected`", got)
	}

	// ── ٧ · **ومحاولةٌ مردودةٌ لا تكتب صفّاً ولا أثراً** ───────────
	if holdsRole(t, hh, victim.ID, ownerRole) {
		t.Error("**مُنع الفعلُ ووقع تبديلُه** — صفُّ الدور مكتوب")
	}
	if n := auditCount(t, hh, "admin.role_grant", victim.ID); n != before {
		t.Errorf("**أثرُ تدقيقٍ لفعلٍ لم يقع**: %d ⇒ %d", before, n)
	}

	// ── ٣ · **ولا يرقّي نفسَه** ───────────────────────────────────
	self := hh.POST(grantPath(actor.ID), tok, map[string]any{
		"role": ownerRole, "reason": "OWN-1-self"})
	t.Logf("OWN-1: ترقيةٌ ذاتيّةٌ ⇒ %d", self.Code)
	if self.Code != http.StatusForbidden {
		t.Errorf("**أدمنٌ رقّى نفسَه مالكاً** (%d) — **وهذا ما قِيس ٢٠٢٦-٠٩-١٢**", self.Code)
	}
	if holdsRole(t, hh, actor.ID, ownerRole) {
		t.Error("**الترقيةُ الذاتيّةُ وقعت في القاعدة**")
	}

	// ── ٤ · **ولا يُخلَق حسابٌ بدور المالك** ──────────────────────
	cr := hh.POST("/api/v1/admin/users", tok, map[string]any{
		"phone": uniqPhone(), "full_name": "qa own", "roles": []string{ownerRole},
		"password": "QaOwner#2026"})
	t.Logf("OWN-1: إنشاءُ حسابٍ بدور المالك ⇒ %d · %s", cr.Code, cr)
	if cr.Code < 400 {
		t.Errorf("**حسابُ مالكٍ أُنشئ من باب إنشاء الحساب** (%d)", cr.Code)
	}
	if got := cr.Err(); got != "owner_role_protected" {
		t.Errorf("**بابُ الإنشاء يردّ بـ%q لا بحارس الحماية** — "+
			"**ورَدٌّ شكليٌّ يُخفي الحارسَ فيُحسَب غائباً**", got)
	}
	if n := ownerCount(t, hh); n != 0 {
		t.Errorf("**مالكٌ وُجد في قاعدةٍ لم يكن فيها** — %d", n)
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**OWN-2 · ومالكٌ يمنح وينزع — وآخرُ مالكٍ لا يُنزَع**
//
// ══════════════════════════════════════════════════════════════════════
func TestOWN2_OwnerGrantsAndLastOwnerHeld(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_own_mgr2", authz.RolesManage)
	owner, otok := capUser(t, hh, ownerRole, "qa_own_mgr2")
	victim, _ := capUser(t, hh, "customer")

	if n := ownerCount(t, hh); n != 1 {
		t.Fatalf("المِسنَدُ بدأ بـ%d مالكاً والمنتظَرُ واحدٌ — والقياسُ بعده لا يعني شيئاً", n)
	}

	// ── ٦ · **آخرُ مالكٍ لا يُنزَع** (ونزعٌ من النفس بالضرورة) ─────
	last := hh.Call("DELETE", revokePath(owner.ID, ownerRole), otok, nil, nil)
	t.Logf("OWN-2: نزعُ آخرِ مالكٍ ⇒ %d · %s", last.Code, last)
	if last.Code < 400 {
		t.Errorf("**نُزع آخرُ مالكٍ** (%d) — **فلا أحدَ يستطيع منحَ الدور بعدها**، "+
			"**ولا يُستعاد إلّا من الخادم.**", last.Code)
	}
	if got := last.Err(); got != "last_owner" {
		t.Errorf("رمزُ الردّ %q والمنتظَرُ `last_owner`", got)
	}
	if !holdsRole(t, hh, owner.ID, ownerRole) {
		t.Error("**مُنع النزعُ ووقع** — والمالكُ فقد دورَه")
	}

	// ── ٥ · ومالكٌ يمنح — وأثرُ تدقيقٍ يُكتب ──────────────────────
	before := auditCount(t, hh, "admin.role_grant", victim.ID)
	gr := hh.POST(grantPath(victim.ID), otok, map[string]any{
		"role": ownerRole, "reason": "OWN-2-grant"})
	t.Logf("OWN-2: مالكٌ يمنح ⇒ %d", gr.Code)
	if gr.Code >= 400 {
		t.Fatalf("**المالكُ لم يستطع منحَ دوره** (%d) — %s", gr.Code, gr)
	}
	if !holdsRole(t, hh, victim.ID, ownerRole) {
		t.Error("**نجح النداءُ ولم يقع التبديل**")
	}
	if n := auditCount(t, hh, "admin.role_grant", victim.ID); n <= before {
		t.Error("**فعلٌ حسّاسٌ وقع بلا أثرِ تدقيق** — `AQ-4`")
	}

	// ── وصار المالكونَ اثنين، فالنزعُ يجوز ────────────────────────
	if n := ownerCount(t, hh); n != 2 {
		t.Fatalf("المالكونَ %d والمنتظَرُ اثنان", n)
	}
	rv := hh.Call("DELETE", revokePath(victim.ID, ownerRole), otok, nil, nil)
	t.Logf("OWN-2: مالكٌ ينزع والثاني قائم ⇒ %d", rv.Code)
	if rv.Code >= 400 {
		t.Errorf("**المالكُ لم يستطع النزعَ وهناك مالكٌ ثانٍ** (%d) — %s", rv.Code, rv)
	}
	if holdsRole(t, hh, victim.ID, ownerRole) {
		t.Error("**نجح النزعُ ولم يقع**")
	}
	if n := auditCount(t, hh, "admin.role_revoke", victim.ID); n == 0 {
		t.Error("**نزعٌ بلا أثرِ تدقيق**")
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**OWN-3 · والتأكيدُ باقٍ · وأدوارُ العمل تُسند كما كانت**
//
// ══════════════════════════════════════════════════════════════════════
func TestOWN3_StepUpStillRequiredAndStaffRolesWork(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_own_mgr3", authz.RolesManage)
	_, tok := capUser(t, hh, "qa_own_mgr3")
	victim, _ := capUser(t, hh, "customer")

	// ── ٨ · **نداءٌ بلا إثباتٍ يُردّ بتحدٍّ لا بحارس الحماية** ─────
	//
	// **والترتيبُ مقصودٌ ومقيس**: التأكيدُ في الوسيط والحمايةُ في
	// الخدمة — **فمن نادى بلا إثباتٍ رأى التحدّي أوّلاً.**
	code, raw := stepUpChallenge(t, hh, tok, "POST", grantPath(victim.ID),
		map[string]any{"role": "observability", "reason": "OWN-3"})
	assertChallengeShape(t, "OWN-3", code, raw, "admin.role_grant")
	if holdsRole(t, hh, victim.ID, "observability") {
		t.Error("**دورٌ مُنح بلا تأكيد**")
	}

	// ── ٩ · وأدوارُ العمل تُسند كما كانت ─────────────────────────
	//
	// **وهذا شاهدٌ موجبٌ للحارس**: **حارسٌ يمنع كلَّ شيءٍ ينجح إلى
	// الأبد** — فيُقاس أنّه يمنع المحميَّ وحدَه.
	for _, role := range []string{"observability", "operations", "finance", "customer_support"} {
		if _, err := hh.Pool.Exec(ctxBG(),
			`INSERT INTO roles (code, name_key) VALUES ($1, $1) ON CONFLICT DO NOTHING`,
			role); err != nil {
			t.Fatalf("تهيئةُ الدور %s: %v", role, err)
		}
		res := hh.POST(grantPath(victim.ID), tok, map[string]any{
			"role": role, "reason": "OWN-3"})
		t.Logf("OWN-3: منحُ %-18s ⇒ %d", role, res.Code)
		if res.Code >= 400 {
			t.Errorf("**دورُ عملٍ لم يُسند بعد الحارس**: %s (%d) — %s",
				role, res.Code, res)
		}
		if !holdsRole(t, hh, victim.ID, role) {
			t.Errorf("**نجح منحُ %s ولم يقع**", role)
		}
		if _, err := hh.Pool.Exec(ctxBG(),
			`DELETE FROM user_roles WHERE user_id = $1::uuid AND role_code = $2`,
			victim.ID, role); err != nil {
			t.Fatalf("تنظيفُ %s: %v", role, err)
		}
	}
}
