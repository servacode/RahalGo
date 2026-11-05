package server

import (
	"context"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/opsmap"
	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// ══════════════════════════════════════════════════════════════════════
// **موضعُ السائق حيّاً على خريطة العمليات** (قرارُ المالك ٢٠٢٦-١٠-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت الخريطةُ تسأل كلَّ عشر ثوانٍ** والكاتبُ لا يبثّ — فيقفز السائقُ قفزاً. **والآن
// كلُّ نبضةٍ تُبثّ بحمولتها** على `ops:drivers` (لمن يملك `drivers.read` وحدَه — ٢٠٢٦-١١-٠٥،
// وكانت على `ops` فتصل كلَّ من يقرأ الطلبات)، **وتحرّكه الخريطةُ بنعومةٍ إلى موضعه الجديد.**
//
// **والحمولةُ عقدٌ لا إشارة** (`hub.go`): معرّفُ السائق وموضعُه واتّجاهُه ووقتُه — **ولا
// اسمَ ولا هاتفَ ولا طلب**: الخريطةُ تعرفها من قائمتها.

// publishDriverLocation **يبثّ موضعاً كُتب للتوّ** — نقطةَ العرض (`drivermonitor.NextShown`) حين
// تكون الحركةُ الناعمةُ مفعّلة، والنقطةَ الخامّةَ حين تكون مطفأة.
//
// **و`at` آخرُ ما سُمع منه لا وقتُ الموضع** — فنقطةٌ ممسوكةٌ (دقّةٌ ضعيفة، رجفةُ واقف) تُبثّ
// بموضعها القديم ووقتٍ جديد: **يبقى «حيّاً» على الخريطة ولا يتحرّك.** وبدقّة النانوثانية
// (٢٠٢٦-١١-٠٥) — الخريطةُ تحسب منه كم بين نبضةٍ وأخرى لكلّ سائق.
//
// **والحقولُ الجديدةُ إضافةٌ لا تبديل**: `speed_mps` و`accuracy_m` حين يقولها الجهاز، و`rough`
// إن كان الموضعُ من نقطةٍ دقّتُها ضعيفة.
func (s *Server) publishDriverLocation(driverID string, p drivermonitor.Shown) {
	if s.hub == nil {
		return
	}
	ev := map[string]any{
		"type":      "driver_location",
		"driver_id": driverID,
		"lat":       p.Lat,
		"lng":       p.Lng,
		"at":        p.At.UTC().Format(time.RFC3339Nano),
		"rough":     p.Rough,
	}
	if b := cleanBearing(p.BearingDeg); b != nil {
		ev["bearing"] = *b
	}
	if p.SpeedMps != nil {
		ev["speed_mps"] = *p.SpeedMps
	}
	if p.AccuracyM != nil {
		ev["accuracy_m"] = *p.AccuracyM
	}
	// **وعلى غرفة من يرى السائقين لا غرفة من يقرأ الطلبات** (`realtime.TopicOpsDrivers`).
	s.hub.Publish(realtime.TopicOpsDrivers, ev)
}

// ══════════════════════════════════════════════════════════════════════
// **نقطةُ العرض لكلّ سائق — في الذاكرة** (L1b، ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **عمليّةٌ واحدةٌ للمحرّك** — والبثُّ نفسُه يفترضها (`realtime/hub.go`). **وبعد إعادة التشغيل
// تُعرض أوّلُ نقطةٍ كما هي** ثمّ يعود الحكم. **ولا يُكتب شيءٌ في القاعدة**: الموضعُ الخامُّ يبقى
// في `users.last_location` والأثرَين، والتوزيعُ يقرأ الخامّ.

type shownEntry struct {
	mu sync.Mutex
	s  drivermonitor.Shown
}

var driverShown sync.Map // driverID → *shownEntry

// nextShown **يمرّر نقطةً أحدثَ على نقطة عرض السائق** ويردّ الجديدة.
func nextShown(driverID string, fix drivermonitor.Point) drivermonitor.Shown {
	v, _ := driverShown.LoadOrStore(driverID, &shownEntry{})
	e := v.(*shownEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	e.s = drivermonitor.NextShown(e.s, fix, drivermonitor.DefaultConfig())
	return e.s
}

// shownOf **نقطةُ عرض السائق الآن** — `false` إن لم تُعرف بعد.
func shownOf(driverID string) (drivermonitor.Shown, bool) {
	v, ok := driverShown.Load(driverID)
	if !ok {
		return drivermonitor.Shown{}, false
	}
	e := v.(*shownEntry)
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.s, e.s.Known()
}

// overlayShown **يضع نقطةَ العرض فوق الموضع الخامّ** في طبقة السائقين — **إن كانت هي الأحدث**:
// دفعةٌ أفرغها طابورٌ بعد انقطاع (لا تمرّ بنقطة العرض) أو إعادةُ تشغيلٍ تتركان الخامَّ أحدث،
// **فيُعرض الخامّ** ولا يُرجَع السائقُ إلى نقطةٍ أقدم.
func overlayShown(drivers []opsmap.Driver) {
	for i := range drivers {
		d := &drivers[i]
		if d.Freshness == opsmap.FreshnessNone {
			continue
		}
		p, ok := shownOf(d.ID)
		if !ok || p.At.Before(d.LastLocationAt) {
			continue
		}
		d.Lat, d.Lng = p.Lat, p.Lng
		d.Bearing = cleanBearing(p.BearingDeg)
		d.SpeedMps = p.SpeedMps
		d.Rough = p.Rough
	}
}

// ══════════════════════════════════════════════════════════════════════
// **ساعةُ الهاتف المتأخّرة — تُقاس قبل أن تُصلَح** (J7، ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **نقطةٌ هي الأحدثُ له وأقدمُ من دقيقتين بساعة الجهاز لا تُبثّ** (`orders.LiveFreshFor`) —
// فلا تحرّكه الخريطةُ إلّا بالمدّة. **ولا يُعرف كم يقع**: فسطرٌ واحدٌ في السجلّ لكلّ سائقٍ في
// الساعة، بعددِ ما وقع في الساعة التي قبلها، **ويُصلَح إن ظهر** (تصحيحُ ساعة الهاتف في نسخةٍ قادمة).

type skewWindow struct {
	mu    sync.Mutex
	start time.Time
	n     int
}

var clockSkew sync.Map // driverID → *skewWindow

// noteClockSkew **يعدّ نقطةً أحدثَ له وأقدمَ من دقيقتين** — ويقول «اكتب سطراً» مرّةً في الساعة،
// مع عدد الساعة السابقة.
func noteClockSkew(driverID string, now time.Time) (logNow bool, prevCount int) {
	v, _ := clockSkew.LoadOrStore(driverID, &skewWindow{})
	w := v.(*skewWindow)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.start.IsZero() || now.Sub(w.start) >= time.Hour {
		prevCount = w.n
		w.start, w.n = now, 1
		return true, prevCount
	}
	w.n++
	return false, 0
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
