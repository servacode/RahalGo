package qa

// **صاحبُ المتجر يؤكّد طريقةَ مستحقّاته بنفسه** — قرارُ المالك ٢٠٢٦-١٠-٠١.
//
// **ومنذ ٢٠٢٦-١١-١٠ (القرار ٩) لا طريقةَ إلّا «نقد»**: «محفظة» تُردّ برمزها
// (`settlement_wallet_removed_test.go`)، **فيبقى البابُ لتأكيد «نقد»** — والتدقيقُ
// وحدودُ المِلكيّة والإيقاف كما كانت.

import (
	"net/http"
	"testing"
)

func TestMSS01_OwnerSwitchesSettlementMethod(t *testing.T) {
	h := New(t)
	f := h.Factory()
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	other := f.Merchant(OwnedByRep(rep.ID))
	tok := h.TokenFor(m.Owner.ID, "merchant")
	path := "/api/v1/merchant/stores/" + m.ID + "/settlement-method"

	// **مِسنَدُ المتاجر يصنعها «محفظة»** (طلباتٌ قديمة) — **وتأكيدُ صاحبه يجعلها «نقد».**
	r := h.Call("PATCH", path, tok, map[string]any{"method": "cash"}, nil)
	if r.Code != http.StatusOK || r.JSON()["settlement_method"] != "cash" {
		t.Fatalf("التأكيدُ «نقد»: %d / %s", r.Code, r.Err())
	}
	var method string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT settlement_method FROM merchants WHERE id = $1`, m.ID).Scan(&method); err != nil || method != "cash" {
		t.Fatalf("لم يُكتب: %q %v", method, err)
	}
	if r := h.Call("PATCH", path, tok, map[string]any{"method": "wallet"}, nil); r.Code != http.StatusBadRequest {
		t.Fatalf("**«محفظة» مرّت بعد إلغائها**: %d", r.Code)
	}
	var audited int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log WHERE action = 'merchant.settlement_update' AND entity_id = $1`, m.ID).Scan(&audited)
	if audited == 0 {
		t.Fatalf("**بدّلها صاحبُها ولا قيدَ في سجلّ المراقبة**")
	}
	if r := h.Call("PATCH", path, tok, map[string]any{"method": "gold"}, nil); r.Code != http.StatusBadRequest {
		t.Fatalf("طريقةٌ مخترعة: %d", r.Code)
	}
	// **ولا يبدّل لمتجرٍ ليس له.**
	if r := h.Call("PATCH", "/api/v1/merchant/stores/"+other.ID+"/settlement-method", tok, map[string]any{"method": "cash"}, nil); r.Code < 400 || r.Code >= 500 {
		t.Fatalf("**بدّل طريقةَ متجرٍ ليس له**: %d", r.Code)
	}
	// **والموقوفُ لا يبدّل.**
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET status = 'suspended' WHERE id = $1`, m.ID); err != nil {
		t.Fatal(err)
	}
	if r := h.Call("PATCH", path, tok, map[string]any{"method": "cash"}, nil); r.Code < 400 || r.Code >= 500 {
		t.Fatalf("**الموقوفُ بدّل طريقتَه**: %d", r.Code)
	}
}
