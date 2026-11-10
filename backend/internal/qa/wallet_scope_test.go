package qa

// ══════════════════════════════════════════════════════════════════════
// **«الكل من المحفظة» · «التوصيل من المحفظة والباقي كاش» · كاش** — قرارُ المالك ٢٠٢٦-١١-٠٩
// ══════════════════════════════════════════════════════════════════════
//
// الطلبُ العاديُّ من أوّله لآخره على المحرّك الحقيقيّ: الإنشاءُ يخصم الحصّةَ المختارة،
// والتبديلُ قبل الباب يخصم الفرقَ أو يردّه، والبابُ يقفله، والإلغاءُ والاستردادُ والتصحيحُ
// تردّ ما دُفع من المحفظة بالضبط. **وكلُّ حالةٍ تُختم بثوابت الدفتر** (FI-03…FI-11 بلا
// خرقٍ جديد) **وبحفظ اقتصاد الطلب نفسِه**: صافي قيوده = نقدُ صندوق الكابتن منه.

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
)

var wsChecks = []string{"FI-03", "FI-04", "FI-05", "FI-06", "FI-10", "FI-11"}

type wsOrder struct {
	scope, method                         string
	total, fee, walletPaid, cashDue, paid int64
	status                                string
}

func wsPrep(t *testing.T, h *Harness) {
	t.Helper()
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("orders.max_open_per_customer", "0")
	// **أجرةٌ موجبةٌ صريحة** — «التوصيل من المحفظة» بلا أجرةٍ لا يقيس شيئاً.
	h.Setting("delivery.fee", "300")
	h.Setting("delivery.by_distance", "false")
	treasury(t, h)
}

// wsMake **طلبٌ عاديٌّ بنطاقٍ معطى** — والمحفظةُ مشحونةٌ بما يكفي.
func wsMake(t *testing.T, h *Harness, cust *User, body map[string]any, fund int64) string {
	t.Helper()
	if fund > 0 {
		h.Factory().Credit(cust.ID, fund, "topup")
	}
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("ws"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	return oid
}

func wsBody(item *Item, qty int, method, scope string) map[string]any {
	b := orderBody(item, qty)
	b["payment_method"] = method // والفارغُ = لم يُرسَل (العميلُ الجديدُ يرسل النطاقَ وحدَه)
	if scope != "" {
		b["wallet_scope"] = scope
	}
	return b
}

// wsRead **الطلبُ كما في القاعدة** — و`paid` صافي ما خرج من المحفظة له (دفعٌ − ردّ).
func wsRead(t *testing.T, h *Harness, oid string) wsOrder {
	t.Helper()
	var o wsOrder
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT wallet_scope, payment_method, total, delivery_fee, wallet_paid, cash_due, status,
		       COALESCE((SELECT -sum(amount) FROM wallet_transactions
		                  WHERE ref = $1 AND kind IN ('order_payment', 'refund')
		                    AND user_id = orders.customer_id), 0)
		FROM orders WHERE id = $1::uuid`, oid).
		Scan(&o.scope, &o.method, &o.total, &o.fee, &o.walletPaid, &o.cashDue, &o.status, &o.paid); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	return o
}

func wsChange(h *Harness, cust *User, oid, scope string, expected int64) Res {
	return h.POSTKey("/api/v1/orders/"+oid+"/wallet-scope", cust.Token, uniq("wsc"),
		map[string]any{"wallet_scope": scope, "expected_total": expected})
}

func wsDrive(t *testing.T, h *Harness, oid string, drv *User, to ...string) {
	t.Helper()
	for _, s := range to {
		if s == "delivered" {
			h.authorizeProofExempt(oid)
		}
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": s}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", s, got)
		}
	}
}

// wsConserved **حفظُ اقتصاد الطلب** — صافي قيوده = نقدُ صندوق الكابتن منه ناقصَ ما ردّه
// الدرجُ سلفةً. **وقبل المحاسبة**: صافيه = −المدفوعُ من المحفظة، **وبعد نهايةٍ قبل الاستلام صفر.**
func wsConserved(t *testing.T, h *Harness, oid string) {
	t.Helper()
	var net, box, adv, profitRows int64
	var status string
	var walletPaid int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1), 0),
		       COALESCE((SELECT sum(amount) FROM driver_cash_entries WHERE ref = $1
		                  AND kind IN ('order_collection', 'collection_reversed')), 0),
		       COALESCE((SELECT sum(amount) FROM office_cash_entries WHERE ref = $1
		                  AND source IN ('advance_return', 'undelivered_return')), 0),
		       (SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'platform_profit'),
		       status, wallet_paid
		FROM orders WHERE id = $1::uuid`, oid).Scan(&net, &box, &adv, &profitRows, &status, &walletPaid); err != nil {
		t.Fatalf("قراءةُ الحفظ: %v", err)
	}
	switch {
	case profitRows > 0:
		if net != box-adv {
			t.Errorf("حفظُ الطلب انكسر: صافي الدفتر %d ≠ نقدُ الصندوق %d − السلفة %d", net, box, adv)
		}
	case status == "cancelled" || status == "rejected":
		if net != 0 {
			t.Errorf("طلبٌ انتهى قبل المحاسبة وصافيه %d — يُنتظر صفر", net)
		}
	default:
		if net != -walletPaid {
			t.Errorf("قبل المحاسبة: صافي الدفتر %d — يُنتظر −المدفوع %d", net, -walletPaid)
		}
	}
}

// ── الإنشاء: «التوصيل من المحفظة» يخصم الأجرةَ وحدَها، والتسليمُ يقبض البضاعة ──
func TestWScope_RegularDelivery_CreateAndDeliver(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	cust := h.Customer()
	item := h.NewItem(2000)
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 50_000)

	o := wsRead(t, h, oid)
	if o.fee != 300 {
		t.Fatalf("أجرةُ التوصيل %d — الاختبارُ يحتاج أجرةً موجبة", o.fee)
	}
	if o.scope != "delivery" || o.method != "wallet" || o.walletPaid != o.fee ||
		o.cashDue != o.total-o.fee || o.paid != o.fee {
		t.Fatalf("بعد الإنشاء: %+v — يُنتظر المحفظةُ = الأجرة %d والنقدُ البضاعة", o, o.fee)
	}
	if bal := h.Factory().Balance(cust.ID); bal != 50_000-o.fee {
		t.Fatalf("رصيدُ الزبون %d — يُنتظر %d", bal, 50_000-o.fee)
	}
	wsConserved(t, h, oid)

	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	var box int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount),0) FROM driver_cash_entries
		WHERE ref = $1 AND kind = 'order_collection'`, oid).Scan(&box); err != nil {
		t.Fatal(err)
	}
	if box != o.cashDue {
		t.Errorf("الكابتن قبض %d — يُنتظر البضاعة %d", box, o.cashDue)
	}
	wsConserved(t, h, oid)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── «الكل» والكاش كما كانا، والتطبيقُ القديمُ (wallet بلا نطاق) = «الكل» ──────
func TestWScope_RegularFullCashAndOldApp(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	item := h.NewItem(1500)

	for _, c := range []struct {
		name, method, scope, want string
	}{
		{"old-app-wallet", "wallet", "", "full"},
		{"full", "", "full", "full"},
		{"cash", "cash", "", "none"},
		{"none", "", "none", "none"},
	} {
		cust := h.Customer()
		oid := wsMake(t, h, cust, wsBody(item, 1, c.method, c.scope), 50_000)
		o := wsRead(t, h, oid)
		wantPaid := int64(0)
		if c.want == "full" {
			wantPaid = o.total
		}
		if o.scope != c.want || o.walletPaid != wantPaid || o.cashDue != o.total-wantPaid || o.paid != wantPaid {
			t.Errorf("%s: %+v — يُنتظر نطاق %s ومحفظة %d", c.name, o, c.want, wantPaid)
		}
		drv := h.driverOf(oid)
		wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
		wsConserved(t, h, oid)
	}
	// **والمتناقضان يُردّان** — كاشٌ مع «الكل».
	cust := h.Customer()
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("ws"), wsBody(item, 1, "cash", "full")); r.Code != http.StatusBadRequest || r.Err() != "bad_payment" {
		t.Errorf("cash+full: %s — يُنتظر 400 bad_payment", r)
	}
	assertNewViolations(t, h, base, wsChecks...)
}

// ── التبديلُ قبل الاستلام: الفرقُ يُخصم أو يُردّ، ونفسُ الاختيار لا شيء ─────────
func TestWScope_ChangeBeforePickup(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	cust := h.Customer()
	item := h.NewItem(3000)
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "full"), 50_000)
	o := wsRead(t, h, oid)

	// «الكل» ← «التوصيل»: يُردّ ثمنُ البضاعة.
	if r := wsChange(h, cust, oid, "delivery", o.total); r.Code >= 400 {
		t.Fatalf("الكل ← التوصيل: %s", r)
	}
	if got := wsRead(t, h, oid); got.scope != "delivery" || got.walletPaid != o.fee || got.paid != o.fee ||
		got.cashDue != o.total-o.fee {
		t.Fatalf("بعد التبديل للتوصيل: %+v", got)
	}
	// **ونفسُ الاختيار ثانيةً لا يخصم ولا يردّ.**
	if r := wsChange(h, cust, oid, "delivery", o.total); r.Code >= 400 {
		t.Fatalf("تكرار: %s", r)
	}
	if got := wsRead(t, h, oid); got.paid != o.fee {
		t.Fatalf("التكرارُ حرّك المال: %+v", got)
	}
	// «التوصيل» ← كاش: تُردّ الأجرة.
	if r := wsChange(h, cust, oid, "none", o.total); r.Code >= 400 {
		t.Fatalf("التوصيل ← كاش: %s", r)
	}
	if got := wsRead(t, h, oid); got.scope != "none" || got.method != "cash" || got.walletPaid != 0 ||
		got.paid != 0 || got.cashDue != o.total {
		t.Fatalf("بعد التبديل للكاش: %+v", got)
	}
	if bal := h.Factory().Balance(cust.ID); bal != 50_000 {
		t.Fatalf("رصيدُ الزبون %d بعد الردّ — يُنتظر 50000", bal)
	}
	// كاش ← «الكل»: يُخصم المجموع.
	if r := wsChange(h, cust, oid, "full", o.total); r.Code >= 400 {
		t.Fatalf("كاش ← الكل: %s", r)
	}
	if got := wsRead(t, h, oid); got.scope != "full" || got.walletPaid != o.total || got.paid != o.total || got.cashDue != 0 {
		t.Fatalf("بعد التبديل للكل: %+v", got)
	}
	// **ومجموعٌ قديمٌ يُردّ** — تغيّر الطلبُ تحت يده.
	if r := wsChange(h, cust, oid, "delivery", o.total+1); r.Code != http.StatusConflict || r.Err() != "order_changed" {
		t.Fatalf("مجموعٌ قديم: %s — يُنتظر 409 order_changed", r)
	}
	// **وزبونٌ آخر لا يبدّل طلبَ غيره.**
	if r := wsChange(h, h.Customer(), oid, "delivery", o.total); r.Code != http.StatusForbidden {
		t.Fatalf("زبونٌ غريب: %s — يُنتظر 403", r)
	}
	wsConserved(t, h, oid)

	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	wsConserved(t, h, oid)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── وبعد الاستلام حتّى الباب: يتبدّل والخزينةُ تُعاد حسبتُها — وعند الباب يُقفل ──
func TestWScope_ChangeAfterPickupThenLockedAtDropoff(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	cust := h.Customer()
	item := h.NewItem(2500)
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 50_000)
	o := wsRead(t, h, oid)
	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up")
	wsConserved(t, h, oid)

	// «التوصيل» ← «الكل» بعد الاستلام: تُخصم البضاعة، ونقدُ الكابتن صفر.
	if r := wsChange(h, cust, oid, "full", o.total); r.Code >= 400 {
		t.Fatalf("بعد الاستلام: %s", r)
	}
	if got := wsRead(t, h, oid); got.walletPaid != o.total || got.cashDue != 0 || got.paid != o.total {
		t.Fatalf("بعد التبديل: %+v", got)
	}
	wsConserved(t, h, oid)
	// «الكل» ← «التوصيل» في الطريق.
	wsDrive(t, h, oid, drv, "on_the_way")
	if r := wsChange(h, cust, oid, "delivery", o.total); r.Code >= 400 {
		t.Fatalf("في الطريق: %s", r)
	}
	wsConserved(t, h, oid)

	// **وعند الباب يُقفل.**
	wsDrive(t, h, oid, drv, "at_dropoff")
	if r := wsChange(h, cust, oid, "full", o.total); r.Code != http.StatusConflict || r.Err() != "wallet_scope_locked" {
		t.Fatalf("عند الباب: %s — يُنتظر 409 wallet_scope_locked", r)
	}
	wsDrive(t, h, oid, drv, "delivered")
	got := wsRead(t, h, oid)
	if got.walletPaid != o.fee || got.cashDue != o.total-o.fee {
		t.Fatalf("بعد التسليم: %+v", got)
	}
	if r := wsChange(h, cust, oid, "full", o.total); r.Code != http.StatusConflict || r.Err() != "wallet_scope_locked" {
		t.Fatalf("بعد التسليم: %s — يُنتظر 409 wallet_scope_locked", r)
	}
	wsConserved(t, h, oid)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── الإلغاءُ يردّ الأجرةَ المدفوعة، والاستردادُ بعد التسليم يردّ المجموع ──────────
func TestWScope_CancelAndRefund(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	item := h.NewItem(1800)

	cust := h.Customer()
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 20_000)
	if r := h.POST("/api/v1/orders/"+oid+"/cancel", cust.Token, map[string]any{"reason": "ws"}); r.Code >= 400 {
		t.Fatalf("الإلغاء: %s", r)
	}
	if got := wsRead(t, h, oid); got.paid != 0 || got.status != "cancelled" {
		t.Fatalf("بعد الإلغاء: %+v — يُنتظر ردُّ الأجرة كاملةً", got)
	}
	if bal := h.Factory().Balance(cust.ID); bal != 20_000 {
		t.Fatalf("رصيدُ الزبون %d بعد الإلغاء — يُنتظر 20000", bal)
	}
	wsConserved(t, h, oid)

	cust2 := h.Customer()
	oid2 := wsMake(t, h, cust2, wsBody(item, 1, "wallet", "delivery"), 20_000)
	o := wsRead(t, h, oid2)
	drv := h.driverOf(oid2)
	wsDrive(t, h, oid2, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	admin := h.NewUser("admin")
	if r := h.POST("/api/v1/admin/orders/"+oid2+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "ws"}); r.Code >= 400 {
		t.Fatalf("الاسترداد: %s", r)
	}
	// **يُردّ للمحفظة المجموعُ كلُّه** — الأجرةُ من المحفظة والبضاعةُ التي دفعها نقداً.
	if bal := h.Factory().Balance(cust2.ID); bal != 20_000-o.fee+o.total {
		t.Fatalf("رصيدُ الزبون %d بعد الاسترداد — يُنتظر %d", bal, 20_000-o.fee+o.total)
	}
	wsConserved(t, h, oid2)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── التصحيحُ (تسليمٌ سُجّل خطأً) يردّ ما دُفع من المحفظة بالضبط ──────────────────
func TestWScope_UndeliverReturnsWalletPart(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	cust := h.Customer()
	item := h.NewItem(1600)
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 20_000)
	o := wsRead(t, h, oid)
	drv := h.driverOf(oid)
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	admin := h.NewUser("admin")
	if r := undeliver(h, admin, oid, "ws — تسليمٌ خاطئ"); r.Code >= 400 {
		t.Fatalf("التصحيح: %s", r)
	}
	if got := wsRead(t, h, oid); got.paid != 0 {
		t.Fatalf("بعد التصحيح: %+v — يُنتظر ردُّ الأجرة %d", got, o.fee)
	}
	if bal := h.Factory().Balance(cust.ID); bal != 20_000 {
		t.Fatalf("رصيدُ الزبون %d — يُنتظر 20000", bal)
	}
	assertNewViolations(t, h, base, wsChecks...)
}

// ── الدَّين لا يُسدَّد أوّلاً (قرارُ المالك ٢: أ) — الطلبُ من الرصيد المتاح ────────
//
// **ولا دَينَ للزبون بصفته زبوناً في المنصّة** (الذممُ للمتجر والمندوب والكابتن) — فالزبونُ
// هنا صاحبُ دينٍ بصفةٍ أخرى على الحساب نفسِه، وهو أقربُ ما يقع.
func TestWScope_DebtNotSettledFirst(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	cust := h.Customer()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO financial_obligations (party_kind, party_id, amount, cause)
		VALUES ('driver', $1::uuid, 700, 'legacy_opening')`, cust.ID); err != nil {
		t.Fatalf("زرعُ الدَّين: %v", err)
	}
	item := h.NewItem(1200)
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 5_000)
	o := wsRead(t, h, oid)
	if o.paid != o.fee || h.Factory().Balance(cust.ID) != 5_000-o.fee {
		t.Fatalf("مع دَين: %+v والرصيد %d — يُنتظر خصمُ الأجرة وحدَها", o, h.Factory().Balance(cust.ID))
	}
	var debt int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		WHERE party_id = $1::uuid`, cust.ID).Scan(&debt); err != nil {
		t.Fatalf("الدَّين: %v", err)
	}
	if debt != 700 {
		t.Fatalf("الدَّين صار %d — يُنتظر 700 كما هو", debt)
	}
	wsConserved(t, h, oid)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── الحرّاس: رصيدٌ لا يكفي · سقفُ COD · قفلُ النقد · سقفُ نقد الكابتن ─────────────
func TestWScope_ChangeGuards(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	item := h.NewItem(4000)

	// رصيدٌ يكفي الأجرةَ ولا يكفي «الكل».
	cust := h.Customer()
	oid := wsMake(t, h, cust, wsBody(item, 1, "", "delivery"), 1_000)
	o := wsRead(t, h, oid)
	if r := wsChange(h, cust, oid, "full", o.total); r.Code != http.StatusConflict || r.Err() != "insufficient_balance" {
		t.Fatalf("رصيدٌ ناقص: %s — يُنتظر 409 insufficient_balance", r)
	}
	if got := wsRead(t, h, oid); got.scope != "delivery" || got.paid != o.fee {
		t.Fatalf("الرفضُ حرّك المال: %+v", got)
	}

	// سقفُ COD: التبديلُ إلى كاشٍ أكثرَ يُسأل عنه.
	h.Setting("customers.cod_limit", fmt.Sprint(o.cashDue))
	if r := wsChange(h, cust, oid, "none", o.total); r.Code != http.StatusConflict || r.Err() != "cod_limit_exceeded" {
		t.Fatalf("سقفُ COD: %s — يُنتظر 409 cod_limit_exceeded", r)
	}
	h.Setting("customers.cod_limit", "100000000")

	// سقفُ نقد الكابتن: كابتنٌ يحمل ما يملأ سقفَه لا يزيد نقدُه بتبديل الزبون.
	drv := h.driverOf(oid)
	// **والسقفُ يعدّ مالَ المنصّة وحدَه** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٧) — لا ثمنَ البضاعة.
	// فسقفٌ بقدر ما بيده الآن يمتلئ، **والتبديلُ إلى كاشٍ يزيده بالأجرة.**
	held, err := cashbox.PlatformCashOf(ctxBG(), h.Pool, oid)
	if err != nil {
		t.Fatal(err)
	}
	h.Setting("drivers.cash_limit", fmt.Sprint(held))
	if r := wsChange(h, cust, oid, "none", o.total); r.Code != http.StatusConflict || r.Err() != "cash_limit_exceeded" {
		t.Fatalf("سقفُ نقد الكابتن: %s — يُنتظر 409 cash_limit_exceeded", r)
	}
	h.Setting("drivers.cash_limit", "900000000")
	if r := wsChange(h, cust, oid, "none", o.total); r.Code >= 400 {
		t.Fatalf("بعد رفع السقف: %s", r)
	}
	wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way", "at_dropoff", "delivered")
	wsConserved(t, h, oid)
	assertNewViolations(t, h, base, wsChecks...)
}

// ── تبديلٌ يسابق الوصولَ للباب: إمّا قبله فيُحسب، أو بعده فيُقفل — ولا نصفَ حال ────
func TestWScope_ConcurrentChangeVsDropoff(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	base := financialBaseline(t, h, wsChecks...)
	item := h.NewItem(2200)
	for i := 0; i < 4; i++ {
		cust := h.Customer()
		oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", "delivery"), 30_000)
		o := wsRead(t, h, oid)
		drv := h.driverOf(oid)
		wsDrive(t, h, oid, drv, "at_pickup", "picked_up", "on_the_way")

		var wg sync.WaitGroup
		var change, door Res
		wg.Add(2)
		go func() { defer wg.Done(); change = wsChange(h, cust, oid, "full", o.total) }()
		go func() {
			defer wg.Done()
			door = h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, map[string]any{"to": "at_dropoff"})
		}()
		wg.Wait()
		if door.Code >= 400 {
			t.Fatalf("الوصولُ للباب: %s", door)
		}
		got := wsRead(t, h, oid)
		switch {
		case change.Code < 400:
			if got.scope != "full" || got.walletPaid != o.total || got.cashDue != 0 || got.paid != o.total {
				t.Fatalf("التبديلُ نجح والطلب %+v", got)
			}
		case change.Err() == "wallet_scope_locked":
			if got.scope != "delivery" || got.walletPaid != o.fee || got.paid != o.fee {
				t.Fatalf("التبديلُ قُفل والطلب %+v", got)
			}
		default:
			t.Fatalf("ردٌّ غيرُ منتظر: %s", change)
		}
		wsDrive(t, h, oid, drv, "delivered")
		wsConserved(t, h, oid)
	}
	assertNewViolations(t, h, base, wsChecks...)
}
