package qa

// ══════════════════════════════════════════════════════════════════════
//  **قبلَ الافتتاح: الزبونُ لا يطلب، والمتجرُ يطلب «لدي توصيلة»**
//  (سؤالُ المالك ٢٠٢٦-١٠-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// **حالُ الإنتاج اليوم**: `launch.customer_orders` و`launch.customer_custom_orders`
// مغلقان («قريبًا يتم افتتاح رحال غو»)، **والمالكُ يريد السائقين في الدوام**
// (`launch.driver_work`) لتوصيلات المتاجر وحدَها. **فيُقاس الحالُ كما هو**:
// طلبُ الزبون العاديُّ والخاصُّ يُردّان `launch_closed`، **وتوصيلةُ المتجر تُنشأ
// وتُقبل وتنزل إلى السائق.**

import (
	"net/http"
	"testing"
)

func TestPLD01_PreLaunch_CustomerBlocked_MerchantDeliveryFlows(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "PL01")
	h.Setting("launch.customer_orders", "false")
	h.Setting("launch.customer_custom_orders", "false")
	h.Setting("launch.merchant_orders", "false")
	h.Setting("launch.driver_work", "false")

	// ── الزبونُ: لا طلبَ عاديّاً ولا خاصّاً ──
	cust := h.NewUser("customer")
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("pl01o"), orderBody(m.fx.Item, 1)); r.Code != http.StatusServiceUnavailable || r.Err() != "launch_closed" {
		t.Fatalf("**الزبونُ طلب قبل الافتتاح** — %d %s", r.Code, r.Err())
	}
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("pl01c"), zoneCustomBody(m.dropLat, m.dropLng)); r.Code != http.StatusServiceUnavailable || r.Err() != "launch_closed" {
		t.Fatalf("**الزبونُ أرسل طلباً خاصّاً قبل الافتتاح** — %d %s", r.Code, r.Err())
	}

	// ── السائقُ: **المفتاحُ هو الحَكَم** — مطفأً لا دوام، مشغَّلاً يبدأ ──
	drv := m.f.Driver()
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Code != http.StatusServiceUnavailable || r.Err() != "launch_closed" {
		t.Fatalf("**بدأ السائقُ دوامَه ومفتاحُه مطفأ** — %d %s", r.Code, r.Err())
	}
	h.Setting("launch.driver_work", "true")
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Code != http.StatusOK {
		t.Fatalf("**السائقُ لم يبدأ دوامَه** — %d %s", r.Code, r.Err())
	}

	// ── المتجرُ: «لدي توصيلة» تُسعَّر وتُنشأ وتُقبل وتنزل إلى السائق ──
	if r := h.GET(m.url("/delivery-quote?lat="+ftoa(m.dropLat)+"&lng="+ftoa(m.dropLng)), m.fx.Tok); r.Code != http.StatusOK {
		t.Fatalf("**تسعيرُ التوصيلة رُدّ** — %d %s", r.Code, r.Err())
	}
	// **ولا دينَ على أجرة توصيلة** (قرارُ المالك ٢٠٢٦-١٠-١٥) — فالمحفظةُ تُشحن قبل الطلب.
	fundWallet(t, h, m.admin, m.fx.M.Owner.ID, 1000)
	id := m.create(t, h, "merchant", uniq("pl01d"))
	m.accept(t, h, id)
}

// TestPLD02_CustomOrdersOpenAlone — **«الطلباتُ المخصّصة» وحدَها مفتوحة**: المشترياتُ
// والأمانةُ تُنشآن، **والطلبُ العاديُّ من متجرٍ يبقى مغلقاً.**
func TestPLD02_CustomOrdersOpenAlone(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "PLD02")
	h.Setting("launch.customer_orders", "false")
	h.Setting("launch.customer_custom_orders", "true")
	h.Setting("launch.driver_work", "true")

	cust := h.NewUser("customer")
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("pld02o"), orderBody(m.fx.Item, 1)); r.Code != http.StatusServiceUnavailable || r.Err() != "launch_closed" {
		t.Fatalf("**الطلبُ العاديُّ فُتح مع المخصّص** — %d %s", r.Code, r.Err())
	}
	for _, mode := range []string{"purchase", "amanah"} {
		b := zoneCustomBody(m.dropLat, m.dropLng)
		b["custom_mode"] = mode
		r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("pld02"+mode), b)
		if r.Code != http.StatusCreated {
			t.Fatalf("**%s لم يُنشأ والمخصّصُ مفتوح** — %d %s", mode, r.Code, r.Err())
		}
	}
}
