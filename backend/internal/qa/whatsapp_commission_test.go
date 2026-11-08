package qa

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ متجرنا على طلب الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٨
// ══════════════════════════════════════════════════════════════════════
//
// متجرٌ من متاجرنا وثمنٌ مكتوب ⇒ عند التسليم تُخصم عمولتُه على الثمن من محفظة صاحبه إلى الخزينة،
// **وما لم يكفِه المتاحُ دينٌ عليه.** متجرٌ غيرُ تابعٍ أو بلا ثمن ⇒ لا شيء. والدولارُ بسعر المتجر.
// **والاسترجاعُ بعد التسليم يردّها.** وثوابتُ الدفتر خضراءُ في كلّ سيناريو.

import (
	"net/http"
	"strings"
	"testing"
)

// waComm **طلبُ واتسابٍ من متجرنا بنسبة ١٠٪** — والمحفظةُ بالرصيد المطلوب.
type waComm struct {
	w       waFx
	ownerID string
	ops     *User
}

func newWAComm(t *testing.T, h *Harness, name string, balance int64) waComm {
	t.Helper()
	w := waSimpleFx(t, h, name)
	c := waComm{w: w, ownerID: w.md.fx.M.Owner.ID, ops: h.NewUser("operations")}
	// **وبلا مندوب** — نصيبُه سيناريو وحدَه (`TestWACOMM05`)، وهنا الدفترُ عمولةُ المتجر وحدَها.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET commission_percent = 10, sales_rep_user_id = NULL WHERE id = $1`,
		w.md.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	if cur := waBalance(t, h, c.ownerID); cur != balance {
		w.md.f.Credit(c.ownerID, balance-cur, "adjustment")
	}
	return c
}

func waBalance(t *testing.T, h *Harness, userID string) int64 {
	t.Helper()
	var b int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1), 0)`,
		userID).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

func waPlatformCommission(t *testing.T, h *Harness, id string) int64 {
	t.Helper()
	var c int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT platform_commission FROM orders WHERE id = $1`, id).Scan(&c); err != nil {
		t.Fatal(err)
	}
	return c
}

// waCommDebt **دينُ عمولة الطلب** — مبلغُه وباقيه، و-1 إن لم يكن.
func waCommDebt(t *testing.T, h *Harness, id string) (amount, open int64) {
	t.Helper()
	amount, open = -1, -1
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT amount, amount - settled FROM financial_obligations
		 WHERE order_id = $1 AND cause = 'whatsapp_order_commission' AND party_kind = 'merchant'`, id).
		Scan(&amount, &open)
	return
}

// waDeliver **السائقُ يمضي بالطلب إلى التسليم** — ويكتب الثمنَ بمكان الاستلام إن أُعطي.
func waDeliver(t *testing.T, h *Harness, id string, goods int64) {
	t.Helper()
	drv := h.driverOf(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "at_pickup"}); r.Code >= 400 {
		t.Fatalf("وصلت المتجر: %s", r)
	}
	if goods > 0 {
		if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": goods}); r.Code != http.StatusOK {
			t.Fatalf("كتابةُ الثمن: %d / %s", r.Code, r.Err())
		}
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
}

func (c waComm) ourStore(t *testing.T, h *Harness, extra map[string]any) string {
	t.Helper()
	b := map[string]any{"store_id": c.w.md.fx.M.ID, "recipient_phone": uniqPhone(), "fee": 600}
	for k, v := range extra {
		b[k] = v
	}
	return waCreated(t, waPost(t, h, c.ops.Token, b))
}

func waRefund(t *testing.T, h *Harness, admin *User, id string) {
	t.Helper()
	if r := h.POST("/api/v1/admin/orders/"+id+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "WA-COMM"}); r.Code >= 400 {
		t.Fatalf("الاسترجاع: %s", r)
	}
}

// TestWACOMM01_OurStoreWalletPays **متجرُنا والمحفظةُ تكفي** ⇒ تُخصم ١٠٪ من ٢٤٠٠ = ٢٤٠ من محفظة صاحبه
// إلى الخزينة، و`platform_commission = 240`. ثمّ **الاسترجاعُ يردّها** والعمودُ يعود صفراً.
//
// **وعلى الشيفرة القديمة يسقط**: الدفترُ أجرُ السائق وحصّةُ المنصّة لا غير، والمحفظةُ لا تتحرّك.
func TestWACOMM01_OurStoreWalletPays(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-COMM-01", 1000)
	base := mdaBaseline(t, h)
	id := c.ourStore(t, h, nil)
	waDeliver(t, h, id, 2400)

	if got := waBalance(t, h, c.ownerID); got != 760 {
		t.Fatalf("**محفظةُ المتجر %d والمنتظَر 760** — العمولةُ ٢٤٠ لم تُخصم", got)
	}
	led := waLedgerOf(t, h, id)
	if led["order_payment"] != -240 || led["driver_earning"] != 540 || led["platform_profit"] != 300 || len(led) != 3 {
		t.Fatalf("**الدفتر**: %v — المنتظَر خصمُ ٢٤٠ وأجرُ ٥٤٠ وحصّةُ المنصّة ٦٠+٢٤٠", led)
	}
	if pc := waPlatformCommission(t, h, id); pc != 240 {
		t.Fatalf("**platform_commission %d والمنتظَر 240**", pc)
	}
	if a, _ := waCommDebt(t, h, id); a != -1 {
		t.Fatalf("**دينٌ والمحفظةُ تكفي**: %d", a)
	}
	var note string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT note FROM wallet_transactions WHERE ref = $1 AND kind = 'order_payment'`, id).Scan(&note)
	if note != "عمولة طلب واتساب" {
		t.Fatalf("**ملاحظةُ الكشف %q**", note)
	}
	mdaNoNewViolations(t, h, base)

	// ── والاسترجاعُ بعد التسليم يردّها ─────────────────────────────────
	waRefund(t, h, c.w.md.admin, id)
	if got := waBalance(t, h, c.ownerID); got != 1000 {
		t.Fatalf("**الاسترجاعُ لم يردّ العمولة**: المحفظة %d والمنتظَر 1000", got)
	}
	led = waLedgerOf(t, h, id)
	if led["refund"] != 240 || led["platform_profit"] != 60 {
		t.Fatalf("**الدفترُ بعد الاسترجاع**: %v", led)
	}
	if pc := waPlatformCommission(t, h, id); pc != 0 {
		t.Fatalf("**platform_commission بعد الاسترجاع %d**", pc)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWACOMM02_EmptyWalletBecomesDebt **المحفظةُ فارغة** ⇒ دينٌ ٢٤٠ على المتجر بسبب
// `whatsapp_order_commission`، **والمحفظةُ لا تتحرّك** (لا سالب). ثمّ الاسترجاعُ يُسقط الدين.
func TestWACOMM02_EmptyWalletBecomesDebt(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-COMM-02", 0)
	base := mdaBaseline(t, h)
	id := c.ourStore(t, h, nil)
	waDeliver(t, h, id, 2400)

	if got := waBalance(t, h, c.ownerID); got != 0 {
		t.Fatalf("**المحفظةُ تحرّكت %d**", got)
	}
	if a, open := waCommDebt(t, h, id); a != 240 || open != 240 {
		t.Fatalf("**الدين %d/%d والمنتظَر 240**", a, open)
	}
	led := waLedgerOf(t, h, id)
	if _, ok := led["order_payment"]; ok || led["platform_profit"] != 60 || len(led) != 2 {
		t.Fatalf("**الدفتر**: %v — لا خصمَ والدينُ خارجَه", led)
	}
	if pc := waPlatformCommission(t, h, id); pc != 240 {
		t.Fatalf("**platform_commission %d والمنتظَر 240**", pc)
	}
	mdaNoNewViolations(t, h, base)

	waRefund(t, h, c.w.md.admin, id)
	if _, open := waCommDebt(t, h, id); open != 0 {
		t.Fatalf("**الدينُ باقٍ بعد الاسترجاع**: %d", open)
	}
	if got := waBalance(t, h, c.ownerID); got != 0 {
		t.Fatalf("**المحفظةُ بعد الاسترجاع %d** — لم يُدفع شيءٌ ليُردّ", got)
	}
	if _, ok := waLedgerOf(t, h, id)["refund"]; ok {
		t.Fatal("**ردٌّ لعمولةٍ لم تُدفع**")
	}
	mdaNoNewViolations(t, h, base)
}

// TestWACOMM03_NoCommissionCases **متجرٌ غيرُ تابع · بلا ثمن · دولارٌ بلا سعر** ⇒ لا عمولة.
func TestWACOMM03_NoCommissionCases(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-COMM-03", 1000)
	base := mdaBaseline(t, h)

	check := func(name, id string) {
		t.Helper()
		led := waLedgerOf(t, h, id)
		if len(led) != 2 || led["platform_profit"] != 60 || waPlatformCommission(t, h, id) != 0 {
			t.Fatalf("**%s: عمولةٌ لا تُستحقّ**: %v", name, led)
		}
		if a, _ := waCommDebt(t, h, id); a != -1 {
			t.Fatalf("**%s: دين**", name)
		}
	}
	// متجرٌ غيرُ تابعٍ باسمه وثمنٍ كتبه السائق.
	ext := waCreated(t, waPost(t, h, c.ops.Token, waSimpleBody(uniqPhone(), 600)))
	waDeliver(t, h, ext, 2400)
	check("متجرٌ غيرُ تابع", ext)
	// متجرُنا بلا ثمن.
	none := c.ourStore(t, h, nil)
	waDeliver(t, h, none, 0)
	check("بلا ثمن", none)
	// دولارٌ ومتجرٌ بلا سعر صرف.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET exchange_rate = NULL WHERE id = $1`, c.w.md.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	usd := c.ourStore(t, h, map[string]any{"goods_amount": 10, "amount_currency": "USD"})
	waDeliver(t, h, usd, 0)
	check("دولارٌ بلا سعر", usd)

	if got := waBalance(t, h, c.ownerID); got != 1000 {
		t.Fatalf("**المحفظةُ تحرّكت %d**", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWACOMM04_USDGoodsUseMerchantRate **ثمنٌ ١٠$ وسعرُ المتجر ١٥٠** ⇒ ١٥٠٠ ل.س ⇒ عمولةٌ ١٥٠.
func TestWACOMM04_USDGoodsUseMerchantRate(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-COMM-04", 1000)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET exchange_rate = 150 WHERE id = $1`, c.w.md.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	base := mdaBaseline(t, h)
	id := c.ourStore(t, h, map[string]any{"goods_amount": 10, "amount_currency": "USD"})
	waDeliver(t, h, id, 0)
	if pc := waPlatformCommission(t, h, id); pc != 150 {
		t.Fatalf("**platform_commission %d والمنتظَر 150**", pc)
	}
	if got := waBalance(t, h, c.ownerID); got != 850 {
		t.Fatalf("**المحفظة %d والمنتظَر 850**", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWACOMM05_RepGetsShare **مندوبُ متجرنا يأخذ نصيبَه من عمولة طلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٨:
// «المندوب ياخذ عمولة طبعا اذا المتجر تابع للمندوب») — نسبتُه من لقطة الطلب على عمولة ٢٤٠، **بالوضعين** (العمولةُ
// والهامش: طلبُ الواتساب بلا هامشٍ فعمولتُه قاعدتُه). **ومتجرٌ بلا مندوبٍ لا يُسقط التسليم.**
//
// **وعلى الشيفرة القديمة يسقط**: لا قيدَ `commission` للمندوب.
func TestWACOMM05_RepGetsShare(t *testing.T) {
	for _, src := range []string{`"platform_commission"`, `"pricing_margin"`, `"both"`} {
		t.Run(strings.Trim(src, `"`), func(t *testing.T) { waRepShare(t, src) })
	}
}

func waRepShare(t *testing.T, src string) {
	{
		h := New(t)
		c := newWAComm(t, h, "منطقةُ WA-COMM-05", 1000)
		h.Setting("sales.commission_source", src)
		h.Setting("sales.activation_orders", "0")
		rep := c.w.md.f.RepAccount()
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET sales_rep_user_id = $2 WHERE id = $1`,
			c.w.md.fx.M.ID, rep.ID); err != nil {
			t.Fatal(err)
		}
		base := mdaBaseline(t, h)
		id := c.ourStore(t, h, nil)
		waDeliver(t, h, id, 2400)
		var got int64
		if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
			WHERE user_id = $1 AND ref = $2 AND kind = 'commission'`, rep.ID, id).Scan(&got); err != nil {
			t.Fatal(err)
		}
		var pct int64
		_ = h.Pool.QueryRow(ctxBG(), `SELECT snap_rep_commission_percent FROM orders WHERE id = $1`, id).Scan(&pct)
		if want := 240 * pct / 100; pct <= 0 || got != want {
			t.Fatalf("%s: **نصيبُ المندوب %d والمنتظَر %d** (نسبتُه %d٪ من ٢٤٠)", src, got, want, pct)
		}
		if pc := waPlatformCommission(t, h, id); pc != 240 {
			t.Fatalf("%s: عمولةُ المنصّة %d", src, pc)
		}
		mdaNoNewViolations(t, h, base)
	}
}

// TestWACOMM06_ZeroIsAmanah **صفرٌ = أمانةٌ بلا ثمن** (قرارُ المالك ٢٠٢٦-١١-٠٨: «خيار امانه بدون سعر ويقبل
// رقم صفر») — يُوثَّق ويمضي التسليم، **بلا ثمنٍ يُقبض ولا رسالةِ مبالغ ولا عمولة.**
//
// **وعلى الشيفرة القديمة يسقط**: الصفرُ يُردّ ٤٠٠ ويبقى الاستلامُ موقوفاً عليه.
func TestWACOMM06_ZeroIsAmanah(t *testing.T) {
	h := New(t)
	n := armFeedback(t, h, true)
	c := newWAComm(t, h, "منطقةُ WA-COMM-06", 1000)
	base := mdaBaseline(t, h)
	phone := uniqPhone()
	id := c.ourStore(t, h, map[string]any{"recipient_phone": phone})
	before := len(n.to(phone))
	drv := h.driverOf(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "at_pickup"}); r.Code >= 400 {
		t.Fatalf("وصلت المتجر: %s", r)
	}
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 0}); r.Code != http.StatusOK {
		t.Fatalf("**الصفرُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	var collect *int64
	var status string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT collect_amount, COALESCE(fetch_amount_status, '') FROM orders WHERE id = $1`, id).
		Scan(&collect, &status)
	if collect != nil || status != "approved" {
		t.Fatalf("**الأمانةُ**: ثمنٌ للقبض %v · حال %q — المنتظَر لا ثمن وموثَّق", collect, status)
	}
	if after := len(n.to(phone)); after != before {
		t.Fatalf("**رسالةُ مبالغ بُعثت لأمانةٍ بلا ثمن**: %d ← %d", before, after)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
	if got := waBalance(t, h, c.ownerID); got != 1000 {
		t.Fatalf("**عمولةٌ على أمانة**: المحفظة %d", got)
	}
	if pc := waPlatformCommission(t, h, id); pc != 0 {
		t.Fatalf("**عمولةٌ على أمانة**: %d", pc)
	}
	mdaNoNewViolations(t, h, base)
}

// TestWAORD050_ArrivedMessage **«وصلت» ⇒ البوتُ يُخبر الزبون** (قرارُ المالك ٢٠٢٦-١١-٠٨: «سائق رحال غو وصل
// الى موقعك الرجاء الخروج لاستلام طلبك») — مرّةً عند «وصلت»، ولا قبلها.
//
// **وعلى الشيفرة القديمة يسقط**: لا رسالةَ عند الوصول.
func TestWAORD050_ArrivedMessage(t *testing.T) {
	h := New(t)
	n := armFeedback(t, h, true)
	c := newWAComm(t, h, "منطقةُ WA-050", 0)
	phone := uniqPhone()
	id := c.ourStore(t, h, map[string]any{"recipient_phone": phone})
	drv := h.driverOf(id)
	step := func(to string) {
		t.Helper()
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s: %s", to, r)
		}
	}
	has := func() int {
		k := 0
		for _, m := range n.to(phone) {
			if strings.Contains(m.text, "وصل إلى موقعك") {
				k++
			}
		}
		return k
	}
	step("at_pickup")
	if r := h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": 0}); r.Code != http.StatusOK {
		t.Fatalf("الأمانة: %s", r)
	}
	step("picked_up")
	step("on_the_way")
	if k := has(); k != 0 {
		t.Fatalf("**رسالةُ الوصول قبل «وصلت»**: %d", k)
	}
	step("at_dropoff")
	if k := has(); k != 1 {
		t.Fatalf("**رسالةُ الوصول**: %d رسالة — المنتظَر واحدة", k)
	}
}

// TestWAORD051_ChangeDriver **«بدّل السائق» في طلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٨: «طلب الواتس لازم في
// تبديل السائق») — قبل الاستلام يعيده المكتبُ للطابور بسبب، **ولا يعود لمن تُرك**، ثمّ يُسنده لسائقٍ آخر.
func TestWAORD051_ChangeDriver(t *testing.T) {
	h := New(t)
	c := newWAComm(t, h, "منطقةُ WA-051", 0)
	admin := h.NewUser("admin")
	for _, at := range []string{"assigned", "at_pickup"} {
		id := c.ourStore(t, h, nil)
		old := h.driverOf(id)
		if at == "at_pickup" {
			if r := h.POST("/api/v1/driver/orders/"+id+"/transition", old.Token, map[string]any{"to": "at_pickup"}); r.Code >= 400 {
				t.Fatalf("وصلت المتجر: %s", r)
			}
		}
		if r := h.POST("/api/v1/admin/orders/"+id+"/transition", c.ops.Token,
			map[string]any{"to": "dispatching", "note": "السائق تأخر"}); r.Code >= 400 {
			t.Fatalf("%s: **المكتبُ لم يُعِده للطابور**: %s", at, r)
		}
		var drv *string
		_ = h.Pool.QueryRow(ctxBG(), `SELECT driver_id::text FROM orders WHERE id = $1`, id).Scan(&drv)
		if drv != nil || h.statusOf(id) != "dispatching" {
			t.Fatalf("%s: **السائقُ القديمُ باقٍ**: %v · %s", at, drv, h.statusOf(id))
		}
		if r := h.POST("/api/v1/admin/orders/"+id+"/assign", admin.Token,
			map[string]any{"driver_id": old.ID}); r.Code < 400 {
			t.Fatalf("%s: **عاد الطلبُ لمن تُرك**", at)
		}
		fresh := c.w.md.f.Driver(OnShift())
		if r := h.POST("/api/v1/admin/orders/"+id+"/assign", admin.Token,
			map[string]any{"driver_id": fresh.ID}); r.Code >= 400 {
			t.Fatalf("%s: **الإسنادُ للسائق الجديد**: %s", at, r)
		}
		_ = h.Pool.QueryRow(ctxBG(), `SELECT driver_id::text FROM orders WHERE id = $1`, id).Scan(&drv)
		if drv == nil || *drv != fresh.ID {
			t.Fatalf("%s: **السائقُ الجديدُ لم يأخذه**: %v", at, drv)
		}
	}
}
