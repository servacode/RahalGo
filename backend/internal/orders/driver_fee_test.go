package orders_test

// ══════════════════════════════════════════════════════════════════════
// **«توصيلٌ مجّانيّ» لا يُصفّر أجرَ السائق** — قرارُ المالك ٢٠٢٦-٠٩-٢٧
// ══════════════════════════════════════════════════════════════════════
//
// **الوعدُ على المنصّة لا على السائق.** كودُ `free_delivery` يُصفّر ما يدفعه
// الزبونُ (`delivery_fee`)، **وأجرُ السائقِ (`driver_fee`) يبقى أجرةَ التوصيل
// الأساسَ** المُلتقَطةَ لحظةَ الإنشاء قبل الخصم — **والفرقَ تموّله الخزينة.**
//
// **وهذا الاختبارُ يمشي المسارَ الحيَّ**: إنشاءٌ بكودٍ يُصفّر التوصيل، ثمّ
// تسليمٌ، **فيُقاس أنّ الزبونَ لم يُشحن توصيلاً، وأنّ السائقَ قبض أجرَه كاملاً،
// وأنّ الثابتَ FI-06.c لم يُخرَق** (أجرُ السائقِ المقيَّد = `driver_fee`).

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// armFreeDeliveryPromo كودُ توصيلٍ مجّانيٍّ — يُصفّر `delivery_fee` وحدَه.
func (f *fixture) armFreeDeliveryPromo(t *testing.T, code string) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO promo_codes (code, kind, value, once_per_user, max_uses, active)
		VALUES ($1, 'free_delivery', 0, false, NULL, true)
		RETURNING id`, code).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ كود التوصيل المجّانيّ: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_redemptions WHERE promo_id = $1`, id)
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1`, id)
	})
	return id
}

// TestFreeDelivery_DriverStillEarns **الزبونُ لا يدفع التوصيلَ والسائقُ يقبضه.**
func TestFreeDelivery_DriverStillEarns(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	f.armTreasury(t)
	itemID := f.armItem(t)
	f.armFreeDeliveryPromo(t, "FREEDEL1")

	// **وأجرةُ التوصيلِ مقطوعةٌ ٥٠٠٠** (`delivery.fee`، بلا مسافة): تُقرأ من
	// قاعدةِ التسعير لا من عمودِ الدائرة — **فتُبذَر صراحةً ليكون الأساسُ معلوماً.**
	const baseFee = 5000
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('delivery.fee', '5000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatalf("تعذّر ضبطُ أجرة التوصيل: %v", err)
	}

	// **صورةٌ للخرق القائم قبل الطلب** — فالقاعدةُ مشتركةٌ وقد تحمل خروقاً من
	// سيناريوهاتٍ أخرى؛ **ولا يُدان طلبُنا إلّا بما استجدّ به وحدَه.** وتشمل
	// عائلةَ الحفظ كلَّها (`FI-06`): **06.a تُثبت أنّ الخزينةَ موّلت الفرقَ**
	// (المدفوعُ = الأطرافُ + الخزينةُ)، و06.c أنّ أجرَ السائقِ = `driver_fee`.
	base, err := fininv.Capture(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatalf("تعذّرت صورةُ الأساس: %v", err)
	}

	o, err := f.svc.Create(ctx, f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "توصيلٌ مجّانيّ",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
		PromoCode:     "FREEDEL1",
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
	})

	// ── ما يدفعه الزبونُ توصيلاً = صفر · وأجرُ السائقِ المعتمَد = الأساس ──
	var deliveryFee, driverFee int64
	if err := f.pool.QueryRow(ctx,
		`SELECT delivery_fee, driver_fee FROM orders WHERE id = $1`, o.ID).
		Scan(&deliveryFee, &driverFee); err != nil {
		t.Fatalf("تعذّرت قراءةُ الأجرتين: %v", err)
	}
	if deliveryFee != 0 {
		t.Errorf("توصيلُ الزبون = %d، والمتوقّع 0 — **العرضُ لم يُصفّره**", deliveryFee)
	}
	if driverFee != baseFee {
		t.Fatalf("أجرُ السائقِ المعتمَد = %d، والمتوقّع %d — **صُفّر مع دفعِ الزبون**",
			driverFee, baseFee)
	}

	// ── يُسنَد السائقُ ويُسلَّم — كما تفعل العُدّةُ في مسارِ التسليم ──
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET driver_id = $2, status = 'at_dropoff' WHERE id = $1`,
		o.ID, f.driver); err != nil {
		t.Fatalf("تعذّر إسنادُ السائق: %v", err)
	}
	before := f.balance(t, f.driver)
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, o.ID, "delivered", ""); err != nil {
		t.Fatalf("التسليمُ فشل: %v", err)
	}

	// ── قبض السائقُ أجرَه كاملاً وإن لم يدفعه الزبون ──
	if got := f.balance(t, f.driver) - before; got != driverFee {
		t.Errorf("أجرُ السائق = %d، والمتوقّع %d (أجرُه المعتمَد) — **قاد مشوارَه في طلبٍ مجّانيٍّ وأخِذ غيرَه**",
			got, driverFee)
	}

	// ── وثوابتُ الحفظِ لم تُخرَق: الخزينةُ موّلت الفرقَ، والمقيَّدُ = `driver_fee` ──
	after, err := fininv.Run(ctx, f.pool, "FI-06")
	if err != nil {
		t.Fatalf("تعذّر فحصُ FI-06: %v", err)
	}
	if fresh := base.New(after); len(fresh) > 0 {
		t.Fatalf("عائلةُ الحفظ FI-06 استجدّ خرقُها بطلبِ التوصيل المجّانيّ: %v", fresh)
	}
}
