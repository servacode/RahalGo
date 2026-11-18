package roadmatch

// ══════════════════════════════════════════════════════════════════════
// **مختبر الحقيقة للكيلومترات** (طلب المالك ٢٠٢٦-١١-١٨: «الكيلومترات اللي مشيتها الدراجة لازم تكون رقم حقيقي»)
// ══════════════════════════════════════════════════════════════════════
//
// مشاوير دراجة مصنوعة **ع شوارع الرقة الحقيقية** (طرق Valhalla مسجّلة) بطول معروف، بتمرق من نفس
// السلسلة اللي بالمنصّة:
//
//  1. الجوال: نقطة كل ٣ ث مع طلب (وحدّ ٥ م)، وكل ٥ ث بلا طلب، ونبضة الواقف اللي بتعيد آخر موضع
//     (١٢٠ ث مع طلب، ١٠ ث بلا طلب، لحد عمر ٣٠٠ ث) — متل `LocationService.kt`.
//  2. ضجيج GPS مترابط بالزمن (σ ٥–١٥ م برّا، ٢٥ م جوّا)، ونتوءات ٣٠–٥٠ م، وسرعة جوال فيها ضجيج،
//     وزحمة ٥–١٥ كم/س، وإشارات حمرا، وانقطاع إشارة، ووقوف جوّا محل.
//  3. الخادم: قفزة أسرع من مركبة بتنرمى (`impossibleJump`)، والتخفيف (`drivermonitor.Filter`).
//  4. المقياس بلا مطابقة (`Analyze`)، والمطابقة: `Chunks` → Valhalla (أجوبة مسجّلة) → `Rows` → `Analyze`.
//
// **بلا شبكة بالبناء**: الطرق والأجوبة بـ`testdata/truth`. ولإعادة تسجيلها من Valhalla محلّي (مو الإنتاج):
//
//	TRUTH_VALHALLA_URL=http://127.0.0.1:8012 go test ./internal/roadmatch -run TestTruth -count=1
//
// (وإذا تغيّر التقطيع أو الطلب، الأجوبة لازم تنعاد تسجيلها — المفتاح بصمة جسم الطلب.)

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/polyline"
)

const truthDir = "testdata/truth"

// ───────────────────────── الطرق ─────────────────────────

// trRoute طريق Valhalla مسجّل: الخط (polyline6) وطوله بالمتر.
type trRoute struct {
	Via   [][2]float64 `json:"via"` // [lat, lng]
	Shape string       `json:"shape"`
	LenM  float64      `json:"len_m"`
}

// trRouteVia نقاط كل طريق (lat, lng) بالرقة — متسلسلة: آخر كل طريق أول اللي بعده بالمشوار المختلط.
var trRouteVia = map[string][][2]float64{
	"west_east":   {{35.9640, 38.9880}, {35.9420, 39.0300}},
	"south_north": {{35.9500, 39.0050}, {35.9700, 39.0200}},
	"downtown":    {{35.9450, 38.9950}, {35.9480, 39.0150}},
	"short":       {{35.9560, 39.0100}, {35.9590, 39.0130}},
	"to_store":    {{35.9480, 39.0150}, {35.9600, 38.9990}},
	"to_customer": {{35.9600, 38.9990}, {35.9520, 39.0230}},
	"back_home":   {{35.9520, 39.0230}, {35.9480, 39.0150}},
	"ring":        {{35.9580, 38.9960}, {35.9660, 39.0120}, {35.9470, 39.0260}, {35.9420, 39.0010}},
}

// ───────────────────────── السيناريوهات ─────────────────────────

type trGap struct {
	at  float64 // جزء الطريق اللي بيبلّش عنده
	dur time.Duration
}

type trLeg struct {
	route      string
	live       bool    // معه طلب: نقطة كل ٣ ث وحدّ ٥ م
	vMin, vMax float64 // سرعة السير م/ث
	stopGo     bool    // زحمة ٥–١٥ كم/س
	lights     int     // إشارات حمرا
	gaps       []trGap // انقطاع إشارة GPS والدراجة ماشية
}

type trStay struct {
	dur    time.Duration
	indoor bool
	live   bool
}

type trScenario struct {
	name     string
	seed     int64
	sigma    float64 // ضجيج GPS برّا (م)
	outlierP float64 // احتمال نتوء ٣٠–٥٠ م بكل نقطة
	steps    []any
	normal   bool // مشوار عادي: هدف ±٥٪ مع المطابقة
}

var trScenarios = []trScenario{
	{name: "order_city", seed: 1, sigma: 6, outlierP: 0.01, normal: true, steps: []any{
		trLeg{route: "west_east", live: true, vMin: 7, vMax: 11, lights: 4}}},
	{name: "idle_cruise", seed: 2, sigma: 8, outlierP: 0.01, normal: true, steps: []any{
		trLeg{route: "south_north", vMin: 6, vMax: 10, lights: 2}}},
	{name: "slow_traffic", seed: 3, sigma: 6, outlierP: 0.01, normal: true, steps: []any{
		trLeg{route: "downtown", live: true, stopGo: true}}},
	{name: "slow_traffic_idle", seed: 4, sigma: 8, normal: true, steps: []any{
		trLeg{route: "downtown", stopGo: true}}},
	{name: "red_lights", seed: 5, sigma: 5, normal: true, steps: []any{
		trLeg{route: "ring", live: true, vMin: 6, vMax: 10, lights: 9}}},
	{name: "noisy_gps", seed: 6, sigma: 15, outlierP: 0.04, normal: true, steps: []any{
		trLeg{route: "west_east", live: true, vMin: 7, vMax: 11, lights: 3}}},
	{name: "signal_gap_3min", seed: 7, sigma: 7, normal: true, steps: []any{
		trLeg{route: "west_east", live: true, vMin: 7, vMax: 10, lights: 2, gaps: []trGap{{0.35, 3 * time.Minute}}}}},
	{name: "signal_gap_idle_5min", seed: 8, sigma: 7, normal: true, steps: []any{
		trLeg{route: "ring", vMin: 6, vMax: 9, lights: 2, gaps: []trGap{{0.3, 5 * time.Minute}}}}},
	{name: "signal_gap_1min_x3", seed: 9, sigma: 7, normal: true, steps: []any{
		trLeg{route: "west_east", live: true, vMin: 7, vMax: 10, lights: 2,
			gaps: []trGap{{0.2, time.Minute}, {0.5, 70 * time.Second}, {0.75, 80 * time.Second}}}}},
	{name: "short_hop", seed: 10, sigma: 6, normal: true, steps: []any{
		trLeg{route: "short", live: true, vMin: 5, vMax: 8, lights: 1}}},
	{name: "indoor_45min", seed: 11, sigma: 8, outlierP: 0.03, steps: []any{
		trStay{dur: 45 * time.Minute, indoor: true}}},
	{name: "indoor_45min_live", seed: 12, sigma: 8, outlierP: 0.03, steps: []any{
		trStay{dur: 45 * time.Minute, indoor: true, live: true}}},
	{name: "parked_outdoor_30min", seed: 13, sigma: 6, outlierP: 0.02, steps: []any{
		trStay{dur: 30 * time.Minute}}},
	{name: "delivery_day", seed: 14, sigma: 8, outlierP: 0.015, normal: true, steps: []any{
		trLeg{route: "downtown", vMin: 6, vMax: 9, lights: 2},
		trStay{dur: 12 * time.Minute, indoor: true},
		trLeg{route: "to_store", live: true, vMin: 6, vMax: 10, lights: 2},
		trStay{dur: 7 * time.Minute, indoor: true, live: true},
		trLeg{route: "to_customer", live: true, stopGo: true, gaps: []trGap{{0.6, 2 * time.Minute}}},
		trStay{dur: 4 * time.Minute, live: true},
		trLeg{route: "back_home", vMin: 6, vMax: 9, lights: 1},
		trStay{dur: 20 * time.Minute, indoor: true}}},
}

// ───────────────────────── المحاكاة ─────────────────────────

const trStep = 500 * time.Millisecond

var trT0 = time.Date(2026, 11, 18, 9, 0, 0, 0, time.UTC)

// trState حالة الدراجة الحقيقية بلحظة.
type trState struct {
	at       time.Time
	lat, lng float64
	v        float64
	indoor   bool
	live     bool
	signal   bool
}

// trSimulate **الحركة الحقيقية** كل نص ثانية، والمسافة الحقيقية بالمتر.
func trSimulate(sc trScenario, routes map[string]trRoute, rng *rand.Rand) ([]trState, float64) {
	var out []trState
	at := trT0
	truth := 0.0
	var lastLat, lastLng float64
	for _, st := range sc.steps {
		switch s := st.(type) {
		case trLeg:
			r := routes[s.route]
			line := polyline.Decode6(r.Shape)
			cum := polyline.Cumulative(line)
			L := cum[len(cum)-1]
			truth += L
			lights := make([]float64, s.lights)
			for i := range lights {
				lights[i] = L * (0.08 + 0.84*rng.Float64())
			}
			sort.Float64s(lights)
			type gapAbs struct{ from, to time.Time }
			var gaps []gapAbs
			gapPending := append([]trGap(nil), s.gaps...)
			pos, v := 0.0, 0.0
			target := s.vMin + (s.vMax-s.vMin)*rng.Float64()
			nextTarget := at.Add(time.Duration(10+rng.Intn(20)) * time.Second)
			var holdUntil time.Time
			goPhase, phaseEnd := true, at.Add(time.Duration(5+rng.Intn(10))*time.Second)
			li := 0
			for pos < L {
				dt := trStep.Seconds()
				for len(gapPending) > 0 && pos >= gapPending[0].at*L {
					gaps = append(gaps, gapAbs{at, at.Add(gapPending[0].dur)})
					gapPending = gapPending[1:]
				}
				want := target
				switch {
				case at.Before(holdUntil):
					want = 0
				case s.stopGo:
					if !at.Before(phaseEnd) {
						goPhase = !goPhase
						if goPhase {
							phaseEnd = at.Add(time.Duration(5+rng.Intn(11)) * time.Second)
							target = 1.5 + 2.7*rng.Float64() // ٥–١٥ كم/س
						} else {
							phaseEnd = at.Add(time.Duration(3+rng.Intn(8)) * time.Second)
						}
					}
					if !goPhase {
						want = 0
					} else {
						want = target
					}
				default:
					if !at.Before(nextTarget) {
						target = s.vMin + (s.vMax-s.vMin)*rng.Float64()
						nextTarget = at.Add(time.Duration(10+rng.Intn(20)) * time.Second)
					}
				}
				if li < len(lights) && pos >= lights[li] && !at.Before(holdUntil) {
					holdUntil = at.Add(time.Duration(20+rng.Intn(41)) * time.Second)
					li++
					want = 0
				}
				// تسارع ٢ م/ث² وفرملة ٣ م/ث².
				if want > v {
					v = math.Min(want, v+2*dt)
				} else {
					v = math.Max(want, v-3*dt)
				}
				pos = math.Min(L, pos+v*dt)
				p := polyline.At(line, cum, pos)
				sig := true
				for _, g := range gaps {
					if !at.Before(g.from) && at.Before(g.to) {
						sig = false
					}
				}
				out = append(out, trState{at: at, lat: p[1], lng: p[0], v: v, live: s.live, signal: sig})
				lastLat, lastLng = p[1], p[0]
				at = at.Add(trStep)
			}
		case trStay:
			if lastLat == 0 {
				r := routes["downtown"]
				for _, x := range sc.steps {
					if l, ok := x.(trLeg); ok {
						r = routes[l.route]
						break
					}
				}
				first := polyline.Decode6(r.Shape)[0]
				lastLat, lastLng = first[1], first[0]
			}
			end := at.Add(s.dur)
			for at.Before(end) {
				out = append(out, trState{at: at, lat: lastLat, lng: lastLng, indoor: s.indoor, live: s.live, signal: true})
				at = at.Add(trStep)
			}
		}
	}
	return out, truth
}

const (
	trMetersLat = 111195.0
)

func trOffset(lat, lng, east, north float64) (float64, float64) {
	return lat + north/trMetersLat, lng + east/(trMetersLat*math.Cos(lat*math.Pi/180))
}

func trF(v float64) *float64 { return &v }

// trPhone **نقاط الجوال** من الحركة الحقيقية: ضجيج مترابط، نتوءات، دقّة وسرعة الجهاز، تواتر التطبيق ونبضته.
func trPhone(sc trScenario, states []trState, rng *rand.Rand) []drivermonitor.Point {
	var out []drivermonitor.Point
	var ex, ey float64 // ضجيج مترابط (م)
	var nextFix time.Time
	var lastSent time.Time
	var last *drivermonitor.Point // آخر نقطة حقيقية بعتها الجوال
	outlierLeft := 0
	var ox, oy, oacc float64
	for i, s := range states {
		sigma, tau := sc.sigma, 40.0
		if s.indoor {
			sigma, tau = 25, 20
		}
		phi := math.Exp(-trStep.Seconds() / tau)
		ex = ex*phi + math.Sqrt(1-phi*phi)*sigma*rng.NormFloat64()
		ey = ey*phi + math.Sqrt(1-phi*phi)*sigma*rng.NormFloat64()
		if i == 0 {
			nextFix = s.at
			lastSent = s.at
		}
		every := 120 * time.Second
		if !s.live {
			every = 10 * time.Second
		}
		if !s.at.Before(nextFix) {
			interval := 5*time.Second + time.Duration(rng.Intn(300))*time.Millisecond
			if s.live {
				interval = 3*time.Second + time.Duration(rng.Intn(500)-200)*time.Millisecond
			}
			nextFix = s.at.Add(interval)
			if s.signal {
				p := trFix(sc, s, ex, ey, rng, &outlierLeft, &ox, &oy, &oacc)
				if s.live && last != nil && drivermonitor.Dist(*last, p) < 5 {
					// حدّ الحركة ٥ م أثناء التوصيل: النظام ما بيسلّم النقطة.
				} else {
					out = append(out, p)
					q := p
					last = &q
					lastSent = s.at
				}
				continue
			}
		}
		// نبضة الواقف: بتعيد آخر موضع (بلا وقت التقاط — الخادم بيحطّ وقته)، أو بتسأل النظام.
		if s.at.Sub(lastSent) >= every {
			lastSent = s.at
			switch {
			case last != nil && s.at.Sub(last.At) <= 300*time.Second:
				out = append(out, drivermonitor.Point{At: s.at, Lat: last.Lat, Lng: last.Lng})
			case s.signal:
				p := trFix(sc, s, ex, ey, rng, &outlierLeft, &ox, &oy, &oacc)
				out = append(out, p)
				q := p
				last = &q
			}
		}
	}
	return out
}

func trFix(sc trScenario, s trState, ex, ey float64, rng *rand.Rand, left *int, ox, oy, oacc *float64) drivermonitor.Point {
	e, n := ex+1.5*rng.NormFloat64(), ey+1.5*rng.NormFloat64()
	acc := math.Max(3, sc.sigma*(0.8+0.6*rng.Float64()))
	if s.indoor {
		acc = 18 + 27*rng.Float64()
	}
	if *left == 0 && rng.Float64() < sc.outlierP {
		*left = 1 + rng.Intn(2)
		a := 2 * math.Pi * rng.Float64()
		d := 30 + 20*rng.Float64()
		*ox, *oy = d*math.Cos(a), d*math.Sin(a)
		if rng.Intn(2) == 0 {
			*oacc = 30 + 20*rng.Float64()
		} else {
			*oacc = 8 + 7*rng.Float64() // الجوال بيكذب بدقّته
		}
	}
	if *left > 0 {
		*left--
		e += *ox
		n += *oy
		acc = *oacc
	}
	lat, lng := trOffset(s.lat, s.lng, e, n)
	var spd *float64
	switch {
	case rng.Float64() < 0.1:
	case s.indoor:
		spd = trF(2.5 * rng.Float64()) // سرعة عشوائية جوّا
	case s.v < 0.2:
		spd = trF(math.Abs(0.2 * rng.NormFloat64()))
	default:
		spd = trF(math.Max(0, s.v+0.4*rng.NormFloat64()))
	}
	return drivermonitor.Point{At: s.at, Lat: lat, Lng: lng, AccuracyM: trF(math.Round(acc*10) / 10), SpeedMps: spd}
}

// trServer **الخادم**: قفزة أسرع من ٧٠ م/ث وأبعد من ١ كم بآخر ١٠ دقايق بتنرمى، وبعدها تخفيف الأثر.
func trServer(pts []drivermonitor.Point) []drivermonitor.Point {
	var kept []drivermonitor.Point
	var ref *drivermonitor.Point
	for _, p := range pts {
		if ref != nil {
			dt := p.At.Sub(ref.At)
			d := drivermonitor.Dist(*ref, p)
			if dt <= 10*time.Minute && d >= 1000 && dt > 0 && d/dt.Seconds() > 70 {
				continue
			}
		}
		q := p
		ref = &q
		kept = append(kept, p)
	}
	return drivermonitor.Filter(nil, kept)
}

// ───────────────────────── Valhalla: تسجيل وإعادة ─────────────────────────

type trFixtures struct {
	Routes  map[string]trRoute `json:"routes"`
	Answers map[string]Answer  `json:"answers"`
	dirty   bool
}

func trKey(sent []drivermonitor.Point) string {
	h := sha256.Sum256(Request(sent))
	return hex.EncodeToString(h[:12])
}

func trLoad(t *testing.T) *trFixtures {
	fx := &trFixtures{Routes: map[string]trRoute{}, Answers: map[string]Answer{}}
	f, err := os.Open(filepath.Join(truthDir, "fixtures.json.gz"))
	if err == nil {
		defer f.Close()
		zr, err := gzip.NewReader(f)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewDecoder(zr).Decode(fx); err != nil {
			t.Fatal(err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	return fx
}

func (fx *trFixtures) save(t *testing.T) {
	if !fx.dirty {
		return
	}
	if err := os.MkdirAll(truthDir, 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	enc := json.NewEncoder(zw)
	if err := enc.Encode(fx); err != nil {
		t.Fatal(err)
	}
	_ = zw.Close()
	if err := os.WriteFile(filepath.Join(truthDir, "fixtures.json.gz"), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

// trRouteFetch طريق دراجة من Valhalla المحلّي.
func trRouteFetch(base string, via [][2]float64) (trRoute, error) {
	type loc struct {
		Lat  float64 `json:"lat"`
		Lon  float64 `json:"lon"`
		Type string  `json:"type,omitempty"`
	}
	var locs []loc
	for i, v := range via {
		l := loc{Lat: v[0], Lon: v[1]}
		if i > 0 && i < len(via)-1 {
			l.Type = "through" // بلا رجعة U عند النقطة الوسطى
		}
		locs = append(locs, l)
	}
	body, _ := json.Marshal(map[string]any{"locations": locs, "costing": Costing, "units": "kilometers"})
	resp, err := http.Post(base+"/route", "application/json", bytes.NewReader(body))
	if err != nil {
		return trRoute{}, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return trRoute{}, fmt.Errorf("route %d: %s", resp.StatusCode, raw)
	}
	var j struct {
		Trip struct {
			Legs []struct {
				Shape string `json:"shape"`
			} `json:"legs"`
		} `json:"trip"`
	}
	if err := json.Unmarshal(raw, &j); err != nil {
		return trRoute{}, err
	}
	var line [][2]float64
	for _, l := range j.Trip.Legs {
		seg := polyline.Decode6(l.Shape)
		if len(line) > 0 && len(seg) > 0 && line[len(line)-1] == seg[0] {
			seg = seg[1:]
		}
		line = append(line, seg...)
	}
	return trRoute{Via: via, Shape: polyline.Encode6(line), LenM: polyline.Length(line)}, nil
}

// trMatcher بيرجّع الجواب المسجّل، أو بيسأل Valhalla المحلّي ويسجّل (وضع التسجيل).
type trMatcher struct {
	fx     *trFixtures
	client *Client
	miss   int
	used   map[string]bool
}

func (m *trMatcher) Match(ctx context.Context, sent []drivermonitor.Point) (Answer, error) {
	k := trKey(sent)
	if m.used == nil {
		m.used = map[string]bool{}
	}
	m.used[k] = true
	if a, ok := m.fx.Answers[k]; ok {
		if a.Shape == "" && len(a.MatchedPoints) == 0 {
			return Answer{}, &NoMatchError{Code: "recorded"}
		}
		return a, nil
	}
	if m.client == nil {
		m.miss++
		return Answer{}, ErrDown
	}
	a, err := m.client.Match(ctx, sent)
	var nm *NoMatchError
	switch {
	case err == nil:
	case errors.As(err, &nm):
		a = Answer{}
	default:
		return Answer{}, err
	}
	m.fx.Answers[k] = a
	m.fx.dirty = true
	if err != nil {
		return Answer{}, err
	}
	return a, nil
}

// ───────────────────────── المقياس ─────────────────────────

// trRoad **نفس ما بيكتبه المطابق ويقراه `LoadRoad`**: قطع → أسطر → قطع طريق (بدقّة float32 متل القاعدة).
func trRoad(t *testing.T, trail []drivermonitor.Point, m Matcher) *drivermonitor.Road {
	cfg := drivermonitor.DefaultConfig()
	until := trail[len(trail)-1].At.Add(time.Hour)
	var rows []Row
	for _, c := range Chunks(trail, cfg, until) {
		if len(c.Points) < 2 {
			continue
		}
		if c.SentCount() < 2 {
			rows = append(rows, FlatRow(c, "ok", ""))
			continue
		}
		a, err := m.Match(context.Background(), sentPoints(c))
		var nm *NoMatchError
		switch {
		case err == nil:
			rows = append(rows, Rows(c, a)...)
		case errors.As(err, &nm):
			rows = append(rows, FlatRow(c, "no_match", nm.Code))
		default:
			return nil // ما في جواب مسجّل
		}
	}
	var chunks []drivermonitor.RoadChunk
	for _, r := range rows {
		c := drivermonitor.RoadChunk{FromAt: r.FromAt, ToAt: r.ToAt, Closed: r.Closed, OK: r.Status == "ok",
			PointAt: r.PointAt, Line: polyline.Decode6(r.Line6)}
		for _, v := range r.PosM {
			c.PosM = append(c.PosM, float64(float32(v)))
		}
		chunks = append(chunks, c)
	}
	return drivermonitor.NewRoad(chunks, true)
}

type trResult struct {
	name, gapNote        string
	normal               bool
	truthKm              float64
	offKm, onKm          float64 // المقياس المعتمد (`Moved`) بلا مطابقة ومعها
	oldOffKm, oldOnKm    float64 // القاعدة القديمة (`RealDrive`) بلا مطابقة ومعها
	downKm               float64 // المطابقة شغّالة و Valhalla واقع (ولا قطعة)
	countedKm, roadShare float64
	fixes, trail         int
}

func (r trResult) errPct(km float64) float64 {
	if r.truthKm == 0 {
		return math.NaN()
	}
	return (km - r.truthKm) / r.truthKm * 100
}

// trFramed **السائق واقف قبل المشوار وبعده** (دقيقتين برّا) — متل الواقع: الأثر ما بيخلص والدراجة ماشية.
func trFramed(sc trScenario) trScenario {
	if len(sc.steps) == 0 {
		return sc
	}
	if l, ok := sc.steps[0].(trLeg); ok {
		sc.steps = append([]any{trStay{dur: 2 * time.Minute, live: l.live}}, sc.steps...)
	}
	if l, ok := sc.steps[len(sc.steps)-1].(trLeg); ok {
		sc.steps = append(append([]any(nil), sc.steps...), trStay{dur: 2 * time.Minute, live: l.live})
	}
	return sc
}

// trRun بيشغّل سيناريو كامل.
func trRun(t *testing.T, sc trScenario, fx *trFixtures, m Matcher) trResult {
	rng := rand.New(rand.NewSource(sc.seed))
	states, truth := trSimulate(sc, fx.Routes, rng)
	phone := trPhone(sc, states, rng)
	trail := trServer(phone)
	res := trResult{name: sc.name, normal: sc.normal, truthKm: truth / 1000, fixes: len(phone), trail: len(trail)}
	for _, st := range sc.steps {
		if l, ok := st.(trLeg); ok && len(l.gaps) > 0 {
			res.gapNote = "gap"
		}
	}
	cfg := drivermonitor.DefaultConfig()
	old := cfg
	old.Meter = drivermonitor.RealDrive{Cfg: cfg}
	measure := func(c drivermonitor.Config, road *drivermonitor.Road) drivermonitor.Summary {
		return drivermonitor.Analyze(drivermonitor.Input{Points: trail, Cfg: c, Road: road}).Summary
	}
	off := measure(cfg, nil)
	res.offKm, res.countedKm = off.DriveKmTotal, off.KmTotal
	res.oldOffKm = measure(old, nil).DriveKmTotal
	// **Valhalla واقع**: المطابقة شغّالة وما في ولا قطعة ⇒ «لسّا عم تنحسب» بالخط المستقيم، مو صفر.
	res.downKm = measure(cfg, drivermonitor.NewRoad(nil, true)).DriveKmTotal
	road := trRoad(t, trail, m)
	if road == nil {
		res.onKm, res.oldOnKm = math.NaN(), math.NaN()
		return res
	}
	on := measure(cfg, road)
	res.onKm, res.roadShare = on.DriveKmTotal, on.RoadShare
	res.oldOnKm = measure(old, road).DriveKmTotal
	return res
}

// trSeeds كل سيناريو بتلات بذور — مو رقم محظوظ.
var trSeeds = []int64{0, 1, 2}

// trAll بيحمّل الأجهزة (ويسجّل إذا في Valhalla محلّي) وبيشغّل كل السيناريوهات.
func trAll(t *testing.T) []trResult {
	t.Helper()
	fx := trLoad(t)
	base := strings.TrimRight(os.Getenv("TRUTH_VALHALLA_URL"), "/")
	if strings.Contains(base, "rahalgo.com") || strings.Contains(base, "195.201.141.130") {
		t.Fatal("مختبر الحقيقة ما بيسأل الإنتاج أبداً")
	}
	m := &trMatcher{fx: fx}
	if base != "" {
		m.client = &Client{Base: base, HTTP: &http.Client{}, Timeout: 20 * time.Second}
		for name, via := range trRouteVia {
			if _, ok := fx.Routes[name]; ok && os.Getenv("TRUTH_REFRESH_ROUTES") == "" {
				continue
			}
			r, err := trRouteFetch(base, via)
			if err != nil {
				t.Fatalf("طريق %s: %v", name, err)
			}
			fx.Routes[name] = r
			fx.dirty = true
		}
	}
	for name := range trRouteVia {
		if _, ok := fx.Routes[name]; !ok {
			t.Fatalf("طريق %s مو مسجّل — سجّل بـTRUTH_VALHALLA_URL", name)
		}
	}
	var out []trResult
	for _, sc := range trScenarios {
		for _, off := range trSeeds {
			run := sc
			run.seed = sc.seed*1000 + off
			out = append(out, trRun(t, trFramed(run), fx, m))
		}
	}
	if m.client != nil {
		// بس الأجوبة المستعملة بتنحفظ — والمحاولة التانية بتلاقيها كلها مسجّلة.
		for k := range fx.Answers {
			if !m.used[k] {
				delete(fx.Answers, k)
				fx.dirty = true
			}
		}
		fx.save(t)
	}
	if m.miss > 0 {
		t.Fatalf("%d جواب Valhalla مو مسجّل — التقطيع أو الطلب تغيّر؛ سجّل من جديد بـTRUTH_VALHALLA_URL", m.miss)
	}
	return out
}

// trAgg سيناريو بكل بذوره: المتوسّط وأسوأ بذرة.
type trAgg struct {
	name                         string
	normal, gap                  bool
	truthKm                      float64
	n                            int
	oldOff, oldOn, off, on, down float64 // متوسّط الخطأ ٪ (أو كم للواقف)
	onWorst, offWorst            float64 // أسوأ بذرة (٪ أو كم)
}

func trAggregate(rs []trResult) []trAgg {
	var out []trAgg
	idx := map[string]int{}
	for _, r := range rs {
		i, ok := idx[r.name]
		if !ok {
			i = len(out)
			idx[r.name] = i
			out = append(out, trAgg{name: r.name, normal: r.normal, gap: r.gapNote != "", truthKm: r.truthKm})
		}
		a := &out[i]
		v := func(km float64) float64 {
			if r.truthKm == 0 {
				return km
			}
			return r.errPct(km)
		}
		a.n++
		a.oldOff += v(r.oldOffKm)
		a.oldOn += v(r.oldOnKm)
		a.off += v(r.offKm)
		a.on += v(r.onKm)
		a.down += v(r.downKm)
		if math.Abs(v(r.onKm)) > math.Abs(a.onWorst) || math.IsNaN(v(r.onKm)) {
			a.onWorst = v(r.onKm)
		}
		if math.Abs(v(r.offKm)) > math.Abs(a.offWorst) {
			a.offWorst = v(r.offKm)
		}
	}
	for i := range out {
		f := float64(out[i].n)
		out[i].oldOff /= f
		out[i].oldOn /= f
		out[i].off /= f
		out[i].on /= f
		out[i].down /= f
	}
	return out
}

func trTable(t *testing.T, as []trAgg) {
	t.Helper()
	var b strings.Builder
	f := func(a trAgg, v float64) string {
		if a.truthKm == 0 {
			return fmt.Sprintf("%+.3fkm", v)
		}
		return fmt.Sprintf("%+.1f%%", v)
	}
	fmt.Fprintf(&b, "\nmean over %d seeds (worst seed in brackets)\n%-22s %7s | %9s %9s | %9s %9s %10s %10s | %9s\n", len(trSeeds),
		"scenario", "truth", "old-off", "old-on", "new-off", "new-on", "worst-off", "worst-on", "down")
	for _, a := range as {
		fmt.Fprintf(&b, "%-22s %7.3f | %9s %9s | %9s %9s %10s %10s | %9s\n", a.name, a.truthKm,
			f(a, a.oldOff), f(a, a.oldOn), f(a, a.off), f(a, a.on), "["+f(a, a.offWorst)+"]", "["+f(a, a.onWorst)+"]", f(a, a.down))
	}
	t.Log(b.String())
}

// TestTruth_KmVsGroundTruth **الرقم الحقيقي**: كل سيناريو (بتلات بذور) مقابل طوله الحقيقي ع طرق Valhalla مسجّلة.
//
//   - الواقف (جوّا محل ٤٥ د أو برّا) ⇒ ≤ ١٠٠ م بكل بذرة، بلا مطابقة ومعها (القديمة: ٠٫٦–١ كم).
//   - المشوار العادي **مع المطابقة** ⇒ المتوسّط ±٥٪ (وبانقطاع إشارة ±٨٪) وأسوأ بذرة ±١٠٪.
//   - **بلا مطابقة** ⇒ المتوسّط ±٥٪ (المشوار القصير < ١ كم ±١٠٪، وبانقطاع إشارة −١٣٪: خط مستقيم فوقه).
//   - Valhalla واقع والمطابقة شغّالة ⇒ **نفس رقم بلا مطابقة بالضبط** (مو صفر).
func TestTruth_KmVsGroundTruth(t *testing.T) {
	rs := trAll(t)
	for _, r := range rs {
		if r.downKm != r.offKm {
			t.Errorf("%s/%d: Valhalla واقع ⇒ %.3f كم والمتوقَّع نفس بلا مطابقة %.3f", r.name, r.trail, r.downKm, r.offKm)
		}
		if r.truthKm == 0 && (r.offKm > 0.1 || r.onKm > 0.1) {
			t.Errorf("%s: واقف وطلع %.3f كم بلا مطابقة و%.3f معها — رجفة صارت كيلومترات", r.name, r.offKm, r.onKm)
		}
	}
	as := trAggregate(rs)
	trTable(t, as)
	for _, a := range as {
		if !a.normal {
			continue
		}
		onLim := 5.0
		if a.gap {
			onLim = 8 // انقطاع ٥ د: القفزة فوق ٢ كم بتنقطع عند Valhalla ⇒ خط مستقيم
		}
		if math.IsNaN(a.on) || math.Abs(a.on) > onLim || math.Abs(a.onWorst) > 10 {
			t.Errorf("%s: مع المطابقة %+.1f%% (أسوأ %+.1f%%) — الهدف ±%.0f%% (وأسوأ ±١٠٪)", a.name, a.on, a.onWorst, onLim)
		}
		lim := 5.0
		switch {
		case a.gap:
			lim = 13
		case a.truthKm < 1:
			lim = 10
		}
		if math.Abs(a.off) > lim {
			t.Errorf("%s: بلا مطابقة %+.1f%% — الحدّ ±%.0f%%", a.name, a.off, lim)
		}
	}
}
