-- ══════════════════════════════════════════════════════════════════════
-- **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦
-- ══════════════════════════════════════════════════════════════════════
--
-- أحياناً يرفض المتجرُ أن يخصم عمولتَنا، فيأخذ من السائق ثمنَ البضاعة كاملاً
-- عند الاستلام. **فالسائقُ لا يحمل العمولةَ أصلاً**، والمكتبُ يمحوها من الطلب:
--
--   السائق    سطرٌ في صندوقه بالسالب (`commission_waived`) — لم يعد يدين بها.
--   المتجر    قبض العمولةَ نقداً من السائق: قيدُ احتباسٍ موجبٌ ثمّ دفعٌ سالبٌ بمقدارها
--             (يتصافيان في الاحتباس)، **وصفُّ التسوية يحمل المقدارَ ورقمَي القيدين.**
--   المندوب   نصيبُه من العمولة يُعكَس — وما عجزت عنه محفظتُه دَينٌ بسببٍ جديد.
--   الطلب     العمولةُ صفر، ومقدارُ المحوِ ومن ومتى ولماذا.
--   الخزينة   تُعاد حسبتُها بالفرق — فتنقص بالعمولة وتسترجع نصيبَ المندوب.
--
-- **ولا يُمحى قيدٌ ولا يُعدَّل** — كلُّ تصحيحٍ قيدٌ جديدٌ بمرجع الطلب.

-- ── ١ · الطلب ─────────────────────────────────────────────────────────
ALTER TABLE orders ADD COLUMN IF NOT EXISTS commission_waived bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS commission_waived_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS commission_waived_by uuid REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS commission_waived_reason text;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_commission_waived_check;
ALTER TABLE orders ADD CONSTRAINT orders_commission_waived_check
    CHECK (commission_waived >= 0
           AND (commission_waived = 0) = (commission_waived_at IS NULL));

-- ── ٢ · صفُّ تسوية المتجر ─────────────────────────────────────────────
ALTER TABLE merchant_settlements ADD COLUMN IF NOT EXISTS commission_waived bigint NOT NULL DEFAULT 0;
ALTER TABLE merchant_settlements ADD COLUMN IF NOT EXISTS waived_accrued_tx_id bigint REFERENCES wallet_transactions(id);
ALTER TABLE merchant_settlements ADD COLUMN IF NOT EXISTS waived_paid_tx_id bigint REFERENCES wallet_transactions(id);

ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_commission_waived_check;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_commission_waived_check
    CHECK (commission_waived >= 0
           AND (commission_waived = 0) = (waived_accrued_tx_id IS NULL)
           AND (commission_waived = 0) = (waived_paid_tx_id IS NULL)
           AND (commission_waived = 0 OR (method = 'cash' AND paid_via = 'driver')));

-- ── ٣ · صندوقُ السائق ─────────────────────────────────────────────────
ALTER TABLE driver_cash_entries DROP CONSTRAINT IF EXISTS driver_cash_entries_kind_check;
ALTER TABLE driver_cash_entries ADD CONSTRAINT driver_cash_entries_kind_check
    CHECK (kind IN ('order_collection', 'settlement', 'adjustment', 'commission_waived'));

-- **ومرّةً واحدةً للطلب** — والقفلُ على صفّ الطلب يمنع التكرارَ قبله.
CREATE UNIQUE INDEX IF NOT EXISTS driver_cash_entries_waiver_once
    ON driver_cash_entries (ref) WHERE kind = 'commission_waived';

-- ── ٤ · دَينُ المندوب إن عجزت محفظتُه عن العكس ──────────────────────
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
        'waived_rep_commission'));
