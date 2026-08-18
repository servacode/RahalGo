-- ══════════════════════════════════════════════════════════════════════
-- **المكافأةُ مرّةً واحدةً للرقم — لا للحساب**
-- ══════════════════════════════════════════════════════════════════════
--
-- (قرارُ المالك ٢٠٢٦-٠٨-١٨: «لا بالطبع مرّةً واحدةً الهديّة، ولو حذف
--  حسابَه وأعاد تسجيلَ جديدٍ لا يكسب شيئاً. وأيضاً الدعوة نفسُ الشيء —
--  مرّةً واحدةً للرقم».)
--
-- # الثغرةُ التي سدَّها هذا
--
-- **الحذفُ لا يمحو السطرَ بل يحرّر الرقم** (`phone = 'deleted-'||id`)
-- ليستطيع صاحبُه التسجيلَ من جديد — وهو مقصود.
--
-- **فالتسجيلُ بالرقم نفسِه يُنشئ حساباً بمعرّفٍ جديد.** وكانت الهديّةُ
-- تُفحص بمعرّف الحساب، **فتُصرف من جديدٍ في كلّ دورة.**
--
-- **ومكافأةُ الدعوة كذلك**: `referrals.invitee_id` مفتاحٌ أوّليّ، ومدعوٌّ
-- بمعرّفٍ جديدٍ صفٌّ جديد — **فيُكافأ الداعي مرّةً بعد مرّة.**
--
-- **واثنان متّفقان يدوران الحلقةَ فيطبعان مالاً من خزينة المنصّة**: يدعو
-- فيسجّل فيقبضان، ثمّ يحذف فيسجّل فيقبضان.
--
-- # ولماذا بصمةٌ لا رقم
--
-- **الحذفُ يمحو الرقمَ عمداً** — حقُّ صاحبه. **وحفظُه هنا يُبطل ذلك
-- الحقَّ من بابٍ خلفيّ.**
--
-- **والبصمةُ باتّجاهٍ واحد**: تُحسب من الرقم ولا يُستخرج منها رقم.
-- **وبملحٍ سرّيّ** — بلاه تُجرَّب أرقامُ سوريا كلُّها في دقائق، فتُعرف
-- أيُّ رقمٍ سجّل يوما.
--
-- # والملحُ في القاعدة لا في البيئة
--
-- **متغيّرُ بيئةٍ يُنسى عند نقل الخادم** — فتتبدّل البصماتُ كلُّها
-- **وتُصرف الهدايا من جديدٍ بلا أن يعلم أحد.** وهنا يُولَّد مرّةً
-- ويبقى.

CREATE TABLE IF NOT EXISTS app_secrets (
	key        text PRIMARY KEY,
	value      text NOT NULL,
	created_at timestamptz NOT NULL DEFAULT now()
);

-- **ولا يُولَّد إلّا مرّةً** — `ON CONFLICT` يحرسه من إعادة التشغيل.
INSERT INTO app_secrets (key, value)
VALUES ('phone_pepper', gen_random_uuid()::text || gen_random_uuid()::text)
ON CONFLICT (key) DO NOTHING;

CREATE TABLE IF NOT EXISTS phone_claims (
	phone_hash text NOT NULL,
	-- kind ما حُجز — **والهديّةُ غيرُ الدعوة**: قد يُغيَّر أحدُهما
	-- ويبقى الآخر.
	kind       text NOT NULL CHECK (kind IN ('signup_bonus', 'referral')),
	created_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (phone_hash, kind)
);

-- ══════════════════════════════════════════════════════════════════════
-- **ومن قبض قبل اليوم يُسجَّل — وإلّا قبض مرّةً ثانية**
-- ══════════════════════════════════════════════════════════════════════
--
-- **وأرقامُ المحذوفين لا تُسجَّل**: صارت `deleted-<id>` فبصمتُها بصمةُ
-- نصٍّ لا بصمةُ رقم. **ومن حذف حسابَه قبل هذا التعديل يقبض مرّةً
-- أخيرة** — ولا سبيلَ لمعرفة رقمه، وهو ما أراده حين حذف.

INSERT INTO phone_claims (phone_hash, kind)
SELECT DISTINCT
       encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper')
                      || u.phone)::bytea), 'hex'),
       'signup_bonus'
FROM wallet_transactions wt
JOIN users u ON u.id = wt.user_id
WHERE wt.kind = 'reward'
  AND wt.ref = u.id::text
  AND u.phone NOT LIKE 'deleted-%'
ON CONFLICT DO NOTHING;

INSERT INTO phone_claims (phone_hash, kind)
SELECT DISTINCT
       encode(sha256(((SELECT value FROM app_secrets WHERE key = 'phone_pepper')
                      || u.phone)::bytea), 'hex'),
       'referral'
FROM referrals r
JOIN users u ON u.id = r.invitee_id
WHERE u.phone NOT LIKE 'deleted-%'
ON CONFLICT DO NOTHING;
