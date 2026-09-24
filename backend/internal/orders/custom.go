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

	// ErrQuoteChanged **تغيّر العرضُ بين عرضِه وتأكيدِه** — Batch 2a.
	//
	// **الزبونُ يؤكّد مبلغاً ونسخةً بعينهما**: فإن تبدّل أحدُهما قبل أن يصل
	// تأكيدُه رُدّ **ليقرأ العرضَ الجديدَ ويؤكّده** — **ولا يُقفَل سعرٌ لم يره.**
	ErrQuoteChanged = httpx.NewError(http.StatusConflict,
		"quote_changed", "errors.quote_changed")

	// ErrQuoteNotConfirmed **ولا يبدأ الشراءُ قبل أن يؤكّد الزبونُ العرضَ الحاليّ** — Batch 2a.
	ErrQuoteNotConfirmed = httpx.NewError(http.StatusConflict,
		"quote_not_confirmed", "errors.quote_not_confirmed")

	// ErrCustomLocked **ولا يُعدَّل العرضُ بعد الاستلام** — Batch 2a.
	//
	// **بعد أن يستلم السائقُ البضاعةَ يُقفَل السعر**: لا السائقُ يغيّره ولا
	// يُعاد التأكيد. **وما بعدَه تصحيحٌ أو تعويضٌ بيد الأدمن، موثَّقٌ.**
	ErrCustomLocked = httpx.NewError(http.StatusConflict,
		"custom_locked", "errors.custom_locked")
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

// customFeePolicy **لقطةُ سياسة أجرة المخصَّص لحظةَ الإنشاء** — Batch 2a.
//
// **السياسةُ العامّةُ قد تتبدّل بعد الإنشاء**، فتُلتقط على الطلب فلا تتبعه.
//
//   - driver_defined (الافتراض): السائقُ يحدّد الأجرة — لا لقطة، ويجوز له.
//   - admin_defined: المنصةُ تحدّدها — تُلتقط قيمتُها، وهل يجوز للسائق تغييرُها.
//
// **ويُقرأ من منفّذ النداء** (`s.on(q)`) داخلَ المعاملة — لا من المَسبَح.
func (s *Service) customFeePolicy(ctx context.Context) (source string, snapshot *int64, mayChange bool) {
	source, mayChange = "driver_defined", true
	if s.settings == nil {
		return
	}
	if s.settings.GetString(ctx, "delivery.custom_fee_source") == "admin_defined" {
		source = "admin_defined"
		fee := s.settings.GetInt(ctx, "delivery.custom_fee")
		snapshot = &fee
		mayChange = s.settings.GetBool(ctx, "delivery.custom_driver_may_change_fee")
	}
	return
}

// CreateCustom **يُنشئ طلباً خاصّاً — بلا متجرٍ ولا سعر.**
//
// **وأعمدةُ المال أصفار**: `subtotal` و`total` و`delivery_fee` و`cash_due`.
// **فيمرّ على الدفتر كأنّه لم يكن** — وهو ما يجب.
//
// **والدفعُ نقدٌ أو محفظة** — (قرارُ المالك ٢٠٢٦-٠٨-٠٩). **ولا يُخصم شيءٌ
// هنا**: المبلغُ مجهولٌ حتّى يتّفقا، **والخصمُ عند التسليم لا عند الطلب.**
func (s *Service) CreateCustom(ctx context.Context, customerID, request,
	addressText, payment, notes string, lat, lng float64) (*Order, error) {
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

	// **وتوثيقُ واتساب يشمل الخاصَّ كما يشمل العاديّ** — `D8` (انظر الشرحَ في
	// `CreateCustomTx`). **والسياسةُ واحدةٌ لكلّ بابٍ يُنشئ طلباً خاصّاً.**
	if s.settings != nil && s.settings.RequireWhatsApp(ctx, "customers.require_whatsapp") {
		var verified bool
		if err := s.db.QueryRow(ctx,
			`SELECT whatsapp_verified_at IS NOT NULL FROM users WHERE id = $1`,
			customerID).Scan(&verified); err != nil {
			return nil, err
		}
		if !verified {
			return nil, ErrWhatsAppRequired
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
	// **ولقطةُ سياسة الأجرة تُثبَّت مع الطلب** — Batch 2a.
	feeSource, feeSnap, feeMayChange := s.customFeePolicy(ctx)
	var id string
	err = s.db.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
		                    status, payment_method, subtotal, delivery_fee, total, cash_due,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders, notes,
		                    custom_fee_source, custom_fee_snapshot, custom_driver_may_change_fee)
		VALUES ('custom', $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5,
		        'pending', $6, 0, 0, 0, 0, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id::text`,
		customerID, addressText, lat, lng, request, payment,
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders, notes,
		feeSource, feeSnap, feeMayChange).Scan(&id)
	if err != nil {
		return nil, err
	}
	// ══════════════════════════════════════════════════════════════════
	// **وإنشاءُ الطلبِ حدثٌ يُسجَّل كالعاديّ** (`D9`، `CUST-CUSTOM-010`)
	// ══════════════════════════════════════════════════════════════════
	//
	// **العاديُّ يقيّد لحظةَ ميلادِه في `order_events`** (`service.go`:
	// `'', 'pending'`)، **والخاصُّ كان لا يقيّدها** — **فيبدأ سجلُّ حالاته
	// من أوّل انتقالٍ لا من الإنشاء**، ويُقرأ الطلبُ بلا لحظةِ نشأة.
	// **والفاعلُ صاحبُه** — هو من أنشأه.
	if _, err := s.db.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'pending', $2, '')`, id, customerID); err != nil {
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
	addressText, payment, notes string, lat, lng float64) (*Order, func(), error) {
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
	// **سلطةُ الجغرافيا الإداريّة قبل التغطية** (Batch 3a) — كالعاديّ حرفاً:
	// صحّةُ النقطة، ثمّ محافظةٌ/مدينةٌ مُطلَقةٌ (`classifyPlace`)، ثمّ التغطية.
	if !ValidPoint(lat, lng) {
		return nil, nil, ErrBadPoint
	}
	if err := s.requirePlaceLaunched(ctx, q, lat, lng); err != nil {
		return nil, nil, err
	}
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

	// ══════════════════════════════════════════════════════════════════
	// **وتوثيقُ واتساب يشمل الخاصَّ كما يشمل العاديّ** — `D8`
	// ══════════════════════════════════════════════════════════════════
	//
	// **الشرطُ يُفحَص في `CreateTx` وحدَها** — **والخاصُّ كان بابَه المفتوح**:
	// حين يُشغّل المالكُ التوثيقَ، من لم يوثّق رقمَه يُنشئ طلباً خاصّاً ويلتفّ
	// على الشرط. **والسياسةُ واحدةٌ لبابين** (`RequireWhatsApp`: المفتاحُ العامُّ
	// `auth.require_whatsapp` يعلو مفتاحَ الدور `customers.require_whatsapp`)،
	// والرسالةُ نفسُها (`ErrWhatsAppRequired`)، **ولا نسخةَ ثانيةً من المنطق.**
	//
	// **ولمّا كان العامُّ مطفأً كان كامناً** — `RequireWhatsApp` تردّ `false`،
	// فلا فرقَ بين البابين. **فإن شُغّل ظهر الفرقُ ما لم يُسدّ هنا** — والمفتاحُ
	// مطفأٌ في الإنتاج اليومَ، والحارسُ يمتّنه لِما بعدَ تشغيله.
	//
	// **ويُفحَص قبل القفل والكتابة** — فلا يُكتب شيءٌ لطلبٍ يُردّ.
	if u := s.on(q); u.settings != nil && u.settings.RequireWhatsApp(ctx, "customers.require_whatsapp") {
		var verified bool
		if err := q.QueryRow(ctx,
			`SELECT whatsapp_verified_at IS NOT NULL FROM users WHERE id = $1`,
			customerID).Scan(&verified); err != nil {
			return nil, nil, err
		}
		if !verified {
			return nil, nil, ErrWhatsAppRequired
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
	// **ولقطةُ سياسة الأجرة تُثبَّت مع الطلب في معاملته** — Batch 2a.
	// **وتُقرأ من المنفّذ المُمرَّر** (`s.on(q)`) — لا من المَسبَح داخلَ معاملة.
	feeSource, feeSnap, feeMayChange := s.on(q).customFeePolicy(ctx)
	var id string
	err = q.QueryRow(ctx, `
		INSERT INTO orders (kind, customer_id, address_text, dropoff, custom_request,
		                    status, payment_method, subtotal, delivery_fee, total, cash_due,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders, notes,
		                    custom_fee_source, custom_fee_snapshot, custom_driver_may_change_fee)
		VALUES ('custom', $1, $2, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography, $5,
		        'pending', $6, 0, 0, 0, 0, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING id::text`,
		customerID, addressText, lat, lng, request, payment,
		snap.MerchantCommissionPercent, snap.RepCommissionPercent,
		snap.CommissionSource, snap.ActivationOrders, notes,
		feeSource, feeSnap, feeMayChange).Scan(&id)
	if err != nil {
		return nil, nil, err
	}
	// **وحدثُ الإنشاء يُقيَّد في المعاملة نفسِها** (`D9`، `CUST-CUSTOM-010`) —
	// **مع الطلبِ أو لا** (لا حدثَ ميلادٍ لطلبٍ ارتدّت معاملتُه). كالعاديّ:
	// `'', 'pending'` والفاعلُ صاحبُه.
	if _, err := q.Exec(ctx, `
		INSERT INTO order_events (order_id, from_status, to_status, actor_id, note)
		VALUES ($1, '', 'pending', $2, '')`, id, customerID); err != nil {
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

	// **والعملُ في معاملةٍ مقفولة** — Batch 2a: **التغييرُ وضبطُ الحجز
	// والتدقيقُ فعلٌ واحدٌ يتمّ كلُّه أو لا يقع منه شيء.**
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	row, err := s.lockCustomRow(ctx, tx, orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return httpx.ErrNotFound
	}
	if err != nil {
		return err
	}
	if row.kind != KindCustom {
		return ErrNotCustom
	}
	// **ولا يوثّق إلّا من يحمله** — هو من اتّفق.
	if row.driverID == nil {
		return ErrCustomTooEarly
	}
	if *row.driverID != driverID {
		return httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
	}
	// **ولا يُعدَّل العرضُ بعد الاستلام** — Batch 2a: يُقفَل السعرُ لحظةَ
	// خروجِ البضاعة، **فلا السائقُ يغيّره** (ما بعدَه للأدمن تصحيحاً موثَّقاً).
	if row.pickedUpAt != nil {
		return ErrCustomLocked
	}

	// ── سلطةُ الأجرة من لقطة السياسة، لا من الإعداد العامّ الحاليّ ──────
	//
	// **admin_defined بلا إذنٍ للسائق**: تُفرض قيمةُ اللقطة ويُهمَل ما أرسله
	// السائق — **خادمٌ يُغلق البابَ لا شاشةٌ تُخفي الحقل.** **وما عداه أجرةُ
	// السائق** (سائقيَّ المصدر، أو أدمنيَّه بإذنٍ للتغيير). **والبضاعةُ للسائق
	// دائماً** — هو من اشتراها.
	effGoods, effFee := goods, fee
	if row.feeSource == "admin_defined" && !row.driverMayChangeFee {
		effFee = 0
		if row.feeSnapshot != nil {
			effFee = *row.feeSnapshot
		}
	}

	// **والجوهرُ في `applyCustomQuoteTx`** — كتابةُ الأعمدة، وتزايدُ النسخة
	// عند تغييرٍ حقيقيٍّ وحدَه، وسياسةُ التأكيد/الحجز، والتدقيق. **بابٌ واحدٌ
	// يشترك فيه السائقُ والأدمن** — ولا نسختان من المنطق الماليّ.
	if _, err := s.applyCustomQuoteTx(ctx, tx, row, effGoods, effFee, quoteMutator{
		actorID: driverID, role: "driver", source: "driver_agree",
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	// **ويصل صاحبَ الطلب ما وُثّق باسمه** (`D22`) — بعد التثبيت لا داخلَه.
	// **ومحفظتُه إن تحرّك حجزُها.**
	if o, err := s.GetByID(ctx, orderID); err == nil {
		s.publishOrder(o)
		s.publishWalletsOf(ctx, orderID)
	}
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
// # ويُسوّى الحجزُ لا الرصيدُ المتاح — Batch 2a
//
// **المالُ محجوزٌ منذ تأكيد الزبون** (`custom_reserved_amount`)، **والحجزُ
// يمنع إنفاقَه في غيره**: فرصيدٌ لا يكفي مستحيلٌ هنا — القيدُ `reserved <=
// balance` يرفض أيَّ خصمٍ ينزل بالرصيد تحت المحجوز. **فيُفكّ الحجزُ ويُخصَم
// معاً** (`SettleReservedTx`)، **ويُودَع للسائق.**
//
// **وحجزٌ صفرٌ في طلبِ محفظةٍ بلغ التسليمَ شذوذٌ** — فالاستلامُ مقفولٌ على
// تأكيدٍ يحجز. **فيُقيَّد في السجلّ ولا يُخصَم** (لا حجزَ يُسوّى)، **ولا
// يُبطَل تسليمٌ وقع.**
func (s *Service) settleCustomWallet(ctx context.Context, q wallet.Querier,
	in settlement) error {
	if in.driverID == nil {
		return nil
	}
	var method string
	var reserved int64
	var paidAt *time.Time
	if err := q.QueryRow(ctx, `
		SELECT payment_method, custom_reserved_amount, custom_paid_at
		FROM orders WHERE id = $1 FOR UPDATE`, in.orderID).
		Scan(&method, &reserved, &paidAt); err != nil {
		return err
	}
	// **دفعٌ من محفظة، ولم يُدفع قبلُ** — والعلامةُ (`custom_paid_at`) تمنع
	// التكرار: **التسليمُ قد يُنادى مرّتين**، فيُخصم مرّتين لولاها.
	if method != "wallet" || paidAt != nil {
		return nil
	}
	if reserved <= 0 {
		s.logger.Warn("الطلب الخاصّ: تسليمٌ بمحفظةٍ بلا حجزٍ حيّ — لا خصم",
			"order", in.orderID)
		return nil
	}

	// **يُفكّ الحجزُ ويُخصَم معاً** — فعلٌ واحدٌ لا يفترق (`SettleReservedTx`).
	if _, err := s.wallet.SettleReservedTx(ctx, q, in.customerID, reserved, "order_payment",
		in.orderID, "طلبٌ خاصّ — دفعٌ من المحفظة", &in.actorID); err != nil {
		return err
	}
	if _, err := s.wallet.ApplyTx(ctx, q, *in.driverID, reserved, "driver_earning",
		in.orderID, "طلبٌ خاصّ — تحصيلٌ من محفظة الزبون", &in.actorID); err != nil {
		return err
	}
	// **ويُصفَّر الحجزُ على الطلب مع علامة الدفع** — فلا يبقى محجوزٌ بعد التسوية.
	_, err := q.Exec(ctx,
		`UPDATE orders SET custom_reserved_amount = 0, custom_paid_at = now(),
		        updated_at = now() WHERE id = $1`, in.orderID)
	return err
}
