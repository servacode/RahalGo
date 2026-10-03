package server

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» في يد السائق** (٢٠٢٦-١٠-٠٢)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت `dropoff_known` و`parcel_note` و`fee_payer` في القاعدة ولا تبلغ
// `/driver/orders`** — والتطبيقُ يفترض «النقطةُ معروفة» حين يغيب الحقل،
// فيُقاد السائقُ بعد الاستلام إلى المتجر نفسِه على أنّه بابُ المستلِم.

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/routing"
)

// fakeRoutes **محرّكٌ يردّ خطّاً مستقيماً** — ليُعرف أنّ الرفضَ من الباب لا من غياب المحرّك.
type fakeRoutes struct{ calls int }

func (f *fakeRoutes) Enabled() bool { return true }

func (f *fakeRoutes) Route(ctx context.Context, from, to routing.Point) (*routing.Route, error) {
	f.calls++
	return &routing.Route{DistanceM: 500, DurationS: 60, Geometry: []routing.Point{from, to}}, nil
}

func (f *fakeRoutes) RouteSet(ctx context.Context, from, to routing.Point) ([]*routing.Route, error) {
	r, err := f.Route(ctx, from, to)
	return []*routing.Route{r}, err
}

// deliveryOrderAt **توصيلةٌ بيد السائق في حالٍ بعينها** — والنقطةُ معروفةٌ أو لا.
func (f *driverFixture) deliveryOrderAt(t *testing.T, status, driverID string, known bool, payer string) string {
	t.Helper()
	ctx := context.Background()
	// **والمتجرُ له موضع** — وإلّا لا نقطةَ استلامٍ يُقاس منها.
	if _, err := f.pool.Exec(ctx, `
		UPDATE merchants SET location = ST_SetSRID(ST_MakePoint(39.0100, 35.9500), 4326)::geography
		WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	cash := int64(0)
	if payer != "merchant" {
		cash = 5_000
	}
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (kind, merchant_id, status, address_text, dropoff, dropoff_known,
			recipient_name, recipient_phone, parcel_note, fee_payer,
			payment_method, subtotal, delivery_fee, driver_fee, total, wallet_paid, cash_due,
			driver_id, accepted_at,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ('merchant_delivery', $1, $2, 'حيّ الفردوس قرب الجامع',
			ST_SetSRID(ST_MakePoint(39.0100, 35.9500), 4326)::geography, $3,
			'أبو سامر', '0999000111', 'كيس طعام ساخن', $4,
			'cash', 0, 5000, 4500, 5000, 0, $5,
			$6, now(), `+qaSnapSQL()+`)
		RETURNING id`, f.merchantID, status, known, payer, cash, driverID).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ التوصيلة: %v", err)
	}
	t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id) })
	return id
}

// TestDriverOrders_MerchantDeliveryFields **ما يحتاجه حاملُ التوصيلة يبلغه.**
func TestDriverOrders_MerchantDeliveryFields(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	id := f.deliveryOrderAt(t, "assigned", d, false, "merchant_cash")

	w := f.call(f.srv.handleDriverOrders, http.MethodGet, "/driver/orders", "", d, []string{"driver"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("طلباتُه ردّت %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	for _, o := range body.Data {
		if o["id"] == id {
			row = o
		}
	}
	if row == nil {
		t.Fatalf("التوصيلةُ ليست في طلباته: %s", w.Body.String())
	}
	if v, ok := row["dropoff_known"]; !ok || v != false {
		t.Errorf("dropoff_known = %v (موجود=%v) — **غيابُه يُقرأ «معروفة» فيُقاد إلى المتجر**", v, ok)
	}
	if row["parcel_note"] != "كيس طعام ساخن" {
		t.Errorf("parcel_note = %v — **لا يعرف ما يحمل**", row["parcel_note"])
	}
	if row["fee_payer"] != "merchant_cash" {
		t.Errorf("fee_payer = %v — **لا يعرف أنّه يقبض الأجرةَ من المتجر**", row["fee_payer"])
	}
	// **والمشوارُ إلى نقطةٍ مجهولةٍ مجهول** — لا صفر.
	if lm, _ := row["leg_m"].(float64); lm >= 0 {
		t.Errorf("leg_m = %v — **نقطةٌ مجهولةٌ لا طولَ إليها**", lm)
	}
}

// TestDriverRoute_NoRouteToUnknownDropoff **بعد الاستلام لا طريقَ إلى نقطةٍ مكتوبةٍ مكانَ المتجر.**
func TestDriverRoute_NoRouteToUnknownDropoff(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	eng := &fakeRoutes{}
	f.srv.route = eng

	route := func(id string) map[string]any {
		w := f.call(f.srv.handleDriverOrderRoute, http.MethodGet, "/driver/orders/"+id+"/route", id, d,
			[]string{"driver"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("المسارُ ردّ %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		return body.Data
	}

	unknown := f.deliveryOrderAt(t, "on_the_way", d, false, "recipient")
	if r := route(unknown); r["available"] != false {
		t.Errorf("نقطةٌ مجهولة: available=%v — **يُقاد إلى المتجر على أنّه بابُ المستلِم**", r["available"])
	}
	// **وقبل الاستلام الطريقُ إلى المتجر قائم** — والمجهولُ هو الباب لا المتجر.
	if _, err := f.pool.Exec(context.Background(), `UPDATE orders SET status = 'assigned' WHERE id = $1`, unknown); err != nil {
		t.Fatal(err)
	}
	if r := route(unknown); r["available"] != true {
		t.Errorf("قبل الاستلام: available=%v — **الطريقُ إلى المتجر لا يُحجب**", r["available"])
	}
	// **والمعروفةُ تبقى كما كانت.**
	known := f.deliveryOrderAt(t, "on_the_way", d, true, "recipient")
	if r := route(known); r["available"] != true {
		t.Errorf("نقطةٌ معروفة: available=%v", r["available"])
	}
}

// TestDriverRoute_CustomAfterPurchase **الطلبُ الخاصّ بعد الشراء يُرسم طريقُه إلى الزبون** —
// (بلاغُ المالك ٢٠٢٦-١٠-٠٣: «الطريقُ على الزبون مستقيم»). **كانت نقطةُ المتجر الغائبةُ تُقرأ
// رقماً فيسقط المسحُ بـ٥٠٠** والشاشةُ ترسم مستقيماً. **وقبل الشراء لا طريق** — لا متجرَ يُقصد.
func TestDriverRoute_CustomAfterPurchase(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.route = &fakeRoutes{}
	ord := f.customOrderAt(t, pdLat, pdLng, "dispatching")
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET status = 'on_the_way', driver_id = $2 WHERE id = $1`, ord, d); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE users SET last_location = ST_SetSRID(ST_MakePoint(39.01, 35.95), 4326)::geography WHERE id = $1`, d); err != nil {
		t.Fatal(err)
	}
	call := func() map[string]any {
		w := f.call(f.srv.handleDriverOrderRoute, http.MethodGet, "/driver/orders/"+ord+"/route", ord, d,
			[]string{"driver"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("المسارُ ردّ %d: %s", w.Code, w.Body.String())
		}
		var body struct {
			Data map[string]any `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		return body.Data
	}
	if r := call(); r["available"] != true {
		t.Errorf("بعد الشراء: available=%v — **والطريقُ إلى الزبون يُرسم**", r["available"])
	}
	if _, err := f.pool.Exec(context.Background(), `UPDATE orders SET status = 'assigned' WHERE id = $1`, ord); err != nil {
		t.Fatal(err)
	}
	if r := call(); r["available"] != false {
		t.Errorf("قبل الشراء: available=%v — **ولا متجرَ يُقصد**", r["available"])
	}
}
