package orders_test

// قسمُ «العروض والخصومات» — قراراتُ المالك ٢٠٢٦-١٠-٠٤ في محرّك الطلب.
//
//   - كلفةُ التوصيل المجّانيّ تُكتب مع الطلب (`promo_delivery_waived`).
//   - خصمُ الصنف يُكتب في البند (`offer_cut` و`offer_borne_by`).
//   - كودُ النسبة يُقصّ بسقفه بالليرة.
//   - «مرّةً لكلّ مستخدم» و«أوّلُ طلب» بالرقم: من حذف حسابَه وعاد بالرقم
//     نفسِه لا يأخذ الكودَ مرّةً ثانية.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func (f *fixture) placeWithCode(t *testing.T, itemID, code string) *orders.Order {
	t.Helper()
	o, err := f.svc.Create(context.Background(), f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "عروض",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
		PromoCode:     code,
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
	})
	return o
}

func (f *fixture) armCode(t *testing.T, code, kind string, value int64, maxDiscount *int64, firstOnly, once bool) {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO promo_codes (code, kind, value, max_discount, first_order_only, once_per_user, active)
		VALUES ($1, $2, $3, $4, $5, $6, true) RETURNING id`,
		code, kind, value, maxDiscount, firstOnly, once).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ الكود: %v", err)
	}
	t.Cleanup(func() {
		ctx := context.Background()
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_redemptions WHERE promo_id = $1`, id)
		_, _ = f.pool.Exec(ctx, `DELETE FROM promo_codes WHERE id = $1`, id)
	})
}

// TestPROMOS_FreeDeliveryCostRecorded **كلفةُ التوصيل المجّانيّ مكتوبةٌ مع الطلب.**
func TestPROMOS_FreeDeliveryCostRecorded(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	f.armTreasury(t)
	itemID := f.armItem(t)
	f.armFreeDeliveryPromo(t, "PFREE"+f.customer[:4])
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ('delivery.fee', '5000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = excluded.value`); err != nil {
		t.Fatal(err)
	}
	o := f.placeWithCode(t, itemID, "PFREE"+f.customer[:4])
	var fee, driverFee, waived, subtotal, discount, total int64
	if err := f.pool.QueryRow(ctx, `
		SELECT delivery_fee, driver_fee, promo_delivery_waived, subtotal, discount, total
		FROM orders WHERE id = $1`, o.ID).Scan(&fee, &driverFee, &waived, &subtotal, &discount, &total); err != nil {
		t.Fatalf("قراءةُ الطلب: %v", err)
	}
	if fee != 0 || waived != driverFee || waived <= 0 {
		t.Fatalf("التوصيل %d والمُعفى %d وأجرُ السائق %d — **كلفةُ التوصيل المجّانيّ لم تُكتب**",
			fee, waived, driverFee)
	}
	if total != subtotal+fee-discount {
		t.Fatalf("المجموع %d ≠ %d + %d − %d", total, subtotal, fee, discount)
	}
}

// TestPROMOS_ItemDiscountRecordedOnLine **خصمُ الصنف ومن يتحمّله في البند.**
func TestPROMOS_ItemDiscountRecordedOnLine(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	base := f.placeOne(t, itemID, "بلا خصم")
	_, _, baseUnit, _ := f.orderMoney(t, base.ID)

	f.svc.SetOffers(stubDiscount{itemID: itemID, percent: 20, borneBy: offers.ByPlatform})
	t.Cleanup(func() { f.svc.SetOffers(nil) })
	cut := f.placeOne(t, itemID, "بخصم")
	_, _, cutUnit, _ := f.orderMoney(t, cut.ID)

	var offerCut int64
	var by *string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT offer_cut, offer_borne_by FROM order_items WHERE order_id = $1`, cut.ID).
		Scan(&offerCut, &by); err != nil {
		t.Fatalf("قراءةُ البند: %v", err)
	}
	if offerCut != baseUnit-cutUnit || offerCut <= 0 || by == nil || *by != offers.ByPlatform {
		t.Fatalf("خصمُ البند %d ومن يتحمّله %v — والمنتظَر %d على المنصّة", offerCut, by, baseUnit-cutUnit)
	}
}

// TestPROMOS_PercentCapInSYP **كودُ النسبة لا يتجاوز سقفَه بالليرة.**
func TestPROMOS_PercentCapInSYP(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	capSYP := int64(2000)
	code := "PCAP" + f.customer[:4]
	f.armCode(t, code, "percent", 50, &capSYP, false, false)
	o := f.placeWithCode(t, itemID, code)
	var discount int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT discount FROM orders WHERE id = $1`, o.ID).Scan(&discount); err != nil {
		t.Fatal(err)
	}
	if discount != capSYP {
		t.Fatalf("الخصم %d والسقف %d — **السقفُ بالليرة لم يُطبَّق**", discount, capSYP)
	}
}

// TestPROMOS_PercentOver90Refused **القاعدةُ ترفض نسبةً فوق ٩٠** — كانت ١٥٠
// تُقبَل فيأكل الخصمُ التوصيلَ وينكسر فحصُ الدفتر.
func TestPROMOS_PercentOver90Refused(t *testing.T) {
	pool := testdb.Pool(t)
	_, err := pool.Exec(context.Background(), `
		INSERT INTO promo_codes (code, kind, value, max_discount) VALUES ('P150X', 'percent', 150, 1000)`)
	if err == nil {
		_, _ = pool.Exec(context.Background(), `DELETE FROM promo_codes WHERE code = 'P150X'`)
		t.Fatal("**قُبلت نسبةُ ١٥٠ في القاعدة**")
	}
}

// TestPROMOS_OncePerPhoneAndFirstOrderByPhone **الرقمُ لا الحساب.**
func TestPROMOS_OncePerPhoneAndFirstOrderByPhone(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	once := "PONE" + f.customer[:4]
	first := "PFST" + f.customer[:4]
	f.armCode(t, once, "fixed", 1000, nil, false, true)
	f.armCode(t, first, "fixed", 1000, nil, true, false)
	f.placeWithCode(t, itemID, once)

	var phone string
	if err := f.pool.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, f.customer).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	if err := identity.NewRepo(f.pool).AnonymizeUser(ctx, f.customer); err != nil {
		t.Fatalf("الحذف: %v", err)
	}
	again := testdb.NewUser(t, f.pool, "customer")
	if _, err := f.pool.Exec(ctx, `UPDATE users SET phone = $2 WHERE id = $1`, again, phone); err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{once, first} {
		p, err := f.svc.PreviewPromo(ctx, code, again, 30000, 5000)
		if err != nil {
			t.Fatalf("المعاينة: %v", err)
		}
		if p.Valid {
			t.Fatalf("**الكود %s صلح لحسابٍ جديدٍ بالرقم نفسِه** — والمالكُ ربطه بالرقم", code)
		}
	}
}
