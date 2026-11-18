-- ══════════════════════════════════════════════════════════════════════
-- «لازم يوصلنا موقع مباشر» — تذكير الزبون تلقائياً (قرار المالك ٢٠٢٦-١١-١٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- قيس على السيرفر الحقيقي: ٥ طلبات من ١٥ صارت «بالطريق» وصل فيها موقع مباشر. فالمحرّك صار يذكّر:
--   live_nudge_push_at  بعد دقيقة بالطريق بلا موقع: إشعار للزبون وطلب موقع (متل زر الكابتن، بلا ما ينحسب عليه).
--   live_nudge_wa_at    بعد دقيقتين بلا موقع أبداً: رسالة واتساب «ردّ بموقعك المباشر» — مرّة وحدة للطلب.
-- والعمودان علامة بالقاعدة لا بالذاكرة: إعادة تشغيل الخادم ما بتعيد الإرسال.
--
--   live_first_at / live_first_source  أوّل موقع مباشر وصل للطلب ومن وين (app · whatsapp) —
--   لأنّ صفّ order_live_locations بينمسح لما توقف المشاركة، فما كان في طريقة نعرف «وصل موقع ولا لأ».

ALTER TABLE orders ADD COLUMN IF NOT EXISTS live_nudge_push_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS live_nudge_wa_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS live_first_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS live_first_source text;
