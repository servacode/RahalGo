-- **خلفيّةُ شاشات الدخول نوعُ وسائطٍ جديد.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «بدّي أضيف باك‌غراوند خلف صفحات تسجيل الدخول
--  والحساب الجديد والاستعادة، بحيث يكون في خيار بالإعدادات أرفع الصورة
--  وأغيّرها إيمت ما بدّي».)
--
-- # ولماذا هجرةٌ جديدةٌ لا تعديلُ ٠٠٨٠
--
-- **قاعدةٌ مُلزمة**: ممنوعٌ تعديلُ هجرةٍ بعد دمجها — من نفّذ القديمةَ على
-- قاعدته لا يُعيد تنفيذَها، **فيبقى قيدُه القديم ويسقط الرفعُ عنده وحدَه.**
--
-- # وحارسٌ يمسك الفرق
--
-- `TestKindsMatchDatabaseConstraint` يقارن `validKinds` في Go بقيد القاعدة —
-- **فنوعٌ يُضاف في أحدهما دون الآخر يُسقط البناء** بدل أن يسقط الرفعُ في
-- الإنتاج بخطأٍ من بوستغرس لا يفهمه أحد.

ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;

ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text
  ])
);
