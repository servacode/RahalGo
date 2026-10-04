-- ══════════════════════════════════════════════════════════════════════
-- **الديون — تُسدَّد وتُشطب بيد المكتب** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- # ما كان
--
-- **الدينُ لا يُسدَّد إلّا من مستحقّاتٍ قادمة.** متجرٌ لا يبيع إلّا «لدي
-- توصيلة» بلغ سقفَ دينه **فمُنع من التوصيل للأبد** ولو جاب المال، ومندوبٌ
-- ترك العمل بقي دينُه مفتوحاً بلا شطب. **ولا زرَّ للمكتب.**
--
-- # ما صار
--
--   ١ · «دفع نقدي بالمكتب» — نقدٌ يدخل صندوقَ المكتب ويعود للخزينة.
--   ٢ · شحنُ المحفظة يسدّ الدينَ أوّلاً — ويُبلَّغ صاحبُه.
--   ٣ · الشطب — الماليّةُ تقترح بسببٍ مكتوب ومديرُ المنصّة يوافق.
--
-- **والأوّلُ والثالثُ اقتراحٌ يوافق عليه غيرُ مقترحه** — كحركة المحفظة اليدويّة.

-- ── ١ · طريقةُ التسوية تُكتب ولا تُخمَّن ─────────────────────────────────
--
-- **كان الإسقاطُ (توصيلةٌ أُلغيت) سطراً بلا قيدِ دفتر** — والصفحةُ تعرضه
-- «مسدَّداً» كأنّ المتجر دفع. **فصارت الطريقةُ عموداً.**
ALTER TABLE obligation_settlements
    ADD COLUMN IF NOT EXISTS method text NOT NULL DEFAULT 'earning';
ALTER TABLE obligation_settlements DROP CONSTRAINT IF EXISTS obligation_settlements_method_ck;
ALTER TABLE obligation_settlements ADD CONSTRAINT obligation_settlements_method_ck
    CHECK (method IN ('earning', 'voided', 'office_cash', 'wallet_topup', 'written_off'));
-- **ومرجعُ ما سدّه** — طلبُ الدفع أو الشطب أو طلبُ الشحن.
ALTER TABLE obligation_settlements
    ADD COLUMN IF NOT EXISTS ref text NOT NULL DEFAULT '';

-- **والقديمُ يُصنَّف من أثره**: الإسقاطُ وحدَه كان يُكتب بلا قيد.
UPDATE obligation_settlements SET method = 'voided'
 WHERE ledger_tx_id IS NULL AND method = 'earning';

-- ── ٢ · طلباتُ الدفع بالمكتب والشطب ───────────────────────────────────
--
-- **بعقد الموافقات الموحّد**: id · status · amount · note · proposed_by ·
-- decided_by · created_at · decided_at.
CREATE TABLE IF NOT EXISTS obligation_requests (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    obligation_id  uuid NOT NULL REFERENCES financial_obligations (id) ON DELETE CASCADE,
    kind           text NOT NULL CHECK (kind IN ('office_cash', 'write_off')),
    amount         bigint NOT NULL CHECK (amount > 0),
    -- **السببُ المكتوب** — إلزاميٌّ للاثنين.
    note           text NOT NULL CHECK (length(btrim(note)) > 0),
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected')),
    proposed_by    uuid NOT NULL REFERENCES users (id),
    decided_by     uuid REFERENCES users (id),
    decided_at     timestamptz,
    decision_note  text NOT NULL DEFAULT '',
    self_approved  boolean NOT NULL DEFAULT false,
    created_at     timestamptz NOT NULL DEFAULT now()
);
-- **طلبٌ معلَّقٌ واحدٌ لكلّ دين** — فلا يُدفع الدينُ نفسُه مرّتين باقتراحين.
CREATE UNIQUE INDEX IF NOT EXISTS obligation_requests_one_pending
    ON obligation_requests (obligation_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS obligation_requests_pending
    ON obligation_requests (created_at) WHERE status = 'pending';

-- ── ٣ · موافقةُ الشطب لمدير المنصّة ومالكها ───────────────────────────────
INSERT INTO role_capabilities (role_code, capability_code)
SELECT r, 'finance.writeoff.approve' FROM unnest(ARRAY['admin', 'owner_super_admin']) AS r
 WHERE EXISTS (SELECT 1 FROM roles WHERE code = r)
ON CONFLICT DO NOTHING;
