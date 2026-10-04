-- ══════════════════════════════════════════════════════════════════════
-- **صفحةُ «التعويضات» الواحدة وطريقُ موافقةٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- كلُّ تعويضٍ في المنصّة صار طلباً في هذا الجدول: **تعويضُ السائق** (المحرّكُ أو
-- المكتبُ يقترح)، **وتعويضُ المتجر عن بضاعةٍ رُدّت** (الماليّةُ تكتب المبلغ)،
-- **وتعويضُ الشكوى** (الدعمُ يقترح لصاحب الشكوى). والماليّةُ توافق بكلمة السرّ،
-- **ولا يوافق أحدٌ على ما اقترحه**، والمبلغُ يُخصم من الخزينة دائماً.
--
-- **واسمُ الجدول باقٍ** — أقسامٌ أخرى تقرؤه وتكتبه (غرفةُ الطوارئ ورئيسيّةُ
-- اللوحة)، **والقيدُ الفريدُ (طلب، سائق) باقٍ** لأنّ غرفةَ الطوارئ تكتب بـ
-- `ON CONFLICT (order_id, driver_id)`.
--
-- **و`driver_id` صار «المستفيد»**: السائق، أو صاحبُ المتجر، أو صاحبُ الشكوى.
ALTER TABLE driver_compensation_requests
    ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'driver',
    ADD COLUMN IF NOT EXISTS ticket_id uuid REFERENCES tickets(id) ON DELETE CASCADE,
    -- **فراغُه = المحرّك** — والمحرّكُ ليس شخصاً يُمنع من الموافقة.
    ADD COLUMN IF NOT EXISTS proposed_by uuid REFERENCES users(id),
    -- **ملاحظةُ الاقتراح الداخليّة** — لا يراها المستفيد.
    ADD COLUMN IF NOT EXISTS note text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS self_approved boolean NOT NULL DEFAULT false,
    -- **متى نُبّه المالكُ إلى تأخّره** — مرّةً واحدةً لكلّ طلب.
    ADD COLUMN IF NOT EXISTS overdue_alerted_at timestamptz;

ALTER TABLE driver_compensation_requests ALTER COLUMN order_id DROP NOT NULL;
ALTER TABLE driver_compensation_requests ALTER COLUMN fault SET DEFAULT '';

ALTER TABLE driver_compensation_requests
    DROP CONSTRAINT IF EXISTS driver_compensation_requests_kind_check;
ALTER TABLE driver_compensation_requests
    ADD CONSTRAINT driver_compensation_requests_kind_check
    CHECK (kind IN ('driver', 'merchant_goods', 'complaint'));

-- **وذنبُ السائق كما كان** — وغيرُه يحمل ذنبَ الطلب إن عُرف، أو فراغاً.
ALTER TABLE driver_compensation_requests
    DROP CONSTRAINT IF EXISTS driver_compensation_requests_fault_check;
ALTER TABLE driver_compensation_requests
    ADD CONSTRAINT driver_compensation_requests_fault_check
    CHECK ((kind = 'driver' AND fault IN ('customer', 'merchant', 'platform'))
        OR (kind <> 'driver' AND fault IN ('', 'customer', 'merchant', 'platform', 'driver')));

-- **الشكوى بتذكرتها، والباقي بطلبه.**
ALTER TABLE driver_compensation_requests
    DROP CONSTRAINT IF EXISTS driver_compensation_requests_target_check;
ALTER TABLE driver_compensation_requests
    ADD CONSTRAINT driver_compensation_requests_target_check
    CHECK ((kind = 'complaint' AND ticket_id IS NOT NULL)
        OR (kind <> 'complaint' AND order_id IS NOT NULL));

-- **شكوى واحدةٌ لا يُقترح لها تعويضان معاً** — والمرفوضُ لا يمنع اقتراحاً جديداً.
CREATE UNIQUE INDEX IF NOT EXISTS driver_compensation_requests_ticket_live
    ON driver_compensation_requests (ticket_id)
    WHERE kind = 'complaint' AND status <> 'rejected';

-- **سجلُّ القرارات بالأحدث** — تبويباتُ «تمت الموافقة» و«مرفوضة» و«الكل».
CREATE INDEX IF NOT EXISTS driver_compensation_requests_history
    ON driver_compensation_requests (status, created_at DESC);
