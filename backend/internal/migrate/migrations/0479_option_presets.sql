-- ══════════════════════════════════════════════════════════════════════
-- **خياراتُ الصنف الجاهزة — لكلّ تصنيفٍ ما يناسبه** (قرارُ المالك ٢٠٢٦-١٠-٢٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- كانت ثلاثةً ثابتةً في التطبيق لكلّ المتاجر (الحجم · الوزن · إضافات أخرى)،
-- **فمحلُّ الأحذية لا يجد «النمرة» ولا محلُّ الألبسة «المقاس».**
--
-- # الطبقات الثلاث
--
--   option_presets            القائمةُ الكاملة — تُدار من اللوحة.
--   category_option_presets   ما يطلع لكلّ تصنيفٍ افتراضاً.
--   merchants.option_preset_ids  اختيارُ المتجر نفسِه — **وفراغُه (NULL)
--                             «اتبع تصنيفك»**، فمن لم يدخل الإعدادات قطّ
--                             يجد ما يناسبه لا القائمةَ كلَّها.
--
-- **والخياراتُ داخلَ القالب بدايةٌ لا قيد**: تُنسخ إلى الصنف عند التعليم،
-- ثمّ يغيّر صاحبُ المتجر أسماءها وأسعارها كما يشاء.
CREATE TABLE IF NOT EXISTS option_presets (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL,
    min_select int  NOT NULL DEFAULT 0 CHECK (min_select >= 0),
    max_select int  NOT NULL DEFAULT 1 CHECK (max_select >= 1),
    -- [{"name": "صغير", "price_delta": 0}, ...]
    options    jsonb NOT NULL DEFAULT '[]'::jsonb,
    sort_order int  NOT NULL DEFAULT 0,
    active     boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (min_select <= max_select)
);

CREATE TABLE IF NOT EXISTS category_option_presets (
    category_id uuid NOT NULL REFERENCES categories (id) ON DELETE CASCADE,
    preset_id   uuid NOT NULL REFERENCES option_presets (id) ON DELETE CASCADE,
    PRIMARY KEY (category_id, preset_id)
);

ALTER TABLE merchants ADD COLUMN IF NOT EXISTS option_preset_ids uuid[];

-- القائمةُ الكاملة كما وافق عليها المالك.
INSERT INTO option_presets (name, min_select, max_select, options, sort_order)
SELECT v.name, v.mn, v.mx, v.opts::jsonb, v.ord
FROM (VALUES
    ('الحجم',    1, 1,  '[{"name":"صغير","price_delta":0},{"name":"وسط","price_delta":0},{"name":"كبير","price_delta":0}]', 1),
    ('الوزن',    1, 1,  '[{"name":"ربع كيلو","price_delta":0},{"name":"نص كيلو","price_delta":0},{"name":"كيلو","price_delta":0}]', 2),
    ('النمرة',   1, 1,  '[{"name":"38","price_delta":0},{"name":"39","price_delta":0},{"name":"40","price_delta":0},{"name":"41","price_delta":0},{"name":"42","price_delta":0},{"name":"43","price_delta":0},{"name":"44","price_delta":0},{"name":"45","price_delta":0}]', 3),
    ('المقاس',   1, 1,  '[{"name":"S","price_delta":0},{"name":"M","price_delta":0},{"name":"L","price_delta":0},{"name":"XL","price_delta":0},{"name":"XXL","price_delta":0}]', 4),
    ('اللون',    1, 1,  '[{"name":"أسود","price_delta":0},{"name":"أبيض","price_delta":0},{"name":"كحلي","price_delta":0}]', 5),
    ('العبوة',   1, 1,  '[{"name":"حبة","price_delta":0},{"name":"علبة","price_delta":0},{"name":"كرتونة","price_delta":0}]', 6),
    ('الإضافات', 0, 99, '[]', 7),
    ('بدون',     0, 99, '[{"name":"بدون بصل","price_delta":0},{"name":"بدون ثوم","price_delta":0}]', 8)
) AS v(name, mn, mx, opts, ord)
WHERE NOT EXISTS (SELECT 1 FROM option_presets);

-- **والافتراضُ لكلّ تصنيف بالاسم** — وما لم يوجد من التصنيفات بعدُ
-- (ألبسة · أحذية) يُربط من اللوحة يومَ يُضاف.
INSERT INTO category_option_presets (category_id, preset_id)
SELECT c.id, p.id
FROM (VALUES
    ('مطاعم', 'الحجم'), ('مطاعم', 'الإضافات'), ('مطاعم', 'بدون'),
    ('بقالة', 'الوزن'), ('بقالة', 'العبوة'),
    ('حلويات', 'الوزن'), ('حلويات', 'الحجم'),
    ('صيدليات', 'العبوة'),
    ('هدايا', 'الحجم'), ('هدايا', 'اللون'),
    ('ألبسة', 'المقاس'), ('ألبسة', 'اللون'),
    ('ملابس', 'المقاس'), ('ملابس', 'اللون'),
    ('أحذية', 'النمرة'), ('أحذية', 'اللون')
) AS m(cat, preset)
JOIN categories c ON c.name = m.cat
JOIN option_presets p ON p.name = m.preset
ON CONFLICT DO NOTHING;
