package orders_test

// ══════════════════════════════════════════════════════════════════════
//  **خصمٌ بمبلغٍ ثابت — يبلغ الدفترَ كما تبلغه النسبة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٣٠: «ابنِ ميزة إنشاء عرض بمبلغ ثابت أيضاً».)
//
// # وأخطرُ ما في هذه الميزة
//
// **مسارُ المال كان يقرأ `LiveDiscount`** — وهي تردّ **النسبةَ** وحدَها.
// **فعرضٌ بمبلغٍ ثابتٍ يمرّ عليها بصفر**: **الشاشةُ تعرض السعرَ مخفوضاً
// والطلبُ يُبنى بالسعر كاملاً** — **فيرى الزبونُ رقماً ويُحاسَب بآخر.**
//
// **وذلك عطبٌ صامتٌ لا يسقط فيه اختبارٌ قائم**: كلُّ الحرّاس القديمة
// تمرّر النسبةَ، **فتبقى خضراء والميزةُ الجديدةُ مكسورة.**
//
// # فهذه تقيس الدفترَ لا الشاشة
//
// **وتقرأ المالَ من الصفّ** (`orderMoney`) لا من الكائن الذي ردّته
// الدالّة — **فما لم يُكتب في القاعدة لم يقع.**

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/offers"
)

// ── أ · تتحمّله المنصّة: ينزل ما يدفعه الزبون، والشراءُ كما هو ────────
func TestFixedAmount_PlatformBorne_CutsExactlyTheAmount(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)

	base := f.placeOne(t, itemID, "بلا خصم")
	baseSub, baseTotal, baseUnit, baseMerchant := f.orderMoney(t, base.ID)

	// **ومبلغٌ أصغرُ من السعر بيقين** — وإلّا قُصّ إلى السعر كلِّه
	// فصار الاختبارُ يقيس القصَّ لا الخصم.
	const amount int64 = 500
	if baseUnit <= amount {
		t.Fatalf("سعرُ الوحدة %d لا يتّسع لخصمِ %d — عتادٌ لا يقيس", baseUnit, amount)
	}

	f.svc.SetOffers(stubDiscount{itemID: itemID, amount: amount, borneBy: offers.ByPlatform})
	t.Cleanup(func() { f.svc.SetOffers(nil) })

	cut := f.placeOne(t, itemID, "بخصمٍ ثابتٍ من المنصة")
	cutSub, cutTotal, cutUnit, cutMerchant := f.orderMoney(t, cut.ID)

	t.Logf("بلا خصم: وحدة=%d مجموع=%d كلّي=%d شراء=%d", baseUnit, baseSub, baseTotal, baseMerchant)
	t.Logf("بخصمٍ ثابتٍ %d: وحدة=%d مجموع=%d كلّي=%d شراء=%d",
		amount, cutUnit, cutSub, cutTotal, cutMerchant)

	// **والمبلغُ يُطرح كما هو — لا بنسبةٍ تُشتقّ منه.**
	if want := baseUnit - amount; cutUnit != want {
		t.Fatalf("سعرُ الوحدة %d والمنتظَر %d — **الخصمُ الثابتُ لم يبلغ اللقطة**",
			cutUnit, want)
	}
	if cutSub >= baseSub {
		t.Fatalf("المجموعُ %d ولم ينزل عن %d — **خصمٌ يُعرض ولا يُطبَّق**", cutSub, baseSub)
	}
	if cutTotal >= baseTotal {
		t.Fatalf("الكلّيُّ %d ولم ينزل عن %d — **والزبونُ يدفع ثمنَ ما وُعد بحسمِه**",
			cutTotal, baseTotal)
	}
	// **وسعرُ الشراء لا يُمسّ** — المنصّةُ تحمّلت، والمتجرُ يقبض كاملاً.
	if cutMerchant != baseMerchant {
		t.Errorf("سعرُ الشراء %d وكان %d — **حُمّل المتجرُ خصماً تحمّلته المنصّة**",
			cutMerchant, baseMerchant)
	}
}

// ── ب · يتحمّله المتجر: ينزل الشراءُ بالمقدار نفسِه ──────────────────
//
// **وهو ما يجعل الهامشَ كما هو** — والفرقُ بين الطريقتين هنا أدقُّ ما
// في الميزة: **المقدارُ واحدٌ في الجانبين**، **ولا نسبةَ تُشتقّ.**
func TestFixedAmount_MerchantBorne_CutsBothSidesByTheSameAmount(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)

	base := f.placeOne(t, itemID, "بلا خصم")
	_, _, baseUnit, baseMerchant := f.orderMoney(t, base.ID)

	const amount int64 = 500
	if baseUnit <= amount || baseMerchant <= amount {
		t.Fatalf("عتادٌ لا يقيس: وحدة=%d شراء=%d وخصم=%d", baseUnit, baseMerchant, amount)
	}

	f.svc.SetOffers(stubDiscount{itemID: itemID, amount: amount, borneBy: offers.ByMerchant})
	t.Cleanup(func() { f.svc.SetOffers(nil) })

	cut := f.placeOne(t, itemID, "بخصمٍ ثابتٍ من المتجر")
	_, _, cutUnit, cutMerchant := f.orderMoney(t, cut.ID)

	t.Logf("بلا خصم: وحدة=%d شراء=%d", baseUnit, baseMerchant)
	t.Logf("بخصمٍ ثابتٍ %d من المتجر: وحدة=%d شراء=%d", amount, cutUnit, cutMerchant)

	if want := baseUnit - amount; cutUnit != want {
		t.Fatalf("سعرُ الوحدة %d والمنتظَر %d", cutUnit, want)
	}
	if want := baseMerchant - amount; cutMerchant != want {
		t.Fatalf("سعرُ الشراء %d والمنتظَر %d — **المتجرُ لم يتحمّل ما وُعد به**",
			cutMerchant, want)
	}
	// **والهامشُ كما هو** — وهو عقدُ «يتحمّله المتجر» حرفاً.
	if (baseUnit - baseMerchant) != (cutUnit - cutMerchant) {
		t.Errorf("الهامشُ كان %d وصار %d — **وعقدُ «يتحمّله المتجر» أن يبقى**",
			baseUnit-baseMerchant, cutUnit-cutMerchant)
	}
}

// ── ج · ولا يُنزَل السعرُ تحت الصفر ──────────────────────────────────
//
// **ومبلغٌ أكبرُ من السعر يقع فعلاً**: العرضُ يُنشأ اليومَ بمبلغٍ ثابتٍ،
// **ثمّ يخفض المتجرُ سعرَ الصنف غداً** — فيصير الخصمُ أكبرَ من الثمن.
// **وسعرٌ سالبٌ يعني أنّ المنصّة تدفع للزبون ليشتري.**
func TestFixedAmount_NeverBelowZero(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)

	base := f.placeOne(t, itemID, "بلا خصم")
	_, _, baseUnit, _ := f.orderMoney(t, base.ID)

	// **أكبرُ من السعر بيقين.**
	huge := baseUnit * 10
	f.svc.SetOffers(stubDiscount{itemID: itemID, amount: huge, borneBy: offers.ByPlatform})
	t.Cleanup(func() { f.svc.SetOffers(nil) })

	cut := f.placeOne(t, itemID, "بخصمٍ يفوق السعر")
	cutSub, cutTotal, cutUnit, _ := f.orderMoney(t, cut.ID)

	t.Logf("سعرُ الوحدة %d وخصمٌ %d ⇒ وحدة=%d مجموع=%d كلّي=%d",
		baseUnit, huge, cutUnit, cutSub, cutTotal)

	if cutUnit != 0 {
		t.Fatalf("سعرُ الوحدة %d — **والمنتظَر صفرٌ لا أقلّ ولا أكثر**", cutUnit)
	}
	if cutSub < 0 || cutTotal < 0 {
		t.Fatalf("مجموعٌ %d وكلّيٌّ %d — **رقمٌ سالبٌ في دفتر**", cutSub, cutTotal)
	}
}

// ── د · والحسبةُ نفسُها بلا قاعدة — حدودُ `Cut` ────────────────────────
//
// **وهي أسرعُ من أن تُمشى بطلبٍ كامل**، وتقيس ما لا يبلغه الطلبُ سهلاً:
// **الفارغَين، والسالب، والصفر، والنسبةَ والمبلغَ معاً.**
func TestCut_Boundaries(t *testing.T) {
	p := func(v int) *int { return &v }
	a := func(v int64) *int64 { return &v }

	cases := []struct {
		name    string
		price   int64
		percent *int
		amount  *int64
		want    int64
	}{
		{"لا خصمَ فلا قصّ", 1000, nil, nil, 0},
		{"نسبةٌ عاديّة", 1000, p(20), nil, 200},
		{"مبلغٌ عاديّ", 1000, nil, a(250), 250},
		{"مبلغٌ يفوق السعر يُقصّ إليه", 1000, nil, a(5000), 1000},
		{"نسبةٌ صفرٌ لا تخصم", 1000, p(0), nil, 0},
		{"مبلغٌ صفرٌ لا يخصم", 1000, nil, a(0), 0},
		{"مبلغٌ سالبٌ لا يرفع السعر", 1000, nil, a(-500), 0},
		{"سعرٌ صفرٌ لا يُخصم منه", 0, p(50), nil, 0},
		{"سعرٌ سالبٌ لا يُخصم منه", -100, nil, a(50), 0},
		// **والنسبةُ تسبق إن حضرا معاً** — **حالةٌ يمنعها قيدُ القاعدة**،
		// **والحسبةُ لا تفترض سلامتَها**: جوابٌ واحدٌ معروفٌ خيرٌ من سلوكٍ
		// يتبدّل بترتيبِ شرطٍ يُعاد كتابتُه يوماً.
		{"الاثنان معاً: النسبةُ تسبق", 1000, p(10), a(900), 100},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := offers.Cut(c.price, c.percent, c.amount)
			if got != c.want {
				t.Fatalf("Cut(%d) = %d والمنتظَر %d", c.price, got, c.want)
			}
			after := offers.AfterCut(c.price, c.percent, c.amount)
			if after != c.price-c.want {
				t.Fatalf("AfterCut = %d والمنتظَر %d", after, c.price-c.want)
			}
		})
	}
}

// ── هـ · والنسبةُ لم تتبدّل — `AfterDiscount` كما كانت ────────────────
//
// **وحارسُ ارتدادٍ صريح**: الميزةُ الجديدةُ أعادت كتابةَ المسار،
// **ومن أعاد كتابتَه فليُثبت أنّ القديمَ لم يتحرّك.**
func TestFixedAmount_PercentPathUnchanged(t *testing.T) {
	for _, price := range []int64{1, 99, 100, 1000, 33_333, 1_000_000} {
		for _, pct := range []int{1, 7, 20, 50, 90} {
			old := offers.AfterDiscount(price, pct)
			p := pct
			neu := offers.AfterCut(price, &p, nil)
			if old != neu {
				t.Fatalf("سعر=%d نسبة=%d: القديمُ %d والجديدُ %d — **المسارُ تحرّك**",
					price, pct, old, neu)
			}
		}
	}
}
