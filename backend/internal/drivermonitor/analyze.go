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
	}
}

// Window نافذة طلب للسائق: متى أُسند، ومتى استلم، ومتى انتهى.
type Window struct {
	OrderID    string
	Number     int64
	Store      string
	AssignedAt *time.Time
	PickedUpAt *time.Time
	EndAt      *time.Time // تسليم أو إغلاق؛ فارغ = ما زال مفتوحا
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
	// Road أمتار الطريق المطابق (`road.go`، `LoadRoad`). **nil = متل قبل بالضبط**: خطوط مستقيمة.
	Road *Road
}

// Part جزء من المسار بتصنيف واحد (للخريطة).
//
// **وبينقسم كمان** لما يتغيّر نوع الطريق أو الطلبات اللي معه — فطريق طلب واحد بيتلوّن لحاله.
type Part struct {
	Class  Class        `json:"class"`
	Coords [][2]float64 `json:"coords"` // [lat, lng]
	// Road ع الطريق المطابق (خط صلب). Inferred الشارع مقدَّر (منقّط). Straight المطابق حاول وما
	// زبط ⇒ خط مستقيم (متقطّع). Pending لسّا ما انطابق. **وكلّن فاضيين = متل قبل.**
	Road     bool `json:"road,omitempty"`
	Inferred bool `json:"inferred,omitempty"`
	Straight bool `json:"straight,omitempty"`
	Pending  bool `json:"pending,omitempty"`
	// From و To وقت أول وآخر نقطة بالجزء.
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
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
	// KmStraight نفس المقاطع المحسوبة بخط مستقيم — للمقارنة كم أسبوع بعد ما تشتغل المطابقة.
	KmStraight float64 `json:"km_straight"`
	// RoadShare حصّة الأمتار المحسوبة ع الطريق المطابق (٠–١).
	RoadShare float64 `json:"road_share"`
	// InferredKm منها: الشارع مقدَّر (نقطتين أبعد من ٥٠٠ م).
	InferredKm float64 `json:"inferred_km"`
	// PendingKm محسوبة بخط مستقيم لأنها لسّا ما انطابقت — **بس والمطابقة شغّالة** ⇒ الرقم تقريبي.
	PendingKm float64 `json:"pending_km"`
}

// Result نتيجة تحليل سائق ليوم.
type Result struct {
	Summary Summary `json:"summary"`
	Path    []Part  `json:"path"`
	Events  []Event `json:"events"`
	// Score ترتيب الخطر: الأعلى أولا.
	Score float64 `json:"score"`
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
	raw := append([]Point(nil), in.Points...)
	sort.Slice(raw, func(a, b int) bool { return raw[a].At.Before(raw[b].At) })

	res := Result{Path: []Part{}, Events: []Event{}}
	res.Summary.Points = len(raw)
	res.Summary.Orders = len(in.Windows)

	// ── المزيَّف: يُعدّ من الخام قبل أي تنظيف ──
	var mockedFirst *Point
	for i := range raw {
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

	orderState := func(t time.Time) (with, to bool) {
		for _, w := range in.Windows {
			end := time.Time{}
			if w.EndAt != nil {
				end = *w.EndAt
			} else if !in.Now.IsZero() {
				end = in.Now
			}
			if !end.IsZero() && t.After(end) {
				continue
			}
			if w.PickedUpAt != nil && !t.Before(*w.PickedUpAt) {
				with = true
				continue
			}
			if w.AssignedAt != nil && !t.Before(*w.AssignedAt) {
				to = true
			}
		}
		return
	}
	active := func(t time.Time) bool { w, to := orderState(t); return w || to }

	// ── انقطاعات: على الخام (أي إشارة تعني التطبيق حيّ) ──
	for k := 1; k < len(raw); k++ {
		prev, cur := raw[k-1], raw[k]
		dt := cur.At.Sub(prev.At)
		if dt <= cfg.GapAfter {
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

	// ── تنظيف: دقة سيّئة وقفزات مستحيلة تُهمَل بالقياس (`Clean`) ──
	pts := Clean(raw, cfg)

	// ── الوقفات ──
	longStops := longStops(pts, cfg)
	inStop := make([]int, len(pts))
	for i := range inStop {
		inStop[i] = -1
	}
	for si, s := range longStops {
		for k := s.i; k <= s.j; k++ {
			inStop[k] = si
		}
		res.Summary.StoppedMinutes += int(s.dur() / time.Minute)
	}

	// ── المقاطع: تصنيف + كيلومترات (ع الطريق إن انطابق) + مسار ──
	// windowsAt مفتاح الطلبات اللي مع السائق بلحظة — الجزء بينقسم لما يتغيّر.
	windowsAt := func(t time.Time) string {
		key := ""
		for _, w := range in.Windows {
			start := w.AssignedAt
			if start == nil {
				start = w.PickedUpAt
			}
			if start == nil || t.Before(*start) {
				continue
			}
			if w.EndAt != nil && t.After(*w.EndAt) {
				continue
			}
			key += w.OrderID + ","
		}
		return key
	}
	var cur *Part
	curKey := ""
	var roadM, countedM float64
	for k := 1; k < len(pts); k++ {
		a, b := pts[k-1], pts[k]
		cls := ClassNoOrder
		switch {
		case b.At.Sub(a.At) > cfg.GapAfter:
			cls = ClassGap
		case inStop[k] >= 0 && inStop[k] == inStop[k-1]:
			cls = ClassStopped
		default:
			mid := a.At.Add(b.At.Sub(a.At) / 2)
			if w, to := orderState(mid); w {
				cls = ClassWithOrder
			} else if to {
				cls = ClassToPickup
			}
		}
		counted := cls == ClassWithOrder || cls == ClassToPickup || cls == ClassNoOrder
		seg := RoadSeg{Kind: SegNone, M: Dist(a, b), Line: [][2]float64{{a.Lat, a.Lng}, {b.Lat, b.Lng}}}
		if counted && in.Road != nil {
			seg = in.Road.Segment(a, b, cfg)
		}
		d := seg.M
		switch cls {
		case ClassWithOrder:
			res.Summary.KmWithOrder += d / 1000
		case ClassToPickup:
			res.Summary.KmToPickup += d / 1000
		case ClassNoOrder:
			res.Summary.KmNoOrder += d / 1000
		}
		if counted {
			res.Summary.KmStraight += Dist(a, b) / 1000
			countedM += d
			switch seg.Kind {
			case SegRoad:
				roadM += d
			case SegInferred:
				roadM += d
				res.Summary.InferredKm += d / 1000
			case SegPending:
				res.Summary.PendingKm += d / 1000
			}
		}
		part := Part{Class: cls, Road: seg.Kind == SegRoad || seg.Kind == SegInferred,
			Inferred: seg.Kind == SegInferred, Straight: seg.Kind == SegStraight, Pending: seg.Kind == SegPending}
		key := ""
		if counted {
			key = windowsAt(a.At.Add(b.At.Sub(a.At) / 2))
		}
		if cur == nil || cur.Class != part.Class || cur.Road != part.Road || cur.Inferred != part.Inferred ||
			cur.Straight != part.Straight || cur.Pending != part.Pending || curKey != key {
			part.From, part.To = a.At, b.At
			part.Coords = [][2]float64{seg.Line[0]}
			res.Path = append(res.Path, part)
			cur = &res.Path[len(res.Path)-1]
			curKey = key
		}
		line := seg.Line
		if n := len(cur.Coords); n > 0 && len(line) > 0 && cur.Coords[n-1] == line[0] {
			line = line[1:]
		}
		cur.Coords = append(cur.Coords, line...)
		cur.To = b.At
	}
	res.Summary.KmTotal = res.Summary.KmWithOrder + res.Summary.KmToPickup + res.Summary.KmNoOrder
	if countedM > 0 {
		res.Summary.RoadShare = roadM / countedM
	}

	// ── وقفات عند متاجر بلا طلب + نمط «طلب من ورا ضهري» ──
	short := findStops(pts, cfg.PatternDropMin, cfg.StopRadiusM, cfg.GapAfter)
	for a := range short {
		s1 := short[a]
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
	res.Score = float64(res.Summary.Patterns)*1000 + float64(res.Summary.StoreStops)*50 +
		float64(flaggedGaps)*100 + res.Summary.KmNoOrder
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

// longStops الوقفات الطويلة اللي بيصنّفها `Analyze` «واقف» — على نقاط منظّفة (المزيَّفة ضمنها).
func longStops(pts []Point, cfg Config) []stop {
	return findStops(pts, cfg.StopMin, cfg.StopRadiusM, cfg.GapAfter)
}

// LongStops **نفس الوقفات الطويلة اللي بيشوفها `Analyze`** — على نقاط منظّفة بـ`Clean` ومرتّبة،
// **والمزيَّفة ضمنها** (بعكس `FindStops`). المطابق بيقطّع عندها، فالاتنين بيشوفوا نفس الوقفات.
func LongStops(pts []Point, cfg Config) []Stop {
	if cfg.GapAfter == 0 {
		cfg = DefaultConfig()
	}
	var out []Stop
	for _, s := range longStops(pts, cfg) {
		out = append(out, Stop{Start: s.start, End: s.end, Lat: s.lat, Lng: s.lng})
	}
	return out
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
