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
DELETE FROM order_ratings;
DELETE FROM order_events;
DELETE FROM order_items;
DELETE FROM promo_redemptions;
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

-- ٧ · وأرقامُ الطلبات تعود إلى أوّلها
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
UNION ALL SELECT 'الأصناف', count(*)::text FROM menu_items;
