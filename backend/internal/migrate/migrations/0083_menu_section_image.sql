-- ٠٠٨٣ · صورةُ قسم المتجر — **وجهُ القائمة لصاحبها ولزبونه**
--
-- # المسألة
--
-- **قسمُ قائمةِ المتجر اسمٌ ولا شيءَ سواه**: «المقبّلات» · «المشاوي» ·
-- «المشروبات» — أسطرٌ متشابهةٌ في قائمةٍ واحدة. **وصاحبُ مطعمٍ بعشرة أقسامٍ
-- يمسح القائمةَ بعينه فلا يجد قسمَه إلّا بالقراءة.**
--
-- (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «أساساً القسم لازم يكون له صورةٌ واسم، وكلُّ
--  صنفٍ تفوت عليه تضيف المنتجاتِ بداخله… مثل الأصناف بلوحة الأدمن، يجب أن
--  تكون الطريقةُ مركزيّة».)
--
-- **وقسمُ المنصة نال هذا في ٠٠٦٦** — وهذا أخوه في المتجر: **الطريقةُ نفسُها
-- بالعمود نفسِه وبالقيد نفسِه**، فلا تُبنى ثانيةً بشكلٍ ثانٍ.
--
-- # ولماذا `ON DELETE SET NULL`
--
-- **حذفُ صورةٍ من الوسائط لا يحذف قسماً** — والقسمُ فيه أصنافٌ وطلباتٌ
-- تشير إليه. **فيبقى بلا صورةٍ كما كان قبل أن تُرفع**، ولا شيءَ يسقط.
--
-- # والنوعُ يُضاف إلى القيد في الهجرة نفسِها
--
-- **`TestKindsMatchDatabaseConstraint` يقارن `validKinds` في Go بقيد
-- القاعدة** — ومن أضاف في أحدهما ونسي الآخر **يُسقط البناء**، لا يُسلّم
-- نوعاً يردّه الخادمُ بخطأٍ لا يفهمه صاحبُ المطعم.

ALTER TABLE menu_sections
	ADD COLUMN IF NOT EXISTS image_media_id uuid REFERENCES media(id) ON DELETE SET NULL;

ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'menu_section'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text
  ])
);
