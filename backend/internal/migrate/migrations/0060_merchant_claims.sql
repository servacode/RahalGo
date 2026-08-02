-- ٠٠٦٠ · النزاعُ مع المتجر — **المنصةُ تدفع أوّلاً ثمّ تُطالِب**
--
-- # قرارُ المالك (٢٠٢٦-٠٨-٠٣)
--
-- **«نعم، المنصة تعوّضه — وبفتح نزاع مع المتجر لحلّ القصة.»**
--
-- والحالةُ: السائقُ قاد إلى المتجر، **فاعتذر المتجرُ عن الصنف**. فالمشوارُ
-- ضاع ولا ذنبَ للسائق.
--
-- # ولماذا الدفعُ قبل الحسم
--
-- **السائقُ لا ينتظر نزاعاً ليُقبض له.** ونزاعٌ يستغرق يوماً يترك من قاد
-- مشوارَه بلا مقابلٍ يومَه كلَّه، **ومن قاد بلا مقابلٍ مرّةً يتردّد في
-- الثانية.**
--
-- **والمطالبةُ تجري في مسارها** — بين المنصة والمتجر، ولا يقف عليها أحدٌ ثالث.
--
-- # وكان لا تعويضَ أصلاً
--
-- بُنيت القاعدةُ سابقاً: «ذنبُ المتجر لا تعويضَ فيه — المنصةُ تتحمّل بضاعتَه
-- وتعوّض سائقَها فلا نجمع عليها الاثنين». **وهي تظلم السائق**: يُحرم أجرَه
-- بسبب متجرٍ اعتذر متأخّراً، **وهو لا يملك من أمر ذلك شيئاً.**

-- **المطالبةُ تُسجَّل على الإنذار نفسِه** — لا في جدولٍ ثالث.
--
-- الإنذارُ يقول **ماذا فعل**، والمطالبةُ تقول **كم كلّف**. **وجدولان لواقعةٍ
-- واحدة يفترقان**: يُغلق أحدُهما ويبقى الآخر، فيُطالَب متجرٌ بما سُوّي.
ALTER TABLE merchant_warnings ADD COLUMN IF NOT EXISTS claim_amount bigint NOT NULL DEFAULT 0;

-- settlement مصيرُ المطالبة: فارغٌ = مفتوحة · `charged` = خُصمت · `waived` = أُعفيت.
--
-- **والإعفاءُ قرارٌ يُسجَّل لا صمتٌ يمرّ**: من أعفى متجراً مرّةً يُسأل عن
-- الثانية، **ومطالبةٌ تُنسى تبدو كأنّها لم تكن.**
ALTER TABLE merchant_warnings ADD COLUMN IF NOT EXISTS settlement text
    CHECK (settlement IS NULL OR settlement IN ('charged', 'waived'));
ALTER TABLE merchant_warnings ADD COLUMN IF NOT EXISTS settled_at timestamptz;
ALTER TABLE merchant_warnings ADD COLUMN IF NOT EXISTS settled_by uuid REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS merchant_warnings_open_claims
    ON merchant_warnings (merchant_id, created_at DESC)
    WHERE claim_amount > 0 AND settlement IS NULL;
