package orders_test

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestDriverView_MasksContactsInCustomerText — **ما كتبه الزبونُ بيده لا يحمل رقمَه إلى
// السائق** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ١ج): الملاحظاتُ ونصُّ الطلب الخاصّ والعنوانُ
// وملاحظةُ الصنف تُقرأ مُخفاةً للسائق، **والمكتبُ يقرأ الأصل.** و«لدي توصيلة» لا تُمسّ:
// المستلِمُ ورقمُه عند السائق بالتصميم.
func TestDriverView_MasksContactsInCustomerText(t *testing.T) {
	o := &orders.Order{
		Kind:          "custom",
		Notes:         "اتصل فيني 0991234567",
		CustomRequest: "هات خبز وكلمني عالواتس",
		AddressText:   "بناية الأمل +963 991 234 567",
		Items:         []orders.OrderItem{{Name: "فلافل", Qty: 1, Note: "رقمي ٠٩٩١٢٣٤٥٦٧"}},
	}
	v := orders.ViewFor(orders.AudienceDriver, o)
	for _, k := range []string{"notes", "custom_request", "address_text"} {
		s, _ := v[k].(string)
		if strings.Contains(s, "0991234567") || strings.Contains(s, "991 234") || strings.Contains(s, "واتس") {
			t.Errorf("driver %s = %q, want masked", k, s)
		}
		if !strings.Contains(s, "***") {
			t.Errorf("driver %s = %q, want ***", k, s)
		}
	}
	items, _ := v["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("driver items = %v", v["items"])
	}
	if it, _ := items[0].(map[string]any); it["note"] != "رقمي ***" {
		t.Errorf("driver item note = %v, want masked", it["note"])
	}
	// **والأصلُ للمكتب** — ولا تُمسّ الحمولةُ الأصليّة.
	ops := orders.ViewFor(orders.AudienceOps, o)
	if ops["notes"] != "اتصل فيني 0991234567" || o.Notes != "اتصل فيني 0991234567" {
		t.Errorf("ops notes = %v (order %q), want original", ops["notes"], o.Notes)
	}
	if o.Items[0].Note != "رقمي ٠٩٩١٢٣٤٥٦٧" {
		t.Errorf("order item note mutated: %q", o.Items[0].Note)
	}
	// **و«لدي توصيلة»** — ملاحظةُ صاحب المتجر لسائقه كما كتبها.
	md := &orders.Order{Kind: orders.KindMerchantDelivery, Notes: "اتصل بالمستلم 0991234567"}
	if got := orders.ViewFor(orders.AudienceDriver, md)["notes"]; got != "اتصل بالمستلم 0991234567" {
		t.Errorf("merchant_delivery notes = %v, want untouched", got)
	}
}
