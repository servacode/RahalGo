-- ══════════════════════════════════════════════════════════════════════
-- **أسماءُ الأحياء والشوارع من عناوين الزبائن** (قرارُ المالك ٢٠٢٦-١١-٠٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «بالضبط من عناوين الزباين راح نستفيد من اسماء الاحياء والشوارع».
--
-- المرشَّحون يدخلون جدولَ المرشَّحين نفسَه (`map_place_candidates`) بمصدرين جديدين:
--   `neighbourhoods`  حيٌّ ذكره ٣ زبائنَ فأكثر: نقطةُ تسمية + غلافٌ محدَّب (`geom`) + نصفُ قطر.
--   `streets`         شارعٌ لصق عليه زبونان فأكثر بأغلبيّة: خطٌّ (`geom`) + معرّفاتُ الطرق (`way_ids`).
-- والموافَقُ عليه يُنسخ إلى `map_area_names` — **الجدولُ الذي تقرؤه الخرائطُ** وتصدّره دفعةُ بناء البلاطات.
ALTER TABLE map_place_candidates DROP CONSTRAINT IF EXISTS map_place_candidates_source_check;
ALTER TABLE map_place_candidates ADD CONSTRAINT map_place_candidates_source_check
    CHECK (source IN ('stops', 'addresses', 'neighbourhoods', 'streets'));
ALTER TABLE map_place_candidates ADD COLUMN IF NOT EXISTS geom geography(Geometry, 4326);
ALTER TABLE map_place_candidates ADD COLUMN IF NOT EXISTS radius_m real NOT NULL DEFAULT 0;
ALTER TABLE map_place_candidates ADD COLUMN IF NOT EXISTS way_ids bigint[] NOT NULL DEFAULT '{}';

ALTER TABLE map_place_runs ADD COLUMN IF NOT EXISTS neighbourhoods integer NOT NULL DEFAULT 0;
ALTER TABLE map_place_runs ADD COLUMN IF NOT EXISTS streets integer NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS map_area_names (
    id           bigserial PRIMARY KEY,
    kind         text NOT NULL CHECK (kind IN ('neighbourhood', 'street')),
    name         text NOT NULL,
    name_key     text NOT NULL DEFAULT '',
    point        geography(Point, 4326) NOT NULL,
    geom         geography(Geometry, 4326),
    radius_m     real NOT NULL DEFAULT 0,
    way_ids      bigint[] NOT NULL DEFAULT '{}',
    candidate_id bigint,
    source       text NOT NULL DEFAULT 'customers',
    created_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS map_area_names_point_idx ON map_area_names USING gist (point);
CREATE INDEX IF NOT EXISTS map_area_names_geom_idx ON map_area_names USING gist (geom);
CREATE INDEX IF NOT EXISTS map_area_names_kind_idx ON map_area_names (kind, name_key);
