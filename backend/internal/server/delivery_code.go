package server

// ══════════════════════════════════════════════════════════════════════
// **كودُ التسليم — حارسُه عند «سُلّم»** (قرارُ المالك ٢٠٢٦-١٠-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// الكودُ يُولَّد عند الاستلام ويصل الزبونَ (`orders/delivery_code.go`). **وهنا
// يُطلب**: طلبٌ عليه كودٌ لا يُغلق «سُلّم» إلّا بكوده. **والخطأُ يُعدّ** — خمسٌ
// تُقفله، **وفكُّه بيد العمليّات** (`proof-exception` يمحو الكود).
//
// **ولا يرى السائقُ الكود أبداً** — يرى `delivery_code_required` وحدَه، **ويقرؤه
// موظّفُ العمليّات** (`orders.intervene`) بعد أن يتحقّق من الزبون بالهاتف.

import (
	"context"
	"crypto/subtle"
	"net/http"
	"strconv"
	"strings"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// errDeliveryCodeLocked **أُقفل الطلبُ بعد خمسِ محاولاتٍ خاطئة.**
var errDeliveryCodeLocked = httpx.NewError(http.StatusLocked,
	"delivery_code_locked", "errors.delivery_code_locked")

// errDeliveryCodeWrong **كودٌ غائبٌ أو خاطئ** — ومعه كم بقي.
func errDeliveryCodeWrong(left int) error {
	return &httpx.AppError{
		Status: http.StatusUnprocessableEntity, Code: "delivery_code_wrong",
		MessageKey: "errors.delivery_code_wrong",
		Details:    map[string]any{"attempts_left": strconv.Itoa(left)},
	}
}

// requireDeliveryCode **يمنع «سُلّم» بلا كود الطلب** — إن كان عليه كود.
func (s *Server) requireDeliveryCode(ctx context.Context, orderID, given string) error {
	var code *string
	var attempts int
	if err := s.pg.QueryRow(ctx,
		`SELECT delivery_code, delivery_code_attempts FROM orders WHERE id = $1`,
		orderID).Scan(&code, &attempts); err != nil {
		return err
	}
	if code == nil {
		return nil
	}
	if attempts >= orders.DeliveryCodeMaxAttempts {
		return errDeliveryCodeLocked
	}
	given = strings.TrimSpace(given)
	if given != "" && subtle.ConstantTimeCompare([]byte(given), []byte(*code)) == 1 {
		return nil
	}
	// **والعدُّ ذرّيٌّ** — محاولتان متزامنتان لا تُحسبان واحدة.
	var n int
	if err := s.pg.QueryRow(ctx, `
		UPDATE orders SET delivery_code_attempts = delivery_code_attempts + 1
		WHERE id = $1 AND delivery_code IS NOT NULL
		RETURNING delivery_code_attempts`, orderID).Scan(&n); err != nil {
		return err
	}
	if n >= orders.DeliveryCodeMaxAttempts {
		return errDeliveryCodeLocked
	}
	return errDeliveryCodeWrong(orders.DeliveryCodeMaxAttempts - n)
}

// whatsAppText **البوتُ بلغة المحرّك** — «وصلت» أم لا.
type whatsAppText struct{ s *Server }

func (w whatsAppText) Ready() bool { return w.s.merchantReady() }
func (w whatsAppText) SendText(ctx context.Context, phone, text string) bool {
	return w.s.sendText(ctx, phone, text)
}
