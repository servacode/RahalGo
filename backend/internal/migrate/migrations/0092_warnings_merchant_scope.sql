-- **الإنذارُ يُنسب إلى حسابٍ أو إلى متجر — وأحدُهما يكفي.**
--
-- (تتمّةُ توحيد الإنذارات ٢٠٢٦-٠٨-٠٩.)
--
-- # ما كشفته المجموعة
--
-- **الجدولُ الموحَّدُ كان يشترط حساباً** (`user_id NOT NULL`) — **ومتجرٌ بلا
-- حسابِ صاحبٍ لا يُنذَر أصلاً**، فتسقط مخالفاتُه من العدّاد بصمت.
--
-- **وسقط اختباران** عرّفا متجراً بلا صاحب: «امتناعُ المتجر لم يُحسب عليه».
--
-- # والمخالفةُ للكيان والإشعارُ للحساب
--
-- **المتجرُ هو من رفض التسليم** — والحظرُ يقع عليه، **والعدّادُ يعدّ سلوكَ
-- المحلّ لا سلوكَ شخص.** وصاحبُه يُشعَر إن كان له صاحب.
--
-- **وهما واقعتان لا واحدة**: من أُنذر يُخبَر إن أمكن، **ومن خالف يُعدّ عليه
-- دائماً.**
ALTER TABLE warnings
  ALTER COLUMN user_id DROP NOT NULL,
  ADD COLUMN IF NOT EXISTS merchant_id uuid REFERENCES merchants(id) ON DELETE CASCADE;

-- **ولا إنذارَ بلا صاحبٍ ولا كيان** — صفٌّ لا يُنسب إلى شيءٍ لا يُقرأ ولا يُعدّ.
ALTER TABLE warnings DROP CONSTRAINT IF EXISTS warnings_has_subject;
ALTER TABLE warnings
  ADD CONSTRAINT warnings_has_subject
  CHECK (user_id IS NOT NULL OR merchant_id IS NOT NULL);

-- **وما نُقل من القديم يُنسَب إلى متجره أيضاً** — فيَعدّه العدّادُ كما كان.
UPDATE warnings w
SET merchant_id = (SELECT m.id FROM merchants m WHERE m.owner_user_id = w.user_id LIMIT 1)
WHERE w.role_code = 'merchant' AND w.merchant_id IS NULL AND w.user_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS warnings_merchant_idx
  ON warnings (merchant_id, created_at DESC) WHERE merchant_id IS NOT NULL;
