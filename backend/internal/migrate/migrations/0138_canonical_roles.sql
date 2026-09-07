-- ══════════════════════════════════════════════════════════════════════
-- **الأدوارُ التسعةُ الوظيفيّةُ ومصفوفتُها**
-- ══════════════════════════════════════════════════════════════════════
--
-- (`ADG-2` · `AQ-1` — دورةُ إصلاحٍ ٢٥، ٢٠٢٦-٠٩-٠٧ · بعقدِ مالكٍ صريح.)
--
-- # ولماذا بذرٌ لمرّةٍ واحدة
--
-- **الأدمنُ يُبدّل هذه الصفوفَ من اللوحة** — **وإقلاعٌ يُعيد كتابتها
-- يمحو عملَه في كلّ مرّة.** **فهجرةٌ تقع مرّةً، ولا يُنادى بذرٌ عند
-- كلّ تشغيل.**
--
-- **و`ON CONFLICT DO NOTHING`** — **فإعادةُ الهجرة لا تُعيد ما نُزع
-- عمداً.**

-- ── تعريفاتُ الأدوار ─────────────────────────────────────────────────
--
-- **ولا سلطةَ في الاسم** — **القدراتُ هي الحقيقة.** والاسمُ للعرض.
INSERT INTO roles (code, name_key) VALUES
    ('owner_super_admin',    'roles.owner_super_admin'),
    ('operations',           'roles.operations'),
    ('customer_support',     'roles.customer_support'),
    ('driver_verification',  'roles.driver_verification'),
    ('merchant_verification','roles.merchant_verification'),
    ('trust_safety',         'roles.trust_safety'),
    ('marketing_content',    'roles.marketing_content'),
    ('analytics',            'roles.analytics')
ON CONFLICT (code) DO NOTHING;

-- **و`finance` قائمٌ من قبل** — لا يُعاد إنشاؤه ولا يُبدَّل اسمُه.

-- ══════════════════════════════════════════════════════════════════════
-- **المصفوفةُ الكانونيّة**
-- ══════════════════════════════════════════════════════════════════════

-- **`owner_super_admin`** — كلُّ قدرةٍ في المعجم.
--
-- **وهو وحدَه يملك `roles.manage`** بقرار المالك.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'owner_super_admin', c FROM unnest(ARRAY[
    'orders.read', 'orders.intervene',
    'users.read', 'users.status.manage', 'roles.manage',
    'finance.read', 'finance.manage', 'payouts.decide',
    'merchants.manage', 'merchants.verify', 'drivers.manage',
    'settings.general.manage', 'settings.financial.manage',
    'settings.security.manage',
    'content.manage', 'analytics.read',
    'support.manage', 'safety.manage', 'audit.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`operations`** — طلباتٌ وتشغيلٌ وقراءات.
--
-- **ولا مالَ ولا أدوارَ ولا إعداداتٍ حسّاسةٍ ولا حظر.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'operations', c FROM unnest(ARRAY[
    'orders.read', 'orders.intervene',
    'users.read',
    'merchants.manage', 'drivers.manage',
    'analytics.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`customer_support`** — تذاكرُ ونزاعاتٌ وطوارئُ وقراءات.
--
-- **ولا مالَ ولا إعداداتٍ حسّاسةٍ ولا حظر.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'customer_support', c FROM unnest(ARRAY[
    'support.manage',
    'orders.read', 'users.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`driver_verification`** — مراجعةُ السائقين وقراءاتُهم.
--
-- **ولا مسارَ توثيقٍ مستقلٍّ للسائق اليوم** — **فلا قدرةَ تُخترَع
-- له**، ويُسجَّل ذلك بندَ ترحيلٍ قبل الإنتاج.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'driver_verification', c FROM unnest(ARRAY[
    'drivers.manage', 'users.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`merchant_verification`** — مراجعةُ المرشَّحين والقوائم.
--
-- **ولا `merchants.manage`**: **المراجعُ يوافق ويرفض ولا يُنشئ متجراً
-- ولا يُحرّر قوائمَ غيرِه.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'merchant_verification', c FROM unnest(ARRAY[
    'merchants.verify', 'users.read', 'orders.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`finance`** — مالٌ وسحوباتٌ وإعداداتٌ ماليّةٌ وقراءات.
--
-- **ولا أدوارَ ولا إعداداتٍ أمنيّةٍ ولا سلطاتِ سلامة.**
--
-- **ويُضاف إليه ما لم يكن عنده**: `settings.financial.manage` —
-- **وهو صريحٌ في نصّ `AQ-1`** («التسعيرَ والهوامش · العمولات»).
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'finance', c FROM unnest(ARRAY[
    'finance.read', 'finance.manage', 'payouts.decide',
    'settings.financial.manage',
    'orders.read', 'users.read', 'analytics.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`trust_safety`** — الإيقافُ والحظرُ والإنذاراتُ والمخالفات.
--
-- **ولا محفظةَ ولا خزينةَ ولا سحوبات.**
--
-- **ولا `settings.security.manage`**: **نصُّ العقد لا يجعل إعداداتِ
-- الأمن من عمله** — **ولا يُفترَض ما لم يُقَل.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'trust_safety', c FROM unnest(ARRAY[
    'safety.manage', 'users.status.manage',
    'users.read', 'orders.read', 'audit.read'
]) AS c ON CONFLICT DO NOTHING;

-- **`marketing_content`** — محتوىً وإعداداتٌ عامّة.
--
-- **ولا ماليٍّ ولا أمنيٍّ ولا حظرَ ولا مال.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'marketing_content', c FROM unnest(ARRAY[
    'content.manage', 'settings.general.manage'
]) AS c ON CONFLICT DO NOTHING;

-- **`analytics`** — قراءةٌ محضة.
--
-- **ولا تبديلَ عملٍ إطلاقاً** — ويحرسه فحصٌ يمرّ على كلّ صنفِ تبديل.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'analytics', c FROM unnest(ARRAY[
    'analytics.read', 'orders.read', 'users.read', 'finance.read'
]) AS c ON CONFLICT DO NOTHING;

-- ══════════════════════════════════════════════════════════════════════
-- **والأدوارُ القديمةُ تُكمَّل بما فرضته القدراتُ الجديدة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **ولا تُحذَف ولا تُرقّى ولا تُعاد تسميتُها** (قرارُ المالك) —
-- **ووصولُها المُثبَتُ يبقى حتّى يقع ترحيلُ الحسابات.**
--
-- **وأربعُ قدراتٍ أُضيفت في هذه الدورة** (`support.manage` ·
-- `safety.manage` · `merchants.verify` · `audit.read`) — **فتُمنَح
-- لمن كان يبلغ مساراتِها بالحارس القديم**، وإلّا انكسر عملٌ قائم.

-- `admin` كان يبلغ كلَّ شيء.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'admin', c FROM unnest(ARRAY[
    'merchants.verify', 'support.manage', 'safety.manage', 'audit.read'
]) AS c ON CONFLICT DO NOTHING;

-- `ops` كان يبلغ لوحةَ العمليّات ومراجعةَ القوائم والدعمَ والمخالفات.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'ops', c FROM unnest(ARRAY[
    'merchants.verify', 'support.manage', 'safety.manage'
]) AS c ON CONFLICT DO NOTHING;

-- `finance` كان يبلغ سجلَّ التدقيق والتذاكرَ والنزاعاتِ بالحارس القديم
-- (`admin,finance`) — **ويبقى حتّى ترحيلِ الحسابات.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'finance', c FROM unnest(ARRAY[
    'audit.read', 'support.manage'
]) AS c ON CONFLICT DO NOTHING;
