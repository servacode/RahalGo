package server

// **قائمةُ الخطورة الواحدة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات، البند ٧).

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestSETTINGS_BadgeImpliesRiskList **كلُّ مفتاحٍ عليه شارةُ «يمسّ المال» في
// الفهرس في القائمة** — وإلّا رآه المالكُ محميّاً وغيّره من لا يملك الماليّة
// بلا كلمة سرّ (وقع في حصّة المنصّة من الأجرة).
func TestSETTINGS_BadgeImpliesRiskList(t *testing.T) {
	for _, d := range settings.Catalog {
		if d.Sensitive && settingRisk(d.Key) == "" {
			t.Errorf("%s عليه شارةٌ وليس في قائمة الخطورة — **شارةٌ بلا حماية**", d.Key)
		}
	}
}

// **والمستويان وقدرتاهما وخطوةُ التحقّق من المصدر نفسِه.**
func TestSETTINGS_RiskLevelsDriveCapabilityAndStepUp(t *testing.T) {
	cases := []struct {
		key  string
		risk string
		cap  authz.Capability
	}{
		{"delivery.platform_percent", riskMoney, authz.SettingsFinancialManage},
		{"delivery.custom_fee_min", riskMoney, authz.SettingsFinancialManage},
		{"finance.manual_wallet_max", riskMoney, authz.SettingsFinancialManage},
		{"security.session_days", riskSecurity, authz.SettingsSecurityManage},
		{"app.min_version.driver", riskSecurity, authz.SettingsFinancialManage},
		{"orders.delivery_estimate_min", "", authz.SettingsGeneralManage},
	}
	for _, c := range cases {
		if got := settingRisk(c.key); got != c.risk {
			t.Errorf("%s: خطورة %q والمنتظَرُ %q", c.key, got, c.risk)
		}
		if got := settingCapability(c.key); got != c.cap {
			t.Errorf("%s: قدرة %q والمنتظَرُ %q", c.key, got, c.cap)
		}
		if (c.risk != "") != criticalSettingKey(c.key) {
			t.Errorf("%s: خطوةُ التحقّق لا تطابق الخطورة", c.key)
		}
	}
}

// **والتساوي بالقيمة المطبَّعة** — `10` و`10.0` سواء فلا يُحفظ «١٠ ← ١٠».
func TestSETTINGS_SameValueNormalised(t *testing.T) {
	if !sameSettingValue(float64(10), int64(10)) {
		t.Fatal("10.0 و10 عُدّا مختلفين")
	}
	if sameSettingValue(float64(10), int64(12)) {
		t.Fatal("10 و12 عُدّا متساويين")
	}
	if !sameSettingValue(true, true) || sameSettingValue("a", "b") {
		t.Fatal("مقارنةُ غير الأرقام خاطئة")
	}
}
