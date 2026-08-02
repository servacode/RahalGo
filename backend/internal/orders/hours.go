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
