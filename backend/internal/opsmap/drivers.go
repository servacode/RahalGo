package opsmap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ══════════════════════════════════════════════════════════════════════
// **طزاجةُ الموضع — مشتقّةٌ لا مخترَعة**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا `SLO` جديدٌ يُخترَع (البند ٥)
//
// **المنصّةُ تملك رقمَها**: `drivers.location_ping_sec` — كم بين نبضةٍ
// وأخرى (افتراضُه ٦٠ ثانية). **ومنه تُشتقّ الطزاجةُ ولا تُبتكَر:**
//
//	LIVE   ≤ 3 نبضاتٍ    — نبضةٌ ضاعت أو اثنتان، والشبكةُ تتعثّر
//	FRESH  ≤ 15 دقيقة    — **وهو الحدُّ الذي يستعمله المحرّك فعلاً**
//	STALE  > 15 دقيقة    — لا يُوزَّع عليه طلبٌ أصلاً
//	NONE   لا موضعَ قطّ
//
// **والخمسَ عشرةَ دقيقةً ليست رأياً**: هي المكتوبةُ في `queries.go`
// و`routeeta.go` لاختيار السائق — **فالخريطةُ تقول ما يقوله المحرّك،
// ولا تخترع حدّاً ثانياً يخالفه.**
//
// **ومن غيّر النبضةَ من اللوحة تحرّك حدُّ `LIVE` معه** — وهذا هو
// «التشغيل بلا شيفرة».

const (
	// FreshnessLive **نابضٌ الآن.**
	FreshnessLive = "LIVE"
	// FreshnessFresh **صالحٌ للتوزيع** — وهو حدُّ المحرّك.
	FreshnessFresh = "FRESH"
	// FreshnessStale **شاخ** — المحرّكُ لا يوزّع عليه.
	FreshnessStale = "STALE"
	// FreshnessNone **لا موضعَ قطّ** — ولا يُرسَم على الأرض.
	FreshnessNone = "NO_LOCATION"
)

// AssignableWindow **حدُّ المحرّك للموضع الصالح** — ١٥ دقيقة.
//
// **ومكتوبٌ هنا مرّةً واحدةً بمرجعه** — `orders/queries.go:47` و
// `orders/routeeta.go:64`. **ومن بدّله هناك بدّله هنا** — ويُسقط
// `TestOpsMap_FreshnessMatchesEngine` من نسي.
const AssignableWindow = 15 * time.Minute

// Freshness حكمُ الطزاجة على موضعٍ بعمره.
//
// **و`pingSec` من الإعدادات** — فما ضُبط في اللوحة يحكم.
func Freshness(age time.Duration, hasLocation bool, pingSec int64) string {
	if !hasLocation {
		return FreshnessNone
	}
	if pingSec <= 0 {
		pingSec = 60
	}
	if age <= time.Duration(pingSec)*3*time.Second {
		return FreshnessLive
	}
	if age <= AssignableWindow {
		return FreshnessFresh
	}
	return FreshnessStale
}

// ══════════════════════════════════════════════════════════════════════
// **نافذةُ العرض — لا تُحمَّل سوريا كلُّها** (البندان ٣١ و٤٣)
// ══════════════════════════════════════════════════════════════════════

// BBox مستطيلُ المشهد.
type BBox struct {
	MinLng, MinLat, MaxLng, MaxLat float64
}

// Valid **أمستطيلٌ معقول؟** — وإحداثيّاتٌ خارجَ الأرض تُردّ.
func (b BBox) Valid() bool {
	return b.MinLng >= -180 && b.MaxLng <= 180 &&
		b.MinLat >= -90 && b.MaxLat <= 90 &&
		b.MinLng < b.MaxLng && b.MinLat < b.MaxLat
}

// SQL شرطُ الاحتواء لعمودٍ جغرافيّ.
//
// **ويُبنى بمعاملاتٍ لا بنصّ** — والأرقامُ تأتي من `Args`.
func (b BBox) SQL(col string, n int) string {
	return fmt.Sprintf(
		"ST_Intersects(%s::geometry, ST_MakeEnvelope($%d,$%d,$%d,$%d,4326))",
		col, n, n+1, n+2, n+3)
}

// Args معاملاتُ المستطيل بترتيب `SQL`.
func (b BBox) Args() []any { return []any{b.MinLng, b.MinLat, b.MaxLng, b.MaxLat} }

// ══════════════════════════════════════════════════════════════════════
// **السائقون**
// ══════════════════════════════════════════════════════════════════════

// Driver سائقٌ على الخريطة.
//
// **ولا هاتفَ فيه** (البند ٣٣) — **الخريطةُ تقول أين ومن، لا كيف
// يُتَّصل به.** ومن أراد الاتّصالَ فتح بطاقتَه في «الحسابات».
type Driver struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Status   string  `json:"status"`
	OnShift  bool    `json:"on_shift"`
	ActiveN  int     `json:"active_orders"`
	CurrentO *string `json:"current_order,omitempty"`

	LastLocationAt time.Time `json:"last_location_at"`
	AgeSec         int64     `json:"age_sec"`
	Freshness      string    `json:"freshness"`

	// CashHeld **ما في صندوقه** — **ويبقى فارغاً لمن لا يملك
	// `VIEW_MAP_FINANCIALS`** (البند ٦).
	CashHeld *int64 `json:"cash_held,omitempty"`
}

// DriverFilter مُرشِّحاتُ طبقة السائقين (البند ٧).
//
// **وما لا نموذجَ له لا يُرشَّح به** — **ولا فرعَ ولا منطقةَ تشغيليّةَ
// للسائق في القاعدة اليومَ**، فلا يُخترَع حقلٌ فارغٌ يُوهم.
type DriverFilter struct {
	// CityID **مدينةُ السائق تُستنتج من موضعه** — ولا عمودَ لها فيه.
	CityID string
	// OnShift ثلاثيّةٌ: فارغٌ = الكلّ.
	OnShift *bool
	// HasActive سائقٌ بيده طلبٌ حيّ.
	HasActive *bool
	// Stale شاخ موضعُه.
	Stale *bool
	// Status حالُ الحساب — `active` · `suspended` · …
	Status string
	// Search اسمٌ أو جزءٌ منه.
	Search string
}

// Querier ما تحتاجه الحزمةُ من القاعدة.
//
// **وواجهةٌ لا `*pgxpool.Pool`** — **فتُختبر الاستعلاماتُ بمعاملةٍ
// تُلغى**، ولا يلزم خادمٌ كامل. **و`pgxpool.Pool` و`pgx.Tx` كلاهما
// يحقّقها كما هي.**
type Querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Execer ما يكتب — **ويُفصَل عن القارئ عمداً**: أكثرُ الخريطة قراءة،
// **ومن مرّر كاتباً إلى قارئٍ فتح باباً لا يحتاجه.**
type Execer interface {
	Querier
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Drivers سائقو المشهد.
//
// # ولماذا يُقرأ من لا موضعَ له أيضاً
//
// **«سائقٌ على الدوام بلا موضع» أهمُّ ما تقوله الخريطة** — **وهو تطبيقٌ
// قتله النظام** (D16/D18 مسجَّلان). **ومن رسم من له موضعٌ فقط أخفى
// المشكلةَ ذاتَها.**
//
// **فيُردّون بحالِ `NO_LOCATION` ولا يُرسمون على الأرض** (البند ٤) —
// **ولا يُخترَع لهم موضعٌ وهميّ.**
func Drivers(ctx context.Context, q Querier, box *BBox, f DriverFilter,
	pingSec int64, withMoney bool, limit int) ([]Driver, error) {

	var where []string
	var args []any
	add := func(cond string, vals ...any) {
		where = append(where, cond)
		args = append(args, vals...)
	}

	// **والمشهدُ يقيّد من له موضعٌ فقط** — ومن لا موضعَ له لا مكانَ له
	// في مستطيل، **فيُردّ دائماً ليُعَدّ في اللوحة الجانبيّة.**
	if box != nil {
		if !box.Valid() {
			return nil, fmt.Errorf("مستطيلُ مشهدٍ غيرُ معقول")
		}
		cond := fmt.Sprintf("(u.last_location IS NULL OR %s)",
			box.SQL("u.last_location", len(args)+1))
		add(cond, box.Args()...)
	}
	if f.Status != "" {
		add(fmt.Sprintf("u.status = $%d", len(args)+1), f.Status)
	}
	if f.OnShift != nil {
		add(fmt.Sprintf("u.on_shift = $%d", len(args)+1), *f.OnShift)
	}
	if f.Search != "" {
		add(fmt.Sprintf("u.full_name ILIKE $%d", len(args)+1), "%"+f.Search+"%")
	}
	if f.Stale != nil {
		cond := fmt.Sprintf(
			"(u.last_location_at IS NULL OR u.last_location_at <= now() - interval '%d seconds')",
			int(AssignableWindow.Seconds()))
		if !*f.Stale {
			cond = fmt.Sprintf(
				"(u.last_location_at > now() - interval '%d seconds')",
				int(AssignableWindow.Seconds()))
		}
		add(cond)
	}
	if f.HasActive != nil {
		cond := "EXISTS (SELECT 1 FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL)"
		if !*f.HasActive {
			cond = "NOT " + cond
		}
		add(cond)
	}
	if f.CityID != "" {
		// **ومدينةُ السائق موضعُه داخلَ دائرتها** — ولا عمودَ ينسبه.
		add(fmt.Sprintf(`EXISTS (SELECT 1 FROM cities c WHERE c.id = $%d
			AND u.last_location IS NOT NULL
			AND ST_DWithin(c.center, u.last_location, c.radius_m))`, len(args)+1), f.CityID)
	}

	cond := ""
	if len(where) > 0 {
		cond = " AND " + strings.Join(where, " AND ")
	}
	if limit <= 0 || limit > 2000 {
		limit = 2000
	}

	sql := `
		SELECT u.id::text, u.full_name, u.status, u.on_shift,
		       ST_Y(u.last_location::geometry), ST_X(u.last_location::geometry),
		       u.last_location_at,
		       (SELECT count(*) FROM orders o
		         WHERE o.driver_id = u.id AND o.closed_at IS NULL),
		       (SELECT o.id::text FROM orders o
		         WHERE o.driver_id = u.id AND o.closed_at IS NULL
		         ORDER BY o.created_at LIMIT 1),
		       COALESCE(cb.held, 0)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		LEFT JOIN driver_cash_boxes cb ON cb.driver_id = u.id
		WHERE true` + cond + `
		ORDER BY u.on_shift DESC, u.last_location_at DESC NULLS LAST
		LIMIT ` + fmt.Sprint(limit)

	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Driver{}
	now := time.Now()
	for rows.Next() {
		var d Driver
		var lat, lng *float64
		var at *time.Time
		var cur *string
		var held int64
		if err := rows.Scan(&d.ID, &d.Name, &d.Status, &d.OnShift,
			&lat, &lng, &at, &d.ActiveN, &cur, &held); err != nil {
			return nil, err
		}
		d.CurrentO = cur
		if lat != nil && lng != nil {
			d.Lat, d.Lng = *lat, *lng
		}
		if at != nil {
			d.LastLocationAt = *at
			d.AgeSec = int64(now.Sub(*at).Seconds())
		}
		d.Freshness = Freshness(now.Sub(deref(at, now)), lat != nil && at != nil, pingSec)
		if withMoney {
			h := held
			d.CashHeld = &h
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func deref(t *time.Time, def time.Time) time.Time {
	if t == nil {
		return def
	}
	return *t
}
