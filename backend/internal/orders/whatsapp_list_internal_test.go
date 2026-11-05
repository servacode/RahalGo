package orders

// **طلبُ الواتساب في القوائم** (قرارُ المالك ٢٠٢٦-١١-٠٥) — فلترُه بأصله، و«توصيلات المتاجر» بلاه،
// وبحثُ الموظّفين باسم متجره ورقمه المكتوبين. **بلا قاعدة**: الشرطُ نصٌّ يُقرأ.

import (
	"strings"
	"testing"
)

func TestWhatsAppListKindFilter(t *testing.T) {
	where, _, err := ListWhere(ListFilter{Kind: "whatsapp"}, StuckLimits{})
	if err != nil {
		t.Fatalf("**فلترُ «طلبات الواتساب» مرفوض**: %v", err)
	}
	if !strings.Contains(where, `o.origin = 'whatsapp'`) {
		t.Fatalf("**الفلترُ لا يقرأ الأصل**: %s", where)
	}
	md, _, err := ListWhere(ListFilter{Kind: KindMerchantDelivery}, StuckLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, `o.origin = 'app'`) {
		t.Fatalf("**«توصيلات المتاجر» تشمل طلبَ الواتساب**: %s", md)
	}
}

func TestWhatsAppStaffSearchReadsTypedStore(t *testing.T) {
	if !strings.Contains(staffSearchName("$1"), "o.pickup_name") {
		t.Fatal("**بحثُ الموظّفين لا يقرأ اسمَ المتجر المكتوب**")
	}
	if !strings.Contains(staffSearchPhone("$1"), "o.pickup_phone") {
		t.Fatal("**بحثُ الموظّفين لا يقرأ رقمَ المتجر المكتوب**")
	}
}
