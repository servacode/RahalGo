package qa

// ══════════════════════════════════════════════════════════════════════
// **عينُ الأمان عبر الموجّه كلِّه** (مراقبةُ المنصّة — الدفعةُ الثانية ٢٠٢٦-١٠-١٧) — `SECW-*`
// ══════════════════════════════════════════════════════════════════════
//
// **الحماياتُ القائمة تُقاد كما يقودها المهاجم** — كلماتُ سرٍّ غلطٌ حتّى القفل،
// كودُ تسليمٍ غلطٌ حتّى القفل، صورةُ تسليمٍ بموقعٍ مزيَّف — **ثمّ تُقرأ رسالةُ
// المالك كما كانت ستصله على واتسابه**: بآخر ثلاثة أرقامٍ لا بالرقم، وبرقم الطلب،
// وباسم السائق. **وسطرُ الأمان في تقرير الصباح.**

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// secwSender **يلتقط ما كان سيُرسَل للمالك على الواتساب.**
type secwSender struct {
	mu   sync.Mutex
	msgs []string
}

func (c *secwSender) SendToSelf(_ context.Context, text string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if text == "" {
		return errors.New("empty")
	}
	c.msgs = append(c.msgs, text)
	return nil
}

// with **الرسائلُ التي تبدأ بهذا** — وغيرُها (طلبٌ عالقٌ من فحصٍ آخر) لا يعني الفحص.
func (c *secwSender) with(prefix string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []string
	for _, m := range c.msgs {
		if strings.HasPrefix(m, prefix) {
			out = append(out, m)
		}
	}
	return out
}

func (c *secwSender) all() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.msgs...)
}

// secwHarness **مِسنَدٌ بمُرسِلٍ ملتقِط** — والتقريرُ الصباحيُّ والدفترُ مطفآن في
// النبضة كي لا يسبقا الفحص (التقريرُ يُنادى صراحةً).
func secwHarness(t *testing.T) (*Harness, *secwSender) {
	t.Helper()
	h := New(t)
	h.Setting("monitoring.enabled", "true")
	h.Setting("monitoring.security_alerts", "true")
	h.Setting("monitoring.morning_report", "false")
	h.Setting("monitoring.money_check", "false")
	out := &secwSender{}
	h.API.SetAlertSender(out)
	return h, out
}

// TestSECW_LoginLockAlertMaskedPhone **كلماتٌ غلطٌ حتّى القفل ⇒ تنبيهٌ بآخر ثلاثة أرقام**،
// مرّةً لا أكثر، **وسطرُ الأمان في تقرير الصباح.**
func TestSECW_LoginLockAlertMaskedPhone(t *testing.T) {
	h, out := secwHarness(t)
	u := h.NewUser("merchant")
	hash, _ := auth.HashPassword("Right12345x")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET password_hash = $2 WHERE id = $1`, u.ID, hash); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 5; i++ {
		r := h.POST("/api/v1/auth/login", "", map[string]any{"phone": u.Phone, "password": "Wrong" + strconv.Itoa(i) + "xxxxx"})
		if r.Err() != "invalid_credentials" {
			t.Fatalf("SECW-1 المحاولةُ %d: %s — يُنتظر invalid_credentials", i, r)
		}
	}
	if r := h.POST("/api/v1/auth/login", "", map[string]any{"phone": u.Phone, "password": "Right12345x"}); r.Err() != "too_many_attempts" {
		t.Fatalf("SECW-1 الحمايةُ تبدّلت — بعد خمسٍ يُنتظر too_many_attempts: %s", r)
	}
	h.API.MonitorTickForTest(ctxBG(), time.Now())
	got := out.with("🔐 حساب انقفل")
	tail := u.Phone[len(u.Phone)-3:]
	if len(got) != 1 || !strings.HasPrefix(got[0], "🔐 حساب انقفل من كتر محاولات كلمة السر: رقم ينتهي بـ …"+tail+"\n") {
		t.Fatalf("SECW-1 تنبيهُ القفل = %q — يُنتظر واحدٌ بـ…%s", got, tail)
	}
	t.Logf("التنبيه:\n%s", got[0])
	// **ولا رقمٌ كاملٌ ولا كلمةُ سرٍّ في أيّ رسالة.**
	digits := strings.TrimPrefix(u.Phone, "+")
	for _, m := range out.all() {
		for _, leak := range []string{u.Phone, digits, digits[len(digits)-6:], "Wrong1xxxxx", "Right12345x"} {
			if strings.Contains(m, leak) {
				t.Fatalf("SECW-1 رسالةٌ للمالك تحمل %q:\n%s", leak, m)
			}
		}
	}
	// **والقفلُ القائمُ لا يُعيد التنبيه.**
	for range 3 {
		h.POST("/api/v1/auth/login", "", map[string]any{"phone": u.Phone, "password": "Wrong9xxxxx"})
	}
	h.API.MonitorTickForTest(ctxBG(), time.Now())
	if n := len(out.with("🔐 حساب انقفل")); n != 1 {
		t.Fatalf("SECW-1 تنبيهُ القفل تكرّر — %d", n)
	}

	// **وتقريرُ صباح الغد يحمل سطرَ الأمان** — من الجدول بعد الكتابة.
	rep := h.API.MorningReportForTest(ctxBG(), time.Now().Add(24*time.Hour))
	var line string
	for _, l := range strings.Split(rep, "\n") {
		if strings.HasPrefix(l, "الأمان مبارح: ") {
			line = l
		}
	}
	if line == "" || strings.Contains(line, "ولا محاولة مشبوهة") ||
		!strings.Contains(line, "محاولة دخول غلط · ") || !strings.Contains(line, "حساب انقفل · ") {
		t.Fatalf("SECW-4 تقريرُ الصباح بلا سطر الأمان المنتظر:\n%s", rep)
	}
	t.Logf("سطرُ الأمان: %s", line)
}

// TestSECW_DeliveryCodeLockAlert **كودٌ غلطٌ حتّى القفل ⇒ تنبيهٌ برقم الطلب وحدَه.**
func TestSECW_DeliveryCodeLockAlert(t *testing.T) {
	h, out := secwHarness(t)
	h.Setting("delivery.code_required", "true")
	oid, _, drv := dcodeOrder(t, h)
	code, _ := dcodeOf(t, h, oid)
	if code == nil {
		t.Fatal("SECW-2 لم يولد كود")
	}
	for i := 1; i <= 5; i++ {
		dcodeDeliver(h, drv, oid, wrongOf(*code))
	}
	if got := dcodeDeliver(h, drv, oid, *code); got.Err() != "delivery_code_locked" {
		t.Fatalf("SECW-2 الحمايةُ تبدّلت — بعد خمسٍ يُنتظر delivery_code_locked: %s", got)
	}
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1::uuid`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}
	h.API.MonitorTickForTest(ctxBG(), time.Now())
	got := out.with("🔢 كود التسليم انقفل")
	want := "🔢 كود التسليم انقفل على الطلب #" + strconv.FormatInt(number, 10) + "\n"
	if len(got) != 1 || !strings.HasPrefix(got[0], want) {
		t.Fatalf("SECW-2 تنبيهُ الكود = %q — يُنتظر %q", got, want)
	}
	t.Logf("التنبيه:\n%s", got[0])
}

// TestSECW_MockedProofAlert **صورةُ تسليمٍ بموقعٍ مزيَّف ⇒ تنبيهٌ باسم السائق ورقم الطلب.**
func TestSECW_MockedProofAlert(t *testing.T) {
	h, out := secwHarness(t)
	h.Setting("delivery.code_required", "false")
	oid, _, drv := dcodeOrder(t, h)
	name := "سائق الفحص " + uniq("n")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET full_name = $2 WHERE id = $1::uuid`, drv.ID, name); err != nil {
		t.Fatal(err)
	}
	var img bytes.Buffer
	if err := png.Encode(&img, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "door.png")
	_, _ = part.Write(img.Bytes())
	_ = mw.WriteField("lat", "33.5138")
	_ = mw.WriteField("lng", "36.2765")
	_ = mw.WriteField("mocked", "true")
	_ = mw.Close()
	req, _ := http.NewRequest("POST", h.Srv.URL+"/api/v1/driver/orders/"+oid+"/proof", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+drv.Token)
	resp, err := h.Srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(body), `"proof_mocked"`) {
		t.Fatalf("SECW-3 الحمايةُ تبدّلت — يُنتظر 422 proof_mocked: %d %s", resp.StatusCode, body)
	}
	var number int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT number FROM orders WHERE id = $1::uuid`, oid).Scan(&number); err != nil {
		t.Fatal(err)
	}
	h.API.MonitorTickForTest(ctxBG(), time.Now())
	got := out.with("📍 سائق حاول يزيّف موقعه")
	want := "📍 سائق حاول يزيّف موقعه: " + name + " — الطلب #" + strconv.FormatInt(number, 10) + "\n"
	if len(got) != 1 || !strings.HasPrefix(got[0], want) {
		t.Fatalf("SECW-3 تنبيهُ التزييف = %q — يُنتظر %q", got, want)
	}
	t.Logf("التنبيه:\n%s", got[0])
}
