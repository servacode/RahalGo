-- ══════════════════════════════════════════════════════════════════════
-- **بلاغاتُ السائقين القديمة — تُنسب إلى من هي عليه**
-- ══════════════════════════════════════════════════════════════════════
--
-- (شكوى المالك ٢٠٢٦-٠٨-١٣: «أنا كسائقٍ قمتُ بالإبلاغ على زبونٍ ومتجر،
--  ولكنّ البلاغَ تمّ فهمُه على أنّه ضدّي».)
--
-- **كان `DriverReport` يُدرج التذكرةَ بلا `against_user_id`** — وفارغُه
-- في شاشة السمعة يعني «تُعرض للجميع»، **فيقرأ السائقُ بلاغَه هو في
-- «الشكاوى عليك».**
--
-- **وأُصلح الإدراجُ**، ويبقى ما فُتح قبله. **وصفٌّ قديمٌ لا يُصلَح يبقى
-- يتّهم صاحبَه** — والشاشةُ تُخفيه بشرطها الثاني، **لكنّ العمودَ يبقى
-- كاذباً لمن يقرأ القاعدة.**
--
-- **والجهةُ تُشتقّ من الرمز** — الأربعةُ التي تبدأ بـ`merchant_` على
-- صاحب المتجر، **والأربعةُ التي تبدأ بـ`customer_` على زبون الطلب.**
-- **و«سببٌ آخر» يبقى فارغاً**: لا تُخمَّن جهةٌ على أحد.
--
-- **ولا تُمسّ تذاكرُ الزبائن** — شرطُ `opened_by_customer = false`
-- و`created_by` سائقُ الطلب، **فلا يُبدَّل ما لم يُخطئ.**

UPDATE tickets t
SET against_user_id = m.owner_user_id
FROM orders o
JOIN merchants m ON m.id = o.merchant_id
WHERE t.order_id = o.id
  AND t.against_user_id IS NULL
  AND t.opened_by_customer = false
  AND t.created_by = o.driver_id
  AND t.reason LIKE 'merchant\_%';

UPDATE tickets t
SET against_user_id = o.customer_id
FROM orders o
WHERE t.order_id = o.id
  AND t.against_user_id IS NULL
  AND t.opened_by_customer = false
  AND t.created_by = o.driver_id
  AND t.reason LIKE 'customer\_%';
