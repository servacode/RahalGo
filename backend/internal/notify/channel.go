package notify

import (
	"context"
	"errors"
	"log/slog"
)

// ══════════════════════════════════════════════════════════════════════
// **بأيّ طريقٍ يصل رمزُ التحقّق — يُبدَّل من اللوحة لا بنشرٍ جديد**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «بدّي أضيف ميزة تحقّق SMS ونجهّزها أيضاً،
//
//	ونقدر نفعّلها أو نطفّيها من لوحة التحكّم».)
//
// # ولماذا يلزم البديل أصلاً
//
// **البوتُ غيرُ رسميّ**: يُقترن بهاتفٍ حقيقيّ، **ومن أكثر من الإرسال
// الآليِّ حُظر حسابُه** — فيقف بابُ الدخول للمنصّة كلِّها بلا سابق
// إنذار. **ومنصّةٌ بابُها واحدٌ يُغلق بقرارِ شركةٍ أخرى ليست منصّة.**
//
// **ورقمٌ ليس على واتساب لا يصله شيء** — وهو حالٌ قائمةٌ اليوم لا
// احتمال: `ErrNoWhatsApp` كُتبت لأنّها وقعت.
//
// # وثلاثةُ أوضاع
//
//	whatsapp            — كما هو اليوم
//	sms                 — الرسائلُ النصّيّةُ وحدَها
//	whatsapp_then_sms   — واتسابُ أوّلاً، **وSMS لمن لا يصله**
//
// **والثالثُ هو الوضعُ الذي يُنصح به حين يوجد مزوّد**: أرخصُ ما أمكن،
// **ولا يقف البابُ حين يقف البوت.**
//
// # ولا يُبدَّل إلى ما لا يعمل
//
// **من ضبط `sms` بلا بوّابةٍ مضبوطةٍ في البيئة أغلق بابَ الدخول على
// الجميع** — **وخيارٌ في اللوحة يُطفئ المنصّةَ بضغطةٍ ليس خياراً بل
// فخّ.** فيُفحص قبل أن يُطاع، **ويُسقَط إلى واتساب مع سطرٍ في السجلّ.**

// OTPChannel **مُرسِلٌ يقرأ قناتَه عند كلّ إرسال.**
//
// **ويُقرأ عند كلّ إرسالٍ لا مرّةً عند الإقلاع**: مفتاحٌ يُبدَّل في
// اللوحة يعمل في اللحظة، **وإعدادٌ يحتاج إعادةَ تشغيلٍ ليعمل يُنسى
// فيُظنّ أنّه لا يعمل.**
type OTPChannel struct {
	wa     OTPSender
	sms    *SMSSender
	text   func(code string) string
	mode   func(ctx context.Context) string
	logger *slog.Logger
}

// NewOTPChannel **يربط المُرسِلَين بمفتاحِ اللوحة.**
//
// **و`text` تصنع نصَّ الرسالة النصّيّة** — **ولا يُعاد استعمالُ قالب
// واتساب**: قالبُ واتساب قد يكون بأسطرٍ ورموز، **والرسالةُ النصّيّةُ
// تُحاسَب بالحرف** فسطرٌ زائدٌ رسالتان.
func NewOTPChannel(wa OTPSender, sms *SMSSender, text func(code string) string,
	mode func(ctx context.Context) string, logger *slog.Logger) *OTPChannel {
	return &OTPChannel{wa: wa, sms: sms, text: text, mode: mode, logger: logger}
}

const (
	ChannelWhatsApp     = "whatsapp"
	ChannelSMS          = "sms"
	ChannelWhatsAppThen = "whatsapp_then_sms"
)

// resolve **الوضعُ الفعليُّ بعد فحصِ ما يعمل.**
func (c *OTPChannel) resolve(ctx context.Context) string {
	mode := ChannelWhatsApp
	if c.mode != nil {
		if m := c.mode(ctx); m != "" {
			mode = m
		}
	}
	if mode == ChannelWhatsApp {
		return mode
	}
	if !c.sms.Configured() {
		// **ولا يُسكَت عن السقوط** — من ضبط `sms` في اللوحة يظنّها تعمل،
		// **وسطرٌ في السجلّ هو ما يجعله يسأل «لماذا لم تعمل؟».**
		c.logger.Warn("قناةُ الرمز مضبوطةٌ على الرسائل النصّيّة وبوّابتُها غيرُ مضبوطة — رُدَّت إلى واتساب",
			"mode", mode)
		return ChannelWhatsApp
	}
	return mode
}

// SendOTP **يُرسل بالقناة المضبوطة.**
func (c *OTPChannel) SendOTP(ctx context.Context, phone, code string) error {
	switch c.resolve(ctx) {
	case ChannelSMS:
		return c.sms.SendOTP(ctx, phone, code, c.text(code))
	case ChannelWhatsAppThen:
		err := c.wa.SendOTP(ctx, phone, code)
		if err == nil {
			return nil
		}
		// **ولا يُسقَط إلى SMS على كلّ خطأ.**
		//
		// **ضغطُ الحدِّ ليس عطبَ قناة**: من بلغ حدَّه يُقال له ذلك،
		// **وإرسالُ رسالةٍ نصّيّةٍ له يجعل الحدَّ بلا معنىً** — يُعيد
		// المحاولةَ حتّى تنفد رسائلُ المنصّة.
		if !errors.Is(err, ErrSMSNotConfigured) {
			c.logger.Info("واتساب لم يوصّل الرمز — تُجرَّب الرسالةُ النصّيّة",
				"phone", phone, "err", err)
		}
		if smsErr := c.sms.SendOTP(ctx, phone, code, c.text(code)); smsErr != nil {
			// **ويُردّ خطأُ واتساب لا خطأُ الرسائل**: هو القناةُ الأولى،
			// **ورسالةُ «بوّابةُ الرسائل لا تستجيب» لا تعني شيئاً لمن
			// طلب رمزاً على واتساب.**
			return err
		}
		return nil
	default:
		return c.wa.SendOTP(ctx, phone, code)
	}
}
