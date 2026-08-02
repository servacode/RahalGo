package orders

// دوامُ المتجر — **ومنتصفُ الليل يقطعه.**
//
// # الخللُ الذي كان
//
// كان الفحصُ سطراً واحداً:
//
//	(now)::time BETWEEN h.open_time AND h.close_time
//
// **وهو صحيحٌ لمن يفتح ٠٩:٠٠ ويغلق ٢٣:٠٠، وكاذبٌ أبداً لمن يفتح ١٨:٠٠ ويغلق
// ٠٢:٠٠**: `BETWEEN 18:00 AND 02:00` مجالٌ مقلوبٌ لا يصدق على شيء —
// **فالمتجرُ يظهر مغلقاً طوالَ دوامه كلِّه.**
//
// وليست حالةً نادرة: **مطاعمُ الشاورما في الرقّة تعمل ليلاً**، وهي أكثرُ ما
// يُطلب. **ومن فتح دوامَه على هذا النحو رأى متجرَه مخفياً ولم يعرف لماذا** —
// فيظنّ العطلَ في المنصة، وهو في سطرٍ واحد.
//
// # والصوابُ نصفان لا نصف
//
//   - **ذيلُ أمس**: الساعةُ ٠١:٠٠ اليومَ تقع في دوام أمس الذي يمتدّ إلى ٠٢:٠٠
//   - **صدرُ اليوم**: الساعةُ ٢٠:٠٠ تقع في دوام اليوم الذي بدأ ١٨:٠٠
//
// **ومن نظر إلى صفّ اليوم وحدَه أغلق المتجرَ في أوّل ساعتين من ليله.**

// OpenNowSQL شرطُ «المتجر يستقبل الآن» — يُستعمل في العرض **وفي إنشاء الطلب**.
//
// **ونصٌّ واحدٌ لا نصّان.** كان يُكتب في `customer_handlers.go` للعرض وحدَه،
// **ولا فحصَ عند الإنشاء أصلاً**: من فتح الصفحةَ قبل الإغلاق بدقيقة، أو تركها
// مفتوحةً ساعةً، **يطلب من متجرٍ مغلق** — فيصل الطلبُ ولا أحدَ يحضّره.
//
// ولو نُسخ النصُّ لموضعين لَافترقا: يُصلَح أحدُهما ويبقى الآخر — **والعرضُ
// يقول مغلقٌ والإنشاءُ يقبل، أو العكس.**
//
// يفترض أن جدول المتاجر باسم `m`.
const OpenNowSQL = `(m.status = 'active' AND NOT m.emergency_closed AND (
	-- **لا صفوفَ دوامٍ تعني مفتوحاً دائماً** — من لم يحدّد لم يقيّد نفسه.
	NOT EXISTS (SELECT 1 FROM merchant_hours h WHERE h.merchant_id = m.id)
	-- صدرُ اليوم: دوامٌ عاديّ، أو الشقُّ المسائيُّ من دوامٍ يعبر منتصف الليل.
	OR EXISTS (
		SELECT 1 FROM merchant_hours h
		WHERE h.merchant_id = m.id AND NOT h.closed
		  AND h.day_of_week = EXTRACT(dow FROM (now() AT TIME ZONE 'Asia/Damascus'))::int
		  AND CASE
		      WHEN h.close_time > h.open_time
		        THEN (now() AT TIME ZONE 'Asia/Damascus')::time BETWEEN h.open_time AND h.close_time
		      ELSE (now() AT TIME ZONE 'Asia/Damascus')::time >= h.open_time
		      END)
	-- **ذيلُ أمس**: الساعةُ الواحدةُ ليلاً تقع في دوامٍ بدأ أمسِ السادسةَ مساءً.
	OR EXISTS (
		SELECT 1 FROM merchant_hours h
		WHERE h.merchant_id = m.id AND NOT h.closed
		  AND h.close_time <= h.open_time
		  AND h.day_of_week = EXTRACT(dow FROM (now() AT TIME ZONE 'Asia/Damascus') - interval '1 day')::int
		  AND (now() AT TIME ZONE 'Asia/Damascus')::time < h.close_time)))`

// NextOpenSQL موعدُ فتحِ المتجر القادم — و`NULL` إن كان مفتوحاً الآن أو بلا دوام.
//
// # لماذا موعدٌ لا كلمة
//
// **قل متى يعود لا أنه غيرُ متاح.** «متاح من ١٠ صباحاً» **موعدٌ يُعاد إليه**،
// و«غير متاح» **طريقٌ مسدود** — من قرأه أغلق الصفحة، ومن قرأ الأوّل عاد.
//
// **والفرق بين «مغلق» و«يفتح ١١:٠٠» هو الفرق بين زبونٍ ضاع وزبونٍ عاد.**
//
// # والبياناتُ موجودةٌ منذ البداية
//
// `merchant_hours` تحمل الجواب، **ولا تصعد إلى الردّ** — فالزائرُ يقرأ «مغلق
// الآن» ولا يعرف أن عليه العودة بعد ثلاث ساعات. (وهي `D-11` المؤجَّلة.)
//
// # ولماذا سبعةُ أيام
//
// متجرٌ يفتح الجمعةَ وحدَها **يجب أن يقول ذلك لا أن يسكت.** والبحثُ في الغد
// وحدَه يعيد فراغاً لمن يفتح بعد ثلاثة أيام، **وفراغٌ يُعرض «غير متاح» فنعود
// إلى ما هربنا منه.**
//
// يفترض أن جدول المتاجر باسم `m`.
const NextOpenSQL = `(
	SELECT (date_trunc('day', (now() AT TIME ZONE 'Asia/Damascus') + make_interval(days => d))
	        + h.open_time) AT TIME ZONE 'Asia/Damascus'
	FROM generate_series(0, 7) d
	JOIN merchant_hours h ON h.merchant_id = m.id
	 AND h.day_of_week = EXTRACT(dow FROM
	     (now() AT TIME ZONE 'Asia/Damascus') + make_interval(days => d))::int
	WHERE NOT h.closed
	  -- **وما مضى اليومَ لا يُعرض موعداً**: من فتح التاسعةَ والساعةُ الثالثة
	  -- لا يُقال له «يفتح التاسعة» — **موعدٌ في الماضي أسوأُ من لا موعد.**
	  AND (date_trunc('day', (now() AT TIME ZONE 'Asia/Damascus') + make_interval(days => d))
	       + h.open_time) > (now() AT TIME ZONE 'Asia/Damascus')
	ORDER BY 1 LIMIT 1)`
