package qa

import (
	"strings"
	"testing"
	"time"
)

// TestSTOREOFFERS_FilterAndCount **زرُّ «🔥 العروض» بصفحة المتجر** (قرارُ المالك ٢٠٢٦-١١-١٢) — الملخّصُ
// يقول كم صنفاً عليه حسمٌ ساري، **و`?offers=1` يعرض تلك وحدَها** عبر الأقسام.
func TestSTOREOFFERS_FilterAndCount(t *testing.T) {
	h := New(t)
	m := h.Factory().Merchant()
	plain := h.NewItemFor(m, 1000)
	sale := h.NewItemFor(m, 2000)
	admin := h.NewUser("admin")
	made := h.POST("/api/v1/admin/offers", admin.Token, map[string]any{
		"kind": "discount", "title": "حسم", "menu_item_id": sale.ID, "discount_percent": 15,
		"starts_at": time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
	})
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ العرض: %s", made)
	}
	sum := h.GET("/api/v1/public/stores/"+m.ID+"/summary", "")
	if !strings.Contains(string(sum.Body), `"offers":1`) {
		t.Fatalf("**عددُ العروض بالملخّص**: %s", sum.Body)
	}
	got := h.GET("/api/v1/public/stores/"+m.ID+"/items?offers=1", "")
	if !strings.Contains(string(got.Body), sale.ID) || strings.Contains(string(got.Body), plain.ID) {
		t.Fatalf("**الفلترُ لا يعرض العروضَ وحدَها**: %s", got.Body)
	}
	all := h.GET("/api/v1/public/stores/"+m.ID+"/items", "")
	if !strings.Contains(string(all.Body), plain.ID) || !strings.Contains(string(all.Body), sale.ID) {
		t.Fatalf("**بلا فلتر لازم يطلع الكل**: %s", all.Body)
	}
}
