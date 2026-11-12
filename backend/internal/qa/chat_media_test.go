package qa

// **صورٌ ورسائلُ صوتيّةٌ في حديث الطلب** — قرارُ المالك ٢٠٢٦-١١-١٢ (هجرة ٠٥٤٧).
//
// طرفا الحديث يرسلان صورةً وصوتاً، ومن ليس طرفاً يُردّ، والرابطُ العاري لا يُخدَم، والقناةُ
// المغلقةُ ترفض، والتطبيقُ القديمُ يقرأ نصّاً بديلاً، وفوق العشرين يُردّ، وما ليس صوتاً يُردّ.

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
)

// qaM4A **بايتاتٌ تبدأ كملفّ M4A** — `ftyp` في البايت الرابع.
func qaM4A() []byte {
	b := []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'M', '4', 'A', ' ', 0, 0, 0, 0}
	return append(b, bytes.Repeat([]byte{0x11}, 512)...)
}

func (h *Harness) sendAttachment(token, orderID, kind string, data []byte, fields map[string]string) Res {
	h.T.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("kind", kind)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	fw, _ := mw.CreateFormFile("file", "x.bin")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", h.Srv.URL+"/api/v1/orders/"+orderID+"/messages/attachment", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := h.Srv.Client().Do(req)
	if err != nil {
		h.T.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Res{Code: resp.StatusCode, Body: b, Head: resp.Header.Clone()}
}

// ageChat **يُزيح الزمن** — فاصلُ المعدّل عقدٌ قائمٌ لا يُمسّ.
func ageChat(t *testing.T, h *Harness, orderID string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE order_messages SET created_at = created_at - interval '1 minute'
		 WHERE order_id = $1::uuid`, orderID); err != nil {
		t.Fatalf("إزاحةُ الزمن: %v", err)
	}
}

func mediaCode(t *testing.T, h *Harness, url string) int {
	t.Helper()
	resp, err := h.Srv.Client().Get(h.Srv.URL + url)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	return resp.StatusCode
}

func attachmentOf(t *testing.T, m map[string]any) map[string]any {
	t.Helper()
	a, _ := m["attachment"].(map[string]any)
	if a == nil {
		t.Fatalf("**الرسالةُ بلا مرفق**: %v", m)
	}
	return a
}

func TestCHATMEDIA_PartiesSendPhotoAndVoice(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)
	acceptOrder(t, h, fx.DrvA, fx.Order)

	// **الزبونُ يرسل صورةً بتعليق.**
	r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", qaJPEG(t), map[string]string{"body": "هيك الباب"})
	if r.Code != http.StatusCreated {
		t.Fatalf("صورةُ الزبون: %s", r)
	}
	if body, _ := r.JSON()["body"].(string); !strings.HasPrefix(body, comms.FallbackPhoto) {
		t.Fatalf("**لا نصَّ بديلاً للتطبيق القديم**: %q", body)
	}
	a := attachmentOf(t, r.JSON())
	if a["kind"] != "photo" || a["caption"] != "هيك الباب" {
		t.Fatalf("مرفقُ الصورة: %v", a)
	}
	photoURL, _ := a["url"].(string)
	if !strings.Contains(photoURL, "sig=") || !strings.HasPrefix(photoURL, "/media/") {
		t.Fatalf("**رابطٌ غيرُ موقَّع**: %q", photoURL)
	}
	if th, _ := a["thumb_url"].(string); th == "" {
		t.Fatalf("صورةٌ بلا مصغَّرة: %v", a)
	}
	ageChat(t, h, fx.Order)

	// **والسائقُ يرسل رسالةً صوتيّة.**
	r = h.sendAttachment(fx.DrvA.Token, fx.Order, "audio", qaM4A(), map[string]string{"duration_ms": "12000"})
	if r.Code != http.StatusCreated {
		t.Fatalf("صوتُ السائق: %s", r)
	}
	if body, _ := r.JSON()["body"].(string); body != comms.FallbackAudio {
		t.Fatalf("**النصُّ البديلُ للصوت**: %q", body)
	}
	a = attachmentOf(t, r.JSON())
	if a["kind"] != "audio" || a["duration_ms"] != float64(12000) {
		t.Fatalf("مرفقُ الصوت: %v", a)
	}
	audioURL, _ := a["url"].(string)
	if !strings.Contains(audioURL, ".m4a?") {
		t.Fatalf("امتدادُ الصوت: %q", audioURL)
	}
	ageChat(t, h, fx.Order)

	// **والزبونُ يرسل صوتاً والسائقُ صورة** — الاتّجاهان.
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "audio", qaM4A(), map[string]string{"duration_ms": "3000"}); r.Code != http.StatusCreated {
		t.Fatalf("صوتُ الزبون: %s", r)
	}
	ageChat(t, h, fx.Order)
	if r := h.sendAttachment(fx.DrvA.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusCreated {
		t.Fatalf("صورةُ السائق: %s", r)
	}

	// **والقراءةُ تحمل المرفقَ للطرفين.**
	for _, tok := range []string{fx.Cust.Token, fx.DrvA.Token} {
		n := 0
		for _, m := range readChat(t, h, tok, fx.Order) {
			if m["attachment"] != nil {
				n++
			}
		}
		if n != 4 {
			t.Fatalf("**المرفقاتُ في القراءة %d لا ٤**", n)
		}
	}

	// **والرابطُ الموقَّعُ يُخدَم، والعاري لا.**
	if c := mediaCode(t, h, photoURL); c != http.StatusOK {
		t.Fatalf("الرابطُ الموقَّعُ للصورة: %d", c)
	}
	if c := mediaCode(t, h, audioURL); c != http.StatusOK {
		t.Fatalf("الرابطُ الموقَّعُ للصوت: %d", c)
	}
	for _, u := range []string{photoURL, audioURL} {
		bare := u[:strings.Index(u, "?")]
		if c := mediaCode(t, h, bare); c != http.StatusNotFound {
			t.Fatalf("**الرابطُ العاري خُدم** (%s): %d", bare, c)
		}
	}

	// **ومن ليس طرفاً يُردّ** — زبونٌ آخر وسائقٌ آخر.
	stranger := h.Customer()
	for _, tok := range []string{stranger.Token, fx.DrvB.Token} {
		if r := h.sendAttachment(tok, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusNotFound && r.Code != http.StatusForbidden {
			t.Fatalf("**غريبٌ أرسل صورة**: %s", r)
		}
		if r := h.GET("/api/v1/orders/"+fx.Order+"/messages", tok); r.Code != http.StatusNotFound && r.Code != http.StatusForbidden {
			t.Fatalf("**غريبٌ قرأ الحديث**: %s", r)
		}
	}

	// **والإدارةُ ترى المرفقات.**
	admin := h.NewUser("admin")
	ar := h.GET("/api/v1/admin/orders/"+fx.Order+"/chat", admin.Token)
	if ar.Code != http.StatusOK {
		t.Fatalf("حديثُ الإدارة: %s", ar)
	}
	lines, _ := ar.JSON()["lines"].([]any)
	n := 0
	for _, x := range lines {
		if m, _ := x.(map[string]any); m["attachment"] != nil {
			n++
		}
	}
	if n != 4 {
		t.Fatalf("**الإدارةُ ترى %d مرفقاً لا ٤**", n)
	}
}

func TestCHATMEDIA_RejectsBadInput(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)
	acceptOrder(t, h, fx.DrvA, fx.Order)

	// **ما ليس صوتاً لا يُقبل صوتاً.**
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "audio", []byte("just some text, not audio"), map[string]string{"duration_ms": "2000"}); r.Code != http.StatusBadRequest || r.Err() != "invalid_audio" {
		t.Fatalf("بايتاتٌ ليست صوتاً: %s", r)
	}
	// **ولا صورةً.**
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", []byte("not an image"), nil); r.Code != http.StatusBadRequest {
		t.Fatalf("بايتاتٌ ليست صورة: %s", r)
	}
	// **وأطولُ من دقيقة.**
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "audio", qaM4A(), map[string]string{"duration_ms": "90000"}); r.Code != http.StatusBadRequest {
		t.Fatalf("صوتٌ بتسعين ثانية: %s", r)
	}
	// **ونوعٌ لا نعرفه.**
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "video", qaM4A(), nil); r.Code != http.StatusBadRequest {
		t.Fatalf("نوعٌ مجهول: %s", r)
	}

	// **وفاصلُ المعدّل نفسُه** — مرفقان متلاحقان.
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusCreated {
		t.Fatalf("الصورةُ الأولى: %s", r)
	}
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusTooManyRequests {
		t.Fatalf("**مرفقان بلا فاصل**: %s", r)
	}

	// **وفوق العشرين يُردّ** — تسعَ عشرةَ أخرى تُكتب في القاعدة، والحادية والعشرون تُرفض.
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_messages (order_id, sender_id, sender_role, body, driver_id, attachment_kind, created_at)
		SELECT $1::uuid, $2::uuid, 'customer', 'x', $3::uuid, 'photo', now() - interval '1 hour'
		  FROM generate_series(1, 19)`, fx.Order, fx.Cust.ID, fx.DrvA.ID); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	ageChat(t, h, fx.Order)
	if r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusConflict || r.Err() != "chat_attachments_limit" {
		t.Fatalf("**المرفقُ الحادي والعشرون قُبل**: %s", r)
	}
	// **والسقفُ لكلّ مرسِل** — السائقُ يرسل.
	if r := h.sendAttachment(fx.DrvA.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusCreated {
		t.Fatalf("سقفُ الزبون منع السائق: %s", r)
	}
	// **والنصُّ ما زال يُرسل** بعد السقف.
	ageChat(t, h, fx.Order)
	if r := say(t, h, fx.Cust.Token, fx.Order, "تمام"); r.Code != http.StatusCreated {
		t.Fatalf("النصُّ بعد السقف: %s", r)
	}
}

func TestCHATMEDIA_ClosedChannelRejects(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)
	acceptOrder(t, h, fx.DrvA, fx.Order)
	if _, err := h.Pool.Exec(ctxBG(), `
		UPDATE orders SET status = 'delivered', delivered_at = now() - interval '1 hour'
		 WHERE id = $1::uuid`, fx.Order); err != nil {
		t.Fatalf("تهيئة: %v", err)
	}
	r := h.sendAttachment(fx.Cust.Token, fx.Order, "photo", qaJPEG(t), nil)
	if r.Code != http.StatusConflict || r.Err() != "comms_closed" {
		t.Fatalf("**صورةٌ في حديثٍ مغلق**: %s", r)
	}
	if r := h.sendAttachment(fx.DrvA.Token, fx.Order, "audio", qaM4A(), map[string]string{"duration_ms": "2000"}); r.Code < 400 {
		t.Fatalf("**صوتٌ من السائق بعد الانتهاء**: %s", r)
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM media WHERE kind IN ('chat_photo','chat_audio')
		   AND created_by IN ($1::uuid, $2::uuid)`, fx.Cust.ID, fx.DrvA.ID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("**كُتب ملفٌّ لرسالةٍ رُفضت**: %d", n)
	}
}
