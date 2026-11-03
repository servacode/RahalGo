package server

// ══════════════════════════════════════════════════════════════════════
// **عدّاداتُ القائمة وشريطُ الانتباه — نداءٌ واحدٌ للوحة كلِّها**
// ══════════════════════════════════════════════════════════════════════
//
// (إعادةُ تصميم اللوحة ٢٠٢٦-١١-٠٣.) كانت القائمةُ تعرف عدّادَين فقط (السوق والطوارئ)،
// والرئيسيّةُ لمدير المنصّة وحدَه. **وصار لكلّ موظّفٍ شريطُ «بدّه منك انتباه» بما يملكه.**
//
// # القاعدة
//
// **قراءةٌ فقط** — لا يكتب ولا يحرّك مالاً ولا حالة. **وكلُّ رقمٍ بقدرة صفحته**: من لا يملك
// القدرةَ لا يصله الرقمُ (`null`) ولا يُحسب له أصلاً. **وهي مستثناةٌ من جدول السياسة**
// (`authz.Exempt`) لأنّ ما فيها قدراتٌ شتّى يفحصها المعالِجُ بنفسه، كما فعلت `/ops/status`.
//
// # ولا شرطَ عدٍّ ثانياً
//
// **نفسُ شروط الرئيسيّة** (`buildOverview`) وصفحاتِها: العالقُ والبلا-سائق من `BoardCounts`،
// والنقدُ من `cashOverviewOf`، والشكاوى من `support.Count`، والتوسّعُ من `expWaitingSQL`.
// **والرقمُ الذي لم يُقرأ `null` واسمُه في `missing`** — لا صفرٌ يكذب.

import (
	"context"
	"net/http"
	"sort"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/support"
)

// menuAttention **ما ينتظر انتباهاً** — كلُّ حقلٍ `null` إن لم يملك الفاعلُ قدرتَه أو لم يُقرأ.
type menuAttention struct {
	// ── orders.read ──
	StuckOrders      num      `json:"stuck_orders"`
	UnassignedOrders num      `json:"unassigned_orders"`
	ReportsWaiting   num      `json:"reports_waiting"`
	StoresSilent     num      `json:"stores_silent"`
	StoresSilentName []string `json:"stores_silent_names"`
	ExpansionWaiting num      `json:"expansion_waiting"`
	// ── finance.read ──
	CashOver            num      `json:"cash_over"`
	CashOverNames       []string `json:"cash_over_names"`
	CashOverdue         num      `json:"cash_overdue"`
	PayoutsPending      num      `json:"payouts_pending"`
	ObligationsOverdue  num      `json:"obligations_overdue"`
	ObligationRequests  num      `json:"obligation_requests"`
	CompensationPending num      `json:"compensations_pending"`
	// ── support.manage ──
	TicketsOpen num `json:"tickets_open"`
	TicketsLate num `json:"tickets_late"`
	// ── emergencies.manage ──
	EmergenciesOpen    num `json:"emergencies_open"`
	EmergenciesUnacked num `json:"emergencies_unacked"`
	// ── merchants.verify ──
	LeadsNew num `json:"leads_new"`
}

// menuPulse **نبضُ المنصّة الآن** — لمن يملك `orders.read`.
type menuPulse struct {
	OrdersOpen     num            `json:"orders_open"`
	Stages         map[string]num `json:"stages"`
	DriversOnShift num            `json:"drivers_on_shift"`
	DriversBusy    num            `json:"drivers_busy"`
	DriversFree    num            `json:"drivers_free"`
	StoresOpenNow  num            `json:"stores_open_now"`
	StoresActive   num            `json:"stores_active"`
	// AvgDeliveryMin **متوسّطُ زمن التوصيل اليوم** (يومُ المنصّة) — `null` إن لم يُسلَّم شيء.
	AvgDeliveryMin *float64 `json:"avg_delivery_min"`
}

// menuMoney **الرصيدُ الآن** — لمن يملك `finance.read`.
type menuMoney struct {
	CashHeld        num `json:"cash_held"`
	TreasuryBalance num `json:"treasury_balance"`
}

type menuCounts struct {
	GeneratedAt time.Time `json:"generated_at"`
	Today       string    `json:"today"`
	// DayStartHour ساعةُ بدء يوم المنصّة — يُقال في سطر الرئيسيّة.
	DayStartHour int64         `json:"day_start_hour"`
	Missing      []string      `json:"missing"`
	Attention    menuAttention `json:"attention"`
	Pulse        *menuPulse    `json:"pulse"`
	Money        *menuMoney    `json:"money"`
}

// handleMenuCounts `GET /admin/menu-counts`.
func (s *Server) handleMenuCounts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := timeoutCtx(r, 10*time.Second)
	defer cancel()
	httpx.JSON(w, http.StatusOK, s.buildMenuCounts(ctx, func(c authz.Capability) bool { return hasCap(r, c) }))
}

// firstTwo أوّلُ اسمين بلا فراغ ولا تكرار.
func firstTwo(names []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, n := range names {
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
		if len(out) == 2 {
			break
		}
	}
	return out
}

func (s *Server) buildMenuCounts(ctx context.Context, can func(authz.Capability) bool) menuCounts {
	now := time.Now()
	startHour := s.settings.GetInt(ctx, "platform.day_start_hour")
	today, _, _ := overviewDays(now, startHour)
	out := menuCounts{
		GeneratedAt: now, Today: today.Format("2006-01-02"), DayStartHour: startHour, Missing: []string{},
		Attention: menuAttention{StoresSilentName: []string{}, CashOverNames: []string{}},
	}
	miss := func(section string, err error) bool {
		if err == nil {
			return false
		}
		s.logger.Error("عدّاداتُ القائمة: قسمٌ لم يُقرأ", "section", section, "error", err)
		out.Missing = append(out.Missing, section)
		return true
	}
	a := &out.Attention

	// ── orders.read ─────────────────────────────────────────────────
	if can(authz.OrdersRead) {
		pulse := &menuPulse{Stages: map[string]num{}}
		out.Pulse = pulse
		if c, err := s.orders.BoardCounts(ctx); !miss("board", err) {
			a.StuckOrders = n64(int64(c[orders.BoardStuck]))
			a.UnassignedOrders = n64(int64(c[orders.BoardNoDriver]))
			a.ReportsWaiting = n64(int64(c[orders.BoardReports]))
		}
		// **متاجرُ ما ردّت**: طلباتٌ علقت بسبب «بانتظار قبول المتجر» — متجرٌ واحدٌ يُعدّ مرّة.
		if al, err := s.orders.Alerts(ctx); !miss("silent_stores", err) {
			stores := map[string]bool{}
			names := []string{}
			for _, x := range al {
				if x.Reason != orders.StuckNoAccept {
					continue
				}
				if !stores[x.MerchantName] {
					stores[x.MerchantName] = true
					names = append(names, x.MerchantName)
				}
			}
			a.StoresSilent = n64(int64(len(stores)))
			a.StoresSilentName = firstTwo(names)
		}
		{
			var n int64
			err := s.pg.QueryRow(ctx, `SELECT count(*) FROM coverage_requests r WHERE `+expWaitingSQL).Scan(&n)
			if !miss("expansion", err) {
				a.ExpansionWaiting = n64(n)
			}
		}
		// ── النبض ──
		{
			rows, err := s.pg.Query(ctx, `SELECT status, count(*) FROM orders WHERE closed_at IS NULL GROUP BY status`)
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
				pulse.OrdersOpen = n64(open)
				for _, stage := range orders.LiveStages {
					var c int64
					for _, st := range orders.LiveStageStatuses(stage) {
						c += byStatus[st]
					}
					pulse.Stages[stage] = n64(c)
				}
			}
		}
		if onShift, busy, err := s.driverShiftCounts(ctx); !miss("drivers", err) {
			pulse.DriversOnShift, pulse.DriversBusy, pulse.DriversFree = n64(onShift), n64(busy), n64(onShift-busy)
		}
		{
			var open, active int64
			err := s.pg.QueryRow(ctx, `
				SELECT count(*) FILTER (WHERE `+orders.OpenNowSQL+`),
				       count(*) FILTER (WHERE m.status = 'active')
				FROM merchants m`).Scan(&open, &active)
			if !miss("stores", err) {
				pulse.StoresOpenNow, pulse.StoresActive = n64(open), n64(active)
			}
		}
		{
			// **من ساعة بدء يوم المنصّة** — كالرئيسيّة (`overviewDay`).
			var avg *float64
			err := s.pg.QueryRow(ctx, `
				SELECT round((avg(EXTRACT(EPOCH FROM delivered_at - created_at)) / 60)::numeric, 1)::float8
				FROM orders WHERE status = 'delivered' AND delivered_at >= $1 AND delivered_at < $2`,
				today, today.AddDate(0, 0, 1)).Scan(&avg)
			if !miss("avg_delivery", err) {
				pulse.AvgDeliveryMin = avg
			}
		}
	}

	// ── finance.read ────────────────────────────────────────────────
	if can(authz.FinanceRead) {
		money := &menuMoney{}
		out.Money = money
		if cash, err := s.cashOverviewOf(ctx); !miss("cash", err) {
			a.CashOver, a.CashOverdue = n64(cash.OverCount), n64(cash.OverdueCount)
			money.CashHeld = n64(cash.Total)
			names := []string{}
			over := []cashHolder{}
			for _, h := range cash.Holders {
				if h.OverLimit {
					over = append(over, h)
				}
			}
			// **الأكثرُ تعرّضاً أوّلاً** — الاسمُ الذي يُقال أهمُّ من الأقدم.
			sort.SliceStable(over, func(i, j int) bool { return over[i].Exposure > over[j].Exposure })
			for _, h := range over {
				n := h.Name
				if n == "" {
					n = h.Phone
				}
				names = append(names, n)
			}
			a.CashOverNames = firstTwo(names)
		}
		{
			var pay, comp int64
			err := s.pg.QueryRow(ctx, `
				SELECT (SELECT count(*) FROM payout_requests p WHERE p.status = 'pending'),
				       (SELECT count(*) FROM driver_compensation_requests WHERE status = 'pending')`).Scan(&pay, &comp)
			if !miss("payouts", err) {
				a.PayoutsPending, a.CompensationPending = n64(pay), n64(comp)
			}
		}
		if sum, err := s.obligationsSummary(ctx); !miss("obligations", err) {
			a.ObligationsOverdue, a.ObligationRequests = n64(sum.Overdue), n64(sum.PendingRequests)
		}
		{
			var bal int64
			err := s.pg.QueryRow(ctx, `SELECT COALESCE(sum(balance), 0) FROM wallets WHERE is_treasury`).Scan(&bal)
			if !miss("treasury", err) {
				money.TreasuryBalance = n64(bal)
			}
		}
	}

	// ── support.manage ──────────────────────────────────────────────
	if can(authz.SupportManage) && s.support != nil {
		if n, err := s.support.Count(ctx, support.TicketFilter{Status: "unresolved"}); !miss("tickets", err) {
			a.TicketsOpen = n64(int64(n))
		}
		if n, err := s.support.Count(ctx, support.TicketFilter{LateHours: s.ticketLateHours(ctx)}); !miss("tickets_late", err) {
			a.TicketsLate = n64(int64(n))
		}
	}

	// ── emergencies.manage ──────────────────────────────────────────
	if can(authz.EmergenciesManage) {
		var open, unacked int64
		err := s.pg.QueryRow(ctx, `
			SELECT count(*), count(*) FILTER (WHERE acknowledged_at IS NULL)
			FROM driver_emergencies WHERE status = 'open'`).Scan(&open, &unacked)
		if !miss("emergencies", err) {
			a.EmergenciesOpen, a.EmergenciesUnacked = n64(open), n64(unacked)
		}
	}

	// ── merchants.verify ────────────────────────────────────────────
	if can(authz.MerchantsVerify) {
		var n int64
		err := s.pg.QueryRow(ctx, `
			SELECT count(*) FROM merchant_leads l WHERE l.status = 'new' AND l.sales_rep_user_id IS NOT NULL`).Scan(&n)
		if !miss("leads", err) {
			a.LeadsNew = n64(n)
		}
	}
	return out
}
