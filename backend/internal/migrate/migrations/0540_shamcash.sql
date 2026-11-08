-- ══════════════════════════════════════════════════════════════════════
-- ٠٥٤٠ · الدفع بشام كاش — جاهز ومطفي (قرار المالك ٢٠٢٦-١١-٠٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون بيشحن محفظته أو بيدفع طلبه بشام كاش عن طريق الواجهة الرسمية.
-- المال بيوصل حساب المنصة بشام كاش — مو درج المكتب — فإله دفتر لحالو
-- (shamcash_account_entries) ونوعا قيد بالمحفظة لحالهن.
-- ما في شي بيشتغل لحتى تنحط المفاتيح وينفتح الإعداد customers.shamcash_enabled.

-- ── ١ · الفواتير ─────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS shamcash_bills (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    -- رقم الفاتورة عند شام كاش (billNo) — جديد بكل محاولة
    number        text NOT NULL UNIQUE,
    user_id       uuid NOT NULL REFERENCES users(id),
    purpose       text NOT NULL CHECK (purpose IN ('topup', 'order')),
    amount        bigint NOT NULL CHECK (amount > 0),
    currency      smallint NOT NULL DEFAULT 2,
    status        text NOT NULL DEFAULT 'pending'
                  CHECK (status IN ('pending', 'paid', 'expired', 'failed', 'refunded', 'partly_refunded')),
    payment_url   text NOT NULL DEFAULT '',
    -- رقم حركة الدفع عند شام كاش — فريد: حركة وحدة ما بتنحسب مرتين
    sham_tran_id  bigint UNIQUE,
    -- سلّة الطلب كما بعتها الزبون — الطلب بينعمل بعد ما يندفع
    order_input   jsonb,
    order_id      uuid REFERENCES orders(id),
    order_error   text NOT NULL DEFAULT '',
    refunded      bigint NOT NULL DEFAULT 0 CHECK (refunded >= 0 AND refunded <= amount),
    history       jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at    timestamptz NOT NULL DEFAULT now(),
    expires_at    timestamptz NOT NULL,
    paid_at       timestamptz,
    checked_at    timestamptz,
    CHECK ((purpose = 'order') = (order_input IS NOT NULL)),
    CHECK (status IN ('pending', 'expired', 'failed') OR (paid_at IS NOT NULL AND sham_tran_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS shamcash_bills_user_idx ON shamcash_bills (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS shamcash_bills_pending_idx ON shamcash_bills (created_at) WHERE status = 'pending';

-- ── ٢ · الإرجاع لشام كاش (إدارة فقط) ──────────────────────────────────
CREATE TABLE IF NOT EXISTS shamcash_refunds (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    bill_id       uuid NOT NULL REFERENCES shamcash_bills(id),
    amount        bigint NOT NULL CHECK (amount > 0),
    status        text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'done', 'failed')),
    sham_tran_id  bigint UNIQUE,
    note          text NOT NULL DEFAULT '',
    error         text NOT NULL DEFAULT '',
    created_by    uuid REFERENCES users(id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    done_at       timestamptz,
    CHECK ((status = 'done') = (sham_tran_id IS NOT NULL))
);
CREATE INDEX IF NOT EXISTS shamcash_refunds_bill_idx ON shamcash_refunds (bill_id);

-- ── ٣ · دفتر حساب المنصة بشام كاش — مو درج المكتب ─────────────────────
CREATE TABLE IF NOT EXISTS shamcash_account_entries (
    id          bigserial PRIMARY KEY,
    direction   text NOT NULL CHECK (direction IN ('in', 'out')),
    amount      bigint NOT NULL CHECK (amount > 0),
    -- bill_paid (داخل، المرجع الفاتورة) · refund_sent (طالع، المرجع الإرجاع)
    source      text NOT NULL CHECK ((source = 'bill_paid' AND direction = 'in')
                                  OR (source = 'refund_sent' AND direction = 'out')),
    ref         uuid NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    UNIQUE (source, ref)
);

-- ── ٤ · نوعا القيد بالمحفظة ───────────────────────────────────────────
SELECT wallet_kinds_add('topup_shamcash');
SELECT wallet_kinds_add('refund_shamcash');

-- والإيداع من فاتورة وحدة مرة وحدة بس — قيد بالقاعدة فوق قفل الصف
CREATE UNIQUE INDEX IF NOT EXISTS wallet_tx_topup_shamcash_once
    ON wallet_transactions (ref) WHERE kind = 'topup_shamcash';
