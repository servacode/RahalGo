package orders_test

// **الزبونُ يفتح ما شاء من الطلبات — بلا حدّ.**
//
// # الثغرة
//
// حرّاسُ الإنشاء كلُّها على **المحتوى**: عددُ المصادر · صحّةُ الأصناف · حالُ
// المتجر · توثيقُ واتساب · المنطقة. **ولا حارسَ على الكمّ.**
//
// # ولماذا تهمّ في منصّةٍ نقديّة
//
// **الزبونُ لا يدفع شيئاً حتّى يستلم.** فمن أراد الأذى يفتح خمسين طلباً في
// دقيقة **ولا يخسر ليرة**:
//
//	المتجرُ   خمسون طلباً تنتظر قَبوله — **ويعجز عن تمييز الحقيقيّ**
//	الطابورُ  يمتلئ بما لا يُسلَّم
//	السائقون  تُعرض عليهم مشاويرُ وهميّة **فيتركون الدوام**
//	الراصدُ   خمسون إنذارَ «لم يُقبل» تُغرق العمليات
//
// **وتوثيقُ واتساب يحدّ الحساباتِ لا الطلبات** — حسابٌ واحدٌ موثَّقٌ يكفي.
//
// # وليست نظريّة
//
// **مدينةٌ صغيرةٌ ومنافسٌ ومزحةٌ ثقيلة** — والكلفةُ على المتجر لا على من فعل.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestCreate_OpenOrdersAreCapped **وسقفٌ للمفتوح لا للمجموع.**
//
// **والسقفُ على المفتوح لا على ما مضى**: زبونٌ وفيٌّ طلب ألفَ مرّةٍ في سنة
// **لا يُمنع**، ومن بين يديه عشرةٌ لم تُسلَّم **يُسأل قبل الحادي عشر.**
func TestCreate_OpenOrdersAreCapped(t *testing.T) {
	f := setup(t, "pending", 100_000, 10_000, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	itemID := f.armItem(t)
	f.setSetting(t, "orders.max_open_per_customer", 3)

	in := orders.CreateInput{
		CustomerID:    f.customer,
		MerchantID:    f.merchantID,
		Items:         []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
		AddressText:   "إغراقٌ بالطلبات",
		Lat:           35.9528,
		Lng:           39.0079,
		PaymentMethod: "cash",
	}

	made := 0
	var lastErr error
	for i := 0; i < 6; i++ {
		o, err := f.svc.Create(ctx, f.customer, []string{"customer"}, in, "127.0.0.1")
		if err != nil {
			lastErr = err
			continue
		}
		made++
		id := o.ID
		t.Cleanup(func() {
			_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id)
		})
	}

	// **والعُدّةُ تضع طلباً مفتوحاً سلفاً** — فالسقفُ يُبلَغ بعد اثنين.
	t.Logf("نجح %d من ٦ · آخرُ خطأ: %v", made, lastErr)
	if made > 3 {
		t.Fatalf("فُتح %d طلباً والسقفُ ٣ — **والمتجرُ يغرق بما لا يُسلَّم**", made)
	}
	if lastErr == nil {
		t.Fatal("لم يُردَّ طلبٌ واحد — **ولا حارسَ على الكمّ**")
	}
}
