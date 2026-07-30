package catalog

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// مناطق التغطية — نموذج الدوائر (قرار مثبّت): اسم + مركز + نصف قطر قابل
// للتمديد والتقليص، برسم توصيل وحد أدنى لكل منطقة. أبسط إدارياً من رسم
// المضلعات ويناسب نموذج "مركز المدينة الآن، فروع لاحقاً".

var ErrBadZone = httpx.NewError(http.StatusBadRequest, "invalid_zone", "errors.invalid_zone")

type Zone struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	RadiusM     int     `json:"radius_m"`
	DeliveryFee int64   `json:"delivery_fee"`
	MinOrder    int64   `json:"min_order"`
	Active      bool    `json:"active"`
	SortOrder   int     `json:"sort_order"`
}

const zoneCols = `id, name, ST_Y(center::geometry), ST_X(center::geometry),
	radius_m, delivery_fee, min_order, active, sort_order`

func scanZone(row pgx.Row) (*Zone, error) {
	var z Zone
	err := row.Scan(&z.ID, &z.Name, &z.Lat, &z.Lng, &z.RadiusM,
		&z.DeliveryFee, &z.MinOrder, &z.Active, &z.SortOrder)
	if err != nil {
		return nil, err
	}
	return &z, nil
}

func (s *Service) ListZones(ctx context.Context) ([]Zone, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+zoneCols+` FROM delivery_zones ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Zone{}
	for rows.Next() {
		z, err := scanZone(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *z)
	}
	return out, rows.Err()
}

type ZoneInput struct {
	Name        *string  `json:"name"`
	Lat         *float64 `json:"lat"`
	Lng         *float64 `json:"lng"`
	RadiusM     *int     `json:"radius_m"`
	DeliveryFee *int64   `json:"delivery_fee"`
	MinOrder    *int64   `json:"min_order"`
	Active      *bool    `json:"active"`
}

func validRadius(r *int) bool { return r == nil || (*r >= 100 && *r <= 50000) }

func (s *Service) CreateZone(ctx context.Context, actorID string, in ZoneInput, ip string) (*Zone, error) {
	if in.Name == nil || *in.Name == "" || in.Lat == nil || in.Lng == nil || !validRadius(in.RadiusM) {
		return nil, ErrBadZone
	}
	z, err := scanZone(s.db.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, center, radius_m, delivery_fee, min_order, sort_order)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography,
		        COALESCE($4, 2000), COALESCE($5, 0), COALESCE($6, 0),
		        COALESCE((SELECT max(sort_order)+1 FROM delivery_zones), 1))
		RETURNING `+zoneCols,
		*in.Name, *in.Lat, *in.Lng, in.RadiusM, in.DeliveryFee, in.MinOrder))
	if err != nil {
		return nil, ErrBadZone
	}
	s.audit(ctx, actorID, "admin.zone_create", "zone", z.ID, ip)
	return z, nil
}

func (s *Service) UpdateZone(ctx context.Context, actorID, id string, in ZoneInput, ip string) (*Zone, error) {
	if !validRadius(in.RadiusM) {
		return nil, ErrBadZone
	}
	z, err := scanZone(s.db.QueryRow(ctx, `
		UPDATE delivery_zones SET
			name         = COALESCE($2, name),
			center       = COALESCE(
				CASE WHEN $3::float8 IS NOT NULL AND $4::float8 IS NOT NULL
				     THEN ST_SetSRID(ST_MakePoint($4::float8, $3::float8), 4326)::geography END,
				center),
			radius_m     = COALESCE($5, radius_m),
			delivery_fee = COALESCE($6, delivery_fee),
			min_order    = COALESCE($7, min_order),
			active       = COALESCE($8, active)
		WHERE id = $1
		RETURNING `+zoneCols,
		id, in.Name, in.Lat, in.Lng, in.RadiusM, in.DeliveryFee, in.MinOrder, in.Active))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, ErrBadZone
	}
	s.audit(ctx, actorID, "admin.zone_update", "zone", id, ip)
	return z, nil
}

func (s *Service) DeleteZone(ctx context.Context, actorID, id, ip string) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM delivery_zones WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.zone_delete", "zone", id, ip)
	return nil
}

// ZoneForPoint يعيد المنطقة الفعالة التي تغطي النقطة (الأقرب مركزاً عند التداخل)
// — تُستخدم لحساب رسوم التوصيل من دبوس الزبون.
func (s *Service) ZoneForPoint(ctx context.Context, lat, lng float64) (*Zone, error) {
	z, err := scanZone(s.db.QueryRow(ctx, `
		SELECT `+zoneCols+` FROM delivery_zones
		WHERE active
		  AND ST_DWithin(center, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography, radius_m)
		ORDER BY ST_Distance(center, ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography)
		LIMIT 1`, lat, lng))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	return z, err
}
