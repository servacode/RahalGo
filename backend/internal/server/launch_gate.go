package server

import (
	"context"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ══════════════════════════════════════════════════════════════════════
// **بوّابةُ الإطلاق — ما هو مفتوحٌ للناس الآن**
// ══════════════════════════════════════════════════════════════════════
//
// # المسألة
//
// **والمنصّةُ تُنزَّل قبل أن تُفتح**: يُثبِّت الناسُ التطبيقَ ويُسجّلون
// ويتصفّحون، **والطلبُ لا يُستقبَل** حتّى تمتلئ السوقُ بالمتاجر.
//
// # ولماذا موضعٌ واحد
//
// **وزرٌّ مخفيٌّ في أندرويد ليس منعاً** — **ونداءٌ مباشرٌ يتجاوزه.**
// **والمنعُ هنا في المحرّك**، **وموضعٌ واحدٌ يقرؤه الجميع** فلا يُنسى
// عند بابٍ.
//
// # والإخفاقُ مغلق
//
// **وقراءةٌ تعذّرت تسقط على افتراض الفهرس — وهو `false`** — **فعطبٌ
// في القاعدة لا يفتح عملاً مُغلقاً.** **والمنصّةُ العاملةُ اليومَ
// تُبذَر مفتوحةً بالهجرة، فلا يتبدّل عندها شيء.**
//
// # وهي غيرُ أدنى نسخة
//
// **و`app.min_version.*` تقول أيُّ نسخةِ تطبيقٍ يُسمَح لها**، **وهذه
// تقول أيُّ عملٍ مفتوحٌ أصلاً.** **ومن جمعهما أوقف نسخةً ليُغلق باباً.**

// launchKey مفاتيحُ الإطلاق — **ولا نصَّ حرفيٌّ في معالِج.**
const (
	launchCustomerSignup       = "launch.customer_signup"
	launchCustomerBrowse       = "launch.customer_browse"
	launchCustomerOrders       = "launch.customer_orders"
	launchCustomerCustomOrders = "launch.customer_custom_orders"
	launchDriverWork           = "launch.driver_work"
	launchMerchantOrders       = "launch.merchant_orders"
	launchRepAcquisition       = "launch.rep_acquisition"
	launchNotice               = "launch.notice"
)

// ErrLaunchClosed **بابٌ لم يُفتح بعد** — لا عطبٌ ولا منعُ صلاحيّة.
//
// **ورمزٌ خاصٌّ به**: **٤٠٣ «ممنوع» يُقرأ «لستَ أهلاً»**، **وهذا
// «ليس الآن».** **والفرقُ يقرؤه الزبونُ في رسالةٍ ويقرؤه العميلُ في
// رمز.**
//
// **و٥٠٣ لا ٤٠٣**: **الخدمةُ غيرُ متاحةٍ مؤقّتاً** — وهو ما يعنيه
// الحال. **ولا يُقرأ عطبَ خادمٍ**: الجسمُ يقول السببَ صراحةً.
var ErrLaunchClosed = httpx.NewError(http.StatusServiceUnavailable,
	"launch_closed", "errors.launch_closed")

// launchOpen **أهذا البابُ مفتوح؟** — والافتراضُ مغلق.
func (s *Server) launchOpen(ctx context.Context, key string) bool {
	return s.settings.GetBool(ctx, key)
}

// requireLaunch **يردّ النداءَ إن كان البابُ مغلقاً.**
//
// **ويُنادى قبل أيّ عملٍ** — **وقبل قراءةِ الجسم وقبل أيّ كتابة**،
// **فلا يُستهلك مفتاحُ تفرّدٍ ولا يُقيَّد شيءٌ لبابٍ مغلق.**
func (s *Server) requireLaunch(w http.ResponseWriter, r *http.Request, key string) bool {
	if s.launchOpen(r.Context(), key) {
		return true
	}
	s.respondErr(w, s.launchError(r.Context()))
	return false
}

// launchError **الخطأُ وفيه نصُّ المالك إن ضبطه.**
//
// **ونصٌّ مكتوبٌ في الشيفرة لا يُصحَّح إلّا بنشر** — **ويومَ يُفتح
// البابُ يبقى معروضاً.** **فيُقرأ من الإعدادات، وفارغُه يقع على
// رسالة المعجم.**
func (s *Server) launchError(ctx context.Context) error {
	notice := strings.TrimSpace(s.settings.GetString(ctx, launchNotice))
	if notice == "" {
		return ErrLaunchClosed
	}
	// **والنصُّ في `Details` لا في `MessageKey`** — **فالمفتاحُ عقدٌ
	// تقرؤه الواجهةُ، والنصُّ حالٌ يضبطها المالك.** **ومن أبدل المفتاحَ
	// بنصٍّ كسر ترجمةَ كلّ عميلٍ لا يعرفه.**
	e := *ErrLaunchClosed
	e.Details = map[string]any{"notice": notice}
	return &e
}

// launchGate **وسيطٌ يحرس مجموعةَ أبوابٍ بمفتاحٍ واحد.**
//
// **وثلاثةُ أبوابٍ للتصفّح** — الرئيسيّةُ والعروضُ والصنف — **وفحصٌ
// مكتوبٌ في كلٍّ يُنسى عند رابعٍ يُضاف غداً.** **فالوسيطُ يُلَفّ على
// المجموعة.**
func (s *Server) launchGate(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !s.requireLaunch(w, r, key) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
