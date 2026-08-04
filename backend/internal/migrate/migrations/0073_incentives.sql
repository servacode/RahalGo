-- المكافآتُ والعقوبات — **مالٌ يخرج بتقديرٍ لا بمعادلة.**
--
-- # المسألة
--
-- المنصةُ تدفع بالمعادلة: أجرُ توصيلٍ وعمولةُ مندوبٍ ومستحقُّ متجر. **وكلُّها
-- تقع وحدَها بلا أن يقرّر أحد.**
--
-- **وما لا معادلةَ له لم يكن له باب**: سائقٌ عمل في ليلة عاصفةٍ حين اعتذر
-- الجميع، وآخرُ تركَ زبوناً واقفاً وأغلق هاتفَه. **لا يُكافأ الأوّلُ ولا
-- يُحاسَب الثاني** — أو يُكافأ بحوالةٍ خارج النظام لا أثرَ لها في أيّ دفتر.
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «مكافآت... قسمٌ خاصٌّ لإرسال المكافآت، والسائقُ
-- يشوف شو متلقّي، ومكافآت وعقوبات عند تحقيق الأهداف».)
--
-- # ولماذا جدولٌ والمالُ في المحفظة
--
-- **المالُ في المحفظة لأنّ المحفظةَ هي الدفتر** — قيدٌ خارجها مالٌ لا يُقرأ.
--
-- **والجدولُ لأنّ للحدث معنًى لا يحمله القيد**: من كافأ، ولماذا، وأهي مكافأةٌ
-- عن هدفٍ بلغه أم عن ليلةٍ بعينها. **وسطرٌ في المحفظة يقول «٥٠٬٠٠٠» ونصّاً
-- حرّاً لا يُعدّ ولا يُقاس** — فلا يُعرف كم كوفئ هذا الشهر ولا على ماذا.
--
-- # والعقوبةُ مالٌ يعود لا مالٌ يُمحى
--
-- تُخصم منه وتُقيَّد للخزينة — **ودفترٌ يأخذ من طرفٍ ولا يعطي آخرَ دفترٌ لا
-- يتوازن**، وتظهر المنصةُ خاسرةً وقد قبضت.

CREATE TABLE IF NOT EXISTS incentives (
	id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	user_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	-- kind مكافأةٌ أو عقوبة — **والمبلغُ موجبٌ في الحالين**: الإشارةُ في
	-- النوع لا في الرقم. **ورقمٌ سالبٌ في عمودٍ اسمُه «مبلغ» يُقرأ خطأً
	-- في أوّل تقرير.**
	kind       text NOT NULL CHECK (kind IN ('reward', 'penalty')),
	amount     bigint NOT NULL CHECK (amount > 0),
	-- reason **إلزاميّ** — مالٌ يخرج بتقدير إنسان، **وبلا كلمةٍ لا يُراجَع
	-- ولا يُقاس من يُكثره.**
	reason     text NOT NULL CHECK (btrim(reason) <> ''),
	-- forTarget أهي عن هدفٍ بلغه — **فتُعدّ في مكانها**، أم عن واقعةٍ بعينها.
	for_target boolean NOT NULL DEFAULT false,
	created_by uuid NOT NULL REFERENCES users(id),
	created_at timestamptz NOT NULL DEFAULT now()
);

-- **والسؤالُ الغالبُ «ماذا له هذا الشهر؟»** — فيُفهرَس بصاحبه ووقته.
CREATE INDEX IF NOT EXISTS incentives_user_idx
	ON incentives (user_id, created_at DESC);

-- **وأنواعُ الحركة تتّسع لهما.**
--
-- كان القيدُ يحصرها في أحدَ عشرَ نوعاً، **ومكافأةٌ تُقيَّد `adjustment` تختلط
-- بتسويةٍ محاسبية** — فلا يُعرف في كشف السائق أيُّ سطرٍ مكافأةٌ وأيُّه تصحيحُ
-- خطأ. **ونوعٌ يحمل معنيين لا يُقاس.**
ALTER TABLE wallet_transactions DROP CONSTRAINT IF EXISTS wallet_transactions_kind_check;
ALTER TABLE wallet_transactions ADD CONSTRAINT wallet_transactions_kind_check
	CHECK (kind IN ('topup', 'order_payment', 'refund', 'compensation', 'commission',
	                'merchant_earning', 'driver_earning', 'payout', 'adjustment',
	                'platform_profit', 'platform_expense', 'reward', 'penalty'));
