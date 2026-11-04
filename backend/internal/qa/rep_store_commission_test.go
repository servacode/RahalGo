package qa

// ══════════════════════════════════════════════════════════════════════
// **المندوبُ يضبط نسبةَ عمولة متجر عميله** (قرارُ المالك ٢٠٢٦-١١-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// «بين ال0 وال10 بالمية» — فوراً بلا موافقة المكتب، وبحدَّين من الإعدادات،
// وكلُّ تبديلٍ مقيَّدٌ بقيمتيه. **والطلبُ المسلَّمُ لا يُمسّ، والمفتوحُ يُسوّى بالحاليّة**
// (الخيار الأوّل — كتعديل الأدمن). **ولا يصل المندوبَ هامشُ المنصّة** (`RQ-7`).

import (
	"encoding/json"
	"strings"
	"testing"
)

// commissionChanges أحداثُ `merchant.commission_change` لمتجرٍ بقيمتيها، الأحدثُ آخراً.
func commissionChanges(t *testing.T, h *Harness, merchantID string) []map[string]any {
	t.Helper()
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT actor_user_id::text, details FROM audit_log
		 WHERE action = 'merchant.commission_change' AND entity_id = $1
		 ORDER BY created_at, id`, merchantID)
	if err != nil {
		t.Fatalf("قراءةُ التدقيق: %v", err)
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var actor string
		var raw []byte
		if err := rows.Scan(&actor, &raw); err != nil {
			t.Fatalf("فكُّ التدقيق: %v", err)
		}
		d := map[string]any{}
		_ = json.Unmarshal(raw, &d)
		d["actor"] = actor
		out = append(out, d)
	}
	return out
}

func merchantCommissionText(t *testing.T, h *Harness, merchantID string) string {
	t.Helper()
	var v *string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT commission_percent::text FROM merchants WHERE id = $1::uuid`, merchantID).Scan(&v); err != nil {
		t.Fatalf("قراءةُ نسبة المتجر: %v", err)
	}
	if v == nil {
		return ""
	}
	return *v
}

func TestRepStoreCommission_SetBoundsAuditAndIDOR(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "10")
	f := h.Factory()
	rep := h.NewUser("sales")
	other := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	path := "/api/v1/rep/stores/" + m.ID + "/commission"

	// **٧٫٥ تُقبل وتسري فوراً** — في الحقلين كما يقرؤهما الأدمن.
	res := h.PATCH(path, rep.Token, map[string]any{"percent": 7.5})
	if res.Code != 200 {
		t.Fatalf("rep set 7.5: %s", res)
	}
	if got := string(rawData(res)); !strings.Contains(got, `"store_commission_percent":7.5`) {
		t.Errorf("response = %s — want store_commission_percent 7.5", got)
	}
	if got := merchantCommissionText(t, h, m.ID); got != "7.50" {
		t.Errorf("merchants.commission_percent = %q — want 7.50", got)
	}
	admin := h.NewUser("admin")
	am := h.GET("/api/v1/admin/merchants/"+m.ID, admin.Token)
	var amv map[string]json.RawMessage
	_ = json.Unmarshal(rawData(am), &amv)
	if string(amv["commission_percent_exact"]) != "7.5" || string(amv["commission_percent"]) != "8" {
		t.Errorf("admin view exact=%s rounded=%s — want 7.5 / 8", amv["commission_percent_exact"], amv["commission_percent"])
	}

	// **والتدقيقُ بقيمتيه وفاعله**.
	ch := commissionChanges(t, h, m.ID)
	if len(ch) != 1 {
		t.Fatalf("commission_change rows = %d — want 1: %v", len(ch), ch)
	}
	if ch[0]["actor"] != rep.ID || ch[0]["by"] != "rep" || ch[0]["from"] != nil || ch[0]["to"] != 7.5 {
		t.Errorf("audit = %v — want actor=rep by=rep from=nil to=7.5", ch[0])
	}

	// **خارجَ الحدّين يُردّ** — ولا يتغيّر شيء.
	for _, bad := range []any{10.01, -1, 11} {
		r := h.PATCH(path, rep.Token, map[string]any{"percent": bad})
		if r.Code != 400 || r.Err() != "rep_commission_out_of_range" {
			t.Errorf("percent %v: %s — want 400 rep_commission_out_of_range", bad, r)
		}
	}
	// **وثلاثُ منازل تُردّ عند الفكّ** — بدقّة حقل الأدمن.
	if r := h.PATCH(path, rep.Token, map[string]any{"percent": 7.555}); r.Code != 400 {
		t.Errorf("percent 7.555: %s — want 400", r)
	}
	if got := merchantCommissionText(t, h, m.ID); got != "7.50" {
		t.Errorf("after rejects commission = %q — want 7.50", got)
	}
	// **والحدّان طرفاه مقبولان.**
	if r := h.PATCH(path, rep.Token, map[string]any{"percent": 10}); r.Code != 200 {
		t.Errorf("percent 10: %s — want 200", r)
	}
	if r := h.PATCH(path, rep.Token, map[string]any{"percent": 0}); r.Code != 200 {
		t.Errorf("percent 0: %s — want 200", r)
	}

	// **والحدُّ إعدادٌ يغيّره المالك.**
	h.Setting("sales.rep_commission_max", "5")
	if r := h.PATCH(path, rep.Token, map[string]any{"percent": 6}); r.Err() != "rep_commission_out_of_range" {
		t.Errorf("max=5, percent 6: %s — want rep_commission_out_of_range", r)
	}

	// **ومندوبٌ آخر ٤٠٣** — ولا يتغيّر شيء.
	if r := h.PATCH(path, other.Token, map[string]any{"percent": 3}); r.Code != 403 {
		t.Errorf("other rep: %s — want 403", r)
	}
	if got := merchantCommissionText(t, h, m.ID); got != "0.00" {
		t.Errorf("after IDOR commission = %q — want 0.00", got)
	}

	// **وتعديلُ الأدمن يُقيَّد بقيمتيه كذلك** — مساعدٌ واحدٌ للطريقين.
	if r := h.PATCH("/api/v1/admin/merchants/"+m.ID, admin.Token,
		map[string]any{"commission_percent": 2.25}); r.Code != 200 {
		t.Fatalf("admin set 2.25: %s", r)
	}
	ch = commissionChanges(t, h, m.ID)
	last := ch[len(ch)-1]
	if last["actor"] != admin.ID || last["by"] != "admin" || last["from"] != 0.0 || last["to"] != 2.25 {
		t.Errorf("admin audit = %v — want by=admin from=0 to=2.25", last)
	}
}

func TestRepStoreCommission_DeliveredOrderUnchanged_NoProfitLeak(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "10")
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	item := h.NewItemPriced(m, 10_000, 13_000)
	withMargin(t, h, item, 3000)

	// **طلبٌ مسلَّمٌ بعشرةٍ عامّة** — ١٠٬٠٠٠ × ١٠٪ = ١٬٠٠٠.
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("repc"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("create order: %s", made)
	}
	delivered, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, delivered, h.driverOf(delivered))
	before, earnBefore, _ := decimalLedger(t, h, delivered)
	if before != 1000 {
		t.Fatalf("platform_commission before = %d — want 1000", before)
	}

	// **وطلبٌ مفتوحٌ قبل التعديل** — يُسوّى بالحاليّة عند التسليم (الخيار الأوّل).
	made2 := h.POSTKey("/api/v1/orders", cust.Token, uniq("repc2"), orderBody(item, 1))
	if made2.Code >= 400 {
		t.Fatalf("create open order: %s", made2)
	}
	open, _ := made2.JSON()["id"].(string)

	if r := h.PATCH("/api/v1/rep/stores/"+m.ID+"/commission", rep.Token,
		map[string]any{"percent": 7.5}); r.Code != 200 {
		t.Fatalf("rep set 7.5: %s", r)
	}

	after, earnAfter, _ := decimalLedger(t, h, delivered)
	if after != before || earnAfter != earnBefore {
		t.Errorf("delivered order changed: commission %d→%d earning %d→%d", before, after, earnBefore, earnAfter)
	}
	var settled int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT amount FROM merchant_settlements WHERE order_id = $1::uuid`, delivered).Scan(&settled); err != nil {
		t.Fatalf("settlement: %v", err)
	}
	if settled != 9000 {
		t.Errorf("delivered settlement = %d — want 9000", settled)
	}

	deliverOrder(t, h, open, h.driverOf(open))
	if got, _, _ := decimalLedger(t, h, open); got != 750 {
		t.Errorf("open order settled commission = %d — want 750 (7.5%% at delivery)", got)
	}

	// **وتفصيلُ المندوب يحمل نسبةَ المتجر وحدّيها — لا هامشَ المنصّة.**
	res := h.GET("/api/v1/rep/merchants/"+m.ID, rep.Token)
	if res.Code >= 400 {
		t.Fatalf("rep detail: %s", res)
	}
	raw := string(res.Body)
	for _, want := range []string{`"store_commission_percent":7.5`, `"store_commission_min":0`, `"store_commission_max":10`} {
		if !strings.Contains(raw, want) {
			t.Errorf("rep detail missing %s: %s", want, raw)
		}
	}
	for _, leak := range []string{"platform_commission", "forfeited_commission", "margin", "commission_taken", "\"commission_percent\"", "commission_percent_exact"} {
		if strings.Contains(raw, leak) {
			t.Errorf("RQ-7: rep detail leaks %s", leak)
		}
	}
}
