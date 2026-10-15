-- ══════════════════════════════════════════════════════════════════════
-- **المصروفاتُ والتعويضات — قراراتُ المالك ٢٠٢٦-١٠-١٥**
-- ══════════════════════════════════════════════════════════════════════
--
--   اسمُ الموظّف       **حقلٌ اختياريٌّ على المصروف والاقتراح** — وإلزاميٌّ في بابِ
--                      الرواتب (`is_salary`). كان الراتبُ يُكتب في الملاحظة نصّاً حرّاً،
--                      **فلا يُعرف كم قبض كلُّ موظّفٍ في الشهر.**
--   المصروفُ الشهريّ    **علَمٌ على الباب** (`monthly_recurring`) — وأوّلَ كلّ شهرٍ بيوم
--                      دمشق يُذكَّر المالكُ والماليّةُ بأبوابه. **والتذكيرُ مرّةً للشهر**
--                      (`expense_monthly_reminders`) — إعادةُ تشغيل الخادم لا تكرّره.
--   الإيصالُ الواحد    **صورةُ إيصالٍ لمصروفٍ واحد** — كانت تُرفق بمصروفين فيُصرف المالُ
--                      مرّتين بورقةٍ واحدة. الحارسُ الأوّلُ في المعالج، **وهذان الفهرسان الثاني.**
--   التعويضُ المرفوض    **يُقترح ثانيةً** — كان `UNIQUE (order_id, driver_id)` يمنع أيَّ
--                      اقتراحٍ بعد الرفض. **والفهرسُ الآن على الحيّ وحدَه** (معلَّقٌ أو مقبول).

-- ── ١ · اسمُ الموظّف ──────────────────────────────────────────────────
ALTER TABLE expenses
    ADD COLUMN IF NOT EXISTS employee_name text NOT NULL DEFAULT '';
ALTER TABLE expense_requests
    ADD COLUMN IF NOT EXISTS employee_name text NOT NULL DEFAULT '';

-- ── ٢ · علَما الباب: الرواتبُ والشهريُّ المتكرّر ────────────────────────
--
-- **والرواتبُ علَمٌ لا اسم** — الاسمُ يُعدَّل من الشاشة، **وحكمٌ على اسمٍ يسقط يومَ
-- يُكتب «رواتب الموظفين».**
ALTER TABLE expense_categories
    ADD COLUMN IF NOT EXISTS is_salary boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS monthly_recurring boolean NOT NULL DEFAULT false;

UPDATE expense_categories SET is_salary = true WHERE btrim(name) = 'رواتب';
-- **اقتراحٌ يبدأ به المالك** — يُطفئه أو يزيد عليه من «إدارة الأبواب».
UPDATE expense_categories SET monthly_recurring = true WHERE btrim(name) IN ('إيجار', 'رواتب');

-- ── ٣ · تذكيرُ الشهر مرّةً واحدة ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS expense_monthly_reminders (
    -- أوّلُ الشهر بيوم دمشق.
    month   date PRIMARY KEY,
    sent_at timestamptz NOT NULL DEFAULT now()
);

-- ── ٤ · الإيصالُ لمصروفٍ واحد ────────────────────────────────────────
--
-- **والملغى والمرفوضُ لا يحجزان إيصالَهما** — يُسجَّل المصروفُ مصحَّحاً بورقته نفسِها.
-- **والاقتراحُ المقبولُ ومصروفُه يحملان الإيصالَ نفسَه** — فالفهرسُ لكلّ جدولٍ وحدَه،
-- والتقاطعُ يحرسه المعالج.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM expenses
                WHERE receipt_media_id IS NOT NULL AND voided_at IS NULL
                GROUP BY receipt_media_id HAVING count(*) > 1) THEN
        RAISE NOTICE 'expenses: shared receipts exist — expenses_receipt_once not created yet';
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS expenses_receipt_once
            ON expenses (receipt_media_id)
            WHERE receipt_media_id IS NOT NULL AND voided_at IS NULL;
    END IF;
    IF EXISTS (SELECT 1 FROM expense_requests
                WHERE receipt_media_id IS NOT NULL AND status = 'pending'
                GROUP BY receipt_media_id HAVING count(*) > 1) THEN
        RAISE NOTICE 'expense_requests: shared receipts exist — expense_requests_receipt_once not created yet';
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS expense_requests_receipt_once
            ON expense_requests (receipt_media_id)
            WHERE receipt_media_id IS NOT NULL AND status = 'pending';
    END IF;
END $$;

-- ── ٥ · التعويضُ المرفوضُ يُقترح ثانيةً ──────────────────────────────
--
-- **والحيُّ واحدٌ لكلّ طلبٍ ومستفيد** — معلَّقٌ أو مقبول. **وشرطُ الفهرس هو شرطُ
-- `ON CONFLICT` في المحرّك حرفاً** (`requestDriverCompensation`)، وإلّا لم يُستدَلّ عليه.
ALTER TABLE driver_compensation_requests
    DROP CONSTRAINT IF EXISTS driver_compensation_requests_order_id_driver_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS driver_compensation_requests_order_live
    ON driver_compensation_requests (order_id, driver_id)
    WHERE status IN ('pending', 'approved');
