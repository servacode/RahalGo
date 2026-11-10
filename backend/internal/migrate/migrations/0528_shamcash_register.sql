-- ══════════════════════════════════════════════════════════════════════
-- **سجلّ شام كاش اليدويّ** — قرار المالك ٢٠٢٦-١١-١٠
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون أحياناً بيحوّل على حساب شام كاش تبع المالك بدل ما يدفع للسائق نقداً. **المالك بيعطي السائق
-- المبلغ نقداً من جيبه**، والسائق بيسلّم المكتب المبلغ كامل متل العادة.
--
-- **فالدفتر والمحافظ وصندوق السائق ودرج المكتب والخزينة ما بيتغيّر فيهم شي.** هاد سجلّ جانبيّ بس:
--
--   driver_transfer  السائق قال عند التسليم «انقبض شام كاش» (مرّة وحدة لكل طلب)
--   deposit          المالك حطّ بحسابه من عنده
--   withdrawal       المالك سحب من حسابه
--
-- الرصيد لكل عملة = رصيد البداية + تحويلات السائقين (غير «ما وصل») + الإيداعات − السحوبات،
-- والملغى ما بينحسب، وما قبل تاريخ رصيد البداية داخل فيه.
--
-- **والدولار بالسنتات** (١٢٫٥٠$ ⇒ ١٢٥٠) متل باقي المنصّة (`orders.AmountUnits`).
CREATE TABLE IF NOT EXISTS shamcash_movements (
    id              uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    kind            text        NOT NULL CHECK (kind IN ('driver_transfer', 'deposit', 'withdrawal')),
    order_id        uuid        REFERENCES orders (id) ON DELETE CASCADE,
    driver_id       uuid        REFERENCES users (id),
    currency        text        NOT NULL CHECK (currency IN ('SYP', 'USD')),
    amount          bigint      NOT NULL CHECK (amount > 0),
    occurred_at     timestamptz NOT NULL DEFAULT now(),
    note            text        NOT NULL DEFAULT '',
    -- **«ما وصل»**: السائق قال حوّلوا والمبلغ ما وصل الحساب — بيطلع من الرصيد.
    status          text        NOT NULL DEFAULT 'claimed' CHECK (status IN ('claimed', 'not_arrived')),
    not_arrived_at  timestamptz,
    not_arrived_by  uuid        REFERENCES users (id),
    -- **«سلّمت السائق بدالها»**: المالك عطى السائق المبلغ نقداً.
    compensated_at  timestamptz,
    compensated_by  uuid        REFERENCES users (id),
    -- **إلغاء إيداع أو سحب انكتب غلط** — بسبب، وما بينحذف الصفّ.
    voided_at       timestamptz,
    voided_by       uuid        REFERENCES users (id),
    void_reason     text        NOT NULL DEFAULT '',
    created_by      uuid        REFERENCES users (id),
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT shamcash_driver_shape CHECK (
        (kind = 'driver_transfer') = (order_id IS NOT NULL AND driver_id IS NOT NULL)),
    CONSTRAINT shamcash_owner_shape CHECK (
        kind = 'driver_transfer'
        OR (status = 'claimed' AND compensated_at IS NULL AND not_arrived_at IS NULL)),
    CONSTRAINT shamcash_void_owner_only CHECK (voided_at IS NULL OR kind <> 'driver_transfer'),
    CONSTRAINT shamcash_not_arrived_shape CHECK ((status = 'not_arrived') = (not_arrived_at IS NOT NULL)),
    CONSTRAINT shamcash_compensated_claimed CHECK (compensated_at IS NULL OR status = 'claimed')
);

-- **تحويل واحد لكل طلب** — إعادة نداء التسليم (الشبكة) ما بتكرّره.
CREATE UNIQUE INDEX IF NOT EXISTS shamcash_one_per_order
    ON shamcash_movements (order_id) WHERE kind = 'driver_transfer';
CREATE INDEX IF NOT EXISTS shamcash_by_time ON shamcash_movements (occurred_at DESC);
CREATE INDEX IF NOT EXISTS shamcash_by_driver ON shamcash_movements (driver_id, occurred_at DESC)
    WHERE driver_id IS NOT NULL;

-- **رصيد البداية لكل عملة** — والمالك بيعدّله؛ «بتاريخ» = من وين بلّش العدّ.
CREATE TABLE IF NOT EXISTS shamcash_opening (
    currency    text        PRIMARY KEY CHECK (currency IN ('SYP', 'USD')),
    amount      bigint      NOT NULL DEFAULT 0 CHECK (amount >= 0),
    as_of       timestamptz NOT NULL DEFAULT now(),
    updated_by  uuid        REFERENCES users (id),
    updated_at  timestamptz NOT NULL DEFAULT now()
);
