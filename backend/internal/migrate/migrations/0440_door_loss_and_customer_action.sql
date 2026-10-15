-- ══════════════════════════════════════════════════════════════════════
-- **المكتبُ يقرّر من يدفع عند إنهاء طلبٍ فاشل** (قرارُ المالك ٢٠٢٦-١٠-١٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- كان ذنبُ السائق عند الباب لا يحمّله شيئاً، **وذنبُ الزبون يمنعه من النقد آليّاً**
-- بلا كلمةٍ من المكتب. وصار المكتبُ يختار ساعةَ الإنهاء:
--
--   failed_order_driver  دينٌ على السائق بخسارة الطلب كلِّها أو بنسبةٍ منها
--                        — مصدرُه معرّفُ الطلب (`source_ref`)
--   customer_action      ما يقع على الزبون: منعُ النقد · حظرُ الحساب · لا شيء
--
-- **والطلباتُ القديمةُ بذنب الزبون تُكتب منعاً للنقد** — فالمنعُ القائمُ اليومَ يبقى
-- كما هو، **وما يفشل بعدها بلا اختيارٍ من المكتب لا يمنع أحداً.** (كتابةٌ لا افتراضُ
-- `NULL` في الاستعلام: طلبٌ يفشل غداً من بابٍ غيرِ الباب لا يُقرأ منعاً.)

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
        'failed_order_driver'));

ALTER TABLE orders ADD COLUMN IF NOT EXISTS customer_action text;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_customer_action_ck') THEN
        ALTER TABLE orders ADD CONSTRAINT orders_customer_action_ck
            CHECK (customer_action IN ('cash_ban', 'block', 'none'));
    END IF;
END $$;

UPDATE orders SET customer_action = 'cash_ban'
 WHERE fault = 'customer' AND customer_action IS NULL;
