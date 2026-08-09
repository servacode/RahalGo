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
	"errors"

	"github.com/jackc/pgx/v5"

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
	//
	// **وصار إلى الجدول الموحَّد** (٢٠٢٦-٠٨-٠٩): إنذارٌ واحدٌ لكلّ الأدوار.
	if _, err := s.db.Exec(ctx, `
		INSERT INTO warnings (user_id, merchant_id, role_code, reason, order_id)
		VALUES ($1, $2, 'merchant', $3, $4)
		ON CONFLICT (order_id) WHERE order_id IS NOT NULL DO NOTHING`,
		ownerID, merchantID, reason, orderID); err != nil {
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

// UncountedWarnings عددُ الإنذارات التي لا يعدّها صفُّ الطلب.
//
// **ولا يُعدّ ما عُدَّ**: ما نشأ عن طلبٍ صار مخالفةً يُعدّ من الطلب نفسِه في
// `MerchantViolations` — **ولو عُدّ من الموضعين لَحُسب مرّتين**، فيُحظر المتجرُ
// على نصف ما استحقّ.
//
// **والشرطُ في `warningsWhere` لا هنا** — يُقرأ منه العدُّ والقائمةُ معاً.
func (s *Service) UncountedWarnings(ctx context.Context, q wallet.Querier, merchantID string) (int, error) {
	days := int64(30)
	if s.settings != nil {
		if v := s.settings.GetInt(ctx, "merchants.cancel_ban_days"); v > 0 {
			days = v
		}
	}
	var n int
	err := q.QueryRow(ctx, `
		SELECT count(*)
		FROM warnings w
		JOIN merchants m ON m.id = $1
		`+warningsWhere("$1", "$2", "m"),
		merchantID, days).Scan(&n)
	return n, err
}

// openMerchantClaim يفتح مطالبةً على المتجر بما دفعته المنصةُ بسببه.
//
// # المنصةُ تدفع أوّلاً ثمّ تُطالِب
//
// **السائقُ لا ينتظر نزاعاً ليُقبض له.** ونزاعٌ يستغرق يوماً يترك من قاد
// مشوارَه بلا مقابلٍ يومَه كلَّه — **ومن قاد بلا مقابلٍ مرّةً يتردّد في
// الثانية.**
//
// **والمطالبةُ تُسجَّل على الإنذار نفسِه** لا في جدولٍ ثالث: الإنذارُ يقول
// **ماذا فعل**، والمطالبةُ تقول **كم كلّف**. **وجدولان لواقعةٍ واحدة
// يفترقان** — يُغلق أحدُهما ويبقى الآخر، فيُطالَب متجرٌ بما سُوّي.
//
// **ولا تُخصم هنا**: الخصمُ قرارُ إنسانٍ بعد أن يسمع المتجر. **ومالٌ يخرج من
// محفظةِ متجرٍ قبل أن يُسأل نزاعٌ خُسر قبل أن يُفتح.**
func (s *Service) openMerchantClaim(ctx context.Context, q wallet.Querier,
	orderID string, amount int64) error {
	if amount <= 0 {
		return nil
	}
	// **والإنذارُ يُكتب هنا إن لم يكن كُتب** — فقد يقع التعويضُ قبله.
	// `ON CONFLICT` يجعل الترتيبَ لا يهمّ: **من سبق كتب، ومن تلاه أضاف
	// المطالبة.**
	//
	// **والمبلغُ لم يعد يُكتب هنا** (هجرة `0063`): الإنذارُ سلوكٌ يُعدّ ولا
	// يُسوّى، **والنزاعُ مالٌ يُسوّى ويُغلق** — وهما واقعتان لا واحدة. وبقاءُ
	// المال في صفّ الإنذار **هو ما منع أن يكون للسائق أو الزبون نزاعٌ أصلاً.**
	// **وصار إلى الجدول الموحَّد** (٢٠٢٦-٠٨-٠٩) — وينسب إلى صاحب المتجر.
	//
	// **ومتجرٌ بلا صاحبٍ لا يُنذَر ولا يُسقط المطالبة**: لا حسابَ يقرأ الإنذار،
	// **والمالُ خرج فعلاً فالمطالبةُ قائمةٌ بذاتها.** (ويقع قبل أن يُسنِد
	// المندوبُ صاحباً للمتجر.)
	//
	// **و`nil` لا نصٌّ فارغ**: `warning_id` مفتاحٌ أجنبيّ، **ونصٌّ فارغٌ ليس
	// معرّفاً** فيُردّ القيدُ كلُّه.
	var warningID *string
	err := q.QueryRow(ctx, `
		INSERT INTO warnings (user_id, merchant_id, role_code, reason, order_id)
		SELECT m.owner_user_id, m.id, 'merchant',
		       COALESCE(o.fail_reason, 'merchant_refused'), o.id
		FROM orders o JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1
		ON CONFLICT (order_id) WHERE order_id IS NOT NULL
		DO UPDATE SET reason = warnings.reason
		RETURNING id::text`, orderID).Scan(&warningID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	// **والنزاعُ في جدوله، ومرجعُه إنذارُه.**
	//
	// `ON CONFLICT` على (الطلب · الطرف) يحرس التكرار: **التعويضُ قد يُنادى
	// مرّتين لطلبٍ واحد** (طارئٌ ثمّ فشل)، فيُطالَب المتجرُ مرّتين بالواقعة
	// نفسِها. **ويُجمع لا يُستبدل** — كلفتان وقعتا فعلاً.
	_, err = q.Exec(ctx, `
		INSERT INTO disputes (party_role, merchant_id, order_id, warning_id, reason, amount)
		SELECT 'merchant', o.merchant_id, o.id, $3,
		       COALESCE(o.fail_reason, 'merchant_refused'), $2
		FROM orders o WHERE o.id = $1
		ON CONFLICT (order_id, party_role) WHERE order_id IS NOT NULL
		DO UPDATE SET amount = disputes.amount + $2`,
		orderID, amount, warningID)
	return err
}
