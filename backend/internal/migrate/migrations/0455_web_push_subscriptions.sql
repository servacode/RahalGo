-- ══════════════════════════════════════════════════════════════════════
-- **اشتراكاتُ إشعارات المتصفّح — Web Push** (قرارُ المالك ٢٠٢٦-١٠-١٧)
-- ══════════════════════════════════════════════════════════════════════
--
-- الزبونُ وصاحبُ المتجر على الويب يشتركان من متصفّحهما، **ويصلهما الخبرُ
-- والصفحةُ مغلقة** كما يصل الهاتف.
--
--   endpoint    عنوانُ الاشتراك عند مزوّد الدفع — **وهو الرمزُ في صفوف النقل**
--               (`notification_deliveries.token` بمنصّة `web`)
--   user_id     صاحبُه — والعنوانُ ينتقل إلى آخرِ من سجّله
--   session_id  عائلةُ الجلسة — **يُحذف مع الخروج** كرمز الهاتف (`D12`)
--   app         customer · merchant — **وإشعارٌ للمتجر وحدَه لا يرنّ عند الزبون**
--   p256dh/auth مفتاحا التشفير من المتصفّح (base64url)
--   last_ok_at  آخرُ قبولٍ من المزوّد · failures إخفاقاتٌ متتالية
--
-- **ويُحذف الصفُّ حين يردّ المزوّدُ ٤٠٤ أو ٤١٠** — اشتراكٌ انتهى لا يُنادى عليه
-- في كلّ حدثٍ إلى الأبد.
CREATE TABLE IF NOT EXISTS web_push_subscriptions (
    endpoint    text PRIMARY KEY,
    user_id     uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    session_id  uuid,
    app         text NOT NULL CHECK (app IN ('customer', 'merchant')),
    p256dh      text NOT NULL,
    auth        text NOT NULL,
    user_agent  text NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    last_ok_at  timestamptz,
    failures    integer NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS web_push_subscriptions_user_idx
    ON web_push_subscriptions (user_id);
CREATE INDEX IF NOT EXISTS web_push_subscriptions_session_idx
    ON web_push_subscriptions (session_id) WHERE session_id IS NOT NULL;
