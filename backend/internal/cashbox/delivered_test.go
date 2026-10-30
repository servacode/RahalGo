package cashbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// **كلُّ ما سلّمه السائقُ يُرى — لا النقديُّ وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٣٠): طلبٌ نقديٌّ له قيدٌ
// بذمّته، وطلبٌ من المحفظة بلا قيد، **وكلاهما في القائمة**؛ وما خارجَ المدّة أو لسائقٍ آخر لا يدخل.
func TestDelivered_ListsCashAndWalletOrders(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := cashbox.NewService(pool, settings.NewStore(pool))
	driver := testdb.NewUser(t, pool, "driver")
	other := testdb.NewUser(t, pool, "driver")
	customer := testdb.NewUser(t, pool, "customer")

	var merchant string
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id)
		VALUES ('متجرُ اختبار', (SELECT id FROM categories ORDER BY created_at LIMIT 1))
		RETURNING id`).Scan(&merchant); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}
	insert := func(drv, payment string, wallet, cash int64, at time.Time) string {
		t.Helper()
		var id string
		if err := pool.QueryRow(ctx, `
			INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
				payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
				platform_commission, delivered_at)
			VALUES ($1, $2, $3, 'delivered', 'عنوان اختبار',
				ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
				$4, 1000, 100, 1100, $5, $6, 145, $7)
			RETURNING id`, customer, merchant, drv, payment, wallet, cash, at).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ طلب: %v", err)
		}
		return id
	}
	now := time.Now()
	cashOrder := insert(driver, "cash", 0, 1100, now.Add(-2*time.Hour))
	walletOrder := insert(driver, "wallet", 1100, 0, now.Add(-1*time.Hour))
	insert(driver, "cash", 0, 1100, now.Add(-72*time.Hour)) // خارجَ المدّة
	insert(other, "cash", 0, 1100, now.Add(-1*time.Hour))   // سائقٌ آخر
	if err := svc.Collect(ctx, driver, 245, cashOrder, nil); err != nil {
		t.Fatalf("تحصيل: %v", err)
	}

	got, err := svc.Delivered(ctx, driver, now.Add(-24*time.Hour), now)
	if err != nil {
		t.Fatalf("Delivered: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("المسلَّمُ في المدّة = %d — والمتوقّعُ ٢ (نقديٌّ ومحفظة)", len(got))
	}
	if got[0].OrderID != walletOrder || got[1].OrderID != cashOrder {
		t.Fatalf("الترتيبُ من الأحدث خطأ: %s, %s", got[0].OrderID, got[1].OrderID)
	}
	if got[0].CashCollected != 0 || got[0].WalletPaid != 1100 {
		t.Errorf("طلبُ المحفظة: بذمّته %d ومن المحفظة %d", got[0].CashCollected, got[0].WalletPaid)
	}
	if got[1].CashCollected != 245 || got[1].Commission != 145 || got[1].DeliveryFee != 100 {
		t.Errorf("الطلبُ النقديّ: بذمّته %d عمولة %d أجرة %d", got[1].CashCollected, got[1].Commission, got[1].DeliveryFee)
	}
	if got[1].MerchantName != "متجرُ اختبار" {
		t.Errorf("اسمُ المتجر = %q", got[1].MerchantName)
	}
}
