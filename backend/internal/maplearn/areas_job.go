package maplearn

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// بعدٌ يُعدّ عنده اسمُ الحيّ الموجودُ في الخريطة هو نفسَه.
	hoodKnownRadiusM = 3000.0
	// بعدُ مطابقة قرارٍ سابقٍ على الاسم نفسه.
	areaDecidedRadiusM = 1500.0
	// مطابقةُ مقترَحٍ منتظرٍ بالقرب فتبقى أرقامُه.
	areaKeepRadiusM = 1500.0
)

// AreaResult نتيجةُ جولة الأحياء والشوارع.
type AreaResult struct {
	Neighbourhoods int
	Streets        int
}

// loadAddressEvidence عناوينُ الزبائن المسلَّمة ونقاطُ تسليمها (معرّفُ الزبون فقط — لا اسمَ ولا هاتف).
func loadAddressEvidence(ctx context.Context, pg *pgxpool.Pool) ([]AddrEvidence, error) {
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
		return nil, err
	}
	defer rows.Close()
	var ev []AddrEvidence
	for rows.Next() {
		var e AddrEvidence
		if err := rows.Scan(&e.Customer, &e.Text, &e.Lat, &e.Lng); err != nil {
			return nil, err
		}
		ev = append(ev, e)
	}
	return ev, rows.Err()
}

// RunAreas **جولةُ الأحياء والشوارع**: تقرأ العناوينَ وتحفظ المرشَّحين، وتكتب أعدادَها في آخر سطرٍ من
// `map_place_runs` (سطرِ جولة الأماكن التي سبقتها). `loc` فارغٌ = تُترك الشوارعُ ويعمل الأحياء.
func RunAreas(ctx context.Context, pg *pgxpool.Pool, loc Locator, p AreaParams) (res AreaResult, err error) {
	if !placeMu.TryLock() {
		return res, errors.New("maplearn: جولةُ أماكنَ تعمل الآن")
	}
	defer placeMu.Unlock()
	p = p.filled()
	defer func() {
		if err == nil {
			_, _ = pg.Exec(context.WithoutCancel(ctx), `
				UPDATE map_place_runs SET neighbourhoods = $1, streets = $2
				 WHERE id = (SELECT max(id) FROM map_place_runs)`, res.Neighbourhoods, res.Streets)
		}
	}()
	ev, err := loadAddressEvidence(ctx, pg)
	if err != nil {
		return res, err
	}
	if res.Neighbourhoods, err = saveHoods(ctx, pg, ClusterHoods(ev, p)); err != nil {
		return res, err
	}
	if loc == nil {
		return res, nil
	}
	res.Streets, err = saveStreets(ctx, pg, loc, GroupStreets(ev, p), p)
	return res, err
}

func hullWKT(h [][2]float64) *string {
	if len(h) < 4 {
		return nil
	}
	parts := make([]string, len(h))
	for i, c := range h {
		parts[i] = fmt.Sprintf("%.7f %.7f", c[0], c[1])
	}
	s := "POLYGON((" + strings.Join(parts, ",") + "))"
	return &s
}

func linesWKT(lines [][][2]float64) *string {
	var ls []string
	for _, l := range lines {
		if len(l) < 2 {
			continue
		}
		parts := make([]string, len(l))
		for i, c := range l {
			parts[i] = fmt.Sprintf("%.7f %.7f", c[0], c[1])
		}
		ls = append(ls, "("+strings.Join(parts, ",")+")")
	}
	if len(ls) == 0 {
		return nil
	}
	s := "MULTILINESTRING(" + strings.Join(ls, ",") + ")"
	return &s
}

// hoodKnown: أفي الخريطة (معالمُ `place=*`) أو في ما أُقرّ حيٌّ بالاسم نفسه، أو قرارٌ سابقٌ عليه؟
func hoodKnown(ctx context.Context, q pgx.Tx, c HoodCluster) (bool, error) {
	rows, err := q.Query(ctx, `
		SELECT name FROM map_landmarks
		 WHERE kind LIKE 'place=%' AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, $3)`,
		c.Lat, c.Lng, hoodKnownRadiusM)
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
	return decidedArea(ctx, q, "neighbourhood", "neighbourhoods", c.Key, c.Lat, c.Lng)
}

// decidedArea: اسمٌ أُقرّ في `map_area_names` أو قُرّر عليه (رفضاً) قربَ النقطة.
func decidedArea(ctx context.Context, q pgx.Tx, kind, source, key string, lat, lng float64) (bool, error) {
	var ok bool
	err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM map_area_names
		                WHERE kind = $1 AND name_key = $3
		                  AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography, $6))
		    OR EXISTS (SELECT 1 FROM map_place_candidates
		                WHERE source = $2 AND status <> 'pending' AND name_key = $3
		                  AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($5, $4), 4326)::geography, $6))`,
		kind, source, key, lat, lng, areaDecidedRadiusM).Scan(&ok)
	return ok, err
}

// upsertArea يحدّث المقترَحَ المنتظرَ القريبَ بالاسم نفسه أو يضيف جديداً، ويردّ معرّفَه.
func upsertArea(ctx context.Context, tx pgx.Tx, source, name, key string, lat, lng float64, geomWKT *string,
	radius float64, ways []int64, customers, evidence int, samples []string, keep map[int64]bool) (int64, error) {
	if ways == nil {
		ways = []int64{}
	}
	var id int64
	qerr := tx.QueryRow(ctx, `
		SELECT id FROM map_place_candidates
		 WHERE status = 'pending' AND source = $1 AND name_key = $4 AND id <> ALL($5::bigint[])
		   AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $6)
		 ORDER BY ST_Distance(point, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography), id LIMIT 1`,
		source, lat, lng, key, keysOf(keep), areaKeepRadiusM).Scan(&id)
	if qerr == nil {
		_, err := tx.Exec(ctx, `
			UPDATE map_place_candidates
			   SET point = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, name = $4,
			       geom = CASE WHEN $5::text IS NULL THEN NULL ELSE ST_GeogFromText($5::text) END,
			       radius_m = $6, way_ids = $7, customers = $8, evidence = $9, samples = $10, updated_at = now()
			 WHERE id = $1`, id, lat, lng, name, geomWKT, radius, ways, customers, evidence, samples)
		return id, err
	}
	if !errors.Is(qerr, pgx.ErrNoRows) {
		return 0, qerr
	}
	kind := "neighbourhood"
	if source == "streets" {
		kind = "street"
	}
	err := tx.QueryRow(ctx, `
		INSERT INTO map_place_candidates (source, point, name, name_key, kind, geom, radius_m, way_ids, customers, evidence, samples)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4, $5, $6,
		        CASE WHEN $7::text IS NULL THEN NULL ELSE ST_GeogFromText($7::text) END, $8, $9, $10, $11, $12)
		RETURNING id`, source, lat, lng, name, key, kind, geomWKT, radius, ways, customers, evidence, samples).Scan(&id)
	return id, err
}

func saveHoods(ctx context.Context, pg *pgxpool.Pool, clusters []HoodCluster) (int, error) {
	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	keep := map[int64]bool{}
	stored := 0
	for _, c := range clusters {
		kn, err := hoodKnown(ctx, tx, c)
		if err != nil {
			return 0, err
		}
		if kn {
			continue
		}
		id, err := upsertArea(ctx, tx, "neighbourhoods", c.Name, c.Key, c.Lat, c.Lng, hullWKT(c.Hull), c.RadiusM,
			nil, c.Customers, c.Evidence, c.Samples, keep)
		if err != nil {
			return 0, err
		}
		keep[id] = true
		stored++
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM map_place_candidates WHERE status = 'pending' AND source = 'neighbourhoods' AND id <> ALL($1::bigint[])`,
		keysOf(keep)); err != nil {
		return 0, err
	}
	return stored, tx.Commit(ctx)
}

func saveStreets(ctx context.Context, pg *pgxpool.Pool, loc Locator, groups []StreetGroup, p AreaParams) (int, error) {
	used := 0
	incomplete := false
	var resolved []*StreetResolved
	for _, g := range groups {
		r, err := ResolveStreet(ctx, loc, g, p, &used)
		if errors.Is(err, ErrLocateBudget) {
			incomplete = true
			break
		}
		if err != nil {
			// تعثّرُ المحرّك لاسمٍ لا يُسقط الجولة، ولا يُحذف بسببه مقترَحٌ قديم.
			incomplete = true
			continue
		}
		if r != nil {
			resolved = append(resolved, r)
		}
	}
	tx, err := pg.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	keep := map[int64]bool{}
	stored := 0
	for _, r := range resolved {
		if r.Named {
			continue
		}
		kn, err := decidedArea(ctx, tx, "street", "streets", r.Key, r.Lat, r.Lng)
		if err != nil {
			return 0, err
		}
		if kn {
			continue
		}
		id, err := upsertArea(ctx, tx, "streets", r.Name, r.Key, r.Lat, r.Lng, linesWKT(r.Lines), 0, r.WayIDs,
			r.Customers, r.Evidence, r.Samples, keep)
		if err != nil {
			return 0, err
		}
		keep[id] = true
		stored++
	}
	if !incomplete {
		if _, err := tx.Exec(ctx, `
			DELETE FROM map_place_candidates WHERE status = 'pending' AND source = 'streets' AND id <> ALL($1::bigint[])`,
			keysOf(keep)); err != nil {
			return 0, err
		}
	}
	return stored, tx.Commit(ctx)
}
