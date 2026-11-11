package server

// ══════════════════════════════════════════════════════════════════════
//  **تذكيرُ الصباح بالأصناف بلا صور** (قرارُ المالك ٢٠٢٦-١١-١١)
// ══════════════════════════════════════════════════════════════════════
//
// كلَّ صباحٍ نحو العاشرة بدمشق: صاحبُ المتجر الذي في متجره الفعّال أصنافٌ بلا
// صور يصله إشعارٌ واحد «عندك X صنف بلا صورة». **مرّةً في اليوم على الأكثر**
// (`merchant_photo_nudge.last_day` يُحجز قبل الإرسال)، **ولا يُرسل إن كان
// العدد صفراً**، **وتُتخطّى المتاجرُ الموقوفةُ والمغلقة** (حالتها ليست `active`
// أو عليها إغلاقُ طوارئ). **ومفتاحُه** `merchants.photo_nudge_enabled`.

import (
	"context"
	"strconv"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// photoNudgeHour ساعةُ الإرسال بتوقيت دمشق.
const photoNudgeHour = 10

// RunMerchantPhotoNudge نبضةٌ كلَّ دقيقة — تُرسل داخلَ ساعة الإرسال فقط.
func (s *Server) RunMerchantPhotoNudge(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			s.merchantPhotoNudgeTick(ctx, now)
		}
	}
}

func (s *Server) merchantPhotoNudgeTick(ctx context.Context, now time.Time) int {
	if s.pg == nil || s.settings == nil || !s.settings.GetBool(ctx, "merchants.photo_nudge_enabled") {
		return 0
	}
	loc, err := time.LoadLocation("Asia/Damascus")
	if err != nil {
		loc = time.FixedZone("Damascus", 3*3600)
	}
	local := now.In(loc)
	if local.Hour() != photoNudgeHour {
		return 0
	}
	day := local.Format("2006-01-02")
	sent := 0
	// **كلُّ الأصحاب لا أوّلُ ألفٍ** — دفعةٌ بعد دفعة، ومن حُجز يومُه لا يعود.
	for round := 0; round < 100; round++ {
		rows, err := s.pg.Query(ctx, `
			SELECT m.owner_user_id::text, count(*)::int
			FROM merchants m
			JOIN menu_items i ON i.merchant_id = m.id AND i.image_media_id IS NULL
			JOIN users u ON u.id = m.owner_user_id AND u.status = 'active'
			WHERE m.status = 'active' AND NOT m.emergency_closed
			  AND NOT EXISTS (SELECT 1 FROM merchant_photo_nudge n
			                   WHERE n.user_id = m.owner_user_id AND n.last_day = $1::date)
			GROUP BY m.owner_user_id
			ORDER BY m.owner_user_id
			LIMIT 500`, day)
		if err != nil {
			s.logger.Warn("تذكيرُ الصور: تعذّرت القراءة", "err", err)
			return sent
		}
		type due struct {
			id string
			n  int
		}
		var list []due
		for rows.Next() {
			var d due
			if rows.Scan(&d.id, &d.n) == nil && d.n > 0 {
				list = append(list, d)
			}
		}
		rows.Close()
		if len(list) == 0 {
			break
		}
		for _, d := range list {
			// **يُحجز اليومُ قبل الإرسال** — ومن سبقته نبضةٌ أخرى لا يُرسَل له.
			tag, err := s.pg.Exec(ctx, `
				INSERT INTO merchant_photo_nudge (user_id, last_day, updated_at)
				VALUES ($1::uuid, $2::date, now())
				ON CONFLICT (user_id) DO UPDATE SET last_day = EXCLUDED.last_day, updated_at = now()
				 WHERE merchant_photo_nudge.last_day <> EXCLUDED.last_day`, d.id, day)
			if err != nil || tag.RowsAffected() == 0 {
				continue
			}
			if s.notify != nil {
				s.notify.Notify(ctx, notifications.Input{
					UserID: d.id, Kind: notifications.KindAccount,
					Title: "عندك " + strconv.Itoa(d.n) + " صنف بلا صورة 📸",
					Body:  "الأصناف اللي إلها صور بتنباع أكتر",
					Apps:  []string{notifications.AppMerchant},
				})
			}
			sent++
		}
	}
	if sent > 0 {
		s.logger.Info("تذكيرُ الصور: أُرسل", "day", day, "count", sent)
	}
	return sent
}

// MerchantPhotoNudgeTickForTest نبضةٌ بوقتٍ يُمرَّر — للفحوص وحدَها.
func (s *Server) MerchantPhotoNudgeTickForTest(ctx context.Context, now time.Time) int {
	return s.merchantPhotoNudgeTick(ctx, now)
}
