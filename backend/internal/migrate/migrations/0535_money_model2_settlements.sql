-- ══════════════════════════════════════════════════════════════════════
-- **النموذج ٢: صفّ تسوية المتجر سجلّ بلا دفتر** — خطّة المال ٢٠٢٦-١١-١٠ (الدفعة B4)
-- ══════════════════════════════════════════════════════════════════════
--
-- بطلب النموذج ٢ (orders.money_model = 2، هجرة ٠٥٣٢) البضاعة برّا كلّ الحسابات: الكابتن
-- بيدفع للمتجر من جيبه عند الاستلام وبيقبض من الزبون عند التسليم. **وصفّ
-- merchant_settlements بينكتب للسجلّ بس** («الكابتن دفع للمتجر X») — بلا قيد احتباس ولا
-- قيد دفع (accrued_tx_id / paid_tx_id فاضيين).
--
--   off_ledger       صفّ سجلّ بلا دفتر (النموذج ٢) — الحرّاس القديمة (FI-14) بتستثنيه.
--
-- **وصفّ عكس بلا قيد** (اقتطاع دين متجر قديم من صفّ سجلّ) بيصير مسموح لصفّ off_ledger
-- وحده — tx_id بيصير يقبل فراغ، والحارس FI-14.c بيضل يطابق المجموع.
ALTER TABLE merchant_settlements
    ADD COLUMN IF NOT EXISTS off_ledger boolean NOT NULL DEFAULT false;

-- **صفّ السجلّ ما إلو قيود**: لا مستحقّ ولا احتباس ولا دفع.
ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_off_ledger_ck;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_off_ledger_ck
    CHECK (NOT off_ledger
        OR (method = 'cash' AND earning_tx_id IS NULL AND accrued_tx_id IS NULL
            AND paid_tx_id IS NULL AND waived_accrued_tx_id IS NULL AND waived_paid_tx_id IS NULL));

-- **والعمولة الممحيّة على صفّ سجلّ ما إلها قيود** — الكابتن سطره بصندوقه وبس.
ALTER TABLE merchant_settlements DROP CONSTRAINT IF EXISTS merchant_settlements_commission_waived_check;
ALTER TABLE merchant_settlements ADD CONSTRAINT merchant_settlements_commission_waived_check
    CHECK (commission_waived >= 0
       AND (off_ledger
            OR ((commission_waived = 0) = (waived_accrued_tx_id IS NULL)
                AND (commission_waived = 0) = (waived_paid_tx_id IS NULL)))
       AND (commission_waived = 0 OR (method = 'cash' AND paid_via = 'driver')));

ALTER TABLE merchant_settlement_reversals ALTER COLUMN tx_id DROP NOT NULL;
