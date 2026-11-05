package maplearn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// ValhallaMatcher يطابق الأثرَ بنداء `trace_attributes` (map_snap) بتكلفة الدراجة الناريّة كالتوجيه.
// **Valhalla وحدَه**: OSRM لا يردّ معرّفَ الطريق ولا اتّجاهَ الحافّة، وخريطةُ الرقّة عليه أصلاً.
type ValhallaMatcher struct{ Base string }

// NewMatcher يردّ مطابقاً من عنوان Valhalla، أو nil إن غاب العنوان.
func NewMatcher(valhallaURL string) Matcher {
	if valhallaURL == "" {
		return nil
	}
	return ValhallaMatcher{Base: valhallaURL}
}

type vhTrace struct {
	Edges []struct {
		WayID          int64    `json:"way_id"`
		Names          []string `json:"names"`
		Length         float64  `json:"length"` // كم
		BeginShape     int      `json:"begin_shape_index"`
		EndShape       int      `json:"end_shape_index"`
		Traversability string   `json:"traversability"`
	} `json:"edges"`
	Shape         string `json:"shape"`
	MatchedPoints []struct {
		Type      string `json:"type"`
		EdgeIndex *int   `json:"edge_index"`
	} `json:"matched_points"`
}

// Match يطابق المقطع. ردٌّ 4xx (لا حافّاتٍ مناسبة) = ErrNoMatch؛ و5xx أو انقطاعٌ = خطأ.
func (v ValhallaMatcher) Match(ctx context.Context, pts []drivermonitor.Point) (MatchResult, error) {
	shape := make([]map[string]float64, len(pts))
	for i, p := range pts {
		shape[i] = map[string]float64{"lat": p.Lat, "lon": p.Lng}
	}
	payload, _ := json.Marshal(map[string]any{
		"shape":       shape,
		"costing":     "motorcycle",
		"shape_match": "map_snap",
		"filters": map[string]any{
			"action": "include",
			"attributes": []string{
				"edge.way_id", "edge.names", "edge.length", "edge.begin_shape_index",
				"edge.end_shape_index", "edge.traversability", "shape", "matched.edge_index", "matched.type",
			},
		},
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(v.Base, "/")+"/trace_attributes", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return MatchResult{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return MatchResult{}, fmt.Errorf("valhalla trace_attributes: %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return MatchResult{}, ErrNoMatch
	}
	var body vhTrace
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return MatchResult{}, err
	}
	return body.toResult(len(pts)), nil
}

func (b vhTrace) toResult(nPts int) MatchResult {
	shape := polyline.Decode6(b.Shape)
	res := MatchResult{Edges: make([]MatchedEdge, len(b.Edges)), PointEdge: make([]int, nPts)}
	for i, e := range b.Edges {
		if e.BeginShape < 0 || e.EndShape >= len(shape) || e.EndShape <= e.BeginShape {
			continue // حافّةٌ بلا هندسة — مفتاحُها فارغٌ فتُهمل
		}
		travel := shape[e.BeginShape : e.EndShape+1]
		key, dir, canon := CanonEdge(e.WayID, travel)
		name := ""
		if len(e.Names) > 0 {
			name = e.Names[0]
		}
		res.Edges[i] = MatchedEdge{
			Key: key, WayID: e.WayID, Name: name, Length: e.Length * 1000, Dir: dir,
			OneWayKnown: e.Traversability != "" && e.Traversability != "both",
			Line:        canon,
		}
	}
	for i := range res.PointEdge {
		res.PointEdge[i] = -1
		if i < len(b.MatchedPoints) {
			mp := b.MatchedPoints[i]
			if mp.EdgeIndex != nil && mp.Type != "unmatched" && *mp.EdgeIndex < len(b.Edges) {
				res.PointEdge[i] = *mp.EdgeIndex
			}
		}
	}
	return res
}
