-- ٠٠٥٦ · السعران — سعرُ الشراء وسعرُ البيع
--
-- # النموذجُ التجاريّ
--
--	سعرُ الشراء  ←  ما وضعه المتجر — **وهو ما يقبضه**
--	سعرُ البيع   ←  ما تضعه المنصة — **وهو ما يدفعه الزبون**
--	الهامش      ←  الفرق — **وهو ربحُ المنصة**
--
-- # ولماذا عمودٌ جديدٌ لا إعادةُ تفسيرٍ للقائم
--
-- `menu_items.price` هو ما يدفعه الزبونُ اليوم، **وكلُّ ما بُني عليه يقرؤه
-- كذلك**: السلّة، والفاتورة، والتقارير، وطلباتُ الأمس. **وإعادةُ تفسيره سعرَ
-- شراءٍ تقلب معنى كلِّ صفٍّ قديمٍ في القاعدة** — فتصير فواتيرُ الشهر الماضي
-- تقرأ أرقاماً لم تُدفع.
--
-- **فالجديدُ يحمل المعنى الجديد**، و`price` يبقى على معناه: ما يدفعه الزبون.
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS merchant_price bigint;

-- **الترحيلُ بالسعر نفسه — والهامشُ صفرٌ حتى يُقرّره المالك.**
--
-- **وأيُّ افتراضٍ غيرِه يخترع ربحاً أو خسارةً لم تقع**: لو حُسب سعرُ الشراء
-- بطرح هامشٍ مفترضٍ من السعر القائم **لَظهر لكلّ متجرٍ ربحٌ لم نتّفق عليه**،
-- ولقُيّد للمنصة هامشٌ لم تكسبه.
UPDATE menu_items SET merchant_price = price WHERE merchant_price IS NULL;
ALTER TABLE menu_items ALTER COLUMN merchant_price SET NOT NULL;
ALTER TABLE menu_items ALTER COLUMN merchant_price SET DEFAULT 0;

-- **لقطةُ سعر الشراء لحظةَ الطلب.**
--
-- **وليست ترفاً**: يرفع المتجرُ سعرَه غداً **فتُعاد قراءةُ طلبات الأمس بتكلفةٍ
-- لم تقع** — فيبدو هامشُنا أصغرَ أو أكبرَ ممّا كان، **وتقريرُ ربحٍ يقرأ أسعارَ
-- اليوم على طلبات الأمس تقريرٌ يكذب بلا أن يخطئ أحد.**
--
-- والقائمُ يُملأ بسعر الوحدة نفسِه — **فهامشُ ما مضى صفر**، وهو الصدق: لم يكن
-- ثمّة هامشٌ حين وقعت.
ALTER TABLE order_items ADD COLUMN IF NOT EXISTS merchant_price bigint;
UPDATE order_items SET merchant_price = unit_price WHERE merchant_price IS NULL;
ALTER TABLE order_items ALTER COLUMN merchant_price SET NOT NULL;
ALTER TABLE order_items ALTER COLUMN merchant_price SET DEFAULT 0;

-- # الهامشُ يُورَث، ويُتجاوَز لمن يستحقّ
--
--	هامشُ الصنف     ←  إن وُضع
--	هامشُ التصنيف   ←  وإلّا
--	الهامشُ العام    ←  وإلّا
--
-- **ولا أحدَ يُسعّر ألفَ صنفٍ بيده.** والتجاوزُ `NULL` لا `0`: **الصفرُ قرارٌ
-- («لا هامشَ على هذا») والفراغُ غيابُ قرار** («اتبع ما فوقك») — ومن خلط بينهما
-- جعل كلَّ صنفٍ لم يُلمس بلا هامش.
ALTER TABLE menu_items  ADD COLUMN IF NOT EXISTS margin_override bigint;
ALTER TABLE categories  ADD COLUMN IF NOT EXISTS margin_override bigint;

-- **العمولةُ على سعر الشراء لا على سعر البيع.**
--
-- بيعةُ المتجر هي سعرُ شرائنا، **وعمولةٌ على سعرٍ لا يراه المتجرُ عمولةٌ لا
-- يفهمها** — يُقال له «٢٪» فيحسبها على رقمه، فإن حُسبت على رقمنا **وجد خصماً
-- لم يتوقّعه ولم يُخبَر به.**
INSERT INTO app_settings (key, value) VALUES
    ('pricing.margin_mode',  '"percent"'::jsonb),
    ('pricing.margin_value', '0'::jsonb),
    ('pricing.rounding',     '500'::jsonb),
    ('merchants.commission_percent_default', '2'::jsonb)
ON CONFLICT (key) DO NOTHING;
