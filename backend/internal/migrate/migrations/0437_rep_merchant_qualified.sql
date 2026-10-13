-- ══════════════════════════════════════════════════════════════════════
-- **متجرُ المندوب يُحسب حين يجهز لا حين يُسجَّل** (قرارُ المالك ٢٠٢٦-١٠-١٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «المندوب مو مجرد ما يسجل المتجر ينحسبله — لازم يحقق الشروط: ساعات الدوام،
-- وموقعه على الخريطة، وعدد أصناف نحدده من اللوحة.» والأصنافُ بصورها.
--
-- `rep_qualified_at` **لحظةُ اكتمال الشروط** — وبها يُعدّ في هدف الشهر.
-- **ومن اكتمل يبقى محسوباً** ولو نقص بعدها (قرارُ المالك).
--
-- **والقديمُ يبقى كما حُسب**: ما فُتح قبل اليوم وقته وقتُ تسجيله، فلا يتبدّل
-- عدُّ شهرٍ مضى ولا مكافأةٌ صُرفت.

ALTER TABLE merchants ADD COLUMN IF NOT EXISTS rep_qualified_at timestamptz;

UPDATE merchants SET rep_qualified_at = created_at
 WHERE opened_by_rep_id IS NOT NULL AND rep_qualified_at IS NULL;

CREATE INDEX IF NOT EXISTS merchants_rep_qualified_idx
    ON merchants (opened_by_rep_id, rep_qualified_at);

-- **والمنتظرُ يُمسح دوريّاً** — قليلٌ دائماً، فالفهرسُ الجزئيُّ يكفيه.
CREATE INDEX IF NOT EXISTS merchants_rep_pending_idx
    ON merchants (opened_by_rep_id) WHERE rep_qualified_at IS NULL;
