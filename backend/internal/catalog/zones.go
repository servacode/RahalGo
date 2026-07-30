package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// مناطق التغطية — مضلعات GeoJSON تُخزَّن في PostGIS، برسوم وحد أدنى لكل منطقة.

var ErrBadPolygon = httpx.NewError(http.StatusBadRequest, "invalid_polygon", "errors.invalid_polygon")

// Ring حلقة إحداثيات [lng, lat] — تُغلق تلقائياً إن لم تكن مغلقة.
type Ring [][2]float64

type Zone struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Polygon     Ring   `json:"polygon"`
	DeliveryFee int64  `json:"delivery_fee"`
	MinOrder    int64  `json:"min_order"`
	Active      bool   `json:"active"`
	SortOrder   int    `json:"sort_order"`
}

func ringToGeoJSON(r Ring) (string, error) {
	if len(r) < 3 {
		return "", ErrBadPolygon
	}
	if r[0] != r[len(r)-1] {
		r = append(r, r[0]) // إغلاق الحلقة
	}
	g := map[string]any{"type": "Polygon", "coordinates": []Ring{r}}
	b, err := json.Marshal(g)
	return string(b), err
}

func parsePolygon(raw []byte) Ring {
	var g struct {
		Coordinates []Ring `json:"coordinates"`
	}
	if err := json.Unmarshal(raw, &g); err != nil || len(g.Coordinates) == 0 {
		return Ring{}
	}
	return g.Coordinates[0]
}

func (s *Service) ListZones(ctx context.Context) ([]Zone, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, name, ST_AsGeoJSON(polygon::geometry), delivery_fee, min_order, active, sort_order
		FROM delivery_zones ORDER BY sort_order, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Zone{}
	for rows.Next() {
		var z Zone
		var raw []byte
		if err := rows.Scan(&z.ID, &z.Name, &raw, &z.DeliveryFee, &z.MinOrder, &z.Active, &z.SortOrder); err != nil {
			return nil, err
		}
		z.Polygon = parsePolygon(raw)
		out = append(out, z)
	}
	return out, rows.Err()
}

type ZoneInput struct {
	Name        *string `json:"name"`
	Polygon     *Ring   `json:"polygon"`
	DeliveryFee *int64  `json:"delivery_fee"`
	MinOrder    *int64  `json:"min_order"`
	Active      *bool   `json:"active"`
}

func (s *Service) CreateZone(ctx context.Context, actorID string, in ZoneInput, ip string) (*Zone, error) {
	if in.Name == nil || *in.Name == "" || in.Polygon == nil {
		return nil, ErrNameRequired
	}
	geo, err := ringToGeoJSON(*in.Polygon)
	if err != nil {
		return nil, err
	}
	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO delivery_zones (name, polygon, delivery_fee, min_order, sort_order)
		VALUES ($1, ST_GeomFromGeoJSON($2)::geography, COALESCE($3,0), COALESCE($4,0),
		        COALESCE((SELECT max(sort_order)+1 FROM delivery_zones), 1))
		RETURNING id`, *in.Name, geo, in.DeliveryFee, in.MinOrder).Scan(&id)
	if err != nil {
		return nil, ErrBadPolygon
	}
	s.audit(ctx, actorID, "admin.zone_create", "zone", id, ip)
	return s.zoneByID(ctx, id)
}

func (s *Service) UpdateZone(ctx context.Context, actorID, id string, in ZoneInput, ip string) (*Zone, error) {
	var geo *string
	if in.Polygon != nil {
		g, err := ringToGeoJSON(*in.Polygon)
		if err != nil {
			return nil, err
		}
		geo = &g
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE delivery_zones SET
			name         = COALESCE($2, name),
			polygon      = COALESCE(ST_GeomFromGeoJSON($3)::geography, polygon),
			delivery_fee = COALESCE($4, delivery_fee),
			min_order    = COALESCE($5, min_order),
			active       = COALESCE($6, active)
		WHERE id = $1`, id, in.Name, geo, in.DeliveryFee, in.MinOrder, in.Active)
	if err != nil {
		return nil, ErrBadPolygon
	}
	if tag.RowsAffected() == 0 {
		return nil, httpx.ErrNotFound
	}
	s.audit(ctx, actorID, "admin.zone_update", "zone", id, ip)
	return s.zoneByID(ctx, id)
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

func (s *Service) zoneByID(ctx context.Context, id string) (*Zone, error) {
	var z Zone
	var raw []byte
	err := s.db.QueryRow(ctx, `
		SELECT id, name, ST_AsGeoJSON(polygon::geometry), delivery_fee, min_order, active, sort_order
		FROM delivery_zones WHERE id = $1`, id).
		Scan(&z.ID, &z.Name, &raw, &z.DeliveryFee, &z.MinOrder, &z.Active, &z.SortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	z.Polygon = parsePolygon(raw)
	return &z, nil
}

// ZoneForPoint يعيد المنطقة الفعالة التي تحوي النقطة (لحساب رسوم التوصيل لاحقاً).
func (s *Service) ZoneForPoint(ctx context.Context, lng, lat float64) (*Zone, error) {
	var id string
	err := s.db.QueryRow(ctx, `
		SELECT id FROM delivery_zones
		WHERE active AND ST_Covers(polygon, ST_SetSRID(ST_MakePoint($1,$2),4326)::geography)
		ORDER BY sort_order LIMIT 1`, lng, lat).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.zoneByID(ctx, id)
}
