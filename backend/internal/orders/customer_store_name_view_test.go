package orders_test

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// اسمُ المتجر يصل الزبونَ في طلب المحلّ الواحد (قرارُ المالك ٢٠٢٦-١١-١١)،
// ولا يصله في الطلب بعدّة محطّات، ولا يصله معه معرّفُ المتجر ولا شعارُه.
func TestCustomerSeesStoreNameOnlyForSingleStore(t *testing.T) {
	single := &orders.Order{MerchantID: "m1", MerchantName: "مطعم الشام"}
	v := orders.ViewFor(orders.AudienceCustomer, single)
	if v["merchant_name"] != "مطعم الشام" {
		t.Fatalf("اسمُ المتجر لم يصل الزبون: %v", v["merchant_name"])
	}
	for _, k := range []string{"merchant_id", "merchant_logo_thumb_url", "merchant_phone"} {
		if _, ok := v[k]; ok {
			t.Errorf("%s وصل الزبونَ", k)
		}
	}

	multi := &orders.Order{MerchantID: "m1", MerchantName: "مطعم الشام", Sections: []orders.CustomerSection{{Seq: 1}, {Seq: 2}}}
	if _, ok := orders.ViewFor(orders.AudienceCustomer, multi)["merchant_name"]; ok {
		t.Error("الطلبُ بمحطّتين لا يحمل اسمَ متجرٍ واحد")
	}
}
