package server

// إرسالُ الطلب إلى المتجر برسالةٍ نصّية.
//
// ## لماذا
//
// **ليس كلُّ متجرٍ يجلس إلى شاشة.** مطعمٌ صغير في الرقة لا يملك جهازاً في
// المطبخ ولا من يراقبه — يعمل على واتساب كما يعمل يومَه كلَّه. وإلزامُه ببوابةٍ
// يفتحها يعني طلباتٍ تتأخّر حتى يتذكّر أحدهم أن ينظر.
//
// فصار للمنصة وضعان (`merchants.self_manage_orders`):
//
//   - **المتجر يدير**: يستقبل الطلب في بوابته ويقبله ويحضّره — وهذا الأصل.
//   - **المنصة تدير**: العمليات تقبل نيابةً عنه، ثم **تُرسل له الطلب برسالةٍ
//     نصّية**. فيصله في هاتفه بلا تطبيقٍ ولا اقتران.
//
// ## ولماذا رسالةٌ نصّية لا بوت واتساب
//
// بوتُ واتساب عندنا **غيرُ رسميّ** (`whatsmeow`): يحتاج اقتراناً بهاتف،
// وينقطع، **ويُحظَر حسابُه إن أكثر من الإرسال الآليّ** — وحظرُه يُوقف رموزَ
// التحقّق معه، فيُعطَّل الدخول كلُّه لأجل إبلاغِ مطعم.
//
// والرسالةُ النصّية تصل أيَّ هاتفٍ بلا شيء. **وواتسابٌ رسميّ لاحقاً** — يدخل
// من الواجهة نفسها بلا تغييرٍ فيمن يستعملها.
//
// ## وماذا في الرسالة
//
// **رقمُ الطلب والأصنافُ وملاحظةُ الزبون — ولا شيء غير ذلك.**
//
// لا اسمَ زبونٍ ولا هاتفَ ولا عنوانَ ولا مبلغ: **الرسالةُ تخضع لما يخضع له
// الردّ** (`merchant_privacy.go`) — وإلّا صارت الرسالةُ باباً خلفياً لما سُدّ
// في الواجهة. **وثغرةٌ في قناةٍ ثانية تُبطل الحجب في الأولى.**
//
// والمالُ خارجها كذلك: هي **ورقةُ مطبخ** تقول ماذا يُطبخ وما يُراعى فيه.

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
)

var (
	errNoMerchantPhone  = httpx.NewError(http.StatusConflict, "no_merchant_phone", "errors.no_merchant_phone")
	errSMSNotConfigured = httpx.NewError(http.StatusServiceUnavailable, "sms_not_configured", "errors.sms_not_configured")
	errSMSFailed        = httpx.NewError(http.StatusBadGateway, "sms_failed", "errors.sms_failed")
	errBadChannel       = httpx.NewError(http.StatusBadRequest, "bad_channel", "errors.bad_channel")
)

// buildMerchantMessage نصُّ الرسالة — مصدرٌ واحد يقرؤه الإرسالُ والمعاينة.
//
// **ولو بُني في الواجهة لاختلف عمّا يُرسَل**: يرى الموظّف نصّاً ويصل المتجرَ
// غيرُه، ولا يكتشف ذلك أحدٌ حتى يشتكي متجر.
// **والقالبُ من اللوحة لا من الشيفرة** — (قرارُ المالك ٢٠٢٦-٠٨-١٠: «رسالةُ
// الطلبات كيف رح يكون القالبُ تبعها — لازم تكون موجودة بلوحة الأدمن»).
//
// **ونصٌّ يصل متجراً ويُصحَّح بنشرٍ ليس نصّاً — هو إصدار.**
//
// **والبنودُ تُحقن في {items}**: عددُها يتبدّل بكلّ طلب، **فلا تُكتب بيد.**
// و{number} رقمُ الطلب.
//
// **وقالبٌ فارغٌ يعود إلى الافتراض** — **ورسالةٌ فارغةٌ تصل متجراً أسوأُ من
// قالبٍ لم يُعدَّل.**
func buildMerchantMessage(tpl string, o *orderMessage) string {
	var b strings.Builder
	for _, it := range o.Items {
		fmt.Fprintf(&b, "• %s ×%d\n", it.Name, it.Qty)
		if len(it.Options) > 0 {
			fmt.Fprintf(&b, "   %s\n", strings.Join(it.Options, "، "))
		}
		if it.Note != "" {
			fmt.Fprintf(&b, "   ملاحظة: %s\n", it.Note)
		}
	}
	if o.Notes != "" {
		fmt.Fprintf(&b, "ملاحظة الزبون: %s\n", o.Notes)
	}
	items := strings.TrimRight(b.String(), "\n")
	if strings.TrimSpace(tpl) == "" {
		tpl = "طلب جديد #{number}\n{items}"
	}
	out := strings.ReplaceAll(tpl, "{number}", fmt.Sprintf("%d", o.Number))
	return strings.ReplaceAll(out, "{items}", items)
}

type orderMessageItem struct {
	Name    string
	Qty     int
	Options []string
	Note    string
}

type orderMessage struct {
	Number int64
	Items  []orderMessageItem
	Notes  string
}

// merchantPhones رقما المتجر — **لأن القناتين لا تريدان الرقمَ نفسه.**
type merchantPhones struct {
	// SMS خطُّ المحلّ أوّلاً ثم خطُّ صاحبه: الرسالةُ النصّية تصل أيَّ خطّ.
	SMS string
	// WhatsApp الموثَّق أوّلاً، **ثم جوّالُ صاحبه، وخطُّ المحلّ آخراً**.
	//
	// **ورابطُ واتساب على خطٍّ أرضيّ رابطٌ ميّت**: يفتح التطبيقَ فيقول «الرقم
	// غير مسجّل»، ويظنّ الموظّفُ أنه حوّل الطلبَ وقد فتح صفحةَ خطأ.
	//
	// وخطُّ محلِّ المطعم في الرقة أرضيٌّ غالباً (`022…`) بينما صاحبُه على جوّال
	// (`09…`) — **فترتيبُ SMS مقلوبٌ هنا عمداً**: تلك تريد خطَّ المحلّ لأنه
	// يصله من في المطبخ، وهذه تريد من يحمل واتساب.
	WhatsApp string
}

// loadOrderMessage يجمع ما يدخل الرسالة — ولا شيء غيره.
func (s *Server) loadOrderMessage(ctx context.Context, orderID string) (*orderMessage, merchantPhones, error) {
	var msg orderMessage
	var ph merchantPhones
	if err := s.pg.QueryRow(ctx, `
		SELECT o.number, o.notes,
		       COALESCE(NULLIF(m.phone::text, ''),
		                NULLIF(ou.whatsapp_phone::text, ''),
		                NULLIF(ou.phone::text, ''), ''),
		       COALESCE(NULLIF(ou.whatsapp_phone::text, ''),
		                NULLIF(ou.phone::text, ''),
		                NULLIF(m.phone::text, ''), '')
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users ou ON ou.id = m.owner_user_id
		WHERE o.id = $1`, orderID).
		Scan(&msg.Number, &msg.Notes, &ph.SMS, &ph.WhatsApp); err != nil {
		return nil, merchantPhones{}, err
	}

	rows, err := s.pg.Query(ctx, `
		SELECT oi.name, oi.qty, oi.note,
		       COALESCE((SELECT array_agg(x->>'name' ORDER BY ord)
		                 FROM jsonb_array_elements(COALESCE(oi.options, '[]'::jsonb))
		                      WITH ORDINALITY AS t(x, ord)), '{}')
		FROM order_items oi WHERE oi.order_id = $1 ORDER BY oi.id`, orderID)
	if err != nil {
		return nil, merchantPhones{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var it orderMessageItem
		if err := rows.Scan(&it.Name, &it.Qty, &it.Note, &it.Options); err != nil {
			return nil, merchantPhones{}, err
		}
		msg.Items = append(msg.Items, it)
	}
	return &msg, ph, rows.Err()
}

// waLink رابطُ واتساب يفتح محادثةَ المتجر والنصُّ مكتوبٌ فيها.
//
// **يُبنى في الخادم لا في المتصفّح** — للسبب الذي بُني لأجله النصُّ نفسه:
// لو رُكّب في الواجهة لأمكن أن يختلف الرابطُ عمّا عُرض في المعاينة، فيرى
// الموظّفُ نصّاً ويفتح واتساب بغيره. **ومصدرٌ واحد لا مصدران متشابهان.**
//
// وتوحيدُ الرقم هنا كذلك: `wa.me` تريده دولياً بلا `+` ولا صفرٍ بادئ،
// و`identity.NormalizePhone` هي التي تعرف الصيغ السورية. **ولا تُعاد كتابتها
// في TypeScript** — فنسختان من قاعدةٍ واحدة تفترقان يوماً.
func waLink(phone, text string) string {
	e164, ok := identity.NormalizePhone(phone)
	if !ok {
		return ""
	}
	return "https://wa.me/" + strings.TrimPrefix(e164, "+") +
		"?text=" + url.QueryEscape(text)
}

// handleOrderMessagePreview نصُّ الرسالة كما سيصل المتجر — قبل الإرسال.
//
// **ما يُرسَل باسم المنصة يُقرأ قبل أن يُرسَل.**
func (s *Server) handleOrderMessagePreview(w http.ResponseWriter, r *http.Request) {
	msg, ph, err := s.loadOrderMessage(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	text := buildMerchantMessage(s.settings.GetString(r.Context(), "whatsapp.order_template"), msg)

	// **وكم سائقاً في الدوام الآن.**
	//
	// التحويلُ يُبلّغ المطعمَ فيبدأ الطبخ، **ثمّ ينزل الطلبُ إلى من يحمله.**
	// فإن لم يكن أحدٌ في الدوام **طُبخ طعامٌ لا حاملَ له** — ويبرد بينما تنتظر
	// العملياتُ سائقاً لا يأتي.
	//
	// **ولا يُمنع التحويل**: قد يفتح سائقٌ دوامَه بعد دقيقة، **والمنعُ يقرّر
	// عن المالك ما لا يعرفه.** يُقال له الرقمُ ويقرّر هو.
	//
	// قرارُ المالك (٢٠٢٦-٠٨-٠٣): «عندما لا يكون هناك سائق على الدوام يجب
	// تنبيه الإدارة **قبل** تحويل الطلب للمتجر».
	var onShift int
	if err := s.pg.QueryRow(r.Context(), `
		SELECT count(*) FROM users u
		WHERE u.deleted_at IS NULL AND u.status = 'active' AND u.on_shift
		  AND EXISTS (SELECT 1 FROM user_roles r
		              WHERE r.user_id = u.id AND r.role_code = 'driver')`).
		Scan(&onShift); err != nil {
		s.logger.Error("تعذّر عدّ السائقين في الدوام", "error", err)
		onShift = -1 // **مجهولٌ لا صفر** — وصفرٌ كاذبٌ يوقف العمل بلا سبب
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"text":             text,
		"phone":            ph.WhatsApp,
		"wa_link":          waLink(ph.WhatsApp, text),
		"drivers_on_shift": onShift,
		// أمُهيَّأةٌ بوّابةُ الرسائل؟ **الواجهةُ لا تعرض زرّاً لا يعمل.**
		// وزرٌّ يُضغط فيردّ «غير مضبوطة» يُعلّم الموظّفَ ألّا يثق بالأزرار.
		"sms_ready": s.textSender.Configured(),
	})
}

// handleSendOrderToMerchant يُبلّغ المتجرَ بطلبه — بقناتين لا واحدة.
//
// ## قناتان لأن إحداهما لا تكفي
//
//   - **`whatsapp`** — تحويلٌ يدويّ: يفتح الموظّفُ محادثةَ المتجر والنصُّ
//     مكتوبٌ فيها، ويضغط إرسال. **يعمل اليوم بلا بوّابةٍ ولا اشتراكٍ ولا
//     اتفاق**، ومن حساب المنصة الرسميّ في التطبيق الذي يعمل عليه المطعمُ
//     أصلاً. **والإنسانُ هو من يضغط — فلا حظرَ لإرسالٍ آليّ.**
//   - **`sms`** — إرسالٌ آليّ عبر بوّابة. لا يحتاج موظّفاً، ويحتاج مزوّداً.
//
// ## وما يُسجَّل يفرّق بينهما
//
// **«أُبلغ المتجر» وحدها لا تكفي حين يقول المطعمُ «لم يصلني».** فالسجلّ يقول
// بأيّ قناة: بوّابةٌ ردّت بنجاح، أم موظّفٌ فتح واتساب. والسؤالُ التالي يختلف.
//
// ## وحدَّ ما نعرفه لا أكثر
//
// في التحويل اليدويّ **نعلم أن الموظّف فتح المحادثة، ولا نعلم أنه ضغط إرسال**.
// فالوسمُ يعني «حُوِّل» لا «وصل» — واللفظُ في الشاشة يقول ذلك بلا تجميل.
func (s *Server) handleSendOrderToMerchant(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	req, err := decode[struct {
		Channel string `json:"channel"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Channel != "whatsapp" && req.Channel != "sms" {
		s.respondErr(w, errBadChannel)
		return
	}

	msg, ph, err := s.loadOrderMessage(r.Context(), orderID)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	phone := ph.WhatsApp
	if req.Channel == "sms" {
		phone = ph.SMS
	}
	if phone == "" {
		s.respondErr(w, errNoMerchantPhone)
		return
	}

	if req.Channel == "sms" {
		if !s.textSender.Configured() {
			s.respondErr(w, errSMSNotConfigured)
			return
		}
		text := buildMerchantMessage(s.settings.GetString(r.Context(), "whatsapp.order_template"), msg)
		if err := s.textSender.SendText(r.Context(), ph.SMS, text); err != nil {
			// **السببُ في السجلّ والرسالةُ العامّة للشاشة**: ردُّ المزوّد قد
			// يحمل مفتاحاً أو تفصيلَ حسابٍ لا يُعرض لموظّف.
			s.logger.Error("dispatch: تعذّر إرسال الطلب للمتجر",
				"order", orderID, "phone", phone, "error", err)
			s.respondErr(w, errSMSFailed)
			return
		}
	}

	// **يُسجَّل**: إبلاغٌ باسم المنصة إلى طرفٍ خارجها — ومن أبلغ وبأيّ قناة
	// سؤالان يُطرحان حين يقول المطعمُ «لم يصلني الطلب».
	s.audit(r, "ops.order_notify", "order", orderID, map[string]any{
		"phone": phone, "number": msg.Number, "channel": req.Channel,
	})
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE orders SET sent_to_merchant_at = now() WHERE id = $1`, orderID); err != nil {
		s.logger.Error("dispatch: تعذّر وسم الإرسال", "order", orderID, "error", err)
	}
	// **الآن — لا قبل الآن — يُستدعى السائق.**
	//
	// المطعمُ علم بالطلب، فبدأ عدّادُ التحضير عنده فعلاً. واستدعاءُ السائق قبل
	// هذه اللحظة يرسله إلى بابٍ لم يُطبخ خلفه شيء.
	//
	// وفشلُ الإنزال لا يُبطل إرسالاً وقع: الرسالةُ وصلت المتجرَ ولا تُستردّ.
	// **يبقى الطلب حيث هو وتراه العملياتُ بزرّ «طلب سائق» كما كان.**
	//
	// # ولا يُشترط بمفتاح
	//
	// كان مشروطاً بـ`orders.auto_dispatch`. **والمفتاحُ ليس في الفهرس** فيُقرأ
	// «لا» — **فيبقى الطلبُ في «تم القبول» بلا بابٍ يوصله إلى سائق.**
	//
	// **والتحويلُ هو الفعلُ لا خطوةٌ قبله**: قرارُ المالك (٢٠٢٦-٠٨-٠٤) أنّ
	// المنصةَ تُحوّل إلى المتجر أو إلى متجرٍ آخر، **ولا شيءَ بعدها اسمُه «طلب
	// سائق» تنتظره ضغطةٌ ثانية.**
	//
	// **وشرطٌ على مفتاحٍ لا وجودَ له شرطٌ لا يتحقّق أبداً** — ووعدٌ في تعليقٍ
	// لا يفي به الكود.
	if err := s.orders.AutoDispatch(r.Context(), userIDFrom(r), orderID); err != nil {
		s.logger.Warn("الإنزال بعد الإبلاغ تعثّر — ينتظر إسناداً يدوياً",
			"order", orderID, "error", err)
	}

	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"sent": true, "phone": phone, "channel": req.Channel,
	})
}
