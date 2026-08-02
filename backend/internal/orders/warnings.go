package orders

// إنذاراتُ المتجر — **ما يُعدّ لا ما يُقرأ ويُنسى.**
//
// # المسألة
//
// المتجرُ الذي لا يسلّم البضاعةَ للسائق كان يُرسَل إليه إشعارٌ ويمضي.
// **والإشعارُ يُقرأ ويُنسى، ولا يُعدّ** — فلا يُعرف كم مرّةً وقع منه ذلك.
//
// **وعدُّ المخالفات كان يرى نصفَ الصورة**: يعدّ ما ألغاه بيده، **ولا يعدّ ما
// أفشله بامتناعه.** والثاني أسوأ: في الإلغاء يعرف الزبونُ باكراً، **وفي
// الامتناع يكون السائقُ قد قاد والزبونُ قد انتظر** — ثمّ لا شيء.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// warnMerchantOnFault يُسجّل إنذاراً حين يفشل الطلبُ بذنب المتجر.
//
// **يُنادى بعد الإيداع لا داخله**: الإنذارُ سجلٌّ لا مال، **وتعثّرُه يجب ألّا
// يردّ فشلاً وقع فعلاً** — وطلبٌ رُدَّ إفشالُه لأن الإنذار تعثّر يترك السائقَ
// واقفاً عند بابٍ مغلق.
func (s *Service) warnMerchantOnFault(ctx context.Context, orderID, fault, reason string) {
	if fault != FaultMerchant {
		return
	}
	var merchantID, name string
	var ownerID *string
	if err := s.db.QueryRow(ctx, `
		SELECT m.id, m.name, m.owner_user_id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&merchantID, &name, &ownerID); err != nil {
		return
	}
	// **والفهرسُ الفريد يمنع التكرار** — لا فحصٌ قبله.
	if _, err := s.db.Exec(ctx, `
		INSERT INTO merchant_warnings (merchant_id, reason, order_id)
		VALUES ($1, $2, $3) ON CONFLICT (order_id) WHERE order_id IS NOT NULL DO NOTHING`,
		merchantID, reason, orderID); err != nil {
		s.logger.Error("الإنذارات: تعذّر التسجيل", "merchant", merchantID, "error", err)
		return
	}

	// **ويعلم به صاحبُه.**
	//
	// إنذارٌ يُسجَّل ولا يُقال عقوبةٌ تُفاجئ: يُحظر المتجرُ يوماً **ولم يكن
	// يعلم أن عليه شيئاً.** ومن أُنذر مرّةً يصحّح، **ومن لم يُنذر لا يصحّح.**
	if s.notify != nil && ownerID != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: *ownerID, Kind: notifications.KindOrder,
			Title: t.warningIssued, Body: warningText(reason),
			Entity: "merchant", EntityID: merchantID, Href: "/portal/reviews",
		})
	}
	s.enforceMerchantViolations(ctx, orderID)
}

// warningText نصُّ الإنذار بلفظه — والرمزُ يبقى للعدّ.
func warningText(reason string) string {
	switch reason {
	case "merchant_closed":
		return "كان متجرُك مغلقاً والسائقُ عنده"
	case "merchant_refused":
		return "رُفض تسليمُ الطلب للسائق"
	case "merchant_not_ready":
		return "لم يكن الطلبُ جاهزاً والسائقُ ينتظر"
	case "order_unknown":
		return "لم يُعرَف الطلبُ عند الاستلام"
	}
	return "إنذارٌ على طلب"
}

// ManualWarnings عددُ الإنذارات اليدوية داخل النافذة وبعد آخر عفو.
//
// **واليدويةُ وحدَها هنا**: ما نشأ عن طلبٍ يُعدّ من الطلبات نفسِها في
// `MerchantViolations` — **ولو عُدّ من الموضعين لَحُسب مرّتين**، فيُحظر المتجرُ
// على نصف ما استحقّ.
func (s *Service) ManualWarnings(ctx context.Context, q wallet.Querier, merchantID string) (int, error) {
	days := int64(30)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "merchants.cancel_ban_days"); v > 0 {
			days = v
		}
	}
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM merchant_warnings w
		JOIN merchants m ON m.id = w.merchant_id
		WHERE w.merchant_id = $1
		  AND w.order_id IS NULL
		  AND w.created_at > now() - make_interval(days => $2::int)
		  AND (m.violations_cleared_at IS NULL OR w.created_at > m.violations_cleared_at)`,
		merchantID, days).Scan(&n)
	return n, err
}
