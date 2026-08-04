-- قواعدُ المال تُقرأ حيّةً — لا لقطةً تُنسخ مرّةً وتبقى.
--
-- # المسألة
--
-- عمولةُ المنصة كانت تُنسخ في عمود المتجر لحظةَ إنشائه:
--
--	COALESCE($11, (SELECT ... 'merchants.default_commission_percent'), 10)
--
-- **فتغييرُ المفتاح لا يمسّ متجراً قائماً.** يرفع المالكُ العمولةَ من ١٠ إلى
-- ١٥ ويظنّ أنّه رفعها على الجميع — **وهو لم يرفعها على أحد**، إنّما على من
-- سيأتي. ولا شيءَ في الشاشة يقول ذلك.
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «لازم كلُّ المشروع يأخذ الإعداداتِ هذه بحيث
-- تُطبَّق بشكلٍ حقيقيٍّ فوريٍّ عند أيّ تغيّر».)
--
-- # الحلّ: العمودُ يصير تجاوزاً لا نسخة
--
-- `NULL` تعني **«اتبع العامّ»**، ورقمٌ فيه يعني **«اتُّفق مع هذا المتجر على
-- غيره»**. وهي طبقةُ وراثة هامش الصنف نفسُها: صنفٌ ← قسمٌ ← عامّ.
--
-- **والفراغُ غيرُ الصفر**: الصفرُ قرارٌ («لا عمولةَ على هذا المتجر») والفراغُ
-- غيابُ قرار. **ومن خلط بينهما جعل كلَّ متجرٍ لم يُلمس بلا عمولة** — فتعمل
-- المنصةُ بلا دخلٍ ولا يظهر ذلك إلّا في آخر الشهر.
--
-- # ولماذا يُفرَّغ من ساوى العامّ وحدَه
--
-- متجرٌ رقمُه يساوي العامّ لم يُتّفق معه على شيء — **ورث الافتراضَ ساكتاً**،
-- فيُعاد إلى الوراثة. ومن رقمُه غيرُ العامّ اتُّفق معه فعلاً، **فيبقى** —
-- **وترحيلٌ يمحو اتفاقاً قائماً يُكتشف حين يشكو صاحبُه.**

ALTER TABLE merchants ALTER COLUMN commission_percent DROP NOT NULL;
ALTER TABLE merchants ALTER COLUMN commission_percent DROP DEFAULT;

UPDATE merchants
   SET commission_percent = NULL
 WHERE commission_percent = COALESCE(
         (SELECT (value #>> '{}')::int FROM app_settings
           WHERE key = 'merchants.default_commission_percent'), 10);

COMMENT ON COLUMN merchants.commission_percent IS
  'تجاوزُ عمولة هذا المتجر — وNULL تعني «اتبع merchants.commission_value»';

-- # ونقلُ المفاتيح: نمطٌ وقيمة لكلّ مالٍ يُشتقّ
--
-- «نسبةٌ أم مقطوع؟» سؤالٌ يُسأل عن الهامش وعن عمولة المتجر وعن عمولة المندوب.
-- **وكان يُجاب في الهامش وحدَه** — والآخران نسبةٌ لا ثالثَ لها، واسمُهما يقوله
-- (`_percent`).
--
-- **والقيمةُ تُنقل ولا تُخترع**: من ضبط عمولتَه على ١٥ يجدها ١٥.

INSERT INTO app_settings (key, value)
SELECT 'merchants.commission_value', value FROM app_settings
 WHERE key = 'merchants.default_commission_percent'
ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value)
SELECT 'sales.commission_value', value FROM app_settings
 WHERE key = 'sales.commission_percent'
ON CONFLICT (key) DO NOTHING;

-- **والقديمُ يُحذف لا يُترك.**
--
-- مفتاحٌ مهجورٌ في القاعدة يظهر في كلّ استعلامٍ يقرأ الجدول، **ويُضبط يوماً
-- ظنّاً أنّه الفاعل** — فيُغيَّر رقمٌ لا يقرؤه أحد ويبقى الأثرُ غائباً.
DELETE FROM app_settings
 WHERE key IN ('merchants.default_commission_percent', 'sales.commission_percent');
