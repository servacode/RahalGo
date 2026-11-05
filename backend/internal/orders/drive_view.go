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
	// Final ثبت الرقم (بعد ساعتين وعشر دقائق من سكّر الدوام).
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
	rows, err := s.db.Query(ctx, `
		SELECT d.order_id::text, d.driver_id::text, COALESCE(u.full_name, ''), d.started_at,
		       d.picked_up_at, d.ended_at, d.end_kind, d.to_pickup_m, d.with_order_m, d.shared,
		       d.gap_min, d.final
		  FROM order_drives d
		  LEFT JOIN users u ON u.id = d.driver_id
		 WHERE d.order_id = ANY($1::uuid[])
		 ORDER BY d.started_at`, ids)
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
