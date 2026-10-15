-- ══════════════════════════════════════════════════════════════════════
-- **الأهدافُ والمكافآت — قراراتُ المالك ٢٠٢٦-١٠-١٥**
-- ══════════════════════════════════════════════════════════════════════
--
--   الدورُ على الطلب والقيد   **من له دورا سائقٍ ومندوبٍ معاً كان يُعدّ مرّتين** في كروت
--                            الصفحة: مكافأتُه اليدويّةُ تظهر في التبويبين. **والآن يُكتب
--                            الدورُ الذي اقتُرحت له** (`incentive_requests.role`) ويُحمَل
--                            إلى صفّ الحافز (`incentives.role`)، **ويُعدّ فيه وحدَه.**
--   مكافأةُ الموقوف تُحفظ     **من بلغ مرحلةً وهو موقوفٌ لا يخسرها** («يظل إذا حقق الشروط»):
--                            صفٌّ في `incentive_pending_grants` يُصرف يومَ يُعاد تفعيلُه،
--                            **وجولةٌ دوريّةٌ تلتقط ما فات.**

-- ── ١ · الدورُ الذي يُعدّ فيه ─────────────────────────────────────────
ALTER TABLE incentive_requests
    ADD COLUMN IF NOT EXISTS role text CHECK (role IS NULL OR role IN ('driver', 'sales'));
ALTER TABLE incentives
    ADD COLUMN IF NOT EXISTS role text CHECK (role IS NULL OR role IN ('driver', 'sales'));

-- **والماضي أفضلَ جهد**: مكافأةُ الهدف دورُها في `target_role`، والطلبُ القديمُ دورُ
-- صاحبه (السائقُ أوّلاً كما يُكتب الدين)، والقيدُ اليدويُّ دورُ طلبه.
UPDATE incentives SET role = target_role
 WHERE role IS NULL AND for_target AND target_role IN ('driver', 'sales');

UPDATE incentive_requests r
   SET role = CASE
         WHEN EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = r.user_id AND ur.role_code = 'driver')
           THEN 'driver'
         WHEN EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = r.user_id AND ur.role_code = 'sales')
           THEN 'sales'
       END
 WHERE r.role IS NULL;

UPDATE incentives i
   SET role = r.role
  FROM incentive_requests r
 WHERE r.incentive_id = i.id AND i.role IS NULL AND r.role IS NOT NULL;

UPDATE incentives i
   SET role = CASE
         WHEN EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = i.user_id AND ur.role_code = 'driver')
           THEN 'driver'
         WHEN EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = i.user_id AND ur.role_code = 'sales')
           THEN 'sales'
       END
 WHERE i.role IS NULL;

-- ── ٢ · مكافأةٌ بلغها وهو موقوف ──────────────────────────────────────
--
-- **صفٌّ مفتوحٌ واحدٌ لكلّ شخصٍ ودورٍ وشهر** — والصرفُ نفسُه يمرّ بالفهرس الفريد
-- (`incentives_one_target_per_month`)، **فلا تُدفع مرحلةٌ مرّتين ولو أُعيد الصرف.**
CREATE TABLE IF NOT EXISTS incentive_pending_grants (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         text NOT NULL CHECK (role IN ('driver', 'sales')),
    month        text NOT NULL CHECK (month ~ '^[0-9]{4}-[0-9]{2}$'),
    created_at   timestamptz NOT NULL DEFAULT now(),
    paid_at      timestamptz,
    paid_amount  bigint NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS incentive_pending_grants_open
    ON incentive_pending_grants (user_id, role, month) WHERE paid_at IS NULL;
