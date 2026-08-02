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
//
// ensureTreasuryWallet يجعل محفظةَ الحساب المختار **هي الخزينة** — ولا غيرها.
//
// # لماذا يُعاد كلَّ مرّة
//
// كان `wallets.is_treasury` عموداً يُضبط بيد، و`platform.treasury_user_id`
// مفتاحاً يُضبط بأخرى — **مصدرانِ لحقيقةٍ واحدة**. ومن غيّر المفتاح ونسي
// العمود **ترك خزينةً لا تُقيَّد**: أوّلُ نصيبٍ سالبٍ يُردّ بـ
// `insufficient_balance`، **فيسقط تسليمُ طلبٍ بسبب إعدادٍ لم يُتمّه أحد.**
//
// **فصار المفتاحُ هو الحقيقةَ والعمودُ أثرَها**: يُصحَّح عند كلِّ قيد.
// **ونظامٌ يُصحّح نفسَه أوثقُ من نظامٍ يطلب أن يُصحَّح.**
//
// والصفُّ يُنشأ إن لم يكن: **حسابٌ لم يقبض شيئاً قطُّ لا محفظةَ له**، وهو
// أوّلُ ما يقع للخزينة — تدفع قبل أن تقبض.
func (s *Service) ensureTreasuryWallet(ctx context.Context, q wallet.Querier, tid string) error {
	if _, err := q.Exec(ctx,
		`INSERT INTO wallets (user_id, is_treasury) VALUES ($1, true)
		 ON CONFLICT (user_id) DO NOTHING`, tid); err != nil {
		return err
	}
	// **واحدةٌ لا اثنتان**: الفهرسُ الفريد يمنع الثانية، فتُنزع الصفةُ عمّن
	// سبق قبل أن تُمنح لمن اختير.
	if _, err := q.Exec(ctx,
		`UPDATE wallets SET is_treasury = false WHERE is_treasury AND user_id <> $1`, tid); err != nil {
		return err
	}
	_, err := q.Exec(ctx,
		`UPDATE wallets SET is_treasury = true WHERE user_id = $1 AND NOT is_treasury`, tid)
	return err
}

func (s *Service) creditTreasury(ctx context.Context, q wallet.Querier, orderID, actorID string) error {
	tid := s.treasuryID(ctx)
	if tid == "" {
		return nil
	}
	if err := s.ensureTreasuryWallet(ctx, q, tid); err != nil {
		return err
	}

	var walletPaid, cashDue int64
	var status string
	if err := q.QueryRow(ctx,
		`SELECT wallet_paid, cash_due, status FROM orders WHERE id = $1`, orderID).
		Scan(&walletPaid, &cashDue, &status); err != nil {
		return err
	}
	paid := walletPaid
	if status == StDelivered {
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
	if err := s.ensureTreasuryWallet(ctx, q, tid); err != nil {
		return err
	}
	// **`platform_expense` لا `platform_profit`**: هذه نفقةٌ قرّرها إنسان،
	// **والمحرّكُ يجمع أرباحَه ليعرف كم بقي عليه** — فلو وجدها بينها لحسبها
	// من عمله وصحّحها، **فيمحو تعويضاً وقع فعلاً.**
	_, err := s.wallet.ApplyTx(ctx, q, tid, -amount, "platform_expense",
		ref, note, &actorID)
	return err
}
