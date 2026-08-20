package qa

// **الطبقةُ الحادية عشرة — سوقُ المدينة.**
//
// المعرّفات: `CITY-*` · الوسم: `@api @critical @release`
//
// (قرارُ المالك ٢٠٢٦-٠٨-٢٠: «مو معقول شخصٌ بالشام يطلب من الرقّة… ولا
//
//	زبونٌ بأوّل الشام من مطعمٍ بآخر الشام».)
//
// # وما كان يقع
//
// **الدمشقيُّ يرى متاجرَ الرقّة كلَّها ويملأ سلّتَه** — **ثمّ يُردّ عند
// الإرسال.** وهذا أسوأُ من المنع: **يُقرأ عطباً في التطبيق لا حدّاً
// للخدمة.**
//
// # وما يُقاس
//
// **المدينةُ تُرشِّح السوق · والمسافةُ تُرشِّح المتجر · ولا موضعَ يعني
// لا ترشيح.**

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"testing"
)

// cityAt **يصنع مدينةً بمركزٍ ومدى** — وتُمحى بعد الاختبار.
func (h *Harness) cityAt(name string, lat, lng float64, radiusM int, maxDeliveryM *int) string {
	h.T.Helper()
	var id string
	err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO cities (name, center, radius_m, max_delivery_m, active)
		VALUES ($1, ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, $4, $5, true)
		RETURNING id::text`, uniq(name), lat, lng, radiusM, maxDeliveryM).Scan(&id)
	if err != nil {
		h.T.Fatalf("CITY: تعذّر إنشاءُ مدينة: %v", err)
	}
	// **وسياقُ الاختبارِ يُلغى قبل أن يُنادى التنظيف** — فمُحيَ بسياقٍ
	// مستقلّ. **ومدينةٌ باقيةٌ تجعل «أقربَ مدينة» تختار مدينةَ اختبارٍ
	// ميّت** — وقع، فسقطت CITY-001 على بيانات لا على شيفرة.
	h.T.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM cities WHERE id = $1::uuid`, id)
	})

	// **ومدينتان بمركزٍ واحدٍ لا تقعان في الحقيقة** — لكنّ الهجرةَ
	// زرعت «الرقّة» ثمّ يصنع الاختبارُ رقّتَه فوقَها. **و«أقربُ
	// مدينة» بينهما قرعةٌ لا قاعدة**، فسقطت CITY-001 مرّةً على ذلك.
	//
	// **فتُنوَّم المدينةُ المزروعةُ ما دام الاختبارُ حيّاً** — ولا
	// تُمحى: **محوُ صفٍّ زرعته هجرةٌ يفسد كلَّ اختبارٍ بعده.**
	rows, err := h.Pool.Query(context.Background(), `
		UPDATE cities SET active = false
		 WHERE id <> $1::uuid AND active
		   AND ST_DWithin(center,
		       ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography, radius_m)
		RETURNING id::text`, id, lat, lng)
	if err != nil {
		h.T.Fatalf("CITY: تعذّر تنويمُ المدن المجاورة: %v", err)
	}
	var slept []string
	for rows.Next() {
		var s string
		_ = rows.Scan(&s)
		slept = append(slept, s)
	}
	rows.Close()
	h.T.Cleanup(func() {
		for _, s := range slept {
			_, _ = h.Pool.Exec(context.Background(),
				`UPDATE cities SET active = true WHERE id = $1::uuid`, s)
		}
	})
	return id
}

// placeMerchant **يضع متجرَ الصنف في مدينةٍ وموضع.**
func (h *Harness) placeMerchant(item *Item, cityID string, lat, lng float64) {
	h.T.Helper()
	_, err := h.Pool.Exec(context.Background(), `
		UPDATE merchants
		   SET city_id = $2::uuid,
		       location = ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography
		 WHERE id = $1::uuid`, item.MerchantID, cityID, lat, lng)
	if err != nil {
		h.T.Fatalf("CITY: تعذّر وضعُ المتجر: %v", err)
	}
}

// itemsAt **أصنافُ قسمٍ كما يراها من يقف في هذا الموضع.**
func (h *Harness) itemsAt(sectionID string, lat, lng *float64) []string {
	h.T.Helper()
	path := "/api/v1/public/sections/" + sectionID + "/items"
	if lat != nil && lng != nil {
		path += fmt.Sprintf("?lat=%f&lng=%f", *lat, *lng)
	}
	res := h.GET(path, "")
	if res.Code >= 400 {
		h.T.Fatalf("CITY: نداءُ الأصناف رُدّ: %s", res)
	}
	var body struct {
		Data struct {
			Items []struct {
				Name string `json:"name"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		h.T.Fatalf("CITY: ردٌّ غيرُ مقروء: %v", err)
	}
	out := make([]string, 0, len(body.Data.Items))
	for _, i := range body.Data.Items {
		out = append(out, i.Name)
	}
	return out
}

func ptr(v float64) *float64 { return &v }
func iptr(v int) *int        { return &v }

// TestCITY_001_OtherCityHidden **ومتجرُ مدينةٍ أخرى لا يُرى.**
func TestCITY_001_OtherCityHidden(t *testing.T) {
	h := New(t)
	// **الرقّةُ ودمشقُ** — بإحداثيّاتهما الحقيقيّة.
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	damascus := h.cityAt("دمشق", 33.5138, 36.2765, 25000, nil)

	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)
	_ = damascus

	// **ومن يقف في الرقّة يراه.**
	inRaqqa := h.itemsAt(item.SectionID, ptr(35.9506), ptr(39.0094))
	if len(inRaqqa) == 0 {
		t.Fatal("CITY-001 **الرقّاويُّ لا يرى متجرَ الرقّة** — الترشيحُ يُقصي من يستحقّ")
	}

	// **ومن يقف في دمشق لا يراه.**
	inDamascus := h.itemsAt(item.SectionID, ptr(33.5138), ptr(36.2765))
	if len(inDamascus) > 0 {
		t.Errorf("CITY-001 **الدمشقيُّ يرى متجرَ الرقّة**: %v", inDamascus)
	}
}

// TestCITY_002_NoLocationNoFilter **ولا موضعَ يعني لا ترشيح.**
//
// **وعميلٌ قديمٌ لا يرسل موقعَه يرى ما كان يراه** — **وترشيحٌ يُفرض على
// من لا يعرف عنه يُخفي السوقَ كلَّه عن نسخةٍ قديمة.**
func TestCITY_002_NoLocationNoFilter(t *testing.T) {
	h := New(t)
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)

	if got := h.itemsAt(item.SectionID, nil, nil); len(got) == 0 {
		t.Error("CITY-002 **نداءٌ بلا موضعٍ لا يرى شيئا** — والعميلُ القديمُ يعمى")
	}
}

// TestCITY_010_ReachWithinCity **والمسافةُ تُرشِّح داخل المدينة.**
//
// **وهذه مشكلةُ دمشق**: المزّةُ وجرمانا في مدينةٍ واحدةٍ وبينهما عشرون
// كيلومترا.
func TestCITY_010_ReachWithinCity(t *testing.T) {
	h := New(t)
	// **مدينةٌ واسعةٌ ومداها ستّةُ كيلومترات** — كدمشق.
	big := h.cityAt("مدينة واسعة", 33.5138, 36.2765, 40000, iptr(6000))

	item := h.NewItem(1000)
	// **والمتجرُ في وسطها.**
	h.placeMerchant(item, big, 33.5138, 36.2765)

	// **ومن يقف بجانبه يراه.**
	if got := h.itemsAt(item.SectionID, ptr(33.5150), ptr(36.2780)); len(got) == 0 {
		t.Error("CITY-010 **القريبُ لا يرى المتجرَ الذي يجاوره**")
	}
	// **ومن يقف على بعد عشرين كيلومتراً في المدينة نفسِها لا يراه.**
	far := h.itemsAt(item.SectionID, ptr(33.6900), ptr(36.2765))
	if len(far) > 0 {
		t.Errorf("CITY-010 **البعيدُ يرى متجراً لا يصله**: %v", far)
	}
}

// TestCITY_011_NoReachMeansNoLimit **وبلا مدىً لا حدَّ للمسافة.**
//
// **والرقّةُ لا تحتاج ضبطاً** — تُترك بلا مدىً فتغطّيها كلَّها.
func TestCITY_011_NoReachMeansNoLimit(t *testing.T) {
	h := New(t)
	city := h.cityAt("مدينة بلا مدى", 35.9506, 39.0094, 40000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, city, 35.9506, 39.0094)

	// **وطرفُ المدينةِ يراه** — وافتراضُ المنصّةِ صفرٌ يعني بلا حدّ.
	h.Setting("delivery.default_radius_m", "0")
	if got := h.itemsAt(item.SectionID, ptr(36.1000), ptr(39.0094)); len(got) == 0 {
		t.Error("CITY-011 **مدينةٌ بلا مدىً حجبت متجرَها عن ساكنيها**")
	}
}

// TestCITY_020_MerchantOverridesCity **ومدى المتجر يُلغي مدى مدينته.**
func TestCITY_020_MerchantOverridesCity(t *testing.T) {
	h := New(t)
	city := h.cityAt("مدينة ضيّقة", 33.5138, 36.2765, 40000, iptr(2000))
	item := h.NewItem(1000)
	h.placeMerchant(item, city, 33.5138, 36.2765)

	far := ptr(33.5600) // **نحوَ خمسةِ كيلومترات**
	if got := h.itemsAt(item.SectionID, far, ptr(36.2765)); len(got) > 0 {
		t.Fatalf("CITY-020 مدى المدينة (٢ كم) لم يُطبَّق: %v", got)
	}

	// **ثمّ يُوسَّع مدى المتجر وحدَه.**
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE merchants SET max_delivery_m = 20000 WHERE id = $1::uuid`,
		item.MerchantID); err != nil {
		t.Fatalf("CITY-020 تعذّر الضبط: %v", err)
	}
	if got := h.itemsAt(item.SectionID, far, ptr(36.2765)); len(got) == 0 {
		t.Error("CITY-020 **مدى المتجر لم يُلغِ مدى المدينة**")
	}
}

// TestCITY_030_SearchIsScoped **والبحثُ في سوقِ مدينته.**
//
// **وبحثٌ يجد ما لا يُطلب أسوأُ من بحثٍ لا يجد.**
func TestCITY_030_SearchIsScoped(t *testing.T) {
	h := New(t)
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)

	q := item.Name
	if len([]rune(q)) > 12 {
		q = string([]rune(q)[:12])
	}
	base := "/api/v1/public/search/items?q=" + url.QueryEscape(q)

	near := h.GET(base+"&lat=35.9506&lng=39.0094", "")
	if near.Code >= 400 {
		t.Fatalf("CITY-030 البحثُ رُدّ: %s", near)
	}
	far := h.GET(base+"&lat=33.5138&lng=36.2765", "")
	if far.Code >= 400 {
		t.Fatalf("CITY-030 البحثُ رُدّ: %s", far)
	}

	count := func(res Res) int {
		var b struct {
			Data struct {
				Items []any `json:"items"`
			} `json:"data"`
		}
		_ = json.Unmarshal(res.Body, &b)
		return len(b.Data.Items)
	}
	if count(near) == 0 {
		t.Error("CITY-030 **البحثُ في مدينته لا يجد صنفَه**")
	}
	if count(far) > 0 {
		t.Error("CITY-030 **البحثُ من مدينةٍ أخرى يجد ما لا يُطلب**")
	}
}

// TestCITY_031_MerchantSearchIsScoped **وبحثُ المتاجر كبحثِ الأصناف.**
func TestCITY_031_MerchantSearchIsScoped(t *testing.T) {
	h := New(t)
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)

	var name string
	if err := h.Pool.QueryRow(context.Background(),
		`SELECT name FROM merchants WHERE id = $1::uuid`, item.MerchantID).Scan(&name); err != nil {
		t.Fatalf("CITY-031 تعذّرت قراءةُ اسم المتجر: %v", err)
	}
	base := "/api/v1/public/search?q=" + url.QueryEscape(name)

	hits := func(res Res) int {
		if res.Code >= 400 {
			t.Fatalf("CITY-031 البحثُ رُدّ: %s", res)
		}
		var b struct {
			Data []any `json:"data"`
		}
		_ = json.Unmarshal(res.Body, &b)
		return len(b.Data)
	}
	if hits(h.GET(base+"&lat=35.9506&lng=39.0094", "")) == 0 {
		t.Error("CITY-031 **البحثُ في مدينته لا يجد متجرَها**")
	}
	if hits(h.GET(base+"&lat=33.5138&lng=36.2765", "")) > 0 {
		t.Error("CITY-031 **البحثُ من دمشقَ يجد متجرَ الرقّة**")
	}
}

// TestCITY_040_SectionCountIsScoped **وعدُّ القسمِ يعدُّ ما يصلُه هو.**
//
// **وقسمٌ يعد ثلاثين ثمّ يُفتح فارغاً يُقرأ عطباً في التطبيق.**
func TestCITY_040_SectionCountIsScoped(t *testing.T) {
	h := New(t)
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)

	countAt := func(lat, lng float64) int {
		res := h.GET(fmt.Sprintf(
			"/api/v1/public/sections?lat=%f&lng=%f", lat, lng), "")
		if res.Code >= 400 {
			t.Fatalf("CITY-040 نداءُ الأقسام رُدّ: %s", res)
		}
		var b struct {
			Data struct {
				Sections []struct {
					ID    string `json:"id"`
					Count int    `json:"count"`
				} `json:"sections"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &b); err != nil {
			t.Fatalf("CITY-040 ردٌّ غيرُ مقروء: %v", err)
		}
		for _, sec := range b.Data.Sections {
			if sec.ID == item.SectionID {
				return sec.Count
			}
		}
		return -1
	}

	if n := countAt(35.9506, 39.0094); n < 1 {
		t.Errorf("CITY-040 **قسمُ الرقّاويِّ يعدُّ %d وفيه صنف**", n)
	}
	if n := countAt(33.5138, 36.2765); n > 0 {
		t.Errorf("CITY-040 **قسمٌ يعدُّ %d لمن لا يصله شيء**", n)
	}
}

// TestCITY_050_OffersAreScoped **وعرضٌ على صنفٍ لا يصله لا يُعرض.**
//
// **يضغطه فيدخل سلّتَه ثمّ يُردّ عند الإرسال** — وهو المبدأُ نفسُه الذي
// يحجب الخصمَ على صنفٍ غيرِ متاح.
func TestCITY_050_OffersAreScoped(t *testing.T) {
	h := New(t)
	raqqa := h.cityAt("الرقة", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, raqqa, 35.9506, 39.0094)

	admin := h.NewUser("admin")
	var offerID string
	title := uniq("عرض QA ")
	if err := h.Pool.QueryRow(context.Background(), `
		INSERT INTO offers (kind, title, menu_item_id, discount_percent,
		                    borne_by, active, created_by)
		VALUES ('discount', $1, $2::uuid, 10, 'platform', true, $3::uuid)
		RETURNING id::text`, title, item.ID, admin.ID).Scan(&offerID); err != nil {
		t.Fatalf("CITY-050 تعذّر إنشاءُ عرض: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(context.Background(),
			`DELETE FROM offers WHERE id = $1::uuid`, offerID)
	})

	shown := func(lat, lng float64) bool {
		res := h.GET(fmt.Sprintf(
			"/api/v1/public/offers?lat=%f&lng=%f", lat, lng), "")
		if res.Code >= 400 {
			t.Fatalf("CITY-050 نداءُ العروض رُدّ: %s", res)
		}
		var b struct {
			Data struct {
				Offers []struct {
					ID string `json:"id"`
				} `json:"offers"`
			} `json:"data"`
		}
		if err := json.Unmarshal(res.Body, &b); err != nil {
			t.Fatalf("CITY-050 ردٌّ غيرُ مقروء: %v", err)
		}
		for _, o := range b.Data.Offers {
			if o.ID == offerID {
				return true
			}
		}
		return false
	}

	if !shown(35.9506, 39.0094) {
		t.Error("CITY-050 **عرضُ الرقّة محجوبٌ عن الرقّاويّ**")
	}
	if shown(33.5138, 36.2765) {
		t.Error("CITY-050 **الدمشقيُّ يرى عرضاً على صنفٍ لا يصله**")
	}
}

// TestCITY_060_PublicCitiesShowsActiveOnly **ولا تُعرض مدينةٌ مطفأة.**
//
// **ومن اختار مدينةً ثمّ أُطفئت لا يجدها** — وهو الصواب: **قائمةٌ تعرض
// ما لا يعمل تُقرأ خدمةً موجودة.**
func TestCITY_060_PublicCitiesShowsActiveOnly(t *testing.T) {
	h := New(t)
	on := h.cityAt("مدينة تعمل", 35.9506, 39.0094, 25000, nil)
	off := h.cityAt("مدينة مطفأة", 34.7300, 36.7100, 25000, nil)
	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE cities SET active = false WHERE id = $1::uuid`, off); err != nil {
		t.Fatalf("CITY-060 تعذّر الإطفاء: %v", err)
	}

	res := h.GET("/api/v1/public/cities", "")
	if res.Code != 200 {
		t.Fatalf("CITY-060 نداءُ المدن رُدّ: %s", res)
	}
	var b struct {
		Data struct {
			Cities []struct {
				ID      string  `json:"id"`
				Lat     float64 `json:"lat"`
				Lng     float64 `json:"lng"`
				RadiusM int     `json:"radius_m"`
			} `json:"cities"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &b); err != nil {
		t.Fatalf("CITY-060 ردٌّ غيرُ مقروء: %v", err)
	}
	seenOn, seenOff := false, false
	for _, c := range b.Data.Cities {
		switch c.ID {
		case on:
			seenOn = true
			// **والإحداثيّان يخرجان كما دخلا** — والتطبيقُ يبني عليهما
			// نقطةَ التصفّح، **ومقلوبان يضعان الرقّةَ في المحيط.**
			if int(c.Lat*100) != 3595 || int(c.Lng*100) != 3900 {
				t.Errorf("CITY-060 **إحداثيّا المدينة مقلوبان**: %f, %f", c.Lat, c.Lng)
			}
			if c.RadiusM != 25000 {
				t.Errorf("CITY-060 نصفُ القطر %d لا ٢٥٠٠٠", c.RadiusM)
			}
		case off:
			seenOff = true
		}
	}
	if !seenOn {
		t.Error("CITY-060 **مدينةٌ تعمل ولا تظهر في القائمة**")
	}
	if seenOff {
		t.Error("CITY-060 **مدينةٌ مطفأةٌ تظهر للزبون**")
	}
}

// TestCITY_070_DeleteCityWithMerchantsRefused **ولا تُحذف مدينةٌ فيها
// متاجر.**
//
// **والحذفُ يُفرغ `city_id` فتصير متاجرُها بلا مدينةٍ فتُخفى عن
// الجميع** — بلا خطأٍ ولا سجلّ.
func TestCITY_070_DeleteCityWithMerchantsRefused(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	id := h.cityAt("مدينة فيها متجر", 35.9506, 39.0094, 25000, nil)
	item := h.NewItem(1000)
	h.placeMerchant(item, id, 35.9506, 39.0094)

	res := h.Call("DELETE", "/api/v1/admin/cities/"+id, admin.Token, nil, nil)
	if res.Code != 400 {
		t.Errorf("CITY-070 **حذفُ مدينةٍ فيها متجرٌ رُدَّ بـ%d لا ٤٠٠**", res.Code)
	}
	var n int
	_ = h.Pool.QueryRow(context.Background(),
		`SELECT count(*)::int FROM cities WHERE id = $1::uuid`, id).Scan(&n)
	if n != 1 {
		t.Error("CITY-070 **المدينةُ حُذفت ومتاجرُها فيها**")
	}
}

// TestCITY_071_CitiesAreAdminOnly **ولا يفتح المدنَ إلّا الأدمن.**
//
// **ومن يكتب مدينةً يكتب من يرى السوقَ ومن لا يراه** — وهو حقٌّ أخطرُ
// من كثيرٍ ممّا يُحرَس.
func TestCITY_071_CitiesAreAdminOnly(t *testing.T) {
	h := New(t)
	body := map[string]any{
		"name": uniq("مدينة دخيل "), "lat": 35.0, "lng": 39.0, "radius_m": 10000,
	}
	for _, role := range []string{"customer", "driver", "merchant"} {
		u := h.NewUser(role)
		res := h.Call("POST", "/api/v1/admin/cities", u.Token, body, nil)
		if res.Code != 401 && res.Code != 403 {
			t.Errorf("CITY-071 **%s أنشأ مدينةً** — الردّ %d", role, res.Code)
		}
	}
	res := h.Call("POST", "/api/v1/admin/cities", "", body, nil)
	if res.Code != 401 && res.Code != 403 {
		t.Errorf("CITY-071 **ضيفٌ أنشأ مدينة** — الردّ %d", res.Code)
	}
}
