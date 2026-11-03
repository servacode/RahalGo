package maplearn

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Options خياراتُ جولةٍ واحدة.
type Options struct {
	// Batch أكثرُ ما يُقاس من النقاط في الجولة — الباقي في التالية.
	Batch int
	// Workers عددُ الطلبات المتوازية إلى محرّك الخريطة.
	Workers int
	// FarM بُعدٌ عن الطريق يُعدّ بعده الموضعُ «خارج الخريطة».
	FarM   float64
	Detect Params
}

// DefaultOptions الافتراضات.
func DefaultOptions() Options {
	return Options{Batch: 20000, Workers: 8, FarM: 18, Detect: DefaultParams()}
}

// Result نتيجةُ جولة.
type Result struct {
	Measured   int
	FarPoints  int
	Candidates int
}

// ErrNoRoads لا محرّكَ خرائطَ مُعدّاً للقياس عليه.
var ErrNoRoads = errors.New("maplearn: لا OSRM ولا Valhalla مُعدّ")

// ErrRoadsDown محرّكُ الخرائط لا يردّ — الجولةُ توقّفت ولم تُلمس الاقتراحات.
var ErrRoadsDown = errors.New("maplearn: محرّك الخرائط لا يردّ")

var runMu sync.Mutex

// Busy أتعمل جولةٌ الآن؟
func Busy() bool {
	if runMu.TryLock() {
		runMu.Unlock()
		return false
	}
	return true
}

// Run **جولةٌ كاملة**: قياسُ ما لم يُقَس، ثمّ كشفُ الخطوط، ثمّ حفظُ المقترحات. وتسجَّل في `map_street_runs`.
// جولتان معاً تُرفض الثانية.
func Run(ctx context.Context, pg *pgxpool.Pool, roads RoadFinder, opt Options) (res Result, err error) {
	if roads == nil {
		return res, ErrNoRoads
	}
	if !runMu.TryLock() {
		return res, errors.New("maplearn: جولةٌ تعمل الآن")
	}
	defer runMu.Unlock()
	if opt.Batch <= 0 || opt.Workers <= 0 {
		d := DefaultOptions()
		if opt.Batch <= 0 {
			opt.Batch = d.Batch
		}
		if opt.Workers <= 0 {
			opt.Workers = d.Workers
		}
	}
	if opt.FarM <= 0 {
		opt.FarM = 18
	}
	if opt.Detect.CellM <= 0 {
		opt.Detect = DefaultParams()
	}

	var runID int64
	if err := pg.QueryRow(ctx, `INSERT INTO map_street_runs DEFAULT VALUES RETURNING id`).Scan(&runID); err != nil {
		return res, err
	}
	defer func() {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		// السياقُ الأصليُّ قد يكون أُلغي — السجلُّ يُكتب مع ذلك.
		_, _ = pg.Exec(context.WithoutCancel(ctx), `
			UPDATE map_street_runs SET finished_at = now(), measured = $2, far_points = $3,
			       candidates = $4, error = $5 WHERE id = $1`,
			runID, res.Measured, res.FarPoints, res.Candidates, msg)
	}()

	if res.Measured, err = measure(ctx, pg, roads, opt); err != nil {
		return res, err
	}
	res.FarPoints, res.Candidates, err = detectAndStore(ctx, pg, opt)
	return res, err
}

// measure يقيس بُعد النقاط غير المقيسة ويكتبه.
func measure(ctx context.Context, pg *pgxpool.Pool, roads RoadFinder, opt Options) (int, error) {
	rows, err := pg.Query(ctx, `
		SELECT id, ST_Y(point::geometry), ST_X(point::geometry)
		  FROM map_trace_points WHERE road_dist_m IS NULL ORDER BY id LIMIT $1`, opt.Batch)
	if err != nil {
		return 0, err
	}
	type pt struct {
		id       int64
		lat, lng float64
	}
	var todo []pt
	for rows.Next() {
		var p pt
		if err := rows.Scan(&p.id, &p.lat, &p.lng); err != nil {
			rows.Close()
			return 0, err
		}
		todo = append(todo, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(todo) == 0 {
		return 0, nil
	}

	dist := make([]float32, len(todo))
	ok := make([]bool, len(todo))
	var fails, done int64
	var wg sync.WaitGroup
	jobs := make(chan int)
	for w := 0; w < opt.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				d, err := roads.NearestM(ctx, todo[i].lat, todo[i].lng)
				if err != nil {
					atomic.AddInt64(&fails, 1)
					continue
				}
				dist[i], ok[i] = float32(d), true
				atomic.AddInt64(&done, 1)
			}
		}()
	}
feed:
	for i := range todo {
		// **محرّكٌ ميّتٌ يوقف الجولة** لا يُدور على عشرين ألف مهلة.
		if atomic.LoadInt64(&fails) >= 50 && atomic.LoadInt64(&done) == 0 {
			break feed
		}
		select {
		case jobs <- i:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()

	var ids []int64
	var ds []float32
	for i, p := range todo {
		if ok[i] {
			ids = append(ids, p.id)
			ds = append(ds, dist[i])
		}
	}
	if len(ids) > 0 {
		if _, err := pg.Exec(ctx, `
			UPDATE map_trace_points m SET road_dist_m = v.d
			  FROM unnest($1::bigint[], $2::real[]) AS v(id, d) WHERE m.id = v.id`, ids, ds); err != nil {
			return 0, err
		}
	}
	if done == 0 {
		return 0, ErrRoadsDown
	}
	return len(ids), ctx.Err()
}

// detectAndStore يقرأ البعيدَ، يكشف، ويكتب المقترحات.
func detectAndStore(ctx context.Context, pg *pgxpool.Pool, opt Options) (far, stored int, err error) {
	// **ما غطّاه خطٌّ قُرّر (موافَق أو مرفوض) لا يُعاد اقتراحُه.**
	rows, err := pg.Query(ctx, `
		SELECT ST_Y(p.point::geometry), ST_X(p.point::geometry), p.day::text
		  FROM map_trace_points p
		 WHERE p.road_dist_m >= $1
		   AND NOT EXISTS (SELECT 1 FROM map_street_candidates c
		                    WHERE c.status <> 'pending' AND ST_DWithin(c.line, p.point, 30))`, opt.FarM)
	if err != nil {
		return 0, 0, err
	}
	var pts []Point
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.Lat, &p.Lng, &p.Day); err != nil {
			rows.Close()
			return 0, 0, err
		}
		pts = append(pts, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	cands := Detect(pts, opt.Detect)

	tx, err := pg.Begin(ctx)
	if err != nil {
		return len(pts), 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// **المقترحاتُ المنتظرة تُطابَق بالقرب** فتبقى أرقامُها — ومن لم يعد يطابقه شيءٌ يُحذف.
	keep := map[int64]bool{}
	for _, c := range cands {
		wkt := wktLine(c.Line)
		var id int64
		qerr := tx.QueryRow(ctx, `
			SELECT id FROM map_street_candidates
			 WHERE status = 'pending' AND id <> ALL($2::bigint[])
			   AND ST_DWithin(line, ST_GeogFromText($1), 20)
			 ORDER BY id LIMIT 1`, wkt, keysOf(keep)).Scan(&id)
		if qerr == nil {
			if _, err := tx.Exec(ctx, `
				UPDATE map_street_candidates SET line = ST_GeogFromText($2), length_m = $3, points = $4,
				       days = $5, first_day = $6::date, last_day = $7::date, updated_at = now()
				 WHERE id = $1`, id, wkt, c.LengthM, c.Points, c.Days, c.FirstDay, c.LastDay); err != nil {
				return len(pts), 0, err
			}
			keep[id] = true
			continue
		}
		if err := tx.QueryRow(ctx, `
			INSERT INTO map_street_candidates (line, length_m, points, days, first_day, last_day)
			VALUES (ST_GeogFromText($1), $2, $3, $4, $5::date, $6::date) RETURNING id`,
			wkt, c.LengthM, c.Points, c.Days, c.FirstDay, c.LastDay).Scan(&id); err != nil {
			return len(pts), 0, err
		}
		keep[id] = true
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM map_street_candidates WHERE status = 'pending' AND id <> ALL($1::bigint[])`,
		keysOf(keep)); err != nil {
		return len(pts), 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return len(pts), 0, err
	}
	return len(pts), len(cands), nil
}

func keysOf(m map[int64]bool) []int64 {
	out := make([]int64, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func wktLine(l [][2]float64) string {
	s := "SRID=4326;LINESTRING("
	for i, p := range l {
		if i > 0 {
			s += ","
		}
		s += fmt.Sprintf("%.7f %.7f", p[0], p[1])
	}
	return s + ")"
}
