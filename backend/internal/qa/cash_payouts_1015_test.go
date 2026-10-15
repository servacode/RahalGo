package qa

// ══════════════════════════════════════════════════════════════════════
// **صفحةُ نقد السائقين — قراراتُ المالك ٢٠٢٦-١٠-١٥**
// ══════════════════════════════════════════════════════════════════════
//
//	جيبٌ فارغٌ ونقدٌ في الطريق   ← يظهر في الصفحة وفي «فوق السقف»
//	عند السقف تماماً             ← ليس «فوقه» (الحارسُ يقبل ≤)
//	المتأخّرُ بالتسليم           ← `cash_overdue` لا «بلغت سقف النقد»
//	«مستحقّ للمتاجر نقداً»       ← رقمٌ واحدٌ في الخزينة وتبويب المتاجر
//	كشفُ السائق                  ← يُقلَّب، وبطلباته المفتوحة، وبسقفه الخاصّ
//	رفضُ طلب الشحن               ← يصل صاحبَ المحفظة بسببه

import (
	"strings"
	"testing"
	"time"
)

// openCashOn **طلبٌ مفتوحٌ بيد السائق بنقدٍ معلوم** — كما في `TestCASH_CapCountsOpenOrderCash`.
func openCashOn(t *testing.T, h *Harness, fx cashFixture, cash int64) string {
	t.Helper()
	oid, _ := cashOrder(t, h, fx.Cust, 10_000, 1)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET driver_id = $2::uuid, cash_due = $3
		WHERE id = $1::uuid`, oid, fx.Driver.ID, cash); err != nil {
		t.Fatal(err)
	}
	return oid
}

func overCount(t *testing.T, h *Harness, token string) float64 {
	t.Helper()
	r := h.GET("/api/v1/admin/cash/outstanding", token)
	if r.Code != 200 {
		t.Fatalf("كشفُ النقد: %s", r)
	}
	n, _ := r.JSON()["over_count"].(float64)
	return n
}

// ── ١ · جيبُه صفرٌ ونقدُ طلباته المفتوحة فوق السقف ⇒ يظهر ويُعدّ ──────────

func TestCASH1015_ZeroHeldOverByOpenCashListed(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 500_000)
	before := overCount(t, h, fx.Admin.Token)
	openCashOn(t, h, fx, 500_001)

	m := cashHolderOf(t, h, fx.Admin.Token, fx.Driver.ID) // يفشل إن غاب
	if m["held"] != float64(0) || m["open_cash"] != float64(500_001) {
		t.Errorf("المحمولُ والمفتوح: %v", m)
	}
	if m["over_limit"] != true {
		t.Errorf("**ممنوعٌ بنقدٍ في الطريق والصفحةُ لا تعدّه فوق السقف**: %v", m)
	}
	if after := overCount(t, h, fx.Admin.Token); after != before+1 {
		t.Errorf("**«فوق السقف» %v قبل و%v بعد** — يُنتظر زيادةُ واحد", before, after)
	}
}

// ── ٢ · عند السقف تماماً ليس فوقه ───────────────────────────────────────

func TestCASH1015_AtLimitIsNotOver(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 500_000)
	before := overCount(t, h, fx.Admin.Token)
	seedDriverCash(t, h, fx.Driver.ID, 300_000, time.Now())
	openCashOn(t, h, fx, 200_000)

	m := cashHolderOf(t, h, fx.Admin.Token, fx.Driver.ID)
	if m["exposure"] != float64(500_000) {
		t.Fatalf("التعرّض %v — يُنتظر السقفُ تماماً", m["exposure"])
	}
	if m["over_limit"] != false || m["at_limit"] != true {
		t.Errorf("**عند السقف تماماً وعُدّ فوقه** (الحارسُ يقبل ≤): %v", m)
	}
	if after := overCount(t, h, fx.Admin.Token); after != before {
		t.Errorf("**«فوق السقف» زاد بسائقٍ عند السقف**: %v ⇒ %v", before, after)
	}
}

// ── ٣ · الموقوفُ بالتأخّر يقرأ سببَه ────────────────────────────────────

func TestCASH1015_OverdueStopSaysOverdue(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 5_000_000)
	h.Setting("drivers.cash_overdue_days", "3")
	h.Setting("drivers.cash_overdue_stop", "true")
	seedDriverCash(t, h, fx.Driver.ID, 10_000, time.Now().Add(-4*24*time.Hour))

	oid, _ := cashOrder(t, h, fx.Cust, 10_000, 1)
	got := h.POST("/api/v1/driver/orders/"+oid+"/accept", fx.Driver.Token, nil)
	if got.Err() != "cash_overdue" {
		t.Fatalf("**المتأخّرُ بالتسليم قرأ غيرَ سببه**: %s — يُنتظر cash_overdue", got)
	}
}

// ── ٤ · «مستحقّ للمتاجر نقداً» رقمٌ واحد ─────────────────────────────────

func TestCASH1015_MerchantCashDueOneNumber(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")

	newOrder := func(cash bool) (string, string) {
		item := h.NewItem(20_000)
		if cash {
			if _, err := h.Pool.Exec(ctxBG(),
				`UPDATE merchants SET settlement_method = 'cash' WHERE id = $1`, item.MerchantID); err != nil {
				t.Fatal(err)
			}
		}
		made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
		if made.Code >= 400 {
			t.Fatalf("الطلب: %s", made)
		}
		oid, _ := made.JSON()["id"].(string)
		return oid, item.MerchantID
	}

	// مستحقٌّ قائم (احتباسٌ بلا دفع).
	o1, m1 := newOrder(false)
	seedLegacyCashDue(t, h, o1, m1, 18_000)
	// ومستحقٌّ دفعه المكتب.
	o2, m2 := newOrder(false)
	seedLegacyCashDue(t, h, o2, m2, 11_000)
	var sid string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT id::text FROM merchant_settlements
		WHERE order_id = $1 AND method = 'cash'`, o2).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	if r := h.POST("/api/v1/admin/merchant-cash-settlements/"+sid+"/pay", admin.Token,
		map[string]any{"note": "دُفع من الدرج"}); r.Code != 200 {
		t.Fatalf("دفعُ المكتب: %s", r)
	}
	// ومستحقٌّ دفعه السائقُ عند الاستلام (المسارُ الحقيقيّ).
	o3, _ := newOrder(true)
	deliverOrder(t, h, o3, h.driverOf(o3))
	var byDriver int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM merchant_settlements
		WHERE order_id = $1 AND paid_via = 'driver'`, o3).Scan(&byDriver)
	if byDriver == 0 {
		t.Fatalf("لا مستحقَّ دفعه السائق — التجهيزُ لم يقع")
	}

	// **وقيدٌ في محفظة الاحتباس بلا صفّ** — ما يمسكه `FI-14.a`. **البطاقةُ تقرأ
	// الصفوف فلا يحرّكها**؛ وكانت تقرأ الرصيدَ فتفترق عن التبويب.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE wallets SET balance = balance + 7777 WHERE is_cash_holding`); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `UPDATE wallets SET balance = balance - 7777 WHERE is_cash_holding`)
	})

	ov := h.GET("/api/v1/admin/treasury/overview", admin.Token)
	if ov.Code != 200 {
		t.Fatalf("الخزينة: %s", ov)
	}
	card, _ := ov.JSON()["merchant_cash_due"].(float64)
	dues := h.GET("/api/v1/admin/cash/merchant-dues", admin.Token)
	if dues.Code != 200 {
		t.Fatalf("تبويبُ المتاجر: %s", dues)
	}
	tab, _ := dues.JSON()["total"].(float64)
	if card != tab {
		t.Fatalf("**«مستحقّ للمتاجر نقداً» رقمان**: الخزينة %v والتبويب %v", card, tab)
	}
	var rows int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount - reversed_amount), 0)
		FROM merchant_settlements WHERE method = 'cash' AND state = 'cash_due'`).Scan(&rows)
	if int64(card) != rows {
		t.Errorf("البطاقة %v والصفوفُ %d", card, rows)
	}
}

// ── ٥ · كشفُ السائق: صفحاتٌ وطلباتٌ مفتوحةٌ وسقفُه الخاصّ ─────────────────

func TestCASH1015_DriverStatementPagesOpenOrdersOverride(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 500_000)
	for i := 0; i < 3; i++ {
		seedDriverCash(t, h, fx.Driver.ID, 1_000, time.Now())
	}
	oid := openCashOn(t, h, fx, 42_000)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET cash_limit_override = 77000 WHERE id = $1`,
		fx.Driver.ID); err != nil {
		t.Fatal(err)
	}
	r := h.GET("/api/v1/admin/drivers/"+fx.Driver.ID+"/cash?limit=2&page=2", fx.Admin.Token)
	if r.Code != 200 {
		t.Fatalf("الكشف: %s", r)
	}
	b := r.JSON()
	if b["limit"] != float64(77_000) || b["limit_is_override"] != true {
		t.Errorf("**السقفُ المعروضُ غيرُ سقفه الخاصّ**: %v / %v", b["limit"], b["limit_is_override"])
	}
	entries, _ := b["entries"].([]any)
	if b["total"] != float64(3) || b["page"] != float64(2) || len(entries) != 1 {
		t.Errorf("الترقيم: المجموع %v الصفحة %v والسطور %d — يُنتظر 3 و2 و1", b["total"], b["page"], len(entries))
	}
	open, _ := b["open_orders"].([]any)
	found := false
	for _, x := range open {
		o, _ := x.(map[string]any)
		if o["order_id"] == oid && o["expected_cash"] == float64(42_000) {
			found = true
		}
	}
	if !found || b["open_cash"] != float64(42_000) {
		t.Errorf("**الطلبُ المفتوحُ غائبٌ عن الكشف**: %v", open)
	}
}

// ── ٦ · رفضُ طلب الشحن يصل صاحبَه ───────────────────────────────────────

func TestWALLETREQ1015_RejectedTopupNotifies(t *testing.T) {
	h := New(t)
	treasury(t, h)
	fin := h.NewUser("finance")
	approver := h.NewUser("admin")
	target := h.Factory().NewUserWith("customer")

	r := walletPropose(h, fin, target.ID, map[string]any{"amount": 5000, "kind": "topup", "note": "شحن نقدي"})
	if r.Code != 201 {
		t.Fatalf("الاقتراح: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if rej := h.POST("/api/v1/admin/wallet-requests/"+reqID+"/reject", approver.Token,
		map[string]any{"note": "المبلغ ما وصل الصندوق"}); rej.Code != 200 {
		t.Fatalf("الرفض: %s", rej)
	}
	var title, body string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT title, body FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC LIMIT 1`, target.ID).Scan(&title, &body)
		if title != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if title != "انرفض طلب شحن محفظتك" || !strings.Contains(body, "المبلغ ما وصل الصندوق") {
		t.Fatalf("**الرفضُ لم يصل صاحبَ المحفظة بسببه**: %q / %q", title, body)
	}
}
