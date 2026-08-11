package server

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
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

func (a pushAdapter) SendToUser(ctx context.Context, userID string, m notifications.PushMessage) {
	a.svc.SendToUser(ctx, userID, push.Message{
		Title:  m.Title,
		Body:   m.Body,
		Data:   m.Data,
		Urgent: m.Urgent,
	})
}
