-- ══════════════════════════════════════════════════════════════════════
-- **تصفيرُ الإنتاج قبل الشغل الحقيقيّ — والإعدادُ لا يُمسّ** (قرارُ المالك ٢٠٢٦-١٠-٠٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- «بدّي سيرفر الإنتاج نظيف لنبدأ شغل حقيقي… نصفّره من الحسابات والطلبات والمتاجر
--  والإشعارات وكلّشي، بس الأقسام والصور والإعدادات لا تلمسهن… حساب الأدمن وتبع
--  مراقب غوغل بلاي خلّيهن».
--
--	يبقى  ←  ثلاثةُ حسابات: المالك (+963985395131) · مراقبُ غوغل بلاي (+963900000099)
--	         · حسابُ النظام (+000000000001، مستحقّاتُ المتاجر النقديّة المعلّقة)
--	         · الإعداداتُ والأسرار · أقسامُ السوق وتصنيفاتُها · السلايدر · **كلُّ الصور**
--	         · المدنُ والمحافظاتُ والنواحي ومناطقُ التوصيل وساعاتُها · الدوامُ والإيقاف
--	         · الأدوارُ والصلاحيّات · أبوابُ المصروف · **جلسةُ بوت الواتساب (whatsmeow_*)**
--	يُمحى ←  كلُّ ما سواها: الحساباتُ الأخرى (والمحذوفةُ القديمة) · الطلبات · المتاجر
--	         · المالُ والدفتر · الإشعارات · سجلُّ الأحداث · الأكواد · الإحصاءات
--
-- **ولا يعمل إلّا بإذنين صريحين**: `APP_ENV=production` **و`CONFIRM=RESET-PRODUCTION`**
-- — فلا يُشغَّل بالخطأ على التجهيز ولا يُشغَّل على الإنتاج بلا قصد.
-- **والمعاملةُ واحدة**: إن سقط سطرٌ رُدّ كلُّ شيء.
--
-- التشغيل (بعد نسخةٍ احتياطيّة):
--   docker exec -i -e APP_ENV=production -e CONFIRM=RESET-PRODUCTION rahalgo-postgres-1 \
--     psql -U rahalgo -d rahalgo -v ON_ERROR_STOP=1 < reset-production-keep-config.sql

\getenv app_env APP_ENV
\getenv confirm CONFIRM
SELECT CASE WHEN :'app_env' IS DISTINCT FROM 'production'
              OR :'confirm' IS DISTINCT FROM 'RESET-PRODUCTION'
            THEN 1/0 END AS refuse_without_both_flags \gset

BEGIN;

CREATE TEMP TABLE keep_users ON COMMIT DROP AS
SELECT id FROM users
 WHERE phone IN ('+963985395131', '+963900000099', '+000000000001');

DO $$
BEGIN
  IF (SELECT count(*) FROM keep_users) <> 3 THEN
    RAISE EXCEPTION 'الحساباتُ الثلاثةُ ليست كلُّها موجودة — لا يُمحى شيء';
  END IF;
END $$;

ALTER TABLE audit_log DISABLE TRIGGER USER;

-- ١ · كلُّ جدولٍ ليس من الإعداد يُفرَغ — **والقائمةُ من القاعدة لا من اليد**.
DO $$
DECLARE
  keep text[] := ARRAY[
    'schema_migrations', 'spatial_ref_sys',
    'app_settings', 'app_secrets', 'banners', 'categories', 'cities',
    'governorates', 'districts', 'delivery_zones', 'delivery_zone_hours',
    'branches', 'operational_areas', 'platform_sections', 'platform_hours',
    'service_closure', 'roles', 'role_capabilities', 'expense_categories',
    'geocode_settings', 'geocode_settings_default', 'media',
    -- **وما يبقى منه صفوفُ الثلاثة** — يُرشَّح تحت لا يُفرَغ.
    'users', 'user_roles', 'wallets', 'refresh_tokens', 'device_tokens',
    'user_addresses', 'admin_market_seen', 'step_up_grants'];
  tabs text;
BEGIN
  SELECT string_agg(format('%I', tablename), ', ')
    INTO tabs
    FROM pg_tables
   WHERE schemaname = 'public'
     AND tablename <> ALL (keep)
     -- **وجلسةُ بوت الواتساب لا تُمسّ** — ومحوُها يفكّ ربطَ الرقم.
     AND tablename NOT LIKE 'whatsmeow\_%';
  EXECUTE 'TRUNCATE TABLE ' || tabs || ' RESTART IDENTITY';
END $$;

-- ٢ · صفوفُ الثلاثة وحدَها في الجداول المرشَّحة.
DELETE FROM refresh_tokens    WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM user_roles        WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM wallets           WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM device_tokens     WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM user_addresses    WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM admin_market_seen WHERE user_id NOT IN (SELECT id FROM keep_users);
DELETE FROM step_up_grants    WHERE user_id NOT IN (SELECT id FROM keep_users);

-- ٣ · **وما يبقى ويشير إلى حسابٍ يُمحى يُنسب إلى المالك** — الصورُ والأقسامُ
-- واللافتاتُ والإعداداتُ لا تُحذف لأنّ من رفعها حسابٌ قديم.
DO $$
DECLARE
  owner uuid := (SELECT id FROM users WHERE phone = '+963985395131');
  r record;
BEGIN
  FOR r IN
    SELECT c.conrelid::regclass::text AS tbl, a.attname AS col
      FROM pg_constraint c
      JOIN pg_attribute a ON a.attrelid = c.conrelid AND a.attnum = ANY (c.conkey)
     WHERE c.contype = 'f' AND c.confrelid = 'users'::regclass
       AND c.conrelid <> 'users'::regclass
  LOOP
    EXECUTE format(
      'UPDATE %s SET %I = $1 WHERE %I IS NOT NULL AND %I NOT IN (SELECT id FROM keep_users)',
      r.tbl, r.col, r.col, r.col) USING owner;
  END LOOP;
END $$;

DELETE FROM users WHERE id NOT IN (SELECT id FROM keep_users);
UPDATE wallets SET balance = 0, reserved = 0;

ALTER TABLE audit_log ENABLE TRIGGER USER;

COMMIT;

SELECT 'users' AS t, count(*) FROM users
UNION ALL SELECT 'orders', count(*) FROM orders
UNION ALL SELECT 'merchants', count(*) FROM merchants
UNION ALL SELECT 'notifications', count(*) FROM notifications
UNION ALL SELECT 'audit_log', count(*) FROM audit_log
UNION ALL SELECT 'platform_sections', count(*) FROM platform_sections
UNION ALL SELECT 'categories', count(*) FROM categories
UNION ALL SELECT 'banners', count(*) FROM banners
UNION ALL SELECT 'media', count(*) FROM media
UNION ALL SELECT 'app_settings', count(*) FROM app_settings
UNION ALL SELECT 'delivery_zones', count(*) FROM delivery_zones
UNION ALL SELECT 'role_capabilities', count(*) FROM role_capabilities
-- **وجداولُ البوت باقية** — عددُها لا محتواها (ولا تُفترض أسماؤها).
UNION ALL SELECT 'whatsmeow_tables', count(*) FROM pg_tables WHERE tablename LIKE 'whatsmeow\_%';
