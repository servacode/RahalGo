package qa

import "testing"

// TestDoor_NoDeliveryWhileReportOpen **لا تسليمَ والبلاغُ مفتوح** (المالك ٢٠٢٦-١٠-٢٢): بلّغ
// السائقُ عند الباب ⇒ «سلّمت» يُردّ حتّى تأمر الإدارة «سلّم الآن».
func TestDoor_NoDeliveryWhileReportOpen(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, map[string]any{"to": to})
	}
	h.authorizeProofExempt(oid)
	code := firstDoorReport(t, h, drv)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/report", drv.Token,
		map[string]any{"code": code}); got.Code >= 400 {
		t.Fatalf("البلاغ: %s", got)
	}
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code != 409 || got.JSON()["error"].(map[string]any)["code"] != "door_report_pending" {
		t.Fatalf("**سُلّم والبلاغُ مفتوح**: %s", got)
	}
	ops := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/door-resolution", ops.Token,
		map[string]any{"action": "deliver_now"}); got.Code >= 400 {
		t.Fatalf("سلّم الآن: %s", got)
	}
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليمُ بعد أمر الإدارة رُدّ: %s", got)
	}
}

// TestDoor_ReturnWithReasonNeedsNoNote **سببٌ من القائمة يكفي** (المالك ٢٠٢٦-١٠-٢٢: «اسباب
// جاهزة او الملاحظة مو اجبارية») — والعودةُ بلا كلمة.
func TestDoor_ReturnWithReasonNeedsNoNote(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, map[string]any{"to": to})
	}
	code := firstDoorReport(t, h, drv)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/report", drv.Token,
		map[string]any{"code": code}); got.Code >= 400 {
		t.Fatalf("البلاغ: %s", got)
	}
	ops := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/door-resolution", ops.Token,
		map[string]any{"action": "return_to_office", "fault": "customer", "reason": code}); got.Code >= 400 {
		t.Fatalf("**العودةُ بسببٍ بلا كلمةٍ رُدّت**: %s", got)
	}
	if st := h.statusOf(oid); st != "failed" {
		t.Fatalf("الحالُ %q — يُنتظر failed", st)
	}
}
