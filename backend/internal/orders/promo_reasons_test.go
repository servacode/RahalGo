package orders

// ══════════════════════════════════════════════════════════════════════
// **لكلّ سببِ رفضٍ للكود رمزُه** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت كلُّها `invalid_promo`** — «كود الخصم غير صالح أو مستخدم سابقاً» — فمن طلب بأقلّ
// من الحدّ الأدنى لا يعرف أنّ طلباً أكبرَ يُصلحه. **والطلبُ والمعاينةُ يقولان الرمزَ نفسَه.**

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/pricing"
)

func promoCodeOf(err error) (string, map[string]any) {
	var e *httpx.AppError
	if errors.As(err, &e) {
		return e.Code, e.Details
	}
	return "", nil
}

// TestPromoReasons_EachRejectionHasItsCode **الغائبُ والموقوفُ والمنتهي والخالصُ والحدُّ الأدنى.**
func TestPromoReasons_EachRejectionHasItsCode(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name, setup, want string
	}{
		{"موقوف", `active = false`, "invalid_promo"},
		{"منتهٍ", `expires_at = now() - interval '1 hour'`, "promo_expired"},
		{"خلص", `max_uses = 1, used_count = 1`, "promo_exhausted"},
		{"دون الحدّ الأدنى", `min_order = 50000`, "promo_min_order"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc, code, customer, id := promoFixture(t, "percent", 10, 0, true)
			if _, err := svc.db.Exec(ctx, `UPDATE promo_codes SET `+c.setup+` WHERE id = $1`, id); err != nil {
				t.Fatal(err)
			}
			tx, err := svc.db.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = tx.Rollback(ctx) }()
			fee := int64(1000)
			_, _, vErr := svc.validatePromo(ctx, tx, code, customer, 30000, &fee)
			got, details := promoCodeOf(vErr)
			if got != c.want {
				t.Fatalf("**الرمزُ %q والمنتظَر %q** (%v)", got, c.want, vErr)
			}
			if c.want == "promo_min_order" && details["min_order"] != int64(50000) {
				t.Fatalf("الحدُّ الأدنى لا يُقال برقمه: %v", details)
			}
		})
	}
	t.Run("غائب", func(t *testing.T) {
		svc, _, customer, _ := promoFixture(t, "percent", 10, 0, true)
		tx, err := svc.db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		fee := int64(1000)
		_, _, vErr := svc.validatePromo(ctx, tx, "LA-YUJAD-ABADAN", customer, 30000, &fee)
		if got, _ := promoCodeOf(vErr); got != "invalid_promo" {
			t.Fatalf("كودٌ غائب: %q", got)
		}
	})
}

// TestPromoReasons_UsedAndFirstOrder **«استعملته قبل» و«للطلب الأوّل بس».**
func TestPromoReasons_UsedAndFirstOrder(t *testing.T) {
	ctx := context.Background()
	svc, code, customer, id := promoFixture(t, "percent", 10, 0, true)
	// **وطلبٌ سابقٌ للزبون** — بلا متجرٍ ولا بنود: يكفي ليُعدّ «له طلبات».
	var orderID string
	if err := svc.db.QueryRow(ctx, `
		INSERT INTO orders (customer_id, kind, status, address_text, dropoff, payment_method,
			subtotal, delivery_fee, total, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, 'custom', 'delivered', 'عنوان', ST_SetSRID(ST_MakePoint(39.0, 35.9), 4326)::geography,
			'cash', 1000, 0, 1000, 1000, 0, 0, $2, 1)
		RETURNING id::text`, customer, pricing.RepCommissionSourceFixed).Scan(&orderID); err != nil {
		t.Fatalf("تعذّر طلبٌ سابق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = svc.db.Exec(context.Background(), `DELETE FROM promo_redemptions WHERE order_id = $1`, orderID)
		_, _ = svc.db.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})
	check := func(setup, want string) {
		t.Helper()
		if _, err := svc.db.Exec(ctx, `UPDATE promo_codes SET `+setup+` WHERE id = $1`, id); err != nil {
			t.Fatal(err)
		}
		got, err := svc.PreviewPromo(ctx, code, customer, 30000, 1000)
		if err != nil {
			t.Fatal(err)
		}
		raw, _ := json.Marshal(got)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		if m["valid"] != false || m["reason"] != want {
			t.Fatalf("**المعاينةُ لا تقول السبب**: %s — والمنتظَر %s", raw, want)
		}
	}
	check(`first_order_only = true, once_per_user = false`, "promo_first_order_only")
	if _, err := svc.db.Exec(ctx, `INSERT INTO promo_redemptions (promo_id, order_id, user_id)
		VALUES ($1, $2, $3)`, id, orderID, customer); err != nil {
		t.Fatal(err)
	}
	check(`first_order_only = false, once_per_user = true`, "promo_already_used")
}

// TestPromoReasons_PreviewSaysMinAndFreeDelivery **الحدُّ الأدنى برقمه، والتوصيلُ المجانيّ باسمه.**
func TestPromoReasons_PreviewSaysMinAndFreeDelivery(t *testing.T) {
	ctx := context.Background()
	read := func(p any) map[string]any {
		raw, _ := json.Marshal(p)
		var m map[string]any
		_ = json.Unmarshal(raw, &m)
		return m
	}
	svc, code, customer, _ := promoFixture(t, "percent", 10, 50000, true)
	got, err := svc.PreviewPromo(ctx, code, customer, 30000, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if m := read(got); m["reason"] != "promo_min_order" || m["min_order"] != float64(50000) {
		t.Fatalf("**المعاينةُ لا تقول الحدَّ الأدنى**: %v", m)
	}
	svc, code, customer, _ = promoFixture(t, "free_delivery", 0, 0, true)
	got, err = svc.PreviewPromo(ctx, code, customer, 30000, 1500)
	if err != nil {
		t.Fatal(err)
	}
	if m := read(got); m["kind"] != "free_delivery" || m["delivery_waived"] != float64(1500) {
		t.Fatalf("**كودُ التوصيل المجانيّ يُقرأ «خصم ٠»**: %v", m)
	}
}
