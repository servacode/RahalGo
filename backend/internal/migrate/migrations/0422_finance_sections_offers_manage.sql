-- ══════════════════════════════════════════════════════════════════════
-- **أقسامُ موظّف الماليّة — قرارُ المالك 2026-10-05**
-- ══════════════════════════════════════════════════════════════════════
--
-- يرى `finance` خمسةَ أقسامٍ لا غير: الخزينة · الديون · الخسائر والنزاعات ·
-- الأهداف والمكافآت · العروض والخصومات. **ولا الطلبات ولا سجلّ الطلبات ولا
-- الحسابات** — **ولا التقارير ولا سجلّ الأحداث ولا الإعدادات.**
-- (وصفحاتُ المال التي تُفتح من الخزينة — النقد · السحوبات · الأرباح ·
-- المصروفات · التعويضات · محفظتي — جزءٌ من قسم الخزينة.)
--
-- # 1 · العروضُ والخصوماتُ بقدرتها (`offers.manage`)
--
-- كانت الأكوادُ والخصوماتُ والدعواتُ بقدرة `content.manage` — **ومنحُها
-- للماليّة يفتح لها اللافتاتِ والحملاتِ والبثَّ وصورَ المنصّة.** فقدرةٌ بعينها،
-- **تُمنح لكلّ دورٍ يملك المحتوى فلا يفقد أحدٌ العروض**، ثمّ للماليّة.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT DISTINCT rc.role_code, 'offers.manage'
  FROM role_capabilities rc
 WHERE rc.capability_code = 'content.manage'
ON CONFLICT DO NOTHING;

INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'offers.manage'
  FROM roles r
 WHERE r.code = 'finance'
ON CONFLICT DO NOTHING;

-- # 2 · ما ليس من الأقسام الخمسة يُنزع من الماليّة
--
--   orders.read                 الطلبات · سجلّ الطلبات · الخريطة · التوسّع · المراقبة
--   users.read                  الحسابات
--   analytics.read              التقارير
--   audit.read                  سجلّ الأحداث
--   settings.financial.manage   الإعدادات — **ومنها سقوفُ التعويض والمحفظة اليدويّة:
--                               من يُقيَّد بالسقف لا يرفعه**؛ تبقى للمالك ومدير المنصّة.
--
-- **ويبقى**: finance.read · finance.manage · finance.export · payouts.decide ·
-- disputes.manage · settings.read · offers.manage.
DELETE FROM role_capabilities
 WHERE role_code = 'finance'
   AND capability_code IN ('orders.read', 'users.read', 'analytics.read',
                           'audit.read', 'settings.financial.manage');
