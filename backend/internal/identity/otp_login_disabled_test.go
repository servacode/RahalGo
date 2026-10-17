package identity

import (
	"context"
	"errors"
	"testing"
)

// TestOTPLoginDisabledSendsNoCode **بابُ الرمز مطفأٌ ⇒ لا رمزَ يُرسَل ولا يُقبل** (٢٠٢٦-١٠-١٧).
//
// **كان `RequestOTP` لا يقرأ `auth.otp_login` أصلاً** — الإعدادُ يُخفي التبويبَ
// في الشاشة، والبابُ يُرسل رمزاً لكلّ من ناداه. **والخدمةُ هنا بلا ذاكرةٍ ولا
// قاعدة**: إن تخطّت الرفضَ ومضت إلى العدّ أو الإرسال سقطت — وهو الدليلُ أنّها
// كانت ستُرسل.
func TestOTPLoginDisabledSendsNoCode(t *testing.T) {
	ctx := context.Background()
	svc := &Service{}
	svc.SetSettingReader(func(_ context.Context, key string, fallback int64) int64 {
		if key == "auth.otp_login" {
			return 0
		}
		return fallback
	})

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("**الرمزُ مضى إلى الإرسال والبابُ مطفأ**: %v", r)
			}
		}()
		if err := svc.RequestOTP(ctx, "0933123456", "1.2.3.4"); !errors.Is(err, ErrOTPLoginDisabled) {
			t.Fatalf("**طلبُ الرمز والبابُ مطفأ**: %v — والمنتظَر otp_login_disabled", err)
		}
	}()
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("**التحقّقُ مضى إلى القاعدة والبابُ مطفأ**: %v", r)
			}
		}()
		if _, err := svc.VerifyOTP(ctx, "0933123456", "123456", "ua", "1.2.3.4"); !errors.Is(err, ErrOTPLoginDisabled) {
			t.Fatalf("**التحقّقُ والبابُ مطفأ**: %v", err)
		}
	}()
}
