-- ══════════════════════════════════════════════════════════════════════
-- **قراراتُ المالك ٢٠٢٦-١٠-١٦**
-- ══════════════════════════════════════════════════════════════════════
--
--   refund_rep_share   **سببُ دينٍ جديد** — استرجاعٌ بعد التسليم: المندوبُ يبقى بعمولته،
--                      والمكتبُ حمّل نصيبَه المتجرَ أو السائق (`orders.ChargeRefundRepShareTx`).
--   delivery.merchant_*  **مفاتيحُ أجرة «لدي توصيلة» وحدَها** — تُنسخ من قيم الطلب العاديّ
--                      المحفوظة **فلا يتبدّل رقمٌ يومَ النشر** حتّى يعدّلها المالك.
--   سائقٌ ومندوبٌ معاً  **ممنوعٌ في المحرّك** (`identity.ErrDriverSalesConflict`) — وما كان قائماً
--                      **يُبلَّغ عنه ولا يُكسَر به النشر.**

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
        'refund_rep_share'));

-- **والنسخُ لما حُفظ وحدَه** — والمفتاحُ غيرُ المحفوظ افتراضُه في الفهرس افتراضُ العاديّ نفسُه.
INSERT INTO app_settings (key, value)
SELECT m.new_key, s.value
  FROM (VALUES ('delivery.fee',         'delivery.merchant_base_fee'),
               ('delivery.by_distance', 'delivery.merchant_by_distance'),
               ('delivery.per_km',      'delivery.merchant_per_km'),
               ('delivery.max_fee',     'delivery.merchant_max_fee')) AS m(old_key, new_key)
  JOIN app_settings s ON s.key = m.old_key
ON CONFLICT (key) DO NOTHING;

-- **سائقٌ ومندوبٌ معاً — يُبلَّغ ولا يُسقط الهجرة.** المكتبُ يسحب أحدَ الدورين بيده.
DO $$
DECLARE
    r record;
    n int := 0;
BEGIN
    FOR r IN
        SELECT u.id::text AS id, COALESCE(u.full_name, '') AS name, COALESCE(u.phone, '') AS phone
          FROM users u
         WHERE EXISTS (SELECT 1 FROM user_roles d WHERE d.user_id = u.id AND d.role_code = 'driver')
           AND EXISTS (SELECT 1 FROM user_roles s WHERE s.user_id = u.id AND s.role_code = 'sales')
    LOOP
        n := n + 1;
        RAISE WARNING 'سائقٌ ومندوبٌ معاً: الحساب % (% · %)', r.id, r.name, r.phone;
    END LOOP;
    IF n > 0 THEN
        RAISE WARNING 'سائقٌ ومندوبٌ معاً: % حساباً يحمل الدورين — يُسحب أحدُهما باليد', n;
    END IF;
END $$;
