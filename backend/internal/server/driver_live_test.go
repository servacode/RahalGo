package server

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// TestDriverLocation_BroadcastsToOps **نبضةُ السائق تُبثّ لخريطة العمليات بحمولتها**
// (قرارُ المالك ٢٠٢٦-١٠-١٩) — ونبضةٌ مُهمَلةٌ (قفزةٌ مستحيلة) لا تُبثّ.
//
// **وعلى `ops:drivers` لا `ops`** (٢٠٢٦-١١-٠٥، عن قصد): `ops` تصل كلَّ من يقرأ الطلبات،
// **ومنهم خدمةُ الزبائن والأمانُ ولا يرون السائقين.**
func TestDriverLocation_BroadcastsToOps(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ch, cancel := f.srv.hub.Subscribe([]string{"ops:drivers"})
	defer cancel()
	orders, cancelOrders := f.srv.hub.Subscribe([]string{"ops"})
	defer cancelOrders()
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
		t.Fatal("النبضةُ لم تُبثّ على ops:drivers — **فتبقى الخريطةُ تسأل وتقفز**")
	}
	if ev["driver_id"] != d || ev["lat"].(float64) < 35.9 || ev["bearing"].(float64) != 90 {
		t.Fatalf("حمولةٌ غيرُ صحيحة: %v", ev)
	}
	// **ولا تصل غرفةَ الطلبات** — من يقرأ الطلبات ولا يرى السائقين لا يصله موضعُهم.
	// (والبثُّ متزامن: ما وصل `ops:drivers` كان سيصل `ops` معه في النداء نفسِه.)
drain:
	for {
		select {
		case raw := <-orders:
			var leaked map[string]any
			_ = json.Unmarshal(raw, &leaked)
			if leaked["type"] == "driver_location" {
				t.Fatalf("موضعُ السائق بُثّ على ops: %v — **فيصل خدمةَ الزبائن والأمانَ بلا صلاحيّة**", leaked)
			}
		default:
			break drain
		}
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

// TestDriverLocation_BroadcastsShownPoint **النبضةُ تبثّ نقطةَ العرض لا الخامّ** (L-T4، ٢٠٢٦-١١-٠٥:
// «ما بدنا نطّات»). نقطةٌ دقّتُها ٣٠٠ م تُكتب موضعاً (التوزيعُ يقرؤها) **وتُبثّ بالموضع الممسوك
// ووقتٍ جديد** فيبقى «حيّاً» ولا يقفز؛ ومعها سرعتُه ودقّتُه. **ومفتاحُ اللوحة مطفأً يرجّع الخامّ.**
func TestDriverLocation_BroadcastsShownPoint(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ch, cancel := f.srv.hub.Subscribe([]string{"ops:drivers"})
	defer cancel()
	t.Cleanup(func() { f.setSetting(t, "drivers.ops_map_smooth", true) })
	ping := func(lat string, acc, speed string, at time.Time) {
		t.Helper()
		w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", d, []string{"driver"},
			`{"lat":`+lat+`,"lng":39.0094,"accuracy_m":`+acc+`,"speed_mps":`+speed+`,"bearing_deg":90,"recorded_at":"`+
				at.UTC().Format(time.RFC3339)+`"}`)
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
				if ev["type"] == "driver_location" && ev["driver_id"] == d {
					return ev
				}
			case <-deadline:
				t.Fatal("لا بثّ")
				return nil
			}
		}
	}
	atOf := func(ev map[string]any) time.Time {
		s, _ := ev["at"].(string)
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t.Fatalf("وقتٌ غيرُ مقروء: %v", ev["at"])
		}
		return v
	}

	now := time.Now().Truncate(time.Second)
	ping("35.9506", "6", "3.5", now.Add(-10*time.Second))
	first := next()
	if first["lat"].(float64) != 35.9506 {
		t.Fatalf("النقطةُ الدقيقة لم تُبثّ كما هي: %v", first)
	}

	// **نقطةٌ دقّتُها ٣٠٠ م وعلى بُعد ٣٠٠ م** ⇒ تُبثّ بالموضع الممسوك ووقتِها هي.
	ping("35.9533", "300", "0", now.Add(-6*time.Second))
	ev := next()
	if ev["lat"].(float64) != 35.9506 {
		t.Fatalf("**نقطةٌ دقّتُها ٣٠٠ م حرّكت السائق على الخريطة**: %v", ev)
	}
	if !atOf(ev).Equal(now.Add(-6 * time.Second)) {
		t.Fatalf("الوقتُ لم يتحدّث — فيُقرأ السائقُ غيرَ حيّ: %v", ev["at"])
	}
	// **ومعها سرعتُه ودقّتُه** — إضافةٌ على الحمولة لا تبديل.
	if first["accuracy_m"] != 6.0 || first["speed_mps"] != 3.5 || first["rough"] != false {
		t.Fatalf("الحمولةُ بلا سرعةٍ أو دقّة: %v", first)
	}
	// **والموضعُ الخامُّ كُتب** — التوزيعُ ومراقبةُ الموضع يقرآنه.
	var lat float64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT ST_Y(last_location::geometry) FROM users WHERE id = $1`, d).Scan(&lat); err != nil {
		t.Fatal(err)
	}
	if lat < 35.953 {
		t.Fatalf("الموضعُ الخامُّ لم يُكتب (%v) — نقطةُ العرض للعرض وحدَه", lat)
	}

	// **والمفتاحُ مطفأً ⇒ الخامّ**، كما كان.
	f.setSetting(t, "drivers.ops_map_smooth", false)
	ping("35.9540", "300", "0", now.Add(-2*time.Second))
	ev = next()
	if ev["lat"].(float64) != 35.9540 || ev["accuracy_m"] != 300.0 {
		t.Fatalf("المفتاحُ مطفأٌ والبثُّ ليس الخامّ: %v", ev)
	}
}

// TestOpsMapDrivers_OverlaysShownPoint **طبقةُ السائقين تردّ نقطةَ العرض واتّجاهَها** (L-T5) —
// فلا يرجع جوابُ المدّة السائقَ إلى موقعٍ مسكته النبضة. **ومطفأً: الخامّ بلا اتّجاه.**
func TestOpsMapDrivers_OverlaysShownPoint(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.onShift(t, d, true)
	t.Cleanup(func() { f.setSetting(t, "drivers.ops_map_smooth", true) })
	ping := func(lat, acc string, at time.Time) {
		t.Helper()
		w := f.call(f.srv.handleDriverLocation, http.MethodPost, "/driver/location", "", d, []string{"driver"},
			`{"lat":`+lat+`,"lng":39.0094,"accuracy_m":`+acc+`,"speed_mps":4,"bearing_deg":90,"recorded_at":"`+
				at.UTC().Format(time.RFC3339)+`"}`)
		if w.Code != http.StatusOK {
			t.Fatalf("النبضةُ ردّت %d: %s", w.Code, w.Body.String())
		}
	}
	type row struct {
		ID      string   `json:"id"`
		Lat     float64  `json:"lat"`
		Bearing *float64 `json:"bearing"`
		Rough   bool     `json:"rough"`
	}
	poll := func() row {
		t.Helper()
		w := f.call(f.srv.handleOpsMapDrivers, http.MethodGet,
			"/ops-map/drivers?bbox=39.0,35.94,39.02,35.96&on_shift=true", "", d, []string{"operations"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("الطبقةُ ردّت %d: %s", w.Code, w.Body.String())
		}
		var out struct {
			Data struct {
				Drivers []row `json:"drivers"`
			} `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		for _, r := range out.Data.Drivers {
			if r.ID == d {
				return r
			}
		}
		t.Fatal("السائقُ ليس في الطبقة")
		return row{}
	}

	now := time.Now().Truncate(time.Second)
	ping("35.9506", "6", now.Add(-10*time.Second))
	ping("35.9533", "300", now.Add(-5*time.Second))
	r := poll()
	if r.Lat != 35.9506 {
		t.Fatalf("**الطبقةُ ردّت الموضعَ الخامّ** (%v) — فيرجع السائقُ إليه مع كلّ مدّة", r.Lat)
	}
	if r.Bearing == nil || *r.Bearing != 90 || r.Rough {
		t.Fatalf("اتّجاهُ نقطة العرض غاب: %+v", r)
	}

	f.setSetting(t, "drivers.ops_map_smooth", false)
	if r := poll(); r.Lat != 35.9533 || r.Bearing != nil {
		t.Fatalf("المفتاحُ مطفأٌ والطبقةُ ليست الخامّ: %+v", r)
	}
}
