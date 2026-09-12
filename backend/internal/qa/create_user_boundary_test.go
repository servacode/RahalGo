package qa

// ══════════════════════════════════════════════════════════════════════
//  **بابُ إنشاء الحساب لا يتجاوز سياسةَ المنح** (`CUB-1`)
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس قبل الحدّ (٢٠٢٦-٠٩-١٢، بالمسار الحقيقيّ على التجهيز)
//
// **سلسلةٌ كاملةٌ لا تحتاج `roles.manage` إطلاقاً**:
//
//	trust_safety:  GET /admin/roles ⇒ 403  ·  POST .../roles ⇒ 403
//	ثمّ POST /admin/users {roles:["admin"], password: يختارها} ⇒ **201**
//	     **وبلا تأكيدٍ** — فالمسلكُ ليس فعلاً حسّاساً
//	ثمّ يدخل بالحساب المُنشَأ ⇒ GET /admin/roles ⇒ **200**
//
// **فقدرةُ `users.status.manage` كانت تُنتج `roles.manage` بخطوتين.**
//
// # وما يُحرَس
//
//	١ · `trust_safety` لا يُنشئ أدمناً
//	٢ · و`finance` كذلك
//	٣ · و`operations` كذلك
//	٤ · **ولا أدمنٌ يملك `roles.manage`** — المرتفعُ للمالك وحدَه
//	٥ · ومالكٌ يمنح `admin` بمساره — **ولا يُخلَق به حسابٌ مباشرةً**
//	٦ · والتأكيدُ باقٍ على منحِ المالك
//	٧ · ومنحُ `admin` الناجحُ يُكتب في التدقيق
//	٨ · **ومحاولةٌ مردودةٌ لا تُنشئ حساباً ولا تكتب أثراً**
//	٩ · وصفةُ الحساب تُنشأ كما كانت — زبونٌ وسائقٌ ومندوب
//	١٠ · وأدوارُ العمل تُسند بمسار المنح — مراقبةٌ وعملياتٌ وماليّةٌ ودعم
//	١١ · **ودورٌ مجهولٌ لا يُخلَق به حسابٌ** ولا يكسب سلطةً مرتفعة
//	١٢ · **و`ops` إرثٌ لا يُمنَح جديداً** — ومن يحمله يُنزَع منه

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// createUser **بابُ إنشاء الحساب** — بالمسار الحقيقيّ.
func createUser(hh *Harness, tok, phone, role string) Res {
	return hh.POST("/api/v1/admin/users", tok, map[string]any{
		"phone": phone, "full_name": "cub " + role,
		"roles": []string{role}, "password": "CubProbe#2026",
	})
}

// userExists **أوُجد حسابٌ بهذا الرقم؟** — في القاعدة لا في جوابٍ.
func userExists(t *testing.T, hh *Harness, phone string) bool {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM users WHERE phone = $1`, phone).Scan(&n); err != nil {
		t.Fatalf("قراءةُ الحسابات: %v", err)
	}
	return n > 0
}

// ══════════════════════════════════════════════════════════════════════
//
//	**CUB-1 · موظّفٌ لا يُنشئ أدمناً — ولا أدمنٌ يُنشئ أدمناً**
//
// ══════════════════════════════════════════════════════════════════════
func TestCUB1_StaffCannotCreateAdmin(t *testing.T) {
	hh := New(t)

	// **وأدوارُ العمل من المصفوفة الكانونيّة نفسِها** — لا مصطنعة:
	// **قدراتُها ما بذرته هجرةُ `0138`.**
	//
	// **وطبقتا المنعِ مختلفتان، وكلتاهما منعٌ** (قِيس، لم يُفترَض):
	//
	//	trust_safety  **يملك `users.status.manage`** فيصل البابَ
	//	              ⇒ يردُّه حارسُ السلطة: `owner_role_protected`
	//	finance · operations  **لا يملكانها أصلاً**
	//	              ⇒ يردُّهما وسيطُ القدرة قبل الباب: `forbidden`
	//
	// **والثاني منعٌ أقوى** — **لا يبلغ المسلكَ إطلاقاً.**
	for _, c := range []struct {
		role string
		want string
	}{
		{"trust_safety", "owner_role_protected"},
		{"finance", "forbidden"},
		{"operations", "forbidden"},
	} {
		_, tok := capUser(t, hh, c.role)
		phone := uniqPhone()
		res := createUser(hh, tok, phone, authz.RoleAdmin)
		t.Logf("CUB-1: %-14s يُنشئ أدمناً ⇒ %d · %s", c.role, res.Code, res)
		if res.Code != http.StatusForbidden {
			t.Errorf("**%s أنشأ حسابَ أدمنٍ** (%d) — "+
				"**فقدرةُ `users.status.manage` تُنتج `roles.manage` بخطوتين**", c.role, res.Code)
		}
		if got := res.Err(); got != c.want {
			t.Errorf("%s: رمزُ الردّ %q والمنتظَرُ %q — "+
				"**وتبدُّلُ طبقةِ المنع يُقرأ هنا** (وسيطُ القدرة أم حارسُ السلطة)",
				c.role, got, c.want)
		}
		// ── ٨ · **ومحاولةٌ مردودةٌ لا تُنشئ حساباً** ───────────────
		if userExists(t, hh, phone) {
			t.Errorf("**%s: مُنع الفعلُ ووقع** — حسابٌ أُنشئ بالرقم %s", c.role, phone)
		}
	}

	// ── ٤ · **ولا أدمنٌ يملك `roles.manage`** ─────────────────────
	capRole(t, hh, "qa_cub_mgr", authz.RolesManage, authz.UsersStatusManage)
	_, mtok := capUser(t, hh, "qa_cub_mgr")
	phone := uniqPhone()
	res := createUser(hh, mtok, phone, authz.RoleAdmin)
	t.Logf("CUB-1: roles.manage يُنشئ أدمناً ⇒ %d · %s", res.Code, res)
	if res.Code != http.StatusForbidden || res.Err() != "owner_role_protected" {
		t.Errorf("**`roles.manage` أنشأت أدمناً** (%d / %q) — **والمرتفعُ للمالك وحدَه**",
			res.Code, res.Err())
	}
	if userExists(t, hh, phone) {
		t.Error("**مُنع الفعلُ ووقع** — حسابٌ أُنشئ")
	}

	// ── ٤ب · **ولا يمنحه بمسار المنح كذلك** ──────────────────────
	victim, _ := capUser(t, hh, "customer")
	before := auditCount(t, hh, "admin.role_grant", victim.ID)
	gr := hh.POST(grantPath(victim.ID), mtok, map[string]any{
		"role": authz.RoleAdmin, "reason": "CUB-1"})
	t.Logf("CUB-1: roles.manage تمنح admin ⇒ %d · %s", gr.Code, gr)
	if gr.Code != http.StatusForbidden || gr.Err() != "owner_role_protected" {
		t.Errorf("**`roles.manage` منحت الدورَ المرتفع** (%d / %q)", gr.Code, gr.Err())
	}
	if holdsRole(t, hh, victim.ID, authz.RoleAdmin) {
		t.Error("**مُنع المنحُ ووقع**")
	}
	if n := auditCount(t, hh, "admin.role_grant", victim.ID); n != before {
		t.Errorf("**أثرُ تدقيقٍ لفعلٍ لم يقع**: %d ⇒ %d", before, n)
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**CUB-2 · ومالكٌ يمنح المرتفعَ بمساره — لا من باب الإنشاء**
//
// ══════════════════════════════════════════════════════════════════════
func TestCUB2_OwnerGrantsElevatedThroughGrantPath(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_cub_own", authz.RolesManage, authz.UsersStatusManage)
	// **ومالكٌ معزولٌ يُمحى بعدَه** — **فاختبارٌ يترك مالكاً يكسر
	// قياسَ آخرَ لعددهم.** (وقع.)
	_, otok := soleOwner(t, hh, "qa_cub_own")
	victim, _ := capUser(t, hh, "customer")

	// ── ٥ · **ولا يُخلَق حسابُ أدمنٍ مباشرةً ولو كان الفاعلُ مالكاً**
	//
	// **وجوابُ التدفّق لا جوابُ الأمن**: `role_not_creatable`.
	phone := uniqPhone()
	cr := createUser(hh, otok, phone, authz.RoleAdmin)
	t.Logf("CUB-2: مالكٌ يُنشئ أدمناً مباشرةً ⇒ %d · %s", cr.Code, cr)
	if cr.Code != http.StatusForbidden {
		t.Errorf("**حسابُ أدمنٍ أُنشئ من بابِ الإنشاء** (%d) — "+
			"**والتخويلُ خطوتان بقصد**", cr.Code)
	}
	if got := cr.Err(); got != "role_not_creatable" {
		t.Errorf("رمزُ الردّ %q والمنتظَرُ `role_not_creatable` — "+
			"**والمالكُ يملك الصلاحيةَ، فرسالةُ «لا تملك» تكذب عليه**", got)
	}
	if userExists(t, hh, phone) {
		t.Error("**مُنع الإنشاءُ ووقع**")
	}

	// ── ٦ · **والتأكيدُ باقٍ على منحِ المالك** ────────────────────
	code, raw := stepUpChallenge(t, hh, otok, "POST", grantPath(victim.ID),
		map[string]any{"role": authz.RoleAdmin, "reason": "CUB-2"})
	assertChallengeShape(t, "CUB-2", code, raw, "admin.role_grant")
	if holdsRole(t, hh, victim.ID, authz.RoleAdmin) {
		t.Error("**دورٌ مرتفعٌ مُنح بلا تأكيد**")
	}

	// ── ٥ب + ٧ · ومالكٌ يمنحه بمساره — وأثرُ تدقيقٍ يُكتب ────────
	before := auditCount(t, hh, "admin.role_grant", victim.ID)
	gr := hh.POST(grantPath(victim.ID), otok, map[string]any{
		"role": authz.RoleAdmin, "reason": "CUB-2-grant"})
	t.Logf("CUB-2: مالكٌ يمنح admin ⇒ %d", gr.Code)
	if gr.Code >= 400 {
		t.Fatalf("**المالكُ لم يستطع منحَ الدور المرتفع** (%d) — %s", gr.Code, gr)
	}
	if !holdsRole(t, hh, victim.ID, authz.RoleAdmin) {
		t.Error("**نجح النداءُ ولم يقع التبديل**")
	}
	if n := auditCount(t, hh, "admin.role_grant", victim.ID); n <= before {
		t.Error("**منحُ دورٍ مرتفعٍ بلا أثرِ تدقيق** — `AQ-4`")
	}

	// ── ونزعُه خفضٌ لا رفع — فيكفيه `roles.manage` ───────────────
	//
	// **ولو لزمه مالكٌ ولا مالكَ في الإنتاج لَصار كلُّ أدمنٍ دائماً.**
	capRole(t, hh, "qa_cub_mgr2", authz.RolesManage)
	_, mtok := capUser(t, hh, "qa_cub_mgr2")
	rv := hh.Call("DELETE", revokePath(victim.ID, authz.RoleAdmin), mtok, nil, nil)
	t.Logf("CUB-2: roles.manage تنزع admin ⇒ %d · %s", rv.Code, rv)
	if rv.Code >= 400 {
		t.Errorf("**نزعُ الدور المرتفع تعذّر** (%d) — **فلا يُسحب أدمنٌ أبداً**", rv.Code)
	}
	if holdsRole(t, hh, victim.ID, authz.RoleAdmin) {
		t.Error("**نجح النزعُ ولم يقع**")
	}
}

// ══════════════════════════════════════════════════════════════════════
//
//	**CUB-3 · وصفةُ الحساب تُنشأ · وتخويلُ العمل بخطوتين**
//
// ══════════════════════════════════════════════════════════════════════
func TestCUB3_AccountTypesCreateAndStaffGrantFlow(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_cub_mgr3", authz.RolesManage, authz.UsersStatusManage)
	_, tok := capUser(t, hh, "qa_cub_mgr3")

	// ── ٩ · صفةُ الحساب تُنشأ كما كانت ───────────────────────────
	//
	// **و`merchant` تُستثنى بقرارٍ قائم** (`ErrMerchantNeedsStore`):
	// **دورُ المتجر يُمنح مع إنشاء متجره**، **وهذا سلوكٌ مقصودٌ لا
	// أثرٌ لهذه الدورة.**
	for _, role := range []string{"customer", "driver", "sales"} {
		phone := uniqPhone()
		res := createUser(hh, tok, phone, role)
		t.Logf("CUB-3: إنشاءُ حسابِ %-9s ⇒ %d", role, res.Code)
		if res.Code != http.StatusCreated {
			t.Errorf("**صفةُ حسابٍ لم تُنشأ بعد الحدّ**: %s (%d) — %s", role, res.Code, res)
		}
		if !userExists(t, hh, phone) {
			t.Errorf("**نجح الإنشاءُ ولم يقع**: %s", role)
		}
	}
	// **و`merchant` تبقى ممنوعةً بسببها القديم لا بحدِّنا** — يُقاس.
	mres := createUser(hh, tok, uniqPhone(), "merchant")
	t.Logf("CUB-3: إنشاءُ حسابِ merchant ⇒ %d · %s", mres.Code, mres)
	if got := mres.Err(); got != "merchant_needs_store" {
		t.Errorf("**دورُ المتجر مُنع بسببٍ غيرِ سببه**: %q — "+
			"**وحدُّنا يجب ألّا يُغيّر رسالةَ قرارٍ قائم**", got)
	}

	// ── ١٠ · وتخويلُ العمل: حسابٌ يُخلَق ثمّ دورٌ يُمنَح ─────────
	for _, role := range []string{"observability", "operations", "finance", "customer_support"} {
		// **والدورُ يُهيَّأ إن لم تبذره الهجرةُ** — `observability`
		// أنشأه المالكُ من اللوحة لا هجرة.
		if _, err := hh.Pool.Exec(ctxBG(),
			`INSERT INTO roles (code, name_key) VALUES ($1, $1) ON CONFLICT DO NOTHING`,
			role); err != nil {
			t.Fatalf("تهيئةُ الدور %s: %v", role, err)
		}
		// **الخطوةُ الأولى** — حسابٌ بصفةِ زبون.
		phone := uniqPhone()
		cr := createUser(hh, tok, phone, "customer")
		if cr.Code != http.StatusCreated {
			t.Fatalf("**الخطوةُ الأولى أخفقت** لـ%s: %s", role, cr)
		}
		uid, _ := cr.JSON()["id"].(string)
		// **والخطوةُ الثانية بمسارها المخوَّل** — `roles.manage` وتأكيد.
		gr := hh.POST(grantPath(uid), tok, map[string]any{
			"role": role, "reason": "CUB-3"})
		t.Logf("CUB-3: حسابٌ ثمّ منحُ %-18s ⇒ %d/%d", role, cr.Code, gr.Code)
		if gr.Code >= 400 {
			t.Errorf("**دورُ عملٍ لم يُسند بمساره**: %s (%d) — %s", role, gr.Code, gr)
		}
		if !holdsRole(t, hh, uid, role) {
			t.Errorf("**نجح منحُ %s ولم يقع**", role)
		}
		// ── ١٧ · **وفشلُ الخطوة الثانية يترك الحسابَ قائماً** ─────
		//
		// **وهذه هي الحالةُ التي تُعرَض على المالك صريحةً**: **حسابٌ
		// أُنشئ ودورٌ لم يُمنَح** — **ولا يُقال «تمّ» عن نصفِ عمل.**
		if !userExists(t, hh, phone) {
			t.Error("**الحسابُ اختفى بعد المنح**")
		}
	}

	// ── ١١ · **ودورٌ مجهولٌ لا يُخلَق به حسابٌ** ─────────────────
	if _, err := hh.Pool.Exec(ctxBG(),
		`INSERT INTO roles (code, name_key) VALUES ('zz_night_crew', 'فريق الليل')
		 ON CONFLICT DO NOTHING`); err != nil {
		t.Fatalf("تهيئةُ دورٍ مجهول: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = 'zz_night_crew'`)
	})
	uphone := uniqPhone()
	ures := createUser(hh, tok, uphone, "zz_night_crew")
	t.Logf("CUB-3: إنشاءُ حسابٍ بدورٍ مجهول ⇒ %d · %s", ures.Code, ures)
	if ures.Code < 400 {
		t.Error("**حسابٌ أُنشئ بدورٍ مجهول** — **وبابُ الإنشاء لصفةِ الحساب وحدَها**")
	}
	if userExists(t, hh, uphone) {
		t.Error("**مُنع الإنشاءُ ووقع**")
	}
	// **ويُسند بمسار المنح** — فدورٌ أنشأه المالكُ يعمل.
	vic, _ := capUser(t, hh, "customer")
	ugr := hh.POST(grantPath(vic.ID), tok, map[string]any{
		"role": "zz_night_crew", "reason": "CUB-3"})
	t.Logf("CUB-3: منحُ دورٍ مجهولٍ بمساره ⇒ %d", ugr.Code)
	if ugr.Code >= 400 {
		t.Errorf("**دورٌ أنشأه المالكُ من اللوحة لا يُسند** (%d) — %s", ugr.Code, ugr)
	}

	// ── ١٢ · **و`ops` إرثٌ لا يُمنَح جديداً** (`OPS-5`) ───────────
	ops := hh.POST(grantPath(vic.ID), tok, map[string]any{
		"role": "ops", "reason": "CUB-3"})
	t.Logf("CUB-3: منحُ ops جديداً ⇒ %d · %s", ops.Code, ops)
	if ops.Code != http.StatusForbidden || ops.Err() != "role_grant_retired" {
		t.Errorf("**`ops` يُمنَح جديداً** (%d / %q) — "+
			"**واسمُه العربيُّ «العمليات» كاسم `operations` وقدراتُهما مختلفة**",
			ops.Code, ops.Err())
	}
	if holdsRole(t, hh, vic.ID, "ops") {
		t.Error("**مُنع منحُ `ops` ووقع**")
	}
	// **ومن يحمله يُنزَع منه** — فالإرثُ يُفرَّغ تدريجاً لا يُجمَّد.
	if _, err := hh.Pool.Exec(ctxBG(), `
		INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'ops')
		ON CONFLICT DO NOTHING`, vic.ID); err != nil {
		t.Fatalf("تهيئةُ حاملِ `ops`: %v", err)
	}
	orv := hh.Call("DELETE", revokePath(vic.ID, "ops"), tok, nil, nil)
	t.Logf("CUB-3: نزعُ ops من حامله ⇒ %d", orv.Code)
	if orv.Code >= 400 {
		t.Errorf("**`ops` لا يُنزَع من حامله** (%d) — **فالإرثُ يُجمَّد ولا يُفرَّغ**", orv.Code)
	}
}
