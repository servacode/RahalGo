package server

// **التحويلُ والسائقُ واقفٌ عند المتجر الأوّل** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «لازم يتحوّل لطريقٍ
// جديد ويروح «استلمت الطلب» ونرجع لنقطة الذهاب للمتجر الجديد ونكمل المشوار») — **قِيس على
// جهازه**: بقي الطلبُ «وصلت المتجر» وهو عند المتجر القديم، **وزرُّ «استلمت الطلب» ظاهرٌ لمتجرٍ
// لم يصله**، ولا خبرَ يصله بالتحويل.

import (
	"context"
	"net/http"
	"testing"
)

func TestTransfer_AtStoreSendsDriverToNewStore(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "at_pickup", d)

	var other string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent)
		SELECT 'المتجر البديل', category_id, 10 FROM merchants WHERE id = $1
		RETURNING id`, f.merchantID).Scan(&other); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, other)
	})

	w := f.call(f.srv.handleTransferOrder, http.MethodPost,
		"/admin/orders/"+orderID+"/transfer", orderID, ops, []string{"ops"},
		`{"merchant_id":"`+other+`","note":"المتجرُ مغلق"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("التحويلُ ردّ %d: %s", w.Code, w.Body.String())
	}

	status, driver := f.orderRow(t, orderID)
	if status != "assigned" {
		t.Fatalf("الحالُ %q بعد التحويل — والمتوقّعُ assigned: يتّجه إلى المتجر الجديد", status)
	}
	if driver == nil || *driver != d {
		t.Fatalf("السائقُ تبدّل (%v) — والطلبُ يبقى معه", driver)
	}
	var events int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM order_events
		WHERE order_id = $1 AND from_status = 'at_pickup' AND to_status = 'assigned'`,
		orderID).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("أحداثُ الرجوع إلى الطريق %d — والمتوقّعُ واحد", events)
	}
	var told int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications
		WHERE user_id = $1 AND entity_id = $2 AND body LIKE '%المتجر البديل%'`,
		d, orderID).Scan(&told); err != nil {
		t.Fatal(err)
	}
	if told != 1 {
		t.Fatalf("أخبارُ التحويل للسائق %d — والمتوقّعُ واحد باسم المتجر الجديد", told)
	}
}
