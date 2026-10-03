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

		// ══════════════════════════════════════════════════════════
		// **بانتظار قرارك — ما لا يتحرّك حتّى تتحرّك يد**
		// ══════════════════════════════════════════════════════════
		//
		// (قرارُ المالك ٢٠٢٦-١٠-٠٣: أعلى الرئيسيّة بطاقاتٌ تُفتح كلٌّ
		// على صفحتها.) **و`payouts_pending` و`tickets_open` من فوق** —
		// لا تُعدّ مرّتين.
		CompensationsPending int `json:"compensations_pending"`
		// ReportsWaiting **بلاغُ سائقٍ ينتظر قرارَ المكتب** — `CountAwaitingOffice`.
		ReportsWaiting  int `json:"reports_waiting"`
		EmergenciesOpen int `json:"emergencies_open"`
		// OrdersUnassigned **في الطابور بلا سائقٍ بعد مهلة الإسناد اليدويّ** —
		// المهلةُ نفسُها التي تُظهر زرَّ «إسناد» في البطاقة.
		OrdersUnassigned int `json:"orders_unassigned"`
		LeadsNew         int `json:"leads_new"`
		MenuPending      int `json:"menu_pending"`
		// DriversOverCash **سائقون بلغ نقدُهم السقف** — بحساب صفحة النقد نفسِه.
		DriversOverCash int `json:"drivers_over_cash"`
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

	// ══════════════════════════════════════════════════════════════════
	// **وما ينتظر القرارَ يُسقط الردَّ إن عَطِب** — لا كالزوّار
	// ══════════════════════════════════════════════════════════════════
	//
	// **صفرٌ في «بانتظار قرارك» يُقرأ «لا شيءَ ينتظرك»** — وهو كذبٌ يترك
	// سائقاً في الشارع. **والعطبُ المُعلَن أصدقُ منه.**
	if err := s.pg.QueryRow(r.Context(), `
		SELECT
			(SELECT count(*) FROM driver_compensation_requests WHERE status = 'pending'),
			(SELECT count(*) FROM driver_emergencies WHERE status = 'open'),
			(SELECT count(*) FROM orders
			  WHERE status = 'dispatching' AND driver_id IS NULL AND closed_at IS NULL
			    AND dispatched_at < now() - make_interval(mins => $1::int)),
			(SELECT count(*) FROM merchant_leads WHERE status = 'new'),
			(SELECT count(*) FROM menu_items WHERE NOT approved),
			(SELECT count(*) FROM (
			    SELECT driver_id FROM driver_cash_entries GROUP BY driver_id
			    HAVING COALESCE(sum(amount), 0) > 0 AND COALESCE(sum(amount), 0) >= $2) x)`,
		s.settings.GetInt(r.Context(), "orders.manual_assign_after_min"),
		s.cashbox.Limit(r.Context())).
		Scan(&st.CompensationsPending, &st.EmergenciesOpen, &st.OrdersUnassigned,
			&st.LeadsNew, &st.MenuPending, &st.DriversOverCash); err != nil {
		s.respondErr(w, err)
		return
	}
	n, err := s.orders.CountAwaitingOffice(r.Context())
	if err != nil {
		s.respondErr(w, err)
		return
	}
	st.ReportsWaiting = n

	httpx.JSON(w, http.StatusOK, st)
}
