package qa

import (
	"net/http"
	"testing"
	"time"
)

// waPickedUp **السائقُ استلم البضاعة وهو في الطريق إلى المستلِم.**
func waPickedUp(t *testing.T, h *Harness, orderID string) *User {
	t.Helper()
	drv := h.driverOf(orderID)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'on_the_way', picked_up_at = now() WHERE id = $1::uuid`, orderID); err != nil {
		t.Fatal(err)
	}
	return drv
}

// waDriverSignalled **وصلت غرفةَ السائق إشارةُ طلبٍ** — تطبيقُه يعيد قائمتَه بأيّ إشارة.
func waDriverSignalled(c *Capture) bool {
	for _, m := range c.Drain(700 * time.Millisecond) {
		if m != nil && m["type"] == "order" {
			return true
		}
	}
	return false
}

// TestWAORD032_LocationArrivesAfterPickup **المستلِمُ بعث موقعه والسائقُ في الطريق إليه**
// (طلبُ المالك ٢٠٢٦-١١-٠٦: «ينزل عند السائق بشكل صحيح وياخده ع موقعه بمرحلة الذهاب للزبون»).
//
// **بلا نقطةٍ ثمّ وصلت**، و**بنقطةٍ تقريبيّةٍ ثمّ وصلت الدقيقة** — في الحالين: النقطةُ في الطلب،
// وإشارةٌ لغرفة السائق، وإشعارٌ يوقظ هاتفه، **وقائمتُه تقول النقطةَ الجديدة معروفة.**
// **والتحديثُ المباشرُ التالي لا يحرّكها.**
func TestWAORD032_LocationArrivesAfterPickup(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ WA-031")
	h.Setting("delivery.custom_fee_min", "10")
	h.Setting("delivery.custom_fee_max", "1000")
	ops := h.NewUser("operations")

	// **١ · بلا نقطة.**
	phone := uniqPhone()
	reqID, _, _ := waLocReq(t, h, ops.Token, phone, "")
	b := w.body()
	delete(b, "lat")
	delete(b, "lng")
	b["recipient_phone"] = phone
	b["location_request_id"] = reqID
	b["fee"] = 650
	id := waCreated(t, waPost(t, h, ops.Token, b))
	drv := waPickedUp(t, h, id)
	if got := driverOrderOf(t, h, drv.Token, id); got["dropoff_known"] != false {
		t.Fatalf("**قبل الموقع النقطةُ معروفة عند السائق**: %v", got["dropoff_known"])
	}
	room := h.Listen("driver:" + drv.ID)
	if r := h.API.HandleWALocation(ctxBG(), phone, w.md.dropLat, w.md.dropLng, 8); r == "" {
		t.Fatal("**لا ردَّ على موقعٍ لطلبٍ في الطريق**")
	}
	if !waDriverSignalled(room) {
		t.Fatal("**غرفةُ السائق لم تصلها إشارةٌ بعد وصول الموقع**")
	}
	got := driverOrderOf(t, h, drv.Token, id)
	if got["status"] != "on_the_way" || got["dropoff_known"] != true ||
		!near(got["lat"], w.md.dropLat) || !near(got["lng"], w.md.dropLng) {
		t.Fatalf("**السائقُ في الطريق لا يرى الموقعَ الواصل**: %v · %v · %v,%v",
			got["status"], got["dropoff_known"], got["lat"], got["lng"])
	}
	if n := waNotes(t, h, drv.ID, id, "وصل موقع المستلم — صار على الخريطة"); len(n) != 1 {
		t.Fatalf("**إشعارُ السائق بعد الاستلام**: %v", n)
	}
	// **والتحديثُ المباشرُ التالي صامت** — لا يحرّك النقطةَ ولا يُشعر ثانيةً.
	if r := h.API.HandleWALocation(ctxBG(), phone, w.md.dropLat+0.001, w.md.dropLng, 8); r != "" {
		t.Fatalf("تحديثٌ مباشرٌ ثانٍ ردّ: %q", r)
	}
	if lat, _, _, _ := waOrderPoint(t, h, id); !nearF(lat, w.md.dropLat) {
		t.Fatalf("**التحديثُ الثاني حرّك النقطة**: %v", lat)
	}

	// **٢ · نقطةٌ تقريبيّةٌ من المكتب ثمّ وصلت الدقيقة** — تُستبدل والسائقُ في الطريق.
	phone2 := uniqPhone()
	req2, _, _ := waLocReq(t, h, ops.Token, phone2, "")
	b2 := w.body()
	b2["recipient_phone"] = phone2
	b2["location_request_id"] = req2
	r2 := waPost(t, h, ops.Token, b2)
	if r2.Code != http.StatusCreated {
		t.Fatalf("إنشاءٌ بنقطة: %s", r2)
	}
	id2 := waCreated(t, r2)
	drv2 := waPickedUp(t, h, id2)
	before := driverOrderOf(t, h, drv2.Token, id2)
	if before["dropoff_known"] != true {
		t.Fatalf("الطلبُ بنقطةٍ غيرُ معروفة: %v", before["dropoff_known"])
	}
	newLat, newLng := w.md.dropLat+0.002, w.md.dropLng+0.001
	room2 := h.Listen("driver:" + drv2.ID)
	if r := h.API.HandleWALocation(ctxBG(), phone2, newLat, newLng, 6); r == "" {
		t.Fatal("**لا ردَّ على الموقع الدقيق**")
	}
	if !waDriverSignalled(room2) {
		t.Fatal("**غرفةُ السائق لم تصلها إشارةٌ بالنقطة الجديدة**")
	}
	after := driverOrderOf(t, h, drv2.Token, id2)
	if after["dropoff_known"] != true || !near(after["lat"], newLat) || !near(after["lng"], newLng) {
		t.Fatalf("**السائقُ ما زال يرى النقطةَ القديمة**: %v,%v (المنتظَر %v,%v)",
			after["lat"], after["lng"], newLat, newLng)
	}
}
