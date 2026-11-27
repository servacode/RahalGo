-- ══════════════════════════════════════════════════════════════════════
-- **من يدفع أجرة أمانة البوت** (قرار المالك ٢٠٢٦-١١-٢٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- البوت يسأل «من سيدفع أجرة التوصيل؟ أنا / الشخص الآخر»، والدافع يدفع نقداً للكابتن حيث يقف:
-- الواقف عند نقطة الاستلام يدفع عند الاستلام، والواقف عند نقطة التسليم يدفع عند التسليم.
--
-- فالأمانةُ (`wa_amanah`) وحدَها يُسمح لها بـ`fee_payer = 'merchant_cash'` — وهو في «لدي توصيلة»
-- معنى «من يقف عند نقطة الاستلام يدفع نقداً عندها» (`cashAtPickup`)، والتسويةُ القائمة تقيّده في صندوق
-- الكابتن عند الاستلام ولا تقبضه ثانيةً عند التسليم. ولا متجرَ في طلب الواتساب (`merchant_id` فارغ)
-- فلا محفظةَ تُخصم ولا تُردّ. وما سوى الأمانة باقٍ `recipient` كما كان.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_whatsapp_shape;
ALTER TABLE orders ADD CONSTRAINT orders_whatsapp_shape CHECK (
    origin <> 'whatsapp' OR (
        kind = 'merchant_delivery'
        AND merchant_id IS NULL AND customer_id IS NULL
        AND (fee_payer = 'recipient' OR (wa_amanah AND fee_payer = 'merchant_cash'))
        AND delivery_mode = 'send'
        AND subtotal = 0 AND wallet_paid = 0
        AND length(btrim(COALESCE(pickup_name, ''))) > 0
    )
);
