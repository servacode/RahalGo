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
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/catalog"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// حدودُ النصوص بالحروف.
const (
	// maxPersonName **اسمُ شخص** — في التسجيل وتعديل الحساب.
	maxPersonName = 80
	// maxStoreName **اسمُ متجر** — حدُّ اسم الصنف نفسُه (`catalog.MaxItemName`).
	maxStoreName = 120
	// maxDescription **وصفُ متجرٍ أو صنف.**
	maxDescription = 500
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
	// maxStageNote **ملاحظةُ بلاغ السائق** — كانت تُقصّ بثلاثمئة بصمت.
	maxStageNote = 300
)

// tf **حقلٌ للحارس** — اسمُه وموضعُ قيمته وحدُّه ونوعُه. **وقيمتُه تُنظَّف في موضعها.**
func tf(name string, v *string, max int, p textguard.Policy) textguard.Field {
	return textguard.Field{Name: name, Value: v, Max: max, Policy: p}
}

// guardText **يمرّ بالحارس المركزيّ** (`textguard`) — نداءٌ واحدٌ لكلّ باب.
//
// **ويردّ الألفاظَ التي أُخفيت** في الحديث والشكوى — وصاحبُ الباب يُنبّه بها
// الإدارةَ (`alertOffensive`) **ومعها رابطُ ما وقعت فيه.**
func (s *Server) guardText(ctx context.Context, fields ...textguard.Field) ([]string, error) {
	return s.textguard.Apply(ctx, fields...)
}

// alertOffensive **لفظٌ مسيءٌ أُخفي في حديثٍ أو شكوى — تُخبَر الإدارة.**
//
// (قرارُ المالك ٢٠٢٦-١٠-٠٣: «في الحديث والشكاوى إخفاءٌ وتنبيهٌ للإدارة —
//
//	شكوى الزبون الغاضب تصل، ويُعرف من يسبّ».)
//
// **ولا يُكتب اللفظُ في الإشعار** — يُقرأ على شاشةٍ مقفلة؛ **ويبقى في السجلّ**
// (`order_messages.flag_word`) لمن يراجع.
func (s *Server) alertOffensive(ctx context.Context, masked []string, body, entity, entityID, href string) {
	if len(masked) == 0 || s.notify == nil {
		return
	}
	s.notify.NotifyOps(ctx, notifications.Input{
		Kind: notifications.KindTicket, Title: notifTitles.offensiveText, Body: body,
		Entity: entity, EntityID: entityID, Href: href,
	})
}

// guardMenuItem **اسمُ الصنف ووصفُه** — للمتجر والمندوب والإدارة معاً.
func (s *Server) guardMenuItem(r *http.Request, in *catalog.MenuItemInput) error {
	// **وخطأُ الطول خطأُ القائمة القائم** (`item_text_too_long`) — لا يتبدّل عقدُها.
	name := tf("name", in.Name, catalog.MaxItemName, textguard.Name)
	name.TooLong = catalog.ErrItemTextTooLong
	desc := tf("description", in.Description, catalog.MaxItemDescription, textguard.Notes)
	desc.TooLong = catalog.ErrItemTextTooLong
	_, err := s.guardText(r.Context(), name, desc)
	return err
}

// guardMerchant **اسمُ المتجر ووصفُه وعنوانُه.**
func (s *Server) guardMerchant(r *http.Request, in *catalog.MerchantInput) error {
	_, err := s.guardText(r.Context(),
		tf("name", in.Name, maxStoreName, textguard.Name),
		tf("description", in.Description, maxDescription, textguard.Notes),
		tf("address_text", in.AddressText, maxAddressText, textguard.Address),
	)
	return err
}
