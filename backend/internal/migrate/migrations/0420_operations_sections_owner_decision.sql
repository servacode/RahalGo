-- ══════════════════════════════════════════════════════════════════════
-- **أقسامُ موظّف العمليّات — قرارُ المالك 2026-10-05**
-- ══════════════════════════════════════════════════════════════════════
--
-- يرى `operations` تسعةَ أقسامٍ لا غير: الطلبات · سجلّ الطلبات · السوق ·
-- الشكاوى والتقييمات · الطوارئ · طلبات الانضمام · خريطة العمليّات ·
-- طلبات التوسّع · مراقبة التشغيل. **ولا الحسابات ولا التقارير** — هما لمدير
-- المنصّة وحدَه. **ولا شيءَ من المال والأدوار والإعدادات والعروض والحملات
-- والسجلّ.**
--
-- # 1 · السوقُ بقدرته (`market.manage`)
--
-- كانت أقسامُ السوق وأصنافُه وتصنيفاتُه بقدرة `content.manage` — **ومنحُها
-- للعمليّات يفتح لها العروضَ واللافتاتِ والحملاتِ والبثّ.** فقدرةٌ بعينها،
-- **تُمنح لكلّ دورٍ يملك المحتوى فلا يفقد أحدٌ السوق**، ثمّ للعمليّات.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT DISTINCT rc.role_code, 'market.manage'
  FROM role_capabilities rc
 WHERE rc.capability_code = 'content.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'market.manage'
  FROM roles r
 WHERE r.code = 'operations'
ON CONFLICT DO NOTHING;

-- # 2 · الشكاوى والتقييمات وطلبات الانضمام للعمليّات
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, c.cap
  FROM roles r
 CROSS JOIN (VALUES ('support.manage'), ('merchants.verify')) AS c(cap)
 WHERE r.code = 'operations'
ON CONFLICT DO NOTHING;

-- # 3 · الحساباتُ والتقاريرُ ليست للعمليّات
--
-- **ويبقى `users.contact.read`**: الاتّصالُ بالزبون والسائق من الطلب والطارئ.
-- **و«طلباتُ التوسّع» صارت بـ`orders.read`** فلا تلزمها `analytics.read`.
DELETE FROM role_capabilities
 WHERE role_code = 'operations'
   AND capability_code IN ('users.read', 'analytics.read');
