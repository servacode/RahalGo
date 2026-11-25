-- ══════════════════════════════════════════════════════════════════════
-- **شحن محفظة المتجر بشام كاش — ورصيد شحن ما بينسحب** — قرار المالك ٢٠٢٦-١١-٢٦
-- ══════════════════════════════════════════════════════════════════════
--
-- المتجر صار يقدر يشحن محفظته بشام كاش متل الزبون. بس محفظة المتجر بتنسحب، والشحن من برّا
-- لو انسحب بيصير المنصّة قناة لتصريف مال. فـ:
--
--   wallets.topup_locked   «رصيد شحن» — جزء من الرصيد (مو فوقه) بينصرف عالتوصيل والديون بس،
--                          وما بينسحب: طلب السحب ما بيتجاوز (الرصيد − المحجوز − رصيد الشحن).
--                          بيزيد بموافقة شحن شام كاش (لأيّ حدا — الزبون ما بيسحب أصلاً)،
--                          وبينقص أوّلاً بأجرة التوصيل والديون والعقوبات، ولآخر شي بغيرها.
--   wallet_transactions.locked_delta
--                          قدّيش غيّر هالقيد برصيد الشحن — فمجموعه لكل محفظة = topup_locked
--                          (الثابت FI-16.b)، والاسترجاع بيرجّع لرصيد الشحن اللي انصرف منه بنفس المرجع.
--   shamcash_topups.role   مين طلب: زبون أو متجر — للّوحة وتنبيه المالك.
--
-- **ولا ترحيل للقديم**: شحن شام كاش للزبون لسّا ما نزل عالإنتاج، والقديم عالتجهيز يبقى بلا قفل.

ALTER TABLE wallets ADD COLUMN IF NOT EXISTS topup_locked bigint NOT NULL DEFAULT 0;
ALTER TABLE wallets DROP CONSTRAINT IF EXISTS wallets_topup_locked_check;
ALTER TABLE wallets ADD CONSTRAINT wallets_topup_locked_check
    CHECK (topup_locked >= 0 AND topup_locked <= GREATEST(balance, 0));

ALTER TABLE wallet_transactions ADD COLUMN IF NOT EXISTS locked_delta bigint NOT NULL DEFAULT 0;
-- **فهرس صغير** — بس للقيود اللي مسّت رصيد الشحن (الاسترجاع بيسأل عنها بالمرجع).
CREATE INDEX IF NOT EXISTS wallet_tx_locked_ref
    ON wallet_transactions (user_id, ref) WHERE locked_delta <> 0;

ALTER TABLE shamcash_topups ADD COLUMN IF NOT EXISTS role text NOT NULL DEFAULT 'customer';
ALTER TABLE shamcash_topups DROP CONSTRAINT IF EXISTS shamcash_topups_role_check;
ALTER TABLE shamcash_topups ADD CONSTRAINT shamcash_topups_role_check
    CHECK (role IN ('customer', 'merchant'));
