package server

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **أسماءُ الأحياء والشوارع المُقرَّة من عناوين الزبائن** (قرارُ المالك ٢٠٢٦-١١-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// الجدولُ `map_area_names` يحوي ما وافق عليه المدير من `map_place_candidates`. تقرؤه الخرائطُ هنا
// (طبقتا اسم الحيّ واسم الشارع على خطّه) ويصدّره المدير لدفعة بناء البلاطات التالية فتدخل الأسماءُ البلاطةَ نفسَها.
// **لا شيءَ شخصيّ هنا**: اسمٌ وهندسة. فالقراءةُ عامّة كمعالم الخريطة.

var errAreaFormat = httpx.NewError(http.StatusUnprocessableEntity, "bad_format", "errors.validation")

// parseBBox يحلّل `غرب,جنوب,شرق,شمال`. فارغٌ = كلُّ المغطّى؛ وغيرُ الصالح خطأ.
func parseBBox(raw string) (minLng, minLat, maxLng, maxLat float64, err error) {
	minLng, minLat, maxLng, maxLat = -180, -90, 180, 90
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	vals := make([]float64, 0, 4)
	for _, p := range strings.Split(raw, ",") {
		v, perr := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if perr != nil || math.IsNaN(v) || math.IsInf(v, 0) {
			break
		}
		vals = append(vals, v)
	}
	if len(vals) != 4 || vals[0] >= vals[2] || vals[1] >= vals[3] {
		return 0, 0, 0, 0, errAreaFormat
	}
	return vals[0], vals[1], vals[2], vals[3], nil
}

type areaRow struct {
	ID       int64
	Kind     string
	Name     string
	Lat, Lng float64
	RadiusM  float64
	WayIDs   []int64
	Geom     string
}

// handlePublicMapAreas GET /public/map/areas?bbox=غرب,جنوب,شرق,شمال
// — GeoJSON: نقطةٌ لكلّ حيٍّ (`kind=neighbourhood`) وخطٌّ لكلّ شارعٍ (`kind=street`) باسمه العربيّ.
func (s *Server) handlePublicMapAreas(w http.ResponseWriter, r *http.Request) {
	minLng, minLat, maxLng, maxLat, err := parseBBox(r.URL.Query().Get("bbox"))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, kind, name, ST_Y(point::geometry), ST_X(point::geometry), radius_m,
		       CASE WHEN kind = 'street' THEN COALESCE(ST_AsGeoJSON(geom::geometry)::text, '') ELSE '' END
		  FROM map_area_names
		 WHERE btrim(name) <> ''
		   AND COALESCE(geom, point) && ST_MakeEnvelope($1,$2,$3,$4,4326)::geography
		 ORDER BY id LIMIT 5000`, minLng, minLat, maxLng, maxLat)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	feats := []map[string]any{}
	for rows.Next() {
		var a areaRow
		if err := rows.Scan(&a.ID, &a.Kind, &a.Name, &a.Lat, &a.Lng, &a.RadiusM, &a.Geom); err != nil {
			s.respondErr(w, err)
			return
		}
		var geometry any = map[string]any{"type": "Point", "coordinates": []float64{a.Lng, a.Lat}}
		if a.Kind == "street" {
			if a.Geom == "" {
				continue
			}
			geometry = json.RawMessage(a.Geom)
		}
		feats = append(feats, map[string]any{
			"type":       "Feature",
			"geometry":   geometry,
			"properties": map[string]any{"id": a.ID, "kind": a.Kind, "name": a.Name, "radius_m": a.RadiusM},
		})
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Cache-Control", "public, max-age=60")
	_ = json.NewEncoder(w).Encode(map[string]any{"type": "FeatureCollection", "features": feats})
}

// handleAreaNamesExport GET /ops-map/area-names/export?format=osm|geojson|waynames
// — الموافَقُ عليه وحدَه، لدفعة بناء البلاطات التالية:
//
//	osm        عقدُ الأحياء (`place=neighbourhood` باسمَي `name` و`name:ar`) لتُدمج بـ`osmium merge`.
//	waynames   `[{way_id,name}]` — أسماءُ الشوارع على طرقٍ موجودةٍ في OSM (لا تُنشأ طرقٌ جديدة).
//	geojson    الاثنان معاً بهندستهما، للمراجعة.
func (s *Server) handleAreaNamesExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "osm" && format != "geojson" && format != "waynames" {
		s.respondErr(w, errAreaFormat)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, kind, name, ST_Y(point::geometry), ST_X(point::geometry), radius_m, way_ids,
		       COALESCE(ST_AsGeoJSON(geom::geometry)::text, '')
		  FROM map_area_names ORDER BY id`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	var all []areaRow
	for rows.Next() {
		var a areaRow
		if err := rows.Scan(&a.ID, &a.Kind, &a.Name, &a.Lat, &a.Lng, &a.RadiusM, &a.WayIDs, &a.Geom); err != nil {
			s.respondErr(w, err)
			return
		}
		all = append(all, a)
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	switch format {
	case "osm":
		// معرّفاتٌ فوق نطاق المباني (٢٠ مليار) والطرق المصدَّرة (٢٥ مليار) فلا تصطدم بالحقيقيّ.
		var b strings.Builder
		b.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<osm version=\"0.6\" generator=\"rahalgo-areas\">\n")
		for _, a := range all {
			if a.Kind != "neighbourhood" {
				continue
			}
			n := html.EscapeString(a.Name)
			fmt.Fprintf(&b, "  <node id=\"%d\" version=\"1\" lat=\"%.7f\" lon=\"%.7f\">\n", 27_000_000_000+a.ID, a.Lat, a.Lng)
			b.WriteString("    <tag k=\"place\" v=\"neighbourhood\"/>\n")
			fmt.Fprintf(&b, "    <tag k=\"name\" v=\"%s\"/>\n    <tag k=\"name:ar\" v=\"%s\"/>\n", n, n)
			b.WriteString("    <tag k=\"source\" v=\"RahalGo customer addresses\"/>\n  </node>\n")
		}
		b.WriteString("</osm>\n")
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-neighbourhoods.osm"`)
		_, _ = w.Write([]byte(b.String()))
	case "waynames":
		out := []map[string]any{}
		for _, a := range all {
			if a.Kind != "street" {
				continue
			}
			for _, wid := range a.WayIDs {
				out = append(out, map[string]any{"way_id": wid, "name": a.Name})
			}
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-way-names.json"`)
		_ = json.NewEncoder(w).Encode(out)
	default:
		feats := []map[string]any{}
		for _, a := range all {
			var g any = map[string]any{"type": "Point", "coordinates": []float64{a.Lng, a.Lat}}
			if a.Kind == "street" && a.Geom != "" {
				g = json.RawMessage(a.Geom)
			}
			feats = append(feats, map[string]any{"type": "Feature", "geometry": g,
				"properties": map[string]any{"id": a.ID, "kind": a.Kind, "name": a.Name, "way_ids": a.WayIDs, "radius_m": a.RadiusM}})
		}
		w.Header().Set("Content-Type", "application/geo+json")
		w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-area-names.geojson"`)
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "FeatureCollection", "features": feats})
	}
}
