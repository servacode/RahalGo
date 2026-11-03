-- ══════════════════════════════════════════════════════════════════════
-- **الخريطة من حركة السائقين — الخطوات ١ و٢ و٣** (قرارُ المالك ٢٠٢٦-١١-٠٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «جهّز الخريطة لنستفيد من حركة السائقين». نقاطُ `map_trace_points` (هجرة ٠٤٦٦) بلا هويّة ولا ساعة،
-- وهنا ثلاثةُ أشياء فوقها:
--
--   ١ · `road_dist_m` — بُعدُ النقطة عن أقرب طريقٍ في خريطتنا، يُقاس دفعاتٍ بالأمر
--       `cmd/maplearn` (لا مع كلّ طلب). `NULL` = لم يُقَس بعد.
--   ٢ · `map_street_candidates` — خطوطٌ مقترحة: نقاطٌ بعيدةٌ عن كلّ طريقٍ تتكرّر في أكثر من يوم.
--   ٣ · `map_street_runs` — سجلُّ جولات التحليل (آخرُها يُعرض في اللوحة).
--
-- **والموافَقُ عليه يُصدَّر ملفَّ OSM ليدخل بناءَ الخريطة** (الخطوة ٤، لم تُنفَّذ).
ALTER TABLE map_trace_points ADD COLUMN IF NOT EXISTS road_dist_m real;

-- **غيرُ المقيس** أوّلاً — ترتيبُ الدفعة.
CREATE INDEX IF NOT EXISTS map_trace_points_unmeasured_idx ON map_trace_points (id) WHERE road_dist_m IS NULL;
-- **والبعيدُ وحدَه** — ما يُقرأ للتجميع.
CREATE INDEX IF NOT EXISTS map_trace_points_far_idx ON map_trace_points (day) WHERE road_dist_m >= 15;

CREATE TABLE IF NOT EXISTS map_street_candidates (
    id          bigserial PRIMARY KEY,
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    line        geography(LineString, 4326) NOT NULL,
    name        text NOT NULL DEFAULT '',
    length_m    real NOT NULL,
    points      integer NOT NULL,
    days        integer NOT NULL,
    first_day   date NOT NULL,
    last_day    date NOT NULL,
    detected_at timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    decided_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    decided_at  timestamptz
);
CREATE INDEX IF NOT EXISTS map_street_candidates_line_idx ON map_street_candidates USING gist (line);
CREATE INDEX IF NOT EXISTS map_street_candidates_status_idx ON map_street_candidates (status);

CREATE TABLE IF NOT EXISTS map_street_runs (
    id          bigserial PRIMARY KEY,
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    measured    integer NOT NULL DEFAULT 0,
    far_points  integer NOT NULL DEFAULT 0,
    candidates  integer NOT NULL DEFAULT 0,
    error       text NOT NULL DEFAULT ''
);
