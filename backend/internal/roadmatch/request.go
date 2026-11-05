package roadmatch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// قيم الطلب (B3) — **بدايات**، بتتعدّل بعد تقييم التجهيز (§5.2) مع رفع `ParamsVersion`.
const (
	// Costing نفس مركبة الملاحة (`routing/valhalla.go`).
	Costing = "motor_scooter"
	// SearchRadiusM نصف قطر البحث العام.
	SearchRadiusM = 50
	// InterpolationM نقاط ضمن هالمسافة من اللي قبلها بتنحسب بلا مطابقة (رجفة تحت ١٠ م بتختفي لحالها).
	InterpolationM = 10
	// TurnPenalty بيخفّف نتوءات اللفّة من الرجفة.
	TurnPenalty = 100
	// CallTimeout حارس النداء الواحد — **مو هدف**: علامة النجاح ≤ ١٫٥ ث. والمهلة بتقسم القطعة، ما بتعيدها.
	CallTimeout = 3 * time.Second
	// MinSplitSent أصغر قطعة بتنقسم عند المهلة؛ أصغر منها بتنجرّب مرّتين وبعدين `no_match('timeout')`.
	MinSplitSent = 30
)

// Attributes الحقول المطلوبة من الجواب.
var Attributes = []string{
	"shape",
	"matched.point", "matched.type", "matched.edge_index", "matched.distance_along_edge",
	"matched.distance_from_trace_point", "matched.begin_route_discontinuity", "matched.end_route_discontinuity",
	"edge.length", "edge.begin_shape_index", "edge.end_shape_index",
}

// pointRadius نصف قطر البحث لنقطة: ضعف دقتها بين ٢٥ و١٠٠ م (أقصى الخدمة)، و٥٠ إذا مجهولة.
func pointRadius(acc *float64) int {
	if acc == nil {
		return 50
	}
	return int(math.Round(math.Max(25, math.Min(100, 2**acc))))
}

// gpsAccuracy وسيط دقّة النقاط المبعوتة بين ٥ و٣٠ م.
func gpsAccuracy(pts []drivermonitor.Point) int {
	var acc []float64
	for _, p := range pts {
		if p.AccuracyM != nil {
			acc = append(acc, *p.AccuracyM)
		}
	}
	if len(acc) == 0 {
		return 10
	}
	sort.Float64s(acc)
	m := acc[len(acc)/2]
	if len(acc)%2 == 0 {
		m = (acc[len(acc)/2-1] + acc[len(acc)/2]) / 2
	}
	return int(math.Round(math.Max(5, math.Min(30, m))))
}

// Request **جسم نداء `trace_attributes`** للنقاط المبعوتة.
//
// الاتجاه الواحد محترَم (`ignore_oneways` لأ — قرار المالك ٢٠٢٦-١٠-٠٢: كل جهة من الشارع المقسوم
// طريق باتجاه واحد) والطرق الممنوعة كمان (`ignore_access` لأ، وإلا صارت الأرصفة مرشّحة). المنعطفات
// الممنوعة والطرق المسكّرة متجاهَلة: الدرّاج بيلفّ وين ممنوع. **وبلا أي تفضيل توجيه** — هاد طريقه هو
// مو طريقنا. و`breakage_distance` مو منها: المحرّك ما بيقبل تغييرها، فالتقطيع بيقطع عند ٢ كم لحاله.
func Request(sent []drivermonitor.Point) []byte {
	type loc struct {
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		Time   int64   `json:"time"`
		Radius int     `json:"radius"`
	}
	shape := make([]loc, len(sent))
	for i, p := range sent {
		shape[i] = loc{Lat: p.Lat, Lon: p.Lng, Time: p.At.Unix(), Radius: pointRadius(p.AccuracyM)}
	}
	body := map[string]any{
		"shape":   shape,
		"costing": Costing,
		"costing_options": map[string]any{
			Costing: map[string]any{
				"ignore_oneways":      false,
				"ignore_access":       false,
				"ignore_restrictions": true,
				"ignore_closures":     true,
			},
		},
		"shape_match":    "map_snap",
		"use_timestamps": true,
		"trace_options": map[string]any{
			"search_radius":          SearchRadiusM,
			"gps_accuracy":           gpsAccuracy(sent),
			"interpolation_distance": InterpolationM,
			"turn_penalty_factor":    TurnPenalty,
		},
		"filters": map[string]any{"action": "include", "attributes": Attributes},
	}
	b, _ := json.Marshal(body)
	return b
}

// MatchedPoint نقطة من الجواب.
type MatchedPoint struct {
	Lat                     float64  `json:"lat"`
	Lon                     float64  `json:"lon"`
	Type                    string   `json:"type"`
	EdgeIndex               *int64   `json:"edge_index"`
	DistanceAlongEdge       *float64 `json:"distance_along_edge"`
	DistanceFromTracePoint  *float64 `json:"distance_from_trace_point"`
	BeginRouteDiscontinuity bool     `json:"begin_route_discontinuity"`
	EndRouteDiscontinuity   bool     `json:"end_route_discontinuity"`
}

// Edge حافّة من الجواب.
type Edge struct {
	Length          float64 `json:"length"` // كم
	BeginShapeIndex int     `json:"begin_shape_index"`
	EndShapeIndex   int     `json:"end_shape_index"`
}

// Answer جواب `trace_attributes`.
type Answer struct {
	Shape         string         `json:"shape"`
	Edges         []Edge         `json:"edges"`
	MatchedPoints []MatchedPoint `json:"matched_points"`
}

// أصناف الأخطاء (B3).
var (
	// ErrDown المحرّك ما ردّ (5xx أو انقطاع) ⇒ ما بيتخزّن شي، وبتنعاد.
	ErrDown = errors.New("roadmatch: محرّك المطابقة ما ردّ")
	// ErrTimeout خلصت مهلة النداء ⇒ القطعة بتنقسم.
	ErrTimeout = errors.New("roadmatch: خلصت مهلة النداء")
)

// NoMatchError 4xx: ما في مطابقة لهالقطعة ⇒ بتتخزّن `no_match` برمز Valhalla.
type NoMatchError struct{ Code string }

func (e *NoMatchError) Error() string { return "roadmatch: لا مطابقة (" + e.Code + ")" }

// Matcher بيطابق نقاط مبعوتة. (`Client` الحقيقي، وبالاختبارات مزيَّف.)
type Matcher interface {
	Match(ctx context.Context, sent []drivermonitor.Point) (Answer, error)
}

// Client عميل Valhalla — **كل نداء بيمرق من البوّابة** (`gate.go`).
//
// **ونصّ الخطأ قصير وبلا عناوين** (رمز الحالة أو «بلا اتصال»): بيطلع بصفحة صحّة التشغيل.
type Client struct {
	Base    string
	HTTP    *http.Client
	Timeout time.Duration
}

// NewClient عميل من عنوان Valhalla، أو nil إذا فاضي.
func NewClient(base string) *Client {
	if strings.TrimSpace(base) == "" {
		return nil
	}
	return &Client{Base: strings.TrimRight(base, "/"), HTTP: &http.Client{}, Timeout: CallTimeout}
}

// Match نداء واحد عبر البوّابة بمهلة `CallTimeout` (المهلة بتبلّش بعد البوّابة).
func (c *Client) Match(ctx context.Context, sent []drivermonitor.Point) (Answer, error) {
	release, err := Acquire(ctx)
	if err != nil {
		return Answer{}, err
	}
	defer release()
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = CallTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodPost, c.Base+"/trace_attributes", bytes.NewReader(Request(sent)))
	req.Header.Set("Content-Type", "application/json")
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		if errors.Is(cctx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return Answer{}, ErrTimeout
		}
		if ctx.Err() != nil {
			return Answer{}, ctx.Err()
		}
		return Answer{}, fmt.Errorf("%w: بلا اتصال", ErrDown)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 16<<20))
	if err != nil {
		if errors.Is(cctx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return Answer{}, ErrTimeout
		}
		return Answer{}, fmt.Errorf("%w: انقطع الجواب", ErrDown)
	}
	switch {
	case resp.StatusCode >= 500:
		return Answer{}, fmt.Errorf("%w: %d", ErrDown, resp.StatusCode)
	case resp.StatusCode >= 400:
		var e struct {
			Code json.Number `json:"error_code"`
		}
		_ = json.Unmarshal(body, &e)
		code := e.Code.String()
		if code == "" {
			code = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		return Answer{}, &NoMatchError{Code: code}
	}
	var a Answer
	if err := json.Unmarshal(body, &a); err != nil {
		return Answer{}, fmt.Errorf("%w: جواب ما بينقرأ", ErrDown)
	}
	return a, nil
}

// TilesTime وقت بناء البلاطات من `/status` إذا المحرّك بيذكره (`tileset_last_modified`)، وإلا صفر.
func (c *Client) TilesTime(ctx context.Context) time.Time {
	cctx, cancel := context.WithTimeout(ctx, CallTimeout)
	defer cancel()
	req, _ := http.NewRequestWithContext(cctx, http.MethodGet, c.Base+"/status", nil)
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return time.Time{}
	}
	defer resp.Body.Close()
	var s struct {
		Tiles int64 `json:"tileset_last_modified"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(resp.Body).Decode(&s) != nil || s.Tiles <= 0 {
		return time.Time{}
	}
	return time.Unix(s.Tiles, 0)
}
