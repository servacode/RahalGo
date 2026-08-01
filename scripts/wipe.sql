-- ══════════════════════════════════════════════════════════════════════════
--  تنظيف قاعدة التطوير: كل الحسابات وما يخصّها.
--
--  **يُحذف**: الحسابات كلُّها (٢٣) · المتاجر (٦) بقوائمها وساعاتها · الطلبات
--  (١٢) بأصنافها وأحداثها وتقييماتها · المحافظ ودفاترها · صناديق السائقين ·
--  طلبات السحب والانضمام · التذاكر · الإشعارات · العناوين · المفضّلة ·
--  الجلسات · سجلّ التدقيق · الوسائط (شعارات وصور الحسابات المحذوفة).
--
--  **يبقى**: التصنيفات · مناطق التوصيل · الإعدادات (٢٣) · الأدوار · اللافتات
--  · أكواد الخصم. هذه **بنية عملٍ لا بيانات عرض** — بلا منطقةِ توصيلٍ واحدة
--  لا يُقبل أيّ طلب، وبلا تصنيفٍ لا يُنشأ متجر.
--
--  والحذف داخل معاملة واحدة: إمّا أن يتمّ كلُّه أو لا يتمّ منه شيء. وقاعدةٌ
--  نُظّفت نصفَ تنظيفٍ أسوأ من قاعدةٍ لم تُمسّ.
-- ══════════════════════════════════════════════════════════════════════════
BEGIN;

-- الإعدادات تبقى، لكنّ عمود «من غيّرها» يشير إلى حسابٍ سيُحذف.
UPDATE app_settings SET updated_by = NULL;

-- ── ما يتعلّق بالطلبات ───────────────────────────────────────────────────
DELETE FROM order_ratings;
DELETE FROM order_events;
DELETE FROM order_items;
DELETE FROM promo_redemptions;
DELETE FROM orders;

-- ── الدفاتر المالية ──────────────────────────────────────────────────────
DELETE FROM driver_cash_entries;
DELETE FROM driver_cash_boxes;
DELETE FROM payout_requests;
DELETE FROM wallet_transactions;
DELETE FROM wallets;

-- ── الدعم ───────────────────────────────────────────────────────────────
DELETE FROM ticket_replies;
DELETE FROM tickets;

-- ── المتاجر وقوائمها ────────────────────────────────────────────────────
DELETE FROM modifier_options;
DELETE FROM modifier_groups;
DELETE FROM menu_items;
DELETE FROM menu_sections;
DELETE FROM merchant_hours;
DELETE FROM merchant_leads;
DELETE FROM merchants;

-- ── ما يخصّ الحسابات ────────────────────────────────────────────────────
DELETE FROM user_favorites;
DELETE FROM user_addresses;
DELETE FROM notifications;
DELETE FROM refresh_tokens;
DELETE FROM otp_codes;
DELETE FROM audit_log;
DELETE FROM user_roles;

-- الوسائط: شعارات متاجر حُذفت وصور حسابات حُذفت. واللافتات صفرٌ فلا يتيم.
DELETE FROM media;

DELETE FROM users;

COMMIT;

\echo ══════ ما بقي ══════
SELECT 'الحسابات' AS الجدول, count(*) AS العدد FROM users
UNION ALL SELECT 'المتاجر', count(*) FROM merchants
UNION ALL SELECT 'الطلبات', count(*) FROM orders
UNION ALL SELECT 'المحافظ', count(*) FROM wallets
UNION ALL SELECT 'قيود المحافظ', count(*) FROM wallet_transactions
UNION ALL SELECT 'سجلّ التدقيق', count(*) FROM audit_log
UNION ALL SELECT 'الإشعارات', count(*) FROM notifications
UNION ALL SELECT 'الوسائط', count(*) FROM media;

\echo
\echo ══════ البنية الباقية ══════
SELECT 'التصنيفات' AS الجدول, count(*) AS العدد FROM categories
UNION ALL SELECT 'مناطق التوصيل', count(*) FROM delivery_zones
UNION ALL SELECT 'الإعدادات', count(*) FROM app_settings
UNION ALL SELECT 'الأدوار', count(*) FROM roles
UNION ALL SELECT 'أكواد الخصم', count(*) FROM promo_codes
UNION ALL SELECT 'الهجرات', count(*) FROM schema_migrations;
