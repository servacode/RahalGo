-- ══════════════════════════════════════════════════════════════════════
-- **طلباتُ السحب — قراراتُ المالك ٢٠٢٦-١٠-١٥**
-- ══════════════════════════════════════════════════════════════════════
--
--   طلبٌ مفتوحٌ واحد   **المفتوحُ معلَّقٌ أو قيدُ الصرف** — كان الفهرسُ الفريدُ على
--                      `pending` وحدَه، **فمن بدأ صرفُ طلبه قدّم ثانياً** ومالُه محجوزٌ
--                      للأوّل. والحارسُ الأوّلُ في `handleCreatePayout` داخلَ المعاملة،
--                      **وهذا الفهرسُ الثاني.**
--   دورُ الطالب        **يُكتب على الطلب لحظةَ إنشائه** — من التطبيق الذي طلب منه. كان
--                      يُشتقّ من أدوار الحساب (أوّلُها أبجديّاً)، **فسائقٌ له متجرٌ يظهر
--                      طلبُه «سائق» ولو طلب من تطبيق المتجر.** والقديمُ فارغٌ فيُشتقّ كما كان.

-- ── ١ · دورُ الطالب ───────────────────────────────────────────────────
ALTER TABLE payout_requests
    ADD COLUMN IF NOT EXISTS role text
    CHECK (role IS NULL OR role IN ('driver', 'merchant', 'sales'));

-- ── ٢ · طلبٌ مفتوحٌ واحدٌ لكلّ حساب ───────────────────────────────────
--
-- **ولا يُمسّ طلبٌ قائم**: إن وُجد حسابٌ له اليومَ معلَّقٌ وقيدُ صرفٍ معاً (قبل هذا
-- الحارس) **لا يُلغى أحدُهما بهجرة** — فهو مالٌ محجوزٌ وقرارُه للماليّة. **فيبقى الفهرسُ
-- القديمُ ويُنبَّه**، والحارسُ في المعالج يمنع الجديد. **ومتى صُفّيا يُنشأ الفهرسُ بهجرةٍ
-- لاحقة** (`IF NOT EXISTS`).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM payout_requests
                WHERE status IN ('pending', 'processing')
                GROUP BY user_id HAVING count(*) > 1) THEN
        RAISE NOTICE 'payout_requests: open duplicates exist — payout_requests_one_open not created yet';
    ELSE
        CREATE UNIQUE INDEX IF NOT EXISTS payout_requests_one_open
            ON payout_requests (user_id) WHERE status IN ('pending', 'processing');
        DROP INDEX IF EXISTS payout_requests_one_pending;
    END IF;
END $$;
