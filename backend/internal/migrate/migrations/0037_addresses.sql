-- عناوين الزبون المحفوظة.
--
-- كان يكتب عنوانه ويضع الدبّوس على الخريطة **في كل طلب**. وزبونٌ يطلب ثلاث
-- مرّات أسبوعياً يكرّرها مئةً وخمسين مرّة في السنة إلى بيته نفسه.
--
-- والأثر عندنا أكبر منه في غيرنا: عنواننا **دبّوسٌ على خريطة** لا نصّاً يُلصق —
-- وإعادة ضبطه في كل مرّة عذاب، ودقّتُه هي ما يبلغ به السائق الباب.
CREATE TABLE user_addresses (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- تسمية يعرفها صاحبها: «البيت»، «المكتب»، «بيت أهلي»
    label        text NOT NULL,
    address_text text NOT NULL,
    location     geography(Point, 4326) NOT NULL,
    -- الافتراضي يُختار تلقائياً عند الطلب، فلا يُسأل من له عنوان واحد
    is_default   boolean NOT NULL DEFAULT false,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (label <> '' AND address_text <> '')
);

CREATE INDEX user_addresses_user_idx ON user_addresses (user_id, created_at DESC);

-- افتراضيٌّ واحد لكل مستخدم — يُفرض في القاعدة لا في الشيفرة: قاعدةٌ تُحرَس في
-- مكانٍ واحد لا تُخرَق من مسارٍ نُسي.
CREATE UNIQUE INDEX user_addresses_one_default ON user_addresses (user_id)
    WHERE is_default;

-- سقفٌ ليس ترقيماً بل حاجزُ معنى: من له عشرون عنواناً لا يجد عنوانه بينها.
-- يُفرض في المعالِج (القاعدة لا تملك عدّاً في CHECK).
