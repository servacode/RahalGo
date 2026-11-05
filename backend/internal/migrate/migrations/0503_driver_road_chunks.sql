-- ══════════════════════════════════════════════════════════════════════
-- **مسار السائق ع الطرق اللي مشي فيها فعلاً** (طلب المالك ٢٠٢٦-١١-٠٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- «والطريق يحسب حسب طريق السائق مو ع كيفنا».
--
-- كل قطعة من `driver_trail` بتنطابق ع الشوارع **مرّة وحدة** (Valhalla `trace_attributes`) وبيتخزّن هون
-- موضع كل نقطة ع الخط المطابق والخط نفسه. منها بتنحسب كل المسافات وبيترسم كل مسار (`drivermonitor`).
-- `driver_trail` بيضل إضافة بس (بلا تعديل). **والمطابقة مطفية افتراضياً** (`drivers.road_match_enabled`).
--
--   `driver_road_chunks`   سطر لكل قطعة مطابقة (أو جزء منها عند انقطاع بالطريق). `closed = false` =
--                          ذيل مؤقّت بيتبدّل وهو عم يكبر. وبيتقلّم مع الأثر (`drivers.trail_retention_days`).
--   `driver_road_progress` مؤشّرين لكل سائق: الحيّ لقدّام (`matched_until`) والرجوع لورا (`backfill_before`)،
--                          و`frozen_before` لعقد كنّاس العدّاد (قبله ما بينمسّ)، و`dirty_from` لنقاط متأخرة.
CREATE TABLE IF NOT EXISTS driver_road_chunks (
    id          bigserial     PRIMARY KEY,
    driver_id   uuid          NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    from_at     timestamptz   NOT NULL,
    to_at       timestamptz   NOT NULL CHECK (to_at >= from_at),
    closed      boolean       NOT NULL DEFAULT true,   -- false = ذيل مؤقّت
    status      text          NOT NULL CHECK (status IN ('ok', 'no_match')),
    error_code  text          NOT NULL DEFAULT '',     -- رمز Valhalla لـ no_match، أو 'timeout'
    point_at    timestamptz[] NOT NULL,                -- كل نقطة منظّفة بالمدى، بالترتيب
    pos_m       real[]        NOT NULL,                -- متر ع line6 عند النقطة ('NaN' = بلا موضع ع الطريق)
    line6       text          NOT NULL DEFAULT '',     -- الخط المطابق (polyline6، صيغة Valhalla)
    road_m      real          NOT NULL DEFAULT 0,
    straight_m  real          NOT NULL DEFAULT 0,
    params      smallint      NOT NULL,                -- roadmatch.ParamsVersion
    took_ms     integer       NOT NULL DEFAULT 0,
    matched_at  timestamptz   NOT NULL DEFAULT now(),
    CHECK (cardinality(point_at) = cardinality(pos_m))
);
CREATE INDEX IF NOT EXISTS driver_road_chunks_driver_idx ON driver_road_chunks (driver_id, from_at);
CREATE INDEX IF NOT EXISTS driver_road_chunks_to_at_idx  ON driver_road_chunks (to_at);

CREATE TABLE IF NOT EXISTS driver_road_progress (
    driver_id       uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    matched_until   timestamptz,   -- المؤشّر الحيّ: آخر قطعة مسكّرة؛ لقدّام بس
    backfill_before timestamptz,   -- مؤشّر الرجوع: لورا لحد أول الأثر
    frozen_before   timestamptz,   -- كنّاس العدّاد خلّى ما قبله نهائي ⇒ ما بينطابق قبله
    dirty_from      timestamptz,   -- دفعة متأخرة نزلت قبل matched_until
    dirty_rev       integer NOT NULL DEFAULT 0,  -- بيزيد مع كل علامة ⇒ الدورة ما بتمحي علامة إجت بنصّها
    tail_at         timestamptz    -- آخر مطابقة مؤقّتة للذيل
);
