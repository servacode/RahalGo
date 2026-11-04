package drivermonitor

import (
	"context"
	"sort"
	"time"
)

// Loc توقيت دمشق — اليوم عند المنصّة يبدأ منتصف الليل بتوقيتها.
func Loc() *time.Location {
	if l, err := time.LoadLocation("Asia/Damascus"); err == nil {
		return l
	}
	return time.FixedZone("Asia/Damascus", 3*60*60)
}

// DayBounds بداية اليوم الدمشقيّ ونهايته لتاريخ `YYYY-MM-DD`.
func DayBounds(day string) (time.Time, time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", day, Loc())
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return d, d.AddDate(0, 0, 1), nil
}

// Today تاريخ اليوم الدمشقيّ.
func Today(now time.Time) string { return now.In(Loc()).Format("2006-01-02") }

// LoadStores المتاجر الفعّالة التي لها موقع.
func LoadStores(ctx context.Context, db DB) ([]Store, error) {
	rows, err := db.Query(ctx, `
		SELECT id::text, name, ST_Y(location::geometry), ST_X(location::geometry)
		  FROM merchants
		 WHERE location IS NOT NULL AND status = 'active'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Store
	for rows.Next() {
		var s Store
		if err := rows.Scan(&s.ID, &s.Name, &s.Lat, &s.Lng); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// LoadPoints نقاط سائق في مدى زمني.
func LoadPoints(ctx context.Context, db DB, driverID string, from, to time.Time) ([]Point, error) {
	rows, err := db.Query(ctx, `
		SELECT recorded_at, lat, lng, accuracy_m, speed_mps, mocked
		  FROM driver_trail
		 WHERE driver_id = $1::uuid AND recorded_at >= $2 AND recorded_at < $3
		 ORDER BY recorded_at`, driverID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Point
	for rows.Next() {
		var p Point
		var acc, spd *float32
		if err := rows.Scan(&p.At, &p.Lat, &p.Lng, &acc, &spd, &p.Mocked); err != nil {
			return nil, err
		}
		if acc != nil {
			v := float64(*acc)
			p.AccuracyM = &v
		}
		if spd != nil {
			v := float64(*spd)
			p.SpeedMps = &v
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// LoadWindows نوافذ طلبات السائق التي تمسّ المدى.
//
// بداية النافذة: أول سطر إسناد له بسجل الطلب، وإلا القبول، وإلا الاستلام.
// **والطلب المنقول لسائق آخر لا يظهر هنا** — النافذة للسائق الحالي للطلب.
func LoadWindows(ctx context.Context, db DB, driverID string, from, to time.Time) ([]Window, error) {
	rows, err := db.Query(ctx, `
		SELECT o.id::text, o.number, COALESCE(m.name, ''),
		       COALESCE((SELECT min(e.created_at) FROM order_events e
		                  WHERE e.order_id = o.id AND e.driver_id = $1::uuid AND e.to_status = 'assigned'),
		                o.accepted_at, o.dispatched_at, o.picked_up_at),
		       o.picked_up_at,
		       COALESCE(o.delivered_at, o.returned_at, o.closed_at)
		  FROM orders o
		  LEFT JOIN merchants m ON m.id = o.merchant_id
		 WHERE o.driver_id = $1::uuid
		   AND COALESCE(o.picked_up_at, o.accepted_at, o.dispatched_at, o.created_at) < $3
		   AND COALESCE(o.delivered_at, o.returned_at, o.closed_at, $3) > $2
		 ORDER BY o.created_at`, driverID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Window
	for rows.Next() {
		var w Window
		if err := rows.Scan(&w.OrderID, &w.Number, &w.Store, &w.AssignedAt, &w.PickedUpAt, &w.EndAt); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

// DriverDay تحليل سائق ليوم دمشقيّ.
func DriverDay(ctx context.Context, db DB, driverID, day string, stores []Store, now time.Time) (Result, []Window, error) {
	from, to, err := DayBounds(day)
	if err != nil {
		return Result{}, nil, err
	}
	pts, err := LoadPoints(ctx, db, driverID, from, to)
	if err != nil {
		return Result{}, nil, err
	}
	// نافذة طلب بدأ قبل اليوم تحتاج نقطتها لتصنيف أول اليوم — يكفي الطلب نفسه.
	wins, err := LoadWindows(ctx, db, driverID, from, to)
	if err != nil {
		return Result{}, nil, err
	}
	in := Input{Points: pts, Windows: wins, Stores: stores, Cfg: DefaultConfig()}
	if now.Before(to) {
		in.Now = now
	} else {
		in.Now = to
	}
	return Analyze(in), wins, nil
}

// Row سطر سائق في قائمة اليوم.
type Row struct {
	DriverID string  `json:"driver_id"`
	Name     string  `json:"name"`
	Phone    string  `json:"phone"`
	Summary  Summary `json:"summary"`
	Flagged  int     `json:"flagged_gaps"`
	Alert    bool    `json:"alert"`
	Score    float64 `json:"score"`
	// Live الحالة الحيّة — null لأيام ماضية. (قرار المالك ٢٠٢٦-١١-٠٤)
	Live *Live `json:"live"`
}

// Overview سائقو اليوم الذين لهم أثر، الأخطر أولا.
func Overview(ctx context.Context, db DB, day string, alertKm float64, now time.Time) ([]Row, error) {
	from, to, err := DayBounds(day)
	if err != nil {
		return nil, err
	}
	// اليوم الحالي: يدخل كل سائق على الدوام ولو بلا أثر بعد، ليرى المالك كل العاملين.
	isToday := day == Today(now)
	rows, err := db.Query(ctx, `
		SELECT u.id::text, u.full_name, u.phone, u.on_shift
		  FROM users u
		 WHERE EXISTS (SELECT 1 FROM driver_trail t
		                WHERE t.driver_id = u.id AND t.recorded_at >= $1 AND t.recorded_at < $2)
		    OR ($3 AND u.on_shift AND EXISTS (SELECT 1 FROM user_roles ur
		                WHERE ur.user_id = u.id AND ur.role_code = 'driver'))`, from, to, isToday)
	if err != nil {
		return nil, err
	}
	var out []Row
	onShift := map[string]bool{}
	for rows.Next() {
		var r Row
		var shift bool
		if err := rows.Scan(&r.DriverID, &r.Name, &r.Phone, &shift); err != nil {
			rows.Close()
			return nil, err
		}
		onShift[r.DriverID] = shift
		out = append(out, r)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	stores, err := LoadStores(ctx, db)
	if err != nil {
		return nil, err
	}
	for i := range out {
		res, wins, err := DriverDay(ctx, db, out[i].DriverID, day, stores, now)
		if err != nil {
			return nil, err
		}
		if isToday {
			recent, err := LoadPoints(ctx, db, out[i].DriverID, now.Add(-LiveLookback), now.Add(time.Second))
			if err != nil {
				return nil, err
			}
			lv := DeriveLive(recent, now)
			lv.OnShift = onShift[out[i].DriverID]
			for _, w := range wins {
				if w.EndAt == nil {
					lv.HasOrder = true
				}
			}
			out[i].Live = &lv
		}
		out[i].Summary = res.Summary
		out[i].Flagged = res.FlaggedGaps()
		out[i].Score = res.Score
		out[i].Alert = alertKm > 0 && res.Summary.KmNoOrder >= alertKm
		if out[i].Alert {
			out[i].Score += 300
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Score > out[b].Score })
	return out, nil
}
