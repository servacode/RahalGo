-- ══════════════════════════════════════════════════════════════════════
-- **الخزينة وصندوقُ المكتب والإغلاقُ اليوميّ** — قراراتُ المالك ٢٠٢٦-١٠-٠٤
-- ══════════════════════════════════════════════════════════════════════
--
-- (المالكُ أذن صراحةً بمسّ دفتر المال لهذا: ربطُ صندوق المكتب، وكلُّ حركةٍ يدويّةٍ بطرفين.)

-- ── ٠ · إضافةُ نوعِ قيدٍ بلا محوِ أنواعِ غيره ─────────────────────────
--
-- **كلُّ هجرةٍ كانت تُسقط القيدَ وتعيد بناءه بقائمةٍ مكتوبةٍ باليد** — فهجرتان
-- تُكتبان معاً في فرعين تمحو الثانيةُ نوعَ الأولى. **فالدالّةُ تقرأ القائمةَ
-- الحيّةَ من القاعدة وتضيف إليها** — ومن أراد نوعاً نادى `wallet_kinds_add`.
CREATE OR REPLACE FUNCTION wallet_kinds_add(new_kind text) RETURNS void AS $$
DECLARE
    def   text;
    kinds text[];
BEGIN
    SELECT pg_get_constraintdef(c.oid) INTO def
      FROM pg_constraint c JOIN pg_class t ON t.oid = c.conrelid
     WHERE t.relname = 'wallet_transactions'
       AND c.conname = 'wallet_transactions_kind_check';
    IF def IS NULL THEN
        RAISE EXCEPTION 'wallet_transactions_kind_check missing';
    END IF;
    SELECT array_agg(m.k[1] ORDER BY m.n) INTO kinds
      FROM regexp_matches(def, '''([a-z_]+)''', 'g') WITH ORDINALITY AS m(k, n);
    IF new_kind = ANY (kinds) THEN
        RETURN;
    END IF;
    kinds := kinds || new_kind;
    EXECUTE 'ALTER TABLE wallet_transactions DROP CONSTRAINT wallet_transactions_kind_check';
    EXECUTE 'ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check '
         || 'CHECK (kind = ANY (ARRAY['
         || (SELECT string_agg(quote_literal(k), ', ') FROM unnest(kinds) AS k)
         || ']))';
END;
$$ LANGUAGE plpgsql;

-- ── ١ · سحبُ الأدمن من رصيد الخزينة ─────────────────────────────────
--
-- «هو المسؤول الوحيد، وحتّى لو دفع من المحفظة رح يكون واضح إنّ الأدمن سحب من
-- رصيد الخزينة». **نوعُ قيدٍ خاصٌّ به** — فلا يُحسب خسارةً ولا مصروفاً ولا ربحاً،
-- ويظهر في كشف الخزينة باسم صاحبه.
SELECT wallet_kinds_add('treasury_withdrawal');

CREATE TABLE IF NOT EXISTS treasury_withdrawals (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    amount        bigint NOT NULL CHECK (amount > 0),
    note          text NOT NULL CHECK (length(btrim(note)) > 0),
    withdrawn_by  uuid NOT NULL REFERENCES users(id),
    -- **نقدٌ أُخذ من درج المكتب** — فيخرج من الصندوق أيضاً.
    from_cashbox  boolean NOT NULL DEFAULT true,
    created_at    timestamptz NOT NULL DEFAULT now()
);

-- ── ٢ · صندوقُ المكتب ─────────────────────────────────────────────────
CREATE INDEX IF NOT EXISTS office_cash_entries_time ON office_cash_entries (created_at);

-- ── ٣ · الإغلاقُ اليوميّ ──────────────────────────────────────────────
--
-- **موظّفٌ يعدّ ويسجّل، وغيرُه يراجع.** المتوقَّعُ = المعدودُ في آخر إغلاقٍ
-- مُراجَع + ما دخل − ما خرج بعده. **والفرقُ = المعدود − المتوقَّع.**
CREATE TABLE IF NOT EXISTS office_cash_closes (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    day            date NOT NULL,                 -- يومُ دمشق
    opening        bigint NOT NULL,               -- معدودُ الإغلاق السابق
    cash_in        bigint NOT NULL DEFAULT 0,
    cash_out       bigint NOT NULL DEFAULT 0,
    expected       bigint NOT NULL,
    counted        bigint NOT NULL CHECK (counted >= 0),
    difference     bigint NOT NULL,
    amount         bigint NOT NULL CHECK (amount >= 0),   -- |الفرق| — لعقد الموافقات
    note           text NOT NULL DEFAULT '',
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected')),
    proposed_by    uuid NOT NULL REFERENCES users(id),
    decided_by     uuid REFERENCES users(id),
    decided_at     timestamptz,
    decision_note  text NOT NULL DEFAULT '',
    self_approved  boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CHECK (difference = counted - expected),
    CHECK (amount = abs(difference))
);
CREATE UNIQUE INDEX IF NOT EXISTS office_cash_closes_day
    ON office_cash_closes (day) WHERE status <> 'rejected';
CREATE UNIQUE INDEX IF NOT EXISTS office_cash_closes_one_pending
    ON office_cash_closes ((true)) WHERE status = 'pending';

-- ── ٤ · نقصُ الصندوق ──────────────────────────────────────────────────
--
-- **يُسجَّل حين يُراجَع الإغلاق**، ويُحلّ إن وُجد المال، **وإن لم يُحلّ خلال يومٍ
-- صار خسارةً على المنصّة بموافقة مدير المنصّة** — وعندها قيدُ `platform_expense`
-- في الخزينة بمرجع النقص.
CREATE TABLE IF NOT EXISTS office_cash_shortfalls (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    close_id       uuid NOT NULL UNIQUE REFERENCES office_cash_closes(id),
    amount         bigint NOT NULL CHECK (amount > 0),
    note           text NOT NULL DEFAULT '',
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected', 'resolved')),
    proposed_by    uuid NOT NULL REFERENCES users(id),
    eligible_at    timestamptz NOT NULL,
    decided_by     uuid REFERENCES users(id),
    decided_at     timestamptz,
    decision_note  text NOT NULL DEFAULT '',
    self_approved  boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS office_cash_shortfalls_pending
    ON office_cash_shortfalls (created_at) WHERE status = 'pending';

-- **وكلُّ سطرٍ في الصندوق يُحسب في إغلاقٍ واحدٍ بالضبط** — يُوسَم به حين يُسجَّل
-- الإغلاق، ويُفكّ وسمُه إن رُفض. (لا حدودَ زمنيّةٌ تفوّت سطراً كُتب في معاملةٍ
-- بدأت قبل الإغلاق وثُبّتت بعده.)
ALTER TABLE office_cash_entries
    ADD COLUMN IF NOT EXISTS close_id uuid REFERENCES office_cash_closes(id);
CREATE INDEX IF NOT EXISTS office_cash_entries_open
    ON office_cash_entries (created_at) WHERE close_id IS NULL;

-- ── ٥ · قدرةُ مدير المنصّة على الخزينة ────────────────────────────────
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r, 'treasury.manage' FROM unnest(ARRAY['admin', 'owner_super_admin']) AS r
 WHERE EXISTS (SELECT 1 FROM roles WHERE code = r)
ON CONFLICT DO NOTHING;
