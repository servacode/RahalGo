package qa

// ══════════════════════════════════════════════════════════════════════
// **مشترياتٌ من متجرٍ تابعٍ إلنا — الربطُ ورسالةُ واتساب** (قرارُ المالك ٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
// **يربط المكتبُ المتجرَ قبل توثيق الثمن، ثمّ يرسل له الطلبَ بالبابين القائمين**
// (`/message` و`/whatsapp`): «طلب جديد #رقم» وما كتبه الزبونُ بلفظه.

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/pricing"
)

// partnerPurchaseOrder **طلبُ مشترياتٍ مُسنَدٌ بأجرةٍ موثَّقةٍ وثمنٍ ينتظر.**
func (h *Harness) partnerPurchaseOrder(t *testing.T, request string) (string, int64) {
	t.Helper()
	cust := h.Customer()
	drv := h.NewUser("driver")
	var id string
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO orders (customer_id, driver_id, kind, custom_mode, custom_request, status,
		                    address_text, dropoff, payment_method, subtotal, delivery_fee, total,
		                    custom_fee, custom_goods_amount, custom_goods_pending,
		                    snap_merchant_commission_percent, snap_rep_commission_percent,
		                    snap_commission_source, snap_activation_orders)
		VALUES ($1, $2, 'custom', 'purchase', $3, 'assigned', 'الرقة',
		        ST_SetSRID(ST_MakePoint(39.0094, 35.9506),4326)::geography, 'cash', 0, 0, 0,
		        1000, 0, true, 10, 10, $4, 1)
		RETURNING id::text, number`, cust.ID, drv.ID, request,
		string(pricing.RepCommissionSourceFixed)).Scan(&id, &number); err != nil {
		t.Fatalf("زرعُ الطلب: %v", err)
	}
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM orders WHERE id = $1::uuid`, id) })
	return id, number
}

// TestPARTNER_001_WhatsAppPreview **المعاينةُ تعمل لمشترياتٍ رُبطت بمتجرٍ تابع.**
func TestPARTNER_001_WhatsAppPreview(t *testing.T) {
	h := New(t)
	ops := h.NewUser("operations")
	store := h.NewItem(1000)
	const request = "كيلو بندورة وربطة خبز"
	oid, number := h.partnerPurchaseOrder(t, request)

	// **وقبل الربط لا رسالة** — الخاصُّ بلا متجرٍ لا متجرَ يُرسَل له.
	if got := h.GET("/api/v1/admin/orders/"+oid+"/message", ops.Token); got.Code < 400 {
		t.Fatalf("PARTNER-001 رسالةٌ لطلبٍ بلا متجر: %s", got)
	}
	linked := h.POST("/api/v1/admin/orders/"+oid+"/partner", ops.Token,
		map[string]any{"merchant_id": store.MerchantID})
	if linked.Code >= 400 {
		t.Fatalf("PARTNER-001 الربطُ رُدّ: %s", linked)
	}
	res := h.GET("/api/v1/admin/orders/"+oid+"/message", ops.Token)
	if res.Code >= 400 {
		t.Fatalf("PARTNER-001 المعاينةُ رُدّت: %s", res)
	}
	var body struct {
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("PARTNER-001 ردٌّ غيرُ مقروء: %v", err)
	}
	text, _ := body.Data["text"].(string)
	if !strings.Contains(text, fmt.Sprintf("#%d", number)) || !strings.Contains(text, request) {
		t.Errorf("PARTNER-001 نصُّ الرسالة لا يحمل الرقمَ والطلب: %q", text)
	}
	if link, _ := body.Data["wa_link"].(string); link == "" {
		t.Error("PARTNER-001 لا رابطَ واتساب")
	}
	// **والإرسالُ يَسِم الطلب.**
	sent := h.POST("/api/v1/admin/orders/"+oid+"/whatsapp", ops.Token, map[string]any{"channel": "whatsapp"})
	if sent.Code >= 400 {
		t.Fatalf("PARTNER-001 الإرسالُ رُدّ: %s", sent)
	}
	var marked bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT sent_to_merchant_at IS NOT NULL FROM orders
		WHERE id = $1::uuid`, oid).Scan(&marked); err != nil || !marked {
		t.Errorf("PARTNER-001 لم يُوسَم الإرسال (%v)", err)
	}
}

// TestPARTNER_002_LinkRefusedAfterGoods **لا ربطَ بعد توثيق الثمن — بخطئه الصريح.**
func TestPARTNER_002_LinkRefusedAfterGoods(t *testing.T) {
	h := New(t)
	ops := h.NewUser("operations")
	store := h.NewItem(1000)
	oid, _ := h.partnerPurchaseOrder(t, "غرض")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET custom_goods_amount = 500,
		custom_goods_pending = false WHERE id = $1::uuid`, oid); err != nil {
		t.Fatalf("توثيقُ الثمن: %v", err)
	}
	got := h.POST("/api/v1/admin/orders/"+oid+"/partner", ops.Token,
		map[string]any{"merchant_id": store.MerchantID})
	if got.Code != 409 || !strings.Contains(string(got.Body), "partner_goods_documented") {
		t.Fatalf("PARTNER-002 المنتظَر 409 partner_goods_documented — جاء: %s", got)
	}
}
