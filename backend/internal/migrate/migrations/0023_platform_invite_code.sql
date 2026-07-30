-- كود دعوة خاص بالمنصة: التسجيلات التي لا تأتي عبر مندوب تُنسب للمنصة (لا تُرفض).
-- يظهر للمتجر في نموذج التسجيل (للقراءة فقط)، ويحق للمدير تغييره من الإعدادات.
INSERT INTO app_settings (key, value) VALUES
    ('platform.invite_code', '"RAHALGO"')
ON CONFLICT (key) DO NOTHING;
