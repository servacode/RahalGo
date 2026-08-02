package pricing

// التسعير — **من سعر الشراء إلى سعر البيع.**
//
// # النموذج
//
//	سعرُ الشراء  ←  ما وضعه المتجر — وهو ما يقبضه
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
	GetString(ctx context.Context, key, fallback string) string
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
		Mode:     st.GetString(ctx, "pricing.margin_mode", "percent"),
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
