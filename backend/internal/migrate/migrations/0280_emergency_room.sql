-- ══════════════════════════════════════════════════════════════════════
-- ٠٢٨٠ · غرفةُ الطوارئ — صندوقٌ واحدٌ بنوعٍ مخزَّن وخطواتِ حلّ
-- ══════════════════════════════════════════════════════════════════════
--
-- (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — «غرفةُ الطوارئ».)
--
-- # ما كان
--
-- **الجدولُ يحمل طارئَ السائق بعد الاستلام وحدَه** — والحادثُ والعطلُ والظرفُ
-- القاهرُ قبل الاستلام لا يدخلونه (بابُ «لدي مشكلة» يحرّر الطلبَ وينبّه ويمضي)،
-- **وإغلاقُ المتجر الطارئ وتوقّفُ المنصّة لهما مواضعُ أخرى.** فصفحةُ الطوارئ لا ترى
-- إلّا ثلثَ ما يقع.
--
-- # وما صار
--
-- **الجدولُ نفسُه صندوقُ كلّ الطوارئ** (`driver_emergencies` اسمُه التاريخيّ —
-- لا يُعاد تسميتُه فتنكسر قراءاتٌ قائمة) **بنوعٍ مخزَّن**:
--
--	accident       حادث — الأخطر، ويقفل دوامَ السائق حتّى «السائقُ بخير»
--	breakdown      عطل («تعطّلت درّاجتي»)
--	force_majeure  ظرفٌ قاهر
--	store_closure  إغلاقٌ طارئٌ لمتجر (`merchant_id`)
--	platform_halt  توقّفُ المنصّة
--
-- **وما قبل هذا العمود يُقرأ «حادثاً»** — كان الزرُّ زرَّ الحادث بعد الاستلام،
-- **والإنذارُ الأشدُّ خيرٌ من الأخفّ لما لا نعرف نوعَه.**
--
-- # وخطواتُ الحلّ أعمدةٌ لا نصّ
--
--	استلمتها        acknowledged_at/by (0181)
--	السائقُ بخير؟   driver_ok_at/by
--	مصيرُ الطلب     outcome · outcome_at/by
--	المال           money_request_id (طلبُ تعويضٍ في driver_compensation_requests)
--	                أو money_skipped — **ولا مالَ يُدفع من هنا أبداً**
--	تمّ             status = 'resolved' (0055/0108)
--
-- **وسجلُّ ملاحظاتٍ يُضاف إليه ولا يُعدَّل** (`emergency_notes`) — كان الحلُّ جملةً واحدة.

ALTER TABLE driver_emergencies ALTER COLUMN driver_id DROP NOT NULL;

ALTER TABLE driver_emergencies
    ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'accident';
ALTER TABLE driver_emergencies DROP CONSTRAINT IF EXISTS driver_emergencies_kind_check;
ALTER TABLE driver_emergencies ADD CONSTRAINT driver_emergencies_kind_check
    CHECK (kind IN ('accident', 'breakdown', 'force_majeure', 'store_closure', 'platform_halt'));

-- **المتجرُ للإغلاق الطارئ** — ويُمحى البلاغُ بمحو متجره.
ALTER TABLE driver_emergencies
    ADD COLUMN IF NOT EXISTS merchant_id uuid REFERENCES merchants(id) ON DELETE CASCADE;

-- **أين كانت البضاعة لحظةَ الطارئ** — قبل الاستلام أم بعده. فارغٌ لما لا طلبَ له.
ALTER TABLE driver_emergencies
    ADD COLUMN IF NOT EXISTS stage text NOT NULL DEFAULT '';
ALTER TABLE driver_emergencies DROP CONSTRAINT IF EXISTS driver_emergencies_stage_check;
ALTER TABLE driver_emergencies ADD CONSTRAINT driver_emergencies_stage_check
    CHECK (stage IN ('', 'before_pickup', 'after_pickup'));

ALTER TABLE driver_emergencies
    ADD COLUMN IF NOT EXISTS driver_ok_at timestamptz,
    ADD COLUMN IF NOT EXISTS driver_ok_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS outcome text NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS outcome_at timestamptz,
    ADD COLUMN IF NOT EXISTS outcome_by uuid REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS money_request_id uuid
        REFERENCES driver_compensation_requests(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS money_skipped boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS money_at timestamptz,
    ADD COLUMN IF NOT EXISTS money_by uuid REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE driver_emergencies DROP CONSTRAINT IF EXISTS driver_emergencies_outcome_check;
ALTER TABLE driver_emergencies ADD CONSTRAINT driver_emergencies_outcome_check
    CHECK (outcome IN ('', 'redispatch', 'return_store', 'cancel', 'continue'));

-- **إغلاقٌ مفتوحٌ واحدٌ لكلّ متجر · وتوقّفٌ مفتوحٌ واحدٌ للمنصّة.**
CREATE UNIQUE INDEX IF NOT EXISTS driver_emergencies_one_open_store
    ON driver_emergencies (merchant_id) WHERE status = 'open' AND kind = 'store_closure';
CREATE UNIQUE INDEX IF NOT EXISTS driver_emergencies_one_open_halt
    ON driver_emergencies ((kind)) WHERE status = 'open' AND kind = 'platform_halt';

-- **والإغلاقاتُ القائمةُ تدخل الصندوق** — باستلامها إن استُلمت.
INSERT INTO driver_emergencies (kind, merchant_id, created_at, acknowledged_at, acknowledged_by)
SELECT 'store_closure', m.id, COALESCE(m.emergency_closed_at, now()),
       m.emergency_ack_at, m.emergency_ack_by
  FROM merchants m
 WHERE m.emergency_closed
   AND NOT EXISTS (SELECT 1 FROM driver_emergencies e
                    WHERE e.merchant_id = m.id AND e.status = 'open'
                      AND e.kind = 'store_closure');

-- ── سجلُّ الملاحظات ──────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS emergency_notes (
    id           bigserial PRIMARY KEY,
    emergency_id uuid NOT NULL REFERENCES driver_emergencies(id) ON DELETE CASCADE,
    author_id    uuid REFERENCES users(id) ON DELETE SET NULL,
    body         text NOT NULL CHECK (length(btrim(body)) > 0),
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS emergency_notes_by_emergency
    ON emergency_notes (emergency_id, id);

-- **يُضاف إليه ولا يُعدَّل** — والحذفُ بحذف البلاغ وحدَه.
CREATE OR REPLACE FUNCTION emergency_notes_append_only() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'emergency_notes append-only';
END;
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS emergency_notes_append_only ON emergency_notes;
CREATE TRIGGER emergency_notes_append_only BEFORE UPDATE ON emergency_notes
    FOR EACH ROW EXECUTE FUNCTION emergency_notes_append_only();
