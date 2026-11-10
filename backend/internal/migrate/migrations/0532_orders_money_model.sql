-- نموذج المال على الطلب (خطة المال 2026-11-10، الدفعة B1).
-- 1 = القديم: البضاعة تمرّ بالدفتر (احتباس، محفظة متجر، سلفة الدرج).
-- 2 = الجديد: البضاعة خارج كل حساب، والمنصة تلمس الأجرة والعمولة فقط.
-- يُختم عند الإنشاء من المفتاح finance.goods_outside (مطفأ افتراضاً)، ولا يتغيّر بعدها.
-- الافتراضي 1 فكل طلب قائم يبقى على قاعدته القديمة. ولا شيء يقرأ 2 في هذه الدفعة.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS money_model smallint NOT NULL DEFAULT 1;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'orders_money_model_ck') THEN
        ALTER TABLE orders ADD CONSTRAINT orders_money_model_ck CHECK (money_model IN (1, 2));
    END IF;
END $$;
