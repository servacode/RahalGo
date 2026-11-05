package drivermonitor

// ══════════════════════════════════════════════════════════════════════
// **أرقام الدوام ومسافة كل طلب — تحليلٌ واحد لكل دوام** (قرار المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// الكانس (`server/drive_sweeper.go`) ينادي `MeasureShift` لكل دوام: نقاطه وولاياته →
// `Analyze` واحد → قيادة الدوام وتقسيمها، ومسافة كل طلب. **ومن تحليلٍ واحد** تتطابق
// الأجزاء: مجموع الطلبات = القيادة بطلب. وعدّاد السائق وسجلّه وملف الطلب يقرؤون ما
// كتبه الكانس — **ولا أحد يقيس مرّةً ثانية بطريقة ثانية.**
//
// # ومسافةُ الطلب قطعٌ بعدد الدوامات التي مسّها
//
// طلبٌ بقي بيد السائق وانسكّر دوامُه (طارئٌ بعد الاستلام، أو ترك طلبٍ وبيده آخرُ مستلَم)
// **له قطعةٌ في كلّ دوام**: الأولى تنتهي بسكّر دوامها (`shift_closed`)، والثانية من فتح
// الدوام التالي — **إن أنهاه هو فيه تسليماً أو فشلاً** (`ShiftTenures`). وطلبٌ أعادته العمليّات
// للطابور أو ألغته أو بقي مفتوحاً لا يأخذ من الدوام الجديد شيئاً: **طلبٌ منسيٌّ لا يبلع
// كيلومتراتِ يومٍ كامل.** ومسافةُ الطلب مجموعُ قطعه (`order_drives` بمفتاح الدوام).

import (
	"context"
	"math"
	"sort"
	"time"
)

const (
	// ShiftPad نقاط قبل الدوام وبعده للتنظيف والوقفات — والحساب داخل الدوام وحده.
	ShiftPad = 10 * time.Minute
	// FinalAfter **متى يثبت الرقم**: الدفعة تقبل نقاطاً عمرها ساعتان (`batchMaxAge`) + هامش.
	FinalAfter = 2*time.Hour + 10*time.Minute
	// CloseGrace **نهايةُ الطلب بعد السكّر بلحظات** — التركُ والطارئ يسكّران الدوام ثمّ ينقلان
	// الطلب — تبقى نهايتَه الحقيقيّة («تركه») لا «انسكّر الدوام».
	CloseGrace = time.Minute
	// OpenMeasureMax **أطولُ ما يُقاس حيّاً من دوامٍ مفتوح**. المالكُ قرّر ألّا يُسكَّر الدوامُ
	// تلقائيّاً، فمفتاحٌ بقي شغّالاً أيّاماً لا يُحمَّل أثرُه كلُّه كلَّ دقيقة: يُقاس حيّاً على
	// آخر يومٍ منه، **وكاملاً حين يُسكَّر.** (ويُرى أحمرَ بالمراقبة: «دوام مفتوح من … ساعة».)
	OpenMeasureMax = 24 * time.Hour
)

// ShiftSpan دوامٌ بحدّيه — والمفتوح بلا نهاية.
type ShiftSpan struct {
	ID       string
	DriverID string
	OpenedAt time.Time
	ClosedAt *time.Time
}

// ShiftKm أرقام الدوام بالمتر.
type ShiftKm struct {
	DriveM      int // قيادة فعلية كلها
	OrdersM     int // منها بطلب (للمتجر + للزبون)
	NoOrderM    int // منها بلا طلب
	NotDrivingM int // حركة ما انحسبت (رجفة أو بطيء)
	GapMin      int
	OrdersN     int
	Points      int
}

// OrderDrive مسافة طلبٍ واحد لسائقه **في دوامٍ واحد** (قطعة).
type OrderDrive struct {
	OrderID    string
	StartedAt  time.Time
	PickedUpAt *time.Time
	EndedAt    *time.Time // فارغ = لسّا معه (ولا يكون إلّا بدوامٍ مفتوح)
	// EndKind open · delivered · failed · returned · cancelled · released · store_failed ·
	// shift_closed (انسكّر الدوام والطلبُ معه) · other
	EndKind    string
	ToPickupM  int
	WithOrderM int
	Shared     bool
	GapMin     int
	Points     int
}

func meters(km float64) int { return int(math.Round(km * 1000)) }

func windowStartOf(w Window) *time.Time {
	if w.AssignedAt != nil {
		return w.AssignedAt
	}
	return w.PickedUpAt
}

// ShiftTenures **ولاياتُ الدوام**: ما بدأ فيه كلُّه، **وما بدأ قبل فتحه** (دوامٌ سابقٌ سُكّر
// والطلبُ بيده) **إن أنهاه في هذا الدوام تسليماً أو فشلاً** — فيأخذ الدوامُ مشوارَه إلى الزبون.
// وما أعادته العمليّات للطابور أو ألغته، أو بقي مفتوحاً، أو انتهى بعد سكّر الدوام: لا شيء.
func ShiftTenures(ws []Window, s ShiftSpan) []Window {
	out := make([]Window, 0, len(ws))
	for _, w := range ws {
		st := windowStartOf(w)
		if st == nil || !st.Before(s.OpenedAt) || carriedInto(w, s) {
			out = append(out, w)
		}
	}
	return out
}

// carriedInto ولايةٌ بدأت قبل الدوام وانتهت فيه بيد صاحبها (تسليمٌ أو فشلٌ أو مشوارُ إرجاعه).
func carriedInto(w Window, s ShiftSpan) bool {
	switch w.EndKind {
	case "delivered", "failed", "returned":
	default:
		return false
	}
	e := w.EndEventAt
	if e == nil {
		e = w.EndAt
	}
	if e == nil || e.Before(s.OpenedAt) {
		return false
	}
	return s.ClosedAt == nil || !e.After(s.ClosedAt.Add(CloseGrace))
}

// MeasureShift **تحليلٌ واحد لدوام**: أرقامه ومسافة كل طلبٍ مسّه (قطعتُه فيه).
func MeasureShift(ctx context.Context, db DB, s ShiftSpan, now time.Time) (ShiftKm, []OrderDrive, error) {
	end := now
	if s.ClosedAt != nil && s.ClosedAt.Before(now) {
		end = *s.ClosedAt
	}
	begin := s.OpenedAt
	partial := s.ClosedAt == nil && end.Sub(begin) > OpenMeasureMax
	if partial {
		begin = end.Add(-OpenMeasureMax)
	}
	from, to := begin.Add(-ShiftPad), end.Add(ShiftPad)
	pts, err := LoadPoints(ctx, db, s.DriverID, from, to)
	if err != nil {
		return ShiftKm{}, nil, err
	}
	all, err := LoadTenures(ctx, db, s.DriverID, from, to, now)
	if err != nil {
		return ShiftKm{}, nil, err
	}
	wins := ShiftTenures(all, s)
	res := Analyze(Input{Points: pts, Windows: wins, Now: now, From: begin, To: end, Cfg: DefaultConfig()})
	sum := res.Summary
	km := ShiftKm{
		DriveM:      meters(sum.DriveKmTotal),
		OrdersM:     meters(sum.DriveKmWithOrder + sum.DriveKmToPickup),
		NoOrderM:    meters(sum.DriveKmNoOrder),
		NotDrivingM: meters(sum.NotDrivingKm),
		GapMin:      sum.GapMinutes,
		Points:      sum.Points,
	}
	// **طلبٌ له ولايتان في الدوام يُجمع قطعةً واحدة** — من أوّل قبولٍ لآخر خروج.
	byOrder := map[string]*OrderDrive{}
	var order []string
	for _, w := range wins {
		t := res.Tenures[w.OrderID]
		if t == nil {
			continue // ما مسّ الدوام
		}
		st := windowStartOf(w)
		if st == nil {
			continue
		}
		start := *st
		if start.Before(s.OpenedAt) {
			start = s.OpenedAt // قطعةٌ كمّلت من دوامٍ سابق
		}
		endAt, kind := w.EndAt, w.EndKind
		// **ودوامٌ مسكَّر يقصّ قطعتَه عند سكّره** — فلا يبقى صفٌّ «لسّا معه» بدوامٍ ثبت.
		if s.ClosedAt != nil && (endAt == nil || endAt.After(s.ClosedAt.Add(CloseGrace))) {
			c := *s.ClosedAt
			endAt, kind = &c, "shift_closed"
		}
		d := byOrder[w.OrderID]
		if d == nil {
			// **والمسافةُ من `Tenures` بمعرّف الطلب** — مجموعُ ولاياته في الدوام أصلاً.
			d = &OrderDrive{OrderID: w.OrderID, StartedAt: start,
				ToPickupM: int(math.Round(t.ToPickupM)), WithOrderM: int(math.Round(t.WithOrderM)),
				Shared: t.Shared, GapMin: t.GapMin, Points: t.Points}
			byOrder[w.OrderID] = d
			order = append(order, w.OrderID)
		}
		if start.Before(d.StartedAt) {
			d.StartedAt = start
		}
		if d.PickedUpAt == nil && w.PickedUpAt != nil {
			p := *w.PickedUpAt
			d.PickedUpAt = &p
		}
		switch {
		case endAt == nil:
			d.EndedAt, d.EndKind = nil, "open"
		case d.EndKind == "open":
			// ولايةٌ مفتوحةٌ تغلب ما قبلها.
		case d.EndedAt == nil || endAt.After(*d.EndedAt):
			e := *endAt
			d.EndedAt, d.EndKind = &e, kind
		}
	}
	out := make([]OrderDrive, 0, len(order))
	for _, id := range order {
		d := byOrder[id]
		// **ودوامٌ مفتوحٌ قِيس على آخر يومٍ منه**: طلبٌ بدأ قبل ذلك لا يُكتب جزءُه — يبقى صفُّه
		// كما حُسب، ويُقاس كاملاً حين يُسكَّر الدوام.
		if partial && d.StartedAt.Before(begin) {
			continue
		}
		if d.EndKind == "" {
			d.EndKind = "other"
		}
		out = append(out, *d)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	km.OrdersN = len(out)
	return km, out, nil
}
