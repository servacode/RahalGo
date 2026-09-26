package qa

// ══════════════════════════════════════════════════════════════════════
// **RQ-7 — هامشُ المنصّة داخليٌّ لا يُسلسَل في تفاصيل عميل المندوب**
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٦.) **`GET /api/v1/rep/merchants/{id}`
// (`handleRepMerchantDetail`)**: كائناتُ الطلبات لا تحمل المفتاحين
// `platform_commission` ولا `forfeited_commission` (صارا `json:"-"`)، **وتبقى
// تحمل `forfeited_share` و`my_share`.** فيرى المندوبُ نصيبَه لا هامشَنا.

import (
	"strings"
	"testing"
)

func TestRepDetail_HidesInternalCommission(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()

	rep := h.NewUser("sales") // له جلسةٌ ليُصنَع إثباتُ التأكيد عند الباب إن لزم
	merchant := f.Merchant(OwnedByRep(rep.ID))
	// **صنفٌ بهامشٍ** — فيُقيَّد نصيبُ المندوب ويظهر في الجواب.
	item := h.NewItemPriced(merchant, 5000, 8000)
	withMargin(t, h, item, 3000)

	// **طلبٌ مُسلَّمٌ لهذا المتجر** — فيوجد كائنُ طلبٍ واحدٌ على الأقلّ.
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("rq7"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("create order: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))

	res := h.GET("/api/v1/rep/merchants/"+merchant.ID, rep.Token)
	if res.Code >= 400 {
		t.Fatalf("rep detail rejected: %s", res)
	}
	raw := string(res.Body)

	// **كائنُ طلبٍ حاضرٌ** — وإلّا لا شيءَ لنقيس عليه غيابَ المفاتيح.
	if !strings.Contains(raw, "forfeited_share") {
		t.Fatalf("response has no per-order objects (missing forfeited_share): %s", raw)
	}
	if !strings.Contains(raw, "my_share") {
		t.Errorf("response is missing the rep's own key my_share")
	}
	if strings.Contains(raw, "platform_commission") {
		t.Errorf("RQ-7 broken: response leaks internal key platform_commission")
	}
	if strings.Contains(raw, "forfeited_commission") {
		t.Errorf("RQ-7 broken: response leaks internal key forfeited_commission")
	}
	t.Logf("RQ-7 = PASS — forfeited_share/my_share present; platform_commission/forfeited_commission absent")
}
