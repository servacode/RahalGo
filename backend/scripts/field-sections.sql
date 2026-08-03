-- **أقسامُ المنصة تُملأ من المتجر الواحد** — ولا صنفَ من خارجه.
--
-- # لماذا لزمت
--
-- `field-single-merchant.sql` أطفأ الأقسامَ كلَّها **لأنّ أصنافَها كانت من
-- متاجرَ أخرى** — وهي «الأصناف من خارج المتجر» التي أراد المالكُ إزالتها.
--
-- **لكنّ الزبونَ يتصفّح أقساماً لا متاجر.** فصفرُ أقسامٍ صفرُ متجر: الرئيسيةُ
-- تعرض فراغاً، **ولا سبيلَ إلى الطلب أصلاً.**
--
-- **فالصوابُ أن تُملأ من بيت الرقة لا أن تُطفأ**: قسمُ المنصة يبقى قسمَ منصة،
-- **والمصدرُ واحدٌ فلا يختلط شيء.**
--
-- # والمقابلةُ بأسماء أقسام المتجر
--
--	المشاوي على الفحم  ←  مشاوي
--	الساندويشات        ←  شاورما
--	الوجبات والصواني   ←  وجبات شعبية
--	المقبلات والسلطات  ←  وجبات شعبية
--	المشروبات          ←  مشروبات
--	الحلويات           ←  حلويات
--
-- **وما بقي من الأقسام يبقى مطفأً** — قسمٌ فارغٌ يُفتح فلا يُوجد فيه شيء
-- **أسوأُ من قسمٍ لا يظهر**: الأوّلُ يُقرأ عطباً والثاني لا يُقرأ شيئاً.

BEGIN;

UPDATE menu_items i SET platform_section_id = ps.id
FROM menu_sections ms, platform_sections ps
WHERE i.section_id = ms.id
  AND ps.name = CASE ms.name
      WHEN 'المشاوي على الفحم' THEN 'مشاوي'
      WHEN 'الساندويشات'       THEN 'شاورما'
      WHEN 'الوجبات والصواني'  THEN 'وجبات شعبية'
      WHEN 'المقبلات والسلطات' THEN 'وجبات شعبية'
      WHEN 'المشروبات'         THEN 'مشروبات'
      WHEN 'الحلويات'          THEN 'حلويات'
      END;

-- **ولا يُفعَّل إلّا ما فيه صنف.**
UPDATE platform_sections ps SET active = EXISTS (
    SELECT 1 FROM menu_items i WHERE i.platform_section_id = ps.id AND i.available);

COMMIT;

SELECT ps.name AS القسم,
       count(i.id) AS أصناف,
       ps.active AS فعّال
FROM platform_sections ps
LEFT JOIN menu_items i ON i.platform_section_id = ps.id
GROUP BY ps.id, ps.name, ps.sort_order, ps.active
ORDER BY ps.sort_order;
