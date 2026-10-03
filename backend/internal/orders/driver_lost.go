package orders

// ══════════════════════════════════════════════════════════════════════
// **السائقُ يُخبَر حين يُؤخذ منه طلبُه** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس حيّاً**: ألغى الزبونُ أو العملياتُ طلباً بيد سائق، أو أعادته العملياتُ
// إلى الطابور — **فلا إشعارَ ولا سطرَ في صندوقه**؛ والتطبيقُ يُغلق الملاحةَ
// ويقفز إلى «الطلبات» صامتاً. **فيقف في الشارع لا يعرف: أعطبٌ في هاتفه أم
// طلبٌ أُلغي؟** ويتّصل بالمكتب ليسأل عمّا كان يكفيه سطر.
//
// **و`notifyTransition` يُخبر الزبونَ والعملياتِ والمتجرَ — ولا يعرف السائق.**
//
// # ومتى يُخبَر
//
// **حين يخرج الطلبُ من يده بفعل غيره** — إلغاءٌ، أو إعادةٌ إلى الطابور، أو
// إنهاءٌ يدويّ، أو ردٌّ إلى المكتب لتبديل المتجر. **ولا يُخبَر بفعله هو**: من
// ضغط «أعد الطلب» يعرف أنّه أعاده، **ورنّةٌ تخبره بما فعله للتوّ ضجيج.**

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// رموزُ خروج الطلب من يد السائق — **يقرؤها التطبيقُ نصّاً بلغته**، والخادمُ
// يرسل معها جملتَه لمن لا يعرف الرمز.
const (
	LossCancelledCustomer = "cancelled_customer"
	LossCancelledMerchant = "cancelled_merchant"
	LossCancelledOps      = "cancelled_ops"
	LossRequeuedOps       = "requeued_ops"
	LossRequeuedSystem    = "requeued_system"
	LossMerchantBlocked   = "merchant_blocked"
	LossFailedOps         = "failed_ops"
	// LossReturnToOffice **أنهته الإدارةُ عند باب الزبون — عُد بالطلب إلى المكتب**
	// (قرارُ المالك مساءَ ٢٠٢٦-١٠-٠٢).
	LossReturnToOffice = "return_to_office"
	// LossReturnToStore **وإلى المتجر الذي يقبل الاسترداد** (قرارُ المالك ٢٠٢٦-١٠-٠٣).
	LossReturnToStore = "return_to_store"
)

// lossText **جملةُ كلّ رمز** — قصيرةٌ تُقرأ من شاشةٍ مقفلة.
var lossText = map[string]string{
	LossCancelledCustomer: "ألغى الزبون الطلب",
	LossCancelledMerchant: "ألغى المتجر الطلب",
	LossCancelledOps:      "ألغت الإدارة الطلب",
	LossRequeuedOps:       "أعادته الإدارة إلى الطابور",
	LossRequeuedSystem:    "أعادته المنصة إلى الطابور",
	LossMerchantBlocked:   "عاد الطلب إلى الإدارة لتبديل المتجر",
	LossFailedOps:         "أنهته الإدارة — تعذّر التسليم",
	LossReturnToOffice:    "الإدارة: عُد إلى المكتب بالطلب",
	LossReturnToStore:     "الإدارة: أرجِع البضاعة إلى المتجر",
}

// LossText **جملةُ الرمز** — وفارغةٌ لرمزٍ لا يُعرف.
func LossText(code string) string { return lossText[code] }

// DriverLossCode **لماذا خرج الطلبُ من يده** — من الحال التي صار إليها ومن أنهاه.
//
// **وفارغٌ يعني «لا خبرَ له»**: سلّمه هو أو أعاده هو أو ما زال بيده.
//
//	to       حالُ الطلب بعد الخروج
//	endedBy  الدورُ الذي أنهاه (`orders.ended_by`) — للإلغاء والإنهاء
//	byHim    أفعلُه هو؟
//	system   بلا فاعلٍ إنسان (كانسٌ أو مهلة)
func DriverLossCode(to, endedBy string, byHim, system bool) string {
	// **والردُّ إلى المكتب لتبديل المتجر يُقال ولو كان بضغطته** — بلاغُه عن
	// المتجر لا يقول له إنّ الطلبَ صار عند الإدارة. **ولا ذكرَ للتعويض** — قرارُ المالك
	// ٢٠٢٦-١٠-٠٣: «الإدارة هي تقرّر بدون أيّ شيء»، والسائقُ لا يُوعَد فلا يطمع.
	if to == StAccepted {
		return LossMerchantBlocked
	}
	if byHim {
		return ""
	}
	switch to {
	case StCancelled:
		switch endedBy {
		case "customer":
			return LossCancelledCustomer
		case "merchant":
			return LossCancelledMerchant
		}
		return LossCancelledOps
	case StDispatching:
		if system {
			return LossRequeuedSystem
		}
		return LossRequeuedOps
	case StFailed:
		return LossFailedOps
	}
	return ""
}

// DriverLossCodeFrom **كـ`DriverLossCode` ومعه الحالُ التي خرج منها.**
//
// **والفشلُ من باب الزبون أمرٌ لا خبر**: لا يصل إليه إلّا المكتبُ بقراره
// (`ResolveDoor`)، **والسائقُ يحمل البضاعةَ ويحتاج أن يعرف إلى أين يعود بها.**
func DriverLossCodeFrom(from, to, endedBy string, byHim, system bool) string {
	// **وفي الطريق كذلك** (٢٠٢٦-١٠-٠٣): البضاعةُ معه أينما أُنهي الطلب.
	if AfterPickup(from) && to == StFailed && !byHim {
		return LossReturnToOffice
	}
	return DriverLossCode(to, endedBy, byHim, system)
}

// notifyDriverLost **يُخبر السائقَ بأنّ طلبَه لم يعد معه** — عاجلٌ ومحفوظ.
//
// **والصندوقُ يحفظه** (لا `Transient`): من كان يقود ولم يقرأ الرنّةَ يجده حين
// يقف. **والنوعُ «طلب» عاجلٌ في الدفع** — يوقظ الهاتفَ المقفل.
func (s *Service) notifyDriverLost(ctx context.Context, orderID, driverID, code string) {
	if s.notify == nil || driverID == "" || code == "" {
		return
	}
	var number int64
	var returnTo string
	if err := s.db.QueryRow(ctx, `SELECT number, COALESCE(return_to, '') FROM orders WHERE id = $1`,
		orderID).Scan(&number, &returnTo); err != nil {
		return
	}
	// **ووجهةُ الإرجاع تُقال باسمها** — «إلى المتجر» غيرُ «إلى المكتب».
	if code == LossReturnToOffice && returnTo == ReturnToStore {
		code = LossReturnToStore
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: driverID, Kind: notifications.KindOrder,
		Title:  t.driverLost,
		Body:   fmt.Sprintf("#%d — %s", number, lossText[code]),
		Entity: "order", EntityID: orderID, Href: "/portal",
		// **وتطبيقُ السائق وحدَه** — الحسابُ نفسُه قد يكون زبوناً.
		Apps: []string{notifications.AppDriver},
	})
}

// DriverOutcome **ما آل إليه طلبٌ كان بيده** — يقرؤه التطبيقُ حين يختفي الطلب
// من قائمته، **فيقول له لماذا بدل أن يقفز صامتاً.**
type DriverOutcome struct {
	OrderID string `json:"order_id"`
	Number  int64  `json:"number"`
	// Status **حالُ الطلب الآن.**
	Status string `json:"status"`
	// Reason **رمزُ الخروج** — وفارغٌ: لا خبر (سلّمه هو أو ما زال بيده).
	Reason string `json:"reason"`
	// Message **جملةُ الرمز بالعربيّة** — لتطبيقٍ لا يعرف رمزاً جديداً.
	Message string    `json:"message"`
	At      time.Time `json:"at"`
	// DoorInstruction **أمرُ الإدارة عند باب الزبون** — `deliver_now` أو
	// `return_to_office` أو فارغ (مساءَ ٢٠٢٦-١٠-٠٢). **ويُقرأ والطلبُ بيده بعد**:
	// «سلّم الآن» لا يُخرجه منه.
	DoorInstruction string `json:"door_instruction"`
	// DoorNote **كلمةُ الإدارة مع أمرها** — وفارغةٌ بلا أمر.
	DoorNote string `json:"door_note"`
	// ReturnTo **مشوارُ إرجاعٍ قائم** — `office` أو `store`، **وفارغٌ بلا مشوار أو بعد
	// «سلّمت البضاعة»** (قرارُ المالك ٢٠٢٦-١٠-٠٣).
	ReturnTo string `json:"return_to"`
}

// ══════════════════════════════════════════════════════════════════════
// RepeatsOwnTransition **أهذه إعادةٌ لخطوةٍ خطاها هو للتوّ؟** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **ردٌّ ضاع في الشبكة والخطوةُ ثبتت** — فيُعيدها التطبيقُ فيردّ المحرّكُ
// «انتقالٌ غيرُ جائز»، **ويرى السائقُ خطأً على خطوةٍ وقعت.**
//
// **والجوابُ من السجلّ**: حدثٌ إلى الحال نفسِها بفعله في الدقائق العشر الأخيرة،
// **ولم يمسّ الطلبَ بعده أحدٌ غيرُه** — فما بعده من فعله (استلم ثمّ انطلق). **و«تعذّر»
// عند المتجر يُكتب «إلى المكتب»** (`merchant_blocked.go`) فيُقرأ معه.
//
// **ولا تُلَفّ الانتقالاتُ بمنسّق المفاتيح** (`server/idempotency.go`): آلةُ الحال
// تفتح معاملتَها وتُطلق بعدها إشعاراتٍ وبثّاً، **والمنسّقُ يشترط أن يقع العملُ
// وعلامتُه في معاملته هو.** والحقيقةُ هنا في السجلّ أصلاً.
func (s *Service) RepeatsOwnTransition(ctx context.Context, orderID, driverID, to string) bool {
	targets := []string{to}
	if to == StFailed {
		targets = append(targets, StAccepted)
	}
	var again bool
	if err := s.db.QueryRow(ctx, `
		SELECT EXISTS (
		    SELECT 1 FROM order_events e
		    WHERE e.order_id = $1::uuid AND e.actor_id = $2::uuid
		      AND e.to_status = ANY($3::text[])
		      AND e.created_at > now() - interval '10 minutes'
		      AND NOT EXISTS (
		          SELECT 1 FROM order_events l
		          WHERE l.order_id = e.order_id AND l.id > e.id
		            AND l.actor_id IS DISTINCT FROM $2::uuid))`,
		orderID, driverID, targets).Scan(&again); err != nil {
		return false
	}
	return again
}

// ErrNotDriversOrder **لم يكن هذا الطلبُ بيده قطّ.**
var ErrNotDriversOrder = httpx.ErrNotFound

// DriverOutcomeOf **آخرُ حدثٍ كان فيه الطلبُ بيده** — ومنه يُقرأ لماذا خرج.
//
// **ومن سجلّ الأحداث لا من حال الطلب**: كلُّ انتقالٍ يكتب حاملَه قبله
// (`order_events.driver_id`، ٠١٧٢) — **فالحدثُ الذي أخرجه هو آخرُ حدثٍ يحمل
// اسمَه**، وبعده يُكتب اسمُ من تلاه أو لا اسم.
func (s *Service) DriverOutcomeOf(ctx context.Context, orderID, driverID string) (*DriverOutcome, error) {
	var (
		out      DriverOutcome
		from, to string
		actor    *string
		endedBy  string
		holdsNow bool
		handed   bool
	)
	err := s.db.QueryRow(ctx, `
		SELECT o.id::text, o.number, o.status, e.from_status, e.to_status, e.actor_id::text,
		       COALESCE(o.ended_by, ''), e.created_at,
		       (o.driver_id IS NOT DISTINCT FROM $2::uuid AND o.closed_at IS NULL),
		       o.door_instruction, o.door_instruction_note,
		       CASE WHEN o.status = 'failed' AND o.goods_handed_at IS NULL
		            THEN COALESCE(o.return_to, '') ELSE '' END,
		       o.goods_handed_at IS NOT NULL
		FROM order_events e
		JOIN orders o ON o.id = e.order_id
		WHERE e.order_id = $1::uuid AND e.driver_id = $2::uuid
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT 1`, orderID, driverID).
		Scan(&out.OrderID, &out.Number, &out.Status, &from, &to, &actor, &endedBy, &out.At,
			&holdsNow, &out.DoorInstruction, &out.DoorNote, &out.ReturnTo, &handed)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotDriversOrder
	}
	if err != nil {
		return nil, err
	}
	if holdsNow {
		return &out, nil
	}
	// **ومن سلّم البضاعةَ أنهى مشوارَه بيده** — لا خبرَ له عمّا فعله للتوّ.
	if handed {
		return &out, nil
	}
	byHim := actor != nil && *actor == driverID
	out.Reason = DriverLossCodeFrom(from, to, endedBy, byHim, actor == nil)
	if out.Reason == LossReturnToOffice && out.ReturnTo == ReturnToStore {
		out.Reason = LossReturnToStore
	}
	out.Message = lossText[out.Reason]
	return &out, nil
}
