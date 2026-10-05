package server

// **إشعاراتُ الوجبات — تُطلَق من حلقة الحملات نفسِها.** انظر `campaigns/meals.go`.
//
// **ولا تُرسَل والمنصّةُ لا تستقبل** — **وتذكيرٌ بالغداء والطلبُ مغلقٌ
// يفتح التطبيقَ على «قريباً».** فبابُ الإطلاق وحالُ الاستقبال يُسألان قبل كلّ وجبة.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/campaigns"
)

// mealsOnce **جولةُ الوجبات** — تُنادى مع كلّ دورةٍ من عامل الحملات.
func (s *Server) mealsOnce(ctx context.Context) {
	if !s.settings.GetBool(ctx, "meals.enabled") {
		return
	}
	now := s.campaigns.Now()
	var owner string
	// **والمفاتيحُ مكتوبةً كاملةً** — حارسُ الفهرس يقرأ الأسماءَ من الشيفرة.
	for _, k := range [...]struct{ key, at, texts string }{
		{"breakfast", "meals.breakfast_at", "meals.breakfast_texts"},
		{"lunch", "meals.lunch_at", "meals.lunch_texts"},
		{"dinner", "meals.dinner_at", "meals.dinner_texts"},
	} {
		key := k.key
		m := campaigns.Meal{
			Key:   key,
			At:    s.settings.GetString(ctx, k.at),
			Texts: s.settings.GetString(ctx, k.texts),
		}
		day, due := campaigns.MealDue(m, now)
		if !due {
			continue
		}
		title, body, ok := campaigns.MealText(m.Texts, day)
		if !ok || !s.mealsMayRun(ctx) {
			continue
		}
		// **والحملةُ تُنسب لمالك المنصّة** — العمودُ إلزاميٌّ ولا آليَّ في القاعدة.
		if owner == "" {
			if err := s.pg.QueryRow(ctx, `
				SELECT u.id::text FROM users u JOIN user_roles r ON r.user_id = u.id
				 WHERE r.role_code = 'owner_super_admin' ORDER BY u.created_at LIMIT 1`).Scan(&owner); err != nil {
				return
			}
		}
		c, err := s.campaigns.Create(ctx, owner, "meal:"+key+":"+day, campaigns.Input{
			Title: title, Body: body,
			AudienceType: campaigns.AudienceRole, AudienceRef: "customer",
			DestType: campaigns.DestHome,
		})
		if err != nil {
			s.logger.Warn("إشعارُ الوجبة: تعذّر إنشاؤه", "meal", key, "error", err)
			continue
		}
		// **و`Send` لا يلتقط ما أُرسل** — فالدورةُ التالية لا تكرّر.
		_, _ = s.campaigns.Send(ctx, c.ID)
	}
}

// mealsMayRun **أتستقبل المنصّةُ الطلباتِ الآن؟** — بابُ الإطلاق وحالُ الدوام والإيقاف.
func (s *Server) mealsMayRun(ctx context.Context) bool {
	if !s.launchOpen(ctx, launchCustomerOrders) {
		return false
	}
	st, err := s.platform.State(ctx, s.pg)
	return err == nil && st.OrderingAvailable
}
