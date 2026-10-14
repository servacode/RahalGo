package qa

import (
	"net/http"
	"strings"
	"testing"
)

// **الاستعادةُ برقمٍ بلا حسابٍ تُقال من أوّل خطوة** (قرارُ المالك ٢٠٢٦-١٠-١٤):
// كان يُفتح له الواتساب ويصله رمزٌ ثمّ تسقط الكلمةُ الجديدة بلا تفسير.
func TestResetUnknownNumberSaysNoAccount(t *testing.T) {
	hh := New(t)
	hh.Setting("platform.whatsapp", `"963900000001"`)
	unknown := uniqPhone()
	for _, c := range []struct {
		path string
		body map[string]any
	}{
		{"/api/v1/auth/password/reset/request", map[string]any{"phone": unknown}},
		{"/api/v1/auth/wa/ticket", map[string]any{"phone": unknown, "purpose": "reset"}},
	} {
		r := hh.POST(c.path, "", c.body)
		if r.Code != http.StatusNotFound || r.Err() != "no_account" {
			t.Errorf("%s برقمٍ بلا حساب ⇒ %s — والمنتظَر 404 no_account", c.path, r)
		}
	}
	// **وصاحبُ الحساب يأخذ تذكرتَه كما كان.**
	u := hh.Customer()
	if r := hh.POST("/api/v1/auth/wa/ticket", "", map[string]any{"phone": u.Phone, "purpose": "reset"}); r.Code != http.StatusOK {
		t.Fatalf("تذكرةُ استعادةٍ لصاحب حساب ⇒ %s", r)
	}
	// **والتوثيقُ لا يُسأل عن الحساب** — يُطلب قبل إنشائه أحياناً.
	if r := hh.POST("/api/v1/auth/wa/ticket", "", map[string]any{"phone": unknown, "purpose": "verify"}); r.Code != http.StatusOK {
		t.Fatalf("تذكرةُ توثيقٍ ⇒ %s", r)
	}
}

// **«انتهت صلاحية الطلب» لا تُقال قبل العشر دقائق** (بلاغُ المالك ٢٠٢٦-١٠-١٤):
// وصلت الرسالةُ مرّتين فجاء الرمزُ ثمّ «انتهت» بعده بثوانٍ.
func TestWAInboundRepeatWithinTTLIsNotExpired(t *testing.T) {
	hh := New(t)
	u := hh.Customer()
	tag, err := hh.Identity.WATicket(ctxBG(), u.Phone, "reset")
	if err != nil {
		t.Fatal(err)
	}
	from := "963911111111"
	first := hh.Identity.HandleWAInbound(ctxBG(), from, "أريد استعادة كلمة المرور\nرمز الطلب: "+tag)
	if !strings.Contains(first, "رمز رحال غو") {
		t.Fatalf("الرسالةُ الأولى ⇒ %q", first)
	}
	again := hh.Identity.HandleWAInbound(ctxBG(), from, "رمز الطلب: "+tag)
	if again != first {
		t.Fatalf("**الرسالةُ نفسُها مرّةً ثانية ⇒ %q** — والمنتظَر الرمزُ نفسُه: %q", again, first)
	}
	// **وغيرُ المرسِل الأوّل لا يأخذ رمزاً** — الوسمُ المسرَّب لا يُعطي شيئاً.
	if other := hh.Identity.HandleWAInbound(ctxBG(), "963922222222", "رمز الطلب: "+tag); other != "" {
		t.Fatalf("مرسِلٌ آخرُ بالوسم نفسِه ⇒ %q", other)
	}
	// **وبعد انقضاء التذكرة تُقال «انتهت».**
	hh.Redis().Del(ctxBG(), "wa:ticket:"+tag)
	if late := hh.Identity.HandleWAInbound(ctxBG(), from, "رمز الطلب: "+tag); !strings.Contains(late, "انتهت") {
		t.Fatalf("بعد انقضائها ⇒ %q", late)
	}
}
