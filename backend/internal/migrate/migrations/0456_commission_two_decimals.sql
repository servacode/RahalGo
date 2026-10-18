-- ══════════════════════════════════════════════════════════════════════
-- **عمولةُ المنصّة بمنزلتين عشريّتين** (قرارُ المالك ٢٠٢٦-١٠-١٨: «١٫٥٪»)
-- ══════════════════════════════════════════════════════════════════════
--
-- كانت النسبةُ عدداً صحيحاً في ثلاثة أعمدة:
--
--   merchants.commission_percent        تجاوزُ المتجر (int، وفراغُه «اتبع العامّ»)
--   orders.snap_merchant_commission_percent  النسبةُ الملقوطةُ لحظةَ الطلب (int)
--   orders.partner_commission_percent   نسبةُ المتجر التابع في المشتريات (bigint)
--
-- **فصارت `numeric(5,2)`** — من ٠٫٠٠ إلى ١٠٠٫٠٠.
--
-- **والقيمُ الصحيحةُ تبقى كما هي**: ١٠ تصير ١٠٫٠٠ — **والحسبةُ في المحرّك
-- بأجزاء المئة وبالبتر نفسِه**، فلا يتبدّل قرشٌ لمتجرٍ على نسبةٍ صحيحة.
--
-- **والقيودُ القائمةُ تبقى**: `BETWEEN 0 AND 100` يصدق على العشريّ كما كان،
-- و«اللقطةُ كلٌّ أو لا شيء» يفحص الفراغَ لا النوع.
--
-- **ونسبةُ المندوب (`snap_rep_commission_percent`) باقيةٌ صحيحة** — ليست في
-- القرار.

ALTER TABLE merchants
    ALTER COLUMN commission_percent TYPE numeric(5,2) USING commission_percent::numeric(5,2);

ALTER TABLE orders
    ALTER COLUMN snap_merchant_commission_percent TYPE numeric(5,2)
        USING snap_merchant_commission_percent::numeric(5,2),
    ALTER COLUMN partner_commission_percent TYPE numeric(5,2)
        USING partner_commission_percent::numeric(5,2);

COMMENT ON COLUMN merchants.commission_percent IS
    'تجاوزُ عمولة المنصّة لهذا المتجر بمنزلتين (١٫٥ = ١٫٥٪) — NULL: اتبع merchants.commission_percent العامّ';
COMMENT ON COLUMN orders.snap_merchant_commission_percent IS
    'نسبةُ عمولة المنصّة الملقوطةُ لحظةَ الطلب بمنزلتين (XG-25)';
COMMENT ON COLUMN orders.partner_commission_percent IS
    'نسبةُ عمولة المتجر التابع الملقوطةُ لحظةَ الربط بمنزلتين';
