-- ══════════════════════════════════════════════════════════════════════
-- **نقاطُ حركة السائقين لرسم الخريطة — بلا هويّة** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «اكيد لازم نستفيد من حركة السائقين لنرسم خريطة الرقة بدقة اكثر من مرور الوقت».
-- خريطةُ OpenStreetMap ناقصةٌ حاراتٍ كثيرةً في الرقة — **والسائقون يمشون فيها كلَّ يوم.**
--
-- **ولا سائقَ ولا ساعةَ هنا**: نقطةٌ ودقّتُها وسرعتُها واتّجاهُها ويومُها — **فلا يُعرف منها
-- من مرّ ولا متى**، ويُعرف أنّ طريقاً هنا يُسلَك. **والأثرُ الشخصيُّ** (`driver_track`)
-- **يبقى ساعتين كما كان.**
--
-- **ويُحفظ الصالحُ للرسم وحدَه** (`driver_live.go`): دقيقٌ (≤١٥ م)، ماشٍ (≥١ م/ث)، غيرُ
-- مزيّف، وبعيدٌ عن النقطة السابقة (≥٨ م) فلا يتكدّس الواقف. **ويُقلَّم بعد ٩٠ يوماً.**
CREATE TABLE IF NOT EXISTS map_trace_points (
    id         bigserial PRIMARY KEY,
    point      geography(Point, 4326) NOT NULL,
    accuracy_m real,
    speed_mps  real,
    bearing    smallint,
    day        date NOT NULL DEFAULT (now() AT TIME ZONE 'Asia/Damascus')::date
);
CREATE INDEX IF NOT EXISTS map_trace_points_point_idx ON map_trace_points USING gist (point);
CREATE INDEX IF NOT EXISTS map_trace_points_day_idx ON map_trace_points (day);
