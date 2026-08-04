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

// treasuryID حسابُ الخزينة، أو فراغٌ إن لم تُوسَم محفظةٌ بعد.
//
// # ولماذا من المحفظة لا من الإعدادات
//
// كان مفتاحاً في `app_settings` يحمل معرّفَ مستخدم. **والعمودُ `is_treasury`
// قائمٌ في `wallets` أيضاً** — فصار للخزينة مصدران: مفتاحٌ يقول من هي،
// وعمودٌ يقول من هي. **وإن افترقا فأيُّهما يُصدَّق؟**
//
// **والعمودُ أصدقُ**: عليه الفهرسُ الفريد الذي يمنع خزينتين، وعليه شرطُ
// `balance >= 0 OR is_treasury` الذي يسمح لها وحدَها بالسالب. **فالقاعدةُ
// تحرسه ولا تحرس المفتاح.**
//
// **وصفةٌ في حسابٍ ليست إعداداً**: هي كالدور — تُمنح لحسابٍ بعينه، ولا تُضبط
// برقمٍ في شاشة. (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «كلُّ المبالغ تُضاف وتُخصم من
// محفظة الأدمن فقط» — بعد «لا أريد أن يبقى أيُّ إعداد».)
//
// **وفراغُه لا يُعطّل تسليماً**: يُسلَّم الطلبُ ويبقى الدفترُ ناقصَ طرفٍ حتى
// تُوسَم محفظة. **وطلبٌ يُرفض لأنّ صفةً لم تُمنح خسارةٌ لا تُحتمَل.**
func (s *Service) treasuryID(ctx context.Context) string {
	var id string
	if err := s.db.QueryRow(ctx,
		`SELECT user_id::text FROM wallets WHERE is_treasury LIMIT 1`).Scan(&id); err != nil {
		return ""
	}
	return id
}

// creditTreasury يُسوّي نصيبَ المنصة من الطلب — **بالفرق لا بالمجموع**.
//
// # لماذا الفرق
//
// التسويةُ تقع على مرحلتين: **مستحقُّ المتجر عند الاستلام**، وأجرُ السائق
// ونصيبُ المندوب عند التسليم. **فتُنادى هذه مرّتين للطلب الواحد.**
//
// ولو قيّدت المجموعَ في كلِّ مرّةٍ **لتضاعف ربحُ الطلب** — فتقرأ المنصةُ ضعفَ
// ما كسبت وتبني عليه قراراً.
//
// **فتُقيّد ما بقي**: تحسب النصيبَ كاملاً بحاله الآن، وتطرح ما قُيّد سابقاً،
// وتضع الفرق. **ونداءٌ ثالثٌ بلا تغيّرٍ يضع صفراً ولا يكتب شيئاً.**
//
// # وما دفعه الزبونُ يُقرأ من واقعه لا من عمود
//
//   - **النقديُّ لا يُحسب حتى يُسلَّم**: `cash_due` وعدٌ لا قبض. وحسبانُه عند
//     الاستلام يجعل الخزينةَ رابحةً قبل أن يُدفع لها شيء.
//   - **وما رُدَّ يُطرح**: طلبٌ استُرجع ثمنُه لم يُدفع للمنصة، **وإبقاؤه في
//     الحساب يُظهر ربحاً من طلبٍ خسرته.**
func (s *Service) creditTreasury(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	tid := s.treasuryID(ctx)
	if tid == "" {
		return nil
	}
	var walletPaid, cashDue int64
	var status string
	if err := q.QueryRow(ctx,
		`SELECT wallet_paid, cash_due, status FROM orders WHERE id = $1`, orderID).
		Scan(&walletPaid, &cashDue, &status); err != nil {
		return err
	}
	// **والاسترجاعُ لا يُلغي قبضاً وقع.**
	//
	// كان الشرطُ `status == StDelivered` وحدَه — **فطلبٌ نقديٌّ سُلّم ثمّ
	// استُرجع تُقرأ حالتُه `refunded` فيصير المقبوضُ صفراً**، والزبونُ قبضه
	// السائقُ فعلاً وهو في صندوقه الآن. **فتُقيَّد الخسارةُ مرّتين**: مرّةً
	// بأنّ المالَ لم يدخل، ومرّةً بأنّه رُدّ.
	//
	// وقع فعلاً في `#1003`: طلبٌ قدرُه ٢٢٬٠٠٠ خسرت فيه الخزينةُ ٣٩٬٨٠٠،
	// **وأجرُ السائق وحدَه سبعةُ آلاف.**
	//
	// **و`refunded` لا تأتي إلّا من `delivered`** (خارطةُ `statuses.go`) —
	// فوجودُها إقرارٌ بأنّ المالَ قُبض. **والالتزامُ المقابلُ يُقيَّد وحدَه
	// في `refund`**، وهذا هو التمييزُ المحاسبيُّ بين قبضٍ وقع والتزامٍ نشأ.
	paid := walletPaid
	if status == StDelivered || status == StRefunded {
		paid += cashDue
	}

	// **ما قُيّد للأطراف وما رُدَّ للزبون وما سبق أن أخذته الخزينة** — ثلاثةٌ
	// تُقرأ من الدفتر في نداءٍ واحد. **والقراءةُ من الدفتر لا من المعادلة**:
	// قيدٌ رُفض لنقص رصيدٍ يظهر أثرُه هنا فوراً.
	var toParties, refunded, posted int64
	if err := q.QueryRow(ctx, `
		SELECT
			COALESCE(sum(amount) FILTER (WHERE kind IN
				('merchant_earning','driver_earning','commission')), 0),
			COALESCE(sum(amount) FILTER (WHERE kind = 'refund'), 0),
			COALESCE(sum(amount) FILTER (WHERE kind = 'platform_profit'), 0)
		FROM wallet_transactions WHERE ref = $1`, orderID).
		Scan(&toParties, &refunded, &posted); err != nil {
		return err
	}

	delta := (paid - refunded - toParties) - posted
	if delta == 0 {
		return nil
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, delta, "platform_profit",
		orderID, "تسويةُ نصيب المنصة", &actorID)
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
	// **`platform_expense` لا `platform_profit`**: هذه نفقةٌ قرّرها إنسان،
	// **والمحرّكُ يجمع أرباحَه ليعرف كم بقي عليه** — فلو وجدها بينها لحسبها
	// من عمله وصحّحها، **فيمحو تعويضاً وقع فعلاً.**
	_, err := s.wallet.ApplyTx(ctx, q, tid, -amount, "platform_expense",
		ref, note, &actorID)
	return err
}

// CreditTreasuryTx يُعيد حسابَ نصيب المنصة من خارج المحرّك — بعد إرجاعٍ أو
// تعويضٍ يقرّره إنسان.
//
// **ويُنادى داخل معاملة المستدعي**: قيدُ الإرجاع ونصيبُ الخزينة إمّا يقعان
// معاً أو لا يقع أحدُهما — **ودفترٌ نصفُه مكتوبٌ أسوأُ من دفترٍ لم يُكتب.**
func (s *Service) CreditTreasuryTx(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	return s.creditTreasury(ctx, q, orderID, actorID)
}

// CreditTreasuryDirect يقيّد مبلغاً موجباً للخزينة بمرجعٍ حرّ.
//
// **وتختلف عن `CreditTreasuryTx`**: تلك تحسب نصيبَ المنصة من طلبٍ بعينه
// **فرقاً بين ما قُبض وما دُفع**، وهذه **تقيّد مبلغاً بعينه** جاء من خارج
// الطلب — استردادُ مطالبةٍ من متجر مثلاً.
//
// **ولولاها لَخُصم من المتجر ولم يعد شيءٌ إلى الخزينة** — فتبدو المنصةُ خاسرةً
// وقد استُرِدّ لها. **والربحُ الذي لا يعرف ما عاد إليه ليس ربحاً.**
func (s *Service) CreditTreasuryDirect(ctx context.Context, q wallet.Querier,
	amount int64, ref, note, actorID string) error {
	if amount <= 0 {
		return nil
	}
	tid := s.treasuryID(ctx)
	if tid == "" {
		return nil
	}
	_, err := s.wallet.ApplyTx(ctx, q, tid, amount, "platform_profit", ref, note, &actorID)
	return err
}
