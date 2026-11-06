-- ══════════════════════════════════════════════════════════════════════
-- **طلبٌ واحدٌ بعدّة محطّات** — قرارُ المالك ٢٠٢٦-١١-٠٥ («١ موافق»)
-- ══════════════════════════════════════════════════════════════════════
--
-- طلبٌ واحد · محطّةُ استلامٍ لكلّ متجر · سائقٌ واحد · أجرةٌ واحدة · كودُ تسليمٍ واحد.
--
--   ١ · `orders.stops`             عددُ متاجر الطلب لحظةَ إنشائه — **ولا يُنقَص أبداً.**
--                                  ١ لكلّ طلبٍ قائمٍ ولكلّ طلبٍ من متجرٍ واحد.
--   ٢ · `order_events.merchant_id` حدثُ قسمٍ من الطلب (متجرٌ قبل قسمَه أو رفضه…) —
--                                  **وفارغٌ لكلّ حدثٍ على الطلب كلِّه**، وهو كلُّ حدثٍ قائم.
--   ٣ · `order_parts`              صفٌّ لكلّ متجرٍ **في طلبٍ أُنشئ من متجرين فأكثر وحدَه.**
--                                  **والطلبُ من متجرٍ واحدٍ بلا صفّ** — ويبقى على مساره اليوم.
--
-- **ولا يُملأ شيءٌ لِما مضى**: كلُّ طلبٍ قائمٍ `stops = 1` بلا أقسام، **ويبقى على سلوكه.**

-- ── ١ · عددُ المحطّات ──────────────────────────────────────────────────
ALTER TABLE orders ADD COLUMN IF NOT EXISTS stops smallint NOT NULL DEFAULT 1
    CHECK (stops BETWEEN 1 AND 10);

-- ── ٢ · حدثُ القسم ─────────────────────────────────────────────────────
ALTER TABLE order_events ADD COLUMN IF NOT EXISTS merchant_id uuid REFERENCES merchants (id);
-- **فهرسٌ جزئيٌّ لحدث القسم وحدَه** — **وحذفُ متجرٍ يفحص مفتاحَه الأجنبيّ**: بلا فهرسٍ
-- يمسح جدولَ الأحداث كلَّه، **والجزئيُّ لا يحمل إلّا ما له قيمة** (كلُّ حدثٍ قائمٍ فارغ).
CREATE INDEX IF NOT EXISTS order_events_by_merchant
    ON order_events (merchant_id) WHERE merchant_id IS NOT NULL;

-- ── ٣ · أقسامُ الطلب — صفٌّ لكلّ متجر ─────────────────────────────────
CREATE TABLE IF NOT EXISTS order_parts (
    order_id       uuid NOT NULL REFERENCES orders (id) ON DELETE CASCADE,
    merchant_id    uuid NOT NULL REFERENCES merchants (id),
    -- ١ = المتجرُ الأوّل (`orders.merchant_id`، أوّلُ ما في السلّة) — **وترتيبُ زيارةٍ مقترح.**
    seq            smallint NOT NULL,
    status         text NOT NULL DEFAULT 'pending'
                   CHECK (status IN ('pending', 'accepted', 'rejected', 'removed', 'picked_up')),
    answered_at    timestamptz,
    answered_by    uuid REFERENCES users (id) ON DELETE SET NULL,
    answered_role  text NOT NULL DEFAULT ''
                   CHECK (answered_role IN ('', 'merchant', 'ops', 'admin', 'system')),
    prep_minutes   int CHECK (prep_minutes IS NULL OR prep_minutes BETWEEN 0 AND 600),
    -- **«جاهز» وقتٌ لا حال** — كـ`orders.ready_at` اليوم.
    ready_at       timestamptz,
    -- أُبلغ المتجرُ بقسمه (واتساب أو يدويّاً).
    sent_at        timestamptz,
    sent_channel   text NOT NULL DEFAULT '',
    -- **وصلَ السائقُ الحاليُّ إلى هذه المحطّة** — ويُمحى كلّما عاد الطلبُ إلى التوزيع.
    arrived_at     timestamptz,
    picked_up_at   timestamptz,
    -- **السائقُ الذي بيده بضاعةُ هذه المحطّة.**
    picked_up_by   uuid REFERENCES users (id) ON DELETE SET NULL,
    -- سببُ الرفض أو الإزالة — **إلزاميٌّ حين يُرفض القسم.**
    reason         text NOT NULL DEFAULT '',
    removed_fault  text NOT NULL DEFAULT ''
                   CHECK (removed_fault IN ('', 'merchant', 'platform')),
    -- **لقطةُ ما خرج من الطلب** — الاسمُ والعددُ والخياراتُ وسعرا البيع والشراء —
    -- **فيبقى في سجلّ المتجر ما رفضه وقد حُذفت بنودُه.**
    removed_items  jsonb NOT NULL DEFAULT '[]'::jsonb,
    -- ما كانت تلك البنودُ على الزبون.
    removed_total  bigint NOT NULL DEFAULT 0,
    -- **تفصيلُ عمولة المندوب لكلّ متجر** — للشاشة وتقسيم الاسترداد، **والدفترُ هو الحقيقة.**
    rep_id         uuid REFERENCES users (id) ON DELETE SET NULL,
    rep_commission bigint NOT NULL DEFAULT 0,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (order_id, merchant_id),
    UNIQUE (order_id, seq)
);
-- **أقسامُ متجرٍ بالأحدث** — قائمةُ طلبات المتجر، **وفحصُ المفتاح الأجنبيّ عند حذف متجر.**
CREATE INDEX IF NOT EXISTS order_parts_by_merchant ON order_parts (merchant_id, created_at DESC);
