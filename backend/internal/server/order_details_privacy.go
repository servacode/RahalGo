package server

// ══════════════════════════════════════════════════════════════════════
// **تفاصيلُ زبون الطلب لثلاثةٍ وحدَهم** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (سجلُّ الطلبات، البند ٣)
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **قائمةُ الطلبات تردّ الطلبَ كاملاً** — هاتفَ الزبون وإحداثيّاتِ بيته وصورةَ
// إثبات التسليم — **لكلّ من يملك `orders.read`**: الماليّةُ ومراقبُ المنصّة
// والثقةُ والأمان. **ومن يسوّي حساباً لا يتّصل بأحدٍ ولا يزور بيتاً.**
//
// # وما صار
//
// **من لا يملك `orders.customer_details.read`** يأخذ:
//
//   - **الهاتفَ مخفيّاً جزئيّاً** (`customer_phone_masked`: `+963 9•• ••• 123`)
//     — يكفي ليُطابَق مع شكوى، ولا يكفي ليُتّصل به.
//   - **ولا إحداثيّات ولا صورة** — **تُحذف المفاتيحُ لا تُفرَّغ**: صفرٌ في خطّ
//     العرض يكذب (نقطةٌ في المحيط)، والغائبُ يقول الحقيقة.
//
// **وفي الخادم لا في الشاشة** — الردُّ الخامُ هو الحقيقة (`XG-42`).

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// orderDetailFields **ما يُحجَب عمّن لا يملك القدرة** — بوسم `json`.
var orderDetailFields = []string{"customer_phone", "lat", "lng", "proof_url"}

// MaskPhone **هاتفٌ مخفيٌّ جزئيّاً** — رمزُ البلد وأوّلُ رقمٍ وآخرُ ثلاثة.
//
//	+963912345123 ⇒ +963 9•• ••• 123
//
// **وما لا يُعرف شكلُه يُخفى إلّا آخرَ ثلاثة** — والفارغُ فارغ.
func MaskPhone(p string) string {
	var d strings.Builder
	for _, r := range p {
		if r >= '0' && r <= '9' {
			d.WriteRune(r)
		}
	}
	digits := d.String()
	if digits == "" {
		return ""
	}
	if strings.HasPrefix(digits, "963") && len(digits) >= 10 {
		nat := digits[3:]
		return "+963 " + nat[:1] + "•• ••• " + nat[len(nat)-3:]
	}
	if len(digits) <= 3 {
		return strings.Repeat("•", len(digits))
	}
	return strings.Repeat("•", len(digits)-3) + digits[len(digits)-3:]
}

// canSeeOrderDetails **أيملك الفاعلُ تفاصيلَ زبون الطلب؟**
func (s *Server) canSeeOrderDetails(r *http.Request) bool {
	return s.hasCapability(r, authz.OrdersCustomerDetailsRead)
}

// maskOrderDetails **يملأ الهاتفَ المخفيّ** لصفحةٍ من الطلبات — ويقول أيُحجَب الباقي.
func (s *Server) maskOrderDetails(r *http.Request, list []orders.Order) bool {
	if s.canSeeOrderDetails(r) {
		return false
	}
	for i := range list {
		list[i].CustomerPhoneMasked = MaskPhone(list[i].CustomerPhone)
		list[i].CustomerPhone = ""
	}
	return true
}

// writeOrdersJSON **يكتب ردَّ الطلبات** — محذوفاً منه ما لا يجوز لقارئه.
//
// **ويسقط مغلقاً**: تعذّرَ التشكيلُ فلا يُكتب الخامُ (`order_view.go`).
func (s *Server) writeOrdersJSON(w http.ResponseWriter, strip bool, v any) {
	if !strip {
		httpx.JSON(w, http.StatusOK, v)
		return
	}
	raw, err := json.Marshal(v)
	if err != nil {
		s.respondErr(w, errPayloadUnsafe)
		return
	}
	var body any
	if err := json.Unmarshal(raw, &body); err != nil {
		s.respondErr(w, errPayloadUnsafe)
		return
	}
	stripFields(body, orderDetailFields)
	httpx.JSON(w, http.StatusOK, body)
}
