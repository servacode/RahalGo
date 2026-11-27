-- ══════════════════════════════════════════════════════════════════════
-- **قائمة بوت الواتساب** — قرار المالك ٢٠٢٦-١١-٢٧
-- ══════════════════════════════════════════════════════════════════════
--
-- كلُّ من يراسل رقمَ المنصّة (عدا الكباتن والمندوبين والمتاجر) تصله قائمةٌ بستّة خيارات.
-- انظر `server/wa_bot_menu.go`.
--
-- ١) `wa_bot_chats` — حالُ كلّ محادثة: آخرُ نشاط (للسكوت قبل إعادة القائمة) والخطوةُ الجارية
--    وأجوبتُها. **في القاعدة لا في الذاكرة** — فإعادةُ تشغيل الخادم لا تقطع طلباً بنصفه.
-- ٢) `captain_applications` — «التسجيل ككابتن» من البوت، وتُعرض في اللوحة.
-- ٣) `orders.wa_bot` — طلبُ واتساب أنشأه البوت **بانتظار موافقة الإدارة**: لا يُقبل تلقائيّاً ولا
--    ينزل للكباتن قبل أن يكتب الموظّفُ الأجرةَ ويوافق.

CREATE TABLE IF NOT EXISTS wa_bot_chats (
    phone            text        PRIMARY KEY,
    last_activity_at timestamptz NOT NULL DEFAULT now(),
    state            text        NOT NULL DEFAULT 'idle'
                                 CHECK (state IN ('idle', 'menu', 'closed', 'order', 'track', 'admin', 'signup', 'done')),
    step             text        NOT NULL DEFAULT '',
    data             jsonb       NOT NULL DEFAULT '{}'::jsonb,
    -- **متى سُئل السؤالُ الجاري** — منه يُحسب التذكيرُ بعد ٣ دقائق.
    step_at          timestamptz,
    reminded_at      timestamptz,
    -- **ردودٌ غيرُ مفهومةٍ متتالية على الخطوة نفسِها** — بعد الثالثة يسكت البوت.
    unclear          int         NOT NULL DEFAULT 0,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS wa_bot_chats_flow_idx ON wa_bot_chats (step_at)
    WHERE state IN ('order', 'signup');

CREATE TABLE IF NOT EXISTS captain_applications (
    id            uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    phone         text        NOT NULL,
    full_name     text        NOT NULL,
    age           text        NOT NULL DEFAULT '',
    area          text        NOT NULL DEFAULT '',
    has_ebike     boolean     NOT NULL DEFAULT false,
    has_motorbike boolean     NOT NULL DEFAULT false,
    has_license   boolean     NOT NULL DEFAULT false,
    has_android   boolean     NOT NULL DEFAULT false,
    experience    text        NOT NULL DEFAULT '',
    hours         text        NOT NULL DEFAULT '',
    status        text        NOT NULL DEFAULT 'new' CHECK (status IN ('new', 'contacted', 'rejected')),
    updated_by    uuid        REFERENCES users (id),
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS captain_applications_status_idx ON captain_applications (status, created_at DESC);

ALTER TABLE orders ADD COLUMN IF NOT EXISTS wa_bot boolean NOT NULL DEFAULT false;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_wa_bot_ck;
ALTER TABLE orders ADD CONSTRAINT orders_wa_bot_ck CHECK (NOT wa_bot OR origin = 'whatsapp');
