package server

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **نظيرُ السائق الأدنى — لشهود الزبون وحدَها** (٢٠٢٦-٠٩-٢٢، قرارُ المالك)
// ══════════════════════════════════════════════════════════════════════
//
// **لا قبولَ سائقٍ عامّ، ولا جلسةَ سائقٍ/أدمن قابلةً لإعادة الاستعمال.** يسوق
// طلبَ زبون QA المخصّصَ النقديَّ عبر الحالات (ليقرأ الزبونُ بطاقتَه في كلّ طور،
// ويظهر اسمُ السائق، وتُفتَح المحادثةُ ثمّ تُغلَق، ويصير التقييمُ متاحاً).
//
// # ولماذا مخصّصٌ نقديّ حصراً — الحيادُ الماليّ
//
// **التسويةُ في `settle` تتخطّى المخصّصَ كلَّه إلّا إن كان محفظةً** (`case 0`):
// **فطلبٌ مخصّصٌ نقديٌّ يبلغ «سُلّم» بلا أيّ قيدِ محفظةٍ أو صندوقِ سائقٍ أو خزينة**
// — صفرُ أثرٍ ماليّ. **والعاديُّ يقيّد أرباحَ متجرٍ وسائقٍ وخزينةً**، فيُرفض هنا.
//
// # والسائقُ هويّةٌ لا جلسة
//
// **يُختار سائقٌ فعليٌّ نشط** (يحتاجه `AssignDriver` لضبط `orders.driver_id`
// ⇒ اسمُ السائق ونظيرُ المحادثة)، **ويُستعمل معرّفُه فاعلاً للانتقالات** —
// **بلا إصدار توكن، بلا كشف جلسة.**

// qaOrderLadder **سُلّمُ الطلب المخصّص** — الترتيبُ الذي يُساق فيه.
//
// **المخصّصُ يتخطّى accepted/preparing/at_pickup** (خريطةُ `customTransitions`):
// pending ⇒ dispatching ⇒ assigned ⇒ picked_up ⇒ on_the_way ⇒ at_dropoff ⇒ delivered.
var qaOrderLadder = []string{
	"pending", "dispatching", "assigned", "picked_up", "on_the_way", "at_dropoff", "delivered",
}

func qaLadderIndex(status string) int {
	for i, s := range qaOrderLadder {
		if s == status {
			return i
		}
	}
	return -1
}

// qaDriverIdentity **يردّ معرّفَ سائقٍ فعليٍّ نشط** — لا توكن، لا جلسة.
func (s *Server) qaDriverIdentity(w http.ResponseWriter, r *http.Request) (string, bool) {
	var did string
	err := s.pg.QueryRow(r.Context(), `
		SELECT u.id::text FROM users u
		JOIN user_roles ur ON ur.user_id = u.id AND ur.role_code = 'driver'
		WHERE u.status = 'active'
		ORDER BY u.created_at LIMIT 1`).Scan(&did)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_driver", "errors.conflict"))
		return "", false
	} else if err != nil {
		s.respondErr(w, err)
		return "", false
	}
	return did, true
}

// qaOrderAdvance **يسوق طلبَ زبون QA المخصّصَ النقديَّ إلى حالةٍ هدف.**
//
// **QA-scoped**: يرفض طلباً لا يملكه زبونُ QA. **محايدٌ ماليّاً**: يرفض غيرَ
// المخصّص أو غيرَ النقديّ. **لا رجوعَ**: الهدفُ فوقَ الحالة الجارية.
func (s *Server) qaOrderAdvance(w http.ResponseWriter, r *http.Request, uid, orderID, target string) {
	if !isUUID(orderID) {
		s.respondErr(w, errValidation)
		return
	}
	toIdx := qaLadderIndex(target)
	if toIdx <= 0 { // pending (0) ليس هدفاً، والمجهولُ يُرفض
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_bad_target", "errors.validation"))
		return
	}
	ctx := r.Context()
	var custID, kind, payment, status string
	err := s.pg.QueryRow(ctx,
		`SELECT customer_id::text, kind, payment_method, status FROM orders WHERE id = $1::uuid`,
		orderID).Scan(&custID, &kind, &payment, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	// **QA-scoped**: طلبُ زبون QA وحدَه.
	if custID != uid {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_qa_order", "errors.forbidden"))
		return
	}
	// **الحيادُ الماليّ**: مخصّصٌ نقديٌّ حصراً (لا قيدَ محفظةٍ/صندوقٍ/خزينة).
	if kind != "custom" || payment != "cash" {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_needs_custom_cash", "errors.validation"))
		return
	}
	fromIdx := qaLadderIndex(status)
	if fromIdx < 0 {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_order_off_ladder", "errors.conflict"))
		return
	}
	if toIdx <= fromIdx {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_backward", "errors.conflict"))
		return
	}
	// السائقُ الفعليّ — يحتاجه ضبطُ driver_id واسمُه ونظيرُ المحادثة.
	driverID, ok := s.qaDriverIdentity(w, r)
	if !ok {
		return
	}
	// **الساقُ خطوةً خطوة** — كلُّ طورٍ بفاعله ودورِه الصحيح.
	for i := fromIdx + 1; i <= toIdx; i++ {
		to := qaOrderLadder[i]
		switch to {
		case "dispatching":
			// المخصّص: pending ⇒ dispatching (ops).
			if _, err := s.orders.Transition(ctx, driverID, []string{"ops"}, orderID, to, "QA advance"); err != nil {
				s.respondErr(w, err)
				return
			}
		case "assigned":
			// يضبط driver_id ثمّ يُسند (dispatching ⇒ assigned) — الطريقُ المُعتمَد.
			if _, err := s.orders.AssignDriver(ctx, driverID, []string{"ops"}, orderID, driverID, "QA assign"); err != nil {
				s.respondErr(w, err)
				return
			}
		default:
			// picked_up / on_the_way / at_dropoff / delivered — فعلُ السائق.
			if _, err := s.orders.Transition(ctx, driverID, []string{"driver"}, orderID, to, "QA advance"); err != nil {
				s.respondErr(w, err)
				return
			}
		}
	}
	var finalStatus string
	_ = s.pg.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1::uuid`, orderID).Scan(&finalStatus)
	s.logger.Warn("QA order advanced (staging-only)",
		"order", orderID, "from", status, "to", finalStatus, "driver", driverID)
	httpx.JSON(w, http.StatusOK, map[string]any{
		"order_id": orderID, "status": finalStatus, "driver_id": driverID, "was": status,
	})
}

// qaOrderChatSend **يرسل رسالةَ سائقٍ على طلبِ زبون QA** (SUP-002) — عبر
// `comms.Send` بنفس مسار التطبيق، لا كتابةَ قاعدةٍ خام. QA-scoped وعلى طلبٍ
// مُسنَدٍ (القناةُ مفتوحةٌ من `assigned` حتّى `at_dropoff`).
func (s *Server) qaOrderChatSend(w http.ResponseWriter, r *http.Request, uid, orderID, body string) {
	if !isUUID(orderID) || body == "" {
		s.respondErr(w, errValidation)
		return
	}
	ctx := r.Context()
	var custID string
	var driverID *string
	err := s.pg.QueryRow(ctx,
		`SELECT customer_id::text, driver_id::text FROM orders WHERE id = $1::uuid`,
		orderID).Scan(&custID, &driverID)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	} else if err != nil {
		s.respondErr(w, err)
		return
	}
	if custID != uid {
		s.respondErr(w, httpx.NewError(http.StatusForbidden, "qa_not_qa_order", "errors.forbidden"))
		return
	}
	if driverID == nil || *driverID == "" {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_driver_on_order", "errors.conflict"))
		return
	}
	// دورُ المرسِل يُشتقّ من `orders.driver_id` داخلَ `Permit` — لا يُمرَّر.
	p, err := s.comms.Permit(ctx, orderID, *driverID)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	msg, err := s.comms.Send(ctx, p, body)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.logger.Warn("QA driver chat sent (staging-only)", "order", orderID, "driver", *driverID)
	httpx.JSON(w, http.StatusOK, map[string]any{"order_id": orderID, "sent": true, "message_id": msg.ID})
}
