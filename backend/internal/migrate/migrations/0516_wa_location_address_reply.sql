-- ══════════════════════════════════════════════════════════════════════
-- **عنوانُ الزبون يصل بردّه على البوت** — قرار المالك ٢٠٢٦-١١-٠٦
-- ══════════════════════════════════════════════════════════════════════
--
-- طلبُ الواتساب من اللوحة صار برقم الزبون وحدَه: **الموقعُ والعنوانُ المكتوبُ يطلبهما البوتُ في
-- رسالةٍ واحدة**، والموقعُ يملأ نقطةَ الطلب (كما كان)، **وأوّلُ ردٍّ نصّيٍّ من الرقم نفسِه يُكتب
-- عنوانَ الطلب** (`address_text`). فيُحفظ هنا ما وصل ومتى — **مرّةً واحدة**، ومتابعةُ «اكتبلنا
-- عنوانك» لا تُبعث بعده.
ALTER TABLE recipient_location_requests
    ADD COLUMN IF NOT EXISTS address_text        text,
    ADD COLUMN IF NOT EXISTS address_received_at timestamptz;

-- **ومتجرٌ غيرُ تابعٍ باسمه وحدَه بلا نقطة** (قرارُ المالك ٢٠٢٦-١١-٠٦: «بس اسم المحل، بلا موقع ولا
-- خريطة») — فنقطةُ الاستلام اختياريّة، **واسمُ المتجر باقٍ إلزاميّاً**: السائقُ يعرف من أين يستلم.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_whatsapp_shape;
ALTER TABLE orders ADD CONSTRAINT orders_whatsapp_shape CHECK (
    origin <> 'whatsapp' OR (
        kind = 'merchant_delivery'
        AND merchant_id IS NULL AND customer_id IS NULL
        AND fee_payer = 'recipient' AND delivery_mode = 'send'
        AND subtotal = 0 AND wallet_paid = 0
        AND length(btrim(COALESCE(pickup_name, ''))) > 0
    )
);

-- **وطلبُ الواتساب يولد بلا عنوانٍ مكتوب** — الموظّفُ يكتب رقمَ الزبون وحدَه، والعنوانُ يصل من ردّه.
-- فشرطُ «لدي توصيلة» يبقى كما هو لغيره، **ولطلب الواتساب العنوانُ موجودٌ ولو فارغاً.**
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_merchant_delivery_shape;
ALTER TABLE orders ADD CONSTRAINT orders_merchant_delivery_shape CHECK (
    kind <> 'merchant_delivery' OR (
        (merchant_id IS NOT NULL OR origin = 'whatsapp')
        AND recipient_name IS NOT NULL AND length(btrim(recipient_name)) > 0
        AND recipient_phone IS NOT NULL AND length(btrim(recipient_phone)) > 0
        AND address_text IS NOT NULL
        AND (origin = 'whatsapp' OR length(btrim(address_text)) > 0)
        AND fee_payer IN ('merchant', 'merchant_cash', 'recipient')
    )
);
