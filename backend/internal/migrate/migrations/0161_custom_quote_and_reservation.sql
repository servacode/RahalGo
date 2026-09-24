-- ══════════════════════════════════════════════════════════════════════
-- 0161 · عرضُ السعر المخصَّص وحجزُه المملوك للطلب — Batch 2a
-- ══════════════════════════════════════════════════════════════════════
--
-- **الطلبُ المخصَّص كان يُتّفق عليه صامتاً**: السائقُ يكتب البضاعةَ والأجرة،
-- فتُدهَس القيمُ القديمةُ بلا نسخةٍ ولا تدقيقٍ ولا تأكيدٍ من الزبون. **وهذه
-- الهجرةُ تجعل العرضَ عقداً**: نسخةٌ تتزايد مع كلّ تغييرٍ حقيقيّ، وتأكيدٌ من
-- الزبون مربوطٌ بنسخةٍ بعينها، وحجزٌ للمحفظة مملوكٌ للطلب نفسِه لا مجمَّعاً.
--
-- **لماذا حجزٌ مملوكٌ للطلب** (`custom_reserved_amount`):
--   `wallets.reserved` مجموعٌ عارٍ، والحارسُ الماليُّ (fininv) يطابقه اليومَ
--   بـ SUM(طلبات السحب المعلَّقة). **فحجزُ طلبٍ مخصَّصٍ فيه وحدَه يكسر ذلك
--   المطابقة.** فيصير لكلّ طلبٍ عمودُه الخاصّ — يُطابَق المجموعُ = السحوباتُ +
--   حجوزُ الطلبات، وكلُّ حجزٍ منسوبٌ لطلبه فيُطلَق ويُسوّى مستقلّاً.
--
-- **لماذا لقطةٌ لسياسة الأجرة** (`custom_fee_source`/`_snapshot`/`_driver_may_change`):
--   السياسةُ العامّةُ قد تتبدّل بعد إنشاء الطلب. **فتُلتقط لحظةَ الإنشاء على
--   الطلب**، فلا يُعاد كتابةُ طلبٍ قائمٍ حين يغيّر الأدمن الإعداد العامّ لاحقاً.
--
-- **إضافةٌ آمنةٌ عكوسة**: أعمدةٌ بقيمٍ افتراضيّةٍ لا تمسّ صفّاً قائماً؛ الصفوفُ
-- التاريخيّةُ كلُّها تصير (نسخة 0، لا تأكيد، لا حجز، مصدرٌ = سائق).
--   (التراجع: ALTER TABLE orders DROP COLUMN IF EXISTS
--     quote_version, quote_confirmed_at, quote_confirmed_total,
--     quote_confirmed_version, custom_reserved_amount,
--     custom_fee_source, custom_fee_snapshot, custom_driver_may_change_fee;
--    ثمّ DROP CONSTRAINT IF EXISTS للقيود الستّة أدناه.)

-- ── الأعمدة ───────────────────────────────────────────────────────────
ALTER TABLE orders
    -- نسخةُ العرض: تتزايد مع كلّ تغييرٍ حقيقيّ في البضاعة أو الأجرة.
    ADD COLUMN IF NOT EXISTS quote_version bigint NOT NULL DEFAULT 0,
    -- تأكيدُ الزبون: متى أكّد، أيَّ مبلغٍ، وأيَّ نسخةٍ أكّد.
    ADD COLUMN IF NOT EXISTS quote_confirmed_at timestamptz,
    ADD COLUMN IF NOT EXISTS quote_confirmed_total bigint,
    ADD COLUMN IF NOT EXISTS quote_confirmed_version bigint,
    -- الحجزُ المملوكُ لهذا الطلب من رصيد محفظة الزبون.
    ADD COLUMN IF NOT EXISTS custom_reserved_amount bigint NOT NULL DEFAULT 0,
    -- لقطةُ سياسة الأجرة لحظةَ الإنشاء (لا تتبع الإعدادَ العامّ بعدها).
    ADD COLUMN IF NOT EXISTS custom_fee_source text NOT NULL DEFAULT 'driver_defined',
    ADD COLUMN IF NOT EXISTS custom_fee_snapshot bigint,
    ADD COLUMN IF NOT EXISTS custom_driver_may_change_fee boolean NOT NULL DEFAULT true;

-- ── القيود الثابتة (ما يصحّ دائماً؛ الحالُ المعتمدةُ على الحالة في fininv) ──

-- ١· الحجزُ لا يكون سالباً.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_reserved_nonneg;
ALTER TABLE orders
    ADD CONSTRAINT orders_custom_reserved_nonneg
    CHECK (custom_reserved_amount >= 0);

-- ٢· طلبٌ غيرُ مخصَّصٍ لا يحمل حجزاً مخصَّصاً أبداً.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_reserved_only_custom;
ALTER TABLE orders
    ADD CONSTRAINT orders_custom_reserved_only_custom
    CHECK (kind = 'custom' OR custom_reserved_amount = 0);

-- ٣· الحجزُ لا يوجد إلّا لطلبِ محفظةٍ أكّده الزبون.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_reserved_wallet_confirmed;
ALTER TABLE orders
    ADD CONSTRAINT orders_custom_reserved_wallet_confirmed
    CHECK (
        custom_reserved_amount = 0
        OR (payment_method = 'wallet' AND quote_confirmed_at IS NOT NULL)
    );

-- ٤· التأكيدُ لا يكتمل إلّا بمبلغٍ ونسخة.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_quote_confirm_complete;
ALTER TABLE orders
    ADD CONSTRAINT orders_quote_confirm_complete
    CHECK (
        quote_confirmed_at IS NULL
        OR (quote_confirmed_total IS NOT NULL AND quote_confirmed_version IS NOT NULL)
    );

-- ٥· مصدرُ الأجرةِ محصورٌ في قيمتين.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_fee_source_check;
ALTER TABLE orders
    ADD CONSTRAINT orders_custom_fee_source_check
    CHECK (custom_fee_source IN ('driver_defined', 'admin_defined'));

-- ٦· لقطةُ الأجرة: موجودةٌ (وغيرُ سالبة) للأدمن، غائبةٌ للسائق.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_custom_fee_snapshot_shape;
ALTER TABLE orders
    ADD CONSTRAINT orders_custom_fee_snapshot_shape
    CHECK (
        (custom_fee_source = 'admin_defined') = (custom_fee_snapshot IS NOT NULL)
        AND (custom_fee_snapshot IS NULL OR custom_fee_snapshot >= 0)
    );
