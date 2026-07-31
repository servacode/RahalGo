package server

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// إحصاءات لوحة القيادة — أرقام حية من قاعدة البيانات.
// handleAdminStats لوحة القيادة.
//
// كانت ثمانية أرقام كلُّها `count(*)`: كم زبوناً، كم متجراً، كم صنفاً. وهي
// **أرقام مخزون لا أرقام حركة** — تقول ماذا يملك النظام لا ماذا يجري فيه.
// فمن يفتح اللوحة صباحاً لا يعرف منها كيف كان أمس ولا ما يجري الآن.
//
// فصارت ثلاث طبقات: **الآن** (ما يحتاج تدخّلاً في هذه اللحظة)، ثم **اليوم**،
// ثم المخزون. والترتيب مقصود: من يفتح لوحةً يقرأ أعلاها.
//
// واليوم بتوقيت دمشق لا UTC: يوم المنصة ينتهي عند أهلها لا في غرينتش — وبفارق
// الثلاث ساعات تظهر طلبات الليل في يوم الغد.
func (s *Server) handleAdminStats(w http.ResponseWriter, r *http.Request) {
	var st struct {
		// ── الآن ──
		OrdersOpen     int `json:"orders_open"`  // طلبات جارية لم تُغلق
		OrdersStuck    int `json:"orders_stuck"` // منها ما تجاوز مهلته
		DriversOnShift int `json:"drivers_on_shift"`
		DriversBusy    int `json:"drivers_busy"`   // من بيده طلب الآن
		MerchantsOpen  int `json:"merchants_open"` // مفتوحة فعلاً حسب ساعاتها
		PayoutsPending int `json:"payouts_pending"`
		TicketsOpen    int `json:"tickets_open"`
		// ── اليوم ──
		OrdersToday    int   `json:"orders_today"`
		DeliveredToday int   `json:"delivered_today"`
		CancelledToday int   `json:"cancelled_today"`
		SalesToday     int64 `json:"sales_today"`     // مجمل المُسلَّم
		NetToday       int64 `json:"net_today"`       // صافي المنصة
		CashHeldTotal  int64 `json:"cash_held_total"` // نقدٌ بذمم السائقين الآن
		// ── المخزون ──
		Customers       int `json:"customers"`
		Drivers         int `json:"drivers"`
		SalesReps       int `json:"sales_reps"`
		MerchantsActive int `json:"merchants_active"`
		MerchantsTotal  int `json:"merchants_total"`
		MenuItems       int `json:"menu_items"`
		ZonesActive     int `json:"zones_active"`
		PromosActive    int `json:"promos_active"`
	}
	err := s.pg.QueryRow(r.Context(), `
		WITH day AS (
			SELECT date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')
			       AT TIME ZONE 'Asia/Damascus' AS start
		),
		-- أجور السائقين عن تسليمات اليوم في CTE مستقلّ: حسابُها داخل استعلامٍ
		-- تجميعيّ يجعل day.start عموداً غير مجمَّع في استعلامٍ فرعيّ مرتبط،
		-- وPostgres يرفضه بحقّ. (ولا علامات اقتباس خلفية هنا: النصّ الخام
		-- في Go محدَّدٌ بها، فواحدةٌ داخله تُنهيه.)
		driver_pay AS (
			SELECT COALESCE(sum(t.amount), 0) AS amount
			FROM wallet_transactions t
			JOIN orders o2 ON o2.id::text = t.ref
			CROSS JOIN day
			WHERE t.kind = 'driver_earning' AND o2.delivered_at >= day.start
		),
		lim AS (
			SELECT COALESCE((SELECT (value#>>'{}')::int FROM app_settings
			                 WHERE key='orders.accept_timeout_min'), 5) AS accept_min,
			       COALESCE((SELECT (value#>>'{}')::int FROM app_settings
			                 WHERE key='orders.driver_timeout_min'), 10) AS driver_min,
			       COALESCE((SELECT (value#>>'{}')::int FROM app_settings
			                 WHERE key='orders.delivery_timeout_min'), 60) AS delivery_min
		)
		SELECT
			(SELECT count(*) FROM orders WHERE closed_at IS NULL),
			(SELECT count(*) FROM orders o, lim WHERE o.closed_at IS NULL AND (
				(o.status = 'pending' AND o.created_at < now() - make_interval(mins => lim.accept_min))
				OR (o.status IN ('preparing','dispatching') AND o.updated_at < now() - make_interval(mins => lim.driver_min))
				OR (o.created_at < now() - make_interval(mins => lim.delivery_min)))),
			(SELECT count(*) FROM users WHERE on_shift),
			(SELECT count(DISTINCT driver_id) FROM orders WHERE driver_id IS NOT NULL AND closed_at IS NULL),
			(SELECT count(*) FROM merchants m WHERE m.status = 'active' AND NOT m.emergency_closed),
			(SELECT count(*) FROM payout_requests WHERE status = 'pending'),
			(SELECT count(*) FROM tickets WHERE status <> 'resolved'),

			(SELECT count(*) FROM orders, day WHERE created_at >= day.start),
			(SELECT count(*) FROM orders, day WHERE status = 'delivered' AND delivered_at >= day.start),
			(SELECT count(*) FROM orders, day WHERE status IN ('cancelled','rejected','failed')
			   AND closed_at >= day.start),
			(SELECT COALESCE(sum(total), 0) FROM orders, day
			   WHERE status = 'delivered' AND delivered_at >= day.start),
			-- صافي المنصة: عمولتها من المتاجر + ما بقي لها من رسم التوصيل بعد
			-- أجر السائق. ولا محفظة للمنصة تُجمَع منها — المنصة ليست طرفاً في
			-- دفترها، هي الدفتر. فيُحسب صافيها من الطلبات لا من قيدٍ لها.
			(SELECT COALESCE(sum(o.platform_commission), 0) + COALESCE(sum(o.delivery_fee), 0)
			        - (SELECT amount FROM driver_pay)
			 FROM orders o, day WHERE o.status = 'delivered' AND o.delivered_at >= day.start),
			(SELECT COALESCE(sum(held), 0) FROM driver_cash_boxes),

			(SELECT count(*) FROM user_roles WHERE role_code = 'customer'),
			(SELECT count(*) FROM user_roles WHERE role_code = 'driver'),
			(SELECT count(*) FROM user_roles WHERE role_code = 'sales'),
			(SELECT count(*) FROM merchants WHERE status = 'active'),
			(SELECT count(*) FROM merchants),
			(SELECT count(*) FROM menu_items),
			(SELECT count(*) FROM delivery_zones WHERE active),
			(SELECT count(*) FROM promo_codes WHERE active)`).
		Scan(&st.OrdersOpen, &st.OrdersStuck, &st.DriversOnShift, &st.DriversBusy,
			&st.MerchantsOpen, &st.PayoutsPending, &st.TicketsOpen,
			&st.OrdersToday, &st.DeliveredToday, &st.CancelledToday,
			&st.SalesToday, &st.NetToday, &st.CashHeldTotal,
			&st.Customers, &st.Drivers, &st.SalesReps, &st.MerchantsActive,
			&st.MerchantsTotal, &st.MenuItems, &st.ZonesActive, &st.PromosActive)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, st)
}
