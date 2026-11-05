-- ══════════════════════════════════════════════════════════════════════
-- **معالمُ السائقين: خمسةٌ جديدةٌ كلَّ يوم** (قرارُ المالك ٢٠٢٦-١١-٠٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- «لازم كل سائق باليوم يرسل اقل شي 5 معالم، واذا ارسلو شي قديم طالعه، والي ما يبعث
-- معالم 5 باليوم يصله تنبيه انه لم تكتمل المعالم ال5 الخاصة بك بعد»
--
-- ١ · **يومُ المنصّة مختومٌ على كلّ صفّ** (`platform_day`) — من ساعة فتح الدوام لمثلها
--     (`account_day`، الهجرة 0476)، **فتبديلُ الساعة بعدين لا يحرّك القديم.**
-- ٢ · **المكرَّرُ يُحفظ بحالة `duplicate`** ومعه ما طابقه (`dup_*`) — لا يُعدّ، ولا يظهر
--     للموافقة ولا على خريطة العمليّات، **وتراه اللوحةُ** فتعرف من يكرّر.
-- ٣ · **وموقعُه يُصحَّح من نبضة السائق** إن كانت نقطةُ التطبيق قديمة (آخرُ تسليم):
--     `point_from_ping`، **ونقطةُ التطبيق محفوظةٌ في `sent_point`**، و`ping_gap_m` بُعدُ ما
--     بينهما (وفارغُه «بلا تحقّق»: لا نبضةَ حديثة).
-- ٤ · **ختمُ التذكير** (سائق · يوم · سبب) **وختمُ ملخّص المالك** (يوم) — فإعادةُ تشغيل
--     الخادم لا تكرّر شيئاً.
ALTER TABLE landmark_suggestions
    ADD COLUMN IF NOT EXISTS platform_day    date,
    ADD COLUMN IF NOT EXISTS sent_point      geography(Point, 4326),
    ADD COLUMN IF NOT EXISTS point_from_ping boolean NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS ping_gap_m      int,
    ADD COLUMN IF NOT EXISTS dup_source      text CHECK (dup_source IN ('map', 'store', 'suggestion', 'self')),
    ADD COLUMN IF NOT EXISTS dup_ref         text,
    ADD COLUMN IF NOT EXISTS dup_name        text,
    ADD COLUMN IF NOT EXISTS dup_distance_m  int;

-- **والقديمُ يومُه يومُ إرساله** — بساعة فتح الدوام المضبوطة اليوم.
UPDATE landmark_suggestions SET platform_day = account_day(created_at) WHERE platform_day IS NULL;
ALTER TABLE landmark_suggestions
    ALTER COLUMN platform_day SET DEFAULT account_day(now()),
    ALTER COLUMN platform_day SET NOT NULL;

-- **والحالةُ الرابعة: `duplicate`** — والقيدُ القديمُ بلا اسمٍ في 0470، **فلا يُحذف باسمٍ
-- نخمّنه**: لو اختلف الاسمُ لمرّ `DROP ... IF EXISTS` صامتاً وبقي القديم، **فيسقط كلُّ مكرَّرٍ
-- بـ٥٠٠.** فكلُّ قيد CHECK على عمود `status` يُقرأ من `pg_constraint` ويُحذف، ثمّ يُضاف الجديد.
DO $$
DECLARE
    c record;
BEGIN
    FOR c IN
        SELECT DISTINCT con.conname
          FROM pg_constraint con
          JOIN pg_attribute a ON a.attrelid = con.conrelid AND a.attnum = ANY (con.conkey)
         WHERE con.conrelid = 'landmark_suggestions'::regclass
           AND con.contype = 'c'
           AND a.attname = 'status'
    LOOP
        EXECUTE format('ALTER TABLE landmark_suggestions DROP CONSTRAINT %I', c.conname);
    END LOOP;
END $$;
ALTER TABLE landmark_suggestions ADD CONSTRAINT landmark_suggestions_status_check
    CHECK (status IN ('pending', 'approved', 'rejected', 'duplicate'));

-- **عدُّ السائق بيومه · ولوحةُ اليوم · و«موجود من قبل» بالمسافة.**
CREATE INDEX IF NOT EXISTS landmark_suggestions_driver_day_idx ON landmark_suggestions (suggested_by, platform_day);
CREATE INDEX IF NOT EXISTS landmark_suggestions_day_idx ON landmark_suggestions (platform_day);
CREATE INDEX IF NOT EXISTS landmark_suggestions_point_idx ON landmark_suggestions USING gist (point);

-- **ختمُ التذكير: مرّةً لكلّ سائقٍ ويومٍ وسبب** — بالساعة (`hour`) أو عند إغلاق الدوام (`shift_end`).
CREATE TABLE IF NOT EXISTS landmark_quota_reminders (
    driver_id  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day        date        NOT NULL,
    reason     text        NOT NULL CHECK (reason IN ('hour', 'shift_end')),
    sent_count int         NOT NULL,
    sent_at    timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (driver_id, day, reason)
);
CREATE INDEX IF NOT EXISTS landmark_quota_reminders_day_idx ON landmark_quota_reminders (day);

-- **ختمُ ملخّص المالك: مرّةً لليوم.**
CREATE TABLE IF NOT EXISTS landmark_daily_digests (
    day     date PRIMARY KEY,
    sent_at timestamptz NOT NULL DEFAULT now()
);
