-- كتالوج الإعدادات: بذر ما نقص، وتطهير ما لا يعرفه الخادم.
--
-- سبعةُ أرقام كانت مكتوبة في الشيفرة تحكم سلوك المنصة، ولا يملك المالك
-- تغييرها إلا بنشرٍ جديد. وقاعدة المشروع صريحة: **ما يحكم العمل يُدار من
-- اللوحة لا من الشيفرة**.

-- ── فصلُ أجر السائق إلى مفتاحين ──────────────────────────────────────────
--
-- كان `drivers.share_value` واحداً يخدم معنيين: نسبةً مئوية في نمط، ومبلغاً
-- بالليرة في نمطٍ آخر. فوجب أن يتّسع مداه لكليهما — فلم يحرس أيّاً منهما:
-- المئتان تمرّ لأنها مبلغٌ معقول، وهي نسبةٌ كارثية.
--
-- والقيمة القائمة تُنقل إلى مفتاحها الصحيح حسب النمط المعمول به الآن، كي لا
-- يتغيّر أجرُ سائقٍ واحد بهذا الترحيل.
INSERT INTO app_settings (key, value)
SELECT 'drivers.share_percent',
       COALESCE((SELECT value FROM app_settings WHERE key = 'drivers.share_value'), '70'::jsonb)
WHERE COALESCE((SELECT value#>>'{}' FROM app_settings WHERE key = 'drivers.share_mode'), 'percent') = 'percent'
ON CONFLICT (key) DO NOTHING;

INSERT INTO app_settings (key, value)
SELECT 'drivers.share_fixed',
       COALESCE((SELECT value FROM app_settings WHERE key = 'drivers.share_value'), '5000'::jsonb)
WHERE (SELECT value#>>'{}' FROM app_settings WHERE key = 'drivers.share_mode') = 'fixed'
ON CONFLICT (key) DO NOTHING;

-- والباقي بافتراضه — كي يجد المالك المفتاحين معاً حين يبدّل النمط
INSERT INTO app_settings (key, value) VALUES ('drivers.share_percent', '70'::jsonb)
    ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('drivers.share_fixed', '5000'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- ── جديدة ────────────────────────────────────────────────────────────────

-- كان السائق يأخذ ما شاء من الطلبات ما دام سقفه النقدي يتّسع. وخمسةُ طلبات
-- بيد سائقٍ واحد تعني أربعة زبائن ينتظرون ساعة — والسقف النقدي لا يمنع ذلك
-- لأن الطلبات الصغيرة تمرّ منه جميعاً.
INSERT INTO app_settings (key, value) VALUES ('drivers.max_active_orders', '2'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- لم يكن حدٌّ أدنى لطلب السحب: طلبٌ بليرةٍ واحدة يمرّ بدورة الموافقة كاملةً
-- ويشغل المالية — وقيمة القرار أكبر من قيمة المبلغ.
INSERT INTO app_settings (key, value) VALUES ('payouts.min_amount', '50000'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- كان `maxAddresses = 10` ثابتاً في المعالِج.
INSERT INTO app_settings (key, value) VALUES ('customers.max_addresses', '10'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- كان ٢٠ دقيقة افتراضَ عمودٍ في الترحيل 0035 — يرثه كل متجرٍ جديد ولا يملك
-- المالك تغييره لمن يأتي بعده.
INSERT INTO app_settings (key, value) VALUES ('merchants.default_prep_minutes', '20'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- طول كلمة المرور وعدد محاولات الدخول: قرارا أمانٍ يتّخذهما المالك، لا
-- قرارا نشرٍ ينتظران مبرمجاً.
INSERT INTO app_settings (key, value) VALUES ('security.password_min_length', '8'::jsonb)
    ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('security.login_max_attempts', '5'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- رقمٌ يظهر للزبون حين لا يجد جواباً. ولم يكن له وجود: الشكوى تذهب إلى
-- التذاكر وحدها، ومن لا يعرف التذاكر لا يجد باباً.
INSERT INTO app_settings (key, value) VALUES ('platform.support_phone', '""'::jsonb)
    ON CONFLICT (key) DO NOTHING;

-- ── تطهير ────────────────────────────────────────────────────────────────
--
-- كانت `Set` تقبل أيّ مفتاح: خطأٌ مطبعيّ يُنشئ مفتاحاً جديداً لا يقرؤه أحد،
-- فيمضي النظام بالافتراضي والمالك يظنّ أنه غيّر. صار المفتاح المجهول مرفوضاً
-- في الخادم — وهذا يحذف ما تسرّب قبل الحراسة.
--
-- والقائمة هنا مكتوبةٌ صراحةً لا مشتقّة: حذفٌ بشرطٍ ضمنيّ قد يمحو مفتاحاً
-- أُضيف في فرعٍ آخر ولم يُدمج بعد.
DELETE FROM app_settings WHERE key NOT IN (
    'orders.accept_timeout_min',
    'orders.driver_timeout_min',
    'orders.delivery_timeout_min',
    'orders.delivery_estimate_min',
    'orders.customer_cancel_window_sec',
    'drivers.share_mode',
    'drivers.share_percent',
    'drivers.share_fixed',
    'drivers.cash_limit',
    'drivers.max_active_orders',
    'merchants.default_commission_percent',
    'merchants.menu_requires_approval',
    'merchants.default_prep_minutes',
    'sales.commission_percent',
    'sales.activation_orders',
    'sales.monthly_target',
    'customers.max_addresses',
    'payouts.min_amount',
    'security.password_min_length',
    'security.login_max_attempts',
    'platform.invite_code',
    'platform.support_phone',
    'whatsapp.otp_template'
);
