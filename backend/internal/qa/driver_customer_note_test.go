package qa

/*
**«ملاحظات للسائق» بتوصل الكابتن** (قرارُ المالك ٢٠٢٦-١١-٢٠: «للكابتن والمتجر»).

كانت الخانةُ بشاشة الطلب الخاص وبالسلّة تُكتب ولا تصل — **الخادمُ كان يرسلها للكابتن في
«لدي توصيلة» وحدَها.** وفحصُ التجربة شافها غايبة. وهون: بالعاديّ والخاصّ بتوصل، **ورقمُ
الجوال فيها ينشال** متل نصّ الطلب.
*/

import (
	"encoding/json"
	"strings"
	"testing"
)

const noteText = "جيب النوع الكبير ورن الجرس مرتين"
const notePhone = "0991234567"

func driverNoteOf(t *testing.T, h *Harness, drv *User, oid string) (string, bool) {
	t.Helper()
	r := h.GET("/api/v1/driver/orders", drv.Token)
	if r.Code != 200 {
		t.Fatalf("طلبات الكابتن: %s", r)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("فكّ طلبات الكابتن: %v", err)
	}
	for _, x := range env.Data {
		if x["id"] == oid {
			n, _ := x["customer_note"].(string)
			return n, true
		}
	}
	return "", false
}

func TestDriverSeesCustomerNote(t *testing.T) {
	h := New(t)

	// ── الخاص ──
	cust := h.Customer()
	b := customBody("كيلو بندورة وخبز")
	b["notes"] = noteText + " " + notePhone
	cx := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("cn"), b)
	if cx.Code >= 400 {
		t.Fatalf("إنشاء الخاص: %s", cx)
	}
	cid, _ := cx.JSON()["id"].(string)
	drv := h.driverOf(cid)
	n, ok := driverNoteOf(t, h, drv, cid)
	if !ok {
		t.Fatalf("الطلب الخاص مو بقائمة الكابتن")
	}
	if !strings.Contains(n, noteText) {
		t.Errorf("**ملاحظة الزبون ما وصلت الكابتن بالخاص**: %q", n)
	}
	if strings.Contains(n, notePhone) {
		t.Errorf("**رقم الجوال وصل الكابتن بالملاحظة**: %q", n)
	}

	// ── العادي ──
	c2 := h.Customer()
	ob := orderBody(h.NewItem(1000), 1)
	ob["notes"] = noteText
	made := h.POSTKey("/api/v1/orders", c2.Token, uniq("cn"), ob)
	if made.Code >= 400 {
		t.Fatalf("إنشاء العادي: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	d2 := h.driverOf(oid)
	if n, ok := driverNoteOf(t, h, d2, oid); !ok || n != noteText {
		t.Errorf("**ملاحظة الزبون ما وصلت الكابتن بالعادي**: %q (موجود %v)", n, ok)
	}
}
