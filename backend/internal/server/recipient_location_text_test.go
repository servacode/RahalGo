package server

import (
	"strings"
	"testing"
)

// TestLocReqTextNoLink **لا رابطَ في أيّ رسالةٍ تصل المستلِم** (قرارُ المالك ٢٠٢٦-١١-٠٦):
// رسالةُ طلب الموقع بكلّ أنواعه، ورسالةُ المتابعة. **والأمانةُ «لديك أمانة»، وغيرُها «لديك طلب».**
func TestLocReqTextNoLink(t *testing.T) {
	for _, mode := range []string{"send", "fetch", "amanah", locReqModeWhatsApp, ""} {
		txt := locReqText(mode)
		if strings.Contains(txt, "http") || strings.Contains(txt, "/loc/") {
			t.Errorf("%s: رابطٌ في الرسالة: %q", mode, txt)
		}
		want := "لديك طلب من تطبيق رحّال غو"
		if mode == "amanah" {
			want = "لديك أمانة من تطبيق رحّال غو"
		}
		lines := strings.Split(txt, "\n")
		if len(lines) < 2 || !strings.HasPrefix(lines[1], want) {
			t.Errorf("%s: السطرُ بعد السلام %q — والمنتظَر أن يبدأ بـ %q", mode, lines, want)
		}
		if !strings.Contains(txt, "الحالي أو المباشر") {
			t.Errorf("%s: لا ذكرَ للموقع المباشر: %q", mode, txt)
		}
	}
	if strings.Contains(waLocFollowUpText, "http") || strings.Contains(waLocFollowUpText, "/loc/") {
		t.Errorf("رسالةُ المتابعة فيها رابط: %q", waLocFollowUpText)
	}
}
