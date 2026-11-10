-- ══════════════════════════════════════════════════════════════════════
-- **النموذج ٢: البضاعة الراجعة بعد الاستلام** — خطّة المال ٢٠٢٦-١١-١٠ (الدفعة B5، القرار ٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- بطلب النموذج ٢ الكابتن دفع للمتجر من جيبه. إذا فشل الطلب أو انلغى بعد الاستلام:
--
--   goods_returned   البضاعة رجعت للمتجر، والمتجر بيرجّع للكابتن مصاريه (برّا الحسابات) — سجلّ بس.
--   held_by_driver   المتجر رفض يرجّع (البضاعة للمكتب) — «مصاري الكابتن عالقة»، والمكتب بيقرّر
--                    كل مرّة: المنصّة بتعوّض الكابتن، أو دين على الزبون.
--
-- والمنصّة ما بتسجّل خسارة بضاعة، وما في دين بضاعة على الكابتن.
ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_method_state_ck;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_method_state_ck
    CHECK ((method = 'wallet' AND state IN ('wallet_credited', 'wallet_reversed'))
        OR (method = 'cash' AND state IN ('cash_due', 'cash_paid', 'cash_reversed'))
        OR (method = 'cash' AND off_ledger AND state IN ('goods_returned', 'held_by_driver')));

-- **مصاري الكابتن العالقة وقرار المكتب عليها**:
--   stuck_goods_amount   شو دفع الكابتن للمتجر وما رجعله
--   stuck_goods_decision platform (المنصّة عوّضته من الخزينة) · customer (دين على الزبون — سجلّ)
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS stuck_goods_amount bigint NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS stuck_goods_decision text,
    ADD COLUMN IF NOT EXISTS stuck_goods_decided_at timestamptz,
    ADD COLUMN IF NOT EXISTS stuck_goods_decided_by uuid REFERENCES users(id);

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_stuck_goods_ck;
ALTER TABLE orders ADD CONSTRAINT orders_stuck_goods_ck
    CHECK (stuck_goods_amount >= 0
       AND (stuck_goods_decision IS NULL OR stuck_goods_decision IN ('platform', 'customer'))
       AND (stuck_goods_decision IS NULL OR stuck_goods_amount > 0));
