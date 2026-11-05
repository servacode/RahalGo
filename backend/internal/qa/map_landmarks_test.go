package qa

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode"
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

	// بلا نطاقٍ: كلُّ المغطّى (وهو ما تقرؤه التطبيقات) — كان 500.
	if all := h.GET("/api/v1/public/map/landmarks?format=geojson&stores=0", ""); all.Code != 200 || !strings.Contains(string(all.Body), far) {
		t.Fatalf("بلا نطاق: %s", all)
	}
	if bad := h.GET("/api/v1/public/map/landmarks?bbox=1,2,3", ""); bad.Code != 400 {
		t.Fatalf("نطاقٌ فاسد يجب 400: %d", bad.Code)
	}
}

// TestMapLandmarks_LabelArabicOnly **تسميةُ المعلم عربيّةٌ وحدَها** (قرارُ المالك ٢٠٢٦-١١-٠٣: «كلشي
// بالعربي لازم»). كان الاسمُ الخامُ يُكتب في `name:ar` كما هو، **فيظهر «مستشفى الرازّي - Alrazzi
// hospital» على خريطة الزبون.** فالمخلوطُ يُرسم بعربيّه، **واللاتينيُّ وحدَه بلا تسمية** — والنقطةُ
// (الأيقونة) باقية، **والاسمُ الخامُ في `name` كما هو** (لا تقرؤه الخريطةُ إلّا مفتاحاً لجدول التعريب).
//
// **ومتجرُنا باسمه كما كتبه صاحبُه** — القرارُ عن بيانات الخريطة لا عن اسم المحلّ، وتطبيقا الزبون
// والسائق يرسمانه خاماً من `/public/map/stores`. **كان التنظيفُ يمرّ عليه أيضاً** فيصير «KFC الرقة»
// «الرقة» ويُرسم متجرٌ لاتينيُّ الاسم بلا اسم في خرائط الموقع وتطبيقَي المتجر والمندوب.
func TestMapLandmarks_LabelArabicOnly(t *testing.T) {
	h := New(t)
	tag := uniq("")
	mixed := "مستشفى الرازّي " + tag + " - Alrazzi hospital"
	latin := "Norhan Shop " + tag
	ins := func(name, kind string, lng, lat float64) {
		if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO map_landmarks (name, kind, point, source)
			VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, 'qa')`, name, kind, lng, lat); err != nil {
			t.Fatal(err)
		}
	}
	// صندوقٌ صغيرٌ لا يستعمله غيرُه.
	const lng, lat = 38.9501, 35.8501
	ins(mixed, "amenity=hospital", lng, lat)
	ins(latin, "shop=clothes", lng+0.002, lat)
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM map_landmarks WHERE name IN ($1,$2)`, mixed, latin)
	})

	// props **خصائصُ نقاط الصندوق باسمها الخام.**
	props := func(stores string) map[string]map[string]any {
		t.Helper()
		res := h.GET("/api/v1/public/map/landmarks?format=geojson&stores="+stores+"&bbox=38.94,35.84,38.96,35.86", "")
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
		out := map[string]map[string]any{}
		for _, f := range fc.Features {
			if name, ok := f.Properties["name"].(string); ok {
				out[name] = f.Properties
			}
		}
		return out
	}
	// landmarks **المخلوطُ بعربيّه واللاتينيُّ بلا تسمية** — بلا متاجرَ ومعها.
	landmarks := func(got map[string]map[string]any) {
		t.Helper()
		p, ok := got[mixed]
		if !ok {
			t.Fatalf("المعلمُ المخلوطُ غاب: %v", got)
		}
		label, _ := p["name:ar"].(string)
		if want := "مستشفى الرازّي " + tag; label != want {
			t.Fatalf("تسميةُ المخلوط %q — يُنتظر %q", label, want)
		}
		for _, r := range label {
			if unicode.Is(unicode.Latin, r) {
				t.Fatalf("حرفٌ لاتينيٌّ في التسمية: %q", label)
			}
		}
		if p, ok = got[latin]; !ok {
			t.Fatalf("المعلمُ اللاتينيُّ غاب: %v", got)
		}
		if v, ok := p["name:ar"]; ok {
			t.Fatalf("اللاتينيُّ وحدَه لا تسميةَ له، وجدتُ name:ar=%v", v)
		}
		if p["class"] != "shop" {
			t.Fatalf("نقطةُ المعلم اللاتينيّ بلا صنف: %v", p)
		}
	}
	landmarks(props("0"))

	// **متجرانِ نشطانِ في الصندوق نفسِه**: لاتينيُّ الاسم ومخلوطُه — **وكلاهما باسمه كما هو.**
	fac := h.Factory()
	stores := []string{"Pizza Roma " + tag, "KFC الرقة " + tag}
	for i, name := range stores {
		m := fac.Merchant()
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET name = $2,
			location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography WHERE id = $1::uuid`,
			m.ID, name, lng+0.004+0.002*float64(i), lat); err != nil {
			t.Fatal(err)
		}
	}
	got := props("1")
	landmarks(got)
	for _, name := range stores {
		p, ok := got[name]
		if !ok {
			t.Fatalf("المتجرُ %q غاب عن الخريطة: %v", name, got)
		}
		if id, _ := p["id"].(string); !strings.HasPrefix(id, "s") {
			t.Fatalf("نقطةُ %q ليست متجراً: %v", name, p)
		}
		if p["name:ar"] != name {
			t.Fatalf("تسميةُ المتجر %q صارت %v — يُنتظر اسمُه كما كتبه صاحبُه", name, p["name:ar"])
		}
	}
}
