package qa

import "testing"

// **«كسبت اليوم» بيوم التسليم لا بيوم القيد** (المالك ٢٠٢٦-١٠-١٣): تصحيحُ ثمن بضاعة
// طلبٍ سُلّم أمس قُيِّد اليوم، **فقرأ السائقُ «كسبت −1,100» قبل أن يعمل.**
func TestDriverTodayEarned_ByDeliveryDay(t *testing.T) {
	h := New(t)
	oid, drv := activeOrderFor(t, h)
	if _, err := h.Pool.Exec(h.T.Context(), `
		UPDATE orders SET status = 'delivered', delivered_at = now() - interval '2 days',
		       closed_at = now() - interval '2 days'
		WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("تسليمُ الطلب أمس: %v", err)
	}
	// أجرُه وثمنُ البضاعة أمس، والتصحيحُ اليوم.
	if err := h.walletApply(drv.ID, 1300, "driver_earning", oid); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(h.T.Context(), `
		UPDATE wallet_transactions SET created_at = now() - interval '2 days'
		WHERE ref = $1 AND kind = 'driver_earning'`, oid); err != nil {
		t.Fatal(err)
	}
	if err := h.walletApply(drv.ID, -1100, "driver_earning", oid); err != nil {
		t.Fatal(err)
	}
	me := h.GET("/api/v1/driver/me", drv.Token)
	if me.Code >= 400 {
		t.Fatalf("لوحةُ السائق: %s", me)
	}
	if got, _ := me.JSON()["today_earned"].(float64); got != 0 {
		t.Fatalf("كسبت اليوم = %v — والطلبُ سُلّم أمس فلا يدخل يومَه، ولا تصحيحُه", got)
	}
}
