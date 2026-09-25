package qa

// ══════════════════════════════════════════════════════════════════════
//  CUST-DEF-001 — **التسجيلُ لا يصير استعادةً ولا استيلاءً**
// ══════════════════════════════════════════════════════════════════════
//
// (بوّابةُ العوائق قبل قبول الزبون · `CUSTOMER-ACCEPTANCE-MASTER.md` §40.3.)
//
// # العيب
//
// **`ConfirmSignup` كان يكتب كلمةَ حسابٍ قائمٍ ويُصدر له جلسة** حين يُطفأ
// `auth.signup_verify` — **فمن عرف رقمَ هاتفٍ ملك حسابَه**، أيّاً كان دورُه.
// **والأدمنُ بلا PIN يُسلَّم تحدّيَ إنشاءِ رمزه**، **والموقوفُ تُكتب كلمتُه
// قبل أن يُرفض**، **وعلَمُ «بدّل كلمتك» يُمحى.** **والمفتاحُ يُقرأ من القاعدة
// فإن فشلت القراءةُ عُدّ مطفأً.** **ولا بوّابةَ إطلاقٍ ولا حدَّ محاولاتٍ على
// باب التأكيد.**
//
// # والعقدُ (قرارُ المالك ٢٠٢٦-٠٩-١٩)
//
// **معرفةُ رقمِ حسابٍ قائمٍ لا تُعطي أبداً**: كتابةَ كلمته · تبديلَ هويّته ·
// محوَ علَمِ التبديل · جلسةً · رمزَ أدمن · مسَّ حسابٍ موقوفٍ أو محظور —
// **ولو كان التحقّقُ مطفأً.** **وفشلُ قراءةِ الإعداد يُغلِق لا يفتح.**
//
// # وكيف تُفحص
//
// **بالشبكة لا بالدالّة** — `POST /api/v1/auth/signup/confirm` كما يناديه
// التطبيق. **وجسرُ الإعدادات يُحقَن كما يحقنه `main.go`** (`GetNum`)،
// **فالقيمةُ من القاعدة لا من الاحتياطيّ.** **ولكلّ فحصٍ عنوانُه** (`X-Real-IP`)
// كي لا تقفل عدّاداتُه ما بعده، **ويُمحى ما كتبه في الذاكرة.**
//
// **ولا حسابَ حقيقيّ** — حساباتٌ تُنشأ وتُحذف في الفحص نفسِه.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

var suSeq atomic.Int64

// suIP **عنوانٌ لكلّ فحص** — عدّاداتُ القفل للعنوان، فلا يرث فحصٌ قفلَ آخر.
func suIP(t *testing.T, h *Harness) string {
	t.Helper()
	n := suSeq.Add(1)
	ip := fmt.Sprintf("10.201.%d.%d", (n/250)%250, n%250+1)
	t.Cleanup(func() { h.Redis().Del(ctxBG(), "login:fail:i:"+ip, "signup:fail:i:"+ip) })
	return ip
}

// suWire **جسرُ الإعدادات كما في `main.go`** — `settingsStore.GetNum`.
func suWire(h *Harness) {
	h.Identity.SetSettingReader(h.Settings.GetNum)
}

// suPolicy **بابُ التسجيل مفتوحٌ والتحقّقُ كما يُطلب.**
func suPolicy(h *Harness, verify bool) {
	suWire(h)
	h.Setting("launch.customer_signup", "true")
	h.Setting("auth.signup_verify", fmt.Sprintf("%t", verify))
}

type suAcct struct {
	ID, Phone string
}

// suAccount **حسابٌ قائمٌ بأدواره وحاله** — وبكلمةٍ أو بدونها.
func suAccount(t *testing.T, h *Harness, roles []string, status string, password bool, mustChange bool) suAcct {
	t.Helper()
	u := h.NewUser(roles[0])
	for _, r := range roles[1:] {
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, $2)
			ON CONFLICT DO NOTHING`, u.ID, r); err != nil {
			t.Fatalf("دورٌ إضافيّ: %v", err)
		}
	}
	if password {
		hash, err := auth.HashPassword("Original-Pass-777")
		if err != nil {
			t.Fatalf("بصمةُ الكلمة: %v", err)
		}
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE users SET password_hash = $2, must_change_password = $3 WHERE id = $1::uuid`,
			u.ID, hash, mustChange); err != nil {
			t.Fatalf("كلمةُ الحساب: %v", err)
		}
	}
	if status != "active" {
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE users SET status = $2 WHERE id = $1::uuid`, u.ID, status); err != nil {
			t.Fatalf("حالُ الحساب: %v", err)
		}
	}
	t.Cleanup(func() { h.Redis().Del(ctxBG(), "login:fail:p:"+u.Phone, "signup:fail:p:"+u.Phone) })
	return suAcct{ID: u.ID, Phone: u.Phone}
}

// suState **ما لا يجوز أن يتبدّل** — كلمةٌ · اسمٌ · علَمٌ · حالٌ · رمزُ أدمن · جلسات.
type suState struct {
	Hash, Name, Status, Pin string
	Must                    bool
	Sessions                int
}

func suRead(t *testing.T, h *Harness, id string) suState {
	t.Helper()
	var s suState
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(password_hash, ''), COALESCE(full_name, ''), status,
		       COALESCE(admin_pin_hash, ''), must_change_password,
		       (SELECT count(*) FROM refresh_tokens WHERE user_id = u.id)
		FROM users u WHERE id = $1::uuid`, id).
		Scan(&s.Hash, &s.Name, &s.Status, &s.Pin, &s.Must, &s.Sessions); err != nil {
		t.Fatalf("قراءةُ الحساب: %v", err)
	}
	return s
}

// suPlant **رمزُ تسجيلٍ مزروعٌ لرقم** — بالبصمة التي يقرؤها المحرّك (`plantOTP`).
func suPlant(t *testing.T, h *Harness, phone string) string {
	t.Helper()
	code := "515151"
	mac := hmac.New(sha256.New, []byte(jwtSecret))
	mac.Write([]byte(phone + ":" + code))
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO otp_codes (phone, code_hash, purpose, expires_at)
		VALUES ($1, $2, 'signup', now() + interval '10 minutes')`,
		phone, hex.EncodeToString(mac.Sum(nil))); err != nil {
		t.Fatalf("زرعُ الرمز: %v", err)
	}
	return code
}

// suConfirm **بابُ التأكيد كما يناديه التطبيق.**
func suConfirm(h *Harness, ip, phone, code string) Res {
	return h.Call("POST", "/api/v1/auth/signup/confirm", "", map[string]any{
		"phone": phone, "code": code, "full_name": "Attacker Name",
		"password": "Attacker-Pass-999",
	}, map[string]string{"X-Real-IP": ip})
}

// suNoTakeover **لا جلسةَ ولا تحدّيَ ولا تبدّل.**
func suNoTakeover(t *testing.T, h *Harness, a suAcct, before suState, res Res) {
	t.Helper()
	if res.Code < 400 {
		t.Errorf("**قُبل تأكيدُ التسجيل على حسابٍ قائم** — %d %s", res.Code, trimBody(res))
	}
	if suToken(res) != "" {
		t.Errorf("**صدرت جلسةٌ لحسابٍ قائمٍ عبر التسجيل**")
	}
	if strings.Contains(string(res.Body), "challenge") {
		t.Errorf("**صدر تحدّي PIN عبر التسجيل**")
	}
	after := suRead(t, h, a.ID)
	if after.Hash != before.Hash {
		t.Errorf("**تبدّلت كلمةُ الحساب عبر التسجيل**")
	}
	if after.Name != before.Name {
		t.Errorf("**تبدّل اسمُ الحساب عبر التسجيل**: %q ⇐ %q", before.Name, after.Name)
	}
	if after.Must != before.Must {
		t.Errorf("**تبدّل علَمُ «بدّل كلمتك» عبر التسجيل**: %v ⇐ %v", before.Must, after.Must)
	}
	if after.Status != before.Status {
		t.Errorf("**تبدّل حالُ الحساب**: %s ⇐ %s", before.Status, after.Status)
	}
	if after.Pin != before.Pin {
		t.Errorf("**تبدّل رمزُ الأدمن عبر التسجيل**")
	}
	if after.Sessions != before.Sessions {
		t.Errorf("**أُنشئت جلسةٌ لحسابٍ قائم**: %d ⇐ %d", before.Sessions, after.Sessions)
	}
}

// suToken **توكنُ الوصول إن صدر** — والحقلُ موجودٌ فارغاً في ردّ خطوة الرمز.
func suToken(r Res) string {
	tok, _ := r.JSON()["tokens"].(map[string]any)
	s, _ := tok["access_token"].(string)
	return s
}

func trimBody(r Res) string {
	b := string(r.Body)
	if len(b) > 160 {
		b = b[:160]
	}
	return b
}

// suNoUser **لا حسابَ أُنشئ لهذا الرقم.**
func suNoUser(t *testing.T, h *Harness, phone string) {
	t.Helper()
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM users WHERE phone = $1`, phone).Scan(&n)
	if n != 0 {
		t.Errorf("**أُنشئ حسابٌ لرقمٍ لم يُثبت أحدٌ ملكيّتَه**")
	}
}

func suDropPhone(t *testing.T, h *Harness, phone string) {
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM users WHERE phone = $1`, phone)
		h.Redis().Del(ctxBG(), "login:fail:p:"+phone, "signup:fail:p:"+phone)
	})
}

// ── ١ · الأدوارُ كلُّها والتحقّقُ مطفأ ─────────────────────────────────

func TestSU01_ExistingAccountsAreNotTakenOverWhenVerifyIsOff(t *testing.T) {
	h := New(t)
	suPolicy(h, false)
	for _, roles := range [][]string{
		{"customer"},
		{"driver", "customer"},
		{"merchant", "customer"},
		{"sales", "customer"},
	} {
		t.Run(roles[0], func(t *testing.T) {
			a := suAccount(t, h, roles, "active", true, false)
			before := suRead(t, h, a.ID)
			suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
		})
	}
}

// ── ٢ · الأدمنُ بلا PIN وبـPIN ─────────────────────────────────────────

func TestSU02_AdminCannotBeReachedThroughSignup(t *testing.T) {
	h := New(t)
	suPolicy(h, false)

	t.Run("no_pin", func(t *testing.T) {
		a := suAccount(t, h, []string{"admin", "customer"}, "active", true, false)
		before := suRead(t, h, a.ID)
		res := suConfirm(h, suIP(t, h), a.Phone, "")
		// **ومن نال تحدّياً يحاول إنشاءَ الرمز** — وهذا هو الاستيلاءُ الكامل.
		if ch, _ := res.JSON()["challenge"].(string); ch != "" {
			pin := h.Call("POST", "/api/v1/auth/pin/setup", "", map[string]any{
				"challenge": ch, "pin": "4829",
			}, nil)
			if pin.Code < 400 {
				t.Errorf("**أُنشئ رمزُ أدمن وصدرت جلسةُ أدمن عبر التسجيل** — %d", pin.Code)
			}
		}
		suNoTakeover(t, h, a, before, res)
	})

	t.Run("with_pin", func(t *testing.T) {
		a := suAccount(t, h, []string{"admin", "customer"}, "active", true, false)
		pinHash, _ := auth.HashPassword("739164")
		if _, err := h.Pool.Exec(ctxBG(),
			`UPDATE users SET admin_pin_hash = $2, admin_pin_set_at = now() WHERE id = $1::uuid`,
			a.ID, pinHash); err != nil {
			t.Fatalf("رمزُ الأدمن: %v", err)
		}
		before := suRead(t, h, a.ID)
		suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
	})

	t.Run("passwordless_admin_with_valid_code", func(t *testing.T) {
		a := suAccount(t, h, []string{"admin", "customer"}, "active", false, false)
		code := suPlant(t, h, a.Phone)
		before := suRead(t, h, a.ID)
		suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, code))
	})
}

// ── ٣ · الموقوفُ والمحظورُ لا يُمَسّان ────────────────────────────────

func TestSU03_SuspendedAndBlockedAccountsAreNotAltered(t *testing.T) {
	h := New(t)
	suPolicy(h, false)
	for _, status := range []string{"suspended", "blocked"} {
		t.Run(status+"_with_password", func(t *testing.T) {
			a := suAccount(t, h, []string{"customer"}, status, true, false)
			before := suRead(t, h, a.ID)
			suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
		})
		t.Run(status+"_passwordless_with_valid_code", func(t *testing.T) {
			a := suAccount(t, h, []string{"customer"}, status, false, false)
			code := suPlant(t, h, a.Phone)
			before := suRead(t, h, a.ID)
			suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, code))
		})
	}
}

// ── ٤ · علَمُ «بدّل كلمتك» لا يُمحى بالتسجيل ──────────────────────────

func TestSU04_MustChangePasswordIsNotClearedBySignup(t *testing.T) {
	h := New(t)
	suPolicy(h, false)
	a := suAccount(t, h, []string{"customer"}, "active", true, true)
	before := suRead(t, h, a.ID)
	if !before.Must {
		t.Fatal("التجهيزةُ: العلَمُ لم يُرفع")
	}
	suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
}

// ── ٥ · التحقّقُ مُشعَلٌ: الحسابُ القائمُ محميٌّ ولو بيد المهاجم رمز ────

func TestSU05_VerifyOnStillProtectsExistingAccounts(t *testing.T) {
	h := New(t)
	suPolicy(h, true)
	t.Run("wrong_code", func(t *testing.T) {
		a := suAccount(t, h, []string{"customer"}, "active", true, false)
		before := suRead(t, h, a.ID)
		suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, "000000"))
	})
	t.Run("valid_signup_code_on_password_account", func(t *testing.T) {
		// **ورمزُ تسجيلٍ بقي من قبل أن توضع الكلمة** لا يصير بابَ استعادة.
		a := suAccount(t, h, []string{"customer"}, "active", true, false)
		code := suPlant(t, h, a.Phone)
		before := suRead(t, h, a.ID)
		suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, code))
	})
}

// ── ٦ · فشلُ قراءةِ الإعداد يُغلِق ─────────────────────────────────────

func TestSU06_SettingsReadFailureFailsClosed(t *testing.T) {
	h := New(t)
	h.Setting("launch.customer_signup", "true")
	// **القراءةُ تفشل** — وهكذا يردّ `GetNum` حين يفشل: الاحتياطيَّ الذي مُرِّر.
	h.Identity.SetSettingReader(func(ctx context.Context, key string, fallback int64) int64 {
		if key == "auth.signup_verify" {
			return fallback
		}
		return h.Settings.GetNum(ctx, key, fallback)
	})
	phone := uniqPhone()
	suDropPhone(t, h, phone)
	res := suConfirm(h, suIP(t, h), phone, "")
	if res.Code < 400 {
		t.Errorf("**فشلُ قراءةِ الإعداد فتح التسجيلَ بلا رمز** — %d", res.Code)
	}
	suNoUser(t, h, phone)

	a := suAccount(t, h, []string{"customer"}, "active", true, false)
	before := suRead(t, h, a.ID)
	suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
}

// ── ٧ · التسجيلُ المشروعُ ما زال يعمل ──────────────────────────────────

func TestSU07_LegitimateNewSignupStillWorks(t *testing.T) {
	h := New(t)
	t.Run("verify_on_with_code", func(t *testing.T) {
		suPolicy(h, true)
		phone := uniqPhone()
		suDropPhone(t, h, phone)
		code := suPlant(t, h, phone)
		res := suConfirm(h, suIP(t, h), phone, code)
		if res.Code >= 400 || suToken(res) == "" {
			t.Fatalf("**تسجيلٌ مشروعٌ برمزٍ صحيحٍ رُفض** — %d %s", res.Code, trimBody(res))
		}
		var roles string
		_ = h.Pool.QueryRow(ctxBG(), `
			SELECT string_agg(r.role_code, ',') FROM users u
			JOIN user_roles r ON r.user_id = u.id WHERE u.phone = $1`, phone).Scan(&roles)
		if roles != "customer" {
			t.Errorf("**الحسابُ الجديدُ ليس زبوناً وحده**: %q", roles)
		}
	})
	t.Run("verify_off_without_code", func(t *testing.T) {
		// **قرارُ المالك ٢٠٢٦-٠٨-٢٥**: التحقّقُ حين يُطفأ لا يُطلب لرقمٍ جديد.
		suPolicy(h, false)
		phone := uniqPhone()
		suDropPhone(t, h, phone)
		res := suConfirm(h, suIP(t, h), phone, "")
		if res.Code >= 400 || suToken(res) == "" {
			t.Fatalf("**تسجيلٌ جديدٌ بلا رمزٍ والتحقّقُ مطفأ رُفض** — %d %s", res.Code, trimBody(res))
		}
	})
	t.Run("verify_on_without_code_is_rejected", func(t *testing.T) {
		suPolicy(h, true)
		phone := uniqPhone()
		suDropPhone(t, h, phone)
		if res := suConfirm(h, suIP(t, h), phone, "000000"); res.Code < 400 {
			t.Errorf("**تسجيلٌ برمزٍ خاطئٍ قُبل** — %d", res.Code)
		}
		suNoUser(t, h, phone)
	})
}

// ── ٨ · الحسابُ بلا كلمةٍ لا يُكمَل إلّا برمز ─────────────────────────

func TestSU08_PasswordlessCompletionRequiresProof(t *testing.T) {
	h := New(t)
	suPolicy(h, false)
	t.Run("no_code_rejected", func(t *testing.T) {
		a := suAccount(t, h, []string{"customer"}, "active", false, false)
		before := suRead(t, h, a.ID)
		suNoTakeover(t, h, a, before, suConfirm(h, suIP(t, h), a.Phone, ""))
	})
	t.Run("valid_code_completes", func(t *testing.T) {
		a := suAccount(t, h, []string{"customer"}, "active", false, false)
		code := suPlant(t, h, a.Phone)
		res := suConfirm(h, suIP(t, h), a.Phone, code)
		if res.Code >= 400 || suToken(res) == "" {
			t.Fatalf("**إكمالُ حسابٍ بلا كلمةٍ برمزٍ صحيحٍ رُفض** — %d %s", res.Code, trimBody(res))
		}
		if suRead(t, h, a.ID).Hash == "" {
			t.Errorf("**لم تُوضع الكلمةُ بعد إكمالٍ مشروع**")
		}
	})
}

// ── ٩ · المحاولاتُ المتكرّرةُ تُقفَل بالقفل المركزيّ ─────────────────

func TestSU09_RepeatedConfirmAttemptsAreLimited(t *testing.T) {
	h := New(t)
	suPolicy(h, true)
	t.Run("per_phone", func(t *testing.T) {
		phone := uniqPhone()
		suDropPhone(t, h, phone)
		ip := suIP(t, h)
		last := 0
		for i := 0; i < 8; i++ {
			last = suConfirm(h, ip, phone, "000000").Code
		}
		if last != 429 {
			t.Errorf("**ثماني محاولاتٍ خاطئةٍ لرقمٍ واحدٍ بلا قفل** — آخرُها %d", last)
		}
	})
	t.Run("per_ip", func(t *testing.T) {
		ip := suIP(t, h)
		last := 0
		for i := 0; i < 34; i++ {
			phone := uniqPhone()
			suDropPhone(t, h, phone)
			last = suConfirm(h, ip, phone, "000000").Code
		}
		if last != 429 {
			t.Errorf("**أربعٌ وثلاثون محاولةً من عنوانٍ واحدٍ بلا قفل** — آخرُها %d", last)
		}
	})
}

// ── ١٠ · بابُ التسجيل المغلقُ مغلقٌ للتأكيد أيضاً ─────────────────────

func TestSU10_SignupLaunchClosureCoversConfirm(t *testing.T) {
	h := New(t)
	suWire(h)
	h.Setting("launch.customer_signup", "false")
	h.Setting("auth.signup_verify", "false")
	phone := uniqPhone()
	suDropPhone(t, h, phone)
	res := suConfirm(h, suIP(t, h), phone, "")
	if res.Code != 503 || !strings.Contains(string(res.Body), "launch_closed") {
		t.Errorf("**التأكيدُ يتجاوز إغلاقَ بابِ التسجيل** — %d %s", res.Code, trimBody(res))
	}
	suNoUser(t, h, phone)
}
