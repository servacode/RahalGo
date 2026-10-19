package server

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestDriverLocation_BroadcastsToOps **نبضةُ السائق تُبثّ لخريطة العمليات بحمولتها**
// (قرارُ المالك ٢٠٢٦-١٠-١٩) — ونبضةٌ مُهمَلةٌ (قفزةٌ مستحيلة) لا تُبثّ.
func TestDriverLocation_BroadcastsToOps(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ch, cancel := f.srv.hub.Subscribe([]string{"ops"})
	defer cancel()
	ping := func(lat, lng string, at time.Time) {
		t.Helper()
		w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", d, []string{"driver"},
			`{"lat":`+lat+`,"lng":`+lng+`,"bearing_deg":90,"recorded_at":"`+at.UTC().Format(time.RFC3339)+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("النبضةُ ردّت %d: %s", w.Code, w.Body.String())
		}
	}
	next := func() map[string]any {
		t.Helper()
		deadline := time.After(2 * time.Second)
		for {
			select {
			case raw := <-ch:
				var ev map[string]any
				_ = json.Unmarshal(raw, &ev)
				if ev["type"] == "driver_location" {
					return ev
				}
			case <-deadline:
				return nil
			}
		}
	}

	now := time.Now()
	ping("35.9506", "39.0094", now.Add(-2*time.Second))
	ev := next()
	if ev == nil {
		t.Fatal("النبضةُ لم تُبثّ على ops — **فتبقى الخريطةُ تسأل وتقفز**")
	}
	if ev["driver_id"] != d || ev["lat"].(float64) < 35.9 || ev["bearing"].(float64) != 90 {
		t.Fatalf("حمولةٌ غيرُ صحيحة: %v", ev)
	}
	// **وقفزةٌ مستحيلةٌ لا تُبثّ** — دمشق بعد ثانية.
	ping("33.5138", "36.2765", now.Add(-1*time.Second))
	if ev := next(); ev != nil {
		t.Fatalf("قفزةٌ مُهمَلةٌ بُثّت: %v", ev)
	}
}

// TestTrackPruneDue **التقليمُ مرّةً في الدقيقة لكلّ سائق** لا مع كلّ نبضة.
func TestTrackPruneDue(t *testing.T) {
	s := &Server{}
	now := time.Now()
	if !s.trackPruneDue("prune-a", now) {
		t.Fatal("أوّلُ نبضةٍ لا تُقلِّم")
	}
	if s.trackPruneDue("prune-a", now.Add(10*time.Second)) {
		t.Fatal("قُلِّم بعد عشر ثوانٍ")
	}
	if !s.trackPruneDue("prune-b", now.Add(10*time.Second)) {
		t.Fatal("سائقٌ آخرُ انتظر غيرَه")
	}
	if !s.trackPruneDue("prune-a", now.Add(61*time.Second)) {
		t.Fatal("لم يُقلَّم بعد دقيقة")
	}
}
