-- ══════════════════════════════════════════════════════════════════════
-- **نسبةُ العمولة التي سُوّي بها المتجر — في صفّ تسويته** (خلل PD-1، ٢٠٢٦-١١-٠٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- يُسوّى المتجرُ عند الاستلام (`settleMerchant`) **ثمّ يُنادى ثانيةً عند التسليم احتياطاً**،
-- فيُعاد حسابُ حقّه ويُطابَق مع الصفّ القائم (`validateSettlement`). **وكانت النسبةُ تُقرأ حيّةً**
-- من `merchants.commission_percent` في المرّتين — فمندوبٌ أو أدمنٌ بدّلها بين الاستلام والتسليم
-- يجعل المبلغَ المُعاد غيرَ المقيَّد، **فيسقط التسليمُ بـ XS-1 ويعلق السائقُ عند «سلّمت».**
--
-- **فتُحفظ النسبةُ في صفّ التسوية لحظةَ كتابته**، ويقرؤها `merchantShares` لمتجرٍ له صفّ بدل
-- الحيّة: التسليمُ يُطابَق بنسبة الاستلام، و`platform_commission` يتبعها، وتصحيحُ تعديل الطلب بعد
-- الاستلام يُقيّد فرقَ البنود وحدَه لا فرقَ النسبة. **والنسبةُ الجديدة تسري على كلّ طلبٍ لم يُستلم
-- من المتجر بعد.**
--
-- **وفراغُها صفٌّ أقدمُ من هذه الهجرة** — يبقى على حسبة ما قبلها.
ALTER TABLE merchant_settlements ADD COLUMN IF NOT EXISTS commission_percent numeric(5,2);

COMMENT ON COLUMN merchant_settlements.commission_percent IS
    'نسبةُ عمولة المنصّة التي سُوّي بها المتجرُ لحظةَ كتابة الصفّ بمنزلتين — NULL: صفٌّ أقدمُ من 0506';

-- ══════════════════════════════════════════════════════════════════════
-- **وصفوفُ ما قبلها في طلبٍ لم يُغلق تُعبّأ** — وإلّا بقي طلبٌ استُلم قبل النشر على الحيّة،
-- **فتبديلُ نسبته قبل تسليمه يُسقط «سلّمت» بـ XS-1 كما كان.**
-- ══════════════════════════════════════════════════════════════════════
--
-- **والنسبةُ تُكتب حيث تُعيد مبلغَ الصفّ بعينه وحدَه** — بحسبة `merchantShares` حرفاً: سعرُ شراء
-- بنود المتجر، ناقصَ عمولتها بالنسبة الحاليّة (تجاوزُ المتجر ثمّ لقطةُ الطلب) مقرّبةً لأقرب ليرة
-- والنصفُ يصعد (`pricing.Amount.Of`)، يساوي `amount + adjusted_amount`. **فصفٌّ بُدّلت نسبةُ متجره
-- بين الاستلام والنشر لا تطابقه الحاليّة فيبقى فارغاً** — لا تُثبَّت عليه نسبةٌ لم يُسوَّ بها، ويبقى
-- علاجُه القديم نافعاً (تُعاد النسبةُ فيُسلَّم).
--
-- **والطلبُ المُغلق لا يُمسّ** (مسلَّمٌ أو ملغًى أو مرفوضٌ أو فاشلٌ أو مُسترجَع) — لا تسويةَ تُنادى
-- عليه بعد. **ولا يُمسّ غيرُ هذا العمود** — لا مبلغٌ ولا قيد.
UPDATE merchant_settlements ms
   SET commission_percent = c.pct
  FROM (
        SELECT s.order_id, s.merchant_id,
               COALESCE(m.commission_percent, o.snap_merchant_commission_percent) AS pct,
               s.amount + s.adjusted_amount AS booked,
               (SELECT sum(oi.merchant_price * oi.qty)
                  FROM order_items oi
                 WHERE oi.order_id = s.order_id
                   AND COALESCE(oi.merchant_id, o.merchant_id) = s.merchant_id)::bigint AS cost
          FROM merchant_settlements s
          JOIN orders o ON o.id = s.order_id
          JOIN merchants m ON m.id = s.merchant_id
         WHERE s.commission_percent IS NULL
           AND o.status NOT IN ('delivered', 'cancelled', 'rejected', 'failed', 'refunded')
       ) c
 WHERE ms.order_id = c.order_id AND ms.merchant_id = c.merchant_id
   AND ms.commission_percent IS NULL
   AND c.pct IS NOT NULL AND c.cost > 0
   AND c.booked = c.cost - LEAST(
           (c.cost * round(c.pct * 100)::bigint + 5000) / 10000,
           c.cost);
