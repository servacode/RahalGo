package server

// **مرشّحو التحويل مرتّبين بما يقدّمون ثمّ بقربهم** (قرارُ المالك ٢٠٢٦-١٠-٠٣).
//
// **والقاعدةُ مشتركةٌ مع اختباراتٍ أخرى** — ففيها متاجرُ غيرُ متاجرنا. **فلا يُقاس
// الترتيبُ المطلق بل ترتيبُ متاجرنا فيما بينها.**

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

type candidatesResp struct {
	DistanceFrom string `json:"distance_from"`
	Items        []struct {
		OrderItemID string `json:"order_item_id"`
		Name        string `json:"name"`
	} `json:"items"`
	Stores []struct {
		MerchantID string   `json:"merchant_id"`
		Matched    int      `json:"matched"`
		Total      int      `json:"total"`
		DistanceM  *float64 `json:"distance_m"`
		Items      []struct {
			OrderItemID string `json:"order_item_id"`
			Match       *struct {
				MenuItemID    string `json:"menu_item_id"`
				MerchantPrice int64  `json:"merchant_price"`
			} `json:"match"`
		} `json:"items"`
		Menu []struct {
			MenuItemID string `json:"menu_item_id"`
		} `json:"menu"`
	} `json:"stores"`
}

func (f *driverFixture) candidates(t *testing.T, ops, orderID, query string) candidatesResp {
	t.Helper()
	w := f.call(f.srv.handleTransferCandidates, http.MethodGet,
		"/admin/orders/"+orderID+"/transfer-candidates"+query, orderID, ops, []string{"ops"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("المرشّحون ردّوا %d: %s", w.Code, w.Body.String())
	}
	var out struct {
		Data candidatesResp `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out.Data
}

// positions مواضعُ متاجرنا في القائمة — و`-1` لمن غاب.
func (r candidatesResp) positions(ids ...string) []int {
	out := make([]int, len(ids))
	for k, id := range ids {
		out[k] = -1
		for i, s := range r.Stores {
			if s.MerchantID == id {
				out[k] = i
			}
		}
	}
	return out
}

func TestTransferCandidates_RankedByCoverageThenDistance(t *testing.T) {
	f := newDriverFixture(t, 1)
	ops := f.armOps(t)
	ctx := context.Background()
	// **المتجرُ الحاليّ في الرقّة** — ونقطةُ القياس منه ما دام الطلبُ بلا سائق.
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET location =
		ST_SetSRID(ST_MakePoint(39.00, 35.95), 4326)::geography WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	near := f.transferStore(t, "قريب يقدّم واحداً", 35.951, 39.00) // ~١١٠م
	mid := f.transferStore(t, "وسط يقدّم الكلّ", 35.96, 39.00)     // ~١.١كم
	far := f.transferStore(t, "بعيد يقدّم الكلّ", 36.05, 39.00)    // ~١١كم
	closed := f.transferStore(t, "مغلق طارئاً", 35.9505, 39.00)
	if _, err := f.pool.Exec(ctx, `UPDATE merchants SET emergency_closed = true WHERE id = $1`, closed); err != nil {
		t.Fatal(err)
	}

	f.menuItem(t, near, "أرز مصري", 9_000, true)
	f.menuItem(t, near, "بطاطا", 1_000, true)

	f.menuItem(t, mid, "رز مصري", 9_000, true)
	f.menuItem(t, mid, "الزيت دوار الشمس", 9_000, true)
	f.menuItem(t, mid, "سكر ناعم", 9_000, true)

	farRice := f.menuItem(t, far, "أرز مصري", 10_200, true)
	f.menuItem(t, far, "رز مصري حبة طويلة", 10_000, true)
	f.menuItem(t, far, "زيت دوار الشمس", 9_000, true)
	f.menuItem(t, far, "سكر", 9_000, true)
	farGone := f.menuItem(t, far, "شاي", 9_000, false)

	f.menuItem(t, closed, "رز مصري", 9_000, true)
	f.menuItem(t, closed, "زيت دوار الشمس", 9_000, true)
	f.menuItem(t, closed, "سكر", 9_000, true)

	orderID, items := f.transferOrder(t, "رز مصري", "زيت دوار الشمس", "سكر")

	r := f.candidates(t, ops, orderID, "")
	if r.DistanceFrom != "store" {
		t.Fatalf("نقطةُ القياس %q — والمتوقّعُ المتجرُ الحاليّ", r.DistanceFrom)
	}
	if len(r.Items) != 3 {
		t.Fatalf("أصنافُ الطلب %d — والمتوقّع ٣", len(r.Items))
	}
	p := r.positions(mid, far, near, closed, f.merchantID)
	if p[3] != -1 || p[4] != -1 {
		t.Fatalf("المغلقُ طارئاً أو متجرُ الطلب نفسُه في القائمة: %v", p)
	}
	if p[0] < 0 || p[1] < 0 || p[2] < 0 || !(p[0] < p[1] && p[1] < p[2]) {
		t.Fatalf("الترتيب %v — والمتوقّعُ: الوسط (٣ من ٣ أقرب) ثمّ البعيد (٣ من ٣) ثمّ القريب (١ من ٣)", p)
	}
	st := r.Stores[p[1]]
	if st.Matched != 3 || st.Total != 3 || st.DistanceM == nil {
		t.Fatalf("البعيد: %d من %d، مسافة %v", st.Matched, st.Total, st.DistanceM)
	}
	if n := r.Stores[p[2]].Matched; n != 1 {
		t.Fatalf("القريب يقدّم %d — والمتوقّعُ واحد («أرز مصري»)", n)
	}
	// **والمقابلُ المقترحُ لـ«رز مصري» في البعيد «أرز مصري»** — تامٌّ بعد التسوية،
	// **ويسبق «رز مصري حبة طويلة» ولو كان سعرُه أقرب.**
	for _, it := range st.Items {
		if it.OrderItemID == items[0] {
			if it.Match == nil || it.Match.MenuItemID != farRice {
				t.Fatalf("مقابلُ «رز مصري» في البعيد %+v — والمتوقّع «أرز مصري»", it.Match)
			}
		}
	}

	// **ومتجرٌ بعينه يأتي بقائمته المتاحة** — للاختيار باليد.
	one := f.candidates(t, ops, orderID, "?merchant_id="+far)
	if len(one.Stores) != 1 || one.Stores[0].MerchantID != far {
		t.Fatalf("طلبُ متجرٍ بعينه ردّ %d متجراً", len(one.Stores))
	}
	if n := len(one.Stores[0].Menu); n != 4 {
		t.Fatalf("قائمةُ البعيد %d صنفاً — والمتوقّعُ ٤ متاحة", n)
	}
	for _, m := range one.Stores[0].Menu {
		if m.MenuItemID == farGone {
			t.Fatal("صنفٌ غيرُ متاحٍ في قائمة الاختيار")
		}
	}

	// **ومع السائق تُقاس المسافةُ من موضعه** — وهو عند البعيد، فيسبق الوسط.
	d := f.drivers[0]
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'assigned', driver_id = $2 WHERE id = $1`,
		orderID, d); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE users SET last_location =
		ST_SetSRID(ST_MakePoint(39.00, 36.049), 4326)::geography, last_location_at = now() WHERE id = $1`, d); err != nil {
		t.Fatal(err)
	}
	r = f.candidates(t, ops, orderID, "")
	if r.DistanceFrom != "driver" {
		t.Fatalf("نقطةُ القياس %q — والمتوقّعُ السائق", r.DistanceFrom)
	}
	if p := r.positions(far, mid); !(p[0] >= 0 && p[0] < p[1]) {
		t.Fatalf("الترتيبُ من موضع السائق %v — والمتوقّعُ البعيدُ أوّلاً", p)
	}
}
