package orders

// **الإنذارُ بالقدرة لا باسم الدور — ويتجدّد حين يتبدّل السبب**
// (قراراتُ المالك ٢٠٢٦-١٠-٠٤، قسمُ «الطلبات»: البند ٦ · والمشكلتان ٢ و٩.)
//
// **وهنا لا في `orders_test`** — `escalate` غيرُ مصدَّرة، كأختها في
// `watchdog_r22_test.go`.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestStuckAlert_RoutesByCapabilityAndRealertsOnNewReason
//
//	العالقُ   ⇒ من يملك `orders.intervene` (موظّفُ `operations`) — لا الدعم
//	الطارئُ   ⇒ `orders.intervene` و`support.manage` معاً
//	سببٌ جديدٌ ⇒ إنذارٌ جديد · والسببُ نفسُه لا يُعاد
func TestStuckAlert_RoutesByCapabilityAndRealertsOnNewReason(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	ops := testdb.NewUser(t, pool, "operations")
	support := testdb.NewUser(t, pool, "customer_support")
	ns := notifications.New(pool, noopPublisher{}, quietLogger())
	svc := &Service{db: pool, logger: quietLogger(), notify: ns, pub: noopPublisher{}}
	oid := seedAlertableOrder(t, pool)

	count := func(user string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM notifications
			WHERE user_id = $1 AND entity = 'order' AND entity_id = $2`, user, oid).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	alert := Alert{OrderID: oid, Number: 1, Status: "pending", MerchantName: "متجر", Reason: StuckNoAccept}
	svc.escalate(ctx, []Alert{alert})
	if got := count(ops); got != 1 {
		t.Fatalf("**موظّفُ العمليّات (`operations`) لم يصله إنذارُ العالق** — وصله %d", got)
	}
	if got := count(support); got != 0 {
		t.Fatalf("الدعمُ وصله إنذارُ طلبٍ عالقٍ لا يملك التدخّلَ فيه: %d", got)
	}
	var href string
	_ = pool.QueryRow(ctx, `SELECT href FROM notifications WHERE user_id = $1 AND entity_id = $2`,
		ops, oid).Scan(&href)
	if href != "/dashboard/orders?id="+oid {
		t.Errorf("رابطُ الإنذار %q لا يفتح الطلبَ نفسَه", href)
	}

	// **والسببُ نفسُه لا يُعاد** — كلَّ ثلاثين ثانية.
	svc.escalate(ctx, []Alert{alert})
	if got := count(ops); got != 1 {
		t.Fatalf("أُعيد الإنذارُ بالسبب نفسِه: %d", got)
	}
	// **وسببٌ جديدٌ إنذارٌ جديد** — كان يُكتم إلى الأبد.
	alert.Reason = StuckNoDriver
	svc.escalate(ctx, []Alert{alert})
	if got := count(ops); got != 2 {
		t.Fatalf("**تبدّل السببُ (بلا سائق) ولم يصل إنذار** — العدد %d", got)
	}

	// **والطارئُ يصل الدعمَ والعمليّاتِ معاً.**
	ns.NotifyCaps(ctx, EmergencyAlertCaps, notifications.Input{
		Kind: notifications.KindOrder, Title: "طارئٌ لدى سائق", Entity: "order", EntityID: oid,
		Href: "/dashboard/emergencies",
	})
	if got := count(support); got != 1 {
		t.Fatalf("**الدعمُ لم يصله الطارئ**: %d", got)
	}
	if got := count(ops); got != 3 {
		t.Fatalf("العمليّاتُ لم يصلها الطارئ: %d", got)
	}
}
