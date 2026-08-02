-- **نفقةُ المنصة نوعٌ قائمٌ بذاته — لا ربحٌ سالب.**
--
-- ## لماذا
--
-- تُقيَّد على الخزينة قيودٌ من مصدرين مختلفين:
--
--   1. **ربحُ الطلب** — يصدره المحرّك عن كلِّ تسوية، **ويُعاد حسابُه** كلَّما
--      تغيّر ما قُيّد للأطراف (الآن يقع على مرحلتين: استلامٌ ثم تسليم).
--   2. **نفقةٌ يقرّرها إنسان** — تعويضُ سائقٍ أو بضاعةٍ لم تُسترَدّ.
--
-- **ولو حملا نوعاً واحداً لَما أمكن حسابُ الفرق**: المحرّكُ يجمع ما قُيّد
-- ليعرف كم بقي عليه أن يقيّد، **فيجد تعويضاً قرّره موظّفٌ فيحسبه من عمله
-- ويُصحّحه** — فيمحو تعويضاً وقع فعلاً.
--
-- **وقيدٌ يُصحّح قيداً لا يخصّه أخطرُ من قيدٍ ناقص**: الناقصُ يظهر في مراجعة،
-- **والمصحَّحُ يبدو صحيحاً.**
ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_kind_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
    CHECK (kind = ANY (ARRAY[
        'topup', 'order_payment', 'refund', 'compensation', 'commission',
        'merchant_earning', 'driver_earning', 'payout', 'adjustment',
        'platform_profit',   -- نصيبُ الخزينة من طلب — يحسبه المحرّك ويُصحّحه
        'platform_expense'   -- ما خرج منها بقرار إنسان — لا يمسّه المحرّك
    ]));

-- ونقلُ ما سبق: التعويضاتُ التي قُيّدت قبل الفصل تُعاد إلى نوعها الصحيح،
-- **وإلّا حسبها المحرّكُ من عمله في أوّل تسويةٍ قادمة.**
UPDATE wallet_transactions SET kind = 'platform_expense'
WHERE kind = 'platform_profit' AND amount < 0
  AND note NOT LIKE 'عكسُ ربح%';
