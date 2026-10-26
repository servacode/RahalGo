-- ══════════════════════════════════════════════════════════════════════
-- **يومُ الحساب من ٩ الصبح لـ ٩ الصبح** — قرارُ المالك ٢٠٢٦-١٠-٢٦
-- ══════════════════════════════════════════════════════════════════════
--
-- «كل يوم الساعة 9 صبح نستلم من السائقين، حسابنا من ال9 صباحاً لل9 صباحاً.
-- كل يوم وقت نفتح الدوام نحاسب على اليوم السابق.» و«كلشي لازم يكون أوّل ما
-- نفتح الدوام الصبح 9».
--
-- **ما قُبض قبل التاسعة يُحسب على اليوم السابق** — بتوقيت دمشق (+3 ثابتة).

-- **والساعةُ إعدادٌ لا رقمٌ مكتوب** (المالك: «اذا فتحنا 8 يصير الحساب ع8») —
-- `platform.day_start_hour`، افتراضُها 9. وكلُّ يومٍ في المنصّة يُقاس بها:
-- الاستلامُ من السائقين، والإغلاقُ، والتقاريرُ، وعددُ الطلبات اليوميّ.
CREATE OR REPLACE FUNCTION day_start_hour()
RETURNS int LANGUAGE sql STABLE AS $$
    SELECT COALESCE((SELECT (value #>> '{}')::numeric::int FROM app_settings
                      WHERE key = 'platform.day_start_hour'), 9)
$$;

-- account_day **يومُ الحساب للحظةٍ ما** — تاريخُ اليوم الذي بدأ بساعة فتح الدوام.
CREATE OR REPLACE FUNCTION account_day(ts timestamptz)
RETURNS date LANGUAGE sql STABLE AS $$
    SELECT ((ts AT TIME ZONE 'Asia/Damascus') - make_interval(hours => day_start_hour()))::date
$$;

-- account_day_start **لحظةُ بدء يوم الحساب** — ساعةُ فتح الدوام بدمشق.
CREATE OR REPLACE FUNCTION account_day_start(d date)
RETURNS timestamptz LANGUAGE sql STABLE AS $$
    SELECT (d::timestamp + make_interval(hours => day_start_hour())) AT TIME ZONE 'Asia/Damascus'
$$;

-- driver_cash_overdue_stopped **التأخّرُ بأيّام الحساب لا بالساعات** — اختيارُ المالك
-- ٢٠٢٦-١٠-٢٦: «بعد يوم حساب كامل». مالُ يومٍ يُسلَّم بتاسعة اليوم التالي، **فإن مرّت
-- عليه تاسعةٌ ثانيةٌ ولم يُسلَّم صار متأخّراً** (والأيّامُ من الإعداد، افتراضُها 3).
CREATE OR REPLACE FUNCTION driver_cash_overdue_stopped(p_driver uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT CASE
        WHEN COALESCE((SELECT (value #>> '{}')::boolean FROM app_settings
                        WHERE key = 'drivers.cash_overdue_stop'), false)
        THEN COALESCE(driver_cash_oldest_unpaid(p_driver) < account_day_start(account_day(now())) - make_interval(
                 days => COALESCE((SELECT (value #>> '{}')::numeric::int FROM app_settings
                                    WHERE key = 'drivers.cash_overdue_days'), 3)), false)
        ELSE false
    END
$$;
