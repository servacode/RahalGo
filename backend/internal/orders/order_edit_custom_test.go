package orders_test

// ══════════════════════════════════════════════════════════════════════
// **صنفٌ من خارج القائمة** (قرارُ المالك ٢٠٢٦-١١-٠٣ — بطاقةُ لوحة الطلبات)
// ══════════════════════════════════════════════════════════════════════
//
// المكتبُ يكتب اسمَه وسعرَه. **سعرُ المتجر = سعرُ الزبون، وعمولةُ المتجر تُقتطع منه كأيّ
// سطر** (`EditCustom`). يُقاس: السطرُ بلا صنفٍ في القائمة، والمجموعُ والنقد، وحقُّ المتجر
// قبل الاستلام وبعده، ثمّ التسليمُ وثوابتُ الدفتر خضراء (ومنها FI-14.h).

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ── قبل الاستلام: السطرُ يُكتب بسعرين متساويين وبلا صنف، والحدودُ تُردّ ──────
func TestOrderEdit_CustomLine_BeforePickup(t *testing.T) {
	f := iuSetup(t, "accepted", "cash") // ٢٦٠٠
	ctx := context.Background()
	edit := func(req orders.EditRequest) (orders.EditResult, error) {
		_, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, req, false, nil)
		return res, err
	}
	// **الحدود**: بلا اسم، وبلا سعر، ومتجرٌ ليس في الطلب.
	if _, err := edit(orders.EditRequest{Reason: "س", Custom: []orders.EditCustom{{Name: "  ", Qty: 1, Price: 100}}}); err == nil {
		t.Fatal("صنفٌ بلا اسم قُبل")
	}
	if _, err := edit(orders.EditRequest{Reason: "س", Custom: []orders.EditCustom{{Name: "ثلج", Qty: 1, Price: 0}}}); err == nil {
		t.Fatal("صنفٌ بلا سعر قُبل")
	}
	other, _ := (&cashFixture{pool: f.pool, categoryID: f.categoryOf(t)}).merchant(t, "cash", 10)
	if _, err := edit(orders.EditRequest{Reason: "س", Custom: []orders.EditCustom{
		{Name: "ثلج", Qty: 1, Price: 100, MerchantID: other}}}); !errors.Is(err, orders.ErrOrderEditStore) {
		t.Fatalf("متجرٌ ليس في الطلب: %v", err)
	}
	if st := f.read(t); st.total != 2_600 {
		t.Fatalf("تغيّر الطلبُ بعد ردود: %+v", st)
	}

	res, err := edit(orders.EditRequest{Reason: "طلبه الزبون", Custom: []orders.EditCustom{{Name: "كيس ثلج", Qty: 2, Price: 250}}})
	if err != nil {
		t.Fatalf("الإضافة: %v", err)
	}
	if res.NewTotal != 3_100 || len(res.Changes) != 1 || res.Changes[0].NewQty != 2 || res.Changes[0].OldQty != 0 {
		t.Fatalf("النتيجة: %+v — يُنتظر 3100", res)
	}
	if st := f.read(t); st.subtotal != 3_000 || st.total != 3_100 || st.cashDue != 3_100 {
		t.Fatalf("بعد الإضافة: %+v", st)
	}
	var unit, cost int64
	var menu *string
	var merchant string
	if err := f.pool.QueryRow(ctx, `
		SELECT unit_price, merchant_price, menu_item_id::text, merchant_id::text
		FROM order_items WHERE order_id = $1 AND name = 'كيس ثلج'`, f.orderID).
		Scan(&unit, &cost, &menu, &merchant); err != nil {
		t.Fatalf("السطر: %v", err)
	}
	if unit != 250 || cost != 250 || menu != nil || merchant != f.merchantID {
		t.Fatalf("السطر: %d/%d menu=%v merchant=%s — يُنتظر 250/250 بلا صنف ومتجرُ الطلب", unit, cost, menu, merchant)
	}
	v, err := fininv.Run(ctx, f.pool, "FI-03.b")
	if err != nil || len(v) != 0 {
		t.Fatalf("FI-03.b: %v %+v", err, v)
	}
}

// ── قبل الاستلام ثمّ تسليم: حقُّ المتجر = (البضاعة + الصنف) − نسبته ───────────
func TestOrderEdit_CustomLine_ThenDeliverGreen(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	ctx := context.Background()
	if _, _, err := f.svc.EditOrder(ctx, f.driver, oid, orders.EditRequest{
		Reason: "قبل الاستلام", Custom: []orders.EditCustom{{Name: "صنف خارجي", Qty: 1, Price: 5_000}},
	}, false, nil); err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	if s := f.settlement(t, oid, m); s.amount != 49_500 || s.state != "cash_paid" {
		t.Fatalf("التسوية: %+v — يُنتظر 49500 مدفوعة", s)
	}
	f.assertDriverPayGreen(t)
	f.green(t, "بعد التسليم")
}

// ── بعد الاستلام لمتجرٍ «نقداً»: السائقُ يدفع الفرق، والعمولةُ على الصنف ───────
func TestOrderEditAP_CashStoreCustomLine(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	heldBefore := f.held(t)
	f.pickup(t, oid) // دفع السائقُ ٤٥٬٠٠٠

	res := f.edit(t, oid, orders.EditRequest{Custom: []orders.EditCustom{{Name: "صنف خارجي", Qty: 1, Price: 5_000}}})
	if !res.AfterPickup || len(res.Stores) != 1 || res.Stores[0].DriverPays != 4_500 || res.Stores[0].NewDue != 49_500 {
		t.Fatalf("فرقُ المتجر: %+v — يُنتظر 4500 يدفعها السائق", res)
	}
	if total, cash, _ := f.orderRow(t, oid); total != 65_000 || cash != 65_000 {
		t.Fatalf("المجموع %d والنقد %d — يُنتظر 65000", total, cash)
	}
	f.green(t, "بعد التعديل")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	if got := f.held(t) - heldBefore; got != 65_000-49_500 {
		t.Fatalf("صندوقُ السائق زاد %d لا %d", got, 65_000-49_500)
	}
	var commission int64
	_ = f.pool.QueryRow(context.Background(), `SELECT platform_commission FROM orders WHERE id = $1`, oid).Scan(&commission)
	if commission != 5_500 {
		t.Fatalf("عمولةُ المنصّة %d لا 5500", commission)
	}
	f.green(t, "بعد التسليم")
}

// ── بعد الاستلام لمتجرٍ «محفظة»: يُقيَّد الفرقُ في محفظته، ثمّ يُحذف الصنفُ فيُستردّ ──
func TestOrderEditAP_WalletStoreCustomLine(t *testing.T) {
	f := newCashFixture(t)
	m, owner := f.merchant(t, "wallet", 10)
	oid := f.order(t, m, 50_000, 10_000, "wallet")
	f.pickup(t, oid) // ٤٥٬٠٠٠ في محفظته

	res := f.edit(t, oid, orders.EditRequest{Custom: []orders.EditCustom{{Name: "صنف خارجي", Qty: 2, Price: 1_000}}})
	if len(res.Stores) != 1 || res.Stores[0].NewDue != 46_800 {
		t.Fatalf("حقُّ المتجر: %+v — يُنتظر 46800", res.Stores)
	}
	if got := f.balance(t, owner); got != 46_800 {
		t.Fatalf("محفظةُ المتجر %d لا 46800", got)
	}
	f.green(t, "بعد الإضافة")
	line := f.exec(t, `SELECT id::text FROM order_items WHERE order_id = $1 AND name = 'صنف خارجي'`, oid)
	res = f.edit(t, oid, orders.EditRequest{StoreRefunded: true, Lines: []orders.EditLine{{ItemID: line, Qty: 0}}})
	if res.Stores[0].NewDue != 45_000 {
		t.Fatalf("بعد الحذف: %+v — يُنتظر 45000", res.Stores)
	}
	if got := f.balance(t, owner); got != 45_000 {
		t.Fatalf("محفظةُ المتجر %d لا 45000", got)
	}
	f.green(t, "بعد الحذف")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	f.green(t, "بعد التسليم")
}
