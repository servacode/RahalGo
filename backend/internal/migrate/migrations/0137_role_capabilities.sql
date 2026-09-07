-- ══════════════════════════════════════════════════════════════════════
-- **دورٌ ← قدرة: يُبدّلها الأدمن بلا مهندس**
-- ══════════════════════════════════════════════════════════════════════
--
-- (`ADG-1` · `AQ-1` — دورةُ إصلاحٍ ٢٤، ٢٠٢٦-٠٩-٠٧ · بعقدِ مالكٍ صريح.)
--
-- # العقد
--
--	NO-CODE FOR OPERATIONS · CODE FOR NEW CAPABILITIES
--
-- **معجمُ القدرات في الشيفرة** (`internal/authz`) — **وقدرةٌ جديدةٌ
-- تحتاج مهندساً.**
--
-- **وربطُ الدور بالقدرة هنا** — **وتبديلُ من يملك قدرةً قائمةً لا
-- يحتاج مهندساً.**
--
-- # ولا مصدرَ حقيقةٍ ثانٍ للتعريف
--
-- **لا جدولَ `capabilities` يُكتَب فيه** — **فتعريفان يفترقان يومَ
-- يُضاف واحد.** **والصفُّ هنا يحمل نصَّ القدرة**، **وحارسٌ في الفحص
-- يرفض ما ليس في المعجم**، ومجهولُها يُمنَع عند القرار أيضاً.
--
-- # والافتراضُ منع
--
-- **دورٌ بلا صفوفٍ هنا لا يملك شيئاً** — **ولا يُبذَر شيءٌ لدورٍ لم
-- يُذكَر.**

CREATE TABLE IF NOT EXISTS role_capabilities (
    role_code       text NOT NULL REFERENCES roles(code) ON DELETE CASCADE,
    capability_code text NOT NULL,
    granted_at      timestamptz NOT NULL DEFAULT now(),
    granted_by      uuid REFERENCES users(id),
    PRIMARY KEY (role_code, capability_code)
);

-- **والقرارُ يبحث بالدور** — والصفوفُ لدورٍ واحدٍ قليلة.
CREATE INDEX IF NOT EXISTS role_capabilities_role_idx
    ON role_capabilities (role_code);

-- ══════════════════════════════════════════════════════════════════════
-- **بذرٌ انتقاليٌّ للأدوار القائمة — صريحٌ ومؤقّت**
-- ══════════════════════════════════════════════════════════════════════
--
-- **والمنصّةُ تعمل اليومَ بثلاثة أدوارٍ تدخل اللوحة**: `admin` و`ops`
-- و`finance`. **وترحيلٌ يُفرغ صلاحيّاتِهم يُوقف العمل.**
--
-- **فتُبذَر قدراتُهم كما هي مقيسةٌ من المسارات** — **لا أكثر.**
--
-- **و`admin` ليس `owner_super_admin`** (قرارُ المالك): **يأخذ ما
-- يحرسه اليومَ ولا يُرقّى.** **والأدوارُ التسعةُ وترحيلُ الحسابات
-- عملُ `ADG-2`.**
--
-- **وما ليس في هذه القائمة لا يملك شيئاً** — `customer` و`driver`
-- و`merchant` و`sales` لا صفَّ لها هنا.

-- `admin` — كلُّ ما يحرسه اليوم.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'admin', c FROM unnest(ARRAY[
    'orders.read', 'orders.intervene',
    'users.read', 'users.status.manage', 'roles.manage',
    'finance.read', 'finance.manage', 'payouts.decide',
    'merchants.manage', 'drivers.manage',
    'settings.general.manage', 'settings.financial.manage',
    'settings.security.manage',
    'content.manage', 'analytics.read'
]) AS c
ON CONFLICT DO NOTHING;

-- `ops` — تشغيلٌ بلا مالٍ ولا أدوارٍ ولا إعداداتٍ حسّاسة.
--
-- **وهو مقيسٌ من المسارات**: `RequireRoles("admin","ops","finance")`
-- على لوحة العمليّات، **و`RequireRoles("admin","ops")` على تدخّلات
-- الطلبات.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'ops', c FROM unnest(ARRAY[
    'orders.read', 'orders.intervene',
    'users.read',
    'merchants.manage', 'drivers.manage',
    'analytics.read'
]) AS c
ON CONFLICT DO NOTHING;

-- `finance` — مالٌ وتقاريرُ وإعداداتٌ ماليّة.
--
-- **ولا أدوارَ ولا إعداداتٍ أمنيّة** — **ولا حظرَ حسابات.**
INSERT INTO role_capabilities (role_code, capability_code)
SELECT 'finance', c FROM unnest(ARRAY[
    'orders.read',
    'users.read',
    'finance.read', 'finance.manage', 'payouts.decide',
    'analytics.read'
]) AS c
ON CONFLICT DO NOTHING;
