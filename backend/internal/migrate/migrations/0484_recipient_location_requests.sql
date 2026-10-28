-- ══════════════════════════════════════════════════════════════════════
-- **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩)
-- ══════════════════════════════════════════════════════════════════════
--
-- كان المتجرُ يلصق رابطَ موقعٍ يطلبه من المستلِم بالواتساب — نسخٌ ومشاركةٌ
-- ولصقٌ متعِب. **والآن يضغط زرّاً**: يصل المستلِمَ رابطٌ لصفحةٍ عامّةٍ صغيرة
-- فيها زرٌّ واحد «أرسل موقعي»، **ويصل الموقعُ إلى نموذج التوصيلة مباشرةً.**
--
-- # طلبٌ قبل التوصيلة لا عليها
--
-- **الطلبُ مسودّةٌ مستقلّةٌ مربوطةٌ بالمتجر ورقمِ المستلِم** — لا صفَّ طلبٍ
-- ولا مال. **والأجرةُ تُحسب من النقطة عند الإرسال كما تُحسب من الرابط**،
-- فلا تُعدَّل أجرةُ توصيلةٍ بعد إنشائها.
--
-- # والرمزُ لا يُخزَّن
--
-- **بصمتُه وحدَها** (`token_hash`، sha256) — من قرأ القاعدةَ لا يملك رابطاً.
-- **وينتهي بعد ٢٤ ساعة أو عند أوّل إرسال**، ولا تكشف الصفحةُ إلّا اسمَ المتجر.
CREATE TABLE IF NOT EXISTS recipient_location_requests (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id  uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
    requested_by uuid REFERENCES users(id) ON DELETE SET NULL,
    phone        text NOT NULL,
    -- `send` · `fetch` — يغيّر جملةَ الصفحة («لاستلام طلبك» أو «ليستلم السائقُ منك»).
    mode         text NOT NULL DEFAULT 'send' CHECK (mode IN ('send', 'fetch')),
    token_hash   text NOT NULL UNIQUE,
    -- أوصلت رسالةُ الواتساب؟ — وإلّا يشارك المتجرُ الرابطَ بنفسه.
    whatsapp_sent boolean NOT NULL DEFAULT false,
    lat          double precision,
    lng          double precision,
    accuracy_m   double precision,
    received_at  timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL
);

CREATE INDEX IF NOT EXISTS recipient_location_requests_merchant_idx
    ON recipient_location_requests (merchant_id, created_at DESC);
