package server

// **خلل PD-8 — لا تحويلَ لمتجرٍ آخر بعد ما استلم السائقُ البضاعة** (قرارُ المالك ٢٠٢٦-١١-٠٦ «أ»).
//
// الطارئُ بعد الاستلام يُعيد الطلبَ إلى «بانتظار سائق» (`picked_up → dispatching` بيد العمليات)
// **والبضاعةُ مع السائق الأوّل ومستحقُّ متجره مقيَّد.** وكان شرطُ التحويل بالحال وحدَها
// فيُقبل هنا — **فيكتب الاستلامُ الثاني تسويةً للمتجر الجديد وتبقى تسويةُ الأوّل.**

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// pickedThenRedispatched طلبٌ استلمه السائقُ فعلاً ثمّ أعادته العملياتُ إلى الطابور.
func (f *driverFixture) pickedThenRedispatched(t *testing.T, opsID, driverID string) string {
	t.Helper()
	orderID := f.problemOrderAt(t, "at_pickup", driverID)
	if w := f.transitionKeyed(orderID, driverID, `{"to":"picked_up"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("الاستلامُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if _, err := f.srv.orders.Transition(context.Background(), opsID, []string{"ops"}, orderID,
		orders.StDispatching, "طارئ لدى السائق"); err != nil {
		t.Fatalf("الإعادةُ إلى الطابور: %v", err)
	}
	return orderID
}

func (f *driverFixture) otherStore(t *testing.T) string {
	t.Helper()
	var other string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO merchants (name, category_id, commission_percent)
		SELECT 'متجر بديل بعد الاستلام', category_id, 10 FROM merchants WHERE id = $1
		RETURNING id`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	return other
}

func (f *driverFixture) orderMerchantKind(t *testing.T, orderID string) (merchant, kind string) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE(merchant_id::text, ''), kind FROM orders WHERE id = $1`, orderID).
		Scan(&merchant, &kind); err != nil {
		t.Fatal(err)
	}
	return merchant, kind
}

func TestTransfer_RefusedAfterPickupRedispatch(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	orderID := f.pickedThenRedispatched(t, ops, f.drivers[0])
	other := f.otherStore(t)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other)
	})

	w := f.call(f.srv.handleTransferOrder, http.MethodPost,
		"/admin/orders/"+orderID+"/transfer", orderID, ops, []string{"ops"},
		`{"merchant_id":"`+other+`","note":"المتجر مغلق"}`)
	if w.Code != http.StatusConflict || !strings.Contains(w.Body.String(), `"transfer_after_pickup"`) {
		t.Fatalf("التحويلُ بعد الاستلام ردّ %d: %s — المنتظَر 409 transfer_after_pickup",
			w.Code, w.Body.String())
	}
	if m := f.orderMerchant(t, orderID); m != f.merchantID {
		t.Fatalf("متجرُ الطلب صار %s — والمنتظَر أن يبقى %s", m, f.merchantID)
	}
}

func TestEXTSTORE_RefusedAfterPickupRedispatch(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	orderID := f.pickedThenRedispatched(t, ops, f.drivers[0])
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})

	w := f.call(f.srv.handleOrderExternalStore, http.MethodPost,
		"/admin/orders/"+orderID+"/external-store", orderID, ops, []string{"ops"},
		`{"reason":"المتجر ما عنده","store_name":"متجر الاختبار"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("القلبُ لمتجرٍ غير تابع بعد الاستلام ردّ %d: %s — المنتظَر 409", w.Code, w.Body.String())
	}
	if m, kind := f.orderMerchantKind(t, orderID); m != f.merchantID || kind != orders.KindStandard {
		t.Fatalf("الطلبُ صار (%s، %s) — والمنتظَر أن يبقى عاديّاً على متجره", m, kind)
	}
}
