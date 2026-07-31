package orders

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// إشعارات دورة حياة الطلب.
//
// كان نظام الإشعارات موصولاً بأحداث الإدارة فقط، ودورة الطلب — قلب المنصة —
// لا تُشعر أحداً: الزبون لا يعرف أن طلبه قُبل، والمتجر لا يجد أثراً لطلب جديد
// إن أغلق التبويب، والمندوب لا يعلم بعمولة دخلت محفظته.
//
// المبدأ: **إشعار عند ما يهمّ فعلاً** لا عند كل انتقال. حالات المرور الداخلية
// (dispatching، at_pickup، at_dropoff) لا تُشعر الزبون — يتابعها في خط التقدّم
// الحي. الإشعار لما يغيّر انتظاره أو ماله.

// نصوص إشعارات الطلبات — مجمّعة كي لا تتناثر (كنمط notifTitles في الخادم).
var t = struct {
	newOrderMerchant, newOrderOps            string
	accepted, preparing, onTheWay, delivered string
	rejected, cancelled, failed, refunded    string
	merchantDelivered, commission            string
}{
	newOrderMerchant:  "طلب جديد وصلك",
	newOrderOps:       "طلب جديد في المنصة",
	accepted:          "قبل المتجر طلبك",
	preparing:         "طلبك قيد التحضير",
	onTheWay:          "طلبك في الطريق إليك",
	delivered:         "تم تسليم طلبك",
	rejected:          "اعتذر المتجر عن طلبك",
	cancelled:         "أُلغي طلبك",
	failed:            "تعذّر تسليم طلبك",
	refunded:          "استُرجع مبلغ طلبك",
	merchantDelivered: "سُلّم طلب من متجرك",
	commission:        "عمولة جديدة في محفظتك",
}

// orderParties أطراف الطلب الذين قد يُشعَرون.
type orderParties struct {
	number        int64
	customerID    string
	merchantOwner *string
	merchantName  string
	repID         *string
}

func (s *Service) parties(ctx context.Context, orderID string) (orderParties, error) {
	var p orderParties
	err := s.db.QueryRow(ctx, `
		SELECT o.number, o.customer_id, mm.owner_user_id, mm.name, mm.sales_rep_user_id
		FROM orders o JOIN merchants mm ON mm.id = o.merchant_id
		WHERE o.id = $1`, orderID).
		Scan(&p.number, &p.customerID, &p.merchantOwner, &p.merchantName, &p.repID)
	return p, err
}

// notifyCreated المتجر يعرف بطلب جديد ولو أغلق تبويبه، ومكتب المنصة يتابع الحركة.
func (s *Service) notifyCreated(ctx context.Context, o *Order) {
	if s.notify == nil || o == nil {
		return
	}
	p, err := s.parties(ctx, o.ID)
	if err != nil {
		return
	}
	ref := fmt.Sprintf("#%d", p.number)
	if p.merchantOwner != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *p.merchantOwner, Kind: notifications.KindOrder,
			Title: t.newOrderMerchant, Body: ref,
			Entity: "order", EntityID: o.ID, Href: "/portal",
		})
	}
	s.notify.NotifyRoles(ctx, notifications.OpsDesk, notifications.Input{
		Kind: notifications.KindOrder, Title: t.newOrderOps,
		Body:   ref + " — " + p.merchantName,
		Entity: "order", EntityID: o.ID, Href: "/dashboard/orders",
	})
}

// customerTitles الانتقالات التي تستحق إشعاراً للزبون.
var customerTitles = map[string]string{
	StAccepted:  t.accepted,
	StPreparing: t.preparing,
	StOnTheWay:  t.onTheWay,
	StDelivered: t.delivered,
	StRejected:  t.rejected,
	StCancelled: t.cancelled,
	StFailed:    t.failed,
	StRefunded:  t.refunded,
}

// notifyTransition يُعلم من يخصّه هذا الانتقال. يُستدعى بعد نجاح الإيداع:
// فشل الإشعار لا يُبطل تسليماً وقع فعلاً.
func (s *Service) notifyTransition(ctx context.Context, orderID, to, note string) {
	if s.notify == nil {
		return
	}
	title, worth := customerTitles[to]
	if !worth {
		return // حالة مرور داخلية — يتابعها الزبون في خط التقدّم الحي
	}
	p, err := s.parties(ctx, orderID)
	if err != nil {
		return
	}
	ref := fmt.Sprintf("#%d", p.number)
	body := ref + " — " + p.merchantName
	if note != "" && (to == StRejected || to == StCancelled || to == StFailed) {
		body = ref + " — " + note // السبب أهمّ من اسم المتجر عند الرفض
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: p.customerID, Kind: notifications.KindOrder,
		Title: title, Body: body,
		Entity: "order", EntityID: orderID, Href: "/orders/" + orderID,
	})

	// التسليم يخصّ المتجر أيضاً (اكتمل التزامه)
	if to == StDelivered && p.merchantOwner != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *p.merchantOwner, Kind: notifications.KindOrder,
			Title: t.merchantDelivered, Body: ref,
			Entity: "order", EntityID: orderID, Href: "/portal",
		})
	}
}

// notifyCommission المندوب يعرف بعمولته لحظة قيدها — مصدر دخله لا يُترك للاكتشاف.
func (s *Service) notifyCommission(ctx context.Context, repID, orderID string, amount int64) {
	if s.notify == nil || amount <= 0 {
		return
	}
	p, err := s.parties(ctx, orderID)
	if err != nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: repID, Kind: notifications.KindWallet,
		Title:  t.commission,
		Body:   fmt.Sprintf("#%d — %s", p.number, p.merchantName),
		Entity: "wallet", EntityID: orderID, Href: "/portal/wallet",
	})
}
