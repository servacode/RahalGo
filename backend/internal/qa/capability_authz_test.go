package qa

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **اسمُ الدور ليس صلاحيّة** — `ADG-1` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ بنصّه
//
//	NO-CODE FOR OPERATIONS · CODE FOR NEW CAPABILITIES
//
// **معجمُ القدرات في الشيفرة · وربطُ الدور بالقدرة في القاعدة يُبدّله
// الأدمن.** **والافتراضُ منع.**
//
// # والسؤالُ الكانونيّ
//
//	أيملك هذا الفاعلُ هذه القدرة؟   — لا «أدورُه `admin`؟»

// capRole **دورٌ صناعيٌّ بقدراتٍ معلومة** — لاختبار العقد لا الأدوار
// القائمة.
//
// **وحسابٌ بدورٍ لا يملك شيئاً حتّى يُمنَح** — وهو الافتراض.
func capRole(t *testing.T, hh *Harness, code string, caps ...authz.Capability) {
	t.Helper()
	if _, err := hh.Pool.Exec(ctxBG(),
		`INSERT INTO roles (code, name_key) VALUES ($1, $1) ON CONFLICT DO NOTHING`,
		code); err != nil {
		t.Fatalf("إنشاءُ الدور %s: %v", code, err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = $1`, code)
	})
	for _, c := range caps {
		if _, err := hh.Pool.Exec(ctxBG(), `
			INSERT INTO role_capabilities (role_code, capability_code)
			VALUES ($1, $2) ON CONFLICT DO NOTHING`, code, string(c)); err != nil {
			t.Fatalf("منحُ القدرة %s: %v", c, err)
		}
	}
}

// capUser حسابٌ بأدوارٍ صناعيّةٍ وجلسةٌ دائمةٌ له.
func capUser(t *testing.T, hh *Harness, roles ...string) (u *User, tok string) {
	t.Helper()
	u = hh.NewUser("customer")
	for _, r := range roles {
		if _, err := hh.Pool.Exec(ctxBG(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, u.ID, r); err != nil {
			t.Fatalf("منحُ الدور %s: %v", r, err)
		}
	}
	var sid string
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`,
		u.ID, "qa-"+uniq("h")).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	return u, hh.TokenWithSession(u.ID, sid, append(roles, "customer")...)
}

// financePath فعلٌ ماليٌّ حقيقيّ — قيدُ محفظة.
func financePath(uid string) string {
	return "/api/v1/admin/users/" + uid + "/wallet"
}

// ══════════════════════════════════════════════════════════════════════
// **D1+D2+D5 · قدرةٌ ممنوحةٌ تمرّ · وغيرُها يُمنَع · ودورٌ خالٍ يُمنَع**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D1D2D5_CapabilityDecides(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")

	capRole(t, hh, "qa_money", authz.FinanceManage)
	capRole(t, hh, "qa_reader", authz.UsersRead)
	capRole(t, hh, "qa_empty")

	body := map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"}

	_, moneyTok := capUser(t, hh, "qa_money")
	_, readTok := capUser(t, hh, "qa_reader")
	_, emptyTok := capUser(t, hh, "qa_empty")

	d1 := hh.POST(financePath(victim.ID), moneyTok, body)
	d2 := hh.POST(financePath(victim.ID), readTok, body)
	d5 := hh.POST(financePath(victim.ID), emptyTok, body)
	t.Logf("D1 قدرةٌ ممنوحة ⇒ %d · D2 دورٌ بلا القدرة ⇒ %d · D5 دورٌ خالٍ ⇒ %d",
		d1.Code, d2.Code, d5.Code)

	if d1.Code >= 400 {
		t.Errorf("**قدرةٌ ممنوحةٌ لم تمرّ** (%d) — %s", d1.Code, d1)
	}
	if d2.Code != http.StatusForbidden {
		t.Errorf("**دورٌ بلا القدرة مرّ** (%d)", d2.Code)
	}
	if d5.Code != http.StatusForbidden {
		t.Errorf("**دورٌ بلا منحٍ مرّ** (%d) — **والافتراضُ منع.**", d5.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D3 · دورٌ لا تعرفه المنصّةُ ⇒ لا قدرةَ له**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D3_UnknownRoleHasNothing(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")

	// **دورٌ يُخترَع في الرمز ولا وجودَ له في القاعدة.**
	u := hh.NewUser("customer")
	var sid string
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`,
		u.ID, "qa-"+uniq("h")).Scan(&sid); err != nil {
		t.Fatalf("صفُّ الجلسة: %v", err)
	}
	tok := hh.TokenWithSession(u.ID, sid, "qa_ghost_role", "admin")

	got := hh.POST(financePath(victim.ID), tok,
		map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"})
	t.Logf("D3: دورٌ مخترَعٌ في الرمز (ومعه `admin`) ⇒ %d", got.Code)
	if got.Code != http.StatusForbidden {
		t.Errorf("**ادّعاءُ دورٍ في الرمز خوّل** (%d) — "+
			"**والحقيقةُ في القاعدة.** (`ADG-1` · `R15`)", got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D4 · قدرةٌ مجهولةٌ ⇒ منع**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا تُقاس بنداءٍ**: **الحارسُ يُوقف الإقلاع** إن كُتبت في مسار.
// **فتُقاس في المعجم نفسِه.**
func TestADG1_D4_UnknownCapabilityDenied(t *testing.T) {
	if authz.Known("لا.وجودَ.لها") {
		t.Error("**قدرةٌ مجهولةٌ عُدّت معروفة**")
	}
	if authz.Known("") {
		t.Error("**الفراغُ عُدّ قدرة**")
	}
	for _, c := range authz.All() {
		if !authz.Known(c) || authz.Describe(c) == "" {
			t.Errorf("**قدرةٌ مسجَّلةٌ بلا وصف**: %q", c)
		}
	}
	t.Logf("D4: المعجمُ %d قدرةً · والمجهولةُ تُمنَع", authz.Count())
}

// ══════════════════════════════════════════════════════════════════════
// **D6+D7+D8 · اتّحادٌ · ونزعُ دورٍ · ونزعُ منحٍ — كلٌّ يسري فوراً**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D6D7D8_UnionAndImmediateRevocation(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_a", authz.FinanceManage)
	capRole(t, hh, "qa_b", authz.OrdersIntervene)
	u, tok := capUser(t, hh, "qa_a", "qa_b")

	money := map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"}
	oid, _ := activeOrderFor(t, hh)
	move := func() int {
		return hh.POST("/api/v1/admin/orders/"+oid+"/transition", tok,
			map[string]any{"to": "cancelled", "note": "ADG-1"}).Code
	}

	// ── D6 · الاتّحاد ─────────────────────────────────────────────
	d6a := hh.POST(financePath(victim.ID), tok, money).Code
	d6b := move()
	t.Logf("D6: قدرةُ `qa_a` ⇒ %d · قدرةُ `qa_b` ⇒ %d", d6a, d6b)
	// **والمقيسُ المنعُ لا تعارضُ العمل**: **`409` يعني أنّ القدرةَ
	// مرّت ورَدَّ منطقُ الطلب** — **وهو نجاحُ تخويلٍ لا فشلُه.**
	if d6a == http.StatusForbidden || d6b == http.StatusForbidden {
		t.Errorf("**الاتّحادُ لم يعمل**: %d · %d", d6a, d6b)
	}

	// ── D7 · يُنزَع دورٌ فتذهب قدرتُه وحدَها ─────────────────────
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM user_roles WHERE user_id = $1::uuid AND role_code = 'qa_a'`,
		u.ID); err != nil {
		t.Fatalf("نزعُ الدور: %v", err)
	}
	d7a := hh.POST(financePath(victim.ID), tok, money).Code
	oid2, _ := activeOrderFor(t, hh)
	d7b := hh.POST("/api/v1/admin/orders/"+oid2+"/transition", tok,
		map[string]any{"to": "cancelled", "note": "ADG-1"}).Code
	t.Logf("D7: بعد نزع `qa_a` — الماليّة ⇒ %d · التدخّل ⇒ %d", d7a, d7b)
	if d7a != http.StatusForbidden {
		t.Errorf("**قدرةُ دورٍ نُزع ما زالت تعمل** (%d) — `R15`", d7a)
	}
	if d7b == http.StatusForbidden {
		t.Errorf("**قدرةُ دورٍ لم يُنزَع ذهبت** (%d)", d7b)
	}

	// ── D8 · ويُنزَع المنحُ من الدور فتذهب قدرتُه ────────────────
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM role_capabilities WHERE role_code = 'qa_b'`); err != nil {
		t.Fatalf("نزعُ المنح: %v", err)
	}
	oid3, _ := activeOrderFor(t, hh)
	d8 := hh.POST("/api/v1/admin/orders/"+oid3+"/transition", tok,
		map[string]any{"to": "cancelled", "note": "ADG-1"}).Code
	t.Logf("D8: بعد نزع منحِ `qa_b` — التدخّل ⇒ %d", d8)
	if d8 != http.StatusForbidden {
		t.Errorf("**قدرةٌ نُزعت من الدور ما زالت تعمل** (%d)", d8)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D9 · ومنحُ قدرةٍ يسري من القاعدة**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D9_GrantBecomesEffective(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_grow")
	_, tok := capUser(t, hh, "qa_grow")
	money := map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"}

	before := hh.POST(financePath(victim.ID), tok, money).Code
	if _, err := hh.Pool.Exec(ctxBG(), `
		INSERT INTO role_capabilities (role_code, capability_code)
		VALUES ('qa_grow', $1)`, string(authz.FinanceManage)); err != nil {
		t.Fatalf("منحُ القدرة: %v", err)
	}
	after := hh.POST(financePath(victim.ID), tok, money).Code
	t.Logf("D9: قبل المنح ⇒ %d · بعده بالرمز نفسِه ⇒ %d", before, after)

	if before != http.StatusForbidden {
		t.Fatalf("**التركيبةُ خطأ**: مرّ قبل المنح (%d)", before)
	}
	if after >= 400 {
		t.Errorf("**المنحُ لم يسرِ** (%d) — **والحقيقةُ في القاعدة.**", after)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D10 · ولا ادّعاءَ في رمزٍ يعلو على القاعدة**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D10_StaleTokenCannotOverrideDB(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_none")
	u, _ := capUser(t, hh, "qa_none")

	// **رمزٌ يحمل `admin` و`finance` وصاحبُه لا يملكهما في القاعدة.**
	var sid string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT session_id::text FROM refresh_tokens
		 WHERE user_id = $1::uuid ORDER BY created_at DESC LIMIT 1`,
		u.ID).Scan(&sid); err != nil {
		t.Fatalf("معرّفُ الجلسة: %v", err)
	}
	fat := hh.TokenWithSession(u.ID, sid, "admin", "finance", "ops")

	got := hh.POST(financePath(victim.ID), fat,
		map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"})
	t.Logf("D10: رمزٌ يدّعي `admin`+`finance`+`ops` ⇒ %d", got.Code)
	if got.Code != http.StatusForbidden {
		t.Errorf("**ادّعاءُ الرمز علا على القاعدة** (%d) — `R15` · `ADG-1`",
			got.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D11+D12+D13 · إعدادٌ حسّاسٌ · مالٌ · أدوارٌ — كلٌّ بقدرته**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D11D12D13_SensitiveRoutesNeedCapability(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")

	// **دورٌ يملك العامَّ من الإعدادات وحدَه.**
	capRole(t, hh, "qa_content", authz.SettingsGeneralManage, authz.UsersRead)
	_, tok := capUser(t, hh, "qa_content")

	fin := hh.Call("PUT", "/api/v1/admin/settings/merchants.commission_percent",
		tok, map[string]any{"value": 11}, nil)
	sec := hh.Call("PUT", "/api/v1/admin/settings/security.session_days",
		tok, map[string]any{"value": 20}, nil)
	money := hh.POST(financePath(victim.ID), tok,
		map[string]any{"amount": 1000, "kind": "topup", "note": "ADG-1"})
	roles := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", tok,
		map[string]any{"role": "finance", "reason": "ADG-1"})

	t.Logf("D11: إعدادٌ ماليٌّ ⇒ %d · أمنيٌّ ⇒ %d · D12 مالٌ ⇒ %d · D13 أدوارٌ ⇒ %d",
		fin.Code, sec.Code, money.Code, roles.Code)

	for name, res := range map[string]Res{
		"إعدادٌ ماليّ": fin, "إعدادٌ أمنيّ": sec,
		"قيدُ محفظة": money, "منحُ دور": roles,
	} {
		if res.Code != http.StatusForbidden {
			t.Errorf("**%s مرّ بقدرةِ المحتوى وحدَها** (%d) — "+
				"**ولكلّ إعدادٍ صلاحيّتُه بحسب أثره.** (`ADG-1`)",
				name, res.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D14+D15 · والتدقيقُ محفوظ · والمنعُ لا يترك أثراً كاذباً**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D14D15_AuditPreservedAndDenialLeavesNothing(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_norole", authz.UsersRead)
	_, weak := capUser(t, hh, "qa_norole")

	// ── D15 · منعٌ ⇒ لا تبديلَ ولا أثرٌ كاذب ─────────────────────
	denied := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", weak,
		map[string]any{"role": "finance", "reason": "ADG-1"})
	var has bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = 'finance')`,
		victim.ID).Scan(&has); err != nil {
		t.Fatalf("قراءةُ الأدوار: %v", err)
	}
	n := auditCount(t, hh, "admin.role_grant", victim.ID)
	t.Logf("D15: المنعُ ⇒ %d · الدورُ مُنح=%v · آثارٌ=%d", denied.Code, has, n)
	if denied.Code != http.StatusForbidden {
		t.Fatalf("**مرّ منحُ دورٍ بلا قدرة** (%d)", denied.Code)
	}
	if has {
		t.Error("**مُنع الفعلُ ووقع تبديلُه**")
	}
	if n != 0 {
		t.Errorf("**أثرٌ كاذبٌ لفعلٍ مُنع**: %d", n)
	}

	// ── D14 · وفعلٌ مسموحٌ يُقيَّد كما كان ────────────────────────
	capRole(t, hh, "qa_roles", authz.RolesManage)
	_, strong := capUser(t, hh, "qa_roles")
	ok := hh.POST("/api/v1/admin/users/"+victim.ID+"/roles", strong,
		map[string]any{"role": "finance", "reason": "ADG-1"})
	after := auditCount(t, hh, "admin.role_grant", victim.ID)
	t.Logf("D14: المسموحُ ⇒ %d · آثارٌ=%d", ok.Code, after)
	if ok.Code >= 400 {
		t.Fatalf("**قدرةُ `roles.manage` لم تمرّ** (%s)", ok)
	}
	if after == 0 {
		t.Error("**فعلٌ حسّاسٌ وقع بلا أثر** — **و`AQ-4` مغلقة.**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **D16 · والبثُّ يتبع القرارَ نفسَه**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_D16_WSUsesSameCapabilityTruth(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "qa_ops_read", authz.OrdersRead)
	capRole(t, hh, "qa_blind")
	_, seeing := capUser(t, hh, "qa_ops_read")
	_, blind := capUser(t, hh, "qa_blind")

	a := hh.GET("/api/v1/ws?token="+seeing, "")
	b := hh.GET("/api/v1/ws?token="+blind, "")
	t.Logf("D16: بقدرة `orders.read` ⇒ %d · بلا قدرةٍ ⇒ %d", a.Code, b.Code)

	// **والمصافحةُ تمرّ للاثنين** — **والفرقُ في المواضيع.**
	// **والمقيسُ أنّ منطقاً واحداً يحكمهما**: `dbCaps` في `ws.go`.
	src := mustRead(t, filepath.Join(r16Root(t), "backend/internal/server/ws.go"))
	if !strings.Contains(src, "authz.OrdersRead") {
		t.Error("**البثُّ لا يستعمل القرارَ المركزيّ** — " +
			"**ومنطقُ أدوارٍ ثانٍ يفترق.** (`ADG-1`)")
	}
	if strings.Contains(src, `role == "admin" || role == "ops"`) {
		t.Error("**عاد فحصُ أسماء الأدوار في البثّ**")
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1+C2 · نزعُ دورٍ ونزعُ منحٍ يتزامنان مع فعلٍ مخوَّل**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_C1C2_RevocationVsPrivilegedRequest(t *testing.T) {
	hh := New(t)
	victim := hh.NewUser("customer")
	capRole(t, hh, "qa_race", authz.FinanceManage)
	u, tok := capUser(t, hh, "qa_race")
	money := map[string]any{"amount": 500, "kind": "topup", "note": "ADG-1"}

	// ── C1 · نزعُ الدور ──────────────────────────────────────────
	r1 := Race(t, DefaultRaceTimeout,
		Actor{Name: "فعلٌ ماليّ", Do: func(context.Context) any {
			return hh.POST(financePath(victim.ID), tok, money)
		}},
		Actor{Name: "نزعُ الدور", Do: func(context.Context) any {
			_, err := hh.Pool.Exec(ctxBG(),
				`DELETE FROM user_roles WHERE user_id = $1::uuid AND role_code = 'qa_race'`,
				u.ID)
			return err
		}},
	)
	if r1.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r1)
	}
	after1 := hh.POST(financePath(victim.ID), tok, money)
	t.Logf("C1: بعد ثبوت النزع ⇒ %d — %s", after1.Code, r1)
	if after1.Code != http.StatusForbidden {
		t.Errorf("**فعلٌ مخوَّلٌ يمرّ بعد ثبوت نزع الدور** (%d)", after1.Code)
	}

	// ── C2 · نزعُ المنح من الدور ─────────────────────────────────
	capRole(t, hh, "qa_race2", authz.FinanceManage)
	u2, tok2 := capUser(t, hh, "qa_race2")
	_ = u2
	r2 := Race(t, DefaultRaceTimeout,
		Actor{Name: "فعلٌ ماليّ", Do: func(context.Context) any {
			return hh.POST(financePath(victim.ID), tok2, money)
		}},
		Actor{Name: "نزعُ المنح", Do: func(context.Context) any {
			_, err := hh.Pool.Exec(ctxBG(),
				`DELETE FROM role_capabilities WHERE role_code = 'qa_race2'`)
			return err
		}},
	)
	if r2.TimedOut {
		t.Fatalf("السباقُ عَلِق — %s", r2)
	}
	after2 := hh.POST(financePath(victim.ID), tok2, money)
	t.Logf("C2: بعد ثبوت نزع المنح ⇒ %d — %s", after2.Code, r2)
	if after2.Code != http.StatusForbidden {
		t.Errorf("**فعلٌ مخوَّلٌ يمرّ بعد ثبوت نزع المنح** (%d)", after2.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **حرّاسٌ بنيويّة**
// ══════════════════════════════════════════════════════════════════════
func TestADG1_StructuralGuards(t *testing.T) {
	root := r16Root(t)

	// ── ولا معرّفَ مكرَّرٌ ولا معجمٌ ينكمش ───────────────────────
	seen := map[authz.Capability]bool{}
	for _, c := range authz.All() {
		if seen[c] {
			t.Errorf("**معرّفُ قدرةٍ مكرَّر**: %q", c)
		}
		seen[c] = true
	}
	if authz.Count() < 15 {
		t.Errorf("**المعجمُ انكمش**: %d", authz.Count())
	}
	t.Logf("المعجمُ = %d قدرةً", authz.Count())

	// ── والمساراتُ المُرحَّلةُ تُحرَس بقدرةٍ لا باسم دور ─────────
	srv := mustRead(t, filepath.Join(root, "backend/internal/server/server.go"))
	for _, migrated := range []string{
		`Post("/users/{id}/wallet"`,
		// **والمعالِجُ لا المسارُ**: `/orders/{id}/transition` يتكرّر
		// للسائق والمتجر والإدارة — **والمُرحَّلُ واحدٌ منها.**
		`s.handleOrderTransition)`,
		`Post("/users/{id}/roles"`,
		`Delete("/users/{id}/roles/{role}"`,
	} {
		i := strings.Index(srv, migrated)
		if i < 0 {
			t.Errorf("**اختفى مسارٌ مُرحَّل**: %s", migrated)
			continue
		}
		// **والحارسُ في السطر الذي قبله أو معه.**
		window := srv[max0(i-220):i]
		if !strings.Contains(window, "RequireCapability(") {
			t.Errorf("**مسارٌ مُرحَّلٌ بلا حارسِ قدرة**: %s — "+
				"**وعودةٌ إلى اسم الدور تراجع.** (`ADG-1`)", migrated)
		}
	}

	// ── والإعدادُ يُحرَس بقدرةٍ بحسب مفتاحه ──────────────────────
	h := mustRead(t, filepath.Join(root,
		"backend/internal/server/admin_marketing_handlers.go"))
	if !strings.Contains(h, "settingCapability(key)") {
		t.Error("**تبديلُ الإعداد بلا قدرةٍ بحسب أثره** (`ADG-1`)")
	}

	// ── ولا تعريفَ ثانٍ للحسّاس ──────────────────────────────────
	cap := mustRead(t, filepath.Join(root, "backend/internal/server/capability.go"))
	if !strings.Contains(cap, "criticalSettingKey(key)") {
		t.Error("**تصنيفُ الحسّاس كُرّر** — **وتصنيفان يفترقان.**")
	}
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
