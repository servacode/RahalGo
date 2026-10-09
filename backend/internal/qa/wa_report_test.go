package qa

import (
	"context"
	"strings"
	"testing"
)

// **«وضع المنصة» على الواتساب** (طلبُ المالك ٢٠٢٦-١٠-٠٩): المالكُ من محادثته مع نفسه،
// ورقمُ القائمة من رسالة — **وغيرُهما لا يصله شيء.**
func TestWAR01_PlatformReportOverWhatsApp(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)
	staff := h.NewUser("operations")
	placedOrder(t, h, staff.Token)
	ctx := context.Background()

	r := h.API.ReportForSelf(ctx, "وضع المنصة")
	for _, want := range []string{"وضع رحّال غو", "الطلبات اليوم", "السائقين", "المال اليوم", "الصحة", "القاعدة ✅"} {
		if !strings.Contains(r, want) {
			t.Fatalf("التقريرُ بلا %q:\n%s", want, r)
		}
	}
	if !strings.Contains(r, "١") {
		t.Fatalf("طلبٌ أُنشئ ولا يظهر عددُه:\n%s", r)
	}
	if got := h.API.ReportForSelf(ctx, "ملاحظة لنفسي: اشتري خبز"); got != "" {
		t.Fatalf("**رُدّ على ملاحظةٍ ليست أمراً**: %q", got)
	}
	if got := h.API.ReportForSelf(ctx, "مال"); !strings.Contains(got, "المال اليوم") || strings.Contains(got, "السائقين:") {
		t.Fatalf("«مال» وحدَه: %q", got)
	}

	// **رقمٌ غريبٌ لا يصله شيء — ورقمُ القائمة يصله.**
	if got := h.API.ReportForPhone(ctx, "+963933000111", "تقرير"); got != "" {
		t.Fatalf("**رقمٌ ليس في القائمة أخذ التقرير**")
	}
	h.Setting("ops.report_phones", `"0933000111, +963944000222"`)
	if got := h.API.ReportForPhone(ctx, "+963933000111", "تقرير"); !strings.Contains(got, "وضع رحّال غو") {
		t.Fatalf("رقمُ القائمة لم يصله التقرير: %q", got)
	}
	if got := h.API.ReportForPhone(ctx, "+963933000111", "مرحبا"); got != "" {
		t.Fatalf("كلامٌ عاديٌّ من رقم القائمة رُدّ عليه بتقرير")
	}
}
