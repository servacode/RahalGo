package qa

// ══════════════════════════════════════════════════════════════════════
// **«سلفة» من الواتساب** (قرار المالك ٢٠٢٦-١١-١٩)
// ══════════════════════════════════════════════════════════════════════
//
// المالك بيكتب بمحادثته مع نفسه «سلفة <اسم> <مبلغ>» ← طلب سلفة معلّق ١٠ دقايق ← بيتطبّق لحالو بطريق
// الموافقة نفسه (المحفظة تنقص وسطر خارج من الدرج). و«الغاء» بهالوقت بيرفضه، والمحفظة ما بتتحرّك.

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func waAdvanceNamed(t *testing.T, h *Harness, role, name string) *User {
	t.Helper()
	u := h.NewUser(role)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET full_name = $2 WHERE id = $1::uuid`, u.ID, name); err != nil {
		t.Fatal(err)
	}
	return u
}

func waAdvanceReq(t *testing.T, h *Harness, userID string) (id, status string, auto bool) {
	t.Helper()
	_ = h.Pool.QueryRow(ctxBG(), `SELECT id::text, status, auto_apply_at IS NOT NULL FROM wallet_requests
		WHERE user_id = $1::uuid AND kind = 'advance' ORDER BY created_at DESC LIMIT 1`, userID).Scan(&id, &status, &auto)
	return
}

// TestWAADV01_RecordThenApplyAfterDelay **بينسجّل معلّق، وبعد وقته بينخصم وبيطلع من الدرج.**
func TestWAADV01_RecordThenApplyAfterDelay(t *testing.T) {
	h := New(t)
	h.NewUser("owner_super_admin")
	tag := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	drv := waAdvanceNamed(t, h, "driver", "زيد الكابتن "+tag)

	reply := h.API.ReportForSelf(ctxBG(), "سلفة زيد الكابتن "+tag+" ٥٠")
	if !strings.HasPrefix(reply, "✅") || !strings.Contains(reply, "الغاء") {
		t.Fatalf("الردّ: %q", reply)
	}
	id, status, auto := waAdvanceReq(t, h, drv.ID)
	if id == "" || status != "pending" || !auto {
		t.Fatalf("**الطلب**: %q %q %v", id, status, auto)
	}
	if b := walletBalance(t, h, drv.ID); b != 0 {
		t.Fatalf("**انخصمت قبل وقتها**: %d", b)
	}
	// **قبل العشر دقايق ما بيتطبّق.**
	h.API.SweepWAAdvanceOnce(ctxBG())
	if _, st, _ := waAdvanceReq(t, h, drv.ID); st != "pending" {
		t.Fatalf("**تطبّقت قبل وقتها**: %s", st)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE wallet_requests SET auto_apply_at = now() - interval '1 second' WHERE id = $1::uuid`, id); err != nil {
		t.Fatal(err)
	}
	h.API.SweepWAAdvanceOnce(ctxBG())
	h.API.SweepWAAdvanceOnce(ctxBG())
	if _, st, _ := waAdvanceReq(t, h, drv.ID); st != "approved" {
		t.Fatalf("**ما تطبّقت**: %s", st)
	}
	if b := walletBalance(t, h, drv.ID); b != -50 {
		t.Fatalf("**الرصيد %d والمنتظَر −٥٠**", b)
	}
	if amt, dir := advanceCash(t, h, id); amt != 50 || dir != "out" {
		t.Fatalf("**ما طلعت من الدرج**: %d %q", amt, dir)
	}
	// **وردّ البوت نفسه لمّا يرجع للمحادثة ما بيعمل شي.**
	if again := h.API.ReportForSelf(ctxBG(), reply); again != "" {
		t.Fatalf("**ردّ البوت انقرأ أمر**: %q", again)
	}
}

// TestWAADV02_CancelAndChoice **«الغاء» بيرفضها بلا ما تتحرّك المحفظة، والاسم المكرّر بيسأل «أيّ واحد».**
func TestWAADV02_CancelAndChoice(t *testing.T) {
	h := New(t)
	h.NewUser("owner_super_admin")
	tag := strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	drv := waAdvanceNamed(t, h, "driver", "عمر سامي "+tag)
	rep := waAdvanceNamed(t, h, "sales", "عمر سامي "+tag)
	cust := waAdvanceNamed(t, h, "customer", "خالد زبون "+tag)

	// **الاسم المكرّر ⇒ قائمة.**
	list := h.API.ReportForSelf(ctxBG(), "سلفه عمر سامي "+tag+" 30")
	if !strings.HasPrefix(list, "❓") || !strings.Contains(list, "كابتن") || !strings.Contains(list, "وكيل") {
		t.Fatalf("القائمة: %q", list)
	}
	pick := "1"
	if strings.Index(list, "وكيل") < strings.Index(list, "كابتن") {
		pick = "2"
	}
	if r := h.API.ReportForSelf(ctxBG(), pick); !strings.HasPrefix(r, "✅") || !strings.Contains(r, "كابتن") {
		t.Fatalf("الاختيار: %q", r)
	}
	if id, _, _ := waAdvanceReq(t, h, rep.ID); id != "" {
		t.Fatal("**انسجّلت للوكيل بدل الكابتن**")
	}

	// **«الغاء» ⇒ مرفوضة، والمحفظة ما تحرّكت، وما بتتطبّق بعدين.**
	if r := h.API.ReportForSelf(ctxBG(), "الغاء"); !strings.HasPrefix(r, "✅") {
		t.Fatalf("الإلغاء: %q", r)
	}
	id, st, _ := waAdvanceReq(t, h, drv.ID)
	if st != "rejected" {
		t.Fatalf("**بعد الإلغاء**: %s", st)
	}
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE wallet_requests SET auto_apply_at = now() - interval '1 second' WHERE id = $1::uuid`, id)
	h.API.SweepWAAdvanceOnce(ctxBG())
	if b := walletBalance(t, h, drv.ID); b != 0 {
		t.Fatalf("**الملغاة انخصمت**: %d", b)
	}
	if r := h.API.ReportForSelf(ctxBG(), "الغاء"); !strings.HasPrefix(r, "❓") {
		t.Fatalf("إلغاء بلا شي: %q", r)
	}

	// **والزبون ما بيطلع، والكلام العادي ما إلو ردّ.**
	if r := h.API.ReportForSelf(ctxBG(), "سلفة خالد زبون "+tag+" 20"); !strings.HasPrefix(r, "❓") {
		t.Fatalf("الزبون: %q", r)
	}
	if id, _, _ := waAdvanceReq(t, h, cust.ID); id != "" {
		t.Fatal("**انسجّلت سلفة لزبون**")
	}
	if r := h.API.ReportForSelf(ctxBG(), "مرحبا"); r != "" {
		t.Fatalf("كلام عادي: %q", r)
	}
	if r := h.API.ReportForSelf(ctxBG(), "سلفة عمر"); !strings.HasPrefix(r, "❓") {
		t.Fatalf("بلا مبلغ: %q", r)
	}
}
