package identity

// **مضيفٌ واحدٌ يطلب رموزاً لأرقامٍ لا تنتهي.**
//
// (كشفه فحصُ المشروع ٢٠٢٦-٠٨-٠٧، وأُصلح بقرار المالك: «أكمل».)
//
// # المرض
//
// حدُّ الرموز كان لكلّ **رقم**: ثلاثةٌ في ربع ساعة. **ولا بُعدَ لعنوان
// الشبكة** — بينما مسارُ الدخول له `loginMaxPerIP`.
//
// **فمن ملك مضيفاً واحداً أرسل ثلاثةَ رموزٍ لكلّ رقمٍ في سوريا** ولم يبلغ
// حدّاً قطّ: **رسائلُ لا يريدها أصحابُها**، ورصيدُ واتساب يُحرق، ورقمُ
// المنصة يُبلَّغ عنه سبَماً فيُحجب.
//
// **ولا يظهر في أيّ سجلّ**: كلُّ طلبٍ صالحٌ ويردّ ٢٠٠.
//
// # ولماذا الحدُّ للعنوان أوسعُ من الحدّ للرقم
//
// **مقهىً أو مكتبٌ يشترك فيه عشرة** — وحدٌّ ضيّقٌ يمنع من لم يُخطئ.

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/redis/go-redis/v9"
)

// testRedis يفتح ردِس الاختبار أو يتخطّى.
func testRedis(t *testing.T) *redis.Client {
	t.Helper()
	url := os.Getenv("TEST_REDIS_URL")
	if url == "" {
		url = "redis://localhost:6380/9"
	}
	opt, err := redis.ParseURL(url)
	if err != nil {
		t.Skipf("TEST_REDIS_URL غير صالح: %v", err)
	}
	c := redis.NewClient(opt)
	if err := c.Ping(context.Background()).Err(); err != nil {
		t.Skipf("ردِس لا يستجيب — يُتخطّى: %v", err)
	}
	if err := c.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("تعذّر تفريغُ قاعدة الاختبار: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

func TestOTPIsCappedPerIP(t *testing.T) {
	rdb := testRedis(t)
	svc := &Service{rdb: rdb}

	const ip = "203.0.113.7"

	// **أرقامٌ مختلفةٌ في كلّ مرّة** — فحدُّ الرقم لا يمسّها.
	sent := 0
	var lastErr error
	for i := 0; i < otpMaxPerIP+5; i++ {
		phone := "+96393" + pad(i)
		if err := svc.noteOTPSend(context.Background(), phone, ip); err != nil {
			lastErr = err
			break
		}
		sent++
	}

	if sent > otpMaxPerIP {
		t.Fatalf("**أُرسل %d رمزاً من عنوانٍ واحدٍ لأرقامٍ مختلفة** — والحدُّ %d. "+
			"لا حدَّ للعنوان: مضيفٌ واحدٌ يقصف كلَّ رقمٍ في البلد، **ورصيدُ "+
			"واتساب يُحرق ورقمُ المنصة يُبلَّغ عنه سبَماً.**", sent, otpMaxPerIP)
	}
	if !errors.Is(lastErr, ErrOTPRateLimited) {
		t.Fatalf("توقّفَ الإرسالُ بخطأٍ غيرِ متوقَّع: %v", lastErr)
	}

	// **وعنوانٌ آخرُ لا يُعاقَب بذنب غيرِه.**
	if err := svc.noteOTPSend(context.Background(), "+963930000999", "198.51.100.4"); err != nil {
		t.Fatalf("عنوانٌ نظيفٌ مُنع: %v", err)
	}
}

func pad(i int) string {
	s := "0000000" + itoa(i)
	return s[len(s)-7:]
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b strings.Builder
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	b.WriteString(digits)
	return b.String()
}
