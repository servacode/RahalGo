package qa

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعارُ قرارِ السحب يقول أثرَه في الرصيد صراحةً**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٦: «لازم يكون واضحاً أنّه خُصم من الرصيد ليعرف الشخصُ
// صحّ».) **كان الجسدُ ملاحظةَ الماليّة فقط** — فيرى العاملُ رصيدَه نقص ولا يعرف
// لماذا. **فصار يقول المبلغَ واتّجاهَه**: `paid` خُصم من الرصيد، و`rejected`
// أُعيد إليه.
func TestPAYOUT_NotifBodyStatesBalanceEffect(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	admin := h.NewUser("admin")

	// ── مدفوع ⇒ «خُصم … من رصيدك» + المبلغُ بفواصله ──
	drv := f.Driver()
	f.Credit(drv.ID, 100_000, "topup")
	made := h.POST("/api/v1/me/payouts", drv.Token, map[string]any{"amount": 100_000})
	if made.Code >= 400 {
		t.Fatalf("طلبُ السحب رُدّ: %s", made)
	}
	pid, _ := made.JSON()["id"].(string)
	dec := h.POST("/api/v1/admin/payouts/"+pid+"/decide", admin.Token,
		map[string]any{"status": "paid", "decision": "تمّت المعالجة"})
	if dec.Code >= 400 {
		t.Fatalf("قرارُ الدفع رُدّ: %s", dec)
	}
	body := lastWalletNotifBody(t, h, drv.ID)
	t.Logf("paid body = %q", body)
	if !strings.Contains(body, "خُصم") || !strings.Contains(body, "من رصيدك") {
		t.Errorf("إشعارُ الدفع لا يقول «خُصم … من رصيدك»: %q", body)
	}
	if !strings.Contains(body, "100,000") {
		t.Errorf("إشعارُ الدفع لا يذكر المبلغَ بفواصله (100,000): %q", body)
	}

	// ── مرفوض ⇒ «أُعيد … إلى رصيدك» ──
	drv2 := f.Driver()
	f.Credit(drv2.ID, 50_000, "topup")
	made2 := h.POST("/api/v1/me/payouts", drv2.Token, map[string]any{"amount": 50_000})
	if made2.Code >= 400 {
		t.Fatalf("طلبُ السحب الثاني رُدّ: %s", made2)
	}
	pid2, _ := made2.JSON()["id"].(string)
	dec2 := h.POST("/api/v1/admin/payouts/"+pid2+"/decide", admin.Token,
		map[string]any{"status": "rejected", "decision": "سببُ الرفض"})
	if dec2.Code >= 400 {
		t.Fatalf("قرارُ الرفض رُدّ: %s", dec2)
	}
	body2 := lastWalletNotifBody(t, h, drv2.ID)
	t.Logf("rejected body = %q", body2)
	if !strings.Contains(body2, "أُعيد") || !strings.Contains(body2, "إلى رصيدك") {
		t.Errorf("إشعارُ الرفض لا يقول «أُعيد … إلى رصيدك»: %q", body2)
	}
	if !strings.Contains(body2, "50,000") {
		t.Errorf("إشعارُ الرفض لا يذكر المبلغَ بفواصله (50,000): %q", body2)
	}
}

// lastWalletNotifBody **آخرُ إشعارِ محفظةٍ لهذا المستخدم.**
func lastWalletNotifBody(t *testing.T, h *Harness, uid string) string {
	t.Helper()
	var body string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT body FROM notifications WHERE user_id = $1::uuid AND kind = 'wallet'
		 ORDER BY created_at DESC LIMIT 1`, uid).Scan(&body); err != nil {
		t.Fatalf("قراءةُ الإشعار: %v", err)
	}
	return body
}
