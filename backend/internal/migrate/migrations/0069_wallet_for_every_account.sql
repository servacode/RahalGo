-- محفظةٌ لكلّ حساب — **تُخلق معه لا عند أوّل حركة.**
--
-- # المسألة
--
-- كانت تُنشأ كسولةً: `INSERT ... ON CONFLICT DO NOTHING` داخل `wallet.ApplyTx`.
-- **فمن لم يقبض ولم يُخصم منه لا محفظةَ له** — خمسةٌ من عشرة في هذه القاعدة،
-- **ومنهم حسابُ المالك نفسِه.**
--
-- وأثرُه ليس نظريّاً: **شاشةُ المحفظة تُفتح على لا شيء**، وكشفُ الحساب يقول
-- «لا محفظة» بدل «رصيدُك صفر» — **والفرقُ بينهما ثقة.** ومن يريد أن يشحن
-- حساباً لم يتحرّك يجد نفسَه ينشئ لا يودع.
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «أضف المحفظة لحساب الأدمن، وأيُّ حسابٍ يُنشأ يجب
-- أن يكون له محفظة».)
--
-- # ولماذا في القاعدة لا في الشيفرة
--
-- **سبعةُ مواضع تُنشئ مستخدماً**: واحدٌ في الإنتاج (`identity/repo.go`) وستّةٌ
-- في البذر والاختبار. **وقاعدةٌ تُكتب في سبعةٍ تُنسى في الثامن** — ويُنشأ
-- حسابٌ بلا محفظةٍ من مسارٍ أُضيف بعد اليوم.
--
-- **والزنادُ يقع على الصفّ لا على المنادي**: من كتب في `users` كُتبت محفظتُه،
-- **ولو كتب بـpsql بيده.**

-- ١ · ما مضى — كلُّ حسابٍ بلا محفظةٍ تُنشأ له
INSERT INTO wallets (user_id)
SELECT u.id FROM users u
WHERE NOT EXISTS (SELECT 1 FROM wallets w WHERE w.user_id = u.id);

-- ٢ · وما يأتي — بالزناد
--
-- **و`ON CONFLICT DO NOTHING` يبقى**: الزنادُ لا يُلغي الإنشاءَ الكسول في
-- `ApplyTx`، **وحارسان لشيءٍ واحدٍ أهونُ من حارسٍ يسقط.**
CREATE OR REPLACE FUNCTION wallet_for_new_user() RETURNS trigger AS $$
BEGIN
	INSERT INTO wallets (user_id) VALUES (NEW.id) ON CONFLICT (user_id) DO NOTHING;
	RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS users_wallet_trigger ON users;
CREATE TRIGGER users_wallet_trigger
	AFTER INSERT ON users
	FOR EACH ROW EXECUTE FUNCTION wallet_for_new_user();

COMMENT ON FUNCTION wallet_for_new_user() IS
	'محفظةٌ لكلّ حسابٍ عند إنشائه — القاعدةُ في القاعدة لا في سبعةِ مواضعَ تُنسى';
