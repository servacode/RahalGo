package server

// ══════════════════════════════════════════════════════════════════════
// **«في N صنف من الأمين بلا قسم — رتّبهن من صفحة مزامنة الأسعار»** (قرار المالك ٢٠٢٦-١١-٠٧)
// ══════════════════════════════════════════════════════════════════════
//
// بعد مزامنة حقيقيّة (مو «جرّب») — من رفع الملف أو من الساعي — إذا ضلّ أصناف بقسم الانتظار:
//
//	١ · **إشعار** لصاحب المتجر (تطبيق المتجر + صندوقه، والضغط بالموقع بيفتح /store/price-sync)
//	    وللإدارة اللي بتدير المتاجر (`merchants.manage`).
//	٢ · **وإذا بعد يومين من الإشعار لسّا في أصناف بالانتظار** ⇒ **تذكير واحد** للاتنين، وبعدها ولا شي.
//
// **والساعي بيزامن كلّ كم دقيقة** — فالإشعار ما بيتكرّر مع كلّ مزامنة: بينبعت إذا ما في إشعار
// قائم (`price_sync_pending_notified_at` فاضي) **أو إذا المزامنة حطّت أصناف جديدة بالانتظار** —
// وهي «مزامنة جديدة تركت أصناف» اللي بتفتح دورة جديدة (إشعار + تذكير واحد).
// **ولمّا يفضى قسم الانتظار** الختمين بينمسحوا، فأوّل مزامنة بعدها بتبلّغ من جديد.
//
// **والختم قبل الإشعار** (`UPDATE … RETURNING`) — طلبان سوا أو إعادة تشغيل ما بيكرّروا.

import (
	"context"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/catalog"
	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// priceSyncRemindAfter **بعد قدّيش من الإشعار بينبعت التذكير الوحيد.**
const priceSyncRemindAfter = 48 * time.Hour

// priceSyncPendingNotice **بعد مزامنة حقيقيّة**: إشعار إذا ضلّ أصناف بالانتظار — ويردّ إذا انبعت.
func (s *Server) priceSyncPendingNotice(ctx context.Context, merchantID string, rep *catalog.SyncReport, now time.Time) bool {
	if rep == nil || rep.DryRun {
		return false
	}
	n, err := s.catalog.PendingSyncCount(ctx, merchantID)
	if err != nil {
		s.logger.Warn("price sync pending count", "err", err, "merchant", merchantID)
		return false
	}
	if n == 0 {
		s.clearPriceSyncStamps(ctx, merchantID)
		return false
	}
	var owner *string
	var name string
	err = s.pg.QueryRow(ctx, `
		UPDATE merchants SET price_sync_pending_notified_at = $2, price_sync_pending_reminded_at = NULL
		 WHERE id = $1::uuid AND (price_sync_pending_notified_at IS NULL OR $3::bool)
		RETURNING owner_user_id::text, name`, merchantID, now, rep.CreatedPending > 0).Scan(&owner, &name)
	if err != nil {
		return false
	}
	s.sendPriceSyncPending(ctx, merchantID, owner, name, n, notifTitles.priceSyncPendingTitle)
	return true
}

// clearPriceSyncStamps **قسم الانتظار فضي** — الدورة خلصت.
func (s *Server) clearPriceSyncStamps(ctx context.Context, merchantID string) {
	_, _ = s.pg.Exec(ctx, `
		UPDATE merchants SET price_sync_pending_notified_at = NULL, price_sync_pending_reminded_at = NULL
		 WHERE id = $1::uuid AND price_sync_pending_notified_at IS NOT NULL`, merchantID)
}

// sendPriceSyncPending **للمتجر وللإدارة** بنفس العدد.
func (s *Server) sendPriceSyncPending(ctx context.Context, merchantID string, owner *string, store string, n int, title string) {
	body := landmarkText(notifTitles.priceSyncPendingBody, "{n}", strconv.Itoa(n))
	if owner != nil && *owner != "" {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *owner, Kind: notifications.KindAccount,
			Title: title, Body: body,
			Entity: "price_sync", EntityID: merchantID, Href: "/store/price-sync",
			Apps: []string{notifications.AppMerchant},
		})
	}
	adminTitle := notifTitles.priceSyncPendingAdminTitle
	if title == notifTitles.priceSyncPendingRemindTitle {
		adminTitle = title
	}
	s.notify.NotifyCaps(ctx, []string{string(authz.MerchantsManage)}, notifications.Input{
		Kind:  notifications.KindAccount,
		Title: adminTitle,
		Body: landmarkText(notifTitles.priceSyncPendingAdminBody,
			"{store}", store, "{n}", strconv.Itoa(n)),
		Entity: "price_sync", EntityID: merchantID,
		Href: "/dashboard/merchants/" + merchantID + "/menu",
	})
}

// RemindPriceSyncPending **جولة التذكير** — تُنادى دوريّاً وبالاختبار مباشرةً، وتردّ كم تذكير انبعت.
// متجر فضي قسم انتظاره بتنمسح ختومه.
func (s *Server) RemindPriceSyncPending(ctx context.Context, now time.Time) (int, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, price_sync_pending_reminded_at IS NULL AND price_sync_pending_notified_at <= $1
		  FROM merchants WHERE price_sync_pending_notified_at IS NOT NULL`, now.Add(-priceSyncRemindAfter))
	if err != nil {
		return 0, err
	}
	type cand struct {
		id  string
		due bool
	}
	var list []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.due); err != nil {
			rows.Close()
			return 0, err
		}
		list = append(list, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	sent := 0
	for _, c := range list {
		n, err := s.catalog.PendingSyncCount(ctx, c.id)
		if err != nil {
			return sent, err
		}
		if n == 0 {
			s.clearPriceSyncStamps(ctx, c.id)
			continue
		}
		if !c.due {
			continue
		}
		var owner *string
		var name string
		if err := s.pg.QueryRow(ctx, `
			UPDATE merchants SET price_sync_pending_reminded_at = $2
			 WHERE id = $1::uuid AND price_sync_pending_reminded_at IS NULL
			   AND price_sync_pending_notified_at <= $3
			RETURNING owner_user_id::text, name`, c.id, now, now.Add(-priceSyncRemindAfter)).Scan(&owner, &name); err != nil {
			continue
		}
		s.sendPriceSyncPending(ctx, c.id, owner, name, n, notifTitles.priceSyncPendingRemindTitle)
		sent++
	}
	return sent, nil
}

// RunPriceSyncReminder **نبضة دوريّة** — والختم بيمنع التكرار، فربع ساعة بتكفي.
func (s *Server) RunPriceSyncReminder(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := s.RemindPriceSyncPending(ctx, time.Now()); err != nil {
				s.logger.Warn("price sync reminder", "err", err)
			}
		}
	}
}
