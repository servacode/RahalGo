-- **مرجعُ النزاع ينتقل إلى جدول الإنذارات الموحَّد.**
--
-- (وُحِّدت الإنذاراتُ ٢٠٢٦-٠٨-٠٩.)
--
-- **`disputes.warning_id` كان يشير إلى `merchant_warnings`** — والشيفرةُ صارت
-- تكتب في `warnings`، **فيُردّ كلُّ نزاعٍ يُفتح على متجرٍ رفض التسليم** بخرق
-- مفتاحٍ أجنبيّ.
--
-- **وأمسكته مجموعةُ الاختبارات لا القراءة**: خمسةُ اختباراتٍ سقطت دفعةً واحدة.
-- **ولولاها لَانكسر أوّلُ إقفالِ طلبٍ رفضه متجر** في الإنتاج.
--
-- # وملفٌّ جديدٌ لا تعديلُ السابق
--
-- **الهجرةُ التي وُسمت مطبَّقةً لا تُعاد** — فتعديلُها يمرّ على قاعدةٍ جديدةٍ
-- ولا يمسّ القائمة. **وقاعدتان تفترقان بصمتٍ أخطرُ من هجرةٍ تسقط.**

-- **والقديمُ يُنقل بمعرّفه** — فيبقى النزاعُ موصولاً بإنذاره.
--
-- **وهجرةُ ٠٠٨٨ نقلت بمعرّفاتٍ جديدة** (كانت تُنشئ صفوفاً لا تَنسخها)، **فلا
-- يجد النزاعُ إنذارَه.** فتُعاد هنا بالمعرّف نفسِه لِما له نزاع.
INSERT INTO warnings (id, user_id, role_code, reason, note, order_id, issued_by, created_at)
SELECT w.id, m.owner_user_id, 'merchant', w.reason, COALESCE(w.note, ''),
       w.order_id, w.issued_by, w.created_at
FROM merchant_warnings w
JOIN merchants m ON m.id = w.merchant_id
WHERE m.owner_user_id IS NOT NULL
  AND EXISTS (SELECT 1 FROM disputes d WHERE d.warning_id = w.id)
ON CONFLICT DO NOTHING;

-- **وما لم يُنقَل يُفرَّغ مرجعُه ولا يُحذف نزاعُه.**
--
-- **والنزاعُ مالٌ يُسوّى** — وإنذارٌ ضائعٌ لا يُسقط مطالبةً قائمة. **ومرجعٌ
-- معلَّقٌ يمنع القيدَ كلَّه**، فيُفرَّغ ويبقى النزاع.
UPDATE disputes SET warning_id = NULL
WHERE warning_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM warnings x WHERE x.id = disputes.warning_id);

ALTER TABLE disputes DROP CONSTRAINT IF EXISTS disputes_warning_id_fkey;
ALTER TABLE disputes
  ADD CONSTRAINT disputes_warning_id_fkey
  FOREIGN KEY (warning_id) REFERENCES warnings(id) ON DELETE SET NULL;
