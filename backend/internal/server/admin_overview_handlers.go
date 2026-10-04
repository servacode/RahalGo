package server

// ══════════════════════════════════════════════════════════════════════
// **رئيسيّةُ مدير المنصّة — كلُّ ما يجري في صفحةٍ واحدة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الرئيسيّة لمدير المنصّة وحدَه» · «لازم أفهم
//  كلّ شي عم يصير بالمنصّة لأصغر تفصيل بدون ما أتنقّل بين الصفحات».)
//
// # الترتيبُ من أعلى إلى أسفل
//
//	بانتظار قرارك · الآن · اليوم مقابل أمس · المال اليوم · آخر الأحداث ·
//	المنصّة بالأرقام
//
// # وكلُّ رقمٍ بابٌ يفتح صفحةً فيها الرقمُ نفسُه
//
// **لا يُكتب هنا شرطُ عدٍّ ثانٍ لشيءٍ تعدّه صفحةٌ أخرى**: البلاغاتُ من
// `CountAwaitingOffice`، والشكاوى من `support.Count`، والنقدُ من
// `cashHolders`، والعالقُ من `orders.Alerts`، والصافي من `platformProfit` —
// **وهي ما تقرؤه صفحاتُها.** والباقي عدٌّ بشرط صفحته حرفاً، **ويحرسه
// اختبارٌ يقارن الرقمين** (`admin_overview_test.go`).
//
// # والرقمُ الغائبُ «غير معروف» لا صفر
//
// **كلُّ قسمٍ يُقرأ وحدَه**، فإن عَطِب صار رقمُه `null` واسمُه في `missing`
// — **والشاشةُ تكتب «غير معروف»**. **وصفرٌ أخضرُ على رقمٍ لم يُقرأ يقول
// «لا شيءَ ينتظرك» وفي الشارع سائقٌ ينتظر.**
//
// # واليومُ يومُ دمشق
//
// **حدودُ اليوم والأمس والأسبوع تُحسب بمنطقة المنصّة** (`platform.Location`)
// — **وبفارق الثلاث ساعات تقع طلباتُ الليل في يومٍ آخر عند غرينتش.**

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/routing"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// num رقمٌ قد لا يُعرف — و`nil` يُرسَم «غير معروف».
type num = *int64

func n64(v int64) num { return &v }

type ovAwaiting struct {
	ReportsWaiting       num `json:"reports_waiting"`
	EmergenciesOpen      num `json:"emergencies_open"`
	OrdersUnassigned     num `json:"orders_unassigned"`
	CompensationsPending num `json:"compensations_pending"`
	PayoutsPending       num `json:"payouts_pending"`
	DriversOverCash      num `json:"drivers_over_cash"`
	TicketsOpen          num `json:"tickets_open"`
	TicketsLate          num `json:"tickets_late"`
	LeadsNew             num `json:"leads_new"`
	ExpansionWaiting     num `json:"expansion_waiting"`
}

type ovLive struct {
	// Stages **الجاري بمراحله** — مفاتيحُها `orders.LiveStages`.
	Stages         map[string]num `json:"stages"`
	OrdersOpen     num            `json:"orders_open"`
	OrdersStuck    num            `json:"orders_stuck"`
	DriversOnShift num            `json:"drivers_on_shift"`
	DriversFree    num            `json:"drivers_free"`
	DriversBusy    num            `json:"drivers_busy"`
	StoresOpenNow  num            `json:"stores_open_now"`
	StoresActive   num            `json:"stores_active"`
}

type ovHealth struct {
	API      string `json:"api"`
	Database string `json:"database"`
	WhatsApp string `json:"whatsapp"`
	Routing  string `json:"routing"`
}

type ovDay struct {
	Orders         num      `json:"orders"`
	Delivered      num      `json:"delivered"`
	Failed         num      `json:"failed"`
	AvgDeliveryMin *float64 `json:"avg_delivery_min"`
	NewCustomers   num      `json:"new_customers"`
	AvgPlatform    *float64 `json:"avg_platform_rating"`
	AvgDriver      *float64 `json:"avg_driver_rating"`
	Ratings        num      `json:"ratings"`
}

type ovFailure struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

type ovMoney struct {
	Sales                num `json:"sales"`
	Net                  num `json:"net"`
	Losses               num `json:"losses"`
	TreasuryBalance      num `json:"treasury_balance"`
	CashHeld             num `json:"cash_held"`
	CompensationsCount   num `json:"compensations_count"`
	CompensationsAmount  num `json:"compensations_amount"`
	DeliveredOrdersCount num `json:"delivered_orders"`
}

type ovEvent struct {
	ID        int64     `json:"id"`
	Action    string    `json:"action"`
	Entity    string    `json:"entity"`
	EntityID  string    `json:"entity_id"`
	Label     string    `json:"label"`
	Number    *int64    `json:"number"`
	ActorName string    `json:"actor_name"`
	CreatedAt time.Time `json:"created_at"`
}

type ovCount struct {
	Key   string `json:"key"`
	Total int64  `json:"total"`
	Today int64  `json:"today"`
	Week  int64  `json:"week"`
}

type ovPlatform struct {
	Groups               []ovCount `json:"groups"`
	Staff                []ovCount `json:"staff"`
	CustomersOrderedWeek num       `json:"customers_ordered_week"`
	DriversActive        num       `json:"drivers_active"`
	DriversSuspended     num       `json:"drivers_suspended"`
	DriversBlocked       num       `json:"drivers_blocked"`
	StoresActive         num       `json:"stores_active"`
	StoresNotActive      num       `json:"stores_not_active"`
}

type overview struct {
	GeneratedAt time.Time        `json:"generated_at"`
	Today       string           `json:"today"`
	Missing     []string         `json:"missing"`
	Emergency   *emergencyBanner `json:"emergency"`
	Awaiting    ovAwaiting       `json:"awaiting"`
	Live        ovLive           `json:"live"`
	Health      ovHealth         `json:"health"`
	TodayStats  ovDay            `json:"today_stats"`
	Yesterday   ovDay            `json:"yesterday_stats"`
	Failures    []ovFailure      `json:"failures_today"`
	Money       ovMoney          `json:"money"`
	Events      []ovEvent        `json:"events"`
	Platform    ovPlatform       `json:"platform"`
}

// overviewEvents **الأحداثُ المهمّة** — ما يغيّر حالَ المنصّة أو مالَها.
//
// **ولا دخولَ ولا تجديدَ جلسة**: أكثرُ ما يُكتب في السجلّ، **وعشرةُ أسطرٍ
// منها تدفن متجراً جديداً أُضيف قبل دقيقة.**
var overviewEvents = []string{
	"admin.merchant_create", "admin.user_create", "admin.role_grant", "admin.role_revoke",
	"admin.user_update", "admin.platform_closure", "admin.setting_update",
	"admin.promo_create", "admin.zone_create", "admin.zone_delete", "coverage.set_active",
	"finance.driver_compensation", "finance.driver_compensation_rejected",
	"finance.payout_decide", "finance.wallet_apply", "finance.expense_added",
	"finance.incentive", "finance.ticket_resolve",
	"ops.merchant_suspend", "ops.warning_issued", "ops.merchant_warning_issued",
	"ops.dispute_opened", "ops.dispute_settled", "ops.emergency_resolved",
	"ops.emergency_ack", "ops.store_emergency_ack", "ops.order_transferred",
	"ops.door_resolution", "ops.campaign_send", "ops.broadcast",
	"driver.emergency", "customer.complaint_opened", "platform.app_upload",
	"user.self_delete",
}

// overviewDays **حدودُ الأيّام بمنطقة المنصّة.**
func overviewDays(now time.Time) (today, yesterday, week time.Time) {
	loc := platform.Location()
	l := now.In(loc)
	today = time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, loc)
	return today, today.AddDate(0, 0, -1), today.AddDate(0, 0, -6)
}

// ticketLateHours **مهلةُ الردّ على الشكوى** — من الإعدادات (افتراضُها ساعتان).
func (s *Server) ticketLateHours(ctx context.Context) int {
	h := int(s.settings.GetInt(ctx, "support.late_reply_hours"))
	if h <= 0 {
		h = 2
	}
	return h
}

// handleAdminOverview **رئيسيّةُ مدير المنصّة.**
//
// **والقدرةُ تُفحص هنا أيضاً لا في الجدول وحدَه** — فمعالِجٌ يُركَّب غداً
// خلف وسيطٍ آخر لا يفتح المالَ لمن لا يملكه.
func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	if !hasCap(r, authz.PlatformOverview) {
		s.respondErr(w, errForbidden)
		return
	}
	httpx.JSON(w, http.StatusOK, s.buildOverview(r.Context()))
}

func hasCap(r *http.Request, c authz.Capability) bool {
	for _, x := range capabilitiesFrom(r) {
		if x == string(c) {
			return true
		}
	}
	return false
}

func (s *Server) buildOverview(ctx context.Context) overview {
	now := time.Now()
	today, yesterday, week := overviewDays(now)
	tomorrow := today.AddDate(0, 0, 1)
	ov := overview{
		GeneratedAt: now, Today: today.Format("2006-01-02"),
		Missing: []string{}, Failures: []ovFailure{}, Events: []ovEvent{},
	}
	miss := func(section string, err error) bool {
		if err == nil {
			return false
		}
		s.logger.Error("الرئيسيّة: قسمٌ لم يُقرأ", "section", section, "error", err)
		ov.Missing = append(ov.Missing, section)
		return true
	}

	// ── الطارئ ──────────────────────────────────────────────────────
	if b, err := s.emergencyBannerData(ctx); !miss("emergency", err) {
		ov.Emergency = &b
	}

	// ── بانتظار قرارك ───────────────────────────────────────────────
	a := &ov.Awaiting
	{
		var comp, emerg, pay, leads, exp int64
		err := s.pg.QueryRow(ctx, `
			SELECT
				(SELECT count(*) FROM driver_compensation_requests WHERE status = 'pending'),
				(SELECT count(*) FROM driver_emergencies WHERE status = 'open'),
				(SELECT count(*) FROM payout_requests p WHERE p.status = 'pending'),
				(SELECT count(*) FROM merchant_leads l WHERE l.status = 'new'),
				-- **«ينتظرون ولم يُبلَّغوا» بشرط صفحة التوسّع نفسِه** (expWaitingSQL):
				-- لا الملغى ولا المرفوض ولا من لا حسابَ له.
				(SELECT count(*) FROM coverage_requests r WHERE `+expWaitingSQL+`)`).
			Scan(&comp, &emerg, &pay, &leads, &exp)
		if !miss("awaiting", err) {
			a.CompensationsPending, a.EmergenciesOpen = n64(comp), n64(emerg)
			a.PayoutsPending, a.LeadsNew, a.ExpansionWaiting = n64(pay), n64(leads), n64(exp)
		}
	}
	if n, err := s.orders.CountAwaitingOffice(ctx); !miss("reports_waiting", err) {
		a.ReportsWaiting = n64(int64(n))
	}
	if s.support != nil {
		if n, err := s.support.Count(ctx, support.TicketFilter{Status: "unresolved"}); !miss("tickets", err) {
			a.TicketsOpen = n64(int64(n))
		}
		if n, err := s.support.Count(ctx, support.TicketFilter{
			LateHours: s.ticketLateHours(ctx)}); !miss("tickets_late", err) {
			a.TicketsLate = n64(int64(n))
		}
	} else {
		ov.Missing = append(ov.Missing, "tickets")
	}
	if holders, total, err := s.cashHolders(ctx); !miss("cash", err) {
		limit := s.cashbox.Limit(ctx)
		var over int64
		for _, h := range holders {
			if h.Held >= limit {
				over++
			}
		}
		a.DriversOverCash = n64(over)
		ov.Money.CashHeld = n64(total)
	}

	// ── الآن ───────────────────────────────────────────────────────
	lv := &ov.Live
	lv.Stages = map[string]num{}
	{
		rows, err := s.pg.Query(ctx, `
			SELECT status, count(*) FROM orders WHERE closed_at IS NULL GROUP BY status`)
		byStatus := map[string]int64{}
		if err == nil {
			for rows.Next() {
				var st string
				var c int64
				if err = rows.Scan(&st, &c); err != nil {
					break
				}
				byStatus[st] = c
			}
			rows.Close()
			if err == nil {
				err = rows.Err()
			}
		}
		if !miss("live_orders", err) {
			var open int64
			for _, c := range byStatus {
				open += c
			}
			lv.OrdersOpen = n64(open)
			for _, stage := range orders.LiveStages {
				var c int64
				for _, st := range orders.LiveStageStatuses(stage) {
					c += byStatus[st]
				}
				lv.Stages[stage] = n64(c)
			}
		}
	}
	// ══════════════════════════════════════════════════════════════════
	// **«طلباتٌ بلا سائق» بشرط لوحة الطلبات نفسِه** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// كانت مرحلةَ «البحث عن سائق» كلَّها وتُفتح على `?status=dispatching` —
	// **وفلترُ اللوحة «بلا سائق» يعدّ ما تجاوز مهلةَ إيجاد السائق**، فيقول
	// الرقمُ شيئاً والقائمةُ غيرَه. **وصارت تُعدّ بـ`orders.NoDriverSQL`** عبر
	// `BoardCounts` وتُفتح على `?filter=no_driver`.
	if c, err := s.orders.BoardCounts(ctx); !miss("no_driver", err) {
		a.OrdersUnassigned = n64(int64(c[orders.BoardNoDriver]))
	}
	if al, err := s.orders.Alerts(ctx); !miss("stuck", err) {
		lv.OrdersStuck = n64(int64(len(al)))
	}
	{
		// **ونصُّ العدِّ واحدٌ مع شاشة المراقب** (`driverShiftCounts`).
		onShift, busy, err := s.driverShiftCounts(ctx)
		if !miss("drivers", err) {
			lv.DriversOnShift, lv.DriversBusy, lv.DriversFree = n64(onShift), n64(busy), n64(onShift-busy)
		}
	}
	{
		var open, active int64
		err := s.pg.QueryRow(ctx, `
			SELECT count(*) FILTER (WHERE `+orders.OpenNowSQL+`),
			       count(*) FILTER (WHERE m.status = 'active')
			FROM merchants m`).Scan(&open, &active)
		if !miss("stores", err) {
			lv.StoresOpenNow, lv.StoresActive = n64(open), n64(active)
		}
	}

	// ── صحّةُ النظام ───────────────────────────────────────────────
	ov.Health = s.overviewHealth(ctx)

	// ── اليوم مقابل أمس ────────────────────────────────────────────
	if d, err := s.overviewDay(ctx, today, tomorrow); !miss("today", err) {
		ov.TodayStats = d
	}
	if d, err := s.overviewDay(ctx, yesterday, today); !miss("yesterday", err) {
		ov.Yesterday = d
	}
	{
		rows, err := s.pg.Query(ctx, `
			SELECT status, COALESCE(NULLIF(fail_reason, ''), ''), count(*)
			FROM orders
			WHERE status IN ('cancelled','rejected','failed')
			  AND closed_at >= $1 AND closed_at < $2
			GROUP BY 1, 2 ORDER BY 3 DESC, 1, 2 LIMIT 10`, today, tomorrow)
		if err == nil {
			for rows.Next() {
				var f ovFailure
				if err = rows.Scan(&f.Status, &f.Reason, &f.Count); err != nil {
					break
				}
				ov.Failures = append(ov.Failures, f)
			}
			rows.Close()
			if err == nil {
				err = rows.Err()
			}
		}
		miss("failures", err)
	}

	// ── المال اليوم ────────────────────────────────────────────────
	//
	// **والصافي من حساب صفحة الأرباح نفسِه** (`platformProfit`) بمدى اليوم —
	// **لا معادلةٌ ثالثة.**
	if p, err := s.platformProfit(ctx, ov.Today, ov.Today); !miss("profit", err) {
		ov.Money.Sales, ov.Money.Net, ov.Money.Losses = n64(p.Sales), n64(p.Net), n64(p.Losses())
		ov.Money.DeliveredOrdersCount = n64(int64(p.Orders))
	}
	{
		var bal, cn, camt int64
		err := s.pg.QueryRow(ctx, `
			SELECT
				(SELECT COALESCE(sum(balance), 0) FROM wallets WHERE is_treasury),
				(SELECT count(*) FROM driver_compensation_requests
				  WHERE status = 'approved' AND decided_at >= $1 AND decided_at < $2),
				(SELECT COALESCE(sum(amount), 0) FROM driver_compensation_requests
				  WHERE status = 'approved' AND decided_at >= $1 AND decided_at < $2)`,
			today, tomorrow).Scan(&bal, &cn, &camt)
		if !miss("treasury", err) {
			ov.Money.TreasuryBalance = n64(bal)
			ov.Money.CompensationsCount, ov.Money.CompensationsAmount = n64(cn), n64(camt)
		}
	}

	// ── آخرُ الأحداث ───────────────────────────────────────────────
	{
		rows, err := s.pg.Query(ctx, `
			SELECT a.id, a.action, a.entity, COALESCE(a.entity_id, ''),
			       COALESCE(CASE WHEN a.entity_id ~ '^[0-9a-fA-F-]{36}$' THEN
			           CASE a.entity
			           WHEN 'merchant' THEN (SELECT m.name FROM merchants m
			                                  WHERE m.id = a.entity_id::uuid)
			           WHEN 'user' THEN (SELECT COALESCE(NULLIF(u2.full_name, ''), u2.phone::text)
			                              FROM users u2 WHERE u2.id = a.entity_id::uuid)
			           END END, ''),
			       CASE WHEN a.entity_id ~ '^[0-9a-fA-F-]{36}$' THEN
			           CASE a.entity
			           WHEN 'order' THEN (SELECT o.number FROM orders o WHERE o.id = a.entity_id::uuid)
			           WHEN 'ticket' THEN (SELECT t.number FROM tickets t WHERE t.id = a.entity_id::uuid)
			           END END,
			       COALESCE(NULLIF(u.full_name, ''), u.phone::text, ''),
			       a.created_at
			FROM audit_log a
			LEFT JOIN users u ON u.id = a.actor_user_id
			WHERE a.action = ANY($1)
			ORDER BY a.id DESC LIMIT 10`, overviewEvents)
		if err == nil {
			for rows.Next() {
				var e ovEvent
				if err = rows.Scan(&e.ID, &e.Action, &e.Entity, &e.EntityID, &e.Label,
					&e.Number, &e.ActorName, &e.CreatedAt); err != nil {
					break
				}
				ov.Events = append(ov.Events, e)
			}
			rows.Close()
			if err == nil {
				err = rows.Err()
			}
		}
		miss("events", err)
	}

	// ── المنصّة بالأرقام ───────────────────────────────────────────
	if p, err := s.overviewPlatform(ctx, today, week); !miss("platform", err) {
		ov.Platform = p
	} else {
		ov.Platform = ovPlatform{Groups: []ovCount{}, Staff: []ovCount{}}
	}
	return ov
}

// overviewDay **أرقامُ يومٍ واحد** — `[from, to)`.
func (s *Server) overviewDay(ctx context.Context, from, to time.Time) (ovDay, error) {
	var d ovDay
	var orders_, delivered, failed, newCust, ratings int64
	err := s.pg.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM orders WHERE created_at >= $1 AND created_at < $2),
			(SELECT count(*) FROM orders WHERE status = 'delivered'
			   AND delivered_at >= $1 AND delivered_at < $2),
			(SELECT count(*) FROM orders WHERE status IN ('cancelled','rejected','failed')
			   AND closed_at >= $1 AND closed_at < $2),
			(SELECT round((avg(EXTRACT(EPOCH FROM delivered_at - created_at)) / 60)::numeric, 1)::float8
			   FROM orders WHERE status = 'delivered'
			   AND delivered_at >= $1 AND delivered_at < $2),
			(SELECT count(*) FROM user_roles WHERE role_code = 'customer'
			   AND granted_at >= $1 AND granted_at < $2),
			(SELECT round(avg(platform_stars)::numeric, 1)::float8 FROM order_ratings
			   WHERE created_at >= $1 AND created_at < $2 AND platform_stars IS NOT NULL),
			(SELECT round(avg(driver_stars)::numeric, 1)::float8 FROM order_ratings
			   WHERE created_at >= $1 AND created_at < $2 AND driver_stars IS NOT NULL),
			(SELECT count(*) FROM order_ratings WHERE created_at >= $1 AND created_at < $2)`,
		from, to).Scan(&orders_, &delivered, &failed, &d.AvgDeliveryMin, &newCust,
		&d.AvgPlatform, &d.AvgDriver, &ratings)
	if err != nil {
		return d, err
	}
	d.Orders, d.Delivered, d.Failed = n64(orders_), n64(delivered), n64(failed)
	d.NewCustomers, d.Ratings = n64(newCust), n64(ratings)
	return d, nil
}

// overviewPlatform **المنصّةُ بالأرقام** — كلُّ نوعٍ: مجموعُه وجديدُ اليوم
// وجديدُ آخر سبعة أيّام.
func (s *Server) overviewPlatform(ctx context.Context, today, week time.Time) (ovPlatform, error) {
	p := ovPlatform{Groups: []ovCount{}, Staff: []ovCount{}}
	rows, err := s.pg.Query(ctx, `
		SELECT role_code, count(*),
		       count(*) FILTER (WHERE granted_at >= $1),
		       count(*) FILTER (WHERE granted_at >= $2)
		FROM user_roles GROUP BY role_code ORDER BY role_code`, today, week)
	if err != nil {
		return p, err
	}
	byRole := map[string]ovCount{}
	for rows.Next() {
		var c ovCount
		if err := rows.Scan(&c.Key, &c.Total, &c.Today, &c.Week); err != nil {
			rows.Close()
			return p, err
		}
		byRole[c.Key] = c
		if authz.ClassOf(c.Key) != authz.ClassAccountType {
			p.Staff = append(p.Staff, c)
		}
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return p, err
	}
	for _, k := range []string{"customer", "driver", "sales"} {
		c := byRole[k]
		c.Key = k
		p.Groups = append(p.Groups, c)
	}

	type trip struct{ key, table, where string }
	for _, t := range []trip{
		{"stores", "merchants", "true"},
		{"items", "menu_items", "true"},
		{"zones", "delivery_zones", "active"},
		{"promos", "promo_codes", "active"},
	} {
		c := ovCount{Key: t.key}
		// **والأسماءُ ثوابتُ هنا لا مُدخَل** — فتُركَّب في النصّ بلا معاملات.
		if err := s.pg.QueryRow(ctx, `
			SELECT count(*), count(*) FILTER (WHERE created_at >= $1),
			       count(*) FILTER (WHERE created_at >= $2)
			FROM `+t.table+` WHERE `+t.where, today, week).
			Scan(&c.Total, &c.Today, &c.Week); err != nil {
			return p, err
		}
		p.Groups = append(p.Groups, c)
	}

	var ordered, dAct, dSus, dBlk, sAct, sNot int64
	if err := s.pg.QueryRow(ctx, `
		SELECT
			(SELECT count(DISTINCT customer_id) FROM orders
			  WHERE created_at >= $1 AND customer_id IS NOT NULL),
			(SELECT count(*) FROM users u JOIN user_roles ur
			   ON ur.user_id = u.id AND ur.role_code = 'driver' WHERE u.status = 'active'),
			(SELECT count(*) FROM users u JOIN user_roles ur
			   ON ur.user_id = u.id AND ur.role_code = 'driver' WHERE u.status = 'suspended'),
			(SELECT count(*) FROM users u JOIN user_roles ur
			   ON ur.user_id = u.id AND ur.role_code = 'driver' WHERE u.status = 'blocked'),
			(SELECT count(*) FROM merchants WHERE status = 'active'),
			(SELECT count(*) FROM merchants WHERE status <> 'active')`, week).
		Scan(&ordered, &dAct, &dSus, &dBlk, &sAct, &sNot); err != nil {
		return p, err
	}
	p.CustomersOrderedWeek = n64(ordered)
	p.DriversActive, p.DriversSuspended, p.DriversBlocked = n64(dAct), n64(dSus), n64(dBlk)
	p.StoresActive, p.StoresNotActive = n64(sAct), n64(sNot)
	return p, nil
}

// ── صحّةُ النظام ───────────────────────────────────────────────────────

// routeProbe **آخرُ فحصٍ لمحرّك الطرق** — يُحفظ دقيقةً: الرئيسيّةُ تُحدَّث
// مع كلّ حدث، **ومسارٌ يُطلب في كلّ تحديثٍ حِملٌ بلا خبرٍ جديد.**
var routeProbe struct {
	sync.Mutex
	at     time.Time
	status string
}

func (s *Server) overviewHealth(ctx context.Context) ovHealth {
	h := ovHealth{API: "ok", Database: "ok", WhatsApp: "unknown", Routing: "unknown"}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := s.pg.Ping(pctx); err != nil {
		h.Database = "down"
	}
	if s.otpStatus != nil {
		st := s.otpStatus()
		connected, _ := st["connected"].(bool)
		logged, _ := st["logged_in"].(bool)
		switch {
		case st["provider"] == "dev":
			h.WhatsApp = "dev"
		case connected && logged:
			h.WhatsApp = "ok"
		case connected:
			h.WhatsApp = "not_paired"
		default:
			h.WhatsApp = "down"
		}
	}
	h.Routing = s.routingHealth(ctx)
	return h
}

func (s *Server) routingHealth(ctx context.Context) string {
	if s.route == nil || !s.route.Enabled() {
		return "off"
	}
	routeProbe.Lock()
	defer routeProbe.Unlock()
	if time.Since(routeProbe.at) < time.Minute && routeProbe.status != "" {
		return routeProbe.status
	}
	pctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	// **نقطتان في الرقّة** — مسارٌ قصيرٌ يكفي ليقول «المحرّكُ يجيب».
	_, err := s.route.Route(pctx, routing.Point{Lat: 35.9528, Lng: 39.0079},
		routing.Point{Lat: 35.9500, Lng: 39.0150})
	routeProbe.at = time.Now()
	routeProbe.status = "ok"
	if err != nil {
		routeProbe.status = "down"
	}
	return routeProbe.status
}
