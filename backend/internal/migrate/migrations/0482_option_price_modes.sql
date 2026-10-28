-- ══════════════════════════════════════════════════════════════════════
-- **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- لكلّ مجموعة خياراتٍ طريقةُ سعرٍ واحدةٌ من ثلاث:
--
--   priced   «بسعر»      لكلّ خيارٍ فرقُ سعرٍ يكتبه المتجر (الحجم · العبوة · الإضافات).
--   none     «بدون سعر»  لا سعرَ للخيار أصلاً (المقاس · اللون · النمرة · بدون) —
--                         والفرقُ صفرٌ دائماً مهما أُرسل.
--   weight   «حسب الوزن» سعرُ الصنف لوزنٍ أساسيّ (١٠٠ غ افتراضاً)، ولكلّ خيارٍ
--                         وزنُه بالغرام، **وسعرُه = سعرُ الصنف × غراماته ÷ الأساس،
--                         مقرَّباً لأقرب ٥ ل.س.**
--
-- # والفرقُ يبقى مخزَّناً في `price_delta`
--
-- كلُّ ما يقرأ الخيارات اليوم (بناءُ الطلب، التسعيرة، صفحةُ الصنف، اللوحة)
-- يقرأ `price_delta` — فيبقى كما هو، **وتحسبه القاعدةُ بنفسها بالمشغّلات أدناه**:
-- فكلُّ بابٍ يكتب سعرَ الصنف (الحفظ، تغييرُ الأسعار دفعةً، سعرُ الصرف للدولار،
-- ربطُ برنامج الأمين، اللوحة، النسخ) يُصلح أسعارَ الأوزان معه بلا أن يتذكّر ذلك.
--
-- **والمجموعاتُ المحفوظةُ قبل اليوم تبقى «بسعر»** — لا تُمسّ أصنافُ المتاجر.

ALTER TABLE option_presets ADD COLUMN IF NOT EXISTS price_mode text NOT NULL DEFAULT 'priced'
    CHECK (price_mode IN ('priced', 'none', 'weight'));

ALTER TABLE modifier_groups ADD COLUMN IF NOT EXISTS price_mode text NOT NULL DEFAULT 'priced'
    CHECK (price_mode IN ('priced', 'none', 'weight'));
-- الوزنُ الذي يخصّه سعرُ الصنف — لمجموعة «حسب الوزن» وحدَها، وفراغُه ١٠٠ غ.
ALTER TABLE modifier_groups ADD COLUMN IF NOT EXISTS base_grams int
    CHECK (base_grams IS NULL OR base_grams > 0);
-- وزنُ الخيار بالغرام — لمجموعة «حسب الوزن» وحدَها.
ALTER TABLE modifier_options ADD COLUMN IF NOT EXISTS grams int
    CHECK (grams IS NULL OR grams > 0);

-- **المعادلةُ في مكانٍ واحد** — فرقُ خيار الوزن عن سعر الصنف.
CREATE OR REPLACE FUNCTION weight_option_delta(item_price bigint, grams int, base_grams int)
RETURNS bigint LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT (round(item_price::numeric * grams / COALESCE(NULLIF(base_grams, 0), 100) / 5) * 5)::bigint
           - item_price
$$;

-- **الخيارُ يُسعَّر عند كتابته** — «بدون سعر» صفر، و«حسب الوزن» من سعر صنفه.
-- والخيارُ بلا غرامات في مجموعة وزنٍ (من نسخة تطبيقٍ قديمة) يبقى بسعره المُرسَل.
CREATE OR REPLACE FUNCTION modifier_option_price_mode() RETURNS trigger
LANGUAGE plpgsql AS $$
DECLARE
    mode  text;
    base  int;
    price bigint;
BEGIN
    SELECT g.price_mode, g.base_grams, i.merchant_price
      INTO mode, base, price
      FROM modifier_groups g JOIN menu_items i ON i.id = g.item_id
     WHERE g.id = NEW.group_id;
    IF mode = 'none' THEN
        NEW.price_delta := 0;
        NEW.price_delta_usd_cents := NULL;
    ELSIF mode = 'weight' AND NEW.grams IS NOT NULL THEN
        NEW.price_delta := weight_option_delta(price, NEW.grams, base);
        NEW.price_delta_usd_cents := NULL;
    END IF;
    RETURN NEW;
END
$$;

DROP TRIGGER IF EXISTS modifier_options_price_mode ON modifier_options;
CREATE TRIGGER modifier_options_price_mode
    BEFORE INSERT OR UPDATE ON modifier_options
    FOR EACH ROW EXECUTE FUNCTION modifier_option_price_mode();

-- **وتبديلُ طريقة المجموعة أو وزنِها الأساسيّ يعيد تسعيرَ خياراتها.**
CREATE OR REPLACE FUNCTION modifier_group_price_mode() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE modifier_options SET grams = grams WHERE group_id = NEW.id;
    RETURN NULL;
END
$$;

DROP TRIGGER IF EXISTS modifier_groups_price_mode ON modifier_groups;
CREATE TRIGGER modifier_groups_price_mode
    AFTER UPDATE OF price_mode, base_grams ON modifier_groups
    FOR EACH ROW
    WHEN (OLD.price_mode IS DISTINCT FROM NEW.price_mode OR OLD.base_grams IS DISTINCT FROM NEW.base_grams)
    EXECUTE FUNCTION modifier_group_price_mode();

-- **وتبديلُ سعر الصنف — من أيّ باب — يعيد تسعيرَ أوزانه.**
CREATE OR REPLACE FUNCTION menu_item_weight_prices() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
    UPDATE modifier_options o
       SET price_delta = weight_option_delta(NEW.merchant_price, o.grams, g.base_grams)
      FROM modifier_groups g
     WHERE o.group_id = g.id AND g.item_id = NEW.id
       AND g.price_mode = 'weight' AND o.grams IS NOT NULL;
    RETURN NULL;
END
$$;

DROP TRIGGER IF EXISTS menu_items_weight_prices ON menu_items;
CREATE TRIGGER menu_items_weight_prices
    AFTER UPDATE OF merchant_price ON menu_items
    FOR EACH ROW
    WHEN (OLD.merchant_price IS DISTINCT FROM NEW.merchant_price)
    EXECUTE FUNCTION menu_item_weight_prices();

-- ── القوالب الجاهزة ───────────────────────────────────────────────────

-- # النمراتُ ثلاثٌ والمقاسُ اثنان (قرارُ المالك ٢٠٢٦-١٠-٢٨)
--
-- «النمرة» تصير «نمرة رجالي» **بمعرّفها نفسِه** — فما اختاره متجرٌ
-- (`merchants.option_preset_ids`) وما رُبط بتصنيفٍ يبقى قائماً ويشير إليها.
-- وتُبدَّل خياراتُها إن بقيت كما زُرعت (٣٨…٤٥). والأصنافُ التي حفظت مجموعة
-- «النمرة» قبل اليوم لا تُمسّ: المجموعةُ نسخةٌ في الصنف لا رابطٌ بالقالب.
UPDATE option_presets
   SET options = (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
                    FROM generate_series(39, 46) n)
 WHERE name = 'النمرة'
   AND options = (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
                    FROM generate_series(38, 45) n);
UPDATE option_presets SET name = 'نمرة رجالي'
 WHERE name = 'النمرة' AND NOT EXISTS (SELECT 1 FROM option_presets WHERE name = 'نمرة رجالي');

-- «المقاس» للكبار يزيد 3XL إن لم يكن فيه.
UPDATE option_presets
   SET options = options || '[{"name":"3XL","price_delta":0}]'::jsonb
 WHERE name = 'المقاس'
   AND NOT EXISTS (SELECT 1 FROM jsonb_array_elements(options) e WHERE e->>'name' = '3XL');

INSERT INTO option_presets (name, min_select, max_select, options, sort_order, price_mode)
SELECT v.name, 1, 1, v.opts, 0, 'none'
  FROM (VALUES
    ('نمرة رجالي', (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
                      FROM generate_series(39, 46) n)),
    ('نمرة نسائي', (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
                      FROM generate_series(35, 41) n)),
    ('نمرة أطفال', (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
                      FROM generate_series(20, 35) n)),
    ('مقاس أطفال', (SELECT jsonb_agg(jsonb_build_object('name', s, 'price_delta', 0) ORDER BY o)
                      FROM unnest(ARRAY['0–3 أشهر', '3–6 أشهر', '6–12 شهراً', '1–2 سنة', '2–3 سنوات',
                                        '3–4 سنوات', '4–5 سنوات', '5–6 سنوات', '7–8 سنوات',
                                        '9–10 سنوات', '11–12 سنة', '13–14 سنة'])
                           WITH ORDINALITY AS t(s, o)))
  ) AS v(name, opts)
 WHERE NOT EXISTS (SELECT 1 FROM option_presets p WHERE p.name = v.name);

-- الترتيبُ: النمراتُ معاً والمقاساتُ معاً.
UPDATE option_presets p SET sort_order = v.ord
  FROM (VALUES ('الحجم', 1), ('الوزن', 2), ('نمرة رجالي', 3), ('نمرة نسائي', 4), ('نمرة أطفال', 5),
               ('المقاس', 6), ('مقاس أطفال', 7), ('اللون', 8), ('العبوة', 9), ('الإضافات', 10), ('بدون', 11)
       ) AS v(name, ord)
 WHERE p.name = v.name;

-- الافتراضُ لكلّ تصنيف — يُضاف ولا يُرفع ما ربطته اللوحة.
INSERT INTO category_option_presets (category_id, preset_id)
SELECT c.id, p.id
  FROM (VALUES
    ('أحذية', 'نمرة رجالي'), ('أحذية', 'نمرة نسائي'), ('أحذية', 'نمرة أطفال'),
    ('ألبسة', 'المقاس'), ('ألبسة', 'مقاس أطفال'), ('ألبسة', 'اللون'),
    ('ملابس', 'المقاس'), ('ملابس', 'مقاس أطفال'), ('ملابس', 'اللون')
  ) AS m(cat, preset)
  JOIN categories c ON c.name = m.cat
  JOIN option_presets p ON p.name = m.preset
ON CONFLICT DO NOTHING;

UPDATE option_presets SET price_mode = 'none'
 WHERE name IN ('المقاس', 'مقاس أطفال', 'اللون', 'النمرة', 'نمرة رجالي', 'نمرة نسائي', 'نمرة أطفال', 'بدون');

UPDATE option_presets SET price_mode = 'priced'
 WHERE name IN ('الحجم', 'العبوة', 'الإضافات');

-- «الوزن» بالغرامات — وتُبدَّل خياراتُه إن بقيت كما زُرعت (لم تعدّلها اللوحة).
UPDATE option_presets
   SET price_mode = 'weight',
       options = '[{"name":"250 غ","grams":250,"price_delta":0},{"name":"نصف كيلو","grams":500,"price_delta":0},{"name":"كيلو","grams":1000,"price_delta":0}]'::jsonb
 WHERE name = 'الوزن'
   AND NOT EXISTS (
       SELECT 1 FROM jsonb_array_elements(options) e
        WHERE e->>'name' NOT IN ('ربع كيلو', 'نص كيلو', 'نصف كيلو', 'كيلو'));
UPDATE option_presets SET price_mode = 'weight' WHERE name = 'الوزن';

-- ألوانٌ أكثر — يُضاف ما ليس في القالب، بترتيبه.
UPDATE option_presets p
   SET options = p.options || COALESCE((
       SELECT jsonb_agg(jsonb_build_object('name', c, 'price_delta', 0) ORDER BY ord)
         FROM unnest(ARRAY['أسود', 'أبيض', 'كحلي', 'أحمر', 'أزرق', 'أخضر', 'رمادي', 'بيج', 'بني', 'زهري'])
              WITH ORDINALITY AS t(c, ord)
        WHERE NOT EXISTS (SELECT 1 FROM jsonb_array_elements(p.options) e WHERE e->>'name' = c)), '[]'::jsonb)
 WHERE p.name = 'اللون';
