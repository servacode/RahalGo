package server

// ══════════════════════════════════════════════════════════════════════
// **«الكابتن قريب منك» — من نبضة موقع السائق** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **يُنادى بعد أن يُكتب موضعُ السائق** (`driver_location.go` والدفعة) — **والحكمُ
// والعلامةُ في `orders.NearNoticeDue`**: هنا الحدُّ من اللوحة والإشعارُ وحدَهما.
//
// # ولا يُسقط النبضة
//
// **موضعُ السائق هو ما يُسأل عنه كلَّ لحظة** — **وتنبيهٌ تعثّر لا يردّ خطأً على
// نبضةٍ كُتبت**، فيُعيدها التطبيقُ إلى الأبد.

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// nearNoticeTitle **نصُّ تنبيه القرب** — بنصّ المالك.
const nearNoticeTitle = "الكابتن قريب منك — اطلع لعند الباب أو الشباك ليوصلك أسرع"

// notifyNearCustomers **يُنبَّه زبونُ كلّ طلبٍ دخل كابتنُه حدَّ القرب** — ويردّ كم نُبّه.
//
// **عابرٌ لا محفوظ** (`Transient`): «اطلع الآن» فعلٌ يُنتظَر في لحظته، **وبعد
// التسليم لا معنى له في الصندوق** (قاعدةُ الإشعار، `noise_test.go`).
func (s *Server) notifyNearCustomers(ctx context.Context, driverID string, lat, lng float64) int {
	if s.orders == nil || driverID == "" {
		return 0
	}
	due, err := s.orders.NearNoticeDue(ctx, driverID, lat, lng,
		s.liveSetting(ctx, orders.SettingNearNoticeM), s.liveRule(ctx))
	if err != nil {
		s.logger.Warn("تنبيهُ القرب: تعذّر الحساب", "driver", driverID, "error", err)
	}
	for _, n := range due {
		if s.notify == nil {
			continue
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: n.CustomerID, Kind: notifications.KindOrder,
			Title: nearNoticeTitle, Body: fmt.Sprintf("#%d", n.Number),
			// **ويفتح طلبَه** — كإشعارات تقدّم الطلب (`orders/notify.go`).
			Entity: "order", EntityID: n.OrderID, Href: "/portal/orders",
			Apps:      []string{notifications.AppCustomer},
			Transient: true,
		})
	}
	return len(due)
}
