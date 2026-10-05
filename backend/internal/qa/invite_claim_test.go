package qa

// ══════════════════════════════════════════════════════════════════════
//  INV-CLAIM — **الدعوةُ بلا نسخِ رمز** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// الصديقُ يكتب رقمَه في صفحة الدعوة ⇒ `POST /public/invite/claim` ⇒ ثمّ
// يسجّل من التطبيق **بلا `ref`** ⇒ يُنسب لمن دعاه ويُحذف الحجز.
//
// **وبالشبكة لا بالدالّة** — كما تناديه الصفحةُ والتطبيق.

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

// icInviter **زبونٌ له رمزُ دعوةٍ معروف.**
func icInviter(t *testing.T, h *Harness) (id, code string) {
	t.Helper()
	u := h.NewUser("customer")
	code = fmt.Sprintf("QIC%d", time.Now().UnixNano()%1e12+seq.Add(1))
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET invite_code = $2 WHERE id = $1::uuid`, u.ID, code); err != nil {
		t.Fatalf("رمزُ الداعي: %v", err)
	}
	return u.ID, code
}

// icClaim **حجزٌ كما تُرسله الصفحة** — بعنوانٍ خاصٍّ بالفحص.
func icClaim(t *testing.T, h *Harness, code, phone string) Res {
	t.Helper()
	ip := suIP(t, h)
	t.Cleanup(func() { h.Redis().Del(ctxBG(), "invite:claim:ip:"+ip) })
	return h.Call("POST", "/api/v1/public/invite/claim", "",
		map[string]any{"code": code, "phone": phone}, map[string]string{"X-Real-IP": ip})
}

// icLocal **الرقمُ بصيغة الناس** — `09…` — ليُفحص التطبيعُ معه.
func icLocal(e164 string) string { return "0" + strings.TrimPrefix(e164, "+963") }

func icCleanup(t *testing.T, h *Harness, phone string) {
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM invite_claims WHERE phone = $1`, phone)
	})
}

// ── ١ · حجزٌ ثمّ تسجيلٌ بلا `ref` ⇒ النسبُ وقع والحجزُ حُذف ──────────
func TestINVCLAIM01_ClaimThenSignupWithoutRefAttaches(t *testing.T) {
	h := New(t)
	suPolicy(h, true)
	inviterID, code := icInviter(t, h)
	phone := uniqPhone()
	suDropPhone(t, h, phone)
	icCleanup(t, h, phone)

	if r := icClaim(t, h, strings.ToLower(code), icLocal(phone)); r.Code != http.StatusOK {
		t.Fatalf("**رُدّ حجزٌ صحيح**: %s", r)
	}
	var stored string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT invite_code FROM invite_claims WHERE phone = $1`, phone).Scan(&stored); err != nil || stored != code {
		t.Fatalf("**الحجزُ لم يُحفظ بالرقم المطبَّع**: %q %v", stored, err)
	}

	res := suConfirm(h, suIP(t, h), phone, suPlant(t, h, phone))
	if res.Code >= 400 || suToken(res) == "" {
		t.Fatalf("التسجيلُ رُدّ: %d %s", res.Code, trimBody(res))
	}
	var got string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT r.inviter_id::text FROM referrals r JOIN users u ON u.id = r.invitee_id
		 WHERE u.phone = $1`, phone).Scan(&got); err != nil || got != inviterID {
		t.Fatalf("**لم يُنسب الحسابُ لمن دعاه**: %q %v", got, err)
	}
	var left int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM invite_claims WHERE phone = $1`, phone).Scan(&left)
	if left != 0 {
		t.Errorf("**الحجزُ بقي بعد أخذه**")
	}
}

// ── ٢ · رمزٌ مجهولٌ يُردّ، وحجزٌ منتهٍ لا يُنسب ─────────────────────────
func TestINVCLAIM02_InvalidCodeRejectedAndExpiredClaimIgnored(t *testing.T) {
	h := New(t)
	suPolicy(h, true)
	phone := uniqPhone()
	suDropPhone(t, h, phone)
	icCleanup(t, h, phone)

	if r := icClaim(t, h, "ZZZZZZZZZ", icLocal(phone)); r.Code != http.StatusNotFound || r.Err() != "bad_invite_code" {
		t.Errorf("**رمزٌ مجهولٌ قُبل**: %s", r)
	}
	if r := icClaim(t, h, "", icLocal(phone)); r.Err() != "bad_invite_code" {
		t.Errorf("**رمزٌ فارغٌ قُبل**: %s", r)
	}
	_, code := icInviter(t, h)
	if r := icClaim(t, h, code, "12345"); r.Err() != "invalid_phone" {
		t.Errorf("**رقمٌ فاسدٌ قُبل**: %s", r)
	}

	// **والمنتهي**: حجزٌ عمرُه واحدٌ وثلاثون يوماً لا يُنسب.
	if r := icClaim(t, h, code, icLocal(phone)); r.Code != http.StatusOK {
		t.Fatalf("رُدّ حجزٌ صحيح: %s", r)
	}
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE invite_claims SET created_at = now() - interval '31 days' WHERE phone = $1`, phone); err != nil {
		t.Fatal(err)
	}
	res := suConfirm(h, suIP(t, h), phone, suPlant(t, h, phone))
	if res.Code >= 400 {
		t.Fatalf("التسجيلُ رُدّ: %d %s", res.Code, trimBody(res))
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM referrals r JOIN users u ON u.id = r.invitee_id
		 WHERE u.phone = $1`, phone).Scan(&n)
	if n != 0 {
		t.Errorf("**نُسب الحسابُ بحجزٍ منتهٍ**")
	}
}

// ── ٣ · رقمٌ له حسابٌ يُردّ برمزٍ يُفهم ─────────────────────────────────
func TestINVCLAIM03_ExistingUserPhoneRejected(t *testing.T) {
	h := New(t)
	_, code := icInviter(t, h)
	u := h.NewUser("customer")
	var phone string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT phone FROM users WHERE id = $1::uuid`, u.ID).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	icCleanup(t, h, phone)
	r := icClaim(t, h, code, icLocal(phone))
	if r.Code != http.StatusConflict || r.Err() != "phone_taken" {
		t.Errorf("**حُجز رقمٌ له حساب**: %s", r)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM invite_claims WHERE phone = $1`, phone).Scan(&n)
	if n != 0 {
		t.Errorf("**كُتب حجزٌ لرقمٍ مسجَّل**")
	}
}

// ── ٤ · الأحدثُ يغلب ────────────────────────────────────────────────────
func TestINVCLAIM04_LatestClaimWins(t *testing.T) {
	h := New(t)
	_, first := icInviter(t, h)
	_, second := icInviter(t, h)
	phone := uniqPhone()
	icCleanup(t, h, phone)
	if r := icClaim(t, h, first, icLocal(phone)); r.Code != http.StatusOK {
		t.Fatalf("%s", r)
	}
	if r := icClaim(t, h, second, phone); r.Code != http.StatusOK {
		t.Fatalf("%s", r)
	}
	var got string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT invite_code FROM invite_claims WHERE phone = $1`, phone).Scan(&got)
	if got != second {
		t.Errorf("**الحجزُ الأحدثُ لم يغلب**: %q والمنتظَرُ %q", got, second)
	}
}
