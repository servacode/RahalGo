-- ══════════════════════════════════════════════════════════════════════
-- **«لدي توصيلة» تُكمَل في القاعدة** — ٢٠٢٦-١٠-٠١
-- ══════════════════════════════════════════════════════════════════════
--
-- # ما كان
--
-- **`0166` بنت أعمدةَ التوصيلة ولم تُرخِ قيدين قديمين** — فكلُّ توصيلةٍ
-- تسقط بـ٥٠٠ **ولم تُنشأ واحدةٌ على القاعدة قطّ**: حرّاسُها كانت حسبةَ
-- النسبة وحدَها. **وأمسكها أوّلُ حارسٍ يمشيها** (`qa/merchant_delivery_test.go`).
--
-- **وهو صنفُ `OFFER-EXP` بعينه**: ميزةٌ تُكتب في الشيفرة، وقيدٌ قديمٌ لا يُرخى.

-- ── ١ · التوصيلةُ بلا زبون ──────────────────────────────────────────────
--
-- **المستلِمُ ليس مستخدماً** (قرارُ المالك ٤) — و`customer_id` NOT NULL منذ
-- الجدول الأوّل. **ويبقى إلزاميّاً لكلّ نوعٍ غيرِها** — فلا يولد طلبُ سوقٍ
-- أو طلبٌ خاصٌّ بلا صاحب.
ALTER TABLE orders ALTER COLUMN customer_id DROP NOT NULL;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_customer_unless_delivery;
ALTER TABLE orders ADD CONSTRAINT orders_customer_unless_delivery
    CHECK (kind = 'merchant_delivery' OR customer_id IS NOT NULL);

-- ── ١-ب · «أنا نقداً» — دافعٌ ثالث ────────────────────────────────────
--
-- (نصُّ المالك ٢٠٢٦-١٠-٠١: «لازم في أنا نقدي من يدفع أجرة التوصيل».)
-- **المتجرُ يدفع الأجرةَ نقداً بيد السائق عند الاستلام** — لا محفظةَ ولا دين.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_merchant_delivery_shape;
ALTER TABLE orders ADD CONSTRAINT orders_merchant_delivery_shape CHECK (
    kind <> 'merchant_delivery' OR (
        merchant_id IS NOT NULL
        AND recipient_name IS NOT NULL AND length(btrim(recipient_name)) > 0
        AND recipient_phone IS NOT NULL AND length(btrim(recipient_phone)) > 0
        AND address_text IS NOT NULL AND length(btrim(address_text)) > 0
        AND fee_payer IN ('merchant', 'merchant_cash', 'recipient')
    )
);

-- ── ١-ج · نقطةُ التسليم اختياريّة ──────────────────────────────────────
--
-- (نصُّ المالك: «نقطة التسليم غير إجباريّة… لأنّه يمكن لا يملك عنوانَ
-- المستلِم على الخريطة».) **و`dropoff` NOT NULL تقرؤه سبعةُ مواضعَ رقماً** —
-- وإفراغُه يكسرها كلَّها. **فيُكتب فيه موقعُ المتجر ويُعلَّم «غيرُ معروف»**،
-- **فلا يُوجَّه السائقُ إلى نقطةٍ كاذبة**: يقرأ العنوانَ ويتّصل بالمستلِم.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS dropoff_known boolean NOT NULL DEFAULT true;

-- ── ٢ · دينُ أجرةِ التوصيلة سببٌ مقبول ──────────────────────────────────
--
-- **`chargeMerchantDelivery` يكتب دينَ المتجر بسبب `merchant_delivery_fee`**
-- حين لا تكفي محفظتُه — **وقيدُ `0130` لا يعرف إلّا أربعةَ أسباب.**
ALTER TABLE financial_obligations DROP CONSTRAINT IF EXISTS financial_obligations_cause_check;
ALTER TABLE financial_obligations ADD CONSTRAINT financial_obligations_cause_check
    CHECK (cause IN (
        'refund_merchant_earning',
        'refund_rep_commission',
        'returned_goods',
        'legacy_opening',
        'merchant_delivery_fee'));
