-- ══════════════════════════════════════════════════════════════════════
-- **رئيسيّةُ مدير المنصّة — وشريطُ الطوارئ بزرّ «استلمتها»**
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الرئيسيّة لمدير المنصّة وحدَه» · «لازم أفهم
--  كلّ شي عم يصير بالمنصّة لأصغر تفصيل بدون ما أتنقّل بين الصفحات».)
--
-- # ١ · قدرةٌ لا اسمُ دور
--
-- **الرئيسيّةُ تعرض المالَ كلَّه** — المبيعاتِ وصافيَ المنصّة ورصيدَ الخزينة.
-- **فلا تُفتح بـ`analytics.read`** (يملكها موظّفُ العمليّات وليس طرفاً في
-- المال). **وقدرةٌ جديدةٌ تُمنح للأدمن وللمالك الأعلى وحدَهما** — ومن أراد
-- غداً أن يمنحها لدورٍ آخر منحها من اللوحة بلا سطرِ شيفرة.
INSERT INTO role_capabilities (role_code, capability_code) VALUES
    ('admin',             'platform.overview'),
    ('owner_super_admin', 'platform.overview')
ON CONFLICT DO NOTHING;

-- # ٢ · «استلمتها» — الطارئُ يبقى أحمرَ حتّى يستلمه إنسان
--
-- **الاستلامُ غيرُ الإغلاق**: من ضغط «استلمتها» قال «أنا عليه»، **والطارئُ
-- يبقى مفتوحاً حتّى يُحلّ** (`resolved`). **والشريطُ الأحمرُ أعلى كلّ صفحةٍ
-- يعرض ما لم يُستلَم بعد.**
ALTER TABLE driver_emergencies
    ADD COLUMN IF NOT EXISTS acknowledged_at timestamptz,
    ADD COLUMN IF NOT EXISTS acknowledged_by uuid REFERENCES users(id) ON DELETE SET NULL;

-- **والإغلاقُ الطارئُ للمتجر طارئٌ كذلك** — يُستلَم مرّةً لكلّ إغلاق.
-- `emergency_closed_at` **متى أُغلق**، و`emergency_ack_at` **متى استُلم** —
-- ويُصفَّر الاستلامُ عند كلّ إغلاقٍ جديد.
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS emergency_closed_at timestamptz,
    ADD COLUMN IF NOT EXISTS emergency_ack_at timestamptz,
    ADD COLUMN IF NOT EXISTS emergency_ack_by uuid REFERENCES users(id) ON DELETE SET NULL;

-- **والمغلقُ الآن يُؤرَّخ بآخر تعديلٍ له** — ولا يُعدّ مستلَماً: يظهر في الشريط
-- مرّةً حتّى يستلمه أحد.
UPDATE merchants SET emergency_closed_at = updated_at
 WHERE emergency_closed AND emergency_closed_at IS NULL;
