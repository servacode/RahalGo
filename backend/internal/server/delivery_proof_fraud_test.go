package server

// ══════════════════════════════════════════════════════════════════════
// **منعُ احتيال السائق عند التسليم** (قرارُ المالك ٢٠٢٦-١٠-٠٣ مساءً: «منع — تمنع احتيالَ
// السائق، مع تنبيهٍ للسائق»)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس في تجربة القبول**: صورةٌ من مكانٍ بعيدٍ عن الزبون تُقبل، وصورةٌ بـ`mocked=true`
// تُقبل بلا موضع — **فيمضي التسليمُ بإثباتٍ لا يشهد بشيء.**

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/media"
)

// proofUpload يرفع صورةَ تسليمٍ حقيقيّةً بموضعٍ ووسم تزييف.
func (f *driverFixture) proofUpload(t *testing.T, driverID, orderID string, lat, lng float64, mocked bool) *httptest.ResponseRecorder {
	return f.proofUploadAcc(t, driverID, orderID, lat, lng, mocked, -1)
}

// proofUploadAcc **ومع دقّة الجوال** — وسالبُها لا يُرسل الحقل (نسخةٌ قديمة).
func (f *driverFixture) proofUploadAcc(t *testing.T, driverID, orderID string, lat, lng float64, mocked bool, acc float64) *httptest.ResponseRecorder {
	t.Helper()
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "door.png")
	_, _ = part.Write(img.Bytes())
	_ = mw.WriteField("lat", strconv.FormatFloat(lat, 'f', 6, 64))
	_ = mw.WriteField("lng", strconv.FormatFloat(lng, 'f', 6, 64))
	if mocked {
		_ = mw.WriteField("mocked", "true")
	}
	if acc >= 0 {
		_ = mw.WriteField("accuracy", strconv.FormatFloat(acc, 'f', 1, 64))
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/driver/orders/"+orderID+"/proof", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", orderID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, ctxUserID, driverID)
	ctx = context.WithValue(ctx, ctxRoles, []string{"driver"})
	w := httptest.NewRecorder()
	f.srv.handleDeliveryProof(w, req.WithContext(ctx))
	return w
}

func TestDeliveryProof_FarOrMockedIsRejected(t *testing.T) {
	f := newDriverFixture(t, 1)
	m, err := media.NewService(f.pool, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.srv.media = m
	d := f.drivers[0]
	ctx := context.Background()
	id := f.problemOrderAt(t, "at_dropoff", d)
	var dLat, dLng float64
	if err := f.pool.QueryRow(ctx, `
		SELECT ST_Y(dropoff::geometry), ST_X(dropoff::geometry) FROM orders WHERE id = $1`, id).
		Scan(&dLat, &dLng); err != nil {
		t.Fatal(err)
	}
	pod := func() bool {
		var has bool
		_ = f.pool.QueryRow(ctx, `SELECT pod_media_id IS NOT NULL FROM orders WHERE id = $1`, id).Scan(&has)
		return has
	}

	// **بعيدٌ ٣٠٠ م تقريباً** (٠٫٠٠٢٧° عرضاً) — يُرفض ويُقال كم.
	w := f.proofUpload(t, d, id, dLat+0.0027, dLng, false)
	if code := errCode(t, w); code != "proof_too_far" {
		t.Fatalf("صورةٌ من بعيدٍ ردّت %d %q: %s — والمتوقّعُ proof_too_far", w.Code, code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"distance_m":"29`) {
		t.Errorf("المسافةُ غائبةٌ عن التفصيل: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"max_m":"15"`) {
		t.Errorf("الحدُّ ليس ١٥ م: %s", w.Body.String())
	}
	if pod() {
		t.Fatal("صورةٌ مرفوضةٌ حُفظت إثباتاً")
	}

	// **بموقعٍ مزيَّف** — يُرفض ولو كان عند الباب.
	w = f.proofUpload(t, d, id, dLat, dLng, true)
	if code := errCode(t, w); code != "proof_mocked" {
		t.Fatalf("صورةٌ بموقعٍ مزيَّفٍ ردّت %d %q: %s — والمتوقّعُ proof_mocked", w.Code, code, w.Body.String())
	}
	if pod() {
		t.Fatal("صورةٌ مزيَّفةٌ حُفظت إثباتاً")
	}

	// **والحدُّ ١٥ م لا ١٥٠** (قرارُ المالك ٢٠٢٦-١٠-٠٣) — ٤٠ م بلا دقّةٍ تُرفض…
	if code := errCode(t, f.proofUpload(t, d, id, dLat+0.00036, dLng, false)); code != "proof_too_far" {
		t.Fatalf("صورةٌ على ٤٠ م بلا دقّة: %q — والحدُّ ١٥ م", code)
	}
	// **…وبدقّة ٣٠ م تُقبل** (١٥ + ٣٠) — **والهامشُ سقفُه ٥٠**: ٨٠ م بدقّة ٢٠٠ تُرفض (١٥ + ٥٠).
	if code := errCode(t, f.proofUploadAcc(t, d, id, dLat+0.00072, dLng, false, 200)); code != "proof_too_far" {
		t.Fatalf("صورةٌ على ٨٠ م بدقّة ٢٠٠: %q — والهامشُ سقفُه ٥٠ م", code)
	}
	if pod() {
		t.Fatal("صورةٌ بعيدةٌ حُفظت إثباتاً")
	}
	if w := f.proofUploadAcc(t, d, id, dLat+0.00036, dLng, false, 30); w.Code != http.StatusCreated {
		t.Fatalf("صورةٌ على ٤٠ م بدقّة ٣٠ ردّت %d: %s — **والهامشُ يُضاف**", w.Code, w.Body.String())
	}
	if !pod() {
		t.Fatal("صورةٌ عند الباب لم تُحفظ")
	}
	// **وعند الباب تُقبل** — ١٠ م تقريباً.
	if w := f.proofUpload(t, d, id, dLat+0.00009, dLng, false); w.Code != http.StatusCreated {
		t.Fatalf("صورةٌ عند الباب ردّت %d: %s", w.Code, w.Body.String())
	}
	if !pod() {
		t.Fatal("صورةٌ عند الباب لم تُحفظ")
	}
}

// TestDeliveryProof_UnknownDoorSkipsDistance **وبابٌ لا يُعرف لا يُقاس عليه** — «لدي توصيلة»
// بلا نقطة: نقطتُها المكتوبةُ نقطةُ المتجر، **فصورةٌ عند المستلِم تبدو بعيدة.**
func TestDeliveryProof_UnknownDoorSkipsDistance(t *testing.T) {
	f := newDriverFixture(t, 1)
	m, err := media.NewService(f.pool, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	f.srv.media = m
	d := f.drivers[0]
	id := f.problemOrderAt(t, "at_dropoff", d)
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET dropoff_known = false WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	if w := f.proofUpload(t, d, id, 33.5138, 36.2765, false); w.Code != http.StatusCreated {
		t.Fatalf("بابٌ مجهولٌ رُدّت صورتُه %d: %s", w.Code, w.Body.String())
	}
}
