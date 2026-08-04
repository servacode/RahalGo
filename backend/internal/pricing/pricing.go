package pricing

// التسعير — **من سعر الشراء إلى سعر البيع.**
//
// # النموذج
//
//	سعرُ الشراء  ←  ما وضعه المتجر — وتُخصم منه العمولة
//	الهامش      ←  ما تضيفه المنصة — وهو ربحُها
//	التقريب     ←  ما يجعل الرقمَ يُقرأ
//	سعرُ البيع   ←  ما يدفعه الزبون
//
// # ولماذا الهامشُ يُورَث
//
// **ولا أحدَ يُسعّر ألفَ صنفٍ بيده.** فالصنفُ يرث تصنيفَه، والتصنيفُ يرث العام
// — **ومن أراد أن يخصّ صنفاً خصّه، ومن لم يُرد لم يفعل شيئاً.**
//
// **والتجاوزُ فراغٌ لا صفر**: الصفرُ قرارٌ («لا هامشَ على هذا») والفراغُ غيابُ
// قرار («اتبع ما فوقك»). **ومن خلط بينهما جعل كلَّ صنفٍ لم يُلمس بلا هامش.**
//
// # ولماذا نمطان
//
// **الثابتُ يوافق تكلفتَك**: سائقٌ وعملياتٌ لا يتضاعفان بتضاعف قيمة الطعام —
// **وصينيةٌ بستّين ألفاً لا تكلّفك عشرةَ أضعاف الشاورما.**
//
// **والنسبةُ تمنع أن يبدو الرخيصُ غالياً**: خمسون بالمئة على الشاورما **يراها
// من يعرف سعرَها.** وطبقةُ التصنيف تحلّهما معاً.
//
// # والتقريبُ ليس تجميلاً
//
// قائمةٌ فيها ٨٬١٢٥ و١١٬٣٧٥ **تقول «هذه آلةٌ تحسب»**، وقائمةٌ فيها ٨٬٠٠٠
// و١١٬٥٠٠ تقول **«هذا سعرُنا»**. وفي سوريا حيث الفئاتُ كبيرة **هو ما يجعل
// الحسابَ ممكناً في الجيب.**

import "context"

// MarginRule ما يلزم لحساب سعر بيعٍ من سعر شراء.
type MarginRule struct {
	// Mode "percent" أو "fixed".
	Mode string
	// Value الهامشُ العام — يُستعمل حين لا تجاوزَ للصنف ولا لتصنيفه.
	Value int64
	// Rounding خانةُ التقريب — و`0` تعني بلا تقريب.
	Rounding int64
}

// Store ما يلزم لقراءة المفاتيح — واجهةٌ ضيّقة **كي لا تجرّ الحزمةُ إعداداتٍ
// كاملةً خلفها**، ولتُختبَر بلا قاعدة بيانات.
type Store interface {
	// GetString **بلا احتياطيٍّ من المنادي** — الافتراضُ في الفهرس وحدَه.
	//
	// كان يأخذه، **فكتب كلُّ منادٍ افتراضَه بيده** ووُجد الافتراضُ نفسُه في
	// موضعين. **ورقمان لمعنًى واحدٍ يفترقان.**
	GetString(ctx context.Context, key string) string
	GetInt(ctx context.Context, key string) int64
}

// RuleFrom يقرأ مفاتيح التسعير.
//
// **وبلا مخزنٍ لا هامش**: صفرٌ يعني «سعرُ البيع = سعرُ الشراء» — **وهو الحال
// قبل أن يقرّر المالك**، ولا يخترع ربحاً لم يُتّفق عليه.
func RuleFrom(ctx context.Context, st Store) MarginRule {
	if st == nil {
		return MarginRule{Mode: "percent"}
	}
	return MarginRule{
		Mode:     st.GetString(ctx, "pricing.margin_mode"),
		Value:    st.GetInt(ctx, "pricing.margin_value"),
		Rounding: st.GetInt(ctx, "pricing.rounding"),
	}
}

// SalePrice سعرُ البيع من سعر الشراء والتجاوزات.
//
// `itemOverride` و`categoryOverride` فراغُهما يعني «اتبع ما فوقك».
func (r MarginRule) SalePrice(merchantPrice int64, itemOverride, categoryOverride *int64) int64 {
	margin := r.Value
	switch {
	case itemOverride != nil:
		margin = *itemOverride
	case categoryOverride != nil:
		margin = *categoryOverride
	}

	var sale int64
	if r.Mode == "fixed" {
		sale = merchantPrice + margin
	} else {
		sale = merchantPrice + merchantPrice*margin/100
	}
	// **ولا يُباع بأقلّ من ثمنه.** هامشٌ سالبٌ بالخطأ يجعل المنصةَ تدفع من
	// جيبها عن كلّ بيعة — **وخسارةٌ تتكرّر بلا حدث تُكتشف في آخر الشهر.**
	if sale < merchantPrice {
		sale = merchantPrice
	}
	return roundTo(sale, r.Rounding)
}

// roundTo يقرّب إلى أقرب مضاعفٍ لأعلى.
//
// **ولأعلى لا لأقرب**: التقريبُ لأقرب يبتلع من الهامش نصفَ الخانة في نصف
// الأصناف — **وهامشٌ قرّره المالكُ لا ينبغي أن ينقص لأن الرقمَ لم يستوِ.**
func roundTo(n, step int64) int64 {
	if step <= 1 || n <= 0 {
		return n
	}
	if r := n % step; r != 0 {
		n += step - r
	}
	return n
}

// ── نسبةٌ أو مقطوع — والقاعدةُ واحدةٌ لثلاثة أموال ─────────────────────────
//
// عمولةُ المنصة من المتاجر، وعمولةُ المندوب، وهامشُ البيع: **ثلاثةُ أرقامٍ
// يسألها المالكُ السؤالَ نفسَه** — أنسبةٌ هي أم رقمٌ مقطوع؟
//
// **ولكلٍّ منها كانت حسبتُها في موضعها**: عمولةُ المندوب مكتوبةٌ بـSQL في
// ثلاثة ملفّات، وعمولةُ المتجر عمودٌ يُضرب في مئةٍ في خمسة. **وحسبةٌ في خمسة
// مواضع تفترق في الرابع** — فيُدفع للمندوب غيرُ ما يُعرض له.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «نسبةُ المندوب أو رقمٌ ثابتٌ مقطوع · عمولةُ المنصة
// نسبةٌ أو مقطوعةٌ من المتاجر · ولازم كلُّ المشروع يأخذ الإعداداتِ هذه بحيث
// تُطبَّق بشكلٍ حقيقيٍّ فوريٍّ عند أيّ تغيّر».)

// Amount مبلغٌ يُشتقّ من أساس — **نسبةً منه أو رقماً مقطوعاً بدلَه.**
type Amount struct {
	// Mode "percent" أو "fixed".
	Mode string
	// Value النسبةُ المئوية، أو المبلغُ المقطوع بالليرة.
	Value int64
}

// Of يحسب المبلغ من أساسه.
//
// **ولا يتجاوز الأساسَ ولا ينزل تحت الصفر.** عمولةٌ مقطوعةٌ بعشرين ألفاً على
// طلبٍ بخمسة عشر **تجعل المتجرَ يدفع لنا كي يبيع** — ورقمٌ سالبٌ يخرج من هنا
// يقلب قيداً محاسبيّاً بلا أن يصرخ أحد.
func (a Amount) Of(base int64) int64 {
	if base <= 0 {
		return 0
	}
	var v int64
	if a.Mode == "fixed" {
		v = a.Value
	} else {
		v = base * a.Value / 100
	}
	if v < 0 {
		return 0
	}
	if v > base {
		return base
	}
	return v
}

// amountFrom يقرأ نمطاً وقيمةً من الإعدادات — **عند كلّ استعمالٍ لا مرّةً.**
//
// **وهو ما يجعل التغييرَ يسري فوراً**: لا لقطةَ في عمودٍ ولا قيمةَ تُحمل مع
// الخدمة عند الإقلاع. **ومفتاحٌ يُقرأ مرّةً عند البدء يجعل المالكَ يغيّر
// الرقمَ ويرى القديمَ يعمل** — فيغيّره ثانيةً وثالثة.
func amountFrom(ctx context.Context, st Store, modeKey, valueKey string) Amount {
	// **وبلا مخزنٍ لا مال.**
	//
	// **ورقمٌ يُخترع هنا يخترع ديناً**: خدمةٌ تُبنى بلا مخزن (اختبارٌ أو إقلاعٌ
	// نصفُ مهيَّأ) تقتطع عمولةً لم يقرّرها أحد، **ولا يظهر ذلك إلّا في كشف حساب.**
	if st == nil {
		return Amount{Mode: "percent"}
	}
	return Amount{
		Mode:  st.GetString(ctx, modeKey),
		Value: st.GetInt(ctx, valueKey),
	}
}

// MerchantCommission عمولةُ المنصة من المتجر — **تُقتطع من سعر شرائه.**
//
// **والمتجرُ قد يُخصّ بنسبةٍ غير العامّة** (`merchants.commission_percent`
// في صفّه): تجاوزٌ مثلُ تجاوز هامش الصنف — **وفراغُه «اتبع العام» لا «بلا
// عمولة».** ومن خلط بينهما جعل كلَّ متجرٍ لم يُلمس بلا عمولة.
func MerchantCommission(ctx context.Context, st Store, override *int64) Amount {
	if override != nil {
		return Amount{Mode: "percent", Value: *override}
	}
	return amountFrom(ctx, st, "merchants.commission_mode", "merchants.commission_value")
}

// RepCommission عمولةُ المندوب من عمولة المنصة — **حصّةٌ من حصّتنا لا من البيع.**
func RepCommission(ctx context.Context, st Store) Amount {
	return amountFrom(ctx, st, "sales.commission_mode", "sales.commission_value")
}

// ── أجرةُ التوصيل — ثلاثةُ أنماطٍ ومصدرٌ واحد ──────────────────────────────
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «أجرةُ التوصيل حسب المسافة · مقطوعةٌ رقمٌ ثابت»
// ثمّ «الاثنان معاً — ثلاثةُ أنماط».)

// DeliveryMode أنماطُ أجرة التوصيل.
const (
	// DeliveryFlat رقمٌ واحدٌ للجميع.
	DeliveryFlat = "flat"
	// DeliveryZone أجرةُ المنطقة التي يقع فيها الدبّوس — **القائمُ قبل الأنماط.**
	DeliveryZone = "zone"
	// DeliveryDistance أساسٌ زائدَ رقمٍ لكلّ كيلومتر.
	DeliveryDistance = "distance"
)

// DeliveryRule ما يلزم لحساب أجرة التوصيل.
type DeliveryRule struct {
	Mode  string
	Flat  int64
	Base  int64
	PerKm int64
	// Rounding خانةُ التقريب — **وهي خانةُ التسعير نفسُها.**
	//
	// **ولا خانتان**: قائمةٌ تُقرَّب إلى الخمسمئة وتوصيلٌ بـ٧٬٠٢٠ يجعل المجموعَ
	// رقماً لا يُحسب في الجيب — **وهو ما التقريبُ كلُّه من أجله.**
	Rounding int64
}

// DeliveryFrom يقرأ مفاتيح التوصيل.
//
// **وبلا مخزنٍ يبقى نمطُ المنطقة** — هو ما كانت المنصةُ تعمل به قبل الأنماط،
// **ولا يُخترع سلوكٌ جديدٌ لغياب إعداد.**
func DeliveryFrom(ctx context.Context, st Store) DeliveryRule {
	if st == nil {
		return DeliveryRule{Mode: DeliveryZone}
	}
	return DeliveryRule{
		Mode:     st.GetString(ctx, "delivery.fee_mode"),
		Flat:     st.GetInt(ctx, "delivery.flat_fee"),
		Base:     st.GetInt(ctx, "delivery.base_fee"),
		PerKm:    st.GetInt(ctx, "delivery.per_km"),
		Rounding: st.GetInt(ctx, "pricing.rounding"),
	}
}

// Fee أجرةُ التوصيل — `zoneFee` أجرةُ المنطقة، و`meters` أبعدُ مصدرٍ عن الزبون.
//
// **والمسافةُ أبعدُ مصدرٍ لا أقربُه**: السائقُ يمرّ عليها كلِّها، **وأقربُها
// يجعل طلباً من طرفَي المدينة بأجرة الجار.** ورسمُ الوقفة الزائدة محسوبٌ
// وحدَه (`extraSourceFee`) — **هذا ثمنُ الطريق وذاك ثمنُ الوقفة.**
func (r DeliveryRule) Fee(zoneFee int64, meters float64) int64 {
	var fee int64
	switch r.Mode {
	case DeliveryFlat:
		fee = r.Flat
	case DeliveryDistance:
		km := meters / 1000
		fee = r.Base + int64(km*float64(r.PerKm))
	default:
		fee = zoneFee
	}
	if fee < 0 {
		return 0
	}
	return roundTo(fee, r.Rounding)
}
