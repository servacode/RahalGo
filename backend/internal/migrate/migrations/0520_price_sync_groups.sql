-- ٠٥٢٠ · كلّ صنف من الأمين يروح على قسمو (قرار المالك ٢٠٢٦-١١-٠٧)
--
-- كانت «أنشئ غير الموجود» تحطّ كلّ صنف جديد بقسم انتظار واحد مخفي
-- («من الأمين — بانتظار التصنيف») — وبالتجربة الحقيقية راح ٥٣٠٨ صنف كلّهم لهنيك.
--
-- ١. price_sync_groups: لكلّ متجر، مجموعات الأمين اللي شافها وأيّ قسم سوق اختار
--    لكلّ مجموعة. القسم فاضي = لسّا ما تقرّر.
-- ٢. menu_items.external_group: مجموعة الصنف بالأمين كما وصلت.
-- ٣. menu_items.section_source: من وين إجا قسم الصنف:
--      pending = بقسم الانتظار · group = من مجموعة الأمين · name = من اسمه
--      manual  = حدا اختاره بإيده — والمزامنة ما بتحرّكه أبداً.
--    الافتراض manual فكلّ صنف موجود بيضل مكانه، إلّا اللي قاعدين بقسم الانتظار:
--    هدول ما حدا حطّهم بإيده، فبيصيروا pending ليقدر «وزّع تلقائياً» يرتّبهم.

CREATE TABLE IF NOT EXISTS price_sync_groups (
    merchant_id         uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    group_key           text NOT NULL,
    group_name          text NOT NULL,
    platform_section_id uuid REFERENCES platform_sections(id) ON DELETE SET NULL,
    updated_at          timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (merchant_id, group_key)
);

ALTER TABLE menu_items
    ADD COLUMN IF NOT EXISTS external_group text,
    ADD COLUMN IF NOT EXISTS section_source text NOT NULL DEFAULT 'manual';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'menu_items_section_source_check') THEN
        ALTER TABLE menu_items ADD CONSTRAINT menu_items_section_source_check
            CHECK (section_source IN ('pending', 'group', 'name', 'manual'));
    END IF;
END $$;

UPDATE menu_items SET section_source = 'pending'
 WHERE section_source = 'manual'
   AND platform_section_id IN (
       SELECT id FROM platform_sections
        WHERE section_name_key(name) = section_name_key('من الأمين — بانتظار التصنيف'));
