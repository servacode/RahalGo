package orders

// ══════════════════════════════════════════════════════════════════════
// **الإدارةُ تُنهي عند باب الزبون — والسائقُ يُبلّغ وينتظر** (قرارُ المالك
// مساءَ ٢٠٢٦-١٠-٠٢، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
// «دائماً إذا في مشكلة بين السائق والزبون يكون الردّ: انتظر، الإدارة تقوم
// بالتواصل مع الزبون، **ويبقى الطلبُ مع السائق إلى أن تُحلّ القصّة** والزبونُ
// يستلم أو يرفض نهائيّاً. **وقتها الإدارةُ هي تُنهي الطلبَ من عندها، يصل أمرٌ
// للسائق — مثلاً العودة إلى المكتب.**»
//
// # ما كان
//
// **السائقُ يضغط «الزبونُ غير موجود» فيُغلق الطلبُ فشلاً** — ويُمنع الزبونُ من
// النقد شهراً بضغطةٍ واحدة، **ولا أحدَ في المكتب قال شيئاً.**
//
// # وما صار — أمران لا ثالثَ لهما
//
//	سلّم الآن         `deliver_now`       الطلبُ يبقى عند الباب مع السائق،
//	                                       ويُسلّمه كالعادة (صورةٌ ثمّ «سلّمت»)
//	عُد إلى المكتب    `return_to_office`  الطلبُ يُنهى فشلاً **بذنبٍ يكتبه المكتب**،
//	                                       والبضاعةُ إلى المكتب (`goods.go` يحسمها
//	                                       هناك)، والتعويضُ طلبٌ معلَّقٌ بحسب الذنب
//
// **والأمرُ يُكتب على الطلب** (`door_instruction`) — **فيقرؤه تطبيقُ السائق ولو
// ضاع الإشعار**، ويُدفَع إليه عاجلاً ويُبثّ حيّاً.
//
// # والذنبُ قرارٌ لا سبب
//
// **السببُ يقترح** (`SuggestedFault`)، **والمكتبُ يحكم**: «الزبونُ غير موجود»
// قد يكون عنواناً كتبته الخريطةُ خطأً — ذنبُ المنصّة. **ومن الذنب يقع الباقي
// بلا يدٍ ثانية**: الزبونُ ⇒ تعويضٌ معلَّقٌ ومنعُ النقد (`cashban.go`) ·
// المتجرُ ⇒ تعويضٌ ومطالبةٌ عند الموافقة · المنصّةُ ⇒ تعويضٌ من الخزينة ·
// **السائقُ ⇒ لا تعويض**، **والخصمُ أو الإنذارُ بقرار المكتب من بابه القائم**
// (`POST /admin/users/{id}/incentive` بنوع `penalty`) — **لا آليّاً.**

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// أمرا المكتب عند الباب.
const (
	DoorDeliverNow     = "deliver_now"
	DoorReturnToOffice = "return_to_office"
)

// أخطاءُ الإنهاء.
var (
	// ErrNotAtDoor **الطلبُ ليس عند باب الزبون** — أُنهي أو لم يصل بعد.
	ErrNotAtDoor = httpx.NewError(http.StatusConflict, "order_not_at_door", "errors.order_not_at_door")
	// ErrDoorBadAction **أمرٌ لا يُعرف.**
	ErrDoorBadAction = httpx.NewError(http.StatusBadRequest, "door_bad_action", "errors.door_bad_action")
	// ErrDoorBadFault **العودةُ إلى المكتب بلا ذنبٍ معروف** — والذنبُ يقرّر المال.
	ErrDoorBadFault = httpx.NewError(http.StatusBadRequest, "door_bad_fault", "errors.door_bad_fault")
	// ErrDoorNeedsNote **إنهاءٌ بلا كلمة** — ما لا يُستدرَك يلزمه سبب.
	ErrDoorNeedsNote = httpx.NewError(http.StatusBadRequest, "reason_required", "errors.reason_required")
)

// DoorResolution **ما قرّره المكتبُ عند الباب.**
type DoorResolution struct {
	// Action `deliver_now` أو `return_to_office`.
	Action string
	// Fault **على من الذنب** — يلزم مع العودة إلى المكتب وحدَها.
	Fault string
	// Reason **بلاغُ الباب الذي يُنهى به** — وفارغُه: آخرُ بلاغٍ من السائق.
	Reason string
	// Note **كلمةُ المكتب** — تصل السائقَ مع الأمر، **وتلزم مع العودة.**
	Note string
}

// doorTitle **عنوانُ الأمر كما يقرؤه السائقُ من شاشةٍ مقفلة.**
var doorTitle = map[string]string{
	DoorDeliverNow:     "الإدارة: سلّم الآن",
	DoorReturnToOffice: "الإدارة: عُد إلى المكتب بالطلب",
}

// doorContinueTitle **«سلّم الآن» قبل الباب تُقرأ «أكمل التوصيل»** (٢٠٢٦-١٠-٠٣) —
// الأمرُ نفسُه، والسائقُ ما زال في الطريق.
const doorContinueTitle = "الإدارة: أكمل التوصيل"

// doorCollectTitle **وعند المتجر «استلم الطلب»** (٢٠٢٦-١٠-٠٣) — اتّصلت الإدارةُ بالمتجر فحُلّ.
const doorCollectTitle = "الإدارة: استلم الطلب"

// DoorTitle **نصُّ الأمر** — وفارغٌ لأمرٍ لا يُعرف.
func DoorTitle(action string) string { return doorTitle[action] }

// ResolveDoor **يُنهي المكتبُ ما وقع عند باب الزبون** — `hook` أثرُ التدقيق في
// المعاملة نفسِها (`XG-20`).
func (s *Service) ResolveDoor(ctx context.Context, actorID string, actorRoles []string,
	orderID string, in DoorResolution, hook func(context.Context, dbtx.Querier) error) (*Order, error) {
	in.Note = strings.TrimSpace(in.Note)
	switch in.Action {
	case DoorDeliverNow:
		return s.doorDeliverNow(ctx, orderID, in, hook)
	case DoorReturnToOffice:
		if !IsFault(in.Fault) {
			return nil, ErrDoorBadFault
		}
		if in.Note == "" {
			return nil, ErrDoorNeedsNote
		}
		reason := strings.TrimSpace(in.Reason)
		if reason == "" {
			reason = s.lastDoorReport(ctx, orderID)
		}
		var status string
		if err := s.db.QueryRow(ctx,
			`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, httpx.ErrNotFound
			}
			return nil, err
		}
		if !OfficeDecides(status) {
			return nil, ErrNotAtDoor
		}
		// **والأمرُ يُكتب مع الانتقال في معاملته** — فلا يُقرأ «فشل» بلا أمر.
		mark := func(ctx context.Context, q dbtx.Querier) error {
			if _, err := q.Exec(ctx, `
				UPDATE orders SET door_instruction = $2, door_instruction_note = $3,
				                  door_instruction_at = now()
				WHERE id = $1`, orderID, DoorReturnToOffice, in.Note); err != nil {
				return err
			}
			if hook != nil {
				return hook(ctx, q)
			}
			return nil
		}
		o, err := s.transitionTx(ctx, actorID, actorRoles, orderID, StFailed,
			in.Note, reason, in.Fault, mark)
		if err != nil {
			// **وطلبٌ تحرّك بين القراءة والقفل** — لم يعد عند الباب.
			if errors.Is(err, ErrBadTransition) {
				return nil, ErrNotAtDoor
			}
			return nil, err
		}
		// **والسائقُ يُخبَر بالأمر** — `notifyDriverLost` بعد الانتقال يقوله
		// (`LossReturnToOffice`)، **عاجلاً ومحفوظاً.**
		return o, nil
	}
	return nil, ErrDoorBadAction
}

// doorDeliverNow **«سلّم الآن» — الطلبُ يبقى عند الباب والأمرُ يصل السائق.**
func (s *Service) doorDeliverNow(ctx context.Context, orderID string, in DoorResolution,
	hook func(context.Context, dbtx.Querier) error) (*Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var driverID *string
	var number int64
	if err := tx.QueryRow(ctx,
		`SELECT status, driver_id::text, number FROM orders WHERE id = $1 FOR UPDATE`, orderID).
		Scan(&status, &driverID, &number); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, httpx.ErrNotFound
		}
		return nil, err
	}
	if !OfficeDecides(status) || driverID == nil {
		return nil, ErrNotAtDoor
	}
	if _, err := tx.Exec(ctx, `
		UPDATE orders SET door_instruction = $2, door_instruction_note = $3,
		                  door_instruction_at = now(), updated_at = now()
		WHERE id = $1`, orderID, DoorDeliverNow, in.Note); err != nil {
		return nil, err
	}
	if hook != nil {
		if err := hook(ctx, tx); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}
	s.publishOrder(o)
	title := doorTitle[DoorDeliverNow]
	switch status {
	case StAtPickup:
		title = doorCollectTitle
	case StPickedUp, StOnTheWay:
		title = doorContinueTitle
	}
	s.notifyDoorInstruction(ctx, orderID, *driverID, number, title, in.Note)
	return o, nil
}

// notifyDoorInstruction **الأمرُ يصل السائقَ عاجلاً ومحفوظاً** — يوقظ الهاتفَ
// المقفل، **ويُقرأ في صندوقه إن فاتته الرنّة.**
func (s *Service) notifyDoorInstruction(ctx context.Context, orderID, driverID string,
	number int64, title, note string) {
	if s.notify == nil || driverID == "" {
		return
	}
	body := fmt.Sprintf("#%d", number)
	if note != "" {
		body += " — " + note
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindOrder,
		Title: title, Body: body,
		Entity: "order", EntityID: orderID, Href: "/portal",
		Apps: []string{notifications.AppDriver},
	})
	s.pub.Publish("driver:"+driverID, map[string]any{"type": "order"})
}

// lastDoorReport **آخرُ بلاغٍ بعد الاستلام لهذا الطلب** — سببُ الإنهاء إن لم يُذكر.
//
// **ومن سجلّ التدقيق** (`driver.stage_report`) — هناك يُكتب البلاغ. **وفارغٌ إن
// لم يُبلَّغ شيء**: إنهاءٌ بلا بلاغٍ قرارُ مكتبٍ بكلمته وحدَها.
func (s *Service) lastDoorReport(ctx context.Context, orderID string) string {
	var code string
	_ = s.db.QueryRow(ctx, `
		SELECT details->>'code' FROM audit_log
		WHERE action = 'driver.stage_report' AND entity = 'order' AND entity_id = $1
		  AND details->>'status' IN ($2, $3, $4, $5)
		ORDER BY created_at DESC LIMIT 1`, orderID, StAtPickup, StPickedUp, StOnTheWay, StAtDropoff).Scan(&code)
	if !IsTripReport(code) {
		return ""
	}
	return code
}
