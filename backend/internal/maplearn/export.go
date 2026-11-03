package maplearn

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// **معرّفاتٌ موجبةٌ فوق الحقيقيّ وخارجَ نطاق المباني** — القواعدُ في `maps/scripts/buildings-to-osm.py`:
// المباني عقدُها من ٢٠ مليار وطرقُها من ١٥ مليار. فهنا العقدُ من ٢٥ ملياراً والطرقُ من ١٦ ملياراً —
// لا تصطدمان بالحقيقيّ ولا بالمباني، وPlanetiler يرفض السالب.
const (
	exportNodeStart int64 = 25_000_000_000
	exportWayStart  int64 = 16_000_000_000
)

// Line خطٌّ موافَقٌ عليه للتصدير.
type Line struct {
	ID     int64
	Name   string
	Coords [][2]float64 // (lng, lat)
}

// OSM **ملفُّ OSM XML** لخطوطٍ موافَقٍ عليها: عقدٌ ثمّ طرق (الترتيبُ الذي يشترطه `osmium merge`).
// الوسمُ `highway=residential` — يمرّ بتوجيه الدراجة الناريّة ويظهر في البلاطات.
func OSM(lines []Line) string {
	var nodes, ways strings.Builder
	nid := exportNodeStart
	wid := exportWayStart
	for _, l := range lines {
		if len(l.Coords) < 2 {
			continue
		}
		wid++
		var refs []int64
		for _, c := range l.Coords {
			nid++
			refs = append(refs, nid)
			fmt.Fprintf(&nodes, "  <node id=\"%d\" version=\"1\" lat=\"%.7f\" lon=\"%.7f\"/>\n", nid, c[1], c[0])
		}
		fmt.Fprintf(&ways, "  <way id=\"%d\" version=\"1\">\n", wid)
		for _, r := range refs {
			fmt.Fprintf(&ways, "    <nd ref=\"%d\"/>\n", r)
		}
		ways.WriteString("    <tag k=\"highway\" v=\"residential\"/>\n")
		if l.Name != "" {
			fmt.Fprintf(&ways, "    <tag k=\"name\" v=\"%s\"/>\n", html.EscapeString(l.Name))
		}
		ways.WriteString("    <tag k=\"source\" v=\"RahalGo driver traces\"/>\n")
		fmt.Fprintf(&ways, "    <tag k=\"rahalgo:candidate_id\" v=\"%d\"/>\n", l.ID)
		ways.WriteString("  </way>\n")
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<osm version=\"0.6\" generator=\"rahalgo-maplearn\">\n" +
		nodes.String() + ways.String() + "</osm>\n"
}

// GeoJSON **FeatureCollection** لخطوطٍ موافَقٍ عليها.
func GeoJSON(lines []Line) ([]byte, error) {
	feats := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		feats = append(feats, map[string]any{
			"type":       "Feature",
			"geometry":   map[string]any{"type": "LineString", "coordinates": l.Coords},
			"properties": map[string]any{"id": l.ID, "name": l.Name, "highway": "residential"},
		})
	}
	return json.Marshal(map[string]any{"type": "FeatureCollection", "features": feats})
}
