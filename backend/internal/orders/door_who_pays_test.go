package orders_test

// ══════════════════════════════════════════════════════════════════════
// **المكتبُ يقرّر من يدفع عند الباب** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
//	ذنبُ السائق  المنصّةُ تتحمّل ⇒ لا دين · كاملاً ⇒ دينٌ بالخسارة · نسبة ٥٠ ⇒ نصفُها
//	ذنبُ الزبون  منعُ النقد ⇒ يُمنع · لا شيء ⇒ لا يُمنع
//
// **والخسارةُ ما قيّدته الخزينةُ على الطلب** — سالبُ مجموع `platform_profit` بمرجعه.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// failedByDriver **طلبٌ نقديٌّ استُلم من المتجر ثمّ أنهاه المكتبُ عند الباب بذنب السائق.**
func failedByDriver(t *testing.T, share string, percent int) (f *fixture, loss, debt int64) {
	t.Helper()
	ctx := context.Background()
	f = setup(t, "at_pickup", 100_000, 10_000, 0)
	f.armTreasury(t)
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
			t.Fatalf("الانتقالُ إلى %s: %v", to, err)
		}
	}
	ops := testdb.NewUser(t, f.pool, "operations")
	if _, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, f.orderID, orders.DoorResolution{
		Action: orders.DoorReturnToOffice, Fault: orders.FaultDriver, Reason: "customer_refused",
		Note: "تأخّر السائقُ ساعتين", DriverShare: share, DriverPercent: percent,
	}, nil); err != nil {
		t.Fatalf("الإنهاءُ رُدّ: %v", err)
	}
	dropReturnTrip(t, f.pool, f.orderID)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(),
			`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, f.driver)
		_, _ = f.pool.Exec(context.Background(), `UPDATE users SET driver_debt = 0 WHERE id = $1`, f.driver)
	})
	if err := f.pool.QueryRow(ctx, `
		SELECT -COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'platform_profit'`, f.orderID).Scan(&loss); err != nil {
		t.Fatal(err)
	}
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM financial_obligations
		 WHERE party_kind = 'driver' AND party_id = $1 AND cause = 'failed_order_driver'
		   AND source_ref = $2 AND order_id = $2::uuid`, f.driver, f.orderID).Scan(&debt); err != nil {
		t.Fatal(err)
	}
	return f, loss, debt
}

// TestDoor_DriverFault_WhoPays **من يتحمّل خسارةَ الطلب بذنب السائق.**
func TestDoor_DriverFault_WhoPays(t *testing.T) {
	cases := []struct {
		name    string
		share   string
		percent int
		want    func(loss int64) int64
	}{
		{"المنصة", orders.DriverSharePlatform, 0, func(int64) int64 { return 0 }},
		{"كاملاً", orders.DriverShareFull, 0, func(l int64) int64 { return l }},
		{"نسبة50", orders.DriverSharePercent, 50, func(l int64) int64 { return l * 50 / 100 }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, loss, debt := failedByDriver(t, c.share, c.percent)
			t.Logf("الخسارة %d · الدين %d", loss, debt)
			if loss <= 0 {
				t.Fatalf("لا خسارةَ على المنصّة (%d) — والطلبُ استُلم ولم يُقبض", loss)
			}
			if want := c.want(loss); debt != want {
				t.Fatalf("دينُ السائق %d والخسارةُ %d — والمتوقّع %d", debt, loss, want)
			}
		})
	}
}

// TestDoor_BadChoiceRefused **اختيارٌ لا يوافق الذنب يُردّ قبل أن يُنهى شيء.**
func TestDoor_BadChoiceRefused(t *testing.T) {
	cases := []orders.DoorResolution{
		{Fault: orders.FaultDriver, DriverShare: orders.DriverSharePercent, DriverPercent: 0},
		{Fault: orders.FaultDriver, DriverShare: orders.DriverSharePercent, DriverPercent: 100},
		{Fault: orders.FaultCustomer, DriverShare: orders.DriverShareFull},
		{Fault: orders.FaultDriver, CustomerAction: orders.CustomerBlock},
		{Fault: orders.FaultCustomer, CustomerAction: "jail"},
	}
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ops := testdb.NewUser(t, f.pool, "operations")
	for _, in := range cases {
		in.Action, in.Note = orders.DoorReturnToOffice, "اختبار"
		if _, err := f.svc.ResolveDoor(context.Background(), ops, []string{"ops"}, f.orderID,
			in, nil); !errors.Is(err, orders.ErrDoorBadChoice) {
			t.Fatalf("%+v مرّ (%v) — والمتوقّع door_bad_choice", in, err)
		}
	}
	if st := f.statusOf(t); st != "at_dropoff" {
		t.Fatalf("الحالُ %q بعد اختياراتٍ مردودة", st)
	}
}

// TestDoor_CustomerAction_CashBan **منعُ النقد يقع بالاختيار وحدَه** — «منع» يمنع،
// و«لا شيء» لا يمنع ولو كان الذنبُ ذنبَه.
func TestDoor_CustomerAction_CashBan(t *testing.T) {
	cases := []struct {
		action  string
		blocked bool
	}{
		{orders.CustomerCashBan, true},
		{orders.CustomerNone, false},
	}
	for _, c := range cases {
		t.Run(c.action, func(t *testing.T) {
			pool := testdb.Pool(t)
			ctx := context.Background()
			f := setup(t, "at_dropoff", 100_000, 10_000, 0)
			if _, err := pool.Exec(ctx, `
				INSERT INTO app_settings (key, value) VALUES
					('customers.cash_ban_failures', '1'::jsonb),
					('customers.cash_ban_days', '30'::jsonb)
				ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
				t.Fatalf("تعذّر ضبط المفاتيح: %v", err)
			}
			t.Cleanup(func() {
				_, _ = pool.Exec(context.Background(),
					`UPDATE app_settings SET value = '0'::jsonb WHERE key = 'customers.cash_ban_failures'`)
			})
			itemID, cleanup := arena(t, pool, f)
			defer cleanup()

			ops := testdb.NewUser(t, pool, "operations")
			if _, err := f.svc.ResolveDoor(ctx, ops, []string{"ops"}, f.orderID, orders.DoorResolution{
				Action: orders.DoorReturnToOffice, Fault: orders.FaultCustomer,
				Reason: "customer_refused", Note: "رفض الاستلام", CustomerAction: c.action,
			}, nil); err != nil {
				t.Fatalf("الإنهاءُ رُدّ: %v", err)
			}
			_, err := placeCash(ctx, f, itemID)
			if got := errors.Is(err, orders.ErrCashBlocked); got != c.blocked {
				t.Fatalf("مُنع النقد = %v (%v) والمتوقّع %v", got, err, c.blocked)
			}
		})
	}
}
