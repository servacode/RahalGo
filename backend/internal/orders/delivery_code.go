package orders

// ══════════════════════════════════════════════════════════════════════
// **كودُ التسليم** (قرارُ المالك ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// أربعةُ أرقامٍ تُولَّد لحظةَ الاستلام (`picked_up`) وتصل الزبونَ بإشعار
// التطبيق أو بالواتساب أو بهما (`delivery.code_channel`)، **ولا يُغلق السائقُ
// الطلبَ «سُلّم» حتّى يقولها له الزبون** (`server/delivery_code.go`).
//
// **وكودٌ لا يصل الزبونَ لا يُطلب**: لا قناةَ ممكنة ⇒ لا يُكتب أصلاً، وواتسابٌ
// وحدَه تعثّر ⇒ يُمحى بعد المحاولة. **فلا يقف تسليمٌ على رسالةٍ لم تصل** —
// ويبقى إثباتُ الصورة كما كان.
//
// **وفي «لدي توصيلة»** المستلِمُ ليس على التطبيق — فالواتساب وحدَه إلى رقمه.
//
// **وفي «إحضار» الكودُ للمتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦) — هو من يستلم الغرض: **يراه في
// شاشة متابعة التوصيلة ويصله إشعار**، ويعطيه للسائق لما يوصل. **والمفتاحُ نفسُه يحكمه**
// (`delivery.code_required`) — ولا واتسابَ ولا رسالةَ للمستلِم فيه.

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// TextSender **مُرسِلُ الواتساب** — يردّ «وصلت» أو «لم تصل».
//
// **وواجهةٌ ضيّقة**: البوتُ في الخادم، **والمحرّكُ يسأل سؤالاً واحداً.**
type TextSender interface {
	Ready() bool
	SendText(ctx context.Context, phone, text string) bool
}

// SetWhatsApp يحقن مُرسِلَ الواتساب — **وبلاه لا كودَ بالواتساب.**
func (s *Service) SetWhatsApp(w TextSender) { s.whatsapp = w }

// DeliveryCodeMaxAttempts **خمسُ محاولاتٍ خاطئةٍ تُقفل الطلب** حتّى تتدخّل العمليّات.
const DeliveryCodeMaxAttempts = 5

// newDeliveryCode **أربعةُ أرقامٍ من مصدرٍ عشوائيٍّ آمن** — `0000`..`9999`.
func newDeliveryCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%04d", n.Int64()), nil
}

// DeliveryCodeText **نصُّ الكود للزبون** — واحدٌ في الإشعار والواتساب.
func DeliveryCodeText(number int64, code string) string {
	return fmt.Sprintf("كود استلام طلبك #%d هو %s — أعطه للسائق عند الاستلام فقط.", number, code)
}

// FetchCodeText **نصُّ الكود للمتجر في «إحضار»** — يعطيه للسائق لما يوصله الغرض.
func FetchCodeText(number int64, code string) string {
	return fmt.Sprintf("كود استلام التوصيلة #%d هو %s — أعطه للسائق لما يوصلك الغرض فقط.", number, code)
}

// codeRoute **أين يصل الكود** — يُحسب داخلَ معاملة الاستلام.
type codeRoute struct {
	code   string
	number int64
	app    string // معرّفُ الزبون إن صحّ الإشعار
	phone  string // رقمُ الواتساب إن صحّ
	// sms **رقمُ المستلم للرسالة العاديّة** — إن لم يصله الواتساب أُرسل الكودُ إلى
	// المنصّة لتبعثه بيدها (فكرةُ المالك ٢٠٢٦-١٠-١٢).
	sms string
	// merchant **صاحبُ المتجر في «إحضار»** — الكودُ له في تطبيقه، لا واتسابَ ولا رسالة.
	merchant string
}

// CodeBySMSText **ما يصل المنصّةَ حين لا يصل الكودُ صاحبَه** — رقمُه والكودُ وما يُفعل.
func CodeBySMSText(number int64, phone, code string) string {
	return fmt.Sprintf("كود استلام لم يصل — الطلب #%d\nالرقم %s ما عليه واتساب.\nابعتله رسالة SMS عادية بكود الاستلام: %s",
		number, phone, code)
}

// issueDeliveryCode **يولّد الكودَ ويكتبه إن كان له طريقٌ إلى الزبون** — داخلَ المعاملة.
//
// **ومطفأٌ أو بلا طريق ⇒ يُمحى أيُّ كودٍ قديم** (استلامٌ ثانٍ بعد طارئ) — فلا يُطلب
// من السائق كودٌ لم يصل أحداً.
func (s *Service) issueDeliveryCode(ctx context.Context, tx pgx.Tx, orderID, kind, customerID string) (*codeRoute, error) {
	clear := func() (*codeRoute, error) {
		_, err := tx.Exec(ctx,
			`UPDATE orders SET delivery_code = NULL, delivery_code_attempts = 0 WHERE id = $1`, orderID)
		return nil, err
	}
	if !s.settings.GetBool(ctx, "delivery.code_required") {
		return clear()
	}
	channel := s.settings.GetString(ctx, "delivery.code_channel")
	allowApp := channel != "whatsapp"
	allowWA := channel != "app"

	var number int64
	var custPhone, recipPhone, mode, owner string
	if err := tx.QueryRow(ctx, `
		SELECT o.number, COALESCE(cu.phone::text, ''), COALESCE(o.recipient_phone, ''),
		       o.delivery_mode, COALESCE(m.owner_user_id::text, '')
		FROM orders o LEFT JOIN users cu ON cu.id = o.customer_id
		LEFT JOIN merchants m ON m.id = o.merchant_id
		WHERE o.id = $1`, orderID).Scan(&number, &custPhone, &recipPhone, &mode, &owner); err != nil {
		return nil, err
	}

	rt := &codeRoute{number: number}
	// **«إحضار»: الكودُ للمتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦) — يقرؤه في تطبيقه، فلا قناةَ تُسأل.
	if kind == KindMerchantDelivery && mode == DeliveryModeFetch {
		if owner == "" {
			return clear()
		}
		rt.merchant = owner
		code, err := newDeliveryCode()
		if err != nil {
			return nil, err
		}
		rt.code = code
		if _, err := tx.Exec(ctx,
			`UPDATE orders SET delivery_code = $2, delivery_code_attempts = 0 WHERE id = $1`,
			orderID, code); err != nil {
			return nil, err
		}
		return rt, nil
	}
	waReady := allowWA && s.whatsapp != nil && s.whatsapp.Ready()
	if kind == KindMerchantDelivery {
		// **المستلِمُ ليس على التطبيق** — رقمُه وحدَه.
		rt.sms = recipPhone
		if waReady && recipPhone != "" {
			rt.phone = recipPhone
		}
	} else {
		rt.sms = custPhone
		if allowApp && customerID != "" && s.notify != nil {
			rt.app = customerID
		}
		if waReady && custPhone != "" {
			rt.phone = custPhone
		}
	}
	// **ولا واتسابَ ولا تطبيق ⇒ المنصّةُ تبعثه رسالةً عاديّة** إن كان مَن يُنبَّه —
	// وإلّا يُمحى كما كان، **فلا يُطلب من السائق كودٌ لم يصل أحداً.**
	if rt.app == "" && rt.phone == "" && (rt.sms == "" || s.notify == nil) {
		return clear()
	}
	code, err := newDeliveryCode()
	if err != nil {
		return nil, err
	}
	rt.code = code
	if _, err := tx.Exec(ctx,
		`UPDATE orders SET delivery_code = $2, delivery_code_attempts = 0 WHERE id = $1`,
		orderID, code); err != nil {
		return nil, err
	}
	return rt, nil
}

// sendDeliveryCode **يرسل الكودَ بعد التثبيت** — ولا يُسقط استلاماً وقع.
//
// **والإشعارُ في لحظته، والواتسابُ في الخلفية** (البوتُ يتمهّل بين الرسائل).
// **وواتسابٌ وحدَه تعثّر ⇒ يُمحى الكود** — بشرط أنّه هو نفسُه لم يتبدّل.
func (s *Service) sendDeliveryCode(ctx context.Context, orderID string, rt *codeRoute) {
	if rt == nil || rt.code == "" {
		return
	}
	// **والمتجرُ في «إحضار» يراه في شاشته** — والإشعارُ تذكيرٌ لا قناةٌ وحيدة، فلا يُمحى.
	if rt.merchant != "" {
		if s.notify != nil {
			s.notify.Notify(ctx, notifications.Input{
				UserID: rt.merchant, Kind: notifications.KindOrder,
				Title: "كود استلام التوصيلة", Body: FetchCodeText(rt.number, rt.code),
				Entity: "order", EntityID: orderID, Href: "/merchant/deliveries",
				Apps: []string{notifications.AppMerchant},
			})
		}
		return
	}
	text := DeliveryCodeText(rt.number, rt.code)
	if rt.app != "" && s.notify != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: rt.app, Kind: notifications.KindOrder,
			Title: "كود استلام طلبك", Body: text,
			Entity: "order", EntityID: orderID, Href: "/portal/orders",
			Apps: []string{notifications.AppCustomer},
		})
	}
	if rt.phone == "" || s.whatsapp == nil {
		if rt.app == "" {
			s.alertCodeBySMS(ctx, orderID, rt)
		}
		return
	}
	appSent := rt.app != ""
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if s.whatsapp.SendText(bg, rt.phone, text) {
			return
		}
		if s.logger != nil {
			s.logger.Warn("كودُ التسليم لم يصل بالواتساب", "order", orderID)
		}
		if appSent {
			return
		}
		// **ولا يُمحى والمنصّةُ تبعثه بيدها** (فكرةُ المالك ٢٠٢٦-١٠-١٢: «تصل رسالة ع رقم
		// المنصة هذا رقم الزبون وماعنده واتس اب نحنا نرسله رسالة عادية ع رقمه بكود الاستلام»).
		// **كان يُمحى فيمرّ التسليمُ بلا تأكيد** — والمستلمُ بلا واتسابٍ كثيرٌ في الرقّة.
		if s.alertCodeBySMS(bg, orderID, rt) {
			return
		}
		if _, err := s.db.Exec(bg,
			`UPDATE orders SET delivery_code = NULL WHERE id = $1 AND delivery_code = $2`,
			orderID, rt.code); err != nil && s.logger != nil {
			s.logger.Warn("محوُ كود التسليم بعد تعثّره سقط", "order", orderID, "error", err)
		}
	}()
}

// alertCodeBySMS **يُبلغ المنصّةَ أن تبعث الكودَ رسالةً عاديّة** — في لوحتها، وعلى رقم
// واتسابها إن ضُبط. **ويردّ أوصل أحداً** — وإلّا محا المُرسِلُ الكود.
func (s *Service) alertCodeBySMS(ctx context.Context, orderID string, rt *codeRoute) bool {
	if rt == nil || rt.sms == "" || rt.code == "" {
		return false
	}
	text := CodeBySMSText(rt.number, rt.sms, rt.code)
	sent := false
	if s.notify != nil {
		s.notify.NotifyOps(ctx, notifications.Input{
			Kind: notifications.KindOrder, Title: "ابعت كود الاستلام برسالة SMS",
			Body: text, Entity: "order", EntityID: orderID, Href: "/dashboard/orders",
		})
		sent = true
	}
	if s.whatsapp != nil && s.whatsapp.Ready() {
		to := strings.TrimSpace(s.settings.GetString(ctx, "platform.whatsapp"))
		if to == "" {
			to = strings.TrimSpace(s.settings.GetString(ctx, "platform.support_phone"))
		}
		if to != "" && s.whatsapp.SendText(ctx, to, text) {
			sent = true
		}
	}
	return sent
}
