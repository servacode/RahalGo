package server

// ══════════════════════════════════════════════════════════════════════
// **التقارير — أرقامُ مدّةٍ بيوم دمشق** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// ما تغيّر عن الصفحة القديمة، وكلُّه بقرار المالك:
//
//   - **اليومُ يومُ دمشق** في المدى وفي المخطّط اليوميّ وفي الافتراض —
//     كان بغرينتش فتقع طلباتُ ما بعد منتصف الليل في يومٍ آخر.
//   - **أرقامُ الطلبات لكلّ من يفتح الصفحة، والمالُ لمن يقرأ المال وحدَه**
//     (الماليّة ومدير المنصّة) — **ويُحذف في الخادم** لا في الشاشة.
//   - **«ربح المنصّة» من حساب صفحة الأرباح نفسِه** (`platformProfit`) —
//     لا معادلةٌ ثانية. و«عمولة المتاجر» بطاقةٌ لحالها.
//   - **المستردُّ بطاقةٌ لحالها** — كان لا مسلَّماً ولا ملغى.
//   - **مقارنةٌ بالفترة السابقة المساوية** (`previous`).
//   - **الزوّارُ أجهزةُ الزبائن وحدَها وبيومٍ تقويميّ** — وعدُّ أجهزة
//     التطبيقات الأخرى مكانُه مراقبةُ التشغيل.
//   - **فلترُ النوع والمدينة**، وأفضلُ المتاجر تُنسَب لكلّ مصدرٍ حصّتُه.
//   - **مدًى مقلوبٌ يُرفض برسالة**، ولا مدى فوق سنة.

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// reportMaxDays **أطولُ مدًى يُقبل** — سنةٌ كاملة.
const reportMaxDays = 366

var (
	errReportRangeInverted = httpx.NewError(http.StatusBadRequest,
		"report_range_inverted", "errors.report_range_inverted")
	errReportRangeTooLong = httpx.NewError(http.StatusBadRequest,
		"report_range_too_long", "errors.report_range_too_long")
)

// reportKinds أنواعُ الطلب المقبولةُ في الفلتر.
var reportKinds = map[string]bool{"standard": true, "custom": true, "merchant_delivery": true}

// reportFilter **ما يُقرأ من الرابط** — يومان بيوم دمشق ونوعٌ ومدينة.
type reportFilter struct {
	From, To time.Time
	Kind     string
	CityID   string
}

func (f reportFilter) fromS() string { return f.From.Format("2006-01-02") }
func (f reportFilter) toS() string   { return f.To.Format("2006-01-02") }

// days عددُ أيّام المدى شاملاً طرفيه.
func (f reportFilter) days() int {
	return int(f.To.Sub(f.From).Hours()/24+0.5) + 1
}

// previous **الفترةُ السابقةُ المساوية** — بالطول نفسِه وتنتهي قبل «من» بيوم.
func (f reportFilter) previous() reportFilter {
	n := f.days()
	p := f
	p.To = f.From.AddDate(0, 0, -1)
	p.From = f.From.AddDate(0, 0, -n)
	return p
}

// parseReportFilter **يقرأ المدى بيوم دمشق** — والافتراضُ آخرُ سبعة أيّام
// تنتهي اليوم (يومَ دمشق لا يومَ غرينتش).
func parseReportFilter(q map[string][]string, now time.Time) (reportFilter, error) {
	get := func(k string) string {
		if v := q[k]; len(v) > 0 {
			return v[0]
		}
		return ""
	}
	loc := platform.Location()
	l := now.In(loc)
	today := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.UTC)
	var f reportFilter
	var err error
	if s := get("to"); s != "" {
		if f.To, err = time.Parse("2006-01-02", s); err != nil {
			return f, errValidation
		}
	} else {
		f.To = today
	}
	if s := get("from"); s != "" {
		if f.From, err = time.Parse("2006-01-02", s); err != nil {
			return f, errValidation
		}
	} else {
		f.From = f.To.AddDate(0, 0, -6)
	}
	if f.From.After(f.To) {
		return f, errReportRangeInverted
	}
	if f.days() > reportMaxDays {
		return f, errReportRangeTooLong
	}
	f.Kind = get("kind")
	if f.Kind != "" && !reportKinds[f.Kind] {
		return f, errValidation
	}
	f.CityID = get("city_id")
	if f.CityID != "" && !isUUID(f.CityID) {
		return f, errValidation
	}
	return f, nil
}

// reportScopeSQL **شرطُ الطلبات في المدى** — يُلحق بـ FROM orders o
// LEFT JOIN merchants mr. المعاملاتُ: ١ من · ٢ إلى · ٣ النوع · ٤ المدينة.
//
// **ومدينةُ الطلب مدينةُ متجره**، والطلبُ الخاصُّ (بلا متجر) مدينتُه
// المدينةُ التي تحوي نقطةَ تسليمه.
const reportScopeSQL = `
		WHERE o.created_at >= ($1::date::timestamp AT TIME ZONE 'Asia/Damascus')
		  AND o.created_at < (($2::date + 1)::timestamp AT TIME ZONE 'Asia/Damascus')
		  AND ($3::text = '' OR o.kind = $3::text)
		  AND ($4::text = '' OR (CASE WHEN o.merchant_id IS NOT NULL
		        THEN mr.city_id = NULLIF($4::text, '')::uuid
		        ELSE EXISTS (SELECT 1 FROM cities c
		                     WHERE c.id = NULLIF($4::text, '')::uuid
		                       AND ST_DWithin(c.center, o.dropoff, c.radius_m)) END))`

func (f reportFilter) args() []any {
	return []any{f.fromS(), f.toS(), f.Kind, f.CityID}
}

// reportSummary **الأرقامُ الرئيسيّة** — والمالُ مؤشّراتٌ تُحذف لمن لا يقرؤه.
type reportSummary struct {
	OrdersTotal int `json:"orders_total"`
	Delivered   int `json:"delivered"`
	Cancelled   int `json:"cancelled"`
	Refunded    int `json:"refunded"`
	// AvgOrderToDeliveryMin **من لحظة الطلب إلى التسليم** — فيه تحضيرُ المتجر.
	AvgOrderToDeliveryMin float64 `json:"avg_delivery_min"`
	ActiveCustomers       int     `json:"active_customers"`

	GrossSales     *int64 `json:"gross_sales,omitempty"`
	DeliveryFees   *int64 `json:"delivery_fees,omitempty"`
	Commissions    *int64 `json:"commissions,omitempty"`
	PlatformProfit *int64 `json:"platform_profit,omitempty"`
	WalletPaid     *int64 `json:"wallet_paid,omitempty"`
	CashCollected  *int64 `json:"cash_collected,omitempty"`
}

// reportSummaryOf **يحسب الأرقامَ الرئيسيّةَ لمدًى** — والمالُ إن طُلب.
//
// **وربحُ المنصّة من `platformProfit` نفسِه** — صافي صفحة الأرباح للمدى
// ذاته، **فلا يقول التقريرُ رقماً وصفحةُ الأرباح غيرَه.**
func (s *Server) reportSummaryOf(ctx context.Context, f reportFilter, money bool) (reportSummary, error) {
	var out reportSummary
	var gross, fees, comm, wallet, cash int64
	err := s.pg.QueryRow(ctx, `
		SELECT
			count(*),
			count(*) FILTER (WHERE o.status = 'delivered'),
			count(*) FILTER (WHERE o.status IN ('cancelled','rejected','failed')),
			count(*) FILTER (WHERE o.status = 'refunded'),
			COALESCE(sum(o.total) FILTER (WHERE o.status = 'delivered'), 0),
			COALESCE(sum(o.delivery_fee) FILTER (WHERE o.status = 'delivered'), 0),
			COALESCE(sum(o.platform_commission) FILTER (WHERE o.status = 'delivered'), 0),
			COALESCE(sum(o.wallet_paid) FILTER (WHERE o.status = 'delivered'), 0),
			COALESCE(sum(o.cash_due) FILTER (WHERE o.status = 'delivered'), 0),
			COALESCE(round(avg(EXTRACT(EPOCH FROM o.delivered_at - o.created_at) / 60)
				FILTER (WHERE o.status = 'delivered' AND o.delivered_at IS NOT NULL))::float8, 0),
			count(DISTINCT o.customer_id)
		FROM orders o
		LEFT JOIN merchants mr ON mr.id = o.merchant_id`+reportScopeSQL, f.args()...).
		Scan(&out.OrdersTotal, &out.Delivered, &out.Cancelled, &out.Refunded,
			&gross, &fees, &comm, &wallet, &cash,
			&out.AvgOrderToDeliveryMin, &out.ActiveCustomers)
	if err != nil {
		return out, err
	}
	if !money {
		return out, nil
	}
	p, err := s.platformProfit(ctx, f.fromS(), f.toS())
	if err != nil {
		return out, err
	}
	out.GrossSales, out.DeliveryFees, out.Commissions = &gross, &fees, &comm
	out.WalletPaid, out.CashCollected = &wallet, &cash
	net := p.Net
	out.PlatformProfit = &net
	return out, nil
}

// reportDay **يومٌ في المخطّط** — بيوم دمشق.
type reportDay struct {
	Day       string `json:"day"`
	Orders    int    `json:"orders"`
	Delivered int    `json:"delivered"`
	Sales     *int64 `json:"sales,omitempty"`
}

type reportStore struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Delivered int    `json:"delivered"`
	Sales     *int64 `json:"sales,omitempty"`
}

type reportDriver struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Phone     string `json:"phone,omitempty"`
	Delivered int    `json:"delivered"`
	Cash      *int64 `json:"cash,omitempty"`
}

// reportVisitors **زوّارُ تطبيق الزبون** — يومٌ تقويميٌّ بدمشق.
type reportVisitors struct {
	OpensToday   int64 `json:"opens_today"`
	Opens7       int64 `json:"opens_7d"`
	Opens30      int64 `json:"opens_30d"`
	DevicesToday int   `json:"devices_today"`
	Devices7     int   `json:"devices_7d"`
	Devices30    int   `json:"devices_30d"`
}

// customerVisitors **فتحاتُ تطبيق الزبون وأجهزتُه** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
//
// **والأجهزةُ أجهزةُ الزبائن وحدَها** (app = 'customer') — كانت تعدّ أجهزةَ
// السائقين والمتاجر والمناديب، **فتقول ٤ أجهزةٍ و١ فتحة.** **واليومُ يومٌ
// تقويميٌّ بدمشق** كالفتحات — كان «آخرَ ٢٤ ساعة». **و«٧ أيّام» اليومُ
// وستّةٌ قبله.** والفتحاتُ لا يعدّها إلّا نداءُ رئيسيّة الزبون (countOpen).
func (s *Server) customerVisitors(ctx context.Context) (reportVisitors, error) {
	var v reportVisitors
	err := s.pg.QueryRow(ctx, `
		WITH d AS (
			SELECT (now() AT TIME ZONE 'Asia/Damascus')::date AS today
		), b AS (
			SELECT (d.today::timestamp AT TIME ZONE 'Asia/Damascus') AS t0,
			       ((d.today - 6)::timestamp AT TIME ZONE 'Asia/Damascus') AS t7,
			       ((d.today - 29)::timestamp AT TIME ZONE 'Asia/Damascus') AS t30
			FROM d
		)
		SELECT
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d WHERE day = d.today), 0),
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d WHERE day > d.today - 7), 0),
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d WHERE day > d.today - 30), 0),
		  (SELECT count(*) FROM device_tokens, b WHERE app = 'customer' AND last_seen_at >= b.t0),
		  (SELECT count(*) FROM device_tokens, b WHERE app = 'customer' AND last_seen_at >= b.t7),
		  (SELECT count(*) FROM device_tokens, b WHERE app = 'customer' AND last_seen_at >= b.t30)
	`).Scan(&v.OpensToday, &v.Opens7, &v.Opens30, &v.DevicesToday, &v.Devices7, &v.Devices30)
	return v, err
}

// handleReports تقريرُ مدّة — انظر رأسَ الملفّ.
func (s *Server) handleReports(w http.ResponseWriter, r *http.Request) {
	f, err := parseReportFilter(r.URL.Query(), time.Now())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	ctx := r.Context()
	// **والمالُ لمن يقرأ المال** — الماليّة ومدير المنصّة (القرار ٣).
	money := s.canSeeAuditMoney(r)

	summary, err := s.reportSummaryOf(ctx, f, money)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	prevF := f.previous()
	previous, err := s.reportSummaryOf(ctx, prevF, money)
	if err != nil {
		s.respondErr(w, err)
		return
	}

	daily, err := s.reportDaily(ctx, f, money)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	stores, err := s.reportTopStores(ctx, f, money)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	drivers, err := s.reportTopDrivers(ctx, f, money, s.hasCapability(r, authz.UsersContactRead))
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **والزوّارُ لا يُسقطون التقرير** — وعطبُهم يُقال لا يُخفى.
	var visitors *reportVisitors
	visitorsFailed := false
	if v, err := s.customerVisitors(ctx); err == nil {
		visitors = &v
	} else {
		visitorsFailed = true
		s.logger.Error("reports visitors", "error", err)
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"from": f.fromS(), "to": f.toS(),
		"kind": f.Kind, "city_id": f.CityID,
		"previous_from": prevF.fromS(), "previous_to": prevF.toS(),
		"money_visible": money,
		// **وربحُ المنصّة للمنصّة كلِّها** — لا يتبع فلترَ النوع والمدينة.
		"profit_unfiltered": f.Kind != "" || f.CityID != "",
		"summary":           summary, "previous": previous,
		"daily": daily, "top_merchants": stores, "top_drivers": drivers,
		"visitors": visitors, "visitors_failed": visitorsFailed,
	})
}

// reportDaily **كلُّ يومٍ في المدى بيوم دمشق** — حتّى الفارغ.
func (s *Server) reportDaily(ctx context.Context, f reportFilter, money bool) ([]reportDay, error) {
	rows, err := s.pg.Query(ctx, `
		WITH scoped AS (
			SELECT o.status, o.total,
			       (o.created_at AT TIME ZONE 'Asia/Damascus')::date AS day
			FROM orders o
			LEFT JOIN merchants mr ON mr.id = o.merchant_id`+reportScopeSQL+`
		)
		SELECT d::date::text,
		       count(sc.day),
		       count(sc.day) FILTER (WHERE sc.status = 'delivered'),
		       COALESCE(sum(sc.total) FILTER (WHERE sc.status = 'delivered'), 0)
		FROM generate_series($1::date, $2::date, '1 day') d
		LEFT JOIN scoped sc ON sc.day = d::date
		GROUP BY d ORDER BY d`, f.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reportDay{}
	for rows.Next() {
		var d reportDay
		var sales int64
		if err := rows.Scan(&d.Day, &d.Orders, &d.Delivered, &sales); err != nil {
			return nil, err
		}
		if money {
			v := sales
			d.Sales = &v
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// reportTopStores **أفضلُ المتاجر — ولكلّ مصدرٍ حصّتُه.**
//
// **كان الطلبُ متعدّدُ المصادر يُنسب كلُّه لأوّل متجر.** فصار لكلّ متجرٍ
// قيمةُ أصنافه هو في الطلب (order_items.merchant_id)، **والطلبُ يُعدّ لكلّ
// متجرٍ شارك فيه.** والطلبُ بلا أصنافٍ مقيّدة (توصيلةُ متجر) يُنسب لمتجره
// بقيمة بضاعته.
//
// **ومن لا يقرأ المالَ يُرتَّب له بعدد الطلبات** — وترتيبٌ بالمبيعات يكشف
// ما حُذف.
func (s *Server) reportTopStores(ctx context.Context, f reportFilter, money bool) ([]reportStore, error) {
	order := `sum(c.sales) DESC`
	if !money {
		order = `count(DISTINCT c.oid) DESC`
	}
	rows, err := s.pg.Query(ctx, `
		WITH scoped AS (
			SELECT o.id, o.merchant_id, o.subtotal
			FROM orders o
			LEFT JOIN merchants mr ON mr.id = o.merchant_id`+reportScopeSQL+`
			  AND o.status = 'delivered'
		), credit AS (
			SELECT oi.merchant_id AS mid, sc.id AS oid, sum(oi.unit_price * oi.qty) AS sales
			FROM scoped sc JOIN order_items oi ON oi.order_id = sc.id
			WHERE oi.merchant_id IS NOT NULL
			GROUP BY oi.merchant_id, sc.id
			UNION ALL
			SELECT sc.merchant_id, sc.id, sc.subtotal
			FROM scoped sc
			WHERE sc.merchant_id IS NOT NULL
			  AND NOT EXISTS (SELECT 1 FROM order_items oi
			                  WHERE oi.order_id = sc.id AND oi.merchant_id IS NOT NULL)
		)
		SELECT m.id::text, m.name, count(DISTINCT c.oid), COALESCE(sum(c.sales), 0)::bigint
		FROM credit c JOIN merchants m ON m.id = c.mid
		GROUP BY m.id, m.name
		ORDER BY `+order+`, m.name
		LIMIT 5`, f.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reportStore{}
	for rows.Next() {
		var t reportStore
		var sales int64
		if err := rows.Scan(&t.ID, &t.Name, &t.Delivered, &sales); err != nil {
			return nil, err
		}
		if money {
			v := sales
			t.Sales = &v
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// reportTopDrivers **أفضلُ السائقين بعدد التسليم** — بالمعرّف لا بالهاتف،
// **والهاتفُ لمن يملك قراءةَ الأرقام**، والنقدُ لمن يقرأ المال.
func (s *Server) reportTopDrivers(ctx context.Context, f reportFilter, money, phone bool) ([]reportDriver, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT u.id::text, COALESCE(u.full_name, ''), u.phone::text,
		       count(*), COALESCE(sum(o.cash_due), 0)
		FROM orders o
		JOIN users u ON u.id = o.driver_id
		LEFT JOIN merchants mr ON mr.id = o.merchant_id`+reportScopeSQL+`
		  AND o.status = 'delivered'
		GROUP BY u.id, u.full_name, u.phone
		ORDER BY count(*) DESC, u.id
		LIMIT 5`, f.args()...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []reportDriver{}
	for rows.Next() {
		var t reportDriver
		var ph string
		var cash int64
		if err := rows.Scan(&t.ID, &t.Name, &ph, &t.Delivered, &cash); err != nil {
			return nil, err
		}
		if phone {
			t.Phone = ph
		}
		if money {
			v := cash
			t.Cash = &v
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
