package server

// ══════════════════════════════════════════════════════════════════════
// **بوّابةُ الطلب من عدّة متاجر — من يعرف أدنى نسخةٍ نافذة** (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١١-٠٥: «١ موافق» — طلبٌ واحدٌ بعدّة محطّات.)
//
// **المحرّكُ يسأل والخادمُ يجيب** (`orders.MultiStopGate`): **أدنى نسخةٍ
// للسائق يُسمَح لها** (`effectiveMinVersion` — الحدُّ اليدويُّ أو رقمُ
// الملفّ المرفوع مع الفرض التلقائيّ) **≥ رقمِ النسخة التي تعرف المحطّات**
// (`orders.multistop_driver_build`).
//
//	المفتاحُ صفر                  ⇒ مغلقة — النسخةُ لم تصدر بعد (والافتراضُ ٢٣: النسخةُ التالية)
//	أدنى نسخةٍ دون الرقم          ⇒ مغلقة — سائقٌ قديمٌ قد يأخذ الطلب
//	أدنى نسخةٍ عند الرقم أو فوقه  ⇒ مفتوحة — **وكلُّ أقدمَ منها يُردّ ٤٢٦**
//	                                 قبل أن يرى طلباً (`min_version.go`)
//
// **ورجوعٌ يُنزل أدنى نسخةٍ يُغلقها في اللحظة** — **أيّاً كان ترتيبُ ما
// بُدّل من إعدادات.**

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// multiStopBuildKey **رقمُ نسخة السائق التي تعرف محطّاتِ الطلب** — صفرٌ: لم تصدر بعد.
const multiStopBuildKey = "orders.multistop_driver_build"

// errMultiStopNotForced **سقفُ المصادر فوق الواحد ونسخةُ السائق لم تُفرض.**
//
// **وكان يُحفظ فلا يقع شيء** — **فيرفع المالكُ السقفَ ويبقى في السلّة
// «متجرٌ واحد» ولا يعرف لماذا.** **فالردُّ يقول السبب.**
var errMultiStopNotForced = httpx.NewError(http.StatusConflict,
	"multistop_driver_not_forced", "errors.multistop_driver_not_forced")

// multiStopReady **أيُفتح الطلبُ من أكثرَ من متجرٍ الآن؟** — يُحقن في المحرّك.
//
// **ويقرأ بالمنفّذ المُمرَّر** (`XG-46`) — **وبلا قراءةِ ملفٍّ قديم**
// (`minVersionOn` بلا `backfill`): **الإغلاقُ عند الشكّ لا الفتح.**
func (s *Server) multiStopReady(ctx context.Context, q dbtx.Querier) bool {
	if s.settings == nil || q == nil {
		return false
	}
	build := s.settings.On(q).GetInt(ctx, multiStopBuildKey)
	if build <= 0 {
		return false
	}
	return s.minVersionOn(ctx, q, "driver", false) >= build
}

// checkMultiStopSources **سقفُ المصادر فوق الواحد لا يُحفظ والبوّابةُ مغلقة.**
//
// **والرفعُ إلى الواحد يُحفظ دائماً** — **ومن أراد أن يُغلق لا يُمنع.**
// **و`norm` القيمةُ بعد `Validate`.**
func (s *Server) checkMultiStopSources(ctx context.Context, key string, norm any) error {
	if key != "orders.max_sources" {
		return nil
	}
	n, ok := norm.(int64)
	if !ok || n <= 1 {
		return nil
	}
	if s.pg != nil && s.multiStopReady(ctx, s.pg) {
		return nil
	}
	return errMultiStopNotForced
}
