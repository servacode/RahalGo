-- ══════════════════════════════════════════════════════════════════════
-- **دفعُ الإشعار: حقيقةٌ دائمةٌ لكلّ هدف**
-- ══════════════════════════════════════════════════════════════════════
--
-- (`PF-09` · `R23` — دورةُ إصلاحٍ ١٥، ٢٠٢٦-٠٩-٠٧.)
--
-- # العلّة
--
-- **`push.SendToUser` يُنادى مرّةً واحدةً بلا إعادة.** فإن سقطت الشبكةُ
-- أو ردّت المنصّةُ خمسمئة أو مات المنفّذُ **ضاع التنبيهُ بلا أثر**:
-- **لا صفَّ انتظارٍ ولا حالٌ معلَّقةٌ ولا سطرٌ يقول إنّه لم يصل.**
--
-- **والخبرُ نفسُه محفوظٌ في `notifications`** — يُقرأ عند الفتح. **لكنّ
-- السائقَ الذي يعلم بطلبٍ بعد ساعةٍ طلبٌ ضائع.**
--
-- # ولماذا لكلّ هدفٍ لا لكلّ إشعار
--
-- **قيس**: `device_tokens` فريدُها **الرمزُ وحدَه** — و`user_id` فهرسٌ
-- غيرُ فريد، **و`Register` لا يحذف رموزَ صاحبه الأخرى.** **فللحساب
-- الواحد أجهزةٌ عدّة، ولتطبيقاتٍ عدّة** (`device_tokens_user_app_idx`).
--
-- **وحالٌ واحدةٌ على الإشعار تكذب**: **جهازٌ قَبِل وجهازٌ انقطع وجهازٌ
-- ماتَ رمزُه** — **ثلاثةُ مصائرَ في خانةٍ واحدة.**
--
-- # ولا كونَ إشعاراتٍ ثانٍ
--
-- **`notifications` تبقى المضمونَ والنيّة** — **وهذه صفوفُ نقلٍ إليها**،
-- لا نسخةٌ من نصّها. **ولا يُكرَّر عنوانٌ ولا نصّ.**
--
-- # والتاريخُ يبقى تاريخاً
--
-- **`push_pending` افتراضُها `false`** — **فلا يُعاد دفعُ إشعارٍ قديم.**
-- **ومن أعاد دفعَ عشرين ألفَ إشعارٍ قديمٍ في لحظةِ هجرةٍ أغرق المنصّةَ
-- وأيقظ كلَّ هاتفٍ بخبرِ الأمس.**

-- ── نيّةُ الدفع على صفّ الإشعار نفسِه ──────────────────────────────────
--
-- **وعمودٌ في الصفّ لا جدولٌ ثانٍ** — **فيُكتب في إدراجِ الإشعار نفسِه،
-- ولا فجوةَ بين كتابتين تُضيّع التنبيه.** (وهي علّةُ `PF-07` بعينها.)
ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS push_pending boolean NOT NULL DEFAULT false;

-- **وتطبيقاتُ الهدف تُحفَظ مع النيّة** — **فإشعارُ سائقٍ لا يرنّ في
-- تطبيق الزبون على الهاتف نفسِه.** (وهو ما كان يفعله `msg.Apps` في
-- النداء المباشر، **ونيّةٌ مؤجَّلةٌ لا تتذكّره ما لم يُكتَب.**)
--
-- **وفارغةٌ تعني كلَّ الأجهزة** — كما كان.
ALTER TABLE notifications
    ADD COLUMN IF NOT EXISTS push_apps text[] NOT NULL DEFAULT '{}';

-- **والفهرسُ جزئيٌّ** — المعلَّقُ قليلٌ والمحفوظُ كثير.
CREATE INDEX IF NOT EXISTS notifications_push_pending_idx
    ON notifications (created_at)
    WHERE push_pending;

-- ── صفُّ نقلٍ لكلّ هدف ────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS notification_deliveries (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,

    -- **والرمزُ نصٌّ بلا مفتاحٍ أجنبيّ** — **فالرمزُ الميّتُ يُحذف من
    -- `device_tokens` ويبقى سجلُّ محاولته يُقرأ.** **ومن ربطهما محا
    -- الدليلَ مع الرمز.**
    token    text NOT NULL,
    platform text NOT NULL,

    -- ══════════════════════════════════════════════════════════════════
    -- **وأسماءُ الحالات تقول ما تعرفه لا ما نتمنّاه**
    -- ══════════════════════════════════════════════════════════════════
    --
    --   pending     لم يُقبَل بعد — ويُعاد حتّى تنفد المحاولات
    --   accepted    **المزوّدُ قَبِل الطلب** — ولا يعني أنّ هاتفاً عرضه
    --   failed      نفدت المحاولاتُ — والخبرُ باقٍ في التطبيق
    --   dead_token  المنصّةُ رفضت الرمزَ نهائيّاً — فحُذف ولا يُعاد
    --
    -- **ولا `delivered_to_device`** — **لا إقرارَ من جهاز، فالاسمُ كذب.**
    state text NOT NULL DEFAULT 'pending'
        CHECK (state IN ('pending', 'accepted', 'failed', 'dead_token')),

    attempts        integer NOT NULL DEFAULT 0,
    next_attempt_at timestamptz NOT NULL DEFAULT now(),

    -- claimed_until **حجزٌ بأجل** — **فمنفّذٌ مات لا يحبس عملاً أبداً.**
    claimed_until timestamptz,

    last_error       text NOT NULL DEFAULT '',
    -- last_error_class **صنفُ الخطأ لا نصُّه** — يُعَدُّ ويُبحَث فيه.
    last_error_class text NOT NULL DEFAULT '',

    -- provider_accepted_at **لحظةُ قبول المزوّد** — لا لحظةُ عرضِ الهاتف.
    provider_accepted_at timestamptz,

    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),

    -- **ولا هدفان لإشعارٍ ورمزٍ واحد** — **الحارسُ ضدّ تفريعٍ مكرَّر.**
    UNIQUE (notification_id, token)
);

-- **والمستحقُّ وحدَه يُفهرَس** — والمقبولُ يتراكم ولا يُقرأ في الجولة.
CREATE INDEX IF NOT EXISTS notification_deliveries_due_idx
    ON notification_deliveries (next_attempt_at)
    WHERE state = 'pending';

-- **وللعرض الإداريّ**: ما أخفق نهائيّاً يُقرأ بلا مسحٍ كامل.
CREATE INDEX IF NOT EXISTS notification_deliveries_failed_idx
    ON notification_deliveries (updated_at)
    WHERE state = 'failed';
