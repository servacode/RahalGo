package notifications

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **قاعدةُ الإشعار — ولا إشعارَ بلا فائدة**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٤: «لا نريد إشعاراتٍ كثيرةً بلا فائدة».)
//
// # ما قِيس قبله
//
// **صندوقُ سائقٍ حقيقيٍّ: عشرةُ أسطرٍ من سبعةَ عشر كانت «طلب جديد
// بانتظارك»** — لطلباتٍ انقضت مهلتُها وذهبت لغيره. **وبينها أجرُ توصيلٍ
// يعنيه فعلاً.**
//
// **ومن وصله عشرةٌ لا يفتح الحادي عشر** — فيضيع الذي كان يعنيه.
//
// # والقاعدةُ سطران
//
//	ما يُنتظَر فيه فعلٌ الآن   →  يرنّ ولا يُحفَظ   (Transient)
//	ما يُقرأ لاحقاً وله أثر    →  يُحفَظ ولا يرنّ    (Silent)
//
// # ولماذا حارس
//
// **العَلَمان سطرٌ واحدٌ في كلّ نداء** — ومن نسيه عاد الضجيجُ صامتاً:
// **لا خطأَ ولا سقوط، إنّما صندوقٌ يمتلئ شهراً بعد شهر** حتّى يكفّ
// صاحبُه عن فتحه.

// TestBothFlagsAreHonoured **العَلَمان يعملان فعلا.**
//
// **ويُقرأ التنفيذُ من الشيفرة** — لا من ذاكرة من كتبه: **حقلٌ يُضاف
// إلى بنيةٍ ولا يُقرأ في `Notify` عَلَمٌ ميّت.**
func TestBothFlagsAreHonoured(t *testing.T) {
	src, err := os.ReadFile("notifications.go")
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	s := string(src)

	// **العابرُ لا يُدرَج** — يُفحص قبل الإدراج.
	if !regexp.MustCompile(`if in\.Transient \{`).MatchString(s) {
		t.Error("Transient لا يُفحص قبل الإدراج — الخبرُ العابرُ يُحفَظ فيمتلئ الصندوق")
	}
	// **والصامتُ لا يُدفَع.**
	if !regexp.MustCompile(`s\.pusher != nil && !in\.Silent`).MatchString(s) {
		t.Error("Silent لا يُفحص قبل الدفع — يرنّ ما لا فعلَ فيه")
	}
}

// TestOfferIsTransientAndMoneyIsSilent **وحيث تُطبَّق القاعدة.**
//
// **وقائمةٌ صحيحةٌ لا ينفع إن لم يقرأها من يُرسل** — وهو ما وقع في
// أسباب البلاغ: الجهةُ مكتوبةٌ منذ اليوم الأوّل ولم تكن تُقرأ.
func TestOfferIsTransientAndMoneyIsSilent(t *testing.T) {
	src, err := os.ReadFile("../orders/notify.go")
	if err != nil {
		t.Fatalf("تعذّرت القراءة: %v", err)
	}
	s := string(src)

	offer := between(s, "func (s *Service) notifyOffer", "\n}\n")
	if !strings.Contains(offer, "Transient: transient") {
		t.Error("عرضُ الطلب يُحفَظ في الصندوق — وهو خبرٌ يموت بعد دقيقتين")
	}
	if !strings.Contains(offer, "transient := title == t.offerDriver") {
		t.Error("العرضُ والإسنادُ سواءٌ في الحفظ — والإسنادُ طلبٌ صار في يده يُسأل عنه غدا")
	}

	credits := between(s, "func (s *Service) notifyCredits", "\n}\n")
	if !strings.Contains(credits, "Silent: c.app == notifications.AppDriver") {
		t.Error("مالُ السائق يرنّ — وهو يقف عند الباب وقد ضغط «سلّمت» قبل ثانية")
	}
	// **ولا يُسكَت زبونُه** — استرجاعٌ يصله وهو خارج التطبيق.
	if strings.Contains(credits, "Silent: true") {
		t.Error("أُسكت المالُ للجميع — ومالُ الزبون يعود بلا خبرٍ يُقرأ ضياعا")
	}
}

// between ما بين علامتين — **وفارغٌ إن لم تُوجد الأولى.**
func between(s, from, to string) string {
	i := strings.Index(s, from)
	if i < 0 {
		return ""
	}
	j := strings.Index(s[i:], to)
	if j < 0 {
		return s[i:]
	}
	return s[i : i+j]
}
