package opsmap

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **نشاطُ المندوبين جغرافيّاً — `MAP-5`**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا مناطقَ إلزاميّةً للمندوب (البند ٢٤)
//
// **العقدُ المعتمد**: **المندوبُ لا يملك منطقةً تمنعه من العمل خارجها.**
//
// **فهذه الطبقةُ تُلاحِظ ولا تحكم** — **ولا مسارَ فيها يردّ تسجيلاً ولا
// تحويلَ متجرٍ بسبب موضعٍ جغرافيّ.** **ولا استعلامَ هنا يُنادى من مسار
// إنشاء** — ويُثبته `TestRep_MapNeverGatesConversion`.
//
// # ولا يُطلَب موضعُ المندوب من هاتفه (البند ٢٥)
//
// **يُقرأ ما هو موجودٌ فعلاً**: **مواضعُ متاجره** — وهي حيثُ عمل.
// **ومن طلب `GPS` في الخلفيّة من المندوب بنى تتبّعاً لم يطلبه أحد.**

// Rep مندوبٌ ونشاطُه.
type Rep struct {
	ID   string `json:"id"`
	Name string `json:"name"`

	// Merchants **عددُ متاجره** — كلُّ ما نُسب إليه.
	Merchants int `json:"merchants"`
	// Converted **ما تحوّل في المدّة** — من مرشَّحٍ إلى متجر.
	Converted int `json:"converted"`

	// Lat/Lng **مركزُ ثقل متاجره** — **وهو «أين يعمل» بلا تتبّعِ هاتف.**
	Lat *float64 `json:"lat,omitempty"`
	Lng *float64 `json:"lng,omitempty"`

	LastActivity *time.Time `json:"last_activity,omitempty"`

	// Earnings **مستحقّاتُه** — **ولمن يملك `VIEW_MAP_FINANCIALS` وحدَه**
	// (البند ٢٦).
	Earnings *int64 `json:"earnings,omitempty"`
}

// RepPoint نقطةُ نشاطٍ واحدة — متجرٌ نُسب إلى مندوب.
type RepPoint struct {
	MerchantID string  `json:"merchant_id"`
	RepID      string  `json:"rep_id"`
	Lat        float64 `json:"lat"`
	Lng        float64 `json:"lng"`
}

// RepFilter مُرشِّحاتُ طبقة المندوبين.
type RepFilter struct {
	RepID  string
	CityID string
	Since  *time.Time
	Until  *time.Time
}

// Reps المندوبون ونشاطُهم الجغرافيّ.
//
// **والمدّةُ تخصُّ التحويلاتِ وحدَها** — **وعددُ المتاجر كلُّ ما نُسب
// إليه**، فمن حوّل عشرةً قبل سنةٍ لا يصير صفراً في تقرير الأسبوع.
func Reps(ctx context.Context, q Querier, f RepFilter, withMoney bool) ([]Rep, error) {
	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}
	add("ur.role_code = 'sales'")
	if f.RepID != "" {
		add(fmt.Sprintf("u.id = $%d", len(args)+1), f.RepID)
	}

	// **ومدّةُ التحويل تُمرَّر مرّتين** — في العدّ وفي آخر نشاط.
	since := "NULL::timestamptz"
	until := "NULL::timestamptz"
	if f.Since != nil {
		args = append(args, *f.Since)
		since = fmt.Sprintf("$%d::timestamptz", len(args))
	}
	if f.Until != nil {
		args = append(args, *f.Until)
		until = fmt.Sprintf("$%d::timestamptz", len(args))
	}
	cityCond := "true"
	if f.CityID != "" {
		args = append(args, f.CityID)
		cityCond = fmt.Sprintf("m.city_id = $%d::uuid", len(args))
	}

	sql := `
		SELECT u.id::text, u.full_name,
		       (SELECT count(*) FROM merchants m
		         WHERE m.sales_rep_user_id = u.id AND ` + cityCond + `),
		       (SELECT count(*) FROM merchants m
		         WHERE m.sales_rep_user_id = u.id AND ` + cityCond + `
		           AND (` + since + ` IS NULL OR m.created_at >= ` + since + `)
		           AND (` + until + ` IS NULL OR m.created_at < ` + until + `)),
		       (SELECT ST_Y(ST_Centroid(ST_Collect(m.location::geometry)))
		          FROM merchants m
		         WHERE m.sales_rep_user_id = u.id AND m.location IS NOT NULL AND ` + cityCond + `),
		       (SELECT ST_X(ST_Centroid(ST_Collect(m.location::geometry)))
		          FROM merchants m
		         WHERE m.sales_rep_user_id = u.id AND m.location IS NOT NULL AND ` + cityCond + `),
		       (SELECT max(m.created_at) FROM merchants m
		         WHERE m.sales_rep_user_id = u.id AND ` + cityCond + `),
		       COALESCE((SELECT w.balance FROM wallets w WHERE w.user_id = u.id), 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		WHERE ` + strings.Join(where, " AND ") + `
		ORDER BY u.full_name`

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Rep{}
	for rows.Next() {
		var x Rep
		var bal int64
		if err := rows.Scan(&x.ID, &x.Name, &x.Merchants, &x.Converted,
			&x.Lat, &x.Lng, &x.LastActivity, &bal); err != nil {
			return nil, err
		}
		if withMoney {
			b := bal
			x.Earnings = &b
		}
		out = append(out, x)
	}
	return out, rows.Err()
}

// RepPoints مواضعُ متاجر المندوبين — **لكثافةِ النشاط** (البند ٢٦).
//
// **وهي نقاطٌ لا مسارات**: **موضعُ متجرٍ فتحه المندوبُ** — **ولا تتبّعَ
// لخطواته.**
func RepPoints(ctx context.Context, q Querier, box *BBox, f RepFilter) ([]RepPoint, error) {
	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}
	add("m.sales_rep_user_id IS NOT NULL")
	add("m.location IS NOT NULL")
	if f.RepID != "" {
		add(fmt.Sprintf("m.sales_rep_user_id = $%d", len(args)+1), f.RepID)
	}
	if f.CityID != "" {
		add(fmt.Sprintf("m.city_id = $%d", len(args)+1), f.CityID)
	}
	if f.Since != nil {
		add(fmt.Sprintf("m.created_at >= $%d", len(args)+1), *f.Since)
	}
	if f.Until != nil {
		add(fmt.Sprintf("m.created_at < $%d", len(args)+1), *f.Until)
	}
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		add(box.SQL("m.location", len(args)+1), box.Args()...)
	}

	rows, err := q.Query(ctx, `
		SELECT m.id::text, m.sales_rep_user_id::text,
		       ST_Y(m.location::geometry), ST_X(m.location::geometry)
		FROM merchants m
		WHERE `+strings.Join(where, " AND ")+`
		LIMIT 5000`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []RepPoint{}
	for rows.Next() {
		var p RepPoint
		if err := rows.Scan(&p.MerchantID, &p.RepID, &p.Lat, &p.Lng); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
