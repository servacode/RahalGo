-- صورة البروفايل لكل حساب (ملاحظة مراجعة قسم الحسابات) — من نظام الوسائط.
ALTER TABLE users ADD COLUMN avatar_media_id uuid REFERENCES media (id) ON DELETE SET NULL;

-- توسيع أنواع الوسائط لتشمل صور الحسابات
ALTER TABLE media DROP CONSTRAINT media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check
    CHECK (kind IN ('merchant_logo', 'menu_item', 'banner', 'avatar'));
