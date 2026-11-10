-- ══════════════════════════════════════════════════════════════════════
-- **إضافة أصناف بصورة على واتساب** (قرار المالك 2026-11-09، المرحلة 1)
-- ══════════════════════════════════════════════════════════════════════
--
-- صاحب المتجر أو موظفه يبعت صورة + «اسم وسعر» لرقم رحّال غو، فينضاف الصنف فوراً.
--
--   menu_items.created_via   من وين وُلد الصنف: '' (تطبيق/لوحة) أو 'whatsapp' ⇒ شارة «من الواتساب»
--   wa_item_pending          أسئلة البوت المعلّقة (فرع؟ سعر؟ اسم؟ مكرر؟ متأكد؟) بمهلة، بمفتاح الرقم
--   wa_item_store_choice     الفرع اللي اختاره الرقم (يبقى حتى يغيّره بكلمة «فرع»)
--   wa_item_msgs             معرّفات الرسائل المعالجة — واتساب قد يعيد التسليم فلا يتكرر صنف
ALTER TABLE menu_items ADD COLUMN IF NOT EXISTS created_via text NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS wa_item_pending (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    phone      text NOT NULL,
    msg_id     text NOT NULL DEFAULT '',
    kind       text NOT NULL CHECK (kind IN ('store', 'store_wait', 'name', 'price', 'dup', 'sanity', 'section')),
    data       jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS wa_item_pending_phone_idx ON wa_item_pending (phone, created_at);

CREATE TABLE IF NOT EXISTS wa_item_store_choice (
    phone       text PRIMARY KEY,
    merchant_id uuid NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS wa_item_msgs (
    msg_id     text PRIMARY KEY,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- سؤال «لأي فرع؟» واحد فقط لكل رقم — صور متزامنة تنتظر خلفه (store_wait) ولا تسأل ثانيةً.
CREATE UNIQUE INDEX IF NOT EXISTS wa_item_pending_one_store_idx
    ON wa_item_pending (phone) WHERE kind = 'store';
