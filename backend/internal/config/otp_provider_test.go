package config

// **مزوّدُ الرمز في الإنتاج لا يكون `dev`.**
//
// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧، وأُصلح بقرار المالك: «أكمل».)
//
// # المرض
//
// `DevSender` **يطبع رمزَ التحقّق في سجلّ الخادم ولا يرسله**. وهو الصحيح
// في التطوير — ومن أقلع بالإنتاج ونسي `OTP_PROVIDER` **فتح البابَ لمن يقرأ
// السجلّ**: رمزُ دخولِ أيِّ حساب، مطبوعاً بجانب رقمه.
//
// **ولا يظهر في أيّ خطأ**: الخادمُ يقلع، والدخولُ يعمل، **والرمزُ يصل من
// لا يجب أن يصله وحدَه.**
//
// # ولماذا حارسٌ لا انتباه
//
// **`JWT_SECRET` له حارسٌ منذ زمنٍ يرفض الافتراضيَّ في الإنتاج** — ونُسي
// أخوه. **وقاعدةٌ يحرسها الانتباه تسقط** عند أوّل نشرٍ مستعجل.

import (
	"strings"
	"testing"
)

func TestDevOTPProviderIsRefusedInProduction(t *testing.T) {
	base := map[string]string{
		"APP_ENV":      "production",
		"DATABASE_URL": "postgres://u:p@h/db",
		"REDIS_URL":    "redis://h:6379/0",
		"JWT_SECRET":   "a-real-secret-not-the-dev-default",
	}

	// **`dev` في الإنتاج يُرفض.**
	env := clone(base)
	env["OTP_PROVIDER"] = "dev"
	for k, v := range env {
		t.Setenv(k, v)
	}
	if _, err := Load(); err == nil {
		t.Fatal("**قُبل `OTP_PROVIDER=dev` في الإنتاج** — ورمزُ الدخول يُطبع في السجلّ " +
			"بجانب رقمِ صاحبه، **ولا خطأَ يقول ذلك.**")
	} else if !strings.Contains(err.Error(), "OTP_PROVIDER") {
		t.Fatalf("رُفض بسببٍ آخر: %v", err)
	}

	// **و`whatsapp` يمرّ** — فحارسٌ يمنع كلَّ شيءٍ لا يمنع شيئاً.
	t.Setenv("OTP_PROVIDER", "whatsapp")
	if _, err := Load(); err != nil {
		t.Fatalf("رُفض المزوّدُ الصحيح: %v", err)
	}

	// **وفي التطوير `dev` هو الصواب** — ولا يُمنع.
	t.Setenv("APP_ENV", "development")
	t.Setenv("OTP_PROVIDER", "dev")
	if _, err := Load(); err != nil {
		t.Fatalf("مُنع `dev` في التطوير: %v", err)
	}
}

func clone(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
