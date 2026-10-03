package server

// **التحويلُ والسائقُ واقفٌ عند المتجر الأوّل** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «لازم يتحوّل لطريقٍ
// جديد ويروح «استلمت الطلب» ونرجع لنقطة الذهاب للمتجر الجديد ونكمل المشوار») — **قِيس على
// جهازه**: بقي الطلبُ «وصلت المتجر» وهو عند المتجر القديم، **وزرُّ «استلمت الطلب» ظاهرٌ لمتجرٍ
// لم يصله**، ولا خبرَ يصله بالتحويل.

import (
	"context"
	"net/http"
	"strings"
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

// TestTransfer_OldStoreReportDoesNotFollowToNewStore **بلاغُ المتجر القديم لا يلحق السائقَ إلى
// الجديد** (تجربةُ القبول ٢٠٢٦-١٠-٠٣، مرّتين على المحاكي): «المتجر مغلق» عند الأوّل ← تحويل ←
// وصل الجديد ← **فبقيت اللوحةُ تقرأ البلاغَ القديم**، وفي التطبيق «وصل بلاغك للإدارة» بلا
// «استلمت الطلب». **والبلاغُ يخصّ وصولاً بعينه** — وصولٌ جديدٌ صفحةٌ بيضاء.
func TestTransfer_OldStoreReportDoesNotFollowToNewStore(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "at_pickup", d)

	if w := f.fail(d, orderID, "merchant_closed", ""); w.Code != http.StatusOK ||
		!strings.Contains(w.Body.String(), `"reported":true`) {
		t.Fatalf("البلاغُ ردّ %d: %s", w.Code, w.Body.String())
	}
	// **والبلاغُ قبل التحويل بدقيقة** — لا يتساوى الطابعان فيُخفيا العطب.
	if _, err := f.pool.Exec(ctx, `
		UPDATE audit_log SET created_at = created_at - interval '1 minute'
		WHERE action = 'driver.stage_report' AND entity_id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.ReportCode != "merchant_closed" {
		t.Fatalf("قبل التحويل البلاغُ %q — والمتوقّعُ merchant_closed", v.ReportCode)
	}

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
	if w := f.call(f.srv.handleTransferOrder, http.MethodPost,
		"/admin/orders/"+orderID+"/transfer", orderID, ops, []string{"ops"},
		`{"merchant_id":"`+other+`","note":"المتجرُ مغلق"}`); w.Code != http.StatusOK {
		t.Fatalf("التحويلُ ردّ %d: %s", w.Code, w.Body.String())
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.ReportCode != "" {
		t.Fatalf("في الطريق إلى الجديد البلاغُ %q — والمتوقّعُ لا شيء", v.ReportCode)
	}

	// **ووصل المتجرَ الجديد.**
	if w := f.transitionKeyed(orderID, d, `{"to":"at_pickup"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("الوصولُ إلى الجديد ردّ %d: %s", w.Code, w.Body.String())
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.ReportCode != "" {
		t.Fatalf("عند المتجر الجديد البلاغُ %q (%v) — **بلاغُ القديم لحقه**", v.ReportCode, v.ReportAt)
	}
	// **ولا شيءَ يمنع الاستلام.**
	if w := f.transitionKeyed(orderID, d, `{"to":"picked_up"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("الاستلامُ من الجديد ردّ %d: %s", w.Code, w.Body.String())
	}
	if st, _ := f.orderRow(t, orderID); st != "picked_up" {
		t.Fatalf("الحالُ %q — والمتوقّعُ picked_up", st)
	}
	// **وبلاغٌ جديدٌ بعد الاستلام يُقرأ كما كان** — المرشّحُ لا يُعمي اللوحة.
	if w := f.fail(d, orderID, "customer_cancelled_by_phone", ""); w.Code != http.StatusOK {
		t.Fatalf("البلاغُ بعد الاستلام ردّ %d: %s", w.Code, w.Body.String())
	}
	if v := f.srv.orders.DoorViewOf(ctx, orderID); v.ReportCode != "customer_cancelled_by_phone" {
		t.Fatalf("بلاغٌ بعد الاستلام قُرئ %q — والمرشّحُ لا يُعمي اللوحة", v.ReportCode)
	}
}
