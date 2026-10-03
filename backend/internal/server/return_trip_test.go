package server

// ══════════════════════════════════════════════════════════════════════
// **مشوارُ إرجاع البضاعة من أبواب السائق والإدارة** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// **كان «عُد إلى المكتب» يُخرج الطلبَ من قائمة السائق صامتاً** — لا طريقَ إلى المكتب
// ولا زرَّ يقول إنّ البضاعةَ وصلت. **وما يُثبت هنا من الأبواب نفسِها**: الطلبُ يبقى
// في قائمته مشوارَ إرجاعٍ وجهتُه المكتب، و«سلّمت البضاعة» يُخرجه، مرّةً ولسائقه وحدَه،
// **و«رجّع للمتجر» يُردّ لمتجرٍ لا يستردّ.**

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// armOffice **يضبط موقعَ المكتب وعنوانَه** — ويُعيدهما بعد الاختبار (القاعدةُ مشتركة).
func (f *driverFixture) armOffice(t *testing.T, location, address string) {
	t.Helper()
	ctx := context.Background()
	for _, key := range []string{"platform.location", "platform.address"} {
		var before *string
		_ = f.pool.QueryRow(ctx, `SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&before)
		k := key
		t.Cleanup(func() {
			c := context.Background()
			if before == nil {
				_, _ = f.pool.Exec(c, `DELETE FROM app_settings WHERE key = $1`, k)
				return
			}
			_, _ = f.pool.Exec(c, `UPDATE app_settings SET value = $2::jsonb WHERE key = $1`, k, *before)
		})
	}
	f.setSetting(t, "platform.location", location)
	f.setSetting(t, "platform.address", address)
	// **والمحرّكُ يقرأ الإعداداتِ كما في الإقلاع** — والعُدّةُ تبنيه بلاها.
	f.srv.orders.SetSettings(f.srv.settings)
}

// goodsHanded **«سلّمت البضاعة» من بابه** — كما يناديه التطبيق.
func (f *driverFixture) goodsHanded(driverID, orderID, body string) (int, string) {
	w := f.call(f.srv.handleDriverGoodsHanded, http.MethodPost,
		"/driver/orders/"+orderID+"/goods-handed", orderID, driverID, []string{"driver"}, body)
	code := ""
	if w.Code != http.StatusOK {
		code = errCodeOf(w.Body.Bytes())
	}
	return w.Code, code
}

// TestReturnTrip_DriverSeesReturnLegThenHandsGoods **العودةُ إلى المكتب مشوارٌ في قائمته** —
// وجهتُه المكتبُ لا بابُ الزبون، **و«سلّمت البضاعة» يُخرجه مرّةً ولسائقه وحدَه.**
func TestReturnTrip_DriverSeesReturnLegThenHandsGoods(t *testing.T) {
	f := newDriverFixture(t, 2)
	driver, stranger := f.drivers[0], f.drivers[1]
	f.armOffice(t, "35.9600,39.0100", "مكتب رحّال — شارع الاختبار")
	orderID := f.atDropoffOrder(t, driver)

	if w := f.endAtDoor(t, orderID, "customer", "customer_refused"); w.Code != http.StatusOK {
		t.Fatalf("العودةُ رُدّت: %d — %s", w.Code, w.Body.String())
	}

	list := f.driverOrders(t, driver)
	if len(list) != 1 {
		t.Fatalf("قائمتُه %d — **والبضاعةُ معه ومشوارُ الإرجاع لا يُرى**", len(list))
	}
	o := list[0]
	lat, _ := o["lat"].(float64)
	lng, _ := o["lng"].(float64)
	if o["status"] != "failed" || o["return_to"] != "office" || o["dropoff_known"] != true ||
		math.Abs(lat-35.96) > 1e-9 || math.Abs(lng-39.01) > 1e-9 ||
		o["address_text"] != "مكتب رحّال — شارع الاختبار" || o["customer_name"] != "" {
		t.Fatalf("المشوار %v — **والوجهةُ المكتبُ لا بابُ زبونٍ رفض**", o)
	}

	if code, _ := f.goodsHanded(stranger, orderID, `{"lat":35.96,"lng":39.01}`); code != http.StatusNotFound {
		t.Fatalf("سائقٌ غريبٌ سلّم بضاعةَ غيره: %d", code)
	}
	if code, ec := f.goodsHanded(driver, orderID, `{"lat":35.9601,"lng":39.0102}`); code != http.StatusOK {
		t.Fatalf("«سلّمت البضاعة» رُدّ: %d %s", code, ec)
	}
	if got := f.driverOrders(t, driver); len(got) != 0 {
		t.Fatalf("بقي المشوارُ بعد التسليم: %d", len(got))
	}
	if code, ec := f.goodsHanded(driver, orderID, `{}`); code != http.StatusConflict || ec != "goods_already_handed" {
		t.Fatalf("سُلّمت مرّتين: %d %s", code, ec)
	}

	var handed bool
	var plat *float64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT goods_handed_at IS NOT NULL, ST_Y(goods_handed_point::geometry)
		FROM orders WHERE id = $1`, orderID).Scan(&handed, &plat); err != nil {
		t.Fatal(err)
	}
	if !handed || plat == nil || math.Abs(*plat-35.9601) > 1e-6 {
		t.Fatalf("التسليم (%v · %v) — **والوقتُ والموضعُ يُكتبان**", handed, plat)
	}
}

// TestReturnTrip_StoreRefusedHTTP **«رجّع للمتجر» لمتجرٍ لا يستردّ ⇒ ٤٠٩** — والطلبُ عند الباب.
func TestReturnTrip_StoreRefusedHTTP(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET accepts_returns = false WHERE id = $1`, f.merchantID); err != nil {
		t.Fatal(err)
	}
	ops := testdb.NewUser(t, f.pool, "ops")
	w := f.call(f.srv.handleDoorResolution, http.MethodPost,
		"/admin/orders/"+orderID+"/door-resolution", orderID, ops, []string{"ops"},
		`{"action":"return_to_office","fault":"customer","note":"رفض","return_to":"store"}`)
	if w.Code != http.StatusConflict || errCodeOf(w.Body.Bytes()) != "return_store_refused" {
		t.Fatalf("ردّ %d %s — **ومتجرٌ لا يستردّ لا تُرجَع إليه البضاعة**", w.Code, w.Body.String())
	}
	var status string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT status FROM orders WHERE id = $1`, orderID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "at_dropoff" {
		t.Fatalf("الحالُ %q — **والرفضُ لا يُنهي شيئاً**", status)
	}
}

// TestReturnTrip_NoTripNoHandover **ولا «سلّمت» على طلبٍ قائم** — لا مشوارَ إرجاعٍ عليه.
func TestReturnTrip_NoTripNoHandover(t *testing.T) {
	f := newDriverFixture(t, 1)
	driver := f.drivers[0]
	orderID := f.atDropoffOrder(t, driver)
	if code, ec := f.goodsHanded(driver, orderID, ""); code != http.StatusConflict || ec != "no_return_trip" {
		t.Fatalf("ردّ %d %s", code, ec)
	}
}

// errCodeOf **رمزُ الخطأ من جسد الردّ** — وفارغٌ إن لم يُقرأ.
func errCodeOf(b []byte) string {
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	_ = json.Unmarshal(b, &body)
	return body.Error.Code
}
