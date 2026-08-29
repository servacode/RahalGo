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

		// ══════════════════════════════════════════════════════════
		// **الزوّار — رقمان لا رقمٌ واحد**
		// ══════════════════════════════════════════════════════════
		//
		// (طلبُ المالك 2026-08-25.)
		//
		// **والفتحةُ غيرُ الشخص**: من فتح التطبيقَ عشرَ مرّاتٍ اليومَ
		// يُعدّ عشراً في `OpensToday` وواحداً في `DevicesToday`.
		//
		// **والأوّلُ يقيس الحركة، والثاني يقيس الناس** — ومن خلط بينهما
		// ظنّ عشرةَ زبائنَ حيث زبونٌ واحدٌ ملّ الانتظار.
		OpensToday   int64 `json:"opens_today"`
		Opens7       int64 `json:"opens_7d"`
		Opens30      int64 `json:"opens_30d"`
		DevicesToday int   `json:"devices_today"`
		Devices7     int   `json:"devices_7d"`
		Devices30    int   `json:"devices_30d"`
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
		-- **والمهلُ تُمرَّر معاملاتٍ من المخزن** — لا تُقرأ هنا بأرقامٍ
		-- مكتوبةٍ ثانيةً يحملها الفهرسُ أيضاً. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)
		lim AS (
			SELECT $1::int AS accept_min, $2::int AS driver_min, $3::int AS delivery_min
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
			(SELECT count(*) FROM promo_codes WHERE active)`,
		s.settings.GetInt(r.Context(), "orders.accept_timeout_min"),
		s.settings.GetInt(r.Context(), "orders.driver_timeout_min"),
		s.settings.GetInt(r.Context(), "orders.delivery_timeout_min")).
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

	// ══════════════════════════════════════════════════════════════════
	// **وأرقامُ الزوّار في استعلامٍ ثانٍ لا في الأوّل**
	// ══════════════════════════════════════════════════════════════════
	//
	// **والأوّلُ استعلامٌ ضخمٌ فيه عشرةُ CTE** — وإقحامُ جدولين جديدين
	// فيه يجعل عطبَ أحدِهما يُسقط الشاشةَ كلَّها.
	//
	// **ولا يُسقَط الردُّ إن عَطِبا**: من عجز عن عدّ الزوّار يبقى يرى
	// طلباتِه وسائقيه. **والصفرُ هنا أهونُ من شاشةٍ فارغة.**
	_ = s.pg.QueryRow(r.Context(), `
		WITH d AS (
			SELECT (now() AT TIME ZONE 'Asia/Damascus')::date AS today
		)
		SELECT
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d
		            WHERE day = d.today), 0),
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d
		            WHERE day > d.today - 7), 0),
		  COALESCE((SELECT sum(opens) FROM app_opens_daily, d
		            WHERE day > d.today - 30), 0),
		  (SELECT count(*) FROM device_tokens
		    WHERE last_seen_at >= now() - interval '1 day'),
		  (SELECT count(*) FROM device_tokens
		    WHERE last_seen_at >= now() - interval '7 days'),
		  (SELECT count(*) FROM device_tokens
		    WHERE last_seen_at >= now() - interval '30 days')
	`).Scan(&st.OpensToday, &st.Opens7, &st.Opens30,
		&st.DevicesToday, &st.Devices7, &st.Devices30)

	httpx.JSON(w, http.StatusOK, st)
}
