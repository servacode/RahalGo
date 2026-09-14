package qa

// ══════════════════════════════════════════════════════════════════════
// **نيّةُ التوسّع — طلبُ منطقةٍ و«أخبرني»** (`CR`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **وأهمُّ ما يُقاس أنّ الخادمَ يُعيد الحكمَ** (`CR-10`…`CR-12`):
// **وعميلٌ معدَّلٌ يلوّث دفترَ الطلب بإشاراتٍ كاذبةٍ يُبنى عليها قرارُ
// توسّع.**

import (
	"net/http"
	"testing"
)

// demand **يُرسل نيّةَ توسّعٍ ويردّ الجسم.**
func demand(t *testing.T, h *Harness, tok string, lat, lng float64, kind string) Res {
	t.Helper()
	body := map[string]any{"lat": lat, "lng": lng, "address_text": "نصٌّ لا يُقرأ جغرافيّاً"}
	if kind != "" {
		body["kind"] = kind
	}
	return h.POST("/api/v1/demand", tok, body)
}

// demandRows عددُ صفوف نيّةٍ لحسابٍ.
func demandRows(t *testing.T, h *Harness, userID, kind string) (rows, requests int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(sum(requests), 0)
		FROM coverage_requests WHERE user_id = $1::uuid AND kind = $2`,
		userID, kind).Scan(&rows, &requests); err != nil {
		t.Fatalf("عدُّ الإشارات: %v", err)
	}
	return rows, requests
}

// zoneForDemand منطقةٌ فعّالةٌ حول الرقّة، بلا سريانِ وقت.
func zoneForDemand(t *testing.T, h *Harness, name string) zoneFx {
	t.Helper()
	ordersOpen(t, h)
	z := newZone(t, h, name, raqqaLat, raqqaLng)
	otherZonesOff(t, h, z.ID)
	zoneHours(t, h, z.ID, false)
	return z
}

// ═════════════════ CR-01 … CR-04 — طلبُ التغطية والتفرّد ═════════════════

// TestCR01_OutsideCoverageCreatesRequest **عنوانٌ في مدينةٍ مخدومةٍ خارجَ
// الشكل يُسجّل طلبَ توسّع.**
func TestCR01_OutsideCoverageCreatesRequest(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-01")
	u := hh.Customer()

	r := demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, "")
	if r.Code != http.StatusOK {
		t.Fatalf("**طلبُ التوسّع رُدّ**: %d / %s", r.Code, r.Err())
	}
	j := r.JSON()
	if j["kind"] != "coverage_request" {
		t.Fatalf("**نيّةٌ غيرُ متوقَّعة**: %v", j["kind"])
	}
	if j["outcome"] != "created" {
		t.Fatalf("**أوّلُ طلبٍ لم يُعَدّ إنشاءً**: %v", j["outcome"])
	}
	if rows, _ := demandRows(t, hh, u.ID, "coverage_request"); rows != 1 {
		t.Fatalf("**صفوفٌ غيرُ متوقَّعة**: %d", rows)
	}
}

// TestCR02_CR03_CR20_SameAreaDedupes **وضغطتان في الحيّ نفسِه طلبٌ
// واحدٌ يُعَدّ مرّتين.**
//
// **ولا تُقارَن الإحداثيّاتُ بالتساوي العشريّ** — **وضغطتان على النقطة
// نفسِها تختلفان في الخانة السابعة.**
func TestCR02_CR03_CR20_SameAreaDedupes(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-02")
	u := hh.Customer()

	// **النقطةُ عينُها.**
	if r := demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, ""); r.Code != http.StatusOK {
		t.Fatalf("الأولى: %d / %s", r.Code, r.Err())
	}
	// **ونقطةٌ تبعد أمتاراً — داخلَ الخليّة نفسِها (≈ ١٫١ كم).**
	second := demand(t, hh, u.Token, raqqaLat+0.0802, raqqaLng+0.0003, "")
	if second.Code != http.StatusOK {
		t.Fatalf("الثانية: %d / %s", second.Code, second.Err())
	}
	if second.JSON()["outcome"] != "already_registered" {
		t.Fatalf("**نقطةٌ مجاورةٌ وُلد لها طلبٌ ثانٍ**: %v", second.JSON()["outcome"])
	}
	// **وثالثةٌ ورابعةٌ لا تؤذيان.**
	for i := 0; i < 3; i++ {
		if r := demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, ""); r.Code != http.StatusOK {
			t.Fatalf("**ضغطةٌ مكرّرةٌ رُدّت بعطب**: %d / %s", r.Code, r.Err())
		}
	}
	rows, requests := demandRows(t, hh, u.ID, "coverage_request")
	if rows != 1 {
		t.Fatalf("**تفرّدٌ مكسور**: %d صفّاً", rows)
	}
	if requests != 5 {
		t.Fatalf("**شدّةُ الطلب لم تُعَدّ**: %d", requests)
	}
}

// TestCR04_DifferentAreaIsDistinct **ومنطقةٌ أخرى إشارةٌ أخرى.**
func TestCR04_DifferentAreaIsDistinct(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-04")
	u := hh.Customer()

	if r := demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, ""); r.Code != http.StatusOK {
		t.Fatalf("الأولى: %d / %s", r.Code, r.Err())
	}
	// **وخليّةٌ أخرى بعيدةٌ بضعةَ كيلومترات.**
	other := demand(t, hh, u.Token, raqqaLat+0.10, raqqaLng+0.03, "")
	if other.Code != http.StatusOK {
		t.Fatalf("الثانية: %d / %s", other.Code, other.Err())
	}
	if other.JSON()["outcome"] != "created" {
		t.Fatalf("**منطقةٌ مختلفةٌ دُمجت في سابقتها**: %v", other.JSON()["outcome"])
	}
	if rows, _ := demandRows(t, hh, u.ID, "coverage_request"); rows != 2 {
		t.Fatalf("**صفّان متوقَّعان**: %d", rows)
	}
}

// ═════════════════ CR-05 … CR-09 — «أخبرني» والمعرّفات ═════════════════

// TestCR05_CR08_CityInterestStoresIDs **مدينةٌ معروفةٌ لم تُطلَق ⇒
// اشتراكٌ بمعرّفاتٍ ثابتة.**
func TestCR05_CR08_CityInterestStoresIDs(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-05")
	u := hh.Customer()

	r := demand(t, hh, u.Token, damLat, damLng, "")
	if r.Code != http.StatusOK {
		t.Fatalf("**«أخبرني» رُدّ**: %d / %s", r.Code, r.Err())
	}
	j := r.JSON()
	if j["kind"] != "service_interest" {
		t.Fatalf("**نيّةٌ غيرُ متوقَّعة لدمشق**: %v", j["kind"])
	}
	if j["reason"] != "city_not_supported" {
		t.Fatalf("**سببٌ غيرُ متوقَّع**: %v", j["reason"])
	}
	var city, gov *string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT city_id::text, governorate_id::text FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = 'service_interest'`,
		u.ID).Scan(&city, &gov); err != nil {
		t.Fatalf("قراءةُ الاشتراك: %v", err)
	}
	if city == nil || *city == "" {
		t.Fatal("**لا معرّفَ مدينةٍ** — والدفعةُ الثامنةُ تحتاجه")
	}
	if gov == nil || *gov == "" {
		t.Fatal("**لا معرّفَ محافظةٍ** — والسؤالُ «كم في دمشق؟» محافظة")
	}
}

// TestCR07_CR09_UnmappedKeepsNullIDs **وموضعٌ مجهولٌ إشارةٌ بإحداثيّةٍ
// بلا مكانٍ مخترَع.**
func TestCR07_CR09_UnmappedKeepsNullIDs(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-07")
	u := hh.Customer()

	r := demand(t, hh, u.Token, 34.20, 38.60, "")
	if r.Code != http.StatusOK {
		t.Fatalf("**إشارةُ البادية رُدّت**: %d / %s", r.Code, r.Err())
	}
	if r.JSON()["reason"] != "area_not_supported" {
		t.Fatalf("**سببٌ غيرُ متوقَّع**: %v", r.JSON()["reason"])
	}
	var city, gov *string
	var cy, cx *float64
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT city_id::text, governorate_id::text, cell_y, cell_x
		FROM coverage_requests WHERE user_id = $1::uuid`,
		u.ID).Scan(&city, &gov, &cy, &cx); err != nil {
		t.Fatalf("قراءةُ الإشارة: %v", err)
	}
	if city != nil || gov != nil {
		t.Fatalf("**اختُرع مكانٌ لنقطةٍ لا تعرفها المنصّة**: city=%v gov=%v", city, gov)
	}
	if cy == nil || cx == nil {
		t.Fatal("**إشارةٌ بلا خليّة** — فلا تُجمَّع ولا تُفرَّد")
	}
}

// ═════════════════ CR-10 … CR-12 — الخادمُ يُعيد الحكم ═════════════════

// TestCR10_CR11_CR12_ServerReevaluates **ولا سببَ يُصدَّق من عميل.**
func TestCR10_CR11_CR12_ServerReevaluates(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-10")
	u := hh.Customer()

	// **CR-10/11 · عنوانٌ مخدومٌ لا يُسجَّل إشارةَ توسّعٍ مهما أرسل.**
	for _, k := range []string{"", "coverage_request", "service_interest"} {
		r := demand(t, hh, u.Token, raqqaLat, raqqaLng, k)
		if r.Code < 400 {
			t.Fatalf("**سُجّلت إشارةٌ لعنوانٍ مخدوم** (kind=%q): %d", k, r.Code)
		}
		if r.Err() != "service_now_available" {
			t.Fatalf("**رمزٌ غيرُ متوقَّع** (kind=%q): %s", k, r.Err())
		}
	}
	if rows, _ := demandRows(t, hh, u.ID, "coverage_request"); rows != 0 {
		t.Fatalf("**صفٌّ وُلد من نداءٍ مردود**: %d", rows)
	}

	// **CR-12 · ونيّةٌ تخالف ما قضاه الخادمُ تُردّ ولا تُقلَب صامتةً.**
	wrong := demand(t, hh, u.Token, damLat, damLng, "coverage_request")
	if wrong.Err() != "reason_mismatch" {
		t.Fatalf("**نيّةٌ خاطئةٌ قُبلت أو رُدّت بغير رمزها**: %d / %s",
			wrong.Code, wrong.Err())
	}
}

// ═════════════════ CR-13 … CR-17 — الأسبابُ التي لا تُعرَض ═════════════════

// TestCR13_CR17_TemporalReasonsExposeNoCTA **والأسبابُ الزمنيّةُ ليست
// أسبابَ توسّع.**
//
// **ومن عُرض عليه «اطلب إضافة منطقتك» لأنّ المتجرَ مغلقٌ ظنّ أنّنا لا
// نصله.**
func TestCR13_CR17_TemporalReasonsExposeNoCTA(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CR-13")
	u := hh.Customer()

	cases := []struct {
		name  string
		setup func()
	}{
		{"دوامُ المنصّة", func() { closedNow(t, hh) }},
		{"إيقافٌ مؤقّت", func() { ordersOpen(t, hh); closure(t, hh, true, "صيانة", nil) }},
		{"وقتُ المنطقة", func() { ordersOpen(t, hh); zoneHours(t, hh, z.ID, true, zhShut()) }},
		{"وضعُ الإطلاق", func() {
			ordersOpen(t, hh)
			zoneHours(t, hh, z.ID, false)
			hh.Setting("launch.customer_orders", "false")
		}},
	}
	for _, c := range cases {
		c.setup()
		r := demand(t, hh, u.Token, raqqaLat, raqqaLng, "")
		if r.Code < 400 {
			t.Fatalf("%s: **عُرضت نيّةُ توسّعٍ لسببٍ زمنيّ**: %d", c.name, r.Code)
		}
		if r.Err() != "reason_mismatch" && r.Err() != "service_now_available" {
			t.Fatalf("%s: **رمزٌ غيرُ متوقَّع**: %s", c.name, r.Err())
		}
	}

	// **CR-16 · وعطبُ إعدادِ التغطية لا يُدعى الناسُ بسببه.**
	ordersOpen(t, hh)
	if _, err := hh.Pool.Exec(ctxBG(),
		`UPDATE delivery_zones SET active = false WHERE id = $1::uuid`, z.ID); err != nil {
		t.Fatalf("إطفاءُ المنطقة: %v", err)
	}
	r := demand(t, hh, u.Token, raqqaLat, raqqaLng, "")
	if r.Code < 400 || r.Err() != "reason_mismatch" {
		t.Fatalf("**`coverage_unavailable` فتح بابَ طلبِ المنطقة**: %d / %s",
			r.Code, r.Err())
	}
	if rows, _ := demandRows(t, hh, u.ID, "coverage_request"); rows != 0 {
		t.Fatalf("**صفٌّ وُلد من سببٍ لا يصلح**: %d", rows)
	}
}

// ═════════════════ CR-18 · CR-19 — الإحداثيّةُ وحدَها ═════════════════

// TestCR18_CR19_CoordinateIsAuthority **ونصُّ العنوان لا يبدّل المكان.**
func TestCR18_CR19_CoordinateIsAuthority(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-18")
	u := hh.Customer()

	// **نصٌّ يقول «الرقّة» وإحداثيّةٌ في دمشق ⇒ اشتراكُ دمشق.**
	r := hh.POST("/api/v1/demand", u.Token, map[string]any{
		"lat": damLat, "lng": damLng,
		"address_text": "الرقة — مركز المدينة",
	})
	if r.Code != http.StatusOK {
		t.Fatalf("**رُدّ**: %d / %s", r.Code, r.Err())
	}
	if r.JSON()["kind"] != "service_interest" {
		t.Fatalf("**النصُّ غلب الإحداثيّة**: %v", r.JSON()["kind"])
	}
	var city string
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(c.name, '') FROM coverage_requests r
		LEFT JOIN cities c ON c.id = r.city_id
		WHERE r.user_id = $1::uuid`, u.ID).Scan(&city); err != nil {
		t.Fatalf("قراءةُ المدينة: %v", err)
	}
	if city != "دمشق" {
		t.Fatalf("**المدينةُ تبعت النصَّ لا الإحداثيّة**: %q", city)
	}
}

// ═════════════════ CR-22 … CR-24 — الإلغاءُ والتاريخ ═════════════════

// TestCR22_CR23_CR24_CancelAndHistory **والاشتراكُ يُلغى، والطلبُ
// التاريخيُّ يبقى.**
func TestCR22_CR23_CR24_CancelAndHistory(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-22")
	u := hh.Customer()

	// **اشتراكٌ في دمشق ثمّ إلغاؤه.**
	if r := demand(t, hh, u.Token, damLat, damLng, ""); r.Code != http.StatusOK {
		t.Fatalf("الاشتراك: %d / %s", r.Code, r.Err())
	}
	if r := hh.POST("/api/v1/me/demand/cancel", u.Token,
		map[string]any{"lat": damLat, "lng": damLng}); r.Code != http.StatusOK {
		t.Fatalf("**الإلغاءُ رُدّ**: %d / %s", r.Code, r.Err())
	}
	var active bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = 'service_interest'`,
		u.ID).Scan(&active); err != nil {
		t.Fatalf("قراءةُ السريان: %v", err)
	}
	if active {
		t.Fatal("**اشتراكٌ مُلغىً ما زال سارياً** — فيُخبَر من طلب ألّا يُخبَر")
	}
	// **ولا يُمحى الصفُّ** — والكثافةُ تبقى.
	if rows, _ := demandRows(t, hh, u.ID, "service_interest"); rows != 1 {
		t.Fatalf("**الإلغاءُ محا التاريخ**: %d", rows)
	}

	// **CR-24 · وطلبُ التغطية لا يُلغى بهذا** — واقعةٌ لا اشتراك.
	if r := demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, ""); r.Code != http.StatusOK {
		t.Fatalf("طلبُ التغطية: %d / %s", r.Code, r.Err())
	}
	_ = hh.POST("/api/v1/me/demand/cancel", u.Token,
		map[string]any{"lat": raqqaLat + 0.08, "lng": raqqaLng})
	var covActive bool
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = 'coverage_request'`,
		u.ID).Scan(&covActive); err != nil {
		t.Fatalf("قراءةُ طلب التغطية: %v", err)
	}
	if !covActive {
		t.Fatal("**إلغاءُ الاشتراك أطفأ طلبَ تغطيةٍ تاريخيّاً**")
	}

	// **والعودةُ تُحيي الاشتراك** — ولا يُحبَس أحدٌ على إلغاءٍ قديم.
	if r := demand(t, hh, u.Token, damLat, damLng, ""); r.Code != http.StatusOK {
		t.Fatalf("الإحياء: %d / %s", r.Code, r.Err())
	}
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT active FROM coverage_requests
		 WHERE user_id = $1::uuid AND kind = 'service_interest'`,
		u.ID).Scan(&active); err != nil {
		t.Fatalf("قراءةُ السريان: %v", err)
	}
	if !active {
		t.Fatal("**من عاد فضغطها لم يُحيَ اشتراكُه**")
	}
}

// ═════════════════ CR-25 — لا توسّعَ تلقائيّ ═════════════════

// TestCR25_NoAutomaticExpansion **وكثافةُ الطلب لا تغيّر خدمةً.**
func TestCR25_NoAutomaticExpansion(t *testing.T) {
	hh := New(t)
	z := zoneForDemand(t, hh, "منطقةُ CR-25")
	u := hh.Customer()

	before := zoneSnapshot(t, hh, z.ID)
	cities := phCount(t, hh, `SELECT count(*) FROM cities WHERE active`)

	for i := 0; i < 6; i++ {
		_ = demand(t, hh, u.Token, raqqaLat+0.08, raqqaLng, "")
		_ = demand(t, hh, u.Token, damLat, damLng, "")
	}

	if after := zoneSnapshot(t, hh, z.ID); after != before {
		t.Fatalf("**الطلبُ بدّل المنطقة**: %q ← %q", before, after)
	}
	if got := phCount(t, hh, `SELECT count(*) FROM cities WHERE active`); got != cities {
		t.Fatalf("**الطلبُ فعّل مدينة**: %d ← %d", cities, got)
	}
	// **والعنوانُ ما زال خارجَ التغطية بعد ستّ ضغطات.**
	it := hh.NewItem(900)
	if got := avReason(avOf(t, hh, u.Token, it, raqqaLat+0.08, raqqaLng)); got != "address_outside_coverage" {
		t.Fatalf("**تبدّلت الإتاحةُ بالطلب**: %v", got)
	}
}

// zoneSnapshot بصمةُ منطقةٍ — شكلاً ونصفَ قطرٍ ورسماً وحدّاً أدنى.
func zoneSnapshot(t *testing.T, h *Harness, id string) string {
	t.Helper()
	var s string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT shape || '|' || radius_m || '|' || delivery_fee || '|' ||
		       min_order || '|' || active::text
		FROM delivery_zones WHERE id = $1::uuid`, id).Scan(&s); err != nil {
		t.Fatalf("بصمةُ المنطقة: %v", err)
	}
	return s
}

// ═════════════════ CR-26 · CR-27 — التجميعُ الإداريّ ═════════════════

// TestCR26_CR27_Aggregation **«كم طلباً في دمشق؟» و«أيُّ خليّةٍ أكثر؟»**
func TestCR26_CR27_Aggregation(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-26")
	_, tok := capUser(t, hh, "admin")

	// **ثلاثةُ حساباتٍ تشترك في دمشق.**
	for i := 0; i < 3; i++ {
		u := hh.Customer()
		if r := demand(t, hh, u.Token, damLat, damLng, ""); r.Code != http.StatusOK {
			t.Fatalf("اشتراكٌ %d: %d / %s", i, r.Code, r.Err())
		}
	}

	res := hh.GET("/api/v1/admin/ops-map/coverage-demand/places?kind=service_interest", tok)
	if res.Code != http.StatusOK {
		t.Fatalf("**بابُ الكثافة رُدّ**: %d / %s", res.Code, res.Err())
	}
	body := res.JSON()
	places, _ := body["places"].([]any)
	if len(places) == 0 {
		t.Fatalf("**لا كثافةَ إداريّةً في الردّ**: %v", body)
	}
	found := false
	for _, p := range places {
		m, _ := p.(map[string]any)
		if m["city_name"] == "دمشق" {
			found = true
			if n, _ := m["people"].(float64); n < 3 {
				t.Fatalf("**عددُ الحسابات أقلُّ من الواقع**: %v", m["people"])
			}
			if m["governorate_name"] == "" || m["governorate_name"] == nil {
				t.Fatal("**المحافظةُ لم تُحَلّ في التجميع**")
			}
		}
	}
	if !found {
		t.Fatalf("**دمشقُ غائبةٌ عن التجميع**: %v", places)
	}
}

// ═════════════════ CR-28 — الخصوصيّة ═════════════════

// TestCR28_NoRedundantIdentity **ولا يُكرَّر ما يعرفه الحساب.**
//
// **واسمٌ وهاتفٌ في صفِّ طلبٍ جغرافيٍّ تكرارٌ يتقادم** — **ويُسرَّب في
// تصديرٍ تحليليٍّ لا يقصد الهويّة.**
func TestCR28_NoRedundantIdentity(t *testing.T) {
	hh := New(t)
	rows, err := hh.Pool.Query(ctxBG(), `
		SELECT column_name FROM information_schema.columns
		 WHERE table_name = 'coverage_requests'`)
	if err != nil {
		t.Fatalf("قراءةُ الأعمدة: %v", err)
	}
	defer rows.Close()
	banned := map[string]bool{
		"phone": true, "full_name": true, "name": true,
		"email": true, "password_hash": true, "pin_hash": true,
	}
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			t.Fatalf("عمود: %v", err)
		}
		if banned[col] {
			t.Errorf("**حقلُ هويّةٍ مكرَّرٌ في صفّ الطلب**: %s — "+
				"**و`user_id` يشير إلى الحساب**", col)
		}
	}

	// **ولا بابَ عامٌّ يُعدّد الطلبات.**
	if r := hh.GET("/api/v1/demand", ""); r.Code < 400 {
		t.Errorf("**بابٌ يقرأ الطلبات بلا حساب**: %d", r.Code)
	}
	// **ولا تُقرأ الكثافةُ بلا حساب** — **و٤٠١ لا ٤٠٤**: **ومسارٌ خاطئٌ
	// يردّ ٤٠٤ فيمرّ الفحصُ وهو لا يقيس شيئاً.** (وقع ذلك فعلاً.)
	if r := hh.GET("/api/v1/admin/ops-map/coverage-demand/places", ""); r.Code != http.StatusUnauthorized {
		t.Errorf("**كثافةُ الطلب لا تُحرَس بالتوثيق**: %d", r.Code)
	}
}

// ═════════════════ CR-29 — العقودُ القائمة ═════════════════

// TestCR29_QuoteContractUnchanged **وعقدُ التسعيرة لم يُمَسّ.**
func TestCR29_QuoteContractUnchanged(t *testing.T) {
	hh := New(t)
	zoneForDemand(t, hh, "منطقةُ CR-29")
	u := hh.Customer()
	it := hh.NewItem(900)

	r := hh.POST("/api/v1/public/quote", u.Token, map[string]any{
		"items": []map[string]any{{"menu_item_id": it.ID, "qty": 1}},
		"lat":   raqqaLat, "lng": raqqaLng,
	})
	j := r.JSON()
	for _, k := range []string{
		"subtotal", "delivery_fee", "total", "serviceable",
		"out_of_zone", "zone_closed", "availability",
	} {
		if _, ok := j[k]; !ok {
			t.Errorf("**حقلٌ قائمٌ اختفى**: %s", k)
		}
	}
}
