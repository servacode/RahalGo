package server

import (
	"strings"
	"testing"
)

// TestLocReqTextNoLink **لا رابطَ في أيّ رسالةٍ تصل المستلِم** (قرارُ المالك ٢٠٢٦-١١-٠٦):
// رسالةُ طلب الموقع بكلّ أنواعه، ورسالةُ المتابعة. **والأمانةُ «لديك أمانة»، وغيرُها «لديك طلب».**
func TestLocReqTextNoLink(t *testing.T) {
	for _, mode := range []string{"send", "fetch", "amanah", locReqModeWhatsApp, ""} {
		txt := locReqText(mode, false)
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
	for _, live := range []bool{false, true} {
		if f := waLocFollowUpText(live); strings.Contains(f, "http") || strings.Contains(f, "/loc/") {
			t.Errorf("رسالةُ المتابعة فيها رابط: %q", f)
		}
	}
	// **والمباشرُ وحدَه** (٢٠٢٦-١١-١١): لا ذكرَ للحاليّ، ويُطلب المباشر.
	for _, txt := range []string{locReqText("send", true), waLocFollowUpText(true), waLiveOnlyReply} {
		if strings.Contains(txt, "الحالي") || !strings.Contains(txt, "الموقع المباشر") {
			t.Errorf("المباشرُ وحدَه: %q", txt)
		}
	}
}
