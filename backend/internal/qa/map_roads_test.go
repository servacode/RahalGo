package qa

// ══════════════════════════════════════════════════════════════════════
// **تعلّم الطرق: اتّجاه واحد وسرعة حقيقية** — قرار المالك ٢٠٢٦-١١-٠٣ («اي موافق»)
// ══════════════════════════════════════════════════════════════════════
//
// آثارٌ اصطناعيّة على شوارع شرقيّة-غربيّة بمطابقٍ مزيَّف (شبكةُ حافّاتٍ بعرض ٠٫٠٠٢° تقريباً ٢٠٠ م):
//   - شارع أ: اتّجاهٌ واحد شرقاً (٦ سائقين × ٦ أيّام + عبورٌ شاذّ غرباً) → مقترح.
//   - شارع ب: اتّجاهان بالتساوي → لا.
//   - شارع ج: سائقٌ واحدٌ فقط (بيانات غير كافية) → لا.
//   - شارع د: اتّجاهٌ واحد غرباً → مقترحٌ بهندسةٍ غربيّة.
//   - شارع س: سريعٌ الساعة ١٠ (١٠ م/ث) وبطيءٌ الساعة ١٧ (٢٫٥ م/ث) → ليس بطيئاً باستمرار.
//   - شارع ز: بطيءٌ في كلّ الساعات → بطيءٌ باستمرار.

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

const (
	mrLatA = 35.40
	mrLatB = 35.41
	mrLatC = 35.42
	mrLatD = 35.43
	mrLatS = 35.44
	mrLatZ = 35.45
)

// mrFake مطابقٌ مزيَّف: الشارعُ بخطّ عرضه، والحافّةُ خليّةٌ بعرض ٠٫٠٠٢° شرقاً-غرباً.
type mrFake struct{}

func (mrFake) Match(_ context.Context, pts []drivermonitor.Point) (maplearn.MatchResult, error) {
	lat := pts[0].Lat
	if lat < 35.395 || lat > 35.455 {
		return maplearn.MatchResult{}, maplearn.ErrNoMatch // أثرُ اختباراتٍ أخرى
	}
	const w = 0.002
	cell := func(lng float64) int { return int(math.Floor((lng - 38.0) / w)) }
	c0, c1 := cell(pts[0].Lng), cell(pts[len(pts)-1].Lng)
	step := 1
	if c1 < c0 {
		step = -1
	}
	way := int64(1000 + math.Round((lat-35.4)*100))
	res := maplearn.MatchResult{PointEdge: make([]int, len(pts))}
	idx := map[int]int{}
	for c := c0; ; c += step {
		a, b := 38.0+float64(c)*w, 38.0+float64(c+1)*w
		travel := [][2]float64{{a, lat}, {b, lat}}
		if step == -1 {
			travel = [][2]float64{{b, lat}, {a, lat}}
		}
		key, dir, canon := maplearn.CanonEdge(way, travel)
		idx[c] = len(res.Edges)
		res.Edges = append(res.Edges, maplearn.MatchedEdge{
			Key: key, WayID: way, Name: "شارع الاختبار", Length: 180, Dir: dir, Line: canon,
		})
		if c == c1 {
			break
		}
	}
	for i, p := range pts {
		if e, ok := idx[cell(p.Lng)]; ok {
			res.PointEdge[i] = e
		} else {
			res.PointEdge[i] = -1
		}
	}
	return res, nil
}

// mrDrive سائقٌ يمشي شرقاً/غرباً على الشارع بسرعةٍ ثابتة، نقطةً كل ٢٠ ثانية، يبدأ بعد `offset` من ١٠:٠٠ قبل `ago` يوماً.
func mrDrive(t *testing.T, h *Harness, driver string, ago int, offset time.Duration, lat, fromLng, toLng, mps float64) {
	t.Helper()
	_, base := dmDay()
	cur := base.AddDate(0, 0, 1-ago).Add(offset) // dmDay = أمس؛ ago=1 → أمس
	mPerDeg := 111320 * math.Cos(lat*math.Pi/180)
	dir := 1.0
	if toLng < fromLng {
		dir = -1
	}
	stepDeg := dir * mps * 20 / mPerDeg
	n := int(math.Abs(toLng-fromLng)/math.Abs(stepDeg)) + 1
	for i := 0; i <= n; i++ {
		_, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_trail (driver_id, recorded_at, lat, lng, accuracy_m, speed_mps, mocked)
			VALUES ($1::uuid, $2, $3, $4, 5, $5, false) ON CONFLICT DO NOTHING`,
			driver, cur.Add(time.Duration(i)*20*time.Second), lat, fromLng+float64(i)*stepDeg, mps)
		if err != nil {
			t.Fatalf("كتابة نقطة: %v", err)
		}
	}
}

type mrOneway struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	WayID      int64   `json:"way_id"`
	MainPasses int     `json:"main_passes"`
	OppPasses  int     `json:"opp_passes"`
	Drivers    int     `json:"drivers"`
	Days       int     `json:"days"`
	Share      float64 `json:"share"`
	Geometry   struct {
		Coordinates [][2]float64 `json:"coordinates"`
	} `json:"geometry"`
}

func mrOnewayList(t *testing.T, h *Harness, tok, status string) []mrOneway {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/oneway-candidates?status="+status, tok)
	if res.Code != 200 {
		t.Fatalf("قائمة الاتّجاه الواحد: %s", res)
	}
	var env struct {
		Data struct {
			Candidates []mrOneway `json:"candidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك: %v (%s)", err, res)
	}
	return env.Data.Candidates
}

func mrOnLat(cs []mrOneway, lat float64) []mrOneway {
	var out []mrOneway
	for _, c := range cs {
		if len(c.Geometry.Coordinates) > 0 && math.Abs(c.Geometry.Coordinates[0][1]-lat) < 0.0005 {
			out = append(out, c)
		}
	}
	return out
}

type mrSpeedFeat struct {
	Geometry struct {
		Coordinates [][2]float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Name   string  `json:"name"`
		Passes int     `json:"passes"`
		Kmh    float64 `json:"kmh"`
		Slow   bool    `json:"slow"`
	} `json:"properties"`
}

func mrSpeeds(t *testing.T, h *Harness, tok, query string) []mrSpeedFeat {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/road-speeds"+query, tok)
	if res.Code != 200 {
		t.Fatalf("السرعات: %s", res)
	}
	var env struct {
		Data struct {
			Features []mrSpeedFeat `json:"features"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك: %v (%s)", err, res)
	}
	return env.Data.Features
}

func mrSpeedOn(fs []mrSpeedFeat, lat float64) []mrSpeedFeat {
	var out []mrSpeedFeat
	for _, f := range fs {
		if len(f.Geometry.Coordinates) > 0 && math.Abs(f.Geometry.Coordinates[0][1]-lat) < 0.0005 {
			out = append(out, f)
		}
	}
	return out
}

func TestRoads_OnewayAndSpeeds(t *testing.T) {
	h := New(t)
	f := h.Factory()
	ctx := ctxBG()
	wipe := func() {
		for _, q := range []string{
			`DELETE FROM map_oneway_candidates`, `DELETE FROM map_edge_passes`, `DELETE FROM map_edge_speed`,
			`DELETE FROM map_road_edges`, `DELETE FROM map_road_days`,
		} {
			if _, err := h.Pool.Exec(ctxBG(), q); err != nil {
				t.Fatal(err)
			}
		}
	}
	wipe()
	t.Cleanup(wipe)
	adm := h.NewUser("admin").Token

	var drv []string
	for i := 0; i < 6; i++ {
		drv = append(drv, f.Driver().ID)
	}
	const west, east = 38.400, 38.420
	for ago := 1; ago <= 6; ago++ {
		for i, d := range drv {
			// أ: شرقاً دائماً (ويمرّ سائقٌ واحدٌ غرباً مرّةً واحدة: شذوذ)
			mrDrive(t, h, d, ago, 0, mrLatA, west, east, 8)
			if ago == 1 && i == 0 {
				mrDrive(t, h, d, ago, 30*time.Minute, mrLatA, east, west, 8)
			}
			// ب: يتناوب الاتّجاه بين السائقين
			if i%2 == 0 {
				mrDrive(t, h, d, ago, time.Hour, mrLatB, west, east, 8)
			} else {
				mrDrive(t, h, d, ago, time.Hour, mrLatB, east, west, 8)
			}
			// د: غرباً دائماً
			mrDrive(t, h, d, ago, 2*time.Hour, mrLatD, east, west, 8)
		}
		// ج: سائقٌ واحدٌ فقط، ٥ عبورات كلّ يوم
		for k := 0; k < 5; k++ {
			mrDrive(t, h, drv[0], ago, 3*time.Hour+time.Duration(k)*10*time.Minute, mrLatC, west, east, 8)
		}
		// س: سريعٌ ١٠ ص، بطيءٌ ٥ م — ثلاثة سائقين
		// ز: بطيءٌ ١٠ ص و١٢ و٢ و٥ م — ثلاثة سائقين
		for _, d := range drv[:3] {
			mrDrive(t, h, d, ago, 20*time.Minute, mrLatS, west, east, 10)
			mrDrive(t, h, d, ago, 7*time.Hour, mrLatS, west, east, 2.5)
			for _, off := range []time.Duration{40 * time.Minute, 150 * time.Minute, 4 * time.Hour, 450 * time.Minute} {
				mrDrive(t, h, d, ago, off, mrLatZ, west, west+0.006, 2.5)
			}
		}
	}

	res, err := maplearn.RunRoads(ctx, h.Pool, mrFake{}, maplearn.DefaultRoadOptions())
	if err != nil {
		t.Fatalf("الجولة: %v", err)
	}
	if res.Days < 6 || res.OneWay == 0 {
		t.Fatalf("نتيجة الجولة %+v", res)
	}

	// ── الاتّجاه الواحد ──
	cs := mrOnewayList(t, h, adm, "pending")
	a := mrOnLat(cs, mrLatA)
	if len(a) < 8 {
		t.Fatalf("شارع أ: %d مقترحاً والمتوقَّع حافّاتٌ كثيرة", len(a))
	}
	for _, c := range a {
		g := c.Geometry.Coordinates
		if g[0][0] >= g[len(g)-1][0] {
			t.Errorf("أ: الهندسة يجب أن تكون شرقاً: %v", g)
		}
		if c.Drivers != 6 || c.Days != 6 || c.Share < 0.95 || c.MainPasses < 36 || c.OppPasses > 1 || c.WayID == 0 {
			t.Errorf("أ: الأدلّة %+v", c)
		}
	}
	for _, c := range mrOnLat(cs, mrLatD) {
		g := c.Geometry.Coordinates
		if g[0][0] <= g[len(g)-1][0] {
			t.Errorf("د: الهندسة يجب أن تكون غرباً: %v", g)
		}
	}
	if len(mrOnLat(cs, mrLatD)) < 8 {
		t.Errorf("شارع د: %d والمتوقَّع حافّاتٌ كثيرة", len(mrOnLat(cs, mrLatD)))
	}
	if n := len(mrOnLat(cs, mrLatB)); n != 0 {
		t.Errorf("ب باتّجاهين: %d مقترحاً والمتوقَّع صفر", n)
	}
	if n := len(mrOnLat(cs, mrLatC)); n != 0 {
		t.Errorf("ج بسائقٍ واحد: %d مقترحاً والمتوقَّع صفر", n)
	}

	// ── القرار والتصدير ──
	idPath := func(id int64) string {
		return "/api/v1/admin/ops-map/oneway-candidates/" + strconv.FormatInt(id, 10) + "/decide"
	}
	if r := h.POST(idPath(a[0].ID), adm, map[string]any{"approve": true}); r.Code != 200 {
		t.Fatalf("الموافقة: %s", r)
	}
	if r := h.POST(idPath(a[0].ID), adm, map[string]any{"approve": false}); r.Code != 409 {
		t.Errorf("قرارٌ ثانٍ = %d والمتوقَّع 409", r.Code)
	}
	if r := h.POST(idPath(a[1].ID), adm, map[string]any{"approve": false}); r.Code != 200 {
		t.Fatalf("الرفض: %s", r)
	}
	osm := h.GET("/api/v1/admin/ops-map/oneway-candidates/export?format=osm", adm)
	if osm.Code != 200 || !strings.Contains(string(osm.Body), `k="oneway" v="yes"`) ||
		!strings.Contains(string(osm.Body), `rahalgo:osm_way_id`) || strings.Count(string(osm.Body), "<way ") != 1 {
		t.Errorf("تصدير OSM: %d %s", osm.Code, osm.Body)
	}
	gj := h.GET("/api/v1/admin/ops-map/oneway-candidates/export?format=geojson", adm)
	if gj.Code != 200 || !strings.Contains(string(gj.Body), `"oneway":"yes"`) {
		t.Errorf("تصدير GeoJSON: %d %s", gj.Code, gj.Body)
	}
	if r := h.GET("/api/v1/admin/ops-map/oneway-candidates/export?format=pdf", adm); r.Code != 422 {
		t.Errorf("صيغةٌ مجهولة = %d والمتوقَّع 422", r.Code)
	}

	// ── السرعة الحقيقية ──
	s10 := mrSpeedOn(mrSpeeds(t, h, adm, "?hour=10"), mrLatS)
	s17 := mrSpeedOn(mrSpeeds(t, h, adm, "?hour=17"), mrLatS)
	if len(s10) < 8 || len(s17) < 8 {
		t.Fatalf("س: %d و%d حافّة", len(s10), len(s17))
	}
	for _, x := range s10 {
		if math.Abs(x.Properties.Kmh-36) > 3 {
			t.Errorf("س ١٠ص: %.1f كم/س والمتوقَّع ~٣٦", x.Properties.Kmh)
		}
		if x.Properties.Slow || x.Properties.Name == "" {
			t.Errorf("س: لا يجب أن تكون بطيئةً باستمرار، والاسمُ ظاهر: %+v", x.Properties)
		}
	}
	for _, x := range s17 {
		if math.Abs(x.Properties.Kmh-9) > 1.5 {
			t.Errorf("س ٥م: %.1f كم/س والمتوقَّع ~٩", x.Properties.Kmh)
		}
	}
	zs := mrSpeedOn(mrSpeeds(t, h, adm, ""), mrLatZ)
	if len(zs) == 0 {
		t.Fatal("ز: لا حافّات")
	}
	for _, x := range zs {
		if !x.Properties.Slow {
			t.Errorf("ز: يجب أن تُعلَّم بطيئةً باستمرار: %+v", x.Properties)
		}
	}
	if n := len(mrSpeedOn(mrSpeeds(t, h, adm, "?hour=3"), mrLatS)); n != 0 {
		t.Errorf("ساعةٌ بلا بيانات: %d حافّة", n)
	}
	sum := h.GET("/api/v1/admin/ops-map/road-speeds/summary", adm)
	var se struct {
		Data maplearn.SpeedSummary `json:"data"`
	}
	if sum.Code != 200 || json.Unmarshal(sum.Body, &se) != nil || se.Data.SlowEdges == 0 || len(se.Data.Buckets) < 4 {
		t.Errorf("ملخّص السرعة: %d %s", sum.Code, sum.Body)
	}

	// ── إعادةُ التشغيل لا تضاعف ولا تعيد المقرَّر ──
	var before, after int
	_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(sum(passes), 0)::int FROM map_edge_speed`).Scan(&before)
	res2, err := maplearn.RunRoads(ctx, h.Pool, mrFake{}, maplearn.DefaultRoadOptions())
	if err != nil || res2.Days != 0 {
		t.Fatalf("الجولة الثانية %+v %v", res2, err)
	}
	_ = h.Pool.QueryRow(ctx, `SELECT COALESCE(sum(passes), 0)::int FROM map_edge_speed`).Scan(&after)
	if before != after {
		t.Errorf("أيّامٌ عُولجت مرّتين: %d ≠ %d", before, after)
	}
	if got := len(mrOnewayList(t, h, adm, "approved")); got != 1 {
		t.Errorf("موافَق: %d والمتوقَّع 1", got)
	}
	if got := len(mrOnewayList(t, h, adm, "rejected")); got != 1 {
		t.Errorf("مرفوض: %d والمتوقَّع 1", got)
	}
	for _, c := range mrOnewayList(t, h, adm, "pending") {
		if c.ID == a[0].ID || c.ID == a[1].ID {
			t.Error("أُعيد اقتراحُ مقرَّر")
		}
	}

	// ── الملخّص والتشغيل ──
	if r := h.GET("/api/v1/admin/ops-map/road-learning/summary", adm); r.Code != 200 || !strings.Contains(string(r.Body), `"days":`) {
		t.Errorf("ملخّص القسم: %d %s", r.Code, r.Body)
	}
}

func TestRoads_Permissions(t *testing.T) {
	h := New(t)
	for _, role := range []string{"customer", "driver", "finance"} {
		tok := h.NewUser(role).Token
		for _, p := range []string{"oneway-candidates", "road-speeds", "road-speeds/summary", "road-learning/summary"} {
			if r := h.GET("/api/v1/admin/ops-map/"+p, tok); r.Code != 403 {
				t.Errorf("%s %s = %d والمتوقَّع 403", role, p, r.Code)
			}
		}
		if r := h.POST("/api/v1/admin/ops-map/road-learning/run", tok, nil); r.Code != 403 {
			t.Errorf("%s: التشغيل = %d والمتوقَّع 403", role, r.Code)
		}
	}
	ops := h.NewUser("operations").Token
	if r := h.POST("/api/v1/admin/ops-map/oneway-candidates/1/decide", ops, map[string]any{"approve": true}); r.Code != 403 {
		t.Errorf("العمليات تقرّر = %d والمتوقَّع 403", r.Code)
	}
	if r := h.POST("/api/v1/admin/ops-map/road-learning/run", ops, nil); r.Code != 403 {
		t.Errorf("العمليات تشغّل = %d والمتوقَّع 403", r.Code)
	}
	if r := h.GET("/api/v1/admin/ops-map/oneway-candidates/export?format=osm", ops); r.Code != 403 {
		t.Errorf("العمليات تصدّر = %d والمتوقَّع 403", r.Code)
	}
	adm := h.NewUser("admin").Token
	if r := h.POST("/api/v1/admin/ops-map/oneway-candidates/999999999/decide", adm, map[string]any{"approve": false}); r.Code != 404 {
		t.Errorf("مجهول = %d والمتوقَّع 404", r.Code)
	}
}
