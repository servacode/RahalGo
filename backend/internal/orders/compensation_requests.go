package orders

// ══════════════════════════════════════════════════════════════════════
// **تعويضُ السائق بعد موافقة العمليات — لا لحظةَ الضغطة** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// `compensateDriverOnFail` كانت تقيّد نصفَ أجر السائق **في معاملة الفشل نفسِها**
// — بلا يد. **وقِيس على التجهيز**: الخادمُ يقبل سبباً لا يخصّ المرحلة، **ويدفع
// ٥٬٠٠٠ فوراً في كلّ مرّة، حتّى لـ«تأخّرتُ أنا».** فسائقٌ يكسب بضغطة.
//
// # فقرّر المالك: «بعد موافقة العمليات»
//
// **وما بقي من القرار القديم**: السائقُ يُعوَّض إن لم يكن الذنبُ ذنبَه (زبونٌ أو
// متجر)، **والمطالبةُ على المتجر تُفتح كما هي.** والذي تغيّر **متى** يُقبض: حين
// يوافق إنسانٌ لا لحظةَ الضغطة.
//
// # ثمّ صار طابوراً واحداً لكلّ تعويض (قرارُ المالك ٢٠٢٦-١٠-٠٤)
//
// **صفحةُ «التعويضات» الواحدة وطريقُ موافقةٍ واحد**: تعويضُ السائق، **وتعويضُ
// المتجر عن بضاعةٍ رُدّت** (الماليّةُ تكتب المبلغ بسقف سعر الشراء)، **وتعويضُ
// الشكوى** (الدعمُ يقترح لصاحب الشكوى). كلُّها صفوفٌ هنا، **والماليّةُ توافق
// بكلمة السرّ، ولا يوافق أحدٌ على ما اقترحه، والمالُ يخرج من الخزينة دائماً**
// — فلا تعويضَ يخلق مالاً من عدم.
//
// **و`driver_id` صار «المستفيد»** — والاسمُ باقٍ لأنّ أقساماً أخرى تكتب فيه.

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// حالاتُ طلب التعويض.
const (
	CompensationPending  = "pending"
	CompensationApproved = "approved"
	CompensationRejected = "rejected"
)

// أنواعُ التعويض — **ثلاثةٌ في طابورٍ واحد.**
const (
	CompKindDriver        = "driver"
	CompKindMerchantGoods = "merchant_goods"
	CompKindComplaint     = "complaint"
)

// أسبابٌ يكتبها النظامُ لغير السائق.
const (
	ReasonGoodsReturned = "goods_returned"
	ReasonComplaint     = "complaint"
)

var (
	// ErrCompensationDuplicate **اقتراحٌ قائمٌ لهذا الطلب أو الشكوى.**
	ErrCompensationDuplicate = errors.New("يوجد طلبُ تعويضٍ قائمٌ لهذا")
	// ErrCompensationInvalid **اقتراحٌ ناقص** — نوعٌ أو مستفيدٌ أو مرجعٌ أو مبلغ.
	ErrCompensationInvalid = errors.New("طلبُ التعويض ناقص")
	// ErrCompensationNotPending **قُضي فيه سلفاً.**
	ErrCompensationNotPending = errors.New("طلبُ التعويض ليس معلَّقاً")
	// ErrAlreadyCompensated **المستفيدُ عُوِّض عن هذا الطلب سلفاً.**
	ErrAlreadyCompensated = errors.New("عُوِّض المستفيدُ عن هذا سلفاً")
)

// CompensationRequest طلبُ تعويضٍ ينتظر — أو قُضي فيه.
type CompensationRequest struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	OrderID     string `json:"order_id"`
	OrderNumber int64  `json:"order_number"`
	OrderStatus string `json:"order_status"`
	TicketID    string `json:"ticket_id"`
	// TicketNumber رقمُ الشكوى — صفرٌ لغير الشكوى.
	TicketNumber int64 `json:"ticket_number"`
	// DriverID **المستفيد** — السائق، أو صاحبُ المتجر، أو صاحبُ الشكوى.
	DriverID   string `json:"driver_id"`
	DriverName string `json:"driver_name"`
	// BeneficiaryRole دورُ المستفيد: driver · merchant · customer · rep.
	BeneficiaryRole string `json:"beneficiary_role"`
	Fault           string `json:"fault"`
	FailReason      string `json:"fail_reason"`
	// ReasonLabel **السببُ بالعربيّة** — لا رمزَ آلةٍ في شاشةٍ ولا إشعار.
	ReasonLabel string `json:"reason_label"`
	// ExpectedFault **ذنبُ السبب كما في القائمة** — وفراغُه سببٌ لا ذنبَ له معروف.
	ExpectedFault string `json:"expected_fault"`
	// FaultMismatch **الذنبُ المكتوبُ لا يطابق السبب** — «المتجرُ مغلق» وذنبُه «الزبون».
	FaultMismatch   bool       `json:"fault_mismatch"`
	SuggestedAmount int64      `json:"suggested_amount"`
	Status          string     `json:"status"`
	Amount          *int64     `json:"amount"`
	Note            string     `json:"note"`
	ProposedBy      *string    `json:"proposed_by"`
	ProposedByName  string     `json:"proposed_by_name"`
	DecidedBy       *string    `json:"decided_by"`
	DecidedByName   string     `json:"decided_by_name"`
	DecidedAt       *time.Time `json:"decided_at"`
	DecisionNote    string     `json:"decision_note"`
	SelfApproved    bool       `json:"self_approved"`
	CreatedAt       time.Time  `json:"created_at"`
	// Overdue **معلَّقٌ تجاوز المهلة** (`compensations.overdue_hours`) — يحمرّ.
	Overdue bool `json:"overdue"`
	// Cap **سقفُ هذا النوع** — وفوقه مديرُ المنصّة وحدَه يوافق.
	Cap int64 `json:"cap"`
}

// reasonLabels **أسبابُ التعويض بالعربيّة** — نصُّها نصُّ `common.failReasons`
// في المعجم، وأسبابُ غرفة الطوارئ، وما يكتبه النظامُ لغير السائق.
var reasonLabels = map[string]string{
	"customer_absent":             "الزبون غير موجود",
	"customer_refused":            "الزبون رفض استلام الطلب",
	"customer_unreachable":        "الزبون لا يرد على الهاتف",
	"customer_cancelled_by_phone": "الزبون ألغى بالهاتف",
	"address_wrong":               "العنوان غير صحيح",
	"driver_late":                 "تأخر السائق فلم يعد الطلب صالحا",
	"merchant_closed":             "المتجر مغلق",
	"merchant_refused":            "المتجر رفض تسليم الطلب",
	"merchant_not_ready":          "المتجر لم يجهز الطلب",
	"order_unknown":               "المتجر لا يعرف هذا الطلب",
	"customer_no_answer":          "لا أحد يجيب عند الباب",
	"bike_broken":                 "تعطلت الدراجة",
	"accident":                    "حادث",
	"force_majeure":               "ظرف قاهر",
	"emergency_accident":          "طارئ: حادث",
	"emergency_breakdown":         "طارئ: عطل",
	"emergency_force_majeure":     "طارئ: ظرف قاهر",
	"emergency_store_closure":     "طارئ: إغلاق المتجر",
	"emergency_platform_halt":     "طارئ: توقف المنصة",
	ReasonGoodsReturned:           "بضاعة رجعت إلى المتجر",
	ReasonComplaint:               "تعويض شكوى",
}

// ReasonLabel **السببُ بالعربيّة** — والفارغُ «بلا سبب مكتوب»، والمجهولُ «سبب آخر».
// **ولا يُعرض رمزُ آلةٍ أبداً**، لا في الصفحة ولا في الإشعار.
func ReasonLabel(code string) string {
	if code == "" {
		return "بلا سبب مكتوب"
	}
	if l, ok := reasonLabels[code]; ok {
		return l
	}
	return "سبب آخر"
}

// ReasonFault **ذنبُ السبب كما في القائمة** — وفراغٌ لما لا ذنبَ له معروف.
func ReasonFault(code string) string {
	for _, list := range [][]FailReason{StageReports, ReleaseReasons} {
		for _, r := range list {
			if r.Code == code {
				return r.Fault
			}
		}
	}
	if strings.HasPrefix(code, "emergency_") {
		return FaultPlatform
	}
	return ""
}

// faultMismatch **الذنبُ لا يطابق السبب** — في تعويض السائق وحدَه.
func faultMismatch(kind, reason, fault string) (string, bool) {
	exp := ReasonFault(reason)
	if kind != CompKindDriver || exp == "" || fault == "" {
		return exp, false
	}
	return exp, exp != fault
}

// CompensationCapKey مفتاحُ سقف النوع في الإعدادات.
func CompensationCapKey(kind string) string {
	return "compensations.cap_" + kind
}

// CompensationCap **سقفُ النوع** — وفوقه مديرُ المنصّة وحدَه يوافق.
func (s *Service) CompensationCap(ctx context.Context, kind string) int64 {
	if s.settings == nil {
		return 0
	}
	return s.settings.GetInt(ctx, CompensationCapKey(kind))
}

// CompensationProposal **اقتراحُ تعويض** — لا يدفع شيئاً.
type CompensationProposal struct {
	Kind          string
	OrderID       string
	TicketID      string
	BeneficiaryID string
	Fault         string
	Reason        string
	Amount        int64
	Note          string
	// ProposedBy صاحبُ الاقتراح — وفراغُه المحرّك.
	ProposedBy string
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// ProposeCompensationTx **يكتب اقتراحَ تعويضٍ معلَّقاً** — البابُ الواحدُ لكلّ
// من يقترح: المحرّك، ومكتبُ الماليّة، وغرفةُ الطوارئ، والدعمُ من الشكوى.
//
// **ولا يقيّد مالاً**: المالُ يخرج عند الموافقة وحدَها.
func ProposeCompensationTx(ctx context.Context, q dbtx.Querier, p CompensationProposal) (string, error) {
	if p.BeneficiaryID == "" || p.Amount < 0 {
		return "", ErrCompensationInvalid
	}
	switch p.Kind {
	case CompKindDriver:
		if p.OrderID == "" || (p.Fault != FaultCustomer && p.Fault != FaultMerchant && p.Fault != FaultPlatform) {
			return "", ErrCompensationInvalid
		}
	case CompKindMerchantGoods:
		if p.OrderID == "" {
			return "", ErrCompensationInvalid
		}
		if p.Reason == "" {
			p.Reason = ReasonGoodsReturned
		}
	case CompKindComplaint:
		if p.TicketID == "" {
			return "", ErrCompensationInvalid
		}
		p.OrderID = ""
		if p.Reason == "" {
			p.Reason = ReasonComplaint
		}
	default:
		return "", ErrCompensationInvalid
	}
	var id string
	err := q.QueryRow(ctx, `
		INSERT INTO driver_compensation_requests
		    (kind, order_id, ticket_id, driver_id, fault, fail_reason,
		     suggested_amount, note, proposed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id::text`,
		p.Kind, nullable(p.OrderID), nullable(p.TicketID), p.BeneficiaryID, p.Fault,
		p.Reason, p.Amount, strings.TrimSpace(p.Note), nullable(p.ProposedBy)).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return "", ErrCompensationDuplicate
	}
	return id, err
}

// requestDriverCompensation **يكتب طلبَ تعويضٍ معلَّقاً — بدل القيد.**
//
// يُنادى من مسارين: الفشلُ عند باب الزبون (`settle` الخطوة ٤)، **وتعذّرُ
// المتجر** (`merchant_blocked.go`). **والذنبُ يُمرَّر لا يُقرأ**.
//
// ويردّ `true` إن كُتب طلبٌ جديد — **فيُنبَّه المكتبُ بعد التثبيت لا قبله.**
func (s *Service) requestDriverCompensation(ctx context.Context, q wallet.Querier,
	orderID string, driverID *string, fault, failReason string, driverFee int64) (bool, error) {
	if driverID == nil || s.settings == nil {
		return false, nil
	}
	// **ذنبُ السائق لا تعويضَ فيه، والمجهولُ لا يُنسب إلى أحد.** وذنبُ المنصّة
	// يُعوَّض كذنب الزبون والمتجر (مساءَ ٢٠٢٦-١٠-٠٢).
	if fault != FaultCustomer && fault != FaultMerchant && fault != FaultPlatform {
		return false, nil
	}
	// **وما عُوِّض لا يُطلَب له ثانيةً** — قيدٌ يدويٌّ سبق للطلب نفسِه.
	var already bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM wallet_transactions
		              WHERE ref = $1 AND kind = 'compensation' AND user_id = $2)`,
		orderID, *driverID).Scan(&already); err != nil {
		return false, err
	}
	if already {
		return false, nil
	}
	// ══════════════════════════════════════════════════════════════════
	// **والنسبةُ صفرٌ لا تُسكت الطلب** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٥)
	// ══════════════════════════════════════════════════════════════════
	//
	// كانت «صفر» تعني «لا يُكتب شيء» — **فيضيع مشوارُ سائقٍ لم يره أحد**،
	// والصفحةُ فارغةٌ كأنّه لم يقع. **والآن يُكتب بمقترَحٍ صفر والماليّةُ تقرّر.**
	pct := s.settings.GetInt(ctx, "drivers.failed_compensation_percent")
	if pct < 0 {
		pct = 0
	}
	suggested := int64(0)
	if driverFee > 0 {
		suggested = driverFee * pct / 100
	}
	tag, err := q.Exec(ctx, `
		INSERT INTO driver_compensation_requests
		    (order_id, driver_id, fault, fail_reason, suggested_amount)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (order_id, driver_id) DO NOTHING`,
		orderID, *driverID, fault, failReason, suggested)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

// alertCompensationPending **تنبيهُ المكتب بعد التثبيت** — طلبُ تعويضٍ ينتظره.
//
// **والسببُ بالعربيّة** — كان الإشعارُ يقول «#1349 — customer_absent».
func (s *Service) alertCompensationPending(ctx context.Context, orderID string) {
	if s.notify == nil {
		return
	}
	var number, suggested int64
	var reason string
	if err := s.db.QueryRow(ctx, `
		SELECT o.number, r.suggested_amount, r.fail_reason
		FROM driver_compensation_requests r JOIN orders o ON o.id = r.order_id
		WHERE r.order_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC LIMIT 1`, orderID).
		Scan(&number, &suggested, &reason); err != nil {
		return
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind:     notifications.KindOrder,
		Title:    "تعويضُ سائقٍ ينتظر موافقتك",
		Body:     CompensationAlertBody(number, reason, suggested),
		Entity:   "order",
		EntityID: orderID,
		Href:     "/dashboard/compensations",
	})
}

// CompensationAlertBody **نصُّ إشعار التعويض** — بالسبب العربيّ لا برمزه.
func CompensationAlertBody(number int64, reason string, suggested int64) string {
	return "#" + itoa(number) + " — " + ReasonLabel(reason) + " — المقترَح " + itoa(suggested)
}

const compensationCols = `
	SELECT r.id::text, r.kind, COALESCE(r.order_id::text, tk.order_id::text, ''),
	       COALESCE(o.number, 0), COALESCE(o.status, ''),
	       COALESCE(r.ticket_id::text, ''), COALESCE(tk.number, 0),
	       r.driver_id::text,
	       CASE WHEN r.kind = 'merchant_goods' THEN COALESCE(NULLIF(m.name, ''), u.full_name, '')
	            ELSE COALESCE(NULLIF(u.full_name, ''), u.phone::text, '') END,
	       CASE r.kind WHEN 'driver' THEN 'driver' WHEN 'merchant_goods' THEN 'merchant'
	            ELSE COALESCE((SELECT ur.role_code FROM user_roles ur
	                 WHERE ur.user_id = r.driver_id
	                   AND ur.role_code IN ('driver', 'merchant', 'rep', 'customer')
	                 ORDER BY array_position(ARRAY['driver','merchant','rep','customer'], ur.role_code)
	                 LIMIT 1), 'customer') END,
	       r.fault, r.fail_reason, r.suggested_amount, r.status, r.amount,
	       r.note, r.proposed_by::text, COALESCE(pu.full_name, ''),
	       r.decided_by::text, COALESCE(du.full_name, ''),
	       r.decided_at, r.decision_note, r.self_approved, r.created_at
	FROM driver_compensation_requests r
	JOIN users u ON u.id = r.driver_id
	LEFT JOIN tickets tk ON tk.id = r.ticket_id
	LEFT JOIN orders o ON o.id = COALESCE(r.order_id, tk.order_id)
	LEFT JOIN merchants m ON m.id = o.merchant_id
	LEFT JOIN users pu ON pu.id = r.proposed_by
	LEFT JOIN users du ON du.id = r.decided_by`

func scanCompensation(row pgx.Row) (*CompensationRequest, error) {
	var c CompensationRequest
	if err := row.Scan(&c.ID, &c.Kind, &c.OrderID, &c.OrderNumber, &c.OrderStatus,
		&c.TicketID, &c.TicketNumber, &c.DriverID, &c.DriverName, &c.BeneficiaryRole,
		&c.Fault, &c.FailReason, &c.SuggestedAmount, &c.Status, &c.Amount,
		&c.Note, &c.ProposedBy, &c.ProposedByName, &c.DecidedBy, &c.DecidedByName,
		&c.DecidedAt, &c.DecisionNote, &c.SelfApproved, &c.CreatedAt); err != nil {
		return nil, err
	}
	c.ReasonLabel = ReasonLabel(c.FailReason)
	c.ExpectedFault, c.FaultMismatch = faultMismatch(c.Kind, c.FailReason, c.Fault)
	return &c, nil
}

// CompensationByIDTx **طلبُ التعويض بمعرّفه — مقفولاً.**
//
// **والموافقةُ بمعرّف الطلب نفسِه لا برقم الطلب** (قرارُ المالك ٢٠٢٦-١٠-٠٤):
// كان الزرُّ يرسل رقمَ الطلب، **فطلبٌ تعذّر مع سائقَين يدفع للأقدم** والنافذةُ
// تعرض اسمَ الثاني.
func (s *Service) CompensationByIDTx(ctx context.Context, q wallet.Querier, id string) (*CompensationRequest, error) {
	if _, err := q.Exec(ctx,
		`SELECT 1 FROM driver_compensation_requests WHERE id = $1 FOR UPDATE`, id); err != nil {
		return nil, err
	}
	c, err := scanCompensation(q.QueryRow(ctx, compensationCols+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

// PendingCompensationTx **طلبُ التعويض المعلَّق لهذا الطلب — مقفولاً.**
//
// للباب القديم (`compensate-driver`) وحدَه: **وإن كان للطلب أكثرُ من معلَّقٍ
// ردّ `ErrCompensationAmbiguous`** — فلا يُدفع للأقدم بدل المقصود.
func (s *Service) PendingCompensationTx(ctx context.Context, q wallet.Querier,
	orderID string) (*CompensationRequest, error) {
	rows, err := q.Query(ctx, `
		SELECT id::text FROM driver_compensation_requests
		WHERE order_id = $1 AND status = 'pending' AND kind = 'driver'
		ORDER BY created_at ASC
		FOR UPDATE`, orderID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	switch len(ids) {
	case 0:
		return nil, nil
	case 1:
		return s.CompensationByIDTx(ctx, q, ids[0])
	}
	return nil, ErrCompensationAmbiguous
}

// ErrCompensationAmbiguous **للطلب أكثرُ من تعويضٍ معلَّق** — فليُسمَّ المقصود.
var ErrCompensationAmbiguous = errors.New("للطلب أكثرُ من تعويضٍ معلَّق")

// DecideCompensationTx **يُغلق الطلبَ المعلَّق** — موافقةً بمبلغٍ أو رفضاً.
func (s *Service) DecideCompensationTx(ctx context.Context, q wallet.Querier,
	id, status string, amount int64, actorID, note string) error {
	var amt any
	if status == CompensationApproved {
		amt = amount
	}
	_, err := q.Exec(ctx, `
		UPDATE driver_compensation_requests
		SET status = $2, amount = $3, decided_by = $4, decided_at = now(),
		    decision_note = $5
		WHERE id = $1 AND status = 'pending'`, id, status, amt, actorID, note)
	return err
}

// ApproveCompensationTx **يدفع التعويضَ المعتمَد** — مالٌ للمستفيد ومثلُه من
// الخزينة في المعاملة نفسِها، ثمّ يُغلق الطلب.
//
//	driver          +محفظة السائق «compensation» بمرجع الطلب · −الخزينة · ومطالبةُ المتجر إن كان الذنبُ ذنبَه
//	merchant_goods  +صاحب المتجر بسقف سعر الشراء · −الخزينة (`CompensateGoodsTx`)
//	complaint       +صاحب الشكوى بمرجع التذكرة · −الخزينة · وتعويضُ التذكرة = المدفوع
//
// **وملاحظةُ الموظّف لا تُكتب في سطر المحفظة** — السائقُ يرى «تسوية من الإدارة»
// ولا يرى ملاحظةَ المكتب (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٣).
func (s *Service) ApproveCompensationTx(ctx context.Context, q wallet.Querier,
	c *CompensationRequest, amount int64, actorID, note string, selfApproved bool) error {
	if c == nil || c.Status != CompensationPending {
		return ErrCompensationNotPending
	}
	if amount <= 0 {
		return ErrCompensationInvalid
	}
	switch c.Kind {
	case CompKindDriver:
		var already bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM wallet_transactions
			              WHERE ref = $1 AND kind = 'compensation' AND user_id = $2)`,
			c.OrderID, c.DriverID).Scan(&already); err != nil {
			return err
		}
		if already {
			return ErrAlreadyCompensated
		}
		if _, err := s.wallet.ApplyTx(ctx, q, c.DriverID, amount,
			"compensation", c.OrderID, "", &actorID); err != nil {
			return err
		}
		if err := s.DebitTreasury(ctx, q, amount, c.OrderID,
			"تعويضُ سائقٍ عن طلبٍ لم يكتمل", actorID); err != nil {
			return err
		}
		if c.Fault == FaultMerchant {
			if err := s.openMerchantClaim(ctx, q, c.OrderID, amount); err != nil {
				return err
			}
		}
	case CompKindMerchantGoods:
		if err := s.CompensateGoodsTx(ctx, q, c.OrderID, actorID, amount); err != nil {
			return err
		}
	case CompKindComplaint:
		var already bool
		if err := q.QueryRow(ctx, `
			SELECT EXISTS(SELECT 1 FROM wallet_transactions
			              WHERE ref = $1 AND kind = 'compensation')`, c.TicketID).Scan(&already); err != nil {
			return err
		}
		if already {
			return ErrAlreadyCompensated
		}
		label := fmt.Sprintf("تعويض شكوى #%d", c.TicketNumber)
		if _, err := s.wallet.ApplyTx(ctx, q, c.DriverID, amount,
			"compensation", c.TicketID, label, &actorID); err != nil {
			return err
		}
		if err := s.DebitTreasury(ctx, q, amount, c.TicketID, label, actorID); err != nil {
			return err
		}
		if _, err := q.Exec(ctx,
			`UPDATE tickets SET compensation = $2, updated_at = now() WHERE id = $1`,
			c.TicketID, amount); err != nil {
			return err
		}
	default:
		return ErrCompensationInvalid
	}
	if err := s.DecideCompensationTx(ctx, q, c.ID, CompensationApproved, amount, actorID, note); err != nil {
		return err
	}
	if selfApproved {
		if _, err := q.Exec(ctx,
			`UPDATE driver_compensation_requests SET self_approved = true WHERE id = $1`, c.ID); err != nil {
			return err
		}
	}
	return nil
}

// OpenMerchantClaimTx **المطالبةُ على المتجر تُفتح بما دُفع فعلاً.**
func (s *Service) OpenMerchantClaimTx(ctx context.Context, q wallet.Querier,
	orderID string, amount int64) error {
	return s.openMerchantClaim(ctx, q, orderID, amount)
}

// CompensationFilter **فلاترُ الصفحة** — كلُّها في الرابط.
type CompensationFilter struct {
	// Status pending · approved · rejected · all (وفراغُه pending).
	Status string
	Kind   string
	Fault  string
	// Person اسمٌ أو هاتفٌ أو معرّفُ المستفيد.
	Person string
	// From/To تاريخان `YYYY-MM-DD` بتوقيت دمشق.
	From, To string
}

func (f CompensationFilter) where() (string, []any) {
	conds := []string{}
	args := []any{}
	add := func(cond string, v any) {
		args = append(args, v)
		conds = append(conds, strings.ReplaceAll(cond, "$?", fmt.Sprintf("$%d", len(args))))
	}
	switch f.Status {
	case "", CompensationPending:
		conds = append(conds, "r.status = 'pending'")
	case CompensationApproved, CompensationRejected:
		add("r.status = $?", f.Status)
	}
	if f.Kind != "" {
		add("r.kind = $?", f.Kind)
	}
	if f.Fault != "" {
		add("r.fault = $?", f.Fault)
	}
	if p := strings.TrimSpace(f.Person); p != "" {
		add("(r.driver_id::text = $? OR u.full_name ILIKE '%' || $? || '%' OR u.phone::text LIKE '%' || $? || '%')", p)
	}
	if f.From != "" {
		add("(r.created_at AT TIME ZONE 'Asia/Damascus')::date >= $?::date", f.From)
	}
	if f.To != "" {
		add("(r.created_at AT TIME ZONE 'Asia/Damascus')::date <= $?::date", f.To)
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// CompensationSummary **بطاقاتُ الصفحة** — ما ينتظر (عدداً ومجموعاً)، وما قُبل
// ورُفض هذا الشهر، وما تجاوز المهلة.
type CompensationSummary struct {
	PendingCount       int   `json:"pending_count"`
	PendingSum         int64 `json:"pending_sum"`
	ApprovedMonthCount int   `json:"approved_month_count"`
	ApprovedMonthSum   int64 `json:"approved_month_sum"`
	RejectedMonthCount int   `json:"rejected_month_count"`
	OverdueCount       int   `json:"overdue_count"`
	OverdueHours       int64 `json:"overdue_hours"`
}

// OverdueHours **مهلةُ الانتظار قبل أن يحمرّ الطلب** — من الإعدادات.
func (s *Service) OverdueHours(ctx context.Context) int64 {
	if s.settings == nil {
		return 24
	}
	h := s.settings.GetInt(ctx, "compensations.overdue_hours")
	if h <= 0 {
		return 24
	}
	return h
}

// Compensations **قائمةُ التعويضات بفلاترها** — الأقدمُ أوّلاً للمعلَّق، والأحدثُ
// أوّلاً للسجلّ.
func (s *Service) Compensations(ctx context.Context, f CompensationFilter, limit, offset int) ([]CompensationRequest, int, error) {
	where, args := f.where()
	var total int
	if err := s.db.QueryRow(ctx, `
		SELECT count(*) FROM driver_compensation_requests r
		JOIN users u ON u.id = r.driver_id`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := " ORDER BY r.created_at DESC"
	if f.Status == "" || f.Status == CompensationPending {
		order = " ORDER BY r.created_at ASC"
	}
	n := len(args)
	rows, err := s.db.Query(ctx, compensationCols+where+order+
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", n+1, n+2), append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	hours := s.OverdueHours(ctx)
	caps := map[string]int64{}
	out := []CompensationRequest{}
	for rows.Next() {
		c, err := scanCompensation(rows)
		if err != nil {
			return nil, 0, err
		}
		if _, ok := caps[c.Kind]; !ok {
			caps[c.Kind] = s.CompensationCap(ctx, c.Kind)
		}
		c.Cap = caps[c.Kind]
		c.Overdue = c.Status == CompensationPending &&
			time.Since(c.CreatedAt) > time.Duration(hours)*time.Hour
		out = append(out, *c)
	}
	return out, total, rows.Err()
}

// PendingCompensations **ما ينتظر قراراً** — بالأقدم أوّلاً.
func (s *Service) PendingCompensations(ctx context.Context, limit, offset int) ([]CompensationRequest, int, error) {
	return s.Compensations(ctx, CompensationFilter{Status: CompensationPending}, limit, offset)
}

// CompensationsSummary **أرقامُ البطاقات** — بتوقيت دمشق للشهر.
func (s *Service) CompensationsSummary(ctx context.Context) (CompensationSummary, error) {
	sum := CompensationSummary{OverdueHours: s.OverdueHours(ctx)}
	err := s.db.QueryRow(ctx, `
		WITH m AS (SELECT date_trunc('month', now() AT TIME ZONE 'Asia/Damascus')
		                  AT TIME ZONE 'Asia/Damascus' AS start)
		SELECT count(*) FILTER (WHERE status = 'pending'),
		       COALESCE(sum(suggested_amount) FILTER (WHERE status = 'pending'), 0),
		       count(*) FILTER (WHERE status = 'approved' AND decided_at >= m.start),
		       COALESCE(sum(amount) FILTER (WHERE status = 'approved' AND decided_at >= m.start), 0),
		       count(*) FILTER (WHERE status = 'rejected' AND decided_at >= m.start),
		       count(*) FILTER (WHERE status = 'pending'
		                         AND created_at < now() - make_interval(hours => $1::int))
		FROM driver_compensation_requests, m`, sum.OverdueHours).
		Scan(&sum.PendingCount, &sum.PendingSum, &sum.ApprovedMonthCount,
			&sum.ApprovedMonthSum, &sum.RejectedMonthCount, &sum.OverdueCount)
	return sum, err
}

// SweepOverdueCompensations **ينبّه المالكَ مرّةً لكلّ طلبٍ تجاوز المهلة**
// (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٤). ويردّ عددَ ما نُبّه عنه.
func (s *Service) SweepOverdueCompensations(ctx context.Context) (int, error) {
	hours := s.OverdueHours(ctx)
	rows, err := s.db.Query(ctx, `
		UPDATE driver_compensation_requests SET overdue_alerted_at = now()
		WHERE status = 'pending' AND overdue_alerted_at IS NULL
		  AND created_at < now() - make_interval(hours => $1::int)
		RETURNING id::text`, hours)
	if err != nil {
		return 0, err
	}
	n := 0
	for rows.Next() {
		n++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if n > 0 && s.notify != nil {
		s.notify.NotifyRoles(ctx, []string{"owner_super_admin"}, notifications.Input{
			Kind:   notifications.KindOrder,
			Title:  "تعويضات تأخر القرار فيها",
			Body:   itoa(int64(n)) + " طلب تعويض ينتظر أكثر من " + itoa(hours) + " ساعة",
			Entity: "compensation",
			Href:   "/dashboard/compensations",
		})
	}
	return n, nil
}
