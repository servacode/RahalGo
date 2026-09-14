package qa

// ══════════════════════════════════════════════════════════════════════
// **الجغرافيا الإداريّة — تصنيفُ موضعٍ لا حدُّ خدمة** (`AG`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والتوصيلُ يبقى لـ`delivery_zones` و`ZoneAt` وحدَها** — **وهذه تصف
// أين وقع العنوانُ لا أيُوصَّل إليه.**
//
// **وأهمُّ ما يُقاس أنّ الوصفَ لا يصير حكماً** (`AG-05`): **ومدينةٌ
// معروفةٌ فعّالةٌ لا تفتح باباً لعنوانٍ خارجَ الأشكال.**

import (
	"testing"
)

// placeOf **ما يقوله المحرّكُ عن موضع هذه الإحداثيّة.**
func placeOf(t *testing.T, h *Harness, tok string, it *Item, lat, lng float64) map[string]any {
	t.Helper()
	return avOf(t, h, tok, it, lat, lng)
}

// ═════════════════ AG-01 · AG-02 — المواضعُ تُسمّى ═════════════════

// TestAG01_RaqqaResolves **نقطةُ الرقّة تُنسَب إلى الرقّة.**
func TestAG01_RaqqaResolves(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-01", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)
	// **والرقّةُ مُطلَقةٌ ومغطّاةٌ — فلا سببَ يمنع.**
	av := placeOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if av["available"] != true {
		t.Fatalf("**نقطةُ الرقّة مُنعت**: %v", av)
	}
}

// TestAG02_DamascusIsNamed **ونقطةُ دمشقَ تُسمّى دمشق.**
//
// **وكانت `area_not_supported` بلا اسم** — **صحيحةً في حكمها، عاجزةً
// عن تسمية مكانها** — **والمطلوبُ «رحّال غو لم يصل إلى دمشق بعد».**
func TestAG02_DamascusIsNamed(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-02", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	av := placeOf(t, hh, u.Token, it, damLat, damLng)

	if av["available"] == true {
		t.Fatalf("**دمشقُ غيرُ مُطلَقةٍ ومع ذلك قُبل الطلب**: %v", av)
	}
	name, _ := av["place_name"].(string)
	if name != "دمشق" {
		t.Fatalf("**موضعُ دمشقَ لم يُسمَّ**: %q (السبب %v)", name, avReason(av))
	}
	// **والسببُ «مدينةٌ لم تُطلَق»** — `cities.active` هي رايةُ الإطلاق.
	if avReason(av) != "city_not_supported" {
		t.Fatalf("**سببٌ غيرُ متوقَّع لمدينةٍ معروفةٍ مُطفأة**: %v", avReason(av))
	}
	// **ومعرّفٌ ثابتٌ يُبنى عليه** — الدفعةُ الرابعةُ تحتاجه.
	if id, _ := av["city_id"].(string); id == "" {
		t.Fatal("**لا معرّفَ للمدينة** — ولا يُخزَّن اهتمامٌ باسمٍ يتبدّل")
	}
	if id, _ := av["governorate_id"].(string); id == "" {
		t.Fatal("**لا معرّفَ للمحافظة** — والمدينةُ منسوبةٌ إلى منطقةٍ ومحافظة")
	}
}

// **وسائرُ مراكز المحافظات تُسمّى كذلك** — **ولا اسمَ مكتوبٌ في شيفرة.**
func TestAG02b_EveryCapitalIsNamed(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-02ب", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	for _, c := range []struct {
		name     string
		lat, lng float64
	}{
		{"حلب", 36.2021, 37.1343},
		{"حمص", 34.7308, 36.7090},
		{"اللاذقية", 35.5196, 35.7915},
		{"دير الزور", 35.3359, 40.1408},
	} {
		av := placeOf(t, hh, u.Token, it, c.lat, c.lng)
		if got, _ := av["place_name"].(string); got != c.name {
			t.Errorf("**%s لم تُسمَّ**: %q (السبب %v)", c.name, got, avReason(av))
		}
	}
}

// ═════════════════ AG-03 — محافظةٌ مُطفأة ═════════════════

// TestAG03_InactiveGovernorate **ومحافظةٌ مُطفأةٌ تسبق مدينتَها.**
func TestAG03_InactiveGovernorate(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-03", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	var gov string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT g.id::text FROM governorates g
		 JOIN districts d ON d.governorate_id = g.id
		 JOIN cities c ON c.district_id = d.id
		 WHERE c.name = 'حلب' LIMIT 1`).Scan(&gov); err != nil {
		t.Skipf("لا محافظةَ منسوبةٌ لحلب: %v", err)
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE governorates SET active = false WHERE id = $1::uuid`, gov); err != nil {
		t.Fatalf("إطفاءُ المحافظة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(),
			`UPDATE governorates SET active = true WHERE id = $1::uuid`, gov)
	})

	u := hh.Customer()
	it := hh.NewItem(900)
	av := placeOf(t, hh, u.Token, it, 36.2021, 37.1343)
	if avReason(av) != "province_not_supported" {
		t.Fatalf("**محافظةٌ مُطفأةٌ لم تُقَل باسم حالها**: %v", avReason(av))
	}
	if name, _ := av["place_name"].(string); name != "حلب" {
		t.Fatalf("**المحافظةُ لم تُسمَّ**: %q", name)
	}
}

// ═════════════════ AG-04 · AG-05 — الوصفُ لا يصير حكماً ═════════════════

// TestAG04_AG05_GeographyNeverGrantsDelivery **ومدينةٌ فعّالةٌ لا تفتح
// باباً لعنوانٍ خارجَ الأشكال.**
//
// **والتغطيةُ تبقى لـ`ZoneAt`** — **ولو قيل «أنت في الرقّة» فالرقّةُ
// واسعةٌ ومناطقُ التوصيل أضيق.**
func TestAG04_AG05_GeographyNeverGrantsDelivery(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-04", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)
	// **نقطةٌ داخلَ مدينة الرقّة وخارجَ نصف قطر المنطقة.**
	far := placeOf(t, hh, u.Token, it, raqqaLat+0.08, raqqaLng)
	if avReason(far) != "address_outside_coverage" {
		t.Fatalf("**مدينةٌ فعّالةٌ فتحت باباً خارجَ الأشكال**: %v", avReason(far))
	}
	if far["order_code"] != "out_of_zone" {
		t.Fatalf("**رمزُ الإنشاء تبدّل**: %v", far["order_code"])
	}
	// **والإنشاءُ يردّها فعلاً** — والوصفُ لا يغلب الحكم.
	made := hh.POST("/api/v1/orders", u.Token, zoneBody(it, raqqaLat+0.08, raqqaLng))
	if made.Err() != "out_of_zone" {
		t.Fatalf("**الإنشاءُ ردَّ بغير `out_of_zone`**: %d / %s", made.Code, made.Err())
	}
	// **واسمُ المدينة يُعرَض معه** — فيعرف أنّ الخدمةَ تصل مدينتَه.
	if name, _ := far["place_name"].(string); name != "الرقة" {
		t.Fatalf("**لم تُسمَّ مدينتُه وهي مخدومة**: %q", name)
	}
}

// ═════════════════ AG-06 · AG-07 — الإحداثيّةُ وحدَها تحكم ═════════════════

// TestAG06_AG07_TextCannotSpoofPlace **ونصُّ العنوان لا يبدّل الموضع.**
//
// **ولا يُقرأ نصُّ العنوان في التصنيف أصلاً** — **والتصنيفُ استعلامٌ
// هندسيٌّ على الإحداثيّة.** **وهذا يقيس ذلك من الباب**: نصٌّ يقول
// «الرقّة» وإحداثيّةٌ في دمشق ⇒ **دمشق.**
func TestAG06_AG07_TextCannotSpoofPlace(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-06", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)

	// **والتسعيرةُ لا تأخذ نصَّ عنوانٍ أصلاً** — فيُقاس الإنشاءُ كذلك.
	spoof := map[string]any{
		"items":          []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"address_text":   "الرقة — مركز المدينة",
		"lat":            damLat,
		"lng":            damLng,
		"payment_method": "cash",
	}
	made := hh.POST("/api/v1/orders", u.Token, spoof)
	if made.Code < 400 {
		t.Fatalf("**نصُّ عنوانٍ في الرقّة مرّر إحداثيّةَ دمشق**: %d", made.Code)
	}

	av := placeOf(t, hh, u.Token, it, damLat, damLng)
	if name, _ := av["place_name"].(string); name != "دمشق" {
		t.Fatalf("**الموضعُ تبع النصَّ لا الإحداثيّة**: %q", name)
	}
}

// ═════════════════ AG-08 — ما لا يُعرَف يبقى مجهولاً ═════════════════

// TestAG08_UnmappedAreaFallsBack **وموضعٌ لا مركزَ يحويه يبقى بلا اسم.**
//
// **ولا يُخترَع اسم** — **وقريةٌ في البادية بعيدةٌ عن كلّ مركز.**
func TestAG08_UnmappedAreaFallsBack(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-08", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	// **البادية — بعيدةٌ عن كلّ مركزِ محافظة.**
	av := placeOf(t, hh, u.Token, it, 34.20, 38.60)
	if avReason(av) != "area_not_supported" {
		t.Fatalf("**موضعٌ مجهولٌ صُنّف غيرَ ذلك**: %v", avReason(av))
	}
	if name, _ := av["place_name"].(string); name != "" {
		t.Fatalf("**سُمّي ما لا يُعرَف**: %q", name)
	}
}

// ═════════════════ AG-09 · AG-10 — العقودُ القائمة ═════════════════

// TestAG09_InactiveCitySemantics **ورايةُ المدينة تبقى رايةَ الإطلاق.**
//
// **ولا يتبدّل تصفّحٌ ببذرِ مدنٍ مُطفأة** — **و`city_filter` يشترط
// `c.active`**، **فالمُطفأةُ لا تدخل حسبتَه.**
func TestAG09_InactiveCitySemantics(t *testing.T) {
	hh := New(t)
	hh.Setting("launch.customer_browse", "true")
	ordersOpen(t, hh)

	// **وتصفّحُ الرقّة يبقى كما كان** — والمدنُ المبذورةُ مُطفأة.
	if r := hh.GET("/api/v1/public/home?lat=35.9506&lng=39.0094", ""); r.Code != 200 {
		t.Fatalf("**تصفّحُ مدينةٍ مخدومةٍ تبدّل**: %d / %s", r.Code, r.Err())
	}
	// **وعددُ المدن الفعّالة لم يتبدّل ببذرِ المُطفأة.**
	var active int
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM cities WHERE active`).Scan(&active); err != nil {
		t.Fatalf("عدُّ المدن: %v", err)
	}
	if active != 1 {
		t.Fatalf("**بذرُ المدن بدّل عددَ الفعّالة**: %d — **والمبذورةُ مُطفأةٌ كلُّها**", active)
	}
}

// TestAG10_ReasonsUnchanged **وأسبابُ الدفعة الثالثة لم تُبدَّل.**
func TestAG10_ReasonsUnchanged(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AG-10", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)

	// **مفتوحٌ ⇒ `service_available`.**
	zoneHours(t, hh, z.ID, false)
	if got := avReason(placeOf(t, hh, u.Token, it, raqqaLat, raqqaLng)); got != "service_available" {
		t.Errorf("**`service_available` تبدّل**: %v", got)
	}
	// **ووقتُ المنطقة ⇒ `zone_closed_now`.**
	zoneHours(t, hh, z.ID, true, zhShut())
	if got := avReason(placeOf(t, hh, u.Token, it, raqqaLat, raqqaLng)); got != "zone_closed_now" {
		t.Errorf("**`zone_closed_now` تبدّل**: %v", got)
	}
	// **ونقطةٌ مشوَّهةٌ ⇒ `invalid_location`.**
	if got := avReason(placeOf(t, hh, u.Token, it, 0, 0)); got != "invalid_location" {
		t.Errorf("**`invalid_location` تبدّل**: %v", got)
	}
}
