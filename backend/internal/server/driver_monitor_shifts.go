package server

// ══════════════════════════════════════════════════════════════════════
// **الدوامات بيوم السائق** (قرار المالك ٢٠٢٦-١١-٠٥) — صفحة المراقبة
// ══════════════════════════════════════════════════════════════════════
//
// كل دوامٍ مسّ اليوم: متى فُتح ومتى سُكّر ومين سكّره، وأرقام التطبيق من **التحليل نفسه**
// اللي بيكتب مسافة كل طلب (`drivermonitor.MeasureShift`) — فما في رقمين بيختلفوا.

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// monitorShift دوامٌ بيوم السائق.
type monitorShift struct {
	ID          string     `json:"id"`
	OpenedAt    time.Time  `json:"opened_at"`
	ClosedAt    *time.Time `json:"closed_at"`
	CloseReason string     `json:"close_reason"`
	OpenedBuild *int       `json:"opened_build"`
	Migrated    bool       `json:"migrated"`
	// أرقام التطبيق (متر) — القيادة الفعلية وتقسيمها.
	DriveM      int  `json:"drive_m"`
	OrdersM     int  `json:"orders_m"`
	NoOrderM    int  `json:"no_order_m"`
	NotDrivingM int  `json:"not_driving_m"`
	GapMin      int  `json:"gap_min"`
	OrdersN     int  `json:"orders_n"`
	Final       bool `json:"final"`
	// OpenHours كم ساعة له مفتوحاً (للمفتوح وحده).
	OpenHours int `json:"open_hours,omitempty"`
}

// monitorShifts دوامات السائق التي تمسّ [from, to) — بأرقامها محسوبةً الآن.
func (s *Server) monitorShifts(ctx context.Context, driverID string, from, to, now time.Time) ([]monitorShift, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, opened_at, closed_at, COALESCE(close_reason, ''), opened_build, migrated, final
		  FROM driver_shifts
		 WHERE driver_id = $1::uuid AND opened_at < $3 AND COALESCE(closed_at, $4) >= $2
		 ORDER BY opened_at`, driverID, from, to, now)
	if err != nil {
		return nil, err
	}
	var out []monitorShift
	for rows.Next() {
		var m monitorShift
		if err := rows.Scan(&m.ID, &m.OpenedAt, &m.ClosedAt, &m.CloseReason, &m.OpenedBuild, &m.Migrated, &m.Final); err != nil {
			rows.Close()
			return nil, err
		}
		out = append(out, m)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		km, _, err := drivermonitor.MeasureShift(ctx, s.pg, drivermonitor.ShiftSpan{
			ID: out[i].ID, DriverID: driverID, OpenedAt: out[i].OpenedAt, ClosedAt: out[i].ClosedAt,
		}, now)
		if err != nil {
			return nil, err
		}
		out[i].DriveM, out[i].OrdersM, out[i].NoOrderM = km.DriveM, km.OrdersM, km.NoOrderM
		out[i].NotDrivingM, out[i].GapMin, out[i].OrdersN = km.NotDrivingM, km.GapMin, km.OrdersN
		if out[i].ClosedAt == nil {
			out[i].OpenHours = int(now.Sub(out[i].OpenedAt) / time.Hour)
		}
	}
	if out == nil {
		out = []monitorShift{}
	}
	return out, nil
}
