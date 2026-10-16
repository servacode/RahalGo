-- ══════════════════════════════════════════════════════════════════════
-- **«إحضار»: المبلغُ الحقيقيُّ يكتبه السائق ويوافق عليه المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- **ما يكتبه المتجرُ عند الطلب تقدير** (`collect_amount`) — والسائقُ بمكان الاستلام
-- يكتب ما دفعه فعلاً، **والمتجرُ يوافق أو يرفض**، ولا «استلمت» قبل الموافقة.
--
-- **والنسخةُ تزيد مع كلّ مبلغٍ جديد** — كنسخة عرض الخاصّ: موافقةٌ على مبلغٍ قديمٍ تُردّ.
--
-- **وكلُّه عرضٌ خارجَ الدفتر والصندوق** — كالتقدير نفسِه، لا قيدَ ولا رصيد.

ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS fetch_amount bigint NULL,
    ADD COLUMN IF NOT EXISTS fetch_amount_version integer NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS fetch_amount_status text NULL,
    ADD COLUMN IF NOT EXISTS fetch_amount_at timestamptz NULL,
    ADD COLUMN IF NOT EXISTS fetch_amount_decided_at timestamptz NULL;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_fetch_amount_check;
ALTER TABLE orders ADD CONSTRAINT orders_fetch_amount_check
    CHECK (fetch_amount IS NULL OR fetch_amount >= 0);

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_fetch_amount_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_fetch_amount_status_check
    CHECK (fetch_amount_status IS NULL OR fetch_amount_status IN ('pending', 'approved', 'rejected'));
