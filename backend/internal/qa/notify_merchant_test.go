package qa

// ══════════════════════════════════════════════════════════════════════
//  **إرسالُ الطلب للمتجر — مفتاحٌ يُطفأ** (طلبُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// «بدي نقدر نوقفها ونقدر نشغلها، وقت تكون واقفه خلص بمجرد قبول الطلب يروح
// للسائق بدون تحويل الطلب للمتجر». **مشغَّلاً** (الافتراض): القبولُ يُبقي الطلبَ
// «مقبولاً» حتّى يُرسَل للمتجر. **ومطفأً**: القبولُ يُنزله إلى طابور السائقين.

import (
	"net/http"
	"testing"
)

func nmAcceptedStatus(t *testing.T, h *Harness, notify bool) string {
	t.Helper()
	h.Setting("customers.require_whatsapp", "false")
	h.Setting("platform.orders_mode", `"platform"`)
	h.Setting("orders.notify_merchant", map[bool]string{true: "true", false: "false"}[notify])
	// **والتحويلُ التلقائيُّ مطفأٌ صراحةً** — وإلّا قُبل الطلبُ قبل أن يقبله المكتب.
	h.Setting("orders.auto_transfer", "false")
	treasury(t, h)
	staff := h.NewUser("operations")
	item := h.NewItem(1000)
	cust := h.NewUser("customer")
	created := h.POSTKey("/api/v1/orders", cust.Token, uniq("nm"), orderBody(item, 1))
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الطلب: %s", created)
	}
	oid, _ := created.JSON()["id"].(string)
	if r := h.POST("/api/v1/admin/orders/"+oid+"/transition", staff.Token,
		map[string]any{"to": "accepted", "note": "NM"}); r.Code >= 400 {
		// **وحالُ الطلب يُكتب مع السقوط** — سقط مرّةً في التشغيل الكامل (٤٠٩) ولم يتكرّر
		// وحدَه (٢٠٢٦-١٠-١٢)؛ **فالحالُ هو ما يقول من سبق المكتبَ إلى قبوله.**
		t.Fatalf("القبول: %s — وحالُ الطلب قبله: %s", r, h.statusOf(oid))
	}
	return h.statusOf(oid)
}

func TestNM01_NotifyOnKeepsOrderForTheMerchant(t *testing.T) {
	h := New(t)
	if st := nmAcceptedStatus(t, h, true); st != "accepted" {
		t.Fatalf("**المفتاحُ مشغَّلٌ ونزل الطلبُ قبل إرساله للمتجر**: %s", st)
	}
}

func TestNM02_NotifyOffSendsStraightToDrivers(t *testing.T) {
	h := New(t)
	if st := nmAcceptedStatus(t, h, false); st != "dispatching" && st != "assigned" {
		t.Fatalf("**المفتاحُ مطفأٌ وبقي الطلبُ ينتظر المتجر**: %s", st)
	}
}
