-- الإعدادات الديناميكية المركزية (GROUND-RULES §1.3):
-- كل إعدادات العمل تُخزَّن هنا وتُدار من لوحة الأدمن — تغييرها لا يتطلب نشراً.
CREATE TABLE app_settings (
    key        text PRIMARY KEY,
    value      jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now(),
    updated_by uuid REFERENCES users (id)
);

-- قالب رسالة OTP عبر واتساب — {code} يُستبدل بالرمز
INSERT INTO app_settings (key, value) VALUES
    ('whatsapp.otp_template',
     '"رمز التحقق الخاص بك في رحال غو هو: {code}\n\nلا تشارك هذا الرمز مع أي شخص."');
