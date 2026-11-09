-- ══════════════════════════════════════════════════════════════════════
-- **ثمنُ البضاعة بالليرة أو بالدولار** — قرارُ المالك ٢٠٢٦-١١-٠٩
-- ══════════════════════════════════════════════════════════════════════
--
-- السائقُ يكتب ثمنَ البضاعة في المشتريات (والواتساب) **بالليرة أو بالدولار**، والزبونُ يرى الرقمَ
-- بعملته كما كُتب — **ولا تحويلَ للعرض ولا جمعَ للعملتين في رقمٍ واحد.**
--
--   custom_goods_usd  **ثمنُ بضاعة المشتريات بالدولار — بالسنتات** (١٢٫٥٠$ ⇒ ١٢٥٠). وNULL: ليرة.
--                     **ومعه `custom_goods_amount` صفرٌ أبداً** — فالمجموعُ والمستحقُّ النقديُّ وحجزُ
--                     المحفظة الأجرةُ وحدَها، **وحسابُ الليرة في الدفتر لا يُمسّ.**
--   goods_fx_rate     **سعرُ الدولار الملقوط** (ليرةٌ جديدةٌ للدولار) — سعرُ المتجر إن كان له، وإلّا
--                     «سعرُ الدولار المرجعيّ» في الإعدادات. **يُلقط لحظةَ تأكيد الزبون** (المشتريات)
--                     **أو لحظةَ كتابة الثمن** (الواتساب) — فتبديلُ السعر بعدها لا يمسّ طلباً قديماً.
--   goods_usd_syp     **ما يعادله الدولارُ بالليرة باللقطة** (مدوَّراً لأقرب ٥) — للمشتريات وحدَها:
--                     منه عمولةُ المتجر التابع، وسقفُ نقد الزبون، وسقفُ نقد السائق، وخسارةُ البضاعة.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS custom_goods_usd bigint;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS goods_fx_rate numeric(14, 2);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS goods_usd_syp bigint NOT NULL DEFAULT 0;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_goods_usd_ck;
ALTER TABLE orders ADD CONSTRAINT orders_custom_goods_usd_ck CHECK (
    custom_goods_usd IS NULL
    OR (custom_goods_usd > 0 AND kind = 'custom' AND custom_mode = 'purchase'
        AND COALESCE(custom_goods_amount, 0) = 0));

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_goods_usd_syp_ck;
ALTER TABLE orders ADD CONSTRAINT orders_goods_usd_syp_ck CHECK (
    goods_usd_syp >= 0 AND (goods_usd_syp = 0 OR custom_goods_usd IS NOT NULL));

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_goods_fx_rate_ck;
ALTER TABLE orders ADD CONSTRAINT orders_goods_fx_rate_ck CHECK (goods_fx_rate IS NULL OR goods_fx_rate > 0);
