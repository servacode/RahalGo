-- ══════════════════════════════════════════════════════════════════════
-- **«أضف فرع» — طلباتُ الفروع** (قرارُ المالك ٢٠٢٦-١١-١٠، «ابدأ كلشي» ٢٠٢٦-١١-١١)
-- ══════════════════════════════════════════════════════════════════════
--
-- صاحبُ المتجر (أو مندوبُه) يطلب فرعاً بالاسم والموقع على الخريطة، **والأدمنُ يوافق ويحدّد
-- العمولة** فيُنشأ متجرٌ جديدٌ مربوطٌ بحساب الصاحب نفسِه (بالطريق نفسِه الذي تنشئ به اللوحةُ
-- متجراً)، أو يرفض بسببٍ اختياريّ.
--
-- **ولا يمسّ الدفترَ ولا المحافظَ** — جدولٌ جانبيٌّ للطلبات وحدَها.
--
--	parent_merchant_id   المتجرُ الأمّ — منه يُؤخذ الصاحبُ والتصنيفُ والتسوية
--	requested_by         من ضغط «أضف فرع» (صاحبُ المتجر أو مندوبُه)
--	requested_by_role    merchant أو sales — لمن يصل الإشعارُ وبأيّ تطبيق
--	branch_merchant_id   المتجرُ المُنشأ عند الموافقة
--	commission_percent   العمولةُ التي حدّدها الأدمن (فارغٌ = اتبع العامّ)
CREATE TABLE IF NOT EXISTS branch_requests (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    requested_by       uuid NOT NULL REFERENCES users (id),
    requested_by_role  text NOT NULL CHECK (requested_by_role IN ('merchant', 'sales')),
    name               text NOT NULL CHECK (btrim(name) <> ''),
    phone              text,
    address_text       text,
    lat                double precision NOT NULL CHECK (lat BETWEEN -90 AND 90),
    lng                double precision NOT NULL CHECK (lng BETWEEN -180 AND 180),
    status             text NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending', 'approved', 'rejected')),
    commission_percent numeric(5,2) CHECK (commission_percent BETWEEN 0 AND 100),
    reject_reason      text,
    decided_by         uuid REFERENCES users (id),
    decided_at         timestamptz,
    branch_merchant_id uuid REFERENCES merchants (id) ON DELETE SET NULL,
    created_at         timestamptz NOT NULL DEFAULT now(),
    -- **الحالةُ والقرارُ متّسقان** — موافقٌ عليه له متجرٌ، ومقرَّرٌ فيه له قارئ ووقت.
    CONSTRAINT branch_requests_decision_ck CHECK (
        (status = 'pending' AND decided_at IS NULL AND branch_merchant_id IS NULL)
        OR (status = 'approved' AND decided_at IS NOT NULL)
        OR (status = 'rejected' AND decided_at IS NOT NULL AND branch_merchant_id IS NULL))
);

-- **طابورُ الأدمن** يقرأ المنتظِرَ وحدَه بالأقدم أوّلاً.
CREATE INDEX IF NOT EXISTS branch_requests_pending_idx
    ON branch_requests (created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS branch_requests_parent_idx
    ON branch_requests (parent_merchant_id, created_at DESC);

-- **ولا طلبان مفتوحان بالاسم نفسِه للمتجر نفسِه** — نقرتان لا تصنعان فرعين.
CREATE UNIQUE INDEX IF NOT EXISTS branch_requests_pending_name_uq
    ON branch_requests (parent_merchant_id, lower(btrim(name))) WHERE status = 'pending';
