package server

import (
	"context"
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
// **أماكن بلا اسم وأسامي** (قرارُ المالك ٢٠٢٦-١١-٠٣) — الدفعة الثانية من «الخريطة من السائقين»
// ══════════════════════════════════════════════════════════════════════
//
// مرشَّحو معالمَ من وقفات السائقين المتكرّرة ومن أسماء المعالم في عناوين الزبائن.
// المدير يسمّي ويوافق فيدخل `map_landmarks` بمصدر `drivers`، أو يرفض. الحسابُ في `internal/maplearn`.

var errPlaceDecided = httpx.NewError(http.StatusConflict, "place_decided", "errors.conflict")

// mapPlacesOnce جولةٌ يوميّة للأماكن (لا تحتاج محرّكَ خرائط).
func (s *Server) mapPlacesOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()
	res, err := maplearn.RunPlaces(runCtx, s.pg, maplearn.DefaultPlaceParams())
	if err != nil {
		s.logger.Warn("الأماكن بلا اسم: تعثّرت الجولة", "error", err)
		return
	}
	s.logger.Info("الأماكن بلا اسم: انتهت الجولة", "stop_places", res.StopPlaces, "name_places", res.NamePlaces)
}

type placeCandidate struct {
	ID        int64    `json:"id"`
	Source    string   `json:"source"`
	Status    string   `json:"status"`
	Name      string   `json:"name"`
	Kind      string   `json:"kind"`
	Lat       float64  `json:"lat"`
	Lng       float64  `json:"lng"`
	Visits    int      `json:"visits"`
	Drivers   int      `json:"drivers"`
	Customers int      `json:"customers"`
	Days      int      `json:"days"`
	Evidence  int      `json:"evidence"`
	Samples   []string `json:"samples"`
	FirstSeen *string  `json:"first_seen"`
	LastSeen  *string  `json:"last_seen"`
}

type placeStore struct {
	ID   string  `json:"id"`
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// handlePlaceCandidates GET /ops-map/place-candidates?status=pending|approved|rejected&source=stops|addresses
// — المرشَّحون وأعدادُهم وآخرُ جولة وأسماءُ المتاجر (تُرسم أسماؤها على الخريطة).
func (s *Server) handlePlaceCandidates(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	status := r.URL.Query().Get("status")
	if status != "approved" && status != "rejected" {
		status = "pending"
	}
	source := r.URL.Query().Get("source")
	if source != "stops" && source != "addresses" {
		source = ""
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id, source, status, name, kind, ST_Y(point::geometry), ST_X(point::geometry),
		       visits, drivers, customers, days, evidence, samples, first_seen::text, last_seen::text
		  FROM map_place_candidates
		 WHERE status = $1 AND ($2 = '' OR source = $2)
		 ORDER BY GREATEST(visits, evidence) DESC, id LIMIT 500`, status, source)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	cands := []placeCandidate{}
	for rows.Next() {
		var c placeCandidate
		if err := rows.Scan(&c.ID, &c.Source, &c.Status, &c.Name, &c.Kind, &c.Lat, &c.Lng, &c.Visits, &c.Drivers,
			&c.Customers, &c.Days, &c.Evidence, &c.Samples, &c.FirstSeen, &c.LastSeen); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		if c.Samples == nil {
			c.Samples = []string{}
		}
		cands = append(cands, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		s.respondErr(w, err)
		return
	}

	stores := []placeStore{}
	srows, err := s.pg.Query(ctx, `
		SELECT id::text, name, ST_Y(location::geometry), ST_X(location::geometry)
		  FROM merchants WHERE location IS NOT NULL AND status = 'active' ORDER BY name LIMIT 2000`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for srows.Next() {
		var p placeStore
		if err := srows.Scan(&p.ID, &p.Name, &p.Lat, &p.Lng); err != nil {
			srows.Close()
			s.respondErr(w, err)
			return
		}
		stores = append(stores, p)
	}
	srows.Close()

	counts := map[string]int{"pending": 0, "approved": 0, "rejected": 0}
	crows, err := s.pg.Query(ctx, `
		SELECT status, count(*)::int FROM map_place_candidates WHERE $1 = '' OR source = $1 GROUP BY status`, source)
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

	out := map[string]any{
		"candidates": cands, "count": len(cands), "counts": counts, "stores": stores,
		"running": maplearn.PlacesBusy(),
	}
	var at *time.Time
	var errTxt string
	var sp, np int
	if err := s.pg.QueryRow(ctx, `
		SELECT finished_at, stop_places, name_places, error FROM map_place_runs
		 WHERE finished_at IS NOT NULL ORDER BY id DESC LIMIT 1`).Scan(&at, &sp, &np, &errTxt); err == nil {
		out["last_run"] = map[string]any{"at": at, "stop_places": sp, "name_places": np, "failed": errTxt != ""}
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handlePlaceCandidatesRun POST /ops-map/place-candidates/run — جولةٌ بالخلفيّة.
func (s *Server) handlePlaceCandidatesRun(w http.ResponseWriter, r *http.Request) {
	if maplearn.PlacesBusy() {
		s.respondErr(w, errMapLearnBusy)
		return
	}
	go s.mapPlacesOnce(context.Background())
	httpx.JSON(w, http.StatusAccepted, map[string]any{"started": true})
}

// handlePlaceCandidateDecide POST /ops-map/place-candidates/{id}/decide  {approve, name, kind}
// — الموافقةُ تُدخل معلماً في `map_landmarks` (مصدرُه `drivers`) فيدخل «قرب المعلم» فوراً.
func (s *Server) handlePlaceCandidateDecide(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Approve bool   `json:"approve"`
		Name    string `json:"name"`
		Kind    string `json:"kind"`
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
	kind := req.Kind
	if req.Approve {
		if n := len([]rune(name)); n < 2 || n > 80 {
			s.respondErr(w, errLandmarkName)
			return
		}
		if kind == "" {
			kind = "other"
		}
		if _, ok := landmarkKinds[kind]; !ok {
			s.respondErr(w, errLandmarkKind)
			return
		}
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	var got int64
	err = tx.QueryRow(ctx, `
		UPDATE map_place_candidates
		   SET status = $2, name = CASE WHEN $2 = 'approved' THEN $3 ELSE name END,
		       kind = CASE WHEN $2 = 'approved' THEN $4 ELSE kind END,
		       decided_by = $5::uuid, decided_at = now(), updated_at = now()
		 WHERE id = $1 AND status = 'pending' RETURNING id`, id, status, name, kind, userIDFrom(r)).Scan(&got)
	if err == pgx.ErrNoRows {
		var exists bool
		_ = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM map_place_candidates WHERE id = $1)`, id).Scan(&exists)
		if !exists {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errPlaceDecided)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	var lmID int64
	if req.Approve {
		if err := tx.QueryRow(ctx, `
			INSERT INTO map_landmarks (name, kind, point, source)
			SELECT name, $2, point, 'drivers' FROM map_place_candidates WHERE id = $1 RETURNING id`,
			id, landmarkKinds[kind]).Scan(&lmID); err != nil {
			s.respondErr(w, err)
			return
		}
		if _, err := tx.Exec(ctx, `UPDATE map_place_candidates SET landmark_id = $2 WHERE id = $1`, id, lmID); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status, "landmark_id": lmID})
}
