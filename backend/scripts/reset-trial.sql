-- **تصفيرُ التجربة** — يمحو ما وقع، ويُبقي من يعمل.
--
-- # لماذا مكتوبٌ لا مكتوبٌ مرّة
--
-- التجربةُ تُعاد، **والتصفيرُ الذي يُكتب في كلّ مرّةٍ من الرأس يفترق في كلّ
-- مرّة**: يُنسى جدولٌ فتبقى بقيّةٌ تلوّث القراءة، أو يُزاد جدولٌ فيضيع ما لا
-- يُزرع ثانيةً. **فيُكتب مرّةً ويُراجَع.**
--
-- # ما يُمحى وما يبقى
--
--	يُمحى  ←  الطلباتُ وما تعلّق بها · قيودُ الدفتر التي لها مرجعُ طلب
--	          صندوقُ السائق · الشكاوى · الاستغاثات · صورُ التسليم
--	يبقى   ←  الحساباتُ والمتاجرُ والقوائمُ والمناطقُ والإعداداتُ والأكواد
--	          **وشحنُ المحفظة المزروع** — فهو رأسُ مالِ الزبون في التجربة
--
-- # والمرجعُ هو الفاصل
--
-- **كلُّ قيدٍ يخصّ طلباً يحمل `ref` بمعرّفه، والمزروعُ يحمل فراغاً.** فالشرطُ
-- `ref <> ''` يفصل بينهما بلا قائمةِ أنواعٍ تُنسى حين يُزاد نوع.
--
-- # والأرصدةُ تُحسب من الدفتر لا تُصفَّر
--
-- **صفرٌ يُكتب باليد كذبةٌ صغيرة**: يقول إنّ المحفظة فارغة وفي الدفتر شحنٌ
-- باقٍ. **فتُجمع من القيود الباقية** — فإن اختلّ الجمعُ ظهر الخللُ هنا لا في
-- كشفِ حسابٍ يقرؤه المالك بعد شهر.

BEGIN;

-- ١ · الطلباتُ وما تعلّق بها
--
-- **والتذاكرُ منها**: شكوى تشير إلى طلبٍ مُحيَ لا تُقرأ، **وقيدُ المفتاح
-- الأجنبيّ يمنع المحوَ أصلاً** — فسقط التصفيرُ في أوّل جولةٍ فُتحت فيها شكوى
-- (٢٠٢٦-٠٨-٠٣). **ونصُّ تنظيفٍ يسقط عند أوّل حالةٍ حقيقية ليس نصَّ تنظيف.**
DELETE FROM ticket_replies WHERE ticket_id IN (SELECT id FROM tickets);
DELETE FROM tickets;
DELETE FROM order_ratings;
DELETE FROM order_events;
DELETE FROM order_items;
DELETE FROM promo_redemptions;
-- **والنزاعاتُ قبل الإنذارات وقبل الطلبات** (هجرة ٠٠٦٣).
--
-- تشير إلى الاثنين بمفتاحين أجنبيّين، **ومحوُ الطلب قبلها يُسقط النصَّ** —
-- وهي العلّةُ نفسُها التي أسقطته عند أوّل شكوى (٢٠٢٦-٠٨-٠٣). **وجدولٌ يُضاف
-- إلى القاعدة ولا يُضاف إلى نصّ التنظيف قنبلةٌ موقوتة**: لا يظهر أثرُه حتى
-- تقع أوّلُ حالةٍ حقيقية.
DELETE FROM disputes;
DELETE FROM merchant_warnings;
DELETE FROM driver_emergencies;
DELETE FROM orders;

-- ٢ · قيودُ الدفتر التي لها مرجعُ طلب — **والشحنُ المزروع يبقى**
DELETE FROM wallet_transactions WHERE ref <> '';
DELETE FROM driver_cash_entries WHERE ref <> '';

-- ٣ · والأرصدةُ تُجمع من الباقي
UPDATE wallets w SET balance = COALESCE(
    (SELECT sum(amount) FROM wallet_transactions t WHERE t.user_id = w.user_id), 0);
UPDATE driver_cash_boxes b SET held = COALESCE(
    (SELECT sum(amount) FROM driver_cash_entries e WHERE e.driver_id = b.driver_id), 0);

-- ٤ · صورُ التسليم — **والملفّاتُ على القرص تبقى يتيمة**
--
-- لا يحذفها هذا النصّ: قرصُ التطوير يحتمل، **وحذفُ ملفّاتٍ من نصِّ SQL لا
-- يُتراجع عنه إن أُجهضت المعاملة.** تُكنس باليد إن ثقلت.
DELETE FROM media WHERE kind = 'delivery_proof';

-- ٥ · وسجلُّ التدقيق لما مُحي
DELETE FROM audit_log WHERE entity = 'order';

-- ٦ · والإشعاراتُ — **كلُّها**، فهي عن أحداثٍ لم تعد موجودة
DELETE FROM notifications;

-- ٧ · ودورُ السائقين يعود إلى أوّله
--
-- **`last_assigned_at` أثرُ الجولة الماضية**: من أخذ آخرَ طلبٍ فيها يبقى في
-- آخر الصفّ في الجولة الجديدة — **فتبدأ التجربةُ بترتيبٍ ورثته ولا تراه**،
-- ويُقرأ العرضُ الأوّل عشوائياً وهو ليس كذلك.
--
-- **وميدانٌ نظيفٌ يبدأ من دورٍ نظيف.**
UPDATE users SET last_assigned_at = NULL
WHERE EXISTS (SELECT 1 FROM user_roles r WHERE r.user_id = users.id AND r.role_code = 'driver');

-- ٨ · والقائمةُ تعود منشورةً كاملة
--
-- مراجعةُ القائمة (هجرة ٠٠٦٤) تُنزل عَلَمَ النشر عن صنفٍ حتى يُقَرّ. **وميدانٌ
-- يبدأ بأصنافٍ معلّقةٍ سوقٌ ناقص**: يتصفّح المالكُ فلا يجد ما وضعه بيده، **ثمّ
-- يظنّ أنّ العرضَ عطب.**
--
-- **وسببُ الردّ يُمحى معه** — كلمةٌ عن قرارٍ لم يعد له وجود.
UPDATE menu_items SET approved = true, review_note = '' WHERE NOT approved OR review_note <> '';

-- ٩ · وأرقامُ الطلبات تعود إلى أوّلها
--
-- **رقمٌ إنسانيّ لا مفتاح**: يُقال في الهاتف ويُكتب في الوثيقة. **وتجربةٌ
-- تبدأ من ١٠٠٧ تُقرأ استئنافاً لما قبلها** — والذي قبلها مُحي.
ALTER TABLE orders ALTER COLUMN number RESTART WITH 1001;

COMMIT;

-- والقراءةُ بعده — **يُقرأ ولا يُفترض**
SELECT 'الطلبات' AS ما, count(*)::text AS كم FROM orders
UNION ALL SELECT 'قيود الدفتر', count(*)::text FROM wallet_transactions
UNION ALL SELECT 'قيود الصندوق', count(*)::text FROM driver_cash_entries
UNION ALL SELECT 'مجموع المحافظ', COALESCE(sum(balance), 0)::text FROM wallets
UNION ALL SELECT 'نقدٌ بيد السائقين', COALESCE(sum(held), 0)::text FROM driver_cash_boxes
UNION ALL SELECT 'الحسابات', count(*)::text FROM users
UNION ALL SELECT 'المتاجر', count(*)::text FROM merchants
UNION ALL SELECT 'الأصناف', count(*)::text FROM menu_items
-- **وما يجب أن يكون صفراً يُقرأ صريحاً** — لا يُستنتج من غيابه.
UNION ALL SELECT 'أصنافٌ معلّقة', count(*)::text FROM menu_items WHERE NOT approved
UNION ALL SELECT 'النزاعات', count(*)::text FROM disputes
UNION ALL SELECT 'الشكاوى', count(*)::text FROM tickets
UNION ALL SELECT 'الإشعارات', count(*)::text FROM notifications;
