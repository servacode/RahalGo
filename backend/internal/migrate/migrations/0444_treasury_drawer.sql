-- ══════════════════════════════════════════════════════════════════════
-- **درجُ المكتب — قراراتُ المالك ٢٠٢٦-١٠-١٥ (الخزينة)**
-- ══════════════════════════════════════════════════════════════════════
--
-- (المالكُ أذن صراحةً بمسّ دفتر المال لهذا.)
--
--   رصيدُ البداية   **إغلاقٌ مُعتمَدٌ من نوع `opening`** — ما عُدّ في الدرج يومَ بدأ التتبّع.
--                   يبتلع سطورَ ما قبله فيصير المتوقَّعُ هو المعدود، **ومرّةً واحدة**:
--                   لا يُكتب بعد إغلاقٍ مُعتمَد. **ولا يحجز يومَ الإغلاق اليوميّ.**
--   منين دفعت؟     **المصروفُ كاشٌ من الدرج أو من برّا الدرج** (`paid_from`). الكاشُ يُخرج
--                   سطرَ `expense_cash` من الصندوق، وإلغاؤه يُدخل `expense_cash_void`.
--                   **والقديمُ «من برّا»** — لم يكن للمصروف أثرٌ في الدرج يومَها.

-- ── ١ · نوعُ الإغلاق ──────────────────────────────────────────────────
ALTER TABLE office_cash_closes
    ADD COLUMN IF NOT EXISTS kind text NOT NULL DEFAULT 'daily';
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'office_cash_closes_kind_check') THEN
        ALTER TABLE office_cash_closes
            ADD CONSTRAINT office_cash_closes_kind_check CHECK (kind IN ('daily', 'opening'));
    END IF;
END $$;

-- **ويومُ الإغلاق اليوميّ لا يحجزه رصيدُ البداية** — فمن بدأ التتبّعَ صباحاً أغلق مساءً.
DROP INDEX IF EXISTS office_cash_closes_day;
CREATE UNIQUE INDEX IF NOT EXISTS office_cash_closes_day
    ON office_cash_closes (day) WHERE status <> 'rejected' AND kind = 'daily';
-- **ورصيدُ بدايةٍ واحدٌ لا غير.**
CREATE UNIQUE INDEX IF NOT EXISTS office_cash_closes_one_opening
    ON office_cash_closes ((true)) WHERE kind = 'opening';

-- ── ٢ · منين دفعت؟ ────────────────────────────────────────────────────
ALTER TABLE expenses
    ADD COLUMN IF NOT EXISTS paid_from text NOT NULL DEFAULT 'outside';
ALTER TABLE expense_requests
    ADD COLUMN IF NOT EXISTS paid_from text NOT NULL DEFAULT 'outside';
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'expenses_paid_from_check') THEN
        ALTER TABLE expenses
            ADD CONSTRAINT expenses_paid_from_check CHECK (paid_from IN ('cash', 'outside'));
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'expense_requests_paid_from_check') THEN
        ALTER TABLE expense_requests
            ADD CONSTRAINT expense_requests_paid_from_check CHECK (paid_from IN ('cash', 'outside'));
    END IF;
END $$;
