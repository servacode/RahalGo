-- ══════════════════════════════════════════════════════════════════════
-- **مشترياتٌ من متجرٍ تابعٍ إلنا** (قرارُ المالك ٢٠٢٦-١٠-١٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- بعضُ طلبات المشتريات تُشترى من متجرٍ بيننا وبينه اتّفاق. **فيربطه المكتبُ بالطلب،
-- ويرسل له الطلبَ على واتساب، ويأخذ عمولتَه**: السائقُ يدفع للمتجر الثمنَ ناقصَ العمولة.
--
--   partner_merchant_id         **المتجرُ التابعُ** — عمودٌ منفصلٌ لا `merchant_id`:
--                               مساراتٌ كثيرةٌ تفترض أنّ الطلبَ الخاصَّ بلا متجر
--                               (بوّابةُ المتجر · مرساةُ التوزيع · رسالةُ المتجر · settleMerchant).
--   partner_commission_percent  **نسبةُ العمولة ملقوطةً لحظةَ الربط** — نسبةُ المتجر إن خُصّ
--                               بها، وإلّا العامّة. **ولا يبدّلها تعديلٌ لاحقٌ في المتجر.**
--   partner_linked_at           **متى رُبط.**
--   custom_store_paid           **ما دفعه السائقُ للمتجر عن طلب المشتريات** — يُكتب عند التسليم:
--                               الثمنُ كاملاً بلا متجرٍ تابع، والثمنُ ناقصَ العمولة معه.
--                               **وNULL للطلبات القديمة** — قبض صندوقُ سائقها المجموعَ كلَّه،
--                               فتعرف حرّاسُ المال أيَّ القاعدتين تقيس.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS partner_merchant_id uuid REFERENCES merchants(id);
ALTER TABLE orders ADD COLUMN IF NOT EXISTS partner_commission_percent bigint;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS partner_linked_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS custom_store_paid bigint;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_partner_ck') THEN
        -- **والمتجرُ التابعُ للمشتريات وحدَها، ونسبتُه معه أو لا شيءَ منهما.**
        ALTER TABLE orders ADD CONSTRAINT orders_partner_ck CHECK (
            (partner_merchant_id IS NULL AND partner_commission_percent IS NULL)
            OR (partner_merchant_id IS NOT NULL AND kind = 'custom' AND custom_mode = 'purchase'
                AND partner_commission_percent BETWEEN 0 AND 100));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_custom_store_paid_ck') THEN
        ALTER TABLE orders ADD CONSTRAINT orders_custom_store_paid_ck CHECK (
            custom_store_paid IS NULL OR (custom_store_paid >= 0 AND kind = 'custom'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS orders_partner_merchant_idx ON orders (partner_merchant_id)
    WHERE partner_merchant_id IS NOT NULL;
