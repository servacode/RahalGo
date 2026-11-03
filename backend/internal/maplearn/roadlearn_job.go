package maplearn

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

// RoadOptions خياراتُ جولة تعلّم الطرق.
type RoadOptions struct {
	Road    RoadParams
	OneWay  OneWayParams
	Workers int
	// MaxDays أكثرُ أيّامٍ تُعالَج في الجولة (الباقي في التالية).
	MaxDays int
	// KeepDays مدّةُ حفظ عبورات الاتّجاه (تُقلَّم الأقدم).
	KeepDays int
}

// DefaultRoadOptions الافتراضات.
func DefaultRoadOptions() RoadOptions {
	return RoadOptions{Road: DefaultRoadParams(), OneWay: DefaultOneWayParams(), Workers: 4, MaxDays: 40, KeepDays: 90}
}

// RoadResult نتيجةُ جولة.
type RoadResult struct {
	Days   int
	Passes int
	OneWay int
}

// ErrNoMatcher لا محرّكَ يطابق الأثر (Valhalla غيرُ مُعدّ).
var ErrNoMatcher = errors.New("maplearn: لا Valhalla مُعدّ لمطابقة الأثر")

// ErrMatcherDown محرّكُ المطابقة لا يردّ — اليومُ لم يُعلَّم معالَجاً.
var ErrMatcherDown = errors.New("maplearn: محرّك المطابقة لا يردّ")

var roadMu sync.Mutex

// RoadsBusy أتعمل جولةُ الطرق الآن؟
func RoadsBusy() bool {
	if roadMu.TryLock() {
		roadMu.Unlock()
		return false
	}
	return true
}

// RunRoads **جولةُ تعلّم الطرق**: تعالج كلَّ يومٍ دمشقيٍّ منتهٍ لم يُعالَج (مرّةً واحدة)، ثمّ تحدّث مقترحات الاتّجاه الواحد.
// وتُسجَّل في `map_road_runs`. جولتان معاً تُرفض الثانية. **بلا مطابقٍ** تُحدَّث المقترحاتُ من المجاميع المخزّنة فقط.
func RunRoads(ctx context.Context, pg *pgxpool.Pool, m Matcher, opt RoadOptions) (res RoadResult, err error) {
	if !roadMu.TryLock() {
		return res, errors.New("maplearn: جولةُ الطرق تعمل الآن")
	}
	defer roadMu.Unlock()
	d := DefaultRoadOptions()
	if opt.Road.MinPts <= 0 {
		opt.Road = d.Road
	}
	if opt.OneWay.MinPasses <= 0 {
		opt.OneWay = d.OneWay
	}
	if opt.Workers <= 0 {
		opt.Workers = d.Workers
	}
	if opt.MaxDays <= 0 {
		opt.MaxDays = d.MaxDays
	}
	if opt.KeepDays <= 0 {
		opt.KeepDays = d.KeepDays
	}

	var runID int64
	if err := pg.QueryRow(ctx, `INSERT INTO map_road_runs DEFAULT VALUES RETURNING id`).Scan(&runID); err != nil {
		return res, err
	}
	defer func() {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		_, _ = pg.Exec(context.WithoutCancel(ctx), `
			UPDATE map_road_runs SET finished_at = now(), days_done = $2, passes = $3, oneway = $4, error = $5
			 WHERE id = $1`, runID, res.Days, res.Passes, res.OneWay, msg)
	}()

	if m != nil {
		days, derr := pendingRoadDays(ctx, pg, opt.MaxDays)
		if derr != nil {
			return res, derr
		}
		for _, day := range days {
			n, perr := processRoadDay(ctx, pg, m, day, opt)
			if perr != nil {
				return res, perr
			}
			res.Days++
			res.Passes += n
		}
	}
	if _, err = pg.Exec(ctx, `DELETE FROM map_edge_passes WHERE day < (now() AT TIME ZONE 'Asia/Damascus')::date - $1::int`, opt.KeepDays); err != nil {
		return res, err
	}
	res.OneWay, err = detectOneWay(ctx, pg, opt)
	return res, err
}

// pendingRoadDays الأيّامُ الدمشقيّةُ المنتهية التي فيها أثرٌ ولم تُعالَج، الأقدمُ أوّلاً.
func pendingRoadDays(ctx context.Context, pg *pgxpool.Pool, limit int) ([]string, error) {
	rows, err := pg.Query(ctx, `
		SELECT d::text FROM (
		    SELECT DISTINCT (recorded_at AT TIME ZONE 'Asia/Damascus')::date AS d FROM driver_trail
		) t
		 WHERE d < (now() AT TIME ZONE 'Asia/Damascus')::date
		   AND NOT EXISTS (SELECT 1 FROM map_road_days r WHERE r.day = t.d)
		 ORDER BY d LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

type passKey struct {
	edge   string
	dir    int
	driver string
}
type speedKey struct {
	edge    string
	hour    int
	weekend bool
}
type speedAcc struct {
	n   int
	sum float64
}

// processRoadDay يعالج يوماً: يقطّع أثرَ كلّ سائق، يطابق، يجمع في الذاكرة، ثمّ يكتب المجاميع ويعلّم اليوم بمعاملةٍ واحدة.
func processRoadDay(ctx context.Context, pg *pgxpool.Pool, m Matcher, day string, opt RoadOptions) (int, error) {
	from, to, err := drivermonitor.DayBounds(day)
	if err != nil {
		return 0, err
	}
	rows, err := pg.Query(ctx, `
		SELECT DISTINCT driver_id::text FROM driver_trail WHERE recorded_at >= $1 AND recorded_at < $2`, from, to)
	if err != nil {
		return 0, err
	}
	var drivers []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, err
		}
		drivers = append(drivers, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	type job struct {
		driver string
		pts    []drivermonitor.Point
	}
	var jobs []job
	for _, id := range drivers {
		pts, err := drivermonitor.LoadPoints(ctx, pg, id, from, to)
		if err != nil {
			return 0, err
		}
		for _, c := range Chunks(pts, opt.Road) {
			jobs = append(jobs, job{driver: id, pts: c})
		}
	}

	var mu sync.Mutex
	edges := map[string]MatchedEdge{}
	dirs := map[passKey]int{}
	speeds := map[speedKey]*speedAcc{}
	var ok, fails int64
	var passTotal int64
	var wg sync.WaitGroup
	ch := make(chan job)
	for w := 0; w < opt.Workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range ch {
				res, merr := m.Match(ctx, j.pts)
				if merr != nil {
					if errors.Is(merr, ErrNoMatch) {
						atomic.AddInt64(&ok, 1) // الردُّ سليم
					} else {
						atomic.AddInt64(&fails, 1)
					}
					continue
				}
				atomic.AddInt64(&ok, 1)
				ps := Passes(j.pts, res, opt.Road)
				mu.Lock()
				for _, p := range ps {
					e := res.Edges[p.Edge]
					edges[e.Key] = e
					dirs[passKey{e.Key, p.Dir, j.driver}]++
					if p.HasSpeed {
						k := speedKey{e.Key, p.Hour, p.Weekend}
						a := speeds[k]
						if a == nil {
							a = &speedAcc{}
							speeds[k] = a
						}
						a.n++
						a.sum += p.SpeedMps
					}
				}
				mu.Unlock()
				atomic.AddInt64(&passTotal, int64(len(ps)))
			}
		}()
	}
feed:
	for _, j := range jobs {
		// **محرّكٌ ميّتٌ يوقف اليوم** — لا يُدار على مئاتِ المهل.
		if atomic.LoadInt64(&fails) >= 20 && atomic.LoadInt64(&ok) == 0 {
			break feed
		}
		select {
		case ch <- j:
		case <-ctx.Done():
			break feed
		}
	}
	close(ch)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if ok == 0 && fails > 0 {
		return 0, ErrMatcherDown
	}

	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var ek, en, eg []string
	var ew []int64
	var el []float32
	var eo []bool
	for _, e := range edges {
		ek, en = append(ek, e.Key), append(en, e.Name)
		ew, el, eo = append(ew, e.WayID), append(el, float32(e.Length)), append(eo, e.OneWayKnown)
		eg = append(eg, wktLine(e.Line))
	}
	if len(ek) > 0 {
		if _, err := tx.Exec(ctx, `
			INSERT INTO map_road_edges (edge_key, way_id, name, length_m, line, one_way_known)
			SELECT k, w, n, l, ST_GeogFromText(g), o
			  FROM unnest($1::text[], $2::bigint[], $3::text[], $4::real[], $5::text[], $6::boolean[]) AS v(k, w, n, l, g, o)
			ON CONFLICT (edge_key) DO UPDATE SET name = EXCLUDED.name, one_way_known = EXCLUDED.one_way_known,
			       updated_at = now()`, ek, ew, en, el, eg, eo); err != nil {
			return 0, err
		}
	}
	if len(dirs) > 0 {
		var k, dr []string
		var dd []int16
		var pp []int32
		for key, n := range dirs {
			k, dr, dd, pp = append(k, key.edge), append(dr, key.driver), append(dd, int16(key.dir)), append(pp, int32(n))
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO map_edge_passes (edge_key, dir, day, driver_id, passes)
			SELECT k, d, $5::date, r::uuid, p FROM unnest($1::text[], $2::smallint[], $3::text[], $4::int[]) AS v(k, d, r, p)
			ON CONFLICT (edge_key, dir, day, driver_id) DO UPDATE SET passes = map_edge_passes.passes + EXCLUDED.passes`,
			k, dd, dr, pp, day); err != nil {
			return 0, err
		}
	}
	if len(speeds) > 0 {
		var k []string
		var hh []int16
		var we []bool
		var nn []int32
		var ss []float64
		for key, a := range speeds {
			k, hh, we = append(k, key.edge), append(hh, int16(key.hour)), append(we, key.weekend)
			nn, ss = append(nn, int32(a.n)), append(ss, a.sum)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO map_edge_speed (edge_key, hour, weekend, passes, speed_sum)
			SELECT * FROM unnest($1::text[], $2::smallint[], $3::boolean[], $4::int[], $5::double precision[])
			ON CONFLICT (edge_key, hour, weekend) DO UPDATE
			   SET passes = map_edge_speed.passes + EXCLUDED.passes, speed_sum = map_edge_speed.speed_sum + EXCLUDED.speed_sum`,
			k, hh, we, nn, ss); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO map_road_days (day, passes, edges, drivers) VALUES ($1::date, $2, $3, $4)
		ON CONFLICT (day) DO NOTHING`, day, int(passTotal), len(edges), len(drivers)); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return int(passTotal), nil
}

// detectOneWay يقرأ مجاميعَ العبور ويحدّث المقترحات: المنتظرةُ تُحدَّث أو تُحذف إن لم تعد تنطبق، والمقرَّرةُ لا تُمسّ.
func detectOneWay(ctx context.Context, pg *pgxpool.Pool, opt RoadOptions) (int, error) {
	rows, err := pg.Query(ctx, `
		SELECT p.edge_key,
		       COALESCE(sum(p.passes) FILTER (WHERE p.dir = 1), 0)::int,
		       COALESCE(sum(p.passes) FILTER (WHERE p.dir = -1), 0)::int,
		       count(DISTINCT p.driver_id)::int, count(DISTINCT p.day)::int
		  FROM map_edge_passes p JOIN map_road_edges e USING (edge_key)
		 WHERE NOT e.one_way_known
		 GROUP BY p.edge_key HAVING sum(p.passes) >= $1`, opt.OneWay.MinPasses)
	if err != nil {
		return 0, err
	}
	var stats []EdgeDirStat
	for rows.Next() {
		var s EdgeDirStat
		if err := rows.Scan(&s.Key, &s.Fwd, &s.Rev, &s.Drivers, &s.Days); err != nil {
			rows.Close()
			return 0, err
		}
		stats = append(stats, s)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	found := DetectOneWay(stats, opt.OneWay)

	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	keep := make([]string, 0, len(found))
	for _, f := range found {
		keep = append(keep, f.Key)
		// الخطُّ بترتيب الاتّجاه الغالب = اتّجاهُ `oneway=yes`.
		if _, err := tx.Exec(ctx, `
			INSERT INTO map_oneway_candidates (edge_key, way_id, name, line, dir, main_passes, opp_passes, drivers, days, share)
			SELECT e.edge_key, e.way_id, e.name,
			       (CASE WHEN $2 = 1 THEN e.line::geometry ELSE ST_Reverse(e.line::geometry) END)::geography,
			       $2, $3, $4, $5, $6, $7
			  FROM map_road_edges e WHERE e.edge_key = $1
			ON CONFLICT (edge_key) DO UPDATE
			   SET name = EXCLUDED.name, line = EXCLUDED.line, dir = EXCLUDED.dir, main_passes = EXCLUDED.main_passes,
			       opp_passes = EXCLUDED.opp_passes, drivers = EXCLUDED.drivers, days = EXCLUDED.days,
			       share = EXCLUDED.share, updated_at = now()
			 WHERE map_oneway_candidates.status = 'pending'`,
			f.Key, f.Dir, f.Main, f.Opp, f.Drivers, f.Days, float32(f.Share)); err != nil {
			return 0, err
		}
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM map_oneway_candidates WHERE status = 'pending' AND NOT (edge_key = ANY($1::text[]))`, keep); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(found), nil
}

// ───────────────────────── ملخّصٌ للقراءة (للـETA لاحقاً) ─────────────────────────

// HourSpeed سرعةُ المدينة في خانة (ساعة، نهايةِ أسبوع): متوسّطٌ موزونٌ بالعبورات.
type HourSpeed struct {
	Hour    int     `json:"hour"`
	Weekend bool    `json:"weekend"`
	Edges   int     `json:"edges"`
	Passes  int     `json:"passes"`
	AvgKmh  float64 `json:"avg_kmh"`
}

// SpeedSummary ملخّصٌ للقراءة فقط: ما في المجاميع وسرعةُ المدينة في كلّ خانة. **لا يدخل الـETA الآن.**
type SpeedSummary struct {
	Edges     int         `json:"edges"`
	Passes    int         `json:"passes"`
	SlowEdges int         `json:"slow_edges"`
	Buckets   []HourSpeed `json:"buckets"`
}

// ReadSpeedSummary يقرأ الملخّص من `map_edge_speed`.
func ReadSpeedSummary(ctx context.Context, pg *pgxpool.Pool, sp SlowParams) (SpeedSummary, error) {
	out := SpeedSummary{Buckets: []HourSpeed{}}
	rows, err := pg.Query(ctx, `
		SELECT hour, weekend, count(*)::int, sum(passes)::int, sum(speed_sum) / sum(passes)
		  FROM map_edge_speed GROUP BY hour, weekend ORDER BY weekend, hour`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var h HourSpeed
		var mps float64
		if err := rows.Scan(&h.Hour, &h.Weekend, &h.Edges, &h.Passes, &mps); err != nil {
			rows.Close()
			return out, err
		}
		h.AvgKmh = float64(int(mps*3.6*10+0.5)) / 10
		out.Buckets = append(out.Buckets, h)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return out, err
	}
	if err := pg.QueryRow(ctx, `SELECT count(DISTINCT edge_key)::int, COALESCE(sum(passes), 0)::int FROM map_edge_speed`).
		Scan(&out.Edges, &out.Passes); err != nil {
		return out, err
	}
	slow, err := ReadSlowEdges(ctx, pg, sp)
	out.SlowEdges = len(slow)
	return out, err
}

// ReadSlowEdges مفاتيحُ الحافّات «البطيئة باستمرار».
func ReadSlowEdges(ctx context.Context, pg *pgxpool.Pool, sp SlowParams) (map[string]bool, error) {
	rows, err := pg.Query(ctx, slowSQL, sp.BucketPasses, sp.SlowMps)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		var passes, days, qual, slow int
		if err := rows.Scan(&k, &passes, &days, &qual, &slow); err != nil {
			return nil, err
		}
		if SlowFromCounts(passes, days, qual, slow, sp) {
			out[k] = true
		}
	}
	return out, rows.Err()
}

// slowSQL عدّاداتُ كلّ حافّة: عبوراتٌ، أيّام، خاناتٌ محسوبة، خاناتٌ بطيئة. $1 أقلُّ عبوراتِ خانة، $2 عتبةُ البطء م/ث.
const slowSQL = `
	SELECT s.edge_key, s.passes, COALESCE(d.days, 0)::int, s.qual, s.slowb FROM (
	    SELECT edge_key, sum(passes)::int AS passes,
	           (count(*) FILTER (WHERE passes >= $1))::int AS qual,
	           (count(*) FILTER (WHERE passes >= $1 AND speed_sum / passes < $2))::int AS slowb
	      FROM map_edge_speed GROUP BY edge_key
	) s LEFT JOIN (
	    SELECT edge_key, count(DISTINCT day) AS days FROM map_edge_passes GROUP BY edge_key
	) d USING (edge_key)`
