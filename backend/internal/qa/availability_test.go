package qa

// ══════════════════════════════════════════════════════════════════════
// **النموذجُ القارئُ الواحد — أيُطلَب ولِمَ لا ومتى** (`AV`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **ويُقاس من باب الشبكة** — **فالشاشةُ تقرؤه من هناك.**
//
// **وأهمُّ ما يُقاس أنّ الشرحَ يصف ما سيفعله المنعُ فعلاً** (`AV-24`
// و`AV-25`): **وشرحٌ يقول «مفتوح» ومنعٌ يردّ أسوأُ من لا شرح.**

import (
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/platform"
)

// avOf **حالُ الإتاحة كما يقرؤها العميل.**
func avOf(t *testing.T, h *Harness, tok string, it *Item, lat, lng float64) map[string]any {
	t.Helper()
	r := h.POST("/api/v1/public/quote", tok, map[string]any{
		"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"lat":   lat, "lng": lng,
	})
	if r.Code != http.StatusOK {
		t.Fatalf("**التسعيرةُ سقطت**: %d / %s", r.Code, r.Err())
	}
	av, _ := r.JSON()["availability"].(map[string]any)
	if av == nil {
		t.Fatal("**لا حالَ إتاحةٍ في الردّ** — فالشاشةُ تخمّن")
	}
	return av
}

// avReason سببُ الإتاحة نصّاً.
func avReason(av map[string]any) string {
	s, _ := av["reason"].(string)
	return s
}

// raqqa نقطةٌ في مدينة الرقّة — المدينةُ الوحيدةُ المبذورة.
const raqqaLat, raqqaLng = 35.9506, 39.0094

// damascus نقطةٌ لا مدينةَ مبذورةً تحويها.
const damLat, damLng = 33.5138, 36.2765

// ═════════════════ AV-01 · AV-12 — كلُّ الأبواب مفتوحة ═════════════════

func TestAV01_AllOpenIsAvailable(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-01", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if av["available"] != true || avReason(av) != "service_available" {
		t.Fatalf("**كلُّ الأبواب مفتوحةٌ ومع ذلك مُنع**: %v", av)
	}
}

// ═════════════════ AV-02 … AV-04 — الأسبقيّة ═════════════════

// TestAV02_LaunchClosedWinsOverEverything **وضعُ الإطلاق يغلب ما بعده.**
func TestAV02_LaunchClosedWinsOverEverything(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-02", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	// **وكلُّ ما بعده مغلقٌ أيضاً** — فلو سبق أحدُها لَظهر.
	zoneHours(t, hh, z.ID, true, zhShut())
	closure(t, hh, true, "صيانة", nil)
	closedNow(t, hh)
	hh.Setting("launch.customer_orders", "false")

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "launch_closed" {
		t.Fatalf("**سببٌ أخصُّ حجب وضعَ الإطلاق**: %v", avReason(av))
	}
	// **ولا موعدَ لبابٍ لم يُفتح بعد.**
	if _, has := av["next_available_at"]; has {
		t.Fatalf("**اخترعَ موعداً لبابٍ لا موعدَ لفتحه**: %v", av["next_available_at"])
	}
}

// TestAV03_ClosureWinsOverLater **ثمّ الإيقافُ المؤقّت.**
func TestAV03_ClosureWinsOverLater(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-03", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())
	closedNow(t, hh)
	closure(t, hh, true, "صيانةٌ مؤقّتة", nil)

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "temporarily_unavailable" {
		t.Fatalf("**سببٌ أخصُّ حجب الإيقافَ المؤقّت**: %v", avReason(av))
	}
	if av["message"] != "صيانةٌ مؤقّتة" {
		t.Fatalf("**نصُّ المالك لم يصل**: %v", av["message"])
	}
}

// TestAV04_PlatformHoursClosed **ثمّ دوامُ المنصّة.**
func TestAV04_PlatformHoursClosed(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-04", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())
	closedNow(t, hh)

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "platform_closed_now" {
		t.Fatalf("**وقتُ المنطقة حجب دوامَ المنصّة**: %v", avReason(av))
	}
}

// ═════════════════ AV-05 · AV-06 — النقطةُ والتغطية ═════════════════

// TestAV05_InvalidLocation **نقطةٌ ليست نقطة.**
func TestAV05_InvalidLocation(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, 0, 0)
	if avReason(av) != "invalid_location" || av["order_code"] != "bad_point" {
		t.Fatalf("**صفرٌ صفرٌ لم يُقَل باسمه**: %v", av)
	}
}

// TestAV06_CoverageUnavailable **لا تغطيةَ صالحةً — حالُ إعدادٍ لا حكمٌ
// على العنوان.**
func TestAV06_CoverageUnavailable(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	// **تُطفأ كلُّ المناطق** — ولا واحدةَ تبقى.
	z := newZone(t, hh, "منطقةُ AV-06", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("إطفاءُ المنطقة: %v", err)
	}

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "coverage_unavailable" {
		t.Fatalf("**لا تغطيةَ صالحةً وقيل غيرُ ذلك**: %v", avReason(av))
	}
	if av["order_code"] != "coverage_unavailable" {
		t.Fatalf("**رمزُ الإنشاء لا يوافق**: %v", av["order_code"])
	}
}

// ═════════════════ AV-07 … AV-09 — الجغرافيا الإداريّة ═════════════════

// TestAV07_UnknownAreaIsNotOutsideCoverage **«لم نصل بعد» غيرُ «عنوانُك
// خارجَ النطاق».**
//
// **ودمشقُ لا مدينةَ مبذورةً تحويها** — **والمحافظاتُ بلا هندسةٍ في
// المخطَّط، فلا يُسمّى ما لا يُعرَف.**
func TestAV07_UnknownAreaIsNotOutsideCoverage(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-07", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, damLat, damLng)
	if avReason(av) == "address_outside_coverage" {
		t.Fatal("**قيل «عنوانُك خارجَ النطاق» لموضعٍ لم تُطلَق فيه الخدمةُ أصلاً** — " +
			"**ومن قرأها ظنّ أنّ المنصّةَ تصله ولا تصل عنوانَه**")
	}
	if avReason(av) != "area_not_supported" {
		t.Fatalf("**سببٌ غيرُ متوقَّع**: %v", avReason(av))
	}
}

// TestAV08_InactiveCityIsNamed **ومدينةٌ معروفةٌ مُطفأةٌ تُسمّى.**
func TestAV08_InactiveCityIsNamed(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-08", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	// **تُطفأ مدينةُ النقطة ثمّ تُعاد.**
	var city string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT id::text FROM cities
		WHERE ST_DWithin(center, ST_SetSRID(ST_MakePoint($2,$1),4326)::geography, radius_m)
		LIMIT 1`, raqqaLat, raqqaLng).Scan(&city); err != nil {
		t.Skipf("لا مدينةَ تحوي النقطةَ في هذه القاعدة: %v", err)
	}
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE cities SET active = false WHERE id = $1::uuid`, city); err != nil {
		t.Fatalf("إطفاءُ المدينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(),
			`UPDATE cities SET active = true WHERE id = $1::uuid`, city)
	})

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "city_not_supported" {
		t.Fatalf("**مدينةٌ مُطفأةٌ لم تُقَل باسم حالها**: %v", avReason(av))
	}
	if name, _ := av["place_name"].(string); name == "" {
		t.Fatal("**مدينةٌ معروفةٌ ولم تُسمَّ** — والرسالةُ تحتاج اسمَها")
	}
}

// TestAV09_InsideCityOutsideZone **أُطلقت الخدمةُ هنا والعنوانُ خارجَ
// الأشكال.**
func TestAV09_InsideCityOutsideZone(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	// **منطقةٌ صغيرةٌ بعيدةٌ عن النقطة داخلَ المدينة نفسِها.**
	z := newZone(t, hh, "منطقةُ AV-09", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)
	// **نقطةٌ في المدينة وخارجَ نصف قطر المنطقة (٢ كم).**
	av := avOf(t, hh, u.Token, it, raqqaLat+0.08, raqqaLng)
	if avReason(av) != "address_outside_coverage" {
		t.Fatalf("**عنوانٌ في مدينةٍ مخدومةٍ خارجَ الأشكال قيل عنه غيرُ ذلك**: %v",
			avReason(av))
	}
	if av["order_code"] != "out_of_zone" {
		t.Fatalf("**رمزُ الإنشاء لا يوافق**: %v", av["order_code"])
	}
}

// ═════════════════ AV-10 · AV-11 — وقتُ المنطقة والمتجر ═════════════════

func TestAV10_ZoneClosedNow(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-10", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "zone_closed_now" {
		t.Fatalf("**منطقةٌ خارجَ وقتها قيل عنها غيرُ ذلك**: %v", avReason(av))
	}
	if _, ok := av["next_available_at"].(string); !ok {
		t.Fatalf("**أُغلقت بلا موعدٍ وهو معلوم**: %v", av)
	}
}

// TestAV11_MerchantClosedNow **المنصّةُ والمنطقةُ مفتوحتان والمتجرُ
// مغلق.**
func TestAV11_MerchantClosedNow(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-11", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)
	// **يُغلَق متجرُ الصنف طارئاً** — وهو أحدُ وجهَي `OpenNowSQL`.
	if _, err := hh.Pool.Exec(ctxBG(), `
		UPDATE merchants SET emergency_closed = true
		WHERE id = (SELECT merchant_id FROM menu_items WHERE id = $1::uuid)`,
		it.ID); err != nil {
		t.Fatalf("إغلاقُ المتجر: %v", err)
	}

	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "merchant_closed_now" {
		t.Fatalf("**متجرٌ مغلقٌ قيل عنه غيرُ ذلك**: %v", avReason(av))
	}
	// **ولا موعدَ لإغلاقٍ طارئ** — **ولا نعرف متى يُرفَع.**
	if _, has := av["next_available_at"]; has {
		t.Fatalf("**اخترعَ موعداً لإغلاقٍ طارئٍ لا يُعرَف رفعُه**: %v",
			av["next_available_at"])
	}
}

// ═════════════════ AV-19 · AV-20 — العنوانُ المختار ═════════════════

// TestAV19_AV20_SelectedAddressDecides **العنوانُ المختارُ هو الحقيقة —
// وتبديلُه يقلب الجواب فوراً.**
//
// **ولا موضعَ جهازٍ في المسار أصلاً** — **والنداءُ يحمل إحداثيّةَ
// العنوان**، **ولا شيءَ في الخادم يقرأ موضعَ الهاتف.**
func TestAV19_AV20_SelectedAddressDecides(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-19", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)

	// **عنوانُ الرقّة يمضي.**
	if av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng); av["available"] != true {
		t.Fatalf("**عنوانُ الرقّة مُنع**: %v", av)
	}
	// **وتبديلُه إلى دمشقَ يقلب الجواب — بالنداء نفسِه.**
	far := avOf(t, hh, u.Token, it, damLat, damLng)
	if far["available"] == true {
		t.Fatalf("**عنوانُ دمشقَ قُبل**: %v", far)
	}
	// **والعودةُ تُرجع القبول.**
	if av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng); av["available"] != true {
		t.Fatalf("**العودةُ إلى الرقّة مُنعت**: %v", av)
	}
}

// ═════════════════ AV-21 · AV-22 · AV-23 — التصفّحُ والعقدُ القديم ═════════════════

// TestAV21_AV22_BrowsingAndOldFields **التصفّحُ يبقى، والحقولُ القائمةُ
// تبقى.**
func TestAV21_AV22_BrowsingAndOldFields(t *testing.T) {
	hh := New(t)
	hh.Setting("launch.customer_browse", "true")
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-21", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, true, zhShut())

	u := hh.Customer()
	it := hh.NewItem(900)

	for _, p := range []string{"/api/v1/public/home", "/api/v1/public/platform"} {
		if r := hh.GET(p, ""); r.Code != http.StatusOK {
			t.Errorf("**بابُ تصفّحٍ أُغلق مع الطلب**: %s ⇒ %d", p, r.Code)
		}
	}

	r := hh.POST("/api/v1/public/quote", u.Token, map[string]any{
		"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"lat":   raqqaLat, "lng": raqqaLng,
	})
	j := r.JSON()
	// **AV-22/23 · وعميلٌ منشورٌ لا يقرأ الحقلَ الجديد يبقى يعمل.**
	for _, k := range []string{
		"subtotal", "delivery_fee", "total", "serviceable",
		"out_of_zone", "zone_closed", "sources", "max_sources",
	} {
		if _, ok := j[k]; !ok {
			t.Errorf("**حقلٌ قائمٌ اختفى من التسعيرة**: %s", k)
		}
	}
}

// ═════════════════ AV-24 · AV-25 — الشرحُ يوافق المنع ═════════════════

// TestAV24_AV25_PreflightMatchesAdmission **ما يقوله الشرحُ هو ما يفعله
// المنع.**
//
// **وشرحٌ يقول «مفتوح» ومنعٌ يردّ أسوأُ من لا شرح** — **ومن ملأ سلّةً
// على وعدٍ ثمّ رُدّ لا يعيد.**
func TestAV24_AV25_PreflightMatchesAdmission(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-24", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)

	cases := []struct {
		name  string
		setup func()
		lat   float64
		lng   float64
	}{
		{"وقتُ المنطقة", func() { zoneHours(t, hh, z.ID, true, zhShut()) }, raqqaLat, raqqaLng},
		{"دوامُ المنصّة", func() { zoneHours(t, hh, z.ID, false); closedNow(t, hh) }, raqqaLat, raqqaLng},
		{"إيقافٌ مؤقّت", func() { ordersOpen(t, hh); closure(t, hh, true, "صيانة", nil) }, raqqaLat, raqqaLng},
		{"خارجَ التغطية", func() { ordersOpen(t, hh) }, damLat, damLng},
	}
	for _, c := range cases {
		c.setup()
		av := avOf(t, hh, u.Token, it, c.lat, c.lng)
		if av["available"] == true {
			t.Fatalf("%s: **الشرحُ يقول مفتوح**: %v", c.name, av)
		}
		want, _ := av["order_code"].(string)

		made := hh.POST("/api/v1/orders", u.Token, zoneBody(it, c.lat, c.lng))
		if made.Code < 400 {
			t.Fatalf("%s: **الشرحُ منع والإنشاءُ قَبِل** — %d", c.name, made.Code)
		}
		if want != "" && made.Err() != want {
			t.Fatalf("%s: **الشرحُ يقول %q والإنشاءُ ردَّ %q**", c.name, want, made.Err())
		}

		// **والمخصَّصُ يوافق صنفَ المنع كذلك** — ولا رمزَ متجرٍ فيه.
		if want != "out_of_zone" {
			cus := hh.POST("/api/v1/orders/custom", u.Token, zoneCustomBody(c.lat, c.lng))
			if cus.Code < 400 {
				t.Fatalf("%s: **المخصَّصُ قَبِل والشرحُ منع** — %d", c.name, cus.Code)
			}
		}
	}
}

// ═════════════════ AV-27 … AV-30 — الحالاتُ المغلقة ═════════════════

// TestAV27_AV30_HTTPContracts **ولا تُوحَّد الحالات.**
func TestAV27_AV30_HTTPContracts(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ AV-27", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)

	u := hh.Customer()
	it := hh.NewItem(900)

	// AV-27 · out_of_zone ⇒ ٤٠٠
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, damLat, damLng)); r.Code != http.StatusBadRequest || r.Err() != "out_of_zone" {
		t.Errorf("**AV-27**: %d / %s — والعقدُ ٤٠٠ `out_of_zone`", r.Code, r.Err())
	}
	// AV-29 · zone_closed_now ⇒ ٥٠٣
	zoneHours(t, hh, z.ID, true, zhShut())
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, raqqaLat, raqqaLng)); r.Code != http.StatusServiceUnavailable || r.Err() != "zone_closed_now" {
		t.Errorf("**AV-29**: %d / %s — والعقدُ ٥٠٣", r.Code, r.Err())
	}
	// AV-28 · coverage_unavailable ⇒ ٥٠٣
	zoneHours(t, hh, z.ID, false)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("إطفاءُ المنطقة: %v", err)
	}
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, raqqaLat, raqqaLng)); r.Code != http.StatusServiceUnavailable || r.Err() != "coverage_unavailable" {
		t.Errorf("**AV-28**: %d / %s — والعقدُ ٥٠٣", r.Code, r.Err())
	}
	// AV-30 · launch_closed ⇒ ٥٠٣
	hh.Setting("launch.customer_orders", "false")
	if r := hh.POST("/api/v1/orders", u.Token, zoneBody(it, raqqaLat, raqqaLng)); r.Code != http.StatusServiceUnavailable || r.Err() != "launch_closed" {
		t.Errorf("**AV-30**: %d / %s — والعقدُ ٥٠٣", r.Code, r.Err())
	}
}

// ═════════════════ اتّساقُ دوام المتجر مع النصّ القائم ═════════════════

// TestAV_MerchantGateMatchesOpenNowSQL **حسبةُ الجداول توافق `OpenNowSQL`.**
//
// **وقيدُ المتجر في تقاطع المواعيد يُبنى من صفوف `merchant_hours`**،
// **و«أمفتوحٌ الآن» يُقاس بـ`OpenNowSQL`** — **وحسبتان على بياناتٍ
// واحدةٍ تفترقان يوماً.**
//
// **وأخطرُ فرقٍ**: **بلا صفوفِ دوامٍ يقول النصُّ «مفتوحٌ دائماً»**،
// **وجدولٌ فارغٌ في الحساب يقول «مغلقٌ أبداً».**
func TestAV_MerchantGateMatchesOpenNowSQL(t *testing.T) {
	hh := New(t)
	ordersOpen(t, hh)
	z := newZone(t, hh, "منطقةُ الاتّساق", raqqaLat, raqqaLng)
	otherZonesOff(t, hh, z.ID)
	zoneHours(t, hh, z.ID, false)

	u := hh.Customer()
	it := hh.NewItem(900)
	var mid string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT merchant_id::text FROM menu_items WHERE id = $1::uuid`, it.ID).Scan(&mid); err != nil {
		t.Fatalf("متجرُ الصنف: %v", err)
	}

	// **بلا صفوفٍ — مفتوحٌ دائماً.**
	if _, err := hh.Pool.Exec(ctxBG(),
		`DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, mid); err != nil {
		t.Fatalf("مسحُ الدوام: %v", err)
	}
	if av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng); av["available"] != true {
		t.Fatalf("**متجرٌ بلا صفوفِ دوامٍ عُدَّ مغلقاً** — "+
			"**و`OpenNowSQL` يقول مفتوحٌ دائماً**: %v", av)
	}

	// ══════════════════════════════════════════════════════════════════
	// **ولا يكفي أن يوافقه في «مفتوحٌ الآن»** (٢٠٢٦-٠٩-١٤)
	// ══════════════════════════════════════════════════════════════════
	//
	// **وقيدُ المتجر في تقاطع المواعيد لا يُسأل إلّا عند المنع** —
	// **فمتجرٌ مفتوحٌ لا يبلغه السؤالُ أصلاً، ويبقى فرقُ «بلا صفوفٍ»
	// مستوراً.**
	//
	// **فيُغلَق سببٌ آخرُ ويُسأل عن الموعد**: **متجرٌ بلا جدولٍ لا
	// يقيّد**، **فالموعدُ موعدُ المنطقة.** **ولو عُدَّ جدولاً فارغاً
	// سارياً لَما تقاطع شيءٌ أبداً ولَعاد بلا موعد.**
	//
	// (**كشفه شاهدٌ سالبٌ مرّ كذباً** — والفحصُ أعلاه وحدَه لم يره.)
	zoneHours(t, hh, z.ID, true, zhShut())
	closedByZone := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(closedByZone) != "zone_closed_now" {
		t.Fatalf("مقدّمةٌ مكسورة: %v", avReason(closedByZone))
	}
	if _, ok := closedByZone["next_available_at"].(string); !ok {
		t.Fatalf("**متجرٌ بلا صفوفِ دوامٍ منع تقاطعَ المواعيد** — "+
			"**و`OpenNowSQL` يقول إنّه لا يقيّد شيئاً**: %v", closedByZone)
	}
	zoneHours(t, hh, z.ID, false)

	// **وبفترةٍ تحوي الآن — مفتوح.**
	now := time.Now().In(platform.Location())
	open := now.Add(-time.Hour)
	shut := now.Add(time.Hour)
	setMerchantDay(t, hh, mid, int(open.Weekday()), open, shut)
	if av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng); av["available"] != true {
		t.Fatalf("**متجرٌ داخلَ دوامه عُدَّ مغلقاً**: %v", av)
	}

	// **وبفترةٍ لا تحوي الآن — مغلق، وله موعد.**
	a := now.Add(2 * time.Hour)
	b := now.Add(3 * time.Hour)
	setMerchantDay(t, hh, mid, int(a.Weekday()), a, b)
	av := avOf(t, hh, u.Token, it, raqqaLat, raqqaLng)
	if avReason(av) != "merchant_closed_now" {
		t.Fatalf("**متجرٌ خارجَ دوامه عُدَّ مفتوحاً**: %v", avReason(av))
	}
	if _, ok := av["next_available_at"].(string); !ok {
		t.Fatalf("**دوامٌ معلومٌ بلا موعدِ فتح**: %v", av)
	}
}

// setMerchantDay يكتب دوامَ يومٍ واحدٍ ويُغلق ما سواه.
func setMerchantDay(t *testing.T, h *Harness, mid string, day int, from, to time.Time) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(),
		`DELETE FROM merchant_hours WHERE merchant_id = $1::uuid`, mid); err != nil {
		t.Fatalf("مسحُ الدوام: %v", err)
	}
	for d := 0; d < 7; d++ {
		closed := d != day
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO merchant_hours (merchant_id, day_of_week, closed, open_time, close_time)
			VALUES ($1::uuid, $2, $3, make_time($4,$5,0), make_time($6,$7,0))`,
			mid, d, closed, from.Hour(), from.Minute(), to.Hour(), to.Minute()); err != nil {
			t.Fatalf("كتابةُ دوام: %v", err)
		}
	}
}
