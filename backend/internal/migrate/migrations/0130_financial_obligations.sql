-- ══════════════════════════════════════════════════════════════════════
-- **الالتزاماتُ الماليّة — واقعةٌ تُقيَّد لا رقمٌ يُزاد**
-- ══════════════════════════════════════════════════════════════════════
--
-- (`XG-31` — دورةُ إصلاحٍ ٣، ٢٠٢٦-٠٩-٠٦.)
--
-- # العلّة
--
-- **كان الالتزامُ عمودَين يُزادان وينقصان**: `merchants.debt` منذ ردّ
-- البضاعة، و`users.commission_debt` منذ دورةِ إصلاحٍ ٢. **ومن رأى ديناً
-- قدرُه ألفٌ ومئتان لم يستطع أن يقول من أين** — لا الطلبَ ولا المبلغَ
-- الأصليَّ ولا الوقتَ ولا السبب.
--
-- **ورقمٌ لا يُفسَّر لا يُراجَع ولا يُنازَع فيه.**
--
-- # ولماذا جدولٌ لا نوعٌ في الدفتر
--
-- **`wallet_transactions` دفترُ محفظةٍ لا دفترُ التزامات**: كلُّ قيدٍ فيه
-- يمرّ بـ`ApplyTx` **فيحرّك `wallets.balance` حتماً**. **والالتزامُ ليس
-- حركةَ مال** — **لم ينتقل قرشٌ حين نشأ**، وإنّما عجز الرصيدُ عن حمل
-- عكسٍ واجب.
--
-- **فقيدُه في الدفتر يُفسد كلَّ رصيدٍ وكلَّ مصالحة.** وفوق ذلك
-- `user_id` مستخدِمٌ، **والتزامُ المتجر على المتجر لا على مالكه** — وقد
-- يتبدّل المالك.
--
-- **فجدولان: نشأةٌ وتسويات.** **ولا يُخترَع عالمٌ محاسبيٌّ موازٍ** —
-- **التسويةُ تبقى قيداً في الدفتر كما هي، ويُربَط بها السطرُ هنا.**
--
-- # وأيُّهما الحقّ
--
-- **هذان الجدولان هما الحقيقة.** **و`merchants.debt` و
-- `users.commission_debt` صورةٌ محفوظةٌ للقراءة السريعة** — يحرسها ثابتٌ
-- دائمٌ يوجب تطابقَهما (`FI-12` · `FI-13`).

CREATE TABLE IF NOT EXISTS financial_obligations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),

    -- **على من**: `merchant` ⇒ `merchants.id` · `rep` ⇒ `users.id`.
    --
    -- **ولا مفتاحَ أجنبيٌّ واحدٌ يصلح للاثنين** — والجدولان مختلفان.
    party_kind text NOT NULL CHECK (party_kind IN ('merchant', 'rep')),
    party_id   uuid NOT NULL,

    -- **المبلغُ الأصليُّ — ولا يُمَسّ أبداً.** **والمسدَّدُ يُجمَع من
    -- `obligation_settlements`، وهذا العمودُ صورتُه.**
    amount     bigint NOT NULL CHECK (amount > 0),
    settled    bigint NOT NULL DEFAULT 0 CHECK (settled >= 0),
    CONSTRAINT obligations_settled_ck CHECK (settled <= amount),

    -- **ولماذا نشأ.**
    cause      text NOT NULL CHECK (cause IN (
        'refund_merchant_earning',  -- استُرِدّ طلبٌ ومستحقُّ المتجر مسحوب
        'refund_rep_commission',    -- استُرِدّ طلبٌ وعمولةُ المندوب مسحوبة
        'returned_goods',           -- بضاعةٌ رُدّت والثمنُ مسحوب
        'legacy_opening')),         -- رصيدٌ مفتوحٌ قبل هذا الجدول

    -- **وعن أيّ طلب** — **ونُلُّه للتركة**: **ولا يُختلَق رقمُ طلب.**
    order_id   uuid REFERENCES orders (id) ON DELETE SET NULL,

    created_by uuid REFERENCES users (id),
    created_at timestamptz NOT NULL DEFAULT now(),
    closed_at  timestamptz,

    -- **والإغلاقُ مشتقٌّ لا مُدَّعى.**
    CONSTRAINT obligations_closed_ck
        CHECK ((closed_at IS NULL) = (settled < amount))
);

-- **ولا نشأتان لطلبٍ واحدٍ بالسبب نفسِه** — **وإعادةُ نداء الاسترداد
-- تُردّ من القاعدة لا من ترتيب الشيفرة.** (البند ٦.)
--
-- **والتركةُ خارجَه** — `order_id` فيها فارغٌ، وقد تتعدّد.
CREATE UNIQUE INDEX IF NOT EXISTS obligations_origin_uq
    ON financial_obligations (party_kind, party_id, order_id, cause)
    WHERE order_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS obligations_open_idx
    ON financial_obligations (party_kind, party_id, created_at)
    WHERE closed_at IS NULL;

-- ══════════════════════════════════════════════════════════════════════
-- **والتسوياتُ وقائعُ لا نقصانُ عمود**
-- ══════════════════════════════════════════════════════════════════════
--
-- **وكلُّ سطرٍ يقول**: أيَّ التزامٍ سدّد · بكم · من أيّ طلبٍ جاء المال ·
-- **وكم بقي بعده.** **والباقي يُكتب لحظتَه** — فمن قرأ السطرَ بعد سنةٍ
-- عرف الحالَ يومَها ولا يعيد الحساب.
CREATE TABLE IF NOT EXISTS obligation_settlements (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    obligation_id  uuid NOT NULL
                   REFERENCES financial_obligations (id) ON DELETE CASCADE,
    amount         bigint NOT NULL CHECK (amount > 0),

    -- **الطلبُ الذي جاء منه المال** — عمولةٌ جديدةٌ أو مستحقٌّ جديد.
    order_id       uuid REFERENCES orders (id) ON DELETE SET NULL,

    -- **والقيدُ في الدفتر الذي حمل الاقتطاع** — **فالوصلُ بين
    -- العالمَين سطرٌ لا حدس.**
    ledger_tx_id   bigint REFERENCES wallet_transactions (id) ON DELETE SET NULL,

    remaining      bigint NOT NULL CHECK (remaining >= 0),
    created_by     uuid REFERENCES users (id),
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS settlements_obligation_idx
    ON obligation_settlements (obligation_id, created_at);

-- ══════════════════════════════════════════════════════════════════════
-- **والتركةُ تُنقَل ولا تُختلَق ولا تُهدَر**
-- ══════════════════════════════════════════════════════════════════════
--
-- **كلُّ رصيدٍ قائمٍ بلا نشأةٍ يصير التزامَ فتحٍ صريحاً** —
-- `legacy_opening` بلا طلبٍ ولا منشئ. **ومن قرأه عرف أنّه سابقٌ للدفتر
-- ولم يُنسَب زوراً إلى طلب.**
--
-- **ولا يُمحى دَينٌ أبداً** — **والعمودان يبقيان كما هما**، فيتطابق
-- المجموعُ من أوّل يوم.
INSERT INTO financial_obligations (party_kind, party_id, amount, cause)
SELECT 'merchant', id, debt, 'legacy_opening' FROM merchants WHERE debt > 0
ON CONFLICT DO NOTHING;

INSERT INTO financial_obligations (party_kind, party_id, amount, cause)
SELECT 'rep', id, commission_debt, 'legacy_opening' FROM users WHERE commission_debt > 0
ON CONFLICT DO NOTHING;
