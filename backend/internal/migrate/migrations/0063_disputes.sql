-- ٠٠٦٣ · النزاعاتُ — **مع أربعةٍ لا مع واحد**
--
-- # قرارُ المالك (٢٠٢٦-٠٨-٠٣)
--
-- **«قسمُ النزاعات يحوي تبويباً للمناديب والمتاجر والسائقين والزبائن لنعرف
-- منازعةَ المنصة مع من — لأنّه قسمٌ خاصٌّ بمنازعات المنصة والطرف الآخر.»**
--
-- # وكان مع المتجر وحدَه
--
-- المطالبةُ كانت تُكتب في `merchant_warnings.claim_amount` — **على صفّ الإنذار
-- نفسِه**، بحجّةٍ مكتوبةٍ في `warnings.go`: «وجدولان لواقعةٍ واحدة يفترقان».
--
-- **والحجّةُ صحيحةٌ لواقعةٍ واحدة — وهاتان واقعتان:**
--
--   الإنذارُ    ←  سلوكٌ وقع · يُعدّ في عدّاد المخالفات · **ولا يُسوّى أبداً**
--   النزاعُ     ←  مالٌ مستحقّ · يُسوّى أو يُسقَط · **وله عمرٌ ينتهي**
--
-- **ولذلك لم يكن للنزاع مع سائقٍ أو زبونٍ أو مندوبٍ مكانٌ إطلاقاً** — لأنّ
-- هؤلاء لا إنذاراتِ لهم، والمالُ كان يسكن في صفّ الإنذار.
--
-- وهي نزاعاتٌ تقع فعلاً: **سائقٌ لم يسلّم صندوقَه · ومندوبٌ أخذ عمولةً على
-- متجرٍ لم يعمل · وزبونٌ استلم ولم يدفع.** ولا مكانَ لأيٍّ منها اليوم، **فتُدار
-- بالهاتف وتُنسى.**
--
-- # والرابطُ يمنع الافتراق
--
-- `warning_id` يعيد النزاعَ إلى إنذاره حين يكون له إنذار. **فمن قرأ الإنذارَ
-- وجد كلفتَه، ومن قرأ النزاعَ وجد سببَه** — ولا نسخةَ ثانيةً من الحقيقة.
--
-- # وما يبقى في مكانه
--
-- `merchant_warnings.claim_amount` **لا يُحذف** — تاريخٌ وقع، والدفاترُ لا
-- تُمحى. **ويتوقّف عن الكتابة** فقط: كلُّ نزاعٍ جديدٍ يُولد هنا.

CREATE TABLE IF NOT EXISTS disputes (
	id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),

	-- **الطرفُ الآخر** — دورُه لا حسابُه.
	--
	-- السؤالُ الذي يُفتح به القسم: **«مع من نتنازع؟»** — والدورُ يجيبه في
	-- عمودٍ واحدٍ يُرشَّح به التبويب، **ولا يُستنتج من جدول الأدوار**: من كان
	-- سائقاً يومَ النزاع قد يصير غداً موظّفاً، **والماضي لا يتغيّر.**
	party_role   text NOT NULL CHECK (party_role IN ('merchant', 'driver', 'sales', 'customer')),

	-- **ومن هو بعينه.**
	--
	-- `party_user_id` للأشخاص الثلاثة، **و`merchant_id` للمتجر** — لأنّ المتجر
	-- كيانٌ لا شخص، وصاحبُه قد يتغيّر والنزاعُ على المتجر لا عليه.
	party_user_id uuid REFERENCES users(id) ON DELETE SET NULL,
	merchant_id   uuid REFERENCES merchants(id) ON DELETE CASCADE,
	CONSTRAINT disputes_party_present CHECK (
		(party_role = 'merchant' AND merchant_id IS NOT NULL)
		OR (party_role <> 'merchant' AND party_user_id IS NOT NULL)
	),

	-- **الواقعةُ التي وُلد منها** — طلبٌ في الغالب، وقد لا يكون.
	order_id     uuid REFERENCES orders(id) ON DELETE SET NULL,
	-- **وإنذارُه إن كان له إنذار** — الرابطُ الذي يمنع نسختين من الحقيقة.
	warning_id   uuid REFERENCES merchant_warnings(id) ON DELETE SET NULL,

	reason       text   NOT NULL,
	note         text   NOT NULL DEFAULT '',
	-- **المبلغُ المطالَبُ به** — موجبٌ دائماً، والاتجاهُ يعرفه الدور.
	amount       bigint NOT NULL CHECK (amount > 0),

	-- open · settled (خُصم أو دُفع) · waived (أُسقط)
	--
	-- **و«أُسقط» ليست «سُوّي»**: الأولى قرارٌ بأن نتحمّل، والثانيةُ مالٌ تحرّك.
	-- **وخلطُهما يجعل تقريرَ الخسائر يقرأ إسقاطاً تحصيلاً.**
	status       text NOT NULL DEFAULT 'open'
	                  CHECK (status IN ('open', 'settled', 'waived')),
	settlement   text,
	settled_at   timestamptz,
	settled_by   uuid REFERENCES users(id) ON DELETE SET NULL,

	created_by   uuid REFERENCES users(id) ON DELETE SET NULL,
	created_at   timestamptz NOT NULL DEFAULT now()
);

-- **والمفتوحُ يُقرأ أوّلاً** — القسمُ يُفتح على ما لم يُحسم.
CREATE INDEX IF NOT EXISTS disputes_open_idx
	ON disputes (party_role, created_at DESC) WHERE status = 'open';

-- **ولا نزاعان على طلبٍ واحدٍ مع طرفٍ واحد.**
--
-- التعويضُ قد يُنادى مرّتين لطلبٍ واحد (طارئٌ ثمّ فشل)، **فيُطالَب المتجرُ
-- مرّتين بالواقعة نفسِها.** والحارسُ في القاعدة لا في الشيفرة: **شرطٌ مكتوبٌ
-- في مكانٍ واحدٍ لا يفترق.**
CREATE UNIQUE INDEX IF NOT EXISTS disputes_order_party_idx
	ON disputes (order_id, party_role) WHERE order_id IS NOT NULL;

-- **وما وقع قبل اليوم يُنقل** — نزاعاتُ المتاجر المفتوحةُ تجد بيتَها الجديد،
-- **ولا يُمحى من صفّ الإنذار شيء.**
INSERT INTO disputes (party_role, merchant_id, order_id, warning_id, reason, note, amount,
                      status, settlement, settled_at, settled_by, created_by, created_at)
SELECT 'merchant', w.merchant_id, w.order_id, w.id, w.reason, w.note, w.claim_amount,
       CASE WHEN w.settlement IS NULL THEN 'open' ELSE 'settled' END,
       w.settlement, w.settled_at, w.settled_by, w.issued_by, w.created_at
FROM merchant_warnings w
WHERE w.claim_amount > 0
ON CONFLICT DO NOTHING;
