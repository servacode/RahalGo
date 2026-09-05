package qa

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **طبقتا المتاجر والطلبات — `MAP-2` · البند ٤٧**
// ══════════════════════════════════════════════════════════════════════

type mapMerchant struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	OpenNow     bool    `json:"open_now"`
	ActiveOrder int     `json:"active_orders"`
	Rep         *string `json:"rep"`
}

type mapOrder struct {
	ID       string   `json:"id"`
	Number   int64    `json:"number"`
	Status   string   `json:"status"`
	DropLat  float64  `json:"drop_lat"`
	DropLng  float64  `json:"drop_lng"`
	Merchant *string  `json:"merchant"`
	PickLat  *float64 `json:"pick_lat"`
	Driver   *string  `json:"driver"`
	Total    *int64   `json:"total"`
}

// placeMapOrder طلبٌ حيٌّ للخريطة — **بالباب العامّ لا بحقنٍ في القاعدة.**
func placeMapOrder(t *testing.T, h *Harness, cust *User, item *Item) string {
	t.Helper()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	id, _ := made.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("طلبٌ بلا معرّف: %s", made)
	}
	return id
}

func merchantsOf(t *testing.T, h *Harness, tok, query string) []mapMerchant {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/merchants"+query, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("طبقةُ المتاجر — %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Merchants []mapMerchant `json:"merchants"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data.Merchants
}

func ordersOf(t *testing.T, h *Harness, tok, query string) []mapOrder {
	t.Helper()
	res := h.GET("/api/v1/admin/ops-map/orders"+query, tok)
	if res.Code != http.StatusOK {
		t.Fatalf("طبقةُ الطلبات — %d · %s", res.Code, string(res.Body))
	}
	var env struct {
		Data struct {
			Orders []mapOrder `json:"orders"`
		} `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data.Orders
}

// TestOpsMap_MerchantMarkerCarriesOperationalSummary **البند ٩.**
func TestOpsMap_MerchantMarkerCarriesOperationalSummary(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	tok := h.NewUser("admin").Token

	list := merchantsOf(t, h, tok, "")
	var found *mapMerchant
	for i := range list {
		if list[i].ID == item.MerchantID {
			found = &list[i]
		}
	}
	if found == nil {
		t.Fatalf("متجرُ الاختبار %s غائبٌ عن الطبقة (%d متجراً)", item.MerchantID, len(list))
	}
	// **ودبّوسٌ حقيقيٌّ لا صفر** — والصفرُ خليجُ غينيا لا الرقّة.
	if found.Lat == 0 || found.Lng == 0 {
		t.Errorf("موضعٌ صفريّ: %+v", found)
	}
	if found.Name == "" {
		t.Error("متجرٌ بلا اسم")
	}
}

// TestOpsMap_MerchantBBoxExcludesFarAway **البندان ٣١ و٤٣.**
func TestOpsMap_MerchantBBoxExcludesFarAway(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	tok := h.NewUser("admin").Token

	// **مستطيلٌ في المحيط الأطلسيّ** — ولا متجرَ فيه.
	far := merchantsOf(t, h, tok, "?bbox=-30,-10,-29,-9")
	for _, x := range far {
		if x.ID == item.MerchantID {
			t.Fatal("متجرُ الرقّة ظهر في مستطيلٍ في الأطلسيّ — المشهدُ لا يقيّد")
		}
	}
	// **ومستطيلٌ يحويه يُظهره.**
	near := merchantsOf(t, h, tok, "?bbox=38.5,35.5,39.5,36.5")
	hit := false
	for _, x := range near {
		if x.ID == item.MerchantID {
			hit = true
		}
	}
	if !hit {
		t.Fatal("متجرُ الرقّة غاب عن مستطيلٍ يحويه")
	}
}

// TestOpsMap_ActiveOrderMarkerAndRelations **البندان ١٠ و١١.**
func TestOpsMap_ActiveOrderMarkerAndRelations(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	cust := h.Customer()
	orderID := placeMapOrder(t, h, cust, item)
	tok := h.NewUser("admin").Token

	list := ordersOf(t, h, tok, "")
	var found *mapOrder
	for i := range list {
		if list[i].ID == orderID {
			found = &list[i]
		}
	}
	if found == nil {
		t.Fatalf("الطلبُ %s غائبٌ عن الطبقة (%d طلباً)", orderID, len(list))
	}
	if found.DropLat == 0 || found.DropLng == 0 {
		t.Errorf("نقطةُ تسليمٍ صفريّة: %+v", found)
	}
	// **والربطُ الجغرافيُّ يحتاج نقطةَ الاستلام** (البند ١١).
	if found.PickLat == nil {
		t.Error("لا نقطةَ استلامٍ — ولا خطَّ ربطٍ يُرسَم")
	}
	if found.Merchant == nil || *found.Merchant == "" {
		t.Error("طلبٌ بلا اسمِ متجر")
	}
	// **والمالُ يظهر للأدمن** — وسيُحجب عن غيره في اختبارٍ آخر.
	if found.Total == nil {
		t.Error("الأدمنُ لا يرى الإجمالَ")
	}
}

// TestOpsMap_ClosedOrderIsNotActive **البند ١٠** — النشطُ افتراضاً.
func TestOpsMap_ClosedOrderIsNotActive(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	cust := h.Customer()
	orderID := placeMapOrder(t, h, cust, item)
	tok := h.NewUser("admin").Token

	if _, err := h.Pool.Exec(context.Background(),
		`UPDATE orders SET closed_at = now() WHERE id = $1::uuid`, orderID); err != nil {
		t.Fatal(err)
	}
	for _, o := range ordersOf(t, h, tok, "") {
		if o.ID == orderID {
			t.Fatal("طلبٌ مغلقٌ ظهر في الطبقة النشطة")
		}
	}
}

// TestOpsMap_PrivacyMoneyHiddenWithoutPermission **البندان ٦ و٣٣.**
//
// **والعملياتُ توزّع ولا تقرأ الأرقامَ الماليّة.**
func TestOpsMap_PrivacyMoneyHiddenWithoutPermission(t *testing.T) {
	h := New(t)
	item := h.NewItem(5000)
	cust := h.Customer()
	_ = placeMapOrder(t, h, cust, item)

	ops := h.NewUser("ops").Token
	for _, o := range ordersOf(t, h, ops, "") {
		if o.Total != nil {
			t.Fatalf("العملياتُ ترى إجماليَّ الطلب %d", *o.Total)
		}
	}
	// **واسمُ المندوب يظهر للعمليات** (لها `VIEW_REP_ACTIVITY`)
	// **ولا يظهر للماليّة.**
	fin := h.NewUser("finance").Token
	for _, x := range merchantsOf(t, h, fin, "") {
		if x.Rep != nil {
			t.Fatalf("الماليّةُ ترى مندوبَ المتجر %q", *x.Rep)
		}
	}
}

// TestOpsMap_LayerPermissionsAreEnforcedPerLayer **البند ٣٢.**
func TestOpsMap_LayerPermissionsAreEnforcedPerLayer(t *testing.T) {
	h := New(t)
	// **والماليّةُ لا ترى مواضعَ السائقين** — طبقةٌ بابُها مغلق.
	if res := h.GET("/api/v1/admin/ops-map/drivers",
		h.NewUser("finance").Token); res.Code != http.StatusForbidden {
		t.Errorf("الماليّةُ فتحت طبقةَ السائقين — %d", res.Code)
	}
	// **والعملياتُ تراها.**
	if res := h.GET("/api/v1/admin/ops-map/drivers",
		h.NewUser("ops").Token); res.Code != http.StatusOK {
		t.Errorf("العملياتُ حُجبت عن طبقة السائقين — %d", res.Code)
	}
}
