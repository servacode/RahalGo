package qa

import (
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **نموُّ السوق وانحسارُها** (`ML`، ٢٠٢٦-٠٩-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// **بنودُ المصفوفة ٣ و٤ و١٢ و١٣ و١٥** من عقد دورة حياة السوق.
//
// **والسوقُ تنمو صنفاً صنفاً** — **فقسمٌ خاوٍ اليومَ عامرٌ غداً بلا
// تحديثِ تطبيق.** **وعكسُه كذلك**: **آخرُ صنفٍ يُسحب يُعيد القسمَ إلى
// خفائه** — **ولا يبقى عنواناً يُفتح على فراغ.**

// TestML9_ActiveMerchantWithoutCityIsNeverSilentlyServed **ومتجرٌ فعّالٌ
// بلا مدينةٍ لا يُخدَم صامتاً** — بندُ ٣.
//
// **وهذا هو العطبُ الذي كُشف على جهازٍ حقيقيّ بعينه**: **متجرٌ `active`
// وله موقعٌ ومدينتُه `NULL`** — **فيراه من لا موضعَ له ولا يراه من
// يقف فوقه.**
//
// **والفحصُ يُثبت الأذى نفسَه لا صورتَه**: **يُفرَغ العمودُ قسراً
// فيختفي المتجرُ عن جاره** — **ثمّ يُعاد فيعود.**
func TestML9_ActiveMerchantWithoutCityIsNeverSilentlyServed(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	const atMerchant = "?lat=35.9506&lng=39.0094"

	before, _ := itemsIn(t, hh, it.SectionID, atMerchant)
	if before == 0 {
		t.Fatal("**الجارُ لا يرى المتجرَ أصلاً** — المِسنَدُ لم يشتقّ المدينة")
	}

	// **تُمحى المدينةُ وحدَها** — والموقعُ والحالةُ كما هما.
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE merchants SET city_id = NULL WHERE id = $1::uuid`,
		it.MerchantID); err != nil {
		t.Fatalf("محوُ المدينة: %v", err)
	}

	// **والأذى بعينه**: **`NULL = x` لا تصحّ** — **فيسقط المتجرُ من
	// شرط المدينة، ويرى الزبونُ الواقفُ فوقَه قسماً فارغاً.**
	blind, _ := itemsIn(t, hh, it.SectionID, atMerchant)
	if blind != 0 {
		t.Errorf("**متجرٌ بلا مدينةٍ ظهر لزبونٍ مُعرَّفِ الموضع (%d)** — "+
			"**فتبدّل شرطُ الترشيح ولم يعد هذا الفحصُ يحرس العطبَ الذي وُلد له.**", blind)
	}

	// **والحالُ التي لا تُحتمَل**: **فعّالٌ وله موقعٌ ولا مدينةَ له.**
	var orphan int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM merchants
		 WHERE status = 'active' AND location IS NOT NULL AND city_id IS NULL
		   AND id = $1::uuid`, it.MerchantID).Scan(&orphan); err != nil {
		t.Fatalf("عدُّ اليتامى: %v", err)
	}
	if orphan == 0 {
		t.Fatal("**لم يُفرَّغ العمودُ أصلاً** — فالشاهدُ لا يشهد")
	}

	// **ويُعاد الحقُّ بالباب المعتمَد** — لا بيدٍ في القاعدة.
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE merchants SET city_id = (
		   SELECT c.id FROM cities c
		    WHERE c.active AND ST_DWithin(c.center, location, c.radius_m)
		    ORDER BY ST_Distance(c.center, location) LIMIT 1)
		 WHERE id = $1::uuid`, it.MerchantID); err != nil {
		t.Fatalf("إعادةُ المدينة: %v", err)
	}
	after, _ := itemsIn(t, hh, it.SectionID, atMerchant)
	if after != before {
		t.Errorf("**عادت المدينةُ ولم يعد المتجر**: %d ⇐ %d", before, after)
	}
}

// TestML10_MovingMerchantRecomputesCityAtRuntime **ومن نقل متجرَه نُقلت
// مدينتُه** — بندُ ٤، **تشغيلاً لا قراءةَ مصدر.**
//
// **و`TestGEO2` تقرأ الجملةَ** — **وهذه تُشغّلها**: **جملةٌ صحيحةُ
// النصّ قد تكون خاطئةَ الأثر.**
func TestML10_MovingMerchantRecomputesCityAtRuntime(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	cityOf := func() string {
		t.Helper()
		var name *string
		if err := hh.Pool.QueryRow(ctxBG(), `
			SELECT c.name FROM merchants m
			  LEFT JOIN cities c ON c.id = m.city_id
			 WHERE m.id = $1::uuid`, it.MerchantID).Scan(&name); err != nil {
			t.Fatalf("قراءةُ المدينة: %v", err)
		}
		if name == nil {
			return ""
		}
		return *name
	}

	start := cityOf()
	if start == "" {
		t.Fatal("**المتجرُ وُلد بلا مدينة**")
	}

	// **يُنقَل إلى دمشق** — بالتعبير المركزيّ نفسِه الذي يستعمله التحديث.
	if _, err := hh.Pool.Exec(ctxBG(), `
		UPDATE merchants SET
		  location = ST_SetSRID(ST_MakePoint($2::float8, $3::float8), 4326)::geography,
		  city_id = (SELECT c.id FROM cities c
		              WHERE c.active
		                AND ST_DWithin(c.center,
		                      ST_SetSRID(ST_MakePoint($2::float8, $3::float8), 4326)::geography,
		                      c.radius_m)
		              ORDER BY ST_Distance(c.center,
		                      ST_SetSRID(ST_MakePoint($2::float8, $3::float8), 4326)::geography)
		              LIMIT 1)
		 WHERE id = $1::uuid`,
		it.MerchantID, 36.2765, 33.5138); err != nil {
		t.Fatalf("نقلُ المتجر: %v", err)
	}

	moved := cityOf()
	if moved == start {
		t.Errorf("**نُقل المتجرُ وبقيت مدينتُه %q** — "+
			"**يراه من لا يصله ولا يراه من يجاوره.**", start)
	}

	// **ولا يراه جارُه الأوّل بعد النقل.**
	if n, _ := itemsIn(t, hh, it.SectionID, "?lat=35.9506&lng=39.0094"); n != 0 {
		t.Errorf("**متجرٌ انتقل ما زال يظهر في مدينته الأولى**: %d صنفاً", n)
	}
}

// TestML11_FirstQualifyingProductRevealsSection **وأوّلُ صنفٍ يُظهر
// قسماً كان خاوياً** — بندُ ١٢، **بلا تحديثِ تطبيق.**
func TestML11_FirstQualifyingProductRevealsSection(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	// **قسمٌ جديدٌ لا صنفَ فيه.**
	var empty string
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO platform_sections (name, active)
		VALUES ($1, true) RETURNING id::text`, uniq("قسم نامٍ ")).Scan(&empty); err != nil {
		t.Fatalf("إنشاءُ قسم: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM platform_sections WHERE id = $1::uuid`, empty)
	})

	count, _, ok := sectionCounts(t, hh, empty, "")
	if !ok {
		t.Fatal("**القسمُ الجديدُ غائبٌ عن الباب**")
	}
	if count != 0 {
		t.Fatalf("**قسمٌ بلا صنفٍ عدُّه %d**", count)
	}

	// **يُضاف أوّلُ صنفٍ مستوفٍ** — معتمَدٌ ومتوفّرٌ لمتجرٍ فعّالٍ في مدينة.
	var itemID string
	if err := hh.Pool.QueryRow(ctxBG(), `
		INSERT INTO menu_items (merchant_id, platform_section_id, name,
		                        price, merchant_price, available, approved)
		VALUES ($1::uuid, $2::uuid, $3, $4, $4, true, true) RETURNING id::text`,
		it.MerchantID, empty, uniq("أوّلُ صنفٍ "), int64(5000)).Scan(&itemID); err != nil {
		t.Fatalf("إضافةُ صنف: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM menu_items WHERE id = $1::uuid`, itemID)
	})

	grown, _, _ := sectionCounts(t, hh, empty, "")
	if grown != 1 {
		t.Errorf("**أُضيف أوّلُ صنفٍ وعدُّ القسم %d** — "+
			"**والقسمُ لا يظهر لصاحبه إلّا بعدٍّ فوق الصفر.**", grown)
	}
	// **ويُفتح فيُرى فيه ما عُدّ.**
	if n, _ := itemsIn(t, hh, empty, ""); n != grown {
		t.Errorf("**العدُّ %d والمفتوحُ %d**", grown, n)
	}
	// **ويراه من يقف في مدينة المتجر.**
	if n, _ := itemsIn(t, hh, empty, "?lat=35.9506&lng=39.0094"); n == 0 {
		t.Error("**القسمُ الناميُ لا يُرى لزبونٍ في مدينة متجره**")
	}
}

// TestML12_WithdrawingLastProductHidesSection **وسحبُ آخرِ صنفٍ يُعيد
// القسمَ إلى خفائه** — بندُ ١٣.
//
// **والعنوانُ الذي يُفتح على فراغٍ أسوأُ من عنوانٍ غائب.**
func TestML12_WithdrawingLastProductHidesSection(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	count, _, ok := sectionCounts(t, hh, it.SectionID, "")
	if !ok || count == 0 {
		t.Fatalf("**القسمُ لم يُولد عامراً**: عدٌّ %d موجودٌ %v", count, ok)
	}

	// **ثلاثُ صورٍ للسحب** — **وكلُّها تُخرج الصنفَ من عدّ الوجود.**
	for _, step := range []struct {
		name string
		sql  string
	}{
		{"غيرُ متوفّر", `UPDATE menu_items SET available = false WHERE id = $1::uuid`},
		{"غيرُ معتمَد", `UPDATE menu_items SET available = true, approved = false WHERE id = $1::uuid`},
	} {
		if _, err := hh.Pool.Exec(ctxBG(), step.sql, it.ID); err != nil {
			t.Fatalf("%s: %v", step.name, err)
		}
		got, orderable, _ := sectionCounts(t, hh, it.SectionID, "")
		switch step.name {
		case "غيرُ معتمَد":
			if got != 0 {
				t.Errorf("**آخرُ صنفٍ %s وعدُّ القسم %d** — "+
					"**فيُفتح القسمُ على فراغ.**", step.name, got)
			}
		default:
			// **وغيرُ المتوفّر يبقى في الوجود ويخرج من الطلب** —
			// **وهو قرارُ المالك عينُه**: **البنيةُ للوجود لا للدوام.**
			if orderable != 0 {
				t.Errorf("**صنفٌ غيرُ متوفّرٍ ما زال يُعَدّ مطلوباً**: %d", orderable)
			}
		}
		// **ويُعاد الحالُ قبل الخطوة التالية.**
		if _, err := hh.Pool.Exec(ctxBG(),
			`UPDATE menu_items SET available = true, approved = true WHERE id = $1::uuid`,
			it.ID); err != nil {
			t.Fatalf("إعادةُ الصنف: %v", err)
		}
	}

	// **ويُحذف حذفاً** — فيعود القسمُ إلى الصفر.
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM menu_items WHERE id = $1::uuid`, it.ID); err != nil {
		t.Fatalf("حذفُ الصنف: %v", err)
	}
	gone, _, _ := sectionCounts(t, hh, it.SectionID, "")
	if gone != 0 {
		t.Errorf("**حُذف آخرُ صنفٍ وعدُّ القسم %d**", gone)
	}
	if n, _ := itemsIn(t, hh, it.SectionID, ""); n != 0 {
		t.Errorf("**القسمُ يفتح %d صنفاً بعد حذفِ آخرِها**", n)
	}
}

// TestML13_ClosedPlatformIsNotAnEmptyMarket **ومنصّةٌ موقوفةٌ مؤقّتاً
// ليست سوقاً خاوية** — بندُ ١٥.
//
// **وأسماءُ الحالات الأربع مقيسةٌ في `AV01..AV11`** — **وهذا يقيس
// ما لم يُقَس**: **أنّ الإيقافَ لا يمحو بنيةَ السوق من تحت الزبون.**
func TestML13_ClosedPlatformIsNotAnEmptyMarket(t *testing.T) {
	hh := New(t)
	// **وتُمسَح الحالُ في المبتدأ** — كعادةِ فحوص المنصّة.
	ordersOpen(t, hh)
	it := hh.NewItem(10000)

	before, _, ok := sectionCounts(t, hh, it.SectionID, "")
	if !ok || before == 0 {
		t.Fatalf("**القسمُ لم يُولد عامراً**: %d", before)
	}

	// **يُوقَف العملُ مؤقّتاً** — حالُ `temporarily_unavailable`.
	closure(t, hh, true, "صيانةٌ مؤقّتة", nil)
	// **وصفُّ الإيقاف واحدٌ للمنصّة كلِّها** — **فمن تركه مُفعَّلاً
	// أسقط فحصَ جارِه.**
	t.Cleanup(func() { closure(t, hh, false, "", nil) })

	after, _, ok := sectionCounts(t, hh, it.SectionID, "")
	if !ok {
		t.Fatal("**اختفى القسمُ بإيقاف المنصّة** — " +
			"**فيُقرأ الإيقافُ المؤقّتُ سوقاً لم تُبنَ بعد.**")
	}
	if after != before {
		t.Errorf("**عدُّ الوجود تبدّل بإيقاف المنصّة**: %d ⇐ %d — "+
			"**والإيقافُ حالُ خدمةٍ لا حالُ محتوى.**", before, after)
	}
	if n, _ := itemsIn(t, hh, it.SectionID, ""); n == 0 {
		t.Error("**القسمُ فارغٌ لأنّ المنصّةَ موقوفة** — " +
			"**والزبونُ يتصفّح وإن لم يستطع الطلب.**")
	}
}

// TestML14_CoverageFailureLeavesBrowsingIntact **وسقوطُ التغطية لا يمحو
// السوق** — تتمّةُ بندَي ٨ و١٥.
//
// **ودلالاتُ `out_of_zone` و`coverage_unavailable` مقيسةٌ في
// `CFC1..CFC8` و`AV06/AV09`** — **وهذا يقيس أنّ التصفّح لم يتبدّل
// بها**: **فالتغطيةُ تمنع التوصيلَ لا العرض.**
//
// **ومن خلطهما أخبر من يسكن خارجَ الأشكال أنّ المنصّةَ بلا بضاعة.**
func TestML14_CoverageFailureLeavesBrowsingIntact(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	before, _, ok := sectionCounts(t, hh, it.SectionID, "?lat=35.9506&lng=39.0094")
	if !ok || before == 0 {
		t.Fatalf("**القسمُ لم يُولد عامراً**: %d", before)
	}

	// **تُطفأ الفعّالةُ كلُّها وتُرجَع** — بالمِعزَل المعتمَد لا بيدٍ عامّة.
	disableAllZones(t, hh)

	after, _, ok := sectionCounts(t, hh, it.SectionID, "?lat=35.9506&lng=39.0094")
	if !ok || after != before {
		t.Errorf("**سقوطُ التغطية غيّر بنيةَ السوق**: %d ⇐ %d موجودٌ %v — "+
			"**والتغطيةُ تمنع التوصيلَ لا التصفّح.**", before, after, ok)
	}
	if n, _ := itemsIn(t, hh, it.SectionID, "?lat=35.9506&lng=39.0094"); n == 0 {
		t.Error("**لا أصنافَ تُعرض حين تسقط التغطية** — " +
			"**فيُقرأ عطبُ إعدادٍ سوقاً خاوية.**")
	}
}
