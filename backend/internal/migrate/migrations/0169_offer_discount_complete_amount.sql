-- ══════════════════════════════════════════════════════════════════════
-- **خصمٌ مكتمل: نسبةٌ أو مبلغ — لا نسبةٌ وحدَها** — `OFFER-EXP`، ٢٠٢٦-٠٩-٣٠
-- ══════════════════════════════════════════════════════════════════════
--
-- # ما وقع
--
-- **`0168` أضافت `discount_amount` وقيدَ «واحدٌ منهما لا اثنان»**
-- (`offers_one_discount_kind`) — **ولم تُرخِ `offers_discount_complete`**
-- القديمَ من `0075`، **وهو يشترط `discount_percent IS NOT NULL`.**
--
-- **فكلُّ عرضٍ بمبلغٍ ثابتٍ رُفض في القاعدة منذ وُلدت الميزة** — **وردَّ
-- المحرّكُ ٥٠٠ فقال الهاتفُ «تعذّر الاتصال».** رآه المالك على التجهيز،
-- **وأمسكه `offers/expired_offer_test.go`** بإدراجٍ حقيقيٍّ لا بحسبةٍ في
-- الذاكرة.
--
-- # والقيدُ الجديد
--
-- **صنفٌ ومن يتحمّل، وخصمٌ بإحدى الطريقتين.** **و«لا اثنان» يبقى في
-- `offers_one_discount_kind`** — فالقيدان معاً: **واحدٌ بالضبط.**
--
-- **والصفوفُ القائمةُ كلُّها بالنسبة** — فتمرّ بلا تعديلِ صفّ.

ALTER TABLE offers DROP CONSTRAINT IF EXISTS offers_discount_complete;
ALTER TABLE offers ADD CONSTRAINT offers_discount_complete CHECK (
	menu_item_id IS NOT NULL AND
	borne_by IS NOT NULL AND
	(discount_percent IS NOT NULL OR discount_amount IS NOT NULL)
);
