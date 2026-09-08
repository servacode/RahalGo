package qa

import (
	"context"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **طبقاتُ الرصيد — والمحجوزُ ليس متاحاً** — `XG-12` · `AQ-3`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان يقع قبل
//
// **رصيدٌ مئةُ ألفٍ · طلبُ سحبٍ بمئةِ ألف ⇒ `201` والرصيدُ لم يُحجَز** ·
// **ثمّ أُنفق كلُّه ⇒ `200`** · **ثمّ قرارُ الدفع ⇒ `409`.**
//
// **ولا مالَ ضاع** — **لكنّ الماليّةَ توافق على سحبٍ يرتدّ**، والطلبُ
// يبقى معلَّقاً، **والمالُ الذي قرّرت عليه ذهب.**
//
// # والعقد
//
//	POSTED       ما في المحفظة فعلاً
//	RESERVED     محجوزٌ لعمليّةٍ جارية — **منه لا فوقه**
//	AVAILABLE    = POSTED − RESERVED — **مشتقٌّ لا مخزَّن**
//	OUTSTANDING  التزامٌ خارج المحفظة — `XG-31` بلا تبديل
//
// **ولا وحدةَ مالٍ تكون متاحةً ومحجوزةً معاً.**

// walletLayers الطبقاتُ الثلاثُ كما في القاعدة.
func walletLayers(t *testing.T, hh *Harness, uid string) (balance, reserved, available int64) {
	t.Helper()
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT balance  FROM wallets WHERE user_id = $1::uuid), 0),
		       COALESCE((SELECT reserved FROM wallets WHERE user_id = $1::uuid), 0)`,
		uid).Scan(&balance, &reserved); err != nil {
		t.Fatalf("الطبقات: %v", err)
	}
	return balance, reserved, balance - reserved
}

// fundWallet يشحن محفظةً عبر المسار الحقيقيّ.
func fundWallet(t *testing.T, hh *Harness, admin *User, uid string, amount int) {
	t.Helper()
	res := hh.POST("/api/v1/admin/users/"+uid+"/wallet", admin.Token,
		map[string]any{"amount": amount, "kind": "topup", "note": "XG-12"})
	if res.Code >= 400 {
		t.Fatalf("الشحن: %s", res)
	}
}

// askPayout يطلب سحباً ويردّ (المعرّف، رمزَ الردّ).
func askPayout(t *testing.T, hh *Harness, u *User, amount int) (string, int) {
	t.Helper()
	res := hh.POST("/api/v1/me/payouts", u.Token, map[string]any{"amount": amount})
	if res.Code != http.StatusCreated {
		return "", res.Code
	}
	id, _ := res.JSON()["id"].(string)
	return id, res.Code
}

// decidePayout قرارُ الماليّة.
func decidePayout(hh *Harness, admin *User, id, status string) Res {
	return hh.POST("/api/v1/admin/payouts/"+id+"/decide", admin.Token,
		map[string]any{"status": status, "decision": "XG-12"})
}

// spend ينفق من محفظةٍ عبر المسار الحقيقيّ.
func spend(hh *Harness, admin *User, uid string, amount int) Res {
	return hh.POST("/api/v1/admin/users/"+uid+"/wallet", admin.Token,
		map[string]any{"amount": amount, "kind": "adjustment", "debit": true, "note": "XG-12"})
}

// ══════════════════════════════════════════════════════════════════════
// **T1 · الطلبُ يحجز فوراً — والإنفاقُ يرى المتاح**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_T1_RequestReservesAndSpendSeesAvailable(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	rep := hh.NewUser("sales")
	fundWallet(t, hh, admin, rep.ID, 100000)

	id, code := askPayout(t, hh, rep, 70000)
	b, res, av := walletLayers(t, hh, rep.ID)
	t.Logf("T1: الطلبُ ⇒ %d · مُقيَّدٌ=%d · محجوزٌ=%d · متاحٌ=%d", code, b, res, av)
	if id == "" {
		t.Fatalf("**T1: تعذّر طلبُ السحب** — %d", code)
	}
	if b != 100000 || res != 70000 || av != 30000 {
		t.Fatalf("**T1: الطبقاتُ خاطئةٌ بعد الطلب** — %d/%d/%d", b, res, av)
	}

	// **وإنفاقُ ما يتجاوز المتاح يُردّ** — والقيدُ هو الحارس.
	over := spend(hh, admin, rep.ID, 40000)
	b2, res2, av2 := walletLayers(t, hh, rep.ID)
	t.Logf("T1: إنفاقُ 40000 (والمتاحُ 30000) ⇒ %d · %d/%d/%d", over.Code, b2, res2, av2)
	if over.Code < 400 {
		t.Errorf("**T1: أُنفق مالٌ محجوز** — %d", over.Code)
	}

	// **وإنفاقُ المتاح يمضي.**
	ok := spend(hh, admin, rep.ID, 30000)
	b3, res3, av3 := walletLayers(t, hh, rep.ID)
	t.Logf("T1: إنفاقُ المتاح ⇒ %d · %d/%d/%d", ok.Code, b3, res3, av3)
	if ok.Code >= 400 {
		t.Errorf("**T1: رُدّ إنفاقُ المتاح** — %s", ok)
	}
	if av3 != 0 || res3 != 70000 {
		t.Errorf("**T1: الطبقاتُ بعد الإنفاق** — %d/%d/%d", b3, res3, av3)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T2 · `paid` يخصم ويفكّ — ولا يرتدّ لأنّ المالَ أُنفق**
// ══════════════════════════════════════════════════════════════════════
//
// **وهذا هو سببُ الحجز**: **القرارُ لا يفشل بعد أن قُرّر.**
func TestXG12_T2_PaidDebitsAndReleases(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	rep := hh.NewUser("sales")
	fundWallet(t, hh, admin, rep.ID, 100000)

	id, _ := askPayout(t, hh, rep, 70000)
	if id == "" {
		t.Fatal("لم يُنشأ الطلب")
	}
	// **ويُنفَق كلُّ المتاح** — والقرارُ يبقى قادراً.
	if res := spend(hh, admin, rep.ID, 30000); res.Code >= 400 {
		t.Fatalf("إنفاقُ المتاح: %s", res)
	}

	dec := decidePayout(hh, admin, id, "paid")
	b, res, av := walletLayers(t, hh, rep.ID)
	var st string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM payout_requests WHERE id = $1::uuid`, id).Scan(&st)
	t.Logf("T2: القرارُ ⇒ %d · الحالُ=%q · %d/%d/%d", dec.Code, st, b, res, av)

	if dec.Code >= 400 {
		t.Fatalf("**T2: رُدّ قرارُ سحبٍ محجوزٍ ماله** — %s · "+
			"**وذاك ما بُني الحجزُ لمنعه.**", dec)
	}
	if b != 0 || res != 0 || av != 0 {
		t.Errorf("**T2: الطبقاتُ بعد الدفع** — %d/%d/%d", b, res, av)
	}
	// **وقيدُ الدفتر واحد.**
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'payout'`,
		id).Scan(&n)
	if n != 1 {
		t.Errorf("**T2: قيودُ الدفع %d** — والواحدُ هو العقد", n)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T3 · `rejected` و`failed` يفكّان بلا خصم**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_T3_RejectAndFailReleaseWithoutDebit(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")

	for _, st := range []string{"rejected", "failed"} {
		u := hh.NewUser("sales")
		fundWallet(t, hh, admin, u.ID, 50000)
		id, _ := askPayout(t, hh, u, 50000)
		if id == "" {
			t.Fatal("لم يُنشأ الطلب")
		}
		dec := decidePayout(hh, admin, id, st)
		b, res, av := walletLayers(t, hh, u.ID)
		t.Logf("T3: %-9s ⇒ %d · %d/%d/%d", st, dec.Code, b, res, av)
		if dec.Code >= 400 {
			t.Errorf("**T3: رُدّ قرارُ %s** — %s", st, dec)
			continue
		}
		if b != 50000 || res != 0 || av != 50000 {
			t.Errorf("**T3: %s بدّل الرصيدَ أو أبقى حجزاً** — %d/%d/%d", st, b, res, av)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T4 · `processing` يُبقي الحجزَ ولا يخصم — ومجهولُ النتيجة يبقى فيه**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_T4_ProcessingHoldsWithoutDebit(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 80000)

	id, _ := askPayout(t, hh, u, 80000)
	dec := decidePayout(hh, admin, id, "processing")
	b, res, av := walletLayers(t, hh, u.ID)
	t.Logf("T4: `processing` ⇒ %d · %d/%d/%d", dec.Code, b, res, av)
	if dec.Code >= 400 {
		t.Fatalf("**T4: رُدّ الانتقالُ إلى `processing`** — %s", dec)
	}
	if b != 80000 || res != 80000 || av != 0 {
		t.Errorf("**T4: `processing` بدّل مالاً** — %d/%d/%d", b, res, av)
	}
	// **ولا يُنفَق المحجوزُ وهو قيدُ التنفيذ.**
	if s := spend(hh, admin, u.ID, 1); s.Code < 400 {
		t.Errorf("**T4: أُنفق مالٌ قيدَ الصرف** — %d", s.Code)
	}
	// **ثمّ يثبت النجاحُ فيُخصَم.**
	paid := decidePayout(hh, admin, id, "paid")
	b2, res2, _ := walletLayers(t, hh, u.ID)
	t.Logf("T4: `processing`→`paid` ⇒ %d · %d/%d", paid.Code, b2, res2)
	if paid.Code >= 400 || b2 != 0 || res2 != 0 {
		t.Errorf("**T4: لم يُختَم الصرف** — %d · %d/%d", paid.Code, b2, res2)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **T5 · `reversed` قيدٌ مقابلٌ لا محوٌ لتاريخ**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_T5_ReversedCompensates(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 60000)

	id, _ := askPayout(t, hh, u, 60000)
	if res := decidePayout(hh, admin, id, "paid"); res.Code >= 400 {
		t.Fatalf("الدفع: %s", res)
	}
	rev := decidePayout(hh, admin, id, "reversed")
	b, res, av := walletLayers(t, hh, u.ID)
	var debits, credits int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FILTER (WHERE kind = 'payout'),
		        count(*) FILTER (WHERE kind = 'refund')
		   FROM wallet_transactions WHERE ref = $1`, id).Scan(&debits, &credits)
	t.Logf("T5: الارتدادُ ⇒ %d · %d/%d/%d · خصمٌ=%d · ردٌّ=%d",
		rev.Code, b, res, av, debits, credits)

	if rev.Code >= 400 {
		t.Fatalf("**T5: رُدّ الارتداد** — %s", rev)
	}
	if b != 60000 || res != 0 {
		t.Errorf("**T5: المالُ لم يعد** — %d/%d", b, res)
	}
	if debits != 1 || credits != 1 {
		t.Errorf("**T5: التاريخُ مُحي أو ازدوج** — خصمٌ=%d · ردٌّ=%d", debits, credits)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1 · طلبان متزامنان لا يحجزان فوق المتاح**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_C1_ConcurrentRequestsCannotOverReserve(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 100000)

	act := func(name string) Actor {
		return Actor{Name: name, Do: func(context.Context) any {
			return hh.POST("/api/v1/me/payouts", u.Token, map[string]any{"amount": 70000})
		}}
	}
	r := Race(t, 0, act("أ"), act("ب"))
	b, res, av := walletLayers(t, hh, u.ID)
	t.Logf("C1: نجح %d من 2 · تداخلٌ=%d · %d/%d/%d",
		r.CountOK(), r.Probe.Max(), b, res, av)

	if res > b {
		t.Errorf("**C1: محجوزٌ يتجاوز المُقيَّد** — %d > %d", res, b)
	}
	if r.CountOK() != 1 {
		t.Errorf("**C1: نجح %d طلباً** — **والمتاحُ يحتمل واحداً**", r.CountOK())
	}
	if r.Probe.Max() < 2 {
		t.Errorf("**C1: تداخلٌ مقيسٌ %d** — والسيناريو يشترط تزامناً", r.Probe.Max())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C2 · حجزٌ يسابق إنفاقاً — ولا يُنفَق المحجوز مرّتين**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_C2_ReserveVsSpend(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 100000)

	r := Race(t, 0,
		Actor{Name: "سحب", Do: func(context.Context) any {
			return hh.POST("/api/v1/me/payouts", u.Token, map[string]any{"amount": 100000})
		}},
		Actor{Name: "إنفاق", Do: func(context.Context) any {
			return spend(hh, admin, u.ID, 100000)
		}},
	)
	b, res, av := walletLayers(t, hh, u.ID)
	t.Logf("C2: نجح %d من 2 · تداخلٌ=%d · %d/%d/%d",
		r.CountOK(), r.Probe.Max(), b, res, av)
	if av < 0 || res > b {
		t.Errorf("**C2: المتاحُ سالبٌ أو الحجزُ فوق المُقيَّد** — %d/%d/%d", b, res, av)
	}
	if r.CountOK() != 1 {
		t.Errorf("**C2: نجح %d** — **ومئةُ ألفٍ تكفي واحداً**", r.CountOK())
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C4 · فكُّ الحجز مرّةً واحدة · C5 · لا خصمَ مزدوجٌ لقرارٍ يُكرَّر**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_C4C5_ReleaseOnceAndNoDoubleDebit(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")

	// ── C4 · رفضٌ يُكرَّر ────────────────────────────────────────
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 40000)
	id, _ := askPayout(t, hh, u, 40000)
	first := decidePayout(hh, admin, id, "rejected")
	second := decidePayout(hh, admin, id, "rejected")
	b, res, _ := walletLayers(t, hh, u.ID)
	t.Logf("C4: الأوّلُ %d · الثاني %d · %d/%d", first.Code, second.Code, b, res)
	if second.Code < 400 {
		t.Errorf("**C4: رفضٌ ثانٍ مضى** — %d", second.Code)
	}
	if b != 40000 || res != 0 {
		t.Errorf("**C4: فكٌّ مزدوجٌ أو ناقص** — %d/%d", b, res)
	}

	// ── C5 · دفعان متزامنان ────────────────────────────────────
	v := hh.NewUser("sales")
	fundWallet(t, hh, admin, v.ID, 50000)
	pid, _ := askPayout(t, hh, v, 50000)
	pay := func(name string) Actor {
		return Actor{Name: name, Do: func(context.Context) any {
			return decidePayout(hh, admin, pid, "paid")
		}}
	}
	r := Race(t, 0, pay("أ"), pay("ب"))
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'payout'`,
		pid).Scan(&n)
	vb, vres, _ := walletLayers(t, hh, v.ID)
	t.Logf("C5: نجح %d من 2 · قيودٌ=%d · %d/%d", r.CountOK(), n, vb, vres)
	if n != 1 {
		t.Errorf("**C5: قيودُ الدفع %d** — والخصمُ مرّةً", n)
	}
	if vb != 0 || vres != 0 {
		t.Errorf("**C5: الطبقاتُ بعد الدفع** — %d/%d", vb, vres)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F1 · F2 · الطلبُ وحجزُه يقعان معاً أو لا يقع أحدُهما**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_F1F2_CreationIsOneUnit(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")

	// ── F1 · الطلبُ يُكتب والحجزُ يسقط ─────────────────────────
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 30000)
	fp := hh.ArmAny("XG12/reserve", "wallets", "UPDATE")
	res := hh.POST("/api/v1/me/payouts", u.Token, map[string]any{"amount": 30000})
	fp.Disarm()
	var rows int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM payout_requests WHERE user_id = $1::uuid`, u.ID).Scan(&rows)
	b, rsv, _ := walletLayers(t, hh, u.ID)
	t.Logf("F1: سقوطُ الحجز ⇒ %d · طلباتٌ=%d · %d/%d", res.Code, rows, b, rsv)
	if res.Code < 400 {
		t.Errorf("**F1: مضى الطلبُ والحجزُ ساقط** — %d", res.Code)
	}
	if rows != 0 || rsv != 0 {
		t.Errorf("**F1: بقي طلبٌ بلا حجز** — طلباتٌ=%d · محجوزٌ=%d", rows, rsv)
	}

	// ── F2 · الحجزُ يُكتب والطلبُ يسقط ─────────────────────────
	v := hh.NewUser("sales")
	fundWallet(t, hh, admin, v.ID, 30000)
	fp2 := hh.ArmAny("XG12/request", "payout_requests", "INSERT")
	res2 := hh.POST("/api/v1/me/payouts", v.Token, map[string]any{"amount": 30000})
	fp2.Disarm()
	var rows2 int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM payout_requests WHERE user_id = $1::uuid`, v.ID).Scan(&rows2)
	b2, rsv2, _ := walletLayers(t, hh, v.ID)
	t.Logf("F2: سقوطُ الطلب ⇒ %d · طلباتٌ=%d · %d/%d", res2.Code, rows2, b2, rsv2)
	if res2.Code < 400 {
		t.Errorf("**F2: مضى الحجزُ والطلبُ ساقط** — %d", res2.Code)
	}
	if rows2 != 0 || rsv2 != 0 {
		t.Errorf("**F2: بقي حجزٌ بلا طلب** — طلباتٌ=%d · محجوزٌ=%d", rows2, rsv2)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F3 · F5 · حالٌ لا تُثبَّت إن سقط ما يوجبه معناها**
// ══════════════════════════════════════════════════════════════════════
func TestXG12_F3F5_TerminalStateNeedsItsMoneyTruth(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")

	// ── F3 · خصمُ الدفع يسقط ⇒ لا يصير `paid` ──────────────────
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 25000)
	id, _ := askPayout(t, hh, u, 25000)
	fp := hh.ArmAny("XG12/paid-debit", "wallet_transactions", "INSERT")
	res := decidePayout(hh, admin, id, "paid")
	fp.MustFire(t)
	var st string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM payout_requests WHERE id = $1::uuid`, id).Scan(&st)
	b, rsv, _ := walletLayers(t, hh, u.ID)
	t.Logf("F3: سقوطُ الخصم ⇒ %d · الحالُ=%q · %d/%d", res.Code, st, b, rsv)
	if st == "paid" {
		t.Errorf("**F3: صار `paid` بلا خصم**")
	}
	if b != 25000 || rsv != 25000 {
		t.Errorf("**F3: الحجزُ لم يبقَ سليماً** — %d/%d", b, rsv)
	}

	// ── F5 · فكُّ الرفض يسقط ⇒ لا يصير `rejected` ──────────────
	v := hh.NewUser("sales")
	fundWallet(t, hh, admin, v.ID, 25000)
	vid, _ := askPayout(t, hh, v, 25000)
	fp2 := hh.ArmAny("XG12/release", "wallets", "UPDATE")
	res2 := decidePayout(hh, admin, vid, "rejected")
	fp2.Disarm()
	var st2 string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM payout_requests WHERE id = $1::uuid`, vid).Scan(&st2)
	b2, rsv2, _ := walletLayers(t, hh, v.ID)
	t.Logf("F5: سقوطُ الفكّ ⇒ %d · الحالُ=%q · %d/%d", res2.Code, st2, b2, rsv2)
	if st2 == "rejected" && rsv2 != 0 {
		t.Errorf("**F5: صار `rejected` وبقي الحجز** — %d", rsv2)
	}
	if st2 != "rejected" && rsv2 != 25000 {
		t.Errorf("**F5: لم يُثبَّت الرفضُ ولا بقي الحجزُ سليماً** — %d", rsv2)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F4 · سقوطُ الأثر يُسقط الصرفَ كلَّه** — `AQ-4` محفوظ
// ══════════════════════════════════════════════════════════════════════
//
// **و`finance.payout_decide` من الصنف `A`** — **فعلٌ حسّاسٌ لا ينجح
// بلا أثر.** **والحجزُ يبقى سليماً** فيُعاد القرار.
func TestXG12_F4_AuditFailureRollsBackPayout(t *testing.T) {
	hh := New(t)
	treasury(t, hh)
	admin := hh.NewUser("admin")
	u := hh.NewUser("sales")
	fundWallet(t, hh, admin, u.ID, 35000)
	id, _ := askPayout(t, hh, u, 35000)

	fp := hh.ArmAny("XG12/payout-audit", "audit_log", "INSERT")
	res := decidePayout(hh, admin, id, "paid")
	fp.MustFire(t)

	var st string
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT status FROM payout_requests WHERE id = $1::uuid`, id).Scan(&st)
	b, rsv, _ := walletLayers(t, hh, u.ID)
	var n int
	_ = hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'payout'`,
		id).Scan(&n)
	t.Logf("F4: سقوطُ الأثر ⇒ %d · الحالُ=%q · %d/%d · قيودٌ=%d",
		res.Code, st, b, rsv, n)

	if st == "paid" || n != 0 {
		t.Errorf("**F4: خرج مالٌ بلا أثر** — الحالُ=%q · قيودٌ=%d", st, n)
	}
	if b != 35000 || rsv != 35000 {
		t.Errorf("**F4: الحجزُ لم يبقَ سليماً** — %d/%d", b, rsv)
	}
}
