-- ══════════════════════════════════════════════════════════════════════
-- «ديون الطلبات» — قرار المالك ٢٠٢٦-١١-٢٠
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبون صار يقدر يدفع **كل الطلب من المحفظة** (مع «التوصيل بس» و«كاش»).
-- بالنموذج ٢ الكابتن بيدفع للمتجر من جيبه وقت الاستلام، والزبون ما بيدفعله شي لأنه دفع من محفظته.
-- **فبيتسجّل دين عالمنصة للكابتن** بقد اللي دفعه وما قبضه — المبلغ بيضل بخزينة المنصة، وما بينزل
-- بمحفظة الكابتن. والمكتب بيحاسبه باليد وبيكبس «تم الدفع»، فبينكتب سطر خارج بصندوق المكتب.
--
--   amount        اللي دفعه الكابتن للمتجر ناقص اللي قبضه من الزبون (موجب دايماً)
--   state         open (إله عالمكتب) · paid (انعطى) · void (الطلب انلغى قبل ما ينعطى)
--
-- **ولا قيد بدفتر المحافظ** — سجل بس، ومقابله بالخزينة إنها ما بتحسب حق البضاعة ربح.

CREATE TABLE IF NOT EXISTS order_driver_debts (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id    uuid NOT NULL REFERENCES orders(id),
    driver_id   uuid NOT NULL REFERENCES users(id),
    amount      bigint NOT NULL CHECK (amount > 0),
    state       text NOT NULL DEFAULT 'open' CHECK (state IN ('open', 'paid', 'void')),
    created_at  timestamptz NOT NULL DEFAULT now(),
    paid_at     timestamptz,
    paid_by     uuid REFERENCES users(id),
    voided_at   timestamptz,
    note        text
);

CREATE UNIQUE INDEX IF NOT EXISTS order_driver_debts_order_uq ON order_driver_debts (order_id, driver_id);
CREATE INDEX IF NOT EXISTS order_driver_debts_open_idx ON order_driver_debts (driver_id) WHERE state = 'open';
