-- ══════════════════════════════════════════════════════════════════════
-- **قسمُ الأدوار والصلاحيّات — قرارات المالك 2026-10-04**
-- ══════════════════════════════════════════════════════════════════════
--
-- # 1 · المالكُ الأعلى صار له حامل (القرار 1)
--
-- **قِيس على التجهيز**: دورُ `owner_super_admin` بلا حامل، وحاملُ `admin`
-- حسابٌ واحدٌ هو المالك. **فكلُّ حمايةٍ مبنيّةٍ على المالك الأعلى لا تعمل**،
-- ونزعُ `roles.manage` من `admin` كان يُقفل اللوحةَ على صاحبها.
--
-- **فيُمنح `owner_super_admin` لحامل/حاملي `admin` الفعّالين — إن لم يكن
-- للمنصّة مالكٌ أعلى بعد.** ويبقى `admin` معه (يبقى لكبار الموظّفين لاحقاً).
-- **وقبلها**: كلُّ قدرةٍ يملكها `admin` تُضاف للمالك الأعلى إن نقصته — فلا
-- يخسر المالكُ شيئاً.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'owner_super_admin', capability_code FROM role_capabilities
 WHERE role_code = 'admin'
ON CONFLICT DO NOTHING;

WITH granted AS (
    INSERT INTO user_roles (user_id, role_code, granted_by)
    SELECT ur.user_id, 'owner_super_admin', NULL
      FROM user_roles ur JOIN users u ON u.id = ur.user_id
     WHERE ur.role_code = 'admin' AND u.status = 'active'
       AND NOT EXISTS (SELECT 1 FROM user_roles WHERE role_code = 'owner_super_admin')
    ON CONFLICT DO NOTHING
    RETURNING user_id
)
INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
SELECT NULL, 'admin.owner_bootstrap', 'user', user_id::text,
       jsonb_build_object('role', 'owner_super_admin', 'via', 'migration 0270',
                          'reason', 'قرار المالك 2026-10-04: حامل مدير المنصة هو المالك')
  FROM granted;

-- # 2 · الدورُ القديم `ops` يُحذف (القرار 3)
--
-- **حاملوه يُنقلون إلى `operations` أوّلاً** — ثمّ يُحذف الدورُ وقدراتُه.
WITH moved AS (
    INSERT INTO user_roles (user_id, role_code, granted_by)
    SELECT user_id, 'operations', granted_by FROM user_roles WHERE role_code = 'ops'
    ON CONFLICT DO NOTHING
    RETURNING user_id
)
INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
SELECT NULL, 'admin.role_grant', 'user', user_id::text,
       jsonb_build_object('role', 'operations', 'via', 'migration 0270',
                          'reason', 'نقل من الدور القديم ops قبل حذفه')
  FROM moved;

DELETE FROM user_roles WHERE role_code = 'ops';

-- # 3 · والأدوارُ التجريبيّةُ والمؤقّتةُ الفارغة تُحذف (القرار 3)
--
-- **كلُّ دورٍ ليس من القائمة الكانونيّة ولا حاملَ له** — مثل `s1_ops_viewer`
-- و`s1_orders_only`. والكانونيّةُ هي ما تصنّفه الشيفرة (`authz.roleClasses`)،
-- **و`observability` منها** (أنشأه المالكُ في الإنتاج وصنّفته الشيفرةُ موظّفاً).
WITH gone AS (
    DELETE FROM roles r
     WHERE r.code NOT IN (
           'owner_super_admin', 'admin',
           'operations', 'finance', 'customer_support', 'analytics',
           'driver_verification', 'merchant_verification', 'marketing_content',
           'trust_safety', 'observability', 'platform_monitor',
           'customer', 'driver', 'merchant', 'sales')
       AND NOT EXISTS (SELECT 1 FROM user_roles ur WHERE ur.role_code = r.code)
    RETURNING r.code, r.name_key
)
INSERT INTO audit_log (actor_user_id, action, entity, entity_id, details)
SELECT NULL, 'admin.role_delete', 'role', code,
       jsonb_build_object('name', name_key, 'via', 'migration 0270',
                          'reason', 'قرار المالك 2026-10-04: حذف الأدوار القديمة والتجريبية الفارغة')
  FROM gone;
