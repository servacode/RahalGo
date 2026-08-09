package server

// **لا رقمَ يعبر بين طرفَي الطلب.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
//
//	المنصّة فقط».)
//
// # ولماذا حارسٌ لا مراجعةٌ بالعين
//
// **الرقمُ يتسرّب بإضافةِ عمودٍ لا بقرارِ أحد.** من كتب `SELECT` جديداً ونسخ
// سطراً من استعلامٍ قديم يجرّ معه `cu.phone`، **ويمرّ البناءُ ويمرّ الاختبار
// وتعمل الشاشة** — ولا شيءَ يقول إنّ سياسةً نُقضت.
//
// **وقد كان موجوداً فعلاً**: `customer_phone` في حمولة السائق مع رابط `tel:`
// يفتح المتّصلَ على الرقم مباشرة.
//
// # وما يُفحص
//
// **مصادرُ ما يبلغ السائق** — لا الشاشة. **رقمٌ يُرسَل ثمّ يُخفى في الواجهة
// موجودٌ لمن فتح أدوات المتصفّح**: الإخفاءُ في الشاشة ستارةٌ لا قفل.
//
// **ورقمُ المتجر مستثنًى**: السائقُ يتّصل بالمطبخ ليسأل عن طلبٍ تأخّر —
// **وهو طرفُ عملٍ لا طرفُ خصوصيّة**، ورقمُه على لافتته.

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// driverFacingFiles **الملفّاتُ التي تُبنى منها حمولاتُ السائق.**
//
// **وتُسمّى صراحةً لا تُستنتَج**: من أضاف ملفّاً ثالثاً يجب أن يضيفه هنا،
// **وقائمةٌ تُحدَّث بيدٍ خيرٌ من استنتاجٍ يمرّ صامتاً.** (وأرضيّةُ العدد أدناه
// تكشف من حذف بدل أن يضيف.)
var driverFacingFiles = []string{
	"driver_handlers.go",
	"driver_history.go",
	"delivery_proof.go",
}

// customerPhoneTag أيُّ حقلٍ يحمل رقمَ الزبون إلى الخارج.
var customerPhoneTag = regexp.MustCompile(`json:"(customer_phone|phone)"`)

// customerPhoneCol وأيُّ عمودٍ يُنتقى من صفّ الزبون.
var customerPhoneCol = regexp.MustCompile(`\bcu\.phone\b|\bcustomer\.phone\b`)

func TestDriverPayloadCarriesNoCustomerPhone(t *testing.T) {
	seen := 0
	for _, name := range driverFacingFiles {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("لم يُقرأ %s: %v — أحُذف أم أُعيدت تسميتُه؟", name, err)
		}
		seen++
		src := string(raw)

		for i, line := range strings.Split(src, "\n") {
			// **والتعليقُ يذكر ما مُنع** — فلا يُحسب ذكرُه نقضاً له.
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "//") {
				continue
			}
			if customerPhoneTag.MatchString(line) {
				t.Errorf("%s:%d يُصدّر رقمَ الزبون إلى السائق:\n    %s\n"+
					"    **والتواصلُ من `/orders/{id}/messages` لا من رقم.**",
					name, i+1, strings.TrimSpace(line))
			}
			if customerPhoneCol.MatchString(line) {
				t.Errorf("%s:%d ينتقي رقمَ الزبون من القاعدة:\n    %s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
	if seen != len(driverFacingFiles) {
		t.Fatalf("قُرئ %d من %d ملفّاً", seen, len(driverFacingFiles))
	}
}

// TestCustomerPayloadCarriesNoDriverPhone **والاتّجاهُ الآخرُ كذلك.**
//
// **وهو مغلقٌ اليوم** — `DriverPhone` تستهلكه شاشاتُ الإدارة وحدَها.
// **والحارسُ يمنع أن يُفتح** حين يطلب أحدٌ «زرّ اتّصال بالسائق» فيبدو الحقلُ
// جاهزاً في الحمولة.
func TestCustomerPayloadCarriesNoDriverPhone(t *testing.T) {
	for _, name := range []string{"customer_handlers.go", "comms_handlers.go"} {
		raw, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("لم يُقرأ %s: %v", name, err)
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if t := strings.TrimSpace(line); strings.HasPrefix(t, "//") {
				continue
			}
			if strings.Contains(line, `json:"driver_phone"`) ||
				strings.Contains(line, "dr.phone") {
				t.Errorf("%s:%d يُصدّر رقمَ السائق إلى الزبون:\n    %s",
					name, i+1, strings.TrimSpace(line))
			}
		}
	}
}
