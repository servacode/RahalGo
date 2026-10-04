-- ══════════════════════════════════════════════════════════════════════
-- **سجلُّ الأحداث: إضافةٌ فقط · وحفظُ الدخول تسعين يوماً · وفهرسان**
-- ══════════════════════════════════════════════════════════════════════
--
-- (قراراتُ المالك ٢٠٢٦-١٠-٠٤ على فحص «سجل الأحداث»: الخامسُ والثاني.)
--
-- # ١ · الجهازُ مع العنوان
--
-- اللوحةُ الجانبيّةُ تعرض «العنوان والجهاز» — والجدولُ كان يحفظ العنوانَ
-- وحدَه. **عمودٌ يُضاف لا سطرٌ يُعدَّل**، والقديمُ يبقى فارغاً.
ALTER TABLE audit_log ADD COLUMN IF NOT EXISTS user_agent text NOT NULL DEFAULT '';

-- # ٢ · فهرسان: بالوقت، وبالفعل ثمّ الوقت
--
-- **فلترُ التاريخ وتبويبُ الفعل كانا يمسحان الجدولَ كلَّه** — وهو أسرعُ
-- جدولٍ نموّاً في المنصّة (٥٠٠ سطرٍ في ثماني ساعاتٍ على التجهيز).
CREATE INDEX IF NOT EXISTS audit_log_created_idx ON audit_log (created_at DESC);
CREATE INDEX IF NOT EXISTS audit_log_action_created_idx ON audit_log (action, created_at DESC);

-- # ٣ · أفعالُ الدخول والجلسة — وحدَها تُحذف بعد مدّة
--
-- **قائمةٌ واحدةٌ يقرؤها الحارسُ والتنظيف** — وفي الشيفرة نسختُها
-- (`auditSessionActions`)، **وفحصٌ يُسقط البناءَ إن افترقتا.**
--
-- **والمالُ والصلاحيّاتُ وتغييرُ كلمة السرّ والرقم لا يُحذف أبداً** —
-- ليست هنا.
CREATE OR REPLACE FUNCTION audit_session_actions() RETURNS text[]
LANGUAGE sql IMMUTABLE AS $fn$
    SELECT ARRAY[
        'auth.otp_login', 'auth.password_login', 'auth.pin_login',
        'auth.pin_verified', 'auth.refresh', 'auth.logout',
        'auth.password_failed', 'auth.sso', 'auth.whatsapp_verified'
    ]::text[]
$fn$;

-- # ٤ · الحارس: لا تعديلَ أبداً، ولا حذفَ إلّا بالتنظيف
--
-- **السجلُّ هو الدليلُ وقتَ الخلاف** — فلا يُعدَّل سطرٌ ولا يُمحى، حتّى
-- ممّن يصل القاعدةَ بيده.
--
-- **والاستثناءُ الوحيد**: حذفُ سطرِ دخولٍ أو جلسة **والرايةُ
-- `rahalgo.audit_prune` مرفوعةٌ في المعاملة نفسِها**. ولا يرفعها إلّا
-- `audit_prune_sessions` أدناه، وتُنزلها قبل أن تعود.
--
-- **ومن رفع الرايةَ بيده لا يحذف بها إلّا سطرَ دخول** — الحارسُ يفحص
-- الفعلَ نفسَه لا الرايةَ وحدَها.
--
-- **وما لا يمنعه هذا** (مكتوبٌ كي لا يُظنّ غيرُه): مالكُ الجدول في
-- بوستغرس يستطيع تعطيلَ الحارس (`ALTER TABLE ... DISABLE TRIGGER`). وتلك
-- خطوةٌ صريحةٌ مقصودةٌ لا خطأٌ عابر — وسكربتاتُ مسح بيئة التجربة تفعلها
-- علناً.
CREATE OR REPLACE FUNCTION audit_log_guard() RETURNS trigger
LANGUAGE plpgsql AS $fn$
BEGIN
    IF TG_OP = 'DELETE'
       AND current_setting('rahalgo.audit_prune', true) = 'on'
       AND OLD.action = ANY (audit_session_actions()) THEN
        RETURN OLD;
    END IF;
    RAISE EXCEPTION 'audit_log is append-only: % rejected', TG_OP
        USING ERRCODE = 'insufficient_privilege';
END
$fn$;

CREATE OR REPLACE FUNCTION audit_log_guard_truncate() RETURNS trigger
LANGUAGE plpgsql AS $fn$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only: TRUNCATE rejected'
        USING ERRCODE = 'insufficient_privilege';
END
$fn$;

DROP TRIGGER IF EXISTS audit_log_append_only ON audit_log;
CREATE TRIGGER audit_log_append_only
    BEFORE UPDATE OR DELETE ON audit_log
    FOR EACH ROW EXECUTE FUNCTION audit_log_guard();

DROP TRIGGER IF EXISTS audit_log_no_truncate ON audit_log;
CREATE TRIGGER audit_log_no_truncate
    BEFORE TRUNCATE ON audit_log
    FOR EACH STATEMENT EXECUTE FUNCTION audit_log_guard_truncate();

-- # ٥ · التنظيف: بابٌ واحدٌ ضيّق
--
-- **يحذف سطورَ الدخول والجلسة الأقدمَ من المدّة لا غير.** والمدّةُ من
-- الإعداد `security.audit_session_retention_days` (افتراضُها ٩٠) — ولا
-- تقلّ عن ثلاثين مهما كُتب.
--
-- `SECURITY DEFINER` كي يبقى البابَ الوحيدَ إن فُصلت أدوارُ القاعدة
-- يوماً: من لا يملك الحذفَ على الجدول يملك نداءَ هذه الدالّة وحدَها.
CREATE OR REPLACE FUNCTION audit_prune_sessions(keep_days int) RETURNS bigint
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $fn$
DECLARE
    n bigint;
BEGIN
    IF keep_days IS NULL OR keep_days < 30 THEN
        RAISE EXCEPTION 'audit_prune_sessions: keep_days must be >= 30 (got %)', keep_days;
    END IF;
    PERFORM set_config('rahalgo.audit_prune', 'on', true);
    DELETE FROM audit_log
     WHERE action = ANY (audit_session_actions())
       AND created_at < now() - make_interval(days => keep_days);
    GET DIAGNOSTICS n = ROW_COUNT;
    PERFORM set_config('rahalgo.audit_prune', 'off', true);
    RETURN n;
END
$fn$;
