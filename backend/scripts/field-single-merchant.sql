-- **ميدانٌ نظيف: مندوبٌ واحد، ومتجرٌ جاء عن طريقه، ولا صنفَ من خارجه.**
--
-- # لماذا
--
-- التجربةُ تُقاس، **وميدانٌ فيه ثلاثةُ متاجرَ وأقسامٌ تخلط أصنافَها يقيس
-- شيئين معاً** — فلا يُعرف أيُّهما أعطى النتيجة.
--
-- قرارُ المالك (٢٠٢٦-٠٨-٠٣): «خلّي المندوب والمتجر عن طريقه، **لا أصناف من
-- خارج المتجر** كي نختبر كلّ شيء».
--
-- # ما يبقى
--
--	المندوب  ←  زياد العبدالله
--	المتجر   ←  مطعم بيت الرقة (جاء عن طريقه) · ٣٢ صنفاً · ولا صنفَ في أقسام المنصة
--	الزبون   ←  سليمان الخطيب
--	السائقون ←  عمر · بشار · مصطفى — **وثلاثتُهم في الدوام**
--
-- # وما يُزال
--
-- **قصر الشام والفرات ومَن معهما** — زُرعت لفحوصٍ سابقة، **وكلُّ أصنافها في
-- أقسام المنصة**: وهي «الأصناف من خارج المتجر» بعينها.
--
-- **ويُزال أصحابُها ومندوبُهم** فلا تبقى حساباتٌ في اللوحة بلا متاجر.
--
-- # وأقسامُ المنصة تُطفأ لا تُحذف
--
-- **هي ميزةٌ لا بيانات**: تُعاد بضغطة حين نختبر تعدّدَ المصادر. **وحذفُها
-- يجعل إعادتَها عملاً، وإطفاؤها يجعلها قراراً.**

BEGIN;

-- ١ · المتاجرُ التي ليست عن طريق زياد — **وأصنافُها معها**
CREATE TEMP TABLE gone AS
SELECT m.id, m.owner_user_id
FROM merchants m
WHERE m.name <> 'مطعم بيت الرقة';

DELETE FROM modifier_options WHERE group_id IN (
    SELECT g.id FROM modifier_groups g
    JOIN menu_items i ON i.id = g.item_id
    WHERE i.merchant_id IN (SELECT id FROM gone));
DELETE FROM modifier_groups WHERE item_id IN (
    SELECT id FROM menu_items WHERE merchant_id IN (SELECT id FROM gone));
DELETE FROM user_favorites WHERE merchant_id IN (SELECT id FROM gone);
DELETE FROM menu_items WHERE merchant_id IN (SELECT id FROM gone);
DELETE FROM menu_sections WHERE merchant_id IN (SELECT id FROM gone);
DELETE FROM merchant_hours WHERE merchant_id IN (SELECT id FROM gone);
DELETE FROM merchant_warnings WHERE merchant_id IN (SELECT id FROM gone);
DELETE FROM merchants WHERE id IN (SELECT id FROM gone);

-- ٢ · وأصحابُها — **ولا يُترك حسابٌ بلا متجر**
DELETE FROM user_roles WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM wallet_transactions WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM wallets WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM user_addresses WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM refresh_tokens WHERE user_id IN (SELECT owner_user_id FROM gone);
-- **وسجلُّ تدقيقهم معهم** — سجلٌّ يشير إلى حسابٍ لا وجودَ له لا يُقرأ، **وهؤلاء
-- زرعُ فحوصٍ لا تاريخُ عمل.**
DELETE FROM notifications WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM ticket_replies WHERE ticket_id IN (
    SELECT id FROM tickets WHERE customer_id IN (SELECT owner_user_id FROM gone) OR created_by IN (SELECT owner_user_id FROM gone));
DELETE FROM tickets WHERE customer_id IN (SELECT owner_user_id FROM gone) OR created_by IN (SELECT owner_user_id FROM gone);
DELETE FROM payout_requests WHERE user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM audit_log WHERE actor_user_id IN (SELECT owner_user_id FROM gone);
DELETE FROM users WHERE id IN (SELECT owner_user_id FROM gone);

-- ٣ · ومَن زُرع للفحوص: مندوبٌ ثانٍ · زبونٌ ثانٍ · سائقٌ رابع
CREATE TEMP TABLE extras AS
SELECT id FROM users
WHERE phone IN ('+963977888999', '+963933000111', '+963955111222');

DELETE FROM user_roles WHERE user_id IN (SELECT id FROM extras);
DELETE FROM wallet_transactions WHERE user_id IN (SELECT id FROM extras);
DELETE FROM wallets WHERE user_id IN (SELECT id FROM extras);
DELETE FROM user_addresses WHERE user_id IN (SELECT id FROM extras);
DELETE FROM driver_cash_entries WHERE driver_id IN (SELECT id FROM extras);
DELETE FROM driver_cash_boxes WHERE driver_id IN (SELECT id FROM extras);
DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM extras);
DELETE FROM notifications WHERE user_id IN (SELECT id FROM extras);
DELETE FROM ticket_replies WHERE ticket_id IN (
    SELECT id FROM tickets WHERE customer_id IN (SELECT id FROM extras) OR created_by IN (SELECT id FROM extras));
DELETE FROM tickets WHERE customer_id IN (SELECT id FROM extras) OR created_by IN (SELECT id FROM extras);
DELETE FROM payout_requests WHERE user_id IN (SELECT id FROM extras);
DELETE FROM audit_log WHERE actor_user_id IN (SELECT id FROM extras);
DELETE FROM users WHERE id IN (SELECT id FROM extras);

-- ٤ · وأقسامُ المنصة تُطفأ — **ميزةٌ تُعاد بضغطة، لا بياناتٌ تُبنى من جديد**
UPDATE platform_sections SET active = false;

-- ٥ · والسائقون الثلاثة في الدوام
--
-- **وترتيبُ الدور بمن بكّر**: `shift_started_at` متفاوتٌ بدقيقة، **فيُقرأ
-- الترتيبُ في التجربة ولا يكون عشوائياً.**
UPDATE users u SET on_shift = true,
    shift_started_at = now() - (
        CASE u.phone WHEN '+963941112233' THEN interval '3 minutes'
                     WHEN '+963942223344' THEN interval '2 minutes'
                     ELSE interval '1 minute' END),
    last_assigned_at = NULL
WHERE u.phone IN ('+963941112233', '+963942223344', '+963943334455');

COMMIT;

-- والقراءةُ بعده
SELECT 'المتاجر' AS ما, string_agg(name, ' · ') AS من FROM merchants
UNION ALL SELECT 'أصنافها', count(*)::text FROM menu_items
UNION ALL SELECT 'أصناف في أقسام المنصة',
       count(*)::text FROM menu_items WHERE platform_section_id IS NOT NULL
UNION ALL SELECT 'أقسام المنصة الفعّالة', count(*)::text FROM platform_sections WHERE active
UNION ALL SELECT 'السائقون في الدوام', string_agg(u.full_name, ' · ' ORDER BY u.shift_started_at)
       FROM users u WHERE u.on_shift
UNION ALL SELECT 'الطلبات', count(*)::text FROM orders
UNION ALL SELECT 'الحسابات', count(*)::text FROM users WHERE deleted_at IS NULL;
