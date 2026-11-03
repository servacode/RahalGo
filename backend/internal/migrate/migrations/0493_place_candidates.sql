-- ══════════════════════════════════════════════════════════════════════
-- **أماكن بلا اسم** (قرارُ المالك ٢٠٢٦-١١-٠٣ «اي موافق»)
-- ══════════════════════════════════════════════════════════════════════
--
-- مرشَّحو معالمَ من مصدرين:
--   `stops`     وقفاتُ السائقين تتكرّر عند مكانٍ بلا معلمٍ ولا متجر (٣ أيّامٍ أو ٣ سائقين).
--   `addresses` عباراتُ «جانب صيدلية الشفاء» في عناوين الزبائن، يتّفق عليها زبونان فأكثر.
--
-- المدير يسمّي ويوافق فيدخل `map_landmarks` بمصدر `drivers`، أو يرفض فلا يُعاد الاقتراح.
-- لا اسمَ زبونٍ ولا هاتفَ هنا: `samples` نصوصُ عناوين ممسوحٌ منها كلُّ رقمٍ طويل.
CREATE TABLE IF NOT EXISTS map_place_candidates (
    id          bigserial PRIMARY KEY,
    source      text NOT NULL CHECK (source IN ('stops', 'addresses')),
    status      text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    point       geography(Point, 4326) NOT NULL,
    name        text NOT NULL DEFAULT '',
    name_key    text NOT NULL DEFAULT '',
    kind        text NOT NULL DEFAULT '',
    visits      integer NOT NULL DEFAULT 0,
    drivers     integer NOT NULL DEFAULT 0,
    customers   integer NOT NULL DEFAULT 0,
    days        integer NOT NULL DEFAULT 0,
    evidence    integer NOT NULL DEFAULT 0,
    samples     text[] NOT NULL DEFAULT '{}',
    first_seen  date,
    last_seen   date,
    detected_at timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    decided_by  uuid REFERENCES users (id) ON DELETE SET NULL,
    decided_at  timestamptz,
    landmark_id bigint
);
CREATE INDEX IF NOT EXISTS map_place_candidates_point_idx ON map_place_candidates USING gist (point);
CREATE INDEX IF NOT EXISTS map_place_candidates_status_idx ON map_place_candidates (status, source);

CREATE TABLE IF NOT EXISTS map_place_runs (
    id          bigserial PRIMARY KEY,
    started_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    stop_places integer NOT NULL DEFAULT 0,
    name_places integer NOT NULL DEFAULT 0,
    error       text NOT NULL DEFAULT ''
);
