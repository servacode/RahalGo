package identity

// ══════════════════════════════════════════════════════════════════════
// **عينُ الأمان على باب الدخول** (مراقبةُ المنصّة — الدفعةُ الثانية ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **قفلُ الرقم بعد خمس محاولاتٍ يعمل ولا يعلم به أحد** — وحدُّ الرموز كذلك.
// فمن يجرّب كلماتِ سرٍّ على حساب متجرٍ ظاهرٍ رقمُه في السوق **يُردّ بصمت**،
// ويجرّب غداً، والمالكُ لا يدري أنّ أحداً يطرق الباب.
//
// # وما صار
//
// **الحمايةُ كما هي حرفاً بحرف** — العدّاداتُ نفسُها والحدودُ نفسُها والردودُ
// نفسُها. **وزاد شيءٌ واحد**: كلُّ رفضٍ يُخبَر به راصدُ الأمان في الخادم
// (`server/monitoring_security.go`) عبر دالّةٍ تُحقن مرّةً عند الإقلاع.
//
// # ولا رقمَ يخرج من هنا
//
// **الحدثُ يحمل آخرَ ثلاثة أرقامٍ لا غير** (`PhoneTail`) — وبصمةً لا تُعكس
// (`Subject`) يُمنع بها تكرارُ التنبيه للرقم نفسِه. **فلا الرقمُ الكامل ولا
// الكلمةُ ولا الرمزُ يغادر خدمةَ الهويّة.**

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

// أنواعُ أحداث الأمان التي تُبلَّغ من الهويّة.
const (
	// SecLoginFailed **كلمةُ سرٍّ خاطئة** — أو رقمٌ لا حسابَ له (يُعدّ مثلَها).
	SecLoginFailed = "login_failed"
	// SecLoginLocked **انقفل الرقمُ** — بلغ `security.login_max_attempts` الآن.
	SecLoginLocked = "login_locked"
	// SecOTPLimited **رُدّ طلبُ رمزٍ لأنّه تجاوز الحدّ** — للرقم أو للعنوان.
	SecOTPLimited = "otp_limited"
)

// SecurityEvent **حدثُ أمانٍ واحد** — بلا رقمٍ كاملٍ ولا سرّ.
type SecurityEvent struct {
	Kind string
	// PhoneTail **آخرُ ثلاثة أرقام** — وحدَها تُكتب في التنبيه.
	PhoneTail string
	// Subject **بصمةُ الرقم** — تُميّز رقماً من رقمٍ ولا تُعيده.
	Subject string
}

// SetSecurityHook **يحقن مستمعَ أحداث الأمان** — يُنادى مرّةً عند الإقلاع.
//
// **وبلا مستمعٍ لا يتغيّر شيء** — الاختباراتُ القديمةُ تعمل كما كانت.
func (s *Service) SetSecurityHook(fn func(ctx context.Context, ev SecurityEvent)) {
	s.securityHook = fn
}

// noteSecurity **يُبلغ حدثاً** — ولا يُعيد شيئاً ولا يُبطئ الردّ.
func (s *Service) noteSecurity(ctx context.Context, kind, phone string) {
	if s.securityHook == nil {
		return
	}
	s.securityHook(ctx, SecurityEvent{Kind: kind, PhoneTail: PhoneTail(phone), Subject: s.phoneSubject(phone)})
}

// otpLimited **الردُّ نفسُه كما كان** (`ErrOTPRateLimited`) — ويُبلَّغ قبله.
func (s *Service) otpLimited(ctx context.Context, phone string) error {
	s.noteSecurity(ctx, SecOTPLimited, phone)
	return ErrOTPRateLimited
}

// PhoneTail **آخرُ ثلاثة أرقامٍ من الرقم** — وما قصر عنها يُعاد فارغاً.
func PhoneTail(phone string) string {
	digits := make([]byte, 0, len(phone))
	for i := 0; i < len(phone); i++ {
		if phone[i] >= '0' && phone[i] <= '9' {
			digits = append(digits, phone[i])
		}
	}
	if len(digits) < 3 {
		return ""
	}
	return string(digits[len(digits)-3:])
}

// phoneSubject **بصمةُ الرقم** — HMAC بسرّ الخادم، ستّةَ عشرَ حرفاً.
//
// **ولا `sha256` عارية**: الأرقامُ السوريّةُ مئةُ مليونٍ تُجرَّب كلُّها في دقائق،
// **وبصمةٌ تُعكس بالتجريب رقمٌ مكشوفٌ بخطوةٍ زائدة.**
func (s *Service) phoneSubject(phone string) string {
	mac := hmac.New(sha256.New, []byte(s.secret))
	mac.Write([]byte("sec:" + phone))
	return hex.EncodeToString(mac.Sum(nil)[:8])
}
