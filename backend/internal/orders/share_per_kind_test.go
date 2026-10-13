package orders_test

// ══════════════════════════════════════════════════════════════════════
// **حصّةُ المنصّة لكلّ نوعٍ وحدَه — نسبةً أو مبلغاً** (قرارُ المالك ٢٠٢٦-١٠-١٣)
// ══════════════════════════════════════════════════════════════════════
//
// «مثلاً ١٥٠ بحيث الـ١٠٠ للسائق والـ٥٠ للمنصّة… إذا ٠ معناها كلّها للسائق».

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func setShare(t *testing.T, kind, mode, value string) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	for k, v := range map[string]string{
		"delivery.share_" + kind:           value,
		"delivery.share_" + kind + "_mode": `"` + mode + `"`,
	} {
		if _, err := pool.Exec(ctx, `INSERT INTO app_settings (key, value) VALUES ($1, $2::jsonb)
			ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`, k, v); err != nil {
			t.Fatal(err)
		}
		key := k
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM app_settings WHERE key = $1`, key)
		})
	}
}

// TestShare_AmanahFixedFiftyOfOneFifty **أمانةٌ ١٥٠٠ وحصّةٌ ثابتة ٥٠٠ ⇒ للسائق ١٠٠٠ وللمنصّة ٥٠٠ (حدُّ أجرة السائق الأدنى ١٠٠٠)** —
// **والمشترياتُ بلا حصّةٍ تبقى أجرتُها كاملةً للسائق.**
func TestShare_AmanahFixedFiftyOfOneFifty(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	setShare(t, "amanah", "fixed", "500")
	// **والمشترياتُ صفرٌ صراحةً** — `cqSetup` تضبط لها ١٠٪ لاختباراتٍ أخرى.
	setShare(t, "purchase", "percent", "0")

	amanah := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	setMode(t, amanah, orders.CustomModeAmanah)
	if err := svc.AgreeCustomStep(ctx, amanah, driver, orders.AgreeStepFee, 0, 1500); err != nil {
		t.Fatalf("أجرةُ الأمانة: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, amanah, customer, "cash", 1500, 1); err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(ctx, driver, []string{"driver"}, amanah, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
	var drv, fixed int64
	if err := testdb.Pool(t).QueryRow(ctx, `SELECT driver_fee, snap_platform_delivery_fixed FROM orders WHERE id = $1`,
		amanah).Scan(&drv, &fixed); err != nil {
		t.Fatal(err)
	}
	if drv != 1000 || fixed != 500 {
		t.Fatalf("أجرُ السائق %d واللقطة %d — والمنتظَرُ ١٠٠٠ و٥٠٠", drv, fixed)
	}
	if b, _ := w.Balance(ctx, driver); b != 1000 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 1000", b)
	}
	if got := cqTreasury(t); got != 500 {
		t.Fatalf("نصيبُ المنصّة=%d — المنتظَر 500", got)
	}

	// **والمشترياتُ لم تُضبط حصّتُها ⇒ الأجرةُ كلُّها للسائق.**
	purchase := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustomStep(ctx, purchase, driver, orders.AgreeStepFee, 0, 1500); err != nil {
		t.Fatalf("أجرةُ المشتريات: %v", err)
	}
	if err := testdb.Pool(t).QueryRow(ctx, `SELECT driver_fee FROM orders WHERE id = $1`, purchase).Scan(&drv); err != nil {
		t.Fatal(err)
	}
	if drv != 1500 {
		t.Fatalf("أجرُ سائق المشتريات %d — والمنتظَرُ ١٥٠٠ كاملة (لا حصّةَ لها)", drv)
	}
}
