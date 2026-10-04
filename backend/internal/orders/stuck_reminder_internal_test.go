package orders

// **تذكيرُ الطلب العالق يتكرّر حتّى يتصرّف أحد**
// (قرارُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «مراقبة التشغيل»، البند ٤.)
//
//	بعد N دقيقة بلا فعل     ⇒ تذكيرٌ ثانٍ بعنوان «تذكير: …»
//	قبل N دقيقة             ⇒ لا شيء
//	تغيّرت الحالةُ أو السائق  ⇒ يتوقّف
//	«أنا عليه»              ⇒ يتوقّف
//	سببٌ جديد               ⇒ إنذارٌ فوراً ويمسح «أنا عليه»

import (
	"context"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestStuckReminder_RepeatsEveryNMinutesUntilSomeoneActs(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	ops := testdb.NewUser(t, pool, "operations")
	ns := notifications.New(pool, noopPublisher{}, quietLogger())
	svc := &Service{db: pool, logger: quietLogger(), notify: ns, pub: noopPublisher{}}

	// **والمدّةُ الافتراضيّةُ ١٥ دقيقة** — بلا مخزن إعداداتٍ يُقرأ الفهرس.
	if got := svc.settingInt(ctx, "ops.stuck_reminder_min"); got != 15 {
		t.Fatalf("مدّةُ التذكير الافتراضيّة %d لا ١٥", got)
	}

	titles := func(oid string) []string {
		t.Helper()
		rows, err := pool.Query(ctx, `SELECT title FROM notifications
			WHERE user_id = $1 AND entity = 'order' AND entity_id = $2 ORDER BY created_at, id`, ops, oid)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			_ = rows.Scan(&s)
			out = append(out, s)
		}
		return out
	}
	// age **يُرجع الإنذارَ الأخيرَ إلى الوراء** — بدل انتظار ربع ساعة.
	age := func(oid string, mins int) {
		t.Helper()
		if _, err := pool.Exec(ctx, `UPDATE orders SET alerted_at = now() - make_interval(mins => $2)
			WHERE id = $1`, oid, mins); err != nil {
			t.Fatal(err)
		}
	}

	// ── ١ · يتكرّر بعد المدّة، ولا قبلها ──────────────────────────────
	oid := seedAlertableOrder(t, pool)
	alert := Alert{OrderID: oid, Number: 1, Status: "pending", MerchantName: "متجر", Reason: StuckNoAccept}
	svc.escalate(ctx, []Alert{alert})
	if got := titles(oid); len(got) != 1 {
		t.Fatalf("الإنذارُ الأوّل: %v", got)
	}
	age(oid, 10)
	svc.escalate(ctx, []Alert{alert})
	if got := titles(oid); len(got) != 1 {
		t.Fatalf("**ذُكّر قبل أن تمضي المدّة** (١٠ من ١٥): %v", got)
	}
	age(oid, 16)
	svc.escalate(ctx, []Alert{alert})
	got := titles(oid)
	if len(got) != 2 {
		t.Fatalf("**مضت ١٦ دقيقةً بلا فعلٍ ولم يُذكَّر المكتب** — الإشعارات: %v", got)
	}
	if !strings.HasPrefix(got[1], reminderPrefix) {
		t.Errorf("التذكيرُ لا يقول إنّه تذكير: %q", got[1])
	}
	age(oid, 16)
	svc.escalate(ctx, []Alert{alert})
	if got := titles(oid); len(got) != 3 {
		t.Fatalf("**التذكيرُ لا يتكرّر مرّةً ثالثة** — %v", got)
	}

	// ── ٢ · «أنا عليه» يوقفه ────────────────────────────────────────
	if ok, err := svc.AckAlert(ctx, oid, ops); err != nil || !ok {
		t.Fatalf("«أنا عليه»: ok=%v err=%v", ok, err)
	}
	age(oid, 60)
	svc.escalate(ctx, []Alert{alert})
	if got := titles(oid); len(got) != 3 {
		t.Fatalf("**ذُكّر بعد «أنا عليه»**: %v", got)
	}
	// **وسببٌ جديدٌ ينذر فوراً ويمسح «أنا عليه».**
	alert.Reason = StuckNotSent
	svc.escalate(ctx, []Alert{alert})
	if got := titles(oid); len(got) != 4 {
		t.Fatalf("**سببٌ جديدٌ بعد «أنا عليه» لم يُنذَر**: %v", got)
	}
	var acked bool
	_ = pool.QueryRow(ctx, `SELECT alert_ack_at IS NOT NULL FROM orders WHERE id = $1`, oid).Scan(&acked)
	if acked {
		t.Fatal("«أنا عليه» بقيت بعد تبدّل السبب — فلا يُذكَّر بالسبب الجديد")
	}

	// ── ٣ · تغيّرُ الحالة يوقفه ─────────────────────────────────────
	oid2 := seedAlertableOrder(t, pool)
	a2 := Alert{OrderID: oid2, Number: 2, Status: "pending", MerchantName: "متجر", Reason: StuckTooLong}
	svc.escalate(ctx, []Alert{a2})
	if _, err := pool.Exec(ctx, `UPDATE orders SET status = 'accepted' WHERE id = $1`, oid2); err != nil {
		t.Fatal(err)
	}
	age(oid2, 30)
	svc.escalate(ctx, []Alert{a2})
	if got := titles(oid2); len(got) != 1 {
		t.Fatalf("**تحرّكت حالةُ الطلب ومع ذلك ذُكّر**: %v", got)
	}

	// ── ٤ · تبدّلُ السائق يوقفه ─────────────────────────────────────
	oid3 := seedAlertableOrder(t, pool)
	drv := testdb.NewUser(t, pool, "driver")
	a3 := Alert{OrderID: oid3, Number: 3, Status: "pending", MerchantName: "متجر", Reason: StuckTooLong}
	svc.escalate(ctx, []Alert{a3})
	if _, err := pool.Exec(ctx, `UPDATE orders SET driver_id = $2 WHERE id = $1`, oid3, drv); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE orders SET driver_id = NULL WHERE id = $1`, oid3)
	})
	age(oid3, 30)
	svc.escalate(ctx, []Alert{a3})
	if got := titles(oid3); len(got) != 1 {
		t.Fatalf("**أُسند سائقٌ ومع ذلك ذُكّر**: %v", got)
	}

	// ── ٥ · والقائمةُ تحمل «أنا عليه» وعددَ التذكيرات ─────────────────
	if _, err := pool.Exec(ctx, `UPDATE orders SET created_at = now() - interval '3 days' WHERE id = $1`, oid2); err != nil {
		t.Fatal(err)
	}
	al, err := svc.Alerts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range al {
		if a.OrderID == oid2 {
			found = true
			if a.AckedAt != nil {
				t.Errorf("طلبٌ لم يُستلَم يظهر مستلَماً")
			}
		}
	}
	if !found {
		t.Fatalf("الطلبُ العالقُ ليس في القائمة")
	}
}
