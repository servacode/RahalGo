package support

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **ولا سببَ متجرٍ في طلبٍ بلا متجر**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٣: «أصلحها» — بعد أن قيس أنّ «المتجر أوقفني
//
//	طويلاً» تُعرض في طلبٍ خاصٍّ لا متجرَ فيه.)
//
// # ولماذا يُحرَس
//
// **سببٌ جديدٌ يُضاف على المتجر** يدخل قائمةَ الخاصّ صامتاً — **فيُعرض
// سؤالٌ عمّا لا وجودَ له**، ومن اختاره فُتح بلاغٌ بلا مشتكًى عليه:
// **تذكرةٌ تذهب إلى العمليات ولا أحدَ فيها.**
//
// **ولا يظهر ذلك في خطأ** — البلاغُ يُفتح ويُقرأ ويُغلق، والجهةُ فارغة.

func TestCustomOrderHasNoMerchantReasons(t *testing.T) {
	list := ReasonsForKind("custom")
	if len(list) == 0 {
		t.Fatal("لا سببَ للخاصّ — أفُرّغت القائمة؟")
	}
	for _, r := range list {
		if r.Against == AgainstMerchant {
			t.Errorf("السببُ %q على المتجر ويُعرض في طلبٍ خاصّ — لا متجرَ فيه", r.Code)
		}
	}
	// **ويبقى ما على الزبون** — وإلّا صار الخاصُّ بلا بابِ بلاغ.
	customer := 0
	for _, r := range list {
		if r.Against == AgainstCustomer {
			customer++
		}
	}
	if customer == 0 {
		t.Error("لا سببَ على الزبون في الخاصّ — أُغلق بابُ البلاغ كلُّه")
	}
}

// TestStandardOrderKeepsEveryReason **والعاديُّ لا يُقصّ منه شيء.**
func TestStandardOrderKeepsEveryReason(t *testing.T) {
	if len(ReasonsForKind("standard")) != len(DriverReportReasons) {
		t.Error("قُصّت أسبابُ الطلب العاديّ — والمتجرُ فيه قائم")
	}
	// **وبلا نوعٍ تُردّ كلُّها** — نداءٌ قديمٌ لا يرسل النوعَ لا ينكسر.
	if len(ReasonsForKind("")) != len(DriverReportReasons) {
		t.Error("نداءٌ بلا نوعٍ قُصّت أسبابُه — وهو ما يرسله عميلٌ قديم")
	}
}

// TestAllowedForKindGuardsTheOpening **والمنعُ عند الفتح لا عند العرض.**
//
// **وقائمةٌ تُصفّى في الشاشة وحدَها لا تمنع من ينادي الواجهةَ مباشرة** —
// وهي عائلةُ «قاعدةٌ تُطبَّق في العرض» التي تكرّرت في هذا المشروع.
func TestAllowedForKindGuardsTheOpening(t *testing.T) {
	if AllowedForKind("custom", "merchant_slow") {
		t.Error("يُقبل سببُ متجرٍ في طلبٍ خاصّ — والمنعُ في الشاشة وحدَها ليس منعا")
	}
	if !AllowedForKind("custom", "customer_absent") {
		t.Error("يُرفض سببُ زبونٍ في طلبٍ خاصّ — وهو ما يقع فيه فعلا")
	}
	if !AllowedForKind("standard", "merchant_slow") {
		t.Error("يُرفض سببُ متجرٍ في طلبٍ عاديّ")
	}
}
