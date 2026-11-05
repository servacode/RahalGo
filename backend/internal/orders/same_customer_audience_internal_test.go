package orders

// **دفعُ العرض في «للجميع» يقرأ حدَّ الزبون نفسَه الذي يقرؤه الطابور** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٤) — **فلا يرنّ هاتفُ سائقٍ بطلبٍ محجوبٍ عن طابوره.**
//
// **وهنا لا في `server`** — `queueAudience` و`publishSameCustomerGraceEnd` غيرُ مصدَّرتين.

import (
	"context"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/settings"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// sameCustomerArena **«للجميع» بلا قرب، حدُّ ٣، مهلةُ ٦٠ — وسائقان على الدوام، A وصّل
// لزبون الطلب ثلاثاً أمس.** **وكلُّ مفتاحٍ يعود كما وُجد** (مراجعةُ الفرع) — كان التنظيفُ
// يُعيد مفتاحين من أربعة، **فيبقى «للجميع» ومهلةُ ستّين ثانيةً لكلّ اختبارٍ بعده في
// القاعدة المشتركة.**
func sameCustomerArena(t *testing.T) (svc *Service, pool *pgxpool.Pool, a, b, ord string) {
	t.Helper()
	pool = testdb.Pool(t)
	ctx := context.Background()
	st := settings.NewStore(pool)
	set := func(key string, v any) {
		t.Helper()
		var before *string
		_ = pool.QueryRow(ctx, `SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&before)
		t.Cleanup(func() {
			c := context.Background()
			if before == nil {
				_, _ = pool.Exec(c, `DELETE FROM app_settings WHERE key = $1`, key)
				return
			}
			_, _ = pool.Exec(c, `UPDATE app_settings SET value = $2::jsonb WHERE key = $1`, key, *before)
		})
		if err := st.SetInternal(ctx, key, v); err != nil {
			t.Fatalf("%s: %v", key, err)
		}
	}
	set("drivers.assignment_mode", "queue")
	set("drivers.proximity_enabled", false)
	set("drivers.same_customer_weekly_max", 3)
	set("drivers.offer_timeout_sec", 60)
	svc = &Service{db: pool, logger: quietLogger(), pub: noopPublisher{}, settings: st}

	a = testdb.NewUser(t, pool, "driver")
	b = testdb.NewUser(t, pool, "driver")
	// **العزل**: سائقو تجاربَ سابقةٍ في القاعدة المشتركة خارجَ الدوام.
	if _, err := pool.Exec(ctx, `UPDATE users SET on_shift = (id = ANY($1::uuid[]))`,
		[]string{a, b}); err != nil {
		t.Fatal(err)
	}
	ord = seedAlertableOrder(t, pool)
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
	return svc, pool, a, b, ord
}

func audienceSet(t *testing.T, svc *Service, ord string) map[string]bool {
	t.Helper()
	ids, err := svc.queueAudience(context.Background(), ord)
	if err != nil {
		t.Fatal(err)
	}
	has := map[string]bool{}
	for _, id := range ids {
		has[id] = true
	}
	return has
}

func TestSameCustomer_QueueAudienceMatchesQueue(t *testing.T) {
	svc, _, a, b, ord := sameCustomerArena(t)
	has := audienceSet(t, svc, ord)
	if has[a] {
		t.Errorf("**الدفعُ يرنّ عند A وهو فوق حدّ الزبون في المهلة** — والطابورُ يحجبه")
	}
	if !has[b] {
		t.Errorf("B لا يصله الدفع: %v", has)
	}
}

// offerSpy **من وصله عرضٌ لأيّ طلب.**
type offerSpy struct {
	countingNotifier
	mu  sync.Mutex
	got map[string][]string // طلب ← سائقون
}

func (s *offerSpy) Notify(_ context.Context, in notifications.Input) {
	if in.Kind != notifications.KindOrderOffer {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.got == nil {
		s.got = map[string][]string{}
	}
	s.got[in.EntityID] = append(s.got[in.EntityID], in.UserID)
}

func (s *offerSpy) to(orderID, driverID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.got[orderID] {
		if d == driverID {
			return true
		}
	}
	return false
}

// TestSameCustomer_GraceEndPushesOverCapDriver **انقضاءُ المهلة يرنّ عند صاحب الحدّ** (مراجعةُ
// الفرع). كان دفعُ العرض يقع مرّةً عند النزول — **وA محجوبٌ عنه حينها** — **ثمّ إشارةٌ لشاشاتٍ
// مفتوحةٍ وحدَها عند الانقضاء.** **فإن لم يأخذه B لم يعلم A به إلّا وشاشتُه مفتوحة**، وانتظر
// الطلبُ الإدارة. **وB لا يرنّ ثانيةً** — وصله عرضُه عند النزول.
func TestSameCustomer_GraceEndPushesOverCapDriver(t *testing.T) {
	svc, pool, a, b, ord := sameCustomerArena(t)
	spy := &offerSpy{}
	svc.notify = spy
	ctx := context.Background()

	// **في المهلة لا شيء** — لم تنقضِ.
	svc.publishSameCustomerGraceEnd(ctx)
	if spy.to(ord, a) {
		t.Fatal("رنّ عند A والمهلةُ قائمة")
	}
	if _, err := pool.Exec(ctx, `UPDATE orders SET dispatched_at = now() - interval '61 seconds' WHERE id = $1`, ord); err != nil {
		t.Fatal(err)
	}
	svc.publishSameCustomerGraceEnd(ctx)
	if !spy.to(ord, a) {
		t.Fatal("**انقضت المهلةُ ولم يرنّ عند A** — لا يعلم بالطلب إلّا وشاشتُه مفتوحة")
	}
	if spy.to(ord, b) {
		t.Error("رنّ عند B ثانيةً — وصله عرضُه عند النزول")
	}
}
