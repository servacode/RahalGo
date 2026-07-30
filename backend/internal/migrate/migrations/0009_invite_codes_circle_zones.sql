-- قرارات جلسة المراجعة: كود دعوة المندوبين + المناطق الدائرية + إعدادات الحوكمة

-- 1) كود الدعوة: لكل مندوب كود فريد يُسجَّل به المتاجر (والزبائن مستقبلاً)
ALTER TABLE users ADD COLUMN invite_code citext UNIQUE;
-- توليد أكواد للمندوبين الحاليين
UPDATE users
SET invite_code = 'RH-' || upper(substr(md5(random()::text || id::text), 1, 5))
WHERE invite_code IS NULL
  AND id IN (SELECT user_id FROM user_roles WHERE role_code = 'sales');

-- 2) المناطق: نموذج الدوائر (مركز + نصف قطر) بدل رسم المضلعات
DELETE FROM delivery_zones;  -- بيانات تجريبية فقط
ALTER TABLE delivery_zones DROP COLUMN polygon;
ALTER TABLE delivery_zones ADD COLUMN center   geography(Point, 4326) NOT NULL;
ALTER TABLE delivery_zones ADD COLUMN radius_m int NOT NULL DEFAULT 2000
    CHECK (radius_m BETWEEN 100 AND 50000);
CREATE INDEX delivery_zones_center_idx ON delivery_zones USING GIST (center);

-- 3) إعدادات الحوكمة الديناميكية (تُدار من اللوحة)
INSERT INTO app_settings (key, value) VALUES
    -- نسبة عمولة المندوب من عمولة المنصة على طلبات متاجره (%)
    ('sales.commission_percent', '10'),
    -- هل تحتاج تعديلات المتاجر على قوائمها موافقة العمليات؟ (القرار: مطفأة بداية)
    ('merchants.menu_requires_approval', 'false')
ON CONFLICT (key) DO NOTHING;
