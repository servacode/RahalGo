-- ══════════════════════════════════════════════════════════════════════
-- «اطلب تعديل الأجرة» — الكابتن يطلب أجرة توصيل جديدة لطلب الواتساب، والإدارة توافق أو ترفض
-- (قرار المالك ٢٠٢٦-١١-١٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- الكابتن ما بيغيّر المصاري بإيده: بيكتب الأجرة الجديدة والسبب، وبيطلع الطلب على كارت اللوحة.
-- الموافقة بتمشي من باب «تعديل الأجرة» نفسه (`EditDeliveryFee`) بنفس الحسبة والقيود والتدقيق.
-- طلب واحد معلّق لكل طلب (فهرس فريد جزئي) — والطلب الجديد بيحلّ محل القديم (superseded).
-- `old_fee` الأجرة وقت الطلب: إذا تغيّرت قبل الموافقة، الموافقة بتنرد والطلب بيصير «قديم» (stale).

CREATE TABLE IF NOT EXISTS order_fee_requests (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id       uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id      uuid NOT NULL,
    old_fee        bigint NOT NULL,
    fee            bigint NOT NULL CHECK (fee > 0),
    reason         text NOT NULL,
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'approved', 'rejected', 'superseded', 'stale')),
    decided_by     uuid,
    decided_at     timestamptz,
    reject_reason  text NOT NULL DEFAULT '',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS order_fee_requests_one_pending
    ON order_fee_requests (order_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS order_fee_requests_order ON order_fee_requests (order_id, created_at DESC);
