package routing

// ══════════════════════════════════════════════════════════════════════
// **Valhalla — محرّكٌ بديلٌ يُختار بالإعداد**
// ══════════════════════════════════════════════════════════════════════
//
// طلبُ المالك ٢٠٢٦-١٠-٠٢: الموتور يدخل كلّ الطرق مو سيارة.
//
// **وملفُّ OSRM ملفُّ سيّارة**: يتجنّب الأزقّةَ والطرقَ الخدميّة. أمّا
// تكلفةُ `motor_scooter` في Valhalla فتسلكها كما يسلكها الموتور.
//
// **ويُطلب الردُّ بصيغة OSRM** (`format=osrm`) فيُقرأ بالقارئ نفسِه
// (`parseOSRMResponse`) — **فالمناوراتُ والهندسةُ لا تعرفان أيَّ محرّكٍ
// ردّ.** ولا عقدَ ولا ارتباطَ هنا: تلك ميزاتُ OSRM وتبقى عليه.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Valhalla بابُ محرّك Valhalla — **وفارغُ العنوان يعني «لا محرّك».**
type Valhalla struct {
	base string
	http *http.Client
	snap SnapPolicy
}

var _ Backend = (*Valhalla)(nil)

// NewValhalla بحدِّ الالتقاط نفسِه الذي يحمي OSRM — TD-SNAP-RADIUS.
func NewValhalla(baseURL string) *Valhalla {
	return &Valhalla{
		base: strings.TrimRight(baseURL, "/"),
		http: &http.Client{Timeout: 4 * time.Second},
		snap: DefaultSnapPolicy(),
	}
}

// Enabled أثمّة محرّكٌ مضبوط؟
func (v *Valhalla) Enabled() bool { return v != nil && v.base != "" }

// Route مسارٌ واحدٌ موصًى به.
func (v *Valhalla) Route(ctx context.Context, from, to Point) (*Route, error) {
	res, err := v.fetch(ctx, from, to, 0)
	if err != nil {
		return nil, err
	}
	return res[0], nil
}

// RouteSet الموصى به وبدائلُه — **وValhalla لا يردّ بدائلَ إلّا بين
// نقطتين**، وهي حالُنا دائماً.
func (v *Valhalla) RouteSet(ctx context.Context, from, to Point) ([]*Route, error) {
	return v.fetch(ctx, from, to, EngineMaxAlternatives)
}

// ══════════════════════════════════════════════════════════════════════
// **تكلفةُ الموتور — أرقامُ المالك**
// ══════════════════════════════════════════════════════════════════════
//
// `service_penalty: 0` و`service_factor: 1` — **الطريقُ الخدميُّ طريقٌ
// عاديٌّ للموتور**، وعليها أكثرُ العناوين داخلَ الأحياء.
var motorScooterCosting = map[string]any{
	"top_speed":          50,
	"use_primary":        0.5,
	"use_living_streets": 0.6,
	"use_tracks":         0.5,
	"service_penalty":    0,
	"service_factor":     1.0,
}

type valhallaLocation struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
	// SearchCutoff **حدُّ الالتقاط بالمتر** — مقابلُ `radiuses` في OSRM؛
	// وبلاه يلتقط Valhalla على بعد ٣٥ كم (قِيس ٢٠٢٦-١٠-٠٢).
	SearchCutoff     float64 `json:"search_cutoff,omitempty"`
	Heading          *int    `json:"heading,omitempty"`
	HeadingTolerance int     `json:"heading_tolerance,omitempty"`
}

type valhallaRequest struct {
	Locations         []valhallaLocation `json:"locations"`
	Costing           string             `json:"costing"`
	CostingOptions    map[string]any     `json:"costing_options"`
	DirectionsOptions map[string]any     `json:"directions_options"`
	Format            string             `json:"format"`
	// ShapeFormat `geojson` — **فقارئُ OSRM ينتظر إحداثيّاتٍ لا خطّاً مرمَّزاً.**
	ShapeFormat string `json:"shape_format"`
	Alternates  int    `json:"alternates,omitempty"`
}

func (v *Valhalla) buildRequest(from, to Point, alternatives int) valhallaRequest {
	origin := valhallaLocation{Lat: from.Lat, Lon: from.Lng, SearchCutoff: v.snap.OriginM}
	if bearingValue(from.Bearing) != "" {
		h := normBearing(*from.Bearing)
		origin.Heading = &h
		origin.HeadingTolerance = 60
	}
	dest := valhallaLocation{Lat: to.Lat, Lon: to.Lng, SearchCutoff: v.snap.DestinationM}
	alt := 0
	if alternatives > 1 {
		// **زيادةً على الموصى به** — كما يعدّها OSRM (الذهبيّةُ فيها ٤ مسارات).
		alt = alternatives
	}
	return valhallaRequest{
		Locations:         []valhallaLocation{origin, dest},
		Costing:           "motor_scooter",
		CostingOptions:    map[string]any{"motor_scooter": motorScooterCosting},
		DirectionsOptions: map[string]any{"units": "kilometers"},
		Format:            "osrm",
		ShapeFormat:       "geojson",
		Alternates:        alt,
	}
}

func (v *Valhalla) fetch(ctx context.Context, from, to Point, alternatives int) ([]*Route, error) {
	if !v.Enabled() {
		return nil, ErrNoEngine
	}
	out, err := v.request(ctx, from, to, alternatives)
	if retryWithoutBearing(err, from) {
		from.Bearing = nil
		out, err = v.request(ctx, from, to, alternatives)
	}
	return out, err
}

func (v *Valhalla) request(ctx context.Context, from, to Point, alternatives int) ([]*Route, error) {
	raw, err := json.Marshal(v.buildRequest(from, to, alternatives))
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, v.base+"/route", bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := v.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	// **والرفضُ بـ٤٠٠ وفيه السبب** — كما في OSRM؛ والخمسمئيّةُ عطب.
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusBadRequest {
		return nil, fmt.Errorf("routing: valhalla ردّ %d", res.StatusCode)
	}
	return parseOSRMResponse(res.Body)
}

// valhallaError **رموزُ Valhalla الأصليّة إلى أخطائنا المسمّاة.**
//
// ١٧١ «لا طريقَ مناسبةً قربَ الموضع» = `ErrNoSegment`؛ و٤٤٢/٤٤٣ «لا
// سبيل» = `ErrNoRoute`. **وصفرٌ يعني لا رمز.**
func valhallaError(code int) error {
	switch code {
	case 0:
		return nil
	case 170, 171:
		return ErrNoSegment
	case 442, 443:
		return ErrNoRoute
	}
	return fmt.Errorf("routing: valhalla error_code %d", code)
}
