-- ══════════════════════════════════════════════════════════════════════
-- **صورةُ المشروب · والمشروباتُ للمطاعم وحدَها** — قرارا المالك ٢٠٢٦-١٠-٣٠
-- ══════════════════════════════════════════════════════════════════════
--
-- ١ · **صورةٌ اختياريّةٌ لكلّ مشروب** في قائمة المتجر، وتُنسخ مع المشروب إلى
--     خيار «مشروبات» في كلّ صنف — فيراها الزبونُ مصغّرةً بجانب اسمه.
--
-- ٢ · **قائمةُ المشروبات للمطاعم فقط.** والتصنيفُ بلا رمزٍ ثابت (اسمُه وحدَه)،
--     فصار له علمٌ `has_drinks` — **صحيحٌ للمطاعم وحدَها**، ويُغيَّر من القاعدة
--     لا من الاسم. ومجموعاتُ «مشروبات» المنسوخةُ في متاجرَ غيرِ المطاعم تُمحى
--     مرّةً هنا؛ وقائمتُها تبقى في `merchant_drinks` فلا يضيع ما كتبه أحد.

ALTER TABLE merchant_drinks
    ADD COLUMN IF NOT EXISTS image_media_id uuid REFERENCES media(id) ON DELETE SET NULL;

ALTER TABLE modifier_options
    ADD COLUMN IF NOT EXISTS image_media_id uuid REFERENCES media(id) ON DELETE SET NULL;

ALTER TABLE categories
    ADD COLUMN IF NOT EXISTS has_drinks boolean NOT NULL DEFAULT false;

UPDATE categories SET has_drinks = true WHERE btrim(name) = 'مطاعم';

DELETE FROM modifier_groups g
 USING menu_items i, merchants m
 WHERE g.item_id = i.id AND g.store_drinks
   AND m.id = i.merchant_id
   AND NOT EXISTS (SELECT 1 FROM categories c WHERE c.id = m.category_id AND c.has_drinks);
