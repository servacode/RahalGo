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
func (s *Server) notifyCityLaunch(ctx context.Context, cityID string) {
	if s.notify == nil || s.orders == nil {
		return
	}
	launched, err := s.orders.CityEffectivelyLaunched(ctx, s.pg, cityID)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر فحصُ إطلاق المدينة", "city", cityID, "error", err)
		return
	}
	if !launched {
		return
	}
	target := "city:" + cityID
	rows, err := s.pg.Query(ctx, `
		SELECT user_id::text FROM coverage_requests
		 WHERE kind = 'service_interest' AND active AND user_id IS NOT NULL
		   AND target_key = $1 AND notified_at IS NULL`, target)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر جلبُ مشترِكي المدينة", "city", cityID, "error", err)
		return
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
		return
	}
	for _, uid := range uids {
		// **الختمُ هو القفل** — من ختمتُ صفَّه (RowsAffected=1) أُرسل إليه وحدَه.
		tag, uerr := s.pg.Exec(ctx, `
			UPDATE coverage_requests SET notified_at = now(), updated_at = now()
			 WHERE user_id = $1::uuid AND kind = 'service_interest'
			   AND target_key = $2 AND active AND notified_at IS NULL`, uid, target)
		if uerr != nil || tag.RowsAffected() != 1 {
			continue
		}
		s.notify.Notify(ctx, notifications.Input{
			UserID: uid, Kind: notifications.KindAccount,
			Title: msgCityLaunched, Entity: "city_launch", EntityID: cityID,
			Apps: []string{notifications.AppCustomer},
		})
	}
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
		  FROM coverage_requests
		 WHERE kind = 'coverage_request' AND active AND user_id IS NOT NULL
		   AND notified_at IS NULL`)
	if err != nil {
		s.logger.Error("geo-notify: تعذّر جلبُ طلبات التغطية", "error", err)
		return
	}
	type pending struct {
		id, uid  string
		lat, lng float64
	}
	var list []pending
	for rows.Next() {
		var p pending
		if rows.Scan(&p.id, &p.uid, &p.lat, &p.lng) == nil {
			list = append(list, p)
		}
	}
	rows.Close()
	if rows.Err() != nil {
		return
	}
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
	}
}
