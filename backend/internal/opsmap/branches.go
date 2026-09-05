package opsmap

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **الفروعُ والمناطقُ التشغيليّة — `MAP-4`**
// ══════════════════════════════════════════════════════════════════════
//
// # والجغرافيا لا تصير فرعاً (البند ١٩)
//
// **`Geography ≠ Branch`** — والمنطقةُ الإداريّةُ تقسيمُ الدولة،
// **والفرعُ قرارُ عملٍ نتّخذه.** **ولا مسارَ في هذه الحزمة يحوّل
// `district` إلى `branch`** — ويُثبته `TestBranch_DistrictNeverBecomesBranch`.

// ErrBadBranch مدخلٌ لا يبني فرعاً.
var ErrBadBranch = httpx.NewError(http.StatusBadRequest,
	"invalid_branch", "errors.validation")

// Branch فرعٌ كما تراه الخريطة.
type Branch struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Status string `json:"status"`

	CityID   string  `json:"city_id"`
	CityName *string `json:"city,omitempty"`

	ParentID   *string `json:"parent_id,omitempty"`
	ParentName *string `json:"parent,omitempty"`

	Lat *float64 `json:"lat,omitempty"`
	Lng *float64 `json:"lng,omitempty"`

	// Areas **عددُ المناطق التشغيليّة تحته** — ورقمٌ يقول «أهو حيٌّ؟».
	Areas int `json:"areas"`
}

// Area منطقةٌ تشغيليّة.
type Area struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`

	CityID   string  `json:"city_id"`
	CityName *string `json:"city,omitempty"`

	BranchID   *string `json:"branch_id,omitempty"`
	BranchName *string `json:"branch,omitempty"`

	// ZoneID **منطقةُ التغطية المرتبطة** — **وقد لا تكون**، فالربطُ
	// اختياريٌّ: هذه تنظيمٌ وتلك شكلٌ يُسعَّر.
	ZoneID   *string `json:"zone_id,omitempty"`
	ZoneName *string `json:"zone,omitempty"`
}

// Branches فروعُ المشهد.
func Branches(ctx context.Context, q Querier, box *BBox, cityID string) ([]Branch, error) {
	var where []string
	var args []any
	if cityID != "" {
		where = append(where, fmt.Sprintf("b.city_id = $%d", len(args)+1))
		args = append(args, cityID)
	}
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		// **وفرعٌ بلا موقعٍ يُردّ دائماً** — يُقرأ في القائمة ولا يُرسَم.
		where = append(where, fmt.Sprintf("(b.location IS NULL OR %s)",
			box.SQL("b.location", len(args)+1)))
		args = append(args, box.Args()...)
	}
	sql := `
		SELECT b.id::text, b.name, b.type, b.status,
		       b.city_id::text, c.name,
		       b.parent_id::text, p.name,
		       ST_Y(b.location::geometry), ST_X(b.location::geometry),
		       (SELECT count(*) FROM operational_areas a WHERE a.branch_id = b.id)
		FROM branches b
		LEFT JOIN cities c ON c.id = b.city_id
		LEFT JOIN branches p ON p.id = b.parent_id`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY c.name, b.type, b.sort_order, b.name"

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Branch{}
	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.ID, &b.Name, &b.Type, &b.Status,
			&b.CityID, &b.CityName, &b.ParentID, &b.ParentName,
			&b.Lat, &b.Lng, &b.Areas); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BranchInput ما يُرسَل لإنشاء فرعٍ أو تعديله (البند ٢١).
type BranchInput struct {
	Name     string   `json:"name"`
	Type     string   `json:"type"`
	CityID   string   `json:"city_id"`
	ParentID *string  `json:"parent_id"`
	Status   string   `json:"status"`
	Lat      *float64 `json:"lat"`
	Lng      *float64 `json:"lng"`
}

// SaveBranch يُنشئ فرعاً أو يعدّله.
//
// # ولا يُصدَّق ما يصل
//
// **الهرمُ محروسٌ في القاعدة** (`branches_parent_ck`) — **وهذا فحصٌ
// ثانٍ يعطي سبباً مقروءاً** بدل `SQLSTATE 23514`.
//
// **وفرعٌ رئيسيٌّ ثانٍ في مدينةٍ يردّه القيدُ الفريدُ الجزئيّ** —
// **وهو الحارسُ الحقّ**: معالجٌ يفحص ثمّ يُدرج يُهزَم بضغطتين متزامنتين.
func SaveBranch(ctx context.Context, e Execer, id string, in BranchInput) (string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || in.CityID == "" {
		return "", ErrBadBranch
	}
	kind := in.Type
	if kind == "" {
		kind = "primary"
	}
	if kind != "primary" && kind != "sub" {
		return "", ErrBadBranch
	}
	// **وفرعُ المدينة لا أبَ له، والفرعيُّ لا بدَّ له من أب.**
	if kind == "primary" && in.ParentID != nil && *in.ParentID != "" {
		return "", httpx.NewError(http.StatusBadRequest, "invalid_branch",
			"فرعُ المدينة الرئيسيُّ لا أبَ له")
	}
	if kind == "sub" && (in.ParentID == nil || *in.ParentID == "") {
		return "", httpx.NewError(http.StatusBadRequest, "invalid_branch",
			"الفرعُ الفرعيُّ يحتاج فرعاً أباً")
	}
	status := in.Status
	if status == "" {
		status = "active"
	}
	if status != "active" && status != "inactive" {
		return "", ErrBadBranch
	}
	if in.Lat != nil && in.Lng != nil {
		if *in.Lat < -90 || *in.Lat > 90 || *in.Lng < -180 || *in.Lng > 180 {
			return "", httpx.NewError(http.StatusBadRequest, "invalid_branch",
				"موقعٌ خارجَ الأرض")
		}
	}

	if id == "" {
		var newID string
		err := e.QueryRow(ctx, `
			INSERT INTO branches (name, type, city_id, parent_id, status, location)
			VALUES ($1, $2, $3::uuid, $4::uuid, $5,
			        CASE WHEN $6::float8 IS NULL OR $7::float8 IS NULL THEN NULL
			             ELSE ST_SetSRID(ST_MakePoint($7, $6), 4326)::geography END)
			RETURNING id::text`,
			name, kind, in.CityID, in.ParentID, status, in.Lat, in.Lng).Scan(&newID)
		return newID, err
	}

	tag, err := e.Exec(ctx, `
		UPDATE branches SET
		    name = $2, type = $3, city_id = $4::uuid, parent_id = $5::uuid, status = $6,
		    location = CASE WHEN $7::float8 IS NULL OR $8::float8 IS NULL THEN location
		                    ELSE ST_SetSRID(ST_MakePoint($8, $7), 4326)::geography END
		WHERE id = $1::uuid`,
		id, name, kind, in.CityID, in.ParentID, status, in.Lat, in.Lng)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", httpx.ErrNotFound
	}
	return id, nil
}

// Areas مناطقُ العمل التشغيليّة.
func Areas(ctx context.Context, q Querier, cityID, branchID string) ([]Area, error) {
	var where []string
	var args []any
	if cityID != "" {
		where = append(where, fmt.Sprintf("a.city_id = $%d", len(args)+1))
		args = append(args, cityID)
	}
	if branchID != "" {
		where = append(where, fmt.Sprintf("a.branch_id = $%d", len(args)+1))
		args = append(args, branchID)
	}
	sql := `
		SELECT a.id::text, a.name, a.active,
		       a.city_id::text, c.name,
		       a.branch_id::text, b.name,
		       a.zone_id::text, z.name
		FROM operational_areas a
		LEFT JOIN cities c ON c.id = a.city_id
		LEFT JOIN branches b ON b.id = a.branch_id
		LEFT JOIN delivery_zones z ON z.id = a.zone_id`
	if len(where) > 0 {
		sql += " WHERE " + strings.Join(where, " AND ")
	}
	sql += " ORDER BY c.name, a.sort_order, a.name"

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Area{}
	for rows.Next() {
		var a Area
		if err := rows.Scan(&a.ID, &a.Name, &a.Active,
			&a.CityID, &a.CityName, &a.BranchID, &a.BranchName,
			&a.ZoneID, &a.ZoneName); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AreaInput ما يُرسَل لإنشاء منطقةٍ تشغيليّةٍ أو تعديلها (البند ٢٢).
type AreaInput struct {
	Name     string  `json:"name"`
	CityID   string  `json:"city_id"`
	BranchID *string `json:"branch_id"`
	ZoneID   *string `json:"zone_id"`
	Active   *bool   `json:"active"`
}

// SaveArea يُنشئ منطقةً تشغيليّةً أو يعدّلها.
func SaveArea(ctx context.Context, e Execer, id string, in AreaInput) (string, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" || in.CityID == "" {
		return "", ErrBadBranch
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	if id == "" {
		var newID string
		err := e.QueryRow(ctx, `
			INSERT INTO operational_areas (name, city_id, branch_id, zone_id, active)
			VALUES ($1, $2::uuid, $3::uuid, $4::uuid, $5)
			RETURNING id::text`,
			name, in.CityID, in.BranchID, in.ZoneID, active).Scan(&newID)
		return newID, err
	}
	tag, err := e.Exec(ctx, `
		UPDATE operational_areas SET
		    name = $2, city_id = $3::uuid, branch_id = $4::uuid,
		    zone_id = $5::uuid, active = $6
		WHERE id = $1::uuid`,
		id, name, in.CityID, in.BranchID, in.ZoneID, active)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() == 0 {
		return "", httpx.ErrNotFound
	}
	return id, nil
}
