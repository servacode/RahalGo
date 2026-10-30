package orders

import (
	"strings"
	"testing"
)

// TestListWhere_HistoryFileFilters **فلاترُ السجلّ الثلاثة والبحثُ الأعرض** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// — طريقةُ الدفع والمتأخّر والعمولةُ الممحيّة، واسمُ السائق والمتجر وهاتفاهما وكودُ الخصم. بلا قاعدة.
func TestListWhere_HistoryFileFilters(t *testing.T) {
	where, _, err := ListWhere(ListFilter{Pay: "cash"}, StuckLimits{})
	if err != nil || !strings.Contains(where, "o.cash_due > 0") {
		t.Fatalf("pay=cash: %v · %s", err, where)
	}
	where, _, err = ListWhere(ListFilter{Pay: "wallet"}, StuckLimits{})
	if err != nil || !strings.Contains(where, "o.wallet_paid > 0") {
		t.Fatalf("pay=wallet: %v · %s", err, where)
	}
	if _, _, err := ListWhere(ListFilter{Pay: "card"}, StuckLimits{}); err == nil {
		t.Fatal("طريقةُ دفعٍ لا تُعرف قُبلت")
	}
	where, _, err = ListWhere(ListFilter{Late: true}, StuckLimits{DeliveryMin: 45})
	if err != nil || !strings.Contains(where, "o.delivered_at > o.created_at + make_interval(mins => 45)") {
		t.Fatalf("late: %v · %s", err, where)
	}
	where, _, err = ListWhere(ListFilter{Waived: true}, StuckLimits{})
	if err != nil || !strings.Contains(where, "o.commission_waived > 0") {
		t.Fatalf("waived: %v · %s", err, where)
	}

	// **والبحثُ الأعرض للوحة وحدَها** — بوّابةُ المتجر والزبون على حالها.
	where, _, _ = ListWhere(ListFilter{Query: "مطعم"}, StuckLimits{})
	if strings.Contains(where, "sd.full_name") || strings.Contains(where, "promo_code") {
		t.Fatalf("البحثُ الأعرض بلا طلب: %s", where)
	}
	where, _, _ = ListWhere(ListFilter{Query: "مطعم", StaffSearch: true}, StuckLimits{})
	for _, want := range []string{"sd.full_name ILIKE", "sm.name ILIKE", "o.promo_code"} {
		if !strings.Contains(where, want) {
			t.Fatalf("بحثُ الاسم بلا %q: %s", want, where)
		}
	}
	where, _, _ = ListWhere(ListFilter{Query: "+963933000111", StaffSearch: true}, StuckLimits{})
	for _, want := range []string{"sd.phone::text LIKE", "sm.phone::text LIKE"} {
		if !strings.Contains(where, want) {
			t.Fatalf("بحثُ الهاتف بلا %q: %s", want, where)
		}
	}
}
