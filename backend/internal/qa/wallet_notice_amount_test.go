package qa

import (
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعارُ حركة المحفظة يقول قدّيش وليش** (بلاغُ المالك ٢٠٢٦-١١-٢٣: «خصمت من المحفظة 5000
// يقلي خصم 0 من محفظتك لازم يعطني ليش انخصم وشقد»)
// ══════════════════════════════════════════════════════════════════════

func lastWalletNotice(t *testing.T, h *Harness, uid string) (title, body string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT title, body FROM notifications WHERE user_id = $1::uuid
		 ORDER BY created_at DESC LIMIT 1`, uid).Scan(&title, &body); err != nil {
		t.Fatalf("لا إشعار: %v", err)
	}
	return
}

func TestWalletNoticeSaysAmountAndReason(t *testing.T) {
	h := New(t)
	treasury(t, h)
	admin := h.NewUser("admin")
	cust := h.Customer()

	if r := h.adminWallet(admin, cust.ID, map[string]any{
		"amount": 7000, "kind": "adjustment", "note": "تسوية طلب قديم"}); r.Code != 200 {
		t.Fatalf("الإيداع: %s", r)
	}
	_, body := lastWalletNotice(t, h, cust.ID)
	if !strings.Contains(body, "7,000 ل.س") || !strings.Contains(body, "السبب: تسوية طلب قديم") {
		t.Errorf("**إشعارُ الإيداع بلا مبلغٍ أو سبب**: %q", body)
	}

	if r := h.adminWallet(admin, cust.ID, map[string]any{
		"amount": 5000, "kind": "adjustment", "debit": true, "note": "أجور توصيل طلب 1234"}); r.Code != 200 {
		t.Fatalf("الخصم: %s", r)
	}
	title, body := lastWalletNotice(t, h, cust.ID)
	if !strings.Contains(body, "انخصم 5,000 ل.س من محفظتك") || !strings.Contains(body, "السبب: أجور توصيل طلب 1234") {
		t.Errorf("**إشعارُ الخصم بلا مبلغٍ أو سبب**: %q / %q", title, body)
	}
}
