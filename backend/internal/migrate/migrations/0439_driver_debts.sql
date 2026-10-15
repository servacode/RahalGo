-- ══════════════════════════════════════════════════════════════════════
-- **ديونُ السائق · وباقي العقوبة · وباقي النزاع** (قرارُ المالك ٢٠٢٦-١٠-١٥)
-- ══════════════════════════════════════════════════════════════════════
--
-- كان الدينُ على المتجر والمندوب وحدَهما. **فالسائقُ الذي عوقب بأكثرَ من رصيده
-- كانت عقوبتُه تُردّ كلُّها**، **والنزاعُ الذي لا يغطّيه رصيدُ الطرف كان يبقى
-- مفتوحاً بلا خصمٍ آليّ.** وصار:
--
--   driver             طرفٌ ثالثٌ للدين — `party_id` = `users.id`
--   penalty_remainder  ما زاد من عقوبةٍ على المتاح في محفظته
--   dispute_remainder  ما زاد من مطالبةِ نزاعٍ على المتاح في محفظة الطرف
--
-- **والدينُ ينقطع كاملاً من أوّل أرباحٍ جاية** — أجرٌ أو مكافأةٌ أو تعويض.

ALTER TABLE financial_obligations DROP CONSTRAINT IF EXISTS financial_obligations_party_kind_check;
ALTER TABLE financial_obligations ADD CONSTRAINT financial_obligations_party_kind_check
    CHECK (party_kind IN ('merchant', 'rep', 'driver'));

ALTER TABLE financial_obligations DROP CONSTRAINT IF EXISTS financial_obligations_cause_check;
ALTER TABLE financial_obligations ADD CONSTRAINT financial_obligations_cause_check
    CHECK (cause IN (
        'refund_merchant_earning',
        'refund_rep_commission',
        'returned_goods',
        'legacy_opening',
        'merchant_delivery_fee',
        'penalty_remainder',
        'dispute_remainder'));

-- **ومن أين نشأ دينٌ بلا طلب** — معرّفُ العقوبة (`incentives.id`) أو النزاع
-- (`disputes.id`). **فلا يبقى دينٌ لا يُقال من أين** (`FI-13.d`).
ALTER TABLE financial_obligations ADD COLUMN IF NOT EXISTS source_ref text;

-- **ونزاعان على الطلب نفسِه لا يُدمجان في دينٍ واحد** — الأصلُ مصدرُه لا طلبُه.
DROP INDEX IF EXISTS obligations_origin_uq;
CREATE UNIQUE INDEX IF NOT EXISTS obligations_origin_uq
    ON financial_obligations (party_kind, party_id, order_id, cause)
    WHERE order_id IS NOT NULL AND source_ref IS NULL;

-- **ولا دينان من مصدرٍ واحد** — إعادةُ النداء لا تزيد ديناً.
CREATE UNIQUE INDEX IF NOT EXISTS obligations_source_uq
    ON financial_obligations (cause, source_ref)
    WHERE source_ref IS NOT NULL;

-- **صورةُ دين السائق المحفوظة** — كـ`users.commission_debt` للمندوب، **وعمودٌ
-- مستقلٌّ** لأنّ الحسابَ قد يحمل الدورين، وصورتان في عمودٍ واحدٍ لا تُطابَقان.
ALTER TABLE users ADD COLUMN IF NOT EXISTS driver_debt bigint NOT NULL DEFAULT 0;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_driver_debt_ck') THEN
        ALTER TABLE users ADD CONSTRAINT users_driver_debt_ck CHECK (driver_debt >= 0);
    END IF;
END $$;
