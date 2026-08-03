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
	"time"

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
	// **وما أفشله بامتناعه يُعدّ كما يُعدّ ما ألغاه بيده.**
	//
	// كان الشرطُ `ended_by = 'merchant'` وحدَه — **فمن أغلق بابَه والسائقُ
	// عنده لم يُحسب عليه شيء**: الطلبُ ينتهي `failed` وينهيه السائقُ لا
	// المتجر، **فـ`ended_by` تقول «السائق» والذنبُ للمتجر.**
	//
	// **وهو أسوأُ من الإلغاء لا أهون**: في الإلغاء يعرف الزبونُ باكراً، وفي
	// الامتناع يكون السائقُ قد قاد والزبونُ قد انتظر — **ثمّ لا شيء.**
	//
	// و`fault` هو الفاصل: يُكتب من قائمة الأسباب لا من تقدير أحد.
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id`+violationsWhere,
		merchantID, days).Scan(&n)
	if err != nil {
		return 0, err
	}
	// **والإنذارُ اليدويُّ يُضاف** — متجرٌ رفع أسعارَه عن المتّفق أو أساء إلى
	// سائق **لا طلبَ يشهد عليه**، ومن لا يُعدّ إنذارُه لا يبلغ حدّاً أبداً.
	manual, err := s.ManualWarnings(ctx, q, merchantID)
	return n + manual, err
}

// ViolationRow مخالفةٌ واحدةٌ مقروءة — **طلبٌ وقع، لا رقمٌ في عدّاد.**
type ViolationRow struct {
	Number   *int64    `json:"order_number"`
	Status   string    `json:"status"`
	Reason   string    `json:"reason"`
	Note     string    `json:"note"`
	Manual   bool      `json:"manual"`
	ClosedAt time.Time `json:"closed_at"`
}

// violationsWhere شرطُ المخالفة — **مكتوبٌ مرّةً يُقرأ في العدّ وفي القائمة.**
//
// **ولو نُسخ لَافترقا يوماً**: يُضاف شرطٌ في العدّ فيُقال «٤ مخالفات» وتُعرض
// ثلاث، **فيُقرأ ذلك عطباً في المنصة لا فرقاً في استعلامين.** وهي عائلةُ الخلل
// التي تكرّرت في هذه الجولة سبعَ مرّات.
const violationsWhere = `
	WHERE o.merchant_id = $1
	  AND ((o.ended_by = 'merchant' AND o.status IN ('rejected', 'cancelled'))
	       OR (o.status = 'failed' AND o.fault = 'merchant'))
	  AND o.closed_at > now() - make_interval(days => $2::int)
	  AND (m.violations_cleared_at IS NULL OR o.closed_at > m.violations_cleared_at)`

// MerchantViolationList **أيُّ طلباتٍ هي** — الجوابُ الذي كان ناقصاً.
//
// كانت العملياتُ ترى «٤ مخالفات» **ولا تملك أن ترى أيَّ طلباتٍ هي** (الثغرة
// `G-01`)، ثمّ تُقرّر الحظرَ أو العفوَ على رقمٍ مجرّد. **وقرارٌ يُبنى على عدد
// بلا وقائعَ قرارٌ لا يُراجَع.**
//
// **والإنذاراتُ اليدويّةُ معها في القائمة نفسِها** — لأنّها تُعدّ في العدّاد
// نفسِه. **ومن رأى ثلاثةً وعدّادُه أربعة يظنّ في المنصة خللاً.**
func (s *Service) MerchantViolationList(ctx context.Context, q wallet.Querier, merchantID string) ([]ViolationRow, error) {
	days := int64(30)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "merchants.cancel_ban_days"); v > 0 {
			days = v
		}
	}
	rows, err := q.Query(ctx, `
		SELECT o.number, o.status,
		       COALESCE(NULLIF(o.fail_reason, ''), o.status),
		       COALESCE(o.cancel_reason, ''), false, o.closed_at
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id`+violationsWhere+`
		UNION ALL
		SELECT NULL, 'warning', w.reason, w.note, true, w.created_at
		FROM merchant_warnings w
		JOIN merchants m2 ON m2.id = w.merchant_id
		WHERE w.merchant_id = $1
		  AND w.order_id IS NULL
		  AND w.created_at > now() - make_interval(days => $2::int)
		  AND (m2.violations_cleared_at IS NULL OR w.created_at > m2.violations_cleared_at)
		ORDER BY 6 DESC
		LIMIT 100`, merchantID, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ViolationRow{}
	for rows.Next() {
		var v ViolationRow
		if err := rows.Scan(&v.Number, &v.Status, &v.Reason, &v.Note,
			&v.Manual, &v.ClosedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
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
