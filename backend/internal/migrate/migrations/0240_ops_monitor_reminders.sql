-- ══════════════════════════════════════════════════════════════════════
--  مراقبة التشغيل — تذكيرُ الطلب العالق يتكرّر حتّى يتصرّف أحد
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «مراقبة التشغيل»، البند ٤.)
--
-- كان الراصدُ يُنذر مرّةً لكلّ سبب، **فإن لم يرَ أحدٌ الإشعارَ بقي الطلبُ عالقاً
-- ساعاتٍ بلا تذكير** (قِيس على التجهيز: ‎#1400‎ «مقبول» منذ ٣٠٨ دقائق).
--
-- وصار يُذكّر كلَّ `ops.stuck_reminder_min` دقيقة **حتّى يتصرّف أحد**:
--
--   تغيّرت حالةُ الطلب   ⇒ alerted_status لم يعد يساوي status
--   أُسند لسائقٍ أو بُدّل ⇒ alerted_driver_id لم يعد يساوي driver_id
--   ضغط موظّفٌ «أنا عليه» ⇒ alert_ack_at
--
-- **والسببُ الجديدُ يبدأ من جديد** — يمسح «أنا عليه» ويُنذر فوراً كما كان.

ALTER TABLE orders ADD COLUMN IF NOT EXISTS alerted_status text;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS alerted_driver_id uuid;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS alert_repeats int NOT NULL DEFAULT 0;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS alert_ack_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS alert_ack_by uuid REFERENCES users(id) ON DELETE SET NULL;
