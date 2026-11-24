package qa

// ══════════════════════════════════════════════════════════════════════
// **شحن المحفظة بشام كاش + «شام كاش عند الاستلام»** — قرار المالك ٢٠٢٦-١١-٢٥
// ══════════════════════════════════════════════════════════════════════
//
// على المحرّك الحقيقيّ: الحدّ الأدنى، ورقم العملية المكرّر، والموافقة مرّة وحدة (قيد واحد
// وحركة وحدة — FI-05.l)، والرفض بسببه، و٤٠٣ لتوكن زبون عاللوحة، وخيار الدفع عند الاستلام.

import (
	"net/http"
	"testing"
)

var scChecks = []string{"FI-02", "FI-03", "FI-04", "FI-05"}

func scPrep(t *testing.T, h *Harness) {
	t.Helper()
	treasury(t, h)
	h.Setting("shamcash.topup_enabled", "true")
	h.Setting("shamcash.min_topup", "500")
	h.Setting("shamcash.account_number", `"QA-0000-TEST"`)
	h.Setting("shamcash.account_name", `"QA"`)
}

func scRequest(h *Harness, cust *User, amount any, txn string) Res {
	return h.POSTKey("/api/v1/my/shamcash/topups", cust.Token, uniq("sct"),
		map[string]any{"amount": amount, "txn_ref": txn})
}

func scCounts(t *testing.T, h *Harness, id string) (txs, movs int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT count(*) FROM wallet_transactions WHERE kind = 'topup' AND ref = $1),
		       (SELECT count(*) FROM shamcash_movements WHERE kind = 'customer_topup' AND topup_id = $1::uuid)`,
		id).Scan(&txs, &movs); err != nil {
		t.Fatalf("العدّ: %v", err)
	}
	return
}

func scID(t *testing.T, r Res) string {
	t.Helper()
	req, _ := r.JSON()["request"].(map[string]any)
	id, _ := req["id"].(string)
	if id == "" {
		t.Fatalf("لا معرّف للطلب: %s", r)
	}
	return id
}

// ── أقل من ٥٠٠ مرفوض، والمطفي مرفوض، والكسور مرفوضة ─────────────────────
func TestShamcash_TopupMinAndDisabled(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	cust := h.Customer()
	if r := scRequest(h, cust, 499, "MIN"+uniq("a")); r.Code != http.StatusBadRequest || r.Err() != "shamcash_topup_min" {
		t.Errorf("٤٩٩: %s — يُنتظر 400 shamcash_topup_min", r)
	}
	if r := scRequest(h, cust, 500.5, "FRC"+uniq("a")); r.Code != http.StatusBadRequest {
		t.Errorf("كسر: %s — يُنتظر 400", r)
	}
	if r := scRequest(h, cust, 500, "OK"+uniq("a")); r.Code != http.StatusCreated {
		t.Errorf("٥٠٠: %s — يُنتظر 201", r)
	}
	h.Setting("shamcash.topup_enabled", "false")
	if r := scRequest(h, cust, 1000, "OFF"+uniq("a")); r.Code != http.StatusConflict || r.Err() != "shamcash_topup_disabled" {
		t.Errorf("مطفي: %s — يُنتظر 409 shamcash_topup_disabled", r)
	}
	// **والكابتن ما بيشحن بشام كاش** — للزبون بس.
	drv := h.NewUser("driver")
	h.Setting("shamcash.topup_enabled", "true")
	if r := scRequest(h, drv, 1000, "DRV"+uniq("a")); r.Code != http.StatusForbidden {
		t.Errorf("سائق: %s — يُنتظر 403", r)
	}
}

// ── رقم العملية ما بيتكرّر (حتى بين زبونين) إلا بعد الرفض ──────────────────
func TestShamcash_DuplicateTxnRef(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	fin := h.NewUser("admin")
	a, b := h.Customer(), h.Customer()
	txn := "DUP" + uniq("d")
	first := scRequest(h, a, 1000, txn)
	if first.Code != http.StatusCreated {
		t.Fatalf("الأول: %s", first)
	}
	if r := scRequest(h, b, 1000, " "+txn+" "); r.Code != http.StatusConflict || r.Err() != "shamcash_txn_duplicate" {
		t.Errorf("مكرّر: %s — يُنتظر 409 shamcash_txn_duplicate", r)
	}
	// **بعد الرفض بينكتب من جديد** — وبيتعلّم «انرفض قبل».
	id := scID(t, first)
	if r := h.POST("/api/v1/admin/shamcash-topups/"+id+"/reject", fin.Token, map[string]any{"reason": "ما وصل"}); r.Code != 200 {
		t.Fatalf("الرفض: %s", r)
	}
	again := scRequest(h, b, 1000, txn)
	if again.Code != http.StatusCreated {
		t.Fatalf("بعد الرفض: %s — يُنتظر 201", again)
	}
	list := h.GET("/api/v1/admin/shamcash-topups?status=pending&q="+txn, fin.Token)
	rows, _ := list.JSON()["requests"].([]any)
	if len(rows) != 1 {
		t.Fatalf("القائمة: %s", list)
	}
	if rb, _ := rows[0].(map[string]any)["rejected_before"].(bool); !rb {
		t.Errorf("ما تعلّم «انرفض قبل»: %s", list)
	}
}

// ── «تم، اشحن» مرّة وحدة: قيد واحد وحركة وحدة، والتانية ولا شي ──────────
func TestShamcash_ApproveIdempotent(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	base := financialBaseline(t, h, scChecks...)
	fin := h.NewUser("admin")
	cust := h.Customer()
	r := scRequest(h, cust, 2500, "APP"+uniq("p"))
	if r.Code != http.StatusCreated {
		t.Fatalf("الطلب: %s", r)
	}
	id := scID(t, r)
	if bal := h.Factory().Balance(cust.ID); bal != 0 {
		t.Fatalf("**الطلب المعلّق حرّك المال**: %d", bal)
	}
	for i := 0; i < 3; i++ {
		ok := h.POST("/api/v1/admin/shamcash-topups/"+id+"/approve", fin.Token, map[string]any{})
		if ok.Code != 200 {
			t.Fatalf("الموافقة %d: %s", i, ok)
		}
		if already, _ := ok.JSON()["already"].(bool); already != (i > 0) {
			t.Errorf("الموافقة %d: already=%v", i, already)
		}
	}
	if bal := h.Factory().Balance(cust.ID); bal != 2500 {
		t.Errorf("الرصيد %d — يُنتظر 2500", bal)
	}
	if txs, movs := scCounts(t, h, id); txs != 1 || movs != 1 {
		t.Errorf("قيود %d وحركات %d — يُنتظر واحد وواحد", txs, movs)
	}
	// **والمقبول ما بينرفض.**
	if r := h.POST("/api/v1/admin/shamcash-topups/"+id+"/reject", fin.Token, map[string]any{"reason": "x"}); r.Code != http.StatusConflict {
		t.Errorf("رفض بعد الموافقة: %s — يُنتظر 409", r)
	}
	// **والسجلّ بيعدّ الشحن بالرصيد.**
	sum := h.GET("/api/v1/admin/shamcash/summary", fin.Token)
	if sum.Code != 200 {
		t.Fatalf("الملخّص: %s", sum)
	}
	assertNewViolations(t, h, base, scChecks...)
}

// ── «مرفوض» بسبب إلزاميّ، وما بيحرّك مال ─────────────────────────────────
func TestShamcash_RejectFlow(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	base := financialBaseline(t, h, scChecks...)
	fin := h.NewUser("admin")
	cust := h.Customer()
	id := scID(t, scRequest(h, cust, 800, "REJ"+uniq("r")))
	if r := h.POST("/api/v1/admin/shamcash-topups/"+id+"/reject", fin.Token, map[string]any{"reason": "  "}); r.Code != http.StatusBadRequest {
		t.Errorf("بلا سبب: %s — يُنتظر 400", r)
	}
	r := h.POST("/api/v1/admin/shamcash-topups/"+id+"/reject", fin.Token, map[string]any{"reason": "المبلغ ما وصل"})
	if r.Code != 200 {
		t.Fatalf("الرفض: %s", r)
	}
	if again := h.POST("/api/v1/admin/shamcash-topups/"+id+"/reject", fin.Token, map[string]any{"reason": "ثاني"}); again.Code != 200 {
		t.Errorf("رفض ثاني: %s — يُنتظر 200 بلا أثر", again)
	}
	if ok := h.POST("/api/v1/admin/shamcash-topups/"+id+"/approve", fin.Token, map[string]any{}); ok.Code != http.StatusConflict {
		t.Errorf("موافقة بعد الرفض: %s — يُنتظر 409", ok)
	}
	if bal := h.Factory().Balance(cust.ID); bal != 0 {
		t.Errorf("المرفوض حرّك المال: %d", bal)
	}
	if txs, movs := scCounts(t, h, id); txs != 0 || movs != 0 {
		t.Errorf("المرفوض إلو أثر: قيود %d حركات %d", txs, movs)
	}
	mine := h.GET("/api/v1/my/shamcash", cust.Token)
	reqs, _ := mine.JSON()["requests"].([]any)
	if len(reqs) != 1 || reqs[0].(map[string]any)["status"] != "rejected" ||
		reqs[0].(map[string]any)["reject_reason"] != "المبلغ ما وصل" {
		t.Errorf("الزبون ما شاف الرفض وسببه: %s", mine)
	}
	assertNewViolations(t, h, base, scChecks...)
}

// ── توكن زبون عاللوحة ⇒ ٤٠٣ ───────────────────────────────────────────────
func TestShamcash_CustomerTokenForbiddenOnAdmin(t *testing.T) {
	h := New(t)
	scPrep(t, h)
	cust := h.Customer()
	id := scID(t, scRequest(h, cust, 1000, "SEC"+uniq("s")))
	for _, c := range []struct{ m, p string }{
		{"GET", "/api/v1/admin/shamcash-topups"},
		{"POST", "/api/v1/admin/shamcash-topups/" + id + "/approve"},
		{"POST", "/api/v1/admin/shamcash-topups/" + id + "/reject"},
	} {
		if r := h.Call(c.m, c.p, cust.Token, map[string]any{"reason": "x"}, nil); r.Code != http.StatusForbidden {
			t.Errorf("%s %s بتوكن زبون: %s — يُنتظر 403", c.m, c.p, r)
		}
	}
	if bal := h.Factory().Balance(cust.ID); bal != 0 {
		t.Errorf("الزبون شحن حاله: %d", bal)
	}
}

// ── «شام كاش عند الاستلام»: مطفي ⇒ ٤٠٩، مع «الكل» ⇒ ٤٠٠، والتبديل قبل الباب ──
func TestShamcash_PayOnDeliveryOption(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	h.Setting("shamcash.pay_on_delivery_enabled", "false")
	item := h.NewItem(1500)
	cust := h.Customer()

	body := wsBody(item, 1, "", "none")
	body["pay_shamcash"] = true
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("scp"), body); r.Code != http.StatusConflict || r.Err() != "shamcash_disabled" {
		t.Errorf("مطفي: %s — يُنتظر 409 shamcash_disabled", r)
	}
	h.Setting("shamcash.pay_on_delivery_enabled", "true")
	full := wsBody(item, 1, "", "full")
	full["pay_shamcash"] = true
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("scp"), full); r.Code != http.StatusBadRequest {
		t.Errorf("مع «الكل»: %s — يُنتظر 400", r)
	}
	oid := wsMake(t, h, cust, body, 0)
	var pay bool
	read := func() {
		if err := h.Pool.QueryRow(ctxBG(), `SELECT pay_shamcash FROM orders WHERE id = $1::uuid`, oid).Scan(&pay); err != nil {
			t.Fatal(err)
		}
	}
	read()
	o := wsRead(t, h, oid)
	if !pay || o.method != "cash" || o.scope != "none" || o.cashDue != o.total {
		t.Fatalf("بعد الإنشاء: pay=%v %+v — يُنتظر كاش كامل مع علم شام كاش", pay, o)
	}
	// **والزبون بيرجع لكاش قبل الباب** — ولا قرش بيتحرّك.
	sw := h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("scs"),
		map[string]any{"wallet_scope": "none", "expected_total": o.total, "pay_shamcash": false})
	if sw.Code != 200 {
		t.Fatalf("التبديل: %s", sw)
	}
	read()
	if pay {
		t.Error("ما رجع كاش")
	}
	if p, _ := sw.JSON()["pay_shamcash"].(bool); p {
		t.Errorf("الردّ لسّا شام كاش: %s", sw)
	}
	// **ورجع لشام كاش**، والكابتن بيشوفه، وعالباب انقفل.
	sw = h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("scs"),
		map[string]any{"wallet_scope": "none", "expected_total": o.total, "pay_shamcash": true})
	if sw.Code != 200 {
		t.Fatalf("التبديل لشام كاش: %s", sw)
	}
	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff")
	sw = h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("scs"),
		map[string]any{"wallet_scope": "none", "expected_total": o.total, "pay_shamcash": false})
	if sw.Code != http.StatusConflict || sw.Err() != "wallet_scope_locked" {
		t.Errorf("عالباب: %s — يُنتظر 409 wallet_scope_locked", sw)
	}
	if after := wsRead(t, h, oid); after.paid != 0 || after.cashDue != o.total {
		t.Errorf("التبديل حرّك مال: %+v", after)
	}
}
