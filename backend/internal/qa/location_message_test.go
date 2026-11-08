package qa

// **رسالةُ طلب الموقع واحدةٌ لكلّ تدفّق** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ج) — نصُّ المالك حرفاً.
// **وبلا رابطٍ ولا اسمِ متجر** (قرارُ المالك ٢٠٢٦-١١-٠٦): «الناس بسوريا تخاف من روابط الهكر».

import (
	"net/http"
	"strings"
	"testing"
)

// waLocText **نصُّ المالك** (٢٠٢٦-١١-٠٦) — سطرُ «لديك أمانة/طلب» بعد السلام، ولا رابط.
func waLocText(amanah bool) string {
	what := "لديك طلب"
	if amanah {
		what = "لديك أمانة"
	}
	return "السلام عليكم ورحمة الله وبركاته\n" +
		what + " من تطبيق رحّال غو للتسوق وطلبات التوصيل بالرقة\n" +
		"ياريت تشارك موقعك أو ترسله عبر الواتساب لنقدر نوصل إليك بسهولة:\n" +
		"ردّ على هالرسالة بموقعك: اضغط 📎 ← الموقع ← أرسل موقعك الحالي أو المباشر"
}

// waLocTextWA **رسالةُ طلب الواتساب** — نصُّ المالك وحدَه، بلا سطر عنوان (قرارُ المالك ٢٠٢٦-١١-٠٨:
// «فقط الموقع وبس»).
func waLocTextWA() string {
	return waLocText(false)
}

// TestLocReqOneMessageForEveryFlow **ج · رسالةٌ واحدةٌ لكلّ طلب موقع** — توصيلةُ المتجر و«إحضار»
// والأمانةُ وطلبُ الواتساب: نصُّ المالك حرفاً، في الواتساب وفي «ابعته بنفسك».
//
// **وعلى الشيفرة القديمة** كان آخرُها «أو من هون: <رابط>» ولا سطرَ «لديك طلب/أمانة».
func TestLocReqOneMessageForEveryFlow(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ LOC-ONE")
	n := armFeedback(t, h, true)
	cust := h.Customer()

	type flow struct {
		name, path, tok string
		body            map[string]any
		amanah          bool
	}
	flows := []flow{
		{"توصيلة المتجر", w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"mode": "send"}, false},
		{"إحضار", w.md.url("/location-requests"), w.md.fx.Tok, map[string]any{"mode": "fetch"}, false},
		{"الأمانة", "/api/v1/my/location-requests", cust.Token, map[string]any{}, true},
		{"طلب الواتساب", waURL + "/location-requests", w.md.admin.Token, map[string]any{"store_name": "فلافل الأمير"}, false},
	}
	for _, f := range flows {
		phone := uniqPhone()
		f.body["phone"] = phone
		r := h.POST(f.path, f.tok, f.body)
		if r.Code != http.StatusCreated {
			t.Errorf("**%s**: %d / %s", f.name, r.Code, r.Err())
			continue
		}
		want := waLocText(f.amanah)
		if f.path == waURL+"/location-requests" {
			want = waLocTextWA()
		}
		if got, _ := r.JSON()["share_text"].(string); got != want {
			t.Errorf("**%s — «ابعته بنفسك»**:\n%q\nوالمنتظَر\n%q", f.name, got, want)
		}
		msgs := n.to(phone)
		if len(msgs) != 1 || msgs[0].text != want {
			t.Errorf("**%s — ما وصل الواتساب**: %+v", f.name, msgs)
		}
	}
}

// TestLocReqNoLinkNoStoreName **لا رابطَ ولا اسمَ متجرٍ فيما يصل المستلِم** (قرارُ المالك
// ٢٠٢٦-١١-٠٦) — رسالةُ الواتساب و«ابعته بنفسك» وعنوانُ صفحة `/loc` القديمة.
//
// **وعلى الشيفرة القديمة**: الرسالةُ فيها `http`، والصفحةُ تعرض اسمَ المتجر أو الزبون أو المكتوب.
func TestLocReqNoLinkNoStoreName(t *testing.T) {
	h := New(t)
	w := newWAFx(t, h, "منطقةُ LOC-NOLINK")
	n := armFeedback(t, h, true)
	cust := h.Customer()
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET full_name = 'زبونٌ اسمُه مميّز' WHERE id = $1`, cust.ID); err != nil {
		t.Fatal(err)
	}

	flows := []struct {
		name, path, tok string
		body            map[string]any
	}{
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
		id, _ := r.JSON()["id"].(string)
		var mName, cName, label string
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT COALESCE(m.name, ''), COALESCE(cu.full_name, ''), COALESCE(l.store_label, '')
			  FROM recipient_location_requests l
			  LEFT JOIN merchants m ON m.id = l.merchant_id
			  LEFT JOIN users cu ON cu.id = l.customer_id
			 WHERE l.id = $1`, id).Scan(&mName, &cName, &label); err != nil {
			t.Fatal(err)
		}
		var names []string
		for _, s := range []string{mName, cName, label} {
			if strings.TrimSpace(s) != "" {
				names = append(names, s)
			}
		}
		if len(names) == 0 {
			t.Fatalf("**%s**: لا اسمَ يُفحص — الاختبارُ لا يقيس شيئاً", f.name)
		}
		var texts []string
		if s, _ := r.JSON()["share_text"].(string); s != "" {
			texts = append(texts, s)
		}
		for _, m := range n.to(phone) {
			texts = append(texts, m.text)
		}
		if len(texts) != 2 {
			t.Errorf("**%s**: نصوصٌ ناقصة %d", f.name, len(texts))
		}
		for _, txt := range texts {
			if strings.Contains(txt, "http") || strings.Contains(txt, "/loc/") {
				t.Errorf("**%s — رابطٌ في الرسالة**: %q", f.name, txt)
			}
			for _, nm := range names {
				if strings.Contains(txt, nm) {
					t.Errorf("**%s — اسمُ «%s» في الرسالة**: %q", f.name, nm, txt)
				}
			}
		}
		// **والصفحةُ القديمة باقية** (رسائلُ وصلت قبل القرار) — **باسم المنصّة وحدَه.**
		link, _ := r.JSON()["link"].(string)
		i := strings.LastIndex(link, "/loc/")
		if i < 0 {
			t.Errorf("**%s**: لا رابطَ للصفحة القديمة في الردّ: %q", f.name, link)
			continue
		}
		pub := h.GET("/api/v1/public/location-requests/"+link[i+len("/loc/"):], "")
		if pub.Code != http.StatusOK || pub.JSON()["store_name"] != "رحّال غو" {
			t.Errorf("**%s — صفحةُ /loc**: %d / %v", f.name, pub.Code, pub.JSON()["store_name"])
		}
	}
}

// TestWALocationLiveReplyAppliedOnce **الموقعُ المباشر** (٢٠٢٦-١١-٠٦): أوّلُ تحديثٍ يملأ الطلبَ
// كالموقع العاديّ، **وما بعده** (صاحبُه يتحرّك) لا يردّ ولا يغيّر النقطةَ ولا يمسّ الطلبَ المربوط.
func TestWALocationLiveReplyAppliedOnce(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	waID, _ := requestHolderLocation(t, h, cust.Token, "0944555630")
	b := amanahBody("أمانة: مفتاح")
	b["holder_phone"] = "0944555630"
	b["location_request_id"] = waID
	created := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("wal-live"), b)
	if created.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ الأمانة: %s", created)
	}
	orderID, _ := created.JSON()["id"].(string)
	drv := h.NewUser("driver")
	assignDriver(t, h, orderID, drv.ID)

	// أوّلُ تحديثٍ مباشر.
	if r := h.API.HandleWALocation(ctxBG(), "+963944555630", amPLat, amPLng, 12); r == "" {
		t.Fatal("أوّلُ تحديثٍ مباشرٍ لم يُقبل")
	}
	// تحديثاتٌ لاحقة بنقاطٍ أخرى — كلُّها صامتة.
	for i := 1; i <= 5; i++ {
		if r := h.API.HandleWALocation(ctxBG(), "+963944555630", amPLat+float64(i)*0.0005, amPLng, 10); r != "" {
			t.Fatalf("تحديثٌ مباشرٌ رقم %d ردّ: %q", i, r)
		}
	}
	got := waLocStatus(t, h, cust.Token, waID)
	if got["status"] != "received" || !near(got["lat"], amPLat) {
		t.Fatalf("الطلبُ بعد التحديثات: %v / %v", got["status"], got["lat"])
	}
	pick := driverOrderOf(t, h, drv.Token, orderID)
	if !near(pick["pickup_lat"], amPLat) {
		t.Fatalf("نقطةُ السائق تبدّلت بالتحديثات: %v", pick["pickup_lat"])
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM recipient_location_requests WHERE phone = '+963944555630' AND customer_id = $1 AND received_at IS NOT NULL`,
		cust.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("طلباتٌ ممتلئة %d (%v)", n, err)
	}
}
