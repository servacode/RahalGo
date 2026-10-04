package server

// ══════════════════════════════════════════════════════════════════════
// **إشعارُ الإطلاق والتغطية — بالسلطةِ الكاملةِ للهرم** (Batch 3d)
// ══════════════════════════════════════════════════════════════════════
//
// **مصدرا اشتراكٍ متمايزان** (`coverage_requests`):
//   - `service_interest` — «أشعرني عند وصول رحال غو إلى مدينتي» (مفتاحُ المدينة).
//   - `coverage_request` — «اطلب تغطية منطقتي» (نقطةٌ جغرافيّةٌ محفوظة).
//
// **والحكمُ للأبِ لا للابن** (تصحيحُ المالك): لا يُشعَر أحدٌ إلّا حين يصير موضعُه
// **مُغطّىً فعليّاً تحت الهرم** — محافظةٌ نشطةٌ ومدينةٌ نشطةٌ ومنطقةٌ نشطةٌ تحويه.
// **ووقتُ المنطقة يُتجاهَل**: نافذةُ ساعةٍ مغلقةٍ ليست «لا تغطية». **ويُشتَقّ من
// دالّتَي `CoverableAt`/`CityEffectivelyLaunched` نفسِهما اللتين تقرؤهما الإتاحةُ
// والإنشاء** — فلا تفترق الإتاحةُ عن الإنشاء عن الإشعار.
//
// **ومرّةً واحدةً ولو تكرّر حفظُ الأدمن**: `notified_at` هو القفل — يُختَم الصفُّ
// أوّلاً (`WHERE notified_at IS NULL` ذرّيّاً) **ثمّ** يُرسَل، فلا يُشعَر أحدٌ مرّتين.
//
// **ولا يُنادى عند الحفظ بعد اليوم** (قرارُ المالك ٢٠٢٦-١٠-٠٤): الإبلاغُ بزرّ
// «بلّغ المنتظرين الآن» في «طلبات التوسّع» (`expansion.go`) — **قد تُرسم التغطيةُ
// قبل تجهيز السائقين.** ويبقى بابُ QA على التجهيز يُطلقه كضغطةِ الزرّ.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

const (
	msgCityLaunched  = "وصل رحال غو إلى مدينتك 🎉 يمكنك الآن استكشاف الخدمات المتاحة."
	msgAreaCoverable = "أصبح التوصيل متاحًا في منطقتك 🎉 افتح رحال غو وابدأ طلبك."
)

// notifyCityLaunch **يُشعر مشترِكي «أشعرني» لمدينةٍ صارت مُطلَقةً فعليّاً.**
//
// **ولا يُرسَل إن لم تُطلَق فعليّاً** (مدينةٌ نشطةٌ تحت محافظةٍ مُطفأةٍ ليست
// مُطلَقةً بعد). **وآمنٌ لكلّ حفظٍ متكرّر** — القفلُ `notified_at`.
func (s *Server) notifyCityLaunch(ctx context.Context, cityID string) int {
	if s.notify == nil || s.orders == nil {
		return 0
	}
	launched, err := s.orders.CityEffectivelyLaunched(ctx, s.pg, cityID)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر فحصُ إطلاق المدينة", "city", cityID, "error", err)
		return 0
	}
	if !launched {
		return 0
	}
	target := "city:" + cityID
	rows, err := s.pg.Query(ctx, `
		SELECT user_id::text FROM coverage_requests
		 WHERE kind = 'service_interest' AND active AND status <> 'rejected' AND user_id IS NOT NULL
		   AND target_key = $1 AND notified_at IS NULL`, target)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر جلبُ مشترِكي المدينة", "city", cityID, "error", err)
		return 0
	}
	var uids []string
	for rows.Next() {
		var uid string
		if rows.Scan(&uid) == nil {
			uids = append(uids, uid)
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return 0
	}
	sent := 0
	for _, uid := range uids {
		// **الختمُ هو القفل** — من ختمتُ صفَّه (RowsAffected=1) أُرسل إليه وحدَه.
		tag, uerr := s.pg.Exec(ctx, `
			UPDATE coverage_requests SET notified_at = now(), updated_at = now()
			 WHERE user_id = $1::uuid AND kind = 'service_interest'
			   AND target_key = $2 AND active AND status <> 'rejected'
			   AND notified_at IS NULL`, uid, target)
		if uerr != nil || tag.RowsAffected() != 1 {
			continue
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: uid, Kind: notifications.KindAccount,
			Title: msgCityLaunched, Entity: "city_launch", EntityID: cityID,
			Apps: []string{notifications.AppCustomer},
		})
		sent++
	}
	return sent
}

// notifyGovernorateLaunch **محافظةٌ صارت نشطةً ⇒ تُشعَر مدنُها النشطةُ.**
//
// **بلا تعارضٍ مع الحكم**: `notifyCityLaunch` يُعيد فحصَ الإطلاق الفعليّ لكلّ
// مدينةٍ (والمحافظةُ صارت نشطةً للتوّ)، **فمدينةٌ مُطفأةٌ تحت محافظةٍ نشطةٍ لا تُشعَر.**
func (s *Server) notifyGovernorateLaunch(ctx context.Context, govID string) {
	if s.notify == nil || s.orders == nil {
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT c.id::text FROM cities c
		 JOIN districts d ON d.id = c.district_id
		 WHERE d.governorate_id = $1::uuid AND c.active`, govID)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر جلبُ مدن المحافظة", "gov", govID, "error", err)
		return
	}
	var cities []string
	for rows.Next() {
		var id string
		if rows.Scan(&id) == nil {
			cities = append(cities, id)
		}
	}
	rows.Close()
	for _, id := range cities {
		s.notifyCityLaunch(ctx, id)
	}
}

// notifyAreaCoverage **مسحُ طلبات «اطلب تغطية منطقتي» المعلَّقة بعد تغيّرِ منطقة.**
//
// **يُشعَر صاحبُ الطلب فقط إن صارت نقطتُه مُغطّاةً فعليّاً** (`CoverableAt`:
// محافظةٌ+مدينةٌ+منطقةٌ نشطةٌ تحويها هندسيّاً، بلا نظرٍ للساعة). **والنقطةُ التي
// ما زالت خارجَ التغطية النشطة لا تُشعَر.** آمنٌ للتغييرات الجماعيّة — القفلُ `notified_at`.
func (s *Server) notifyAreaCoverage(ctx context.Context) {
	if s.notify == nil || s.orders == nil {
		return
	}
	rows, err := s.pg.Query(ctx, `
		SELECT id::text, user_id::text, ST_Y(at::geometry), ST_X(at::geometry)
		  FROM coverage_requests r
		 WHERE `+expWaitingSQL)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر جلبُ طلبات التغطية", "error", err)
		return
	}
	var list []pendingCoverage
	for rows.Next() {
		var p pendingCoverage
		if rows.Scan(&p.id, &p.uid, &p.lat, &p.lng) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return
	}
	s.sendAreaCoverage(ctx, list)
}

// pendingCoverage طلبُ «أضف منطقتي» ينتظر الإبلاغ.
type pendingCoverage struct {
	id, uid  string
	lat, lng float64
}

// sendAreaCoverage **يُبلّغ من صارت نقطتُه مُغطّاةً فعليّاً** — ويُرجع كم بُلّغ.
//
// **والختمُ قبل الإرسال** (`notified_at IS NULL` ذرّيّاً) — فلا يُبلَّغ
// أحدٌ مرّتين ولو ضُغط الزرُّ مرّتين معاً.
func (s *Server) sendAreaCoverage(ctx context.Context, list []pendingCoverage) int {
	sent := 0
	for _, p := range list {
		coverable, cerr := s.orders.CoverableAt(ctx, s.pg, p.lat, p.lng)
		if cerr != nil || !coverable {
			continue
		}
		tag, uerr := s.pg.Exec(ctx, `
			UPDATE coverage_requests SET notified_at = now(), updated_at = now()
			 WHERE id = $1::uuid AND notified_at IS NULL`, p.id)
		if uerr != nil || tag.RowsAffected() != 1 {
			continue
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: p.uid, Kind: notifications.KindAccount,
			Title: msgAreaCoverable, Entity: "area_coverage", EntityID: p.id,
			Apps: []string{notifications.AppCustomer},
		})
		sent++
	}
	return sent
}
