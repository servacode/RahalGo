-- **الطلبُ الخاصّ — خدمةٌ للسائق تُوثَّق ولا تُحاسَب.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «السائق سوف يدفع من جيبه ويستردّه عند الاستلام…
--  المنصّة هي فقط من توثّق المبلغ… السعر والأجرة لن تدخل بالحسابات، لأنّها
--  خدمة للسائق فقط».)
--
-- # ما هو
--
-- **زبونٌ يريد شيئاً ليس في المنصّة**: طعامٌ من مطعمٍ لم يشترك، أو غرضٌ من
-- السوق. **فيكتب ما يريد**، وتوافق الإدارة، ويُسنَد سائقٌ كأيّ طلب.
--
-- **والفرقُ في المال وحدَه**: السائقُ يدفع من جيبه ويستردّ عند التسليم،
-- **والمنصّةُ لا تقبض ولا تدفع** — لا عمولةَ ولا مستحقَّ متجرٍ ولا أجرَ توصيل.
--
-- # ولماذا طلبٌ لا كيانٌ ثانٍ
--
-- **يرث التوزيعَ والدردشةَ ومراحلَ الطريق والشكاوى والتقييمَ بلا سطرٍ جديد.**
-- **وكيانٌ منفصلٌ يعني إعادةَ بناء ذلك كلِّه، ثمّ افتراقَه عنه** — يُصلَح شرطٌ
-- في الطلبات ويُنسى في الطلبات الخاصّة.
--
-- # وأعمدةُ التوثيق منفصلةٌ عن أعمدة المحاسبة
--
-- **ولا تُكتب في `subtotal` و`total` و`delivery_fee`** — تلك يقرؤها الدفترُ
-- والخزينةُ وعمولةُ المندوب وتقاريرُ الدخل. **ورقمٌ يُوثَّق في عمودٍ يُحاسَب
-- يصير مالاً للمنصة بلا أن يقرّر ذلك أحد.**
--
-- **والفصلُ هو الضمانة**: تبقى أعمدةُ المحاسبة أصفاراً في الطلب الخاصّ،
-- **فيمرّ على الدفتر كأنّه لم يكن** — وهو ما يجب.

-- ١ · **ونوعُ الطلب.**
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'standard';

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_kind_check;
ALTER TABLE orders
  ADD CONSTRAINT orders_kind_check CHECK (kind IN ('standard', 'custom'));

-- ٢ · **والمتجرُ يصير اختيارياً — للخاصّ وحدَه.**
--
-- **والشرطُ يحرس العاديّ**: طلبٌ عاديٌّ بلا متجرٍ خطأٌ صامت، **يمرّ فيسقط عند
-- أوّل قراءةٍ لاسم المتجر.**
ALTER TABLE orders ALTER COLUMN merchant_id DROP NOT NULL;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_standard_has_merchant;
ALTER TABLE orders
  ADD CONSTRAINT orders_standard_has_merchant
  CHECK (kind <> 'standard' OR merchant_id IS NOT NULL);

-- ٣ · **وما يطلبه الزبون بلفظه.**
--
-- **ولا أصنافَ ولا أسعار**: هو يصف ما يريد — «شاورما من مطعم الأصيل» أو
-- «دواءٌ من صيدليّة الحيّ». **والتفصيلُ يقع في المحادثة بعد الإسناد.**
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS custom_request text NOT NULL DEFAULT '';

-- ٤ · **وما اتّفقا عليه — يُوثَّق ولا يُحاسَب.**
--
-- **يكتبه السائقُ بعد أن يتّفق مع الزبون في المحادثة**: ثمنُ البضاعة، وأجرةُ
-- التوصيل التي رضياها.
--
-- **وهو حجّةٌ عند الخلاف**: من ادّعى أنّه دفع أكثر، أو أنّ الأجرة كانت أقلّ،
-- **يُرجَع إلى ما وُثّق ووقتِه.** ولولاه لبقيت كلمةٌ ضدّ كلمة.
ALTER TABLE orders
  ADD COLUMN IF NOT EXISTS custom_goods_amount bigint,
  ADD COLUMN IF NOT EXISTS custom_fee bigint,
  ADD COLUMN IF NOT EXISTS custom_agreed_at timestamptz;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_amounts_positive;
ALTER TABLE orders
  ADD CONSTRAINT orders_custom_amounts_positive
  CHECK ((custom_goods_amount IS NULL OR custom_goods_amount >= 0)
         AND (custom_fee IS NULL OR custom_fee >= 0));

CREATE INDEX IF NOT EXISTS orders_custom_idx
  ON orders (created_at DESC) WHERE kind = 'custom';
