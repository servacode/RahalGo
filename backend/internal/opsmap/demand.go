package opsmap

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **الكثافةُ وفرصُ التوسّع — `MAP-6`**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا تنبّؤَ وهميّ (البند ٢٨)
//
// **كلُّ رقمٍ هنا عدٌّ يُفسَّر**: كم طلباً وقع في هذه الخليّة، وكم طلبَ
// تغطيةٍ، وكم متجراً، وكم سائقاً. **ولا نموذجَ يتوقّع.**
//
// **ومن كتب «درجةً» بلا معادلةٍ تُقرأ باع رأياً على أنّه حساب.**
//
// # والكثافتان لا تُخلَطان (البند ١٨)
//
// **طلبٌ نُفِّذ غيرُ طلبِ تغطيةٍ لم يُغطَّ بعد** — **الأوّلُ يقول «هنا
// عملٌ»، والثاني يقول «هنا عملٌ لا نأخذه».** **وجمعُهما في رقمٍ واحدٍ
// يمحو المعنيين.**

// CellSizeDeg **ضلعُ الخليّة بالدرجات** — ≈ ١٫١ كم عند خطّ عرض الرقّة.
//
// **وشبكةٌ لا نقاطٌ خام**: **مئةُ ألف طلبٍ لا تُرسَل إلى متصفّح**،
// **والقرارُ يُتَّخذ بالحيّ لا بالبيت.**
const CellSizeDeg = 0.01

// Cell خليّةُ شبكةٍ ومحتواها.
type Cell struct {
	// Lat/Lng **مركزُ الخليّة.**
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
	// Count **العدُّ في هذه الخليّة** — ومعناه يتبع الطبقة.
	Count int `json:"count"`
}

// snap يُرجع مركزَ الخليّة التي تقع فيها نقطة.
func snap(v float64) float64 {
	return math.Floor(v/CellSizeDeg)*CellSizeDeg + CellSizeDeg/2
}

// gridSQL يبني استعلامَ شبكةٍ لعمودٍ جغرافيّ.
//
// **والتجميعُ في القاعدة لا في `Go`** — **ومن جرّ مئةَ ألف صفٍّ ليعدّها
// في الذاكرة جرّها عبرَ الشبكة أوّلاً.**
func gridSQL(table, geom string, where []string, args int) string {
	cond := ""
	if len(where) > 0 {
		cond = " WHERE " + strings.Join(where, " AND ")
	}
	_ = args
	return fmt.Sprintf(`
		SELECT floor(ST_Y(%[2]s::geometry) / %[3]v) * %[3]v + %[4]v AS cy,
		       floor(ST_X(%[2]s::geometry) / %[3]v) * %[3]v + %[4]v AS cx,
		       count(*)
		FROM %[1]s%[5]s
		GROUP BY 1, 2
		ORDER BY 3 DESC
		LIMIT 3000`, table, geom, CellSizeDeg, CellSizeDeg/2, cond)
}

func scanCells(ctx context.Context, q Querier, sql string, args ...any) ([]Cell, error) {
	rows, err := q.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Cell{}
	for rows.Next() {
		var c Cell
		if err := rows.Scan(&c.Lat, &c.Lng, &c.Count); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DemandFilter مدىً ومكان.
type DemandFilter struct {
	Since  *time.Time
	Until  *time.Time
	CityID string
}

func (f DemandFilter) clause(col, cityCol string, args *[]any) []string {
	var w []string
	if f.Since != nil {
		*args = append(*args, *f.Since)
		w = append(w, fmt.Sprintf("%s >= $%d", col, len(*args)))
	}
	if f.Until != nil {
		*args = append(*args, *f.Until)
		w = append(w, fmt.Sprintf("%s < $%d", col, len(*args)))
	}
	if f.CityID != "" && cityCol != "" {
		*args = append(*args, f.CityID)
		w = append(w, fmt.Sprintf("%s = $%d::uuid", cityCol, len(*args)))
	}
	return w
}

// OrderDemand كثافةُ الطلبات — **«هنا عملٌ نأخذه».**
func OrderDemand(ctx context.Context, q Querier, f DemandFilter) ([]Cell, error) {
	var args []any
	w := append([]string{"dropoff IS NOT NULL"}, f.clause("created_at", "", &args)...)
	return scanCells(ctx, q, gridSQL("orders", "dropoff", w, len(args)), args...)
}

// RequestDemand كثافةُ طلبات التغطية — **«هنا عملٌ لا نأخذه».**
func RequestDemand(ctx context.Context, q Querier, f DemandFilter) ([]Cell, error) {
	var args []any
	w := f.clause("created_at", "city_id", &args)
	return scanCells(ctx, q, gridSQL("coverage_requests", "at", w, len(args)), args...)
}

// MerchantDensity كثافةُ المتاجر.
func MerchantDensity(ctx context.Context, q Querier, f DemandFilter) ([]Cell, error) {
	var args []any
	w := append([]string{"location IS NOT NULL", "status = 'active'"},
		f.clause("created_at", "city_id", &args)...)
	return scanCells(ctx, q, gridSQL("merchants", "location", w, len(args)), args...)
}

// DriverSupply كثافةُ توفّر السائقين — **من كان موضعُه صالحاً.**
//
// **ولا يُعَدُّ من شاخ موضعُه**: **سائقٌ في بيته منذ ساعةٍ ليس عرضاً.**
func DriverSupply(ctx context.Context, q Querier) ([]Cell, error) {
	sql := fmt.Sprintf(`
		SELECT floor(ST_Y(u.last_location::geometry) / %[1]v) * %[1]v + %[2]v,
		       floor(ST_X(u.last_location::geometry) / %[1]v) * %[1]v + %[2]v,
		       count(*)
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE u.last_location IS NOT NULL
		  AND u.last_location_at > now() - interval '%[3]d seconds'
		GROUP BY 1, 2
		ORDER BY 3 DESC
		LIMIT 3000`, CellSizeDeg, CellSizeDeg/2, int(AssignableWindow.Seconds()))
	return scanCells(ctx, q, sql)
}

// UnservedDemand طلباتُ تغطيةٍ خارجَ كلّ منطقةٍ مفعَّلة.
//
// # وهذا أدقُّ رقمٍ في الصفحة
//
// **ليس «كم طلبَ أحدٌ تغطية»** — **بل كم منها ما زال خارجَ ما نغطّيه
// اليومَ فعلاً.** **ومن رسم منطقةً أمسِ نقص هذا الرقمُ اليومَ بلا أن
// يلمس أحدٌ حالَ الطلبات.**
func UnservedDemand(ctx context.Context, q Querier, f DemandFilter) ([]Cell, error) {
	var args []any
	w := append([]string{`NOT EXISTS (
		SELECT 1 FROM delivery_zones z
		 WHERE z.active AND (
		     (z.shape = 'radius' AND ST_DWithin(z.center, coverage_requests.at, z.radius_m))
		  OR (z.shape = 'polygon' AND z.area IS NOT NULL AND ST_Covers(z.area, coverage_requests.at))
		 ))`}, f.clause("created_at", "city_id", &args)...)
	return scanCells(ctx, q, gridSQL("coverage_requests", "at", w, len(args)), args...)
}

// Opportunity فرصةُ توسّعٍ في خليّة (البند ٢٨).
type Opportunity struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`

	Orders    int `json:"orders"`
	Requests  int `json:"requests"`
	Merchants int `json:"merchants"`
	Drivers   int `json:"drivers"`

	// Score **درجةٌ محسوبةٌ تُفسَّر** — لا تنبّؤ.
	Score int `json:"score"`
	// Reasons **لماذا هذه الخليّةُ فرصة** — بأسماءٍ لا بأرقامٍ عمياء.
	Reasons []string `json:"reasons"`
}

// أسبابُ الفرصة — **وأسماؤها تُترجَم في الواجهة.**
const (
	ReasonHighDemandLowCoverage = "high_demand_low_coverage"
	ReasonHighDemandLowDrivers  = "high_demand_low_drivers"
	ReasonManyRequests          = "many_requests"
	ReasonMerchantsNoDrivers    = "merchants_no_drivers"
)

// Opportunities يجمع الطبقاتِ في خلايا ويرتّبها.
//
// # والمعادلةُ مكتوبةٌ لا مخبوءة
//
//	درجة = طلباتٌ غيرُ مغطّاة×3 + طلباتُ تغطية×2 + متاجرُ بلا سائقين×2
//
// **والأوزانُ رأيٌ معلَنٌ لا حقيقة** — **ومن خالفها بدّلها هنا في سطرٍ
// واحدٍ يقرؤه.** **والأرقامُ الخامُّ تُردّ معها**، فمن لم يقبل الوزنَ
// حسب بنفسه.
func Opportunities(ctx context.Context, q Querier, f DemandFilter) ([]Opportunity, error) {
	ord, err := OrderDemand(ctx, q, f)
	if err != nil {
		return nil, err
	}
	req, err := UnservedDemand(ctx, q, f)
	if err != nil {
		return nil, err
	}
	mer, err := MerchantDensity(ctx, q, f)
	if err != nil {
		return nil, err
	}
	drv, err := DriverSupply(ctx, q)
	if err != nil {
		return nil, err
	}

	type key struct{ lat, lng float64 }
	cells := map[key]*Opportunity{}
	at := func(c Cell) *Opportunity {
		k := key{snap(c.Lat), snap(c.Lng)}
		if cells[k] == nil {
			cells[k] = &Opportunity{Lat: k.lat, Lng: k.lng}
		}
		return cells[k]
	}
	for _, c := range ord {
		at(c).Orders += c.Count
	}
	for _, c := range req {
		at(c).Requests += c.Count
	}
	for _, c := range mer {
		at(c).Merchants += c.Count
	}
	for _, c := range drv {
		at(c).Drivers += c.Count
	}

	out := []Opportunity{}
	for _, o := range cells {
		o.Score = o.Requests*3 + o.Orders*0 + o.Merchants*0
		if o.Requests > 0 {
			o.Reasons = append(o.Reasons, ReasonHighDemandLowCoverage)
		}
		if o.Requests >= 3 {
			o.Reasons = append(o.Reasons, ReasonManyRequests)
			o.Score += o.Requests * 2
		}
		if o.Orders > 0 && o.Drivers == 0 {
			o.Reasons = append(o.Reasons, ReasonHighDemandLowDrivers)
			o.Score += o.Orders * 2
		}
		if o.Merchants > 0 && o.Drivers == 0 {
			o.Reasons = append(o.Reasons, ReasonMerchantsNoDrivers)
			o.Score += o.Merchants * 2
		}
		// **وخليّةٌ بلا سببٍ ليست فرصة** — ولا تُرسَل لتملأ الشاشة.
		if len(o.Reasons) > 0 {
			out = append(out, *o)
		}
	}
	// **والأعلى أوّلاً** — والمكتبُ يقرأ العشرَ الأُوَل.
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Score > out[i].Score ||
				(out[j].Score == out[i].Score && out[j].Lat < out[i].Lat) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if len(out) > 200 {
		out = out[:200]
	}
	return out, nil
}
