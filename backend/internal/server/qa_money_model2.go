package server

// ══════════════════════════════════════════════════════════════════════
// **جسرُ شهادةِ النموذج ٢ الحيّة — على التجهيز وحدَه** (موافقةُ المالك ٢٠٢٦-١١-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// المالكُ وافق أن يُجرَّب نموذجُ المال الجديد (`finance.goods_outside`، الدفعات B3…B10) من أوّله
// لآخره على التجهيز. **وجسرُ QA ما كان يكفي**: الطلبُ العاديُّ يقف عند `assigned`، ولا بابَ
// لاسترجاعٍ أو فشلٍ بعد الاستلام أو طلبِ تعديل أو طلبِ واتساب، ولا قراءةَ لنتيجة المال.
//
// # ما يضيفه
//
//	/qa/setting  finance.goods_outside   المفتاحُ يُقلب ويرجع السابق (استثناءٌ ماليٌّ وحيد)
//	/qa/seed     m2_store_setup          متجرُ QA للنموذج ٢: عمولة ١٠٪، صنفٌ بهامش، مفتوحٌ ٢٤ ساعة
//	/qa/seed     order_advance           الطلبُ العاديُّ بالنموذج ٢ للسُّلّم كلِّه حتّى «سُلّم»
//	/qa/orders/{id}/…                    أفعالُ المكتب الحقيقيّة على طلب QA (استرجاع، باب، بضاعة،
//	                                     مصاري عالقة، تعديل، طلبُ تعديل، إثبات، ربطُ متجر)
//	/qa/whatsapp-orders                  طلبُ واتساب لرقم QA ومتجر QA وحدَهما
//	/qa/drivers/{id}/settle              تسليمُ صندوق سائق QA للمكتب (تنظيفٌ بعد الشهادة)
//	GET /qa/money?order_id=              قراءةُ نتيجة المال لطلب QA — قراءةٌ محضة
//
// # وحدودُه
//
// **كلُّه خلف `qaStagingEnabled`** (غيرُ مسجَّلٍ في الإنتاج ⇒ ٤٠٤، ودفاعٌ ثانٍ وقتَ الطلب).
// **ولا منطقَ عملٍ بديل**: أفعالُ المكتب تمرّ بمعالِجاتها الإنتاجيّة عينِها خلف `qaAdminActor`
// (والمُخطّى التخويلُ وحدَه)، والسُّلّمُ يمرّ بخدمة الطلبات الحقيقيّة (`Transition`/`AssignDriver`)
// — **فالدفعُ للمتجر عند الاستلام هو التسويةُ الحقيقيّةُ نفسُها.** **ولا يمسّ دلالةَ الدفتر.**
// **ولا يقبل إلّا طلبَ QA** (زبونُه رقمُ QA، أو مستلِمُ الواتساب رقمُ QA).

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/cashbox"
	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// qaMoneyModel2Key **مفتاحُ النموذج ٢** — الاستثناءُ الماليُّ الوحيد في قائمة `/qa/setting`.
const qaMoneyModel2Key = "finance.goods_outside"

// qaM2StoreName **متجرُ QA للنموذج ٢** — اسمٌ ثابتٌ يُعاد استعمالُه ولا يُضاعَف.
const qaM2StoreName = "QA-Money-Model2"

// qaM2ItemName **صنفُه** — كلفةُ ١٠٠٠٠ وسعرُ الزبون ١١٠٠٠ (هامشٌ ١٠٠٠، عمولةٌ ١٠٪ = ١٠٠٠).
const qaM2ItemName = "QA صنف النموذج ٢"

// qaQAPhoneSet **أرقامُ QA التي يُقبل طلبُها** — الزبونان ثابتان، والمستلِمُ في الواتساب منهما.
func qaQAPhoneSet() map[string]bool {
	m := map[string]bool{}
	for _, p := range []string{qaStagingPhone, qaStagingPhone2} {
		if n, ok := identity.NormalizePhone(p); ok {
			m[n] = true
		}
		m[p] = true
	}
	return m
}

// qaIsQAOrder **طلبُ QA؟** — زبونُه رقمُ QA (أو رقمُ فيديو الشرح)، **أو طلبُ واتساب بلا زبونٍ
// مستلِمُه رقمُ QA.** وكلُّ ما سواه يُرفض: لا يمسّ الجسرُ طلبَ زبونٍ حقيقيّ ولو على التجهيز.
func (s *Server) qaIsQAOrder(ctx context.Context, orderID string) bool {
	if !isUUID(orderID) {
		return false
	}
	var custPhone, recipient string
	var noCustomer bool
	if err := s.pg.QueryRow(ctx, `
		SELECT COALESCE(u.phone, ''), COALESCE(o.recipient_phone, ''), o.customer_id IS NULL
		FROM orders o LEFT JOIN users u ON u.id = o.customer_id
		WHERE o.id = $1::uuid`, orderID).Scan(&custPhone, &recipient, &noCustomer); err != nil {
		return false
	}
	set := qaQAPhoneSet()
	if set[custPhone] || qaIsVideoPhone(custPhone) {
		return true
	}
	return noCustomer && set[recipient]
}

// qaOrderScope **وسيطٌ: `{id}` طلبُ QA وإلّا ٤٠٣** — يُركَّب على كلّ فعلِ مكتبٍ بمعرّف طلب.
func (s *Server) qaOrderScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.qaStagingEnabled() {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		if !s.qaIsQAOrder(r.Context(), chi.URLParam(r, "id")) {
			s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_qa_order", "errors.forbidden"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// qaDriverScope **وسيطٌ: `{id}` سائقُ QA ثابتٌ وإلّا ٤٠٣** — لتسليم صندوقه وحدَه.
func (s *Server) qaDriverScope(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.qaStagingEnabled() {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		id := chi.URLParam(r, "id")
		var phone string
		if isUUID(id) {
			_ = s.pg.QueryRow(r.Context(), `SELECT phone FROM users WHERE id = $1::uuid`, id).Scan(&phone)
		}
		ok := false
		for _, p := range []string{qaStagingDriverPhone, qaStagingDriverPhoneB, qaStagingDriverPhoneC} {
			if n, nok := identity.NormalizePhone(p); (nok && n == phone) || p == phone {
				ok = true
			}
		}
		if !ok || phone == "" {
			s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_qa_driver", "errors.forbidden"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// qaAdminAllCaps **قدراتُ الأدمن كلُّها لهذا الطلب وحدَه** — بعد `qaAdminActor`.
//
// **الحاقنُ يضع الدورَ لا القدرات** (القدراتُ تُقرأ من جلسةٍ لا جلسةَ هنا)، **فمعالِجٌ يسأل
// `hasCapability`** (موافقةُ طلب التعديل بعد الاستلام، حظرُ الحساب من باب الطلب) كان يُردّ.
// **ولا يخرج شيءٌ من السياق**: لا توكنَ ولا جلسة — والتخويلُ وحدَه المُخطّى كبقيّة الجسر.
func (s *Server) qaAdminAllCaps(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.qaStagingEnabled() {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		all := authz.All()
		caps := make([]string, 0, len(all))
		for _, c := range all {
			caps = append(caps, string(c))
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxCaps, caps)))
	})
}

// qaOrderTransition **انتقالُ المكتب الحقيقيّ على طلب QA** (الاسترجاعُ بعد التسليم) — **سطرٌ يُنادي
// `handleOrderTransition` عينَه**. **ولماذا الغلاف**: حارسُ `ADG-1` يقرأ أوّلَ ذكرٍ للمعالِج في
// `server.go` ويطلب «حارسَ قدرة» قبله — **ومسارُ الإدارة هو المحروسُ المقصود**، فلا يُزاحمه هذا.
func (s *Server) qaOrderTransition(w http.ResponseWriter, r *http.Request) {
	s.handleOrderTransition(w, r)
}

// qaWhatsAppGuard **طلبُ واتساب لرقم QA ومتجر QA وحدَهما** — ثمّ المعالِجُ الحقيقيّ.
//
// **ولا رسالةَ لغير رقم QA**: المستلِمُ رقمُ QA، والمتجرُ متجرُ QA بلا رقم، **والموقعُ إلزاميٌّ**
// فلا يُطلب الموقعُ بالواتساب من أحد. **ولا متجرَ مكتوبٌ باليد** (رقمُه قد يكون حقيقيّاً).
func (s *Server) qaWhatsAppGuard(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.qaStagingEnabled() {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<16))
		if err != nil {
			s.respondErr(w, errValidation)
			return
		}
		var body struct {
			StoreID           string   `json:"store_id"`
			StoreName         string   `json:"store_name"`
			StorePhone        string   `json:"store_phone"`
			RecipientPhone    string   `json:"recipient_phone"`
			Lat               *float64 `json:"lat"`
			Lng               *float64 `json:"lng"`
			LocationRequestID string   `json:"location_request_id"`
		}
		if err := json.Unmarshal(raw, &body); err != nil {
			s.respondErr(w, errValidation)
			return
		}
		phone, _ := identity.NormalizePhone(body.RecipientPhone)
		if !qaQAPhoneSet()[phone] {
			s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_phone_not_allowed", "errors.forbidden"))
			return
		}
		var storeName string
		if isUUID(body.StoreID) {
			_ = s.pg.QueryRow(r.Context(), `SELECT name FROM merchants WHERE id = $1::uuid`, body.StoreID).Scan(&storeName)
		}
		if !strings.HasPrefix(storeName, "QA") || body.StoreName != "" || body.StorePhone != "" ||
			body.Lat == nil || body.Lng == nil || body.LocationRequestID != "" {
			s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_whatsapp_not_qa", "errors.forbidden"))
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(raw))
		r.ContentLength = int64(len(raw))
		next(w, r)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **متجرُ QA للنموذج ٢** — `kind=m2_store_setup`
// ══════════════════════════════════════════════════════════════════════

// qaM2StoreSetup **يضمن متجرَ QA للنموذج ٢ وصنفَه** — ويعيد استعمالَه إن وُجد.
//
// **مالكُه مالكُ متجرِ QA الثابت** (`+963900555020`)، عمولتُه ١٠٪ بلا مندوب، **وسعرُ صرفه ١٠٠**
// (لربطه بطلب مشترياتٍ بالدولار)، **ويقبل الاسترداد** (لشاهد «رجعت البضاعة للمتجر»)، **ومفتوحٌ ٢٤ ساعة** (لا صفَّ دوام ⇒ مفتوحٌ دائماً، ولا إغلاقَ
// طارئ). **وموقعُه عنوانُ زبون QA** — فهو داخلَ التغطية. **عتادُ اختبارٍ لا مالٌ**: لا قيدَ ولا رصيد.
func (s *Server) qaM2StoreSetup(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ownerID, err := s.qaFixedUser(ctx, qaMerchantOwnerPhone, "merchant", qaMerchantOwnerName, clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	lat, lng := qaFallbackLat, qaFallbackLng
	_ = s.pg.QueryRow(ctx, `
		SELECT ST_Y(a.location::geometry), ST_X(a.location::geometry)
		FROM addresses a JOIN users u ON u.id = a.user_id
		WHERE u.phone = $1
		ORDER BY a.is_default DESC, a.created_at DESC
		LIMIT 1`, qaStagingPhone).Scan(&lat, &lng)

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var mid string
	reused := true
	err = tx.QueryRow(ctx, `SELECT id::text FROM merchants WHERE name = $1`, qaM2StoreName).Scan(&mid)
	if errors.Is(err, pgx.ErrNoRows) {
		reused = false
		err = tx.QueryRow(ctx, `
			INSERT INTO merchants (name, owner_user_id, category_id, commission_percent, status, location)
			VALUES ($1, $2::uuid, (SELECT id FROM categories ORDER BY id LIMIT 1), 10, 'active',
			        ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)
			RETURNING id::text`, qaM2StoreName, ownerID, lng, lat).Scan(&mid)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **الحالُ نفسُها في كلّ نداء** — فشهادةٌ سابقةٌ بدّلت شيئاً لا تُفسد اللاحقة.
	if _, err := tx.Exec(ctx, `
		UPDATE merchants SET owner_user_id = $2::uuid, status = 'active', emergency_closed = false,
		       commission_percent = 10, sales_rep_user_id = NULL, exchange_rate = 100, accepts_returns = true,
		       location = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography
		WHERE id = $1::uuid`, mid, ownerID, lng, lat); err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, mid); err != nil {
		s.respondErr(w, err)
		return
	}
	var section string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM platform_sections WHERE active ORDER BY sort_order LIMIT 1`).
		Scan(&section); errors.Is(err, pgx.ErrNoRows) {
		if err := tx.QueryRow(ctx, `INSERT INTO platform_sections (name, active) VALUES ('QA قسم النموذج ٢', true)
			RETURNING id::text`).Scan(&section); err != nil {
			s.respondErr(w, err)
			return
		}
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	var itemID string
	err = tx.QueryRow(ctx, `SELECT id::text FROM menu_items WHERE merchant_id = $1::uuid AND name = $2`,
		mid, qaM2ItemName).Scan(&itemID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			INSERT INTO menu_items (merchant_id, platform_section_id, name, merchant_price, price, available, approved)
			VALUES ($1::uuid, $2::uuid, $3, 10000, 11000, true, true) RETURNING id::text`,
			mid, section, qaM2ItemName).Scan(&itemID)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := tx.Exec(ctx, `
		UPDATE menu_items SET merchant_price = 10000, price = 11000, price_usd_cents = NULL,
		       available = true, approved = true
		WHERE id = $1::uuid`, itemID); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	driverID, err := s.qaFixedUser(ctx, qaStagingDriverPhone, "driver", "سائق الاختبار QA", clientIP(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA model-2 store set up (staging-only)", "merchant", mid, "item", itemID, "reused", reused)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"merchant_id": mid, "item_id": itemID, "owner_id": ownerID, "driver_id": driverID,
		"merchant_price": 10000, "price": 11000, "commission_percent": 10,
		"exchange_rate": 100, "open_24h": true, "lat": lat, "lng": lng, "reused": reused,
	})
}

// ══════════════════════════════════════════════════════════════════════
// **سُلّمُ الطلب العاديّ بالنموذج ٢** — تكملةُ `order_advance`
// ══════════════════════════════════════════════════════════════════════

// qaStdLadder **سُلّمُ الطلب العاديّ وطلب الواتساب** حتّى «سُلّم».
var qaStdLadder = []string{
	orders.StPending, orders.StAccepted, orders.StDispatching, orders.StAssigned,
	orders.StAtPickup, orders.StPickedUp, orders.StOnTheWay, orders.StAtDropoff, orders.StDelivered,
}

func qaStdIndex(status string) int {
	if status == orders.StPreparing { // التحضيرُ بين القبول والطابور
		return 1
	}
	for i, st := range qaStdLadder {
		if st == status {
			return i
		}
	}
	return -1
}

// qaOrderAdvanceM2 **يسوق طلبَ QA عاديّاً (أو واتساب) بالنموذج ٢ إلى الهدف** — خطوةً خطوة.
//
// **كلُّ خطوةٍ بفاعلها ودوره عبر خدمة الطلبات الحقيقيّة**:
//
//	pending → accepted        مالكُ المتجر بدور merchant (وبلا مالكٍ: المكتب)
//	accepted → dispatching    المكتب (ops)
//	dispatching → assigned    `AssignDriver` لسائق QA الثابت (حارسُ سقف النقد B9 نفسُه)
//	assigned … delivered      سائقُ QA بدور driver — **و«استلمت» تشغّل الدفعَ للمتجر الحقيقيّ**
//
// **ويُعاد قراءةُ الحال بعد كلّ خطوة** — فالقبولُ قد يُنزل الطلبَ الطابورَ تلقائيّاً. **ولا تسليمَ
// وبلاغُ البابِ أو طلبُ التعديل بانتظار المكتب** (حارسا المعالِج نفسُهما). **والإثباتُ وكودُ
// التسليم حارسا معالِجٍ للجهاز يُخطّيان هنا** — كما يأذن المكتبُ (`proof-exception`).
func (s *Server) qaOrderAdvanceM2(w http.ResponseWriter, r *http.Request, uid, orderID, target string) {
	ctx := r.Context()
	toIdx := qaStdIndex(target)
	if toIdx <= 0 || target == orders.StPreparing {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_bad_target", "errors.validation"))
		return
	}
	read := func() (status string, driverID string, err error) {
		var drv *string
		err = s.pg.QueryRow(ctx, `SELECT status, driver_id::text FROM orders WHERE id = $1::uuid`, orderID).
			Scan(&status, &drv)
		if drv != nil {
			driverID = *drv
		}
		return
	}
	was, _, err := read()
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if from := qaStdIndex(was); from < 0 || from >= toIdx {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_backward", "errors.conflict"))
		return
	}
	qaDriver, ok := s.qaDriverIdentity(w, r)
	if !ok {
		return
	}
	steps := []string{}
	for guard := 0; guard < 12; guard++ {
		status, driverID, err := read()
		if err != nil {
			s.respondErr(w, err)
			return
		}
		at := qaStdIndex(status)
		if at < 0 {
			s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_order_off_ladder", "errors.conflict"))
			return
		}
		if at >= toIdx {
			break
		}
		var terr error
		switch status {
		case orders.StPending:
			actor, roles := s.qaMerchantActor(ctx, orderID, uid)
			_, terr = s.orders.Transition(ctx, actor, roles, orderID, orders.StAccepted, "QA model2 accept")
		case orders.StAccepted, orders.StPreparing:
			actor, aerr := s.qaNonCustomerAuthor(ctx, uid)
			if aerr != nil {
				s.respondErr(w, aerr)
				return
			}
			_, terr = s.orders.Transition(ctx, actor, []string{"ops"}, orderID, orders.StDispatching, "QA model2 dispatch")
		case orders.StDispatching:
			// **والإسنادُ لسائقٍ في دوامه** — وخارجَه يردّ المحرّكُ خطأً لا رمزَ له (٥٠٠)، فيُقال
			// صريحاً: شغّل دوامَ سائق QA أوّلاً (`kind=driver_shift`).
			var onShift bool
			_ = s.pg.QueryRow(ctx, `SELECT on_shift FROM users WHERE id = $1::uuid`, qaDriver).Scan(&onShift)
			if !onShift {
				s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_driver_off_shift", "errors.conflict"))
				return
			}
			_, terr = s.orders.AssignDriver(ctx, qaDriver, []string{"ops"}, orderID, qaDriver, "QA model2 assign")
		default:
			// **ما بعد الإسناد فعلُ سائق QA وحدَه** — وطلبٌ بيد غيره لا يُساق.
			if driverID != qaDriver {
				s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_not_qa_driver", "errors.conflict"))
				return
			}
			next := qaStdLadder[at+1]
			if (next == orders.StAtPickup || next == orders.StPickedUp) && s.orders.StopsOf(ctx, orderID) > 1 {
				s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_multi_stop_unsupported", "errors.conflict"))
				return
			}
			if next == orders.StDelivered {
				if v := s.orders.DoorViewOf(ctx, orderID); v != nil && v.ReportCode != "" && v.ReportAt != nil &&
					(v.InstructionAt == nil || v.ReportAt.After(*v.InstructionAt)) {
					s.respondErr(w, errDoorReportPending)
					return
				}
				if pending, perr := s.orders.HasPendingEditRequest(ctx, orderID); perr != nil {
					s.respondErr(w, perr)
					return
				} else if pending {
					s.respondErr(w, orders.ErrEditRequestPendingDeliver)
					return
				}
			}
			_, terr = s.orders.Transition(ctx, qaDriver, []string{"driver"}, orderID, next, "QA model2 advance")
		}
		if terr != nil {
			s.respondErr(w, terr)
			return
		}
		steps = append(steps, status)
	}
	final, _, _ := read()
	s.logger.Warn("QA model-2 order advanced (staging-only)", "order", orderID, "from", was, "to", final)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"order_id": orderID, "status": final, "was": was, "driver_id": qaDriver, "money_model": 2, "passed": steps,
	})
}

// qaMerchantActor **فاعلُ القبول** — مالكُ المتجر بدور merchant كما يقبل من تطبيقه **حين تدير
// المتاجرُ طلباتها** (`platform.orders_mode=merchants`). **وفي «المنصّة تدير» يُسقط المحرّكُ دورَ
// المتجر** (`rolesUnderMode`) — فالقبولُ قبولُ المكتب كما يقع فعلاً، وكذا بلا مالكٍ أو بلا متجر.
func (s *Server) qaMerchantActor(ctx context.Context, orderID, uid string) (string, []string) {
	if !s.orders.MerchantsSelfManage(ctx) {
		actor, err := s.qaNonCustomerAuthor(ctx, uid)
		if err != nil {
			return uid, []string{"ops"}
		}
		return actor, []string{"ops"}
	}
	var owner *string
	_ = s.pg.QueryRow(ctx, `
		SELECT m.owner_user_id::text FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1::uuid`, orderID).Scan(&owner)
	if owner != nil && *owner != "" {
		return *owner, []string{"merchant"}
	}
	actor, err := s.qaNonCustomerAuthor(ctx, uid)
	if err != nil {
		return uid, []string{"ops"}
	}
	return actor, []string{"ops"}
}

// ══════════════════════════════════════════════════════════════════════
// **قراءةُ نتيجة المال لطلب QA** — `GET /qa/money?order_id=` (قراءةٌ محضة)
// ══════════════════════════════════════════════════════════════════════

// qaM2Families **عوائلُ الثوابت التي يُختم بها السيناريو** — كاختبار `money_model2_test.go`.
var qaM2Families = []string{"FI-03", "FI-04", "FI-05", "FI-06", "FI-09", "FI-10", "FI-13", "FI-14", "FI-15", "FI-17"}

// qaGoodsKinds **قيودُ البضاعة** — النموذج ٢ لا يكتب واحداً منها أبداً.
var qaGoodsKinds = map[string]bool{
	"merchant_earning": true, "merchant_cash_accrued": true, "merchant_cash_paid": true, "custom_goods_return": true,
}

// handleQAMoneyReadback **كلُّ ما يلزم سكربتاً ليحكم على قواعد النموذج ٢** — لطلب QA وحدَه.
//
//	order        أعمدةُ المال على الطلب (النموذج، الطريقة، النقد، المحفظة، الأجرة، العمولة، العالق…)
//	items        بنودُ الطلب (لطلب التعديل)
//	ledger       قيودُ الدفتر بمرجع الطلب مع صاحبها (الخزينة/الزبون/السائق/المتجر/غيره)
//	settlements  صفوفُ تسوية المتجر (off_ledger، الحال، من دفع) وتصحيحاتُها
//	driver_box   سطورُ صندوق السائق للطلب، ومحتجَزُه وسقفُه وانكشافُه ومالُ المنصّة في الطلب
//	office_cash  سطورُ الدرج للطلب (السلفة/الردّ)
//	obligations  الديونُ الناشئةُ عن الطلب
//	invariants   خروقُ ثوابت المال التي تذكر هذا الطلب (لا خروقُ غيره)
//
// **ولا كتابةَ ولا توكنَ ولا بيانةَ زبونٍ خامّة** (لا رقمَ ولا اسم).
func (s *Server) handleQAMoneyReadback(w http.ResponseWriter, r *http.Request) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	oid := strings.TrimSpace(r.URL.Query().Get("order_id"))
	if !isUUID(oid) {
		s.respondErr(w, errValidation)
		return
	}
	if !s.qaIsQAOrder(ctx, oid) {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_qa_order", "errors.forbidden"))
		return
	}

	// ── الطلب ──
	var o struct {
		Number                                  int64
		Kind, Origin, Status, Payment, Scope    string
		Model                                   int16
		Subtotal, DeliveryFee, DriverFee, Total int64
		CashDue, WalletPaid, Commission, Waived int64
		Stuck                                   int64
		StuckDecision                           string
		CustomGoods, CustomFee, CustomGoodsUSD  *int64
		GoodsUSDSYP, QuoteVersion               int64
		ConfirmedVersion                        *int64
		CustomerID, DriverID, MerchantID        string
		CustomMode                              string
	}
	if err := s.pg.QueryRow(ctx, `
		SELECT number, kind, COALESCE(origin, ''), status, payment_method, COALESCE(wallet_scope, ''),
		       money_model, subtotal, delivery_fee, driver_fee, total, cash_due, wallet_paid,
		       platform_commission, promo_delivery_waived,
		       COALESCE((to_jsonb(o)->>'stuck_goods_amount')::bigint, 0),
		       COALESCE(to_jsonb(o)->>'stuck_goods_decision', ''),
		       custom_goods_amount, custom_fee, custom_goods_usd, goods_usd_syp, quote_version,
		       quote_confirmed_version,
		       COALESCE(customer_id::text, ''), COALESCE(driver_id::text, ''), COALESCE(merchant_id::text, ''),
		       COALESCE(custom_mode, '')
		FROM orders o WHERE id = $1::uuid`, oid).Scan(
		&o.Number, &o.Kind, &o.Origin, &o.Status, &o.Payment, &o.Scope,
		&o.Model, &o.Subtotal, &o.DeliveryFee, &o.DriverFee, &o.Total, &o.CashDue, &o.WalletPaid,
		&o.Commission, &o.Waived, &o.Stuck, &o.StuckDecision,
		&o.CustomGoods, &o.CustomFee, &o.CustomGoodsUSD, &o.GoodsUSDSYP, &o.QuoteVersion, &o.ConfirmedVersion,
		&o.CustomerID, &o.DriverID, &o.MerchantID, &o.CustomMode); err != nil {
		s.respondErr(w, err)
		return
	}
	quoteTotal := int64(0)
	if o.CustomGoods != nil {
		quoteTotal += *o.CustomGoods
	}
	if o.CustomFee != nil {
		quoteTotal += *o.CustomFee
	}
	quoteTotal -= o.Waived

	// ── البنود ──
	items := []map[string]any{}
	if rows, err := s.pg.Query(ctx, `
		SELECT id::text, COALESCE(menu_item_id::text, ''), qty, COALESCE(merchant_price, 0), unit_price
		FROM order_items WHERE order_id = $1::uuid ORDER BY id`, oid); err == nil {
		for rows.Next() {
			var id, menu string
			var qty int
			var mp, up int64
			if rows.Scan(&id, &menu, &qty, &mp, &up) == nil {
				items = append(items, map[string]any{"id": id, "menu_item_id": menu, "qty": qty,
					"merchant_price": mp, "unit_price": up})
			}
		}
		rows.Close()
	}

	// ── الدفتر ──
	treasuryID := s.orders.TreasuryID(ctx)
	ledger := []map[string]any{}
	byKind := map[string]int64{}
	byParty := map[string]int64{}
	goodsRows := 0
	var treasuryEffect, customerNet, driverNet, profit int64
	rows, err := s.pg.Query(ctx, `
		SELECT wt.user_id::text, wt.kind, wt.amount,
		       EXISTS (SELECT 1 FROM merchants m WHERE m.owner_user_id = wt.user_id)
		FROM wallet_transactions wt WHERE wt.ref = $1 ORDER BY wt.id`, oid)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var user, kind string
		var amount int64
		var isMerchant bool
		if err := rows.Scan(&user, &kind, &amount, &isMerchant); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		party := "other"
		switch {
		case user == treasuryID && treasuryID != "":
			party = "treasury"
			treasuryEffect += amount
		case user == o.CustomerID && o.CustomerID != "":
			party = "customer"
			customerNet += amount
		case user == o.DriverID && o.DriverID != "":
			party = "driver"
			driverNet += amount
		case isMerchant:
			party = "merchant"
		}
		if kind == "platform_profit" {
			profit += amount
		}
		if qaGoodsKinds[kind] {
			goodsRows++
		}
		byKind[kind] += amount
		byParty[party] += amount
		ledger = append(ledger, map[string]any{"party": party, "kind": kind, "amount": amount})
	}
	rows.Close()

	// ── تسوية المتجر ──
	settlements := []map[string]any{}
	var settledNet int64
	if rows, err := s.pg.Query(ctx, `
		SELECT ms.merchant_id::text, ms.amount, ms.adjusted_amount, ms.reversed_amount, ms.state,
		       COALESCE(ms.paid_via, ''), COALESCE((to_jsonb(ms)->>'off_ledger')::bool, false),
		       ms.accrued_tx_id IS NOT NULL OR ms.paid_tx_id IS NOT NULL OR ms.earning_tx_id IS NOT NULL
		FROM merchant_settlements ms WHERE ms.order_id = $1::uuid ORDER BY ms.created_at`, oid); err == nil {
		for rows.Next() {
			var mid, state, via string
			var amount, adjusted, reversed int64
			var off, withTx bool
			if rows.Scan(&mid, &amount, &adjusted, &reversed, &state, &via, &off, &withTx) == nil {
				settledNet += amount + adjusted - reversed
				settlements = append(settlements, map[string]any{"merchant_id": mid, "amount": amount,
					"adjusted_amount": adjusted, "reversed_amount": reversed, "state": state, "paid_via": via,
					"off_ledger": off, "has_ledger_tx": withTx})
			}
		}
		rows.Close()
	}
	var adjRows, adjWithTx int64
	_ = s.pg.QueryRow(ctx, `
		SELECT count(*), count(*) FILTER (WHERE accrued_tx_id IS NOT NULL OR paid_tx_id IS NOT NULL
		                                    OR earning_tx_id IS NOT NULL)
		FROM merchant_settlement_adjustments WHERE order_id = $1::uuid`, oid).Scan(&adjRows, &adjWithTx)

	// ── صندوقُ السائق والدرج ──
	box := map[string]int64{}
	if rows, err := s.pg.Query(ctx, `SELECT kind, amount FROM driver_cash_entries WHERE ref = $1`, oid); err == nil {
		for rows.Next() {
			var k string
			var a int64
			if rows.Scan(&k, &a) == nil {
				box[k] += a
			}
		}
		rows.Close()
	}
	office := map[string]int64{}
	if rows, err := s.pg.Query(ctx, `SELECT source, direction, amount FROM office_cash_entries WHERE ref = $1`, oid); err == nil {
		for rows.Next() {
			var src, dir string
			var a int64
			if rows.Scan(&src, &dir, &a) == nil {
				if dir == "out" {
					a = -a
				}
				office[src] += a
			}
		}
		rows.Close()
	}
	platformCash, _ := cashbox.PlatformCashOf(ctx, s.pg, oid)
	driver := map[string]any{"limit": s.cashbox.Limit(ctx)}
	if o.DriverID != "" {
		held, _ := s.cashbox.Held(ctx, o.DriverID)
		exposure, _ := s.cashbox.Exposure(ctx, s.pg, o.DriverID)
		var inflight int64
		_ = s.pg.QueryRow(ctx, `SELECT `+cashbox.InflightSQL("$1::uuid")+`::bigint`, o.DriverID).Scan(&inflight)
		driver["id"] = o.DriverID
		driver["held"] = held
		driver["exposure"] = exposure
		driver["inflight_platform_cash"] = inflight
	}

	// ── الديون ──
	obligations := []map[string]any{}
	if rows, err := s.pg.Query(ctx, `
		SELECT party_kind, cause, amount, settled FROM financial_obligations WHERE order_id = $1::uuid`, oid); err == nil {
		for rows.Next() {
			var pk, cause string
			var a, st int64
			if rows.Scan(&pk, &cause, &a, &st) == nil {
				obligations = append(obligations, map[string]any{"party_kind": pk, "cause": cause, "amount": a, "settled": st})
			}
		}
		rows.Close()
	}

	// ── الثوابت: خروقٌ تذكر هذا الطلب وحدَه ──
	checked := 0
	violations := []map[string]any{}
	for _, c := range fininv.Select(qaM2Families...) {
		if !c.Ops {
			continue
		}
		checked++
		vs, err := fininv.Run(ctx, s.pg, c.ID)
		if err != nil {
			violations = append(violations, map[string]any{"id": c.ID, "error": err.Error()})
			continue
		}
		for _, v := range vs {
			hits := 0
			for _, row := range v.Rows {
				if qaRowMentions(row, oid) {
					hits++
				}
			}
			if hits > 0 {
				violations = append(violations, map[string]any{"id": v.Check.ID, "name": v.Check.Name, "rows": hits})
			}
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"order": map[string]any{
			"id": oid, "number": o.Number, "kind": o.Kind, "origin": o.Origin, "status": o.Status,
			"money_model": o.Model, "payment_method": o.Payment, "wallet_scope": o.Scope,
			"subtotal": o.Subtotal, "delivery_fee": o.DeliveryFee, "driver_fee": o.DriverFee, "total": o.Total,
			"cash_due": o.CashDue, "wallet_paid": o.WalletPaid, "platform_commission": o.Commission,
			"promo_delivery_waived": o.Waived, "fee_paid": o.DeliveryFee - o.Waived,
			"stuck_goods_amount": o.Stuck, "stuck_goods_decision": o.StuckDecision,
			"custom_mode": o.CustomMode, "custom_goods_amount": o.CustomGoods, "custom_fee": o.CustomFee,
			"custom_goods_usd": o.CustomGoodsUSD, "goods_usd_syp": o.GoodsUSDSYP,
			"quote_version": o.QuoteVersion, "quote_total": quoteTotal, "quote_confirmed_version": o.ConfirmedVersion,
			"has_driver": o.DriverID != "", "merchant_id": o.MerchantID,
		},
		"items": items,
		"ledger": map[string]any{
			"rows": ledger, "by_kind": byKind, "by_party": byParty, "goods_rows": goodsRows,
			"platform_profit": profit, "treasury_effect": treasuryEffect,
			"customer_net": customerNet, "driver_net": driverNet,
		},
		"settlements": map[string]any{
			"rows": settlements, "net_paid_by_driver": settledNet,
			"adjustments": adjRows, "adjustments_with_ledger_tx": adjWithTx,
		},
		"driver_box":     map[string]any{"by_kind": box, "collection": box["order_collection"]},
		"office_cash":    office,
		"platform_cash":  platformCash,
		"driver":         driver,
		"obligations":    obligations,
		"invariants":     map[string]any{"families": qaM2Families, "checked": checked, "violations": violations},
		"setting_model2": s.settings.GetBool(ctx, qaMoneyModel2Key),
	})
}

// qaRowMentions **صفُّ خرقٍ يذكر الطلب؟** — بمعرّفه نصّاً أو `uuid` خامّاً.
func qaRowMentions(row []any, oid string) bool {
	for _, v := range row {
		if qaCellString(v) == oid {
			return true
		}
	}
	return false
}

func qaCellString(v any) string {
	switch x := v.(type) {
	case [16]byte:
		return uuid.UUID(x).String()
	case string:
		return x
	case fmt.Stringer:
		return x.String()
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}
