package orders

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ══════════════════════════════════════════════════════════════════════
// **«تم استلام أمانتك، والسائق بالطريق إليك»** — قرارُ المالك ٢٠٢٦-١١-١٢
// ══════════════════════════════════════════════════════════════════════
//
// «وتصل رسالة تم استلام امانتك نحن في طريقنا اليك» ثمّ «حاليا نحن موقفين كود الاستلام بس بدنا
// الرسالة». **فالمستلِمُ في أمانة الواتساب كان لا يصله شيءٌ لحظةَ الاستلام** والكودُ مطفأ.
//
// **رسالةٌ واحدةٌ لا رسالتان**: إن وُلد كودٌ يصل بالواتساب نفسِه أُلحق سطرُه بها
// (`codeRoute.waText`)، وإلّا بُعثت وحدَها. **ولا تُسقط استلاماً وقع** — بعد الإيداع وفي الخلفية.

// WAAmanahPickedUpText **نصُّ الخبر** — باسم من استُلمت منه إن عُرف.
func WAAmanahPickedUpText(number int64, from string) string {
	from = strings.TrimSpace(from)
	if from == "" {
		return fmt.Sprintf("تم استلام أمانتك #%d، والسائق بالطريق إليك 🚗", number)
	}
	return fmt.Sprintf("تم استلام أمانتك #%d من %s، والسائق بالطريق إليك 🚗", number, from)
}

// waCodeLine **سطرُ الكود الملحَق** — حين يكون الكودُ مشغَّلاً.
func waCodeLine(code string) string {
	return fmt.Sprintf("كود الاستلام: %s — أعطه للسائق عند التسليم فقط.", code)
}

// withWAPickupNotice **يجهّز خبرَ الاستلام لأمانة الواتساب** — ويردّ مسارَ الكود كما هو أو مُعدَّلاً.
//
// كودٌ بالواتساب ⇒ النصُّ يصير الخبرَ وسطرَ الكود معاً. **وبلا كودٍ بالواتساب ⇒ يُبعث الخبرُ وحدَه.**
func (s *Service) withWAPickupNotice(ctx context.Context, orderID string, rt *codeRoute) *codeRoute {
	if s.whatsapp == nil || !s.whatsapp.Ready() {
		return rt
	}
	var number int64
	var amanah bool
	var origin, phone, from string
	if err := s.db.QueryRow(ctx, `
		SELECT number, wa_amanah, COALESCE(origin, ''), COALESCE(recipient_phone, ''), COALESCE(pickup_name, '')
		FROM orders WHERE id = $1`, orderID).Scan(&number, &amanah, &origin, &phone, &from); err != nil {
		if s.logger != nil {
			s.logger.Warn("خبرُ استلام الأمانة: قراءةُ الطلب سقطت", "order", orderID, "error", err)
		}
		return rt
	}
	if !amanah || origin != "whatsapp" || strings.TrimSpace(phone) == "" {
		return rt
	}
	text := WAAmanahPickedUpText(number, from)
	if rt != nil && rt.code != "" && rt.phone != "" {
		rt.waText = text + "\n" + waCodeLine(rt.code)
		return rt
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if !s.whatsapp.SendText(bg, phone, text) && s.logger != nil {
			s.logger.Warn("خبرُ استلام الأمانة لم يصل بالواتساب", "order", orderID)
		}
	}()
	return rt
}
