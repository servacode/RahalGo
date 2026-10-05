package qa

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» تمرّ بموافقة المنصّة** — قرارُ المالك ٢٠٢٦-١٠-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// **كانت تُولد في طابور السائقين** بلا «معلَّق»، وتُخصم أجرتُها عند الإنشاء.
// **فصارت كأيّ طلب**: `pending` ⇐ يقبلها المكتبُ أو يرفضها ⇐ تنزل الطابور.
// **والخصمُ عند القبول لا عند الإنشاء** — فالرفضُ لا يحتاج ردّاً.
//
// **والمالُ يُقاس من الصفوف**، **وثوابتُ الدفتر (`fininv`) تُصوَّر قبل
// السيناريو ويُطلب ألّا يستجدّ فيها خرق.**

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/fininv"
)

// mdaNoNewViolations **لا خرقَ جديداً في الدفتر** — منذ صورة `base`.
func mdaNoNewViolations(t *testing.T, h *Harness, base fininv.Baseline) {
	t.Helper()
	after, err := fininv.Run(ctxBG(), h.Pool)
	if err != nil {
		t.Fatalf("تعذّر فحصُ الدفتر: %v", err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("**خرقٌ جديدٌ في الدفتر**: %v", fresh)
	}
}

func mdaBaseline(t *testing.T, h *Harness) fininv.Baseline {
	t.Helper()
	base, err := fininv.Capture(ctxBG(), h.Pool)
	if err != nil {
		t.Fatalf("تعذّرت صورةُ الأساس: %v", err)
	}
	return base
}

// TestMDA01_PendingNotInDriverQueueUntilOfficeAccepts **لا تنزل الطابورَ قبل القبول.**
//
// **وعلى الشيفرة القديمة يسقط في أوّل سطر**: كانت تُولد `dispatching`
// بلا موافقةِ أحد، ويُعرض على السائقين فوراً (`OfferNext`).
func TestMDA01_PendingNotInDriverQueueUntilOfficeAccepts(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDA-01")
	base := mdaBaseline(t, h)
	drv := mdDriverAtStore(t, h, m.f, m.fx.M.ID)

	id := m.create(t, h, "recipient", uniq("mda01"))
	if st := h.statusOf(id); st != "pending" {
		t.Fatalf("**الحال %s والمنتظَر pending** — نزلت الطابورَ بلا موافقة المنصّة", st)
	}
	var dispatchedAt *time.Time
	var offered *string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT dispatched_at, offered_driver_id::text FROM orders WHERE id = $1`, id).
		Scan(&dispatchedAt, &offered); err != nil {
		t.Fatal(err)
	}
	if dispatchedAt != nil || offered != nil {
		t.Fatalf("**عُرضت على سائقٍ قبل القبول**: نزلت %v · عُرضت على %v", dispatchedAt, offered)
	}
	if q := h.GET("/api/v1/driver/queue", drv.Token); containsID(q, id) {
		t.Fatalf("**ظهرت في طابور السائق قبل موافقة المنصّة**")
	}
	if r := h.POST("/api/v1/driver/orders/"+id+"/accept", drv.Token, map[string]any{}); r.Code < 400 {
		t.Fatalf("**قبلها سائقٌ وهي بانتظار المكتب**: %d", r.Code)
	}

	// ── وتظهر في لوحة المكتب «جديدة» كسائر الطلبات ────────────────────
	board := h.GET("/api/v1/admin/orders?filter=new", m.admin.Token)
	if board.Code != http.StatusOK || !containsID(board, id) {
		t.Fatalf("**لا تظهر في «جديدة» بلوحة المكتب**: %d", board.Code)
	}
	// **والمكتبُ يُخبَر بوصولها** — كطلبٍ جديد.
	var opsNotes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND title = 'توصيلة جديدة من متجر'`, m.admin.ID).Scan(&opsNotes)
	if opsNotes == 0 {
		t.Fatalf("**لم يُخبَر المكتبُ بتوصيلةٍ تنتظره**")
	}

	// ── المكتبُ يقبل ⇒ تنزل الطابور ثمّ دورةُ السائق كاملة ──────────────
	m.accept(t, h, id)
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT dispatched_at FROM orders WHERE id = $1`, id).Scan(&dispatchedAt); err != nil {
		t.Fatal(err)
	}
	if dispatchedAt == nil {
		t.Fatalf("نزلت الطابورَ بلا وقتِ نزول")
	}
	// **وحدثُ القبول ثمّ الإنزال في السجلّ** — لا قفزةٌ بلا أثر.
	var events string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT string_agg(to_status, ',' ORDER BY created_at, id)
		FROM order_events WHERE order_id = $1`, id).Scan(&events)
	if !strings.HasPrefix(events, "pending,accepted,dispatching") {
		t.Fatalf("سجلُّ الحالات %q والمنتظَرُ أوّلُه pending,accepted,dispatching", events)
	}
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok)
	if st, _ := one.JSON()["status"].(string); one.Code != http.StatusOK || (st != "dispatching" && st != "assigned") {
		t.Fatalf("المتجرُ لا يرى نزولَها: %d / %s", one.Code, one)
	}
	carrier := h.driverOf(id)
	deliverOrder(t, h, id, carrier)
	if st := h.statusOf(id); st != "delivered" {
		t.Fatalf("الحال %s والمنتظَر delivered", st)
	}
	if got := walletOf(t, h, carrier.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDA02_OfficeRejectsStoreNotifiedNoMoney **المكتبُ يرفض ⇒ المتجرُ يُخبَر بالسبب ولا مالَ يتحرّك.**
func TestMDA02_OfficeRejectsStoreNotifiedNoMoney(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDA-02")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	base := mdaBaseline(t, h)

	id := m.create(t, h, "merchant", uniq("mda02"))
	const reason = "العنوان خارج التغطية الفعلية"
	r := h.POST("/api/v1/admin/orders/"+id+"/transition", m.admin.Token,
		map[string]any{"to": "rejected", "note": reason})
	if r.Code != http.StatusOK {
		t.Fatalf("الرفض: %d / %s", r.Code, r.Err())
	}
	if st := h.statusOf(id); st != "rejected" {
		t.Fatalf("الحال %s والمنتظَر rejected", st)
	}
	if got := walletOf(t, h, owner); got != 2000 {
		t.Fatalf("**محفظةُ المتجر %d بعد الرفض والمنتظَر 2000**", got)
	}
	var moves int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions WHERE ref = $1`, id).Scan(&moves)
	var debts int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM financial_obligations WHERE order_id = $1`, id).Scan(&debts)
	if moves != 0 || debts != 0 {
		t.Fatalf("**تحرّك مالٌ لتوصيلةٍ مرفوضة**: قيودٌ %d · ديونٌ %d", moves, debts)
	}
	// **والمتجرُ يُخبَر بالسبب** — بعد الإيداع، فيُنتظَر قليلاً.
	var body string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT body FROM notifications
			WHERE user_id = $1::uuid AND title = 'اعتذرت المنصة عن توصيلتك'
			ORDER BY created_at DESC LIMIT 1`, owner).Scan(&body)
		if body != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(body, reason) {
		t.Fatalf("**لم يُخبَر المتجرُ بالرفض وسببه**: %q", body)
	}
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok)
	if one.Code != http.StatusOK || one.JSON()["status"] != "rejected" {
		t.Fatalf("المتجرُ لا يرى الرفض: %d / %s", one.Code, one)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDA03_WalletChargedAtAcceptanceNotCreation **الخصمُ عند القبول — وإن لم تكفِ رُدّ القبول.**
func TestMDA03_WalletChargedAtAcceptanceNotCreation(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDA-03")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 500)
	base := mdaBaseline(t, h)

	// **توصيلتان والمحفظةُ تكفي واحدة** — الإنشاءُ يفحص ولا يحجز.
	a := m.create(t, h, "merchant", uniq("mda03a"))
	b := m.create(t, h, "merchant", uniq("mda03b"))
	var paid int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT wallet_paid FROM orders WHERE id = $1`, a).Scan(&paid)
	if got := walletOf(t, h, owner); got != 500 || paid != 0 {
		t.Fatalf("**خُصم عند الإنشاء**: محفظةٌ %d · مدفوعٌ %d — والمنتظَرُ 500 و0", got, paid)
	}
	m.accept(t, h, a)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT wallet_paid FROM orders WHERE id = $1`, a).Scan(&paid)
	if got := walletOf(t, h, owner); got != 0 || paid != 500 {
		t.Fatalf("بعد القبول محفظةٌ %d · مدفوعٌ %d — والمنتظَرُ 0 و500", got, paid)
	}
	// **والثانيةُ لا يُقبل قبولُها** — لا محفظةَ ولا سقفَ دين.
	r := h.POST("/api/v1/admin/orders/"+b+"/transition", m.admin.Token,
		map[string]any{"to": "accepted"})
	if r.Code != http.StatusPaymentRequired || r.Err() != "merchant_delivery_unpaid" {
		t.Fatalf("قبولٌ بلا مقدرة: %d / %s — والمنتظَرُ 402 / merchant_delivery_unpaid", r.Code, r.Err())
	}
	if st := h.statusOf(b); st != "pending" {
		t.Fatalf("**الحال %s بعد ردّ القبول والمنتظَر pending** — نزلت توصيلةٌ بلا أجرة", st)
	}
	// **والمقبولةُ تُلغى قبل الاستلام فيعود المال** — created→accepted→cancelled.
	if c := h.POST("/api/v1/merchant/deliveries/"+a+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	if got := walletOf(t, h, owner); got != 500 {
		t.Fatalf("**بعد الإلغاء %d والمنتظَر 500**", got)
	}
	// **والمعلَّقةُ يُلغيها المتجرُ قبل القبول** — ولا مالَ يتحرّك.
	if c := h.POST("/api/v1/merchant/deliveries/"+b+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("إلغاءُ المعلَّقة: %d / %s", c.Code, c.Err())
	}
	if got := walletOf(t, h, owner); got != 500 {
		t.Fatalf("بعد إلغاء المعلَّقة %d والمنتظَر 500", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDA04_MerchantFeeSetting **`delivery.merchant_fee` — وصفرُه يعود إلى العامّة.**
func TestMDA04_MerchantFeeSetting(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDA-04")
	quote := func() float64 {
		q := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
		if q.Code != http.StatusOK {
			t.Fatalf("عرضُ السعر: %d / %s", q.Code, q.Err())
		}
		fee, _ := q.JSON()["fee"].(float64)
		return fee
	}

	h.Setting("delivery.merchant_fee", "700")
	if got := quote(); got != 700 {
		t.Fatalf("عرضُ السعر %v والمنتظَر 700 (أجرةُ «لدي توصيلة»)", got)
	}
	id := m.create(t, h, "recipient", uniq("mda04a"))
	var fee, drv, cash int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, driver_fee, cash_due FROM orders WHERE id = $1`, id).
		Scan(&fee, &drv, &cash)
	if fee != 700 || drv != 630 || cash != 700 {
		t.Fatalf("أجرةٌ %d · للسائق %d · نقدٌ %d — والمنتظَرُ 700 و630 و700", fee, drv, cash)
	}

	h.Setting("delivery.merchant_fee", "0")
	if got := quote(); got != 500 {
		t.Fatalf("صفرٌ ⇒ الأجرةُ العامّة: عرضُ السعر %v والمنتظَر 500", got)
	}
	id2 := m.create(t, h, "recipient", uniq("mda04b"))
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee FROM orders WHERE id = $1`, id2).Scan(&fee)
	if fee != 500 {
		t.Fatalf("صفرٌ ⇒ الأجرةُ العامّة: %d والمنتظَر 500", fee)
	}
}

// TestMDA05_AutoAcceptSweepCharges **القبولُ التلقائيُّ بقواعده — وبلا فاعلٍ يُخصم أيضاً.**
func TestMDA05_AutoAcceptSweepCharges(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDA-05")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 1000)
	id := m.create(t, h, "merchant", uniq("mda05"))
	h.Setting("orders.auto_accept_min", "1")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET created_at = now() - interval '70 seconds' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	h.Orders.SweepAutoAcceptOnce(ctxBG())
	if st := h.statusOf(id); st != "dispatching" && st != "assigned" {
		t.Fatalf("بعد القبول التلقائيّ الحال %s والمنتظَر dispatching", st)
	}
	if got := walletOf(t, h, owner); got != 500 {
		t.Fatalf("**القبولُ التلقائيُّ لم يخصم**: %d والمنتظَر 500", got)
	}
}
