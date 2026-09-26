-- ══════════════════════════════════════════════════════════════════════
-- **طريقةُ تسوية مستحقّات المتجر — نقداً أو محفظةً** — قرارٌ ماليٌّ معتمَد
-- ══════════════════════════════════════════════════════════════════════
--
-- (٢٠٢٦-٠٩-٢٧ · تصميمٌ رُوجع أربعَ مرّاتٍ وأُقرّ — docs/MERCHANT-SETTLEMENT-DESIGN.md.)
--
-- # مفهومان مستقلّان لا يُستنتَج أحدُهما من الآخر
--
--   طريقةُ دفعِ الزبون  (orders.payment_method: cash|wallet)  — كيف يدفع الزبون
--   طريقةُ تسويةِ المتجر (merchants.settlement_method: cash|wallet) — كيف يصل نصيبُ المتجر
--
-- **والافتراضُ نقديّ** (المتاجرُ الجديدةُ تفضّل النقدَ في اليد؛ الأدمنُ يحوّلها
-- إلى المحفظة لاحقاً). **والمتجرُ لا يغيّرها بنفسه.**
--
-- # التغييرُ للمستقبلِ وحدَه — لا يُعادُ كتابةُ الماضي
--
-- **كلُّ طلبٍ يحمل لقطةَ طريقةِ التسوية لكلّ مصدرٍ في `order_items`** — تُثبَّت
-- لحظةَ الإنشاء وتُقرأ لحظةَ التسوية (الاستلام). فتغييرُ الإعداد يمسّ الطلباتِ
-- الجديدةَ فقط.
--
-- # الهُويّةُ الموحّدةُ للتسوية: (order_id, merchant_id)
--
-- **`merchant_settlements` سجلٌّ واحدٌ لكلّ (طلب، متجر) لأيّ طريقة** — يجعل
-- «مرّةً واحدة» و«محفظةٌ أو نقدٌ لا كلاهما» بنيويّين لا فحصيّين. وهو يحلّ ثغرةَ
-- التسوية القديمة التي كانت تُميِّز الطلبَ لا المصدر (فتلتبس متجرَا مالكٍ واحد).

-- ── ١ · إعدادُ المتجر (سلطةُ الأدمن وحدَه) ──────────────────────────────
ALTER TABLE merchants
    ADD COLUMN IF NOT EXISTS settlement_method text NOT NULL DEFAULT 'cash'
    CHECK (settlement_method IN ('cash', 'wallet'));

-- ── ٢ · لقطةُ الطريقة لكلّ مصدرٍ في الطلب (الحقيقةُ الماليّةُ المعتمدة) ──
ALTER TABLE order_items
    ADD COLUMN IF NOT EXISTS merchant_settlement_method text
    CHECK (merchant_settlement_method IN ('cash', 'wallet'));

-- ── ٣ · محفظةُ الاحتباس النظاميّة (مستحقّاتٌ نقديّةٌ معلّقة) ─────────────
-- **حسابٌ نظاميٌّ لا يُنفَق ولا يُسحَب ولا يُعرَض كرصيدِ مستخدم** — يجمع
-- الالتزامَ النقديَّ (منصةٌ تدين للمتجر). **وليس خزينةً**، فيبقى عليه قيدُ
-- `balance >= 0` (فالالتزامُ لا يصير سالباً)، ورصيدُه = مجموعُ المستحقّ النقديّ
-- القائم (FI-14.a).
ALTER TABLE wallets
    ADD COLUMN IF NOT EXISTS is_cash_holding boolean NOT NULL DEFAULT false;

-- **واحدةٌ لا اثنتان** — كالخزينة.
CREATE UNIQUE INDEX IF NOT EXISTS wallets_single_cash_holding_idx
    ON wallets (is_cash_holding) WHERE is_cash_holding;

-- **وتُبذَر بمستخدمٍ نظاميٍّ مخصَّصٍ بلا دورٍ ولا كلمةِ مرور** — فلا يدخل ولا
-- يملك صلاحيّة. رقمُه ليس رقماً سوريّاً حقيقيّاً (+963...) فلا يلتبس بأحد.
DO $$
DECLARE hid uuid;
BEGIN
    SELECT user_id INTO hid FROM wallets WHERE is_cash_holding LIMIT 1;
    IF hid IS NULL THEN
        SELECT id INTO hid FROM users WHERE phone = '+000000000001';
        IF hid IS NULL THEN
            INSERT INTO users (phone, full_name, status)
            VALUES ('+000000000001', 'نظامٌ: مستحقّاتُ المتاجر النقديّة المعلّقة', 'active')
            RETURNING id INTO hid;
        END IF;
        INSERT INTO wallets (user_id) VALUES (hid) ON CONFLICT (user_id) DO NOTHING;
        UPDATE wallets SET is_cash_holding = true WHERE user_id = hid;
    END IF;
END $$;

-- ── ٤ · نوعا القيدِ الجديدان في الدفتر ─────────────────────────────────
-- **`merchant_cash_accrued`** (+ إلى الاحتباس): التزامٌ نقديٌّ نشأ عند الاستلام،
--   و(− بالنوع نفسِه) عكسٌ قبل الدفع. **تراه الخزينةُ في `toParties` فلا تحسبه
--   ربحاً.**
-- **`merchant_cash_paid`** (− من الاحتباس): تسويةُ الالتزام حين يؤكّد الأدمنُ
--   الدفعَ نقداً. **لا تدخل `toParties` للطلب** (تسويةُ التزامٍ لا نصيبُ طرف)،
--   **لكنّها خروجُ نقدٍ خارجيٍّ في مصالحة FI-12 العامّة.**
ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_kind_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
    CHECK (kind = ANY (ARRAY[
        'topup', 'order_payment', 'refund', 'compensation', 'commission',
        'merchant_earning', 'driver_earning', 'payout', 'adjustment',
        'platform_profit',
        'platform_expense',
        'operating_expense',
        'reward',
        'penalty',
        'merchant_cash_accrued',  -- التزامٌ نقديٌّ للمتجر (+احتباس)، وعكسُه (−)
        'merchant_cash_paid'      -- تسويةُ الالتزام نقداً (−احتباس)
    ]));

-- ── ٥ · الهُويّةُ الموحّدةُ للتسوية لكلّ (طلب، متجر) ────────────────────
CREATE TABLE IF NOT EXISTS merchant_settlements (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id           uuid NOT NULL REFERENCES orders (id),
    -- **المتجرُ هو الطرفُ صاحبُ المستحقّ** — لا مالكُه (قد يتبدّل).
    merchant_id        uuid NOT NULL REFERENCES merchants (id),
    -- **ومالكُ لحظةِ النشأة لقطةٌ للتدقيق والإشعار فقط** — لا بديلٌ عن المتجر.
    owner_user_id      uuid NOT NULL REFERENCES users (id),
    method             text NOT NULL CHECK (method IN ('cash', 'wallet')),
    -- **المبلغُ الأصليُّ للمستحقّ (صافي المتجر)** — لا يُمَسّ.
    amount             bigint NOT NULL CHECK (amount > 0),
    -- **والمعكوسُ يُجمَع من `merchant_settlement_reversals`، وهذا صورتُه.**
    reversed_amount    bigint NOT NULL DEFAULT 0
                       CHECK (reversed_amount >= 0 AND reversed_amount <= amount),
    state              text NOT NULL,
    -- **قيدُ الدفتر الموصولُ بكلّ حالة** (وصلٌ لا حدس):
    earning_tx_id      bigint REFERENCES wallet_transactions (id),  -- محفظة: قيدُ merchant_earning
    accrued_tx_id      bigint REFERENCES wallet_transactions (id),  -- نقد: قيدُ +merchant_cash_accrued
    paid_tx_id         bigint REFERENCES wallet_transactions (id),  -- نقد: قيدُ −merchant_cash_paid
    paid_by            uuid REFERENCES users (id),                  -- نقد: الأدمنُ المؤكِّد
    paid_owner_user_id uuid REFERENCES users (id),                  -- نقد: المالكُ لحظةَ الدفع (من دُفع/أُشعر)
    paid_at            timestamptz,
    note               text NOT NULL DEFAULT '',
    created_at         timestamptz NOT NULL DEFAULT now(),

    -- **الهُويّة**: مستحقٌّ واحدٌ لكلّ (طلب، متجر) — مرّةً واحدةً للطريقتين.
    UNIQUE (order_id, merchant_id),

    -- **تماسكُ الطريقة والحالة — تحرسه القاعدة.**
    CONSTRAINT merchant_settlements_method_state_ck CHECK (
        (method = 'wallet' AND state IN ('wallet_credited', 'wallet_reversed'))
     OR (method = 'cash'   AND state IN ('cash_due', 'cash_paid', 'cash_reversed')))
);

CREATE INDEX IF NOT EXISTS merchant_settlements_merchant_state_idx
    ON merchant_settlements (merchant_id, state);

-- ── ٦ · تاريخُ العكسِ — سطرٌ لكلّ حدثِ استردادٍ (لا يُمحى) ──────────────
-- **كلُّ عكسٍ سطرٌ مضافٌ + قيدٌ سالبٌ في الاحتباس** — التاريخُ لا يُعدَّل.
-- `event_ref` يميّز الحدثَ المُطلِق (استردادٌ كاملٌ أو ردُّ بضاعة) فيمنع التكرار.
CREATE TABLE IF NOT EXISTS merchant_settlement_reversals (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    settlement_id uuid NOT NULL REFERENCES merchant_settlements (id),
    tx_id         bigint NOT NULL REFERENCES wallet_transactions (id),  -- قيدُ −merchant_cash_accrued
    amount        bigint NOT NULL CHECK (amount > 0),
    cause         text NOT NULL,   -- 'refund_full' | 'returned_goods' | 'debt_offset'
    event_ref     text NOT NULL,   -- يميّز الحدثَ المُطلِق (منعُ التكرار)
    created_at    timestamptz NOT NULL DEFAULT now(),
    UNIQUE (settlement_id, event_ref)
);

-- ══════════════════════════════════════════════════════════════════════
-- **٧ · التصنيفُ الرجعيُّ — بلا اختلاقِ ماضٍ ماليّ**
-- ══════════════════════════════════════════════════════════════════════
--
-- **لا تُصبَغ كلُّ الطلبات القديمة نقداً** — فتلك التسوياتُ وقعت محفظةً
-- (`merchant_earning`). التصنيفُ من حقيقةِ الدفتر، **لا من مالكِ المتجر اليوم**
-- (فتبدّلُ الملكيّة لا يُحوّل تسويةً محفظيّةً تاريخيّةً إلى نقد):
--
--   أ) طلبٌ له أيُّ `merchant_earning` ⇒ كلُّ مصادرِه سُوّيت محفظةً (كانت
--      التسويةُ قبلَ الميزةِ محفظيّةً وذرّيّةً لكلّ المصادر) ⇒ 'wallet'.
--   ب/ج) ما عداه (قائمٌ قبل التسوية · أو منتهٍ لم يُستحَقّ) ⇒ 'cash'.
--
-- **ولا مستحقّاتٌ نقديّةٌ رجعيّة، ولا قيودُ محفظةٍ رجعيّة، ولا مسٌّ للدفتر.**
-- **ولا يُنشَأ أيُّ صفٍّ في `merchant_settlements` للطلبات التاريخيّة** —
-- الحارسُ الرجعيّ في `settleMerchant` يمنع تكرارَ التسوية للطلبات القائمة.
UPDATE order_items oi SET merchant_settlement_method = 'wallet'
WHERE oi.merchant_settlement_method IS NULL
  AND EXISTS (SELECT 1 FROM wallet_transactions wt
              WHERE wt.ref = oi.order_id::text AND wt.kind = 'merchant_earning');

UPDATE order_items SET merchant_settlement_method = 'cash'
WHERE merchant_settlement_method IS NULL;
