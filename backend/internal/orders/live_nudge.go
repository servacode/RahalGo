package orders

// ══════════════════════════════════════════════════════════════════════
// **«لازم يوصلنا موقع مباشر» — المحرّك يذكّر الزبون لحاله** (قرار المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// قيس على السيرفر الحقيقي: من ١٥ طلب صار «بالطريق» بيوم، ٥ بس وصل فيهن موقع مباشر. الباقي: التطبيق
// كان مسكّر وقت «بالطريق»، أو الإذن مرفوض، أو جوال بلا خدمات غوغل. **والكابتن ما كان يطلب.**
// فصار المحرّك يطلب بنفسه — بنبضة الراصد (كل ٣٠ ث):
//
//	بعد `orders.live_nudge_push_sec` (٦٠) بالطريق وما في موقع حيّ  ←  إشعار للزبون، و`live_requested_at`
//	    يتجدّد فتطبيقه يعيد السؤال أو يبلّش المشاركة التلقائيّة لما ينفتح (متل زر الكابتن — **وما بينحسب
//	    من مرّاته التلاتة**). مرّة وحدة للطلب (`live_nudge_push_at`).
//	بعد `orders.live_nudge_wa_sec` (١٢٠) وما وصل ولا موقع من وقت «بالطريق»  ←  رسالة واتساب: «ردّ على
//	    هالرسالة بموقعك المباشر» — **بلا رابط** (قرار المالك ٢٠٢٦-١١-٠٦: الناس بتخاف من الروابط)،
//	    والبوت بيلتقط الردّ (`server.waCustomerLive`). مرّة وحدة للطلب (`live_nudge_wa_at`) — **والرقم نفسه
//	    رقم رموز التسجيل، فالقلّة حماية من الحظر.**
//
// والمفتاح `orders.live_nudge_enabled` بيطفيه.
//
// **وتعديل ٢٠٢٦-١١-١٩ (قرار المالك): رسالةُ الواتساب انشالت** — الزبون طالب عالتطبيق وعنوانو محدّد،
// والإشعار بيكفي. (فوق: وصفُها لتاريخ القرار.)

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// liveNudgeStatuses **من «بالطريق» لحد الباب** — قبلها الكابتن لسا ما طلع لعند الزبون.
var liveNudgeStatuses = []string{StOnTheWay, StAtDropoff}

// SweepLiveNudgeOnce **جولةٌ واحدة** — والراصدُ يناديها كلّ نبضة.
//
// **ورسالةُ الواتساب انشالت** (قرارُ المالك ٢٠٢٦-١١-١٩): كانت تنبعت لطلبات التطبيق بس (طلبُ الواتساب
// ما إلو زبون عالتطبيق فما بيشمله التذكير)، وطلبُ التطبيق عنوانو محدّد وبيكفيه إشعارُ التطبيق.
// فـ`orders.live_nudge_wa_sec` ما عاد إلو شغل، وردُّ الزبون بموقعه عالواتساب لسا بينقبل لو بعته لحالو.
func (s *Service) SweepLiveNudgeOnce(ctx context.Context) {
	if s.settings == nil || !s.settings.GetBool(ctx, "orders.live_nudge_enabled") {
		return
	}
	pushSec := s.settings.GetInt(ctx, "orders.live_nudge_push_sec")
	// **و«بالطريق» هو الانتقالُ الحقيقيّ** — سطورُ السجلّ اللي بتضل بنفس الحالة (متل سطر التذكير نفسه)
	// ما بتنحسب «بالطريق» جديد.
	rows, err := s.db.Query(ctx, `
		SELECT o.id::text, o.number, o.customer_id::text
		  FROM orders o
		  JOIN LATERAL (SELECT max(e.created_at) AS at FROM order_events e
		                 WHERE e.order_id = o.id AND e.to_status = 'on_the_way'
		                   AND e.from_status IS DISTINCT FROM e.to_status) ow ON true
		  LEFT JOIN order_live_locations l ON l.order_id = o.id
		 WHERE o.status = ANY($1) AND o.closed_at IS NULL AND o.kind = ANY($3)
		   AND o.customer_id IS NOT NULL AND o.live_nudge_push_at IS NULL
		   AND ow.at IS NOT NULL AND ow.at < now() - make_interval(secs => $2)
		   AND (l.at IS NULL OR l.at < now() - interval '60 seconds')
		 ORDER BY ow.at LIMIT 50`, liveNudgeStatuses, pushSec, liveKinds)
	if err != nil {
		s.logger.Warn("تذكير الموقع المباشر: القراءة سقطت", "error", err)
		return
	}
	type due struct {
		id, customer string
		number       int64
	}
	var list []due
	for rows.Next() {
		var d due
		if err := rows.Scan(&d.id, &d.number, &d.customer); err != nil {
			rows.Close()
			s.logger.Warn("تذكير الموقع المباشر: القراءة سقطت", "error", err)
			return
		}
		list = append(list, d)
	}
	rows.Close()
	for _, d := range list {
		s.liveNudgePush(ctx, d.id, d.customer, d.number)
	}
}

func (s *Service) liveNudgePush(ctx context.Context, orderID, customerID string, number int64) {
	tag, err := s.db.Exec(ctx, `
		UPDATE orders SET live_nudge_push_at = now(), live_requested_at = now(), updated_at = now()
		 WHERE id = $1 AND live_nudge_push_at IS NULL`, orderID)
	if err != nil || tag.RowsAffected() != 1 {
		return
	}
	_, _ = s.db.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, note, driver_id)
		SELECT id, status, status, 'ما وصل موقع مباشر — انبعت للزبون تذكير', driver_id FROM orders WHERE id = $1`, orderID)
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
	}
	if s.notify != nil {
		s.notify.Notify(ctx, notificationLiveNudge(customerID, orderID, number))
	}
}

func notificationLiveNudge(customerID, orderID string, number int64) notifications.Input {
	return notifications.Input{
		UserID: customerID, Kind: notifications.KindOrder,
		Title:  t.liveNudge,
		Body:   fmt.Sprintf("#%d", number),
		Entity: "order", EntityID: orderID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	}
}

// LiveStats **الموقعُ المباشر اليوم** (بتوقيت دمشق) — كم طلباً صار بالطريق، وبكم وصل موقعٌ ومن وين،
// وكم ذُكِّر. **سطرٌ بلوحة الطلبات** يقيس إن كانت التصليحات ترفع الرقم.
type LiveStats struct {
	OnTheWay     int `json:"on_the_way"`
	WithLive     int `json:"with_live"`
	FromApp      int `json:"from_app"`
	FromWhatsApp int `json:"from_whatsapp"`
	NudgedPush   int `json:"nudged_push"`
	NudgedWA     int `json:"nudged_wa"`
}

// LiveStatsToday **أرقامُ اليوم** — من علامات الطلب لا من صفّ النقطة (يُمحى عند الإيقاف).
func (s *Service) LiveStatsToday(ctx context.Context) (LiveStats, error) {
	var st LiveStats
	err := s.db.QueryRow(ctx, `
		WITH t AS (
		  SELECT o.* FROM orders o
		   WHERE o.kind = ANY($1) AND o.created_at > now() - interval '36 hours'
		     AND EXISTS (SELECT 1 FROM order_events e WHERE e.order_id = o.id AND e.to_status = 'on_the_way'
		                   AND e.created_at >= (date_trunc('day', now() AT TIME ZONE 'Asia/Damascus') AT TIME ZONE 'Asia/Damascus'))
		)
		SELECT count(*), count(live_first_at), count(*) FILTER (WHERE live_first_source = 'app'),
		       count(*) FILTER (WHERE live_first_source = 'whatsapp'),
		       count(live_nudge_push_at), count(live_nudge_wa_at)
		  FROM t`, liveKinds).Scan(&st.OnTheWay, &st.WithLive, &st.FromApp, &st.FromWhatsApp, &st.NudgedPush, &st.NudgedWA)
	return st, err
}
