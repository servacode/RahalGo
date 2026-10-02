package server

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ══════════════════════════════════════════════════════════════════════
// **الزبونُ يرى «كابتن رحال غو» لا اسمَ السائق** (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// «ما بدّنا ينعرف اسم السائق، وإذا قدّم شكوى فرقمُ الطلب يكفي.» **والسائقُ يرى
// اسمَ الزبون** — فالحجبُ في اتّجاهٍ واحد.

// TestCustomerDriverLabel_OrderView **بطاقةُ الطلب عند الزبون بلا اسم السائق.**
func TestCustomerDriverLabel_OrderView(t *testing.T) {
	name := "سائقٌ باسمه"
	o := &orders.Order{ID: "x", DriverName: &name}

	cv := orders.ViewFor(orders.AudienceCustomer, o)
	if cv["driver_name"] != orders.CustomerDriverLabel {
		t.Errorf("الزبونُ يقرأ %v — والمقرَّرُ «%s»", cv["driver_name"], orders.CustomerDriverLabel)
	}
	dv := orders.ViewFor(orders.AudienceDriver, o)
	if dv["driver_name"] != name {
		t.Errorf("السائقُ يقرأ %v — وحجبُ اسمه عن نفسه ليس المقصود", dv["driver_name"])
	}
	ov := orders.ViewFor(orders.AudienceOps, o)
	if ov["driver_name"] != name {
		t.Errorf("الإدارةُ تقرأ %v — وهي التي تعرف السائقَ من رقم الطلب", ov["driver_name"])
	}
	// **وبلا سائقٍ لا يُخترَع واحد.**
	empty := orders.ViewFor(orders.AudienceCustomer, &orders.Order{ID: "y"})
	if v, ok := empty["driver_name"]; ok && v != nil && v != "" {
		t.Errorf("طلبٌ بلا سائقٍ يقول للزبون %v", v)
	}
}

// TestCustomerDriverLabel_OneText **نصٌّ واحدٌ في الحزمتين.**
func TestCustomerDriverLabel_OneText(t *testing.T) {
	if comms.CustomerDriverLabel != orders.CustomerDriverLabel {
		t.Fatalf("الحديثُ يقول «%s» والبطاقةُ «%s» — **والزبونُ يرى اسمين لسائقٍ واحد**",
			comms.CustomerDriverLabel, orders.CustomerDriverLabel)
	}
}

// TestCustomerDriverLabel_ChatPeer **رأسُ الحديث وسجلُّه عند الزبون بلا اسم السائق.**
func TestCustomerDriverLabel_ChatPeer(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	orderID := f.dispatchingOrder(t, 30_000, 5_000)
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET status = 'assigned', driver_id = $2 WHERE id = $1`,
		orderID, f.drivers[0]); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE users SET full_name = 'سائقٌ معروف' WHERE id = $1`, f.drivers[0]); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	var customer string
	if err := f.pool.QueryRow(ctx,
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	svc := comms.New(f.pool)
	p, err := svc.Permit(ctx, orderID, customer)
	if err != nil {
		t.Fatalf("الصلاحية: %v", err)
	}
	if p.PeerName != orders.CustomerDriverLabel {
		t.Errorf("رأسُ الحديث عند الزبون «%s» — والمقرَّرُ «%s»", p.PeerName, orders.CustomerDriverLabel)
	}
	// **والسائقُ يرى اسمَ الزبون** — لا الشعار.
	dp, err := svc.Permit(ctx, orderID, f.drivers[0])
	if err != nil {
		t.Fatalf("صلاحيةُ السائق: %v", err)
	}
	if dp.PeerName == orders.CustomerDriverLabel {
		t.Error("السائقُ يرى الشعارَ مكانَ اسم الزبون")
	}

	// **وسجلُّ المحادثات كذلك** — سطرٌ في الحديث يكفي ليظهر.
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body, driver_id)
		VALUES ($1, $2, 'driver', 'أنا في الطريق', $2)`, orderID, f.drivers[0]); err != nil {
		t.Fatalf("سطرُ الحديث: %v", err)
	}
	threads, err := svc.Threads(ctx, customer)
	if err != nil {
		t.Fatalf("السجلّ: %v", err)
	}
	found := false
	for _, th := range threads {
		if th.OrderID == orderID {
			found = true
			if th.Peer != orders.CustomerDriverLabel {
				t.Errorf("سجلُّ الزبون يسمّي السائق «%s»", th.Peer)
			}
		}
	}
	if !found {
		t.Fatal("الحديثُ غائبٌ عن سجلّ الزبون")
	}
}
