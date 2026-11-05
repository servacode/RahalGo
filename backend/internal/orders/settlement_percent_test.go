package orders_test

// ══════════════════════════════════════════════════════════════════════
// **نسبةُ المتجر تتثبّت في صفّ تسويته** — خلل PD-1 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **تعديلُ الطلب بعد الاستلام يُصحّح حقَّ المتجر بفرق البنود وحدَه** — ولا يُقيّد تبديلاً لنسبة
// العمولة وقع بعد الاستلام كأنّه تصحيحُ تعديل. (`correctSettlementsTx` يقرأ `merchantShares`،
// **و`merchantShares` كان يقرأ النسبةَ الحيّة.**)

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestSettlementPercent_AfterPickupEditBooksLinesOnly **استلامٌ بعشرة ← النسبةُ تصير خمسة ←
// صنفٌ بـ٥٬٠٠٠ يُضاف بعد الاستلام ⇒ الفرقُ ٤٬٥٠٠ (عشرةٌ على الصنف الجديد)** — لا ٧٬٢٥٠
// (خمسةٌ على الطلب كلّه ناقصَ ما دُفع). **ثمّ يُسلَّم والدفترُ أخضر.**
func TestSettlementPercent_AfterPickupEditBooksLinesOnly(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	add := editMenuItem(t, f, m, "إضافة", 5_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	f.pickup(t, oid) // دفع السائقُ ٤٥٬٠٠٠ بعشرة
	if pct := f.exec(t, `
		SELECT COALESCE(commission_percent::text, '') FROM merchant_settlements
		 WHERE order_id = $1 AND merchant_id = $2`, oid, m); pct != "10.00" {
		t.Errorf("نسبةُ صفّ الاستلام %q — يُنتظر 10.00", pct)
	}

	// **النسبةُ تتبدّل والبضاعةُ في يد السائق.**
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET commission_percent = 5 WHERE id = $1`, m); err != nil {
		t.Fatal(err)
	}

	res := f.edit(t, oid, orders.EditRequest{Add: []orders.EditAdd{{MenuItemID: add, Qty: 1}}})
	if len(res.Stores) != 1 || res.Stores[0].DriverPays != 4_500 || res.Stores[0].NewDue != 49_500 {
		t.Fatalf("فرقُ المتجر: %+v — يُنتظر 4500 (عشرةٌ على الصنف الجديد) لا فرقَ النسبة", res.Stores)
	}
	if got := f.adjusted(t, oid, m); got != 4_500 {
		t.Fatalf("adjusted_amount=%d — يُنتظر 4500", got)
	}
	f.green(t, "بعد التعديل")

	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	var commission int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT platform_commission FROM orders WHERE id = $1`, oid).Scan(&commission); err != nil {
		t.Fatal(err)
	}
	if commission != 5_500 {
		t.Fatalf("عمولةُ المنصّة %d — يُنتظر 5500 (عشرةٌ من 55000)", commission)
	}
	f.green(t, "بعد التسليم")
}
