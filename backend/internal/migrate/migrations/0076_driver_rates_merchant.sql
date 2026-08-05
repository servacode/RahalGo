-- تقييمُ السائق للمتجر — **والطرفُ الذي لا يُقيَّم لا يُقاس.**
--
-- # المسألة
--
-- الزبونُ يقيّم المنصةَ والسائق. **والمتجرُ لا يقيّمه أحد** — ومن يراه عن قربٍ
-- كلَّ يومٍ هو السائق: **يقف عند بابه وينتظر، ويرى أنظيفٌ مطبخُه أم لا، وأيُردّ
-- عليه بأدبٍ أم يُصرَخ فيه.**
--
-- (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «طلباتي السابقة — أضف تقييم السائق للمتجر
-- وتعامله».)
--
-- # ونجمتان لا واحدة
--
-- **«المتجر» و«التعامل» شيئان يفترقان**: مطعمٌ يجهّز في خمس دقائق ويصرخ في
-- السائق، وآخرُ يُبقيه نصفَ ساعةٍ ويعتذر ويسقيه ماءً. **ونجمةٌ واحدةٌ تجمعهما
-- تُخفي أيَّهما المشكلة** — فلا يُعرف أنُكلّم المطبخَ أم صاحبَ المحلّ.
--
-- # ولا يقيّم إلّا من استلم منه فعلاً
--
-- **من لم يقف عند بابه لا رأيَ له فيه.** والشرطُ في القاعدة لا في الشاشة:
-- الطلبُ طلبُه، وقد بلغ `picked_up` أو تعذّر عند باب المتجر.

CREATE TABLE IF NOT EXISTS merchant_ratings (
	order_id    uuid PRIMARY KEY REFERENCES orders(id) ON DELETE CASCADE,
	driver_id   uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	merchant_id uuid NOT NULL REFERENCES merchants(id) ON DELETE CASCADE,
	-- speed_stars سرعةُ التجهيز، و conduct_stars التعامل.
	speed_stars   int NOT NULL CHECK (speed_stars BETWEEN 1 AND 5),
	conduct_stars int NOT NULL CHECK (conduct_stars BETWEEN 1 AND 5),
	comment       text NOT NULL DEFAULT '',
	created_at    timestamptz NOT NULL DEFAULT now()
);

-- **والمفتاحُ الطلبُ نفسُه** — فلا يُقيَّم مرّتين، **وتقييمان لواقعةٍ واحدة
-- يُرجّحان من أعاد الضغط.**

-- **والسؤالُ الغالبُ «كيف يُقيَّم هذا المتجر؟»** — فيُفهرَس به.
CREATE INDEX IF NOT EXISTS merchant_ratings_merchant_idx
	ON merchant_ratings (merchant_id, created_at DESC);
