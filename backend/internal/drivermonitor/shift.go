package drivermonitor

// ══════════════════════════════════════════════════════════════════════
// **أرقام الدوام ومسافة كل طلب — تحليلٌ واحد لكل دوام** (قرار المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// الكانس (`server/drive_sweeper.go`) ينادي `MeasureShift` لكل دوام: نقاطه وولاياته →
// `Analyze` واحد → قيادة الدوام وتقسيمها، ومسافة كل طلب. **ومن تحليلٍ واحد** تتطابق
// الأجزاء: مجموع الطلبات = القيادة بطلب. وعدّاد السائق وسجلّه وملف الطلب يقرؤون ما
// كتبه الكانس — **ولا أحد يقيس مرّةً ثانية بطريقة ثانية.**

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

// OrderDrive مسافة طلبٍ واحد لسائقه.
type OrderDrive struct {
	OrderID    string
	StartedAt  time.Time
	PickedUpAt *time.Time
	EndedAt    *time.Time // فارغ = لسّا معه
	EndKind    string     // open · delivered · failed · returned · cancelled · released · other
	ToPickupM  int
	WithOrderM int
	Shared     bool
	GapMin     int
	Points     int
}

func meters(km float64) int { return int(math.Round(km * 1000)) }

// MeasureShift **تحليلٌ واحد لدوام**: أرقامه ومسافة كل طلبٍ مسّه.
func MeasureShift(ctx context.Context, db DB, s ShiftSpan, now time.Time) (ShiftKm, []OrderDrive, error) {
	end := now
	if s.ClosedAt != nil && s.ClosedAt.Before(now) {
		end = *s.ClosedAt
	}
	from, to := s.OpenedAt.Add(-ShiftPad), end.Add(ShiftPad)
	pts, err := LoadPoints(ctx, db, s.DriverID, from, to)
	if err != nil {
		return ShiftKm{}, nil, err
	}
	wins, err := LoadTenures(ctx, db, s.DriverID, from, to, now)
	if err != nil {
		return ShiftKm{}, nil, err
	}
	res := Analyze(Input{Points: pts, Windows: wins, Now: now, From: s.OpenedAt, To: end, Cfg: DefaultConfig()})
	sum := res.Summary
	km := ShiftKm{
		DriveM:      meters(sum.DriveKmTotal),
		OrdersM:     meters(sum.DriveKmWithOrder + sum.DriveKmToPickup),
		NoOrderM:    meters(sum.DriveKmNoOrder),
		NotDrivingM: meters(sum.NotDrivingKm),
		GapMin:      sum.GapMinutes,
		Points:      sum.Points,
	}
	// **طلبٌ له ولايتان يُجمع صفّاً واحداً** — من أوّل قبولٍ لآخر خروج.
	byOrder := map[string]*OrderDrive{}
	var order []string
	for _, w := range wins {
		t := res.Tenures[w.OrderID]
		if t == nil {
			continue // ما مسّ الدوام
		}
		start := w.AssignedAt
		if start == nil {
			start = w.PickedUpAt
		}
		if start == nil {
			continue
		}
		d := byOrder[w.OrderID]
		if d == nil {
			// **والمسافةُ من `Tenures` بمعرّف الطلب** — مجموعُ ولاياته كلّها أصلاً.
			d = &OrderDrive{OrderID: w.OrderID, StartedAt: *start,
				ToPickupM: int(math.Round(t.ToPickupM)), WithOrderM: int(math.Round(t.WithOrderM)),
				Shared: t.Shared, GapMin: t.GapMin, Points: t.Points}
			byOrder[w.OrderID] = d
			order = append(order, w.OrderID)
		}
		if start.Before(d.StartedAt) {
			d.StartedAt = *start
		}
		if d.PickedUpAt == nil && w.PickedUpAt != nil {
			p := *w.PickedUpAt
			d.PickedUpAt = &p
		}
		switch {
		case w.EndAt == nil:
			d.EndedAt, d.EndKind = nil, "open"
		case d.EndKind == "open":
			// ولايةٌ مفتوحةٌ تغلب ما قبلها.
		case d.EndedAt == nil || w.EndAt.After(*d.EndedAt):
			e := *w.EndAt
			d.EndedAt, d.EndKind = &e, w.EndKind
		}
	}
	out := make([]OrderDrive, 0, len(order))
	for _, id := range order {
		d := byOrder[id]
		if d.EndKind == "" {
			d.EndKind = "other"
		}
		out = append(out, *d)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartedAt.Before(out[j].StartedAt) })
	km.OrdersN = len(out)
	return km, out, nil
}
