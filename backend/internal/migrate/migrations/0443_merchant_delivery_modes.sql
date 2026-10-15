-- ══════════════════════════════════════════════════════════════════════
-- **«لدي توصيلة» — قراراتُ المالك ٢٠٢٦-١٠-١٥**
-- ══════════════════════════════════════════════════════════════════════
--
--   delivery_mode   **«توصيل» أو «إحضار»** — `send` (الافتراض): السائقُ يستلم من المتجر
--                   ويسلّم المستلِم. `fetch`: يستلم من العنوان المكتوب ويرجع به إلى المتجر.
--                   **والعنوانُ والنقطةُ في أعمدتهما نفسِها** (`address_text`/`dropoff`) —
--                   والوضعُ وحدَه يقلب ترتيبَ الوقفتين.
--   collect_amount  **مبلغٌ يقبضه السائقُ من المستلِم للمتجر** — عرضٌ وحدَه: **لا يدخل
--                   الدفترَ ولا الصندوق.** السائقُ يدفعه للمتجر من جيبه ويأخذه من المستلِم
--                   خارجَ النظام. وNULL: لا شيء.
--   delivery_failed_recipient  **سببُ دينٍ جديدٍ على المتجر** — توصيلةٌ فشلت بذنب المستلِم
--                   والمستلِمُ هو الدافع، ومحفظةُ المتجر لا تكفي الأجرة.
--
-- **ولا يُمسّ `merchants.delivery_credit_limit` ولا مفتاحُ `delivery.merchant_debt_open`** —
-- يبقيان في القاعدة ولا يقرؤهما المحرّكُ بعد اليوم (لا دينَ على أجرة توصيلة).

ALTER TABLE orders ADD COLUMN IF NOT EXISTS delivery_mode text NOT NULL DEFAULT 'send';
ALTER TABLE orders ADD COLUMN IF NOT EXISTS collect_amount bigint;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_delivery_mode_ck') THEN
        -- **والإحضارُ للتوصيلة وحدَها** — والعاديُّ والخاصُّ `send` أبداً.
        ALTER TABLE orders ADD CONSTRAINT orders_delivery_mode_ck CHECK (
            delivery_mode = 'send'
            OR (delivery_mode = 'fetch' AND kind = 'merchant_delivery'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_collect_amount_ck') THEN
        ALTER TABLE orders ADD CONSTRAINT orders_collect_amount_ck CHECK (
            collect_amount IS NULL
            OR (collect_amount > 0 AND kind = 'merchant_delivery'));
    END IF;
END $$;

ALTER TABLE financial_obligations DROP CONSTRAINT IF EXISTS financial_obligations_cause_check;
ALTER TABLE financial_obligations ADD CONSTRAINT financial_obligations_cause_check
    CHECK (cause IN (
        'refund_merchant_earning',
        'refund_rep_commission',
        'returned_goods',
        'legacy_opening',
        'merchant_delivery_fee',
        'penalty_remainder',
        'dispute_remainder',
        'failed_order_driver',
        'delivery_failed_recipient'));
