package orders_test

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ── الكابتن بيسحب طلبه المعلّق: بيفك منع التسليم، وإعادة النداء نجاح ─────────
func TestEditRequest_DriverWithdraw(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "ماء", Qty: 1, Price: 500}},
	})
	if p, err := f.svc.HasPendingEditRequest(ctx, oid); err != nil || !p {
		t.Fatalf("المعلّق ما انشاف: %v %v", p, err)
	}
	// **كابتن غيره ما بيسحب طلب غيره.**
	other := testdb.NewUser(t, f.pool, "driver")
	if err := f.svc.WithdrawEditRequest(ctx, other, oid, req.ID, nil); err == nil {
		t.Fatal("كابتن غريب سحب الطلب")
	}
	if err := f.svc.WithdrawEditRequest(ctx, f.driver, oid, req.ID, nil); err != nil {
		t.Fatalf("السحب: %v", err)
	}
	if got := f.requestStatus(t, req.ID); got != "withdrawn" {
		t.Fatalf("الحالة %q لا withdrawn", got)
	}
	if p, _ := f.svc.HasPendingEditRequest(ctx, oid); p {
		t.Fatal("لسا معلّق بعد السحب")
	}
	if err := f.svc.WithdrawEditRequest(ctx, f.driver, oid, req.ID, nil); err != nil {
		t.Fatalf("إعادة نداء السحب: %v", err)
	}
	// **والمكتب ما بيوافق على مسحوب.**
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, false, nil); !errors.Is(err, orders.ErrEditRequestNotPending) {
		t.Fatalf("موافقة على مسحوب: %v", err)
	}
	if total, _, _ := f.orderRow(t, oid); total != 60_000 {
		t.Fatalf("الطلب تغيّر بعد السحب: %d", total)
	}
}
