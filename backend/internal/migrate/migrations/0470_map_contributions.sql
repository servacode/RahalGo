-- ══════════════════════════════════════════════════════════════════════
-- **ما يضيفه السائقون للخريطة** (قرارات المالك ٢٠٢٦-١٠-٢١)
-- ══════════════════════════════════════════════════════════════════════
--
-- ١ · **مدخلُ البناية** — بعد التسليم يحفظ السائقُ البابَ من موقعه، **فيُحفظ مع عنوان
--     الزبون** (لا مع الطلب): الطلبُ التالي إلى العنوان نفسه يوصل أيَّ سائقٍ إلى الباب.
--     **ولا يراه الزبون ولا تتغيّر نقطتُه.**
-- ٢ · **«موقعك الصح هون؟»** — إن بعُد البابُ عن نقطة الزبون أكثرَ من ١٠٠ م يُقترَح عليه
--     تعديلُها بإشعار، **ولا تُعدَّل إلّا بموافقته.**
-- ٣ · **طريقٌ مسكّر** — بلاغُ السائق يجعل التوجيهَ يتجنّبه ٢٤ ساعة، **وثلاثةَ أيّامٍ إن
--     أكّده سائقٌ ثانٍ.** وتزيله الإدارةُ إن انفتح.
-- ٤ · **«أضف معلم»** — لا يظهر لأحدٍ قبل موافقة الإدارة، **ثمّ يدخل «قرب المعلم».**
ALTER TABLE user_addresses ADD COLUMN IF NOT EXISTS entrance geography(Point, 4326);
ALTER TABLE user_addresses ADD COLUMN IF NOT EXISTS entrance_set_at timestamptz;

ALTER TABLE orders ADD COLUMN IF NOT EXISTS dropoff_suggest geography(Point, 4326);

CREATE TABLE IF NOT EXISTS road_closures (
    id          bigserial PRIMARY KEY,
    point       geography(Point, 4326) NOT NULL,
    reason      text NOT NULL,
    reported_by uuid REFERENCES users(id) ON DELETE SET NULL,
    reports     int NOT NULL DEFAULT 1,
    created_at  timestamptz NOT NULL DEFAULT now(),
    expires_at  timestamptz NOT NULL,
    cleared_at  timestamptz,
    cleared_by  uuid REFERENCES users(id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS road_closures_point_idx ON road_closures USING gist (point);
CREATE INDEX IF NOT EXISTS road_closures_live_idx ON road_closures (expires_at) WHERE cleared_at IS NULL;

CREATE TABLE IF NOT EXISTS landmark_suggestions (
    id           bigserial PRIMARY KEY,
    name         text NOT NULL,
    kind         text NOT NULL,
    point        geography(Point, 4326) NOT NULL,
    suggested_by uuid REFERENCES users(id) ON DELETE SET NULL,
    status       text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    decided_by   uuid REFERENCES users(id) ON DELETE SET NULL,
    decided_at   timestamptz
);
CREATE INDEX IF NOT EXISTS landmark_suggestions_pending_idx ON landmark_suggestions (created_at) WHERE status = 'pending';
