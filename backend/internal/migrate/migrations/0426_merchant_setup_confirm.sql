-- ══════════════════════════════════════════════════════════════════════
-- **مَن أكّد طريقةَ مستحقّاته وقبولَ الاسترداد** (طلبُ المالك ٢٠٢٦-١٠-٠٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- `settlement_method` افتراضُه 'cash' و`accepts_returns` افتراضُه false —
-- **فلا يُعرف من اختار ممّن لم ينتبه.** فيُكتب وقتُ أوّل اختيارٍ صريحٍ ومَن
-- اختار ('merchant' من تطبيقه أو 'admin' من اللوحة بعد سؤاله).
-- **وفارغُها يعني: لم يُسأل بعد** — وتعرضه صفحةُ «إعداد المتاجر».
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS settlement_confirmed_at timestamptz,
    ADD COLUMN IF NOT EXISTS settlement_confirmed_by text
        CHECK (settlement_confirmed_by IN ('merchant', 'admin')),
    ADD COLUMN IF NOT EXISTS returns_confirmed_at timestamptz,
    ADD COLUMN IF NOT EXISTS returns_confirmed_by text
        CHECK (returns_confirmed_by IN ('merchant', 'admin'));
