-- ══════════════════════════════════════════════════════════════════════
-- **«تصحيح: الطلب انلغى»** — قرارُ المالك ٢٠٢٦-١٠-٢٨
-- ══════════════════════════════════════════════════════════════════════
--
-- السائقُ ضغط «تم التسليم» خطأً والطلبُ أُلغي فعلاً. **فيُصحّحه الأدمن**: يصير الطلبُ
-- ملغياً، ويُعكَس ما سجّله التسليمُ على السائق والمندوب، ويبقى للمتجر ما قبضه — كطلبٍ
-- انتهى بعد الاستلام بلا تسليم. **ولا يُمحى قيدٌ ولا يُعدَّل**: كلُّ تصحيحٍ قيدٌ جديد.
--
--   السائق   سطرٌ سالبٌ في صندوقه (`collection_reversed`) يقابل تحصيلَ الطلب — لم يقبض
--            من الزبون شيئاً. **ومرّةً واحدةً للطلب.**
--   الدَّين  ما لا يحتمله متاحُ السائق أو المندوب من الأجر أو العمولة دَينٌ عليه بسببٍ جديد
--            (`undelivered_order`) — ولا رصيدَ سالب.

-- ── ١ · صندوقُ السائق ─────────────────────────────────────────────────
ALTER TABLE driver_cash_entries DROP CONSTRAINT IF EXISTS driver_cash_entries_kind_check;
ALTER TABLE driver_cash_entries ADD CONSTRAINT driver_cash_entries_kind_check
    CHECK (kind IN ('order_collection', 'settlement', 'adjustment', 'commission_waived',
                    'recipient_cash_returned', 'collection_reversed'));

CREATE UNIQUE INDEX IF NOT EXISTS driver_cash_entries_collection_reversed_once
    ON driver_cash_entries (ref, driver_id) WHERE kind = 'collection_reversed';

-- ── ٢ · سببُ الدَّين ──────────────────────────────────────────────────
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
        'undelivered_order'));
