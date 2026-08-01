// Package notify قناة إرسال رموز التحقق والإشعارات.
// واجهة مجرّدة (PLAN.md §6.5): التنفيذ الفعلي واتساب (whatsmeow)،
// وفي التطوير مزوّد يكتب الرمز في السجل — التبديل بالتهيئة لا بالكود.
package notify

import (
	"context"
	"log/slog"
)

// TextSender من يستطيع إرسال رسالةٍ حرّة لا رمزَ تحقّقٍ فقط.
//
// **واجهةٌ ثانية لا توسيعٌ للأولى**: مُرسِلُ التطوير يطبع الرمز في الطرفية ولا
// يملك أن يُرسل شيئاً إلى أحد، فإلزامُه بها يجعله يكذب. ومن يحتاج الإرسال
// يسأل بـtype assertion ويتصرّف عند الغياب.
type TextSender interface {
	SendText(ctx context.Context, phone, text string) error
}

type OTPSender interface {
	SendOTP(ctx context.Context, phone, code string) error
}

// DevSender مزوّد التطوير: يطبع الرمز في سجل الخادم فقط.
type DevSender struct {
	Logger *slog.Logger
}

func (d *DevSender) SendOTP(_ context.Context, phone, code string) error {
	d.Logger.Info("DEV OTP (would be sent via WhatsApp)", "phone", phone, "code", code)
	return nil
}
