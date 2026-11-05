package server

import (
	"encoding/json"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **معالمُ الخريطة تُرسم على كلّ خريطة** (قرارُ المالك ٢٠٢٦-١١-٠٣: «الخريطة ما فيها معالم كثيرة»)
// ══════════════════════════════════════════════════════════════════════
//
// `map_landmarks` كان لا يُستعمل إلّا لنصّ «قرب جامع X». **صار يُرسم.**
//
// **والصنفُ بمفردات بلاطاتِنا نفسِها** (`class` في طبقة `poi` من النمط) — فيستعير العارضُ
// أيقونةَ `poi-label` وخطَّها ولونَها كما هي، **فلا يختلف معلمُنا عن معلمِ البلاطة.**
// **والمكرَّرُ مع البلاطة تحجبه المزاحمةُ** (طبقتُنا فوقَ طبقة البلاطة).

// landmarkClass **يردّ صنفَ البلاطة** لنوعِ OSM، **وفارغاً لما لا يُرسم** (مقابر وأسماءُ
// أحياءٍ ومناطقَ — للأخيرة طبقاتُ `place` في النمط).
func landmarkClass(kind string) string {
	switch kind {
	case "amenity=place_of_worship", "amenity=mosque", "building=mosque":
		return "place_of_worship"
	case "amenity=hospital", "building=hospital":
		return "hospital"
	case "amenity=clinic", "amenity=doctors", "healthcare=clinic", "healthcare=doctor":
		return "doctors"
	case "amenity=dentist":
		return "dentist"
	case "amenity=pharmacy":
		return "pharmacy"
	case "amenity=school", "building=school", "office=educational_institution":
		return "school"
	case "amenity=kindergarten":
		return "kindergarten"
	case "amenity=college":
		return "college"
	case "amenity=university", "building=university":
		return "university"
	case "amenity=library":
		return "library"
	case "amenity=police":
		return "police"
	case "amenity=fire_station":
		return "fire_station"
	case "amenity=townhall", "amenity=courthouse", "office=government", "building=government", "building=public":
		return "town_hall"
	case "amenity=post_office":
		return "post"
	case "amenity=fuel":
		return "fuel"
	case "amenity=bank":
		return "bank"
	case "amenity=restaurant", "amenity=fast_food":
		return "restaurant"
	case "amenity=cafe":
		return "cafe"
	case "amenity=marketplace", "shop=mall", "shop=department_store":
		return "shop"
	case "leisure=park", "leisure=garden":
		return "park"
	case "leisure=stadium", "leisure=sports_centre":
		return "stadium"
	case "tourism=hotel", "tourism=hostel":
		return "lodging"
	case "tourism=attraction", "tourism=museum", "tourism=viewpoint",
		"historic=castle", "historic=monument", "historic=memorial", "historic=ruins",
		"historic=archaeological_site", "historic=yes", "place=square", "highway=mini_roundabout",
		"junction=roundabout":
		return "attraction"
	case "shop=bakery", "shop=confectionery":
		return "bakery"
	case "shop=supermarket", "shop=convenience", "shop=greengrocer":
		return "grocery"
	}
	if strings.HasPrefix(kind, "shop=") {
		return "shop"
	}
	return ""
}

// landmarkCategory **اسمُ الفئة للمستهلكين الذين لا يقرؤون مفردات البلاطة** (الأدمن والتقارير).
func landmarkCategory(class string) string {
	switch class {
	case "place_of_worship":
		return "mosque"
	case "hospital", "doctors", "dentist":
		return "hospital"
	case "pharmacy":
		return "pharmacy"
	case "school", "kindergarten", "college", "university", "library":
		return "school"
	case "police", "fire_station", "town_hall", "post":
		return "government"
	case "park":
		return "park"
	case "attraction":
		return "square"
	case "shop", "grocery", "bakery":
		return "market"
	}
	return "other"
}

// landmarkRank **أصغرُه أهمُّه** — يرتّب المزاحمةَ: المسجدُ والمستشفى والمدرسةُ قبل الدكّان.
func landmarkRank(class string) int {
	switch landmarkCategory(class) {
	case "mosque", "hospital", "school", "government":
		return 10
	case "pharmacy", "park", "square":
		return 20
	case "market":
		return 60
	}
	return 40
}

var (
	// labelLatinRun **كلمةٌ لاتينيّةٌ بما التصق بها** من أرقامٍ وشرطةٍ ونقطة:
	// «Al-Bakkor» · «Bra3em» · «Kozmetik.basmacioglu» · «-Hbaebna».
	labelLatinRun = regexp.MustCompile(`[-_.'’&0-9]*\p{Latin}[\p{Latin}\p{M}0-9_.'’&-]*`)
	// labelBareBrackets **قوسان لم يبقَ بينهما عربيّ** — «مطعم (Pizza)» ← «مطعم ( )».
	labelBareBrackets = regexp.MustCompile(`[(\[{][^()\[\]{}\p{Arabic}]*[)\]}]`)
	// labelSepRun **فاصلان تجاورا بعد الإسقاط** — يبقى الأوّل: «قطن - Coton - دول» ← «قطن - دول».
	labelSepRun   = regexp.MustCompile(`([-–—/\\|,،:;·•])(?:[\s\p{Zs}]*[-–—/\\|,،:;·•])+`)
	labelSpaceRun = regexp.MustCompile(`[\s\p{Zs}]+`)
	// labelWrapped **التسميةُ كلُّها بين قوسين** — «Al Buder (قطن حلو الشعيب)» ← «(قطن حلو الشعيب)».
	labelWrapped = regexp.MustCompile(`^[(\[{]([^()\[\]{}]*)[)\]}]$`)
)

// labelEdge **فاصلٌ لا تبدأ به تسميةٌ ولا تنتهي** — بعد إسقاط اللاتينيّ.
func labelEdge(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(`-–—/\|,،:;·•_+=&*`, r)
}

// landmarkLabel **تسميةُ المعلم على الخريطة — عربيّةٌ وحدَها** (قرارُ المالك ٢٠٢٦-١١-٠٣: «كلشي
// بالعربي لازم»، ونمطُ f85a48e6 لا يعرض `name` الخامَ أبداً). وأسماءُ Overture مخلوطة:
//
//	«مستشفى الرازّي - Alrazzi hospital»  ←  «مستشفى الرازّي»
//	«Stirk complex / مجمع سترك»          ←  «مجمع سترك»
//
// **الكلمةُ اللاتينيّةُ تُسقَط بما التصق بها، ثمّ ما تيتّم بعدها** من فواصلَ وأقواسٍ ومسافات.
// **وما لا حرفَ عربيَّ فيه يُردّ فارغاً** — فلا تسميةَ له والأيقونةُ باقية. **والعربيُّ الخالصُ كما هو.**
// **ولأسماء `map_landmarks` وحدَها** — اسمُ متجرِنا لا يمرّ بها.
func landmarkLabel(name string) string {
	s := strings.TrimSpace(name)
	if strings.IndexFunc(s, func(r rune) bool { return unicode.Is(unicode.Latin, r) }) >= 0 {
		s = labelLatinRun.ReplaceAllString(s, " ")
		s = labelBareBrackets.ReplaceAllString(s, " ")
		s = labelSepRun.ReplaceAllString(s, "$1")
		s = strings.TrimFunc(labelSpaceRun.ReplaceAllString(s, " "), labelEdge)
		if m := labelWrapped.FindStringSubmatch(s); m != nil {
			s = strings.TrimFunc(m[1], labelEdge)
		}
	}
	if strings.IndexFunc(s, func(r rune) bool { return unicode.IsLetter(r) && unicode.Is(unicode.Arabic, r) }) < 0 {
		return ""
	}
	return s
}

type landmarkOut struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Class string  `json:"class"`
	Cat   string  `json:"category"`
	Rank  int     `json:"rank"`
	Lat   float64 `json:"lat"`
	Lng   float64 `json:"lng"`
}

// handlePublicMapLandmarks GET /public/map/landmarks
//
//	bbox=غرب,جنوب,شرق,شمال   اختياريّ — وبلاه كلُّ المغطّى
//	stores=0                  بلا متاجرِنا (الخرائطُ التي ترسمها طبقةً مستقلّة)
//	limit=N                   الافتراضيُّ ٦٠٠٠، والأعلى ١٥٠٠٠
//	format=geojson            لطبقات الخريطة؛ وغيرُه `{landmarks:[…]}`
func (s *Server) handlePublicMapLandmarks(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := 6000
	if n, err := strconv.Atoi(q.Get("limit")); err == nil && n > 0 {
		limit = min(n, 15000)
	}
	// ‫`bbox` يُحلَّل ويُتحقَّق منه — وغيرُ الصالح 400 لا يُتجاهل.‬
	// الافتراضيُّ العالمُ كلُّه (والمقارنةُ هندسيّةٌ لا جغرافيّة: غلافُ `geography` بهذا الاتّساع ينقلب فارغاً).
	minLng, minLat, maxLng, maxLat := -179.0, -85.0, 179.0, 85.0
	if bb := strings.TrimSpace(q.Get("bbox")); bb != "" {
		parts := strings.Split(bb, ",")
		vals := make([]float64, 0, 4)
		for _, p := range parts {
			v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
			if err != nil || math.IsNaN(v) || math.IsInf(v, 0) {
				break
			}
			vals = append(vals, v)
		}
		if len(vals) != 4 || vals[0] >= vals[2] || vals[1] >= vals[3] {
			s.respondErr(w, httpx.NewError(http.StatusBadRequest, "bad_bbox", "errors.bad_request"))
			return
		}
		minLng, minLat = max(vals[0], -179), max(vals[1], -85)
		maxLng, maxLat = min(vals[2], 179), min(vals[3], 85)
		if minLng >= maxLng || minLat >= maxLat {
			s.respondErr(w, httpx.NewError(http.StatusBadRequest, "bad_bbox", "errors.bad_request"))
			return
		}
	}
	withStores := q.Get("stores") != "0"

	rows, err := s.pg.Query(r.Context(), `
		SELECT 'l' || id::text, name, kind, ST_Y(point::geometry), ST_X(point::geometry), 0
		  FROM map_landmarks
		 WHERE point::geometry && ST_MakeEnvelope($1,$2,$3,$4,4326)
		   AND btrim(name) <> ''
		UNION ALL
		SELECT 's' || m.id::text, m.name, 'shop=store', ST_Y(m.location::geometry), ST_X(m.location::geometry), 1
		  FROM merchants m
		 WHERE $5 AND m.status = 'active' AND m.location IS NOT NULL
		   AND m.location::geometry && ST_MakeEnvelope($1,$2,$3,$4,4326)
		 LIMIT 40000`, minLng, minLat, maxLng, maxLat, withStores)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	out := []landmarkOut{}
	// **مكرَّرُ الاسم القريب يُسقَط** — الأوّلُ يبقى (المعالمُ قبل المتاجر في الاستعلام).
	seen := map[string][][2]float64{}
	for rows.Next() {
		var id, name, kind string
		var lat, lng float64
		var isStore int
		if err := rows.Scan(&id, &name, &kind, &lat, &lng, &isStore); err != nil {
			s.respondErr(w, err)
			return
		}
		class := landmarkClass(kind)
		if isStore == 1 {
			class = "shop"
		}
		name = strings.TrimSpace(name)
		if class == "" || name == "" {
			continue
		}
		dup := false
		for _, p := range seen[name] {
			// ≈ ١٠٠ م (درجةُ عرضٍ = ١١١ كم)
			if math.Abs(p[0]-lat) < 0.0009 && math.Abs(p[1]-lng) < 0.0011 {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		seen[name] = append(seen[name], [2]float64{lat, lng})
		rank := landmarkRank(class)
		if isStore == 1 {
			rank = 55
		}
		out = append(out, landmarkOut{ID: id, Name: name, Class: class, Cat: landmarkCategory(class), Rank: rank, Lat: lat, Lng: lng})
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	if len(out) > limit {
		// الأهمُّ يبقى حين يزيد العدد.
		kept := make([]landmarkOut, 0, limit)
		for rank := 0; rank <= 100 && len(kept) < limit; rank += 5 {
			for _, x := range out {
				if x.Rank >= rank && x.Rank < rank+5 && len(kept) < limit {
					kept = append(kept, x)
				}
			}
		}
		out = kept
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	if q.Get("format") == "geojson" {
		feats := make([]map[string]any, 0, len(out))
		for _, x := range out {
			// `class` و`name:ar` و`rank` بمفردات البلاطة — فتُستعار طبقتُها كما هي.
			props := map[string]any{
				"id": x.ID, "name": x.Name,
				"class": x.Class, "category": x.Cat, "rank": x.Rank,
			}
			// **وتسميةُ المعلم عربيّةٌ وحدَها** (`landmarkLabel`) — وما لا عربيَّ فيه بلا `name:ar` فلا اسمَ يُرسم.
			// و`name` الخامُ باقٍ: النمطُ لا يعرضه، يقرؤه مفتاحاً لجدول التعريب وحدَه.
			// **ومتجرُنا (`s…`) باسمه كما كتبه صاحبُه** — القرارُ عن بيانات الخريطة لا عن اسم المحلّ،
			// وتطبيقا الزبون والسائق يرسمانه خاماً من `/public/map/stores`: «KFC الرقة» لا «الرقة».
			label := x.Name
			if strings.HasPrefix(x.ID, "l") {
				label = landmarkLabel(x.Name)
			}
			if label != "" {
				props["name:ar"] = label
			}
			feats = append(feats, map[string]any{
				"type":       "Feature",
				"geometry":   map[string]any{"type": "Point", "coordinates": []float64{x.Lng, x.Lat}},
				"properties": props,
			})
		}
		w.Header().Set("Content-Type", "application/geo+json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "FeatureCollection", "features": feats})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"landmarks": out})
}
