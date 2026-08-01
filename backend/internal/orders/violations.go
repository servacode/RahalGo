package orders

// مخالفاتُ المتجر — عدُّ الإلغاء وحظرُ من يُكثره.
//
// # لماذا
//
// **الثقةُ تُبنى مرّةً وتُهدَم مرّة.** وزبونٌ أُلغي طلبُه مرّتين لا يعود، ولا
// يعود وحده — يقول لمن حوله. **والمنصةُ تتحمّل الخسارة لأنها لا تريد أن تفقد
// ثقتَها مقابل طلبٍ أو طلبين**، لكنّ متجراً يجعل ذلك عادةً يُكلّفها ما هو أغلى
// من الطلبات: سمعتَها.
//
// # وثلاثةُ حدودٍ تمنع الظلم
//
//   - **نافذةٌ زمنية لا مدى الحياة.** متجرٌ سلّم ثلاثمئة طلبٍ في سنة وألغى ستّاً
//     ليس متجراً سيّئاً. والعدُّ التراكميّ يحظره يوماً حتماً.
//   - **من ألغى يُسجَّل لا يُخمَّن.** `cancelled` يصل إليها الزبونُ والعملياتُ
//     والأدمن، **وحسبانُها كلَّها على المتجر يحظر بريئاً**.
//   - **والوضعُ اختيارٌ**: آليٌّ لمن أراد، ويدويٌّ لمن أراد أن يرى قبل أن يبطش.
//     **وحظرٌ يقع ليلاً بلا من يراه يُفقد المنصةَ متجراً ويُفقد المتجرَ رزقاً.**
//
// # والعفوُ خطٌّ زمنيّ لا ممحاة
//
// المخالفاتُ طلباتٌ وقعت فعلاً، ولا يجوز محوُ وقوعها. **فالعفوُ يُعدّ ما بعده
// وحده** — ويبقى الماضي مقروءاً لمن يسأل «كم مرّةً سامحناه؟».

import (
	"context"
	"fmt"

	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// MerchantViolations عدُّ مخالفات متجرٍ داخل النافذة وبعد آخر عفو.
func (s *Service) MerchantViolations(ctx context.Context, q wallet.Querier, merchantID string) (int, error) {
	days := int64(30)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "merchants.cancel_ban_days"); v > 0 {
			days = v
		}
	}
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		WHERE o.merchant_id = $1
		  AND o.ended_by = 'merchant'
		  AND o.status IN ('rejected', 'cancelled')
		  AND o.closed_at > now() - make_interval(days => $2::int)
		  AND (m.violations_cleared_at IS NULL OR o.closed_at > m.violations_cleared_at)`,
		merchantID, days).Scan(&n)
	return n, err
}

// enforceMerchantViolations يُحظر المتجرَ إن تجاوز العتبةَ والوضعُ آليّ.
//
// **يُنادى بعد الإيداع لا داخله**: الحظرُ قرارٌ قائمٌ بذاته، وتعثّرُه يجب ألّا
// يُلغي إلغاءً وقع فعلاً — **وطلبٌ أُلغي ثم رُدَّ إلغاؤه لأن الحظر تعثّر يترك
// الزبونَ ينتظر طعاماً لن يأتي.**
func (s *Service) enforceMerchantViolations(ctx context.Context, orderID string) {
	if s.settings == nil {
		return
	}
	var merchantID string
	if err := s.db.QueryRow(ctx,
		`SELECT merchant_id FROM orders WHERE id = $1`, orderID).Scan(&merchantID); err != nil {
		return
	}

	n, err := s.MerchantViolations(ctx, s.db, merchantID)
	if err != nil {
		s.logger.Error("المخالفات: تعذّر العدّ", "merchant", merchantID, "error", err)
		return
	}
	limit := int(s.settings.GetInt(ctx, "merchants.cancel_ban_count"))
	if limit <= 0 || n < limit {
		return
	}

	// **الإشارةُ تُرفع في الوضعين** — الفرقُ في اليد لا في العين. ومن اختار
	// «يدويّ» يريد أن يعلم، لا أن يبقى في العمى.
	auto := s.settings.GetString(ctx, "merchants.cancel_ban_mode", "manual") == "auto"
	if s.notify != nil {
		var name string
		_ = s.db.QueryRow(ctx, `SELECT name FROM merchants WHERE id = $1`,
			merchantID).Scan(&name)
		title := t.violationsWarn
		if auto {
			title = t.violationsBanned
		}
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind:  notifications.KindOrder,
			Title: title,
			Body: fmt.Sprintf("%s — %d %s %d", name, n,
				"مخالفة خلال النافذة، والحدّ", limit),
			Entity: "merchant", EntityID: merchantID,
			Href: "/dashboard/merchants/" + merchantID,
		})
	}

	if !auto {
		return
	}
	// **suspended لا inactive**: الثانيةُ يملكها المتجرُ فيرفع الحظرَ عن نفسه.
	if _, err := s.db.Exec(ctx,
		`UPDATE merchants SET status = 'suspended' WHERE id = $1 AND status <> 'suspended'`,
		merchantID); err != nil {
		s.logger.Error("المخالفات: تعذّر الحظر", "merchant", merchantID, "error", err)
		return
	}
	s.logger.Warn("حُظر متجرٌ آلياً لكثرة الإلغاء",
		"merchant", merchantID, "violations", n, "limit", limit)
}
