package server

// تسعيرةُ السلّة قبل الطلب — **رقمٌ يتغيّر أمام العين.**
//
// # المسألة
//
// رسمُ التوصيل صار يتغيّر بعدد المصادر وبُعدها، **ولم يعد رقماً ثابتاً للمنطقة**.
// وكانت السلّةُ تقرأ رسمَ المنطقة وحدَه — **فيرى الزبونُ رقماً ويُحاسَب بآخر.**
//
// **ورقمٌ يتغيّر أمام العين يُقبل، ورقمٌ يظهر عند الدفع يُراجَع**: من أضاف صنفاً
// فارتفع الرسمُ أمامه يفهم أن السببَ إضافتُه، **ومن رآه عند الدفع يظنّ أنه
// خُدع.**
//
// # ولماذا الخادمُ يحسبها لا السلّة
//
// السلّةُ **لا تعرف المصادر** — أخفيناها عنها عمداً، ولا تعرف المسافةَ بينها.
// **والحسبةُ حيث المعرفة**، وهي عند الخادم وحدَه.
//
// **ولا تُنشئ شيئاً**: قراءةٌ محضة، فمن استعرض عشرَ مرّاتٍ لم يترك أثراً.

import (
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// handleQuote تسعيرةُ سلّةٍ في موقعٍ بعينه.
func (s *Server) handleQuote(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Items []orders.ItemInput `json:"items"`
		Lat   float64            `json:"lat"`
		Lng   float64            `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وسلّةٌ فارغةٌ ليست تسعيرة** — (تدقيقُ الإطلاق ٢٠٢٦-٠٨-١٩ —
	// BUG-006: رُدَّ ٢٠٠ بمجموعٍ صفر).
	//
	// **وصفرٌ يُقرأ سعراً**: تُعرض «الإجماليّ ٠ ل.س» على شاشةٍ لا صنفَ
	// فيها، **فيُظنّ أنّ الطلبَ مجّانيّ** — والصوابُ أن يُقال إنّ
	// السؤالَ نفسَه لا يصحّ.
	if len(req.Items) == 0 {
		s.respondErr(w, errValidation)
		return
	}

	q, err := s.orders.Quote(r.Context(), req.Items, req.Lat, req.Lng)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, q)
}

// handlePromoPreview أثرُ كود الخصم قبل الطلب.
//
// **والزبونُ يُسأل عنه لا الزائر**: القواعدُ فيها «مرّةً لكلّ مستخدم»
// و«لأوّل طلبٍ فقط» — **ومعاينةٌ بلا صاحبٍ تَعِد بما لا يقع.**
//
// **ولا تُنشئ شيئاً**: معاملةٌ تُلغى، فمن جرّب عشرةَ أكوادٍ لم يستهلك واحداً.
func (s *Server) handlePromoPreview(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Code        string `json:"code"`
		Subtotal    int64  `json:"subtotal"`
		DeliveryFee int64  `json:"delivery_fee"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Code == "" || req.Subtotal <= 0 {
		s.respondErr(w, errValidation)
		return
	}
	out, err := s.orders.PreviewPromo(r.Context(), req.Code, userIDFrom(r), req.Subtotal, req.DeliveryFee)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, out)
}
