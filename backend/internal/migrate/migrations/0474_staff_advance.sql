-- ══════════════════════════════════════════════════════════════════════
-- **السلفة** — قرارُ المالك ٢٠٢٦-١٠-٢٦
-- ══════════════════════════════════════════════════════════════════════
--
-- «إذا موظّف أو سائق أو مندوب طلب سلفة لازم نقدر نضيفها، فتصير سالب عليه
-- وتنقطع من محفظته.» **المكتبُ يعطيه نقداً من الدرج**، ومحفظتُه تنقص بالمبلغ
-- (ولو صارت سالبة)، **وكلُّ مالٍ يدخلها بعد ذلك يسدّ السالبَ أوّلاً**.
--
-- # ولماذا «سقفُ سالبٍ» لا فتحُ القيد
--
-- `wallets_balance_check` يقول `balance >= 0 OR is_treasury` — **وفتحُه لواحدٍ
-- يفتحه للجميع**: أيُّ خصمٍ (دفعُ طلب، عقوبة) يُنزل أيَّ محفظةٍ تحت الصفر.
-- **فعمودٌ واحد**: `advance_owed` = ما بقي من السلفة غيرَ مسدَّد، **والقيدُ
-- يسمح بالسالب حتّى حدّه لا أكثر.** ومن كان عليه −٤٠٠ من سلفةٍ **لا يقدر
-- أيُّ خصمٍ آخر أن ينزله إلى −٤٠١.**
--
-- # والعمودُ لا يكتبه أحدٌ إلّا السلفة — ويتقلّص وحدَه
--
-- المُطلِقُ (`wallets_advance_clamp`) يُبقيه `≤ max(0, −balance)` بعد كلّ تعديل:
-- **كلُّ إيداعٍ يرفع الرصيدَ يُنقص السقفَ معه**، فلا يعود السالبُ مسموحاً بعد
-- سداده. **ولا يرفعه إلّا قيدُ السلفة** (`wallet.AdvanceTx`) في التحديث نفسِه.

-- ── ١ · نوعُ القيد ────────────────────────────────────────────────────
SELECT wallet_kinds_add('advance');

-- ── ٢ · نوعُ الطلب ────────────────────────────────────────────────────
ALTER TABLE wallet_requests DROP CONSTRAINT IF EXISTS wallet_requests_kind_check;
ALTER TABLE wallet_requests ADD CONSTRAINT wallet_requests_kind_check
    CHECK (kind IN ('topup', 'compensation', 'adjustment', 'advance'));

-- **والخصمُ كان للتسوية وحدَها** — والسلفةُ خصمٌ دائماً، ولا تكون إيداعاً.
ALTER TABLE wallet_requests DROP CONSTRAINT IF EXISTS wallet_requests_check;
ALTER TABLE wallet_requests ADD CONSTRAINT wallet_requests_check
    CHECK (kind = 'adjustment' OR (kind = 'advance') = debit);

-- ── ٣ · سقفُ السالب ───────────────────────────────────────────────────
ALTER TABLE wallets ADD COLUMN IF NOT EXISTS advance_owed bigint NOT NULL DEFAULT 0;

ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_advance_owed_check;
ALTER TABLE wallets ADD CONSTRAINT wallets_advance_owed_check CHECK (advance_owed >= 0);

-- **وافتراضُه صفرٌ** — فكلُّ محفظةٍ قائمةٍ يبقى قيدُها `balance >= 0` بعينه.
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_balance_check;
ALTER TABLE wallets ADD CONSTRAINT wallets_balance_check
    CHECK (balance >= -advance_owed OR is_treasury);

-- **ومن رصيدُه سالبٌ لا محجوزَ له** — والصفرُ لا يُقارَن بالسالب.
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_reserved_check;
ALTER TABLE wallets ADD CONSTRAINT wallets_reserved_check
    CHECK (reserved >= 0 AND (reserved = 0 OR reserved <= balance OR is_treasury));

CREATE OR REPLACE FUNCTION wallets_advance_clamp() RETURNS trigger AS $$
BEGIN
    NEW.advance_owed := LEAST(NEW.advance_owed, GREATEST(0, -NEW.balance));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS wallets_advance_clamp ON wallets;
CREATE TRIGGER wallets_advance_clamp
    BEFORE INSERT OR UPDATE ON wallets
    FOR EACH ROW EXECUTE FUNCTION wallets_advance_clamp();
