-- ══════════════════════════════════════════════════════════════════════
-- عقوبة تأخير فتح الدوام — قرار المالك ٢٠٢٦-١١-١٤
-- ══════════════════════════════════════════════════════════════════════
--
-- كل سائق فعّال لازم يفتح دوامه خلال دقائق السماح بعد فتح المنصّة (أول فترة بيوم المنصّة).
-- اللي ما فتح: ينخصم من محفظته (عدد توصيلات × أجرة التوصيل العامّة) ومن عدّاد هدفه الشهري
-- (عدد توصيلات) — مرّة وحدة باليوم. إلا اللي عليه «إجازة» لهاليوم.
-- والمفتاح من الإعدادات (`drivers.late_shift_*`)، والكانس `server/late_shift_penalty.go`.

-- ── الإجازة: سائق · يوم المنصّة ────────────────────────────────────────
CREATE TABLE IF NOT EXISTS driver_leaves (
    driver_id  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day        date        NOT NULL,
    created_by uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (driver_id, day)
);

-- ── العقوبة: صفّ واحد لكلّ سائق ويوم (الفهرس الفريد هو ضمانة «مرّة وحدة») ──
CREATE TABLE IF NOT EXISTS driver_late_shift_penalties (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    day               date        NOT NULL,
    -- لحظة فتح المنصّة اللي تأخّر عنها
    opening_at        timestamptz NOT NULL,
    grace_min         int         NOT NULL,
    wallet_deliveries int         NOT NULL DEFAULT 0 CHECK (wallet_deliveries >= 0),
    target_deliveries int         NOT NULL DEFAULT 0 CHECK (target_deliveries >= 0),
    -- أجرة التوصيلة الوحدة ساعة العقوبة (`delivery.fee`)، والمبلغ = العدد × الأجرة
    unit_fee          bigint      NOT NULL DEFAULT 0 CHECK (unit_fee >= 0),
    amount            bigint      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    -- صفّ العقوبة بالحوافز (المحفظة والدين الباقي) — فارغ إذا ما في خصم محفظة
    incentive_id      uuid        REFERENCES incentives (id) ON DELETE SET NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    -- الإلغاء من الإدارة: يرجّع المبلغ ويمحي خصم الهدف
    cancelled_at      timestamptz,
    cancelled_by      uuid        REFERENCES users (id) ON DELETE SET NULL,
    cancel_reason     text        NOT NULL DEFAULT '',
    UNIQUE (driver_id, day)
);
CREATE INDEX IF NOT EXISTS driver_late_shift_penalties_day_idx
    ON driver_late_shift_penalties (day DESC);

-- ── خصم من عدّاد الهدف الشهري ────────────────────────────────────────
-- يُطرح مجموعه من «ما أنجزه» بكلّ مكان يُعدّ فيه الهدف (`incentives.doneSQL`)، والعدّ لا ينزل تحت الصفر.
CREATE TABLE IF NOT EXISTS driver_target_adjustments (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id  uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- وسم الشهر «2026-11» كما في `incentives.PeriodOf`
    month      text        NOT NULL CHECK (month ~ '^[0-9]{4}-[0-9]{2}$'),
    day        date        NOT NULL,
    deliveries int         NOT NULL CHECK (deliveries > 0),
    reason     text        NOT NULL CHECK (btrim(reason) <> ''),
    penalty_id uuid        REFERENCES driver_late_shift_penalties (id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS driver_target_adjustments_driver_month_idx
    ON driver_target_adjustments (driver_id, month);

-- ── إلغاء عقوبة بالحوافز: الصفّ يبقى ويُعلَّم، ولا ينحسب بمجموع العقوبات ──
ALTER TABLE incentives ADD COLUMN IF NOT EXISTS voided_at timestamptz;
ALTER TABLE incentives ADD COLUMN IF NOT EXISTS voided_by uuid REFERENCES users (id) ON DELETE SET NULL;

-- ── تذكير «باقي دقيقتين، افتح دوامك» بختم التذكيرات نفسه ───────────────
ALTER TABLE driver_shift_reminders DROP CONSTRAINT IF EXISTS driver_shift_reminders_kind_check;
ALTER TABLE driver_shift_reminders ADD CONSTRAINT driver_shift_reminders_kind_check
    CHECK (kind IN ('platform_closed', 'next_day', 'open_soon'));
