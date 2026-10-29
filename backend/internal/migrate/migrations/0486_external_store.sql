-- ══════════════════════════════════════════════════════════════════════
-- **«من متجر غير تابع لنا»** — قرارُ المالك ٢٠٢٦-١٠-٢٩
-- ══════════════════════════════════════════════════════════════════════
--
-- طلبٌ عاديٌّ نقديٌّ تعذّر من متجره فيُشترى من متجرٍ ليس في المنصّة. **فيقلبه
-- المكتبُ طلبَ مشترياتٍ** (`kind='custom'`, `custom_mode='purchase'`) — ومن بعدها
-- يمشي في مسار الطلب الخاصّ القائم كلِّه: الأجرةُ تثبّتها المنصّة، والسائقُ يوثّق
-- ثمنَ البضاعة، والزبونُ يؤكّد. **ولا منطقَ ماليَّ جديد.**
--
--   ext_store_merchant_id  **المتجرُ الذي كان الطلبُ منه** — للأثر وحدَه، لا يُسوّى له شيء.
--   ext_store_at           **متى قُلب.**
--   ext_store_by           **من قلبه.**
--   ext_store_reason       **لماذا** — السببُ إلزاميّ.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS ext_store_merchant_id uuid REFERENCES merchants(id),
    ADD COLUMN IF NOT EXISTS ext_store_at timestamptz,
    ADD COLUMN IF NOT EXISTS ext_store_by uuid REFERENCES users(id),
    ADD COLUMN IF NOT EXISTS ext_store_reason text;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_ext_store_ck') THEN
        -- **والقلبُ لا يقع إلّا على طلبٍ صار خاصّاً.**
        ALTER TABLE orders ADD CONSTRAINT orders_ext_store_ck CHECK (
            ext_store_at IS NULL OR kind = 'custom');
    END IF;
END $$;
