package server

import (
	"context"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// ══════════════════════════════════════════════════════════════════════
// **عرضُ الطلب يصل هاتفاً تطبيقُه مغلق — ولا يملأ الصندوق** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **كان العرضُ عابراً** — لا صفَّ له، **وعاملُ النقل يقرأ الصفوف**: فلا دفعَ
// إلى هاتفٍ تطبيقُه مغلق. (قِيس على جهاز المالك.)

// offerRows **صفوفُ العرض لهذا السائق على هذا الطلب.**
func (f *driverFixture) offerRows(t *testing.T, driverID, orderID string) (n int, pending bool, ttl float64, collapse string) {
	t.Helper()
	var p *bool
	var left *float64
	var c *string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*), bool_or(push_pending),
		       max(EXTRACT(EPOCH FROM (expires_at - now()))), max(collapse_key)
		FROM notifications
		WHERE user_id = $1 AND entity_id = $2 AND kind = 'order_offer'`,
		driverID, orderID).Scan(&n, &p, &left, &c); err != nil {
		t.Fatalf("قراءةُ صفوف العرض: %v", err)
	}
	if p != nil {
		pending = *p
	}
	if left != nil {
		ttl = *left
	}
	if c != nil {
		collapse = *c
	}
	return
}

// TestOfferPush_RowQueuedForClosedApp **العرضُ صفٌّ يُدفَع — عاجلٌ بأجلِ مهلته.**
func TestOfferPush_RowQueuedForClosedApp(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.onShift(t, f.drivers[0], true)
	notif := notifications.New(f.pool, f.srv.hub, f.srv.logger)
	f.srv.orders.SetNotifier(notif)

	orderID := f.dispatchingOrder(t, 30_000, 5_000)
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("العرض: %v", err)
	}
	n, pending, ttl, collapse := f.offerRows(t, f.drivers[0], orderID)
	if n != 1 || !pending {
		t.Fatalf("صفوفُ العرض=%d · معلَّقٌ للدفع=%v — **فلا يصل هاتفاً تطبيقُه مغلق**", n, pending)
	}
	if ttl <= 0 || ttl > 61 {
		t.Errorf("أجلُ العرض %.1fث — والمتوقّعُ مهلةُ العرض (٦٠)", ttl)
	}
	if collapse != "offer:"+orderID {
		t.Errorf("مفتاحُ الطيّ %q", collapse)
	}

	// **ولا يُعرض في الصندوق ولا يُعَدّ في شارته** — قرارُ ٢٠٢٦-٠٨-١٤ باقٍ.
	items, unread, err := notif.List(ctx, f.drivers[0], 200, "")
	if err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	for _, it := range items {
		if it.Kind == notifications.KindOrderOffer {
			t.Fatal("العرضُ ظهر في صندوق السائق — **وهو خبرٌ يموت بعد دقيقة**")
		}
	}
	if unread != 0 {
		t.Errorf("غيرُ المقروء %d — والعرضُ لا يُعَدّ في الشارة", unread)
	}
}

// TestOfferPush_QueueModeOnlyWhoSeesIt **في «للجميع» لا يرنّ إلّا لمن يراه.**
//
// **كان يرنّ عند كلّ سائقٍ على ورديّة** — ومنهم من رفض الطلبَ فلا يراه في
// طابوره أصلاً. **والسؤالُ سؤالُ بابِ الطابور نفسُه.**
func TestOfferPush_QueueModeOnlyWhoSeesIt(t *testing.T) {
	f := newDriverFixture(t, 2)
	ctx := context.Background()
	armRotation(t, f, 60)
	f.setSetting(t, "drivers.assignment_mode", "queue")
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}
	f.srv.orders.SetNotifier(notifications.New(f.pool, f.srv.hub, f.srv.logger))

	orderID := f.dispatchingOrder(t, 30_000, 5_000)
	// **الأوّلُ رفضه من قبل** — فلا يراه في طابوره.
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET status = 'preparing', offer_passed = ARRAY[$2::uuid]
		WHERE id = $1`, orderID, f.drivers[0]); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	if _, err := f.srv.orders.Transition(ctx, "", []string{"ops"}, orderID, "dispatching", ""); err != nil {
		t.Fatalf("النزولُ إلى الطابور: %v", err)
	}

	if n, _, _, _ := f.offerRows(t, f.drivers[1], orderID); n != 1 {
		t.Errorf("من يرى الطلبَ لم يصله العرض (%d)", n)
	}
	if n, _, _, _ := f.offerRows(t, f.drivers[0], orderID); n != 0 {
		t.Error("رنّ العرضُ عند من لا يراه في طابوره — **فيفتح طلباً لا يجده**")
	}
}

// TestOfferSweeper_MovesWithinSeconds **العرضُ ينتقل لحظةَ موته** (قرارُ المالك
// ٢٠٢٦-١٠-٠٢) — لا بعد نبضة الراصد (ثلاثون ثانية).
func TestOfferSweeper_MovesWithinSeconds(t *testing.T) {
	f := newDriverFixture(t, 2)
	armRotation(t, f, 60)
	for _, d := range f.drivers {
		f.onShift(t, d, true)
	}
	orderID := f.dispatchingOrder(t, 30_000, 5_000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := f.srv.orders.OfferNext(ctx, orderID, nil); err != nil {
		t.Fatalf("العرض: %v", err)
	}
	first := f.offeredDriver(t, orderID)
	if first == nil {
		t.Fatalf("لا عرض\n%s", f.driverEligibility(t, orderID))
	}
	// **يموت بعد ثانية** — والحلقةُ السريعةُ تعمل كما تعمل في الخادم.
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET offer_expires_at = now() + interval '1 second' WHERE id = $1`,
		orderID); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	go f.srv.orders.RunOfferSweeper(ctx, 200*time.Millisecond)

	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		if now := f.offeredDriver(t, orderID); now != nil && *now != *first {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("العرضُ لم ينتقل خلال أربع ثوانٍ من موته — **والسائقُ الثاني ينتظر نبضةَ الراصد**")
}
