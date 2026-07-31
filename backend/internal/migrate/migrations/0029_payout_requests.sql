-- طلبات سحب الرصيد — الحلقة الأخيرة في دورة المندوب.
-- كانت العمولات تدخل المحفظة وتقف: لا طريق لصرفها ولا أثر لمطالبة. الآن يطلب
-- صاحب الرصيد سحبه، وتراه المالية، وتُصرف بقيد `payout` في الدفتر نفسه.
--
-- الرصيد لا يُخصم عند الطلب بل عند **الصرف الفعلي**: لا نجمّد مال أحد بطلب.
CREATE TABLE payout_requests (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid   NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    amount      bigint NOT NULL CHECK (amount > 0),
    -- pending: بانتظار المالية | paid: صُرف | rejected: مرفوض
    status      text   NOT NULL DEFAULT 'pending'
                CHECK (status IN ('pending', 'paid', 'rejected')),
    note        text   NOT NULL DEFAULT '',   -- ملاحظة الطالب
    decision    text   NOT NULL DEFAULT '',   -- سبب الرفض أو مرجع الصرف
    decided_by  uuid REFERENCES users (id),
    decided_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX payout_requests_user_idx ON payout_requests (user_id, created_at DESC);
CREATE INDEX payout_requests_status_idx ON payout_requests (status, created_at);

-- طلب معلّق واحد لكل حساب: يمنع إغراق المالية بطلبات متتالية للرصيد نفسه.
CREATE UNIQUE INDEX payout_requests_one_pending
    ON payout_requests (user_id) WHERE status = 'pending';
