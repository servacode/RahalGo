package maplearn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/polyline"
)

// ValhallaLocator يلصق النقاطَ بأقرب حافّةٍ بنداء `/locate` (verbose) بتكلفة الدراجة الناريّة كالتوجيه.
type ValhallaLocator struct{ Base string }

// NewLocator يردّ لاصقاً من عنوان Valhalla، أو nil إن غاب العنوان.
func NewLocator(valhallaURL string) Locator {
	if valhallaURL == "" {
		return nil
	}
	return ValhallaLocator{Base: valhallaURL}
}

const locateBatch = 40

type vhLocateEdge struct {
	WayID    int64    `json:"way_id"`
	CorrLat  *float64 `json:"correlated_lat"`
	CorrLon  *float64 `json:"correlated_lon"`
	EdgeInfo struct {
		WayID int64           `json:"way_id"`
		Names []string        `json:"names"`
		Shape json.RawMessage `json:"shape"`
	} `json:"edge_info"`
}

type vhLocation struct {
	InputLat float64        `json:"input_lat"`
	InputLon float64        `json:"input_lon"`
	Edges    []vhLocateEdge `json:"edges"`
}

// Locate يلصق النقاط (lat,lng) دفعاتٍ. ردٌّ 4xx لدفعةٍ = لا طريقَ لنقاطها؛ و5xx أو انقطاعٌ = خطأ.
func (v ValhallaLocator) Locate(ctx context.Context, pts [][2]float64) ([]LocatedEdge, error) {
	out := make([]LocatedEdge, len(pts))
	for start := 0; start < len(pts); start += locateBatch {
		end := start + locateBatch
		if end > len(pts) {
			end = len(pts)
		}
		locs := make([]map[string]float64, 0, end-start)
		for _, p := range pts[start:end] {
			locs = append(locs, map[string]float64{"lat": p[0], "lon": p[1]})
		}
		payload, _ := json.Marshal(map[string]any{"locations": locs, "costing": "motorcycle", "verbose": true})
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(v.Base, "/")+"/locate",
			bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 500 {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("valhalla locate: %d", resp.StatusCode)
		}
		if resp.StatusCode >= 400 {
			_ = resp.Body.Close()
			continue
		}
		var body []vhLocation
		err = json.NewDecoder(resp.Body).Decode(&body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for i := range body {
			if start+i >= end {
				break
			}
			out[start+i] = nearestEdge(body[i], pts[start+i])
		}
	}
	return out, nil
}

// nearestEdge أقربُ حافّةٍ ملتصقةٍ بالنقطة (الحافّتان باتجاهَين تتشاركان المسافة؛ تُؤخذ الأولى).
func nearestEdge(loc vhLocation, p [2]float64) LocatedEdge {
	var best LocatedEdge
	for _, e := range loc.Edges {
		way := e.WayID
		if way == 0 {
			way = e.EdgeInfo.WayID
		}
		if way == 0 || e.CorrLat == nil || e.CorrLon == nil {
			continue
		}
		d := distM(p[0], p[1], *e.CorrLat, *e.CorrLon)
		if best.WayID == 0 || d < best.DistM {
			best = LocatedEdge{WayID: way, Names: e.EdgeInfo.Names, DistM: d, Line: parseShape(e.EdgeInfo.Shape)}
		}
	}
	return best
}

// parseShape يقبل خطّاً مشفَّراً (polyline6) أو مصفوفة {lat,lon}.
func parseShape(raw json.RawMessage) [][2]float64 {
	if len(raw) == 0 {
		return nil
	}
	var enc string
	if json.Unmarshal(raw, &enc) == nil {
		return polyline.Decode6(enc)
	}
	var arr []struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	if json.Unmarshal(raw, &arr) != nil {
		return nil
	}
	out := make([][2]float64, len(arr))
	for i, a := range arr {
		out[i] = [2]float64{a.Lon, a.Lat}
	}
	return out
}
