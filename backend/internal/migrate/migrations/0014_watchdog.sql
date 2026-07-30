-- مهل المراقبة والتصعيد (بالدقائق) — تُدار من شاشة الإعدادات
INSERT INTO app_settings (key, value) VALUES
    ('orders.accept_timeout_min',   '5'),   -- المتجر لم يقبل الطلب
    ('orders.driver_timeout_min',   '10'),  -- بلا سائق (تحضير/بحث) منذ مدة
    ('orders.delivery_timeout_min', '60')   -- طلب جارٍ منذ وقت طويل جداً
ON CONFLICT (key) DO NOTHING;
