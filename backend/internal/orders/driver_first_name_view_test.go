package orders_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestDriverView_FirstNameOnly — **السائقُ يرى الاسمَ الأوّلَ للزبون وحده** (قرارُ المالك
// ٢٠٢٦-١١-٠٥، البند ٥)، والمكتبُ والزبونُ يريان الاسمَ كاملاً. **و«لدي توصيلة» لا تُمسّ**:
// الاسمُ فيها اسمُ المستلِم (وقد يكون محلّاً) ورقمُه عند السائق أصلاً.
func TestDriverView_FirstNameOnly(t *testing.T) {
	const full = "سليمان أحمد الخطيب"
	o := &orders.Order{Kind: "standard", CustomerName: full}
	if got := orders.ViewFor(orders.AudienceDriver, o)["customer_name"]; got != "سليمان" {
		t.Errorf("driver customer_name = %v, want first name", got)
	}
	if got := orders.ViewFor(orders.AudienceOps, o)["customer_name"]; got != full {
		t.Errorf("ops customer_name = %v, want full", got)
	}
	if got := orders.ViewFor(orders.AudienceCustomer, o)["customer_name"]; got != full {
		t.Errorf("customer customer_name = %v, want full", got)
	}
	custom := &orders.Order{Kind: "custom", CustomerName: "عبد الله الأحمد"}
	if got := orders.ViewFor(orders.AudienceDriver, custom)["customer_name"]; got != "عبد الله" {
		t.Errorf("driver custom customer_name = %v, want compound first name", got)
	}
	md := &orders.Order{Kind: orders.KindMerchantDelivery, CustomerName: "محل الخير للتجارة"}
	if got := orders.ViewFor(orders.AudienceDriver, md)["customer_name"]; got != "محل الخير للتجارة" {
		t.Errorf("driver merchant_delivery recipient = %v, want untouched", got)
	}
}
