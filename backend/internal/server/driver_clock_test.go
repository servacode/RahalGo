package server

// **تصحيحُ ساعة الهاتف** (A6.4، `driver_clock.go`) — هاتفٌ ساعتُه متأخّرةٌ ثلاثَ ساعاتٍ كان
// موضعُه لا يُكتب أبداً (أقدمُ من ساعتين)، **والنسخُ القديمة لا يتغيّر لها شيء.**

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
)

func TestNextClockFix_LatencyIsNotSkewAndFixHolds(t *testing.T) {
	s := func(d time.Duration) clockState { return clockState{fix: d, known: true} }
	cases := []struct {
		name     string
		prev     clockState
		observed time.Duration
		want     clockState
	}{
		{"ساعةٌ صحيحةٌ وزمنُ وصولٍ ٢٠ ث — لا تصحيح", clockState{}, 20 * time.Second, s(0)},
		{"هاتفٌ متأخّرٌ ثلاثَ ساعات — يُصحَّح", clockState{}, 3 * time.Hour, s(3 * time.Hour)},
		{"هاتفٌ متقدّمٌ خمسَ دقائق — يُصحَّح للوراء", clockState{}, -5 * time.Minute, s(-5 * time.Minute)},
		{"دون الدقيقة — لا تصحيح", clockState{}, 59 * time.Second, s(0)},
		{"التصحيحُ يثبت مع زمن وصولٍ مختلف", s(3 * time.Hour), 3*time.Hour + 12*time.Second, s(3 * time.Hour)},
		{"قربَ الحدّ لا يتأرجح", s(61 * time.Second), 58 * time.Second, s(61 * time.Second)},
		{"وبعد الصفر يبقى صفراً على تأخّرٍ عابر", s(0), 25 * time.Second, s(0)},
		{"ساعةٌ ضُبطت — يعود صفراً", s(3 * time.Hour), 2 * time.Second, s(0)},
		{"ساعةٌ تبدّلت — يأخذ الجديد", s(3 * time.Hour), time.Hour, s(time.Hour)},
	}
	for _, c := range cases {
		if got := nextClockFix(c.prev, c.observed); got != c.want {
			t.Errorf("%s: %+v والمتوقّع %+v", c.name, got, c.want)
		}
	}
}

// callDevice **نداءُ سائقٍ بساعة جهازه** — فارغةٌ = نسخةٌ قديمةٌ لا ترسل الترويسة.
func (f *driverFixture) callDevice(h http.HandlerFunc, path, userID, body, deviceTime string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if deviceTime != "" {
		req.Header.Set(deviceTimeHeader, deviceTime)
	}
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, chi.NewRouteContext())
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

func TestDriverLocation_DeviceClockCorrected(t *testing.T) {
	f := newDriverFixture(t, 3)
	ctx := context.Background()
	stamp := func(at time.Time) string { return at.UTC().Format(time.RFC3339) }
	now := time.Now().Truncate(time.Second)
	behind := 3 * time.Hour
	// **الهاتفُ متأخّرٌ ثلاثَ ساعات**: التقط النقطةَ قبل ثلاث ثوانٍ، وساعتُه تقول «قبل ثلاث ساعات».
	single := `{"lat":35.95,"lng":39.01,"recorded_at":"` + stamp(now.Add(-behind-3*time.Second)) + `"}`

	// **١ · النسخةُ القديمة بلا ترويسة — كما كان**: أقدمُ من ساعتين فلا يُكتب.
	old := f.drivers[0]
	w := f.callDevice(f.srv.handleDriverLocation, "/driver/location", old, single, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"saved":false`) {
		t.Fatalf("النسخةُ القديمة: %d %s — والمتوقّعُ تجاهلٌ كما كان", w.Code, w.Body.String())
	}

	// **٢ · النسخةُ الجديدة ترسل ساعتَها — فيُكتب بوقت الخادم.**
	fresh := f.drivers[1]
	w = f.callDevice(f.srv.handleDriverLocation, "/driver/location", fresh, single, stamp(now.Add(-behind)))
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), `"saved":false`) {
		t.Fatalf("بساعة الجهاز: %d %s — **هاتفٌ ساعتُه متأخّرةٌ لا يُكتب موضعُه أبداً**", w.Code, w.Body.String())
	}
	if age := f.locationAge(t, fresh); age < 0 || age > 30*time.Second {
		t.Fatalf("عمرُ الموضع بعد التصحيح %v — والمتوقّعُ ثوانٍ", age)
	}

	// **٣ · والدفعةُ كذلك** — نقاطُ عشرِ دقائقَ بساعته المتأخّرة تُقبل كلُّها بأوقاتٍ مصحَّحة.
	type pt struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
		At  string  `json:"at"`
	}
	pts := make([]pt, 10)
	for i := range pts {
		pts[i] = pt{Lat: 35.90 + float64(i)*0.0003, Lng: 39.01,
			At: stamp(now.Add(-behind - time.Duration(12-i)*time.Minute))}
	}
	body, _ := json.Marshal(map[string]any{"points": pts})
	w = f.callDevice(f.srv.handleDriverLocationBatch, "/driver/location/batch", fresh, string(body), stamp(now.Add(-behind)))
	var out struct {
		Data map[string]int `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != http.StatusOK || out.Data["accepted"] != 10 {
		t.Fatalf("الدفعةُ بساعة الجهاز: %d %s — والمتوقّعُ قبولُ العشر", w.Code, w.Body.String())
	}
	var oldest, newest time.Time
	if err := f.pool.QueryRow(ctx, `
		SELECT min(recorded_at), max(recorded_at) FROM driver_track
		 WHERE driver_id = $1 AND recorded_at < now() - interval '2 minutes'`, fresh).Scan(&oldest, &newest); err != nil {
		t.Fatalf("قراءةُ الأثر: %v", err)
	}
	if got, want := oldest, now.Add(-12*time.Minute); got.Sub(want).Abs() > 30*time.Second {
		t.Fatalf("أقدمُ نقطةٍ في الدفعة كُتبت %v والمتوقّعُ نحو %v", got, want)
	}

	// **٤ · وساعةٌ صحيحة لا تُمسّ** — الوقتُ يُكتب كما أُرسل بالضبط.
	right := f.drivers[2]
	sent := now.Add(-2 * time.Second)
	w = f.callDevice(f.srv.handleDriverLocation, "/driver/location", right,
		`{"lat":35.95,"lng":39.01,"recorded_at":"`+stamp(sent)+`"}`, stamp(now.Add(-1*time.Second)))
	if w.Code != http.StatusOK {
		t.Fatalf("ساعةٌ صحيحة: %d %s", w.Code, w.Body.String())
	}
	var got time.Time
	if err := f.pool.QueryRow(ctx, `SELECT max(recorded_at) FROM driver_track WHERE driver_id = $1`, right).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(sent) {
		t.Fatalf("ساعةٌ صحيحة: كُتب %v والمُرسَل %v — **زمنُ الوصول ليس خطأَ ساعة**", got, sent)
	}
}

// **نقاطُ الطابور قبل آخر تبدّلٍ بالتصحيح القديم** (مراجعةُ ٢٠٢٦-١١-٠٦) — هاتفٌ متأخّرٌ ثلاثَ ساعاتٍ
// انضبطت ساعتُه مع عودة الشبكة: **نقاطُ ساعةِ الانقطاع انلقطت بالساعة القديمة**، وتصحيحُ لحظة
// الإرسال (صفر) كان يتركها متأخّرةً ثلاثَ ساعاتٍ فتُرمى.
func TestClockView_QueueBeforeChangeKeepsOldFix(t *testing.T) {
	now := time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)
	behind := 3 * time.Hour
	fixed := clockView{fix: 0, changed: true, prev: behind, changedAt: now}
	before := now.Add(-40 * time.Minute)
	if got := fixed.correct(before.Add(-behind)); !got.Equal(before) {
		t.Errorf("نقطةٌ قبل انضباط الساعة: %v والمتوقّعُ %v", got, before)
	}
	after := now.Add(5 * time.Second)
	if got := fixed.correct(after); !got.Equal(after) {
		t.Errorf("نقطةٌ بعد انضباط الساعة: %v والمتوقّعُ %v", got, after)
	}
	// **وبلا تبدّلٍ كما كان** — والتصحيحُ نفسُه لكلّ النقاط.
	steady := clockView{fix: behind}
	if got := steady.correct(before.Add(-behind)); !got.Equal(before) {
		t.Errorf("بلا تبدّل: %v", got)
	}
	// **وساعةٌ خربت (صفر ← ثلاث ساعات)**: الاثنان منطقيّان ⇒ الجديد كما كان.
	broke := clockView{fix: behind, changed: true, prev: 0, changedAt: now}
	if got := broke.correct(after.Add(-behind)); !got.Equal(after) {
		t.Errorf("ساعةٌ خربت: %v والمتوقّعُ %v", got, after)
	}
}

// **التصحيحُ يثبت بعد إعادة تشغيل الخادم** (مراجعةُ ٢٠٢٦-١١-٠٦) — كان يُحسب من جديدٍ بزمن وصولٍ
// آخر، **فدفعةٌ أُعيد إرسالها تُكتب بأوقاتٍ أخرى** ولا يمنعها `ON CONFLICT (driver_id, recorded_at)`.
func TestDeviceClock_HeldAcrossRestart(t *testing.T) {
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	s := &Server{rdb: rdb}
	driver := "clock-restart-" + time.Now().Format("150405.000000000")
	t.Cleanup(func() { deviceClocks.Delete(driver) })
	now := time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC)
	req := func(dev time.Time) *http.Request {
		r := httptest.NewRequest(http.MethodPost, "/driver/location/batch", nil)
		r.Header.Set(deviceTimeHeader, dev.UTC().Format(time.RFC3339Nano))
		return r
	}
	first := s.deviceClockFix(req(now.Add(-3*time.Hour)), driver, now)
	if first != 3*time.Hour {
		t.Fatalf("أوّلُ تصحيح: %v", first)
	}
	// **إعادةُ تشغيل**: الذاكرةُ فارغة، والنداءُ التالي بزمن وصولٍ أطولَ باثنتي عشرة ثانية.
	deviceClocks.Delete(driver)
	if again := s.deviceClockFix(req(now.Add(-3*time.Hour-12*time.Second)), driver, now); again != first {
		t.Errorf("بعد إعادة التشغيل صار التصحيحُ %v وكان %v — **دفعةٌ مُعادةٌ تتكرّر نقاطُها**", again, first)
	}
}

// **ساعةٌ انضبطت مع عودة الشبكة** (مراجعةُ ٢٠٢٦-١١-٠٦، قاعدة) — الهاتفُ متأخّرٌ ثلاثَ ساعاتٍ
// ومعروفٌ تصحيحُه، ثمّ انقطع أربعين دقيقة، وانضبطت ساعتُه لحظةَ عودة الشبكة فأفرغ طابورَه:
// **نقاطُ الانقطاع تُقبل بأوقاتها الحقيقيّة** (كانت تُصحَّح بصفرٍ فتبقى أقدمَ من ساعتين فتُرمى).
func TestLocationBatch_ClockFixedOnReconnect(t *testing.T) {
	f := newDriverFixture(t, 1)
	ctx := context.Background()
	stamp := func(at time.Time) string { return at.UTC().Format(time.RFC3339) }
	now := time.Now().Truncate(time.Second)
	behind := 3 * time.Hour
	drv := f.drivers[0]
	single := `{"lat":35.95,"lng":39.01,"recorded_at":"` + stamp(now.Add(-behind-45*time.Minute)) + `"}`
	if w := f.callDevice(f.srv.handleDriverLocation, "/driver/location", drv, single, stamp(now.Add(-behind))); w.Code != http.StatusOK {
		t.Fatalf("قبل الانقطاع: %d %s", w.Code, w.Body.String())
	}
	type pt struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
		At  string  `json:"at"`
	}
	pts := make([]pt, 10)
	for i := range pts {
		pts[i] = pt{Lat: 35.95 + float64(i)*0.0003, Lng: 39.01,
			At: stamp(now.Add(-behind - time.Duration(40-i)*time.Minute))}
	}
	body, _ := json.Marshal(map[string]any{"points": pts})
	w := f.callDevice(f.srv.handleDriverLocationBatch, "/driver/location/batch", drv, string(body), stamp(now))
	var out struct {
		Data map[string]int `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if w.Code != http.StatusOK || out.Data["accepted"] != 10 {
		t.Fatalf("الطابور بعد انضباط الساعة: %d %s — والمتوقّعُ قبولُ العشر بأوقاتها", w.Code, w.Body.String())
	}
	var oldest time.Time
	if err := f.pool.QueryRow(ctx, `SELECT min(recorded_at) FROM driver_track WHERE driver_id = $1 AND recorded_at > $2`,
		drv, now.Add(-42*time.Minute)).Scan(&oldest); err != nil {
		t.Fatal(err)
	}
	if want := now.Add(-40 * time.Minute); oldest.Sub(want).Abs() > 30*time.Second {
		t.Errorf("أقدمُ نقطةٍ في الطابور %v والمتوقّعُ نحو %v", oldest, want)
	}
}
