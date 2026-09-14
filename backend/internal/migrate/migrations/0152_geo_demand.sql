-- ══════════════════════════════════════════════════════════════════════
-- **إشارةُ الطلب الجغرافيّ — نوعان في جدولٍ واحد** (`CR`، ٢٠٢٦-٠٩-١٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- # ولمَ لا جدولٌ جديد
--
-- **و`coverage_requests` قائمٌ منذ هجرة ٠١٢٧** — **وله فهارسُه
-- الجغرافيّةُ وحالاتُه الخمسُ وسجلُّ من قرّر**، **وله في `opsmap` قائمةٌ
-- إداريّةٌ وتجميعٌ بالخلايا** (`RequestDemand` · `UnservedDemand` ·
-- `Opportunities`).
--
-- **ونيّةُ «أخبرني عند توفّر الخدمة» أختُ نيّةِ «أضِف منطقتي»** —
-- **كلتاهما تقولان: هنا طلبٌ لا نأخذه.** **وجدولٌ ثانٍ بمخطَّطٍ متطابقٍ
-- يعني تجميعين وقائمتين إداريّتين تفترقان يوماً.**
--
-- **فيُضاف نوعٌ ولا يُبنى نظامٌ ثانٍ.**
--
-- # وهما ليستا شيئاً واحداً
--
--	coverage_request  **الخدمةُ هنا وعنواني خارجَ الشكل**  ⇒ أضِف منطقتي
--	service_interest  **لم تصل الخدمةُ إلى مكاني بعد**     ⇒ أخبرني
--
-- **والرسالتان لا تتبادلان** — **ومن قيل له «سنأخذ منطقتك بعين
-- الاعتبار» ونحن لم نصل مدينتَه أصلاً وُعد بما لا يقع.**

-- ── النوع ─────────────────────────────────────────────────────────────
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'coverage_request';

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'coverage_requests_kind_ck') THEN
        ALTER TABLE coverage_requests
            ADD CONSTRAINT coverage_requests_kind_ck
            CHECK (kind IN ('coverage_request', 'service_interest'));
    END IF;
END $$;

-- ── المحافظةُ إلى جانب المدينة ────────────────────────────────────────
--
-- **والمدينةُ محفوظةٌ منذ ٠١٢٧** — **والمحافظةُ لم تكن**، **والسؤالُ
-- التشغيليُّ «كم طلباً في دمشق؟» محافظةٌ لا مدينة.**
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS governorate_id uuid REFERENCES governorates(id);

-- ── خليّةُ الشبكة — مفتاحُ التفرّد ────────────────────────────────────
--
-- **ولا تُقارَن الإحداثيّاتُ بالتساوي العشريّ** — **وضغطتان على
-- النقطة نفسِها تختلفان في الخانة السابعة**، **فيُولَد طلبان لمكانٍ
-- واحد.**
--
-- **والخليّةُ هي شبكةُ `opsmap` عينُها** (`CellSizeDeg = 0.01` ≈ ١٫١ كم
-- عند خطّ عرض الرقّة) — **وهي التي تُجمَّع بها الخريطةُ الإداريّة
-- أصلاً.** **فمن طلب مرّتين في الحيّ نفسِه طلبٌ واحدٌ يُعدّ مرّتين.**
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS cell_y double precision;
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS cell_x double precision;

-- ── السريانُ والتكرارُ وآخرُ مرّة ─────────────────────────────────────
--
-- **و«أخبرني» اشتراكٌ يُلغى** — **ولا يُحبَس أحدٌ في تسويقٍ دائم.**
-- **و«أضِف منطقتي» واقعةٌ تاريخيّةٌ لا تُمحى** بمغادرة الشاشة.
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS active boolean NOT NULL DEFAULT true;
-- **وعددُ المرّات** — **فالتفرّدُ لا يمحو شدّةَ الطلب.**
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS requests integer NOT NULL DEFAULT 1;
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS last_seen_at timestamptz NOT NULL DEFAULT now();
-- **وسببُ الإتاحة لحظةَ الطلب** — **يُقرأ لاحقاً فيُعرَف ما رآه صاحبُه.**
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS reason text NOT NULL DEFAULT '';
-- **ومتى أُخبِر** — **الدفعةُ الثامنةُ تقرؤه فلا تُخبر مرّتين.**
ALTER TABLE coverage_requests
    ADD COLUMN IF NOT EXISTS notified_at timestamptz;

-- ── وتُملأ الخليّةُ للصفوف القائمة ────────────────────────────────────
--
-- **ولا صفَّ يبقى بلا مفتاحِ تفرّد** — **وإلّا لم يُدمَج فيه جديد.**
UPDATE coverage_requests
   SET cell_y = floor(ST_Y(at::geometry) / 0.01) * 0.01 + 0.005,
       cell_x = floor(ST_X(at::geometry) / 0.01) * 0.01 + 0.005
 WHERE cell_y IS NULL OR cell_x IS NULL;

-- ── التفرّدُ لصاحب حساب ───────────────────────────────────────────────
--
-- **ولمن لا حساب له لا يُفرَّد** — **ولا هويّةَ تجمع ضغطتيه**، **ولا
-- تُخترَع بصمةُ جهازٍ لأجل ذلك** (نهيُ المالك).
--
-- **وجزئيٌّ**: `NULL` لا يساوي `NULL` في فهرسٍ فريد، **فصفوفُ المجهولين
-- لا تتزاحم.**
CREATE UNIQUE INDEX IF NOT EXISTS coverage_requests_dedupe_idx
    ON coverage_requests (user_id, kind, cell_y, cell_x)
 WHERE user_id IS NOT NULL;

-- ── فهارسُ السؤال التشغيليّ ───────────────────────────────────────────
CREATE INDEX IF NOT EXISTS coverage_requests_kind_idx
    ON coverage_requests (kind, created_at DESC);
CREATE INDEX IF NOT EXISTS coverage_requests_gov_idx
    ON coverage_requests (governorate_id) WHERE governorate_id IS NOT NULL;

-- ── والمحافظةُ تُملأ للصفوف التي تُعرَف مدينتُها ──────────────────────
UPDATE coverage_requests r
   SET governorate_id = d.governorate_id
  FROM cities c JOIN districts d ON d.id = c.district_id
 WHERE r.city_id = c.id AND r.governorate_id IS NULL;
