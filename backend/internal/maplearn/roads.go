package maplearn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// NoRoadM **بُعدٌ يُعطى حين لا طريقَ ضمن نطاق البحث** — أكبرُ من أيّ عتبة.
const NoRoadM = 9999.0

// RoadFinder **يقيس بُعدَ نقطةٍ عن أقرب طريقٍ في خريطتنا.**
type RoadFinder interface {
	NearestM(ctx context.Context, lat, lng float64) (float64, error)
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// OSRMRoads يقيس بنداء `nearest` في OSRM — **خريطةُ التوجيه نفسُها**، فالطريقُ الناقصُ فيها ناقصٌ للتوجيه.
type OSRMRoads struct{ Base string }

// NearestM يردّ بُعد النقطة عن أقرب قطعةٍ قابلةٍ للتوجيه.
func (o OSRMRoads) NearestM(ctx context.Context, lat, lng float64) (float64, error) {
	url := fmt.Sprintf("%s/nearest/v1/driving/%.6f,%.6f?number=1", strings.TrimRight(o.Base, "/"), lng, lat)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var body struct {
		Code      string `json:"code"`
		Waypoints []struct {
			Distance float64 `json:"distance"`
		} `json:"waypoints"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	switch body.Code {
	case "Ok":
		if len(body.Waypoints) == 0 {
			return NoRoadM, nil
		}
		return body.Waypoints[0].Distance, nil
	case "NoSegment":
		return NoRoadM, nil
	default:
		return 0, fmt.Errorf("osrm nearest: %s", body.Code)
	}
}

// ValhallaRoads يقيس بنداء `locate` في Valhalla.
type ValhallaRoads struct{ Base string }

// NearestM يردّ بُعد النقطة عن أقرب حافّةٍ (بتكلفة الدراجة الناريّة كالتوجيه).
func (v ValhallaRoads) NearestM(ctx context.Context, lat, lng float64) (float64, error) {
	payload, _ := json.Marshal(map[string]any{
		"locations": []map[string]float64{{"lat": lat, "lon": lng}},
		"costing":   "motorcycle",
	})
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost,
		strings.TrimRight(v.Base, "/")+"/locate", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 500 {
		return 0, fmt.Errorf("valhalla locate: %d", resp.StatusCode)
	}
	var body []struct {
		Edges []struct {
			Lat float64 `json:"correlated_lat"`
			Lon float64 `json:"correlated_lon"`
		} `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, err
	}
	if len(body) == 0 || len(body[0].Edges) == 0 {
		return NoRoadM, nil
	}
	best := NoRoadM
	for _, e := range body[0].Edges {
		if d := HaversineM(lat, lng, e.Lat, e.Lon); d < best {
			best = d
		}
	}
	return best, nil
}

// HaversineM المسافةُ بين نقطتين بالمتر.
func HaversineM(lat1, lng1, lat2, lng2 float64) float64 {
	const r = 6371000.0
	p1, p2 := lat1*math.Pi/180, lat2*math.Pi/180
	dp, dl := p2-p1, (lng2-lng1)*math.Pi/180
	a := math.Sin(dp/2)*math.Sin(dp/2) + math.Cos(p1)*math.Cos(p2)*math.Sin(dl/2)*math.Sin(dl/2)
	return 2 * r * math.Asin(math.Min(1, math.Sqrt(a)))
}

// NewRoadFinder **يختار المقياس بمحرّك التوجيه المُعدّ** — Valhalla إن كان محرّكاً وله عنوان، وإلا OSRM.
// وغيابُ العنوانين يردّ nil: لا قياسَ بلا خريطةٍ نقيس عليها.
func NewRoadFinder(engine, osrmURL, valhallaURL string) RoadFinder {
	if engine == "valhalla" && valhallaURL != "" {
		return ValhallaRoads{Base: valhallaURL}
	}
	if osrmURL != "" {
		return OSRMRoads{Base: osrmURL}
	}
	if valhallaURL != "" {
		return ValhallaRoads{Base: valhallaURL}
	}
	return nil
}
