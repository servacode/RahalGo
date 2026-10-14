package qa

import "testing"

// **زيادةُ المتجر الثاني بدرجات البُعد** (قرارُ المالك ٢٠٢٦-١٠-١٤): قريبٌ بلا
// زيادة · متوسّطٌ نصفُ الأجرة · بعيدٌ أجرةٌ كاملة — والدرجاتُ من اللوحة.
func TestExtraSourceFee_DistanceTiers(t *testing.T) {
	h := New(t)
	u := h.Customer()
	a := h.NewItem(1000)
	b := h.NewItem(1000)
	h.Setting("orders.max_sources", "2")
	h.Setting("orders.extra_source_fee", "0")
	h.Setting("orders.source_proximity_m", "500")
	h.Setting("orders.source_far_m", "2000")
	h.Setting("orders.source_mid_percent", "50")
	h.Setting("orders.source_far_percent", "100")
	// **أجرةُ توصيلٍ ثابتة** — فالنسبةُ تُقاس على رقمٍ لا على صفر.
	h.Setting("delivery.fee", "400")
	h.Setting("delivery.by_distance", "false")

	place := func(it *Item, lat, lng float64) {
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants
			SET location = ST_SetSRID(ST_MakePoint($3, $2), 4326)::geography
			WHERE id = $1::uuid`, it.MerchantID, lat, lng); err != nil {
			t.Fatal(err)
		}
	}
	quote := func() (base, extra float64) {
		r := h.POST("/api/v1/public/quote", u.Token, map[string]any{
			"items": []map[string]any{{"menu_item_id": a.ID, "qty": 1}, {"menu_item_id": b.ID, "qty": 1}},
			"lat":   raqqaLat, "lng": raqqaLng,
		})
		if r.Code >= 400 {
			t.Fatalf("التسعيرة: %s", r)
		}
		j := r.JSON()
		base, _ = j["base_fee"].(float64)
		extra, _ = j["sources_fee"].(float64)
		return
	}

	place(a, raqqaLat, raqqaLng)
	cases := []struct {
		name string
		dLat float64
		want func(base float64) float64
	}{
		{"قريب ~٣٣٠م", 0.003, func(float64) float64 { return 0 }},
		{"متوسّط ~١١٠٠م", 0.010, func(b float64) float64 { return float64(int64(b) * 50 / 100) }},
		{"بعيد ~٣٣٠٠م", 0.030, func(b float64) float64 { return b }},
	}
	for _, c := range cases {
		place(b, raqqaLat+c.dLat, raqqaLng)
		base, extra := quote()
		if base != 400 {
			t.Fatalf("أجرةُ التوصيل %v — والمضبوطةُ 400", base)
		}
		if want := c.want(base); extra != want {
			t.Errorf("%s: الزيادة %v — والمنتظَر %v (الأجرة %v)", c.name, extra, want, base)
		}
	}
}
