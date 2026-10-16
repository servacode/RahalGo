package server

// ══════════════════════════════════════════════════════════════════════
// **موقعُ الزبون المباشر — من يكتب ومن يرى ومتى يُمحى** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
// **خمسةُ أحكامٍ تُقاس على القاعدة الحقيقيّة**: صاحبُ الطلب وحدَه يكتب ·
// والطلبُ في طريقه وحدَه · والسائقُ المُسنَدُ يراه حديثاً ولا يراه شائخاً ·
// والتسليمُ يمحوه · وسائقٌ آخرُ والمتجرُ لا يرونه أبداً.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// liveOrder **طلبٌ بحالٍ وسائقٍ بعينهما** — ويردّ زبونَه معه.
func (f *driverFixture) liveOrder(t *testing.T, status string, driverID *string) (string, string) {
	t.Helper()
	customer := testdb.NewUser(t, f.pool, "customer")
	var id string
	if err := f.pool.QueryRow(context.Background(), `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due,
			snap_merchant_commission_percent, snap_rep_commission_percent,
			snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, $3, $4, 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', 20000, 5000, 25000, 0, 25000,
			`+qaSnapSQL()+`)
		RETURNING id`, customer, f.merchantID, driverID, status).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاء طلب: %v", err)
	}
	return id, customer
}

// liveCall **ينادي معالجاً بمعرّف الطلب وجلسةِ صاحبها** — كما يناديه الموجّه.
func liveCall(h http.HandlerFunc, method, orderID, userID, role, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, "/api/v1/orders/"+orderID+"/live-location", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, userID)
	ctx = context.WithValue(ctx, ctxRoles, []string{role})
	w := httptest.NewRecorder()
	h(w, req.WithContext(ctx))
	return w
}

// farFromPin **نقطةٌ على ثلاثمئة مترٍ ونيّفٍ شمالَ الدبّوس** — فوق الحدّ الافتراضيّ (١٥٠م).
const farFromPin = `{"lat": 35.9558, "lng": 39.0079, "accuracy_m": 12}`

// nearPin **نقطةٌ على نحو خمسين متراً** — دون الحدّ.
const nearPin = `{"lat": 35.95325, "lng": 39.0079, "accuracy_m": 8}`

func (f *driverFixture) liveRows(t *testing.T, orderID string) int {
	t.Helper()
	var n int
	if err := f.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM order_live_locations WHERE order_id = $1`, orderID).Scan(&n); err != nil {
		t.Fatalf("قراءةُ النقطة: %v", err)
	}
	return n
}

// driverOrdersOf **ما تردّه `/driver/orders` لهذا السائق** — حمولةً خامّة.
func (f *driverFixture) driverOrdersOf(t *testing.T, driverID string) []map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/driver/orders", nil)
	ctx := context.WithValue(req.Context(), ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDriverOrders(w, req.WithContext(ctx))
	if w.Code != http.StatusOK {
		t.Fatalf("/driver/orders: %d %s", w.Code, w.Body.String())
	}
	var out struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatalf("ردٌّ غيرُ صالح: %v", err)
	}
	return out.Data
}

func liveOf(list []map[string]any, orderID string) (map[string]any, bool) {
	for _, o := range list {
		if o["id"] == orderID {
			p, ok := o["customer_live"].(map[string]any)
			return p, ok
		}
	}
	return nil, false
}

// TestLiveLocation_OnlyTheCustomerMayPost **صاحبُ الطلب وحدَه يكتب** — وغيرُه «غيرُ موجود».
func TestLiveLocation_OnlyTheCustomerMayPost(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "on_the_way", &d)
	stranger := testdb.NewUser(t, f.pool, "customer")

	for _, who := range []struct{ id, role string }{{stranger, "customer"}, {d, "driver"}} {
		w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, who.id, who.role, farFromPin)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s كتب موقعَ زبونٍ ليس هو: %d %s", who.role, w.Code, w.Body.String())
		}
	}
	if n := f.liveRows(t, orderID); n != 0 {
		t.Fatalf("كُتبت نقطةٌ من غير صاحب الطلب: %d", n)
	}
	// **ولا يُمحى بيد غيره** — والإيقافُ بابُ صاحبه كذلك.
	if w := liveCall(f.srv.handleStopCustomerLiveLocation, http.MethodDelete, orderID, stranger, "customer", ""); w.Code != http.StatusNotFound {
		t.Fatalf("غريبٌ أوقف مشاركةَ غيره: %d", w.Code)
	}

	w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin)
	if w.Code != http.StatusOK {
		t.Fatalf("صاحبُ الطلب لم يكتب: %d %s", w.Code, w.Body.String())
	}
	// **وآخرُ نقطةٍ وحدَها** — الثانيةُ تحلّ محلّ الأولى لا تُضاف إليها.
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", nearPin); w.Code != http.StatusOK {
		t.Fatalf("النقطةُ الثانية: %d %s", w.Code, w.Body.String())
	}
	if n := f.liveRows(t, orderID); n != 1 {
		t.Fatalf("الصفوفُ %d لا ١ — آخرُ نقطةٍ وحدَها تُحفظ", n)
	}
	if w := liveCall(f.srv.handleStopCustomerLiveLocation, http.MethodDelete, orderID, owner, "customer", ""); w.Code != http.StatusOK {
		t.Fatalf("الإيقاف: %d %s", w.Code, w.Body.String())
	}
	if n := f.liveRows(t, orderID); n != 0 {
		t.Fatalf("«وقّف المشاركة» لم يمحُ النقطة: %d", n)
	}
}

// TestLiveLocation_RefusedWhenOrderNotActive **لا موقعَ لطلبٍ ليس في طريقه** — قبل الإسناد وبعد النهاية.
func TestLiveLocation_RefusedWhenOrderNotActive(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	for _, c := range []struct {
		status string
		driver *string
	}{{"dispatching", nil}, {"preparing", nil}, {"delivered", &d}, {"cancelled", &d}} {
		orderID, owner := f.liveOrder(t, c.status, c.driver)
		w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin)
		if w.Code != http.StatusConflict || errCode(t, w) != "live_location_inactive" {
			t.Fatalf("%s: قُبل الموقعُ أو رُدّ بغير سببه: %d %s", c.status, w.Code, w.Body.String())
		}
		if n := f.liveRows(t, orderID); n != 0 {
			t.Fatalf("%s: كُتبت نقطةٌ لطلبٍ ليس في طريقه", c.status)
		}
	}
	// **والطورُ المقبولُ يُقبل** — كي لا يمرّ الاختبارُ لأنّ كلَّ شيءٍ مرفوض.
	for _, st := range orders.LiveStatuses {
		orderID, owner := f.liveOrder(t, st, &d)
		if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", nearPin); w.Code != http.StatusOK {
			t.Fatalf("%s: رُدّ وهو في طريقه: %d %s", st, w.Code, w.Body.String())
		}
	}
}

// TestLiveLocation_DriverSeesFreshOmitsStale **السائقُ المُسنَدُ يراه حديثاً — ولا يراه شائخاً.**
func TestLiveLocation_DriverSeesFreshOmitsStale(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "on_the_way", &d)

	// **قبل المشاركة لا حقل** — يمشي إلى الدبّوس.
	if _, ok := liveOf(f.driverOrdersOf(t, d), orderID); ok {
		t.Fatal("ظهر customer_live قبل أن يشارك الزبون")
	}
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d %s", w.Code, w.Body.String())
	}
	p, ok := liveOf(f.driverOrdersOf(t, d), orderID)
	if !ok {
		t.Fatal("السائقُ المُسنَدُ لا يرى موقعَ الزبون الحديث")
	}
	for _, k := range []string{"lat", "lng", "accuracy_m", "at", "pin_gap_m", "switch_m"} {
		if _, has := p[k]; !has {
			t.Errorf("customer_live بلا %q: %v", k, p)
		}
	}
	gap, _ := p["pin_gap_m"].(float64)
	sw, _ := p["switch_m"].(float64)
	if gap < 300 || gap > 360 || sw != 150 {
		t.Errorf("بُعدُ الدبّوس %v والحدُّ %v — المتوقّع نحو ٣٣٣م وحدّ ١٥٠", gap, sw)
	}
	// **والطريقُ يتحوّل إليه** — الحكمُ نفسُه الذي تقرؤه الشاشة.
	if s := f.srv.liveRouteTarget(context.Background(), orderID, d); s == nil {
		t.Error("بعيدٌ عن الدبّوس ولم يتحوّل الطريقُ إليه")
	}

	// **وما شاخ دقيقتين لا يُرسَل** — يعود السائقُ إلى الدبّوس.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE order_live_locations SET at = now() - interval '3 minutes' WHERE order_id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	if _, ok := liveOf(f.driverOrdersOf(t, d), orderID); ok {
		t.Fatal("نقطةٌ شائخةٌ وصلت السائقَ على أنّها حديثة")
	}
	if s := f.srv.liveRouteTarget(context.Background(), orderID, d); s != nil {
		t.Error("نقطةٌ شائخةٌ حوّلت الطريق")
	}

	// **وقريبٌ من الدبّوس يُرى ولا يحوّل الطريق.**
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", nearPin); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d", w.Code)
	}
	if _, ok := liveOf(f.driverOrdersOf(t, d), orderID); !ok {
		t.Fatal("النقطةُ القريبةُ الحديثةُ لم تصل")
	}
	if s := f.srv.liveRouteTarget(context.Background(), orderID, d); s != nil {
		t.Error("زبونٌ عند دبّوسه حوّل الطريق")
	}
}

// TestLiveLocation_ClearedOnDelivery **التسليمُ يمحو النقطة** — ومن بابه الحقيقيّ.
func TestLiveLocation_ClearedOnDelivery(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	orderID, owner := f.liveOrder(t, "at_dropoff", &d)
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d %s", w.Code, w.Body.String())
	}
	if n := f.liveRows(t, orderID); n != 1 {
		t.Fatalf("لم تُكتب النقطة: %d", n)
	}
	if _, err := f.srv.orders.Transition(context.Background(), d, []string{"driver"}, orderID, "delivered", ""); err != nil {
		t.Fatalf("التسليم: %v", err)
	}
	if n := f.liveRows(t, orderID); n != 0 {
		t.Fatalf("بقي موقعُ الزبون بعد التسليم: %d صفّ", n)
	}

	// **والإلغاءُ كذلك** — النهايةُ من أيّ بابٍ جاءت.
	other, owner2 := f.liveOrder(t, "on_the_way", &d)
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, other, owner2, "customer", farFromPin); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d", w.Code)
	}
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'cancelled', closed_at = now() WHERE id = $1`, other); err != nil {
		t.Fatal(err)
	}
	if n := f.liveRows(t, other); n != 0 {
		t.Fatalf("بقي موقعُ الزبون بعد الإلغاء: %d صفّ", n)
	}
}

// TestLiveLocation_OtherDriversAndMerchantNeverSee **سائقٌ آخرُ والمتجرُ والزبونُ لا يرونه أبداً.**
func TestLiveLocation_OtherDriversAndMerchantNeverSee(t *testing.T) {
	f := newDriverFixture(t, 2)
	mine, other := f.drivers[0], f.drivers[1]
	orderID, owner := f.liveOrder(t, "on_the_way", &mine)
	if w := liveCall(f.srv.handleCustomerLiveLocation, http.MethodPost, orderID, owner, "customer", farFromPin); w.Code != http.StatusOK {
		t.Fatalf("الكتابة: %d", w.Code)
	}

	// **سائقٌ آخرُ ولو عُرض عليه الطلبُ** — فالشرطُ الإسنادُ لا الرؤية.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET offered_driver_id = $2 WHERE id = $1`, orderID, other); err != nil {
		t.Fatal(err)
	}
	for _, o := range f.driverOrdersOf(t, other) {
		if _, has := o["customer_live"]; has {
			t.Fatalf("سائقٌ غيرُ مُسنَدٍ رأى موقعَ الزبون: %v", o)
		}
	}
	got, err := f.srv.orders.LiveFor(context.Background(), []string{orderID}, other, 150)
	if err != nil || len(got) != 0 {
		t.Fatalf("LiveFor ردّ لسائقٍ غيرِ مُسنَد: %v %v", got, err)
	}
	if s := f.srv.liveRouteTarget(context.Background(), orderID, other); s != nil {
		t.Fatal("طريقُ سائقٍ غيرِ مُسنَدٍ تحوّل إلى الزبون")
	}

	// **والطلبُ كما يقرؤه كلُّ بابٍ آخر لا يحمله** — منه تُبنى حمولاتُ المتجر والبثّ.
	o, err := f.srv.orders.GetByID(context.Background(), orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.CustomerLive != nil {
		t.Fatal("استعلامُ الطلب العامّ حمل موقعَ الزبون — فيصل البثَّ والمتجر")
	}
	// **ولو حُمل خطأً** فالمتجرُ والزبونُ والمندوبُ لا يرونه في حمولتهم.
	o.CustomerLive = &orders.LivePoint{Lat: 1, Lng: 1}
	for _, a := range []orders.Audience{orders.AudienceMerchant, orders.AudienceMerchantDelivery,
		orders.AudienceCustomer, orders.AudienceRep} {
		if _, has := orders.ViewFor(a, o)["customer_live"]; has {
			t.Errorf("حمولةُ %q حملت موقعَ الزبون", a)
		}
	}
}
