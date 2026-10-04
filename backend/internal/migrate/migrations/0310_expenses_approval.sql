-- **مصروفاتُ التشغيل — قراراتُ المالك ٢٠٢٦-١٠-٠٤**
--
--   ١ · فوق سقفٍ في الإعدادات (`finance.expense_approval_threshold`) يصير المصروفُ
--       اقتراحاً يوافق عليه شخصٌ ثانٍ، وتحته يُقيَّد مباشرة.
--   ٤ · صورةُ الإيصال اختياريّة، وإلزاميّةٌ فوق السقف.
--   ٥ · الإلغاءُ بسببٍ إلزاميٍّ ومن شخصٍ غيرِ من سجّل، والملغى يبقى ظاهراً.
--
-- **والاقتراحُ في جدولٍ خاصٍّ به لا في `expenses`** — فصفُّ `expenses` يبقى
-- «مالٌ خرج من الخزينة» دائماً، وفحوصُ الدفتر (`FI-04`) لا تتغيّر: لا مصروفَ
-- بلا قيد، ولا قيدَ بلا مصروف. والجدولُ على عقد صفحة الموافقات الموحّدة.

-- ── صورةُ الإيصال: صنفٌ شخصيٌّ محميّ (يُقرأ برابطٍ موقَّع) ──────────────
ALTER TABLE media DROP CONSTRAINT IF EXISTS media_kind_check;
ALTER TABLE media ADD CONSTRAINT media_kind_check CHECK (
  kind = ANY (ARRAY[
    'merchant_logo'::text,
    'menu_item'::text,
    'menu_section'::text,
    'banner'::text,
    'avatar'::text,
    'delivery_proof'::text,
    'platform_logo'::text,
    'auth_background'::text,
    'site_background'::text,
    'expense_receipt'::text
  ])
);

-- ── أعمدةُ المصروف الجديدة ───────────────────────────────────────────
ALTER TABLE expenses
    ADD COLUMN IF NOT EXISTS receipt_media_id uuid REFERENCES media (id) ON DELETE SET NULL,
    -- سببُ الإلغاء — إلزاميٌّ لكلّ إلغاءٍ جديد.
    ADD COLUMN IF NOT EXISTS void_reason text NOT NULL DEFAULT '',
    -- من وافق على المصروف إن جاء من اقتراح (فوق السقف).
    ADD COLUMN IF NOT EXISTS approved_by uuid REFERENCES users (id),
    -- وافق صاحبُ الاقتراح على نفسه (المالكُ وحدَه وبلا بديل).
    ADD COLUMN IF NOT EXISTS self_approved boolean NOT NULL DEFAULT false,
    -- ألغاه من سجّله (المالكُ وحدَه وبلا بديل).
    ADD COLUMN IF NOT EXISTS void_self_approved boolean NOT NULL DEFAULT false;

-- ── اقتراحاتُ المصروف فوق السقف ──────────────────────────────────────
CREATE TABLE IF NOT EXISTS expense_requests (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status           text NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'rejected')),
    category_id      uuid NOT NULL REFERENCES expense_categories (id),
    amount           bigint NOT NULL CHECK (amount > 0),
    note             text NOT NULL DEFAULT '',
    spent_at         date NOT NULL,
    receipt_media_id uuid REFERENCES media (id) ON DELETE SET NULL,
    proposed_by      uuid NOT NULL REFERENCES users (id),
    decided_by       uuid REFERENCES users (id),
    decided_at       timestamptz,
    decision_note    text NOT NULL DEFAULT '',
    self_approved    boolean NOT NULL DEFAULT false,
    -- المصروفُ الذي وُلد من الموافقة.
    expense_id       uuid REFERENCES expenses (id),
    created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS expense_requests_pending_idx
    ON expense_requests (created_at DESC) WHERE status = 'pending';
