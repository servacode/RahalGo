-- ══════════════════════════════════════════════════════════════════════
-- **سجلُّ «انتهت جلستك» — لنعرف لماذا خرج الحساب** (قرارُ المالك ٢٠٢٦-١٠-٢١)
-- ══════════════════════════════════════════════════════════════════════
--
-- «اي لازم نسجل لنعرف سبب تسجيل الخروج». خرجت المندوبةُ جيهان والخادمُ يرى
-- جلستَها حيّة — **ولا سجلَّ لكلّ نداء**، فلم يُعرف السبب.
--
-- **يُكتب صفٌّ حين يُردّ الحسابُ بما يُخرجه**: تجديدٌ فشل (`refresh`)، أو نداءٌ
-- رُفض لأنّ جلستَه انتهت (`request`). **لا لكلّ رمزٍ منتهٍ** — ذاك يُجدَّد عاديّاً.
--
-- **ومعه ما يكفي للتشخيص**: الحسابُ والجلسةُ والتطبيقُ ونسختُه والمسارُ والرمزُ
-- والسبب. **ويُقلَّم بعد ثلاثين يوماً.**
CREATE TABLE IF NOT EXISTS auth_failures (
    id          bigserial PRIMARY KEY,
    at          timestamptz NOT NULL DEFAULT now(),
    kind        text NOT NULL,
    code        text NOT NULL,
    user_id     uuid,
    session_id  uuid,
    path        text NOT NULL DEFAULT '',
    client      text NOT NULL DEFAULT '',
    app_version text NOT NULL DEFAULT '',
    ip          text NOT NULL DEFAULT '',
    detail      text NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS auth_failures_user_at_idx ON auth_failures (user_id, at DESC);
CREATE INDEX IF NOT EXISTS auth_failures_at_idx ON auth_failures (at);
