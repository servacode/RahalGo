package drivermonitor

import (
	"math"
	"sort"
	"time"
)

// Class تصنيف مقطع الحركة.
type Class string

const (
	ClassWithOrder Class = "with_order" // بين الاستلام والتسليم
	ClassToPickup  Class = "to_pickup"  // من الإسناد حتى الاستلام
	ClassNoOrder   Class = "no_order"   // يتحرك وما معه طلب
	ClassStopped   Class = "stopped"    // واقف ≥ ٣ دقائق ضمن ~٤٠ م
	ClassGap       Class = "gap"        // انقطعت الإشارة
)

// Config عتبات التحليل. `DefaultConfig` هي المعتمدة.
type Config struct {
	GapAfter        time.Duration // صمت أطول من هذا = انقطاع
	StopMin         time.Duration // أقل وقوف يُصنَّف stopped
	StopRadiusM     float64       // نصف قطر الوقوف
	StoreRadiusM    float64       // «عند المتجر»
	PatternStoreMin time.Duration // أقل وقوف عند متجر لنمط «طلب من ورا ضهري»
	PatternMoveM    float64       // أقل حركة بعد المتجر
	PatternDropMin  time.Duration // وقفة التسليم: من
	PatternDropMax  time.Duration // وقفة التسليم: إلى
	PatternWindow   time.Duration // أقصى مدة بين الوقفتين
	ReturnRadiusM   float64       // «رجع» إلى المتجر
	MaxAccuracyM    float64       // أسوأ دقة تُقبل بالقياس
	MaxSpeedMps     float64       // أسرع من هذا = قفزة تُهمَل
	DarkResumeM     float64       // رجع للإشارة بعيدا هذه المسافة = اختفاء مشبوه
	PlaceRadiusM    float64       // تسمية المكان باسم أقرب متجر

	// ── القيادة الفعلية (قرار المالك ٢٠٢٦-١١-٠٥: «الطريق يحسب حسب طريق السائق») ──
	//
	// المقطع المحسوب يكون **قيادة** إذا: سرعته ≥ DriveMinSpeedMps، ودقة نقطتيه ≤
	// DriveMaxAccuracyM (الغائبة تمرّ)، وسرعة الجوال للنقطة الثانية ≥ DriveMinPhoneSpeedMps
	// إن أرسلها، ولا نقطة مزيّفة. وغير ذلك «حركة ما انحسبت» (رجفة GPS أو زحف).
	DriveMinSpeedMps      float64
	DriveMinPhoneSpeedMps float64
	DriveMaxAccuracyM     float64
	// Meter **المقياس الواحد** لكل رقم مسافة بالمنصّة — فارغ = RealDrive بهالعتبات.
	Meter DriveMeter
}

// DefaultConfig العتبات المعتمدة (قرار المالك ٢٠٢٦-١١-٠٣).
func DefaultConfig() Config {
	return Config{
		GapAfter:        6 * time.Minute,
		StopMin:         3 * time.Minute,
		StopRadiusM:     40,
		StoreRadiusM:    60,
		PatternStoreMin: 2 * time.Minute,
		PatternMoveM:    800,
		PatternDropMin:  1 * time.Minute,
		PatternDropMax:  10 * time.Minute,
		PatternWindow:   45 * time.Minute,
		ReturnRadiusM:   150,
		MaxAccuracyM:    50,
		MaxSpeedMps:     120 / 3.6,
		DarkResumeM:     500,
		PlaceRadiusM:    100,
		// القاعدة المعتمدة D:/tmp/drivers_real.sql (فحص ٢٠٢٦-١١-٠٤: ٤٢٪ من كيلومترات
		// اليوم كانت رجفة GPS).
		DriveMinSpeedMps:      3,
		DriveMinPhoneSpeedMps: 1.5,
		DriveMaxAccuracyM:     30,
	}
}

// ══════════════════════════════════════════════════════════════════════
// **المقياس الواحد للمسافة المقطوعة فعلاً** (قرار المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// كل رقم مسافة بالمنصّة يمرّ من هون: صفحة المراقبة، ومسافة كل طلب، وعدّاد السائق،
// وأرقام الدوام مقابل العدّاد. **فما في رقمين بيختلفوا لأنهم انحسبوا بطريقتين.**
//
// واليوم المقياس هو القاعدة المعتمدة (`RealDrive`): مسافة الهواء للمقطع وقت يكون
// قيادة. ودفعة لاحقة بتبدّله بمطابقة الطريق (Valhalla) — **بتبديل هالواجهة وحدها**،
// وما بيتغيّر أيّ مستهلك.

// DriveMeter يحوّل نقاطاً منظّفة مرتّبة بالزمن إلى أمتار مقطوعة فعلاً.
type DriveMeter interface {
	// Meters لكل مقطع متتالٍ (pts[k-1] → pts[k]) كم متر انقطع فعلاً في الخانة k،
	// والخانة صفر دائماً صفر. **والصفر يعني «ما انحسب قيادة»**.
	Meters(pts []Point) []float64
}

// RealDrive القاعدة المعتمدة: مسافة الهواء للمقطع حين يكون قيادة (`IsDrive`).
type RealDrive struct{ Cfg Config }

// Meters انظر DriveMeter.
func (r RealDrive) Meters(pts []Point) []float64 {
	out := make([]float64, len(pts))
	for k := 1; k < len(pts); k++ {
		if IsDrive(pts[k-1], pts[k], r.Cfg) {
			out[k] = Dist(pts[k-1], pts[k])
		}
	}
	return out
}

// IsDrive **هل المقطع a→b قيادة فعلية؟** — دالة صافية.
//
// سرعة المقطع ≥ ٣ م/ث (١٠٫٨ كم/س)، ودقة النقطتين ≤ ٣٠ م (الغائبة تمرّ)، وسرعة
// الجوال للنقطة الثانية ≥ ١٫٥ م/ث إن أرسلها، **ولا نقطة مزيّفة** (موقعٌ مزيّف يصنع
// كيلومترات مزيّفة).
func IsDrive(a, b Point, cfg Config) bool {
	if a.Mocked || b.Mocked {
		return false
	}
	dt := b.At.Sub(a.At).Seconds()
	if dt <= 0 || Dist(a, b)/dt < cfg.DriveMinSpeedMps {
		return false
	}
	if b.SpeedMps != nil && *b.SpeedMps < cfg.DriveMinPhoneSpeedMps {
		return false
	}
	if a.AccuracyM != nil && *a.AccuracyM > cfg.DriveMaxAccuracyM {
		return false
	}
	if b.AccuracyM != nil && *b.AccuracyM > cfg.DriveMaxAccuracyM {
		return false
	}
	return true
}

// Window **ولاية سائق على طلب**: متى أُسند له، ومتى استلم، ومتى خرج من يده.
//
// طلبٌ تركه السائق (أو استُرجع منه) له ولاية تنتهي بالترك — فمشواره للمتجر محسوب
// «بطلب» لا «بلا طلب». وطلبٌ أُسند له مرّتين له ولايتان.
type Window struct {
	OrderID    string
	Number     int64
	Store      string
	AssignedAt *time.Time
	PickedUpAt *time.Time
	EndAt      *time.Time // خرج من يده؛ فارغ = ما زال معه
	// EndKind كيف انتهت: delivered · failed · returned · cancelled · released · other
	EndKind string
}

// Store متجر مسجّل له موقع.
type Store struct {
	ID       string
	Name     string
	Lat, Lng float64
}

// Input مدخلات التحليل.
type Input struct {
	Points  []Point
	Windows []Window
	Stores  []Store
	// Now يقصّ الطلبات المفتوحة. فارغ = لا قصّ.
	Now time.Time
	Cfg Config
	// From/To **مدى الحساب** (دوامٌ مثلاً): المقطع اللي منتصفه برّاه ما بينحسب، والنقاط
	// حوله تبقى للتنظيف والوقفات. فارغان = كل النقاط.
	From, To time.Time
}

// Part جزء من المسار بتصنيف واحد (للخريطة).
type Part struct {
	Class  Class        `json:"class"`
	Coords [][2]float64 `json:"coords"` // [lat, lng]
}

// Event سطر من قصة اليوم.
type Event struct {
	// Kind: stop_at_store · pattern · gap · stop · mocked
	Kind    string     `json:"kind"`
	At      time.Time  `json:"at"`
	EndAt   *time.Time `json:"end_at,omitempty"`
	Minutes int        `json:"minutes,omitempty"`
	Lat     float64    `json:"lat"`
	Lng     float64    `json:"lng"`
	Place   string     `json:"place,omitempty"` // اسم أقرب متجر إن وُجد
	// للنمط: الوقفة الثانية والمسافة والعودة
	ToLat     float64    `json:"to_lat,omitempty"`
	ToLng     float64    `json:"to_lng,omitempty"`
	ToAt      *time.Time `json:"to_at,omitempty"`
	ToMinutes int        `json:"to_minutes,omitempty"`
	ToPlace   string     `json:"to_place,omitempty"`
	DistanceM int        `json:"distance_m,omitempty"`
	Returned  bool       `json:"returned,omitempty"`
	// للانقطاع: هل هو مشبوه (كان يمشي أو معه طلب أو عاد بعيدا)
	Flagged   bool `json:"flagged,omitempty"`
	WithOrder bool `json:"with_order,omitempty"`
	Count     int  `json:"count,omitempty"` // للمزيَّف: عدد النقاط
}

// Summary أرقام اليوم.
type Summary struct {
	KmWithOrder    float64 `json:"km_with_order"`
	KmToPickup     float64 `json:"km_to_pickup"`
	KmNoOrder      float64 `json:"km_no_order"`
	KmTotal        float64 `json:"km_total"`
	Orders         int     `json:"orders"`
	Patterns       int     `json:"patterns"`
	StoreStops     int     `json:"store_stops"`
	Gaps           int     `json:"gaps"`
	GapMinutes     int     `json:"gap_minutes"`
	Mocked         int     `json:"mocked"`
	Points         int     `json:"points"`
	StoppedMinutes int     `json:"stopped_minutes"`

	// ── القيادة الفعلية (قرار المالك ٢٠٢٦-١١-٠٥) — منها التنبيه والترتيب ──
	//
	// km_* فوق هي كل مقطع محسوب (فيها رجفة GPS). وهون القيادة وحدها، والباقي
	// «حركة ما انحسبت»: drive_with + drive_to + drive_no + not_driving = km_total.
	DriveKmWithOrder float64 `json:"drive_km_with_order"`
	DriveKmToPickup  float64 `json:"drive_km_to_pickup"`
	DriveKmNoOrder   float64 `json:"drive_km_no_order"`
	DriveKmTotal     float64 `json:"drive_km_total"`
	NotDrivingKm     float64 `json:"not_driving_km"`
}

// TenureKm **مسافة طلبٍ واحد لسائقه** — من القيادة الفعلية وحدها.
//
// المقطع وطلبان بيده: يتقاسمانه بالتساوي (`Shared`) — فمجموع الطلبات = القيادة بطلب.
type TenureKm struct {
	ToPickupM  float64 // من القبول حتى الاستلام
	WithOrderM float64 // من الاستلام حتى خرج الطلب من يده (ومشوار الإرجاع منه)
	Shared     bool    // تقاسم مقطعاً مع طلبٍ آخر بيده
	GapMin     int     // دقائق بلا إشارة أثناء الطلب
	Points     int     // نقاط الأثر أثناء الطلب
}

// Result نتيجة تحليل سائق ليوم.
type Result struct {
	Summary Summary `json:"summary"`
	Path    []Part  `json:"path"`
	Events  []Event `json:"events"`
	// Score ترتيب الخطر: الأعلى أولا.
	Score float64 `json:"score"`
	// Tenures مسافة كل طلب (بمعرّفه) — للمكتب وللسائق. لا تُرسل كما هي.
	Tenures map[string]*TenureKm `json:"-"`
}

// Alert **تنبيه «بلا طلب»** — بالقيادة الفعلية لا برجفة GPS. وصفر = بلا تنبيه.
func (r Result) Alert(alertKm float64) bool {
	return alertKm > 0 && r.Summary.DriveKmNoOrder >= alertKm
}

// stop وقفة مكتشفة: فهرسا أول وآخر نقطة، ومركزها.
type stop struct {
	i, j     int
	start    time.Time
	end      time.Time
	lat, lng float64
}

func (s stop) dur() time.Duration { return s.end.Sub(s.start) }
func (s stop) mid() time.Time     { return s.start.Add(s.dur() / 2) }

// Analyze **يحلّل يوم سائق** — دالة صافية، لا قاعدة ولا ساعة.
func Analyze(in Input) Result {
	cfg := in.Cfg
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	meter := cfg.Meter
	if meter == nil {
		meter = RealDrive{Cfg: cfg}
	}
	raw := append([]Point(nil), in.Points...)
	sort.Slice(raw, func(a, b int) bool { return raw[a].At.Before(raw[b].At) })

	// ── مدى الحساب (دوام) — وفارغه كل النقاط ──
	spanned := !in.From.IsZero() || !in.To.IsZero()
	inSpan := func(t time.Time) bool {
		if !in.From.IsZero() && t.Before(in.From) {
			return false
		}
		return in.To.IsZero() || t.Before(in.To)
	}
	// windowEnd نهاية الولاية — والمفتوحة تُقصّ بـNow (وفارغ Now = بلا نهاية).
	windowEnd := func(w Window) time.Time {
		if w.EndAt != nil {
			return *w.EndAt
		}
		return in.Now
	}
	windowStart := func(w Window) time.Time {
		if w.AssignedAt != nil {
			return *w.AssignedAt
		}
		if w.PickedUpAt != nil {
			return *w.PickedUpAt
		}
		return time.Time{}
	}

	res := Result{Path: []Part{}, Events: []Event{}, Tenures: map[string]*TenureKm{}}
	for _, w := range in.Windows {
		st, end := windowStart(w), windowEnd(w)
		if spanned && ((!in.To.IsZero() && !st.IsZero() && !st.Before(in.To)) ||
			(!in.From.IsZero() && !end.IsZero() && end.Before(in.From))) {
			continue
		}
		res.Summary.Orders++
		t := res.Tenures[w.OrderID]
		if t == nil {
			t = &TenureKm{}
			res.Tenures[w.OrderID] = t
		}
		for _, p := range raw {
			if inSpan(p.At) && !st.IsZero() && !p.At.Before(st) && (end.IsZero() || !p.At.After(end)) {
				t.Points++
			}
		}
	}

	// ── المزيَّف: يُعدّ من الخام قبل أي تنظيف ──
	var mockedFirst *Point
	for i := range raw {
		if !inSpan(raw[i].At) {
			continue
		}
		res.Summary.Points++
		if raw[i].Mocked {
			res.Summary.Mocked++
			if mockedFirst == nil {
				mockedFirst = &raw[i]
			}
		}
	}
	if mockedFirst != nil {
		res.Events = append(res.Events, Event{Kind: "mocked", At: mockedFirst.At,
			Lat: mockedFirst.Lat, Lng: mockedFirst.Lng, Count: res.Summary.Mocked})
	}

	// activeAt **الولايات القائمة لحظةَ t** وحالُ كلٍّ منها (استلم أم لا بعد).
	type act struct {
		id   string
		with bool
	}
	activeAt := func(t time.Time) (with, to bool, list []act) {
		for _, w := range in.Windows {
			if end := windowEnd(w); !end.IsZero() && t.After(end) {
				continue
			}
			if w.PickedUpAt != nil && !t.Before(*w.PickedUpAt) {
				with = true
				list = append(list, act{w.OrderID, true})
				continue
			}
			if w.AssignedAt != nil && !t.Before(*w.AssignedAt) {
				to = true
				list = append(list, act{w.OrderID, false})
			}
		}
		return
	}
	active := func(t time.Time) bool { w, to, _ := activeAt(t); return w || to }

	// ── انقطاعات: على الخام (أي إشارة تعني التطبيق حيّ) ──
	for k := 1; k < len(raw); k++ {
		prev, cur := raw[k-1], raw[k]
		dt := cur.At.Sub(prev.At)
		if dt <= cfg.GapAfter {
			continue
		}
		// **ودقائقُ الصمت داخل كل طلب** — ليعرف المكتب أن رقماً قليلاً سببه جوالٌ صامت.
		for _, w := range in.Windows {
			t := res.Tenures[w.OrderID]
			st, end := windowStart(w), windowEnd(w)
			if t == nil || st.IsZero() {
				continue
			}
			lo, hi := prev.At, cur.At
			if st.After(lo) {
				lo = st
			}
			if !end.IsZero() && end.Before(hi) {
				hi = end
			}
			if !in.From.IsZero() && in.From.After(lo) {
				lo = in.From
			}
			if !in.To.IsZero() && in.To.Before(hi) {
				hi = in.To
			}
			if hi.After(lo) {
				t.GapMin += int(hi.Sub(lo) / time.Minute)
			}
		}
		if !inSpan(prev.At) {
			continue
		}
		moving := prev.SpeedMps != nil && *prev.SpeedMps >= 1.0
		if prev.SpeedMps == nil && k >= 2 {
			if d := Dist(raw[k-2], prev); d > 0 {
				if s := prev.At.Sub(raw[k-2].At).Seconds(); s > 0 && d/s >= 1.0 {
					moving = true
				}
			}
		}
		withOrder := active(prev.At)
		far := Dist(prev, cur) >= cfg.DarkResumeM
		mins := int(dt / time.Minute)
		end := cur.At
		res.Summary.Gaps++
		res.Summary.GapMinutes += mins
		res.Events = append(res.Events, Event{
			Kind: "gap", At: prev.At, EndAt: &end, Minutes: mins,
			Lat: prev.Lat, Lng: prev.Lng, Place: nearestStore(in.Stores, prev.Lat, prev.Lng, cfg.PlaceRadiusM),
			Flagged: moving || withOrder || far, WithOrder: withOrder,
		})
	}

	// ── تنظيف: دقة سيّئة وقفزات مستحيلة تُهمَل بالقياس ──
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

	// ── الوقفات ──
	longStops := findStops(pts, cfg.StopMin, cfg.StopRadiusM, cfg.GapAfter)
	inStop := make([]int, len(pts))
	for i := range inStop {
		inStop[i] = -1
	}
	for si, s := range longStops {
		for k := s.i; k <= s.j; k++ {
			inStop[k] = si
		}
		if inSpan(s.start) {
			res.Summary.StoppedMinutes += int(s.dur() / time.Minute)
		}
	}

	// ── المقاطع: تصنيف + كيلومترات + قيادة فعلية + مسافة كل طلب + مسار ──
	meters := meter.Meters(pts)
	var cur *Part
	for k := 1; k < len(pts); k++ {
		a, b := pts[k-1], pts[k]
		mid := a.At.Add(b.At.Sub(a.At) / 2)
		if !inSpan(mid) {
			cur = nil
			continue
		}
		cls := ClassNoOrder
		var acts []act
		switch {
		case b.At.Sub(a.At) > cfg.GapAfter:
			cls = ClassGap
		case inStop[k] >= 0 && inStop[k] == inStop[k-1]:
			cls = ClassStopped
		default:
			w, to, list := activeAt(mid)
			acts = list
			if w {
				cls = ClassWithOrder
			} else if to {
				cls = ClassToPickup
			}
		}
		d := Dist(a, b)
		drive := 0.0
		if cls != ClassGap && cls != ClassStopped && k < len(meters) {
			drive = meters[k]
		}
		switch cls {
		case ClassWithOrder:
			res.Summary.KmWithOrder += d / 1000
			res.Summary.DriveKmWithOrder += drive / 1000
		case ClassToPickup:
			res.Summary.KmToPickup += d / 1000
			res.Summary.DriveKmToPickup += drive / 1000
		case ClassNoOrder:
			res.Summary.KmNoOrder += d / 1000
			res.Summary.DriveKmNoOrder += drive / 1000
		}
		if (cls == ClassWithOrder || cls == ClassToPickup || cls == ClassNoOrder) && drive <= 0 {
			res.Summary.NotDrivingKm += d / 1000
		}
		// **والقيادةُ بطلبٍ تُقسَم على الطلبات التي بيده بالتساوي** — فلا تُعدّ مرّتين.
		if drive > 0 && len(acts) > 0 {
			share := drive / float64(len(acts))
			for _, x := range acts {
				t := res.Tenures[x.id]
				if t == nil {
					t = &TenureKm{}
					res.Tenures[x.id] = t
				}
				if x.with {
					t.WithOrderM += share
				} else {
					t.ToPickupM += share
				}
				if len(acts) > 1 {
					t.Shared = true
				}
			}
		}
		if cur == nil || cur.Class != cls {
			res.Path = append(res.Path, Part{Class: cls, Coords: [][2]float64{{a.Lat, a.Lng}}})
			cur = &res.Path[len(res.Path)-1]
		}
		cur.Coords = append(cur.Coords, [2]float64{b.Lat, b.Lng})
	}
	res.Summary.KmTotal = res.Summary.KmWithOrder + res.Summary.KmToPickup + res.Summary.KmNoOrder
	res.Summary.DriveKmTotal = res.Summary.DriveKmWithOrder + res.Summary.DriveKmToPickup + res.Summary.DriveKmNoOrder

	// ── وقفات عند متاجر بلا طلب + نمط «طلب من ورا ضهري» ──
	short := findStops(pts, cfg.PatternDropMin, cfg.StopRadiusM, cfg.GapAfter)
	for a := range short {
		s1 := short[a]
		if !inSpan(s1.start) {
			continue
		}
		if s1.dur() < cfg.PatternStoreMin || active(s1.start) || active(s1.mid()) || active(s1.end) {
			continue
		}
		store, ok := storeNear(in.Stores, s1.lat, s1.lng, cfg.StoreRadiusM)
		if !ok {
			continue
		}
		if s1.dur() >= cfg.StopMin {
			res.Summary.StoreStops++
			end := s1.end
			res.Events = append(res.Events, Event{Kind: "stop_at_store", At: s1.start, EndAt: &end,
				Minutes: int(s1.dur() / time.Minute), Lat: s1.lat, Lng: s1.lng, Place: store.Name})
		}
		for b := a + 1; b < len(short); b++ {
			s2 := short[b]
			if s2.start.Sub(s1.end) > cfg.PatternWindow {
				break
			}
			if s2.dur() < cfg.PatternDropMin || s2.dur() > cfg.PatternDropMax {
				continue
			}
			dist := Dist(Point{Lat: s1.lat, Lng: s1.lng}, Point{Lat: s2.lat, Lng: s2.lng})
			if dist < cfg.PatternMoveM {
				continue
			}
			between := s1.end.Add(s2.start.Sub(s1.end) / 2)
			if active(s2.mid()) || active(between) {
				continue
			}
			returned := false
			for _, p := range pts {
				if p.At.After(s2.end) && p.At.Sub(s2.end) <= cfg.PatternWindow &&
					Dist(p, Point{Lat: s1.lat, Lng: s1.lng}) <= cfg.ReturnRadiusM {
					returned = true
					break
				}
			}
			end1 := s1.end
			res.Summary.Patterns++
			res.Events = append(res.Events, Event{Kind: "pattern", At: s1.start, EndAt: &end1,
				Minutes: int(s1.dur() / time.Minute), Lat: s1.lat, Lng: s1.lng, Place: store.Name,
				ToLat: s2.lat, ToLng: s2.lng, ToAt: &s2.start, ToMinutes: int(math.Ceil(s2.dur().Minutes())),
				ToPlace:   nearestStore(in.Stores, s2.lat, s2.lng, cfg.PlaceRadiusM),
				DistanceM: int(dist), Returned: returned})
			break
		}
	}

	sort.SliceStable(res.Events, func(i, j int) bool { return res.Events[i].At.Before(res.Events[j].At) })
	flaggedGaps := 0
	for _, e := range res.Events {
		if e.Kind == "gap" && e.Flagged {
			flaggedGaps++
		}
	}
	// **والترتيبُ بالقيادة بلا طلب لا برجفة GPS** (قرار المالك ٢٠٢٦-١١-٠٥).
	res.Score = float64(res.Summary.Patterns)*1000 + float64(res.Summary.StoreStops)*50 +
		float64(flaggedGaps)*100 + res.Summary.DriveKmNoOrder
	if res.Summary.Mocked > 0 {
		res.Score += 500
	}
	return res
}

// FlaggedGaps كم انقطاعا مشبوها في النتيجة.
func (r Result) FlaggedGaps() int {
	n := 0
	for _, e := range r.Events {
		if e.Kind == "gap" && e.Flagged {
			n++
		}
	}
	return n
}

// findStops وقفات: نقاط متتالية ضمن `radius` من أولها لمدة ≥ minDur بلا انقطاع.
func findStops(pts []Point, minDur time.Duration, radius float64, gap time.Duration) []stop {
	var out []stop
	i := 0
	for i < len(pts) {
		j := i
		for j+1 < len(pts) && Dist(pts[i], pts[j+1]) <= radius && pts[j+1].At.Sub(pts[j].At) <= gap {
			j++
		}
		if j > i && pts[j].At.Sub(pts[i].At) >= minDur {
			var la, ln float64
			for k := i; k <= j; k++ {
				la += pts[k].Lat
				ln += pts[k].Lng
			}
			n := float64(j - i + 1)
			out = append(out, stop{i: i, j: j, start: pts[i].At, end: pts[j].At, lat: la / n, lng: ln / n})
			i = j + 1
			continue
		}
		i++
	}
	return out
}

func storeNear(stores []Store, lat, lng, radius float64) (Store, bool) {
	best, bestD := Store{}, radius+1
	for _, s := range stores {
		d := Dist(Point{Lat: lat, Lng: lng}, Point{Lat: s.Lat, Lng: s.Lng})
		if d <= radius && d < bestD {
			best, bestD = s, d
		}
	}
	return best, bestD <= radius
}

func nearestStore(stores []Store, lat, lng, radius float64) string {
	if s, ok := storeNear(stores, lat, lng, radius); ok {
		return s.Name
	}
	return ""
}

// Stop وقفةٌ مكتشفة للاستعمال خارج الحزمة (الأماكن بلا اسم).
type Stop struct {
	Start, End time.Time
	Lat, Lng   float64
}

// FindStops وقفاتُ نقاطٍ مرتّبة بالزمن: متتاليةٌ ضمن `radius` من أولها لمدة ≥ `minDur` بلا صمتٍ أطولَ من `gap`.
// النقاطُ المزيَّفة تُهمل.
func FindStops(pts []Point, minDur time.Duration, radius float64, gap time.Duration) []Stop {
	clean := make([]Point, 0, len(pts))
	for _, p := range pts {
		if !p.Mocked {
			clean = append(clean, p)
		}
	}
	var out []Stop
	for _, s := range findStops(clean, minDur, radius, gap) {
		out = append(out, Stop{Start: s.start, End: s.end, Lat: s.lat, Lng: s.lng})
	}
	return out
}
