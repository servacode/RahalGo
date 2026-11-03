package qa

// ══════════════════════════════════════════════════════════════════════
// **أسماءُ الأحياء والشوارع من عناوين الزبائن** — قرار المالك ٢٠٢٦-١١-٠٣
// ══════════════════════════════════════════════════════════════════════
//
// عناوينُ اصطناعيّة: حيٌّ ذكره ٣ زبائنَ فيُقترح بغلافٍ ونقطة، وحيٌّ بزبونَين لا، وحيٌّ موجودٌ في المعالم لا،
// وشارعٌ يلصق عليه زبونان فأكثر بلاصقٍ مزيَّف. الموافقةُ باسمٍ معدَّل تُدخل `map_area_names` فتقرؤه الخرائطُ
// ويصدّره المدير، والرفضُ لا يُعاد اقتراحُه، والأبوابُ بالصلاحية.

import (
	"context"
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

type areaCand struct {
	ID        int64           `json:"id"`
	Source    string          `json:"source"`
	Name      string          `json:"name"`
	Lat       float64         `json:"lat"`
	Lng       float64         `json:"lng"`
	Customers int             `json:"customers"`
	Evidence  int             `json:"evidence"`
	Samples   []string        `json:"samples"`
	Geometry  json.RawMessage `json:"geometry"`
	RadiusM   float64         `json:"radius_m"`
	WayIDs    []int64         `json:"way_ids"`
}

func areaList(t *testing.T, h *Harness, tok, query string) []areaCand {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/place-candidates"+query, tok)
	if res.Code != 200 {
		t.Fatalf("قائمة المرشَّحين: %s", res)
	}
	var env struct {
		Data struct {
			Candidates []areaCand `json:"candidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatalf("فك القائمة: %v (%s)", err, res)
	}
	return env.Data.Candidates
}

func areaByName(cs []areaCand, name string) *areaCand {
	for i := range cs {
		if cs[i].Name == name {
			return &cs[i]
		}
	}
	return nil
}

func areaDecide(id int64) string {
	return "/api/v1/admin/ops-map/place-candidates/" + strconv.FormatInt(id, 10) + "/decide"
}

// areaLocator يلصق كلَّ نقطةٍ بأقرب طريقٍ أفقيٍّ من القائمة (خطُّ عرضه ثابت).
type areaLocator struct{ ways map[int64]float64 }

func (a areaLocator) Locate(_ context.Context, pts [][2]float64) ([]maplearn.LocatedEdge, error) {
	out := make([]maplearn.LocatedEdge, len(pts))
	for i, p := range pts {
		for w, lat := range a.ways {
			d := math.Abs(p[0]-lat) * 111320
			if out[i].WayID == 0 || d < out[i].DistM {
				out[i] = maplearn.LocatedEdge{WayID: w, DistM: d,
					Line: [][2]float64{{p[1] - 0.002, lat}, {p[1] + 0.002, lat}}}
			}
		}
	}
	return out, nil
}

func TestAreas_HoodsApproveReadExport(t *testing.T) {
	h := New(t)
	ctx := ctxBG()
	clean := func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_place_candidates WHERE source IN ('neighbourhoods', 'streets')`)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_area_names`)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name = 'المشلب' AND source = 'osm-test'`)
	}
	clean()
	t.Cleanup(clean)
	adm := h.NewUser("admin").Token

	var (
		a = dmSpot{34.2000, 37.5000} // «الدرعية» — ٣ زبائن بصيغٍ مختلفة
		b = dmSpot{34.2500, 37.5000} // «الجزرة» — زبونان فقط
		c = dmSpot{34.3000, 37.5000} // «المشلب» — موجودٌ في المعالم
	)
	mpAddress(t, h, "حي الدرعية، جوال 0933123456", a)
	mpAddress(t, h, "بحي الدرعيّة قرب الفرن", dmSpot{a.lat + 0.0006, a.lng + 0.0004})
	mpAddress(t, h, "الحي الدرعيه", dmSpot{a.lat - 0.0005, a.lng + 0.0007})
	mpAddress(t, h, "حارة الجزرة", b)
	mpAddress(t, h, "حارة الجزرة", dmSpot{b.lat + 0.0003, b.lng})
	for i := 0; i < 3; i++ {
		mpAddress(t, h, "حي المشلب", dmSpot{c.lat + float64(i)*0.0003, c.lng})
	}
	if _, err := h.Pool.Exec(ctx, `
		INSERT INTO map_landmarks (name, kind, point, source)
		VALUES ('المشلب', 'place=suburb', ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, 'osm-test')`,
		c.lat+0.001, c.lng); err != nil {
		t.Fatal(err)
	}

	res, err := maplearn.RunAreas(ctx, h.Pool, nil, maplearn.DefaultAreaParams())
	if err != nil {
		t.Fatalf("الجولة: %v", err)
	}
	if res.Neighbourhoods < 1 || res.Streets != 0 {
		t.Fatalf("النتيجة %+v", res)
	}
	cs := areaList(t, h, adm, "?source=neighbourhoods")
	d := areaByName(cs, "الدرعية")
	if d == nil || d.Customers != 3 || d.Source != "neighbourhoods" || math.Abs(d.Lat-a.lat) > 0.001 {
		t.Fatalf("الدرعية: %+v (كلّها %+v)", d, cs)
	}
	if !strings.Contains(string(d.Geometry), "Polygon") || d.RadiusM <= 0 {
		t.Errorf("الغلاف والنصف قطر: %s %v", d.Geometry, d.RadiusM)
	}
	for _, s := range d.Samples {
		if strings.Contains(s, "0933123456") {
			t.Errorf("هاتفٌ في الدليل: %q", s)
		}
	}
	if areaByName(cs, "الجزرة") != nil {
		t.Error("زبونان لا يكفيان للحيّ")
	}
	if areaByName(cs, "المشلب") != nil {
		t.Error("الحيّ الموجود في المعالم لا يُقترح")
	}
	// ولا يُحمَل تحتَ مصدر المعالم.
	if got := areaList(t, h, adm, "?source=addresses"); areaByName(got, "الدرعية") != nil {
		t.Error("حيٌّ في قائمة المعالم")
	}

	// موافقةٌ باسمٍ قصيرٍ مرفوضة؛ ثم باسمٍ معدَّل.
	if r := h.POST(areaDecide(d.ID), adm, map[string]any{"approve": true, "name": "ح"}); r.Code != 422 {
		t.Errorf("اسم قصير = %d والمتوقَّع 422", r.Code)
	}
	if r := h.POST(areaDecide(d.ID), adm, map[string]any{"approve": true, "name": "حي الدرعية الجديد"}); r.Code != 200 {
		t.Fatalf("موافقة: %s", r)
	}
	if r := h.POST(areaDecide(d.ID), adm, map[string]any{"approve": false}); r.Code != 409 {
		t.Errorf("قرار مكرَّر = %d والمتوقَّع 409", r.Code)
	}
	var n int
	if err := h.Pool.QueryRow(ctx, `
		SELECT count(*)::int FROM map_area_names WHERE kind = 'neighbourhood' AND name = 'حي الدرعية الجديد' AND geom IS NOT NULL`).
		Scan(&n); err != nil || n != 1 {
		t.Fatalf("الحيّ المُقرّ: n=%d err=%v", n, err)
	}
	// ولا يدخل `map_landmarks`.
	if err := h.Pool.QueryRow(ctx, `SELECT count(*)::int FROM map_landmarks WHERE name = 'حي الدرعية الجديد'`).Scan(&n); err != nil || n != 0 {
		t.Errorf("دخل المعالم: n=%d", n)
	}

	// تقرؤه الخريطةُ بلا رمز، ضمن الصندوق فقط.
	box := "37.49,34.19,37.51,34.21"
	pub := h.GET("/api/v1/public/map/areas?bbox="+box, "")
	if pub.Code != 200 || !strings.Contains(string(pub.Body), "حي الدرعية الجديد") ||
		!strings.Contains(string(pub.Body), `"kind":"neighbourhood"`) {
		t.Fatalf("القراءة العامّة: %s", pub)
	}
	if far := h.GET("/api/v1/public/map/areas?bbox=10,10,10.1,10.1", ""); far.Code != 200 || strings.Contains(string(far.Body), "الدرعية") {
		t.Errorf("خارج الصندوق: %s", far)
	}
	if bad := h.GET("/api/v1/public/map/areas?bbox=1,2,3", ""); bad.Code != 422 {
		t.Errorf("صندوق فاسد = %d والمتوقَّع 422", bad.Code)
	}

	// التصدير: عقدةُ حيٍّ في OSM.
	osm := h.GET("/api/v1/admin/ops-map/area-names/export?format=osm", adm)
	if osm.Code != 200 || !strings.Contains(string(osm.Body), `k="place" v="neighbourhood"`) ||
		!strings.Contains(string(osm.Body), `v="حي الدرعية الجديد"`) {
		t.Fatalf("تصدير osm: %s", osm)
	}
	if r := h.GET("/api/v1/admin/ops-map/area-names/export?format=pdf", adm); r.Code != 422 {
		t.Errorf("صيغة فاسدة = %d والمتوقَّع 422", r.Code)
	}

	// الرفض لا يُعاد اقتراحُه.
	mpAddress(t, h, "حارة الجزرة", dmSpot{b.lat, b.lng + 0.0004})
	if _, err := maplearn.RunAreas(ctx, h.Pool, nil, maplearn.DefaultAreaParams()); err != nil {
		t.Fatal(err)
	}
	j := areaByName(areaList(t, h, adm, "?source=neighbourhoods"), "الجزرة")
	if j == nil || j.Customers != 3 {
		t.Fatalf("الجزرة بثلاثة زبائن: %+v", j)
	}
	if r := h.POST(areaDecide(j.ID), adm, map[string]any{"approve": false}); r.Code != 200 {
		t.Fatalf("رفض: %s", r)
	}
	if _, err := maplearn.RunAreas(ctx, h.Pool, nil, maplearn.DefaultAreaParams()); err != nil {
		t.Fatal(err)
	}
	if areaByName(areaList(t, h, adm, "?source=neighbourhoods"), "الجزرة") != nil {
		t.Error("أُعيد اقتراح حيٍّ مرفوض")
	}
	if rej := areaList(t, h, adm, "?status=rejected&source=neighbourhoods"); areaByName(rej, "الجزرة") == nil {
		t.Error("المرفوض لا يظهر في تبويبه")
	}
	// ولا الحيُّ المُقرّ.
	if areaByName(areaList(t, h, adm, "?source=neighbourhoods"), "الدرعية") != nil {
		t.Error("أُعيد اقتراح حيٍّ مُقرّ")
	}
}

func TestAreas_StreetsSnapApproveExport(t *testing.T) {
	h := New(t)
	ctx := ctxBG()
	clean := func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_place_candidates WHERE source IN ('neighbourhoods', 'streets')`)
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_area_names`)
	}
	clean()
	t.Cleanup(clean)
	adm := h.NewUser("admin").Token

	lat := 34.6000
	loc := areaLocator{ways: map[int64]float64{777001: lat, 777002: lat + 0.05}}
	// ثلاثة زبائن على الطريق ٧٧٧٠٠١ يذكرون «شارع الكورنيش»، وزبونٌ واحدٌ يذكر «شارع الوحيد».
	mpAddress(t, h, "شارع الكورنيش مقابل الفرن", dmSpot{lat + 0.0002, 37.60})
	mpAddress(t, h, "على شارع الكورنيش", dmSpot{lat - 0.0003, 37.61})
	mpAddress(t, h, "الشارع الكورنيش رقم 3", dmSpot{lat, 37.62})
	mpAddress(t, h, "شارع الوحيد", dmSpot{lat, 37.63})

	res, err := maplearn.RunAreas(ctx, h.Pool, loc, maplearn.DefaultAreaParams())
	if err != nil || res.Streets < 1 {
		t.Fatalf("الجولة: %+v %v", res, err)
	}
	cs := areaList(t, h, adm, "?source=streets")
	s := areaByName(cs, "شارع الكورنيش")
	if s == nil || s.Customers != 3 || len(s.WayIDs) != 1 || s.WayIDs[0] != 777001 || !strings.Contains(string(s.Geometry), "MultiLineString") {
		t.Fatalf("الكورنيش: %+v (كلّها %+v)", s, cs)
	}
	if areaByName(cs, "شارع الوحيد") != nil {
		t.Error("زبون واحد لا يكفي للشارع")
	}

	if r := h.POST(areaDecide(s.ID), adm, map[string]any{"approve": true, "name": "شارع الكورنيش القديم"}); r.Code != 200 {
		t.Fatalf("موافقة: %s", r)
	}
	pub := h.GET("/api/v1/public/map/areas?bbox=37.5,34.5,37.7,34.7", "")
	if pub.Code != 200 || !strings.Contains(string(pub.Body), "شارع الكورنيش القديم") ||
		!strings.Contains(string(pub.Body), `"kind":"street"`) || !strings.Contains(string(pub.Body), "MultiLineString") {
		t.Fatalf("القراءة العامّة للشارع: %s", pub)
	}
	wn := h.GET("/api/v1/admin/ops-map/area-names/export?format=waynames", adm)
	if wn.Code != 200 || !strings.Contains(string(wn.Body), `"way_id":777001`) || !strings.Contains(string(wn.Body), "شارع الكورنيش القديم") {
		t.Fatalf("تصدير أسماء الطرق: %s", wn)
	}
	gj := h.GET("/api/v1/admin/ops-map/area-names/export?format=geojson", adm)
	if gj.Code != 200 || !strings.Contains(string(gj.Body), "FeatureCollection") {
		t.Fatalf("تصدير geojson: %s", gj)
	}

	// بلا لاصق (Valhalla غائب) تعمل الأحياء ولا تتأذّى الشوارعُ المنتظرة.
	mpAddress(t, h, "شارع الكورنيش الجديد", dmSpot{lat, 37.64})
	if _, err := maplearn.RunAreas(ctx, h.Pool, nil, maplearn.DefaultAreaParams()); err != nil {
		t.Fatalf("بلا لاصق: %v", err)
	}
	// والمُقرُّ لا يُعاد اقتراحُه.
	if _, err := maplearn.RunAreas(ctx, h.Pool, loc, maplearn.DefaultAreaParams()); err != nil {
		t.Fatal(err)
	}
	for _, c := range areaList(t, h, adm, "?source=streets") {
		if c.Name == "شارع الكورنيش" || c.Name == "شارع الكورنيش القديم" {
			t.Errorf("أُعيد اقتراح شارعٍ مُقرّ: %+v", c)
		}
	}
}

func TestAreas_Permissions(t *testing.T) {
	h := New(t)
	for _, role := range []string{"customer", "driver", "finance"} {
		tok := h.NewUser(role).Token
		if r := h.GET("/api/v1/admin/ops-map/area-names/export?format=osm", tok); r.Code != 403 {
			t.Errorf("%s: التصدير = %d والمتوقَّع 403", role, r.Code)
		}
		if r := h.GET("/api/v1/admin/ops-map/place-candidates?source=streets", tok); r.Code != 403 {
			t.Errorf("%s: القائمة = %d والمتوقَّع 403", role, r.Code)
		}
	}
	adm := h.NewUser("admin").Token
	if r := h.POST(areaDecide(999999999), adm, map[string]any{"approve": false}); r.Code != 404 {
		t.Errorf("مجهول = %d والمتوقَّع 404", r.Code)
	}
}
