-- ══════════════════════════════════════════════════════════════════════
-- **المدنُ — سوقٌ لكلّ مدينةٍ لا سوقٌ واحدٌ لسوريا**
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «مو معقول شخصٌ بالشام يطلب من الرقّة».)
--
-- # ولماذا مدينةٌ لا محافظة
--
-- **والمحافظةُ تكذب على الأرض**: الطبقةُ والرقّةُ في محافظةٍ واحدةٍ
-- **وبينهما خمسةٌ وخمسون كيلومترا** — ولا متجرَ في الرقّة يوصّل إليها.
--
-- # وطبقتان لا واحدة
--
-- **المدينةُ تقول «أيَّ سوقٍ ترى»، والمنطقةُ تقول «بكم يصلك»** —
-- فصارت `delivery_zones` بنتاً للمدينة.
--
-- # ومدىً متدرّجٌ ثلاثيّ
--
-- (قرارُ المالك: «اختر الطريقة الاحترافيّة التي لا تتعبنا لاحقاً».)
--
-- **افتراضُ المنصّةِ يعمل بلا ضبط** — والرقّةُ لا تحتاج شيئاً.
-- **والمدينةُ تُلغيه إن لزم** — دمشقُ رقمٌ واحد.
-- **والمتجرُ يُلغيه استثناءً** — لمن يوصّل أبعد.
--
-- **ولا يُسأل متجرٌ عن مداه** إلّا إن أُريد استثناؤه.

CREATE TABLE IF NOT EXISTS cities (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name        text NOT NULL,
    -- **ومركزٌ ونصفُ قطرٍ يقولان أين هي** — تُعرف مدينةُ الزبون من
    -- عنوانه بلا أن يختار.
    center      geography(Point, 4326) NOT NULL,
    radius_m    integer NOT NULL DEFAULT 25000 CHECK (radius_m > 0),
    -- **ومدى التوصيل داخلها** — فارغٌ يعني «خذ افتراضَ المنصّة».
    -- **ومدينةٌ صغيرةٌ تتركه فارغاً فيغطّيها كلَّها.**
    max_delivery_m integer CHECK (max_delivery_m IS NULL OR max_delivery_m > 0),
    active      boolean NOT NULL DEFAULT true,
    sort_order  integer NOT NULL DEFAULT 0,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS cities_center_idx ON cities USING gist (center);
CREATE UNIQUE INDEX IF NOT EXISTS cities_name_key ON cities (name);

-- ══════════════════════════════════════════════════════════════════════
-- **والمتجرُ ينتمي إلى مدينةٍ واحدة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **و`NULL` تعني «لم يُصنَّف بعد»** — ولا تظهر أصنافُه لأحد. **فمتجرٌ
-- بلا مدينةٍ يُخفى لا يُعرض للجميع**: الظهورُ الخاطئُ أسوأُ من الغياب.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS city_id uuid REFERENCES cities(id);
-- **ومدىً خاصٌّ بالمتجر** — استثناءٌ نادرٌ يُلغي مدى مدينته.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS max_delivery_m integer
    CHECK (max_delivery_m IS NULL OR max_delivery_m > 0);
CREATE INDEX IF NOT EXISTS merchants_city_idx ON merchants (city_id) WHERE status = 'active';

ALTER TABLE delivery_zones ADD COLUMN IF NOT EXISTS city_id uuid REFERENCES cities(id);
CREATE INDEX IF NOT EXISTS delivery_zones_city_idx ON delivery_zones (city_id);

-- ══════════════════════════════════════════════════════════════════════
-- **ويُوضع القائمُ في الرقّة**
-- ══════════════════════════════════════════════════════════════════════
--
-- **والبياناتُ اليومَ متجرٌ واحدٌ وثلاثُ مناطقَ في مركز الرقّة** — قِيس
-- ٢٠٢٦-٠٨-٢٠. **وأرخصُ وقتٍ لإضافة بُعدٍ جغرافيٍّ هو حين لا تكون هناك
-- بيانات.**
--
-- **ونصفُ قطرٍ خمسةٌ وعشرون كيلومترا** يغطّي الرقّةَ وضواحيَها.
INSERT INTO cities (name, center, radius_m, sort_order)
VALUES ('الرقة', ST_SetSRID(ST_MakePoint(39.0094, 35.9506), 4326)::geography, 25000, 1)
ON CONFLICT (name) DO NOTHING;

UPDATE merchants
   SET city_id = (SELECT id FROM cities WHERE name = 'الرقة')
 WHERE city_id IS NULL;

UPDATE delivery_zones
   SET city_id = (SELECT id FROM cities WHERE name = 'الرقة')
 WHERE city_id IS NULL;
