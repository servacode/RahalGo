package orders

// خزينةُ المنصة — الطرفُ الرابع في كل طلب.
//
// # القاعدة
//
// **لا يُدفع لأحدٍ إلّا وخرج من الخزينة، ولا يدخل مالٌ إلّا ودخلها.** فما خرج
// بلا مصدرٍ لا يُقرأ لاحقاً، ومنصةٌ تعرف ما دفعت ولا تعرف من أين لا تعرف شيئاً.
//
// # وحسبتُها في سطرٍ واحد
//
//	ربحُ الطلب = ما دفعه الزبون − (المتجر + السائق + المندوب)
//
// وهو عمولةُ المنصة زائدَ رسم التوصيل ناقصَ أجر السائق ونصيب المندوب. **ومالُ
// المتجر ليس مصروفاً عليها، هو مالٌ يعبر.**
//
// # ولماذا يُحسب بالطرح لا بالجمع
//
// **الطرحُ لا يكذب.** لو جُمعت الأجزاء (عمولة + توصيل − سائق − مندوب) لَبقيت
// حسبةً ثانيةً بجانب الحسبة الأولى، **وحسبتان تفترقان يوماً**: يُغيَّر نصيبُ
// السائق في موضعٍ ويُنسى في الآخر، فيظهر ربحٌ لا وجود له.
//
// والطرحُ يقرأ **ما قُيّد فعلاً في الدفتر** لا ما كان يُفترض أن يُقيَّد — فإن
// رُفض قيدٌ لنقص رصيد ظهر أثرُه في الربح فوراً.

import (
	"context"

	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// treasuryID حسابُ الخزينة، أو فراغٌ إن لم يُختَر بعد.
//
// **وفراغُه لا يُعطّل تسليماً**: طلبٌ يُرفض لأن المالك لم يفتح صفحةَ الإعدادات
// خسارةٌ لا تُحتمَل. يُسلَّم الطلبُ ويبقى الدفترُ ناقصَ طرفٍ حتى تُختار.
func (s *Service) treasuryID(ctx context.Context) string {
	if s.settings == nil {
		return ""
	}
	return s.settings.GetString(ctx, "platform.treasury_user_id", "")
}

// creditTreasury يقيّد نصيبَ المنصة من طلبٍ مُسلَّم (أو يعكسه بمبلغٍ سالب).
//
// يُنادى **داخل معاملة التسوية** بعد أن تُقيَّد أنصبةُ الأطراف كلِّها — فيقرأ
// ما وقع لا ما نُوي.
func (s *Service) creditTreasury(ctx context.Context, q wallet.Querier, orderID, actorID string, reverse bool) error {
	tid := s.treasuryID(ctx)
	if tid == "" {
		return nil
	}

	// ما دفعه الزبون: من محفظته ونقداً معاً — **والمصدرُ لا يغيّر الربح**،
	// إنما يغيّر أين يجلس النقدُ الآن (وذاك شأنُ صندوق السائق).
	var paid int64
	if err := q.QueryRow(ctx,
		`SELECT wallet_paid + cash_due FROM orders WHERE id = $1`, orderID).
		Scan(&paid); err != nil {
		return err
	}

	// **ما قُيّد فعلاً للأطراف عن هذا الطلب** — لا ما تقول المعادلة إنه يجب
	// أن يُقيَّد. والخزينةُ نفسها مستثناةٌ من الجمع وإلّا حُسبت في نفسها.
	var toParties int64
	if err := q.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0)
		FROM wallet_transactions
		WHERE ref = $1
		  AND kind IN ('merchant_earning', 'driver_earning', 'commission')`,
		orderID).Scan(&toParties); err != nil {
		return err
	}

	profit := paid - toParties
	if reverse {
		profit = -profit
	}
	if profit == 0 {
		return nil
	}
	note := "ربحُ طلبٍ مُسلَّم"
	if reverse {
		note = "عكسُ ربحِ طلبٍ مُسترجَع"
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, profit, "platform_profit",
		orderID, note, &actorID)
	return err
}

// DebitTreasury يخصم من الخزينة مالاً خرج منها خارج تسوية الطلب.
//
// **تعويضُ سائقٍ أو بضاعةٍ لم تُسترَدّ مالٌ من جيب المنصة** — ولو قُيّد للطرف
// وحده لظهرت المنصةُ رابحةً وهي تدفع. وهو ما كان يقع: الأرباحُ جمعُ عمولات،
// **والعمولةُ لا تعرف أن شيئاً خرج.**
func (s *Service) DebitTreasury(ctx context.Context, q wallet.Querier, amount int64, ref, note, actorID string) error {
	tid := s.treasuryID(ctx)
	if tid == "" || amount <= 0 {
		return nil
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, -amount, "platform_profit",
		ref, note, &actorID)
	return err
}
