package qa

// ══════════════════════════════════════════════════════════════════════
// **عدّاد الدرّاجة عند فتح الدوام وسكّره** — قرارات المالك ٢٠٢٦-١١-٠٥
// ══════════════════════════════════════════════════════════════════════
//
// صورةٌ ورقم عند الفتح والسكّر، **والرقمُ الأقلّ من آخر رقمٍ مرفوض**، والقفزةُ الكبيرة تُؤكَّد،
// **وصورةُ الليل حصراً** للنسخة الجديدة، **والقديمةُ تفتح وتسكّر كما كانت** حتّى يُفرض التحديث،
// وسماحُ العمليّات مرّةً، وتصحيحُ المكتب مُدقَّق، والمراقبةُ تعلّم بالأحمر.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// odoUpload يرفع قراءة عدّاد — `fields` حقولُ النموذج، و`file` صورةٌ صغيرة إن طُلبت.
func odoUpload(t *testing.T, h *Harness, token string, fields map[string]string, file bool, headers map[string]string) Res {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		_ = mw.WriteField(k, v)
	}
	if file {
		var img bytes.Buffer
		if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 8, 8))); err != nil {
			t.Fatal(err)
		}
		part, _ := mw.CreateFormFile("file", "odometer.png")
		_, _ = part.Write(img.Bytes())
	}
	_ = mw.Close()
	req, _ := http.NewRequest("POST", h.Srv.URL+"/api/v1/driver/odometer", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+token)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := h.Srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return Res{Code: resp.StatusCode, Body: b, Head: resp.Header.Clone()}
}

// odoReading يرفع قراءةً صالحة ويردّ معرّفها.
func odoReading(t *testing.T, h *Harness, drv *User, phase, km string) string {
	t.Helper()
	r := odoUpload(t, h, drv.Token, map[string]string{"phase": phase, "km": km}, true, driverApp("22"))
	if r.Code != http.StatusCreated {
		t.Fatalf("قراءة %s %s: %s", phase, km, r)
	}
	id, _ := r.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("قراءة بلا معرّف: %s", r)
	}
	return id
}

// odoShift يفتح الدوام أو يسكّره — `capable` **التطبيقُ يعلن أنّه يعرف ورقةَ العدّاد**
// (`odometer_capable`)، ومنه وحدَه يُطبَّق «إلزامي». ورقمُ النسخة في الترويسة للمعلومة.
func odoShift(t *testing.T, h *Harness, drv *User, on bool, readingID string, capable bool) Res {
	t.Helper()
	body := map[string]any{"on": on}
	if readingID != "" {
		body["odometer_id"] = readingID
	}
	if capable {
		body["odometer_capable"] = true
	}
	return h.Call("POST", "/api/v1/driver/shift", drv.Token, body, driverApp("22"))
}

// odoErrDetails تفصيلُ الخطأ.
func odoErrDetails(r Res) map[string]any {
	var e struct {
		Error struct {
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	_ = json.Unmarshal(r.Body, &e)
	return e.Error.Details
}

// TestOdometer_UploadValidation **الرقمُ والصورة**: حروفٌ وسالبٌ وثماني خاناتٍ ومنزلتان ⇒ ٤٠٠ ·
// بلا صورة ⇒ ٤٠٠ · سكّرٌ والدوامُ مسكَّر ⇒ ٤٠٩ · وأرقامٌ عربيّة ⇒ ٢٠١ «أوّل قراءة» بصنف
// `odometer` محميّ لا يُفتح برابطٍ عارٍ.
func TestOdometer_UploadValidation(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	drv := h.NewUser("driver")
	for _, km := range []string{"12a", "-5", "12345678", "12.34", ""} {
		r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": km}, true, nil)
		if r.Code != 400 || r.Err() != "odometer_bad_km" {
			t.Errorf("km=%q: %s — المتوقَّع 400 odometer_bad_km", km, r)
		}
	}
	if r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "1250"}, false, nil); r.Code != 400 || r.Err() != "odometer_photo_required" {
		t.Errorf("بلا صورة: %s", r)
	}
	if r := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "1250"}, true, nil); r.Code != 409 || r.Err() != "odometer_wrong_phase" {
		t.Errorf("سكّرٌ والدوام مسكَّر: %s", r)
	}
	ok := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "١٢٥٠٫٥"}, true, nil)
	if ok.Code != 201 {
		t.Fatalf("قراءة صالحة: %s", ok)
	}
	if km, _ := ok.JSON()["km"].(float64); km != 1250.5 {
		t.Errorf("الرقم = %v والمتوقَّع 1250.5", ok.JSON()["km"])
	}
	if c, _ := ok.JSON()["check"].(string); c != "first" {
		t.Errorf("check = %q والمتوقَّع first", c)
	}
	var kind, path string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT m.kind, m.path FROM driver_odometer_readings r JOIN media m ON m.id = r.media_id
		 WHERE r.id = $1::uuid`, ok.JSON()["id"]).Scan(&kind, &path); err != nil {
		t.Fatal(err)
	}
	if kind != "odometer" {
		t.Errorf("صنف الوسيط = %q", kind)
	}
	if bare := h.GET("/media/"+path, ""); bare.Code == 200 {
		t.Fatalf("صورة العدّاد انفتحت برابطٍ عارٍ: %d", bare.Code)
	}
	if r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "1250.5"}, true, nil); r.Code != 201 {
		t.Errorf("قراءة ثانية بالرقم نفسه: %s", r)
	}
	// وصورةُ الفتح والدوامُ مفتوح ⇒ ٤٠٩
	openShiftFor(t, h, drv)
	if r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "1260"}, true, nil); r.Code != 409 || r.Err() != "odometer_wrong_phase" {
		t.Errorf("فتحٌ والدوام مفتوح: %s", r)
	}
}

// TestOdometer_ReadingCarriesNoLocation **قراءةُ العدّاد بلا موقع**: صورةُ الفتح تُؤخذ والدوامُ
// مسكَّر (والموقعُ واقف)، فما يرسله تطبيقٌ قديمٌ من موقعٍ — **وعلمُ «مزيَّف» من نقطةٍ قديمة** —
// لا يردّ القراءة ولا يُحفظ: لا أعمدةَ موقعٍ في الجدول أصلاً. (ونصُّ الخصوصيّة يَعِد بالموقع
// «ما دامت ورديّتُك مفتوحة» وحدَها.)
func TestOdometer_ReadingCarriesNoLocation(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	drv := h.NewUser("driver")
	r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "1250",
		"lat": "35.95", "lng": "39.01", "accuracy": "12", "mocked": "true"}, true, driverApp("22"))
	if r.Code != http.StatusCreated {
		t.Fatalf("قراءةٌ مع موقعٍ قديمٍ «مزيَّف»: %s — المتوقَّع 201 (الموقعُ لا يُقرأ)", r)
	}
	var cols []string
	rows, err := h.Pool.Query(ctxBG(), `
		SELECT column_name FROM information_schema.columns
		 WHERE table_name = 'driver_odometer_readings'
		   AND column_name IN ('lat', 'lng', 'accuracy_m', 'mocked')`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, c)
	}
	rows.Close()
	if len(cols) > 0 {
		t.Fatalf("أعمدةُ موقعٍ في قراءات العدّاد: %v — القراءةُ بلا موقع", cols)
	}
}

// TestOdometer_LowerRefusedJumpNeedsConfirm **الأقلّ مرفوض ولو أكّده، والقفزةُ تُؤكَّد**.
func TestOdometer_LowerRefusedJumpNeedsConfirm(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_max_jump_km", "150")
	drv := h.NewUser("driver")
	start := odoReading(t, h, drv, "start", "1250")
	if r := odoShift(t, h, drv, true, start, true); r.Code != 200 {
		t.Fatalf("فتح الدوام بقراءة: %s", r)
	}
	for _, confirm := range []string{"", "true"} {
		r := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "1200", "confirm": confirm}, true, driverApp("22"))
		if r.Code != 409 || r.Err() != "odometer_lower" {
			t.Fatalf("أقلّ (confirm=%q): %s — المتوقَّع 409 odometer_lower", confirm, r)
		}
		if d := odoErrDetails(r); d["last_km"] != "1250" {
			t.Errorf("last_km = %v والمتوقَّع 1250", d["last_km"])
		}
	}
	jump := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "1500"}, true, driverApp("22"))
	if jump.Code != 409 || jump.Err() != "odometer_check" {
		t.Fatalf("قفزة: %s — المتوقَّع 409 odometer_check", jump)
	}
	if d := odoErrDetails(jump); d["diff_km"] != "250" || d["last_km"] != "1250" || d["km"] != "1500" {
		t.Errorf("تفصيل القفزة: %v", d)
	}
	ok := odoUpload(t, h, drv.Token, map[string]string{"phase": "end", "km": "1500", "confirm": "true"}, true, driverApp("22"))
	if ok.Code != 201 || ok.JSON()["check"] != "jump_confirmed" {
		t.Fatalf("قفزة مؤكَّدة: %s", ok)
	}
}

// TestOdometer_RequireOnlyForCapableApps **«إلزامي» لمن أعلن أنّه يعرف الورقة وحدَه** (لا رقمُ
// النسخة): مُعلِنٌ بلا قراءة ⇒ ٤٠٩ · نسخةٌ برقم ٢٢ أو ٩٩ لا تُعلن (تصليحٌ سريعٌ بلا الورقة)
// ⇒ ٢٠٠ كما كان، وصفُّها `opened_capable = false` · والويبُ بلا ترويسة ⇒ ٢٠٠ · مُعلِنٌ بقراءة ⇒
// تُربط بالدوام · قراءةٌ عمرها ٣١ دقيقة ⇒ ٤٠٩ · والسائقُ يقرأ الوضعَ كما هو بالإعداد.
func TestOdometer_RequireOnlyForCapableApps(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_mode", `"require"`)
	drv := h.NewUser("driver")
	if r := odoShift(t, h, drv, true, "", true); r.Code != 409 || r.Err() != "odometer_required" {
		t.Fatalf("تطبيقٌ يعرف الورقة بلا قراءة: %s — المتوقَّع 409 odometer_required", r)
	}
	if onShiftOf(t, h, drv.ID) {
		t.Fatal("فُتح الدوام بلا قراءة")
	}
	// **نسخةٌ أحدثُ رقماً بلا الورقة** — لا تُقفل.
	for _, build := range []string{"22", "99"} {
		open := h.Call("POST", "/api/v1/driver/shift", drv.Token, map[string]any{"on": true}, driverApp(build))
		if open.Code != 200 {
			t.Fatalf("نسخة %s لا تعلن الورقة: %s — تفتح كما كانت", build, open)
		}
		var capable *bool
		if err := h.Pool.QueryRow(ctxBG(), `SELECT opened_capable FROM driver_shifts
			WHERE driver_id = $1::uuid AND closed_at IS NULL`, drv.ID).Scan(&capable); err != nil {
			t.Fatal(err)
		}
		if capable == nil || *capable {
			t.Fatalf("نسخة %s: opened_capable = %v والمتوقَّع false", build, capable)
		}
		if r := h.Call("POST", "/api/v1/driver/shift", drv.Token, map[string]any{"on": false}, driverApp(build)); r.Code != 200 {
			t.Fatalf("نسخة %s تسكّر: %s", build, r)
		}
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": true}); r.Code != 200 {
		t.Fatalf("بلا ترويسة (الويب): %s", r)
	}
	if r := h.POST("/api/v1/driver/shift", drv.Token, map[string]any{"on": false}); r.Code != 200 {
		t.Fatal(r)
	}
	// قراءةٌ قديمة ⇒ لا تُقبل
	stale := odoReading(t, h, drv, "start", "1000")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE driver_odometer_readings SET created_at = now() - interval '31 minutes' WHERE id = $1::uuid`, stale); err != nil {
		t.Fatal(err)
	}
	if r := odoShift(t, h, drv, true, stale, true); r.Code != 409 || r.Err() != "odometer_required" {
		t.Fatalf("قراءة عمرها ٣١ دقيقة: %s", r)
	}
	fresh := odoReading(t, h, drv, "start", "1001")
	if r := odoShift(t, h, drv, true, fresh, true); r.Code != 200 {
		t.Fatalf("تطبيقٌ يعرف الورقة بقراءة: %s", r)
	}
	var attached string
	var capable *bool
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(start_reading_id::text, ''), opened_capable FROM driver_shifts
		WHERE driver_id = $1::uuid AND closed_at IS NULL`, drv.ID).Scan(&attached, &capable); err != nil || attached != fresh {
		t.Fatalf("القراءة ما انربطت بالدوام: %q %v", attached, err)
	}
	if capable == nil || !*capable {
		t.Fatalf("opened_capable = %v والمتوقَّع true", capable)
	}
	// **وقراءةٌ مربوطةٌ لا تُستعمل مرّتين**
	if r := odoShift(t, h, drv, false, odoReading(t, h, drv, "end", "1010"), true); r.Code != 200 {
		t.Fatal(r)
	}
	if r := odoShift(t, h, drv, true, fresh, true); r.Code != 409 {
		t.Fatalf("قراءةٌ مربوطة فتحت دواماً ثانياً: %s", r)
	}
	// والسائق يرى الوضعَ كما هو بالإعداد — والتطبيقُ القديم لا يقرؤه
	for _, hdr := range []map[string]string{driverApp("22"), nil} {
		me := h.Call("GET", "/api/v1/driver/me", drv.Token, nil, hdr)
		if me.JSON()["odometer_mode"] != "require" {
			t.Errorf("odometer_mode = %v والمتوقَّع require", me.JSON()["odometer_mode"])
		}
		if last, _ := me.JSON()["odometer_last_km"].(float64); last != 1010 {
			t.Errorf("odometer_last_km = %v والمتوقَّع 1010", me.JSON()["odometer_last_km"])
		}
	}
}

// TestOdometer_CloseNeedsPhotoForNewApp **صورةُ الليل حصراً** (قرار المالك): «إلزامي» وتطبيقٌ يعرف
// الورقة يسكّر بلا قراءة ⇒ ٤٠٩ ويبقى بالدوام؛ وبقراءة ⇒ سبب «السائق» والقراءة مربوطة. و«يطلب» بلا
// قراءة ⇒ «سكّره بلا صورة». والقديمُ يسكّر كما كان.
func TestOdometer_CloseNeedsPhotoForNewApp(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_mode", `"require"`)
	drv := h.NewUser("driver")
	if r := odoShift(t, h, drv, true, odoReading(t, h, drv, "start", "500"), true); r.Code != 200 {
		t.Fatal(r)
	}
	if r := odoShift(t, h, drv, false, "", true); r.Code != 409 || r.Err() != "odometer_required" {
		t.Fatalf("سكّر بلا صورة (يعرف الورقة): %s — المتوقَّع 409", r)
	}
	if !onShiftOf(t, h, drv.ID) {
		t.Fatal("سُكّر الدوام بلا صورة الليل")
	}
	end := odoReading(t, h, drv, "end", "540")
	if r := odoShift(t, h, drv, false, end, true); r.Code != 200 {
		t.Fatalf("سكّر بصورة: %s", r)
	}
	var reason, endID string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT close_reason, COALESCE(end_reading_id::text, '') FROM driver_shifts
		WHERE driver_id = $1::uuid ORDER BY opened_at DESC LIMIT 1`, drv.ID).Scan(&reason, &endID); err != nil {
		t.Fatal(err)
	}
	if reason != "driver" || endID != end {
		t.Fatalf("بعد السكّر: سبب=%q قراءة=%q والمتوقَّع driver و%s", reason, endID, end)
	}
	// «يطلب»: يُسمح بالتخطّي
	h.Setting("drivers.odometer_mode", `"ask"`)
	if r := odoShift(t, h, drv, true, "", true); r.Code != 200 {
		t.Fatalf("«يطلب» يفتح بلا صورة: %s", r)
	}
	if r := odoShift(t, h, drv, false, "", true); r.Code != 200 {
		t.Fatalf("«يطلب» يسكّر بلا صورة: %s", r)
	}
	if _, reason := lastReason(t, h, drv.ID); reason != "driver_skipped" {
		t.Fatalf("سبب السكّر بلا صورة = %q والمتوقَّع driver_skipped", reason)
	}
	// القديمة
	h.Setting("drivers.odometer_mode", `"require"`)
	if r := odoShift(t, h, drv, true, "", false); r.Code != 200 {
		t.Fatal(r)
	}
	if r := odoShift(t, h, drv, false, "", false); r.Code != 200 {
		t.Fatalf("القديمة تسكّر بلا صورة: %s", r)
	}
}

// TestOdometer_ReadingSurvivesRejectedOpen **فتحٌ رُفض لا يضيّع الصورة**: «وثّق رقمك» ⇒ وثّقه ⇒
// يفتح بالقراءة نفسها.
func TestOdometer_ReadingSurvivesRejectedOpen(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_mode", `"require"`)
	h.Setting("auth.require_whatsapp", "true")
	h.Setting("drivers.require_whatsapp", "true")
	drv := h.NewUser("driver")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET whatsapp_verified_at = NULL WHERE id = $1::uuid`, drv.ID); err != nil {
		t.Fatal(err)
	}
	id := odoReading(t, h, drv, "start", "700")
	if r := odoShift(t, h, drv, true, id, true); r.Err() != "whatsapp_required" {
		t.Fatalf("فتحٌ بلا توثيق: %s — المتوقَّع whatsapp_required", r)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET whatsapp_verified_at = now() WHERE id = $1::uuid`, drv.ID); err != nil {
		t.Fatal(err)
	}
	if r := odoShift(t, h, drv, true, id, true); r.Code != 200 {
		t.Fatalf("الفتح بالقراءة نفسها بعد التوثيق: %s", r)
	}
}

// odoInsertReading قراءةٌ مكتوبةٌ مباشرة بوقتها (للمراقبة).
func odoInsertReading(t *testing.T, h *Harness, driver, phase string, km float64, at time.Time, media *string) string {
	t.Helper()
	var id string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO driver_odometer_readings (driver_id, phase, km, media_id, created_at, app_build)
		VALUES ($1::uuid, $2, $3, $4::uuid, $5, 22) RETURNING id::text`, driver, phase, km, media, at).Scan(&id); err != nil {
		t.Fatalf("قراءة: %v", err)
	}
	return id
}

type odoDayResp struct {
	Shifts []struct {
		ID          string   `json:"id"`
		CloseReason string   `json:"close_reason"`
		OdometerKm  *float64 `json:"odometer_km"`
		ApproxKm    *float64 `json:"approx_km"`
		OffBeforeKm *float64 `json:"off_before_km"`
		UnseenKm    *float64 `json:"unseen_km"`
		DriveM      int      `json:"drive_m"`
		Flags       []string `json:"flags"`
		Start       *struct {
			ThumbURL *string `json:"thumb_url"`
		} `json:"start_reading"`
	} `json:"shifts"`
	Odometer struct {
		OnKm         *float64 `json:"on_km"`
		OffKm        *float64 `json:"off_km"`
		MissingClose int      `json:"missing_close"`
		BigGap       bool     `json:"big_gap"`
	} `json:"odometer"`
}

func hasFlag(fs []string, f string) bool {
	for _, x := range fs {
		if x == f {
			return true
		}
	}
	return false
}

// TestDriverMonitor_ShiftsOdometerAndFlags **العدّاد مقابل التطبيق بالمراقبة**: عدّادٌ ٥٤ كم
// والتطبيقُ ٣٠ ⇒ «فرق كبير» (٢٤ ما سجّلها التطبيق) · مشى ١٠ كم وهو خارج الدوام ⇒ أحمر ·
// دوامٌ سكّره الترك بلا صورة ⇒ «ما صوّر عند الإغلاق» وتقريبيٌّ من فتح الدوام التالي ·
// والصورةُ برابطٍ موقَّع · ومن لا يملك `drivers.monitor` ⇒ ٤٠٣ للتصحيح.
func TestDriverMonitor_ShiftsOdometerAndFlags(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	f := h.Factory()
	drv := f.Driver()
	day, base := rdDayBase()
	h.Setting("drivers.odometer_gap_alert_km", "5")
	h.Setting("drivers.odometer_gap_alert_pct", "20")
	h.Setting("drivers.odometer_off_alert_km", "1")

	// صورةٌ حقيقيّة للقراءة الأولى (الرابطُ موقَّع)
	var mediaID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO media (kind, path, thumb_path, width, height, bytes, created_by)
		VALUES ('odometer', '2026/11/odo-test.jpg', '2026/11/odo-test_t.jpg', 8, 8, 10, $1::uuid)
		RETURNING id::text`, drv.ID).Scan(&mediaID); err != nil {
		t.Fatal(err)
	}

	// دوامُ أمس: سكّره بصورة 1240
	prevOpen := base.Add(-20 * time.Hour)
	prevClose := base.Add(-14 * time.Hour)
	prevStart := odoInsertReading(t, h, drv.ID, "start", 1200, prevOpen, nil)
	prevEnd := odoInsertReading(t, h, drv.ID, "end", 1240, prevClose, nil)
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id, end_reading_id, opened_build)
		VALUES ($1::uuid, $2, $3, 'driver', $4::uuid, $5::uuid, 22)`, drv.ID, prevOpen, prevClose, prevStart, prevEnd); err != nil {
		t.Fatal(err)
	}

	// الدوام الأوّل اليوم: 1250 → 1304، والتطبيق ٣٠ كم قيادة
	open1 := base
	start := rdPoint{at: open1.Add(time.Minute), lat: 35.82, lng: 38.92, acc: rdF(6), spd: rdF(10)}
	trail := append([]rdPoint{start}, rdDrive(start, 30000, 10, 10*time.Second)...)
	rdPut(t, h, drv.ID, trail)
	close1 := rdLast(trail).at.Add(2 * time.Minute)
	r1 := odoInsertReading(t, h, drv.ID, "start", 1250, open1, &mediaID)
	r2 := odoInsertReading(t, h, drv.ID, "end", 1304, close1, nil)
	var shift1 string
	if err := h.Pool.QueryRow(ctxBG(), `INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id, end_reading_id, opened_build)
		VALUES ($1::uuid, $2, $3, 'driver', $4::uuid, $5::uuid, 22) RETURNING id::text`, drv.ID, open1, close1, r1, r2).Scan(&shift1); err != nil {
		t.Fatal(err)
	}
	// الدوام الثاني: فُتح بصورة 1310 وسكّره الترك بلا صورة، والثالث فُتح بـ1330
	open2, close2 := close1.Add(30*time.Minute), close1.Add(90*time.Minute)
	r3 := odoInsertReading(t, h, drv.ID, "start", 1310, open2, nil)
	var shift2 string
	if err := h.Pool.QueryRow(ctxBG(), `INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id, opened_build)
		VALUES ($1::uuid, $2, $3, 'release', $4::uuid, 22) RETURNING id::text`, drv.ID, open2, close2, r3).Scan(&shift2); err != nil {
		t.Fatal(err)
	}
	open3 := close2.Add(30 * time.Minute)
	r4 := odoInsertReading(t, h, drv.ID, "start", 1330, open3, nil)
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id, opened_build)
		VALUES ($1::uuid, $2, $3, 'ops', $4::uuid, 22)`, drv.ID, open3, open3.Add(time.Hour), r4); err != nil {
		t.Fatal(err)
	}

	res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", drv.ID, day), admin.Token)
	if res.Code != 200 {
		t.Fatalf("يوم السائق: %s", res)
	}
	var env struct {
		Data odoDayResp `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	got := env.Data
	byID := map[string]int{}
	for i, s := range got.Shifts {
		byID[s.ID] = i
	}
	s1 := got.Shifts[byID[shift1]]
	if s1.OdometerKm == nil || *s1.OdometerKm != 54 {
		t.Fatalf("حسب العدّاد = %v والمتوقَّع 54", s1.OdometerKm)
	}
	if math.Abs(float64(s1.DriveM)-30000) > 300 {
		t.Fatalf("حسب التطبيق = %d م والمتوقَّع ~٣٠٠٠٠", s1.DriveM)
	}
	if s1.UnseenKm == nil || math.Abs(*s1.UnseenKm-24) > 0.5 || !hasFlag(s1.Flags, "big_gap") {
		t.Errorf("ما سجّلها التطبيق = %v وأعلام %v — المتوقَّع ~٢٤ و big_gap", s1.UnseenKm, s1.Flags)
	}
	if s1.OffBeforeKm == nil || *s1.OffBeforeKm != 10 || !hasFlag(s1.Flags, "off_duty_moved") {
		t.Errorf("خارج الدوام = %v وأعلام %v — المتوقَّع 10 و off_duty_moved", s1.OffBeforeKm, s1.Flags)
	}
	if s1.Start == nil || s1.Start.ThumbURL == nil || !strings.Contains(*s1.Start.ThumbURL, "sig=") {
		t.Errorf("صورة العدّاد بلا رابطٍ موقَّع: %+v", s1.Start)
	}
	s2 := got.Shifts[byID[shift2]]
	if !hasFlag(s2.Flags, "no_close_photo") || s2.CloseReason != "release" {
		t.Errorf("الدوام الثاني: أعلام %v سبب %q — المتوقَّع no_close_photo و release", s2.Flags, s2.CloseReason)
	}
	if s2.ApproxKm == nil || *s2.ApproxKm != 20 {
		t.Errorf("التقريبي = %v والمتوقَّع 20 (1330 − 1310)", s2.ApproxKm)
	}
	if got.Odometer.MissingClose < 2 || !got.Odometer.BigGap {
		t.Errorf("ملخّص اليوم: %+v", got.Odometer)
	}

	// قائمة اليوم تحمل الملخّص
	list := h.GET("/api/v1/admin/driver-monitor?day="+day, admin.Token)
	if !strings.Contains(string(list.Body), `"missing_close"`) {
		t.Errorf("قائمة اليوم بلا ملخّص العدّاد: %s", list.Body)
	}
	// وتصحيح من لا يملك `drivers.monitor` ⇒ ٤٠٣
	ops := h.NewUser("operations")
	if r := h.PATCH("/api/v1/admin/driver-odometer/"+r2, ops.Token, map[string]any{"km": 1300, "note": "غلط"}); r.Code != 403 {
		t.Errorf("تصحيح العمليات = %d والمتوقَّع 403", r.Code)
	}
}

// TestOdometer_CorrectionAudited **تصحيحُ المكتب**: الرقمُ المصحَّح يحكم والأصلُ يبقى، وسطرٌ
// في السجلّ بالقديم والجديد والسبب؛ وبلا سبب ⇒ ٤٠٠.
func TestOdometer_CorrectionAudited(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	id := odoReading(t, h, drv, "start", "12500")
	if r := h.PATCH("/api/v1/admin/driver-odometer/"+id, admin.Token, map[string]any{"km": 1250, "note": ""}); r.Code != 400 {
		t.Fatalf("بلا سبب: %s", r)
	}
	if r := h.PATCH("/api/v1/admin/driver-odometer/"+id, admin.Token, map[string]any{"km": "1250", "note": "زاد صفراً"}); r.Code != 200 {
		t.Fatalf("التصحيح: %s", r)
	}
	var km, corrected float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT km::float8, corrected_km::float8 FROM driver_odometer_readings WHERE id = $1::uuid`, id).Scan(&km, &corrected); err != nil {
		t.Fatal(err)
	}
	if km != 12500 || corrected != 1250 {
		t.Fatalf("بعد التصحيح: الأصل=%v المصحَّح=%v", km, corrected)
	}
	var meta string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT details::text FROM audit_log
		WHERE action = 'ops.odometer_corrected' AND entity_id = $1 ORDER BY created_at DESC LIMIT 1`, drv.ID).Scan(&meta); err != nil {
		t.Fatalf("سطر السجلّ: %v", err)
	}
	if !strings.Contains(meta, "12500") || !strings.Contains(meta, "1250") || !strings.Contains(meta, "زاد صفراً") {
		t.Errorf("سطر السجلّ = %s", meta)
	}
	// والقراءة التالية تُقاس بالمصحَّح: 1300 تمرّ (لا «أقلّ من 12500»)
	if r := odoUpload(t, h, drv.Token, map[string]string{"phase": "start", "km": "1300"}, true, nil); r.Code != 201 {
		t.Fatalf("قراءة بعد التصحيح: %s", r)
	}
}

// TestOdometer_ExemptionOneOpen **سماحُ العمليّات مرّةً واحدة**: يفتح بلا صورة ويُكتب من سمح،
// والفتحُ التالي يطلب الصورة من جديد — والسكّرُ يطلبها دائماً.
func TestOdometer_ExemptionOneOpen(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_mode", `"require"`)
	ops := h.NewUser("operations")
	drv := h.NewUser("driver")
	if r := h.POST("/api/v1/admin/drivers/"+drv.ID+"/odometer-exemption", ops.Token, map[string]any{"reason": ""}); r.Code != 400 {
		t.Fatalf("سماحٌ بلا سبب: %s", r)
	}
	if r := h.POST("/api/v1/admin/drivers/"+drv.ID+"/odometer-exemption", ops.Token, map[string]any{"reason": "الكاميرا مكسورة"}); r.Code != 200 {
		t.Fatalf("السماح: %s", r)
	}
	me := h.Call("GET", "/api/v1/driver/me", drv.Token, nil, driverApp("22"))
	if me.JSON()["odometer_exempt"] != true {
		t.Errorf("odometer_exempt = %v", me.JSON()["odometer_exempt"])
	}
	if r := odoShift(t, h, drv, true, "", true); r.Code != 200 {
		t.Fatalf("فتحٌ بالسماح: %s", r)
	}
	var by, why string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(exempt_by::text, ''), exempt_reason FROM driver_shifts
		WHERE driver_id = $1::uuid AND closed_at IS NULL`, drv.ID).Scan(&by, &why); err != nil {
		t.Fatal(err)
	}
	if by != ops.ID || why != "الكاميرا مكسورة" {
		t.Fatalf("الدوام بلا أثر السماح: %q %q", by, why)
	}
	if r := odoShift(t, h, drv, false, "", true); r.Err() != "odometer_required" {
		t.Fatalf("السكّر بالسماح: %s — صورة الليل حصراً", r)
	}
	if r := odoShift(t, h, drv, false, odoReading(t, h, drv, "end", "900"), true); r.Code != 200 {
		t.Fatal(r)
	}
	if r := odoShift(t, h, drv, true, "", true); r.Err() != "odometer_required" {
		t.Fatalf("الفتح الثاني بلا صورة: %s — السماح لمرّة واحدة", r)
	}
	// والسائق لا يسمح لنفسه
	if r := h.POST("/api/v1/admin/drivers/"+drv.ID+"/odometer-exemption", drv.Token, map[string]any{"reason": "x"}); r.Code != 403 {
		t.Errorf("السائق سمح لنفسه: %d", r.Code)
	}
}

// odoDayOf يومُ السائق بالمراقبة.
func odoDayOf(t *testing.T, h *Harness, admin, driver, day string) odoDayResp {
	t.Helper()
	res := h.GET(fmt.Sprintf("/api/v1/admin/driver-monitor/%s?day=%s", driver, day), admin)
	if res.Code != 200 {
		t.Fatalf("يوم السائق %s: %s", day, res)
	}
	var env struct {
		Data odoDayResp `json:"data"`
	}
	if err := json.Unmarshal(res.Body, &env); err != nil {
		t.Fatal(err)
	}
	return env.Data
}

// TestDriverMonitor_ShiftCountedOnOneDay **الدوامُ ليومٍ واحد**: يومُ الحساب يبدأ ٩ والدوامُ فُتح
// ٨ وسُكّر ٨ المساء (١٠٠٠ → ١٠٥٠) ⇒ يُعدّ بيوم سكّره وحدَه، لا بيومين؛ و«مشى خارج الدوام» من سكّر
// دوام الأمس (٩٩٠) إلى فتحه = ١٠ كم يظهر بيومه — **و«الدوامُ اللي قبله» ليس الدوامَ نفسَه.**
func TestDriverMonitor_ShiftCountedOnOneDay(t *testing.T) {
	h := New(t)
	h.Setting("platform.day_start_hour", "9")
	admin := h.NewUser("admin")
	drv := h.NewUser("driver")
	day, base := rdDayBase() // يوم D الساعة ١٠
	prevDay := base.AddDate(0, 0, -1).Format("2006-01-02")
	shift := func(opened, closed time.Time, startKm, endKm float64) {
		s := odoInsertReading(t, h, drv.ID, "start", startKm, opened, nil)
		e := odoInsertReading(t, h, drv.ID, "end", endKm, closed, nil)
		if _, err := h.Pool.Exec(ctxBG(), `
			INSERT INTO driver_shifts (driver_id, opened_at, closed_at, close_reason, start_reading_id,
			                           end_reading_id, opened_build)
			VALUES ($1::uuid, $2, $3, 'driver', $4::uuid, $5::uuid, 22)`,
			drv.ID, opened, closed, s, e); err != nil {
			t.Fatal(err)
		}
	}
	shift(base.Add(-16*time.Hour), base.Add(-12*time.Hour), 980, 990) // أمس ١٨ ← ٢٢
	shift(base.Add(-2*time.Hour), base.Add(10*time.Hour), 1000, 1050) // D ٨ ← ٢٠ (عبر الساعة ٩)

	today := odoDayOf(t, h, admin.Token, drv.ID, day)
	if today.Odometer.OnKm == nil || *today.Odometer.OnKm != 50 {
		t.Errorf("يوم D: حسب العدّاد = %v والمتوقَّع 50", today.Odometer.OnKm)
	}
	if today.Odometer.OffKm == nil || *today.Odometer.OffKm != 10 {
		t.Errorf("يوم D: خارج الدوام = %v والمتوقَّع 10 (من سكّر الأمس ٩٩٠ لفتح ١٠٠٠)", today.Odometer.OffKm)
	}
	prev := odoDayOf(t, h, admin.Token, drv.ID, prevDay)
	if prev.Odometer.OnKm == nil || *prev.Odometer.OnKm != 10 {
		t.Errorf("يوم D-1: حسب العدّاد = %v والمتوقَّع 10 — دوامُ D انعدّ بيومين", prev.Odometer.OnKm)
	}
}

// TestOdometer_PhotoPurgedNumberKept **صورةُ العدّاد تُمحى بعد مدّة حفظها والرقمُ يبقى**: صورةٌ
// عمرها مئة يوم (والمدّةُ تسعون) ⇒ كانسُ المسافات يمحو صفَّ الوسيط وملفّاته، و`media_id` يصير
// فارغاً، والرقمُ باقٍ؛ وصورةُ اليوم لا تُمسّ.
func TestOdometer_PhotoPurgedNumberKept(t *testing.T) {
	h := New(t)
	h.Setting("launch.driver_work", "true")
	h.Setting("drivers.odometer_photo_retention_days", "90")
	drv := h.NewUser("driver")
	old := odoReading(t, h, drv, "start", "1250")
	fresh := odoReading(t, h, drv, "start", "1260")
	paths := func(reading string) (string, []string) {
		var mediaID, path, thumb string
		if err := h.Pool.QueryRow(ctxBG(), `
			SELECT m.id::text, m.path, m.thumb_path FROM driver_odometer_readings r
			  JOIN media m ON m.id = r.media_id WHERE r.id = $1::uuid`, reading).Scan(&mediaID, &path, &thumb); err != nil {
			t.Fatalf("وسيطُ القراءة %s: %v", reading, err)
		}
		return mediaID, []string{filepath.Join(h.MediaDir, filepath.FromSlash(path)),
			filepath.Join(h.MediaDir, filepath.FromSlash(thumb))}
	}
	oldMedia, oldFiles := paths(old)
	_, freshFiles := paths(fresh)
	for _, f := range append(append([]string{}, oldFiles...), freshFiles...) {
		if _, err := os.Stat(f); err != nil {
			t.Fatalf("ملفّ الصورة غير موجود قبل الكنس: %s", f)
		}
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE media SET created_at = now() - interval '100 days' WHERE id = $1::uuid`, oldMedia); err != nil {
		t.Fatal(err)
	}
	h.API.SweepDrives(context.Background(), time.Now())

	var left int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM media WHERE id = $1::uuid`, oldMedia).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatal("صفُّ وسيطِ الصورة القديمة باقٍ بعد الكنس")
	}
	for _, f := range oldFiles {
		if _, err := os.Stat(f); !os.IsNotExist(err) {
			t.Errorf("ملفُّ الصورة القديمة باقٍ: %s (%v)", f, err)
		}
	}
	var km float64
	var mediaID *string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT km::float8, media_id::text FROM driver_odometer_readings WHERE id = $1::uuid`, old).
		Scan(&km, &mediaID); err != nil {
		t.Fatal(err)
	}
	if km != 1250 || mediaID != nil {
		t.Fatalf("القراءة بعد المحو: رقم=%v وسيط=%v — المتوقَّع 1250 وبلا وسيط", km, mediaID)
	}
	for _, f := range freshFiles {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("صورةُ اليوم مُحيت: %s", f)
		}
	}
}
