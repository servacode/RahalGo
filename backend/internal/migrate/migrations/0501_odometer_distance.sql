-- ══════════════════════════════════════════════════════════════════════
-- **الدوام والعدّاد ومسافة كل طلب** — قرارات المالك ٢٠٢٦-١١-٠٥
-- ══════════════════════════════════════════════════════════════════════
--
-- «بدي بتطبيق السائق اول مايفتح الدوام يطلب منه تصوير العداد الصبح، وعند اغلاق الدوام
-- كمان يطلب منه تصوير العداد … ومع كل طلب يكتب المسافة المقطوعة من بداية استلام الطلب
-- الى انتهاء الطلب … وفي اضافة تطلع للسائق كمان شقد المسافة المقطوعة باليوم.»
--
-- ثلاثة جداول، ولا مال: لا قيد ولا محفظة ولا صندوق (الدفتر ما بيتلمس).
--
--   driver_odometer_readings  قراءة العدّاد: الرقم اللي كتبه السائق (عليه الحساب) وصورته (الدليل)
--   driver_shifts             كل فتح وسكّر للدوام صفّ — مين سكّره وليش، والعدّادان، وأرقام التطبيق
--   order_drives              مسافة كل طلب لسائقه: للمتجر وللزبون — من القيادة الفعلية

-- ── قراءات العدّاد — سلسلة لا تُمحى لكل سائق ─────────────────────────
CREATE TABLE IF NOT EXISTS driver_odometer_readings (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id       uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    -- start عند فتح الدوام · end عند سكّره
    phase           text NOT NULL CHECK (phase IN ('start', 'end')),
    -- **الرقم كما كتبه السائق** — عليه الحساب، والصورة دليله.
    km              numeric(8, 1) NOT NULL CHECK (km >= 0),
    -- **تصحيح المكتب** إن غلط السائق بالكتابة — والأصل يبقى.
    corrected_km    numeric(8, 1) CHECK (corrected_km >= 0),
    corrected_by    uuid REFERENCES users (id) ON DELETE SET NULL,
    corrected_at    timestamptz,
    correction_note text NOT NULL DEFAULT '',
    -- ok · first (أول قراءة له) · jump_confirmed (قفزة كبيرة أكّدها السائق)
    -- **والرقم الأقل من آخر رقم مرفوض أصلاً** (قرار المالك) — فما في حال له.
    check_status    text NOT NULL DEFAULT 'ok' CHECK (check_status IN ('ok', 'first', 'jump_confirmed')),
    prev_km         numeric(8, 1),
    media_id        uuid REFERENCES media (id) ON DELETE SET NULL,
    -- **ولا موقع** — الرقمُ لا يحتاجه، والصورةُ تُؤخذ والدوامُ مسكَّر، ونصُّ الخصوصيّة
    -- يَعِد بالموقع «ما دامت ورديّتُك مفتوحة» وحدَها.
    app_build       int,
    created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS driver_odometer_driver_created_idx
    ON driver_odometer_readings (driver_id, created_at DESC);

-- ── الدوام — كل فتح وسكّر صفّ، أيّاً كان من سكّره ─────────────────────
CREATE TABLE IF NOT EXISTS driver_shifts (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id        uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    opened_at        timestamptz NOT NULL DEFAULT now(),
    closed_at        timestamptz,
    -- driver سكّره السائق · driver_skipped سكّره بلا صورة (وضع «يطلب ويسمح بالتخطّي») ·
    -- logout بالخروج من التطبيق · release لما ترك طلب · emergency بطارئ · accident بعد حادث ·
    -- ops سكّرته العمليات · account انوقف الحساب · qa تجربة
    -- stale صفٌّ بقي مفتوحاً والدوام مسكَّر بغير بابه (تنظيفُ التجهيز، أو المحرّكُ القديم بنافذة النشر)
    -- فيُسكَّر عند الفتح التالي
    close_reason     text CHECK (close_reason IN ('driver', 'driver_skipped', 'logout', 'release',
                                                  'emergency', 'accident', 'ops', 'account', 'qa', 'stale')),
    -- نسخة التطبيق اللي فتحه (فارغ = قديم أو غير معروف) — للمعلومة وحدها
    opened_build     int,
    -- **التطبيقُ أعلن أنّه يعرف ورقةَ العدّاد** (`odometer_capable`) — منه وحدَه يُطبَّق «إلزامي»،
    -- لا من رقم النسخة (تصليحٌ سريعٌ بلا الورقة يأخذ رقماً أكبر). فارغ = فتحٌ من النظام.
    opened_capable   boolean,
    -- كان مفتوحاً يوم نزلت الميزة
    migrated         boolean NOT NULL DEFAULT false,
    start_reading_id uuid REFERENCES driver_odometer_readings (id) ON DELETE SET NULL,
    end_reading_id   uuid REFERENCES driver_odometer_readings (id) ON DELETE SET NULL,
    -- **سماح العمليات بفتح الدوام بلا صورة مرّة وحدة** — مين وليش
    exempt_by        uuid REFERENCES users (id) ON DELETE SET NULL,
    exempt_reason    text NOT NULL DEFAULT '',
    -- أرقام التطبيق للدوام (متر/دقيقة) — يكتبها الكانس، والنهائي بعد ساعتين وعشر دقائق من السكّر
    drive_m          int,
    orders_m         int,
    no_order_m       int,
    not_driving_m    int,
    gap_min          int,
    orders_n         int,
    points           int,
    computed_at      timestamptz,
    final            boolean NOT NULL DEFAULT false
);
CREATE UNIQUE INDEX IF NOT EXISTS driver_shifts_one_open_idx ON driver_shifts (driver_id) WHERE closed_at IS NULL;
CREATE INDEX IF NOT EXISTS driver_shifts_driver_opened_idx ON driver_shifts (driver_id, opened_at DESC);
CREATE INDEX IF NOT EXISTS driver_shifts_pending_idx ON driver_shifts (computed_at) WHERE NOT final;

-- ── مسافة كل طلب لسائقه ───────────────────────────────────────────────
--
-- «من بداية استلام الطلب الى انتهاء الطلب»: من قبوله (للمتجر) لحتى خرج من يده (للزبون،
-- ومعه مشوار إرجاع البضاعة). **وطلبٌ تركه السائق له صفّ كمان** — مشواره للمتجر.
--
-- **صفٌّ لكلّ طلب × سائق × دوام** (قطعة): طلبٌ امتدّ على دوامين (طارئٌ بعد الاستلام ثمّ فتحٌ
-- جديدٌ وتسليم) له قطعتان، **ومسافتُه مجموعُهما** — لا يكتب دوامٌ فوق حصّة الآخر.
CREATE TABLE IF NOT EXISTS order_drives (
    order_id     uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    driver_id    uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    shift_id     uuid NOT NULL REFERENCES driver_shifts (id) ON DELETE CASCADE,
    started_at   timestamptz NOT NULL,
    picked_up_at timestamptz,
    -- فارغ = الطلب لسّا معه (والرقم «لحد هلق») — **ولا يكون إلّا بدوامٍ مفتوح**: دوامٌ مسكَّر
    -- يقصّ قطعتَه عند سكّره (`shift_closed`).
    ended_at     timestamptz,
    -- store_failed تعذّر عند المتجر فعاد الطلب للمكتب · shift_closed انسكّر الدوام والطلبُ معه
    end_kind     text NOT NULL DEFAULT 'open'
                 CHECK (end_kind IN ('open', 'delivered', 'failed', 'returned', 'cancelled', 'released',
                                     'store_failed', 'shift_closed', 'other')),
    to_pickup_m  int NOT NULL DEFAULT 0,
    with_order_m int NOT NULL DEFAULT 0,
    shared       boolean NOT NULL DEFAULT false,
    gap_min      int NOT NULL DEFAULT 0,
    points       int NOT NULL DEFAULT 0,
    computed_at  timestamptz NOT NULL DEFAULT now(),
    final        boolean NOT NULL DEFAULT false,
    PRIMARY KEY (order_id, driver_id, shift_id)
);
CREATE INDEX IF NOT EXISTS order_drives_driver_ended_idx ON order_drives (driver_id, ended_at);

-- **ولايات السائق تُقرأ من سجلّ الطلب باسمه** — وما كان له فهرس إلا بالطلب.
CREATE INDEX IF NOT EXISTS order_events_driver_idx
    ON order_events (driver_id, created_at) WHERE driver_id IS NOT NULL;

-- ── صورة العدّاد: صنفٌ شخصيٌّ محميّ (يُقرأ برابطٍ موقَّع) ──────────────
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'menu_section'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text,
    'expense_receipt'::text,
    'odometer'::text
  ])
);

-- ── سماحُ العمليّات بفتح الدوام بلا صورةٍ مرّةً واحدة ─────────────────
--
-- كاميرا مكسورة أو عدّادٌ معطّل: سماحٌ بسببٍ مكتوب (`drivers.manage`)، **ويُستهلك بأوّل فتح**
-- فيُكتب على الدوام (`exempt_by` · `exempt_reason`). وسماحٌ قائمٌ واحدٌ لكلّ سائق.
CREATE TABLE IF NOT EXISTS driver_odometer_exemptions (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    driver_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    granted_by    uuid REFERENCES users (id) ON DELETE SET NULL,
    reason        text NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now(),
    used_at       timestamptz,
    used_shift_id uuid REFERENCES driver_shifts (id) ON DELETE SET NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS driver_odometer_exemptions_pending_idx
    ON driver_odometer_exemptions (driver_id) WHERE used_at IS NULL;

-- ── الدوامات المفتوحة يوم تنزل الميزة ──────────────────────────────────
INSERT INTO driver_shifts (driver_id, opened_at, migrated)
SELECT id, COALESCE(shift_started_at, now()), true
  FROM users
 WHERE on_shift AND deleted_at IS NULL
ON CONFLICT DO NOTHING;
