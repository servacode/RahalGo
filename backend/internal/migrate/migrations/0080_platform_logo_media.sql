-- **شعارُ المنصة نوعُ وسائطٍ جديد.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تنسَ إضافة هوية المنصة أيضاً — الاسم
--  واللوغو».)
--
-- # ولماذا هجرةٌ جديدةٌ لا تعديلُ قديمة
--
-- **قاعدةٌ مُلزمة**: ممنوعٌ تعديلُ هجرةٍ بعد دمجها — من نفّذ القديمةَ على
-- قاعدته لا يُعيد تنفيذَها، **فيبقى قيدُه القديم ويسقط الإدراجُ عنده وحدَه.**
--
-- # وحارسٌ أمسك الفرق
--
-- `TestKindsMatchDatabaseConstraint` يقارن `validKinds` في Go بقيد القاعدة.
-- **وأضفتُ النوعَ في Go ونسيتُ القاعدة** — فسقط البناء، **ولولاه لَمرّ الرفعُ
-- في التطوير وسقط في الإنتاج بخطأٍ من بوستغرس لا يفهمه أحد.**

ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;

ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text
  ])
);
