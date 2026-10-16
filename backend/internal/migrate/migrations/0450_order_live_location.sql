-- ══════════════════════════════════════════════════════════════════════
-- **موقعُ الزبون المباشر — آخرُ نقطةٍ وحدَها** (قرارُ المالك ٢٠٢٦-١٠-١٦)
-- ══════════════════════════════════════════════════════════════════════
--
-- **دبابيسُ الخريطة في الرقّة كثيراً ما تخطئ** — فيقف الكابتنُ بعيداً عن
-- الزبون. **فصار الزبونُ يشارك موقعَه المباشرَ وهو ينتظر طلبَه**، والسائقُ
-- يراه على خريطته وينتقل الطريقُ إليه إن بعُد عن الدبّوس.
--
-- # ولماذا جدولٌ صغيرٌ لا Redis
--
-- **صفٌّ واحدٌ لكلّ طلب** (`order_id` مفتاحٌ أوّل) — **آخرُ نقطةٍ ووقتُها
-- لا أثر.** **ولا تاريخَ يُحفظ**: ما مضى من مواضع الزبون ليس شأنَ أحد.
--
--   - **يُضمّ في استعلام السائق نفسِه** — لا نداءَ ثانٍ لكلّ طلب.
--   - **ويُمحى بقاعدةٍ لا بذاكرةِ مبرمج** — المحفّزُ أدناه يمحوه عند كلّ
--     نهاية، **وأبوابُ الإنهاء كثيرة** (السائقُ والمكتبُ والزبونُ والمهلة).
--   - **والحِملُ تافه** — كتابةٌ كلَّ عشر ثوانٍ لكلّ زبونٍ يشارك.
--
-- **وRedis يُسقطه الحِملُ على التجهيز** (هشاشةٌ مقيسة) — **وموقعٌ يضيع
-- بسقوط الكاش يُعيد السائقَ إلى الدبّوس الخطأ بصمت.**

CREATE TABLE IF NOT EXISTS order_live_locations (
    order_id    uuid PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
    customer_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    point       geography(Point, 4326) NOT NULL,
    accuracy_m  real,
    at          timestamptz NOT NULL DEFAULT now()
);

-- ══════════════════════════════════════════════════════════════════════
-- **وانتهاءُ الطلب يمحو النقطة — من أيّ بابٍ انتهى**
-- ══════════════════════════════════════════════════════════════════════
--
-- **موقعُ الزبون بعد التسليم ليس لأحد** — لا للسائق ولا للمكتب.
-- **والمحوُ في القاعدة لا في المعالج**: معالجٌ يُضاف غداً ويُنهي طلباً
-- لا يعرف بهذا الجدول، **والمحفّزُ يعرفه دائماً.**
CREATE OR REPLACE FUNCTION order_live_locations_clear() RETURNS trigger AS $$
BEGIN
    DELETE FROM order_live_locations WHERE order_id = NEW.id;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS orders_clear_live_location ON orders;
CREATE TRIGGER orders_clear_live_location
    AFTER UPDATE OF status, closed_at ON orders
    FOR EACH ROW
    WHEN (NEW.closed_at IS NOT NULL
          OR NEW.status IN ('delivered', 'cancelled', 'failed', 'rejected', 'refunded'))
    EXECUTE FUNCTION order_live_locations_clear();
