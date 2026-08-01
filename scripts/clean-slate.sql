-- **بدايةٌ نظيفة** — تُمسح بياناتُ الحسابات وتبقى الحسابات.
--
-- تختلف عن `wipe.sql`: تلك تمسح الحساباتِ نفسها فتلزم زراعةٌ جديدة، **وهذه
-- تُبقي المتجرَ وقائمتَه والمندوبَ والسائقين والزبون** وتمسح ما جرى بينهم.
--
-- ## ما يُمسح
--
-- كلُّ ما هو **حدثٌ وقع**: طلباتٌ وقيودٌ مالية وصناديقُ نقدٍ وإشعاراتٌ
-- وتقييماتٌ وتذاكرُ وعناوين.
--
-- ## وما يبقى
--
-- الحساباتُ وأدوارُها، **وجلساتُها**: حذفُ `refresh_tokens` يُخرج المالكَ من
-- خمس تبويباتٍ مفتوحة قبل تجربةٍ يريد أن يبدأها الآن — **مسحٌ يُتعب من طلبه
-- مسحٌ زائد.** ويبقى المتجرُ وقائمتُه والتصنيفاتُ والمناطقُ والإعدادات.
--
-- ## وثلاثةُ تفاصيلَ لو أُهملت لبقيت التجربةُ متّسخة
--
--   1. **`wallets.balance` عمودٌ مصان لا محسوب** — يُحدَّث مع كل قيد. فحذفُ
--      القيود وحده يترك **رصيداً يتيماً بلا قيدٍ يفسّره**، وهو أسوأ من رصيدٍ
--      خاطئ: يبدو صحيحاً ولا مصدرَ له.
--   2. **`driver_cash_boxes.held` كذلك** — صندوقٌ يقول إن على السائق ديناً
--      عن طلبٍ لم يعد موجوداً.
--   3. **عدّادُ أرقام الطلبات** يُعاد إلى ١٠٠١ — فتُقرأ التجربةُ من أوّلها.
--
-- ## ثم يُعاد رصيدُ الزبون
--
-- **بقيدٍ واحدٍ نظيف** لا برقمٍ يُكتب في العمود: المحفظةُ دفترٌ، **ورصيدٌ بلا
-- قيدٍ يُفسده من أوّل سطر.**

BEGIN;

-- ١ · الأحداث
DELETE FROM order_ratings;
DELETE FROM order_events;
DELETE FROM order_items;
DELETE FROM promo_redemptions;
DELETE FROM orders;

-- ٢ · المال — القيودُ ثم الأرصدة المصانة
DELETE FROM wallet_transactions;
UPDATE wallets SET balance = 0, updated_at = now();

DELETE FROM driver_cash_entries;
UPDATE driver_cash_boxes SET held = 0, updated_at = now();

DELETE FROM payout_requests;

-- ٣ · التواصل والتفضيلات
DELETE FROM ticket_replies;
DELETE FROM tickets;
DELETE FROM notifications;
DELETE FROM user_favorites;
DELETE FROM user_addresses;
DELETE FROM otp_codes;

-- ٤ · العدّادات — تُقرأ التجربةُ من أوّلها
ALTER SEQUENCE orders_number_seq RESTART WITH 1001;
ALTER SEQUENCE tickets_number_seq RESTART WITH 1;

-- ٥ · المتجر: مفتوحٌ دائماً بقرار المالك (تجربة)
--
-- **بحذف ساعاته لا بمدّها إلى ٢٣:٥٩**: قاعدةُ العرض تقول «بلا ساعاتٍ مسجّلة
-- فهو مفتوح» — **فالحذفُ يقول ذلك صراحةً، والمدُّ يقوله بحيلةٍ تُنسى.** ومن
-- أراد ساعاتٍ لاحقاً أضافها.
DELETE FROM merchant_hours;
UPDATE merchants SET status = 'active', emergency_closed = false,
                     violations_cleared_at = NULL;

-- ٦ · السائقون منصرفون — الدوامُ يُرفع باليد في التطبيق، فتُختبر الخطوة
UPDATE users SET on_shift = false, shift_started_at = NULL;

-- ٧ · رصيدُ الزبون — قيدٌ واحدٌ نظيف
INSERT INTO wallet_transactions (user_id, amount, kind, note, created_by)
SELECT u.id, 250000, 'topup', 'رصيد تجريبي — بداية نظيفة',
       (SELECT id FROM users WHERE phone = '+963999000001')
FROM users u WHERE u.phone = '+963935667788';

UPDATE wallets w SET balance = 250000, updated_at = now()
FROM users u WHERE u.id = w.user_id AND u.phone = '+963935667788';

COMMIT;
