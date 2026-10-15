package orders

// ══════════════════════════════════════════════════════════════════════
// **المتجرُ يُلغي توصيلتَه والسائقُ واقفٌ عنده** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **الإلغاءُ من `at_pickup` حقٌّ للمتجر** (الغرضُ ما زال عنده — `merchantDeliveryEdges`)،
// **لكنّ السائقَ قاد إليه وعاد بلا شيء.** وكان يقع صامتاً: لا إنذارَ ولا أثرَ يراه المكتب.
//
// **وصار**: إنذارٌ للمتجر («تكرارها بيأثر على حسابك») · وسطرٌ في سجلّ الطلب يقرؤه المكتب ·
// وتنبيهٌ للمكتب. **ولا تعويضَ آليّ** — المكتبُ يعوّض السائقَ بيده إن رأى، من باب اقتراح
// التعويض القائم (`ProposeCompensationTx`).

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// MerchantLateCancelNote **سطرُ السجلّ** — يقرؤه المكتبُ في الطلب.
const MerchantLateCancelNote = "المتجرُ ألغى التوصيلةَ بعد وصول السائق — لا تعويضَ آليّ، وللمكتب أن يعوّض السائق"

// MerchantLateCancelTitle **عنوانُ إنذار المتجر.**
const MerchantLateCancelTitle = "ألغيت التوصيلة بعد وصول السائق — تكرارها بيأثر على حسابك"

// markMerchantLateCancel **الأثرُ والإنذارُ في معاملة الإلغاء** — إن سقطت سقطا معها.
func markMerchantLateCancel(ctx context.Context, q dbtx.Querier, orderID string, actor any,
	driverID *string) error {
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note, driver_id)
		VALUES ($1, $2, $2, $3, $4, $5)`, orderID, StCancelled, actor,
		MerchantLateCancelNote, driverID); err != nil {
		return err
	}
	var owner string
	var number int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(m.owner_user_id::text, ''), o.number
		  FROM orders o JOIN merchants m ON m.id = o.merchant_id
		 WHERE o.id = $1`, orderID).Scan(&owner, &number); err != nil {
		return err
	}
	return notifications.NotifyUserTx(ctx, q, notifications.Input{
		UserID: owner, Kind: notifications.KindOrder,
		Title:  MerchantLateCancelTitle,
		Body:   "#" + itoa(number),
		Entity: "order", EntityID: orderID, Href: "/merchant/deliveries",
		Apps: []string{notifications.AppMerchant},
	})
}

// merchantDeliveryUnpaidNote **سطرُ السجلّ حين يُردّ القبولُ التلقائيُّ لرصيدٍ لا يكفي.**
const merchantDeliveryUnpaidNote = "محفظةُ المتجر لا تكفي أجرةَ التوصيلة — تبقى معلَّقةً حتّى يشحن أو يرفضها المكتب"

// flagMerchantDeliveryUnpaid **يُخبر المكتبَ مرّةً واحدةً** أنّ توصيلةً معلَّقةً لا يكفي متجرَها
// رصيدُه (قرارُ المالك ٢٠٢٦-١٠-١٥) — **سطرٌ في السجلّ وتنبيه**، ولا يتكرّر في كلّ جولة.
func (s *Service) flagMerchantDeliveryUnpaid(ctx context.Context, orderID string) {
	tag, err := s.db.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, note)
		SELECT o.id, o.status, o.status, $2 FROM orders o
		 WHERE o.id = $1 AND NOT EXISTS (
		       SELECT 1 FROM order_events e WHERE e.order_id = o.id AND e.note = $2)`,
		orderID, merchantDeliveryUnpaidNote)
	if err != nil || tag.RowsAffected() == 0 || s.notify == nil {
		return
	}
	var number int64
	_ = s.db.QueryRow(ctx, `SELECT number FROM orders WHERE id = $1`, orderID).Scan(&number)
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind:     notifications.KindOrder,
		Title:    "توصيلةٌ معلَّقة: محفظةُ المتجر لا تكفي",
		Body:     "#" + itoa(number) + " — " + merchantDeliveryUnpaidNote,
		Entity:   "order",
		EntityID: orderID,
		Href:     "/dashboard/orders",
	})
}

// alertMerchantLateCancel **تنبيهُ المكتب بعد التثبيت** — ليقرّر أيعوّض السائق.
func (s *Service) alertMerchantLateCancel(ctx context.Context, o *Order) {
	if s.notify == nil || o == nil {
		return
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind:     notifications.KindOrder,
		Title:    "متجرٌ ألغى توصيلةً بعد وصول السائق",
		Body:     "#" + itoa(o.Number) + " — " + o.MerchantName,
		Entity:   "order",
		EntityID: o.ID,
		Href:     "/dashboard/orders",
	})
}
