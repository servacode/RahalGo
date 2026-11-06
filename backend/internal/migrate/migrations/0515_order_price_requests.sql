-- ══════════════════════════════════════════════════════════════════════
-- **الكابتن يطلب تعديل ثمن البضاعة والزبون يوافق** — قرار المالك ٢٠٢٦-١١-٠٦
-- ══════════════════════════════════════════════════════════════════════
--
-- بطلب المشتريات بعد الاستلام، الكابتن بيكتب الثمن الجديد وسببه، والطلب بيصير نسخة سعر
-- جديدة بانتظار الزبون (`applyCustomQuoteTx`) — والتسليم موقوف لحد ما يوافق أو يرفض.
-- **هالجدول سجلّ الطلب**: القديم والجديد والسبب ومين قرّر وإمتى — بيقرأه الرفض ليرجّع
-- السعر القديم، والراصد لينذر المكتب بعد المهلة، والمكتب عند الخلاف.
--
-- **ومعلّق واحد بس لكل طلب** — الفهرس الفريد الجزئي بيمنع التاني حتى لو سابقوا بعض.
CREATE TABLE IF NOT EXISTS order_price_requests (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id      uuid NOT NULL,
    old_goods      bigint NOT NULL,
    new_goods      bigint NOT NULL,
    old_total      bigint NOT NULL,
    new_total      bigint NOT NULL,
    quote_version  bigint NOT NULL,
    reason         text NOT NULL DEFAULT '',
    -- pending · approved · rejected · superseded (المكتب عدّل) · closed (الطلب انتهى)
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected', 'superseded', 'closed')),
    created_at     timestamptz NOT NULL DEFAULT now(),
    decided_at     timestamptz,
    alerted_at     timestamptz
);

CREATE UNIQUE INDEX IF NOT EXISTS order_price_requests_one_pending
    ON order_price_requests (order_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS order_price_requests_order
    ON order_price_requests (order_id, created_at DESC);
