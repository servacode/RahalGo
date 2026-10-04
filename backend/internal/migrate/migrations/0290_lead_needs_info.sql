-- ══════════════════════════════════════════════════════════════════════
-- **طلبُ الانضمام «بحاجة معلومات» — يعود للمندوب بملاحظة**
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك 2026-10-04 على صفحة «طلبات الانضمام».)
--
-- **كان أمامَ المكتب بابان فقط: موافقةٌ أو رفض.** وطلبٌ ينقصه شيءٌ صغير
-- (موقعٌ خاطئ، رقمٌ ناقص) **يُرفض فيضيع، أو يُقبل ناقصاً.**
-- فصارت له حالةٌ ثالثة: `needs_info` — **والملاحظةُ في `decision_note`**
-- كما سببُ الرفض، فيقرؤها المندوبُ من قائمته (`GET /rep/leads`).
ALTER TABLE merchant_leads DROP CONSTRAINT IF EXISTS merchant_leads_status_check;
ALTER TABLE merchant_leads ADD CONSTRAINT merchant_leads_status_check
    CHECK (status IN ('new', 'needs_info', 'converted', 'rejected'));

-- **وتحذيرُ الرقم المكرّر يبحث بالرقم** — فهرسٌ يغني عن مسحِ الجدول لكلّ صفّ.
CREATE INDEX IF NOT EXISTS merchant_leads_phone_idx ON merchant_leads (phone);
