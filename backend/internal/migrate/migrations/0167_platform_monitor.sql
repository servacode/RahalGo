-- ══════════════════════════════════════════════════════════════════════
--  موظّفُ مراقبةِ المنصّة — يرى كلَّ شيءٍ ولا يكتب شيئاً
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك ٢٠٢٦-٠٩-٣٠: «ممكن نضيف موظّف مراقب المنصّة، هو يشوف حالة
--  عمل المنصّة والموظّفين وكيف يتم سير الطلبات من كلّ النواحي… فتكون لديه
--  شاشة مراقبة يفهم منها كلّ شي».)
--
-- # ولماذا دورٌ قائمٌ بذاته
--
-- المراقبةُ كانت مبعثرةً على ثلاثة أدوارٍ لا يحمل أحدَها موظّف:
-- `observability` (قدرةٌ واحدة) و`analytics` (واحدة) و`s1_ops_viewer`.
-- **فصحّةُ المنصّة لا يبلغها أحد** — وهو `GAP-SEC-01` بعينه.
--
-- # وصفرُ كتابة — بقصد
--
-- **المراقبُ يرى ولا يفعل.** فلا `manage` ولا `intervene` ولا `decide`.
-- **ومراقبٌ يكتب ليس مراقباً** — يصير طرفاً في ما يراقبه.
--
-- **ولا يرى بياناتَ الاتّصال ولا الحقولَ الحسّاسة** — لا
-- `users.contact.read` ولا `users.sensitive.read`: **مراقبةُ السير لا
-- تحتاج هاتفَ زبونٍ ولا عنوانَه.** وأقلُّ ما يكفي أقلُّ ما يُخترَق.
--
-- # وما يراه، بأعيان حاجات المالك الثلاث
--
--   حالةُ عملِ المنصّة   ⇒  observability.read · analytics.read
--   حالةُ عملِ الموظّفين  ⇒  users.read · audit.read
--   سيرُ الطلبات        ⇒  orders.read · drivers.read · merchants.read

INSERT INTO roles (code, name_key) VALUES
    ('platform_monitor', 'roles.platform_monitor')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code) VALUES
    ('platform_monitor', 'observability.read'),
    ('platform_monitor', 'analytics.read'),
    ('platform_monitor', 'users.read'),
    ('platform_monitor', 'audit.read'),
    ('platform_monitor', 'orders.read'),
    ('platform_monitor', 'drivers.read'),
    ('platform_monitor', 'merchants.read')
ON CONFLICT DO NOTHING;

-- ══════════════════════════════════════════════════════════════════════
--  وصحّةُ المنصّة يبلغها مالكُها
-- ══════════════════════════════════════════════════════════════════════
--
-- **`observability.read` لا تمنحها أيُّ هجرةٍ في الشجرة** — والصفوفُ الحيّةُ
-- على التجهيز مكتوبةٌ بيد (الهجراتُ تبذر ١٠٦ صفَّ قدرات والتجهيز فيه ١٠٩).
-- **فبيئةٌ تُبنى من جديدٍ لا مراقبةَ فيها أصلاً.**
--
-- **ولا يكفي منحُها للمراقب وحدَه**: كلُّ دورٍ غير الزبون «أساسيّ»،
-- **ولا يحمل الحسابُ أساسيَّين** (`identity/one_role.go`) — **فالمالكُ
-- بدور `admin` لا يستطيع حملَ `platform_monitor` معه**، فيبقى لا يرى
-- صحّةَ منصّته. **وهو نصُّ `GAP-SEC-01`: «ولا المالك».**
--
-- **فتُمنَح للأدمن والمالك الأعلى أيضاً** — قراءةٌ محضةٌ لا تُخلّ بعزلٍ.
INSERT INTO role_capabilities (role_code, capability_code) VALUES
    ('admin',             'observability.read'),
    ('owner_super_admin', 'observability.read')
ON CONFLICT DO NOTHING;
