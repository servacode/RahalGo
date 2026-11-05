package orders

// **دفعُ العرض في «للجميع» يقرأ حدَّ الزبون نفسَه الذي يقرؤه الطابور** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٤) — **فلا يرنّ هاتفُ سائقٍ بطلبٍ محجوبٍ عن طابوره.**
//
// **وهنا لا في `server`** — `queueAudience` غيرُ مصدَّرة.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestSameCustomer_QueueAudienceMatchesQueue(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	st := settings.NewStore(pool)
	set := func(key string, v any) {
		t.Helper()
		if err := st.SetInternal(ctx, key, v); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	set("drivers.assignment_mode", "queue")
	set("drivers.proximity_enabled", false)
	set("drivers.same_customer_weekly_max", 3)
	set("drivers.offer_timeout_sec", 60)
	t.Cleanup(func() {
		_ = st.SetInternal(context.Background(), "drivers.proximity_enabled", true)
		_ = st.SetInternal(context.Background(), "drivers.same_customer_weekly_max", 3)
	})
	svc := &Service{db: pool, logger: quietLogger(), pub: noopPublisher{}, settings: st}

	a := testdb.NewUser(t, pool, "driver")
	b := testdb.NewUser(t, pool, "driver")
	// **العزل**: سائقو تجاربَ سابقةٍ في القاعدة المشتركة خارجَ الدوام.
	if _, err := pool.Exec(ctx, `UPDATE users SET on_shift = (id = ANY($1::uuid[]))`,
		[]string{a, b}); err != nil {
		t.Fatal(err)
	}
	ord := seedAlertableOrder(t, pool)
	if _, err := pool.Exec(ctx, `UPDATE orders SET status = 'dispatching', dispatched_at = now() WHERE id = $1`, ord); err != nil {
		t.Fatal(err)
	}
	var cust string
	if err := pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`, ord).Scan(&cust); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		old := seedAlertableOrder(t, pool)
		if _, err := pool.Exec(ctx, `
			UPDATE orders SET customer_id = $2, driver_id = $3, status = 'delivered',
			       picked_up_at = now() - interval '1 day 30 minutes',
			       delivered_at = now() - interval '1 day', closed_at = now() - interval '1 day'
			WHERE id = $1`, old, cust, a); err != nil {
			t.Fatalf("بذرُ تسليم: %v", err)
		}
	}

	ids, err := svc.queueAudience(ctx, ord)
	if err != nil {
		t.Fatal(err)
	}
	has := map[string]bool{}
	for _, id := range ids {
		has[id] = true
	}
	if has[a] {
		t.Errorf("**الدفعُ يرنّ عند A وهو فوق حدّ الزبون في المهلة** — والطابورُ يحجبه")
	}
	if !has[b] {
		t.Errorf("B لا يصله الدفع: %v", ids)
	}
}
