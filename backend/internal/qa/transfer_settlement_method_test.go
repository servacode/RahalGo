package qa

// ══════════════════════════════════════════════════════════════════════
// **التحويلُ لمتجرٍ آخر يأخذ طريقةَ تسوية المتجر الجديد** — خلل PD-3 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **طريقةُ التسوية لقطةٌ في بنود الطلب** (`order_items.merchant_settlement_method`) تُكتب عند
// الإنشاء، و`merchantShares` تقرؤها. **وكان التحويلُ يبدّل المتجرَ والسعرَ ويُبقي لقطةَ القديم**:
// طلبٌ من متجرٍ «محفظة» حُوّل لمتجرٍ «نقداً» يُقيَّد حقُّ الجديد في محفظته، **ولا يرى السائقُ
// ما يدفعه له** — فإن دفع من جيبه دفع بلا أثرٍ في الدفتر.

import (
	"encoding/json"
	"net/http"
	"testing"
)

// TestTransferResnapshotsSettlementMethod **T0c — متجرٌ «محفظة» ← تحويلٌ إلى متجرٍ «نقداً» ⇒
// البنودُ بطريقة الجديد، والسائقُ يرى ما يدفعه له، ويُسوّى نقداً عند الاستلام لا في محفظته.**
func TestTransferResnapshotsSettlementMethod(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	// **المكتبُ يقبل والطلبُ يبقى «مقبولاً»** حتّى يُبلَّغ المتجر — لا يتقدّم وحدَه.
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.notify_merchant", "true")
	h.Setting("orders.auto_transfer", "false")
	h.Setting("merchants.commission_percent", "10")
	treasury(t, h)
	staff := h.NewUser("operations")
	f := h.Factory()
	from := f.Merchant(Commission(10)) // «محفظة» — افتراضُ المصنع
	to := f.Merchant(Commission(10))
	coSetCash(t, h, to.ID)
	item := h.NewItemFor(from, 10_000)
	alt := h.NewItemFor(to, 10_000)
	base := financialBaseline(t, h)

	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("pd3"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	const a = "/api/v1/admin/orders/"
	if r := h.POST(a+oid+"/transition", staff.Token,
		map[string]any{"to": "accepted", "note": "قبولٌ من العمليّات"}); r.Code >= 400 {
		t.Fatalf("القبول: %s", r)
	}
	if st := h.statusOf(oid); st != "accepted" {
		t.Fatalf("الحالُ بعد القبول %q — يُنتظر accepted", st)
	}
	var line string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT id::text FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&line); err != nil {
		t.Fatal(err)
	}
	if r := h.POST(a+oid+"/transfer", staff.Token, map[string]any{
		"merchant_id": to.ID, "note": "المتجر اعتذر",
		"items": []map[string]any{{"order_item_id": line, "menu_item_id": alt.ID}},
	}); r.Code != http.StatusOK {
		t.Fatalf("التحويل: %s", r)
	}

	// **البندُ بطريقة المتجر الجديد.**
	var method string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT string_agg(DISTINCT COALESCE(merchant_settlement_method, ''), ',')
		  FROM order_items WHERE order_id = $1::uuid`, oid).Scan(&method); err != nil {
		t.Fatal(err)
	}
	if method != "cash" {
		t.Errorf("طريقةُ تسوية البنود بعد التحويل %q — يُنتظر cash (طريقةُ المتجر الجديد)", method)
	}

	// **والسائقُ يرى ما يدفعه للمتجر الجديد** قبل أن يصل.
	drv := h.driverOf(oid)
	list := h.GET("/api/v1/driver/orders", drv.Token)
	if list.Code != http.StatusOK {
		t.Fatalf("طلباتُ السائق: %s", list)
	}
	var mine []struct {
		ID           string `json:"id"`
		PayMerchants []struct {
			MerchantID string `json:"merchant_id"`
			Amount     int64  `json:"amount"`
		} `json:"pay_merchants"`
	}
	if err := json.Unmarshal(rawData(list), &mine); err != nil {
		t.Fatalf("فكُّ طلبات السائق: %v", err)
	}
	var pay int64
	for _, o := range mine {
		if o.ID == oid {
			for _, p := range o.PayMerchants {
				if p.MerchantID == to.ID {
					pay = p.Amount
				}
			}
		}
	}
	if pay != 9_000 {
		t.Errorf("ما يدفعه السائقُ للمتجر الجديد %d — يُنتظر 9000 (١٠٬٠٠٠ − ١٠٪)", pay)
	}

	// **وعند الاستلام يُسوّى نقداً بيد السائق** — لا قيدَ في محفظة صاحب المتجر.
	pd1Steps(t, h, oid, drv, "at_pickup", "picked_up")
	amount, state, via, _ := coSettlement(t, h, oid, to.ID)
	if amount != 9_000 || state != "cash_paid" || via != "driver" {
		t.Errorf("تسويةُ المتجر الجديد: %d %s %q — يُنتظر 9000 cash_paid driver", amount, state, via)
	}
	if got := ledgerByRef(t, h, oid, "merchant_earning"); got != 0 {
		t.Errorf("مستحقٌّ في محفظة متجرٍ «نقداً» %d — يُنتظر صفر", got)
	}
	if got := walletBalance(t, h, to.Owner.ID); got != 0 {
		t.Errorf("محفظةُ صاحب المتجر الجديد %d — يُنتظر صفر", got)
	}

	pd1Steps(t, h, oid, drv, "on_the_way", "at_dropoff", "delivered")
	assertNewViolations(t, h, base)
}
