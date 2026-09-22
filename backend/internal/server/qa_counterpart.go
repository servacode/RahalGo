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

// qaOrderAdvance **يسوق طلبَ زبون QA إلى حالةٍ هدف.**
//
// **مساران، كلاهما نقديٌّ محايدٌ ماليّاً و QA-scoped:**
//   - **مخصّص**: السُّلّمُ كاملاً حتّى `delivered` (يُسنِد سائقاً، يتّفق السعر).
//   - **عاديّ**: **حتّى `accepted` فقط** — ما قبل التسوية، بلا سائقٍ ولا مال
//     (لشهود 14-011). أيُّ هدفٍ آخرَ للعاديّ يُرفض.
//
// **QA-scoped**: يرفض طلباً لا يملكه زبونُ QA. **لا رجوعَ**: الهدفُ فوقَ الحالة الجارية.
func (s *Server) qaOrderAdvance(w http.ResponseWriter, r *http.Request, uid, orderID, target string) {
	if !isUUID(orderID) {
		s.respondErr(w, errValidation)
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
	// ══════════════════════════════════════════════════════════════════
	// **الطلبُ العاديّ: «accepted» فقط — لشهود 14-011** (ما قبل التسوية)
	// ══════════════════════════════════════════════════════════════════
	//
	// **التسويةُ تبدأ عند `picked_up`** (`transitions.go`: `settle` حالتا ٢/٣):
	// **فالوقوفُ عند `accepted` لا يمسّ محفظةً ولا صندوقاً ولا خزينةً ولا
	// عمولةً ولا تعويضاً.** **نقديٌّ حصراً** فلا خصمَ إنشاءٍ للمحفظة (العاديُّ
	// يخصم عند الإنشاء إن كان محفظةً). **لا سائقَ**، **وقابلٌ للإلغاء**
	// (accepted ⇒ cancelled للزبون) فيُنظَّف.
	if kind != "custom" {
		if payment != "cash" {
			s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_normal_cash_only", "errors.validation"))
			return
		}
		if target != "accepted" {
			s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_normal_accepted_only", "errors.validation"))
			return
		}
		if status != "pending" {
			s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_backward", "errors.conflict"))
			return
		}
		actor, aerr := s.qaNonCustomerAuthor(ctx, uid)
		if aerr != nil {
			s.respondErr(w, aerr)
			return
		}
		// pending ⇒ accepted (merchant/ops) — بلا سائقٍ ولا تسوية.
		if _, err := s.orders.Transition(ctx, actor, []string{"ops"}, orderID, "accepted", "QA accept"); err != nil {
			s.respondErr(w, err)
			return
		}
		s.logger.Warn("QA normal order accepted (staging-only)", "order", orderID, "actor", actor)
		httpx.JSON(w, http.StatusOK, map[string]any{"order_id": orderID, "status": "accepted", "was": status})
		return
	}
	// **المخصّصُ: الحيادُ الماليّ** — نقديٌّ حصراً.
	if payment != "cash" {
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_needs_custom_cash", "errors.validation"))
		return
	}
	toIdx := qaLadderIndex(target)
	if toIdx <= 0 { // pending (0) ليس هدفاً، والمجهولُ يُرفض
		s.respondErr(w, httpx.NewError(http.StatusBadRequest, "qa_bad_target", "errors.validation"))
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
		case "picked_up":
			// **المخصّصُ يحتاج اتّفاقَ السعر قبل الاستلام** (وإلّا `custom_not_agreed`).
			// **قيمٌ رمزيّةٌ حتميّة، والمخصّصُ النقديُّ لا يُسوّى** (`settle` تخرج قبل
			// قراءة الأعمدة) **فلا عمولةَ ولا مستحقَّ متجرٍ ولا خزينةَ ولا قيدَ صندوق.**
			if err := s.orders.AgreeCustom(ctx, orderID, driverID, 5000, 1000); err != nil {
				s.respondErr(w, err)
				return
			}
			if _, err := s.orders.Transition(ctx, driverID, []string{"driver"}, orderID, to, "QA advance"); err != nil {
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
