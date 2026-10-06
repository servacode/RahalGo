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

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
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

// codeRoute **أين يصل الكود** — يُحسب داخلَ معاملة الاستلام.
type codeRoute struct {
	code   string
	number int64
	app    string // معرّفُ الزبون إن صحّ الإشعار
	phone  string // رقمُ الواتساب إن صحّ
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
	var custPhone, recipPhone string
	if err := tx.QueryRow(ctx, `
		SELECT o.number, COALESCE(cu.phone::text, ''), COALESCE(o.recipient_phone, '')
		FROM orders o LEFT JOIN users cu ON cu.id = o.customer_id
		WHERE o.id = $1`, orderID).Scan(&number, &custPhone, &recipPhone); err != nil {
		return nil, err
	}

	rt := &codeRoute{number: number}
	waReady := allowWA && s.whatsapp != nil && s.whatsapp.Ready()
	if kind == KindMerchantDelivery {
		// **المستلِمُ ليس على التطبيق** — رقمُه وحدَه.
		if waReady && recipPhone != "" {
			rt.phone = recipPhone
		}
	} else {
		if allowApp && customerID != "" && s.notify != nil {
			rt.app = customerID
		}
		if waReady && custPhone != "" {
			rt.phone = custPhone
		}
	}
	if rt.app == "" && rt.phone == "" {
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
		if _, err := s.db.Exec(bg,
			`UPDATE orders SET delivery_code = NULL WHERE id = $1 AND delivery_code = $2`,
			orderID, rt.code); err != nil && s.logger != nil {
			s.logger.Warn("محوُ كود التسليم بعد تعثّره سقط", "order", orderID, "error", err)
		}
	}()
}
