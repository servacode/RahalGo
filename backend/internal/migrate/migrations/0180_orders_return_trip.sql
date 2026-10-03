-- ══════════════════════════════════════════════════════════════════════
-- **إرجاعُ البضاعة مشوارٌ يُرى لا إغلاقٌ صامت** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
-- ══════════════════════════════════════════════════════════════════════
--
-- «الزبونُ رفض أو ألغى بعد الاستلام»: الطلبُ يُنهى فشلاً كما كان، **والسائقُ
-- يعود بالبضاعة في مشوارٍ مرسوم** إلى المكتب أو إلى المتجر الذي يقبل
-- الاسترداد، **ويضغط «سلّمت البضاعة» فيُكتب وقتُه وموضعُه وتراه الإدارة.**
--
--	return_to          إلى أين يُرجعها: `office` أو `store` — وفارغٌ: لا مشوارَ إرجاع
--	goods_handed_at    متى سلّمها السائق — وفارغٌ مع `return_to`: المشوارُ قائم
--	goods_handed_point موضعُه لحظةَ الضغط — وفارغٌ إن لم يُعرف
ALTER TABLE orders ADD COLUMN IF NOT EXISTS return_to text
    CHECK (return_to IN ('office', 'store'));
ALTER TABLE orders ADD COLUMN IF NOT EXISTS goods_handed_at timestamptz;
ALTER TABLE orders ADD COLUMN IF NOT EXISTS goods_handed_point geography(Point, 4326);

-- **ومشاويرُ الإرجاع القائمةُ تُقرأ مع كلّ تحديثٍ لقائمة السائق** — فهرسٌ جزئيٌّ صغير.
CREATE INDEX IF NOT EXISTS orders_return_pending_idx ON orders (driver_id)
    WHERE return_to IS NOT NULL AND goods_handed_at IS NULL;

-- **ولا دعمَ تلقائيّاً للمتجر** («لازم المصاري ترجع ع حالها والإدارة تقرر تعوض المتجر
-- او لا»): المفتاحُ خرج من الفهرس، **وقيمةٌ باقيةٌ في القاعدة بلا قارئٍ تُقرأ قاعدةً حيّة.**
DELETE FROM app_settings WHERE key = 'merchants.return_support_percent';
