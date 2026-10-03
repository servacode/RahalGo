package server

// **التحويلُ بمقابلٍ يختاره الموظّف** (قرارُ المالك ٢٠٢٦-١٠-٠٣: «المفروض ما يكون نفس
// الاسم بالضبط، لأنّه ممكن يكون نفسه بتسميةٍ مختلفة»).
//
// **كان التحويلُ يقف على همزة**: «رز مصري» في الطلب و«أرز مصري» في المتجر الجديد ←
// `409 transfer_items_unmatched`. **وهنا**: المقابلُ الصريحُ يمرّ، **وما يخالفه يُردّ**
// (صنفٌ من متجرٍ آخر · غيرُ متاح · بندٌ بلا مقابل · بندٌ ليس من الطلب)، **والشكلُ
// القديمُ بلا مقابلٍ يبقى كما كان.**

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

// transferStore متجرٌ للتحويل بموضعٍ — ويُحذف بعد الاختبار.
func (f *driverFixture) transferStore(t *testing.T, name string, lat, lng float64) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO merchants (name, category_id, commission_percent, location)
		SELECT $2, category_id, 10, ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography
		FROM merchants WHERE id = $1
		RETURNING id`, f.merchantID, name, lat, lng).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
	})
	return id
}

// menuItem صنفٌ في قائمة متجر بسعر شرائه.
func (f *driverFixture) menuItem(t *testing.T, merchantID, name string, price int64, available bool) string {
	t.Helper()
	ctx := context.Background()
	var section string
	if err := f.pool.QueryRow(ctx, `
		SELECT id FROM menu_sections WHERE merchant_id = $1 LIMIT 1`, merchantID).Scan(&section); err != nil {
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'قسم') RETURNING id`,
			merchantID).Scan(&section); err != nil {
			t.Fatalf("تعذّر إنشاءُ قسم: %v", err)
		}
	}
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id,
		                        name, price, merchant_price, available, approved)
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1),
		        $3, $4, $4, $5, true) RETURNING id`,
		merchantID, section, name, price, available).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ صنف: %v", err)
	}
	return id
}

// transferOrder طلبٌ في الطابور بأصنافه — **وسعرُ الزبون ١٢٠٠٠ وسعرُ الشراء ١٠٠٠٠ لكلّ صنف.**
func (f *driverFixture) transferOrder(t *testing.T, names ...string) (string, []string) {
	t.Helper()
	ctx := context.Background()
	orderID := f.dispatchingOrder(t, 20_000, 5_000)
	t.Cleanup(func() {
		_, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, orderID)
	})
	ids := []string{}
	for _, n := range names {
		var id string
		if err := f.pool.QueryRow(ctx, `
			INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, merchant_id)
			VALUES ($1, $2, 12000, 10000, 1, $3) RETURNING id`,
			orderID, n, f.merchantID).Scan(&id); err != nil {
			t.Fatalf("تعذّر إنشاءُ بند: %v", err)
		}
		ids = append(ids, id)
	}
	return orderID, ids
}

func (f *driverFixture) transfer(t *testing.T, ops, orderID, body string) (int, string) {
	t.Helper()
	w := f.call(f.srv.handleTransferOrder, http.MethodPost,
		"/admin/orders/"+orderID+"/transfer", orderID, ops, []string{"ops"}, body)
	// **والخطأُ في أعلى الجسم** — حيث يقرؤه العميل (`ApiError`).
	var out struct {
		Error struct {
			Code    string `json:"code"`
			Details struct {
				Items []string `json:"items"`
			} `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	if out.Error.Code == "transfer_items_unmatched" && len(out.Error.Details.Items) == 0 {
		t.Fatalf("«لا يطابق» بلا أسماء: %s", w.Body.String())
	}
	return w.Code, out.Error.Code
}

func (f *driverFixture) orderMerchant(t *testing.T, orderID string) string {
	t.Helper()
	var m string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT merchant_id::text FROM orders WHERE id = $1`, orderID).Scan(&m); err != nil {
		t.Fatal(err)
	}
	return m
}

// **الاسمان مختلفان والمقابلُ صريح ← يمرّ** — وكان يُردّ ٤٠٩ قبل هذا.
func TestTransfer_ExplicitMappingAcrossDifferentNames(t *testing.T) {
	f := newDriverFixture(t, 0)
	ops := f.armOps(t)
	ctx := context.Background()
	other := f.transferStore(t, "متجر الأرز", 35.95, 39.01)
	rice := f.menuItem(t, other, "أرز مصري", 11_000, true)
	orderID, items := f.transferOrder(t, "رز مصري")

	// **بلا مقابلٍ يبقى الحرفُ حَكَماً** — السلوكُ القديمُ لم يتبدّل.
	if code, ec := f.transfer(t, ops, orderID,
		`{"merchant_id":"`+other+`","note":"اعتذر"}`); code != http.StatusConflict ||
		ec != "transfer_items_unmatched" {
		t.Fatalf("بلا مقابل: %d %s — والمتوقّعُ ٤٠٩ transfer_items_unmatched", code, ec)
	}

	code, ec := f.transfer(t, ops, orderID, `{"merchant_id":"`+other+`","note":"اعتذر",
		"items":[{"order_item_id":"`+items[0]+`","menu_item_id":"`+rice+`"}]}`)
	if code != http.StatusOK {
		t.Fatalf("التحويلُ بمقابلٍ صريح ردّ %d %s — والمتوقّع ٢٠٠", code, ec)
	}
	var menuID, merchant string
	var unit, buy int64
	if err := f.pool.QueryRow(ctx, `
		SELECT menu_item_id::text, merchant_id::text, unit_price, merchant_price
		FROM order_items WHERE id = $1`, items[0]).Scan(&menuID, &merchant, &unit, &buy); err != nil {
		t.Fatal(err)
	}
	if menuID != rice || merchant != other {
		t.Fatalf("البندُ لم يُنقل: صنف %s متجر %s", menuID, merchant)
	}
	if buy != 11_000 {
		t.Fatalf("سعرُ الشراء %d — والمتوقّعُ سعرُ الصنف المختار ١١٠٠٠", buy)
	}
	if unit != 12_000 {
		t.Fatalf("سعرُ الزبون تبدّل إلى %d — ولا يُمسّ", unit)
	}
	if m := f.orderMerchant(t, orderID); m != other {
		t.Fatalf("متجرُ الطلب %s — والمتوقّعُ الجديد", m)
	}
}

// **الشكلُ القديمُ بالاسم نفسِه يعمل كما كان.**
func TestTransfer_ExactNameStillWorks(t *testing.T) {
	f := newDriverFixture(t, 0)
	ops := f.armOps(t)
	other := f.transferStore(t, "متجر الاسم نفسه", 35.95, 39.01)
	f.menuItem(t, other, "رز مصري", 9_500, true)
	orderID, items := f.transferOrder(t, "رز مصري")
	if code, ec := f.transfer(t, ops, orderID,
		`{"merchant_id":"`+other+`","note":"مغلق"}`); code != http.StatusOK {
		t.Fatalf("التحويلُ بالاسم نفسِه ردّ %d %s", code, ec)
	}
	var buy int64
	_ = f.pool.QueryRow(context.Background(),
		`SELECT merchant_price FROM order_items WHERE id = $1`, items[0]).Scan(&buy)
	if buy != 9_500 {
		t.Fatalf("سعرُ الشراء %d — والمتوقّع ٩٥٠٠", buy)
	}
}

// **وما يخالف المقابلَ يُردّ — والطلبُ لا يُمسّ.**
func TestTransfer_MappingValidation(t *testing.T) {
	f := newDriverFixture(t, 0)
	ops := f.armOps(t)
	target := f.transferStore(t, "المتجر الهدف", 35.95, 39.01)
	third := f.transferStore(t, "متجر ثالث", 35.96, 39.02)
	rice := f.menuItem(t, target, "أرز مصري", 11_000, true)
	sugar := f.menuItem(t, target, "سكر", 5_000, true)
	gone := f.menuItem(t, target, "سكر ناعم", 5_000, false)
	foreign := f.menuItem(t, third, "سكر", 5_000, true)
	orderID, items := f.transferOrder(t, "رز مصري", "سكر")
	// **وبندٌ من طلبٍ آخر** — معرّفٌ صحيحُ الشكل ليس من هذا الطلب.
	_, strangers := f.transferOrder(t, "شاي")

	pick := func(oi, mi string) string {
		return `{"order_item_id":"` + oi + `","menu_item_id":"` + mi + `"}`
	}
	body := func(picks ...string) string {
		s := `{"merchant_id":"` + target + `","note":"اعتذر","items":[`
		for i, p := range picks {
			if i > 0 {
				s += ","
			}
			s += p
		}
		return s + `]}`
	}
	cases := []struct {
		name string
		body string
		code int
		ec   string
	}{
		{"صنفٌ من متجرٍ آخر", body(pick(items[0], rice), pick(items[1], foreign)),
			http.StatusUnprocessableEntity, "transfer_item_wrong_store"},
		{"صنفٌ غيرُ متاح", body(pick(items[0], rice), pick(items[1], gone)),
			http.StatusConflict, "transfer_item_unavailable"},
		{"بندٌ بلا مقابل", body(pick(items[0], rice)),
			http.StatusConflict, "transfer_items_unmatched"},
		{"بندٌ مكرّر", body(pick(items[0], rice), pick(items[0], rice), pick(items[1], sugar)),
			http.StatusBadRequest, "transfer_mapping_invalid"},
		{"بندٌ ليس من الطلب", body(pick(items[0], rice), pick(items[1], sugar), pick(strangers[0], sugar)),
			http.StatusBadRequest, "transfer_mapping_invalid"},
		{"معرّفٌ فاسد", body(pick(items[0], "not-a-uuid"), pick(items[1], sugar)),
			http.StatusBadRequest, "transfer_mapping_invalid"},
	}
	for _, c := range cases {
		code, ec := f.transfer(t, ops, orderID, c.body)
		if code != c.code || ec != c.ec {
			t.Errorf("%s: %d %s — والمتوقّع %d %s", c.name, code, ec, c.code, c.ec)
		}
		if m := f.orderMerchant(t, orderID); m != f.merchantID {
			t.Fatalf("%s: تبدّل متجرُ الطلب رغم الرفض", c.name)
		}
	}
	// **والمقابلُ الكاملُ الصحيحُ يمرّ بعدها.**
	if code, ec := f.transfer(t, ops, orderID, body(pick(items[0], rice), pick(items[1], sugar))); code != http.StatusOK {
		t.Fatalf("المقابلُ الصحيح ردّ %d %s", code, ec)
	}
}
