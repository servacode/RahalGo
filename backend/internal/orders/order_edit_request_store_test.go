package orders_test

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// ── طلب بمتجرين: الكابتن بيختار متجر الصنف، وغير متجر الطلب بينرد ───────────
func TestEditRequest_AddedItemStorePick(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m1, _ := f.merchant(t, "cash", 10)
	m2, _ := f.merchant(t, "cash", 10)
	outsider, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m1, 50_000, 10_000, "cash")
	f.addLine(t, oid, m2, "صنف المتجر الثاني", 5_000, 1)

	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "ماء", Qty: 1, Price: 500, MerchantID: outsider}}}); !errors.Is(err, orders.ErrOrderEditStore) {
		t.Fatalf("متجر خارج الطلب: %v", err)
	}
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{
			{Name: "ماء", Qty: 1, Price: 500, MerchantID: m2},
			{Name: "خبز", Qty: 1, Price: 700}, // بلا متجر ⇒ متجر الطلب الرئيسي
		},
	})
	if len(req.Adds) != 2 || req.Adds[0].MerchantID != m2 || req.Adds[0].MerchantName == "" {
		t.Fatalf("متجر الصنف ما انحفظ بالعرض: %+v", req.Adds)
	}
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, false, nil); err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	store := func(name string) string {
		return f.exec(t, `SELECT merchant_id::text FROM order_items WHERE order_id = $1 AND name = $2`, oid, name)
	}
	if got := store("ماء"); got != m2 {
		t.Fatalf("الماء راح لـ%s — يُنتظر المتجر الثاني %s", got, m2)
	}
	if got := store("خبز"); got != m1 {
		t.Fatalf("الخبز راح لـ%s — يُنتظر الرئيسي %s", got, m1)
	}
}
