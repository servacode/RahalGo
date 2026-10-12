-- ══════════════════════════════════════════════════════════════════════
-- **أسعارٌ بالدولار وسعرُ صرفٍ للمتجر** (طلبُ المالك ٢٠٢٦-١٠-١٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- «كل صنف يختار سعر بالدولار او بالسوري وبنفس الوقت بصفحة متجري في سعر الصرف
-- ع اساسها تتعدل الاسعار».
--
--   merchants.exchange_rate            ليراتٌ (جديدة) للدولار — يكتبه المتجر، **ولا سعرَ عامّ**
--   menu_items.price_usd_cents         سعرُ الصنف بالسنت إن كان دولارياً — وفراغُه: الصنفُ بالليرة
--   modifier_options.price_delta_usd_cents  فرقُ الإضافة بالسنت لصنفٍ دولاريّ
--
-- **والليرةُ تبقى الحقيقةَ المحفوظة**: `merchant_price` و`price_delta` يُكتبان
-- محسوبَين (لأقرب ٥ ليرات) عند الحفظ وعند تبديل سعر الصرف — **فالسوقُ والهامشُ
-- والعروضُ والطلباتُ والدفترُ تقرأ ما كانت تقرؤه ولا يمسّها الدولار.**
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS exchange_rate numeric(14, 4)
    CHECK (exchange_rate IS NULL OR exchange_rate > 0);
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS price_usd_cents bigint
    CHECK (price_usd_cents IS NULL OR price_usd_cents >= 0);
ALTER TABLE modifier_options ADD COLUMN IF NOT EXISTS price_delta_usd_cents bigint;
