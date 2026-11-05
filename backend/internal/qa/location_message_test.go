package qa

// **رسالةُ طلب الموقع واحدةٌ لكلّ تدفّق** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ج) — نصُّ المالك حرفاً.

import (
	"net/http"
	"testing"
)

// waLocText **نصُّ المالك حرفاً** (ج) — والرابطُ آخرُه.
func waLocText(link string) string {
	return "السلام عليكم ورحمة الله وبركاته\n" +
		"معك تطبيق رحّال غو للتسوق وطلبات التوصيل بالرقة\n" +
		"ياريت تشارك موقعك أو ترسله عبر الواتساب لنقدر نوصل إليك بسهولة:\n" +
		"ردّ على هالرسالة بموقعك: اضغط 📎 ← الموقع ← أرسل موقعك الحالي\n" +
		"أو من هون: " + link
}

// TestLocReqOneMessageForEveryFlow **ج · رسالةٌ واحدةٌ لكلّ طلب موقع** — توصيلةُ المتجر و«إحضار»
// والأمانةُ وطلبُ الواتساب: نصُّ المالك حرفاً، في الواتساب وفي «ابعته بنفسك».
//
// **وعلى الشيفرة القديمة** كانت لكلّ تدفّقٍ جملتُه («مرحباً 👋 متجر «…» بدو موقعك…»).
func TestLocReqOneMessageForEveryFlow(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ LOC-ONE")
	n := armFeedback(t, h, true)
	cust := h.Customer()

	type flow struct {
		name, path, tok string
		body            map[string]any
	}
	flows := []flow{
		{"توصيلة المتجر", w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"mode": "send"}},
		{"إحضار", w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"mode": "fetch"}},
		{"الأمانة", "/api/v1/my/location-requests", cust.Token, map[string]any{}},
		{"طلب الواتساب", waURL + "/location-requests", w.md.admin.Token, map[string]any{"store_name": "فلافل الأمير"}},
	}
	for _, f := range flows {
		phone := uniqPhone()
		f.body["phone"] = phone
		r := h.POST(f.path, f.tok, f.body)
		if r.Code != http.StatusCreated {
			t.Errorf("**%s**: %d / %s", f.name, r.Code, r.Err())
			continue
		}
		link, _ := r.JSON()["link"].(string)
		want := waLocText(link)
		if got, _ := r.JSON()["share_text"].(string); got != want {
			t.Errorf("**%s — «ابعته بنفسك»**:\n%q\nوالمنتظَر\n%q", f.name, got, want)
		}
		msgs := n.to(phone)
		if len(msgs) != 1 || msgs[0].text != want {
			t.Errorf("**%s — ما وصل الواتساب**: %+v", f.name, msgs)
		}
	}
}
