package opsmap

import (
	"context"
	"strings"
)

// ══════════════════════════════════════════════════════════════════════
// **البحثُ في الخريطة — البند ٣٦**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا يُبحَث فيما لا يملكه من يبحث
//
// **الصلاحيّاتُ تُمرَّر فيُقصَّ ما لا يراه** — **ومن بحث فوجد اسمَ
// سائقٍ لا يملك رؤيةَ مواضعه عرف أنّه موجود**، وهو تسريبٌ صغيرٌ لا
// يُتساهَل فيه.

// Hit نتيجةُ بحثٍ واحدة.
type Hit struct {
	Kind  string   `json:"kind"`
	ID    string   `json:"id"`
	Label string   `json:"label"`
	Lat   *float64 `json:"lat,omitempty"`
	Lng   *float64 `json:"lng,omitempty"`
}

// Search يبحث في طبقات الخريطة بحسب صلاحيّات الباحث.
func Search(ctx context.Context, q Querier, caps []string, term string) ([]Hit, error) {
	term = strings.TrimSpace(term)
	// **والحرفُ العربيُّ بايتان** — **و`len` بالبايت يمرّر «ا» وحدَها
	// فيُمسَح الجدولُ كلُّه لحرف.** (قِيس ٢٠٢٦-٠٩-٠٦: ردَّ ثمانيَ نتائج.)
	if len([]rune(term)) < 2 {
		return []Hit{}, nil
	}
	like := "%" + term + "%"
	out := []Hit{}

	add := func(sql, kind string, args ...any) error {
		rows, err := q.Query(ctx, sql, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			h := Hit{Kind: kind}
			if err := rows.Scan(&h.ID, &h.Label, &h.Lat, &h.Lng); err != nil {
				return err
			}
			out = append(out, h)
		}
		return rows.Err()
	}

	if Allows(caps, PermViewDrivers) {
		if err := add(`
			SELECT u.id::text, u.full_name,
			       ST_Y(u.last_location::geometry), ST_X(u.last_location::geometry)
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
			WHERE u.full_name ILIKE $1 ORDER BY u.full_name LIMIT 10`, "driver", like); err != nil {
			return nil, err
		}
	}
	if Allows(caps, PermViewMerchants) {
		if err := add(`
			SELECT m.id::text, m.name,
			       ST_Y(m.location::geometry), ST_X(m.location::geometry)
			FROM merchants m
			WHERE m.name ILIKE $1 ORDER BY m.name LIMIT 10`, "merchant", like); err != nil {
			return nil, err
		}
	}
	if Allows(caps, PermViewOrders) {
		if err := add(`
			SELECT o.id::text, '#' || o.number::text,
			       ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry)
			FROM orders o
			WHERE o.number::text ILIKE $1 AND o.closed_at IS NULL
			ORDER BY o.number DESC LIMIT 10`, "order", like); err != nil {
			return nil, err
		}
	}
	if Allows(caps, PermViewMap) {
		if err := add(`
			SELECT b.id::text, b.name,
			       ST_Y(b.location::geometry), ST_X(b.location::geometry)
			FROM branches b WHERE b.name ILIKE $1 ORDER BY b.name LIMIT 10`, "branch", like); err != nil {
			return nil, err
		}
		if err := add(`
			SELECT a.id::text, a.name, NULL::float8, NULL::float8
			FROM operational_areas a WHERE a.name ILIKE $1 ORDER BY a.name LIMIT 10`,
			"area", like); err != nil {
			return nil, err
		}
	}
	if Allows(caps, PermViewRepActivity) {
		if err := add(`
			SELECT u.id::text, u.full_name, NULL::float8, NULL::float8
			FROM users u
			JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'sales'
			WHERE u.full_name ILIKE $1 ORDER BY u.full_name LIMIT 10`, "rep", like); err != nil {
			return nil, err
		}
	}
	return out, nil
}
