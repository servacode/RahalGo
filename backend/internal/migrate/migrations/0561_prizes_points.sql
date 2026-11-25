-- «جوائز رحّال غو» تصير مسابقةَ نقاط (قرارُ المالك ٢٠٢٦-١١-٢٥: «صاحب أكثر تفاعل يستاهل»).
--
-- كانت سحباً عشوائيّاً على الدعوات (هجرة ٠٥٥٧). **ولم تُطلق بعد** (`prizes.enabled` مطفأ)،
-- فلا سحبَ في القاعدة ويُعاد تشكيلُ الجدولين بلا نقل.
--
-- # النقاط
--
-- **تُحسب من مصادرها في كلّ قراءة** (`internal/prizes/points.go`): الطلباتُ المسلَّمة
-- (`orders.delivered_at`) والدعوات (`referrals.created_at`) قائمة، **ويُضاف هنا مصدران**:
-- الحضورُ اليوميّ والمتابعة. فإلغاءُ تسليمٍ يمحو `delivered_at` فتسقط نقاطُه بلا قيدٍ عكسيّ.
--
-- # الحسمُ والإعلان (بتوقيت دمشق)
--
-- الحسمُ يومَ min(29, آخرُ يومٍ − 1) الساعةَ ١٢:٠٠، والإعلانُ آخرَ يومٍ ١٢:٠٠، **وما بعد الحسم
-- للشهر التالي**. وصفٌّ واحدٌ لكلّ شهرٍ في `prize_draws` (مفتاحٌ أوّليّ) يمنع إعلانين.

-- ── الحضورُ اليوميّ: مرّةً باليوم ─────────────────────────────────────
CREATE TABLE IF NOT EXISTS prize_checkins (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- **يومُ دمشق** — المفتاحُ يمنع حضورين في يوم.
    day date NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, day)
);
CREATE INDEX IF NOT EXISTS prize_checkins_at_idx ON prize_checkins (created_at);

-- ── المتابعة: مرّةً في عمر الحساب لكلّ صفحة، بالأمانة بلا تحقّق ─────────
CREATE TABLE IF NOT EXISTS prize_follows (
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    network text NOT NULL CHECK (network IN ('facebook', 'instagram')),
    -- **تُحسب في شهر الضغط.**
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, network)
);
CREATE INDEX IF NOT EXISTS prize_follows_at_idx ON prize_follows (created_at);

-- ── الإعلان: لا فرصَ بعد اليوم ──────────────────────────────────────────
ALTER TABLE prize_draws DROP COLUMN IF EXISTS total_entries;
ALTER TABLE prize_draws
    -- **كم بلغ الحدّ** من المشتركين (`entrants` = من له نقطةٌ فأكثر).
    ADD COLUMN IF NOT EXISTS qualified int NOT NULL DEFAULT 0,
    -- **الحدُّ الأدنى ولحظةُ الحسم كما كانا يومَ الإعلان** — فتغييرُ الإعداد لا يعيد كتابةَ الماضي.
    ADD COLUMN IF NOT EXISTS min_points int NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS cutoff_at timestamptz;

ALTER TABLE prize_winners DROP COLUMN IF EXISTS invites;
ALTER TABLE prize_winners DROP COLUMN IF EXISTS entries;
ALTER TABLE prize_winners
    ADD COLUMN IF NOT EXISTS points int NOT NULL DEFAULT 0 CHECK (points >= 0),
    -- **لحظةُ بلوغ المجموع** — بها كُسر التعادل.
    ADD COLUMN IF NOT EXISTS reached_at timestamptz;

-- ── الطلباتُ المسلَّمة بلحظة تسليمها — مصدرُ نقاط الطلب ─────────────────
CREATE INDEX IF NOT EXISTS orders_delivered_at_idx ON orders (delivered_at) WHERE status = 'delivered';

-- **إعدادُ «كم دعوةً للفرصة» انتهى معناه** — يُمحى ما حُفظ منه.
DELETE FROM app_settings WHERE key = 'prizes.invites_per_entry';
