package server

import (
	"context"
	"sync"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **موضعُ السائق حيّاً على خريطة العمليات** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت الخريطةُ تسأل كلَّ عشر ثوانٍ** والكاتبُ لا يبثّ — فيقفز السائقُ قفزاً. **والآن
// كلُّ نبضةٍ تُبثّ بحمولتها** على `ops` (لمن يملك `orders.read` وحدَه)، **وتحرّكه
// الخريطةُ بنعومةٍ إلى موضعه الجديد.**
//
// **والحمولةُ عقدٌ لا إشارة** (`hub.go`): معرّفُ السائق وموضعُه واتّجاهُه ووقتُه — **ولا
// اسمَ ولا هاتفَ ولا طلب**: الخريطةُ تعرفها من قائمتها.

// publishDriverLocation **يبثّ موضعاً كُتب للتوّ.**
func (s *Server) publishDriverLocation(driverID string, lat, lng float64, bearing *float64, at time.Time) {
	if s.hub == nil {
		return
	}
	ev := map[string]any{
		"type":      "driver_location",
		"driver_id": driverID,
		"lat":       lat,
		"lng":       lng,
		"at":        at.UTC().Format(time.RFC3339),
	}
	if b := cleanBearing(bearing); b != nil {
		ev["bearing"] = *b
	}
	s.hub.Publish("ops", ev)
}

// trackPrune **آخرُ تقليمٍ لأثر كلِّ سائق** — انظر `trackPruneDue`.
var trackPrune sync.Map

// trackPruneEvery **كم بين تقليمين لأثر السائق الواحد.**
const trackPruneEvery = time.Minute

// trackPruneDue **أحان تقليمُ أثره؟** — مرّةً في الدقيقة لكلّ سائق لا مع كلّ نبضة.
func (s *Server) trackPruneDue(driverID string, now time.Time) bool {
	if v, ok := trackPrune.Load(driverID); ok && now.Sub(v.(time.Time)) < trackPruneEvery {
		return false
	}
	trackPrune.Store(driverID, now)
	return true
}

// ══════════════════════════════════════════════════════════════════════
// **نقاطُ الحركة لرسم الخريطة — بلا هويّة** (قرارُ المالك ٢٠٢٦-١٠-٢٠، هجرة ٠٤٦٦)
// ══════════════════════════════════════════════════════════════════════

const (
	mapTraceMaxAccuracyM = 15.0 // **أدقُّ من خمسة عشر متراً** — وإلّا رسمت حارةً في بيت
	mapTraceMinSpeedMps  = 1.0  // **ماشٍ** — الواقفُ لا يرسم طريقاً
	mapTraceMinMoveM     = 8.0  // **بعيدٌ عن السابقة** — فلا تتكدّس النقاطُ في مكانٍ واحد
	mapTraceKeepDays     = 90
)

// mapTraceWorthy **أتصلح هذه النقطةُ لرسم الخريطة؟** — دالّةٌ صافيةٌ يقيسها فحص.
func mapTraceWorthy(accuracyM, speedMps *float64, mocked bool, movedM float64) bool {
	if mocked || accuracyM == nil || speedMps == nil {
		return false
	}
	return *accuracyM > 0 && *accuracyM <= mapTraceMaxAccuracyM &&
		*speedMps >= mapTraceMinSpeedMps && movedM >= mapTraceMinMoveM
}

var mapTracePrune struct {
	sync.Mutex
	last time.Time
}

// saveMapTrace **يحفظ نقطةً صالحةً بلا سائقٍ ولا ساعة** — وتعثّرُه لا يُسقط حفظَ الموضع.
// **ويُقلَّم ما جاوز تسعين يوماً مرّةً في الساعة** — لا مع كلّ نقطة.
func (s *Server) saveMapTrace(ctx context.Context, lat, lng float64, accuracyM, speedMps, bearing *float64) {
	var b *int
	if c := cleanBearing(bearing); c != nil {
		v := int(*c)
		b = &v
	}
	if _, err := s.pg.Exec(ctx, `
		INSERT INTO map_trace_points (point, accuracy_m, speed_mps, bearing)
		VALUES (ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3, $4, $5)`,
		lng, lat, accuracyM, speedMps, b); err != nil {
		s.logger.Warn("نقاطُ الخريطة: تعذّر الحفظ", "error", err)
		return
	}
	mapTracePrune.Lock()
	due := time.Since(mapTracePrune.last) > time.Hour
	if due {
		mapTracePrune.last = time.Now()
	}
	mapTracePrune.Unlock()
	if due {
		_, _ = s.pg.Exec(ctx, `DELETE FROM map_trace_points WHERE day < (now() AT TIME ZONE 'Asia/Damascus')::date - $1::int`,
			mapTraceKeepDays)
	}
}
