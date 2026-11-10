package server

// **دفعة ٢٠٢٦-١١-١٠ الصغيرة** — ما بيتسلّم الطلب وطلب تعديل الكابتن بانتظار المكتب، والكابتن يسحب طلبه،
// وتحويل شام كاش أكبر من المطلوب بيتقبل ومعلَّم.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

func (f *driverFixture) withdrawCall(orderID, driverID, requestID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/driver/orders/"+orderID+"/edit-request/withdraw",
		strings.NewReader(`{"request_id":"`+requestID+`"}`))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDriverWithdrawEditRequest(w, req.WithContext(ctx))
	return w
}

func TestEditRequest_PendingBlocksDeliveryUntilWithdrawn(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ctx := context.Background()
	f.setSetting(t, "drivers.require_delivery_photo", false)
	id := f.problemOrderAt(t, "at_pickup", d)
	// **متجرٌ وبندٌ للطلب** (عُدّة السائق بلا بنود) — ثمّ طلب التعديل بمرحلةٍ يقبلها.
	var merchant string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, owner_user_id, commission_percent, settlement_method, status)
		VALUES ('متجر اختبار', (SELECT id FROM categories LIMIT 1), $1, 10, 'cash', 'active') RETURNING id::text`,
		d).Scan(&merchant); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, merchant) })
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, merchant_id, qty, options, merchant_settlement_method)
		VALUES ($1, 'صنف', 20000, 20000, $2, 1, '[]'::jsonb, 'cash')`, id, merchant); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET merchant_id = $2 WHERE id = $1`, id, merchant); err != nil {
		t.Fatal(err)
	}
	v, err := f.srv.orders.RequestOrderEdit(context.Background(), id, d, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "ماء", Qty: 1, Price: 500}},
	})
	if err != nil {
		t.Fatalf("طلب التعديل: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'at_dropoff', picked_up_at = now() WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	w := f.transitionKeyed(id, d, `{"to":"delivered"}`, "")
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), "edit_request_pending_deliver") {
		t.Fatalf("التسليم مع طلب معلّق: %d %s — يُنتظر 409", w.Code, w.Body.String())
	}
	if st, _ := f.orderRow(t, id); st != "at_dropoff" {
		t.Fatalf("الطلب تحرّك رغم الطلب المعلّق: %s", st)
	}
	if w := f.withdrawCall(id, d, v.ID); w.Code != http.StatusOK {
		t.Fatalf("السحب %d: %s", w.Code, w.Body.String())
	}
	if w := f.transitionKeyed(id, d, `{"to":"delivered"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("التسليم بعد السحب %d: %s", w.Code, w.Body.String())
	}
}

func TestShamcash_AmountAboveDueIsAcceptedAndFlagged(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	flagged := func(id string) (bool, int64) {
		var ex bool
		var due *int64
		if err := f.pool.QueryRow(context.Background(),
			`SELECT exceeds_due, due_at_entry FROM shamcash_movements WHERE order_id = $1`, id).Scan(&ex, &due); err != nil {
			t.Fatalf("قراءة السجلّ: %v", err)
		}
		if due == nil {
			return ex, -1
		}
		return ex, *due
	}
	// **الطلب ٢٥٬٠٠٠**: أكبر منه ⇒ مقبول ومعلَّم.
	over := f.shamcashOrder(t, d)
	if w := f.transitionKeyed(over, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":30000}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("مبلغ أكبر من المطلوب رُفض %d: %s", w.Code, w.Body.String())
	}
	if ex, due := flagged(over); !ex || due != 25000 {
		t.Fatalf("علم أكبر من المطلوب=%v والمطلوب=%d — يُنتظر true و25000", ex, due)
	}
	// **ومساوٍ أو أقل ⇒ بلا علم.**
	same := f.shamcashOrder(t, d)
	if w := f.transitionKeyed(same, d, `{"to":"delivered","shamcash":{"currency":"SYP","amount":25000}}`, ""); w.Code != http.StatusOK {
		t.Fatalf("%d %s", w.Code, w.Body.String())
	}
	if ex, _ := flagged(same); ex {
		t.Fatal("المبلغ المساوي للمطلوب انعلّم")
	}
}
