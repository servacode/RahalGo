package orders_test

// ══════════════════════════════════════════════════════════════════════
// **إرجاعُ البضاعة مشوارٌ يُرى لا إغلاقٌ صامت** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// «عُد إلى المكتب» كان يُنهي الطلبَ ويخرج من يد السائق صامتاً — **لا طريقَ
// ولا زرَّ يقول إنّ البضاعةَ وصلت.** وما يُثبت هنا:
//
//	العودةُ والبضاعةُ معه          ←  مشوارُ إرجاعٍ إلى المكتب (`platform.location`)
//	«رجّع للمتجر» لمتجرٍ لا يستردّ ←  يُردّ قبل أن يُنهى شيء
//	«سلّمت البضاعة»               ←  وقتٌ وموضع، **مرّةً واحدة، ولسائقه وحدَه**

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// officeAt **يضبط موقعَ المكتب وعنوانَه** — والقاعدةُ مشتركةٌ فيُعادان بعد الاختبار.
func (f *fixture) officeAt(t *testing.T, location, address string) {
	t.Helper()
	f.setSetting(t, "platform.location", location)
	f.setSetting(t, "platform.address", address)
}

// returnAtDoor **المكتبُ يعيد السائقَ بالبضاعة من الباب** — إلى `to`.
func (f *fixture) returnAtDoor(t *testing.T, to string) (*orders.Order, error) {
	t.Helper()
	ops := testdb.NewUser(t, f.pool, "ops")
	return f.svc.ResolveDoor(context.Background(), ops, []string{"ops"}, f.orderID,
		orders.DoorResolution{Action: orders.DoorReturnToOffice, Fault: orders.FaultCustomer,
			Reason: "customer_refused", Note: "الزبونُ رفض — ارجع بالبضاعة", ReturnTo: to}, nil)
}

// tripOf **وجهةُ الإرجاع ووقتُ التسليم كما في القاعدة.**
func (f *fixture) tripOf(t *testing.T) (returnTo *string, handed bool, lat, lng *float64) {
	t.Helper()
	if err := f.pool.QueryRow(context.Background(), `
		SELECT return_to, goods_handed_at IS NOT NULL,
		       ST_Y(goods_handed_point::geometry), ST_X(goods_handed_point::geometry)
		FROM orders WHERE id = $1`, f.orderID).Scan(&returnTo, &handed, &lat, &lng); err != nil {
		t.Fatalf("تعذّرت قراءةُ المشوار: %v", err)
	}
	return
}

// TestReturnTrip_ToOfficeCreatesTrip **العودةُ والبضاعةُ معه ⇒ مشوارٌ إلى المكتب** —
// والطلبُ يُنهى كما كان (الحالُ والذنب)، **والوجهةُ موقعُ المكتب من الإعدادات.**
func TestReturnTrip_ToOfficeCreatesTrip(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	f.armTreasury(t)
	f.officeAt(t, "35.9600,39.0100", "مكتب رحّال — شارع الاختبار")

	o, err := f.returnAtDoor(t, "")
	if err != nil {
		t.Fatalf("العودةُ رُدّت: %v", err)
	}
	if o.Status != "failed" || o.Fault != orders.FaultCustomer {
		t.Fatalf("(%s · %s) — **والطلبُ يُنهى كما كان**", o.Status, o.Fault)
	}
	to, handed, _, _ := f.tripOf(t)
	if to == nil || *to != orders.ReturnToOffice || handed {
		t.Fatalf("المشوار (%v · سُلّمت=%v) — **والبضاعةُ معه ولا مشوارَ إلى المكتب**", to, handed)
	}
	if o.ReturnTo == nil || *o.ReturnTo != orders.ReturnToOffice || o.GoodsHandedAt != nil {
		t.Fatalf("بطاقةُ الإدارة لا ترى المشوار: %v · %v", o.ReturnTo, o.GoodsHandedAt)
	}

	p, err := f.svc.ReturnPointOf(context.Background(), f.orderID)
	if err != nil || p == nil {
		t.Fatalf("الوجهة: %v · %v", p, err)
	}
	if p.To != orders.ReturnToOffice || p.Lat == nil || p.Lng == nil ||
		math.Abs(*p.Lat-35.96) > 1e-9 || math.Abs(*p.Lng-39.01) > 1e-9 ||
		p.Address != "مكتب رحّال — شارع الاختبار" {
		t.Fatalf("الوجهة %+v — **والمكتبُ من `platform.location` وحدَه**", p)
	}

	// **والسائقُ يقرأ المشوارَ من مخرَج طلبه** — ولو ضاع الإشعار.
	out, err := f.svc.DriverOutcomeOf(context.Background(), f.orderID, f.driver)
	if err != nil {
		t.Fatalf("المخرَج: %v", err)
	}
	if out.ReturnTo != orders.ReturnToOffice || out.Reason != orders.LossReturnToOffice {
		t.Fatalf("المخرَج (%q · %q)", out.ReturnTo, out.Reason)
	}
}

// TestReturnTrip_OfficeWithoutLocation **مكتبٌ بلا موقعٍ مضبوط ⇒ مشوارٌ بلا طريق** —
// والعنوانُ المكتوبُ يُقرأ.
func TestReturnTrip_OfficeWithoutLocation(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	f.armTreasury(t)
	f.officeAt(t, "", "مكتب رحّال")
	if _, err := f.returnAtDoor(t, orders.ReturnToOffice); err != nil {
		t.Fatalf("العودة: %v", err)
	}
	p, err := f.svc.ReturnPointOf(context.Background(), f.orderID)
	if err != nil || p == nil {
		t.Fatalf("الوجهة: %v · %v", p, err)
	}
	if p.Lat != nil || p.Lng != nil || p.Address != "مكتب رحّال" {
		t.Fatalf("الوجهة %+v — **ولا نقطةَ تُخترع لمكتبٍ لم يُدبَّس**", p)
	}
}

// TestReturnTrip_StoreRefusedWithoutReturns **«رجّع للمتجر» لمتجرٍ لا يستردّ ⇒ يُردّ** —
// قبل أن يُنهى شيء. **ولمن يستردّ ⇒ مشوارٌ إلى موقع المتجر.**
func TestReturnTrip_StoreRefusedWithoutReturns(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	f.armTreasury(t)
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `
		UPDATE merchants SET accepts_returns = false,
		       location = ST_SetSRID(ST_MakePoint(39.02, 35.97), 4326)::geography
		WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.returnAtDoor(t, orders.ReturnToStore); !errors.Is(err, orders.ErrReturnStoreRefused) {
		t.Fatalf("«رجّع للمتجر» لمتجرٍ لا يستردّ: %v", err)
	}
	if st := f.statusOf(t); st != "at_dropoff" {
		t.Fatalf("الحالُ %q — **والرفضُ لا يُنهي شيئاً**", st)
	}
	if _, err := f.returnAtDoor(t, "home"); !errors.Is(err, orders.ErrReturnBadTarget) {
		t.Fatalf("وجهةٌ مجهولة: %v", err)
	}

	if _, err := f.pool.Exec(ctx,
		`UPDATE merchants SET accepts_returns = true WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.returnAtDoor(t, orders.ReturnToStore); err != nil {
		t.Fatalf("«رجّع للمتجر» لمتجرٍ يستردّ رُدّ: %v", err)
	}
	p, err := f.svc.ReturnPointOf(ctx, f.orderID)
	if err != nil || p == nil {
		t.Fatalf("الوجهة: %v · %v", p, err)
	}
	if p.To != orders.ReturnToStore || p.Label != "متجر اختبار التسويات" || p.Lat == nil ||
		math.Abs(*p.Lat-35.97) > 1e-9 || math.Abs(*p.Lng-39.02) > 1e-9 {
		t.Fatalf("الوجهة %+v — **والمتجرُ بموقعه واسمه**", p)
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != orders.LossReturnToStore {
		t.Fatalf("المخرَج %q — **و«إلى المتجر» غيرُ «إلى المكتب»**", out.Reason)
	}
}

// TestReturnTrip_HandGoods_OnceByDriver **«سلّمت البضاعة» — لسائقه وحدَه، ومرّةً واحدة** —
// ويكتب الوقتَ والموضع، **ويُنهي المشوار.**
func TestReturnTrip_HandGoods_OnceByDriver(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	f.armTreasury(t)
	f.officeAt(t, "35.9600,39.0100", "مكتب رحّال")
	if _, err := f.returnAtDoor(t, ""); err != nil {
		t.Fatalf("العودة: %v", err)
	}
	ctx := context.Background()
	lat, lng := 35.9601, 39.0102

	stranger := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.svc.HandGoods(ctx, f.orderID, stranger, &lat, &lng, nil); !errors.Is(err, orders.ErrNotDriversOrder) {
		t.Fatalf("سائقٌ غريبٌ سلّم بضاعةَ غيره: %v", err)
	}
	if _, handed, _, _ := f.tripOf(t); handed {
		t.Fatal("كُتب التسليمُ بضغطة غريب")
	}

	at, err := f.svc.HandGoods(ctx, f.orderID, f.driver, &lat, &lng, nil)
	if err != nil || at.IsZero() {
		t.Fatalf("«سلّمت البضاعة» رُدّ: %v", err)
	}
	_, handed, plat, plng := f.tripOf(t)
	if !handed || plat == nil || plng == nil ||
		math.Abs(*plat-lat) > 1e-6 || math.Abs(*plng-lng) > 1e-6 {
		t.Fatalf("التسليم (%v · %v,%v) — **والوقتُ والموضعُ يُكتبان**", handed, plat, plng)
	}
	if _, err := f.svc.HandGoods(ctx, f.orderID, f.driver, &lat, &lng, nil); !errors.Is(err, orders.ErrGoodsAlreadyHanded) {
		t.Fatalf("سُلّمت مرّتين: %v", err)
	}

	// **والمشوارُ انتهى** — لا وجهةَ ولا خبرَ عمّا فعله هو للتوّ.
	if p, err := f.svc.ReturnPointOf(ctx, f.orderID); err != nil || p != nil {
		t.Fatalf("بقيت وجهةٌ بعد التسليم: %+v · %v", p, err)
	}
	out, err := f.svc.DriverOutcomeOf(ctx, f.orderID, f.driver)
	if err != nil {
		t.Fatal(err)
	}
	if out.Reason != "" || out.ReturnTo != "" {
		t.Fatalf("المخرَج (%q · %q) — **ومن سلّم بيده لا يُقال له «لم يعد معك»**", out.Reason, out.ReturnTo)
	}
	// **والإدارةُ ترى وقتَ التسليم.**
	o, err := f.svc.GetByID(ctx, f.orderID)
	if err != nil || o.GoodsHandedAt == nil {
		t.Fatalf("بطاقةُ الإدارة لا ترى التسليم: %v", err)
	}
}

// TestReturnTrip_HandGoods_ServerLocation **بلا موضعٍ من الجهاز ⇒ آخرُ موضعٍ حديثٍ عند الخادم.**
func TestReturnTrip_HandGoods_ServerLocation(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	f.armTreasury(t)
	if _, err := f.returnAtDoor(t, ""); err != nil {
		t.Fatalf("العودة: %v", err)
	}
	ctx := context.Background()
	if _, err := f.pool.Exec(ctx, `
		UPDATE users SET last_location = ST_SetSRID(ST_MakePoint(39.05, 35.99), 4326)::geography,
		       last_location_at = now()
		WHERE id = $1`, f.driver); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.HandGoods(ctx, f.orderID, f.driver, nil, nil, nil); err != nil {
		t.Fatalf("التسليمُ بلا موضع: %v", err)
	}
	_, handed, plat, plng := f.tripOf(t)
	if !handed || plat == nil || math.Abs(*plat-35.99) > 1e-6 || math.Abs(*plng-39.05) > 1e-6 {
		t.Fatalf("الموضع (%v,%v) — **وموضعُ الخادم الحديثُ يُكتب**", plat, plng)
	}
}

// TestReturnTrip_HandGoods_NoTrip **ولا «سلّمت» بلا مشوار** — طلبٌ قائمٌ أو فشل بلا إرجاع.
func TestReturnTrip_HandGoods_NoTrip(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	ctx := context.Background()
	if _, err := f.svc.HandGoods(ctx, f.orderID, f.driver, nil, nil, nil); !errors.Is(err, orders.ErrNoReturnTrip) {
		t.Fatalf("طلبٌ قائمٌ عند الباب: %v", err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET status = 'failed', closed_at = now() WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.HandGoods(ctx, f.orderID, f.driver, nil, nil, nil); !errors.Is(err, orders.ErrNoReturnTrip) {
		t.Fatalf("فشلٌ بلا مشوار: %v", err)
	}
}

// TestParseGeoSetting **قيمةُ `KindGeo` نصّاً — وفاسدُها «لا موقع».**
func TestParseGeoSetting(t *testing.T) {
	if la, ln, ok := orders.ParseGeoSetting(" 35.95 , 39.01 "); !ok || la != 35.95 || ln != 39.01 {
		t.Fatalf("(%v,%v,%v)", la, ln, ok)
	}
	for _, bad := range []string{"", "35.9", "x,y", "95,10", "10,190"} {
		if _, _, ok := orders.ParseGeoSetting(bad); ok {
			t.Errorf("%q قُرئ موقعاً", bad)
		}
	}
}
