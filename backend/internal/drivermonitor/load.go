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

// ══════════════════════════════════════════════════════════════════════
// **يومُ المراقبة يومُ الحساب** — من ساعة فتح الدوام لمثلها (قرار ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// صفحة المراقبة وعدّاد السائق والنقد يومٌ واحد: `platform.day_start_hour` (٩ افتراضاً)
// بتوقيت دمشق — **من دوالّ القاعدة نفسها** (`account_day` · `account_day_start`، الهجرة 0476)
// فما في ساعة مكتوبة هون تفترق عن الإعداد.

// AccountDay يومُ الحساب للحظةٍ ما (`YYYY-MM-DD`).
func AccountDay(ctx context.Context, db DB, at time.Time) (string, error) {
	var day string
	err := db.QueryRow(ctx, `SELECT account_day($1)::text`, at).Scan(&day)
	return day, err
}

// AccountDayBounds بدايةُ يوم الحساب ونهايتُه لتاريخ `YYYY-MM-DD`.
func AccountDayBounds(ctx context.Context, db DB, day string) (time.Time, time.Time, error) {
	if _, err := time.Parse("2006-01-02", day); err != nil {
		return time.Time{}, time.Time{}, err
	}
	var from, to time.Time
	err := db.QueryRow(ctx, `
		SELECT account_day_start($1::date), account_day_start($1::date + 1)`, day).Scan(&from, &to)
	return from, to, err
}

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

// ReturnTripMax **أطول مشوار إرجاعٍ يُحسب للطلب** — إن لم يضغط السائق «سلّمت البضاعة»
// بقيت ولايته مفتوحة، فتُقصّ هنا كي لا يصير كل ما بعدها «بطلب».
const ReturnTripMax = 2 * time.Hour

// LoadTenures **ولايات السائق على الطلبات التي تمسّ المدى** — من سجلّ الطلب لا من حامله الآن.
//
// **من الانتقالات الحقيقيّة وحدَها** (`from_status <> to_status`): سطرٌ حالُه ما تغيّرت ملاحظةٌ
// (تعديلُ الطلب، صنفٌ غيرُ متوفّر، مبلغُ الإحضار، تحويلُ المتجر…) لا يبدأ ولايةً ولا ينهيها.
//
// البداية: انتقالٌ إلى `assigned` باسمه. والنهاية أول انتقالٍ بعدها يُخرج الطلب من يده:
// رجوعٌ للطابور (تَرَكه، أو استُرجع منه، أو أعادته العمليّات)، **أو «تعذّر» عند المتجر**
// (`→ accepted` باسمه — `orders/merchant_blocked.go`)، أو إسنادٌ **لسائقٍ آخرَ معلوم**، أو نهايةُ
// الطلب. **و`at_pickup → assigned` بلا سائق** (تحويلُ المتجر — `server/order_transfer.go`،
// `orders/external_store.go`) **لا ينهيها**: السائقُ نفسُه يكمل للمتجر الجديد.
//
// **والفاشلُ بمشوار إرجاع** يمتدّ حتى «سلّمت البضاعة» (أو «أُعيدت للمتجر» في المسار القديم)،
// ومفتوحُه يُقصّ بـ`ReturnTripMax`.
//
// **والطلباتُ المرشّحة من فهرسين** لا من تاريخ السائق كلّه: سطورُه في السجلّ منذ يومٍ قبل
// المدى (`order_events_driver_idx`) والطلباتُ المفتوحة بيده (`orders_driver_idx`) — فلا يكبر
// الاستعلامُ مع عمره في المنصّة وهو يُنادى كلّ ثلاثين ثانية.
//
// **والطلبُ بلا سطر إسنادٍ باسمه** (قبل ٠١٧٢) يُقرأ كما كان: من حامله الحالي.
func LoadTenures(ctx context.Context, db DB, driverID string, from, to, now time.Time) ([]Window, error) {
	rows, err := db.Query(ctx, `
		-- **وطلبُ الواتساب باسم متجره المكتوب** (قرارُ المالك ٢٠٢٦-١١-٠٥) — نافذةُ طلبٍ كغيرها.
		WITH cand AS (
		     SELECT e.order_id FROM order_events e
		      WHERE e.driver_id = $1::uuid AND e.created_at >= $2::timestamptz - interval '24 hours'
		        AND e.created_at < $3
		     UNION
		     SELECT o.id FROM orders o WHERE o.driver_id = $1::uuid AND o.closed_at IS NULL
		)
		SELECT o.id::text, o.number, COALESCE(m.name, o.pickup_name, ''), a.created_at,
		       COALESCE(pk.created_at,
		                CASE WHEN o.driver_id = $1::uuid AND o.picked_up_at >= a.created_at
		                      AND (en.created_at IS NULL OR o.picked_up_at <= en.created_at)
		                     THEN o.picked_up_at END),
		       en.created_at, COALESCE(en.to_status, ''),
		       COALESCE(o.driver_id = $1::uuid, false), o.status, o.closed_at, o.delivered_at,
		       o.returned_at, o.return_to IS NOT NULL, o.goods_handed_at
		  FROM cand c
		  JOIN order_events a ON a.order_id = c.order_id
		  JOIN orders o ON o.id = a.order_id
		  LEFT JOIN merchants m ON m.id = o.merchant_id
		  LEFT JOIN LATERAL (
		       SELECT x.id, x.created_at, x.to_status FROM order_events x
		        WHERE x.order_id = a.order_id AND x.id > a.id AND x.from_status <> x.to_status
		          AND (x.to_status IN ('dispatching','delivered','failed','cancelled','rejected','refunded')
		               OR (x.to_status = 'accepted' AND x.driver_id = $1::uuid)
		               OR (x.to_status = 'assigned' AND x.driver_id IS NOT NULL AND x.driver_id <> $1::uuid))
		        ORDER BY x.id LIMIT 1) en ON true
		  LEFT JOIN LATERAL (
		       SELECT x.created_at FROM order_events x
		        WHERE x.order_id = a.order_id AND x.id > a.id AND x.to_status = 'picked_up'
		          AND x.from_status <> x.to_status
		          AND (en.id IS NULL OR x.id < en.id)
		        ORDER BY x.id LIMIT 1) pk ON true
		 WHERE a.driver_id = $1::uuid AND a.to_status = 'assigned' AND a.from_status <> a.to_status
		   AND a.created_at < $3
		   AND COALESCE(en.created_at, o.goods_handed_at, o.returned_at, o.delivered_at, o.closed_at, $3)
		       + CASE WHEN o.return_to IS NOT NULL THEN interval '2 hours' ELSE interval '0' END > $2
		UNION ALL
		SELECT o.id::text, o.number, COALESCE(m.name, o.pickup_name, ''),
		       COALESCE(o.accepted_at, o.dispatched_at, o.picked_up_at), o.picked_up_at,
		       NULL::timestamptz, '', true, o.status, o.closed_at, o.delivered_at,
		       o.returned_at, o.return_to IS NOT NULL, o.goods_handed_at
		  FROM orders o
		  LEFT JOIN merchants m ON m.id = o.merchant_id
		 WHERE o.driver_id = $1::uuid
		   AND o.created_at >= $2::timestamptz - interval '7 days'
		   AND NOT EXISTS (SELECT 1 FROM order_events e
		                    WHERE e.order_id = o.id AND e.driver_id = $1::uuid AND e.to_status = 'assigned'
		                      AND e.from_status <> e.to_status)
		   AND COALESCE(o.picked_up_at, o.accepted_at, o.dispatched_at, o.created_at) < $3
		   AND COALESCE(o.delivered_at, o.returned_at, o.closed_at, $3) > $2`, driverID, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Window
	for rows.Next() {
		var w Window
		var start *time.Time
		var endAt, closedAt, deliveredAt, returnedAt, handedAt *time.Time
		var endStatus, status string
		var holds, returnTrip bool
		if err := rows.Scan(&w.OrderID, &w.Number, &w.Store, &start, &w.PickedUpAt, &endAt, &endStatus,
			&holds, &status, &closedAt, &deliveredAt, &returnedAt, &returnTrip, &handedAt); err != nil {
			return nil, err
		}
		w.AssignedAt = start
		switch {
		case endStatus != "":
			w.EndAt, w.EndKind = endAt, endKindOf(endStatus)
		case holds && closedAt != nil:
			e := closedAt
			if deliveredAt != nil {
				e = deliveredAt
			}
			w.EndAt, w.EndKind = e, endKindOf(status)
		case holds:
			// ما زال معه.
		default:
			// **خرج من يده بلا سطرٍ يُقرأ** — ولايةٌ بطول صفر خيرٌ من ولايةٍ مفتوحةٍ للأبد.
			w.EndAt, w.EndKind = start, "other"
		}
		// **ووقتُ ما أنهاها** قبل أن يمدّها مشوارُ الإرجاع — منه يُعرف بأيّ دوامٍ انتهت.
		w.EndEventAt = w.EndAt
		// **ومشوار إرجاع البضاعة جزءٌ من الطلب** (قرار المالك ٢٠٢٦-١٠-٠٣).
		if w.EndKind == "failed" && w.EndAt != nil {
			back := handedAt
			if back == nil && returnedAt != nil {
				back = returnedAt
			}
			limit := w.EndAt.Add(ReturnTripMax)
			switch {
			case back != nil && back.After(*w.EndAt):
				if back.After(limit) {
					back = &limit
				}
				w.EndAt, w.EndKind = back, "returned"
			case back == nil && returnTrip:
				if now.Before(limit) {
					w.EndAt = nil // المشوار قائم
				} else {
					w.EndAt = &limit
				}
				w.EndKind = "returned"
			}
		}
		out = append(out, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return mergeTenures(out), nil
}

// endKindOf نوع النهاية من حالة الطلب.
func endKindOf(status string) string {
	switch status {
	case "delivered", "refunded":
		return "delivered"
	case "failed":
		return "failed"
	case "cancelled", "rejected":
		return "cancelled"
	case "dispatching", "assigned":
		return "released"
	case "accepted":
		// **«تعذّر» عند المتجر** — عاد الطلبُ للمكتب وخرج من يده (`orders/merchant_blocked.go`).
		return "store_failed"
	}
	return "other"
}

// mergeTenures يدمج ولايتين متداخلتين للطلب نفسه (سطرا إسنادٍ له بلا خروجٍ بينهما)
// — فلا يُقسم مقطعٌ على طلبٍ واحدٍ مرّتين. ويرتّب بالبداية.
func mergeTenures(ws []Window) []Window {
	sort.SliceStable(ws, func(i, j int) bool {
		if ws[i].OrderID != ws[j].OrderID {
			return ws[i].OrderID < ws[j].OrderID
		}
		return timeOr(ws[i].AssignedAt).Before(timeOr(ws[j].AssignedAt))
	})
	out := ws[:0]
	for _, w := range ws {
		if n := len(out); n > 0 && out[n-1].OrderID == w.OrderID {
			p := &out[n-1]
			if p.EndAt == nil || !timeOr(w.AssignedAt).After(*p.EndAt) {
				if p.EndAt != nil && (w.EndAt == nil || w.EndAt.After(*p.EndAt)) {
					p.EndAt, p.EndKind, p.EndEventAt = w.EndAt, w.EndKind, w.EndEventAt
				}
				if p.PickedUpAt == nil {
					p.PickedUpAt = w.PickedUpAt
				}
				continue
			}
		}
		out = append(out, w)
	}
	sort.SliceStable(out, func(i, j int) bool { return timeOr(out[i].AssignedAt).Before(timeOr(out[j].AssignedAt)) })
	return out
}

func timeOr(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// DriverDay تحليل سائق ليوم حساب (من ساعة فتح الدوام لمثلها).
func DriverDay(ctx context.Context, db DB, driverID, day string, stores []Store, now time.Time) (Result, []Window, error) {
	from, to, err := AccountDayBounds(ctx, db, day)
	if err != nil {
		return Result{}, nil, err
	}
	pts, err := LoadPoints(ctx, db, driverID, from, to)
	if err != nil {
		return Result{}, nil, err
	}
	// ولاية طلب بدأت قبل اليوم تحتاج نقطتها لتصنيف أول اليوم — يكفي الطلب نفسه.
	wins, err := LoadTenures(ctx, db, driverID, from, to, now)
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

// Overview سائقو يوم الحساب الذين لهم أثر، الأخطر أولا.
func Overview(ctx context.Context, db DB, day string, alertKm float64, now time.Time) ([]Row, error) {
	from, to, err := AccountDayBounds(ctx, db, day)
	if err != nil {
		return nil, err
	}
	// اليوم الحالي: يدخل كل سائق على الدوام ولو بلا أثر بعد، ليرى المالك كل العاملين.
	today, err := AccountDay(ctx, db, now)
	if err != nil {
		return nil, err
	}
	isToday := day == today
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
		out[i].Alert = res.Alert(alertKm)
		if out[i].Alert {
			out[i].Score += 300
		}
	}
	sort.SliceStable(out, func(a, b int) bool { return out[a].Score > out[b].Score })
	return out, nil
}
