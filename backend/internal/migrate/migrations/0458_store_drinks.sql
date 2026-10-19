-- ══════════════════════════════════════════════════════════════════════
-- **قائمةُ مشروبات المتجر — تُكتب مرّةً وتنضاف لكلّ أصنافه** (قرارُ المالك ٢٠٢٦-١٠-١٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- صاحبُ المتجر يكتب مشروباتِه مرّةً بأسعارها — **بالليرة أو بالدولار** — فتُنسخ
-- إلى كلّ صنفٍ مجموعةَ «مشروبات» اختياريّة، **ويتبع الدولاريُّ سعرَ الصرف** كالأصناف.
--
--   merchant_drinks.price            بالليرة — يُحسب من الدولار إن وُجد
--   merchant_drinks.price_usd_cents  الدولارُ بسنته — وفراغُه: بالليرة
--   modifier_groups.store_drinks     المجموعةُ المنسوخةُ من القائمة —
--                                    **لا يراها محرّرُ الصنف ولا يمحوها حفظُه**
--
-- **والمنسوخةُ خياراتٌ عاديّة** — فالطلبُ يتحقّق منها ويسعّرها بالطريق القائم.
CREATE TABLE IF NOT EXISTS merchant_drinks (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id     uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    name            text NOT NULL,
    price           bigint NOT NULL DEFAULT 0 CHECK (price >= 0),
    price_usd_cents bigint CHECK (price_usd_cents >= 0),
    sort_order      int NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS merchant_drinks_merchant_idx ON merchant_drinks (merchant_id, sort_order);

ALTER TABLE modifier_groups ADD COLUMN IF NOT EXISTS store_drinks boolean NOT NULL DEFAULT false;
