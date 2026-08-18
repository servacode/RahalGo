package orders

// **الطلبُ الخاصّ — ما ليس في المنصّة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «الزبون يريد أكلاً من مطعمٍ محدَّد أو شيئاً من
//
//	سوقٍ غير موجودٍ بالمتجر».)
//
// # وثلاثةُ فروقٍ عن الطلب العاديّ
//
//  1. **لا متجرَ له** — الزبونُ يصف ما يريد بلفظه.
//  2. **ولا سعرَ عند إنشائه** — يُعرَف حين يشتري السائق.
//  3. **ولا أثرَ ماليَّ في المنصّة** — السائقُ يدفع من جيبه ويستردّ عند
//     التسليم، **والمنصّةُ توثّق ولا تحاسب.**
//
// # وما عدا ذلك فهو طلبٌ كسائر الطلبات
//
// **يُوزَّع بالدور، وتُفتح محادثتُه عند الإسناد، ويمشي مراحلَ الطريق، ويُشتكى
// عليه، ويُقيَّم.** ولو كان كياناً ثانياً لَلزم بناءُ ذلك كلِّه من جديد، **ثمّ
// افتراقُه عنه** — يُصلَح شرطٌ هنا ويُنسى هناك.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

var (
	// ErrNotCustom **توثيقُ المبلغ للخاصّ وحدَه.**
	ErrNotCustom = httpx.NewError(http.StatusConflict,
		"not_custom_order", "errors.not_custom_order")

	// ErrCustomTooEarly **ولا يُوثَّق قبل أن يكون له سائق** — هو من يتّفق.
	ErrCustomTooEarly = httpx.NewError(http.StatusConflict,
		"custom_not_assigned", "errors.custom_not_assigned")

	// ErrCustomNotAgreed **ولا يبدأ قبل أن يُوثَّق ما اتُّفق عليه.**
	ErrCustomNotAgreed = httpx.NewError(http.StatusConflict,
		"custom_not_agreed", "errors.custom_not_agreed")
)

// isCustom **أطلبٌ خاصٌّ هو؟** — سؤالٌ يُسأل قبل قرارٍ يخصّه.
//
// **ويردّ `false` عند الشكّ**: خطأُ قراءةٍ يجعله عاديّاً، **وهو الأسلم** —
// العاديُّ مسارُه المبنيّ المختبَر.
func (s *Service) isCustom(ctx context.Context, orderID string) bool {
	var kind string
	if err := s.db.QueryRow(ctx,
		`SELECT kind FROM orders WHERE id = $1`, orderID).Scan(&kind); err != nil {
		return false
	}
	return kind == KindCustom
}

// MaxCustomRequest **حدُّ وصف الطلب.**
//
// **وما يُطلب يُوصف في سطرين**: «شاورما دجاج من مطعم الأصيل، بلا ثوم».
// **وحدٌّ واسعٌ يجعله رسالةً**، والرسالةُ محلُّها المحادثة بعد الإسناد.
const MaxCustomRequest = 600

// CreateCustom **يُنشئ طلباً خاصّاً — بلا متجرٍ ولا سعر.**
//
// **وأعمدةُ المال أصفار**: `subtotal` و`total` و`delivery_fee` و`cash_due`.
// **فيمرّ على الدفتر كأنّه لم يكن** — وهو ما يجب.
//
// **والدفعُ نقدٌ أو محفظة** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩). **ولا يُخصم شيءٌ
// هنا**: المبلغُ مجهولٌ حتّى يتّفقا، **والخصمُ عند التسليم لا عند الطلب.**
func (s *Service) CreateCustom(ctx context.Context, customerID, request,
	addressText, payment string, lat, lng float64) (*Order, error) {
	// **والمحفظةُ خيارٌ** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩). **وما عداهما نقد**:
	// قيمةٌ مجهولةٌ من العميل لا تصير طريقةَ دفع.
	if payment != "wallet" {
		payment = "cash"
	}
	request = strings.TrimSpace(request)
	if request == "" || strings.TrimSpace(addressText) == "" {
		return nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len([]rune(request)) > MaxCustomRequest {
		request = string([]rune(request)[:MaxCustomRequest])
	}

	// **وسقفُ المفتوح يشمله** — القاعدةُ نفسُها: من بيده ثلاثةٌ لا يفتح رابعاً.
	//
	// **ولو استُثني لَصار باباً يلتفّ به على السقف** — يُنشئ خاصّةً بلا حدّ.
	if err := s.checkOpenLimit(ctx, customerID); err != nil {
		return nil, err
	}

	var id string
	err := s.db.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
		                    status, payment_method, subtotal, delivery_fee, total, cash_due)
		VALUES ('custom', $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5,
		        'pending', $6, 0, 0, 0, 0)
		RETURNING id::text`,
		customerID, addressText, lat, lng, request, payment).Scan(&id)
	if err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// AgreeCustom **يوثّق ما اتّفق عليه السائقُ والزبون.**
//
// (قرارُ المالك: «يجب على السائق إرسال طلبٍ بالسعر والأجرة لتوثيق ذلك لدى
//
//	الإدارة، ولكنّ السعر والأجرة لن تدخل بالحسابات».)
//
// # ولماذا يُوثَّق أصلاً إن لم يُحاسَب
//
// **حجّةٌ عند الخلاف.** من ادّعى أنّه دفع أكثر، أو أنّ الأجرة كانت أقلّ،
// **يُرجَع إلى ما وُثّق ووقتِه.** ولولاه لبقيت كلمةٌ ضدّ كلمة، **والعملياتُ
// تحكم بين اثنين لا تملك عن أيّهما شيئاً.**
//
// # ولا يُكتب في أعمدة المحاسبة
//
// **`custom_goods_amount` و`custom_fee` عمودان قائمان بذاتهما** — و`subtotal`
// و`total` يقرؤهما الدفترُ والخزينةُ وعمولةُ المندوب. **ورقمٌ يُوثَّق في عمودٍ
// يُحاسَب يصير مالاً للمنصّة بلا أن يقرّر ذلك أحد.**
//
// # ويُكتب مرّةً ويُعدَّل
//
// **الاتّفاقُ قد يتبدّل**: يجد السائقُ الصنفَ أغلى فيعود إلى الزبون. **فيُحدَّث
// ووقتُه معه** — والوقتُ هو ما يُقرأ عند الخلاف: **متى قال ماذا.**
// # ولماذا يُكتب النوعُ في الجمع
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «عند توثيق السعر بالطلب الخاصّ يعطي: تعذّر
//
//	الاتصال، حاول بعد قليل».)
//
// **وكان الجمعُ بلا نوع** — وسيطان مجهولان، **فتسأل بوستغرس: أيُّ «+»
// هذا؟** فلا تجد واحداً بعينه ويسقط النداءُ بخمسمئة
// (operator is not unique: unknown + unknown — رُئي في سجلّ الخادم).
//
// **والإسنادُ وحدَه يُعطي النوع** — والجمعُ لا: يُحسب قبل أن يُعرف
// طرفاه.
//
// **ولا خطأَ يفهمه صاحبُ التطبيق**: تصل خمسُمئةٍ فتقول الشاشةُ «تعذّر
// الاتصال» — **فيُتّهم الإنترنتُ وتُعاد المحاولةُ عشراً**، والعطبُ في
// حرفين.
func (s *Service) AgreeCustom(ctx context.Context, orderID, driverID string,
	goods, fee int64) error {
	if goods < 0 || fee < 0 {
		return httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}

	var kind string
	var owner *string
	err := s.db.QueryRow(ctx,
		`SELECT kind, driver_id::text FROM orders WHERE id = $1`, orderID).
		Scan(&kind, &owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if kind != KindCustom {
		return ErrNotCustom
	}
	// **ولا يوثّق إلّا من يحمله** — هو من اتّفق.
	if owner == nil {
		return ErrCustomTooEarly
	}
	if *owner != driverID {
		return httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}

	// ══════════════════════════════════════════════════════════════════
	// **وما اتُّفق عليه يُكتب في أعمدة الطلب أيضاً**
	// ══════════════════════════════════════════════════════════════════
	//
	// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «بالطبع يجب أن يُكتب الإجماليُّ وأجرةُ
	//  التوصيل بالطلب أيضاً، لتكون واضحةً لدى الزبون والسائق
	//  والمنصّة».)
	//
	// # وكانت أصفاراً عمداً
	//
	// **خشيةَ أن يصير الرقمُ مالاً للمنصّة بلا قرار** — والتسويةُ تقرأ
	// هذه الأعمدة. **وقيس اليومَ أنّها لا تقرؤها في الخاصّ**: التسويةُ
	// تخرج خروجاً صريحاً قبل أن تصل إليها (`transitions.go`:
	// `if in.custom { … return }`) — **لا عمولةَ ولا مستحقَّ متجرٍ ولا
	// خزينةَ ولا قيدَ صندوق.**
	//
	// **وبقاؤها أصفاراً كان يكذب على ثلاثة**: الزبونُ يقرأ فاتورةً
	// بصفر، والسائقُ يقرأ سجلَّه «٠ ل.س» على طلبٍ حمل فيه ستّةَ آلاف،
	// **والمنصّةُ تحسب يومَها ناقصاً.**
	//
	// # ولا يُمسّ `cash_due`
	//
	// **هو مطلبُ المنصّة على السائق** — وما قبضه في الخاصّ مالُه هو:
	// ثمنٌ دفعه من جيبه وأجرةٌ استحقّها. **ولو كُتب فيه لَظهر في ذمّته
	// دَينٌ لا وجودَ له.**
	_, err = s.db.Exec(ctx, `
		UPDATE orders
		SET custom_goods_amount = $2, custom_fee = $3,
		    -- **والنوعُ يُقال صراحةً في الجمع** — انظر الشرحَ فوق الدالّة.
		    subtotal = $2, delivery_fee = $3, total = $2::bigint + $3::bigint,
		    custom_agreed_at = now(), updated_at = now()
		WHERE id = $1`, orderID, goods, fee)
	if err != nil {
		return err
	}
	// **وتراه العملياتُ فوراً** — التوثيقُ يُقرأ لحظةَ وقوعه لا بعد ساعة.
	s.pub.Publish("ops", map[string]any{"type": "order"})
	return nil
}

// settleCustomWallet **ينقل ما اتُّفق عليه من محفظة الزبون إلى السائق.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «إذا تمّ الدفع من المحفظة يُحوَّل المبلغ بشكلٍ
//
//	تلقائيٍّ إلى السائق».)
//
// # ونقلٌ لا كسب
//
// **لا عمولةَ ولا مستحقَّ متجرٍ ولا خزينة** — المنصّةُ تعبر بلا أن تأخذ.
// **وهي خدمةٌ للسائق كما قال المالك**، والدفعُ من المحفظة تيسيرٌ لا تجارة.
//
// # ولا يقع إلّا بثلاثة
//
// **دفعٌ من محفظة** — ومن اختار النقدَ يقبض السائقُ بيده كما كان.
// **ومبلغٌ موثَّق** — ولا يُخصم ما لم يُتَّفق عليه.
// **ولم يُدفع قبلُ** — والعلامةُ تمنع التكرار: **التسليمُ قد يُنادى مرّتين**
// (شبكةٌ تُعيد الطلب أو موظّفٌ يضغط ثانيةً)، **فيُخصم مرّتين ولا يظهر إلّا في
// شكوى.**
//
// # ولا يُبطل تسليماً وقع
//
// **رصيدٌ لا يكفي لا يردّ التسليم**: البضاعةُ في يد الزبون فعلاً. **فيُقيَّد
// في السجلّ ويبقى الدَّينُ بينهما** — ومن ردّ التسليمَ لأجل رصيدٍ ترك السائقَ
// بلا إقفالٍ ولا مال.
func (s *Service) settleCustomWallet(ctx context.Context, q wallet.Querier,
	in settlement) error {
	if in.driverID == nil {
		return nil
	}
	var method string
	var goods, fee *int64
	var paidAt *time.Time
	if err := q.QueryRow(ctx, `
		SELECT payment_method, custom_goods_amount, custom_fee, custom_paid_at
		FROM orders WHERE id = $1 FOR UPDATE`, in.orderID).
		Scan(&method, &goods, &fee, &paidAt); err != nil {
		return err
	}
	if method != "wallet" || goods == nil || paidAt != nil {
		return nil
	}
	total := *goods
	if fee != nil {
		total += *fee
	}
	if total <= 0 {
		return nil
	}

	var balance int64
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`,
		in.customerID).Scan(&balance); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if balance < total {
		s.logger.Warn("الطلب الخاصّ: رصيدُ الزبون لا يكفي — يبقى الدَّينُ بينهما",
			"order", in.orderID, "need", total, "have", balance)
		return nil
	}

	if _, err := s.wallet.ApplyTx(ctx, q, in.customerID, -total, "order_payment",
		in.orderID, "طلبٌ خاصّ — دفعٌ من المحفظة", &in.actorID); err != nil {
		return err
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *in.driverID, total, "driver_earning",
		in.orderID, "طلبٌ خاصّ — تحصيلٌ من محفظة الزبون", &in.actorID); err != nil {
		return err
	}
	_, err := q.Exec(ctx,
		`UPDATE orders SET custom_paid_at = now() WHERE id = $1`, in.orderID)
	return err
}
