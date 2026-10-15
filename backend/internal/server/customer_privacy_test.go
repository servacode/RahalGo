package server

// **مصدرُ البضاعة لا يخرج إلى الزبون — ولا في طلباته.**
//
// # لماذا هذا الاختبار بالذات
//
// (والتصفّحُ والعروضُ صارا يُخرجان اسمَ المتجر بقرار المالك ٢٠٢٦-١٠-١٥ —
// `TestBrowse_ShowsStoreName`. **والطلباتُ باقيةٌ محجوبة.**)
//
// `TestBrowse_HidesSource` كان يحرس التصفّح منذ المرحلة الثانية، **وطلباتُ الزبون
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
	// **والمُقاسُ هو المرشَّحُ الذي يخرج منه الردُّ فعلاً** — **لا
	// محوٌ على الكائن**: `redactForCustomer` رُفعت في دورةِ ٤٦.
	v := orders.ViewFor(orders.AudienceCustomer, o)
	if v == nil {
		t.Fatal("تعذّر بناءُ حمولة الزبون")
	}

	for _, k := range []string{"merchant_name", "merchant_id", "merchant_logo_thumb_url"} {
		if _, ok := v[k]; ok {
			t.Errorf("%s خرج إلى الزبون — والمصدرُ محجوب: %v", k, v[k])
		}
	}

	// **ولا يُمحى ما ليس مصدراً**: حراسةٌ تمحو الطلبَ نفسَه تُطفأ في أوّل شكوى.
	for _, k := range []string{"total", "status", "items_preview"} {
		if _, ok := v[k]; !ok {
			t.Errorf("مُحي %q ولا يدلّ على المصدر — والحراسةُ الزائدة تُطفأ", k)
		}
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
	v := orders.ViewFor(orders.AudienceCustomer, o)
	if v == nil {
		t.Fatal("تعذّر بناءُ حمولة الزبون")
	}

	b, err := json.Marshal(v)
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

// TestRedactOffersForCustomer_KeepsStoreName **اسمُ المتجر يبقى في العروض.**
//
// كانت العروضُ ثالثةَ أبواب الحجب، **وعكس المالكُ القرارَ ٢٠٢٦-١٠-١٥**:
// الزبونُ يرى اسمَ المتجر تحت اسم الصنف كما في شبكة التصفّح. **وما يخصّ
// الإدارةَ** — من عمل العرضَ وحالُ موافقته — **يبقى ممسوحاً.**
func TestRedactOffersForCustomer_KeepsStoreName(t *testing.T) {
	const name = "مطعم بيت الرقة"
	mid := "cd6ea11d-9cd4-4a13-9f1c-835470cb6471"
	rows := []offers.Offer{{
		Title: "خصمُ الجمعة", ItemName: "شاورما دجاج",
		MerchantName: name, MerchantID: &mid,
		PriceBefore: 30000, PriceAfter: 24000,
		CreatedByName: "موظفُ العروض", CreatedByKind: "staff", ApprovalState: "approved",
	}}
	redactOffersForCustomer(rows)

	if rows[0].MerchantName != name || rows[0].MerchantID == nil || *rows[0].MerchantID != mid {
		t.Errorf("مُسح مصدرُ العرض — والقرارُ أن يظهر: %q · %v", rows[0].MerchantName, rows[0].MerchantID)
	}
	if rows[0].CreatedByName != "" || rows[0].CreatedByKind != "" || rows[0].ApprovalState != "" {
		t.Errorf("خرج إلى الزبون ما يخصّ الإدارة: %+v", rows[0])
	}
	if rows[0].PriceAfter == 0 || rows[0].ItemName == "" || rows[0].Title == "" {
		t.Error("مُحي من العرض ما لا يخصّ الإدارة")
	}
}
