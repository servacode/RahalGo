-- ══════════════════════════════════════════════════════════════════════
-- **رقمُ الطلبات بحسب الدوام — صباحيٌّ ومسائيّ** (قرارُ المالك ٢٠٢٦-١٠-٢٠)
-- ══════════════════════════════════════════════════════════════════════
--
-- «نفس المتجر بدي يكون في رقمين ودوام صباحي ومسائي… لكل دوام رقم هاتف مشان الطلبات
-- وقت تحويل الطلبات ع واتس اب». **فالمتجرُ يختار:**
--
--   order_phone_mode     'full'  رقمٌ واحدٌ كما كان (الافتراض — لا يتغيّر شيءٌ لمتجرٍ قائم)
--                        'split' رقمٌ للصباحيّ ورقمٌ للمسائيّ
--   order_phone_morning  رقمُ الدوام الصباحيّ
--   order_phone_evening  رقمُ الدوام المسائيّ
--   order_phone_evening_from  ساعةُ بدء المسائيّ بتوقيت سوريا — وكلُّ متجرٍ يحدّدها
--
-- **وهما رقما طلبات لا حسابا دخول** — صاحبُ المتجر يبقى حساباً واحداً.
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS order_phone_mode text NOT NULL DEFAULT 'full';
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS order_phone_morning text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS order_phone_evening text;
ALTER TABLE merchants ADD COLUMN IF NOT EXISTS order_phone_evening_from time NOT NULL DEFAULT '16:00';

DO $$ BEGIN
    ALTER TABLE merchants ADD CONSTRAINT merchants_order_phone_mode_chk
        CHECK (order_phone_mode IN ('full', 'split'));
EXCEPTION WHEN duplicate_object THEN NULL; END $$;
