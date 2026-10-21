-- ══════════════════════════════════════════════════════════════════════
-- إشعار «انضمّ متجر جديد إلى عائلة رحّال غو» (قرار المالك ٢٠٢٦-١٠-٢١)
-- ══════════════════════════════════════════════════════════════════════
--
-- announced_at: متى أُعلن عن المتجر للزبائن. يُكتب مرّةً واحدة، فلا يُعاد الإعلان
-- إن أُغلق المتجر ثمّ فُتح.
-- announce_skip: «لا ترسل إشعاراً» لمتجرٍ بعينه (تجريبيّ مثلاً).

ALTER TABLE merchants ADD COLUMN IF NOT EXISTS announced_at timestamptz;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS announce_skip boolean NOT NULL DEFAULT false;

-- المتاجر الظاهرة اليوم «أُعلن عنها» — فلا يصل الزبائنَ عشرون إشعاراً دفعةً واحدة.
UPDATE merchants m SET announced_at = now()
 WHERE m.announced_at IS NULL AND m.status = 'active'
   AND EXISTS (SELECT 1 FROM menu_items i WHERE i.merchant_id = m.id AND i.available AND i.approved);
