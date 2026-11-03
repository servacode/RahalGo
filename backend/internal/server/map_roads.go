package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/maplearn"
)

// ══════════════════════════════════════════════════════════════════════
// **تعلّم الطرق: اتّجاه واحد وسرعة حقيقية** (قرارُ المالك ٢٠٢٦-١١-٠٣) — الدفعة الثالثة من «الخريطة من السائقين»
// ══════════════════════════════════════════════════════════════════════
//
// أثرُ `driver_trail` يُطابَق على حافّات الطرق دفعةً ليليّة (`internal/maplearn/roadlearn*.go`): شوارعُ مقترحةٌ باتّجاه
// واحد (المدير يوافق فيُصدَّر وسمُ `oneway=yes` لإعادة بناء الخريطة)، وسرعةٌ حقيقيّةٌ لكلّ حافّةٍ وساعة.
// **لا يغيّر التوجيهَ ولا الـETA** — القراءةُ لمن يفتح الخريطة، والتشغيلُ والقرارُ والتصديرُ لمن يملك التغطية.

var errOnewayDecided = httpx.NewError(http.StatusConflict, "oneway_decided", "errors.conflict")

// mapRoadsOnce جولةٌ يوميّة للطرق. بلا Valhalla تُحدَّث المقترحاتُ من المجاميع المخزّنة وحدَها.
func (s *Server) mapRoadsOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 60*time.Minute)
	defer cancel()
	res, err := maplearn.RunRoads(runCtx, s.pg, maplearn.NewMatcher(s.cfg.ValhallaURL), maplearn.DefaultRoadOptions())
	if err != nil {
		s.logger.Warn("تعلّم الطرق: تعثّرت الجولة", "error", err)
		return
	}
	s.logger.Info("تعلّم الطرق: انتهت الجولة", "days", res.Days, "passes", res.Passes, "oneway", res.OneWay)
}

type onewayCandidate struct {
	ID         int64           `json:"id"`
	Status     string          `json:"status"`
	Name       string          `json:"name"`
	WayID      int64           `json:"way_id"`
	MainPasses int             `json:"main_passes"`
	OppPasses  int             `json:"opp_passes"`
	Drivers    int             `json:"drivers"`
	Days       int             `json:"days"`
	Share      float64         `json:"share"`
	LengthM    float64         `json:"length_m"`
	Geometry   json.RawMessage `json:"geometry"`
}

// handleOnewayCandidates GET /ops-map/oneway-candidates?status=pending|approved|rejected
// — المقترحاتُ وأعدادُها.
func (s *Server) handleOnewayCandidates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	if status != "approved" && status != "rejected" {
		status = "pending"
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id, status, name, way_id, main_passes, opp_passes, drivers, days, share,
		       ST_Length(line), ST_AsGeoJSON(line::geometry)::text
		  FROM map_oneway_candidates WHERE status = $1 ORDER BY main_passes DESC, id LIMIT 500`, status)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	cands := []onewayCandidate{}
	for rows.Next() {
		var c onewayCandidate
		var g string
		var share float32
		if err := rows.Scan(&c.ID, &c.Status, &c.Name, &c.WayID, &c.MainPasses, &c.OppPasses, &c.Drivers, &c.Days,
			&share, &c.LengthM, &g); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		c.Share = float64(share)
		c.Geometry = json.RawMessage(g)
		cands = append(cands, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	counts := map[string]int{"pending": 0, "approved": 0, "rejected": 0}
	crows, err := s.pg.Query(ctx, `SELECT status, count(*)::int FROM map_oneway_candidates GROUP BY status`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for crows.Next() {
		var st string
		var n int
		if err := crows.Scan(&st, &n); err != nil {
			crows.Close()
			s.respondErr(w, err)
			return
		}
		counts[st] = n
	}
	crows.Close()
	httpx.JSON(w, http.StatusOK, map[string]any{"candidates": cands, "count": len(cands), "counts": counts})
}

// handleOnewayDecide POST /ops-map/oneway-candidates/{id}/decide  {approve}
func (s *Server) handleOnewayDecide(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Approve bool `json:"approve"`
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
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	ctx := r.Context()
	var got int64
	err = s.pg.QueryRow(ctx, `
		UPDATE map_oneway_candidates SET status = $2, decided_by = $3::uuid, decided_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'pending' RETURNING id`, id, status, userIDFrom(r)).Scan(&got)
	if err == pgx.ErrNoRows {
		var exists bool
		_ = s.pg.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM map_oneway_candidates WHERE id = $1)`, id).Scan(&exists)
		if !exists {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errOnewayDecided)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status})
}

// handleOnewayExport GET /ops-map/oneway-candidates/export?format=osm|geojson — الموافَقُ عليه وحدَه.
func (s *Server) handleOnewayExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format != "osm" && format != "geojson" {
		s.respondErr(w, errCandidateFormat)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT id, way_id, name, ST_AsGeoJSON(line::geometry)::text
		  FROM map_oneway_candidates WHERE status = 'approved' ORDER BY id`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	lines := []maplearn.OnewayLine{}
	for rows.Next() {
		var l maplearn.OnewayLine
		var g string
		if err := rows.Scan(&l.ID, &l.WayID, &l.Name, &g); err != nil {
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
		w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-oneway.osm"`)
		_, _ = w.Write([]byte(maplearn.OnewayOSM(lines)))
		return
	}
	b, err := maplearn.OnewayGeoJSON(lines)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/geo+json")
	w.Header().Set("Content-Disposition", `attachment; filename="rahalgo-oneway.geojson"`)
	_, _ = w.Write(b)
}

// handleRoadSpeeds GET /ops-map/road-speeds?hour=-1|0..23&weekend=|0|1&min=3
// — حافّاتٌ بمتوسّط سرعتها الحقيقية في الخانة المختارة (ساعةٌ بتوقيت دمشق؛ فارغٌ/-١ = كلُّ الساعات)،
// وعلامةُ «بطيئةٌ باستمرار» محسوبةٌ على كلّ الخانات لا المختارةِ وحدَها.
func (s *Server) handleRoadSpeeds(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	hour := -1
	if v, err := strconv.Atoi(q.Get("hour")); err == nil && v >= 0 && v <= 23 {
		hour = v
	}
	weekend := q.Get("weekend")
	if weekend != "0" && weekend != "1" {
		weekend = ""
	}
	minPasses := 3
	if v, err := strconv.Atoi(q.Get("min")); err == nil && v >= 1 && v <= 1000 {
		minPasses = v
	}
	slow, err := maplearn.ReadSlowEdges(r.Context(), s.pg, maplearn.DefaultSlowParams())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	rows, err := s.pg.Query(r.Context(), `
		SELECT e.edge_key, e.name, e.length_m, ST_AsGeoJSON(e.line::geometry)::text, sel.passes, sel.mps
		  FROM (
		      SELECT edge_key, sum(passes)::int AS passes, sum(speed_sum) / sum(passes) AS mps
		        FROM map_edge_speed
		       WHERE ($1::int < 0 OR hour = $1::int) AND ($2::text = '' OR weekend = ($2::text = '1'))
		       GROUP BY edge_key HAVING sum(passes) >= $3::int
		  ) sel JOIN map_road_edges e USING (edge_key)
		 ORDER BY sel.passes DESC, e.edge_key LIMIT 8000`, hour, weekend, minPasses)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	feats := []map[string]any{}
	for rows.Next() {
		var key, name, g string
		var length float32
		var passes int
		var mps float64
		if err := rows.Scan(&key, &name, &length, &g, &passes, &mps); err != nil {
			s.respondErr(w, err)
			return
		}
		feats = append(feats, map[string]any{
			"type":     "Feature",
			"geometry": json.RawMessage(g),
			"properties": map[string]any{
				"name": name, "length_m": float64(length), "passes": passes,
				"kmh": float64(int(mps*3.6*10+0.5)) / 10, "slow": slow[key],
			},
		})
	}
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"type": "FeatureCollection", "features": feats})
}

// handleRoadSpeedsSummary GET /ops-map/road-speeds/summary — ملخّصٌ للقراءة فقط يصلح أساساً لـETA لاحقاً.
func (s *Server) handleRoadSpeedsSummary(w http.ResponseWriter, r *http.Request) {
	sum, err := maplearn.ReadSpeedSummary(r.Context(), s.pg, maplearn.DefaultSlowParams())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, sum)
}

// handleRoadLearningSummary GET /ops-map/road-learning/summary — أرقامُ القسم وآخر جولة.
func (s *Server) handleRoadLearningSummary(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	out := map[string]any{"running": maplearn.RoadsBusy(), "can_match": s.cfg.ValhallaURL != ""}
	var days, edges int
	var last *string
	if err := s.pg.QueryRow(ctx, `SELECT count(*)::int, max(day)::text FROM map_road_days`).Scan(&days, &last); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.pg.QueryRow(ctx, `SELECT count(*)::int FROM map_road_edges`).Scan(&edges); err != nil {
		s.respondErr(w, err)
		return
	}
	out["days"], out["edges"], out["last_day"] = days, edges, last
	var at *time.Time
	var d, p, o int
	var errTxt string
	if err := s.pg.QueryRow(ctx, `
		SELECT finished_at, days_done, passes, oneway, error FROM map_road_runs
		 WHERE finished_at IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&at, &d, &p, &o, &errTxt); err == nil {
		out["last_run"] = map[string]any{"at": at, "days": d, "passes": p, "oneway": o, "failed": errTxt != ""}
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleRoadLearningRun POST /ops-map/road-learning/run — جولةٌ بالخلفيّة.
func (s *Server) handleRoadLearningRun(w http.ResponseWriter, r *http.Request) {
	if maplearn.RoadsBusy() {
		s.respondErr(w, errMapLearnBusy)
		return
	}
	go s.mapRoadsOnce(context.Background())
	httpx.JSON(w, http.StatusAccepted, map[string]any{"started": true})
}
