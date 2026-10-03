package orders_test

// **سجلُّ الطلبات يُرشَّح بالتاريخ والمتجر والسائق — ويحمل آخرَ ظهورٍ للسائق.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: مُرشِّحاتُ السجلّ في الرابط، وشارةُ السائق على
// مسار الطلب.)
//
// # وما يُحرَس
//
//   - **اليومُ يومُ دمشق لا غرينتش** — طلبٌ بعد منتصف ليل دمشق بنصف ساعةٍ
//     يقع في اليوم التالي، **ولو كان في غرينتش ما زال أمس.**
//   - **والحدّان شاملان** — `to` يومٌ كاملٌ لا لحظةُ أوّله.
//   - **وتاريخٌ لا يُقرأ يُردّ** — لا يسقط المُرشِّحُ صامتاً فيُعرض السجلُّ كلُّه.
//   - **والمتجرُ والسائقُ يُرشِّحان معه.**
//   - **و`driver_seen_at` آخرُ موضعٍ وصل من السائق** — وفارغٌ بلا سائق.

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestList_DateRangeMerchantDriverAndSeenAt(t *testing.T) {
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
		UPDATE users SET last_location_at = now() - interval '5 minutes' WHERE id = $1`, driver); err != nil {
		t.Fatalf("تعذّر ضبطُ آخر موضع: %v", err)
	}

	var categoryID string
	if err := pool.QueryRow(ctx, `SELECT id FROM categories LIMIT 1`).Scan(&categoryID); err != nil {
		t.Fatalf("لا تصنيفات في القاعدة: %v", err)
	}
	mkMerchant := func(name string) string {
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO merchants (name, category_id, commission_percent)
			VALUES ($1, $2, 10) RETURNING id::text`, name, categoryID).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
		})
		return id
	}
	mA := mkMerchant("متجرُ مُرشِّح أ")
	mB := mkMerchant("متجرُ مُرشِّح ب")

	// **والأوقاتُ بتوقيت دمشق** — تُكتب بمنطقتها فلا يُخمَّن فرقُها.
	mk := func(merchant string, withDriver bool, at string) string {
		var drv any
		if withDriver {
			drv = driver
		}
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (kind, customer_id, merchant_id, driver_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, cash_due, created_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
			VALUES ('standard', $1, $2, $3, 'delivered', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				'cash', 0, 0, 0, 0, ($4::timestamp AT TIME ZONE 'Asia/Damascus'),
			`+qaSnapSQLX()+`)
			RETURNING id::text`, customer, merchant, drv, at).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
		return id
	}
	lateJan10 := mk(mA, true, "2026-01-10 23:30:00")   // **آخرُ يوم ١٠ بدمشق**
	earlyJan11 := mk(mA, false, "2026-01-11 00:30:00") // **وهو ما زال ١٠ في غرينتش**
	jan12B := mk(mB, true, "2026-01-12 12:00:00")

	ids := func(f orders.ListFilter) map[string]bool {
		t.Helper()
		f.CustomerID = customer
		f.PerPage = 50
		p, err := svc.List(ctx, f)
		if err != nil {
			t.Fatalf("تعذّرت القراءة (%+v): %v", f, err)
		}
		out := map[string]bool{}
		for _, o := range p.Orders {
			out[o.ID] = true
		}
		if p.Total != len(out) {
			t.Fatalf("المجموعُ %d والمعروضُ %d — **والعدُّ والقائمةُ بشرطٍ واحد**", p.Total, len(out))
		}
		return out
	}

	// ── يومُ دمشق لا يومُ غرينتش ──────────────────────────────────
	got := ids(orders.ListFilter{From: "2026-01-10", To: "2026-01-10"})
	if !got[lateJan10] || got[earlyJan11] || len(got) != 1 {
		t.Fatalf("يومُ ١٠: %v — **والمطلوبُ طلبُ آخر الليل وحدَه، لا ما بعد منتصف ليل دمشق**", got)
	}
	got = ids(orders.ListFilter{From: "2026-01-11"})
	if got[lateJan10] || !got[earlyJan11] || !got[jan12B] {
		t.Fatalf("من ١١: %v — **والحدُّ الأدنى شاملٌ بيوم دمشق**", got)
	}
	got = ids(orders.ListFilter{To: "2026-01-11"})
	if !got[lateJan10] || !got[earlyJan11] || got[jan12B] {
		t.Fatalf("إلى ١١: %v — **والحدُّ الأعلى يومٌ كاملٌ لا لحظةُ أوّله**", got)
	}

	// ── والمتجرُ والسائقُ ──────────────────────────────────────────
	got = ids(orders.ListFilter{MerchantID: mB})
	if len(got) != 1 || !got[jan12B] {
		t.Fatalf("متجرُ ب: %v", got)
	}
	got = ids(orders.ListFilter{DriverID: driver, From: "2026-01-10", To: "2026-01-11"})
	if len(got) != 1 || !got[lateJan10] {
		t.Fatalf("السائقُ في مدى ١٠–١١: %v", got)
	}

	// ── وتاريخٌ لا يُقرأ يُردّ ─────────────────────────────────────
	_, err := svc.List(ctx, orders.ListFilter{CustomerID: customer, From: "10/01/2026"})
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) || appErr.Status != http.StatusBadRequest {
		t.Fatalf("تاريخٌ لا يُقرأ ردّ %v — **والمُرشِّحُ الساقطُ صامتاً يعرض السجلَّ كلَّه**", err)
	}

	// ── وآخرُ ظهورٍ للسائق ─────────────────────────────────────────
	p, err := svc.List(ctx, orders.ListFilter{CustomerID: customer, PerPage: 50})
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	for _, o := range p.Orders {
		switch o.ID {
		case lateJan10, jan12B:
			if o.DriverSeenAt == nil {
				t.Fatalf("طلبٌ بسائقٍ بلا آخرِ ظهور — **والشارةُ تقول «لم يظهر» عن سائقٍ ظهر قبل خمس دقائق**")
			}
		case earlyJan11:
			if o.DriverSeenAt != nil {
				t.Fatalf("طلبٌ بلا سائقٍ يحمل آخرَ ظهور")
			}
		}
	}
	// **ولا يخرج لغير المكتب** — بابُ السماح.
	for _, a := range []orders.Audience{orders.AudienceCustomer, orders.AudienceMerchant,
		orders.AudienceDriver, orders.AudienceRep} {
		for i := range p.Orders {
			if _, leak := orders.ViewFor(a, &p.Orders[i])["driver_seen_at"]; leak {
				t.Fatalf("driver_seen_at خرج إلى %s", a)
			}
		}
	}
	if _, ok := orders.ViewFor(orders.AudienceOps, &p.Orders[0])["driver_seen_at"]; !ok {
		t.Fatalf("driver_seen_at لا يصل المكتب")
	}
}
