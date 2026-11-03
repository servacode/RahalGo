package maplearn

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/drivermonitor"
)

const (
	// بعدٌ يُعدّ عنده المكانُ «معروفاً» (معلمٌ أو متجرٌ أو قرارٌ سابق).
	knownRadiusM = 60.0
	// بعدُ مطابقة اسمٍ لمعلمٍ موجود.
	nameKnownRadiusM = 200.0
	// مطابقةُ مقترَحٍ منتظرٍ بالقرب فتبقى أرقامُه.
	stopKeepRadiusM = 50.0
	nameKeepRadiusM = 150.0

	stopMinDur   = 2 * time.Minute
	stopRadiusM  = 40.0
	stopGapAfter = 6 * time.Minute
	trailDays    = 45
)

// PlaceResult نتيجةُ جولة الأماكن.
type PlaceResult struct {
	StopPlaces int
	NamePlaces int
}

var placeMu sync.Mutex

// PlacesBusy أتعمل جولةُ أماكنَ الآن؟
func PlacesBusy() bool {
	if placeMu.TryLock() {
		placeMu.Unlock()
		return false
	}
	return true
}

// RunPlaces **جولةُ الأماكن والأسماء**: الوقفاتُ المتكرّرة ثمّ أسماءُ العناوين، وتُسجَّل في `map_place_runs`.
func RunPlaces(ctx context.Context, pg *pgxpool.Pool, p PlaceParams) (res PlaceResult, err error) {
	if !placeMu.TryLock() {
		return res, errors.New("maplearn: جولةُ أماكنَ تعمل الآن")
	}
	defer placeMu.Unlock()
	if p.ClusterM <= 0 {
		p = DefaultPlaceParams()
	}
	var runID int64
	if err := pg.QueryRow(ctx, `INSERT INTO map_place_runs DEFAULT VALUES RETURNING id`).Scan(&runID); err != nil {
		return res, err
	}
	defer func() {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		_, _ = pg.Exec(context.WithoutCancel(ctx), `
			UPDATE map_place_runs SET finished_at = now(), stop_places = $2, name_places = $3, error = $4
			 WHERE id = $1`, runID, res.StopPlaces, res.NamePlaces, msg)
	}()
	if res.StopPlaces, err = stopPlaces(ctx, pg, p); err != nil {
		return res, err
	}
	res.NamePlaces, err = namePlaces(ctx, pg, p)
	return res, err
}

// known: أفي النقطة معلمٌ أو متجرٌ ضمن `r` م؟ (الأحياءُ والمناطق لا تُحسب — مركزُ الحيّ ليس مكاناً.)
func known(ctx context.Context, q pgx.Tx, lat, lng, r float64) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM map_landmarks l
		                WHERE l.kind NOT LIKE 'place=%'
		                  AND ST_DWithin(l.point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3))
		    OR EXISTS (SELECT 1 FROM merchants m
		                WHERE m.location IS NOT NULL AND m.status = 'active'
		                  AND ST_DWithin(m.location, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3))
		    OR EXISTS (SELECT 1 FROM map_place_candidates c
		                WHERE c.status <> 'pending' AND c.source = 'stops'
		                  AND ST_DWithin(c.point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3))`,
		lat, lng, r).Scan(&ok)
	return ok, err
}

// stopPlaces يقرأ أثر السائقين، يستخرج الوقفات، يجمّعها، ويحفظ ما لا معلمَ عنده.
func stopPlaces(ctx context.Context, pg *pgxpool.Pool, p PlaceParams) (int, error) {
	rows, err := pg.Query(ctx, `
		SELECT driver_id::text, recorded_at, lat, lng
		  FROM driver_trail
		 WHERE recorded_at >= now() - make_interval(days => $1) AND NOT mocked
		 ORDER BY driver_id, recorded_at`, trailDays)
	if err != nil {
		return 0, err
	}
	var obs []StopObs
	cur := ""
	var pts []drivermonitor.Point
	flush := func() {
		for _, s := range drivermonitor.FindStops(pts, stopMinDur, stopRadiusM, stopGapAfter) {
			obs = append(obs, StopObs{Lat: s.Lat, Lng: s.Lng, Driver: cur, Day: drivermonitor.Today(s.Start)})
		}
		pts = pts[:0]
	}
	for rows.Next() {
		var d string
		var pt drivermonitor.Point
		if err := rows.Scan(&d, &pt.At, &pt.Lat, &pt.Lng); err != nil {
			rows.Close()
			return 0, err
		}
		if d != cur {
			flush()
			cur = d
		}
		pts = append(pts, pt)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	flush()

	clusters := ClusterStops(obs, p)

	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	keep := map[int64]bool{}
	stored := 0
	for _, c := range clusters {
		kn, err := known(ctx, tx, c.Lat, c.Lng, knownRadiusM)
		if err != nil {
			return 0, err
		}
		if kn {
			continue
		}
		var id int64
		qerr := tx.QueryRow(ctx, `
			SELECT id FROM map_place_candidates
			 WHERE status = 'pending' AND source = 'stops' AND id <> ALL($4::bigint[])
			   AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)
			 ORDER BY ST_Distance(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography), id LIMIT 1`,
			c.Lat, c.Lng, stopKeepRadiusM, keysOf(keep)).Scan(&id)
		if qerr == nil {
			_, err = tx.Exec(ctx, `
				UPDATE map_place_candidates
				   SET point = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, visits = $4, drivers = $5,
				       days = $6, first_seen = $7::date, last_seen = $8::date, updated_at = now()
				 WHERE id = $1`, id, c.Lat, c.Lng, c.Visits, c.Drivers, c.Days, c.FirstDay, c.LastDay)
		} else if errors.Is(qerr, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				INSERT INTO map_place_candidates (source, point, visits, drivers, days, first_seen, last_seen)
				VALUES ('stops', ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3, $4, $5, $6::date, $7::date)
				RETURNING id`, c.Lat, c.Lng, c.Visits, c.Drivers, c.Days, c.FirstDay, c.LastDay).Scan(&id)
		} else {
			err = qerr
		}
		if err != nil {
			return 0, err
		}
		keep[id] = true
		stored++
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM map_place_candidates WHERE status = 'pending' AND source = 'stops' AND id <> ALL($1::bigint[])`,
		keysOf(keep)); err != nil {
		return 0, err
	}
	return stored, tx.Commit(ctx)
}

// nameKnown: أفي النقطة معلمٌ أو متجرٌ باسمٍ يطابق المقترَح، أو قرارٌ سابقٌ على الاسم نفسه؟
func nameKnown(ctx context.Context, q pgx.Tx, c NameCluster) (bool, error) {
	rows, err := q.Query(ctx, `
		SELECT name FROM map_landmarks
		 WHERE kind NOT LIKE 'place=%' AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)
		UNION ALL
		SELECT name FROM merchants
		 WHERE location IS NOT NULL AND status = 'active'
		   AND ST_DWithin(location, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)`,
		c.Lat, c.Lng, nameKnownRadiusM)
	if err != nil {
		return false, err
	}
	found := false
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			rows.Close()
			return false, err
		}
		if NameMatches(n, c.Key) {
			found = true
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil || found {
		return found, err
	}
	var decided bool
	err = q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM map_place_candidates
		                WHERE source = 'addresses' AND status <> 'pending' AND name_key = $3
		                  AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $4))`,
		c.Lat, c.Lng, c.Key, nameKnownRadiusM).Scan(&decided)
	return decided, err
}

// namePlaces يقرأ عناوينَ الزبائن المسلَّمة ويقترح الأسماء المتّفق عليها.
func namePlaces(ctx context.Context, pg *pgxpool.Pool, p PlaceParams) (int, error) {
	rows, err := pg.Query(ctx, `
		SELECT o.customer_id::text, o.address_text || ' | ' || o.notes,
		       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry)
		  FROM orders o
		 WHERE o.status = 'delivered' AND o.dropoff IS NOT NULL AND o.dropoff_known
		   AND o.created_at >= now() - interval '1 year'
		UNION ALL
		SELECT ua.user_id::text, ua.address_text,
		       ST_Y(COALESCE(ua.entrance, ua.location)::geometry), ST_X(COALESCE(ua.entrance, ua.location)::geometry)
		  FROM user_addresses ua`)
	if err != nil {
		return 0, err
	}
	var ev []AddrEvidence
	for rows.Next() {
		var e AddrEvidence
		if err := rows.Scan(&e.Customer, &e.Text, &e.Lat, &e.Lng); err != nil {
			rows.Close()
			return 0, err
		}
		ev = append(ev, e)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	clusters := ClusterNames(ev, p)

	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	keep := map[int64]bool{}
	stored := 0
	for _, c := range clusters {
		kn, err := nameKnown(ctx, tx, c)
		if err != nil {
			return 0, err
		}
		if kn {
			continue
		}
		var id int64
		qerr := tx.QueryRow(ctx, `
			SELECT id FROM map_place_candidates
			 WHERE status = 'pending' AND source = 'addresses' AND name_key = $3 AND id <> ALL($4::bigint[])
			   AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $5)
			 ORDER BY ST_Distance(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography), id LIMIT 1`,
			c.Lat, c.Lng, c.Key, keysOf(keep), nameKeepRadiusM).Scan(&id)
		if qerr == nil {
			_, err = tx.Exec(ctx, `
				UPDATE map_place_candidates
				   SET point = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, name = $4, kind = $5,
				       evidence = $6, customers = $7, samples = $8, updated_at = now()
				 WHERE id = $1`, id, c.Lat, c.Lng, c.Name, c.Kind, c.Evidence, c.Customers, c.Samples)
		} else if errors.Is(qerr, pgx.ErrNoRows) {
			err = tx.QueryRow(ctx, `
				INSERT INTO map_place_candidates (source, point, name, name_key, kind, evidence, customers, samples)
				VALUES ('addresses', ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3, $4, $5, $6, $7, $8)
				RETURNING id`, c.Lat, c.Lng, c.Name, c.Key, c.Kind, c.Evidence, c.Customers, c.Samples).Scan(&id)
		} else {
			err = qerr
		}
		if err != nil {
			return 0, err
		}
		keep[id] = true
		stored++
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM map_place_candidates WHERE status = 'pending' AND source = 'addresses' AND id <> ALL($1::bigint[])`,
		keysOf(keep)); err != nil {
		return 0, err
	}
	return stored, tx.Commit(ctx)
}
