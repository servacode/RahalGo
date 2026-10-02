package orders_test

// **طلبُ تعويضٍ واحدٌ لكلّ (طلب، سائق)** — دورةُ ٢٠٢٦-٠٩-٢٧، ثمّ ٢٠٢٦-١٠-٠٢.
//
// **العلّةُ الأولى**: حظرُ المتجر (`merchant_blocked.go`) يعيد الطلبَ إلى
// `accepted` ويُعوّض السائق، **فإعادةُ توزيعِه ثمّ حظرُه ثانيةً كانت تُعوّض
// السائقَ نفسَه مرّتين.** **وصار التعويضُ بموافقة العمليات** (٢٠٢٦-١٠-٠٢) —
// فالحارسُ اليومَ في موضعين: **طلبٌ معلَّقٌ واحدٌ لكلّ (طلب، سائق)** (قيدٌ فريد)،
// **والموافقةُ لا تقع مرّتين** (`driver_already_compensated`، يُختبر في الخادم).
//
// **والوحدةُ (طلب، سائق) لا (طلب)**: سائقان مختلفان قادا مشوارَهما وحُظرا
// يستحقّان طلبَين — **وثابتُ `FI-05.a` يجمع بـ(ref,user_id).**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func compRows(t *testing.T, f *fixture, orderID, userID string) int64 {
	t.Helper()
	var n int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'compensation' AND user_id = $2`,
		orderID, userID).Scan(&n); err != nil {
		t.Fatalf("عدُّ التعويضات: %v", err)
	}
	return n
}

func assertFI05aGreen(t *testing.T, f *fixture) {
	t.Helper()
	v, err := fininv.Run(context.Background(), f.pool, "FI-05.a")
	if err != nil {
		t.Fatalf("تعذّر تشغيلُ FI-05.a: %v", err)
	}
	if len(v) != 0 {
		t.Fatalf("خُرق FI-05.a: %+v", v)
	}
}

// TestCompensation_MerchantBlockedTwiceSameDriver_CompensatesOnce
// **حظرٌ مرّتين للسائق نفسِه ⇒ تعويضٌ واحد** (الحارسُ الجديد).
func requestRows(t *testing.T, f *fixture, orderID, userID string) int64 {
	t.Helper()
	var n int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM driver_compensation_requests
		WHERE order_id = $1 AND driver_id = $2`, orderID, userID).Scan(&n); err != nil {
		t.Fatalf("عدُّ طلبات التعويض: %v", err)
	}
	return n
}

func TestCompensation_MerchantBlockedTwiceSameDriver_CompensatesOnce(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	for round := 1; round <= 2; round++ {
		if round == 2 {
			// ── يُعاد توزيعُ الطلب إلى السائق نفسِه ويُحظَر ثانيةً ──────────
			if _, err := f.pool.Exec(ctx,
				`UPDATE orders SET status = 'at_pickup', driver_id = $2 WHERE id = $1`,
				f.orderID, f.driver); err != nil {
				t.Fatalf("إعادةُ التوزيع: %v", err)
			}
		}
		if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
			f.orderID, "failed", "", "merchant_refused"); err != nil {
			t.Fatalf("الحظرُ %d فشل: %v", round, err)
		}
	}
	// **طلبٌ معلَّقٌ واحد — ولا قيدَ تعويضٍ بلا موافقة.**
	if n := requestRows(t, f, f.orderID, f.driver); n != 1 {
		t.Fatalf("طلباتُ التعويض للسائق نفسِه = %d، والمتوقّع 1", n)
	}
	if n := compRows(t, f, f.orderID, f.driver); n != 0 {
		t.Fatalf("قُيّد تعويضٌ بلا موافقة: %d صفّاً", n)
	}
	assertFI05aGreen(t, f)
}

func TestCompensation_MerchantBlockedDifferentDrivers_EachCompensatedOnce(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	driverB := testdb.NewUser(t, f.pool, "driver")

	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("حظرُ أ: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET status = 'at_pickup', driver_id = $2 WHERE id = $1`,
		f.orderID, driverB); err != nil {
		t.Fatalf("إعادةُ التوزيع إلى ب: %v", err)
	}
	if _, err := f.svc.TransitionWithReason(ctx, driverB, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("حظرُ ب: %v", err)
	}
	// **كلٌّ له طلبُه** — صفٌّ لكلّ سائق، ولا مالَ قبل الموافقة.
	if n := requestRows(t, f, f.orderID, f.driver); n != 1 {
		t.Fatalf("طلباتُ أ = %d", n)
	}
	if n := requestRows(t, f, f.orderID, driverB); n != 1 {
		t.Fatalf("طلباتُ ب = %d", n)
	}
	if a, b := f.balance(t, f.driver), f.balance(t, driverB); a != 0 || b != 0 {
		t.Fatalf("قُيّد بلا موافقة: أ=%d ب=%d", a, b)
	}
	assertFI05aGreen(t, f)
}
