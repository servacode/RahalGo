package server

import "testing"

// **حدُّ نسخةِ التطبيق إعدادٌ حسّاسٌ يلزمه خطوةُ تحقّق** — قرارُ المالك
// (البند F، ٢٠٢٦-٠٩-٢٧): رفعُ `app.min_version.driver` يقفل تطبيقَ كلِّ سائقٍ
// دون النسخة على شاشةِ تحديثٍ إلزاميّ، فهو فعلٌ تشغيليٌّ خطيرٌ كالمال.
// **ويُقفل هنا** فلا يعود «أفضلَ جهدٍ» صامتاً.
func TestCriticalSettingKey_MinVersionIsCritical(t *testing.T) {
	critical := []string{
		"app.min_version.driver",
		"app.min_version.customer",
		"app.min_version.merchant",
		"app.min_version.rep",
		"security.session_ttl_minutes", // نظيرٌ قائمٌ — يبقى حسّاساً
	}
	for _, k := range critical {
		if !criticalSettingKey(k) {
			t.Errorf("criticalSettingKey(%q)=false — والمتوقّع حسّاسٌ (خطوةُ تحقّق+تدقيقٌ في المعاملة)", k)
		}
	}
}
