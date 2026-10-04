package server

// ══════════════════════════════════════════════════════════════════════
// **غرفةُ الطوارئ** — قراراتُ المالك ٢٠٢٦-١٠-٠٤
// ══════════════════════════════════════════════════════════════════════
//
// # صندوقٌ واحدٌ بنوعٍ مخزَّن (`0280`)
//
// **كلُّ طارئٍ صفٌّ في `driver_emergencies`**: حادث · عطل · ظرفٌ قاهر (قبل الاستلام
// وبعده) · إغلاقٌ طارئٌ لمتجر · توقّفُ المنصّة. **كانت الغرفةُ ترى طارئَ ما بعد
// الاستلام وحدَه** — والحادثُ قبل الاستلام يحرّر الطلبَ ويمضي بلا أثرٍ فيها.
//
// # وخطواتُ الحلّ
//
//	استلمتها  ←  السائقُ بخير؟  ←  مصيرُ الطلب  ←  المال  ←  تمّ
//
// **والمالُ طلبُ تعويضٍ إلى الماليّة لا دفع** — صفٌّ في `driver_compensation_requests`
// تقضي فيه الماليّةُ من بابها القائم (`failure_aftermath.go`). **ولا قيدَ من هنا أبداً.**
// **وسجلُّ ملاحظاتٍ يُضاف إليه ولا يُعدَّل** — كان الحلُّ جملةً واحدة.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// أنواعُ الطارئ — **مخزَّنةٌ لا مستنتَجة.**
const (
	EmergencyAccident     = "accident"
	EmergencyBreakdown    = "breakdown"
	EmergencyForceMajeure = "force_majeure"
	EmergencyStoreClosure = "store_closure"
	EmergencyPlatformHalt = "platform_halt"
)

// مصائرُ الطلب في الغرفة.
const (
	OutcomeRedispatch  = "redispatch"
	OutcomeReturnStore = "return_store"
	OutcomeCancel      = "cancel"
	OutcomeContinue    = "continue"
)

// emergencyStaleSetting **طارئٌ بلا مستلِمٍ هذه المدّة يحمرّ** (افتراضُه ١٠ دقائق).
const emergencyStaleSetting = "ops.emergency_unacked_red_min"

// emergencyKindOf **نوعُ الطارئ من رمزٍ قاله التطبيق** — وفارغٌ لما لا يُعرف.
func emergencyKindOf(code string) string {
	switch strings.TrimSpace(code) {
	case "accident":
		return EmergencyAccident
	case "bike_broken", "breakdown":
		return EmergencyBreakdown
	case "force_majeure":
		return EmergencyForceMajeure
	}
	return ""
}

// نصوصُ ما يصل الزبون — **كان طلبُه يرجع خطوةً بلا كلمة.**
const (
	emergencyCustomerTitle      = "تحديث على طلبك"
	emergencyCustomerRedispatch = "طرأ ظرفٌ على السائق، ونرسل لك سائقاً آخر الآن"
	emergencyCustomerFollowing  = "طرأ ظرفٌ على السائق، والإدارة تتابع طلبك الآن وتخبرك بما يجري"
	emergencyCustomerEnded      = "طرأ ظرفٌ على السائق فتعذّر إكمال طلبك وأُلغي — نعتذر منك"
	emergencyMoneyTitle         = "طلب تعويض من غرفة الطوارئ ينتظر الماليّة"
)

var (
	errEmergencyStepsPending = httpx.NewError(http.StatusConflict,
		"emergency_steps_pending", "errors.emergency_steps_pending")
	errEmergencyStepDone = httpx.NewError(http.StatusConflict,
		"emergency_step_done", "errors.emergency_step_done")
	errEmergencyNoOrder = httpx.NewError(http.StatusConflict,
		"emergency_no_order", "errors.emergency_no_order")
	errEmergencyNoDriver = httpx.NewError(http.StatusConflict,
		"emergency_no_driver", "errors.emergency_no_driver")
	errEmergencyBadOutcome = httpx.NewError(http.StatusBadRequest,
		"emergency_bad_outcome", "errors.emergency_bad_outcome")
	errEmergencyOutcomeStage = httpx.NewError(http.StatusConflict,
		"emergency_outcome_stage", "errors.emergency_outcome_stage")
)

// notifyCustomerEmergency **يُخبَر الزبون** — بما صار إليه طلبُه.
func (s *Server) notifyCustomerEmergency(ctx context.Context, orderID, body string) {
	var customer *string
	if err := s.pg.QueryRow(ctx,
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil ||
		customer == nil || *customer == "" {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: *customer, Kind: notifications.KindOrder,
		Title: emergencyCustomerTitle, Body: body,
		Entity: "order", EntityID: orderID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	})
}

// recordReleaseEmergency **«لدي مشكلة» قبل الاستلام تدخل الغرفة** — بنوعها وموضعِ السائق
// الأخير. **ولا تمنع التركَ إن تعثّرت**: الطلبُ حُرّر، والتنبيهُ يمضي.
func (s *Server) recordReleaseEmergency(ctx context.Context, driverID, orderID, kind, note string) string {
	var id string
	err := s.pg.QueryRow(ctx, `
		INSERT INTO driver_emergencies (driver_id, order_id, at, note, kind, stage)
		SELECT $1, $2, u.last_location, $3, $4, 'before_pickup'
		  FROM users u WHERE u.id = $1
		ON CONFLICT DO NOTHING
		RETURNING id::text`, driverID, orderID, clip(note, 500), kind).Scan(&id)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.Error("الطارئ: تعذّر تسجيلُ التركِ في الغرفة", "order", orderID, "error", err)
	}
	s.touch("emergency", "ops")
	return id
}

// recordStoreClosureEmergency **إغلاقٌ طارئٌ لمتجرٍ يدخل الغرفة** — مرّةً لكلّ إغلاق.
func (s *Server) recordStoreClosureEmergency(ctx context.Context, merchantID string) {
	if _, err := s.pg.Exec(ctx, `
		INSERT INTO driver_emergencies (kind, merchant_id)
		VALUES ('store_closure', $1)
		ON CONFLICT DO NOTHING`, merchantID); err != nil {
		s.logger.Error("الطارئ: تعذّر تسجيلُ إغلاق المتجر", "merchant", merchantID, "error", err)
	}
}

// recordPlatformHalt **توقّفُ المنصّة يدخل الغرفة — ويُغلق بعودتها.**
//
// **حالٌ لا واقعة**: لا سائقَ ولا طلب، **ولا يبقى مفتوحاً بعد أن عادت المنصّة**.
func (s *Server) recordPlatformHalt(ctx context.Context, active bool, message, actorID string) {
	var err error
	if active {
		_, err = s.pg.Exec(ctx, `
			INSERT INTO driver_emergencies (kind, note) VALUES ('platform_halt', $1)
			ON CONFLICT DO NOTHING`, clip(message, 500))
	} else {
		_, err = s.pg.Exec(ctx, `
			UPDATE driver_emergencies
			   SET status = 'resolved', resolved_at = now(), resolved_by = NULLIF($1, '')::uuid,
			       resolution = 'عادت المنصّة للعمل'
			 WHERE kind = 'platform_halt' AND status = 'open'`, actorID)
	}
	if err != nil {
		s.logger.Error("الطارئ: تعذّر تسجيلُ توقّف المنصّة", "error", err)
	}
	s.touch("emergency", "ops")
}

// emergencyRow **ما يلزم الخطواتِ من البلاغ** — يُقرأ مقفولاً في معاملتها.
type emergencyRow struct {
	ID, Kind, Status, Outcome string
	DriverID, OrderID         *string
	MerchantID                *string
	Acked, DriverOK, Money    bool
}

func loadEmergency(ctx context.Context, q dbtx.Querier, id string, lock bool) (*emergencyRow, error) {
	if !isUUID(id) {
		return nil, httpx.ErrNotFound
	}
	sql := `
		SELECT id::text, kind, status, outcome, driver_id::text, order_id::text, merchant_id::text,
		       acknowledged_at IS NOT NULL, driver_ok_at IS NOT NULL,
		       money_request_id IS NOT NULL OR money_skipped
		  FROM driver_emergencies WHERE id = $1`
	if lock {
		sql += ` FOR UPDATE`
	}
	var e emergencyRow
	err := q.QueryRow(ctx, sql, id).Scan(&e.ID, &e.Kind, &e.Status, &e.Outcome,
		&e.DriverID, &e.OrderID, &e.MerchantID, &e.Acked, &e.DriverOK, &e.Money)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, httpx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

// driverKind **طارئُ سائق** — له خطوةُ «السائقُ بخير» ومصيرُ الطلب والمال.
func (e *emergencyRow) driverKind() bool { return e.DriverID != nil }

// missingStep **أوّلُ خطوةٍ لم تُقطع قبل «تمّ»** — وفارغٌ إن قُطعت كلُّها.
func (e *emergencyRow) missingStep() string {
	if !e.Acked {
		return "ack"
	}
	if !e.driverKind() {
		return ""
	}
	if !e.DriverOK {
		return "driver_ok"
	}
	if e.OrderID != nil && e.Outcome == "" {
		return "outcome"
	}
	if e.OrderID != nil && !e.Money {
		return "money"
	}
	return ""
}

// ── العدّاد ──────────────────────────────────────────────────────────

// handleEmergencyCount **عدّادُ القائمة** — المفتوحُ، وما بلا مستلِمٍ أطولَ من المهلة.
func (s *Server) handleEmergencyCount(w http.ResponseWriter, r *http.Request) {
	staleMin := s.settings.GetNum(r.Context(), emergencyStaleSetting, 10)
	var open, unacked, stale int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*),
		       count(*) FILTER (WHERE acknowledged_at IS NULL),
		       count(*) FILTER (WHERE acknowledged_at IS NULL
		                          AND created_at < now() - make_interval(mins => $1::int))
		  FROM driver_emergencies WHERE status = 'open'`, staleMin).
		Scan(&open, &unacked, &stale); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"open": open, "unacked": unacked, "stale": stale, "stale_after_min": staleMin,
	})
}

// ── طبقةُ الخريطة ────────────────────────────────────────────────────

// handleEmergencyMap **الطوارئُ المفتوحةُ نقاطاً** — لخريطة العمليّات.
//
// **موضعُ السائق الحيُّ أوّلاً** ثمّ موضعُ الضغطة، **والمتجرُ بموقعه.** وما لا موضعَ له
// لا يُرسَم — ولا يُخترَع له دبّوس.
func (s *Server) handleEmergencyMap(w http.ResponseWriter, r *http.Request) {
	staleMin := s.settings.GetNum(r.Context(), emergencyStaleSetting, 10)
	rows, err := s.pg.Query(r.Context(), `
		SELECT e.id::text, e.kind,
		       ST_Y(COALESCE(u.last_location, e.at, m.location)::geometry),
		       ST_X(COALESCE(u.last_location, e.at, m.location)::geometry),
		       COALESCE(NULLIF(u.full_name, ''), u.phone::text, m.name, ''),
		       o.number,
		       e.acknowledged_at IS NULL
		         AND e.created_at < now() - make_interval(mins => $1::int)
		  FROM driver_emergencies e
		  LEFT JOIN users u ON u.id = e.driver_id
		  LEFT JOIN merchants m ON m.id = e.merchant_id
		  LEFT JOIN orders o ON o.id = e.order_id
		 WHERE e.status = 'open'
		   AND COALESCE(u.last_location, e.at, m.location) IS NOT NULL
		 ORDER BY e.created_at`, staleMin)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type pin struct {
		ID          string  `json:"id"`
		Kind        string  `json:"kind"`
		Lat         float64 `json:"lat"`
		Lng         float64 `json:"lng"`
		Label       string  `json:"label"`
		OrderNumber *int64  `json:"order_number"`
		Stale       bool    `json:"stale"`
	}
	out := []pin{}
	for rows.Next() {
		var p pin
		if err := rows.Scan(&p.ID, &p.Kind, &p.Lat, &p.Lng, &p.Label, &p.OrderNumber, &p.Stale); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, p)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"emergencies": out, "count": len(out)})
}

// ── صفحةُ الطارئ ─────────────────────────────────────────────────────

type emergencyPerson struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type emergencyDriver struct {
	emergencyPerson
	Lat            *float64   `json:"lat"`
	Lng            *float64   `json:"lng"`
	LocationAt     *time.Time `json:"location_at"`
	OnShift        bool       `json:"on_shift"`
	AccidentLocked bool       `json:"accident_locked"`
	CashHeld       int64      `json:"cash_held"`
}

type emergencyOrder struct {
	ID            string           `json:"id"`
	Number        int64            `json:"number"`
	Status        string           `json:"status"`
	PickedUp      bool             `json:"picked_up"`
	PickedUpAt    *time.Time       `json:"picked_up_at"`
	Total         int64            `json:"total"`
	CashDue       int64            `json:"cash_due"`
	PaymentMethod string           `json:"payment_method"`
	Customer      *emergencyPerson `json:"customer"`
	Store         *emergencyPerson `json:"store"`
	NewDriver     *emergencyPerson `json:"new_driver"`
	AddressText   string           `json:"address_text"`
}

type emergencyOtherOrder struct {
	ID       string `json:"id"`
	Number   int64  `json:"number"`
	Status   string `json:"status"`
	PickedUp bool   `json:"picked_up"`
	Customer string `json:"customer"`
}

type emergencyNote struct {
	ID        int64     `json:"id"`
	Author    string    `json:"author"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type emergencyMoney struct {
	ID              string `json:"id"`
	Status          string `json:"status"`
	SuggestedAmount int64  `json:"suggested_amount"`
	Amount          *int64 `json:"amount"`
}

type emergencyDetail struct {
	ID             string                `json:"id"`
	Kind           string                `json:"kind"`
	Stage          string                `json:"stage"`
	Status         string                `json:"status"`
	Note           string                `json:"note"`
	Lat            *float64              `json:"lat"`
	Lng            *float64              `json:"lng"`
	CreatedAt      time.Time             `json:"created_at"`
	Stale          bool                  `json:"stale"`
	AcknowledgedAt *time.Time            `json:"acknowledged_at"`
	AcknowledgedBy string                `json:"acknowledged_by"`
	DriverOKAt     *time.Time            `json:"driver_ok_at"`
	DriverOKBy     string                `json:"driver_ok_by"`
	Outcome        string                `json:"outcome"`
	OutcomeAt      *time.Time            `json:"outcome_at"`
	OutcomeBy      string                `json:"outcome_by"`
	MoneySkipped   bool                  `json:"money_skipped"`
	MoneyAt        *time.Time            `json:"money_at"`
	MoneyRequest   *emergencyMoney       `json:"money_request"`
	Resolution     string                `json:"resolution"`
	ResolvedAt     *time.Time            `json:"resolved_at"`
	ResolvedBy     string                `json:"resolved_by"`
	NextStep       string                `json:"next_step"`
	Driver         *emergencyDriver      `json:"driver"`
	Order          *emergencyOrder       `json:"order"`
	OtherOrders    []emergencyOtherOrder `json:"other_orders"`
	Merchant       *struct {
		emergencyPerson
		EmergencyClosed bool `json:"emergency_closed"`
	} `json:"merchant"`
	StoreOrders []emergencyOtherOrder `json:"store_orders"`
	Notes       []emergencyNote       `json:"notes"`
}

// nameSQL **اسمُ الشخص أو رقمُه** — لعمودٍ من جدول المستخدمين.
func nameSQL(alias string) string {
	return "COALESCE(NULLIF(" + alias + ".full_name, ''), " + alias + ".phone::text, '')"
}

// handleEmergencyDetail **صفحةُ الطارئ**: الطلبُ وحالُه والسائقُ الجديد · البضاعةُ والمالُ
// مع السائق وطلباتُه الأخرى · أرقامُ الاتّصال · موضعُ السائق الحيّ · وسجلُّ الملاحظات.
func (s *Server) handleEmergencyDetail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	e, err := loadEmergency(ctx, s.pg, chi.URLParam(r, "id"), false)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	staleMin := s.settings.GetNum(ctx, emergencyStaleSetting, 10)
	d := emergencyDetail{OtherOrders: []emergencyOtherOrder{}, StoreOrders: []emergencyOtherOrder{},
		Notes: []emergencyNote{}}
	var moneyID *string
	if err := s.pg.QueryRow(ctx, `
		SELECT e.id::text, e.kind, e.stage, e.status, e.note,
		       ST_Y(e.at::geometry), ST_X(e.at::geometry), e.created_at,
		       e.status = 'open' AND e.acknowledged_at IS NULL
		         AND e.created_at < now() - make_interval(mins => $2::int),
		       e.acknowledged_at, `+nameSQL("ab")+`,
		       e.driver_ok_at, `+nameSQL("ob")+`,
		       e.outcome, e.outcome_at, `+nameSQL("cb")+`,
		       e.money_skipped, e.money_at, e.money_request_id::text,
		       e.resolution, e.resolved_at, `+nameSQL("rb")+`
		  FROM driver_emergencies e
		  LEFT JOIN users ab ON ab.id = e.acknowledged_by
		  LEFT JOIN users ob ON ob.id = e.driver_ok_by
		  LEFT JOIN users cb ON cb.id = e.outcome_by
		  LEFT JOIN users rb ON rb.id = e.resolved_by
		 WHERE e.id = $1`, e.ID, staleMin).Scan(
		&d.ID, &d.Kind, &d.Stage, &d.Status, &d.Note, &d.Lat, &d.Lng, &d.CreatedAt, &d.Stale,
		&d.AcknowledgedAt, &d.AcknowledgedBy, &d.DriverOKAt, &d.DriverOKBy,
		&d.Outcome, &d.OutcomeAt, &d.OutcomeBy, &d.MoneySkipped, &d.MoneyAt, &moneyID,
		&d.Resolution, &d.ResolvedAt, &d.ResolvedBy); err != nil {
		s.respondErr(w, err)
		return
	}
	if d.Status == "open" {
		d.NextStep = e.missingStep()
		if d.NextStep == "" {
			d.NextStep = "resolve"
		}
	}
	if moneyID != nil {
		var m emergencyMoney
		if err := s.pg.QueryRow(ctx, `
			SELECT id::text, status, suggested_amount, amount
			  FROM driver_compensation_requests WHERE id = $1`, *moneyID).
			Scan(&m.ID, &m.Status, &m.SuggestedAmount, &m.Amount); err == nil {
			d.MoneyRequest = &m
		}
	}

	if e.DriverID != nil {
		var dr emergencyDriver
		if err := s.pg.QueryRow(ctx, `
			SELECT u.id::text, `+nameSQL("u")+`, COALESCE(u.phone::text, ''),
			       ST_Y(u.last_location::geometry), ST_X(u.last_location::geometry), u.last_location_at,
			       u.on_shift,
			       u.accident_lock_at IS NOT NULL
			         AND (u.accident_cleared_at IS NULL OR u.accident_cleared_at < u.accident_lock_at),
			       COALESCE((SELECT b.held FROM driver_cash_boxes b WHERE b.driver_id = u.id), 0)
			  FROM users u WHERE u.id = $1`, *e.DriverID).Scan(
			&dr.ID, &dr.Name, &dr.Phone, &dr.Lat, &dr.Lng, &dr.LocationAt, &dr.OnShift,
			&dr.AccidentLocked, &dr.CashHeld); err == nil {
			d.Driver = &dr
		}
		others, err := s.emergencyOrderList(ctx, `
			SELECT o.id::text, o.number, o.status, o.picked_up_at IS NOT NULL, `+nameSQL("c")+`
			  FROM orders o LEFT JOIN users c ON c.id = o.customer_id
			 WHERE o.driver_id = $1 AND o.closed_at IS NULL
			   AND ($2::uuid IS NULL OR o.id <> $2::uuid)
			 ORDER BY o.created_at`, *e.DriverID, e.OrderID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		d.OtherOrders = others
	}

	if e.OrderID != nil {
		o, err := s.emergencyOrderOf(ctx, *e.OrderID, e.DriverID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		d.Order = o
	}

	if e.MerchantID != nil {
		mm := &struct {
			emergencyPerson
			EmergencyClosed bool `json:"emergency_closed"`
		}{}
		if err := s.pg.QueryRow(ctx, `
			SELECT m.id::text, m.name, COALESCE(NULLIF(m.phone::text, ''), ow.phone::text, ''),
			       m.emergency_closed
			  FROM merchants m LEFT JOIN users ow ON ow.id = m.owner_user_id
			 WHERE m.id = $1`, *e.MerchantID).
			Scan(&mm.ID, &mm.Name, &mm.Phone, &mm.EmergencyClosed); err == nil {
			d.Merchant = mm
		}
		// **وطلباتُه المفتوحةُ التي لم تخرج بضاعتُها** — يحوّلها المكتبُ أو يلغيها.
		list, err := s.emergencyOrderList(ctx, `
			SELECT o.id::text, o.number, o.status, false, `+nameSQL("c")+`
			  FROM orders o LEFT JOIN users c ON c.id = o.customer_id
			 WHERE o.merchant_id = $1 AND o.closed_at IS NULL AND o.picked_up_at IS NULL
			   AND o.status <> ALL($2)
			 ORDER BY o.created_at`, *e.MerchantID, orders.TerminalStatuses())
		if err != nil {
			s.respondErr(w, err)
			return
		}
		d.StoreOrders = list
	}

	rows, err := s.pg.Query(ctx, `
		SELECT n.id, `+nameSQL("u")+`, n.body, n.created_at
		  FROM emergency_notes n LEFT JOIN users u ON u.id = n.author_id
		 WHERE n.emergency_id = $1 ORDER BY n.id`, e.ID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	for rows.Next() {
		var n emergencyNote
		if err := rows.Scan(&n.ID, &n.Author, &n.Body, &n.CreatedAt); err != nil {
			rows.Close()
			s.respondErr(w, err)
			return
		}
		d.Notes = append(d.Notes, n)
	}
	rows.Close()
	httpx.JSON(w, http.StatusOK, d)
}

func (s *Server) emergencyOrderList(ctx context.Context, sql string, args ...any) ([]emergencyOtherOrder, error) {
	rows, err := s.pg.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []emergencyOtherOrder{}
	for rows.Next() {
		var o emergencyOtherOrder
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &o.PickedUp, &o.Customer); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// emergencyOrderOf **الطلبُ وأطرافُه** — والسائقُ الجديدُ من لم يكن صاحبَ الطارئ.
func (s *Server) emergencyOrderOf(ctx context.Context, orderID string, oldDriver *string) (*emergencyOrder, error) {
	var o emergencyOrder
	var custID, custName, custPhone, storeID, storeName, storePhone, drvID, drvName, drvPhone *string
	err := s.pg.QueryRow(ctx, `
		SELECT o.id::text, o.number, o.status, o.picked_up_at IS NOT NULL, o.picked_up_at,
		       o.total, o.cash_due, o.payment_method, COALESCE(o.address_text, ''),
		       c.id::text, `+nameSQL("c")+`, c.phone::text,
		       m.id::text, m.name, COALESCE(NULLIF(m.phone::text, ''), ow.phone::text),
		       d.id::text, `+nameSQL("d")+`, d.phone::text
		  FROM orders o
		  LEFT JOIN users c ON c.id = o.customer_id
		  LEFT JOIN merchants m ON m.id = o.merchant_id
		  LEFT JOIN users ow ON ow.id = m.owner_user_id
		  LEFT JOIN users d ON d.id = o.driver_id
		 WHERE o.id = $1`, orderID).Scan(
		&o.ID, &o.Number, &o.Status, &o.PickedUp, &o.PickedUpAt,
		&o.Total, &o.CashDue, &o.PaymentMethod, &o.AddressText,
		&custID, &custName, &custPhone, &storeID, &storeName, &storePhone,
		&drvID, &drvName, &drvPhone)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	person := func(id, name, phone *string) *emergencyPerson {
		if id == nil {
			return nil
		}
		p := &emergencyPerson{ID: *id}
		if name != nil {
			p.Name = *name
		}
		if phone != nil {
			p.Phone = *phone
		}
		return p
	}
	o.Customer = person(custID, custName, custPhone)
	o.Store = person(storeID, storeName, storePhone)
	if drvID != nil && (oldDriver == nil || *drvID != *oldDriver) {
		o.NewDriver = person(drvID, drvName, drvPhone)
	}
	return &o, nil
}

// ── الخطوات ──────────────────────────────────────────────────────────

// handleEmergencyDriverOK **«السائقُ بخير»** — ويرفع قفلَ ما بعد الحادث إن كان.
//
// **وهو البابُ نفسُه في قسم الحسابات** (`clearAccidentLockTx`) — لا نسخةٌ ثانية.
func (s *Server) handleEmergencyDriverOK(w http.ResponseWriter, r *http.Request) {
	req, _ := decode[struct {
		Note string `json:"note"`
	}](r)
	note := ""
	if req != nil {
		note = clip(strings.TrimSpace(req.Note), 500)
	}
	actor := userIDFrom(r)
	var driverID string
	cleared := false
	err := s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		e, err := loadEmergency(ctx, q, chi.URLParam(r, "id"), true)
		if err != nil {
			return err
		}
		if e.Status != "open" {
			return httpx.ErrNotFound
		}
		if !e.driverKind() {
			return errEmergencyNoDriver
		}
		if e.DriverOK {
			return errEmergencyStepDone
		}
		driverID = *e.DriverID
		if cleared, err = s.clearAccidentLockTx(ctx, q, driverID, actor); err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE driver_emergencies SET driver_ok_at = now(), driver_ok_by = $2,
			       acknowledged_at = COALESCE(acknowledged_at, now()),
			       acknowledged_by = COALESCE(acknowledged_by, $2)
			 WHERE id = $1`, e.ID, actor); err != nil {
			return err
		}
		if err := addEmergencyNote(ctx, q, e.ID, actor, note); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "ops.emergency_driver_ok", "emergency", e.ID,
			map[string]any{"driver_id": driverID, "lock_cleared": cleared, "note": note})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if cleared {
		s.notify.Notify(r.Context(), notifications.Input{
			UserID: driverID, Kind: notifications.KindAccount, Title: notifTitles.driverCleared,
			Entity: "user", EntityID: driverID,
		})
		s.touchUser(driverID, "account")
	}
	s.touch("emergency", "ops")
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"driver_ok": true, "lock_cleared": cleared})
}

// addEmergencyNote **سطرٌ في سجلّ الملاحظات** — والفارغُ لا يُكتب.
func addEmergencyNote(ctx context.Context, q dbtx.Querier, id, actor, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	_, err := q.Exec(ctx, `
		INSERT INTO emergency_notes (emergency_id, author_id, body)
		VALUES ($1, NULLIF($2, '')::uuid, $3)`, id, actor, clip(body, 1000))
	return err
}

// handleEmergencyNote **ملاحظةٌ تُضاف إلى السجلّ** — ولا تُعدَّل بعدها.
func (s *Server) handleEmergencyNote(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Body string `json:"body"`
	}](r)
	if err != nil || strings.TrimSpace(req.Body) == "" {
		s.respondErr(w, errValidation)
		return
	}
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		e, err := loadEmergency(ctx, q, chi.URLParam(r, "id"), false)
		if err != nil {
			return err
		}
		if err := addEmergencyNote(ctx, q, e.ID, userIDFrom(r), req.Body); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "ops.emergency_note", "emergency", e.ID,
			map[string]any{"body": clip(strings.TrimSpace(req.Body), 300)})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("emergency", "ops")
	httpx.JSON(w, http.StatusCreated, map[string]any{"added": true})
}

// handleEmergencyOutcome **مصيرُ الطلب — تقرّره الإدارةُ كلَّ مرّة.**
//
//	redispatch    يأخذه سائقٌ آخر — بعد الاستلام من موضع الأوّل (`pickup_override`)
//	return_store  تعود البضاعةُ للمتجر — يُنهى الطلبُ بذنب المنصّة ومشوارِ إرجاع
//	cancel        يُلغى — قبل الاستلام إلغاءٌ، وبعده إنهاءٌ ومشوارٌ إلى المكتب
//	continue      لا شيءَ يلزم — الطلبُ يمضي كما هو (أو انتهى أصلاً)
//
// **ولا يُرسَل آليّاً بعد الاستلام** — والقرارُ هنا يُكتب مرّةً باسم من قرّر.
func (s *Server) handleEmergencyOutcome(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Outcome string `json:"outcome"`
		Note    string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	ctx := r.Context()
	actor := userIDFrom(r)
	note := clip(strings.TrimSpace(req.Note), 500)
	switch req.Outcome {
	case OutcomeRedispatch, OutcomeReturnStore, OutcomeCancel, OutcomeContinue:
	default:
		s.respondErr(w, errEmergencyBadOutcome)
		return
	}
	if req.Outcome != OutcomeContinue && note == "" {
		s.respondErr(w, errReasonRequired)
		return
	}
	e, err := loadEmergency(ctx, s.pg, chi.URLParam(r, "id"), false)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if e.Status != "open" {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if e.OrderID == nil {
		s.respondErr(w, errEmergencyNoOrder)
		return
	}
	if e.Outcome != "" {
		s.respondErr(w, errEmergencyStepDone)
		return
	}
	orderID := *e.OrderID
	var status string
	if err := s.pg.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, orderID).
		Scan(&status); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	after := orders.AfterPickup(status)
	open := !isTerminalStatus(status)
	ops := []string{"ops"}
	customerText := ""

	switch req.Outcome {
	case OutcomeRedispatch:
		switch {
		case after:
			// **والبضاعةُ مع الأوّل** — نقطةُ الاستلام البديلة كُتبت عند الطارئ.
			if _, err := s.orders.Transition(ctx, actor, ops, orderID, orders.StDispatching,
				"طارئٌ لدى السائق — "+note); err != nil {
				s.respondErr(w, err)
				return
			}
		case !open:
			s.respondErr(w, errEmergencyOutcomeStage)
			return
		}
		// **وقبل الاستلام حُرّر سلفاً** — يُكتب القرارُ ويُخبَر الزبون.
		customerText = emergencyCustomerRedispatch
	case OutcomeReturnStore, OutcomeCancel:
		if !open {
			s.respondErr(w, errEmergencyOutcomeStage)
			return
		}
		if orders.OfficeDecides(status) {
			to := orders.ReturnToOffice
			if req.Outcome == OutcomeReturnStore {
				to = orders.ReturnToStore
			}
			if _, err := s.orders.ResolveDoor(ctx, actor, ops, orderID, orders.DoorResolution{
				Action: orders.DoorReturnToOffice, Fault: orders.FaultPlatform,
				Note: note, ReturnTo: to,
			}, nil); err != nil {
				s.respondErr(w, err)
				return
			}
		} else if _, err := s.orders.Transition(ctx, actor, ops, orderID, orders.StCancelled,
			"طارئٌ لدى السائق — "+note); err != nil {
			s.respondErr(w, err)
			return
		}
		customerText = emergencyCustomerEnded
	}

	err = s.inTx(ctx, func(ctx context.Context, q dbtx.Querier) error {
		tag, err := q.Exec(ctx, `
			UPDATE driver_emergencies SET outcome = $2, outcome_at = now(), outcome_by = $3,
			       acknowledged_at = COALESCE(acknowledged_at, now()),
			       acknowledged_by = COALESCE(acknowledged_by, $3)
			 WHERE id = $1 AND outcome = ''`, e.ID, req.Outcome, actor)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errEmergencyStepDone
		}
		if err := addEmergencyNote(ctx, q, e.ID, actor, note); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "ops.emergency_outcome", "emergency", e.ID, map[string]any{
			"order_id": orderID, "outcome": req.Outcome, "status_was": status, "note": note,
		})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if customerText != "" {
		s.notifyCustomerEmergency(ctx, orderID, customerText)
	}
	s.touch("order", "ops")
	s.touch("emergency", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"outcome": req.Outcome})
}

// isTerminalStatus **انتهى الطلب** — من قائمة المحرّك نفسِها.
func isTerminalStatus(status string) bool {
	for _, t := range orders.TerminalStatuses() {
		if t == status {
			return true
		}
	}
	return false
}

// handleEmergencyMoney **خطوةُ المال — طلبُ تعويضٍ إلى الماليّة، أو «لا مال».**
//
// **ولا يُدفع شيءٌ من هنا**: يُكتب صفٌّ معلَّقٌ في `driver_compensation_requests`
// (ذنبُ المنصّة · السببُ `emergency_<النوع>` · المبلغُ المقترَح) **وتقضي فيه الماليّةُ**
// من بابها (`/orders/{id}/compensate-driver` أو رفضُه). **وطلبٌ قائمٌ للطلب والسائق
// نفسِهما يُربط ولا يُكرَّر.**
func (s *Server) handleEmergencyMoney(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Amount int64  `json:"amount"`
		Note   string `json:"note"`
		Skip   bool   `json:"skip"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	note := clip(strings.TrimSpace(req.Note), 500)
	if !req.Skip && (req.Amount <= 0 || note == "") {
		s.respondErr(w, errValidation)
		return
	}
	actor := userIDFrom(r)
	var requestID, orderID string
	var number int64
	err = s.inTx(r.Context(), func(ctx context.Context, q dbtx.Querier) error {
		e, err := loadEmergency(ctx, q, chi.URLParam(r, "id"), true)
		if err != nil {
			return err
		}
		if e.Status != "open" {
			return httpx.ErrNotFound
		}
		if e.Money {
			return errEmergencyStepDone
		}
		if req.Skip {
			if _, err := q.Exec(ctx, `
				UPDATE driver_emergencies SET money_skipped = true, money_at = now(), money_by = $2
				 WHERE id = $1`, e.ID, actor); err != nil {
				return err
			}
			if err := addEmergencyNote(ctx, q, e.ID, actor, note); err != nil {
				return err
			}
			return s.auditTx(ctx, q, r, "ops.emergency_money", "emergency", e.ID,
				map[string]any{"skipped": true, "note": note})
		}
		if e.DriverID == nil {
			return errEmergencyNoDriver
		}
		if e.OrderID == nil {
			return errEmergencyNoOrder
		}
		orderID = *e.OrderID
		err = q.QueryRow(ctx, `
			INSERT INTO driver_compensation_requests
			    (order_id, driver_id, fault, fail_reason, suggested_amount)
			VALUES ($1, $2, 'platform', $3, $4)
			ON CONFLICT (order_id, driver_id) DO NOTHING
			RETURNING id::text`, orderID, *e.DriverID, "emergency_"+e.Kind, req.Amount).
			Scan(&requestID)
		if errors.Is(err, pgx.ErrNoRows) {
			err = q.QueryRow(ctx, `
				SELECT id::text FROM driver_compensation_requests
				 WHERE order_id = $1 AND driver_id = $2`, orderID, *e.DriverID).Scan(&requestID)
		}
		if err != nil {
			return err
		}
		if _, err := q.Exec(ctx, `
			UPDATE driver_emergencies SET money_request_id = $2, money_at = now(), money_by = $3
			 WHERE id = $1`, e.ID, requestID, actor); err != nil {
			return err
		}
		_ = q.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number)
		if err := addEmergencyNote(ctx, q, e.ID, actor, note); err != nil {
			return err
		}
		return s.auditTx(ctx, q, r, "ops.emergency_money", "emergency", e.ID, map[string]any{
			"order_id": orderID, "request_id": requestID, "amount": req.Amount, "note": note,
		})
	})
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if requestID != "" {
		// **والماليّةُ تُنبَّه** — طلبٌ ينتظر قرارها في صفحة التعويضات.
		s.notify.NotifyCaps(r.Context(), []string{"finance.manage"}, notifications.Input{
			Kind: notifications.KindOrder, Title: emergencyMoneyTitle,
			Body:   "#" + strconv.FormatInt(number, 10) + " — " + strconv.FormatInt(req.Amount, 10),
			Entity: "order", EntityID: orderID, Href: "/dashboard/compensations",
		})
		s.touch("wallet", "ops")
	}
	s.touch("emergency", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"request_id": requestID, "skipped": req.Skip})
}
