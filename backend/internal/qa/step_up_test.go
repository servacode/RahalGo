package qa

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **تأكيدُ الفعل الحسّاس بكلمةِ صاحبه** — `ADG-3` · `AQ-1`
// ══════════════════════════════════════════════════════════════════════
//
// # العقدُ ثلاثةُ شروطٍ لا بدائل
//
//	جلسةٌ صالحة `R16` · وقدرةٌ قائمة `ADG-2` · وإثباتُ تأكيدٍ لهذا
//	الفعل بعينه
//
// **والإثباتُ ليس تخويلاً** — **ومن ملك القدرةَ ولم يؤكّد يُردّ،
// ومن أكّد ثمّ نُزعت قدرتُه يُردّ.**
//
// # وواحدٌ لفعلٍ واحد
//
// **لا «وضعُ مالكٍ مرفوع» يدوم خمسَ دقائق** — **إثباتٌ مربوطٌ
// بالفاعل وجلسته وفعله وهدفه وحقولِ تبديله**، **يُستهلَك مرّةً.**

const stepPass = "Qa-StepUp-2026!"

// stepUser حسابٌ بكلمةٍ معلومة — **والكلمةُ تُبصَم بالبدائيّة نفسِها.**
func stepUser(t *testing.T, hh *Harness, role string) *User {
	t.Helper()
	u := hh.NewUser(role)
	hash, err := auth.HashPassword(stepPass)
	if err != nil {
		t.Fatalf("بصمُ الكلمة: %v", err)
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE users SET password_hash = $2 WHERE id = $1::uuid`, u.ID, hash); err != nil {
		t.Fatalf("ضبطُ الكلمة: %v", err)
	}
	return u
}

// askStepUp يطلب إثباتاً لنداءٍ بعينه ويردّ (المعرّفَ، رمزَ الردّ).
func askStepUp(t *testing.T, hh *Harness, token, method, path string,
	body any, password string) (string, int) {
	t.Helper()
	res := hh.POST("/api/v1/admin/step-up", token, map[string]any{
		"method": method, "path": path, "body": body, "password": password,
	})
	if res.Code != http.StatusOK {
		return "", res.Code
	}
	id, _ := res.JSON()["id"].(string)
	return id, res.Code
}

// withGrant ينفّذ الفعلَ الحسّاسَ حاملاً إثباتاً (أو بلا إثبات).
// **والترويسةُ تُمرَّر دائماً** — **ولو فارغة**: **فحصُ `ADG-3` هو
// من يقرّر ما يحمله النداء**، ولا يصنع له المِسنَدُ إثباتاً.
func withGrant(hh *Harness, token, method, path string, body any, grant string) Res {
	return hh.Call(method, path, token, body, map[string]string{"X-Step-Up": grant})
}

// walletPath مسارُ قيدِ محفظةٍ لحسابٍ بعينه — فعلٌ حسّاسٌ بمبلغٍ جوهريّ.
func walletPath(id string) string { return "/api/v1/admin/users/" + id + "/wallet" }

func walletBody(amount int) map[string]any {
	return map[string]any{"amount": amount, "kind": "topup", "note": "ADG-3"}
}

// ══════════════════════════════════════════════════════════════════════
// **S1…S4 · الأساس**
// ══════════════════════════════════════════════════════════════════════

func TestADG3_S1S2S3S4_Basics(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	// ── S1 · قدرةٌ بلا تأكيد ⇒ يُردّ ─────────────────────────────
	no := withGrant(hh, admin.Token, "POST", path, walletBody(100), "")
	t.Logf("S1: بلا إثباتٍ ⇒ %d", no.Code)
	if no.Code != http.StatusForbidden {
		t.Errorf("**S1: مضى فعلٌ حسّاسٌ بلا تأكيد** — %d", no.Code)
	}

	// ── S3 · كلمةٌ خاطئة ⇒ لا إثبات ──────────────────────────────
	bad, code := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), "خطأ-تماماً")
	t.Logf("S3: كلمةٌ خاطئةٌ ⇒ %d · إثباتٌ=%q", code, bad)
	// **ورفضٌ لا عطل** — **و`503` هنا كان بصمَ كلمةٍ داخلَ معاملةٍ
	// مفتوحةٍ يجوّع المسبح**، فيسقط فحصُ جلسةِ نداءٍ آخر.
	if code != http.StatusForbidden {
		t.Errorf("**S3: كلمةٌ خاطئةٌ تُردّ بـ%d لا بمنع**", code)
	}
	if bad != "" || code == http.StatusOK {
		t.Errorf("**S3: صدر إثباتٌ بكلمةٍ خاطئة** — %d", code)
	}

	// ── S2 · كلمةٌ صحيحة ⇒ إثبات ─────────────────────────────────
	g, code := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	t.Logf("S2: كلمةٌ صحيحةٌ ⇒ %d · إثباتٌ=%v", code, g != "")
	if g == "" {
		t.Fatalf("**S2: لم يصدر إثباتٌ بكلمةٍ صحيحة** — %d", code)
	}

	// ── S4 · الإثباتُ المطابق ⇒ يمضي ─────────────────────────────
	ok := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("S4: بالإثبات ⇒ %d", ok.Code)
	if ok.Code >= 400 {
		t.Errorf("**S4: رُدّ فعلٌ مؤكَّدٌ صحيحاً** — %s", ok)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S5…S11 · حدودُ الإثبات**
// ══════════════════════════════════════════════════════════════════════

func TestADG3_S5S6_ReplayAndExpiry(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	// ── S5 · إعادةُ الاستعمال ⇒ يُردّ ────────────────────────────
	g, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	first := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	second := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("S5: الأوّلُ %d · الثاني %d", first.Code, second.Code)
	if first.Code >= 400 {
		t.Fatalf("**S5: سقط الأوّل** — %s", first)
	}
	if second.Code != http.StatusForbidden {
		t.Errorf("**S5: استُعمل إثباتٌ مرّتين** — %d", second.Code)
	}

	// ── S6 · إثباتٌ منتهٍ ⇒ يُردّ ────────────────────────────────
	g2, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(200), stepPass)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE step_up_grants SET expires_at = now() - interval '1 second'
		  WHERE id = $1::uuid`, g2); err != nil {
		t.Fatalf("تقديمُ الانتهاء: %v", err)
	}
	expired := withGrant(hh, admin.Token, "POST", path, walletBody(200), g2)
	t.Logf("S6: منتهٍ ⇒ %d", expired.Code)
	if expired.Code != http.StatusForbidden {
		t.Errorf("**S6: مضى إثباتٌ منتهٍ** — %d", expired.Code)
	}
}

func TestADG3_S7S8_SessionAndUserBinding(t *testing.T) {
	hh := New(t)
	a := stepUser(t, hh, "admin")
	b := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	g, _ := askStepUp(t, hh, a.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}

	// ── S8 · إثباتُ غيرِه ⇒ يُردّ ───────────────────────────────
	other := withGrant(hh, b.Token, "POST", path, walletBody(100), g)
	t.Logf("S8: إثباتُ فاعلٍ آخرَ ⇒ %d", other.Code)
	if other.Code != http.StatusForbidden {
		t.Errorf("**S8: نُقل إثباتٌ إلى فاعلٍ آخر** — %d", other.Code)
	}

	// ── S7 · جلسةٌ أخرى للفاعل نفسِه ⇒ يُردّ ────────────────────
	//
	// **وجلسةٌ ثانيةٌ تُصنَع كما في الإنتاج** — صفٌّ في
	// `refresh_tokens` ثمّ توكنٌ يحمل عائلتَها.
	var sid string
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`, a.ID, "qa-adg3-"+a.ID).Scan(&sid); err != nil {
		t.Fatalf("جلسةٌ ثانية: %v", err)
	}
	tok2, _, err := hh.tokens.IssueAccess(a.ID, []string{"admin"}, sid)
	if err != nil {
		t.Fatalf("توكنُ الجلسة الثانية: %v", err)
	}
	sess := withGrant(hh, tok2, "POST", path, walletBody(100), g)
	t.Logf("S7: جلسةٌ أخرى ⇒ %d", sess.Code)
	if sess.Code != http.StatusForbidden {
		t.Errorf("**S7: عبر الإثباتُ إلى جلسةٍ أخرى** — %d", sess.Code)
	}
}

func TestADG3_S9S10S11_ActionTargetAndParameters(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	a := hh.NewUser("customer")
	b := hh.NewUser("customer")

	g, _ := askStepUp(t, hh, admin.Token, "POST", walletPath(a.ID), walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}

	// ── S10 · هدفٌ آخر ⇒ يُردّ ──────────────────────────────────
	wrongTarget := withGrant(hh, admin.Token, "POST", walletPath(b.ID), walletBody(100), g)
	t.Logf("S10: هدفٌ آخرُ ⇒ %d", wrongTarget.Code)
	if wrongTarget.Code != http.StatusForbidden {
		t.Errorf("**S10: أُجيز فعلٌ على هدفٍ لم يُؤكَّد** — %d", wrongTarget.Code)
	}

	// ── S11 · مبلغٌ آخر ⇒ يُردّ ─────────────────────────────────
	//
	// **ومن أكّد مئةً لا يكون قد أكّد خمسين ألفاً.**
	wrongAmount := withGrant(hh, admin.Token, "POST", walletPath(a.ID), walletBody(50000), g)
	t.Logf("S11: مبلغٌ آخرُ ⇒ %d", wrongAmount.Code)
	if wrongAmount.Code != http.StatusForbidden {
		t.Errorf("**S11: أُجيز مبلغٌ لم يُؤكَّد** — %d", wrongAmount.Code)
	}

	// ── S9 · فعلٌ آخر ⇒ يُردّ ───────────────────────────────────
	wrongAction := withGrant(hh, admin.Token, "POST",
		"/api/v1/admin/users/"+a.ID+"/roles",
		map[string]any{"role": "finance", "reason": "ADG-3"}, g)
	t.Logf("S9: فعلٌ آخرُ ⇒ %d", wrongAction.Code)
	if wrongAction.Code != http.StatusForbidden {
		t.Errorf("**S9: أُجيز فعلٌ لم يُؤكَّد** — %d", wrongAction.Code)
	}

	// **والأصلُ ما زال يمضي** — فالحدودُ ضيّقت ولم تُبطل.
	good := withGrant(hh, admin.Token, "POST", walletPath(a.ID), walletBody(100), g)
	if good.Code >= 400 {
		t.Errorf("**بطل الإثباتُ الصحيحُ بعد محاولاتٍ خاطئة** — %s", good)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S12…S14 · الشرطان الآخران يغلبان**
// ══════════════════════════════════════════════════════════════════════

func TestADG3_S12_CapabilityRevokedAfterIssuance(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	g, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}
	// **ثمّ نُزع الدور** — والقدرةُ تُقرأ من الحقيقة الموثوقة (`R15`).
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM user_roles WHERE user_id = $1::uuid`, admin.ID); err != nil {
		t.Fatalf("نزعُ الدور: %v", err)
	}
	res := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("S12: قدرةٌ نُزعت بعد الإصدار ⇒ %d", res.Code)
	if res.Code != http.StatusForbidden {
		t.Errorf("**S12: أحيا الإثباتُ قدرةً منزوعة** — %d", res.Code)
	}
}

func TestADG3_S13S14_SessionRevokedAndBlocked(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	// ── S13 · جلسةٌ أُبطلت بعد الإصدار ──────────────────────────
	g, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE refresh_tokens SET revoked_at = now() WHERE user_id = $1::uuid`,
		admin.ID); err != nil {
		t.Fatalf("إبطالُ الجلسة: %v", err)
	}
	res := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("S13: جلسةٌ أُبطلت ⇒ %d", res.Code)
	if res.Code != http.StatusUnauthorized && res.Code != http.StatusForbidden {
		t.Errorf("**S13: أحيا الإثباتُ جلسةً مبطَلة** — %d", res.Code)
	}

	// ── S14 · محظورٌ ⇒ إثباتُه لا يُستعمَل ──────────────────────
	admin2 := stepUser(t, hh, "admin")
	g2, _ := askStepUp(t, hh, admin2.Token, "POST", path, walletBody(100), stepPass)
	if g2 == "" {
		t.Fatal("لم يصدر إثبات")
	}
	// **والحظرُ يقع بمساره الحقيقيّ** — **وحظرٌ يُكتب في القاعدة
	// بيدٍ ليس حظراً**: `AdminUpdateUser` هو من يُبطل الجلسات.
	blocker := stepUser(t, hh, "admin")
	bp := "/api/v1/admin/users/" + admin2.ID
	bb := map[string]any{"status": "blocked", "status_reason": "ADG-3"}
	bg, code := askStepUp(t, hh, blocker.Token, "PATCH", bp, bb, stepPass)
	if bg == "" {
		t.Fatalf("تأكيدُ الحظر: %d", code)
	}
	if res := withGrant(hh, blocker.Token, "PATCH", bp, bb, bg); res.Code >= 400 {
		t.Fatalf("الحظر: %s", res)
	}
	res2 := withGrant(hh, admin2.Token, "POST", path, walletBody(100), g2)
	t.Logf("S14: محظورٌ ⇒ %d", res2.Code)
	if res2.Code < 400 {
		t.Errorf("**S14: مضى فعلُ محظورٍ بإثباتٍ سابق** — %d", res2.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S15 · تبديلُ المرء لكلمته يُبطل إثباتَه** — `XG-40` محفوظ
// ══════════════════════════════════════════════════════════════════════
//
// **والجلسةُ تبقى** بعقد دورةِ ١٩ — **والإثباتُ لا يبقى.**
func TestADG3_S15_SelfPasswordChangeKillsGrant(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	g, code0 := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatalf("لم يصدر إثبات — %d", code0)
	}
	const newPass = "Qa-StepUp-New-2026!"
	// **وعبر المسار الحقيقيّ** — `changePassword` من فحوص `XG-40`.
	if ch := changePassword(t, hh, admin.Token, stepPass, newPass); ch.Code >= 400 {
		t.Fatalf("تبديلُ الكلمة: %s", ch)
	}

	stale := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("S15: إثباتٌ قبل التبديل ⇒ %d", stale.Code)
	if stale.Code != http.StatusForbidden {
		t.Errorf("**S15: نجا إثباتٌ صدر باعتمادٍ قديم** — %d", stale.Code)
	}

	// **والكلمةُ الجديدةُ تُصدر إثباتاً جديداً** — والجلسةُ نفسُها.
	g2, code := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), newPass)
	t.Logf("S15: بالكلمة الجديدة ⇒ %d", code)
	if g2 == "" {
		t.Fatalf("**S15: تعذّر التأكيدُ بالكلمة الجديدة** — %d", code)
	}
	ok := withGrant(hh, admin.Token, "POST", path, walletBody(100), g2)
	if ok.Code >= 400 {
		t.Errorf("**S15: رُدّ إثباتٌ جديدٌ صحيح** — %s", ok)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S16 · إعادةُ الإدارة للكلمة** — `R13` محفوظ
// ══════════════════════════════════════════════════════════════════════
func TestADG3_S16_AdminResetKillsGrant(t *testing.T) {
	hh := New(t)
	owner := stepUser(t, hh, "admin")
	target := stepUser(t, hh, "admin")
	victim := hh.NewUser("customer")
	path := walletPath(victim.ID)

	g, _ := askStepUp(t, hh, target.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}
	// **والإعادةُ فعلٌ حسّاسٌ بذاته** — فتُؤكَّد.
	rp := "/api/v1/admin/users/" + target.ID + "/password"
	rb := map[string]any{"password": "Qa-Reset-2026!"}
	rg, code := askStepUp(t, hh, owner.Token, "POST", rp, rb, stepPass)
	if rg == "" {
		t.Fatalf("تأكيدُ الإعادة: %d", code)
	}
	reset := withGrant(hh, owner.Token, "POST", rp, rb, rg)
	if reset.Code >= 400 {
		t.Fatalf("إعادةُ الكلمة: %s", reset)
	}

	res := withGrant(hh, target.Token, "POST", path, walletBody(100), g)
	t.Logf("S16: بعد إعادة الإدارة ⇒ %d", res.Code)
	if res.Code < 400 {
		t.Errorf("**S16: نجا إثباتٌ بعد إعادةِ كلمةٍ إداريّة** — %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S17 · S18 · وما ليس حسّاساً يمضي بلا تأكيد**
// ══════════════════════════════════════════════════════════════════════
//
// **ومن طلب كلمةً في كلّ ضغطةٍ علّم الناسَ أن يكتبوها بلا نظر.**
func TestADG3_S17S18_OrdinaryActionsNeedNoStepUp(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")

	cases := []struct {
		name, method, path string
		body               any
	}{
		{"قراءةُ الحسابات", "GET", "/api/v1/admin/users?limit=1", nil},
		{"لافتات", "GET", "/api/v1/admin/banners", nil},
		{"إعدادٌ عامّ", "PUT", "/api/v1/admin/settings/orders.auto_transfer",
			map[string]any{"value": false}},
		{"إيقافٌ عاديّ", "PATCH", "/api/v1/admin/users/" + v.ID,
			map[string]any{"status": "suspended", "status_reason": "ADG-3"}},
	}
	for _, c := range cases {
		res := withGrant(hh, admin.Token, c.method, c.path, c.body, "")
		t.Logf("S17/S18: %s بلا تأكيدٍ ⇒ %d", c.name, res.Code)
		if res.Code >= 400 {
			t.Errorf("**فعلٌ عاديٌّ طُلب له تأكيد**: %s — %s", c.name, res)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **H1…H15 · مصفوفةُ الأفعال الشديدة**
// ══════════════════════════════════════════════════════════════════════
//
// **والمقياسُ واحد**: **بلا إثباتٍ يُردّ، وبإثباتٍ مطابقٍ يمضي.**
func TestADG3_HighRiskMatrix(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")
	drv := hh.Factory().Driver()

	cases := []struct {
		id, name, method, path string
		body                   any
		wantExec               bool
	}{
		{"H3", "منحُ دورٍ لحساب", "POST", "/api/v1/admin/users/" + v.ID + "/roles",
			map[string]any{"role": "analytics", "reason": "ADG-3"}, true},
		{"H4", "نزعُ دورٍ من حساب", "DELETE",
			"/api/v1/admin/users/" + v.ID + "/roles/analytics", nil, false},
		{"H1", "منحُ قدرةٍ لدور", "POST", "/api/v1/admin/roles/analytics/capabilities",
			map[string]any{"capability": "users.read"}, true},
		{"H2", "نزعُ قدرةٍ من دور", "DELETE",
			"/api/v1/admin/roles/analytics/capabilities/users.read", nil, true},
		{"H5", "إعادةُ كلمةٍ إداريّة", "POST",
			"/api/v1/admin/users/" + v.ID + "/password",
			map[string]any{"password": "Qa-Reset-2026!"}, true},
		{"H6", "حظرُ حساب", "PATCH", "/api/v1/admin/users/" + v.ID,
			map[string]any{"status": "blocked", "status_reason": "ADG-3"}, true},
		{"H9", "قيدُ محفظة", "POST", walletPath(v.ID), walletBody(100), false},
		{"H12", "مصروفٌ جديد", "POST", "/api/v1/admin/expenses",
			map[string]any{"amount": 1000, "note": "ADG-3"}, false},
		{"H13", "إعدادٌ ماليّ", "PUT",
			"/api/v1/admin/settings/merchants.commission_percent",
			map[string]any{"value": 13}, true},
		{"H14", "إعدادٌ أمنيّ", "PUT", "/api/v1/admin/settings/security.session_days",
			map[string]any{"value": 21}, true},
		{"H11", "تسويةُ نقدِ سائق", "POST",
			"/api/v1/admin/drivers/" + drv.ID + "/settle",
			map[string]any{"amount": 0, "note": "ADG-3"}, false},
	}

	for _, c := range cases {
		// **بلا إثباتٍ ⇒ يُردّ.**
		no := withGrant(hh, admin.Token, c.method, c.path, c.body, "")
		if no.Code != http.StatusForbidden {
			t.Errorf("**%s (%s): مضى بلا تأكيد** — %d", c.id, c.name, no.Code)
			continue
		}
		// **وبإثباتٍ مطابقٍ يُقبَل الإثبات.**
		g, code := askStepUp(t, hh, admin.Token, c.method, c.path, c.body, stepPass)
		if g == "" {
			t.Errorf("**%s (%s): تعذّر إصدارُ الإثبات** — %d", c.id, c.name, code)
			continue
		}
		yes := withGrant(hh, admin.Token, c.method, c.path, c.body, g)
		mark := "✓"
		if c.wantExec && yes.Code >= 400 {
			mark = "✗"
			t.Errorf("**%s (%s): رُدّ فعلٌ مؤكَّد** — %s", c.id, c.name, yes)
		}
		// **وما لا يُشترَط نجاحُه يكفي ألّا يُردّ بسبب التأكيد.**
		if !c.wantExec && yes.Code == http.StatusForbidden {
			mark = "✗"
			t.Errorf("**%s (%s): رُدّ التأكيدُ نفسُه** — %s", c.id, c.name, yes)
		}
		t.Logf("%s %-4s %-24s بلا=%d · بإثباتٍ=%d", mark, c.id, c.name, no.Code, yes.Code)
	}

	// ── H8 · H15 · ما لا يُؤكَّد ────────────────────────────────
	for _, c := range []struct {
		id, name, method, path string
		body                   any
	}{
		{"H8", "إيقافٌ عاديّ", "PATCH", "/api/v1/admin/users/" + v.ID,
			map[string]any{"status": "suspended", "status_reason": "ADG-3"}},
		{"H15", "إعدادٌ عامّ", "PUT", "/api/v1/admin/settings/orders.auto_transfer",
			map[string]any{"value": false}},
	} {
		res := withGrant(hh, admin.Token, c.method, c.path, c.body, "")
		t.Logf("✓ %-4s %-24s بلا تأكيدٍ ⇒ %d", c.id, c.name, res.Code)
		if res.Code == http.StatusForbidden {
			t.Errorf("**%s (%s): طُلب تأكيدٌ لفعلٍ عاديّ** — %d", c.id, c.name, res.Code)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1 · إثباتٌ واحدٌ ونداءان متزامنان**
// ══════════════════════════════════════════════════════════════════════
//
// **ونقطةُ الحسم تحديثٌ شرطيٌّ واحد** — **فلا يظفر به إلّا واحد.**
func TestADG3_C1_ConcurrentDoubleUse(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")
	path := walletPath(v.ID)

	g, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}

	act := func(name string) Actor {
		return Actor{Name: name, Do: func(context.Context) any {
			return withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
		}}
	}
	r := Race(t, 0, act("أ"), act("ب"))
	okCount := r.CountOK()
	var used int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM step_up_grants
		  WHERE id = $1::uuid AND consumed_at IS NOT NULL`, g).Scan(&used)
	t.Logf("C1: نجح %d من 2 · تداخلٌ=%d · مستهلَكٌ=%d",
		okCount, r.Probe.Max(), used)

	if okCount != 1 {
		t.Errorf("**C1: ظفر بالإثباتِ %d من نداءين** — **وواحدٌ لا غير.**", okCount)
	}
	if r.Probe.Max() < 2 {
		t.Errorf("**C1: تداخلٌ مقيسٌ %d** — والسيناريو يشترط تزامناً", r.Probe.Max())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C5 · إصدارٌ يسابق تبديلَ الكلمة**
// ══════════════════════════════════════════════════════════════════════
//
// **وإثباتٌ صدر باعتمادٍ قديمٍ لا ينجو من تبديلٍ ثُبِّت** — **والمقارنةُ
// لحظةَ الاستعمال هي الحكم**، كما في `R15`.
func TestADG3_C5_StaleCredentialIssueRace(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")
	path := walletPath(v.ID)

	// **إثباتٌ صدر قبل التبديل** — ثمّ تُدمَغ الحقبة.
	g, _ := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE users SET sessions_revoked_at = now() WHERE id = $1::uuid`,
		admin.ID); err != nil {
		t.Fatalf("دمغُ الحقبة: %v", err)
	}
	res := withGrant(hh, admin.Token, "POST", path, walletBody(100), g)
	t.Logf("C5: حقبةٌ دُمغت بعد الإصدار ⇒ %d", res.Code)
	if res.Code < 400 {
		t.Errorf("**C5: نجا إثباتٌ باعتمادٍ شاخ** — %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **الحقنُ — إثباتٌ لا يُكتب لا يُستعمَل**
// ══════════════════════════════════════════════════════════════════════
func TestADG3_FailureInjection_NoFalseGrant(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")
	path := walletPath(v.ID)

	fp := hh.ArmAny("ADG3/grant-insert", "step_up_grants", "INSERT")
	defer fp.Disarm()
	g, code := askStepUp(t, hh, admin.Token, "POST", path, walletBody(100), stepPass)
	fp.MustFire(t)
	t.Logf("الحقن: كتابةُ الإثبات سقطت ⇒ %d · إثباتٌ=%q", code, g)
	if g != "" {
		t.Error("**صدر إثباتٌ ولم يُكتب**")
	}

	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM step_up_grants WHERE user_id = $1::uuid`, admin.ID).Scan(&n)
	if n != 0 {
		t.Errorf("**بقي أثرُ إثباتٍ لم يُصدَر** — %d", n)
	}
	// **والفعلُ يبقى ممتنعاً.**
	res := withGrant(hh, admin.Token, "POST", path, walletBody(100), "")
	if res.Code != http.StatusForbidden {
		t.Errorf("**مضى الفعلُ بعد سقوط الإصدار** — %d", res.Code)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **وكلُّ فعلٍ في المعجم له مسارٌ في الموجِّه** — حارسٌ دائم
// ══════════════════════════════════════════════════════════════════════
//
// **ومعجمٌ فيه فعلٌ لا مسارَ له يُقرأ عقداً وهو أمنية** — والعكسُ
// أخطر: **مسارٌ شديدٌ لا يُدرَج يمضي بلا تأكيدٍ ولا يُكتشَف.**
func TestADG3_EverySensitiveActionHasARouteAndPolicy(t *testing.T) {
	hh := New(t)
	live := map[string][]string{}
	if err := chi.Walk(hh.API.RouterForWalk(),
		func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
			const prefix = "/api/v1/admin"
			if len(route) > len(prefix) && route[:len(prefix)] == prefix {
				live[method] = append(live[method], route[len(prefix):])
			}
			return nil
		}); err != nil {
		t.Fatalf("المشّاء: %v", err)
	}
	same := func(policy, actual string) bool {
		ps := strings.Split(strings.Trim(policy, "/"), "/")
		as := strings.Split(strings.Trim(actual, "/"), "/")
		if len(ps) != len(as) {
			return false
		}
		for i, seg := range ps {
			if strings.HasPrefix(seg, "{") {
				continue
			}
			if seg != as[i] {
				return false
			}
		}
		return true
	}
	for _, a := range authz.SensitiveActions() {
		hit := false
		for _, rt := range live[a.Method] {
			if same(a.Pattern, rt) {
				hit = true
			}
		}
		if !hit {
			t.Errorf("**فعلٌ حسّاسٌ لا مسارَ له**: %s %s (`ADG-3`)", a.Method, a.Pattern)
		}
		// **ولا فعلَ حسّاسٌ بلا قدرةٍ تحرسه** — الشرطان معاً.
		if _, ok := authz.LookupAdmin(a.Method, a.Pattern); !ok {
			if _, ex := authz.IsExempt(a.Pattern); !ex {
				t.Errorf("**فعلٌ حسّاسٌ بلا سياسةِ قدرة**: %s %s", a.Method, a.Pattern)
			}
		}
	}
	t.Logf("معجمُ الأفعال الشديدة=%d · بلا مسارٍ=0", authz.SensitiveCount())
}

// ══════════════════════════════════════════════════════════════════════
// **ومهلةُ الإثبات لا تتجاوز خمسَ دقائق**
// ══════════════════════════════════════════════════════════════════════
func TestADG3_TTLIsBounded(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	v := hh.NewUser("customer")

	g, _ := askStepUp(t, hh, admin.Token, "POST", walletPath(v.ID), walletBody(100), stepPass)
	if g == "" {
		t.Fatal("لم يصدر إثبات")
	}
	var life time.Duration
	var secs float64
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT extract(epoch FROM (expires_at - created_at))
		   FROM step_up_grants WHERE id = $1::uuid`, g).Scan(&secs); err != nil {
		t.Fatalf("قراءةُ المهلة: %v", err)
	}
	life = time.Duration(secs) * time.Second
	t.Logf("المهلةُ = %s", life)
	if life > 5*time.Minute {
		t.Errorf("**مهلةُ الإثبات %s تتجاوز خمسَ دقائق**", life)
	}
}
