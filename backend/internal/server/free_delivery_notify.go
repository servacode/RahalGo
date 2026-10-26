package server

import (
	"context"
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// ══════════════════════════════════════════════════════════════════════
// **«مبروك، أخذتَ توصيلتك المجانيّة»** (قرارُ المالك ٢٠٢٦-١٠-٢٦)
// ══════════════════════════════════════════════════════════════════════
//
// «بدنا نضيف إشعار للزبون اللي يطلب ضمن العرض أوّل مرّة: مبروك أخذت توصيلتك المجانيّة، خلّيك
// معنا دائماً، في عنّا عروض.» **مرّةً لكلّ إعفاء** (`notified_at`)، ولزبونٍ لا لمتجر («لدي
// توصيلة» صاحبُها متجر)، **ولا لطلبٍ أُلغي** — فحقُّه يعود ولا يُبشَّر بما لم ينله.

// SweepFreeDeliveryNotify **يبشّر كلَّ من نال توصيلةً مجانيّةً ولم يُبشَّر** — ويردّ كم بُشّر.
func (s *Server) SweepFreeDeliveryNotify(ctx context.Context) int {
	rows, err := s.pg.Query(ctx, `
		UPDATE free_delivery_claims c SET notified_at = now()
		  FROM orders o
		 WHERE o.id = c.order_id AND c.notified_at IS NULL
		   AND c.claimant NOT LIKE 'm:%'
		   AND o.customer_id IS NOT NULL
		   AND o.status NOT IN ('cancelled', 'rejected')
		RETURNING o.customer_id::text, o.id::text, o.number`)
	if err != nil {
		s.logger.Warn("free delivery notify sweep", "err", err)
		return 0
	}
	type hit struct {
		customer, order string
		number          int64
	}
	var hits []hit
	for rows.Next() {
		var h hit
		if rows.Scan(&h.customer, &h.order, &h.number) == nil {
			hits = append(hits, h)
		}
	}
	rows.Close()
	for _, h := range hits {
		s.notify.Notify(ctx, notifications.Input{
			UserID: h.customer, Kind: notifications.KindOrder,
			Title:  notifTitles.freeDeliveryClaimed,
			Body:   notifTitles.freeDeliveryClaimedBody,
			Entity: "order", EntityID: h.order, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		})
	}
	return len(hits)
}

// RunFreeDeliveryNotify **نبضةٌ دوريّة** — دقيقةٌ تكفي ليصل الخبرُ والطلبُ طازج.
func (s *Server) RunFreeDeliveryNotify(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepFreeDeliveryNotify(ctx)
		}
	}
}

// handleMyFreeDelivery **ما يستحقّه الزبونُ من عرض التوصيل المجانيّ الآن** — بأنواعه.
//
// **كانت شاشةُ المشتريات والأمانة تعرض الأجرةَ كاملةً قبل الطلب** والإعفاءُ يقع بعد تثبيتها —
// فيظنّ الزبونُ العرضَ خدعةً ولا يطلب. **فالشاشةُ تسأل هنا وتقول «مجانيّة» قبل أن يطلب.**
func (s *Server) handleMyFreeDelivery(w http.ResponseWriter, r *http.Request) {
	httpx.JSON(w, http.StatusOK, s.orders.FreeDeliveryStatus(r.Context(), userIDFrom(r)))
}
