package server

// **مصدرُ البضاعة لا يخرج إلى الزبون — ولا في طلباته.**
//
// # لماذا هذا الاختبار بالذات
//
// `TestBrowse_HidesSource` يحرس التصفّح منذ المرحلة الثانية، **وطلباتُ الزبون
// لم تُحرَس قطّ.** فما حُجب في `/public` ظهر في `/my/orders` — **وهو الموضعُ
// الذي يفتحه الزبونُ أكثر**، ويفتحه بعد أن يعرف أنّ الطلب وصل.
//
// **وشهده المالكُ على شاشته** (٢٠٢٦-٠٨-٠٣) لا اختبارٌ ولا مراجعة: «مازال اسم
// المتجر يظهر للزبون».
//
// **والإخفاءُ في الشاشة لا يكفي**: من فتح أدوات المتصفّح قرأ الردَّ كما هو،
// **ومعرّفٌ في الردّ يُفتح به `/public/merchants/{id}` فيُقرأ الاسمُ كاملاً.**
//
// **وحقلٌ يُضاف يوماً بلا انتباه يهدم هذا كلَّه** — ولا يظهر في أيّ خطأ.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/offers"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestRedactForCustomer_HidesSource **الاسمُ والمعرّفُ والشعار — ثلاثتُها.**
//
// **والشعارُ كالاسم**: صورةُ مطعمٍ يعرفه أهلُ الحيّ **تُعرف قبل أن تُقرأ
// الكلمة**، فحجبُ الاسم وحدَه حجبٌ ناقصٌ يُطمئن ولا يحمي.
func TestRedactForCustomer_HidesSource(t *testing.T) {
	logo := "2026/08/abc_t.png"
	o := &orders.Order{
		ID:                "o-1",
		Number:            1001,
		MerchantID:        "cd6ea11d-9cd4-4a13-9f1c-835470cb6471",
		MerchantName:      "مطعم بيت الرقة",
		MerchantLogoThumb: &logo,
		// وما يبقى — **الطلبُ نفسُه**
		Total:        63000,
		Status:       "pending",
		ItemsPreview: "شاورما دجاج",
	}
	redactForCustomer(o)

	if o.MerchantName != "" {
		t.Errorf("اسمُ المتجر خرج إلى الزبون: %q", o.MerchantName)
	}
	if o.MerchantID != "" {
		t.Errorf("معرّفُ المتجر خرج — ويُفتح به /public/merchants/{id}: %q", o.MerchantID)
	}
	if o.MerchantLogoThumb != nil {
		t.Errorf("شعارُ المتجر خرج — والصورةُ تُعرف قبل الكلمة: %q", *o.MerchantLogoThumb)
	}

	// **ولا يُمحى ما ليس مصدراً**: حراسةٌ تمحو الطلبَ نفسَه تُطفأ في أوّل شكوى.
	if o.Total == 0 || o.Status == "" || o.ItemsPreview == "" {
		t.Error("مُحي من الطلب ما لا يدلّ على مصدره — والحراسةُ الزائدة تُطفأ")
	}
}

// TestCustomerOrderJSON_HasNoSource **الفحصُ على النصّ المرسَل لا على الحقول.**
//
// حقلٌ جديدٌ يُضاف غداً باسم المتجر — `merchant_slug`, `store_name` — **يمرّ من
// فحصِ حقولٍ بأعيانها ولا يمرّ من فحصِ النصّ.**
func TestCustomerOrderJSON_HasNoSource(t *testing.T) {
	const name = "مطعم بيت الرقة"
	logo := "2026/08/abc_t.png"
	o := &orders.Order{
		MerchantID:   "cd6ea11d-9cd4-4a13-9f1c-835470cb6471",
		MerchantName: name, MerchantLogoThumb: &logo,
		CustomerName: "سليمان الخطيب", AddressText: "شارع تل أبيض",
	}
	redactForCustomer(o)

	b, err := json.Marshal(o)
	if err != nil {
		t.Fatalf("ترميز الطلب: %v", err)
	}
	body := string(b)
	for _, leak := range []string{name, "cd6ea11d", "abc_t.png"} {
		if strings.Contains(body, leak) {
			t.Errorf("خرج في الردّ ما يدلّ على المصدر: %q\nالردّ: %s", leak, body)
		}
	}
}

// TestRedactOffersForCustomer_HidesSource **وثالثةُ الأبواب.**
//
// حُجب التصفّحُ ثمّ الطلبات — **وبقيت العروضُ تحمل `merchant_name`
// و`merchant_id`**، وشاشتُها تعرضهما تحت اسم الصنف. **ومن أراد اسمَ المطعم
// يفتح صفحةَ العروض**، فالبابان المغلقان لا يُغنيان عن ثالثٍ مفتوح.
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «المتاجرُ مخفيّةٌ بشكلٍ كاملٍ عن الزبون — نحن
// بالمنصة عندنا السوقُ الذي يجلب من المتاجر».)
func TestRedactOffersForCustomer_HidesSource(t *testing.T) {
	const name = "مطعم بيت الرقة"
	mid := "cd6ea11d-9cd4-4a13-9f1c-835470cb6471"
	rows := []offers.Offer{{
		Title: "خصمُ الجمعة", ItemName: "شاورما دجاج",
		MerchantName: name, MerchantID: &mid,
		PriceBefore: 30000, PriceAfter: 24000,
	}}
	redactOffersForCustomer(rows)

	b, err := json.Marshal(rows)
	if err != nil {
		t.Fatalf("ترميزُ العروض: %v", err)
	}
	body := string(b)
	for _, leak := range []string{name, "cd6ea11d"} {
		if strings.Contains(body, leak) {
			t.Errorf("خرج في العروض ما يدلّ على المصدر: %q · الردّ: %s", leak, body)
		}
	}

	// **ولا يُمحى العرضُ نفسُه** — السعران واسمُ الصنف هما ما يُغري، وحراسةٌ
	// تمحوهما تُطفأ في أوّل شكوى.
	if rows[0].PriceAfter == 0 || rows[0].ItemName == "" || rows[0].Title == "" {
		t.Error("مُحي من العرض ما لا يدلّ على مصدره — والحراسةُ الزائدة تُطفأ")
	}
}
