-- **خلفيّةُ الموقع نوعُ وسائطٍ جديد.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «أريد رفع صورةٍ كخلفيّةٍ للموقع — هي الباك‌غراوند
--  للمشروع كامل».)
--
-- **ومستقلٌّ عن `auth_background`**: خلفيّةُ شاشة الدخول قد تكون صورةً هادئةً
-- وخلفيّةُ الموقع أخرى، **ونوعٌ واحدٌ للاثنين يجعل حذفَ إحداهما يبحث في صور
-- الأخرى.**
--
-- **ولا تُعدَّل هجرةٌ بعد دمجها**: من نفّذ القديمةَ لا يُعيدها، فيبقى قيدُه
-- القديمُ ويسقط الرفعُ عنده وحدَه. و`TestKindsMatchDatabaseConstraint` يمسك
-- الفرقَ بين `validKinds` وقيدِ القاعدة.

ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;

ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text
  ])
);
