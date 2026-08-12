package orders

import (
	"encoding/json"
	"os"
	"testing"
)

// TestEveryOpsStageHasAnArabicName **ولا مفتاحَ آلةٍ يظهر في شاشة.**
//
// **المراحلُ مفاتيحُ إنجليزيّةٌ يرسلها المحرّك** (`seeking_driver`)
// **وأسماؤها في المعجم.** ومن أضاف مرحلةً هنا ونسي اسمَها هناك **لا
// يسقط بناؤه ولا يصرخ شيء** — تظهر البطاقةُ وفيها `waiting_platform`
// بحروفٍ لاتينيّةٍ بين أسماءٍ عربيّة.
//
// **ووقع فعلاً في عائلته**: إعدادٌ أُضيف بلا اسمٍ فأمسكه حارسُ الإعدادات
// (٢٠٢٦-٠٨-١٢) — **وهذا حارسُه في المراحل.**
//
// **والمساران معاً**: العاديُّ والخاصّ — **ومن أضاف مسارَ نوعٍ ثالثٍ
// يضيفه هنا** فيُحرَس من أوّل يوم.
func TestEveryOpsStageHasAnArabicName(t *testing.T) {
	raw, err := os.ReadFile("../../../web/packages/i18n/src/locales/ar.json")
	if err != nil {
		t.Skipf("المعجمُ غيرُ متاحٍ هنا: %v", err)
	}
	var dict struct {
		Admin struct {
			OrdersPage struct {
				Stage map[string]string `json:"stage"`
			} `json:"ordersPage"`
		} `json:"admin"`
	}
	if err := json.Unmarshal(raw, &dict); err != nil {
		t.Fatalf("المعجمُ لا يُقرأ: %v", err)
	}
	names := dict.Admin.OrdersPage.Stage
	if len(names) == 0 {
		t.Fatal("لا أسماءَ مراحلَ في المعجم أصلا")
	}

	seen := map[Stage]bool{}
	for _, list := range [][]Stage{OpsStages(), OpsCustomStages()} {
		for _, st := range list {
			if seen[st] {
				continue
			}
			seen[st] = true
			if names[string(st)] == "" {
				t.Errorf("المرحلةُ %q بلا اسمٍ في admin.ordersPage.stage — "+
					"تظهر بمفتاحها اللاتينيّ في بطاقةٍ عربيّة", st)
			}
		}
	}
}
