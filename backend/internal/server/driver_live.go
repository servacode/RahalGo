package server

import (
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
