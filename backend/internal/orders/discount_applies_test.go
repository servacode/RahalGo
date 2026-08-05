package orders_test

// **الخصمُ يُطبَّق في الدفتر لا في الشاشة.**
//
// # لماذا وُجد هذا الاختبار
//
// **سعرٌ مشطوبٌ وشارةُ «−٢٠٪» تُرسمان في المتصفّح من حقلين يرسلهما الخادم**
// — ولا شيء فيهما يُلزم `orders.subtotal` بأن ينزل. **فالعرضُ قد يُغري
// والفاتورةُ تأتي كاملةً**، ولا يكتشفها الزبونُ إلّا وقد دفع.
//
// **وشرطَه المالكُ صراحةً** (٢٠٢٦-٠٨-٠٥): «لازم تتأكّد أنّ الخصم يُطبَّق
// فعلاً وليس فقط وهميّاً».
//
// # ولم يمسّه اختبارٌ قطّ
//
// `s.offers` **نِيلٌ في عُدّة الاختبارات كلِّها** — و`Create` تتخطّى كتلةَ
// الخصم كاملةً حين يكون نِيلاً. **فالمسارُ الذي يخرج منه المالُ لم يُشغَّل
// مرّةً واحدةً في المجموعة.** وهذه العُدّةُ تحقن القارئ.
//
// # وما يُقاس
//
// **`subtotal` من القاعدة لا الردُّ من الدالّة**: الردُّ يُبنى في الذاكرة،
// **والتسويةُ والمحفظةُ والصندوقُ تقرأ الصفَّ.** ولو نزل الردُّ وحدَه لَظهر
// الخصمُ للزبون **وحُسب الطلبُ كاملاً في كلّ حسبةٍ بعده.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// stubDiscount قارئُ خصمٍ ثابت — **يردّ للصنف المقصود ولا شيءَ لغيره.**
type stubDiscount struct {
	itemID  string
	percent int
	borneBy string
}

func (d stubDiscount) LiveDiscount(_ context.Context, menuItemID string) (int, string) {
	if menuItemID != d.itemID {
		return 0, ""
	}
	return d.percent, d.borneBy
}

// orderMoney يقرأ المالَ من الصفّ — **لا من الكائن الذي ردّته الدالّة.**
func (f *fixture) orderMoney(t *testing.T, orderID string) (subtotal, total, unit, merchantPrice int64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT o.subtotal, o.total, i.unit_price, i.merchant_price
		FROM orders o JOIN order_items i ON i.order_id = o.id
		WHERE o.id = $1`, orderID).Scan(&subtotal, &total, &unit, &merchantPrice); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	return
}

// placeOne طلبٌ بصنفٍ واحدٍ وكمّيةِ واحد.
func (f *fixture) placeOne(t *testing.T, itemID, where string) *orders.Order {
	t.Helper()
	o, err := f.svc.Create(context.Background(), f.customer, []string{"customer"}, orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   where,
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
	}, "127.0.0.1")
	if err != nil {
		t.Fatalf("تعذّر إنشاءُ الطلب: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, o.ID)
	})
	return o
}

// TestDiscount_PlatformBorne_CutsWhatTheCustomerPays
//
// **تتحمّله المنصة**: سعرُ البيع ينزل، **وسعرُ الشراء كما هو** — والمتجرُ
// يقبض كاملاً والهامشُ يضيق. **وهو الشكلُ الذي يُخسر فيه مالٌ حقيقيّ**، فلو
// لم ينزل ما يدفعه الزبونُ لَكانت المنصةُ قد أعلنت خصماً ولم تُعطِه.
func TestDiscount_PlatformBorne_CutsWhatTheCustomerPays(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)

	// **الأساسُ يُقاس أوّلاً بلا خصم** — ورقمٌ بلا ما يُقارَن به لا يُثبت
	// نزولاً، **وسعرُ البيع يُحسب من هوامشَ قد تتغيّر** فلا يُكتب رقماً.
	base := f.placeOne(t, itemID, "بلا خصم")
	baseSub, baseTotal, baseUnit, baseMerchant := f.orderMoney(t, base.ID)

	f.svc.SetOffers(stubDiscount{itemID: itemID, percent: 20, borneBy: offers.ByPlatform})
	t.Cleanup(func() { f.svc.SetOffers(nil) })

	cut := f.placeOne(t, itemID, "بخصمِ المنصة")
	cutSub, cutTotal, cutUnit, cutMerchant := f.orderMoney(t, cut.ID)

	t.Logf("بلا خصم: وحدة=%d مجموع=%d كلّي=%d شراء=%d", baseUnit, baseSub, baseTotal, baseMerchant)
	t.Logf("بخصم ٢٠٪: وحدة=%d مجموع=%d كلّي=%d شراء=%d", cutUnit, cutSub, cutTotal, cutMerchant)

	want := offers.AfterDiscount(baseUnit, 20)
	if cutUnit != want {
		t.Fatalf("سعرُ الوحدة %d والمنتظَر %d — **الخصمُ لم يبلغ اللقطة**", cutUnit, want)
	}
	if cutSub >= baseSub {
		t.Fatalf("المجموعُ %d ولم ينزل عن %d — **خصمٌ يُعرض ولا يُطبَّق**", cutSub, baseSub)
	}
	if cutTotal >= baseTotal {
		t.Fatalf("الكلّيُّ %d ولم ينزل عن %d — **والزبونُ يدفع ثمنَ ما وُعد بحسمِه**", cutTotal, baseTotal)
	}
	// **وسعرُ الشراء لا يُمسّ** — المنصةُ هي من تحمّل، والمتجرُ يقبض كاملاً.
	if cutMerchant != baseMerchant {
		t.Errorf("سعرُ الشراء %d وكان %d — **حُمّل المتجرُ خصماً تحمّلته المنصة**",
			cutMerchant, baseMerchant)
	}
}

// TestDiscount_MerchantBorne_CutsBothSides
//
// **يتحمّله المتجر**: السعران ينزلان **بالمقدار نفسِه لا بالنسبة نفسِها** —
// والهامشُ كما هو. **ولو نزل الشراءُ بالنسبة لَتحمّل المتجرُ أقلَّ ممّا وُعد
// به الزبون** والفرقُ يخرج منّا صامتاً.
func TestDiscount_MerchantBorne_CutsBothSides(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)

	base := f.placeOne(t, itemID, "بلا خصم")
	baseSub, _, baseUnit, baseMerchant := f.orderMoney(t, base.ID)

	f.svc.SetOffers(stubDiscount{itemID: itemID, percent: 20, borneBy: offers.ByMerchant})
	t.Cleanup(func() { f.svc.SetOffers(nil) })

	cut := f.placeOne(t, itemID, "بخصمِ المتجر")
	cutSub, _, cutUnit, cutMerchant := f.orderMoney(t, cut.ID)

	t.Logf("بلا خصم: بيع=%d شراء=%d · بخصم: بيع=%d شراء=%d",
		baseUnit, baseMerchant, cutUnit, cutMerchant)

	if cutUnit != offers.AfterDiscount(baseUnit, 20) || cutSub >= baseSub {
		t.Fatalf("سعرُ البيع %d والمجموع %d — **الخصمُ لم يبلغ الدفتر**", cutUnit, cutSub)
	}

	// **والمقدارُ واحدٌ على الجانبين** — فالهامشُ لا يتغيّر.
	saleCut := baseUnit - cutUnit
	buyCut := baseMerchant - cutMerchant
	if buyCut != saleCut {
		t.Fatalf("نزل البيعُ %d ونزل الشراءُ %d — **والفرقُ %d يخرج من هامشِنا بلا قرار**",
			saleCut, buyCut, saleCut-buyCut)
	}
	if base := baseUnit - baseMerchant; cutUnit-cutMerchant != base {
		t.Errorf("الهامشُ صار %d وكان %d — **وخصمُ المتجر لا يمسّ هامشَنا**",
			cutUnit-cutMerchant, base)
	}
}
