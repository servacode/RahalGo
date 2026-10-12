package server

import (
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/settings"
)

// TestWelcome_ThirdMessageCarriesNoPhoneOrPassword — **الرسالةُ الثالثة بلا الرقم
// والكلمة** (طلبُ المالك ٢٠٢٦-١٠-١٢: كلٌّ منهما برسالةٍ وحدَه قبلها)، **ولو كان
// القالبُ قديماً يذكرهما**؛ **وسطرُ المهلة يُحذف لمن لا يُجبَر.**
func TestWelcome_ThirdMessageCarriesNoPhoneOrPassword(t *testing.T) {
	s := &Server{}
	old := "{title}\nالرقم: {phone}\nكلمة المرور المؤقتة: {password}\nتنتهي بعد {hours} ساعة.\n📱 {app}: {link}"
	for _, tpl := range []string{settings.WelcomeTemplateDefault, old} {
		staff := welcomeText(tpl, "+963911111111", "Pw9secret", "https://x/a", appPurpose(""), s.shopLinkFor(""), 72, false, false)
		field := welcomeText(tpl, "+963911111111", "Pw9secret", "https://x/m", appPurpose("merchant"), s.shopLinkFor("merchant"), 72, false, true)
		for _, msg := range []string{staff, field} {
			if strings.Contains(msg, "Pw9secret") || strings.Contains(msg, "+963911111111") {
				t.Fatalf("**الرقمُ أو الكلمةُ في الرسالة الثالثة**:\n%s", msg)
			}
			if !strings.Contains(msg, "https://x/") {
				t.Fatalf("**الرابطُ غاب عن الرسالة الثالثة**:\n%s", msg)
			}
		}
		if !strings.Contains(staff, "72") {
			t.Fatalf("**الموظّفُ بلا سطر المهلة**:\n%s", staff)
		}
		if strings.Contains(field, "72") || strings.Contains(field, "تغييرها") {
			t.Fatalf("**المتجرُ وصله سطرُ المهلة أو التبديل**:\n%s", field)
		}
	}
}
