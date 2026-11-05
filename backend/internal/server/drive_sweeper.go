package server

// ══════════════════════════════════════════════════════════════════════
// **كانسُ المسافات — كلّ دقيقة** (قرار المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «ومع كل طلب يكتب المسافة المقطوعة من بداية استلام الطلب الى انتهاء الطلب.»
//
// لكلّ دوامٍ لم يثبت رقمُه: تحليلٌ واحد (`drivermonitor.MeasureShift`) → أرقام الدوام
// في `driver_shifts` ومسافة كلّ طلبٍ في `order_drives`. **ويُعاد** حين يتحرّك شيء:
// دوامٌ جديد أو مسكَّر، أو سطرٌ في سجلّ طلبٍ للسائق، أو طلبٌ بيده (كلّ دقيقة — فالرقم
// «لحد هلق» حيّ)، أو مرّت ربعُ ساعة. **ويثبت** بعد ساعتين وعشر دقائق من السكّر —
// فالنقاط المتأخّرة (حتى ساعتين) تدخل، وبعدها ما يتغيّر شيء ولو قُلّم الأثر.
//
// **ولا مال**: لا قيدَ ولا محفظة — أرقامُ مسافةٍ فقط.

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// driveSweepBatch أقصى دوامات في الجولة.
const driveSweepBatch = 50

// RunDriveSweeper **كلَّ دقيقة.**
func (s *Server) RunDriveSweeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			s.SweepDrives(ctx, t)
		}
	}
}

// SweepDrives **جولةٌ واحدة** — كم دواماً قِيس. `now` يُمرَّر (والاختبار يقدّمه).
func (s *Server) SweepDrives(ctx context.Context, now time.Time) int {
	rows, err := s.pg.Query(ctx, `
		SELECT s.id::text, s.driver_id::text, s.opened_at, s.closed_at
		  FROM driver_shifts s
		 WHERE NOT s.final AND s.opened_at <= $1
		   AND (s.computed_at IS NULL
		        OR (s.closed_at IS NOT NULL AND (s.computed_at < s.closed_at
		                                         OR s.closed_at <= $2
		                                         OR s.computed_at < $1 - interval '10 minutes'))
		        OR (s.closed_at IS NULL AND s.computed_at < $1 - interval '15 minutes')
		        OR (s.closed_at IS NULL AND s.computed_at < $1 - interval '1 minute'
		            AND EXISTS (SELECT 1 FROM orders o
		                         WHERE o.driver_id = s.driver_id
		                           AND (o.closed_at IS NULL
		                                OR (o.return_to IS NOT NULL AND o.goods_handed_at IS NULL
		                                    AND o.closed_at > $1 - interval '2 hours'))))
		        OR EXISTS (SELECT 1 FROM order_events e
		                    WHERE e.driver_id = s.driver_id AND e.created_at > s.computed_at
		                      AND e.created_at <= $1)
		        OR EXISTS (SELECT 1 FROM orders o
		                    WHERE o.driver_id = s.driver_id AND o.goods_handed_at > s.computed_at
		                      AND o.goods_handed_at <= $1))
		 ORDER BY s.computed_at NULLS FIRST
		 LIMIT $3`, now, now.Add(-drivermonitor.FinalAfter), driveSweepBatch)
	if err != nil {
		s.logger.Warn("كانس المسافات: تعذّرت قراءة الدوامات", "error", err)
		return 0
	}
	var spans []drivermonitor.ShiftSpan
	for rows.Next() {
		var sp drivermonitor.ShiftSpan
		if err := rows.Scan(&sp.ID, &sp.DriverID, &sp.OpenedAt, &sp.ClosedAt); err != nil {
			rows.Close()
			s.logger.Warn("كانس المسافات: صفّ دوام", "error", err)
			return 0
		}
		spans = append(spans, sp)
	}
	rows.Close()
	n := 0
	for _, sp := range spans {
		if err := s.measureShift(ctx, sp, now); err != nil {
			s.logger.Warn("كانس المسافات: تعذّر قياس دوام", "shift", sp.ID, "driver", sp.DriverID, "error", err)
			continue
		}
		n++
	}
	return n
}

// measureShift يقيس دواماً ويكتب أرقامه ومسافة طلباته في معاملة واحدة.
func (s *Server) measureShift(ctx context.Context, sp drivermonitor.ShiftSpan, now time.Time) error {
	km, drives, err := drivermonitor.MeasureShift(ctx, s.pg, sp, now)
	if err != nil {
		return err
	}
	final := sp.ClosedAt != nil && !sp.ClosedAt.After(now.Add(-drivermonitor.FinalAfter))
	// **أيُّ طلبٍ انتهى منذ آخر حساب؟** — منه وحده يُنبَّه تطبيقُ السائق (عدّادُه يزيد)،
	// لا مع كلّ دقيقةٍ وطلبٌ بيده.
	newlyEnded := false
	if len(drives) > 0 {
		ids := make([]string, 0, len(drives))
		for _, d := range drives {
			if d.EndedAt != nil {
				ids = append(ids, d.OrderID)
			}
		}
		if len(ids) > 0 {
			var known int
			if err := s.pg.QueryRow(ctx, `
				SELECT count(*)::int FROM order_drives
				 WHERE driver_id = $1::uuid AND order_id = ANY($2::uuid[]) AND ended_at IS NOT NULL`,
				sp.DriverID, ids).Scan(&known); err == nil {
				newlyEnded = known < len(ids)
			}
		}
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		UPDATE driver_shifts
		   SET drive_m = $2, orders_m = $3, no_order_m = $4, not_driving_m = $5, gap_min = $6,
		       orders_n = $7, points = $8, computed_at = $9, final = $10
		 WHERE id = $1::uuid AND NOT final`,
		sp.ID, km.DriveM, km.OrdersM, km.NoOrderM, km.NotDrivingM, km.GapMin, km.OrdersN, km.Points,
		now, final); err != nil {
		return err
	}
	b := &pgx.Batch{}
	for _, d := range drives {
		b.Queue(`
			INSERT INTO order_drives (order_id, driver_id, shift_id, started_at, picked_up_at, ended_at,
			                          end_kind, to_pickup_m, with_order_m, shared, gap_min, points,
			                          computed_at, final)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
			ON CONFLICT (order_id, driver_id) DO UPDATE SET
			       shift_id = EXCLUDED.shift_id, started_at = EXCLUDED.started_at,
			       picked_up_at = EXCLUDED.picked_up_at, ended_at = EXCLUDED.ended_at,
			       end_kind = EXCLUDED.end_kind, to_pickup_m = EXCLUDED.to_pickup_m,
			       with_order_m = EXCLUDED.with_order_m, shared = EXCLUDED.shared,
			       gap_min = EXCLUDED.gap_min, points = EXCLUDED.points,
			       computed_at = EXCLUDED.computed_at, final = EXCLUDED.final
			 WHERE NOT order_drives.final`,
			d.OrderID, sp.DriverID, sp.ID, d.StartedAt, d.PickedUpAt, d.EndedAt, d.EndKind,
			d.ToPickupM, d.WithOrderM, d.Shared, d.GapMin, d.Points, now, final && d.EndedAt != nil)
	}
	if len(drives) > 0 {
		br := tx.SendBatch(ctx, b)
		for range drives {
			if _, err := br.Exec(); err != nil {
				_ = br.Close()
				return err
			}
		}
		if err := br.Close(); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	// **وعدّادُ السائق يتحدّث** حين ينتهي طلب — التطبيقُ يعيد قراءة `driver/me` بإشارةٍ واحدة.
	if s.hub != nil && newlyEnded {
		s.hub.Publish("driver:"+sp.DriverID, map[string]any{"type": "me"})
	}
	return nil
}
