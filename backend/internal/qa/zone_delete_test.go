package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **حذفُ منطقة التغطية** (بلاغُ المالك ٢٠٢٦-١١-٢٢: «التغطية ماعم تنحذف يعطي خطا مشان
// اقدر اضيف منطقة وحدة تضم الكل»)
// ══════════════════════════════════════════════════════════════════════
//
// كان `orders.zone_id` يمنع حذفَ أيِّ منطقةٍ انطلب منها مرّةً (`0556`). **الطلبُ المنتهي لا
// يمنع** — يبقى بأرقامه وتُفرَّغ منطقتُه. **والشغّالُ يمنع** برسالةٍ واضحة.

func zdAttach(t *testing.T, h *Harness, orderID, zoneID, status string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET zone_id = $2::uuid, status = $3,
		       closed_at = CASE WHEN $3 IN ('delivered','cancelled') THEN now() ELSE NULL END
		 WHERE id = $1::uuid`, orderID, zoneID, status); err != nil {
		t.Fatalf("ربطُ الطلب بالمنطقة: %v", err)
	}
}

func TestZD01_ZoneWithFinishedOrdersDeletes(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")
	z := newZone(t, hh, "منطقةُ ZD-01", 35.9506, 39.0094)
	fx := populatedOrder(t, hh)
	zdAttach(t, hh, fx.OrderID, z.ID, "delivered")

	r := hh.Call("DELETE", "/api/v1/admin/zones/"+z.ID, tok, nil, nil)
	if r.Code != http.StatusOK && r.Code != http.StatusNoContent {
		t.Fatalf("**منطقةٌ طلباتُها منتهيةٌ لم تُحذف**: %d / %s", r.Code, r.Err())
	}
	var zone *string
	if err := hh.Pool.QueryRow(ctxBG(),
		`SELECT zone_id::text FROM orders WHERE id = $1::uuid`, fx.OrderID).Scan(&zone); err != nil {
		t.Fatalf("**الطلبُ القديمُ انحذف مع المنطقة**: %v", err)
	}
	if zone != nil {
		t.Errorf("**منطقةُ الطلب القديم لم تُفرَّغ**: %v", *zone)
	}
}

func TestZD02_ZoneWithOpenOrderIsRefusedClearly(t *testing.T) {
	hh := New(t)
	_, tok := capUser(t, hh, "admin")
	z := newZone(t, hh, "منطقةُ ZD-02", 35.9506, 39.0094)
	fx := populatedOrder(t, hh)
	zdAttach(t, hh, fx.OrderID, z.ID, "pending")

	r := hh.Call("DELETE", "/api/v1/admin/zones/"+z.ID, tok, nil, nil)
	if r.Code != http.StatusConflict || r.Err() != "zone_has_open_orders" {
		t.Fatalf("**منطقةٌ فيها طلبٌ شغّالٌ لم تُردّ برسالتها**: %d / %s", r.Code, r.Err())
	}
	var still bool
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT EXISTS (SELECT 1 FROM delivery_zones WHERE id = $1::uuid)`, z.ID).Scan(&still)
	if !still {
		t.Fatal("**المنطقةُ انحذفت وفيها طلبٌ شغّال**")
	}
}
