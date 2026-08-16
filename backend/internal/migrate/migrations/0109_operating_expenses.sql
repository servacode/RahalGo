-- **مصروفاتُ التشغيل — ما ينفقه المكتبُ لا ما يخسره العمل.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون وعمّال وما إلى ذلك من
--  المصاريف — يجب أن تُوثَّق بشكلٍ صحيح».)
--
-- # ولماذا نوعٌ ثالثٌ في الدفتر
--
-- **في الخزينة نوعان يخرج بهما المال**:
--
--   `platform_profit`  — يحسبه المحرّكُ ويُصحّحه
--   `platform_expense` — ما قرّره إنسان: **تعويضُ سائقٍ أو بضاعةٌ لم تُسترَدّ**
--
-- **وشاشةُ «الخسائر» تقرأ الثاني كلَّه وتسمّيه خسارة.**
--
-- **وإيجارُ المكتب وراتبُ الموظّف ليسا خسارة** — هما كلفةُ تشغيلٍ مخطَّطة.
-- **والخسارةُ ما لم يكن يجب أن يقع.**
--
-- **ولو خُلطا**: تقريرُ الخسائر يتضخّم بالإيجار **فيبدو أداءُ المنصّة أسوأَ
-- ممّا هو**، ولا يُعرف كم كلّف الفشلُ فعلاً — **وهو الرقمُ الذي تُتَّخذ عليه
-- قراراتُ الحظر والتعويض.** **ونزاعاتُ الاسترداد مرتبطةٌ بالخسائر، والإيجارُ
-- لا يُطالَب به أحد.**
ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_kind_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
    CHECK (kind = ANY (ARRAY[
        'topup', 'order_payment', 'refund', 'compensation', 'commission',
        'merchant_earning', 'driver_earning', 'payout', 'adjustment',
        'platform_profit',
        'platform_expense',
        'operating_expense'  -- إيجارٌ ورواتبُ وكهرباء — كلفةُ مكتبٍ لا خسارةُ عمل
    ]));

-- **وأبوابُ المصروف تُدار ولا تُكتب في الشيفرة.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-١٦: «قائمةٌ ويمكنني الإضافة والحذف والتعديل — أفضلُ
--  من قائمةٍ ثابتة».)
--
-- **وقائمةٌ في الشيفرة تعني نشرةً جديدةً لإضافة بابٍ** — ومن احتاج باباً اليومَ
-- كتبه في «أخرى»، **فيضيع التبويبُ الذي بُنيت الشاشةُ لأجله.**
CREATE TABLE IF NOT EXISTS expense_categories (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name       text NOT NULL UNIQUE CHECK (btrim(name) <> ''),
    -- **ولا يُحذف بابٌ صُرف عليه** — يُطفأ فلا يُختار جديداً،
    -- **وحذفُه يمحو تبويبَ ما مضى** فيُقرأ تاريخُ الإنفاق ناقصاً.
    active     boolean NOT NULL DEFAULT true,
    sort_order int NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS expenses (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id uuid NOT NULL REFERENCES expense_categories (id),
    amount      bigint NOT NULL CHECK (amount > 0),
    note        text NOT NULL DEFAULT '',
    -- **ويومُ الصرف غيرُ يومِ التسجيل** — يُسجَّل إيجارُ الشهر الماضي اليوم،
    -- **وتقريرُ الشهر يُبنى على متى صُرف لا متى كُتب.**
    spent_at    date NOT NULL DEFAULT current_date,
    created_by  uuid REFERENCES users (id),
    created_at  timestamptz NOT NULL DEFAULT now(),
    -- ══════════════════════════════════════════════════════════════════
    -- **والخطأُ يُلغى ولا يُمحى**
    -- ══════════════════════════════════════════════════════════════════
    --
    -- **وقاعدةُ الدفتر عندنا**: لا يُعدَّل بل يُصحَّح بقيدٍ مضادّ. **فحذفُ
    -- الصفّ يترك قيدَه في الخزينة بلا صاحب** — مالٌ خرج ولا يُعرف لماذا.
    --
    -- **فيُوسَم ملغًى ويُقيَّد مقابلُه**، ويبقى الاثنان يُقرآن.
    voided_at   timestamptz,
    voided_by   uuid REFERENCES users (id)
);

-- **والتقريرُ شهريٌّ** — «كم أنفق المكتبُ هذا الشهر» سؤالٌ يُسأل مرّةً كلَّ شهر.
CREATE INDEX IF NOT EXISTS expenses_spent_idx ON expenses (spent_at DESC)
    WHERE voided_at IS NULL;

-- **وأبوابٌ تبدأ بها** — تُعدَّل وتُحذف وتُزاد، **وهي اقتراحٌ لا حكم.**
INSERT INTO expense_categories (name, sort_order) VALUES
    ('إيجار', 1), ('رواتب', 2), ('كهرباء ومولّدة', 3), ('إنترنت', 4),
    ('وقود', 5), ('صيانة', 6), ('قرطاسية', 7), ('أخرى', 99)
ON CONFLICT (name) DO NOTHING;
