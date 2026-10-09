-- ══════════════════════════════════════════════════════════════════════
-- **أخطاءُ الخادم مجمَّعةً بالساعة** (مراقبةُ المنصّة — الدفعةُ الأولى ٢٠٢٦-١٠-٠٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- صفٌّ واحدٌ لكلّ مسارٍ ورمزٍ في كلّ ساعة — **فألفُ خطأٍ على مسارٍ واحدٍ صفٌّ
-- واحدٌ عدُّه ألف**، لا ألفُ صفّ.
--
--   hour        بدايةُ الساعة (UTC)
--   route       نمطُ المسار كما في الموجّه (`/api/v1/orders/{id}`) — **لا الرابطُ
--               الخامّ**: لا معرّفَ ولا رقمَ هاتفٍ ولا نصَّ استعلام.
--   code        رمزُ الحالة (٥٠٠ فما فوق)
--   count       كم مرّةً في هذه الساعة
--   first_seen  أوّلُ مرّةٍ في الساعة
--   last_seen   آخرُ مرّة
--
-- **ويُكتب من الذاكرة دفعةً كلَّ دقيقة** — لا عند كلّ خطأ: **فخادمٌ يتعثّر
-- لا يُثقَل بكتابةٍ إضافيّةٍ مع كلّ ردّ.**
CREATE TABLE IF NOT EXISTS server_errors (
    hour       timestamptz NOT NULL,
    route      text        NOT NULL,
    code       int         NOT NULL,
    count      bigint      NOT NULL DEFAULT 0,
    first_seen timestamptz NOT NULL,
    last_seen  timestamptz NOT NULL,
    PRIMARY KEY (hour, route, code)
);

CREATE INDEX IF NOT EXISTS server_errors_last_seen ON server_errors (last_seen DESC);
