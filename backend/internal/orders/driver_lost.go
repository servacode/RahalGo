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
)

// lossText **جملةُ كلّ رمز** — قصيرةٌ تُقرأ من شاشةٍ مقفلة.
var lossText = map[string]string{
	LossCancelledCustomer: "ألغى الزبون الطلب",
	LossCancelledMerchant: "ألغى المتجر الطلب",
	LossCancelledOps:      "ألغت الإدارة الطلب",
	LossRequeuedOps:       "أعادته الإدارة إلى الطابور",
	LossRequeuedSystem:    "أعادته المنصة إلى الطابور",
	LossMerchantBlocked:   "عاد الطلب إلى الإدارة لتبديل المتجر — وتعويضك بعد موافقتها",
	LossFailedOps:         "أنهته الإدارة — تعذّر التسليم",
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
	// المتجر لا يقول له إنّ الطلبَ صار عند الإدارة وإنّ تعويضَه ينتظرها.
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

// notifyDriverLost **يُخبر السائقَ بأنّ طلبَه لم يعد معه** — عاجلٌ ومحفوظ.
//
// **والصندوقُ يحفظه** (لا `Transient`): من كان يقود ولم يقرأ الرنّةَ يجده حين
// يقف. **والنوعُ «طلب» عاجلٌ في الدفع** — يوقظ الهاتفَ المقفل.
func (s *Service) notifyDriverLost(ctx context.Context, orderID, driverID, code string) {
	if s.notify == nil || driverID == "" || code == "" {
		return
	}
	var number int64
	if err := s.db.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number); err != nil {
		return
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
		to       string
		actor    *string
		endedBy  string
		holdsNow bool
	)
	err := s.db.QueryRow(ctx, `
		SELECT o.id::text, o.number, o.status, e.to_status, e.actor_id::text,
		       COALESCE(o.ended_by, ''), e.created_at,
		       (o.driver_id IS NOT DISTINCT FROM $2::uuid AND o.closed_at IS NULL)
		FROM order_events e
		JOIN orders o ON o.id = e.order_id
		WHERE e.order_id = $1::uuid AND e.driver_id = $2::uuid
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT 1`, orderID, driverID).
		Scan(&out.OrderID, &out.Number, &out.Status, &to, &actor, &endedBy, &out.At, &holdsNow)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotDriversOrder
	}
	if err != nil {
		return nil, err
	}
	if holdsNow {
		return &out, nil
	}
	byHim := actor != nil && *actor == driverID
	out.Reason = DriverLossCode(to, endedBy, byHim, actor == nil)
	out.Message = lossText[out.Reason]
	return &out, nil
}
