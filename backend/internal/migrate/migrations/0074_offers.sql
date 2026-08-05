-- العروضُ والخصومات — **ما تُنزله المنصةُ بنفسها.**
--
-- # وما الفرقُ عن أكواد الخصم؟
--
-- **الكودُ يُكتب والعرضُ يُرى.** كودُ خصمٍ يحتاج من يعرفه: يُنشر في مجموعةٍ
-- أو يُقال لمن يسأل، **ومن لم يسمع به لا يستفيد منه ولا يعلم أنّه فاته.**
--
-- والعرضُ في شاشته: **يفتح الأيقونةَ فيرى** — لا يكتب ولا يبحث.
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «نضيف عند الزبون أيقونة عروضات... المنصةُ ممكن
-- تنزل عرضاً أو خصومات».)
--
-- # نوعان في جدولٍ واحد
--
--	لافتة  ←  صورةٌ ونصٌّ ووجهة — **خبرٌ لا سعر**
--	خصم   ←  صنفٌ بعينه بنسبةٍ — **سعرٌ يتغيّر فعلاً**
--
-- **وجدولان يعنيان شاشتين وقائمتين ونقطتين** لشيءٍ يراه الزبونُ قسماً واحداً.
-- والحقولُ المشتركةُ أكثرُ من المفترقة: عنوانٌ ومدّةٌ وتفعيل.
--
-- # ومن يتحمّل ثمنَ الخصم — **حقلٌ لا قاعدة**
--
-- (قرارُ المالك: «أقرّر لكلّ عرضٍ على حدة».)
--
--	المنصة  ←  المتجرُ يقبض سعرَه كاملاً، والخصمُ يأكل من هامشنا
--	المتجر  ←  يُخصم من مستحقّه، وهو عرضٌ اتّفق عليه
--
-- **وقاعدةٌ واحدةٌ لكلّ العروض تُجبر على واحدٍ من أمرين**: إمّا لا نعرض ما
-- يتحمّله المتجر، أو نخصم من جيبه بلا إذنه.

CREATE TABLE IF NOT EXISTS offers (
	id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
	kind  text NOT NULL CHECK (kind IN ('banner', 'discount')),
	title text NOT NULL CHECK (btrim(title) <> ''),
	body  text NOT NULL DEFAULT '',

	-- ── اللافتة ──────────────────────────────────────────────────────────
	media_id uuid REFERENCES media(id) ON DELETE SET NULL,
	-- href وجهةُ الضغط — **وفارغُه لافتةٌ تُقرأ ولا تُفتح**، وهو مشروع.
	href text NOT NULL DEFAULT '',

	-- ── الخصم ────────────────────────────────────────────────────────────
	menu_item_id uuid REFERENCES menu_items(id) ON DELETE CASCADE,
	-- percent **نسبةٌ لا مبلغ**: صنفٌ بألفٍ وآخرُ بمئةِ ألفٍ لا يقبلان خصماً
	-- واحداً. **وسقفُها تسعون** — ما دونها بيعٌ بلا ثمن.
	discount_percent int CHECK (discount_percent BETWEEN 1 AND 90),
	-- borne_by من يتحمّله — **ولا يُقرأ إلّا في الخصم.**
	borne_by text CHECK (borne_by IN ('platform', 'merchant')),

	-- ── المدّة ───────────────────────────────────────────────────────────
	--
	-- **وفارغُهما «بلا حدّ»**: عرضٌ دائمٌ حالٌ مشروعة، **وإلزامُ تاريخٍ يجعل
	-- من ينشئه يكتب سنةَ ٢٠٩٩** فيصير الحقلُ زينةً لا حدّاً.
	starts_at timestamptz,
	ends_at   timestamptz,
	active    boolean NOT NULL DEFAULT true,

	created_by uuid NOT NULL REFERENCES users(id),
	created_at timestamptz NOT NULL DEFAULT now(),
	updated_at timestamptz NOT NULL DEFAULT now(),

	-- **والخصمُ يلزمه صنفٌ ونسبةٌ ومن يتحمّل** — ثلاثةٌ معاً أو لا خصم.
	-- **وصفٌّ نصفُ مكتوبٍ يُعرض في الشاشة ولا يُطبَّق في السعر**، فيغضب
	-- الزبونُ ولا يفهم أحدٌ لماذا.
	CONSTRAINT offers_discount_complete CHECK (
		kind <> 'discount' OR (
			menu_item_id IS NOT NULL AND
			discount_percent IS NOT NULL AND
			borne_by IS NOT NULL)
	)
);

-- **والسؤالُ الغالبُ «ما العروضُ السارية الآن؟»** — يُسأل في كلّ فتحةِ شاشة.
CREATE INDEX IF NOT EXISTS offers_live_idx
	ON offers (kind, ends_at)
	WHERE active;

-- **وخصمٌ واحدٌ سارٍ لكلّ صنف.**
--
-- **وخصمان على صنفٍ واحدٍ سؤالٌ بلا جواب**: أيُّهما يُطبَّق؟ الأكبرُ أم
-- الأحدث؟ **وأيُّ جوابٍ نختره يفاجئ من أنشأ الثاني** — فيُمنع في القاعدة.
CREATE UNIQUE INDEX IF NOT EXISTS offers_one_live_per_item
	ON offers (menu_item_id)
	WHERE active AND kind = 'discount';
