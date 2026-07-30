-- نظام الوسائط المركزي: كل صور المنصة (شعارات المتاجر، صور الأصناف، البانرات)
-- تمر عبر جدول واحد — الملفات تُعالَج في الخادم (تحقق نوع حقيقي، تصغير، مصغرة)
-- وتُخزَّن على القرص بأسماء عشوائية، والكيانات تشير إليها بمفتاح أجنبي.

CREATE TABLE media (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- merchant_logo | menu_item | banner
    kind       text NOT NULL CHECK (kind IN ('merchant_logo', 'menu_item', 'banner')),
    path       text NOT NULL,   -- مسار نسبي داخل مجلد الرفع (yyyy/mm/uuid.jpg)
    thumb_path text NOT NULL,   -- النسخة المصغرة
    width      int  NOT NULL,
    height     int  NOT NULL,
    bytes      bigint NOT NULL,
    created_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE merchants  ADD COLUMN logo_media_id  uuid REFERENCES media (id) ON DELETE SET NULL;
ALTER TABLE menu_items ADD COLUMN image_media_id uuid REFERENCES media (id) ON DELETE SET NULL;
ALTER TABLE menu_items DROP COLUMN image_url;  -- لم يُستخدم قط — يُستبدل بنظام الوسائط

-- البانرات: الصورة من نظام الوسائط حصراً — العمود النصي القديم يُستبدل
ALTER TABLE banners ADD COLUMN image_media_id uuid REFERENCES media (id) ON DELETE SET NULL;
ALTER TABLE banners DROP COLUMN image_url;
