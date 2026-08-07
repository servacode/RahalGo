package identity

import (
	"context"
)

// توثيق رقم واتساب.
//
// الفرق بينه وبين تغيير رقم الحساب: ذاك يبدّل **هوية الدخول**، وهذا يثبت
// **قناة التواصل**. قد يتطابق الرقمان وقد يفترقان — وكثيراً ما يفترقان: رقم
// الدخول قد يكون خطاً أرضياً أو شريحةً بلا واتساب، ورقم العمل غيره.
//
// ولماذا نوثّق أصلاً؟ لأن الرقم الذي لم نرسل إليه رمزاً قط هو مجرد نصّ كتبه
// أحدهم في حقل. التوثيق يجعله واقعاً: أرسلنا رمزاً عبر واتساب، فوصل، فأدخله.

// RequestWhatsAppVerify يرسل رمزاً عبر واتساب إلى الرقم المراد توثيقه.
func (s *Service) RequestWhatsAppVerify(ctx context.Context, userID, rawPhone, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	// لا فحص تفرّد هنا عمداً: رقم الواتساب قناة تواصل لا هوية دخول، وقد يشترك
	// فيه شريكان في متجر واحد. الفحص الذي يهمّ هو أن يصل الرمز إلى صاحبه.
	_ = userID
	return s.sendOTPFor(ctx, phone, "whatsapp", "otp:wa:", ip)
}

// ConfirmWhatsAppVerify يتحقق من الرمز ويثبّت الرقم موثَّقاً.
func (s *Service) ConfirmWhatsAppVerify(ctx context.Context, userID, rawPhone, code, ip string) error {
	phone, ok := NormalizePhone(rawPhone)
	if !ok {
		return ErrInvalidPhone
	}
	valid, err := s.repo.ConsumeOTP(ctx, phone, s.hashOTP(phone, code), "whatsapp")
	if err != nil {
		return err
	}
	if !valid {
		return ErrOTPInvalid
	}
	if err := s.repo.SetWhatsApp(ctx, userID, phone); err != nil {
		return err
	}
	s.repo.Audit(ctx, &userID, "auth.whatsapp_verified", "user", userID, ip,
		map[string]any{"whatsapp": phone})
	return nil
}

// SetWhatsApp يثبّت رقم واتساب موثَّقاً للحساب.
func (r *Repo) SetWhatsApp(ctx context.Context, userID, phone string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users
		SET whatsapp_phone = $2, whatsapp_verified_at = now(), updated_at = now()
		WHERE id = $1`, userID, phone)
	return err
}
