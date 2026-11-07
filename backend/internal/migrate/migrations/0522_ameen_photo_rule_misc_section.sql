-- ٠٥٢٢ · صنف الأمين بلا صورة بيضلّ مخفي · إشعار «في أصناف بلا قسم» · قسم «منوّعات»
-- (قرارات المالك ٢٠٢٦-١١-٠٧)
--
-- ١. «الصنف بلا صورة يبقى مخفياً، وبس تنزل صورة يصير متوفر» — للأصناف اللي عملتها
--    مزامنة الأمين بس، والأصناف اللي عملها المتجر بإيده ما بتتغيّر:
--      menu_items.ameen_created  الصنف انعمل من مزامنة الأمين.
--      menu_items.ameen_qty      آخر كمية إجت من الأمين (فاضي = ما إجت كمية).
--      menu_items.photo_hold     الكمية موجودة بس الصنف مخفي لأنو ما إلو صورة —
--                                 أوّل صورة بتنزل بتخلّيه متوفر (إلّا إذا آخر كمية صفر).
--    الأصناف اللي انعملت من قبل: هي اللي مصدر قسمها pending/group/name (المزامنة بس
--    بتحطّهم هيك). والمخفي منهم بلا صورة بيصير photo_hold — والمزامنة الجاية بتصحّحه
--    إذا كميته صفر.
--
-- ٢. متى انبعت إشعار «في أصناف من الأمين بلا قسم» للمتجر وللإدارة، ومتى انبعت التذكير
--    الوحيد بعد يومين — ليوقف التكرار.
--
-- ٣. قسم سوق جديد «منوّعات» فعّال وآخر شي — لزرّ «حطّ الباقي بقسم منوّعات».
--    بالمفتاح المنطقي `section_name_key` متل 0521 فإعادة التشغيل آمنة.

ALTER TABLE menu_items
    ADD COLUMN IF NOT EXISTS ameen_created boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS ameen_qty double precision,
    ADD COLUMN IF NOT EXISTS photo_hold boolean NOT NULL DEFAULT false;

UPDATE menu_items SET ameen_created = true
 WHERE NOT ameen_created
   AND section_source IN ('pending', 'group', 'name')
   AND COALESCE(external_code, '') <> '';

UPDATE menu_items SET photo_hold = true
 WHERE ameen_created AND NOT photo_hold AND NOT available AND image_media_id IS NULL;

ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS price_sync_pending_notified_at timestamptz,
    ADD COLUMN IF NOT EXISTS price_sync_pending_reminded_at timestamptz;

INSERT INTO platform_sections (name, icon, sort_order)
SELECT 'منوّعات', 'other', 9000
 WHERE NOT EXISTS (
       SELECT 1 FROM platform_sections p
        WHERE section_name_key(p.name) = section_name_key('منوّعات'));
