package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

// ══════════════════════════════════════════════════════════════════════
// **الخريطة من حركة السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣) — لوحة «الخريطة من السائقين»
// ══════════════════════════════════════════════════════════════════════
//
// خريطةُ كثافة نقاط السائقين (بلا هويّة) · شوارعُ مقترحة (بعيدةٌ عن كلّ طريقٍ وتتكرّر أياماً) · موافقةٌ
// أو رفض · وتصديرُ الموافَق عليه ملفَّ OSM/GeoJSON لبناء الخريطة (الخطوة ٤، لم تُنفَّذ).
// القراءةُ لمن يفتح خريطة العمليات، والقرارُ والتشغيلُ والتصديرُ لمن يملك التغطية.
// الحسابُ في `internal/maplearn` — وبالأمر `cmd/maplearn` أو الجولةِ اليوميّة، لا مع كلّ طلب.

var (
	errCandidateDecided = httpx.NewError(http.StatusConflict, "candidate_decided", "errors.conflict")
	errCandidateFormat  = httpx.NewError(http.StatusUnprocessableEntity, "bad_format", "errors.validation")
	errMapLearnBusy     = httpx.NewError(http.StatusConflict, "map_learning_busy", "errors.conflict")
	errMapLearnNoRoads  = httpx.NewError(http.StatusConflict, "map_learning_no_roads", "errors.conflict")
)

const maxStreetNameLen = 80

// roadFinder مقياسُ البُعد عن الطرق من إعدادات التوجيه (nil = لا محرّك).
func (s *Server) roadFinder() maplearn.RoadFinder {
	return maplearn.NewRoadFinder(s.cfg.RoutingEngine, s.cfg.OSRMURL, s.cfg.ValhallaURL)
}

// RunMapLearning **جولةٌ يوميّة** — الأولى بعد ساعةٍ من الإقلاع لا عنده.
func (s *Server) RunMapLearning(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(time.Hour)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.mapLearningOnce(ctx)
			timer.Reset(interval)
		}
	}
}

func (s *Server) mapLearningOnce(ctx context.Context) {
	// الأماكنُ بلا اسمٍ لا تحتاج محرّكَ خرائط — تعمل ولو غاب.
	s.mapPlacesOnce(ctx)
	rf := s.roadFinder()
	if rf == nil {
		return
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	res, err := maplearn.Run(runCtx, s.pg, rf, maplearn.DefaultOptions())
	if err != nil {
		s.logger.Warn("الخريطة من السائقين: تعثّرت الجولة", "error", err)
		return
	}
	s.logger.Info("الخريطة من السائقين: انتهت الجولة", "measured", res.Measured,
		"far", res.FarPoints, "candidates", res.Candidates)
}

type streetCandidate struct {
	ID       int64           `json:"id"`
	Status   string          `json:"status"`
	Name     string          `json:"name"`
	LengthM  float64         `json:"length_m"`
	Points   int             `json:"points"`
	Days     int             `json:"days"`
	FirstDay string          `json:"first_day"`
	LastDay  string          `json:"last_day"`
	Geometry json.RawMessage `json:"geometry"`
}

// handleStreetCandidates GET /ops-map/street-candidates?status=pending|approved|rejected
func (s *Server) handleStreetCandidates(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "approved" && status != "rejected" {
		status = "pending"
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, status, name, length_m, points, days, first_day::text, last_day::text,
		       ST_AsGeoJSON(line::geometry)::text
		  FROM map_street_candidates WHERE status = $1
		 ORDER BY days DESC, length_m DESC, id LIMIT 500`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []streetCandidate{}
	for rows.Next() {
		var c streetCandidate
		var g string
		if err := rows.Scan(&c.ID, &c.Status, &c.Name, &c.LengthM, &c.Points, &c.Days, &c.FirstDay, &c.LastDay, &g); err != nil {
			s.respondErr(w, err)
			return
		}
		c.Geometry = json.RawMessage(g)
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"candidates": out, "count": len(out)})
}

// handleStreetCandidateDecide POST /ops-map/street-candidates/{id}/decide  {approve, name}
func (s *Server) handleStreetCandidateDecide(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Approve bool   `json:"approve"`
		Name    string `json:"name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	name := strings.TrimSpace(req.Name)
	if len([]rune(name)) > maxStreetNameLen {
		name = string([]rune(name)[:maxStreetNameLen])
	}
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	var got int64
	err = s.pg.QueryRow(r.Context(), `
		UPDATE map_street_candidates
		   SET status = $2, name = CASE WHEN $2 = 'approved' THEN $3 ELSE name END,
		       decided_by = $4::uuid, decided_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'pending' RETURNING id`, id, status, name, userIDFrom(r)).Scan(&got)
	if err == pgx.ErrNoRows {
		var exists bool
		_ = s.pg.QueryRow(r.Context(), `SELECT EXISTS (SELECT 1 FROM map_street_candidates WHERE id = $1)`, id).Scan(&exists)
		if !exists {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errCandidateDecided)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status})
}

// handleMapLearningSummary GET /ops-map/street-candidates/summary — أرقامُ الصفحة وآخر جولة.
func (s *Server) handleMapLearningSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var pts, unmeasured, far int
	var pending, approved, rejected int
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*)::int, count(*) FILTER (WHERE road_dist_m IS NULL)::int,
		       count(*) FILTER (WHERE road_dist_m >= 18)::int FROM map_trace_points`).Scan(&pts, &unmeasured, &far); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE status = 'pending')::int, count(*) FILTER (WHERE status = 'approved')::int,
		       count(*) FILTER (WHERE status = 'rejected')::int FROM map_street_candidates`).Scan(&pending, &approved, &rejected); err != nil {
		s.respondErr(w, err)
		return
	}
	out := map[string]any{
		"points": pts, "unmeasured": unmeasured, "far_points": far,
		"pending": pending, "approved": approved, "rejected": rejected,
		"running": maplearn.Busy(), "can_measure": s.roadFinder() != nil,
	}
	var at *time.Time
	var errTxt string
	var measured, cands int
	if err := s.pg.QueryRow(ctx, `
		SELECT finished_at, measured, candidates, error FROM map_street_runs
		 WHERE finished_at IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&at, &measured, &cands, &errTxt); err == nil {
		out["last_run"] = map[string]any{"at": at, "measured": measured, "candidates": cands, "failed": errTxt != ""}
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleMapLearningHeat GET /ops-map/street-candidates/heat?days=30&far=1 — خلايا ~٢٠ م بعددها.
func (s *Server) handleMapLearningHeat(w http.ResponseWriter, r *http.Request) {
	days, _ := strconv.Atoi(r.URL.Query().Get("days"))
	if days < 1 || days > 90 {
		days = 30
	}
	farOnly := r.URL.Query().Get("far") == "1"
	rows, err := s.pg.Query(r.Context(), `
		SELECT ST_X(g), ST_Y(g), c FROM (
		  SELECT ST_SnapToGrid(point::geometry, 0.0002) AS g, count(*)::int AS c
		    FROM map_trace_points
		   WHERE day >= (now() AT TIME ZONE 'Asia/Damascus')::date - $1::int
		     AND (NOT $2 OR road_dist_m >= 18)
		   GROUP BY 1 ORDER BY 2 DESC LIMIT 40000) t`, days, farOnly)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	feats := []map[string]any{}
	for rows.Next() {
		var lng, lat float64
		var c int
		if err := rows.Scan(&lng, &lat, &c); err != nil {
			s.respondErr(w, err)
			return
		}
		feats = append(feats, map[string]any{
			"type":       "Feature",
			"geometry":   map[string]any{"type": "Point", "coordinates": []float64{lng, lat}},
			"properties": map[string]any{"w": c},
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"type": "FeatureCollection", "features": feats})
}

// handleMapLearningRun POST /ops-map/street-candidates/run — جولةٌ بالخلفيّة.
func (s *Server) handleMapLearningRun(w http.ResponseWriter, r *http.Request) {
	rf := s.roadFinder()
	if rf == nil {
		s.respondErr(w, errMapLearnNoRoads)
		return
	}
	if maplearn.Busy() {
		s.respondErr(w, errMapLearnBusy)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		if _, err := maplearn.Run(ctx, s.pg, rf, maplearn.DefaultOptions()); err != nil {
			s.logger.Warn("الخريطة من السائقين: تعثّرت الجولة", "error", err)
		}
	}()
	httpx.JSON(w, http.StatusAccepted, map[string]any{"started": true})
}

// handleStreetCandidatesExport GET /ops-map/street-candidates/export?format=osm|geojson — الموافَقُ عليه وحدَه.
func (s *Server) handleStreetCandidatesExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "osm" && format != "geojson" {
		s.respondErr(w, errCandidateFormat)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, name, ST_AsGeoJSON(line::geometry)::text
		  FROM map_street_candidates WHERE status = 'approved' ORDER BY id`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	lines := []maplearn.Line{}
	for rows.Next() {
		var l maplearn.Line
		var g string
		if err := rows.Scan(&l.ID, &l.Name, &g); err != nil {
			s.respondErr(w, err)
			return
		}
		var geo struct {
			Coordinates [][2]float64 `json:"coordinates"`
		}
		if err := json.Unmarshal([]byte(g), &geo); err != nil {
			s.respondErr(w, err)
			return
		}
		l.Coords = geo.Coordinates
		lines = append(lines, l)
	}
	if format == "osm" {
		w.Header().Set("Content-Type", "application/xml; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-streets.osm"`)
		_, _ = w.Write([]byte(maplearn.OSM(lines)))
		return
	}
	b, err := maplearn.GeoJSON(lines)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-streets.geojson"`)
	_, _ = w.Write(b)
}
