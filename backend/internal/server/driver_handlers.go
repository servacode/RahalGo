package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// بوابة السائق.
//
// كان السائق **الطرف الوحيد بلا باب**: سبعة انتقالات من أصل أربعة عشر مخوّلة
// لدوره في خارطة الحالات، ولا نقطة واحدة في الخادم تناديها باسمه — فيحرّكها
// موظّف العمليات نيابةً عنه بالهاتف. النظام يقول إنه فاعل، والواقع أن غيره
// يكتب باسمه.
//
// وكل نقطة هنا تتحقق أن الطلب **طلبُه هو** — إلا الطابور، فهو معروضٌ للجميع
// بحكم كونه طابوراً.

var (
	errNotYourOrder  = httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	errOrderTaken    = httpx.NewError(http.StatusConflict, "order_taken", "errors.order_taken")
	errCashLimitFull = httpx.NewError(http.StatusConflict, "cash_limit_reached", "errors.cash_limit_reached")
	errBadFailReason = httpx.NewError(http.StatusBadRequest, "bad_fail_reason", "errors.bad_fail_reason")
	errNotOnShift    = httpx.NewError(http.StatusConflict, "not_on_shift", "errors.not_on_shift")
	errTooManyActive = httpx.NewError(http.StatusConflict, "too_many_active_orders", "errors.too_many_active_orders")
)

// handleDriverMe حالته: دوامه، ونقدٌ بحوزته، وأجرٌ له، وحصيلة يومه.
func (s *Server) handleDriverMe(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	var out struct {
		FullName       string     `json:"full_name"`
		OnShift        bool       `json:"on_shift"`
		ShiftStartedAt *time.Time `json:"shift_started_at"`
		// ما عليه وما له — دفتران لا يلتقيان
		CashHeld  int64 `json:"cash_held"`
		CashLimit int64 `json:"cash_limit"`
		Balance   int64 `json:"balance"`
		// حصيلة اليوم بتوقيت دمشق لا UTC: يومُ السائق ينتهي عنده لا في غرينتش
		TodayDelivered int   `json:"today_delivered"`
		TodayEarned    int64 `json:"today_earned"`
		ActiveOrders   int   `json:"active_orders"`
	}
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.full_name, u.on_shift, u.shift_started_at,
		       COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = u.id), 0),
		       COALESCE((SELECT (value#>>'{}')::bigint FROM app_settings
		                 WHERE key = 'drivers.cash_limit'), 500000),
		       COALESCE((SELECT balance FROM wallets WHERE user_id = u.id), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.status = 'delivered'
		          AND o.delivered_at AT TIME ZONE 'Asia/Damascus' >= date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')),
		       COALESCE((SELECT sum(t.amount) FROM wallet_transactions t
		                 WHERE t.user_id = u.id AND t.kind = 'driver_earning'
		                   AND t.created_at AT TIME ZONE 'Asia/Damascus' >= date_trunc('day', now() AT TIME ZONE 'Asia/Damascus')), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL)
		FROM users u WHERE u.id = $1`, uid).
		Scan(&out.FullName, &out.OnShift, &out.ShiftStartedAt, &out.CashHeld, &out.CashLimit,
			&out.Balance, &out.TodayDelivered, &out.TodayEarned, &out.ActiveOrders)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleDriverShift يرفع علَم الدوام أو ينزله.
func (s *Server) handleDriverShift(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		On bool `json:"on"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// إنهاء الدوام وبيده طلبات جارية يترك زبائن معلّقين — يُنهي ما بدأه أولاً
	if !req.On {
		var active int
		if err := s.pg.QueryRow(r.Context(),
			`SELECT count(*) FROM orders WHERE driver_id = $1 AND closed_at IS NULL`,
			userIDFrom(r)).Scan(&active); err != nil {
			s.respondErr(w, err)
			return
		}
		if active > 0 {
			s.respondErr(w, httpx.NewError(http.StatusConflict, "has_active_orders", "errors.has_active_orders"))
			return
		}
	}
	if _, err := s.pg.Exec(r.Context(), `
		UPDATE users SET on_shift = $2,
			shift_started_at = CASE WHEN $2 THEN now() ELSE NULL END,
			updated_at = now()
		WHERE id = $1`, userIDFrom(r), req.On); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops") // العمليات ترى من يعمل الآن
	httpx.JSON(w, http.StatusOK, map[string]any{"on_shift": req.On})
}

type driverOrder struct {
	ID            string     `json:"id"`
	Number        int64      `json:"number"`
	Status        string     `json:"status"`
	MerchantName  string     `json:"merchant_name"`
	MerchantPhone *string    `json:"merchant_phone"`
	AddressText   string     `json:"address_text"`
	Lat           float64    `json:"lat"`
	Lng           float64    `json:"lng"`
	CustomerName  string     `json:"customer_name"`
	CustomerPhone string     `json:"customer_phone"`
	Total         int64      `json:"total"`
	CashDue       int64      `json:"cash_due"`
	ItemsCount    int        `json:"items_count"`
	ReadyAt       *time.Time `json:"ready_at"`
	PrepMinutes   *int       `json:"prep_minutes"`
	AcceptedAt    *time.Time `json:"accepted_at"`
	CreatedAt     time.Time  `json:"created_at"`
	// PickupLat نقطةُ الاستلام البديلة — تُملأ حين تكون البضاعةُ ليست في
	// المتجر: **طارئٌ وقع لسائقٍ سابقٍ وهي في يده حيث وقف.**
	//
	// **ومن ذهب إلى المطعم استلم طلباً ثانياً من مطبخٍ حضّر واحداً** — فتُدفع
	// البضاعةُ مرّتين، ويبقى الطلبُ الأوّل في الشارع.
	PickupLat  *float64 `json:"pickup_lat"`
	PickupLng  *float64 `json:"pickup_lng"`
	PickupNote string   `json:"pickup_note"`
	// AcceptsReturns هل يستردّ هذا المتجرُ بضاعةَ طلبٍ تعذّر تسليمُه.
	//
	// **سياسةُ متجرٍ لا قاعدةُ منصة** — وهي ما يقرّر إلى أين يمضي السائقُ
	// بالطعام: **إلى المطعم أو إلى المكتب.** ومن لم يعرف وقف في الشارع
	// يتّصل بمن يسأله.
	AcceptsReturns bool `json:"merchant_accepts_returns"`
	// FailReason رمزُ التعذّر — **ليُعرَض عليه سببُه فيما بقي في يده.**
	FailReason string `json:"fail_reason"`
}

const driverOrderSelect = `
	SELECT o.id, o.number, o.status, m.name,
	       NULLIF(COALESCE(mo.whatsapp_phone::text, mo.phone::text), ''),
	       o.address_text, ST_Y(o.dropoff::geometry), ST_X(o.dropoff::geometry),
	       cu.full_name, cu.phone::text, o.total, o.cash_due,
	       COALESCE((SELECT sum(oi.qty) FROM order_items oi WHERE oi.order_id = o.id), 0),
	       o.ready_at, o.prep_minutes, o.accepted_at, o.created_at,
	       ST_Y(o.pickup_override::geometry), ST_X(o.pickup_override::geometry),
	       o.pickup_override_note, m.accepts_returns, COALESCE(o.fail_reason, '')
	FROM orders o
	JOIN merchants m ON m.id = o.merchant_id
	JOIN users cu ON cu.id = o.customer_id
	LEFT JOIN users mo ON mo.id = m.owner_user_id`

func (s *Server) scanDriverOrders(w http.ResponseWriter, r *http.Request, sql string, args ...any) {
	rows, err := s.pg.Query(r.Context(), sql, args...)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	out := []driverOrder{}
	for rows.Next() {
		var o driverOrder
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &o.MerchantName, &o.MerchantPhone,
			&o.AddressText, &o.Lat, &o.Lng, &o.CustomerName, &o.CustomerPhone,
			&o.Total, &o.CashDue, &o.ItemsCount, &o.ReadyAt, &o.PrepMinutes,
			&o.AcceptedAt, &o.CreatedAt,
			&o.PickupLat, &o.PickupLng, &o.PickupNote,
			&o.AcceptsReturns, &o.FailReason); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, o)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleDriverQueue **«طلباتٌ قادمة»** — ما عُرض على هذا السائق في دوره.
//
// # ولم يكن كذلك
//
// كان الشرطُ `offered_driver_id IS NULL OR = أنا` — **فالطلبُ الذي انقضى دورُه
// على الجميع يعود مشاعاً يراه الكلّ ومن سبق أخذ.** وهو **«الأسرعُ التقاطاً»
// بعينه**: النمطُ الذي رفضه المالك.
//
// **فكان النظامُ يقول «بالترتيب» ويعمل «بالأسرع» في كلّ مرّةٍ ينقضي فيها
// الدور** — ولا أحدَ يرى الانتقال، لأنّ الشاشة تُسمّيه «الطابور» في الحالين.
//
// **قرارُ المالك (٢٠٢٦-٠٨-٠٣)**: «نحن الآن نعمل على النظام التلقائيّ لتوزيع
// الطلبات وليس للأسرع التقاطاً» · «أمّا الطابور ألغِه».
//
// # وأين يذهب طلبٌ لم يبقَ له سائق
//
// **إلى العمليات لا إلى المشاع.** يبقى في `dispatching` بلا عرض، **ويظهر
// للإدارة زرُّ الإسناد اليدويّ بعد `orders.manual_assign_after_min`** —
// وينبّهها الحارسُ بـ`no_driver`. **فيدٌ تُسنده خيرٌ من عشرةٍ تتسابق عليه.**
//
// وفي نمط «الأسرع» يبقى السلوكُ القديم — **لأنه هو النمطُ نفسُه.**
func (s *Server) handleDriverQueue(w http.ResponseWriter, r *http.Request) {
	// **في «بالترتيب» لا يرى السائقُ إلّا ما عُرض عليه باسمه.**
	mine := `AND o.offered_driver_id = $1`
	if s.orders.AssignmentMode(r.Context()) != "rotation" {
		mine = `AND (o.offered_driver_id IS NULL OR o.offered_driver_id = $1)`
	}
	// **ومعرّفُ السائق يُمرَّر.**
	//
	// كان `$1` مكتوباً في الاستعلام **ولا يُمرَّر إليه شيء** — فيردّ الاستعلامُ
	// `expected 1 arguments, got 0`، **وتردّ الواجهةُ خمسمئة في كلّ نداء.**
	// وشاشةُ السائق تبتلع الخطأ (`.catch(() => undefined)`) **فتبقى القائمةُ
	// فارغةً بلا كلمة.**
	//
	// **وهذا سببُ «لم يتم تحويل الطلب للسائق»**: لم يكن الترتيبُ معطوباً —
	// **كان الطابورُ نفسُه لا يُقرأ أبداً.** فلم يرَ سائقٌ طلباً قطّ، واضطُرّ
	// المالكُ إلى الإسناد اليدويّ في كلّ مرّة.
	//
	// **ولم يُمسك في بناءٍ ولا `vet` ولا اختبار**: عددُ الوسائط يُفحص وقتَ
	// التنفيذ لا وقتَ الترجمة، **ولا اختبارَ كان ينادي هذا المسار.**
	s.scanDriverOrders(w, r, driverOrderSelect+`
		WHERE o.status = 'dispatching' AND o.driver_id IS NULL
		  `+mine+`
		ORDER BY o.ready_at NULLS LAST, o.created_at
		LIMIT 50`, userIDFrom(r))
}

// handleDriverOrders طلباته هو — الجارية أولاً.
//
// **ومهمّةٌ لا تنتهي بإغلاق الطلب: تنتهي حين تخرج البضاعةُ من يده.**
//
// كان الشرطُ `closed_at IS NULL` وحدَه، **و`failed` نهايةٌ تُغلق** — فيضغط
// السائقُ «تعذّر التسليم» **فيختفي الطلبُ من شاشته والطعامُ في صندوقه.** ولا
// يبقى له ما يقول أين يذهب به، **ولا زرٌّ يُقرّ به أنّه أعاده**، ونقطةُ
// الإرجاع (`handleDriverReturn`) مبنيّةٌ لا يصلها أحد.
//
// **وقاعدةُ المالك تحسمها** (٢٠٢٦-٠٨-٠٣): «يجب أن ينتهي الطلبُ ويعود السائقُ
// إلى المكتب» — **والعودةُ فعلٌ يُقرّ به، لا افتراض.**
//
// فيبقى الطلبُ ظاهراً حتى تُحسم بضاعتُه: **أعادها إلى المتجر** (`returned_at`)
// **أو حسمتها الإدارة** (`goods_settled_to`). ولا ثالثَ يُبقيه معلّقاً إلى
// الأبد.
func (s *Server) handleDriverOrders(w http.ResponseWriter, r *http.Request) {
	s.scanDriverOrders(w, r, driverOrderSelect+`
		WHERE o.driver_id = $1
		  AND (o.closed_at IS NULL
		       OR (o.status = 'failed'
		           AND o.returned_at IS NULL
		           AND o.goods_settled_to IS NULL))
		ORDER BY o.created_at`, userIDFrom(r))
}

// handleDriverAccept يأخذ السائق طلباً من الطابور.
func (s *Server) handleDriverAccept(w http.ResponseWriter, r *http.Request) {
	uid := userIDFrom(r)
	orderID := chi.URLParam(r, "id")

	var onShift bool
	var active int
	var held, limit, cashDue int64
	if err := s.pg.QueryRow(r.Context(), `
		SELECT u.on_shift,
		       COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = u.id), 0),
		       COALESCE((SELECT (value#>>'{}')::bigint FROM app_settings
		                 WHERE key = 'drivers.cash_limit'), 500000),
		       COALESCE((SELECT cash_due FROM orders WHERE id = $2), 0),
		       (SELECT count(*) FROM orders o WHERE o.driver_id = u.id AND o.closed_at IS NULL)
		FROM users u WHERE u.id = $1`, uid, orderID).
		Scan(&onShift, &held, &limit, &cashDue, &active); err != nil {
		s.respondErr(w, err)
		return
	}
	if !onShift {
		s.respondErr(w, errNotOnShift)
		return
	}
	// سقفُ ما بيده: كان يأخذ ما شاء ما دام السقف النقدي يتّسع — والسقف النقدي
	// لا يمنع تكديس الطلبات الصغيرة. وخمسةُ طلبات بيد سائقٍ واحد تعني أربعة
	// زبائن ينتظرون ساعة.
	if int64(active) >= s.settings.GetInt(r.Context(), "drivers.max_active_orders") {
		s.respondErr(w, errTooManyActive)
		return
	}
	// السقف يُفحص **قبل** القبول لا عند التسليم: رفضٌ عند الباب أرحم من طلبٍ
	// يحمله ثم يعجز عن إقفاله.
	if held+cashDue > limit {
		s.respondErr(w, errCashLimitFull)
		return
	}

	// **والحارسُ نفسُه في القبول لا في العرض وحده.**
	//
	// شاشةٌ لا تعرض الطلبَ لا تمنع من ينادي الواجهةَ البرمجية مباشرةً —
	// **وحجبٌ في العرض وحدَه وعدٌ بحجب.**
	own := `AND (offered_driver_id IS NULL OR offered_driver_id = $2)`
	if s.orders.AssignmentMode(r.Context()) == "rotation" {
		own = `AND offered_driver_id = $2`
	}

	// **الإسناد ذرّي**: الشرط `driver_id IS NULL` داخل التحديث نفسه. سائقان
	// يضغطان معاً — أحدهما يُحدّث صفّاً والآخر يجد صفراً. ولو فُحص ثم حُدّث
	// لأخذاه معاً.
	//
	// **والدورُ شرطٌ في التحديث لا فحصٌ قبله**: سائقٌ يرى الطلبَ في لحظة
	// انتقال الدور إليه ثم ينقضي وهو يضغط — الشرطُ هنا يمنعه، والفحصُ قبله
	// يسمح به.
	tag, err := s.pg.Exec(r.Context(), `
		UPDATE orders SET driver_id = $2, updated_at = now(),
		    offered_driver_id = NULL, offer_expires_at = NULL
		WHERE id = $1 AND driver_id IS NULL AND status = 'dispatching'
		  `+own, orderID, uid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, errOrderTaken)
		return
	}

	// **من أخذ طلباً هبط إلى آخر الصفّ** — وهو ما يجعل الترتيبَ يُصحّح نفسه
	// بلا دفترٍ يمسكه أحد.
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE users SET last_assigned_at = now() WHERE id = $1`, uid); err != nil {
		s.logger.Error("الترتيب: تعذّر تحديث آخر إسناد", "driver", uid, "error", err)
	}

	o, err := s.orders.Transition(r.Context(), uid, []string{"driver"}, orderID, orders.StAssigned, "")
	if err != nil {
		// تراجعٌ عن الإسناد: لولاه لبقي الطلب محجوزاً لسائقٍ لم يقبله المحرّك
		_, _ = s.pg.Exec(r.Context(),
			`UPDATE orders SET driver_id = NULL WHERE id = $1 AND driver_id = $2`, orderID, uid)
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// handleDriverTransition ينقل الطلب في مساره — والمحرّك يحكم ما يُسمح.
func (s *Server) handleDriverTransition(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	req, err := decode[struct {
		To   string `json:"to"`
		Note string `json:"note"`
		// Reason رمزٌ من `FailReasons` — **يلزم عند التعذّر**، ومنه يُشتقّ
		// الذنبُ الذي يقرّر التعويض.
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **سببٌ مصنَّفٌ لا نصٌّ حرّ.**
	//
	// كُتب في تجربةٍ حيّة «الزبون لا يقبل او رفض او لم اجد احد او العنوان
	// وهمي» — **أربعةُ أحكامٍ في سطر**، وأحدُها يستوجب مراجعةَ زبونٍ والآخر
	// لا يستوجب شيئاً. **ونصٌّ حرٌّ لا يُعدّ ولا يُقاس.**
	if req.To == orders.StFailed && orders.FaultOf(req.Reason) == "" {
		s.respondErr(w, errBadFailReason)
		return
	}
	// **ما لا يُستدرَك يلزمه سبب — والسائقُ كالعمليات في هذا.**
	//
	// كان الحارسُ على اللوحة وحدها، فيُفشل السائقُ طلباً بلا كلمة. و«فشل»
	// بلا سبب تُقرأ على وجوهٍ: أالزبونُ لم يردّ؟ أالمطعمُ مغلق؟ أالسائق
	// تعب؟ — **ثلاثةُ أخطاءٍ في ثلاث جهاتٍ يُخفيها لفظٌ واحد.**
	//
	// **والرمزُ المصنَّفُ يُغني عن النصّ.**
	//
	// الحارسان معاً كانا يطلبان شيئين عن واقعةٍ واحدة: رمزاً **وكلمةً**. وهو
	// خلافُ ما قرّرناه في `failreasons.go` بالحرف — «تفصيلُ ما وقع يبقى في
	// نصٍّ **اختياريّ** بجانب السبب: القائمةُ تُصنّف والنصُّ يشرح».
	//
	// **ومن أُلزم بالكتابة على درّاجةٍ تحت الشمس كتب حرفاً ليمرّ** — فيمتلئ
	// الحقلُ بـ«1» و«.» و«اا»، **ويصير الإلزامُ ضجيجاً يُفسد ما جُمع.** وقد
	// وقع فعلاً: `#1004` سببُه المحفوظ «1».
	note := strings.TrimSpace(req.Note)
	if requiresReason[req.To] && note == "" && req.Reason == "" {
		s.respondErr(w, errReasonRequired)
		return
	}

	// **ولا «سُلّم» بلا إثبات.**
	//
	// قاعدتُنا أنّ الزبونَ يُصدَّق أوّلَ مرّة — **وقاعدةٌ بلا دليلٍ تكلفةٌ بلا
	// سقف.** والإثباتُ صورةٌ بموقعٍ ووقت، **أو كلمةٌ تقول لماذا تعذّرت.**
	if req.To == orders.StDelivered {
		if err := s.requireProofBeforeDelivery(r, orderID); err != nil {
			s.respondErr(w, err)
			return
		}
	}

	o, err := s.orders.TransitionWithReason(r.Context(), userIDFrom(r), []string{"driver"},
		orderID, req.To, clip(note, 300), req.Reason)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, o)
}

// handleDriverRelease يفكّ إسناده فيعود الطلب إلى الطابور.
//
// حقٌّ يملكه في الخارطة (`assigned → dispatching`): تعطّلت درّاجته أو أخطأ
// التقدير. وتركُ الطلب معلّقاً بلا سائق أسوأ من إعادته إلى الطابور.
func (s *Server) handleDriverRelease(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	if !s.driverOwnsOrder(r, orderID) {
		s.respondErr(w, errNotYourOrder)
		return
	}
	req, _ := decode[struct {
		Note string `json:"note"`
	}](r)
	note := ""
	if req != nil {
		note = clip(req.Note, 300)
	}
	if _, err := s.orders.Transition(r.Context(), userIDFrom(r), []string{"driver"},
		orderID, orders.StDispatching, note); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE orders SET driver_id = NULL WHERE id = $1`, orderID); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"released": true})
}

// driverOwnsOrder يتحقق أن الطلب مُسنَد لهذا السائق.
func (s *Server) driverOwnsOrder(r *http.Request, orderID string) bool {
	var owns bool
	err := s.pg.QueryRow(r.Context(),
		`SELECT EXISTS(SELECT 1 FROM orders WHERE id = $1 AND driver_id = $2)`,
		orderID, userIDFrom(r)).Scan(&owns)
	return err == nil && owns
}

// handleDriverCash كشف صندوقه: ما حصّله وما سلّمه.
func (s *Server) handleDriverCash(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT e.kind, e.amount, e.note, o.number, e.created_at
		FROM driver_cash_entries e
		LEFT JOIN orders o ON o.id::text = e.ref
		WHERE e.driver_id = $1
		ORDER BY e.id DESC LIMIT 100`, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()

	type entry struct {
		Kind        string    `json:"kind"`
		Amount      int64     `json:"amount"`
		Note        string    `json:"note"`
		OrderNumber *int64    `json:"order_number"`
		CreatedAt   time.Time `json:"created_at"`
	}
	out := []entry{}
	for rows.Next() {
		var e entry
		if err := rows.Scan(&e.Kind, &e.Amount, &e.Note, &e.OrderNumber, &e.CreatedAt); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, e)
	}
	httpx.JSON(w, http.StatusOK, out)
}
