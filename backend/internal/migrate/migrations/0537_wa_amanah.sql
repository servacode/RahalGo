-- ══════════════════════════════════════════════════════════════════════
-- **«أمانة من شخص» بطلب الواتساب** — قرار المالك ٢٠٢٦-١١-١١
-- ══════════════════════════════════════════════════════════════════════
--
-- «بطلب الواتس اب احيانا يكون امانة من مكان لمكان فهون لازم نجيب موقع الامانة وموقع الي بدنا
-- نسلمو ياها» ثمّ «لازم في خيار امانة كمان بحيث الرسالة توصل لصاحب الامانة والي بدو يستلمها
-- وهيك الاثنين يوصلنا موقعهم».
--
-- **ولا نوعَ طلبٍ جديد** — طلبُ الواتساب نفسُه (`origin='whatsapp'`، «لدي توصيلة» والمستلِمُ
-- يدفع الأجرة)، **وصاحبُ الأمانة مكانَ المتجر**: اسمُه ورقمُه في `pickup_name/pickup_phone`
-- (يقرؤهما تطبيقُ السائق حيث اعتاد)، **ونقطتُه في `pickup_override` حين تصل.**
--
-- **والفرقان**: لا ثمنَ بضاعة (`collect_amount` فارغ)، **ولا ينزل للسائقين قبل أن تُعرف نقطةُ
-- الاستلام** — يولد «مقبولاً» وينزل الطابورَ لحظةَ وصول موقع صاحب الأمانة.
ALTER TABLE orders ADD COLUMN IF NOT EXISTS wa_amanah boolean NOT NULL DEFAULT false;

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_wa_amanah_ck;
ALTER TABLE orders ADD CONSTRAINT orders_wa_amanah_ck
    CHECK (NOT wa_amanah OR (origin = 'whatsapp' AND collect_amount IS NULL));

-- **وطلبُ موقع صاحب الأمانة طلبُ المكتب نفسُه** (`mode='whatsapp'`) **بهدفٍ آخر**: يملأ نقطةَ
-- الاستلام لا التسليم (`target='pickup'`). **ولا نوعَ جديداً** — فقيودُ `0507` (النوعُ وصاحبُ الطلب
-- وفهرسُ المتابعة) تبقى كما هي، وكلُّ قارئٍ قائمٍ يبقى على `dropoff`.
ALTER TABLE recipient_location_requests
    ADD COLUMN IF NOT EXISTS target text NOT NULL DEFAULT 'dropoff';
ALTER TABLE recipient_location_requests DROP CONSTRAINT IF EXISTS recipient_location_requests_target_ck;
ALTER TABLE recipient_location_requests ADD CONSTRAINT recipient_location_requests_target_ck
    CHECK (target IN ('dropoff', 'pickup') AND (target = 'dropoff' OR mode = 'whatsapp'));

-- **أمانةٌ تنتظر موقعَ صاحبها** — تقرؤها جولةُ الإصلاح.
CREATE INDEX IF NOT EXISTS idx_orders_wa_amanah_pending ON orders (created_at)
    WHERE wa_amanah AND pickup_override IS NULL AND closed_at IS NULL;
