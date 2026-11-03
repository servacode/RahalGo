package maplearn

import (
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// **معرّفاتُ اتّجاه الطريق** — نطاقٌ ثالثٌ لا يصطدم بالشوارع المقترحة ولا بالمباني.
const (
	onewayNodeStart int64 = 26_000_000_000
	onewayWayStart  int64 = 17_000_000_000
)

// OnewayLine طريقٌ موافَقٌ على أنّه باتّجاه واحد. `Coords` بترتيب الاتّجاه المسموح (lng,lat).
type OnewayLine struct {
	ID     int64
	WayID  int64
	Name   string
	Coords [][2]float64
}

// OnewayOSM **ملفُّ OSM** بالحافّات الموافَق عليها اتّجاهاً واحداً: كلُّ طريقٍ بهندسته ووسمِ `oneway=yes`
// ورقمِ طريق OSM الأصليّ في `rahalgo:osm_way_id` ليطابقه بناءُ الخريطة. **لا `highway` هنا** — هذا تعديلٌ لا طريقٌ جديد.
func OnewayOSM(lines []OnewayLine) string {
	var nodes, ways strings.Builder
	nid, wid := onewayNodeStart, onewayWayStart
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
		ways.WriteString("    <tag k=\"oneway\" v=\"yes\"/>\n")
		if l.Name != "" {
			fmt.Fprintf(&ways, "    <tag k=\"name\" v=\"%s\"/>\n", html.EscapeString(l.Name))
		}
		if l.WayID > 0 {
			fmt.Fprintf(&ways, "    <tag k=\"rahalgo:osm_way_id\" v=\"%d\"/>\n", l.WayID)
		}
		ways.WriteString("    <tag k=\"source\" v=\"RahalGo driver traces\"/>\n")
		fmt.Fprintf(&ways, "    <tag k=\"rahalgo:candidate_id\" v=\"%d\"/>\n", l.ID)
		ways.WriteString("  </way>\n")
	}
	return "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<osm version=\"0.6\" generator=\"rahalgo-maplearn\">\n" +
		nodes.String() + ways.String() + "</osm>\n"
}

// OnewayGeoJSON **قائمةٌ** بنفس المعلومات: هندسةٌ بترتيب الاتّجاه المسموح وخصائصُ `oneway`.
func OnewayGeoJSON(lines []OnewayLine) ([]byte, error) {
	feats := make([]map[string]any, 0, len(lines))
	for _, l := range lines {
		feats = append(feats, map[string]any{
			"type":     "Feature",
			"geometry": map[string]any{"type": "LineString", "coordinates": l.Coords},
			"properties": map[string]any{
				"id": l.ID, "name": l.Name, "oneway": "yes", "osm_way_id": l.WayID,
			},
		})
	}
	return json.Marshal(map[string]any{"type": "FeatureCollection", "features": feats})
}
