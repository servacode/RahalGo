package qa

// ══════════════════════════════════════════════════════════════════════
// **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦
// ══════════════════════════════════════════════════════════════════════
//
// بضاعةٌ ١٠٠٠ · عمولةُ المتجر ١٠٪ = ١٠٠ · توصيلٌ ٥٠٠ · المتجرُ «نقداً» يدفعه السائقُ عند
// الاستلام (٩٠٠) · مندوبٌ نصيبُه ١٠٪ من العمولة = ١٠. **فصندوقُ السائق ٦٠٠ = ٥٠٠ + ١٠٠.**
// والمتجرُ رفض أن يخصم العمولة وأخذ ١٠٠٠ — **فتُمحى: صندوقُه ٥٠٠، والعمولةُ صفر، ونصيبُ
// المندوب يُعكَس، والخزينةُ تنقص ٩٠ (١٠٠ − ١٠)** — والثوابتُ خضراء.

import (
	"testing"
	"time"
)

var waiverFamilies = []string{"FI-01", "FI-02", "FI-05", "FI-06", "FI-07", "FI-10", "FI-12", "FI-13", "FI-14", "FI-15"}

func waiverSettings(h *Harness) {
	h.Setting("merchants.commission_percent", "10")
	h.Setting("sales.commission_percent", "10")
	h.Setting("pricing.margin_fixed", "0")
	h.Setting("delivery.fee", "500")
	h.Setting("delivery.by_distance", "false")
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)
}

// waiverOrder **طلبٌ نقديٌّ مسلَّمٌ من متجرٍ بطريقة تسويةٍ ومندوب.**
func waiverOrder(t *testing.T, h *Harness, method string) (oid string, drv *User, rep *Rep, m *Merchant) {
	t.Helper()
	f := h.Factory()
	rep = f.RepAccount()
	m = f.Merchant(OwnedByRep(rep.ID), Commission(10))
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = $2 WHERE id = $1::uuid`,
		m.ID, method); err != nil {
		t.Fatal(err)
	}
	item := h.NewItemFor(m, 1000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("waive"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	drv = h.driverOf(oid)
	deliverOrder(t, h, oid, drv)
	return oid, drv, rep, m
}

func cashHeld(t *testing.T, h *Harness, driverID string) int64 {
	t.Helper()
	var v int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE((SELECT held FROM driver_cash_boxes WHERE driver_id = $1::uuid), 0)`,
		driverID).Scan(&v)
	return v
}

func orderLedgerCount(t *testing.T, h *Harness, oid string) (n int) {
	t.Helper()
	_ = h.Pool.QueryRow(ctxBG(), `SELECT (SELECT count(*) FROM wallet_transactions WHERE ref = $1)
		+ (SELECT count(*) FROM driver_cash_entries WHERE ref = $1)`, oid).Scan(&n)
	return n
}

// statementEntry **سطرُ كشف الصندوق لنوعٍ وطلب** — كما تراه اللوحة.
func statementEntry(t *testing.T, h *Harness, token, driverID, kind, oid string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/admin/drivers/"+driverID+"/cash", token)
	if r.Code != 200 {
		t.Fatalf("الكشف: %s", r)
	}
	entries, _ := r.JSON()["entries"].([]any)
	for _, e := range entries {
		row, _ := e.(map[string]any)
		if row["kind"] == kind && row["ref"] == oid {
			return row
		}
	}
	t.Fatalf("لا سطرَ %s للطلب في الكشف: %s", kind, r)
	return nil
}

func num(v any) int64 {
	f, _ := v.(float64)
	return int64(f)
}

func TestWAIVE_CommissionWaivedEverywhere(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	base := financialBaseline(t, h, waiverFamilies...)
	fin := h.NewUser("finance")

	oid, drv, rep, _ := waiverOrder(t, h, "cash")

	var commission, cashDue int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT platform_commission, cash_due FROM orders WHERE id = $1`, oid).
		Scan(&commission, &cashDue)
	if commission != 100 || cashDue != 1500 {
		t.Fatalf("الطلبُ قبل المحو: عمولة %d نقد %d — يُنتظر ١٠٠ و١٥٠٠", commission, cashDue)
	}
	heldBefore := cashHeld(t, h, drv.ID)
	if heldBefore != 600 {
		t.Fatalf("صندوقُ السائق %d — يُنتظر ٦٠٠ (توصيلٌ ٥٠٠ + عمولةٌ ١٠٠)", heldBefore)
	}
	repBefore := repCommissionOf(t, h, oid)
	if repBefore != 10 {
		t.Fatalf("نصيبُ المندوب %d — يُنتظر ١٠", repBefore)
	}
	repWallet := walletBalance(t, h, rep.ID)
	profitBefore := ledgerByRef(t, h, oid, "platform_profit")

	// **الكشفُ قبل المحو**: الأجرةُ ٥٠٠ + العمولةُ ١٠٠ = التحصيل.
	pre := statementEntry(t, h, fin.Token, drv.ID, "order_collection", oid)
	if num(pre["fee_part"])+num(pre["commission_part"]) != num(pre["amount"]) ||
		num(pre["fee_part"]) != 500 || num(pre["commission_part"]) != 100 || num(pre["commission_waived"]) != 0 {
		t.Errorf("**تفصيلُ التحصيل قبل المحو**: %v", pre)
	}

	// السببُ إلزاميّ.
	if r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "   "}); r.Code != 400 || r.Err() != "commission_waiver_reason_required" {
		t.Fatalf("محوٌ بلا سبب: %s", r)
	}

	r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر رفض يخصم العمولة"})
	if r.Code != 200 {
		t.Fatalf("المحو: %s", r)
	}
	body := r.JSON()
	if num(body["waived"]) != 100 || num(body["driver_held"]) != 500 || body["order_id"] != oid || num(body["number"]) == 0 {
		t.Errorf("ردُّ المحو: %v", body)
	}

	// (١) السائق: نقص بالعمولة بالضبط.
	if got := cashHeld(t, h, drv.ID); got != heldBefore-100 {
		t.Errorf("**صندوقُ السائق %d — يُنتظر %d**", got, heldBefore-100)
	}
	// (٢) الطلب: العمولةُ صفر والمحوُ مكتوبٌ بمن ومتى ولماذا.
	var pc, waived int64
	var by, reason *string
	var at *time.Time
	_ = h.Pool.QueryRow(ctxBG(), `SELECT platform_commission, commission_waived, commission_waived_by::text,
		commission_waived_reason, commission_waived_at FROM orders WHERE id = $1`, oid).
		Scan(&pc, &waived, &by, &reason, &at)
	if pc != 0 || waived != 100 || by == nil || *by != fin.ID || reason == nil || *reason != "المتجر رفض يخصم العمولة" || at == nil {
		t.Errorf("الطلبُ بعد المحو: عمولة %d محو %d من %v سبب %v وقت %v", pc, waived, by, reason, at)
	}
	// (٣) المندوب: نصيبُه عُكس.
	if got := repCommissionOf(t, h, oid); got != 0 {
		t.Errorf("**صافي عمولة المندوب على الطلب %d — يُنتظر صفر**", got)
	}
	if got := walletBalance(t, h, rep.ID); got != repWallet-10 {
		t.Errorf("محفظةُ المندوب %d — يُنتظر %d", got, repWallet-10)
	}
	// (٤) المتجر: قبض العمولةَ نقداً من السائق.
	var msWaived int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(commission_waived), 0) FROM merchant_settlements WHERE order_id = $1`, oid).Scan(&msWaived)
	if msWaived != 100 {
		t.Errorf("صفُّ تسوية المتجر يحمل %d — يُنتظر ١٠٠", msWaived)
	}
	// (٥) الخزينة: −١٠٠ عمولة + ١٠ نصيبُ المندوب.
	if got := ledgerByRef(t, h, oid, "platform_profit"); got != profitBefore-90 {
		t.Errorf("**ربحُ المنصّة من الطلب %d — يُنتظر %d**", got, profitBefore-90)
	}

	// (٦) الكشف: الحقولُ الجديدة على سطر التحصيل وسطر المحو.
	col := statementEntry(t, h, fin.Token, drv.ID, "order_collection", oid)
	if num(col["fee_part"]) != 500 || num(col["commission_part"]) != 100 || num(col["commission_waived"]) != 100 ||
		col["waived_reason"] != "المتجر رفض يخصم العمولة" || col["order_id"] != oid ||
		col["customer_name"] == "" || col["customer_name"] == nil || col["merchant_name"] == "" || col["merchant_name"] == nil ||
		num(col["free_waived"]) != 0 || num(col["order_number"]) == 0 {
		t.Errorf("**سطرُ التحصيل في الكشف**: %v", col)
	}
	wv := statementEntry(t, h, fin.Token, drv.ID, "commission_waived", oid)
	if num(wv["amount"]) != -100 || wv["note"] != "المتجر رفض يخصم العمولة" || num(wv["commission_waived"]) != 100 {
		t.Errorf("**سطرُ المحو في الكشف**: %v", wv)
	}

	// (٧) ثانيةً: تُرفض ولا يُقيَّد شيء.
	n := orderLedgerCount(t, h, oid)
	if r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "مرة ثانية"}); r.Code != 409 || r.Err() != "commission_already_waived" {
		t.Errorf("المحوُ الثاني: %s", r)
	}
	if got := orderLedgerCount(t, h, oid); got != n {
		t.Errorf("**المحوُ الثاني قيّد %d سطراً**", got-n)
	}
	if got := cashHeld(t, h, drv.ID); got != heldBefore-100 {
		t.Errorf("المحوُ الثاني حرّك الصندوق: %d", got)
	}

	// (٨) والتدقيق.
	var audits int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log WHERE action = 'finance.commission_waived' AND entity_id = $1`, oid).Scan(&audits)
	if audits != 1 {
		t.Errorf("سطورُ التدقيق %d — يُنتظر ١", audits)
	}

	assertNewViolations(t, h, base, waiverFamilies...)
}

// **ومتجرُ «محفظة» لا تقع فيه الحالة** — العمولةُ خُصمت من مستحقّه.
func TestWAIVE_WalletStoreRejected(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	fin := h.NewUser("finance")
	oid, drv, _, _ := waiverOrder(t, h, "wallet")
	held := cashHeld(t, h, drv.ID)
	n := orderLedgerCount(t, h, oid)
	r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token, map[string]any{"reason": "سبب"})
	if r.Code != 409 || r.Err() != "commission_waiver_wallet_store" {
		t.Fatalf("متجرُ محفظة: %s", r)
	}
	if orderLedgerCount(t, h, oid) != n || cashHeld(t, h, drv.ID) != held {
		t.Errorf("**الرفضُ حرّك مالاً**")
	}
}

// **والتوصيلُ المجانيّ**: الكشفُ يقول «توصيل: مجاني ٠» والتحصيلُ عمولةٌ وحدَها — ثمّ تُمحى.
func TestWAIVE_FreeDeliveryStatement(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	waiverSettings(h)
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns`) })
	now := time.Now().UTC()
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO free_delivery_campaigns (active, starts_at, ends_at, kinds)
		VALUES (true, $1, $2, ARRAY['standard'])`, now.Add(-time.Hour), now.Add(24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	base := financialBaseline(t, h, waiverFamilies...)
	fin := h.NewUser("finance")
	oid, drv, _, _ := waiverOrder(t, h, "cash")

	col := statementEntry(t, h, fin.Token, drv.ID, "order_collection", oid)
	if num(col["fee_part"]) != 0 || num(col["free_waived"]) != 500 || num(col["commission_part"]) != 100 ||
		num(col["amount"]) != 100 {
		t.Fatalf("**كشفُ طلبٍ مجّانيّ التوصيل**: %v", col)
	}
	held := cashHeld(t, h, drv.ID)
	if r := h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر أخذ الثمن كامل"}); r.Code != 200 {
		t.Fatalf("المحو: %s", r)
	}
	if got := cashHeld(t, h, drv.ID); got != held-100 {
		t.Errorf("الصندوق %d — يُنتظر %d", got, held-100)
	}
	assertNewViolations(t, h, base, waiverFamilies...)
}
