package server

// ══════════════════════════════════════════════════════════════════════
// **ترحيبُ الزبون الجديد — بالواتساب وفي التطبيق** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «بعد ما يسجّل نرحّب فيه كزبون بعائلة رحّال غو، ونقلّو أي مشكلة أو شكوى
//  يمكنك إرسالها من التطبيق أو من خلال التواصل معنا، ونفهّمو إنّ الطلبات من
//  خلال التطبيق وليس الاتصال أو الواتساب» — **"الاثنين أفضل".**
//
// **والإشعارُ في التطبيق دائماً**، **والواتساب بمفتاحٍ** (`customers.welcome_whatsapp`)
// — **فمن خشي على رقمه من الحظر يوم زحمةٍ أطفأه ولم يمسّ شيئاً آخر.**
//
// **ولا يؤخّر التسجيل**: البوتُ يتمهّل بين الرسائل (`whatsapp.send_delay_ms`)،
// **ومن انتظر ردَّ «أُنشئ حسابك» ستَّ ثوانٍ ظنّ التطبيقَ معلَّقاً.** فيُرسَل
// في الخلفية، **وتعثّرُه يُكتب في السجلّ ولا يُسقط حساباً تمّ.**

import (
	"context"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/notifications"
)

// welcomeCustomer **يرحّب بالزبون الجديد** — إشعارٌ في التطبيق، ورسالةُ واتساب إن شُغّلت.
func (s *Server) welcomeCustomer(ctx context.Context, userID, phone string) {
	text := strings.TrimSpace(s.settings.GetString(ctx, "customers.welcome_template"))
	if text == "" {
		return
	}
	// **وسطرُ فيديو الشرح إن نُشر** (٢٠٢٦-١٠-٠٩).
	if url := strings.TrimSpace(s.settings.GetString(ctx, "customers.tutorial_url")); url != "" {
		text += "\n\n🎬 شرح التطبيق بالفيديو: " + url
	}
	title, body := text, ""
	if i := strings.Index(text, "\n"); i > 0 {
		title, body = strings.TrimSpace(text[:i]), strings.TrimSpace(text[i+1:])
	}
	if s.notify != nil {
		s.notify.Notify(ctx, notifications.Input{
			UserID: userID, Kind: notifications.KindAccount,
			Title: title, Body: body,
			Apps: []string{notifications.AppCustomer},
		})
	}
	if phone == "" || !s.settings.GetBool(ctx, "customers.welcome_whatsapp") || !s.merchantReady() {
		return
	}
	go func() {
		bg, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if !s.sendText(bg, phone, text) {
			s.logger.Warn("ترحيبُ الزبون لم يصل بالواتساب", "user", userID)
		}
	}()
}
