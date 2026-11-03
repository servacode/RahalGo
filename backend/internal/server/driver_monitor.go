package server

// ══════════════════════════════════════════════════════════════════════
// **مراقبة السائقين** (قرار المالك ٢٠٢٦-١١-٠٣) — الكتابة والأبواب
// ══════════════════════════════════════════════════════════════════════
//
// الكتابة: نفس معالجَي الموقع (المفرد والدفعة) بعد ما ينجحوا بحفظ الموضع.
// **وتعثّرها ما بيسقط حفظ الموضع** — الموضع هو المطلوب كل لحظة والأثر للمراجعة.
// الأبواب: `drivers.monitor` وحدها. التحليل في `internal/drivermonitor`.

import (
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var errMonitorDriver = httpx.NewError(http.StatusNotFound, "not_found", "errors.not_found")

// trailPoints يحوّل نقاط الدفعة إلى نقاط الأثر.
func trailPoints(pts []trackPoint) []drivermonitor.Point {
	out := make([]drivermonitor.Point, 0, len(pts))
	for _, p := range pts {
		out = append(out, drivermonitor.Point{At: p.At, Lat: p.Lat, Lng: p.Lng,
			AccuracyM: p.AccuracyM, SpeedMps: p.SpeedMps, Mocked: p.Mocked != nil && *p.Mocked})
	}
	return out
}

// recordTrail يكتب الأثر اليومي ويقلّم القديم مرة بالساعة. لا يُسقط شيئا.
func (s *Server) recordTrail(r *http.Request, driverID string, pts []drivermonitor.Point) {
	ctx := r.Context()
	if _, err := drivermonitor.Record(ctx, s.pg, driverID, pts); err != nil {
		s.logger.Warn("مراقبة السائقين: تعذّر كتابةُ الأثر", "driver", driverID, "error", err)
		return
	}
	if drivermonitor.PruneDue(time.Now()) {
		days := int(s.settings.GetInt(ctx, "drivers.trail_retention_days"))
		if _, err := drivermonitor.Prune(ctx, s.pg, days); err != nil {
			s.logger.Warn("مراقبة السائقين: تعذّر التقليم", "error", err)
		}
	}
}

// monitorDay يقرأ ?day= (افتراضيه اليوم الدمشقيّ).
func monitorDay(r *http.Request) (string, bool) {
	day := r.URL.Query().Get("day")
	if day == "" {
		return drivermonitor.Today(time.Now()), true
	}
	if _, _, err := drivermonitor.DayBounds(day); err != nil {
		return "", false
	}
	return day, true
}

// handleDriverMonitorList GET /driver-monitor — سائقو اليوم، الأخطر أولا.
func (s *Server) handleDriverMonitorList(w http.ResponseWriter, r *http.Request) {
	day, ok := monitorDay(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	alert := float64(s.settings.GetInt(r.Context(), "drivers.monitor_km_alert"))
	rows, err := drivermonitor.Overview(r.Context(), s.pg, day, alert, time.Now())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if rows == nil {
		rows = []drivermonitor.Row{}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day":            day,
		"km_alert":       alert,
		"retention_days": s.settings.GetInt(r.Context(), "drivers.trail_retention_days"),
		"drivers":        rows,
	})
}

// handleDriverMonitorDay GET /driver-monitor/{id} — يوم سائق: مسار وقصة وأرقام.
func (s *Server) handleDriverMonitorDay(w http.ResponseWriter, r *http.Request) {
	day, ok := monitorDay(r)
	if !ok {
		s.respondErr(w, errValidation)
		return
	}
	id := chi.URLParam(r, "id")
	var name, phone string
	if err := s.pg.QueryRow(r.Context(), `
		SELECT full_name, phone FROM users u
		 WHERE u.id::text = $1 AND EXISTS (
		   SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_code = 'driver')`, id).
		Scan(&name, &phone); err != nil {
		s.respondErr(w, errMonitorDriver)
		return
	}
	stores, err := drivermonitor.LoadStores(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	res, wins, err := drivermonitor.DriverDay(r.Context(), s.pg, id, day, stores, time.Now())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	alert := float64(s.settings.GetInt(r.Context(), "drivers.monitor_km_alert"))
	type order struct {
		ID         string     `json:"id"`
		Number     int64      `json:"number"`
		Store      string     `json:"store"`
		AssignedAt *time.Time `json:"assigned_at"`
		PickedUpAt *time.Time `json:"picked_up_at"`
		EndAt      *time.Time `json:"end_at"`
	}
	orders := make([]order, 0, len(wins))
	for _, o := range wins {
		orders = append(orders, order{o.OrderID, o.Number, o.Store, o.AssignedAt, o.PickedUpAt, o.EndAt})
	}
	// **متاجر حول المسار بأسمائها** — تُرسم على الخريطة باسمها (طلب المالك ٢٠٢٦-١١-٠٣).
	type near struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	nearStores := []near{}
	const pad = 0.004 // ~٤٤٠ م
	minLat, maxLat, minLng, maxLng := 90.0, -90.0, 180.0, -180.0
	have := false
	for _, part := range res.Path {
		for _, c := range part.Coords {
			have = true
			minLat, maxLat = math.Min(minLat, c[0]), math.Max(maxLat, c[0])
			minLng, maxLng = math.Min(minLng, c[1]), math.Max(maxLng, c[1])
		}
	}
	if have {
		for _, st := range stores {
			if st.Lat >= minLat-pad && st.Lat <= maxLat+pad && st.Lng >= minLng-pad && st.Lng <= maxLng+pad {
				nearStores = append(nearStores, near{st.ID, st.Name, st.Lat, st.Lng})
			}
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day":      day,
		"stores":   nearStores,
		"driver":   map[string]any{"id": id, "name": name, "phone": phone},
		"km_alert": alert,
		"alert":    alert > 0 && res.Summary.KmNoOrder >= alert,
		"summary":  res.Summary,
		"path":     res.Path,
		"events":   res.Events,
		"orders":   orders,
	})
}
