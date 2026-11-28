-- ══════════════════════════════════════════════════════════════════════
-- «الطرف الآخر يشارك موقعه المباشر من تطبيقنا» (قرار المالك ٢٠٢٦-١١-٢٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- الطرف الآخر بالأمانة (مستلم «أبعت» أو صاحب «أستقبل») أو مستلم توصيلة المتجر، إذا رقمه عليه حساب
-- زبون: يوصله إشعار بالتطبيق «لديك أمانة من صاحب الرقم … — هل تريد استقبال الأمانة؟» بزرّين «نعم/لا».
-- «نعم» موافقة تبدأ بعدها مشاركة موقعه المباشر حتى التسليم (أو الاستلام لصاحب «أستقبل»)، و«لا» تنكتب
-- بسجلّ الطلب ويتبلّغ المكتب. ورسالة الواتساب باقية كما هي احتياطاً.
--
-- كل شي على صفّ طلب الموقع نفسه — هو اللي بيربط الرقم بالطلب:
--   party_push_at      متى راح الإشعار الأول (مرة وحدة، والحجز قبل الإرسال)
--   party_reminder_at  متى راح التذكير بعد ٥ دقايق بلا جواب (مرة وحدة)
--   party_answer       جوابه: yes · no
--   party_answered_at  متى جاوب
--   party_live_at      آخر نقطة استلام انكتبت من التطبيق — تخفيف كتابة نقطة الاستلام
ALTER TABLE recipient_location_requests
    ADD COLUMN IF NOT EXISTS party_push_at     timestamptz,
    ADD COLUMN IF NOT EXISTS party_reminder_at timestamptz,
    ADD COLUMN IF NOT EXISTS party_answer      text,
    ADD COLUMN IF NOT EXISTS party_answered_at timestamptz,
    ADD COLUMN IF NOT EXISTS party_live_at     timestamptz;

ALTER TABLE recipient_location_requests DROP CONSTRAINT IF EXISTS recipient_location_requests_party_answer_ck;
ALTER TABLE recipient_location_requests ADD CONSTRAINT recipient_location_requests_party_answer_ck
    CHECK (party_answer IS NULL OR party_answer IN ('yes', 'no'));

-- **الطرف بيدوّر على طلباته برقمه** — والجولة بتقرأ المربوط بطلب وحده.
CREATE INDEX IF NOT EXISTS recipient_location_requests_party_idx
    ON recipient_location_requests (phone, created_at DESC) WHERE order_id IS NOT NULL;
