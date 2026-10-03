package server

// ══════════════════════════════════════════════════════════════════════
// **حدودُ النصوص التي يكتبها الزبون — تُرفض لا تُقصّ ولا تُخزَّن كما هي**
// ══════════════════════════════════════════════════════════════════════
//
// (فحصُ القبول ٢٠٢٦-١٠-٠٣ — بندٌ أمنيّ: `POST /orders` قبل `address_text`
//  من عشرين ألف حرف **وخزّنه**.)
//
// **ونصٌّ بلا حدٍّ بابٌ مفتوح**: يملأ القاعدةَ، ويُثقل كلَّ شاشةٍ تعرض
// الطلب (السائقُ والمتجرُ والمكتب)، **ويُطبع في الإيصال كما هو.**
//
// # ولماذا رفضٌ لا قصّ
//
// **القصُّ الصامتُ يُخزّن غيرَ ما كتبه صاحبُه** — عنوانٌ مقطوعٌ في منتصفه
// يُرسل السائقَ إلى نصف شارع. **والرفضُ يقول له «اختصر» فيختصر هو.**
//
// **والعدُّ بالحروف لا بالبايتات**: الحرفُ العربيُّ بايتان، **وقصٌّ بالبايت
// يكسر الحرفَ الأخير** (وهو ما كانت تفعله `clip` في العناوين).
//
// # والحدودُ نفسُها في التطبيق
//
// `maxLength` في حقول تطبيق الزبون **مرآةٌ لهذه الأرقام** — والحارسُ هنا:
// **وزرٌّ يمنع في الهاتف ليس منعاً.**

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// حدودُ النصوص بالحروف.
const (
	// maxAddressText **أطولُ من أطول سطرٍ تركّبه أجزاءُ العنوان المحفوظ**
	// (`addressLine`: ١٥٠ + ١٥٠ + طابقٌ ٢٠ + فواصل) — **وإلّا رُفض طلبٌ
	// عنوانُه محفوظٌ مقبول.** يحرسه `TestAddressLine_FitsOrderLimit`.
	maxAddressText   = 400
	maxOrderNotes    = 500
	maxItemNote      = 200
	maxCustomRequest = 1000
	maxComplaintNote = 1000
	maxTicketReply   = 1000
	maxAddressPart   = 150
	// maxAddressFloor **الطابقُ رقمٌ قصير** — وكان يُقصّ بعشرين قبلُ.
	maxAddressFloor = 20
	// maxChatBody **حدُّ الحديث هو حدُّ خدمته** — `comms.MaxBody`: **ورقمان
	// لشيءٍ واحدٍ يفترقان**، فيقبل البابُ ما تقصّه الخدمةُ بصمت.
	maxChatBody = comms.MaxBody
)

// textField **حقلٌ واسمُه وحدُّه** — والاسمُ يعود في التفاصيل ليعرف التطبيقُ أيَّها.
type textField struct {
	name  string
	value string
	max   int
}

// checkTextLimits **أوّلُ حقلٍ تجاوز حدَّه يُرفض بـ٤٠٠** — `text_too_long`.
//
// **والتفاصيلُ نصوصٌ لا أرقام** — نموذجُ الخطأ في التطبيق `Map<String, String>`،
// **ورقمٌ فيه يُسقط فكَّ الردّ كلَّه** فيُقرأ «تعذّر الاتصال».
func checkTextLimits(fields ...textField) error {
	for _, f := range fields {
		if utf8.RuneCountInString(f.value) > f.max {
			return &httpx.AppError{
				Status:     http.StatusBadRequest,
				Code:       "text_too_long",
				MessageKey: "errors.text_too_long",
				Details:    map[string]any{"field": f.name, "max": strconv.Itoa(f.max)},
			}
		}
	}
	return nil
}
