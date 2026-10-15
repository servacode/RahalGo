package settings_test

// **حارسُ الليرة الجديدة** — لا افتراضَ ماليٌّ بالليرة القديمة.
//
// # المرض
//
// **تحوّلت المنصّةُ إلى الليرة الجديدة ٢٠٢٦-١٠-٠٩** (١٠٠٠ قديمة = ١٠ جديدة)، **وبقيت
// افتراضاتُ الفهرس بالقديمة**: سقفُ نقد السائق خمسمئةُ ألف، وسقفُ التعويض خمسون ألفاً،
// وأقلُّ سحبٍ خمسون ألفاً. **والإنتاجُ يعمل بالافتراض لكلّ مفتاحٍ لم يضبطه المالك** —
// فكان كلُّ واحدٍ منها مبلغاً أكبرَ بمئة مرّةٍ ممّا يُقصد. **ولا يظهر ذلك في أيّ خطأ.**
//
// # والحدّان
//
//	الافتراضُ ≤ ١٠٬٠٠٠   أكبرُ افتراضٍ مشروعٍ اليوم ٥٬٠٠٠ (سقفُ النقد وسقفُ المحفظة
//	                      والمصروف). **وأصغرُ افتراضٍ قديمٍ كان يُرى ٥٠٬٠٠٠** — فيسقط هنا.
//	الحدُّ الأعلى ≤ ١٬٠٠٠٬٠٠٠  أكبرُ سقفٍ مشروعٍ بالجديدة (مئةُ مليونٍ بالقديمة) —
//	                      **وسقوفُ القديمة كانت ١٠ ملايين و١٠٠ مليون** فتسقط هنا.
//
// **ومن احتاج رقماً فوقهما فليرفع الحدَّ هنا بقرارٍ مكتوب** — لا أن يُعيد رقماً قديماً.

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

const (
	newLiraDefaultBound int64 = 10_000
	newLiraMaxBound     int64 = 1_000_000
)

// isMoneyDef **أمبلغٌ هو؟** — نوعُه ماليٌّ أو وحدتُه العملة (`merchants.max_item_price` عدديٌّ بوحدة العملة).
func isMoneyDef(d settings.Def) bool {
	return d.Kind == settings.KindMoney || d.Unit == "currency"
}

func TestMoneyDefaultsInNewLira(t *testing.T) {
	seen := 0
	for _, d := range settingsCatalog() {
		if !isMoneyDef(d) {
			continue
		}
		seen++
		def := settings.Default(d.Key)
		if def > newLiraDefaultBound {
			t.Errorf("**افتراضُ %s = %d — بالليرة القديمة؟** الحدُّ %d بالجديدة.",
				d.Key, def, newLiraDefaultBound)
		}
		if int64(d.Max) > newLiraMaxBound {
			t.Errorf("**الحدُّ الأعلى لـ%s = %v — بالليرة القديمة؟** الحدُّ %d بالجديدة.",
				d.Key, d.Max, newLiraMaxBound)
		}
		if int64(d.Min) > def || def > int64(d.Max) {
			t.Errorf("افتراضُ %s = %d خارجَ [%v, %v]", d.Key, def, d.Min, d.Max)
		}
	}
	// **وحارسٌ لا يرى شيئاً لا يحرس شيئاً.**
	if seen < 30 {
		t.Fatalf("رأى الحارسُ %d مفتاحاً ماليّاً فقط — تغيّر شكلُ الفهرس؟", seen)
	}

	// **وثمنُ بضاعة الطلب الخاصّ ثابتٌ في الشيفرة لا في الفهرس** — ويُحرس معه.
	if orders.CustomGoodsMax > newLiraMaxBound {
		t.Errorf("**`orders.CustomGoodsMax` = %d — بالليرة القديمة؟**", orders.CustomGoodsMax)
	}
}
