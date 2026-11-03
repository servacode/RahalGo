-- ══════════════════════════════════════════════════════════════════════
-- **تعلّم الطرق من حركة السائقين: اتّجاه واحد وسرعة حقيقية** (قرارُ المالك ٢٠٢٦-١١-٠٣ «اي موافق»)
-- ══════════════════════════════════════════════════════════════════════
--
-- أثرُ `driver_trail` يُطابَق على حافّات الطرق (Valhalla trace_attributes) دفعةً ليليّة، ولا يُخزَّن هنا
-- أيُّ مسارٍ خام — **مجاميعُ فقط**:
--
--   `map_road_edges`    الحافّةُ (مفتاحٌ ثابتٌ + هندستُها بترتيبٍ قانونيّ) ومعروفٌ أنّها باتّجاه واحد في الخريطة أم لا.
--   `map_edge_passes`   مرّاتُ العبور لكلّ (حافّة، اتّجاه، يوم، سائق) — منها عددُ السائقين والأيّام والحصّةُ بكلّ اتّجاه.
--   `map_edge_speed`    مجاميعُ السرعة لكلّ (حافّة، ساعةٍ بتوقيت دمشق، نهايةُ أسبوع) — عددٌ ومجموع، لا نقاط.
--   `map_road_days`     الأيّامُ المعالَجة (لا يُحسب يومٌ مرّتين).
--   `map_oneway_candidates` شوارعُ مقترحةٌ باتّجاه واحد — المدير يوافق أو يرفض، والموافَقُ يُصدَّر لإعادة بناء الخريطة.
--
-- **لا يغيّر التوجيه ولا الـETA الآن.**
CREATE TABLE IF NOT EXISTS map_road_edges (
    edge_key       text PRIMARY KEY,
    way_id         bigint NOT NULL DEFAULT 0,
    name           text NOT NULL DEFAULT '',
    length_m       real NOT NULL,
    line           geography(LineString, 4326) NOT NULL,
    one_way_known  boolean NOT NULL DEFAULT false,
    updated_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS map_road_edges_line_idx ON map_road_edges USING gist (line);

CREATE TABLE IF NOT EXISTS map_edge_passes (
    edge_key  text NOT NULL REFERENCES map_road_edges (edge_key) ON DELETE CASCADE,
    dir       smallint NOT NULL CHECK (dir IN (1, -1)),
    day       date NOT NULL,
    driver_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    passes    integer NOT NULL,
    PRIMARY KEY (edge_key, dir, day, driver_id)
);
CREATE INDEX IF NOT EXISTS map_edge_passes_day_idx ON map_edge_passes (day);

CREATE TABLE IF NOT EXISTS map_edge_speed (
    edge_key  text NOT NULL REFERENCES map_road_edges (edge_key) ON DELETE CASCADE,
    hour      smallint NOT NULL CHECK (hour BETWEEN 0 AND 23),
    weekend   boolean NOT NULL,
    passes    integer NOT NULL,
    speed_sum double precision NOT NULL,
    PRIMARY KEY (edge_key, hour, weekend)
);

CREATE TABLE IF NOT EXISTS map_road_days (
    day          date PRIMARY KEY,
    passes       integer NOT NULL DEFAULT 0,
    edges        integer NOT NULL DEFAULT 0,
    drivers      integer NOT NULL DEFAULT 0,
    processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS map_oneway_candidates (
    id          bigserial PRIMARY KEY,
    edge_key    text NOT NULL UNIQUE,
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    way_id      bigint NOT NULL DEFAULT 0,
    name        text NOT NULL DEFAULT '',
    -- بترتيب الاتّجاه الغالب: هو اتّجاهُ `oneway=yes`.
    line        geography(LineString, 4326) NOT NULL,
    dir         smallint NOT NULL CHECK (dir IN (1, -1)),
    main_passes integer NOT NULL,
    opp_passes  integer NOT NULL,
    drivers     integer NOT NULL,
    days        integer NOT NULL,
    share       real NOT NULL,
    detected_at timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    decided_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    decided_at  timestamptz
);
CREATE INDEX IF NOT EXISTS map_oneway_candidates_status_idx ON map_oneway_candidates (status);
CREATE INDEX IF NOT EXISTS map_oneway_candidates_line_idx ON map_oneway_candidates USING gist (line);

CREATE TABLE IF NOT EXISTS map_road_runs (
    id          bigserial PRIMARY KEY,
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    days_done   integer NOT NULL DEFAULT 0,
    passes      integer NOT NULL DEFAULT 0,
    oneway      integer NOT NULL DEFAULT 0,
    error       text NOT NULL DEFAULT ''
);
