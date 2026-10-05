-- ══════════════════════════════════════════════════════════════════════
-- **تصفيرُ التجهيز إلى أربعة موظّفين** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- «لازم يظلّ عندنا فقط حساب مدير المنصّة وحساب العمليّات وحساب الماليّة
--  وحساب الدعم… والباقي كلّه صفر» — **قبل التجربة الجماعيّة الأولى.**
--
--	يبقى  ←  الحساباتُ الأربعة (بأدوارها وجلساتها) · الإعداداتُ والأسرار ·
--	         الأدوارُ والصلاحيّات · التغطيةُ (محافظاتٌ ومدنٌ ومناطقُ ونطاقات) ·
--	         أقسامُ السوق وتصنيفاتُ المتاجر · ساعاتُ المنصّة وحالُ إيقافها ·
--	         الوسائطُ التي يشير إليها ما يبقى (صورُ الأقسام واللافتات والحسابات)
--	يُمحى ←  كلُّ ما سواها: الحساباتُ الأخرى · المتاجرُ والأصناف · الطلبات ·
--	         المالُ كلُّه (والمحافظُ الأربعُ تعود صفراً) · الشكاوى · الطوارئ ·
--	         الإشعارات · سجلُّ الأحداث · طلباتُ الانضمام والتوسّع
--
-- **وأبوابُ المصروف الثمانية تُعاد** (كما تزرعها الهجرة ٠١٠٩).
--
-- **ويُطفئ حارسَ السجلّ داخلَ المعاملة ثمّ يعيده** — مسموحٌ على التجهيز
-- والتطوير وحدَهما، **ولا يعمل على الإنتاج أبداً** (`APP_ENV=production`
-- يُرفض قبل أن يمسّ شيئاً).
--
-- التشغيل:
--   docker exec -i -e APP_ENV=staging rahalgo-staging-postgres \
--     psql -U rahalgo -d rahalgo_staging -v ON_ERROR_STOP=1 < reset-to-four-staff.sql

\getenv app_env APP_ENV
SELECT CASE WHEN :'app_env' = 'production'
            THEN 1/0 END AS refuse_production \gset

BEGIN;

CREATE TEMP TABLE keep_users ON COMMIT DROP AS
SELECT id FROM users
 WHERE phone IN ('+963985395131', '+963990000001', '+963990000002', '+963990000003');

DO $$
BEGIN
  IF (SELECT count(*) FROM keep_users) <> 4 THEN
    RAISE EXCEPTION 'الحساباتُ الأربعةُ ليست كلُّها موجودة — لا يُمحى شيء';
  END IF;
END $$;

ALTER TABLE audit_log DISABLE TRIGGER USER;

-- ١ · كلُّ جدولٍ ليس من الإعداد يُفرَغ — **والقائمةُ من القاعدة لا من اليد**،
-- فجدولٌ يُضاف غداً يُفرَغ معها.
DO $$
DECLARE
  keep text[] := ARRAY[
    'schema_migrations', 'spatial_ref_sys',
    'app_settings', 'app_secrets', 'banners', 'categories', 'cities',
    'governorates', 'districts', 'delivery_zones', 'delivery_zone_hours',
    'branches', 'operational_areas', 'platform_sections', 'platform_hours',
    'service_closure', 'roles', 'role_capabilities', 'expense_categories',
    'geocode_settings', 'geocode_settings_default',
    -- **وما يبقى منه صفوفٌ بعينها** — يُرشَّح تحت لا يُفرَغ.
    'users', 'user_roles', 'wallets', 'refresh_tokens', 'media'];
  tabs text;
BEGIN
  SELECT string_agg(format('%I', tablename), ', ')
    INTO tabs
    FROM pg_tables
   WHERE schemaname = 'public' AND tablename <> ALL (keep);
  EXECUTE 'TRUNCATE TABLE ' || tabs || ' RESTART IDENTITY';
END $$;

-- ٢ · ما يبقى منه الأربعةُ وحدَهم.
DELETE FROM refresh_tokens WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM user_roles     WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM wallets        WHERE user_id NOT IN (SELECT id FROM keep_users);
-- ٢أ · الوسائطُ التي لا يشير إليها شيءٌ باقٍ — **قبل محو الحسابات**: الوسيطُ يشير
-- إلى من رفعه، **وما يبقى منه يُنسب إلى مدير المنصّة.**
DELETE FROM media m
 WHERE NOT EXISTS (SELECT 1 FROM platform_sections s WHERE s.image_media_id = m.id)
   AND NOT EXISTS (SELECT 1 FROM banners b WHERE b.image_media_id = m.id)
   AND NOT EXISTS (SELECT 1 FROM users u WHERE u.avatar_media_id = m.id);

UPDATE media SET created_by = (SELECT id FROM users WHERE phone = '+963985395131')
 WHERE created_by IS NOT NULL AND created_by NOT IN (SELECT id FROM keep_users);

DELETE FROM users          WHERE id      NOT IN (SELECT id FROM keep_users);
UPDATE wallets SET balance = 0, reserved = 0;

-- ٤ · أبوابُ المصروف الثمانية.
INSERT INTO expense_categories (name, sort_order)
SELECT v.name, v.sort FROM (VALUES
  ('إيجار', 1), ('رواتب', 2), ('كهرباء ومولّدة', 3), ('إنترنت', 4),
  ('وقود', 5), ('صيانة', 6), ('قرطاسية', 7), ('أخرى', 99)) AS v(name, sort)
WHERE NOT EXISTS (SELECT 1 FROM expense_categories e WHERE e.name = v.name);

ALTER TABLE audit_log ENABLE TRIGGER USER;

COMMIT;

SELECT 'users' AS t, count(*) FROM users
UNION ALL SELECT 'orders', count(*) FROM orders
UNION ALL SELECT 'merchants', count(*) FROM merchants
UNION ALL SELECT 'wallet_transactions', count(*) FROM wallet_transactions
UNION ALL SELECT 'platform_sections', count(*) FROM platform_sections
UNION ALL SELECT 'delivery_zones', count(*) FROM delivery_zones
UNION ALL SELECT 'app_settings', count(*) FROM app_settings
UNION ALL SELECT 'role_capabilities', count(*) FROM role_capabilities
UNION ALL SELECT 'expense_categories', count(*) FROM expense_categories;
