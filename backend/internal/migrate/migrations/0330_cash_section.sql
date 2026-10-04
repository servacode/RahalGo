-- ══════════════════════════════════════════════════════════════════════
-- **قسمُ «النقد والصندوق» — قراراتُ المالك 2026-10-04**
-- ══════════════════════════════════════════════════════════════════════
--
-- # 1 · «منذ» تُحسب بالأقدم الذي لم يُسلَّم بعد (الأقدمُ يُسدَّد أوّلاً)
--
-- كان «منذ» = أقدمُ قبضٍ **بعد آخر تسليم**. فسائقٌ قبض مئةً قبل أسبوعٍ وسلّم
-- خمسين اليوم يظهر «اليوم» — والخمسون الباقية عمرُها أسبوع.
--
-- **والصحيح**: كلُّ تسليمٍ يسدّد أقدمَ ما قُبض أوّلاً. فنجمع القبوضَ بترتيبها،
-- وأوّلُ قبضٍ يتجاوز مجموعُه المتراكمُ مجموعَ ما سُلّم هو أقدمُ مالٍ باقٍ.
-- ولا شيءَ باقٍ ⇒ NULL.
--
-- دالّةٌ واحدة يقرؤها كشفُ النقد، وعدُّ الرئيسيّة، وحارسُ الإيقاف التلقائيّ.
CREATE OR REPLACE FUNCTION driver_cash_oldest_unpaid(p_driver uuid)
RETURNS timestamptz LANGUAGE sql STABLE AS $$
    WITH paid AS (
        SELECT COALESCE(-sum(amount), 0) AS v
        FROM driver_cash_entries WHERE driver_id = p_driver AND amount < 0
    ), pos AS (
        SELECT created_at,
               sum(amount) OVER (ORDER BY created_at, id) AS cum
        FROM driver_cash_entries WHERE driver_id = p_driver AND amount > 0
    )
    SELECT min(pos.created_at) FROM pos, paid WHERE pos.cum > paid.v
$$;

-- # 2 · تنبيهُ «لم يسلّم منذ أيام» + إيقافُ الطلبات النقديّة (اختياريّ، مطفأ)
--
-- عددُ الأيّام إعدادٌ واحد يقرؤه التنبيهُ في الرئيسيّة والإيقافُ معاً.
-- والإيقافُ مطفأٌ افتراضاً بقرار المالك.
INSERT INTO app_settings (key, value) VALUES ('drivers.cash_overdue_days', '3')
ON CONFLICT (key) DO NOTHING;
INSERT INTO app_settings (key, value) VALUES ('drivers.cash_overdue_stop', 'false')
ON CONFLICT (key) DO NOTHING;

-- driver_cash_overdue_stopped **هل يُمنع السائقُ من الطلبات النقديّة الآن؟**
--
-- نعم فقط إن كان الإيقافُ مُشعَلاً، ومالٌ بيده لم يُسلَّم منذ «عدد الأيّام» أو أكثر.
-- يقرؤها حارسُ القبول (`cashbox.GuardTx`) ومرشَّحُ الدور معاً — فلا يُعرض على
-- السائق طلبٌ يُردّ عند الضغط.
-- (والقيمتان الاحتياطيّتان = افتراضُ الفهرس؛ حارسٌ في الاختبار يطابقهما.)
CREATE OR REPLACE FUNCTION driver_cash_overdue_stopped(p_driver uuid)
RETURNS boolean LANGUAGE sql STABLE AS $$
    SELECT CASE
        WHEN COALESCE((SELECT (value #>> '{}')::boolean FROM app_settings
                        WHERE key = 'drivers.cash_overdue_stop'), false)
        THEN COALESCE(driver_cash_oldest_unpaid(p_driver) <= now() - make_interval(
                 days => COALESCE((SELECT (value #>> '{}')::numeric::int FROM app_settings
                                    WHERE key = 'drivers.cash_overdue_days'), 3)), false)
        ELSE false
    END
$$;
