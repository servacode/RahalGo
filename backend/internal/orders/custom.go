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
	"github.com/servacode/rahalgo/backend/internal/dbtx"
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

	// **ومن مُنع النقدَ في العاديّ يُمنعه في الخاصّ** — `D6` (انظر الشرحَ في
	// `CreateCustomTx`). **والحدُّ واحدٌ لكلّ بابٍ يُنشئ طلباً خاصّاً.**
	if payment == "cash" {
		blocked, err := s.cashBlocked(ctx, s.db, customerID)
		if err != nil {
			return nil, err
		}
		if blocked {
			return nil, ErrCashBlocked
		}
	}

	// **وسقفُ المفتوح يشمله** — القاعدةُ نفسُها: من بيده ثلاثةٌ لا يفتح رابعاً.
	//
	// **ولو استُثني لَصار باباً يلتفّ به على السقف** — يُنشئ خاصّةً بلا حدّ.
	if err := s.checkOpenLimit(ctx, s.db, customerID); err != nil {
		return nil, err
	}

	// **والطلبُ الخاصُّ يُلتقَط اقتصادُه كغيره** — `XQ-2`: **سعرُه
	// يُتّفق عليه لاحقاً، ونسبُه تُثبَّت اليوم.**
	snap, err := s.snapshotNow(ctx, s.db)
	if err != nil {
		return nil, err
	}
	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
		                    status, payment_method, subtotal, delivery_fee, total, cash_due,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders)
		VALUES ('custom', $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5,
		        'pending', $6, 0, 0, 0, 0, $7, $8, $9, $10)
		RETURNING id::text`,
		customerID, addressText, lat, lng, request, payment,
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders).Scan(&id)
	if err != nil {
		return nil, err
	}
	o, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	// **ويصل صاحبَه كما يصله العاديّ** — `D22`.
	s.publishOrder(o)
	return o, nil
}

// CreateCustomTx كـ`CreateCustom` **في معاملةٍ مُمرَّرة** — `XG-33`.
//
// **وهو إدخالٌ واحدٌ أصلاً** (`R11` مُنفيّة)، **لكنّه يشارك معاملةَ منع
// التكرار** ليُثبَّت الطلبُ وعلامتُه معاً.
// **ويُرجع ما يقع بعد التثبيت** — البثّ. **ولا يقع داخلَها**:
// **بثٌّ خرج ثمّ ارتدّت المعاملةُ يَعِد بطلبٍ لا وجودَ له**، وهو حدُّ
// `R24`/`XG-44` نفسُه الذي يمشي عليه `CreateTx`.
func (s *Service) CreateCustomTx(ctx context.Context, q dbtx.Querier, customerID, request,
	addressText, payment string, lat, lng float64) (*Order, func(), error) {
	// ══════════════════════════════════════════════════════════════════
	// **ونقطةُ التسليم تحكم هنا كما تحكم في العاديّ** (٢٠٢٦-٠٩-١٣)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان المخصَّصُ بلا فحصِ تغطيةٍ إطلاقاً** — **صفرُ ذكرٍ لـ`ZoneAt`
	// في الملفّ كلِّه** (قِيس ٢٠٢٦-٠٩-١٣) — **فيُقبَل طلبٌ إلى أيّ
	// نقطةٍ في العالم**، **ويُطلَب سائقٌ إلى مدينةٍ لا أحدَ فيها.**
	//
	// **وقبلَ كلّ شيء** — قبل اللقطة وقبل الكتابة وقبل سقفِ المفتوح:
	// **فلا يُقيَّد ولا يُحجَز شيءٌ لطلبٍ يُردّ.**
	//
	// **وبالمُنفّذ المُمرَّر لا بالمَسبَح** — **فالفحصُ والكتابةُ في
	// معاملةٍ واحدةٍ**، ولا تتبدّل التغطيةُ بينهما في عين هذه المعاملة.
	// **ووقتُ المنطقة بعد جغرافيتها** (`ZH`) — **والمنطقةُ هي التي
	// ردّتها بوّابةُ القبول نفسُها، لا نتيجةُ استعلامٍ ثانٍ** (`ZH-34`).
	z, err := s.RequireServiceable(ctx, q, lat, lng)
	if err != nil {
		return nil, nil, err
	}
	if err := s.requireZoneOpen(ctx, q, z); err != nil {
		return nil, nil, err
	}

	// **والمحفظةُ خيارٌ** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩). **وما عداهما نقد**:
	// قيمةٌ مجهولةٌ من العميل لا تصير طريقةَ دفع.
	if payment != "wallet" {
		payment = "cash"
	}
	request = strings.TrimSpace(request)
	if request == "" || strings.TrimSpace(addressText) == "" {
		return nil, nil, httpx.NewError(http.StatusBadRequest, "validation", "errors.validation")
	}
	if len([]rune(request)) > MaxCustomRequest {
		request = string([]rune(request)[:MaxCustomRequest])
	}

	// ══════════════════════════════════════════════════════════════════
	// **ومن مُنع النقدَ في العاديّ يُمنعه في الخاصّ** — `D6`
	// ══════════════════════════════════════════════════════════════════
	//
	// **حظرُ النقد يُفحَص في `CreateTx` وحدَها** (`cashBlocked`) — **والخاصُّ
	// كان بابَه المفتوح**: من رفض الاستلامَ فقُفل عليه النقدُ يُنشئ طلباً
	// خاصّاً نقداً ويلتفّ على القفل. **والحدُّ واحدٌ لبابين** — القرارُ نفسُه
	// والرسالةُ نفسُها (`ErrCashBlocked`)، **ولا نسخةَ ثانيةً من المنطق.**
	//
	// **ويُفحَص قبل القفل والكتابة** — فلا يُقيَّد ولا يُحجَز شيءٌ لطلبٍ يُردّ.
	// **والإعدادُ يُقرأ من المعاملة** (`s.on(q)`) — كما في العاديّ.
	if payment == "cash" {
		blocked, err := s.on(q).cashBlocked(ctx, q, customerID)
		if err != nil {
			return nil, nil, err
		}
		if blocked {
			return nil, nil, ErrCashBlocked
		}
	}

	// **وسقفُ المفتوح يشمله** — القاعدةُ نفسُها: من بيده ثلاثةٌ لا يفتح رابعاً.
	//
	// **ولو استُثني لَصار باباً يلتفّ به على السقف** — يُنشئ خاصّةً بلا حدّ.
	// **وقفلُ قبولِ الزبون قبل العدّ** — `D4`.
	//
	// **والعدُّ ثمّ الإدراج ليس ذرّيّاً هنا كما لم يكن في العاديّ** —
	// **وقيس: السقفُ واحدٌ وطلبان خاصّان متزامنان ⇒ مفتوحان في ثلاثين
	// جولةً من ثلاثين.**
	//
	// **والقاعدةُ واحدةٌ لبابين** (`checkOpenLimit`) — **فحارسُها
	// واحدٌ كذلك**، **ولا يُترك بابٌ يلتفّ به عليها.**
	//
	// **والفضاءُ نفسُه** (`customer-admit:`) — **فبابا الطلب يتزاحمان
	// على قفلٍ واحدٍ للزبون الواحد**، **ولا يمرّ خاصٌّ وعاديٌّ معاً.**
	if _, err := q.Exec(ctx,
		`SELECT pg_advisory_xact_lock(hashtext($1))`,
		"customer-admit:"+customerID); err != nil {
		return nil, nil, err
	}
	// **وقراءةُ السقف من المعاملة لا من المَسبَح** — `XG-46`/`XG-48`.
	//
	// **وكانت تُنادى بـ`s`** — **فتقرأ الإعدادَ باتّصالٍ ثانٍ وهي
	// داخلَ معاملة.** **وذاك جذرُ `XG-48` بعينه** (دورةُ ٥٠):
	// **معاملةٌ تمسك اتّصالاً ثمّ تطلب ثانياً** — **ومَسبَحٌ ضيّقٌ
	// تحت حملٍ يجمد عندها حتّى تنفد مهلةُ النداء.**
	//
	// **وقفلُ القبول قبلها يُطيل النافذة** — **فيصير ما كان
	// محتمَلاً واقعاً.**
	if err := s.on(q).checkOpenLimit(ctx, q, customerID); err != nil {
		return nil, nil, err
	}

	// **واللقطةُ مع الطلب في معاملته** — `XQ-2`.
	snap, err := s.snapshotNow(ctx, q)
	if err != nil {
		return nil, nil, err
	}
	var id string
	err = q.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
		                    status, payment_method, subtotal, delivery_fee, total, cash_due,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders)
		VALUES ('custom', $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5,
		        'pending', $6, 0, 0, 0, 0, $7, $8, $9, $10)
		RETURNING id::text`,
		customerID, addressText, lat, lng, request, payment,
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders).Scan(&id)
	if err != nil {
		return nil, nil, err
	}
	o, err := s.getByID(ctx, q, id)
	if err != nil {
		return nil, nil, err
	}
	return o, func() { s.publishOrder(o) }, nil
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
	// ══════════════════════════════════════════════════════════════════
	// **ويصل صاحبَ الطلب ما وُثّق باسمه** — `D22`
	// ══════════════════════════════════════════════════════════════════
	//
	// **وكان يبثّ `ops` وحدَها**: **تراه العملياتُ ولا يراه من يدفع.**
	// **واتّفاقٌ لا يراه صاحبُه ليس اتّفاقاً** — يُقال له في المحادثة
	// **ويبقى مكتوباً حيث يراه.**
	//
	// **والبابُ واحدٌ للجميع** (`publishOrder`): **هو يحلّ الأطرافَ
	// ويبني لكلٍّ حمولتَه ويسقط مغلقاً** — **ولا تُنثَر نداءاتُ بثٍّ
	// في المعالِجات.**
	//
	// **وبعد الكتابة لا داخلَها**: هذا النداءُ خارج معاملة (`s.db.Exec`
	// أعلاه مثبَّتة) — **وبثٌّ يسبق التثبيتَ يَعِد بما قد يُلغى.**
	o, err := s.GetByID(ctx, orderID)
	if err != nil {
		return err
	}
	s.publishOrder(o)
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

	// **والمتاحُ لا الرصيد** — `XG-12`: **الرصيدُ يشمل ما حُجز لسحبٍ
	// جارٍ**، ودفعُ الطلب منه يرتدّ عند القيد.
	var balance int64
	if err := q.QueryRow(ctx,
		`SELECT COALESCE((SELECT balance  FROM wallets WHERE user_id = $1), 0)
		      - COALESCE((SELECT reserved FROM wallets WHERE user_id = $1), 0)`,
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
