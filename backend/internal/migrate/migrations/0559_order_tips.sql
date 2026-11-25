-- ══════════════════════════════════════════════════════════════════════
-- **إكراميةُ الكابتن** — قرارُ المالك ٢٠٢٦-١١-٢٥
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبونُ اللي بيدفع من محفظته بيقدر يزيد إكرامية (٥ / ١٠ / ٢٠ أو مبلغ تاني).
-- **بتنحجز من محفظته** (`tip_reserved`، متل حجز الطلب الخاصّ) **وبتنقيّد عند التسليم**
-- بثلاث سطور من نوع `tip` بمرجع الطلب: سالبٌ على الزبون، وموجبٌ للكابتن اللي سلّم،
-- ونسبةُ المنصّة (إن وُجدت) للخزينة — **مجموعها صفر**، فما بيتغيّر حساب الطلب ولا الخزينة.
-- والإلغاء بيفكّ الحجز.

-- ── ١ · نوعُ القيد ────────────────────────────────────────────────────
SELECT wallet_kinds_add('tip');

-- ── ٢ · أعمدةُ الطلب ──────────────────────────────────────────────────
--   tip_amount            الإكراميةُ اللي اختارها الزبون (صفر = بلا)
--   tip_reserved          المحجوزُ منها هلّق — بيصير صفر بالتسليم أو الإلغاء
--   tip_platform_percent  نسبةُ المنصّة لحظةَ الاختيار (لقطة، لا إعدادٌ حيّ)
--   tip_platform_cut      ما أخذته المنصّة فعلاً عند التسليم
--   tip_paid_at           لحظةُ قيدها للكابتن — مرّة وحدة
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_amount bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_reserved bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_platform_percent smallint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_platform_cut bigint NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS tip_paid_at timestamptz;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_tip_shape;
ALTER TABLE orders ADD CONSTRAINT orders_tip_shape CHECK (
    tip_amount >= 0
    AND tip_reserved >= 0 AND tip_reserved <= tip_amount
    AND tip_platform_percent BETWEEN 0 AND 100
    AND tip_platform_cut >= 0 AND tip_platform_cut <= tip_amount
    -- **والمحجوزُ والمدفوعُ ما بيجتمعوا** — إمّا لسّا محجوزة أو انقيّدت.
    AND NOT (tip_reserved > 0 AND tip_paid_at IS NOT NULL)
);
