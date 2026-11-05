-- ══════════════════════════════════════════════════════════════════════
-- **طلباتُ الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥
-- ══════════════════════════════════════════════════════════════════════
--
-- «عندي طلبات خارجية على الواتس اب، بدي خيار باللوحة… وينزل طلب نظامي ع التطبيق،
-- كأنه طلب جاي من تطبيق». **فيكتب الموظّفُ الطلبَ من اللوحة** وينزل للسائقين مباشرةً
-- بلا موافقة مكتب، ويُحسب طلباً نظاميّاً في كلّ مكان (القوائم والتقارير ومراقبة السائقين).
--
-- **ولا نوعَ جديد**: الطلبُ «لدي توصيلة» (`kind='merchant_delivery'`) بوضع «توصيل» والمستلِمُ
-- يدفع الأجرة نقداً — **وعلامتُه `origin='whatsapp'`**. والبضاعةُ خارجَ الدفتر كما في «لدي
-- توصيلة» (`collect_amount`): السائقُ يدفعها للمتجر ويقبضها من المستلِم، **والمسجَّلُ الأجرةُ وحدَها.**
--
-- # والمتجرُ منسوخٌ لا مربوط
--
-- **لا `merchant_id` على طلب الواتساب أبداً** — ربطُه يُشعل آثارَ المتجر كلَّها (توصيلاتي،
-- إلغاؤه، إشعاراتُه، تحميلُه الفشل، عرضُ التوصيل المجّانيّ). فاسمُ المتجر ورقمُه وعنوانُه
-- يُنسخون على الطلب (`pickup_*`)، **ونقطتُه في `pickup_override`** (سابقةُ الأمانة)، و
-- `pickup_merchant_id` أثرٌ وحدَه للمتجر المختار من متاجرنا — **لا يُسوّى له شيء.**

-- ── ١ · الأعمدة ─────────────────────────────────────────────────────────
--
-- **ومن كتبه من الموظّفين في `orders.created_by`** — عمودٌ قائمٌ منذ `0011` («موظّف = طلبٌ
-- هاتفيّ») ولم يكتبه شيءٌ قبل اليوم، فلا يُضاف ثانيةً.
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS origin             text NOT NULL DEFAULT 'app',
    -- **مكانُ الاستلام كما كُتب** — اسمُه ورقمُه وعنوانُه.
    ADD COLUMN IF NOT EXISTS pickup_name        text,
    ADD COLUMN IF NOT EXISTS pickup_phone       text,
    ADD COLUMN IF NOT EXISTS pickup_address     text,
    ADD COLUMN IF NOT EXISTS pickup_merchant_id uuid REFERENCES merchants (id) ON DELETE SET NULL;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_origin_ck;
ALTER TABLE orders ADD CONSTRAINT orders_origin_ck CHECK (origin IN ('app', 'whatsapp'));

-- ── ٢ · شكلُ طلب الواتساب ───────────────────────────────────────────────
--
-- **بلا متجرٍ مربوطٍ ولا زبون**، والمستلِمُ يدفع نقداً، ولا بضاعةَ في المجموع ولا محفظة،
-- **ونقطةُ الاستلام واسمُها حاضران** — فلا يُقاد سائقٌ إلى لا مكان.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_whatsapp_shape;
ALTER TABLE orders ADD CONSTRAINT orders_whatsapp_shape CHECK (
    origin <> 'whatsapp' OR (
        kind = 'merchant_delivery'
        AND merchant_id IS NULL AND customer_id IS NULL
        AND fee_payer = 'recipient' AND delivery_mode = 'send'
        AND subtotal = 0 AND wallet_paid = 0
        AND pickup_override IS NOT NULL
        AND length(btrim(COALESCE(pickup_name, ''))) > 0
    )
);

-- ── ٣ · «لدي توصيلة» بلا متجرٍ مربوطٍ لطلب الواتساب وحدَه ───────────────
--
-- (`0170` كانت تشترط المتجرَ لكلّ توصيلة.) **والباقي كما هو حرفاً.**
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_merchant_delivery_shape;
ALTER TABLE orders ADD CONSTRAINT orders_merchant_delivery_shape CHECK (
    kind <> 'merchant_delivery' OR (
        (merchant_id IS NOT NULL OR origin = 'whatsapp')
        AND recipient_name IS NOT NULL AND length(btrim(recipient_name)) > 0
        AND recipient_phone IS NOT NULL AND length(btrim(recipient_phone)) > 0
        AND address_text IS NOT NULL AND length(btrim(address_text)) > 0
        AND fee_payer IN ('merchant', 'merchant_cash', 'recipient')
    )
);

-- ── ٤ · قائمةُ طلبات الواتساب ───────────────────────────────────────────
CREATE INDEX IF NOT EXISTS idx_orders_whatsapp ON orders (created_at DESC) WHERE origin = 'whatsapp';

-- ── ٥ · القدرة: لكلّ دورٍ يتدخّل في الطلبات ─────────────────────────────
--
-- (قرارُ المالك ٢٠٢٦-١١-٠٥: «الادمن والموظفين».) **ويمنحها الأدمنُ لأيّ دورٍ آخر من
-- «الأدوار والصلاحيات»** بلا شيفرة.
INSERT INTO role_capabilities (role_code, capability_code)
SELECT DISTINCT role_code, 'orders.whatsapp'
  FROM role_capabilities
 WHERE capability_code = 'orders.intervene'
ON CONFLICT DO NOTHING;
