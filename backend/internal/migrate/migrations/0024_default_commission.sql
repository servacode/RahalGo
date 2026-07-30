-- نسبة عمولة المنصة الافتراضية على المتاجر الجديدة (يعدّلها المدير من إعدادات
-- العمولات). كل متجر يبقى قابلاً لتخصيص نسبته الخاصة من صفحته.
INSERT INTO app_settings (key, value) VALUES
    ('merchants.default_commission_percent', '10')
ON CONFLICT (key) DO NOTHING;
