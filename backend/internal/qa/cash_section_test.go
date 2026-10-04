package qa

// ══════════════════════════════════════════════════════════════════════
// **قسمُ «النقد والصندوق» — قراراتُ المالك ٢٠٢٦-١٠-٠٤** (`CASH-*`)
// ══════════════════════════════════════════════════════════════════════

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// seedDriverCash **حركةٌ في صندوق سائقٍ بتاريخٍ معلوم** — الصندوقُ وقيودُه معاً
// (فيبقى FI-10.a سليماً). والموجبُ «تعديل» بلا مرجعِ طلب، والسالبُ تسليم.
func seedDriverCash(t *testing.T, h *Harness, driverID string, amount int64, at time.Time) {
	t.Helper()
	kind := "adjustment"
	if amount < 0 {
		kind = "settlement"
	}
	ctx := ctxBG()
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO driver_cash_boxes (driver_id) VALUES ($1) ON CONFLICT (driver_id) DO NOTHING`,
		driverID); err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	if _, err := h.Pool.Exec(ctx, `
		UPDATE driver_cash_boxes SET held = held + $2 WHERE driver_id = $1`,
		driverID, amount); err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO driver_cash_entries (driver_id, amount, kind, created_at)
		VALUES ($1, $2, $3, $4)`, driverID, amount, kind, at); err != nil {
		t.Fatalf("القيد: %v", err)
	}
}

func cashHolderOf(t *testing.T, h *Harness, token, driverID string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/admin/cash/outstanding", token)
	if r.Code != 200 {
		t.Fatalf("كشفُ النقد: %s", r)
	}
	list, _ := r.JSON()["holders"].([]any)
	for _, x := range list {
		m, _ := x.(map[string]any)
		if m["driver_id"] == driverID {
			return m
		}
	}
	t.Fatalf("السائقُ %s غائبٌ عن كشف النقد: %s", driverID, r)
	return nil
}

func ageDays(t *testing.T, v any) float64 {
	t.Helper()
	s, _ := v.(string)
	if s == "" {
		return -1
	}
	at, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t.Fatalf("تاريخ: %v", err)
	}
	return time.Since(at).Hours() / 24
}

// ── المشكلة ١ · «منذ» لا تصفّرها التسليمةُ الجزئيّة ─────────────────────

func TestCASH_SinceSurvivesPartialHandover(t *testing.T) {
	h := New(t)
	fin := h.NewUser("finance")
	now := time.Now()

	// قبض مئةً قبل أسبوع، وسلّم خمسين اليوم ⇒ الخمسون الباقية عمرُها أسبوع.
	a := h.NewUser("driver")
	seedDriverCash(t, h, a.ID, 100_000, now.Add(-7*24*time.Hour))
	seedDriverCash(t, h, a.ID, -50_000, now)
	if d := ageDays(t, cashHolderOf(t, h, fin.Token, a.ID)["oldest_at"]); d < 6.9 {
		t.Errorf("**التسليمُ الجزئيُّ صفّر القِدَم**: منذ %.1f يوم والباقي عمرُه أسبوع", d)
	}

	// والأقدمُ يُسدَّد أوّلاً: ستّون قبل أسبوع + أربعون قبل يوم، وسُلّم سبعون
	// ⇒ الستّون سُدّدت كلُّها، والباقي من قبضِ أمس.
	b := h.NewUser("driver")
	seedDriverCash(t, h, b.ID, 60_000, now.Add(-7*24*time.Hour))
	seedDriverCash(t, h, b.ID, 40_000, now.Add(-24*time.Hour))
	seedDriverCash(t, h, b.ID, -70_000, now)
	if d := ageDays(t, cashHolderOf(t, h, fin.Token, b.ID)["oldest_at"]); d < 0.9 || d > 1.1 {
		t.Errorf("**الأقدمُ لم يُسدَّد أوّلاً**: منذ %.1f يوم والمنتظَر يوم", d)
	}
}

// ── المشكلة ١٠ · السقفُ يُعرض كما يمنع: الجيبُ + نقدُ الطلبات المفتوحة ──

func TestCASH_CapCountsOpenOrderCash(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 500_000)
	seedDriverCash(t, h, fx.Driver.ID, 400_000, time.Now())
	oid, _ := cashOrder(t, h, fx.Cust, 50_000, 3)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET driver_id = $2::uuid, cash_due = 150000
		WHERE id = $1::uuid`, oid, fx.Driver.ID); err != nil {
		t.Fatal(err)
	}
	m := cashHolderOf(t, h, fx.Admin.Token, fx.Driver.ID)
	if m["open_cash"] != float64(150_000) || m["exposure"] != float64(550_000) {
		t.Errorf("نقدُ الطلبات المفتوحة لا يُحسب: %v", m)
	}
	if m["over_limit"] != true {
		t.Errorf("**ممنوعٌ من النقديّ والصفحةُ تعرضه تحت السقف**: %v", m)
	}
	ov := h.GET("/api/v1/admin/overview", fx.Admin.Token)
	aw, _ := ov.JSON()["awaiting"].(map[string]any)
	if n, _ := aw["drivers_over_cash"].(float64); n < 1 {
		t.Errorf("الرئيسيّةُ لا تعدّه فوق السقف: %s", ov)
	}
}

// ── التنبيهُ بعد أيّام + الإيقافُ الاختياريّ (مطفأٌ افتراضاً) ──────────────

func TestCASH_OverdueAlertAndOptionalStop(t *testing.T) {
	h := New(t)
	fx := newCashFixture(t, h, 5_000_000)
	h.Setting("drivers.cash_overdue_days", "3")
	seedDriverCash(t, h, fx.Driver.ID, 10_000, time.Now().Add(-4*24*time.Hour))

	if m := cashHolderOf(t, h, fx.Admin.Token, fx.Driver.ID); m["overdue"] != true {
		t.Errorf("أربعةُ أيّامٍ والتنبيهُ بعد ثلاثة — لم يُعلَّم متأخّراً: %v", m)
	}
	ov := h.GET("/api/v1/admin/overview", fx.Admin.Token)
	aw, _ := ov.JSON()["awaiting"].(map[string]any)
	if n, _ := aw["drivers_cash_overdue"].(float64); n < 1 {
		t.Errorf("**الرئيسيّةُ لا تنبّه للمتأخّر بالتسليم**: %s", ov)
	}

	// مُشعَلاً ⇒ لا يُسنَد إليه نقديّ.
	h.Setting("drivers.cash_overdue_stop", "true")
	oid, _ := cashOrder(t, h, fx.Cust, 10_000, 1)
	got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID})
	if !strings.Contains(string(got.Body), "errors.cash_overdue") {
		t.Errorf("**الإيقافُ مُشعَلٌ وأُسنِد إليه نقديّ**: %s", got)
	}
	// ومطفأً (الافتراض) ⇒ يُسنَد.
	h.Setting("drivers.cash_overdue_stop", "false")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/assign", fx.Admin.Token,
		map[string]any{"driver_id": fx.Driver.ID}); got.Code >= 400 {
		t.Errorf("الإيقافُ مطفأٌ ورُدّ الإسناد: %s", got)
	}
}

// والاحتياطُ في دالّة القاعدة = افتراضُ الفهرس.
func TestCASH_OverdueDefaultsMatchCatalog(t *testing.T) {
	if d := settings.Default("drivers.cash_overdue_days"); d != 3 {
		t.Fatalf("افتراضُ الفهرس %d والهجرةُ تحتاط بثلاثة", d)
	}
	raw, err := os.ReadFile("../migrate/migrations/0330_cash_section.sql")
	if err != nil {
		t.Skip(err)
	}
	if !strings.Contains(string(raw), "'drivers.cash_overdue_days'), 3)") ||
		!strings.Contains(string(raw), "('drivers.cash_overdue_stop', 'false')") {
		t.Error("احتياطُ الهجرة لا يطابق افتراضَ الفهرس")
	}
}

// ── المشكلة ٣ · مستحقّاتُ المتجر النقديّة لا تختفي بتغيير طريقته ─────────

func TestCASH_MerchantDuesListedRegardlessOfMethod(t *testing.T) {
	h := New(t)
	treasury(t, h)
	fin := h.NewUser("finance")
	item := h.NewItem(20_000)
	// متجرٌ يُسوّى نقداً يومَ الطلب.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'cash' WHERE id = $1`,
		item.MerchantID); err != nil {
		t.Fatal(err)
	}
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))

	var due int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount - reversed_amount), 0)
		FROM merchant_settlements WHERE merchant_id = $1 AND method = 'cash' AND state = 'cash_due'`,
		item.MerchantID).Scan(&due)
	if due <= 0 {
		t.Fatalf("لا مستحقَّ نقديّاً بعد التسليم — التجهيزُ لم يقع")
	}
	// حُوّل المتجرُ إلى المحفظة بعد أن صار له مستحقٌّ نقديّ.
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'wallet' WHERE id = $1`,
		item.MerchantID); err != nil {
		t.Fatal(err)
	}
	r := h.GET("/api/v1/admin/cash/merchant-dues", fin.Token)
	if r.Code != 200 {
		t.Fatalf("مستحقّاتُ المتاجر: %s", r)
	}
	list, _ := r.JSON()["merchants"].([]any)
	for _, x := range list {
		m, _ := x.(map[string]any)
		if m["merchant_id"] == item.MerchantID {
			if m["outstanding"] != float64(due) || m["settlement_method"] != "wallet" {
				t.Errorf("المستحقّ %v والمنتظَر %d بطريقة wallet", m, due)
			}
			return
		}
	}
	t.Errorf("**مستحقُّ المتجر اختفى لأنّ طريقته صارت محفظة**: %s", r)
}

// ── المشكلة ٥ · كشفُ صندوق السائق يحمل رقمَ الطلب ─────────────────────

func TestCASH_StatementCarriesOrderNumber(t *testing.T) {
	h := New(t)
	treasury(t, h)
	fin := h.NewUser("finance")
	item := h.NewItem(20_000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)

	r := h.GET("/api/v1/admin/drivers/"+drv.ID+"/cash", fin.Token)
	entries, _ := r.JSON()["entries"].([]any)
	if len(entries) == 0 {
		t.Fatalf("لا قيود: %s", r)
	}
	e, _ := entries[0].(map[string]any)
	if n, _ := e["order_number"].(float64); n <= 0 {
		t.Errorf("**قيدُ التحصيل بلا رقم طلب**: %v", e)
	}
}

// ── المشكلة ١٤ · إشعارُ السائق بالفواصل والعملة ─────────────────────────

func TestCASH_SettleNotificationFormatsMoney(t *testing.T) {
	h := New(t)
	fin := h.NewUser("finance")
	drv := h.NewUser("driver")
	seedDriverCash(t, h, drv.ID, 156_650, time.Now())
	if r := h.POSTKey("/api/v1/admin/drivers/"+drv.ID+"/settle", fin.Token, uniq("s"),
		map[string]any{"amount": 50_000, "note": ""}); r.Code != 200 {
		t.Fatalf("الاستلام: %s", r)
	}
	var body string
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT body FROM notifications WHERE user_id = $1
			ORDER BY created_at DESC LIMIT 1`, drv.ID).Scan(&body)
		if body != "" {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(body, "50,000") || !strings.Contains(body, "106,650") ||
		!strings.Contains(body, "ل.س") {
		t.Errorf("**الإشعارُ بأرقامٍ خام**: %q", body)
	}
}
