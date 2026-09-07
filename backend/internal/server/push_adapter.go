package server

import (
	"context"
	"time"

	"github.com/servacode/rahalgo/backend/internal/push"
)

// pushAdapter **يترجم رسالةَ الإشعارات إلى رسالة الدفع.**
//
// **وهو ثمنُ ألّا تستورد `notifications` حزمةَ `push`** — والاتّجاهُ مقصود:
// **منطقُ المنصّة لا يعرف جوجل**، وتبديلُ ناقلٍ يقع هنا وحدَه.
//
// **وموضعُه الخادم** لأنّه المكانُ الذي يعرف الحزمتين معاً — وهو الذي
// يركّب المنصّةَ أصلاً.
type pushAdapter struct{ svc *push.Service }

// Kick **يوقظ عاملَ النقل** — `PF-09`.
//
// **ولا حمولةَ تُنقَل**: **الحقيقةُ في الصفّ لا في وسيطٍ عابر.**
func (a pushAdapter) Kick(ctx context.Context) { a.svc.Kick(ctx) }

// RunPushDeliveryWorker **حلقةُ نقل الإشعارات** — `PF-09`.
//
// **والخادمُ يملك الخدمةَ فيُشغّلها** — كما الكانسُ والراصد.
func (s *Server) RunPushDeliveryWorker(ctx context.Context, interval time.Duration) {
	s.push.RunDeliveryWorker(ctx, interval)
}

// DeliverPushOnce **جولةُ نقلٍ واحدة** — **مِعراضُ الفحص** (`PF-09`).
//
// **ولا مسارَ شبكةٍ لها** — **ومن فتح نقطةً لأجل اختبارٍ فتحها لغيره.**
func (s *Server) DeliverPushOnce(ctx context.Context) (fanned, sent, retried, failed int) {
	return s.push.DeliverOnce(ctx)
}

// PushHealth حالُ النقل — **يُقرأ في الفحص وفي التشخيص.**
func (s *Server) PushHealth(ctx context.Context) (push.PendingSummary, error) {
	return s.push.DeliveryHealth(ctx)
}
