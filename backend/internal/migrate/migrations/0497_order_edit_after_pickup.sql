-- ══════════════════════════════════════════════════════════════════════
-- **تعديلُ الطلب العاديّ بعد الاستلام** — قرارُ المالك ٢٠٢٦-١١-٠٣
-- ══════════════════════════════════════════════════════════════════════
--
-- بعد الاستلام قُيّد حقُّ كلِّ متجرٍ في صفّ تسويته (`merchant_settlements`). **فتعديلُ
-- البنود بعده يحتاج قيدَ تصحيحٍ في المعاملة نفسِها** — ولا يُمسّ قيدٌ قديم ولا يُمحى.
--
--   ١ · `order_items.returned`  سطرٌ رجّعه الزبونُ بعد الاستلام والمنصّةُ تتحمّله
--                               (البند ٢): سعرُ الزبون صفرٌ وسعرُ المتجر كما هو —
--                               **فحقُّ المتجر وعمولتُه لا يتغيّران.**
--   ٢ · `merchant_settlements.adjusted_amount`  مجموعُ التصحيحات الموقَّعة — **و`amount`
--                               يبقى المبلغَ الأصليَّ لا يُمسّ**؛ الحقُّ الحاليّ = `amount +
--                               adjusted_amount`.
--   ٣ · `merchant_settlement_adjustments`  سطرٌ لكلّ تصحيح: فرقُه وقيودُه في الدفتر (بالأنواع
--                               نفسِها التي تقرؤها الخزينةُ والكشوف) ومن دفع الفرقَ ودينُ ما لم
--                               يكفِه رصيدُ المتجر — **فلا يُقرأ قيدُ التصحيح تكراراً.**
--   ٤ · سببُ دينٍ جديد `order_edit`، وقدرةُ `orders.edit_after_pickup` للمالك والأدمن (البند ٤).

-- ── ١ · سطرٌ رجّعه الزبون ─────────────────────────────────────────────
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS returned boolean NOT NULL DEFAULT false;

-- ── ٢ · مجموعُ التصحيحات على صفّ التسوية ──────────────────────────────
ALTER TABLE merchant_settlements
    ADD COLUMN IF NOT EXISTS adjusted_amount bigint NOT NULL DEFAULT 0;
ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_adjusted_ck;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_adjusted_ck
    CHECK (amount + adjusted_amount >= 0);

-- ── ٣ · سطورُ التصحيح ─────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS merchant_settlement_adjustments (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_id  uuid NOT NULL REFERENCES merchant_settlements (id) ON DELETE CASCADE,
    order_id       uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    -- **الفرقُ الموقَّع**: موجبٌ زيادةُ حقّ المتجر، وسالبٌ نقصُه.
    delta          bigint NOT NULL CHECK (delta <> 0),
    -- محفظة: قيدُ merchant_earning (± ما احتمله رصيدُه).
    earning_tx_id  bigint REFERENCES wallet_transactions (id),
    -- نقد: قيدُ merchant_cash_accrued بالفرق في الاحتباس.
    accrued_tx_id  bigint REFERENCES wallet_transactions (id),
    -- نقد: قيدُ merchant_cash_paid — السائقُ دفع الفرقَ (−) أو ردّه المتجرُ له (+).
    paid_tx_id     bigint REFERENCES wallet_transactions (id),
    paid_by        uuid REFERENCES users (id),
    paid_via       text CHECK (paid_via IS NULL OR paid_via IN ('driver')),
    -- محفظة: **ما لم يكفِه رصيدُ المتجر صار ديناً عليه** بسبب `order_edit`.
    debt           bigint NOT NULL DEFAULT 0 CHECK (debt >= 0),
    reason         text NOT NULL,
    actor_id       uuid REFERENCES users (id),
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS merchant_settlement_adjustments_order_idx
    ON merchant_settlement_adjustments (order_id);
CREATE INDEX IF NOT EXISTS merchant_settlement_adjustments_settlement_idx
    ON merchant_settlement_adjustments (settlement_id);

-- ── ٤ · سببُ الدين ────────────────────────────────────────────────────
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
        'delivery_failed_recipient',
        'refund_rep_share',
        'waived_rep_commission',
        'undelivered_order',
        'order_edit'));

-- ── ٥ · القدرة: للمالك والأدمن وحدَهما (كـ`finance.recompute`، هجرة 0182) ──
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r.code, 'orders.edit_after_pickup'
  FROM roles r
 WHERE r.code IN ('owner_super_admin', 'admin')
ON CONFLICT DO NOTHING;
