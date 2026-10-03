package orders_test

// **آخرُ ظهورٍ للسائق يصل المكتبَ وحدَه.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: شارةُ السائق على مسار الطلب في اللوحة — اسمُه
// واتّصالٌ و«آخرُ ظهورٍ قبل كذا».)
//
// **والحقلُ موضعُ السائق ضمناً** — فلا يخرج لزبونٍ ولا متجرٍ ولا سائقٍ ولا
// مندوب، **ويصل المكتبَ من بابه** (`AudienceOps`).

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestDriverSeenAt_OpsOnly(t *testing.T) {
	pool := testdb.Pool(t)
	store := settings.NewStore(pool)
	svc := orders.NewService(pool, nil, wallet.NewService(pool),
		cashbox.NewService(pool, store), nil,
		slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	svc.SetSettings(store)
	ctx := context.Background()
	customer := testdb.NewUser(t, pool, "customer")
	driver := testdb.NewUser(t, pool, "driver")
	if _, err := pool.Exec(ctx, `
		UPDATE users SET last_location_at = now() - interval '7 minutes' WHERE id = $1`, driver); err != nil {
		t.Fatalf("تعذّر ضبطُ آخر موضع: %v", err)
	}

	mk := func(withDriver bool) string {
		var drv any
		if withDriver {
			drv = driver
		}
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (kind, customer_id, driver_id, status, address_text, dropoff,
				payment_method, custom_request, subtotal, delivery_fee, total, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
			VALUES ('custom', $1, $2, 'cancelled', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 'طلبُ اختبار آخر ظهور', 0, 0, 0, 0,
			`+qaSnapSQLX()+`)
			RETURNING id::text`, customer, drv).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}
	withID, withoutID := mk(true), mk(false)

	o, err := svc.GetByID(ctx, withID)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if o.DriverSeenAt == nil {
		t.Fatalf("طلبٌ بسائقٍ بلا آخرِ ظهور — **والشارةُ تقول «لم يظهر» عن سائقٍ ظهر قبل سبع دقائق**")
	}
	if ago := time.Since(*o.DriverSeenAt); ago < 6*time.Minute || ago > 8*time.Minute {
		t.Fatalf("آخرُ ظهورٍ قبل %v — والمكتوبُ سبعُ دقائق", ago)
	}
	none, err := svc.GetByID(ctx, withoutID)
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	if none.DriverSeenAt != nil {
		t.Fatalf("طلبٌ بلا سائقٍ يحمل آخرَ ظهور")
	}

	for _, a := range []orders.Audience{orders.AudienceCustomer, orders.AudienceMerchant,
		orders.AudienceDriver, orders.AudienceRep, orders.AudienceMerchantDelivery} {
		if _, leak := orders.ViewFor(a, o)["driver_seen_at"]; leak {
			t.Fatalf("driver_seen_at خرج إلى %s — **وهو موضعُ السائق ضمناً**", a)
		}
	}
	if _, ok := orders.ViewFor(orders.AudienceOps, o)["driver_seen_at"]; !ok {
		t.Fatalf("driver_seen_at لا يصل المكتب — **والشارةُ بلا آخرِ ظهور**")
	}
}
