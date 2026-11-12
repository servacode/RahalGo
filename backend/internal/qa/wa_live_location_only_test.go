package qa

import "testing"

// ══════════════════════════════════════════════════════════════════════
// **الموقعُ المباشر وحدَه بالواتساب** (قرارُ المالك ٢٠٢٦-١١-١١: «نطلب من الشخص الموقع المباشر
// وليس الحالي، لازم ما يقبل الموقع الحالي»).
// ══════════════════════════════════════════════════════════════════════

// TestWALiveOnly_FixedRefusedLiveAccepted الثابتُ يُردّ عليه بطلب المباشر ولا يُكتب، والطلبُ
// يبقى ينتظر؛ ثمّ المباشرُ يملؤه. ورقمٌ بلا طلبٍ مفتوح لا يُردّ عليه بشيء.
func TestWALiveOnly_FixedRefusedLiveAccepted(t *testing.T) {
	h := New(t)
	h.Setting("whatsapp.live_location_only", "true") // **والمفتاحُ مطفأٌ افتراضاً** (٢٠٢٦-١١-١٢)
	cust := h.Customer()
	id, _ := requestHolderLocation(t, h, cust.Token, "0944555640")

	r := h.API.HandleWALocationKind(ctxBG(), "+963944555640", amPLat, amPLng, 8, false)
	if r == "" {
		t.Fatal("لا ردَّ على موقعٍ ثابت لطلبٍ مفتوح — المنتظَر طلبُ المباشر")
	}
	if got := waLocStatus(t, h, cust.Token, id); got["status"] == "received" {
		t.Fatalf("الثابتُ انكتب: %v", got)
	}

	if r := h.API.HandleWALocationKind(ctxBG(), "+963944555640", amPLat, amPLng, 8, true); r == "" {
		t.Fatal("المباشرُ لم يُقبل")
	}
	if got := waLocStatus(t, h, cust.Token, id); got["status"] != "received" || !near(got["lat"], amPLat) {
		t.Fatalf("المباشرُ لم يملأ الطلب: %v", got)
	}

	// **بلا طلبٍ مفتوح** — لا ردّ (رسالةٌ عاديّةٌ لا تخصّنا).
	if r := h.API.HandleWALocationKind(ctxBG(), "+963944555641", amPLat, amPLng, 8, false); r != "" {
		t.Fatalf("ردٌّ على رقمٍ بلا طلب: %q", r)
	}
}

// TestWALiveOnly_SwitchOffAcceptsFixed **والمفتاحُ مطفأً يرجع كما كان**: الثابتُ يُقبل.
func TestWALiveOnly_SwitchOffAcceptsFixed(t *testing.T) {
	h := New(t)
	h.Setting("whatsapp.live_location_only", "false")
	cust := h.Customer()
	id, _ := requestHolderLocation(t, h, cust.Token, "0944555642")

	if r := h.API.HandleWALocationKind(ctxBG(), "+963944555642", amPLat, amPLng, 8, false); r == "" {
		t.Fatal("الثابتُ لم يُقبل والمفتاحُ مطفأ")
	}
	if got := waLocStatus(t, h, cust.Token, id); got["status"] != "received" {
		t.Fatalf("الثابتُ لم يملأ الطلب: %v", got)
	}
}
