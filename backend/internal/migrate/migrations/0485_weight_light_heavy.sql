-- 0485 · «الوزن» يصير نوعين (قرارُ المالك ٢٠٢٦-١٠-٢٩)
--
--   «وزن خفيف»  من كيلو وتحت — للبهارات والعطارة، والسعرُ لكلّ ١٠٠ غ.
--   «وزن ثقيل»  من نصف كيلو وفوق — للخضرة والفواكه، والسعرُ لكلّ كيلو.
--
-- «الوزن» يصير «وزن خفيف» **بمعرّفه نفسِه** — فما اختاره متجرٌ أو رُبط بتصنيفٍ يبقى
-- قائماً. وتُبدَّل خياراتُه إن بقيت كما زرعتها 0482 (لم تعدّلها اللوحة). والأصنافُ
-- المحفوظةُ لا تُمسّ: المجموعةُ نسخةٌ في الصنف لا رابطٌ بالقالب.

UPDATE option_presets
   SET options = '[{"name":"50 غ","grams":50,"price_delta":0},{"name":"100 غ","grams":100,"price_delta":0},{"name":"250 غ","grams":250,"price_delta":0},{"name":"نصف كيلو","grams":500,"price_delta":0},{"name":"كيلو","grams":1000,"price_delta":0}]'::jsonb
 WHERE name = 'الوزن'
   AND options = '[{"name":"250 غ","grams":250,"price_delta":0},{"name":"نصف كيلو","grams":500,"price_delta":0},{"name":"كيلو","grams":1000,"price_delta":0}]'::jsonb;

UPDATE option_presets SET name = 'وزن خفيف'
 WHERE name = 'الوزن' AND NOT EXISTS (SELECT 1 FROM option_presets WHERE name = 'وزن خفيف');

INSERT INTO option_presets (name, min_select, max_select, options, sort_order, price_mode)
SELECT 'وزن ثقيل', 1, 1,
       '[{"name":"نصف كيلو","grams":500,"price_delta":0},{"name":"كيلو","grams":1000,"price_delta":0},{"name":"كيلو ونصف","grams":1500,"price_delta":0},{"name":"2 كيلو","grams":2000,"price_delta":0},{"name":"3 كيلو","grams":3000,"price_delta":0},{"name":"5 كيلو","grams":5000,"price_delta":0}]'::jsonb,
       0, 'weight'
 WHERE NOT EXISTS (SELECT 1 FROM option_presets WHERE name = 'وزن ثقيل');

-- الترتيبُ: الوزنان معاً بعد الحجم.
UPDATE option_presets p SET sort_order = v.ord
  FROM (VALUES ('الحجم', 1), ('وزن خفيف', 2), ('وزن ثقيل', 3), ('نمرة رجالي', 4), ('نمرة نسائي', 5),
               ('نمرة أطفال', 6), ('المقاس', 7), ('مقاس أطفال', 8), ('اللون', 9), ('العبوة', 10),
               ('الإضافات', 11), ('بدون', 12)
       ) AS v(name, ord)
 WHERE p.name = v.name;

-- الافتراضُ لكلّ تصنيف — يُضاف ولا يُرفع ما ربطته اللوحة.
INSERT INTO category_option_presets (category_id, preset_id)
SELECT c.id, p.id
  FROM (VALUES
    ('بهارات وعطارة', 'وزن خفيف'),
    ('خضار وفواكه', 'وزن ثقيل'), ('خضروات', 'وزن ثقيل'), ('فواكه', 'وزن ثقيل')
  ) AS m(cat, preset)
  JOIN categories c ON c.name = m.cat
  JOIN option_presets p ON p.name = m.preset
ON CONFLICT DO NOTHING;
