package orders

import (
	"strings"
	"testing"
)

// TestListWhere_CustomModeKinds **«مشتريات» و«أمانة» فلتران منفصلان** (قرارُ المالك ٢٠٢٦-١٠-٢٧)
// — و«custom» القديمُ يبقى يجمعهما للروابط القديمة. بلا قاعدة.
func TestListWhere_CustomModeKinds(t *testing.T) {
	for kind, mode := range map[string]string{"custom_purchase": "purchase", "custom_amanah": "amanah"} {
		where, args, err := ListWhere(ListFilter{Kind: kind}, StuckLimits{})
		if err != nil {
			t.Fatalf("%s ردّ: %v", kind, err)
		}
		if !strings.Contains(where, "o.kind = 'custom' AND o.custom_mode = $1") {
			t.Fatalf("%s بلا شرط custom_mode: %s", kind, where)
		}
		if len(args) != 1 || args[0] != mode {
			t.Fatalf("%s معاملُه %v لا %s", kind, args, mode)
		}
	}
	where, args, err := ListWhere(ListFilter{Kind: "custom"}, StuckLimits{})
	if err != nil || !strings.Contains(where, "o.kind = $1") || strings.Contains(where, "custom_mode") ||
		len(args) != 1 || args[0] != "custom" {
		t.Fatalf("«custom» القديمُ تغيّر: %v · %s · %v", err, where, args)
	}
	if _, _, err := ListWhere(ListFilter{Kind: "custom_other"}, StuckLimits{}); err == nil {
		t.Fatal("نوعٌ لا يُعرف قُبل")
	}
}
