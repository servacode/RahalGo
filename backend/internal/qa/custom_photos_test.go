package qa

// **صورُ الطلب الخاصّ** — قرارُ المالك ٢٠٢٦-١١-١١ (هجرة ٠٥٤١).
//
// تُرفع قبل الطلب (`POST /me/custom-photo`) ثمّ تُربط بمعاملة الإنشاء، وتظهر في قراءة الزبون
// برابطٍ موقَّع. **والصورةُ لصاحبها وحدَه وللطلب الواحد وحدَه، وبسقف ثلاث.**

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

func qaJPEG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for x := 0; x < 40; x++ {
		for y := 0; y < 30; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 6), uint8(y * 8), 90, 255})
		}
	}
	var b bytes.Buffer
	if err := jpeg.Encode(&b, img, nil); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func (h *Harness) uploadCustomPhoto(token string, data []byte) Res {
	h.T.Helper()
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, _ := mw.CreateFormFile("file", "list.jpg")
	_, _ = fw.Write(data)
	_ = mw.Close()
	req, _ := http.NewRequest("POST", h.Srv.URL+"/api/v1/me/custom-photo", &body)
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

func TestCUSTPHOTO_UploadAttachShowOnce(t *testing.T) {
	h := New(t)
	cust, other := h.Customer(), h.Customer()
	data := qaJPEG(t)

	up := h.uploadCustomPhoto(cust.Token, data)
	if up.Code != 201 {
		t.Fatalf("رفعُ الصورة: %s", up)
	}
	id, _ := up.JSON()["id"].(string)
	url, _ := up.JSON()["url"].(string)
	if id == "" || !strings.HasPrefix(url, "/media/") {
		t.Fatalf("ردُّ الرفع بلا معرّفٍ أو رابط: %s", up)
	}

	// **صورةُ غيرِك لا تُربط بطلبك.**
	body := customBody("")
	body["photo_ids"] = []string{id}
	if r := h.POSTKey("/api/v1/orders/custom", other.Token, uniq("k"), body); r.Code != 400 || r.Err() != "custom_photos_invalid" {
		t.Fatalf("ربطُ صورةِ غيري يجب أن يُردّ بـcustom_photos_invalid: %s", r)
	}

	// **والصورةُ تغني عن النصّ** — طلبٌ بلا كتابةٍ وبصورة.
	got := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"), body)
	if got.Code != 201 {
		t.Fatalf("طلبٌ بصورةٍ وبلا نصّ: %s", got)
	}
	if ph, _ := got.JSON()["custom_photos"].([]any); len(ph) != 1 {
		t.Fatalf("ردُّ الإنشاء بلا الصورة: %s", got)
	}
	oid, _ := got.JSON()["id"].(string)
	read := h.GET("/api/v1/my/orders/"+oid, cust.Token)
	if read.Code != 200 {
		t.Fatalf("قراءةُ الطلب: %s", read)
	}
	if ph, _ := read.JSON()["custom_photos"].([]any); len(ph) != 1 {
		t.Fatalf("قراءةُ الزبون بلا الصورة: %s", read)
	}

	// **والصورةُ تخدم طلباً واحداً.**
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"), body); r.Code != 400 {
		t.Fatalf("إعادةُ استعمال صورةٍ مربوطة يجب أن تُردّ: %s", r)
	}

	// **وفوق الثلاث تُردّ.**
	var ids []string
	for i := 0; i < 4; i++ {
		u := h.uploadCustomPhoto(cust.Token, data)
		if u.Code != 201 {
			t.Fatalf("رفع %d: %s", i, u)
		}
		s, _ := u.JSON()["id"].(string)
		ids = append(ids, s)
	}
	four := customBody("قائمة")
	four["photo_ids"] = ids
	if r := h.POSTKey("/api/v1/orders/custom", cust.Token, uniq("k"), four); r.Code != 400 {
		t.Fatalf("أربعُ صورٍ يجب أن تُردّ: %s", r)
	}

	// **وما ليس صورةً لا يُرفع.**
	if r := h.uploadCustomPhoto(cust.Token, []byte("not an image")); r.Code != 400 {
		t.Fatalf("ملفٌّ ليس صورةً يجب أن يُردّ: %s", r)
	}
}
