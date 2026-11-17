package qa

// ══════════════════════════════════════════════════════════════════════
// **«تعديل الأجرة» على التوصيلة وطلب الواتساب — بالمسار الحيّ** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// `POST /api/v1/admin/orders/{id}/delivery-fee` بقدرة تعديل الطلب (`orders.intervene`).
// **والأرقامُ من `newMDFx`**: الأجرةُ ٥٠٠ وحصّةُ المنصّة ١٠٪. **ثمّ يُسلَّم الطلبُ** ويُقاس أجرُ
// السائق وربحُ المنصّة على الأجرة الجديدة، **وثوابتُ الدفتر بلا خرقٍ جديد.**

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

func feeEditCall(h *Harness, tok, id string, fee int64, dry bool) Res {
	return h.POST("/api/v1/admin/orders/"+id+"/delivery-fee", tok,
		map[string]any{"fee": fee, "reason": "المشوار أبعد من المتوقع", "dry_run": dry})
}

func feeEditResult(t *testing.T, r Res) map[string]any {
	t.Helper()
	if r.Code != http.StatusOK {
		t.Fatalf("تعديلُ الأجرة: %s", r)
	}
	res, _ := r.JSON()["result"].(map[string]any)
	if res == nil {
		t.Fatalf("لا نتيجة: %s", r)
	}
	return res
}

func feeNum(m map[string]any, k string) int64 {
	v, _ := m[k].(float64)
	return int64(v)
}

// TestFEE01_MerchantWalletRefundAndCharge **المتجرُ يدفع من محفظته**: بعد القبول تُخصم الزيادةُ
// ويُردّ النقص، **والمعاينةُ لا تمسّ شيئاً**، ثمّ يُسلَّم بالأجرة الأخيرة.
func TestFEE01_MerchantWalletRefundAndCharge(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FEE-01")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "merchant", "", "fee01")
	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("بعد القبول المحفظة %d", got)
	}

	res := feeEditResult(t, feeEditCall(h, m.admin.Token, id, 700, true))
	if feeNum(res, "charged") != 200 || feeNum(res, "driver_fee") != 630 || feeNum(res, "platform_share") != 70 ||
		feeNum(res, "new_total") != 700 || res["payer"] != "merchant" {
		t.Fatalf("المعاينة: %v", res)
	}
	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("**المعاينةُ خصمت من المتجر**: %d", got)
	}

	feeEditResult(t, feeEditCall(h, m.admin.Token, id, 700, false))
	if got := walletOf(t, h, owner); got != 1300 {
		t.Fatalf("بعد الزيادة المحفظة %d والمنتظَر 1300", got)
	}
	res = feeEditResult(t, feeEditCall(h, m.admin.Token, id, 600, false))
	if feeNum(res, "refunded") != 100 {
		t.Fatalf("النقص: %v", res)
	}
	if got := walletOf(t, h, owner); got != 1400 {
		t.Fatalf("بعد النقص المحفظة %d والمنتظَر 1400", got)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity_id = $1
		AND user_id = $2::uuid AND body LIKE '%المشوار أبعد%'`, id, owner).Scan(&told)
	if told != 2 {
		t.Fatalf("إشعاراتُ المتجر %d والمنتظَر 2", told)
	}

	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if got := walletOf(t, h, drv.ID); got != 540 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 540", got)
	}
	if got := mdmProfit(t, h, id); got != 60 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 60", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestFEE02_MerchantWalletShortRefused **زيادةٌ لا يكفيها رصيدُ المتجر تُردّ** — ولا دين.
func TestFEE02_MerchantWalletShortRefused(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FEE-02")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 600)
	id, _ := mdmCreate(t, h, m, "merchant", "", "fee02")
	if r := feeEditCall(h, m.admin.Token, id, 900, false); r.Code != http.StatusPaymentRequired ||
		r.Err() != "fee_edit_merchant_wallet" {
		t.Fatalf("زيادةٌ فوق الرصيد: %s", r)
	}
	if got := walletOf(t, h, owner); got != 100 {
		t.Fatalf("تحرّكت المحفظة: %d", got)
	}
}

// TestFEE03_RecipientCashAtDelivery **المستلِمُ يدفع عند التسليم** — يقبض السائقُ الأجرةَ الجديدة.
func TestFEE03_RecipientCashAtDelivery(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FEE-03")
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "", "fee03")
	mdDriveTo(t, h, id, drv, "picked_up")
	res := feeEditResult(t, feeEditCall(h, m.admin.Token, id, 800, false))
	if feeNum(res, "cash_due") != 800 || feeNum(res, "driver_fee") != 720 || res["payer"] != "recipient" {
		t.Fatalf("التعديل: %v", res)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if sum, _ := mdCollected(t, h, id); sum != 800 {
		t.Fatalf("الصندوق %d والمنتظَر 800", sum)
	}
	if got := walletOf(t, h, drv.ID); got != 720 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 720", got)
	}
	if got := mdmProfit(t, h, id); got != 80 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 80", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestFEE04_MerchantCashAfterPickupRefused **«أنا نقداً»**: قبل الاستلام يُعدَّل فيُقبض الجديد،
// **وبعد أن قُبض يُردّ التعديل** — ثمّ يُسلَّم والدفترُ أخضر. **وبعد التسليم لا تعديل.**
func TestFEE04_MerchantCashAfterPickupRefused(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FEE-04")
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "merchant_cash", "", "fee04")
	feeEditResult(t, feeEditCall(h, m.admin.Token, id, 600, false))
	mdDriveTo(t, h, id, drv, "picked_up")
	if sum, _ := mdCollected(t, h, id); sum != 600 {
		t.Fatalf("قُبض %d عند الاستلام والمنتظَر 600", sum)
	}
	if r := feeEditCall(h, m.admin.Token, id, 900, false); r.Code != http.StatusConflict ||
		r.Err() != "fee_edit_cash_collected" {
		t.Fatalf("**عُدّلت أجرةٌ قُبضت نقداً**: %s", r)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if got := walletOf(t, h, drv.ID); got != 540 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 540", got)
	}
	if got := mdmProfit(t, h, id); got != 60 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 60", got)
	}
	if r := feeEditCall(h, m.admin.Token, id, 700, false); r.Code != http.StatusConflict ||
		r.Err() != "order_edit_closed" {
		t.Fatalf("بعد التسليم: %s", r)
	}
	mdaNoNewViolations(t, h, base)
}

// TestFEE05_WhatsAppOrderFee **طلبُ الواتساب**: الأجرةُ وحدَها تتبدّل — والبضاعةُ برّا الحسابات.
func TestFEE05_WhatsAppOrderFee(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ FEE-05")
	base := mdaBaseline(t, h)
	c := waEngineCreate(t, h, w.md.admin.ID, w.typed())
	res := feeEditResult(t, feeEditCall(h, w.md.admin.Token, c.ID, 700, false))
	if feeNum(res, "new_total") != 700 || feeNum(res, "cash_due") != 700 || feeNum(res, "driver_fee") != 630 {
		t.Fatalf("التعديل: %v", res)
	}
	var collect int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT collect_amount FROM orders WHERE id = $1`, c.ID).Scan(&collect)
	if collect != 2400 {
		t.Fatalf("**تبدّل ثمنُ البضاعة** %d", collect)
	}
	drv := h.driverOf(c.ID)
	deliverOrder(t, h, c.ID, drv)
	if got := walletOf(t, h, drv.ID); got != 630 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 630", got)
	}
	if got := waCashOf(t, h, c.ID); got != 700 {
		t.Fatalf("الصندوق %d والمنتظَر 700", got)
	}
	led := waLedgerOf(t, h, c.ID)
	if led["platform_profit"] != 70 || led["driver_earning"] != 630 {
		t.Fatalf("الدفتر: %v", led)
	}
	mdaNoNewViolations(t, h, base)
}

// TestFEE06_PermissionAndReason **بقدرة تعديل الطلب** — قارئُ الطلبات وحدَه يُردّ، **والسببُ إجباريّ.**
func TestFEE06_PermissionAndReason(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FEE-06")
	id, _ := mdmCreate(t, h, m, "recipient", "", "fee06")
	capRole(t, h, "qa_fee_reader", authz.OrdersRead)
	capRole(t, h, "qa_fee_ops", authz.OrdersRead, authz.OrdersIntervene)
	_, reader := capUser(t, h, "qa_fee_reader")
	_, ops := capUser(t, h, "qa_fee_ops")
	if r := feeEditCall(h, reader, id, 700, true); r.Code != http.StatusForbidden {
		t.Fatalf("**قارئٌ بلا قدرة التدخّل عدّل الأجرة**: %s", r)
	}
	if r := feeEditCall(h, ops, id, 700, true); r.Code != http.StatusOK {
		t.Fatalf("صاحبُ القدرة رُدّ: %s", r)
	}
	if r := h.POST("/api/v1/admin/orders/"+id+"/delivery-fee", ops,
		map[string]any{"fee": 700, "reason": " "}); r.Code != http.StatusBadRequest ||
		r.Err() != "order_edit_reason_required" {
		t.Fatalf("بلا سبب: %s", r)
	}
}
