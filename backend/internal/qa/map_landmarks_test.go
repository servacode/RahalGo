package qa

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestMapLandmarks_BboxClassAndDedupe **معالمُ الخريطة**: النطاقُ يُقصي ما خارجَه، والصنفُ بمفردات
// البلاطة، والمقبرةُ لا تُرسم، والمكرَّرُ القريبُ يسقط، والمتجرُ معلمٌ ويُستثنى بـ`stores=0`.
func TestMapLandmarks_BboxClassAndDedupe(t *testing.T) {
	h := New(t)
	mosque := uniq("جامع الخريطة ")
	far := uniq("مستشفى بعيد ")
	grave := uniq("مقبرة الخريطة ")
	ins := func(name, kind string, lng, lat float64) {
		if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO map_landmarks (name, kind, point, source)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, 'qa')`, name, kind, lng, lat); err != nil {
			t.Fatal(err)
		}
	}
	// صندوقُ الاختبار صغيرٌ في مكانٍ لا يستعمله غيرُه.
	const lng, lat = 38.9001, 35.8001
	ins(mosque, "amenity=place_of_worship", lng, lat)
	ins(mosque, "amenity=place_of_worship", lng+0.0002, lat) // مكرَّرٌ على ~٢٠ م
	ins(grave, "amenity=grave_yard", lng+0.001, lat)
	ins(far, "amenity=hospital", lng+0.5, lat)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name IN ($1,$2,$3)`, mosque, grave, far)
	})

	const bbox = "38.89,35.79,38.91,35.81"
	res := h.GET("/api/v1/public/map/landmarks?format=geojson&stores=0&bbox="+bbox, "")
	if res.Code != 200 {
		t.Fatalf("المعالم: %s", res)
	}
	var fc struct {
		Features []struct {
			Properties map[string]any `json:"properties"`
		} `json:"features"`
	}
	if err := json.Unmarshal(res.Body, &fc); err != nil {
		t.Fatal(err)
	}
	mosques := 0
	for _, f := range fc.Features {
		switch f.Properties["name"] {
		case mosque:
			mosques++
			if f.Properties["class"] != "place_of_worship" || f.Properties["category"] != "mosque" {
				t.Fatalf("صنفُ المسجد: %v", f.Properties)
			}
		case grave, far:
			t.Fatalf("خرج ما لا يجوز: %v", f.Properties)
		}
	}
	if mosques != 1 {
		t.Fatalf("المسجدُ المكرَّر يجب أن يبقى واحداً، وجدتُ %d", mosques)
	}
	if !strings.Contains(res.Head.Get("Cache-Control"), "max-age") {
		t.Fatalf("لا ترويسةَ تخزين: %q", res.Head.Get("Cache-Control"))
	}

	// صندوقٌ يشمل البعيدَ.
	wide := h.GET("/api/v1/public/map/landmarks?stores=0&bbox=38.8,35.7,39.5,35.9", "")
	if wide.Code != 200 || !strings.Contains(string(wide.Body), far) {
		t.Fatalf("الصندوقُ الواسعُ لا يرى المستشفى: %s", wide)
	}

	// المتجر معلمٌ، ويُستثنى بـ stores=0.
	m := h.Factory().Merchant()
	var mlat, mlng float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(location::geometry), ST_X(location::geometry) FROM merchants WHERE id = $1::uuid`, m.ID).Scan(&mlat, &mlng); err != nil {
		t.Fatal(err)
	}
	box := func(extra string) string {
		return "/api/v1/public/map/landmarks?" + extra + "bbox=" +
			ftoa(mlng-0.01) + "," + ftoa(mlat-0.01) + "," + ftoa(mlng+0.01) + "," + ftoa(mlat+0.01)
	}
	var name string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT name FROM merchants WHERE id = $1::uuid`, m.ID).Scan(&name)
	if with := h.GET(box(""), ""); with.Code != 200 || !strings.Contains(string(with.Body), name) {
		t.Fatalf("المتجرُ ليس معلماً: %s", with)
	}
	if without := h.GET(box("stores=0&"), ""); without.Code != 200 || strings.Contains(string(without.Body), `"s`+m.ID+`"`) {
		t.Fatalf("stores=0 لم يُقصِ المتجر: %s", without)
	}

	if bad := h.GET("/api/v1/public/map/landmarks?bbox=1,2,3", ""); bad.Code != 400 {
		t.Fatalf("نطاقٌ فاسد يجب 400: %d", bad.Code)
	}
}
