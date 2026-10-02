-- ══════════════════════════════════════════════════════════════════════
-- **تعويضُ السائق بعد موافقة العمليات — لا لحظةَ الضغطة** (قرارُ المالك ٢٠٢٦-١٠-٠٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- **قِيس قبل القرار** على التجهيز: الخادمُ يقبل سبباً لا يخصّ المرحلة **ويدفع
-- ٥٬٠٠٠ فوراً في كلّ مرّة، حتّى لـ«تأخّرتُ أنا».** فسائقٌ يكسب بضغطة.
--
-- **فالمحرّكُ لم يعد يقيّد التعويض** — يكتب هنا طلباً معلَّقاً بالذنب والمبلغ
-- المقترَح، **والعملياتُ توافق من الباب اليدويّ القائم**
-- (`/admin/orders/{id}/compensate-driver`) الذي يحرس التكرارَ أصلاً.
--
-- **صفٌّ لكلّ (طلب · سائق)**: طلبٌ حُوّل إلى سائقٍ ثانٍ فتعذّر ثانيةً يُعوَّض
-- عنه الثاني أيضاً، **والسائقُ نفسُه لا يُطلَب له مرّتين.**
CREATE TABLE IF NOT EXISTS driver_compensation_requests (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id         uuid NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    driver_id        uuid NOT NULL REFERENCES users(id),
    -- **الذنبُ من السبب** — ولا يُطلَب تعويضٌ لذنب السائق أصلاً.
    fault            text NOT NULL CHECK (fault IN ('customer', 'merchant')),
    fail_reason      text NOT NULL DEFAULT '',
    -- **المقترَحُ بالمعادلة القديمة** (`drivers.failed_compensation_percent`) —
    -- يُعرض للعمليات ولا يُقيَّد إلّا بموافقتها.
    suggested_amount bigint NOT NULL DEFAULT 0 CHECK (suggested_amount >= 0),
    status           text NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'approved', 'rejected')),
    -- **ما قُيّد فعلاً** — يُكتب عند الموافقة وحدَها.
    amount           bigint CHECK (amount IS NULL OR amount > 0),
    decided_by       uuid REFERENCES users(id),
    decided_at       timestamptz,
    decision_note    text NOT NULL DEFAULT '',
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE (order_id, driver_id)
);

-- **قائمةُ ما ينتظر قراراً** — بالأقدم أوّلاً.
CREATE INDEX IF NOT EXISTS driver_compensation_requests_pending
    ON driver_compensation_requests (created_at)
    WHERE status = 'pending';
