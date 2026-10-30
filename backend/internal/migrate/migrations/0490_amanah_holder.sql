-- ══════════════════════════════════════════════════════════════════════
-- **الأمانة: صاحبُها ورقمُه، وموقعُه يطلبه الزبونُ منه** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- **الزبونُ لا يعرف أين الأمانة** — فيكتب اسمَ صاحبها ورقمَه، ويضغط «اطلب الموقع منه»
-- فيصله رابطُ `/loc/<رمز>` كما في «إحضار» المتجر (هجرة ٠٤٨٤). **ونقطتُه تصير نقطةَ
-- الاستلام** (`orders.pickup_override`) فيراها السائقُ على خريطته.
--
-- # صاحبُ الأمانة على الطلب
--
-- يراه الزبونُ (طلبُه) والسائقُ المُسنَد والإدارة — **ولا يراه المتجرُ ولا المندوب.**
ALTER TABLE orders
    ADD COLUMN IF NOT EXISTS amanah_holder_name  text,
    ADD COLUMN IF NOT EXISTS amanah_holder_phone text;

-- # طلبُ الموقع من الزبون لا من المتجر وحدَه
--
-- **الصفُّ نفسُه** — صاحبُه متجرٌ (`merchant_id`) أو زبونٌ (`customer_id`)، والنوعُ
-- `amanah` يغيّر جملةَ الصفحة. **ويُربط بالطلب عند إنشائه** (`order_id`): إن وصل
-- الموقعُ قبلها نُسخ إليه عند الإنشاء، **وإن وصل بعدها كُتب فيه حيّاً.**
ALTER TABLE recipient_location_requests ALTER COLUMN merchant_id DROP NOT NULL;
ALTER TABLE recipient_location_requests
    ADD COLUMN IF NOT EXISTS customer_id uuid REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS order_id    uuid REFERENCES orders(id) ON DELETE SET NULL;

ALTER TABLE recipient_location_requests DROP CONSTRAINT IF EXISTS recipient_location_requests_mode_check;
ALTER TABLE recipient_location_requests
    ADD CONSTRAINT recipient_location_requests_mode_check
    CHECK (mode IN ('send', 'fetch', 'amanah'));

ALTER TABLE recipient_location_requests DROP CONSTRAINT IF EXISTS recipient_location_requests_owner_check;
ALTER TABLE recipient_location_requests
    ADD CONSTRAINT recipient_location_requests_owner_check
    CHECK (merchant_id IS NOT NULL OR customer_id IS NOT NULL);

CREATE INDEX IF NOT EXISTS recipient_location_requests_customer_idx
    ON recipient_location_requests (customer_id, created_at DESC) WHERE customer_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS recipient_location_requests_order_idx
    ON recipient_location_requests (order_id) WHERE order_id IS NOT NULL;
