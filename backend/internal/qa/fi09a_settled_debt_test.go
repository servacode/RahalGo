package qa

import "testing"

// TestFIN_FI09a_SettledRepDebtIsCovered **دَينُ عمولةٍ سُدّد ليس خرقاً.**
//
// (كشفه فحصُ المتصفّح للمال ٢٠٢٦-١٠-٠٥: استُرِدّ طلبٌ بعد صرفِ عمولة
// المندوب فصار عليه دَين، ثمّ دفعه نقداً بالمكتب — فأعلن `moneycheck`
// خرقاً.)
//
// `FI-09.a` كان يقارن الباقي من عمولات الطلبات المسترَدّة بالدَّين القائم
// وحدَه. **والدَّينُ المسدَّد — نقداً أو بشحنٍ أو اقتطاعاً من عمولةٍ قادمة
// أو شطباً — غطّى العمولةَ كما يغطّيها العكس**، فنقصُ الدَّين بسداده كان
// يُقرأ عمولةً لم تُعكَس.
func TestFIN_FI09a_SettledRepDebtIsCovered(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, "FI-09.a")
	rep, item := repFixture(t, h)
	oid := placeAndDeliver(t, h, item)
	s0 := snapRep(t, h, rep.ID, oid)
	if s0.OrderNet <= 0 {
		t.Skip("لم تُقيَّد عمولة")
	}
	h.Factory().Credit(rep.ID, -s0.Balance, "payout")

	admin := h.NewUser("admin")
	if got := h.POST("/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "دَين"}); got.Code >= 400 {
		t.Fatalf("الاستردادُ سقط: %s", got)
	}
	var ob string
	var outstanding int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT id::text, amount - settled FROM financial_obligations
		 WHERE party_kind = 'rep' AND party_id = $1::uuid AND closed_at IS NULL`, rep.ID).
		Scan(&ob, &outstanding); err != nil {
		t.Fatalf("لا دَينَ على المندوب بعد الاسترداد: %v", err)
	}
	fin := h.NewUser("finance")
	r := h.POST("/api/v1/admin/obligations/"+ob+"/office-cash", fin.Token,
		map[string]any{"amount": outstanding, "note": "دفعه نقداً بالمكتب"})
	if r.Code != 201 {
		t.Fatalf("اقتراحُ الدفع: %s", r)
	}
	reqID, _ := r.JSON()["request_id"].(string)
	if ok := h.POST("/api/v1/admin/obligation-requests/"+reqID+"/approve", admin.Token,
		map[string]any{}); ok.Code != 200 {
		t.Fatalf("الموافقة: %s", ok)
	}
	if s := snapRep(t, h, rep.ID, oid); s.Debt != 0 {
		t.Fatalf("الدَّينُ بعد السداد %d", s.Debt)
	}
	assertNewViolations(t, h, base, "FI-09.a")
}
