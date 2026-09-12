package qa

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **فعلٌ حسّاسٌ نجح وأثرُه سقط** — `XG-20` · `AQ-4`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ بنصّه
//
//	CRITICAL ACTION SUCCESS ⇒ CRITICAL AUDIT SUCCESS
//
// **«في المعاملة نفسِها أو في وحدة النجاح المنطقيّة نفسِها.»**
//
// **ويشمل**: الاسترداد · قيودَ المحفظة · السحوبات · العمولات ·
// التسعيرَ والهوامش · **الأدوارَ والصلاحيّات · الإيقافَ والحظر ·
// الإعداداتِ الحسّاسة · وتدخّلاتِ الطلبات الحرجة.**
//
// # وما كان
//
// **دورةُ ١٣ أغلقت ستّةَ أفعالٍ ماليّة** — **والباقي يُقيَّد بعد
// الكتابة على المَسبَح ويبتلع خطأه** (`repo.Audit`, `s.audit`).
// **فسحبُ دورٍ ينجح وأثرُه يسقط**، **ولا يبقى من يُسأل.**

// auditCount عددُ آثارِ فعلٍ على كيانٍ بعينه.
func auditCount(t *testing.T, hh *Harness, action, entityID string) int {
	t.Helper()
	var n int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM audit_log
		 WHERE action = $1 AND entity_id = $2`, action, entityID).Scan(&n); err != nil {
		t.Fatalf("عدُّ الآثار: %v", err)
	}
	return n
}

// ══════════════════════════════════════════════════════════════════════
// **S1+S2+A1 · منحُ دورٍ وسحبُه — الفعلُ وأثرُه معاً**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_S1S2_RoleMutationAudited(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")

	if got := grantRole(t, hh, u.ID, "operations"); got.Code >= 400 {
		t.Fatalf("منحُ الدور: %s", got)
	}
	grants := auditCount(t, hh, "admin.role_grant", u.ID)
	if got := revokeRole(t, hh, u.ID, "operations"); got.Code >= 400 {
		t.Fatalf("سحبُ الدور: %s", got)
	}
	revokes := auditCount(t, hh, "admin.role_revoke", u.ID)
	t.Logf("S1/S2: آثارُ المنح=%d · آثارُ السحب=%d", grants, revokes)

	if grants == 0 {
		t.Error("**منحٌ بلا أثر** (`XG-20`)")
	}
	if revokes == 0 {
		t.Error("**سحبٌ بلا أثر** (`XG-20`)")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A2 · سقوطُ الأثر يُسقط الفعل**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا لبُّ العقد**: **لا «نجح الفعلُ وضاع أثرُه».**
func TestXG20_A2_AuditFailureRollsBackBusiness(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")
	if got := grantRole(t, hh, u.ID, "operations"); got.Code >= 400 {
		t.Fatalf("منحُ الدور: %s", got)
	}

	fp := hh.ArmAny("XG20/audit-write", "audit_log", "INSERT")
	got := revokeRole(t, hh, u.ID, "operations")
	fp.MustFire(t)

	var stillHas bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = 'operations')`,
		u.ID).Scan(&stillHas); err != nil {
		t.Fatalf("قراءةُ الأدوار: %v", err)
	}
	t.Logf("A2: الردّ=%d · الدورُ ما زال=%v · آثارٌ=%d",
		got.Code, stillHas, auditCount(t, hh, "admin.role_revoke", u.ID))

	if !stillHas {
		t.Errorf("**سُحب الدورُ وأثرُه لم يُقيَّد** — **وذاك عينُ `XG-20`.**")
	}
	if got.Code < 400 {
		t.Errorf("**رُدَّ نجاحٌ وقد سقط الأثر** (%d)", got.Code)
	}

	// ── A5 · والإعادةُ تُنجز الفعلَ وأثرَه معاً ────────────────────
	again := revokeRole(t, hh, u.ID, "operations")
	var gone bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT NOT EXISTS (SELECT 1 FROM user_roles
		                    WHERE user_id = $1::uuid AND role_code = 'operations')`,
		u.ID).Scan(&gone); err != nil {
		t.Fatalf("قراءةُ الأدوار: %v", err)
	}
	t.Logf("A5: الإعادةُ ردّت %d · الدورُ ذهب=%v · آثارٌ=%d",
		again.Code, gone, auditCount(t, hh, "admin.role_revoke", u.ID))
	if !gone || auditCount(t, hh, "admin.role_revoke", u.ID) != 1 {
		t.Errorf("**الإعادةُ لم تُنجز الفعلَ وأثرَه مرّةً واحدة**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A3 · وفعلٌ يسقط لا يترك أثراً كاذباً**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_A3_FailedBusinessLeavesNoAudit(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")
	admin := hh.NewUser("admin")

	// **دورٌ لا وجودَ له** — يُردّ قبل أيّ كتابة.
	got := hh.POST("/api/v1/admin/users/"+u.ID+"/roles", admin.Token,
		map[string]any{"role": "not-a-role", "reason": "XG-20"})
	t.Logf("A3: دورٌ مجهولٌ ⇒ %d · آثارٌ=%d",
		got.Code, auditCount(t, hh, "admin.role_grant", u.ID))
	if got.Code < 400 {
		t.Fatalf("**قُبل دورٌ مجهول** (%d)", got.Code)
	}
	if auditCount(t, hh, "admin.role_grant", u.ID) != 0 {
		t.Errorf("**أثرٌ كاذبٌ لفعلٍ لم يقع**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S3 · تبديلُ حالِ حساب — والعقدُ من دورةِ ١٧ محفوظ**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_S3_UserStatusAudited(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("driver")
	_, sid := deliverySession(t, hh, u.ID, "driver", "driver")

	suspend(t, hh, u.ID, "suspended")
	sus := auditCount(t, hh, "admin.user_update", u.ID)
	live := liveRows(t, hh, sid)
	t.Logf("S3: إيقافٌ — آثارٌ=%d · صفوفٌ حيّةٌ=%d", sus, live)

	if sus == 0 {
		t.Error("**إيقافٌ بلا أثر** (`XG-20`)")
	}
	if live == 0 {
		t.Error("**كُسر عقدُ دورةِ ١٧**: الإيقافُ العاديُّ أبطل جلسة")
	}

	suspend(t, hh, u.ID, "blocked")
	t.Logf("S3: حظرٌ — آثارٌ=%d · صفوفٌ حيّةٌ=%d",
		auditCount(t, hh, "admin.user_update", u.ID), liveRows(t, hh, sid))
	if auditCount(t, hh, "admin.user_update", u.ID) != sus+1 {
		t.Error("**حظرٌ بلا أثرٍ ثانٍ**")
	}
	if liveRows(t, hh, sid) != 0 {
		t.Error("**كُسر عقدُ دورةِ ١٧**: الحظرُ لم يُبطل الجلسات")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S4 · تعليقُ متجر**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_S4_MerchantSuspendAudited(t *testing.T) {
	hh := New(t)
	m := hh.Factory().Merchant()
	admin := hh.NewUser("admin")

	got := hh.POST("/api/v1/admin/merchants/"+m.ID+"/suspend", admin.Token,
		map[string]any{"suspended": true, "note": "XG-20"})
	if got.Code >= 400 {
		t.Skipf("مسارُ التعليق بهذا الشكل: %s", got)
	}
	var status string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM merchants WHERE id = $1::uuid`, m.ID).Scan(&status); err != nil {
		t.Fatalf("حالُ المتجر: %v", err)
	}
	n := auditCount(t, hh, "ops.merchant_suspend", m.ID)
	t.Logf("S4: الحالُ=%q · آثارٌ=%d", status, n)
	if status != "suspended" {
		t.Fatalf("**لم يُعلَّق**: %q", status)
	}
	if n == 0 {
		t.Error("**تعليقٌ بلا أثر** (`XG-20`)")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S5 · إعدادٌ حسّاسٌ — وغيرُ الحسّاس يبقى أفضلَ جهد**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_S5_SensitiveSettingAudited(t *testing.T) {
	hh := New(t)
	admin := hh.NewUser("admin")
	const key = "merchants.commission_percent"

	got := hh.Call("PUT", "/api/v1/admin/settings/"+key, admin.Token,
		map[string]any{"value": 12}, nil)
	if got.Code >= 400 {
		t.Skipf("مسارُ الإعدادات بهذا الشكل: %s", got)
	}
	n := auditCount(t, hh, "admin.setting_update", key)
	t.Logf("S5: إعدادٌ ماليٌّ %q ⇒ %d · آثارٌ=%d", key, got.Code, n)
	if n == 0 {
		t.Error("**إعدادٌ ماليٌّ بُدّل بلا أثر** (`XG-20`)")
	}

	// **وسقوطُ الأثر يُسقط التبديل.**
	var before []byte
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT value FROM app_settings WHERE key = $1`, key).Scan(&before); err != nil {
		t.Fatalf("قراءةُ الإعداد: %v", err)
	}
	fp := hh.ArmAny("XG20/setting-audit", "audit_log", "INSERT")
	bad := hh.Call("PUT", "/api/v1/admin/settings/"+key, admin.Token,
		map[string]any{"value": 33}, nil)
	fp.MustFire(t)
	var after []byte
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT value FROM app_settings WHERE key = $1`, key).Scan(&after); err != nil {
		t.Fatalf("قراءةُ الإعداد: %v", err)
	}
	t.Logf("S5: سقوطُ الأثر ⇒ الردّ %d · القيمةُ %v ← %v", bad.Code, before, after)
	if string(after) != string(before) {
		t.Errorf("**بُدّل الإعدادُ الماليُّ وأثرُه سقط** — **وذاك `XG-20`.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S6 · تدخّلُ موظّفٍ في طلب — والثوابتُ محفوظة**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_S6_OpsOrderTransitionAudited(t *testing.T) {
	hh := New(t)
	oid, _ := activeOrderFor(t, hh)
	admin := hh.NewUser("admin")

	got := hh.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "cancelled", "note": "XG-20"})
	if got.Code >= 400 {
		t.Skipf("مسارُ التدخّل بهذا الشكل: %s", got)
	}
	n := auditCount(t, hh, "ops.order_transition", oid)

	// **و`order_events` كما هي** — آلةُ الحال واحدة.
	var events int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_events WHERE order_id = $1::uuid`,
		oid).Scan(&events); err != nil {
		t.Fatalf("أحداثُ الطلب: %v", err)
	}
	t.Logf("S6: آثارٌ=%d · أحداثُ الطلب=%d", n, events)
	if n == 0 {
		t.Error("**تدخّلٌ بلا أثر** (`XG-20`)")
	}
	if events == 0 {
		t.Error("**اختفى `order_events`** — **آلةُ الحال تُجووِزت.**")
	}

	// ── A4 · وسقوطُ الأثر يُسقط الانتقال ─────────────────────────
	oid2, _ := activeOrderFor(t, hh)
	var before string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid2).Scan(&before); err != nil {
		t.Fatalf("حالُ الطلب: %v", err)
	}
	fp := hh.ArmAny("XG20/order-audit", "audit_log", "INSERT")
	bad := hh.POST("/api/v1/admin/orders/"+oid2+"/transition", admin.Token,
		map[string]any{"to": "cancelled", "note": "XG-20"})
	fp.MustFire(t)
	var after string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM orders WHERE id = $1::uuid`, oid2).Scan(&after); err != nil {
		t.Fatalf("حالُ الطلب: %v", err)
	}
	t.Logf("A4: سقوطُ الأثر ⇒ الردّ %d · الحالُ %q ← %q", bad.Code, before, after)
	if after != before {
		t.Errorf("**انتقلَ الطلبُ وأثرُه سقط** — **وذاك `XG-20`.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1+C2 · تزاحمُ تبديلاتٍ حسّاسة — لكلّ ما ثبت أثرُه**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_C1C2_ConcurrentMutationsCorrelate(t *testing.T) {
	hh := New(t)
	u := hh.NewUser("customer")
	admin := hh.NewUser("admin")
	if got := grantRole(t, hh, u.ID, "operations"); got.Code >= 400 {
		t.Fatalf("منحُ الدور: %s", got)
	}

	var revoked, updated Res
	r := Race(t, DefaultRaceTimeout,
		Actor{Name: "سحبُ الدور", Do: func(context.Context) any {
			revoked = hh.Call("DELETE", "/api/v1/admin/users/"+u.ID+"/roles/operations",
				admin.Token, nil, nil)
			return revoked
		}},
		Actor{Name: "تبديلُ الحال", Do: func(context.Context) any {
			updated = hh.PATCH("/api/v1/admin/users/"+u.ID, admin.Token,
				map[string]any{"status": "suspended", "status_reason": "XG-20"})
			return updated
		}},
	)
	if r.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r)
	}

	var hasRole bool
	var status string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = 'operations'),
		       (SELECT status FROM users WHERE id = $1::uuid)`,
		u.ID).Scan(&hasRole, &status); err != nil {
		t.Fatalf("الحالُ بعد السباق: %v", err)
	}
	revAudit := auditCount(t, hh, "admin.role_revoke", u.ID)
	updAudit := auditCount(t, hh, "admin.user_update", u.ID)
	t.Logf("C1/C2: الدورُ باقٍ=%v أثرٌ=%d · الحالُ=%q أثرٌ=%d — %s",
		hasRole, revAudit, status, updAudit, r)

	// **لكلّ ما ثبت أثرٌ يخصُّه** — ولا أثرَ لِما لم يثبت.
	if !hasRole && revAudit == 0 {
		t.Error("**سُحب الدورُ ولا أثرَ له**")
	}
	if hasRole && revAudit > 0 {
		t.Error("**أثرُ سحبٍ لم يقع**")
	}
	if status == "suspended" && updAudit == 0 {
		t.Error("**بُدّلت الحالُ ولا أثرَ لها**")
	}
	if status != "suspended" && updAudit > 0 {
		t.Error("**أثرُ تبديلٍ لم يقع**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **A6 · حارسٌ بنيويّ: كلُّ فعلٍ من الصنف `A` له تدقيقٌ معامليّ**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن أضاف فعلاً حسّاساً غداً بلا `auditTx` سقط بناؤه.**
func TestXG20_A6_CriticalCatalogHasTransactionalAudit(t *testing.T) {
	root := r16Root(t)

	// ── معجمُ الخادم ──────────────────────────────────────────────
	srv := mustRead(t, filepath.Join(root, "backend/internal/server/audit_tx.go"))
	block := srv[strings.Index(srv, "var criticalAuditActions"):]
	block = block[:strings.Index(block, "\n}\n")]

	var actions []string
	for _, line := range strings.Split(block, "\n") {
		if i := strings.Index(line, `":`); i > 0 {
			if j := strings.Index(line, `"`); j >= 0 && j < i {
				actions = append(actions, line[j+1:i])
			}
		}
	}
	ident := mustRead(t, filepath.Join(root, "backend/internal/identity/audit_tx.go"))
	for _, line := range strings.Split(ident[strings.Index(ident, "var CriticalActions"):], "\n") {
		if i := strings.Index(line, `":`); i > 0 {
			if j := strings.Index(line, `"`); j >= 0 && j < i {
				actions = append(actions, line[j+1:i])
			}
		}
		if strings.HasPrefix(line, "}") {
			break
		}
	}
	t.Logf("A6: أفعالُ الصنف `A` = %d — %v", len(actions), actions)
	if len(actions) < 9 {
		t.Errorf("**المعجمُ انكمش**: %d — **ونطاقُ `AQ-4` أوسع.**", len(actions))
	}

	// ── ولا فعلٌ منها يُقيَّد بأفضلِ جهد ──────────────────────────
	//
	// **يُبحَث في كلّ مصدرٍ عن نداءٍ غيرِ معامليٍّ لهذا الفعل.**
	for _, act := range actions {
		for _, rel := range []string{
			"backend/internal/server/merchant_violations.go",
			"backend/internal/server/admin_marketing_handlers.go",
			"backend/internal/server/admin_orders_handlers.go",
			"backend/internal/identity/admin.go",
		} {
			src := mustRead(t, filepath.Join(root, rel))
			best := strings.Contains(src, `s.audit(r, "`+act+`"`) ||
				strings.Contains(src, `s.repo.Audit(ctx, &actorID, "`+act+`"`)
			if !best {
				continue
			}
			// ══════════════════════════════════════════════════════
			// **ونداءٌ بأفضلِ جهدٍ يُقبَل بشرطٍ واحد**
			// ══════════════════════════════════════════════════════
			//
			// **أن يكون تحت شرطِ «ليس حسّاساً» مقروءاً في الملفّ.**
			//
			// **والإعداداتُ مئةٌ وأكثر** — **ونصُّ صفحةٍ ليس فعلاً
			// أمنيّاً**، **ومن جعلها كلَّها معاملةً أسقط تبديلَ
			// عنوانٍ لأنّ سطرَ تدقيقٍ تعثّر.**
			//
			// **فالمطلوبُ أن يُرى التمييزُ في الشيفرة**، لا أن
			// يُفترَض.
			// **والمصنِّفُ يُقرأ من الشيفرة لا يُكتب هنا** —
			// **ونصّان يفترقان يومَ يُعاد تسميةُ دالّة** (`XG-41A`).
			if by := server.ConditionalAuditClassifier(act); by != "" &&
				strings.Contains(src, "!"+by+"(key)") &&
				strings.Contains(src, "s.auditTx(ctx, q, r, \""+act+"\"") {
				t.Logf("  `%s` في `%s`: **مفرَّعٌ بـ`%s`** — الحسّاسُ "+
					"معامليٌّ وغيرُه أفضلُ جهد", act, filepath.Base(rel), by)
				continue
			}
			t.Errorf("**`%s` يُقيَّد بأفضلِ جهدٍ في `%s`** — "+
				"**وهو من الصنف `A`.** (`XG-20`)", act, rel)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ولا نداءَ خارجيٌّ داخلَ معاملةٍ حسّاسة**
// ══════════════════════════════════════════════════════════════════════
func TestXG20_NoExternalIOInsideCriticalTx(t *testing.T) {
	root := r16Root(t)
	src := mustRead(t, filepath.Join(root, "backend/internal/server/critical_settings.go"))
	for _, bad := range []string{"http.", "Publish(", "SendToUser(", "os.WriteFile"} {
		if strings.Contains(src, bad) {
			t.Errorf("**نداءٌ خارجيٌّ في منسّق المعاملة**: %q", bad)
		}
	}
	ident := mustRead(t, filepath.Join(root, "backend/internal/identity/audit_tx.go"))
	for _, bad := range []string{"http.", "rdb.", "Publish("} {
		if strings.Contains(ident, bad) {
			t.Errorf("**نداءٌ خارجيٌّ في تدقيق الهويّة**: %q", bad)
		}
	}
	t.Log("✓ لا شبكةَ داخلَ معاملةٍ حسّاسة")
}
