package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **دورةُ حياة السوق — ووجودُ المحتوى غيرُ إمكانِ الطلب** (`ML`، ٢٠٢٦-٠٩-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// # قرارُ المالك (٢٠٢٦-٠٩-١٦)
//
// **وإغلاقُ المتجر بالساعة لا يمحو بنيةَ السوق.** **والقسمُ يظهر إن
// كان له محتوىً يخصّ مدينةَ الزبون** — **وحالُ الطلب تُقال فوقَه لا
// تُخفيه.**
//
// # العطبُ الذي سبقه
//
// **وكان عدُّ القسم يشترط `OpenNowSQL`** — **فيصير صفراً بعد إغلاق
// المتاجر.** **وقِيس على التجهيز**: **الساعةُ ٢٢:٣٦ فالعدُّ خمسة،
// ولو كانت ٠٢:٠٠ لصار صفراً في الأقسام التسعة كلِّها.**
//
// **ولو بُنيت البنيةُ عليه لَاختفى السوقُ كلَّ ليلة** — **ورأى الزبونُ
// «لا متاجرَ بعد» والمتاجرُ موجودةٌ نائمة.**

// itemsIn **أصنافُ قسمٍ كما يراها الزبون** — بموضعٍ أو بلا.
func itemsIn(t *testing.T, hh *Harness, sectionID, query string) (int, int) {
	t.Helper()
	r := hh.GET("/api/v1/public/sections/"+sectionID+"/items"+query, "")
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ أصناف القسم ردّ %d %s**", r.Code, r.Err())
	}
	items, _ := r.JSON()["items"].([]any)
	return len(items), r.Code
}

// sectionCounts **عدّا القسم من باب الأقسام**: وجودٌ وإمكانُ طلب.
func sectionCounts(t *testing.T, hh *Harness, sectionID, query string) (structural, orderable int, found bool) {
	t.Helper()
	r := hh.GET("/api/v1/public/sections"+query, "")
	if r.Code != http.StatusOK {
		t.Fatalf("**بابُ الأقسام ردّ %d %s**", r.Code, r.Err())
	}
	list, _ := r.JSON()["sections"].([]any)
	for _, raw := range list {
		m, _ := raw.(map[string]any)
		if id, _ := m["id"].(string); id != sectionID {
			continue
		}
		c, _ := m["count"].(float64)
		o, _ := m["orderable_now"].(float64)
		return int(c), int(o), true
	}
	return 0, 0, false
}

// TestML1_LocatedCustomerSeesCityMerchant **زبونٌ في مدينة المتجر يرى
// أصنافَه** — **وهو العطبُ الذي كُشف على جهازٍ حقيقيّ.**
func TestML1_LocatedCustomerSeesCityMerchant(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	// **بلا موضع** — لا ترشيح.
	n, _ := itemsIn(t, hh, it.SectionID, "")
	if n == 0 {
		t.Fatal("**القسمُ فارغٌ بلا موضعٍ أصلاً** — المِسنَدُ لم يزرع صنفاً")
	}

	// **وبموضعٍ فوق المتجر تماماً** — **ووقف الجهازُ داخلَ الرقّة فلم
	// يرَ شيئاً، وهذا ما يحرسه هذا الفحص.**
	got, _ := itemsIn(t, hh, it.SectionID, "?lat=35.9506&lng=39.0094")
	if got == 0 {
		t.Error("**زبونٌ فوق المتجر لا يرى أصنافَه** — " +
			"**ومدينةُ المتجر غيرُ مشتقّةٍ، و`NULL = x` لا تصحّ أبداً.**")
	}
}

// TestML2_CustomerNearbyStillSees **وجارُ المتجر يراه** — على بعد كيلومتر.
func TestML2_CustomerNearbyStillSees(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	// **نقطةُ الجهاز الحقيقيّ في القياس الميدانيّ.**
	got, _ := itemsIn(t, hh, it.SectionID, "?lat=35.9607&lng=39.0139")
	if got == 0 {
		t.Error("**زبونٌ على بعد كيلومترٍ لا يرى المتجر**")
	}
}

// TestML3_OtherCityDoesNotLeak **ولا يتسرّب متجرُ مدينةٍ إلى أخرى.**
func TestML3_OtherCityDoesNotLeak(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	// **دمشق** — بعيدةٌ عن الرقّة بأكثر من نصف قطرها.
	got, _ := itemsIn(t, hh, it.SectionID, "?lat=33.5138&lng=36.2765")
	if got != 0 {
		t.Errorf("**متجرُ الرقّة ظهر لزبونٍ في دمشق**: %d صنفاً", got)
	}
}

// TestML4_MalformedGeographyFailsSafe **وإحداثيٌّ مشوَّهٌ لا يُسقط تصفّحاً.**
//
// **ورفضُ التصفّح على قراءةِ موقعٍ سيّئةٍ يُقرأ عطباً** — **فيُهمَل
// الموضعُ ويُعرض السوق.**
func TestML4_MalformedGeographyFailsSafe(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	for _, q := range []string{
		"?lat=999&lng=999",
		"?lat=abc&lng=def",
		"?lat=&lng=",
		"?lat=0&lng=0",
	} {
		n, code := itemsIn(t, hh, it.SectionID, q)
		if code != http.StatusOK {
			t.Errorf("**%s ردّ %d** — **والتصفّحُ لا يُمنَع بموقعٍ سيّئ**", q, code)
		}
		if n == 0 {
			t.Errorf("**%s أخفى السوقَ** — **والمُهمَلُ يعني «لا ترشيح» لا «لا شيء»**", q)
		}
	}
}

// TestML5_MissingSectionIs404NotEmpty **ولا وجودَ له ≠ فارغ.**
func TestML5_MissingSectionIs404NotEmpty(t *testing.T) {
	hh := New(t)
	r := hh.GET("/api/v1/public/sections/00000000-0000-4000-8000-000000000000/items", "")
	if r.Code != http.StatusNotFound {
		t.Errorf("**قسمٌ لا وجودَ له ردّ %d** — **فيُقرأ عطبُ رابطٍ نفادَ بضاعة**", r.Code)
	}
}

// TestML6_StructuralCountIgnoresOpeningHours **والبنيةُ لا تسقط بالساعة.**
//
// **وهذا لبُّ قرار المالك**: **متجرٌ نائمٌ يبقى قسمُه قائماً.**
func TestML6_StructuralCountIgnoresOpeningHours(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)

	before, _, ok := sectionCounts(t, hh, it.SectionID, "")
	if !ok {
		t.Fatal("**القسمُ غائبٌ عن باب الأقسام**")
	}
	if before == 0 {
		t.Fatal("**عدُّ الوجود صفرٌ والمتجرُ مفتوح** — المِسنَدُ لم يزرع صنفاً")
	}

	// **يُغلَق المتجرُ إغلاقاً طارئاً** — **وهي حالُ «مغلقٌ الآن».**
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE merchants SET emergency_closed = true WHERE id = $1::uuid`,
		it.MerchantID); err != nil {
		t.Fatalf("إغلاقُ المتجر: %v", err)
	}

	after, orderable, ok := sectionCounts(t, hh, it.SectionID, "")
	if !ok {
		t.Fatal("**اختفى القسمُ بإغلاق متجرٍ** — **وبنيةُ السوق لا تُمحى بالساعة**")
	}
	if after != before {
		t.Errorf("**عدُّ الوجود تبدّل بالإغلاق**: %d ⇐ %d — "+
			"**والإغلاقُ حالُ طلبٍ لا حالُ وجود.**", before, after)
	}
	if orderable != 0 {
		t.Errorf("**عدُّ «يُطلب الآن» %d ومتجرُه مغلق** — "+
			"**والحالُ تُقال صادقةً أو لا تُقال.**", orderable)
	}
}

// TestML7_ClosedMerchantStillListsItems **والقسمُ يُفتح ولو أُغلق متجرُه.**
//
// **وقسمٌ يُعَدّ ثمّ يُفتح فارغاً يُقرأ عطباً** — **والعكسُ كذلك.**
func TestML7_ClosedMerchantStillListsItems(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE merchants SET emergency_closed = true WHERE id = $1::uuid`,
		it.MerchantID); err != nil {
		t.Fatalf("إغلاقُ المتجر: %v", err)
	}
	n, _ := itemsIn(t, hh, it.SectionID, "")
	if n == 0 {
		t.Error("**القسمُ فارغٌ لأنّ متجرَه مغلقٌ الآن** — " +
			"**والزبونُ يتصفّح وإن لم يستطع الطلب.**")
	}
}

// TestML8_SectionCountMatchesWhatOpens **وما يُعَدّ هو ما يُفتح.**
func TestML8_SectionCountMatchesWhatOpens(t *testing.T) {
	hh := New(t)
	it := hh.NewItem(10000)
	for _, q := range []string{"", "?lat=35.9506&lng=39.0094"} {
		count, _, ok := sectionCounts(t, hh, it.SectionID, q)
		if !ok {
			t.Fatalf("**القسمُ غائبٌ عن باب الأقسام** (%q)", q)
		}
		items, _ := itemsIn(t, hh, it.SectionID, q)
		if count != items {
			t.Errorf("**العدُّ %d والمفتوحُ %d** (%q) — "+
				"**وقسمٌ يعد شيئاً ويفتح غيرَه يُقرأ عطباً.**", count, items, q)
		}
	}
}
