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
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	errNoMerchantPhone  = httpx.NewError(http.StatusConflict, "no_merchant_phone", "errors.no_merchant_phone")
	errSMSNotConfigured = httpx.NewError(http.StatusServiceUnavailable, "sms_not_configured", "errors.sms_not_configured")
	errSMSFailed        = httpx.NewError(http.StatusBadGateway, "sms_failed", "errors.sms_failed")
)

// buildMerchantMessage نصُّ الرسالة — مصدرٌ واحد يقرؤه الإرسالُ والمعاينة.
//
// **ولو بُني في الواجهة لاختلف عمّا يُرسَل**: يرى الموظّف نصّاً ويصل المتجرَ
// غيرُه، ولا يكتشف ذلك أحدٌ حتى يشتكي متجر.
func buildMerchantMessage(o *orderMessage) string {
	var b strings.Builder
	fmt.Fprintf(&b, "طلب جديد #%d\n", o.Number)
	b.WriteString("——————————————\n")
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
		b.WriteString("——————————————\n")
		fmt.Fprintf(&b, "ملاحظة الزبون: %s\n", o.Notes)
	}
	return strings.TrimRight(b.String(), "\n")
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

// loadOrderMessage يجمع ما يدخل الرسالة — ولا شيء غيره.
func (s *Server) loadOrderMessage(ctx context.Context, orderID string) (*orderMessage, string, error) {
	var msg orderMessage
	var phone string
	// هاتفُ المتجر أوّلاً ثم هاتفُ صاحبه: المتجر قد يكون له خطٌّ للمحلّ، وقد
	// لا يكون — فيُبلَّغ صاحبُه على خطّه.
	if err := s.pg.QueryRow(ctx, `
		SELECT o.number, o.notes,
		       COALESCE(NULLIF(m.phone::text, ''),
		                NULLIF(ou.whatsapp_phone::text, ''),
		                NULLIF(ou.phone::text, ''), '')
		FROM orders o
		JOIN merchants m ON m.id = o.merchant_id
		LEFT JOIN users ou ON ou.id = m.owner_user_id
		WHERE o.id = $1`, orderID).Scan(&msg.Number, &msg.Notes, &phone); err != nil {
		return nil, "", err
	}

	rows, err := s.pg.Query(ctx, `
		SELECT oi.name, oi.qty, oi.note,
		       COALESCE((SELECT array_agg(x->>'name' ORDER BY ord)
		                 FROM jsonb_array_elements(COALESCE(oi.options, '[]'::jsonb))
		                      WITH ORDINALITY AS t(x, ord)), '{}')
		FROM order_items oi WHERE oi.order_id = $1 ORDER BY oi.id`, orderID)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var it orderMessageItem
		if err := rows.Scan(&it.Name, &it.Qty, &it.Note, &it.Options); err != nil {
			return nil, "", err
		}
		msg.Items = append(msg.Items, it)
	}
	return &msg, phone, rows.Err()
}

// handleOrderMessagePreview نصُّ الرسالة كما سيصل المتجر — قبل الإرسال.
//
// **ما يُرسَل باسم المنصة يُقرأ قبل أن يُرسَل.**
func (s *Server) handleOrderMessagePreview(w http.ResponseWriter, r *http.Request) {
	msg, phone, err := s.loadOrderMessage(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"text":  buildMerchantMessage(msg),
		"phone": phone,
	})
}

// handleSendOrderToMerchant يُرسل الطلب إلى المتجر برسالةٍ نصّية.
func (s *Server) handleSendOrderToMerchant(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	msg, phone, err := s.loadOrderMessage(r.Context(), orderID)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if phone == "" {
		s.respondErr(w, errNoMerchantPhone)
		return
	}

	if !s.textSender.Configured() {
		s.respondErr(w, errSMSNotConfigured)
		return
	}
	text := buildMerchantMessage(msg)
	if err := s.textSender.SendText(r.Context(), phone, text); err != nil {
		// **السببُ في السجلّ والرسالةُ العامّة للشاشة**: ردُّ المزوّد قد يحمل
		// مفتاحاً أو تفصيلَ حسابٍ لا يُعرض لموظّف.
		s.logger.Error("dispatch: تعذّر إرسال الطلب للمتجر",
			"order", orderID, "phone", phone, "error", err)
		s.respondErr(w, errSMSFailed)
		return
	}

	// **يُسجَّل**: رسالةٌ باسم المنصة إلى طرفٍ خارجها، ومن أرسلها سؤالٌ يُطرح.
	s.audit(r, "ops.order_notify", "order", orderID, map[string]any{
		"phone": phone, "number": msg.Number,
	})
	if _, err := s.pg.Exec(r.Context(),
		`UPDATE orders SET sent_to_merchant_at = now() WHERE id = $1`, orderID); err != nil {
		s.logger.Error("dispatch: تعذّر وسم الإرسال", "order", orderID, "error", err)
	}
	s.touch("order", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"sent": true, "phone": phone})
}
