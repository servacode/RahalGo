package qa

// ══════════════════════════════════════════════════════════════════════
// **شحن محفظة المتجر بشام كاش — ورصيد شحن ما بينسحب** — قرار المالك ٢٠٢٦-١١-٢٦
// ══════════════════════════════════════════════════════════════════════
//
// على المحرّك الحقيقيّ: المتجر بيشحن وبيتوافق عليه متل الزبون، بس المبلغ بيدخل «رصيد شحن»
// (`wallets.topup_locked`) — طلب السحب ما بيقدر يطلع منه، وأجرة التوصيلة والدين بياكلوا منه
// أوّلاً، والاسترجاع بيرجّعله. والمندوب والسائق لسّا ممنوعين (٤٠٣)، والزبون متل ما هو.

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

var scmChecks = []string{"FI-02", "FI-03", "FI-04", "FI-05", "FI-11", "FI-16"}

// scmTopup **طلب شحن شام كاش وموافقته** — ويرجّع معرّف الطلب.
func scmTopup(t *testing.T, h *Harness, tok string, fin *User, amount int64, body map[string]any) string {
	t.Helper()
	b := map[string]any{"amount": amount, "txn_ref": "SCM" + uniq("m")}
	for k, v := range body {
		b[k] = v
	}
	r := h.POSTKey("/api/v1/my/shamcash/topups", tok, uniq("scm"), b)
	if r.Code != http.StatusCreated {
		t.Fatalf("طلب الشحن: %s", r)
	}
	id := scID(t, r)
	if ok := h.POST("/api/v1/admin/shamcash-topups/"+id+"/approve", fin.Token, map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	return id
}

// scmLayers **رصيد الشحن وما بينسحب** كما بيشوفهم صاحب المحفظة (`GET /my/wallet`).
func scmLayers(t *testing.T, h *Harness, tok string) (balance, locked, withdrawable, available int64) {
	t.Helper()
	r := h.GET("/api/v1/my/wallet", tok)
	if r.Code != 200 {
		t.Fatalf("المحفظة: %s", r)
	}
	j := r.JSON()
	num := func(k string) int64 { v, _ := j[k].(float64); return int64(v) }
	return num("balance"), num("topup_locked"), num("withdrawable"), num("available")
}

func scmRoleOf(t *testing.T, h *Harness, id string) string {
	t.Helper()
	var role string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT role FROM shamcash_topups WHERE id = $1::uuid`, id).Scan(&role); err != nil {
		t.Fatal(err)
	}
	return role
}

// ── شحن المتجر ما بينسحب، وما فوقه بينسحب عادي ──────────────────────────
func TestShamcashMerchant_LockedNotWithdrawable(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	h.Setting("payouts.min_amount", "1")
	base := financialBaseline(t, h, scmChecks...)
	fin := h.NewUser("admin")
	f := h.Factory()
	m := f.Merchant()
	tok := h.TokenFor(m.Owner.ID, "merchant")

	id := scmTopup(t, h, tok, fin, 2000, nil)
	if role := scmRoleOf(t, h, id); role != "merchant" {
		t.Errorf("دور الطلب %q — يُنتظر merchant", role)
	}
	// **واللوحة بتفرز المتجر** — `?role=merchant` وفيه اسم المتجر.
	list := h.GET("/api/v1/admin/shamcash-topups?role=merchant&q="+m.Owner.Phone, fin.Token)
	rows, _ := list.JSON()["requests"].([]any)
	if len(rows) != 1 || rows[0].(map[string]any)["role"] != "merchant" || rows[0].(map[string]any)["store_name"] != m.Name {
		t.Errorf("قائمة اللوحة للمتجر: %s", list)
	}
	if r := h.GET("/api/v1/admin/shamcash-topups?role=driver", fin.Token); r.Code != http.StatusBadRequest {
		t.Errorf("دور غريب: %s — يُنتظر 400", r)
	}

	bal, locked, wd, av := scmLayers(t, h, tok)
	if bal != 2000 || locked != 2000 || wd != 0 || av != 2000 {
		t.Fatalf("بعد الشحن: رصيد %d شحن %d بينسحب %d متاح %d — يُنتظر 2000/2000/0/2000", bal, locked, wd, av)
	}
	// **ولا ليرة منه بتنسحب.**
	if _, code := askPayout(t, h, &User{ID: m.Owner.ID, Token: tok}, 500); code != http.StatusConflict {
		t.Fatalf("**سحب من رصيد الشحن مرّ**: %d — يُنتظر 409", code)
	}
	// **وأرباح المتجر بتنسحب عادي** — والحدّ هو ما فوق رصيد الشحن.
	f.Credit(m.Owner.ID, 1000, "topup")
	if _, _, wd, _ = scmLayers(t, h, tok); wd != 1000 {
		t.Fatalf("بعد شحن نقديّ بينسحب %d — يُنتظر 1000", wd)
	}
	if _, code := askPayout(t, h, &User{ID: m.Owner.ID, Token: tok}, 1001); code != http.StatusConflict {
		t.Errorf("سحب ١٠٠١ مرّ: %d — يُنتظر 409", code)
	}
	pid, code := askPayout(t, h, &User{ID: m.Owner.ID, Token: tok}, 1000)
	if code != http.StatusCreated {
		t.Fatalf("سحب ١٠٠٠: %d — يُنتظر 201", code)
	}
	if r := decidePayout(h, fin, pid, "paid"); r.Code != 200 {
		t.Fatalf("صرف السحب: %s", r)
	}
	if bal, locked, wd, _ = scmLayers(t, h, tok); bal != 2000 || locked != 2000 || wd != 0 {
		t.Errorf("بعد الصرف: رصيد %d شحن %d بينسحب %d — يُنتظر 2000/2000/0", bal, locked, wd)
	}
	assertNewViolations(t, h, base, scmChecks...)
}

// ── أجرة التوصيلة بتاكل من رصيد الشحن أوّلاً، والإلغاء بيرجّعها إله ───────
func TestShamcashMerchant_DeliveryFeeConsumesLocked(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ SCM-02")
	scPrep(t, h)
	base := financialBaseline(t, h, scmChecks...)
	fin := h.NewUser("admin")
	owner := m.fx.M.Owner.ID

	scmTopup(t, h, m.fx.Tok, fin, 1000, nil)
	m.f.Credit(owner, 1000, "topup") // أرباح/نقد بينسحب
	id := m.create(t, h, "merchant", uniq("scm02"))
	m.accept(t, h, id)
	bal, locked, wd, _ := scmLayers(t, h, m.fx.Tok)
	if bal != 1500 || locked != 500 || wd != 1000 {
		t.Fatalf("بعد الأجرة: رصيد %d شحن %d بينسحب %d — يُنتظر 1500/500/1000 (الأجرة من رصيد الشحن)", bal, locked, wd)
	}
	if c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	bal, locked, wd, _ = scmLayers(t, h, m.fx.Tok)
	if bal != 2000 || locked != 1000 || wd != 1000 {
		t.Fatalf("**بعد الإلغاء رصيد %d شحن %d بينسحب %d — يُنتظر 2000/1000/1000** (الاسترجاع لازم يرجع لرصيد الشحن)", bal, locked, wd)
	}
	assertNewViolations(t, h, base, scmChecks...)
}

// ── الدين بيتسدّ من الشحن أوّلاً ─────────────────────────────────────────
func TestShamcashMerchant_DebtConsumesLocked(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	base := financialBaseline(t, h, scmChecks...)
	fin := h.NewUser("admin")
	m := h.Factory().Merchant()
	tok := h.TokenFor(m.Owner.ID, "merchant")
	if _, err := obligations.Create(ctxBG(), h.Pool, obligations.PartyMerchant, m.ID, 300,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatal(err)
	}
	scmTopup(t, h, tok, fin, 1000, nil)
	bal, locked, wd, _ := scmLayers(t, h, tok)
	if bal != 700 || locked != 700 || wd != 0 {
		t.Fatalf("بعد سدّ الدين: رصيد %d شحن %d بينسحب %d — يُنتظر 700/700/0", bal, locked, wd)
	}
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1::uuid`, m.ID).Scan(&debt)
	if debt != 0 {
		t.Errorf("الدين باقي %d — يُنتظر 0", debt)
	}
	assertNewViolations(t, h, base, scmChecks...)
}

// ── حساب زبون ومتجر سوا: الموقع بيبعت `as` — والزبون بلا `as` زبون ────────
func TestShamcashMerchant_RoleChoiceAndCustomerUnchanged(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	fin := h.NewUser("admin")
	both := h.NewUser("customer")
	if _, err := h.Pool.Exec(ctxBG(),
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'merchant') ON CONFLICT DO NOTHING`, both.ID); err != nil {
		t.Fatal(err)
	}
	tok := h.TokenFor(both.ID, "customer", "merchant")
	if role := scmRoleOf(t, h, scmTopup(t, h, tok, fin, 600, nil)); role != "customer" {
		t.Errorf("بلا as: %q — يُنتظر customer", role)
	}
	if role := scmRoleOf(t, h, scmTopup(t, h, tok, fin, 600, map[string]any{"as": "merchant"})); role != "merchant" {
		t.Errorf("as=merchant: %q — يُنتظر merchant", role)
	}
	if r := h.POSTKey("/api/v1/my/shamcash/topups", tok, uniq("scm"),
		map[string]any{"amount": 600, "txn_ref": "BAD" + uniq("m"), "as": "driver"}); r.Code != http.StatusBadRequest {
		t.Errorf("as=driver: %s — يُنتظر 400", r)
	}

	// **الزبون العادي متل ما هو**: رصيده المتاح كلّه، وما في سحب أصلاً.
	cust := h.Customer()
	if role := scmRoleOf(t, h, scmTopup(t, h, cust.Token, fin, 900, nil)); role != "customer" {
		t.Errorf("زبون: %q", role)
	}
	bal, _, _, av := scmLayers(t, h, cust.Token)
	if bal != 900 || av != 900 {
		t.Errorf("رصيد الزبون %d ومتاحه %d — يُنتظر 900/900", bal, av)
	}
	// **وزبون بس ما بيقدر يطلب بصفة متجر.**
	if r := h.POSTKey("/api/v1/my/shamcash/topups", cust.Token, uniq("scm"),
		map[string]any{"amount": 600, "txn_ref": "CUS" + uniq("m"), "as": "merchant"}); r.Code != http.StatusForbidden {
		t.Errorf("زبون بصفة متجر: %s — يُنتظر 403", r)
	}
}

// ── المندوب والسائق ممنوعين — متل قبل ────────────────────────────────────
func TestShamcashMerchant_RepAndDriverForbidden(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	for _, role := range []string{"sales", "driver"} {
		u := h.NewUser(role)
		if r := scRequest(h, u, 1000, "NO"+uniq("x")); r.Code != http.StatusForbidden {
			t.Errorf("%s: %s — يُنتظر 403", role, r)
		}
		if r := h.POSTKey("/api/v1/my/shamcash/topups", u.Token, uniq("scm"),
			map[string]any{"amount": 1000, "txn_ref": "NO" + uniq("y"), "as": "merchant"}); r.Code != http.StatusForbidden {
			t.Errorf("%s بصفة متجر: %s — يُنتظر 403", role, r)
		}
	}
}
