-- ══════════════════════════════════════════════════════════════════════
-- **سلسلةُ تعريف المتجر بالإشعارات** (طلبُ المالك ٢٠٢٦-١٠-١٢)
-- ══════════════════════════════════════════════════════════════════════
--
-- صفٌّ لكلّ صاحبِ متجرٍ وصلته السلسلة: **أين وصل، وفي أيّ موعدٍ آخرَ مرّة.**
--
--   next_step  رقمُ الإشعار التالي (من صفر) في `merchants.onboarding_push_messages`
--   last_slot  آخرُ موعدٍ أُرسل فيه («2026-10-12/11») — **فلا يصل إشعاران في موعدٍ واحد**
--
-- **ومن لا صفَّ له لم يبدأ بعد** — يبدأ في الموعد التالي بعد تشغيل الميزة أو
-- بعد فتح حسابه.
CREATE TABLE IF NOT EXISTS merchant_onboarding_push (
    user_id    uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    next_step  int         NOT NULL DEFAULT 0,
    last_slot  text        NOT NULL DEFAULT '',
    updated_at timestamptz NOT NULL DEFAULT now()
);
