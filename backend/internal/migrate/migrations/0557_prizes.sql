-- «جوائز رحّال غو» — سحبٌ شهريٌّ على الدعوات ورصيدُ جوائز (قرارُ المالك ٢٠٢٦-١١-٢٣).
--
-- # الدعوة
--
-- **تُعدّ لحظةَ إنشاء حساب المدعوّ** — وهي صفُّ `referrals` القائم (هجرة ٠٠٧٧) بتاريخه.
-- كلُّ خمس دعواتٍ في الشهر (الإعداد `prizes.invites_per_entry`) فرصةٌ في السحب.
--
-- # السحب
--
-- أوّلَ كلّ شهرٍ الساعةَ ١٢ ظهراً بتوقيت دمشق على دعوات الشهر الماضي. **صفٌّ واحدٌ لكلّ شهر**
-- (`prize_draws.month` مفتاحٌ أوّليّ) — فإعادةُ التشغيل أو خادمان معاً لا يسحبان مرّتين.
--
-- # رصيدُ الجوائز
--
-- **غيرُ المحفظة**: لا يُسحب ولا يُصرف إلّا في متجر الجوائز، **وينتهي آخرَ الشهر الذي مُنح فيه.**
-- ومنحُه وانتهاؤه **استحقاقٌ لا حركةُ مال** — المالُ يتحرّك حين يُصرف فقط: يُكتب على الطلب
-- خصماً تتحمّله المنصّة (`orders.prize_discount`، جزءٌ من `orders.discount`) كخصم الكود تماماً،
-- **فيبقى حسابُ الخزينة الفرقيّ والدفترُ متوازنَين بلا قيدٍ جديد.**

-- ── متجرُ الجوائز: حالةٌ رابعة ─────────────────────────────────────────
--
-- **كلُّ ما يراه الزبونُ يشترط `m.status = 'active'`** — فمتجرٌ بحالة `prize` غائبٌ عن السوق
-- والبحث والأقسام والعروض والخرائط بلا شرطٍ جديد، **ويُفتح له بابُ صفحة الجوائز وحدَها.**
ALTER TABLE merchants DROP CONSTRAINT IF EXISTS merchants_status_check;
ALTER TABLE merchants ADD CONSTRAINT merchants_status_check
    CHECK (status IN ('active', 'inactive', 'suspended', 'prize'));

-- **ومتجرٌ واحدٌ لا غير.**
CREATE UNIQUE INDEX IF NOT EXISTS merchants_one_prize_store
    ON merchants ((true)) WHERE status = 'prize';

-- ── الطلب ───────────────────────────────────────────────────────────────
ALTER TABLE orders
    -- **ما دُفع من رصيد الجوائز** — داخلَ `discount` لا فوقه.
    ADD COLUMN IF NOT EXISTS prize_discount bigint NOT NULL DEFAULT 0
        CHECK (prize_discount >= 0),
    -- **طلبٌ من متجر الجوائز** — توصيلُه على المنصّة والكابتنُ بأجره كاملاً.
    ADD COLUMN IF NOT EXISTS prize_order boolean NOT NULL DEFAULT false;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_prize_within_discount;
ALTER TABLE orders ADD CONSTRAINT orders_prize_within_discount
    CHECK (prize_discount <= discount);

-- ── الرصيد ──────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS prize_balances (
    user_id uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    balance bigint NOT NULL DEFAULT 0 CHECK (balance >= 0),
    -- **آخرُ ثانيةٍ من الشهر الذي مُنح فيه** — بتوقيت دمشق.
    expires_at timestamptz,
    -- **تذكيرُ «قبل ٣ أيام» لأيّ انتهاء** — يُكتب الانتهاءُ الذي ذُكِّر به فلا يتكرّر.
    reminded_for timestamptz,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- **دفترٌ لا يُعدَّل** — منحٌ وصرفٌ وردٌّ وانتهاء.
CREATE TABLE IF NOT EXISTS prize_entries (
    id bigserial PRIMARY KEY,
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('award', 'spend', 'refund', 'expire')),
    amount bigint NOT NULL CHECK (amount <> 0),
    order_id uuid REFERENCES orders(id) ON DELETE SET NULL,
    draw_month date,
    -- **انتهاءُ الرصيد الذي مسّه القيد** — فالردُّ بعده لا يُحيي رصيداً انتهى.
    expires_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS prize_entries_user_idx ON prize_entries (user_id, id);
CREATE INDEX IF NOT EXISTS prize_entries_order_idx ON prize_entries (order_id) WHERE order_id IS NOT NULL;

-- ── السحب ───────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS prize_draws (
    -- **أوّلُ يومٍ من الشهر المسحوب عليه** — مفتاحٌ يمنع السحبَ مرّتين.
    month date PRIMARY KEY,
    drawn_at timestamptz NOT NULL DEFAULT now(),
    entrants int NOT NULL DEFAULT 0,
    total_entries int NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS prize_winners (
    month date NOT NULL REFERENCES prize_draws(month) ON DELETE CASCADE,
    rank int NOT NULL CHECK (rank > 0),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    invites int NOT NULL,
    entries int NOT NULL CHECK (entries > 0),
    amount bigint NOT NULL CHECK (amount >= 0),
    PRIMARY KEY (month, rank),
    -- **والرابحُ مرّةً واحدةً بالسحب.**
    UNIQUE (month, user_id)
);
