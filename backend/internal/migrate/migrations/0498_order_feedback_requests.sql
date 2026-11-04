-- ══════════════════════════════════════════════════════════════════════
-- **طلبُ رأيِ الزبون بعد التسليم بالواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٤)
-- ══════════════════════════════════════════════════════════════════════
--
-- بعد عشر دقائق من تسليم الطلب يصل الزبونَ بالواتساب رابطٌ لصفحةٍ عامّةٍ صغيرة
-- (`/rate/<رمز>`): نجومُ الخدمة، ونجومُ السائق، وكلمةٌ عن التطبيق. **النجومُ تُكتب في
-- `order_ratings` كما يكتبها التطبيق** فلا يختلف المتوسّطُ ولا السمعة.
--
-- # الرمزُ لا يُخزَّن
--
-- **بصمتُه وحدَها** (`token_hash`، sha256) كطلب الموقع (٠٤٨٤) — وينتهي بعد ٧ أيّام أو
-- عند أوّل إرسال. **وصفٌّ واحدٌ لكلّ طلب** (`order_id` مفتاح) فلا يُسأل الزبونُ مرّتين.
--
-- # والكلمةُ تعود إلى `order_ratings`
--
-- **أسقطت ٠١٢٥ العمودَ `comment`** لأنّ الزبونَ كان يشكو في حقلٍ لا يقرؤه أحد. **والمالكُ
-- أجاز صندوقاً جديداً بشرطٍ واحد: أن تراه المكاتب** — فيظهر على شاشة التقييمات في اللوحة.
-- (ونسخةٌ منه في صفّ الطلب هنا احتياطاً إن سقطت الكتابةُ الثانية.)

CREATE TABLE IF NOT EXISTS order_feedback_requests (
    order_id    uuid PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
    token_hash  text NOT NULL UNIQUE,
    -- فارغٌ ⇒ حُجز الصفُّ ولم تصل الرسالة (لا إعادةَ إرسال، تحرُّزاً من الإزعاج).
    sent_at     timestamptz,
    answered_at timestamptz,
    feedback    text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL
);

ALTER TABLE order_ratings ADD COLUMN IF NOT EXISTS app_feedback text NOT NULL DEFAULT '';
