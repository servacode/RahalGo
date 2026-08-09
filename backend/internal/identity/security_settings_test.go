package identity

// **ثلاثةُ قراراتِ أمانٍ نزلت من الشيفرة إلى اللوحة.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩ بعد جردِ الثوابت: «نطبّقها كلَّها».)
//
// # ولماذا اختبارٌ للوحدة لا للقراءة
//
// **الخطرُ ليس أن تُقرأ القيمةُ، بل أن تُقرأ بوحدةٍ غير وحدتها.**
//
// المفتاحُ يقول «يوماً» والدالّةُ تردّ `time.Duration`، **ورقمٌ يُضرب في
// ساعةٍ بدل يومٍ يجعل الجلسةَ ثلاثين ساعةً لا ثلاثين يوماً** — ولا يسقط
// اختبار، ولا يُخطئ بناء، **ويكتشفه زبونٌ يُخرَج من حسابه كلَّ يومٍ ونصف
// ولا يعرف لماذا.**
//
// **والاحتياطيُّ يُختبر كذلك**: خدمةٌ بلا ربطٍ باللوحة يجب أن تعمل بالثوابت
// — **وصفرٌ مكانَ الاحتياطيّ يعني رمزاً يموت لحظةَ ولادته.**

import (
	"context"
	"testing"
	"time"
)

func TestSecuritySettingsReadWithTheRightUnit(t *testing.T) {
	ctx := context.Background()

	// **بلا ربطٍ باللوحة** — الاحتياطيّاتُ هي ما يعمل.
	bare := &Service{}
	if got := bare.otpLifetime(ctx); got != otpTTL {
		t.Errorf("مهلةُ الرمز بلا لوحة = %v، والاحتياطيُّ %v", got, otpTTL)
	}
	if got := bare.otpQuota(ctx); got != otpMaxPer15m {
		t.Errorf("سقفُ الرموز بلا لوحة = %d، والاحتياطيُّ %d", got, otpMaxPer15m)
	}
	if got := bare.sessionLife(ctx); got != refreshTTL {
		t.Errorf("طولُ الجلسة بلا لوحة = %v، والاحتياطيُّ %v", got, refreshTTL)
	}

	// **ومع لوحةٍ تردّ أرقاماً غيرَ الافتراضيّة** — لِيُرى أنّها تُقرأ حقّاً،
	// **وأنّ كلَّ رقمٍ يُضرب في وحدته هو.**
	svc := &Service{}
	svc.SetSettingReader(func(_ context.Context, key string, fallback int64) int64 {
		switch key {
		case "security.otp_ttl_min":
			return 12
		case "security.otp_max_per_phone":
			return 7
		case "security.session_days":
			return 3
		}
		return fallback
	})

	if got, want := svc.otpLifetime(ctx), 12*time.Minute; got != want {
		t.Errorf("مهلةُ الرمز = %v، والمنتظَر %v — أدقيقةٌ قُرئت ثانيةً؟", got, want)
	}
	if got, want := svc.otpQuota(ctx), int64(7); got != want {
		t.Errorf("سقفُ الرموز = %d، والمنتظَر %d", got, want)
	}
	if got, want := svc.sessionLife(ctx), 3*24*time.Hour; got != want {
		t.Errorf("طولُ الجلسة = %v، والمنتظَر %v — أيومٌ قُرئ ساعةً؟", got, want)
	}
}
