-- 0489 · «مقاس بنطلون» بالأرقام (قرارُ المالك ٢٠٢٦-١٠-٣٠)
--
-- كتيرٌ من محلّات الألبسة يبيع البنطلون بالرقم (28، 30… 44) لا بـ S وM.
-- خيارٌ بلا سعر (كالمقاس والنمرة)، ومربوطٌ بتصنيفَي الألبسة. **ويُضاف ولا يُمسّ
-- ما عدّلته اللوحة**: إن وُجد بالاسم نفسِه لا يُعاد.

INSERT INTO option_presets (name, min_select, max_select, options, sort_order, price_mode)
SELECT 'مقاس بنطلون', 1, 1,
       (SELECT jsonb_agg(jsonb_build_object('name', n::text, 'price_delta', 0) ORDER BY n)
          FROM generate_series(28, 44, 2) n),
       8, 'none'
 WHERE NOT EXISTS (SELECT 1 FROM option_presets WHERE name = 'مقاس بنطلون');

INSERT INTO category_option_presets (category_id, preset_id)
SELECT c.id, p.id
  FROM categories c
  JOIN option_presets p ON p.name = 'مقاس بنطلون'
 WHERE c.name IN ('ألبسة', 'ملابس')
ON CONFLICT DO NOTHING;
