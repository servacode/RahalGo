package qa

// **فلترة صور وتسجيلات الدردشة** — قرار المالك ٢٠٢٦-١١-٢٠.
//
// صورة فيها رقم جوال ← بتنحجز (ما بتوصل للطرف التاني)، والمرسل بياخد chat_photo_held، والإدارة بتشوفها
// وبتمرّرها. صورة فيها سعر بس ← بتوصل عادي. تسجيل فيه رقم محكي ← بيوصل، وبعدين بيتعلّم للإدارة
// وبينكتب إنذار عالمرسل.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"
)

type fakeScan struct{ ocr, voice string }

func (f *fakeScan) OCR(context.Context, string) (string, error)        { return f.ocr, nil }
func (f *fakeScan) Transcribe(context.Context, string) (string, error) { return f.voice, nil }
func (f *fakeScan) PhotoReady() bool                                   { return true }
func (f *fakeScan) VoiceReady() bool                                   { return true }

func chatBodies(t *testing.T, h *Harness, token, order string) []map[string]any {
	t.Helper()
	r := h.GET("/api/v1/orders/"+order+"/messages", token)
	if r.Code != http.StatusOK {
		t.Fatalf("الحديث: %s", r)
	}
	var env struct {
		Data any `json:"data"`
	}
	_ = json.Unmarshal(r.Body, &env)
	arr, _ := env.Data.([]any)
	if m, ok := env.Data.(map[string]any); ok {
		for _, k := range []string{"messages", "items", "lines"} {
			if a, ok := m[k].([]any); ok {
				arr = a
			}
		}
	}
	var out []map[string]any
	for _, x := range arr {
		if m, ok := x.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func TestCHATSCAN_PhotoWithPhoneHeldVoiceFlagged(t *testing.T) {
	h := New(t)
	fx := newTenureFx(t, h)
	acceptOrder(t, h, fx.DrvA, fx.Order)
	sc := &fakeScan{ocr: "سعر الكيلو 25000"}
	h.API.SetMediaScanner(sc)

	// ── ١ · صورة فيها سعر بس ← بتوصل ──────────────────────────────────────
	if r := h.sendAttachment(fx.DrvA.Token, fx.Order, "photo", qaJPEG(t), nil); r.Code != http.StatusCreated {
		t.Fatalf("صورة بلا رقم: %s", r)
	}
	ageChat(t, h, fx.Order)

	// ── ٢ · صورة فيها رقم ← محجوزة ───────────────────────────────────────
	sc.ocr = "اتصل على 0991 234 567"
	r := h.sendAttachment(fx.DrvA.Token, fx.Order, "photo", qaJPEG(t), nil)
	if r.Code != http.StatusConflict || r.Err() != "chat_photo_held" {
		t.Fatalf("**صورة فيها رقم**: %s — يُنتظر 409 chat_photo_held", r)
	}
	var held int
	var heldID string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*), COALESCE(max(id::text), '') FROM order_messages
		WHERE order_id = $1::uuid AND held_at IS NOT NULL AND contact_leak`, fx.Order).Scan(&held, &heldID)
	if held != 1 {
		t.Fatalf("**الصورة المحجوزة ما انكتبت للإدارة**: %d", held)
	}
	photos := func(token string) int {
		n := 0
		for _, m := range chatBodies(t, h, token, fx.Order) {
			if a, _ := m["attachment"].(map[string]any); a != nil && a["kind"] == "photo" {
				n++
			}
		}
		return n
	}
	if n := photos(fx.Cust.Token); n != 1 {
		t.Errorf("**الزبون شاف %d صور** — يُنتظر ١ (المحجوزة ما بتوصل)", n)
	}

	// ── ٣ · الإدارة بتشوفها وبتمرّرها ───────────────────────────────────────
	ops := h.NewUser("admin")
	a := h.GET("/api/v1/admin/orders/"+fx.Order+"/chat", ops.Token)
	if a.Code != http.StatusOK || !strings.Contains(string(a.Body), `"held":true`) || !strings.Contains(string(a.Body), "0991") {
		t.Errorf("**الإدارة ما شافت المحجوزة ونصها**: %s", a)
	}
	if rr := h.POST("/api/v1/admin/order-messages/"+heldID+"/release", ops.Token, map[string]any{}); rr.Code != http.StatusOK {
		t.Fatalf("مرّرها: %s", rr)
	}
	if n := photos(fx.Cust.Token); n != 2 {
		t.Errorf("**بعد التمرير الزبون شاف %d** — يُنتظر ٢", n)
	}
	if rr := h.POST("/api/v1/admin/order-messages/"+heldID+"/release", ops.Token, map[string]any{}); rr.Code != http.StatusConflict {
		t.Errorf("**مرّرها مرتين**: %s", rr)
	}
	ageChat(t, h, fx.Order)

	// ── ٤ · تسجيل فيه رقم محكي ← بيوصل، وبعدين وسم وإنذار ────────────────────
	sc.voice = "حطلي رقمي صفر تسعة تسعة واحد اتنين تلاتة اربعة خمسة ستة سبعة"
	if r := h.sendAttachment(fx.DrvA.Token, fx.Order, "audio", qaM4A(), map[string]string{"duration_ms": "8000"}); r.Code != http.StatusCreated {
		t.Fatalf("التسجيل: %s", r)
	}
	var flagged, warns int
	for i := 0; i < 50; i++ {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_messages
			WHERE order_id = $1::uuid AND attachment_kind = 'audio' AND contact_leak AND scan_text <> ''`, fx.Order).Scan(&flagged)
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM warnings WHERE user_id = $1::uuid AND order_id = $2::uuid`,
			fx.DrvA.ID, fx.Order).Scan(&warns)
		if flagged == 1 && warns == 1 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if flagged != 1 || warns != 1 {
		t.Errorf("**التسجيل**: موسوم %d · إنذارات %d — يُنتظر ١ و١", flagged, warns)
	}
}
