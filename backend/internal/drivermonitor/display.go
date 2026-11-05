package drivermonitor

import (
	"math"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **شو بيعني «تحرّك فعلاً» — تعريف واحد** (طلب المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «بدنا حركة السائق على الخريطة تكون مباشرة ما بدنا نطّات».
//
// **`Clean`** بينظّف النقاط قبل أي قياس (التحليل والحالة الحيّة)، و**`NextShown`** بيقرّر
// وين نرسم السائق على خريطة العمليات. الاتنين بيقروا نفس العتبات (`Config`):
// دقة أسوأ من ٥٠ م ما بتنقاس، وقفزة أسرع من ١٢٠ كم/س وأبعد من ١٠٠ م مستحيلة.
//
// **ونقطة العرض للعرض بس**: `users.last_location` و`driver_track` و`driver_trail`
// والتوزيع بيضلّوا ع النقطة الخام متل ما هي.

// Clean **بينظّف نقاط مرتّبة بالزمن**: الدقة الأسوأ من `MaxAccuracyM` بتنشال، والقفزة الأسرع
// من `MaxSpeedMps` والأبعد من ١٠٠ م عن آخر نقطة مقبولة بتنشال. **دالة صافية** — هي نفسها اللي
// كان `Analyze` يعملها جوّاته، طلعت لهون بلا أي تغيير بالسلوك.
func Clean(raw []Point, cfg Config) []Point {
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	pts := make([]Point, 0, len(raw))
	for _, p := range raw {
		if p.AccuracyM != nil && *p.AccuracyM > cfg.MaxAccuracyM {
			continue
		}
		if n := len(pts); n > 0 {
			dt := p.At.Sub(pts[n-1].At).Seconds()
			d := Dist(pts[n-1], p)
			if dt <= 0 && d > 0 {
				continue
			}
			if dt > 0 && d/dt > cfg.MaxSpeedMps && d > 100 {
				continue
			}
		}
		pts = append(pts, p)
	}
	return pts
}

// عتبات نقطة العرض.
const (
	// ShownRoughAfter إذا ما إجت نقطة دقيقة من هالمدة، منعرض النقطة الضعيفة وهي باهتة.
	ShownRoughAfter = 60 * time.Second
	// ShownHopHold القفزة المستحيلة بتنمسك هالمدة بالأكتر، بعدها بتنقبل — نقطة معروضة
	// غلط ما لازم تعلّق السائق بمكانه.
	ShownHopHold = 30 * time.Second
	// ShownHopMinM القفزة أقل من هيك ما بتنعدّ مستحيلة (نفس قاعدة `Clean`).
	ShownHopMinM = 100.0
	// ShownStillM أقل نصف قطر للرجفة وهو واقف (أو نص الدقة إذا أكبر).
	ShownStillM = 8.0
	// ShownStillSpeedMps سرعة الجهاز تحتها (أو بلا سرعة) = واقف.
	ShownStillSpeedMps = 1.0
)

// Shown **وين منرسم السائق** على خريطة العمليات.
type Shown struct {
	Lat, Lng float64
	// At آخر نقطة سمعناها منه — **بتتحدّث حتى لو النقطة انمسكت**، فبيضل «حي».
	At time.Time
	// PosAt وقت النقطة اللي حطّت الموضع (أو أكّدته وهو واقف) — منه بتنحسب سرعة القفزة.
	PosAt time.Time
	// SpeedMps و AccuracyM و BearingDeg تبع النقطة المعروضة.
	SpeedMps   *float64
	AccuracyM  *float64
	BearingDeg *float64
	// Rough الموضع من نقطة دقتها ضعيفة (ما إجت ولا نقطة دقيقة من دقيقة) — بيترسم باهت.
	Rough bool

	goodAt   time.Time // آخر نقطة دقتها مقبولة
	hopSince time.Time // أول نقطة قفزة مستحيلة انمسكت (صفر = ما في)
}

// Known **في موضع معروض؟** — قبل أول نقطة (أو بعد إعادة تشغيل الخادم) لأ.
func (s Shown) Known() bool { return !s.At.IsZero() }

// NextShown **دالة صافية**: النقطة المعروضة بعد ما وصلت `fix` (أحدث من اللي قبلها).
//
//  1. ما في موضع قبلها ⇒ منعرضها (باهتة إذا دقتها ضعيفة).
//  2. دقتها أسوأ من `MaxAccuracyM` ⇒ الموضع بيضل؛ وإذا ما إجت نقطة دقيقة من
//     `ShownRoughAfter` ⇒ منعرضها باهتة.
//  3. قفزة عن الموضع المعروض أسرع من `MaxSpeedMps` وأبعد من `ShownHopMinM` ⇒ الموضع بيضل
//     لحد `ShownHopHold`، بعدها بتنقبل.
//  4. ضمن max(`ShownStillM`، نص الدقة) وسرعة الجهاز أقل من `ShownStillSpeedMps` (أو بلا سرعة)
//     ⇒ الموضع بيضل ووقته بيتحدّث.
//  5. غير هيك ⇒ منعرضها.
//
// ونقطة أقدم من آخر وحدة سمعناها ما بتغيّر شي.
func NextShown(prev Shown, fix Point, cfg Config) Shown {
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	good := fix.AccuracyM == nil || *fix.AccuracyM <= cfg.MaxAccuracyM
	take := func(rough bool) Shown {
		s := Shown{Lat: fix.Lat, Lng: fix.Lng, At: fix.At, PosAt: fix.At,
			SpeedMps: fix.SpeedMps, AccuracyM: fix.AccuracyM, BearingDeg: fix.BearingDeg,
			Rough: rough, goodAt: prev.goodAt}
		if good {
			s.goodAt = fix.At
		}
		return s
	}

	// ١ · أول نقطة
	if !prev.Known() {
		return take(!good)
	}
	if fix.At.Before(prev.At) {
		return prev
	}
	held := prev
	held.At = fix.At

	// ٢ · دقة ضعيفة
	if !good {
		if prev.goodAt.IsZero() || fix.At.Sub(prev.goodAt) >= ShownRoughAfter {
			return take(true)
		}
		return held
	}
	held.goodAt = fix.At

	// الموضع الباهت ما بيمسك نقطة دقيقة: هي أحسن من اللي عنا.
	if prev.Rough {
		return take(false)
	}

	// ٣ · قفزة مستحيلة
	d := Dist(Point{Lat: prev.Lat, Lng: prev.Lng}, fix)
	dt := fix.At.Sub(prev.PosAt).Seconds()
	if d > ShownHopMinM && (dt <= 0 || d/dt > cfg.MaxSpeedMps) {
		if prev.hopSince.IsZero() {
			held.hopSince = fix.At
			return held
		}
		if fix.At.Sub(prev.hopSince) < ShownHopHold {
			return held
		}
		return take(false)
	}

	// ٤ · رجفة وهو واقف
	still := ShownStillM
	if fix.AccuracyM != nil {
		still = math.Max(still, *fix.AccuracyM/2)
	}
	if d <= still && (fix.SpeedMps == nil || *fix.SpeedMps < ShownStillSpeedMps) {
		held.PosAt = fix.At
		held.SpeedMps = fix.SpeedMps
		held.hopSince = time.Time{}
		return held
	}

	// ٥ · حركة حقيقية
	return take(false)
}
