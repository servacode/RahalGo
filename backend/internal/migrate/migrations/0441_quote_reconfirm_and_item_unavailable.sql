-- ══════════════════════════════════════════════════════════════════════
-- **تعديلُ سعر الخاصّ بعد الاستلام · والصنفُ غيرُ المتوفّر** (قرارا المالك ٢٠٢٦-١٠-١٥)
-- ══════════════════════════════════════════════════════════════════════
--
--   quote_rejected_at  **الزبونُ رفض السعرَ الجديد** — المكتبُ يراه ويقرّر.
--                      يُمحى بتعديلٍ جديدٍ من المكتب أو بتأكيد الزبون.
--
--   sources_fee        **رسمُ المصادر الإضافيّة المضمَّنُ في `delivery_fee`** — يُكتب
--                      عند الإنشاء، **فإن خرج متجرٌ كاملٌ من الطلب بصنفٍ غير متوفّر
--                      عُرف ما يُطرح.** والطلباتُ القديمةُ صفرٌ: لا يُطرح منها شيء.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS quote_rejected_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS sources_fee bigint NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_sources_fee_ck') THEN
        ALTER TABLE orders ADD CONSTRAINT orders_sources_fee_ck CHECK (sources_fee >= 0);
    END IF;
END $$;
