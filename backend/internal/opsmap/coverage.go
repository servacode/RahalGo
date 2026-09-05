package opsmap

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **التغطية — `MAP-3`**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا يُكسَر ما يعمل (البند ١٤)
//
// **`LEGACY_RADIUS` نوعٌ أوّلُ الدرجة لا حالٌ انتقاليّة** — ومنطقةٌ
// دائريّةٌ تبقى دائرةً ما دام صاحبُها لم يطلب غيرَ ذلك.

// ErrBadGeometry شكلٌ لا يُقبَل.
var ErrBadGeometry = httpx.NewError(http.StatusBadRequest,
	"invalid_geometry", "errors.invalid_zone")

// Zone منطقةُ تغطيةٍ كما تراها الخريطة.
type Zone struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Shape  string `json:"shape"`
	Active bool   `json:"active"`

	// Lat/Lng **مركزُ الدائرة، أو مركزُ ثقل المضلَّع.**
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	RadiusM int     `json:"radius_m"`

	// Area **هندسةُ المضلَّع بصيغة GeoJSON** — وفارغةٌ للدوائر.
	Area json.RawMessage `json:"area,omitempty"`

	DeliveryFee int64   `json:"delivery_fee"`
	MinOrder    int64   `json:"min_order"`
	CityID      *string `json:"city_id,omitempty"`
	CityName    *string `json:"city,omitempty"`
}

// Ring حلقةٌ من نقاطٍ — `[[lng,lat], …]`.
type Ring [][2]float64

// ValidateRing يفحص حلقةً قبل أن تلمس القاعدة (البند ١٣).
//
// # ولماذا يُفحَص هنا وفي القاعدة
//
// **الفحصُ هنا يعطي سبباً مقروءاً** — «المضلَّعُ يحتاج ثلاثَ نقاط».
// **والقاعدةُ تعطي `SQLSTATE`** لا يقرؤه من يرسم. **وما يُقبَل هنا
// يُفحَص هناك أيضاً** — فلا يُصدَّق المُرسِل.
func ValidateRing(r Ring) error {
	if len(r) < 3 {
		return fmt.Errorf("المضلَّعُ يحتاج ثلاثَ نقاطٍ على الأقلّ — ووصلت %d", len(r))
	}
	if len(r) > 2000 {
		return fmt.Errorf("مضلَّعٌ بـ%d نقطةٍ — وهذا رسمٌ لا يُقرأ ولا يُخزَّن", len(r))
	}
	for i, p := range r {
		// **وإحداثيّاتٌ خارجَ الأرض تُردّ** — والصفرُ في خليج غينيا
		// **يقع داخلَ المدى**، فيُترك للقاعدة أن تحكم على معناه.
		if p[0] < -180 || p[0] > 180 || p[1] < -90 || p[1] > 90 {
			return fmt.Errorf("نقطةٌ %d خارجَ الأرض: %v", i+1, p)
		}
	}
	return nil
}

// wkt يبني `POLYGON` من حلقةٍ — **ويُغلقها إن لم تُغلَق.**
//
// **ومن رسم بيده لا يعيد النقطةَ الأولى** — والصيغةُ تشترطها.
func wkt(r Ring) string {
	pts := make([]string, 0, len(r)+1)
	for _, p := range r {
		pts = append(pts, fmt.Sprintf("%.7f %.7f", p[0], p[1]))
	}
	if r[0] != r[len(r)-1] {
		pts = append(pts, fmt.Sprintf("%.7f %.7f", r[0][0], r[0][1]))
	}
	return "POLYGON((" + strings.Join(pts, ",") + "))"
}

const zoneSelect = `
	SELECT z.id::text, z.name, z.shape, z.active,
	       ST_Y(z.center::geometry), ST_X(z.center::geometry), z.radius_m,
	       CASE WHEN z.area IS NULL THEN NULL
	            ELSE ST_AsGeoJSON(z.area::geometry) END,
	       z.delivery_fee, z.min_order, z.city_id::text, c.name
	FROM delivery_zones z
	LEFT JOIN cities c ON c.id = z.city_id`

// Zones مناطقُ التغطية كلُّها — **الدوائرُ والمضلَّعاتُ معاً.**
//
// **وهي الشاشةُ الوحيدةُ التي تراهما** — وشاشةُ الإعدادات القديمةُ
// ترسم الدوائرَ وحدَها لأنّها لا تعرف غيرَها.
func Zones(ctx context.Context, q Querier, box *BBox, onlyActive bool) ([]Zone, error) {
	var where []string
	var args []any
	if onlyActive {
		where = append(where, "z.active")
	}
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		// **والدائرةُ تُقاس بمركزها والمضلَّعُ بمساحته.**
		where = append(where, fmt.Sprintf("(%s OR %s)",
			box.SQL("z.center", len(args)+1), box.SQL("COALESCE(z.area, z.center)", len(args)+1)))
		args = append(args, box.Args()...)
	}
	sql := zoneSelect
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY z.sort_order, z.name"

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Zone{}
	for rows.Next() {
		var z Zone
		var area *string
		if err := rows.Scan(&z.ID, &z.Name, &z.Shape, &z.Active,
			&z.Lat, &z.Lng, &z.RadiusM, &area,
			&z.DeliveryFee, &z.MinOrder, &z.CityID, &z.CityName); err != nil {
			return nil, err
		}
		if area != nil {
			z.Area = json.RawMessage(*area)
		}
		out = append(out, z)
	}
	return out, rows.Err()
}

// ZoneInput ما يُرسَل لإنشاء منطقةٍ مضلَّعةٍ أو تعديلها.
type ZoneInput struct {
	Name        string  `json:"name"`
	Ring        Ring    `json:"ring"`
	DeliveryFee int64   `json:"delivery_fee"`
	MinOrder    int64   `json:"min_order"`
	CityID      *string `json:"city_id"`
	Active      *bool   `json:"active"`
}

// SavePolygonZone يُنشئ منطقةً مضلَّعةً أو يعدّل مضلَّعَ قائمة.
//
// # ولا يُثق بما يصل (البند ١٣)
//
// **يُفحَص العددُ والمدى هنا، ثمّ تُسأل `PostGIS` عن صحّة الشكل نفسِه**:
// **`ST_IsValid` تكشف التقاطعَ مع النفس** — وهي ما لا يستطيع فحصٌ نصّيّ.
//
// # والمركزُ يُحسَب لا يُطلَب
//
// **`center` عمودٌ إلزاميٌّ منذ ٢٠٠٩** — **ولا يُسأل عنه من يرسم شكلاً.**
// فيُؤخذ مركزُ ثقل المضلَّع، **وهو ترتيبُ التداخل في `ZoneAt`.**
//
// **ونصفُ القطر يُحشى بأصغرِ قيمةٍ يقبلها القيد** — **ولا يُقرأ لمضلَّعٍ
// أبداً**: `ZoneAt` تفرّق بالشكل، **وشاشةُ الدوائر لا تراه أصلاً.**
func SavePolygonZone(ctx context.Context, e Execer, id string, in ZoneInput) (string, error) {
	if strings.TrimSpace(in.Name) == "" {
		return "", ErrBadGeometry
	}
	if err := ValidateRing(in.Ring); err != nil {
		return "", httpx.NewError(http.StatusBadRequest, "invalid_geometry", err.Error())
	}
	poly := wkt(in.Ring)

	// **وصحّةُ الشكل تُسأل قبل الكتابة** — فلا يُخزَّن مضلَّعٌ يتقاطع
	// مع نفسه ثمّ يُفاجئ `ST_Covers` بجوابٍ لا يُفسَّر.
	var valid bool
	var reason string
	if err := e.QueryRow(ctx,
		`SELECT ST_IsValid(g), ST_IsValidReason(g)
		   FROM (SELECT ST_GeomFromText($1, 4326) AS g) s`,
		poly).Scan(&valid, &reason); err != nil {
		return "", httpx.NewError(http.StatusBadRequest, "invalid_geometry", err.Error())
	}
	if !valid {
		return "", httpx.NewError(http.StatusBadRequest, "invalid_geometry", reason)
	}

	active := true
	if in.Active != nil {
		active = *in.Active
	}

	if id == "" {
		var newID string
		err := e.QueryRow(ctx, `
			INSERT INTO delivery_zones
			    (name, shape, area, center, radius_m, delivery_fee, min_order, active, city_id, sort_order)
			VALUES ($1, 'polygon',
			        ST_Multi(ST_GeomFromText($2, 4326))::geography,
			        ST_Centroid(ST_GeomFromText($2, 4326))::geography,
			        100, $3, $4, $5, $6::uuid,
			        COALESCE((SELECT max(sort_order)+1 FROM delivery_zones), 1))
			RETURNING id::text`,
			in.Name, poly, in.DeliveryFee, in.MinOrder, active, in.CityID).Scan(&newID)
		return newID, err
	}

	tag, err := e.Exec(ctx, `
		UPDATE delivery_zones SET
		    name = $2,
		    shape = 'polygon',
		    area = ST_Multi(ST_GeomFromText($3, 4326))::geography,
		    center = ST_Centroid(ST_GeomFromText($3, 4326))::geography,
		    delivery_fee = $4, min_order = $5, active = $6, city_id = $7::uuid
		WHERE id = $1::uuid`,
		id, in.Name, poly, in.DeliveryFee, in.MinOrder, active, in.CityID)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", httpx.ErrNotFound
	}
	return id, nil
}

// SetZoneActive يُفعّل منطقةً أو يوقفها.
//
// **ولا حذف** (وهي قاعدةُ المنصّة في كلّ سجلٍّ يُنسَب إليه شيء):
// **منطقةٌ حُذفت تترك طلباتٍ تشير إلى عدم.**
func SetZoneActive(ctx context.Context, e Execer, id string, active bool) error {
	tag, err := e.Exec(ctx,
		`UPDATE delivery_zones SET active = $2 WHERE id = $1::uuid`, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return httpx.ErrNotFound
	}
	return nil
}
