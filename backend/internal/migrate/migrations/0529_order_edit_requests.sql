-- ══════════════════════════════════════════════════════════════════════
-- **طلب تعديل من الكابتن — والمكتب يوافق أو يرفض** (قرار المالك ٢٠٢٦-١١-١٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «كلشي لازم يمشي على المبالغ الحقيقية مو على أرقام الطلب الأصلي». بالطلب العادي الكابتن
-- **ما بيغيّر المصاري بإيده**: بيبعت طلب تعديل — سطر «ناقص» بالمتجر، أو «الزبون لغاه»، أو
-- «انزاد صنف» باسمه وكمّيته وسعره — **والمكتب بيوافق** فيمشي التعديل من باب تعديل الطلب
-- القائم (`orders/order_edit.go`) بكل حسبته (حق المتجر والعمولة والمحفظة والمستحق النقدي).
--
--   lines             [{item_id, action: missing|cancelled, qty}] — `qty` كم قطعة بتنشال من السطر
--   adds              [{name, qty, price}] — صنف انزاد، سعر الوحدة بالليرة
--   base_fingerprint  بصمة سطور الطلب ومجموعه وقت الطلب — **إذا تغيّر الطلب بعدها الموافقة بتنرد**
--   old_total/new_total  المجموع وقت الطلب والمتوقّع بعده (معاينة المحرّك نفسه)
--   status            pending · approved · rejected · superseded (طلب أحدث) · stale (الطلب تغيّر)
--                     · closed (الطلب انتهى)
--
-- **ومعلّق واحد بس لكل طلب** — والجديد بيحلّ محل القديم.
CREATE TABLE IF NOT EXISTS order_edit_requests (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id          uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id         uuid NOT NULL,
    lines             jsonb NOT NULL DEFAULT '[]'::jsonb,
    adds              jsonb NOT NULL DEFAULT '[]'::jsonb,
    note              text NOT NULL DEFAULT '',
    base_fingerprint  text NOT NULL,
    old_total         bigint NOT NULL,
    new_total         bigint NOT NULL,
    status            text NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending', 'approved', 'rejected', 'superseded', 'stale', 'closed')),
    decided_by        uuid,
    decided_at        timestamptz,
    reject_reason     text NOT NULL DEFAULT '',
    result            jsonb,
    created_at        timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS order_edit_requests_one_pending
    ON order_edit_requests (order_id) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS order_edit_requests_order
    ON order_edit_requests (order_id, created_at DESC);
