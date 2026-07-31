-- أجر السائق — الطرف الأخير في الدورة المالية بلا قيدٍ لأجره.
--
-- صندوق السائق يسجّل **الدَّين عليه** (نقدٌ قبضه ويدين به للمنصة) ولا يسجّل
-- **الأجر له**. فكان رسم التوصيل يذهب كاملاً للمنصة، والسائق يعمل بلا أجرٍ في
-- الدفتر — ويُدفع له خارج النظام بذاكرةٍ وأوراق.
--
-- **والأجر يبقى خارج مبلغ الدَّين عمداً**: يُقيَّد في محفظته مستقلاً ويسلّم النقد
-- كاملاً للمكتب. خلطُهما (أن يسلّم المبلغ ناقصاً أجره) يجعل تسوية الصندوق غير
-- قابلة للمطابقة: لا يعود مجموع ما حصّله يساوي مجموع ما سلّمه.
ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_kind_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
    CHECK (kind IN (
        'topup',            -- شحن رصيد
        'order_payment',    -- دفع طلب من المحفظة
        'refund',           -- استرجاع طلب
        'compensation',     -- تعويض من المنصة
        'commission',       -- عمولة مندوب
        'merchant_earning', -- مستحقّ متجر عن طلب مُسلَّم
        'driver_earning',   -- أجر سائق عن طلب مُسلَّم
        'payout',           -- صرف رصيد
        'adjustment'        -- تسوية إدارية
    ));

-- نمط حساب الأجر وقيمته — يُبدَّل من الإعدادات بلا نشر.
--   percent → نسبة مئوية من رسم التوصيل
--   fixed   → مبلغ مقطوع لكل طلب مُسلَّم
-- ويبقى المجال مفتوحاً لنمط ثالث حسب المسافة حين تتوفّر مواقع حيّة للسائقين.
INSERT INTO app_settings (key, value) VALUES
    ('drivers.share_mode',  '"percent"'::jsonb),
    ('drivers.share_value', '70'::jsonb)
ON CONFLICT (key) DO NOTHING;
