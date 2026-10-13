package qa

import (
	"testing"
	"time"
)

// **سائقٌ في الدوام انقطع موقعُه يُنبَّه — ولا يُغلَق دوامُه** (قرارُ المالك
// ٢٠٢٦-١٠-١٣). وصاحبُ الموقع الحديث لا يُزعَج.
func TestDriverLocationWatch_NotifiesNeverEndsShift(t *testing.T) {
	h := New(t)
	stale := h.NewUser("driver")
	fresh := h.NewUser("driver")
	for _, d := range []struct {
		id  string
		ago string
	}{{stale.ID, "12 minutes"}, {fresh.ID, "30 seconds"}} {
		if _, err := h.Pool.Exec(ctxBG(), `
			UPDATE users SET on_shift = true, shift_started_at = now() - interval '2 hours',
			       last_location = ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography,
			       last_location_at = now() - $2::interval
			 WHERE id = $1::uuid`, d.id, d.ago); err != nil {
			t.Fatal(err)
		}
	}
	count := func(uid string) int {
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
			WHERE user_id = $1::uuid AND title LIKE 'موقعك ما عم يوصلنا%'`, uid).Scan(&n)
		return n
	}
	now := time.Now()
	h.API.WatchDriverLocations(ctxBG(), now)
	if n := count(stale.ID); n != 1 {
		t.Fatalf("سائقٌ صامتٌ ١٢ دقيقة ⇒ إشعارات %d — والمنتظَر واحد", n)
	}
	if n := count(fresh.ID); n != 0 {
		t.Fatalf("**أُزعج صاحبُ الموقع الحديث** بـ%d إشعار", n)
	}
	// **ولا يتكرّر قبل ربع ساعة.**
	h.API.WatchDriverLocations(ctxBG(), now.Add(time.Minute))
	if n := count(stale.ID); n != 1 {
		t.Fatalf("تكرّر الإشعارُ بعد دقيقة: %d", n)
	}
	h.API.WatchDriverLocations(ctxBG(), now.Add(16*time.Minute))
	if n := count(stale.ID); n != 2 {
		t.Fatalf("بعد ربع ساعة ⇒ %d — والمنتظَر تذكيرٌ ثانٍ", n)
	}
	// **والدوامُ باقٍ** — لا يُغلقه إلّا صاحبُه.
	var on bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT on_shift FROM users WHERE id = $1::uuid`, stale.ID).Scan(&on)
	if !on {
		t.Fatalf("**أُغلق دوامُ السائق آليّاً** — والقرار: لا يُغلقه إلّا هو")
	}
}
