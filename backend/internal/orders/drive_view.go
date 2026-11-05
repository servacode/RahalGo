package orders

// ══════════════════════════════════════════════════════════════════════
// **مسافةُ الطلب على بطاقته** (قرارُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «ومع كل طلب يكتب المسافة المقطوعة من بداية استلام الطلب الى انتهاء الطلب مثلا المسافة 3
// كيلو.» صفٌّ لكلّ سائقٍ حمل الطلب — يكتبه كانسُ المسافات (`server/drive_sweeper.go`) من
// القيادة الفعليّة، **ويُقرأ هنا كما هو** لبطاقة اللوحة والسجلّ وملفّ الطلب. للمكتب وحده.

import (
	"context"
	"time"
)

// DriveView مسافةُ الطلب لسائقٍ حمله.
type DriveView struct {
	DriverID   string     `json:"driver_id"`
	DriverName string     `json:"driver_name"`
	StartedAt  time.Time  `json:"started_at"`
	PickedUpAt *time.Time `json:"picked_up_at"`
	// EndedAt فارغٌ = الطلبُ لسّا معه، والرقمُ «لحد هلق».
	EndedAt *time.Time `json:"ended_at"`
	EndKind string     `json:"end_kind"`
	// ToPickupM من القبول حتى الاستلام · WithOrderM من الاستلام حتى خرج من يده.
	ToPickupM  int  `json:"to_pickup_m"`
	WithOrderM int  `json:"with_order_m"`
	Shared     bool `json:"shared"`
	GapMin     int  `json:"gap_min"`
	// Final ثبت الرقم — كلُّ قطعه ثبتت (بعد ساعتين وعشر دقائق من نهاية كلٍّ منها).
	Final bool `json:"final"`
}

// FillDrives **يملأ `Drive` لصفحةٍ من الطلبات** — بنداءٍ واحد. وتعذّرُه لا يُسقط القائمة.
func (s *Service) FillDrives(ctx context.Context, list []Order) error {
	if len(list) == 0 {
		return nil
	}
	ids := make([]string, len(list))
	at := make(map[string]int, len(list))
	for i := range list {
		ids[i] = list[i].ID
		at[list[i].ID] = i
	}
	// **وقطعُ الطلب الواحد لسائقه تُجمع** — طلبٌ امتدّ على دوامين (طارئٌ ثمّ فتحٌ جديدٌ وتسليم)
	// مسافتُه مجموعُ قطعه، ونهايتُه آخرُها (والمفتوحةُ تغلب)، ويثبت حين تثبت كلُّها.
	rows, err := s.db.Query(ctx, `
		SELECT d.order_id::text, d.driver_id::text, COALESCE(u.full_name, ''), min(d.started_at),
		       min(d.picked_up_at),
		       CASE WHEN bool_or(d.ended_at IS NULL) THEN NULL ELSE max(d.ended_at) END,
		       (array_agg(d.end_kind ORDER BY d.ended_at DESC NULLS FIRST))[1],
		       sum(d.to_pickup_m)::int, sum(d.with_order_m)::int, bool_or(d.shared),
		       sum(d.gap_min)::int, bool_and(d.final)
		  FROM order_drives d
		  LEFT JOIN users u ON u.id = d.driver_id
		 WHERE d.order_id = ANY($1::uuid[])
		 GROUP BY d.order_id, d.driver_id, u.full_name
		 ORDER BY min(d.started_at)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var oid string
		var v DriveView
		if err := rows.Scan(&oid, &v.DriverID, &v.DriverName, &v.StartedAt, &v.PickedUpAt, &v.EndedAt,
			&v.EndKind, &v.ToPickupM, &v.WithOrderM, &v.Shared, &v.GapMin, &v.Final); err != nil {
			return err
		}
		if i, ok := at[oid]; ok {
			list[i].Drive = append(list[i].Drive, v)
		}
	}
	return rows.Err()
}
