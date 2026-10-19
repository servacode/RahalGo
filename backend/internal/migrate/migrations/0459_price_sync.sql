-- ══════════════════════════════════════════════════════════════════════
-- **مزامنةُ الأسعار والكميّات من برنامج الأمين** (قرارُ المالك ٢٠٢٦-١٠-١٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- المولاتُ تحاسب بالأمين — **والسعرُ والكميّةُ هناك أصلاً.** فتُرفع قائمتُه ملفّاً
-- (Excel/CSV) من لوحة المتجر، **أو يبعثها «ساعي» على كمبيوتر المول** برمزٍ خاصّ.
--
--   menu_items.external_code   رقمُ المادّة في الأمين — به يُطابَق الصنف أوّلاً
--   menu_items.barcode         الباركود — ثانياً
--   merchants.price_sync_mapping     أيُّ عمودٍ في ملفّه رقمُ المادّة والسعرُ والكميّة
--   merchants.price_sync_token_hash  رمزُ الساعي — يُحفَظ مُجزَّأً ولا يُقرأ ثانيةً
--   price_sync_runs            سجلُّ كلِّ مزامنة — «آخر مزامنة» وما تغيّر فيها
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS external_code text;
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS barcode text;
CREATE UNIQUE INDEX IF NOT EXISTS menu_items_external_code_uq
    ON menu_items (merchant_id, external_code) WHERE external_code IS NOT NULL AND external_code <> '';
CREATE INDEX IF NOT EXISTS menu_items_barcode_idx
    ON menu_items (merchant_id, barcode) WHERE barcode IS NOT NULL AND barcode <> '';

ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_mapping jsonb;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS price_sync_token_hash text;

CREATE TABLE IF NOT EXISTS price_sync_runs (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    source      text NOT NULL CHECK (source IN ('file', 'agent', 'api')),
    report      jsonb NOT NULL,
    created_by  uuid REFERENCES users (id),
    created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS price_sync_runs_merchant_idx ON price_sync_runs (merchant_id, created_at DESC);
