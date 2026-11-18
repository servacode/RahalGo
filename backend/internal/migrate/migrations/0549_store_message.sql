-- ══════════════════════════════════════════════════════════════════════
-- «رسالة للمتاجر» — واتساب + إشعار التطبيق لكلّ المتاجر، يدويّ أو مجدوَل (قرار المالك ٢٠٢٦-١١-١٨)
-- ══════════════════════════════════════════════════════════════════════
--
-- بوتُ واتساب نفسُه يرسل رموز التسجيل، فحظرُه يوقف التسجيل. لذلك الإرسال متسلسل (رسالة كلّ ٢٠–٤٥ ثانية)
-- من عامل خلفيّ، وكلُّ رسالة صفٌّ هنا:
--   * التشغيلة (store_message_runs): من ضغط/جاء الموعد، بعددها وحالتها.
--   * التسليم (store_message_deliveries): صفّ لكلّ متجر؛ pending ينتظر دوره، ومن انقطع الخادم أثناءه
--     (sending) يُوسم failed عند الإقلاع ولا يُعاد (لا نعرف إن وصل — والتكرارُ أخطر من الفقد)، والباقي
--     pending يكمل بعد الإقلاع.
--
-- «مرّةً لكلّ متجر باليوم» تحرسها القاعدة: فهرسٌ فريد على (متجر، يوم دمشق) لكلّ تسليم غير تجريبيّ.
-- فالضغطتان المتتاليتان أو تداخل اليدويّ مع الآليّ لا يرسلان مرّتين مهما حدث في الشيفرة.
-- والتسليم التجريبيّ (متجر واحد للاختبار) خارج الفهرس عمداً.

CREATE TABLE IF NOT EXISTS store_message_runs (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    trigger      text        NOT NULL CHECK (trigger IN ('manual', 'auto', 'test')),
    status       text        NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'done', 'stopped')),
    -- سببُ التوقّف: bot_not_ready (البوت غير جاهز) — وفارغٌ لغيره
    stop_reason  text        NOT NULL DEFAULT '',
    title        text        NOT NULL DEFAULT '',
    body         text        NOT NULL,
    -- يوم دمشق وقت التخطيط — به يُحجز «تشغيلة آليّة واحدة باليوم»
    day          date        NOT NULL,
    target_count int         NOT NULL DEFAULT 0 CHECK (target_count >= 0),
    sent_count   int         NOT NULL DEFAULT 0 CHECK (sent_count >= 0),
    failed_count int         NOT NULL DEFAULT 0 CHECK (failed_count >= 0),
    created_by   uuid        REFERENCES users (id) ON DELETE SET NULL,
    started_at   timestamptz NOT NULL DEFAULT now(),
    finished_at  timestamptz
);

CREATE INDEX IF NOT EXISTS store_message_runs_started_idx ON store_message_runs (started_at DESC);
CREATE INDEX IF NOT EXISTS store_message_runs_running_idx ON store_message_runs (status) WHERE status = 'running';

CREATE TABLE IF NOT EXISTS store_message_deliveries (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      uuid        NOT NULL REFERENCES store_message_runs (id) ON DELETE CASCADE,
    merchant_id uuid        NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    phone       text        NOT NULL,
    day         date        NOT NULL,
    is_test     boolean     NOT NULL DEFAULT false,
    status      text        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sending', 'sent', 'failed')),
    error       text        NOT NULL DEFAULT '',
    created_at  timestamptz NOT NULL DEFAULT now(),
    sent_at     timestamptz
);

-- الضمانة: متجرٌ واحد · يومٌ واحد · رسالةٌ حقيقيّة واحدة.
CREATE UNIQUE INDEX IF NOT EXISTS store_message_deliveries_once_per_day
    ON store_message_deliveries (merchant_id, day) WHERE NOT is_test;

CREATE INDEX IF NOT EXISTS store_message_deliveries_pending_idx
    ON store_message_deliveries (created_at) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS store_message_deliveries_run_idx ON store_message_deliveries (run_id);
