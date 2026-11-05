package qa

// ══════════════════════════════════════════════════════════════════════
// **مسار السائق ع الطرق اللي مشي فيها** — طلب المالك ٢٠٢٦-١١-٠٥
// («والطريق يحسب حسب طريق السائق مو ع كيفنا»)
// ══════════════════════════════════════════════════════════════════════
//
// المطابق (`internal/roadmatch`) ع محرّك Valhalla مزيَّف (لا محرّك حقيقي): كل نقطة بتنلصق ع خط عرض
// أول نقطة بالطلب. والقراءة من صفحة المراقبة (`drivermonitor.Measure`).

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/polyline"
	"github.com/servacode/rahalgo/backend/internal/roadmatch"
)

// rmFake محرّك مزيَّف: `trace_attributes` بيلصق كل نقطة ع خط عرض أول نقطة، و`/status` بوقت البلاطات.
type rmFake struct {
	srv   *httptest.Server
	calls int32
	mode  atomic.Value // ok | 503 | 400
	tiles int64
}

func newRMFake(t *testing.T) *rmFake {
	f := &rmFake{}
	f.mode.Store("ok")
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/status" {
			_ = json.NewEncoder(w).Encode(map[string]any{"version": "3.9.0", "tileset_last_modified": atomic.LoadInt64(&f.tiles)})
			return
		}
		atomic.AddInt32(&f.calls, 1)
		switch f.mode.Load().(string) {
		case "503":
			w.WriteHeader(503)
			return
		case "400":
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error_code":442,"status_code":400}`))
			return
		}
		var body struct {
			Shape []struct {
				Lat float64 `json:"lat"`
				Lon float64 `json:"lon"`
			} `json:"shape"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		lat := body.Shape[0].Lat
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, p := range body.Shape {
			lo, hi = math.Min(lo, p.Lon), math.Max(hi, p.Lon)
		}
		line := [][2]float64{{lo, lat}, {hi, lat}}
		var mps []map[string]any
		for _, p := range body.Shape {
			mps = append(mps, map[string]any{"lat": lat, "lon": p.Lon, "type": "matched", "edge_index": 0,
				"distance_from_trace_point": polyline.Meters([2]float64{p.Lon, p.Lat}, [2]float64{p.Lon, lat})})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"shape":          polyline.Encode6(line),
			"edges":          []map[string]any{{"length": polyline.Length(line) / 1000, "begin_shape_index": 0, "end_shape_index": 1}},
			"matched_points": mps,
		})
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *rmFake) n() int { return int(atomic.LoadInt32(&f.calls)) }

func rmWorker(h *Harness, f *rmFake, driver string) *roadmatch.Worker {
	c := roadmatch.NewClient(f.srv.URL)
	w := roadmatch.NewWorker(h.Pool, c, nil)
	w.Tiles = c.TilesTime
	w.Opt.Drivers = []string{driver}
	return w
}

// rmDrive سائق ماشي شرقاً ٨ م/ث بنقطة كل ٤ ث من `from` لمدة `dur`، ع خط العرض `lat`. بيتخطّى [skipFrom, skipTo).
func rmDrive(t *testing.T, h *Harness, driver string, from time.Time, dur time.Duration, lat, lng0 float64,
	skipFrom, skipTo time.Duration) {
	t.Helper()
	var at []time.Time
	var lats, lngs []float64
	for s := time.Duration(0); s <= dur; s += 4 * time.Second {
		if s >= skipFrom && s < skipTo {
			continue
		}
		at = append(at, from.Add(s))
		lats = append(lats, lat)
		lngs = append(lngs, lng0+8*s.Seconds()/90700)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
		SELECT $1::uuid, a, la, ln, 5, 8, false FROM unnest($2::timestamptz[], $3::float8[], $4::float8[]) AS v(a, la, ln)
		ON CONFLICT DO NOTHING`, driver, at, lats, lngs); err != nil {
		t.Fatalf("كتابة الأثر: %v", err)
	}
}

func rmTick(t *testing.T, w *roadmatch.Worker, now time.Time, on bool) roadmatch.TickResult {
	t.Helper()
	res, err := w.Tick(ctxBG(), now, on)
	if err != nil {
		t.Fatalf("الدورة: %v", err)
	}
	return res
}

func rmCount(t *testing.T, h *Harness, driver, where string, args ...any) int {
	t.Helper()
	var n int
	q := `SELECT count(*)::int FROM driver_road_chunks WHERE driver_id = $1::uuid`
	if where != "" {
		q += " AND " + where
	}
	if err := h.Pool.QueryRow(ctxBG(), q, append([]any{driver}, args...)...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// Q-R1 — أثر جديد بينطابق **مرّة وحدة**: الحيّ لقدّام، الرجوع لورا لما يفضى الحيّ، والدورة الجاية بلا نداء.
// والمسافة بعدها كلها ع الطريق بلا «لسّا عم تنحسب».
func TestRoadMatcher_MatchesNewTrailOnce(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	_, base := dmDay()
	rmDrive(t, h, drv.ID, base, 20*time.Minute, 35.6100, 38.7000, 0, 0)
	f := newRMFake(t)
	w := rmWorker(h, f, drv.ID)
	now := base.Add(40 * time.Minute)

	r1 := rmTick(t, w, now, true)
	if r1.Calls == 0 || r1.Rows == 0 || r1.Backfill {
		t.Fatalf("الدورة الأولى (الحيّ): %+v", r1)
	}
	var until, bb time.Time
	if err := h.Pool.QueryRow(ctxBG(), `SELECT matched_until, backfill_before FROM driver_road_progress WHERE driver_id = $1::uuid`,
		drv.ID).Scan(&until, &bb); err != nil {
		t.Fatal(err)
	}
	if !until.Equal(base.Add(20*time.Minute)) || !bb.Equal(base.Add(10*time.Minute)) {
		t.Errorf("المؤشّرين: حيّ %v رجوع %v", until.Sub(base), bb.Sub(base))
	}
	if n := rmCount(t, h, drv.ID, "from_at < $2", base.Add(10*time.Minute)); n != 0 {
		t.Errorf("الحيّ طابق قبل مؤشّره: %d", n)
	}
	r2 := rmTick(t, w, now.Add(time.Minute), true)
	if r2.Calls == 0 || !r2.Backfill {
		t.Fatalf("الدورة التانية (الرجوع): %+v", r2)
	}
	before := f.n()
	r3 := rmTick(t, w, now.Add(2*time.Minute), true)
	if r3.Calls != 0 || f.n() != before {
		t.Fatalf("الدورة التالتة لازم بلا نداء: %+v", r3)
	}
	res, err := drivermonitor.Measure(ctxBG(), h.Pool, drv.ID, base, base.Add(21*time.Minute), nil, nil, now, true)
	if err != nil {
		t.Fatal(err)
	}
	s := res.Summary
	if s.PendingKm != 0 || s.RoadShare < 0.999 || math.Abs(s.KmTotal-9.6) > 0.1 {
		t.Errorf("المسافة: كم %.3f حصّة %.3f معلّق %.3f", s.KmTotal, s.RoadShare, s.PendingKm)
	}
}

// Q-R2 — دفعة متأخرة (طابور بعد انقطاع) بنص قطعة مطابقة ⇒ `dirty_from` ⇒ بتنعاد وفيها النقاط الجديدة.
func TestRoadMatcher_LateBatchRematches(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	base := time.Now().UTC().Add(-60 * time.Minute).Truncate(time.Second)
	const lat, lng0 = 35.6200, 38.7000
	rmDrive(t, h, drv.ID, base, 20*time.Minute, lat, lng0, 5*time.Minute, 8*time.Minute)
	f := newRMFake(t)
	w := rmWorker(h, f, drv.ID)
	rmTick(t, w, time.Now(), true) // أول مرّة: الحيّ فاضي، الرجوع بيطابق الكل
	if r := rmTick(t, w, time.Now(), true); r.Calls != 0 {
		t.Fatalf("قبل الدفعة لازم ما في شغل: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, ""); n == 0 {
		t.Fatal("ما انطابق شي")
	}
	// الطابور بيوصل: نقاط الانقطاع.
	var pts []map[string]any
	late := base.Add(6 * time.Minute)
	for s := 5 * time.Minute; s < 8*time.Minute; s += 4 * time.Second {
		pts = append(pts, map[string]any{"lat": lat, "lng": lng0 + 8*s.Seconds()/90700,
			"at": base.Add(s).Format(time.RFC3339), "speed_mps": 8.0, "accuracy_m": 5.0})
	}
	if res := h.POST("/api/v1/driver/location/batch", drv.Token, map[string]any{"points": pts}); res.Code != 200 {
		t.Fatalf("الدفعة: %s", res)
	}
	var dirty *time.Time
	if err := h.Pool.QueryRow(ctxBG(), `SELECT dirty_from FROM driver_road_progress WHERE driver_id = $1::uuid`, drv.ID).
		Scan(&dirty); err != nil || dirty == nil || dirty.After(late) {
		t.Fatalf("الدفعة المتأخرة ما علّمت: %v %v", dirty, err)
	}
	if r := rmTick(t, w, time.Now(), true); r.Calls == 0 {
		t.Fatalf("ما انعادت المطابقة: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, "$2 = ANY(point_at)", late); n != 1 {
		t.Errorf("النقطة المتأخرة بالقطع المخزّنة: %d", n)
	}
	if err := h.Pool.QueryRow(ctxBG(), `SELECT dirty_from FROM driver_road_progress WHERE driver_id = $1::uuid`, drv.ID).
		Scan(&dirty); err != nil || dirty != nil {
		t.Errorf("dirty_from لازم ينمسح: %v", dirty)
	}
}

type rmDay struct {
	Summary struct {
		KmTotal    float64 `json:"km_total"`
		KmNoOrder  float64 `json:"km_no_order"`
		KmStraight float64 `json:"km_straight"`
		RoadShare  float64 `json:"road_share"`
		PendingKm  float64 `json:"pending_km"`
	} `json:"summary"`
	Path []struct {
		Coords   [][2]float64 `json:"coords"`
		Road     bool         `json:"road"`
		Straight bool         `json:"straight"`
		Pending  bool         `json:"pending"`
	} `json:"path"`
}

func rmFetch(t *testing.T, h *Harness, tok, driver, day string) rmDay {
	t.Helper()
	res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", driver, day), tok)
	if res.Code != 200 {
		t.Fatalf("صفحة السائق: %s", res)
	}
	var env struct {
		Data rmDay `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

// Q-R3 — المحرّك واقع (503): ولا سطر، والقاطع بيفتح بعد ٣ أعطال؛ والصفحة بتردّ بخطوط مستقيمة و«لسّا عم تنحسب».
func TestRoadMatcher_DownBacksOffAndPageStillAnswers(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := dmDay()
	rmDrive(t, h, drv.ID, base, 20*time.Minute, 35.6300, 38.7000, 0, 0)
	f := newRMFake(t)
	f.mode.Store("503")
	w := rmWorker(h, f, drv.ID)
	now := base.Add(40 * time.Minute)
	var last roadmatch.TickResult
	for i := 0; i < 3; i++ {
		last = rmTick(t, w, now.Add(time.Duration(i)*time.Minute), true)
	}
	if !last.Breaker {
		t.Errorf("القاطع لازم يفتح بالعطل التالت: %+v", last)
	}
	calls := f.n()
	if r := rmTick(t, w, now.Add(4*time.Minute), true); r.Calls != 0 || f.n() != calls {
		t.Errorf("القاطع مفتوح ⇒ ولا نداء: %+v", r)
	}
	if st := w.Status(now.Add(4 * time.Minute)); st.State != "paused" || st.PausedUntil == nil || st.LastError == "" {
		t.Errorf("الحالة: %+v", st)
	}
	if n := rmCount(t, h, drv.ID, ""); n != 0 {
		t.Errorf("المحرّك واقع وانخزّن %d سطر", n)
	}
	h.Setting("drivers.road_match_enabled", "true")
	got := rmFetch(t, h, admin.Token, drv.ID, day)
	if got.Summary.PendingKm < 9.4 || math.Abs(got.Summary.KmTotal-got.Summary.PendingKm) > 1e-6 {
		t.Errorf("الصفحة والمحرّك واقع: %+v", got.Summary)
	}
}

// Q-R4 — الإعداد مطفي ⇒ ولا نداء حتى لو في أثر جديد؛ والصفحة بتضل تقرأ القطع المخزّنة (ع الطريق، بلا معلّق).
func TestRoadMatcher_SwitchOffStopsCallsKeepsHistory(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := dmDay()
	rmDrive(t, h, drv.ID, base, 20*time.Minute, 35.6400, 38.7000, 0, 0)
	f := newRMFake(t)
	w := rmWorker(h, f, drv.ID)
	now := base.Add(40 * time.Minute)
	rmTick(t, w, now, true)
	rmTick(t, w, now.Add(time.Minute), true)
	// أثر جديد بعد المؤشّر، والمطابقة مطفية.
	rmDrive(t, h, drv.ID, base.Add(45*time.Minute), 10*time.Minute, 35.6400, 38.7200, 0, 0)
	calls := f.n()
	if r := rmTick(t, w, base.Add(70*time.Minute), false); r.Calls != 0 || f.n() != calls {
		t.Errorf("مطفية ⇒ ولا نداء: %+v", r)
	}
	if st := w.Status(base.Add(70 * time.Minute)); st.State != "off" {
		t.Errorf("الحالة: %s", st.State)
	}
	got := rmFetch(t, h, admin.Token, drv.ID, day)
	if got.Summary.PendingKm != 0 {
		t.Errorf("مطفية ⇒ ما في معلّق: %.3f", got.Summary.PendingKm)
	}
	// أول ٢٠ دقيقة ع الطريق (٩٫٦ من ١٤٫٤ كم).
	if got.Summary.RoadShare < 0.6 || got.Summary.RoadShare > 0.7 {
		t.Errorf("حصّة الطريق %.3f (المتوقَّع ~٠٫٦٧)", got.Summary.RoadShare)
	}
}

// Q-R5 — يوم بأثر وقطعة مخزّنة ع شكل L: الصفحة بالكيلومترات ع الطريق، و`km_straight`، والخط من `line6`.
func TestDriverMonitor_DayInRoadMetres(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := dmDay()
	const lat, lng = 35.6500, 38.7000
	east, north := 300.0/90700, 300.0/111200
	a, b := base, base.Add(90*time.Second)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps)
		VALUES ($1::uuid, $2, $3, $4, 5, 8), ($1::uuid, $5, $6, $7, 5, 8)`,
		drv.ID, a, lat, lng, b, lat+north, lng+east); err != nil {
		t.Fatal(err)
	}
	line := [][2]float64{{lng, lat}, {lng + east, lat}, {lng + east, lat + north}}
	road := polyline.Length(line)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_road_chunks (driver_id, from_at, to_at, status, point_at, pos_m, line6, road_m, straight_m, params)
		VALUES ($1::uuid, $2, $3, 'ok', ARRAY[$2, $3]::timestamptz[], ARRAY[0, $4]::real[], $5, $4, 0, $6)`,
		drv.ID, a, b, road, polyline.Encode6(line), roadmatch.ParamsVersion); err != nil {
		t.Fatal(err)
	}
	got := rmFetch(t, h, admin.Token, drv.ID, day)
	if math.Abs(got.Summary.KmNoOrder-0.6) > 0.01 || math.Abs(got.Summary.KmStraight-0.424) > 0.01 {
		t.Errorf("الكيلومترات: طريق %.3f مستقيم %.3f", got.Summary.KmNoOrder, got.Summary.KmStraight)
	}
	if len(got.Path) != 1 || !got.Path[0].Road || len(got.Path[0].Coords) != 3 {
		t.Fatalf("الخط: %+v", got.Path)
	}
	if c := got.Path[0].Coords[1]; math.Abs(c[0]-lat) > 1e-5 || math.Abs(c[1]-(lng+east)) > 1e-5 {
		t.Errorf("الزاوية: %v", c)
	}
}

// Q-R6 — المؤشّرين: أول مرّة الحيّ من قبل ٣٠ دقيقة لقدّام؛ الرجوع الأحدث أوّلاً وبس لما يفضى الحيّ؛
// وما قبل `frozen_before` ما بينمسّ.
func TestRoadMatcher_TwoCursors(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	_, base := dmDay()
	d1, d2 := base.Add(-24*time.Hour), base.Add(-48*time.Hour)
	for _, s := range []time.Time{base, d1, d2} {
		rmDrive(t, h, drv.ID, s, 20*time.Minute, 35.6600, 38.7000, 0, 0)
	}
	frozen := d2.Add(30 * time.Minute)
	if err := roadmatch.Freeze(ctxBG(), h.Pool, drv.ID, frozen); err != nil {
		t.Fatal(err)
	}
	f := newRMFake(t)
	w := rmWorker(h, f, drv.ID)
	w.Opt.BackfillCalls = 1
	now := base.Add(40 * time.Minute)

	if r := rmTick(t, w, now, true); r.Calls == 0 || r.Backfill {
		t.Fatalf("الحيّ أوّلاً: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, "from_at < $2", base.Add(10*time.Minute)); n != 0 {
		t.Errorf("الرجوع اشتغل والحيّ مشغول: %d", n)
	}
	if r := rmTick(t, w, now.Add(time.Minute), true); !r.Backfill || r.Calls == 0 {
		t.Fatalf("الرجوع لما يفضى الحيّ: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, "from_at < $2", base); n != 0 {
		t.Errorf("الرجوع لازم يبلّش بالأحدث (اليوم) قبل المبارح: %d", n)
	}
	if n := rmCount(t, h, drv.ID, "from_at >= $2 AND to_at <= $3", base, base.Add(10*time.Minute)); n == 0 {
		t.Error("أول نافذة رجوع ما انطابقت")
	}
	rmTick(t, w, now.Add(2*time.Minute), true)
	if n := rmCount(t, h, drv.ID, "from_at >= $2 AND to_at <= $3", d1, d1.Add(20*time.Minute)); n == 0 {
		t.Error("المبارح ما انطابق")
	}
	if r := rmTick(t, w, now.Add(3*time.Minute), true); r.Calls != 0 {
		t.Errorf("بعد ما خلص الرجوع لازم ولا نداء: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, "from_at < $2", frozen); n != 0 {
		t.Errorf("انطابق قبل frozen_before: %d", n)
	}
	var bb time.Time
	if err := h.Pool.QueryRow(ctxBG(), `SELECT backfill_before FROM driver_road_progress WHERE driver_id = $1::uuid`, drv.ID).
		Scan(&bb); err != nil || !bb.Equal(d1) {
		t.Errorf("مؤشّر الرجوع %v (المتوقَّع %v) %v", bb, d1, err)
	}
}

// Q-R7 — قطع الطريق بتنمحى مع الأثر (أقدم من مدّة الحفظ)، والجديدة بتضل.
func TestRoadChunks_PrunedWithTrail(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	old := time.Now().AddDate(0, 0, -40)
	fresh := time.Now().Add(-time.Hour)
	for _, at := range []time.Time{old, fresh} {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_road_chunks (driver_id, from_at, to_at, status, point_at, pos_m, params)
			VALUES ($1::uuid, $2, $3, 'ok', ARRAY[$2, $3]::timestamptz[], ARRAY[0, 10]::real[], 1)`,
			drv.ID, at, at.Add(time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := roadmatch.Prune(ctxBG(), h.Pool, 30); err != nil {
		t.Fatal(err)
	}
	if n := rmCount(t, h, drv.ID, ""); n != 1 {
		t.Fatalf("بعد التقليم: %d سطر", n)
	}
	if n := rmCount(t, h, drv.ID, "to_at > $2", time.Now().AddDate(0, 0, -1)); n != 1 {
		t.Error("السطر الجديد انمحى")
	}
}

// Q-R11 — قطعة بقواعد أقدم بتنعاد، و`no_match` أقدم من بناء البلاطات بينعاد **مرّة** وبعدين لأ.
func TestRoadMatcher_ParamsBumpAndNoMatchRetry(t *testing.T) {
	h := New(t)
	drv := h.NewUser("driver")
	_, base := dmDay()
	rmDrive(t, h, drv.ID, base, 10*time.Minute, 35.6800, 38.7000, 0, 0)
	mid, end := base.Add(5*time.Minute), base.Add(10*time.Minute)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_road_progress (driver_id, matched_until, backfill_before) VALUES ($1::uuid, $2, $3)`,
		drv.ID, end, base); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO driver_road_chunks (driver_id, from_at, to_at, status, error_code, point_at, pos_m, params, matched_at)
		VALUES ($1::uuid, $2, $3, 'ok', '', ARRAY[$2, $3]::timestamptz[], ARRAY[0, 0]::real[], $6, now()),
		       ($1::uuid, $3, $4, 'no_match', '442', ARRAY[$3, $4]::timestamptz[], ARRAY['NaN', 'NaN']::real[], $5, $7)`,
		drv.ID, base, mid, end, roadmatch.ParamsVersion, roadmatch.ParamsVersion-1, time.Now().Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	f := newRMFake(t)
	atomic.StoreInt64(&f.tiles, time.Now().Add(-time.Hour).Unix())
	w := rmWorker(h, f, drv.ID)
	now := base.Add(40 * time.Minute)
	if r := rmTick(t, w, now, true); r.Calls < 2 {
		t.Fatalf("الإعادة: %+v", r)
	}
	if n := rmCount(t, h, drv.ID, "params < $2", roadmatch.ParamsVersion); n != 0 {
		t.Errorf("ضل %d سطر بقواعد أقدم", n)
	}
	if n := rmCount(t, h, drv.ID, "status = 'no_match'"); n != 0 {
		t.Errorf("no_match ما انعاد: %d", n)
	}
	// الإعادة بمدى السطر نفسه شاملاً آخر نقطة (حدّ الميكروثانية).
	if n := rmCount(t, h, drv.ID, "to_at = $2", end); n != 1 {
		t.Errorf("آخر نقطة بالمدى ضاعت بالإعادة: %d سطر بينتهي عندها", n)
	}
	// المحرّك صار يرفض ⇒ الإعادة بتعطي no_match جديد أحدث من البلاطات ⇒ ما بينعاد مرّة تانية.
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE driver_road_chunks SET status = 'no_match', matched_at = $2 WHERE driver_id = $1::uuid AND from_at >= $3`,
		drv.ID, time.Now().Add(-48*time.Hour), mid); err != nil {
		t.Fatal(err)
	}
	f.mode.Store("400")
	rmTick(t, w, now.Add(time.Minute), true)
	before := f.n()
	if r := rmTick(t, w, now.Add(2*time.Minute), true); r.Calls != 0 || f.n() != before {
		t.Errorf("no_match بعد البلاطات انعاد مرّة تانية: %+v", r)
	}
}
