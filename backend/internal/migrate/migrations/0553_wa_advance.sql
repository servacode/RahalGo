-- ══════════════════════════════════════════════════════════════════════
-- «سلفة» من الواتساب — قرار المالك ٢٠٢٦-١١-١٩
-- ══════════════════════════════════════════════════════════════════════
--
-- المالك بيكتب بمحادثته مع نفسه «سلفة أحمد 50»، فبينكتب طلب سلفة عادي (wallet_requests kind=advance)
-- **معلّق ١٠ دقايق** وبعدين بيتطبّق لحالو بطريق الموافقة نفسه. «الغاء» بهالوقت بيرفضه.
-- **ولا قيد جديد بالدفتر** — نفس السلفة تبع اللوحة بالضبط.
--
--   auto_apply_at  وقت التطبيق التلقائي؛ فاضي لكل طلب من اللوحة.

ALTER TABLE wallet_requests ADD COLUMN IF NOT EXISTS auto_apply_at timestamptz;

CREATE INDEX IF NOT EXISTS wallet_requests_auto_apply_idx
    ON wallet_requests (auto_apply_at) WHERE status = 'pending' AND auto_apply_at IS NOT NULL;
