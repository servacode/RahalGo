package qa

// ══════════════════════════════════════════════════════════════════════
// **كودُ التسليم** (قرارُ المالك ٢٠٢٦-١٠-٠٦) — المعرّفات `DCODE-*`
// ══════════════════════════════════════════════════════════════════════
//
// أربعةُ أرقامٍ تُولَّد عند الاستلام وتصل الزبون، **ولا «سُلّم» بلا كودها.**
// والخطأُ يُعدّ وخمسٌ تُقفل، **والعمليّاتُ تقرؤه وتفكّه** (`proof-exception`).
// **والسائقُ لا يرى الكود أبداً.** ومطفأٌ أو بلا قناةٍ ⇒ لا كود.

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// dcodeFakeBot **بوتُ واتسابٍ مقيس** — جاهزٌ دائماً، ونتيجتُه بيد الفحص.
type dcodeFakeBot struct {
	fail  bool
	calls atomic.Int32
}

func (b *dcodeFakeBot) Ready() bool { return true }
func (b *dcodeFakeBot) SendText(context.Context, string, string) error {
	b.calls.Add(1)
	if b.fail {
		return errors.New("bot down")
	}
	return nil
}

// dcodeOrder **طلبٌ مستلَمٌ بلغ باب الزبون** — والكودُ وُلد (أو لم يولد) عند الاستلام.
func dcodeOrder(t *testing.T, h *Harness) (oid string, cust, drv *User) {
	t.Helper()
	return dcodeOrderWith(t, h, nil)
}

// dcodeOrderWith **ومعه ما يُركَّب قبل الاستلام** — بوتُ الواتساب يُحقن بعد الإسناد
// فلا يسابق إبلاغُ المتجر الإسنادَ المباشر.
func dcodeOrderWith(t *testing.T, h *Harness, beforePickup func()) (oid string, cust, drv *User) {
	t.Helper()
	cust = h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("DCODE تعذّر إنشاءُ الطلب: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	drv = h.driverOf(oid)
	if beforePickup != nil {
		beforePickup()
	}
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"} {
		got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, map[string]any{"to": to})
		if got.Code >= 400 {
			t.Fatalf("DCODE الانتقالُ إلى %s رُدّ: %s", to, got)
		}
		// **والسائقُ لا يرى الكودَ في ردّ الانتقال**.
		if strings.Contains(string(got.Body), `"delivery_code":`) {
			t.Fatalf("DCODE ردُّ الانتقال يحمل الكودَ للسائق: %s", got)
		}
	}
	// **وإثباتُ الصورة يُستوفى** — فيبقى الكودُ وحدَه حارساً.
	h.authorizeProofExempt(oid)
	return oid, cust, drv
}

func dcodeOf(t *testing.T, h *Harness, oid string) (code *string, attempts int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT delivery_code, delivery_code_attempts FROM orders WHERE id = $1::uuid`, oid).
		Scan(&code, &attempts); err != nil {
		t.Fatalf("DCODE قراءةُ الكود: %v", err)
	}
	return code, attempts
}

func dcodeDeliver(h *Harness, drv *User, oid string, code any) Res {
	body := map[string]any{"to": "delivered"}
	if code != nil {
		body["delivery_code"] = code
	}
	return h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, body)
}

// wrongOf **كودٌ غيرُ الصحيح** — بأربعة أرقام.
func wrongOf(code string) string {
	if code == "0000" {
		return "1111"
	}
	return "0000"
}

// TestDCODE_WrongThenCorrect **غائبٌ وخاطئٌ ⇒ 422 ويُعدّ، والصحيحُ يُسلّم.**
func TestDCODE_WrongThenCorrect(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	h.Setting("delivery.code_channel", `"both"`)
	oid, cust, drv := dcodeOrder(t, h)

	code, attempts := dcodeOf(t, h, oid)
	if code == nil || len(*code) != 4 || attempts != 0 {
		t.Fatalf("DCODE-1 لم يولد كودٌ بأربعة أرقام عند الاستلام (%v · %d)", code, attempts)
	}
	// **والزبونُ يصله الكود في التطبيق.**
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND body LIKE '%' || $3 || '%'`,
		cust.ID, oid, *code).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("DCODE-2 إشعارُ الكود للزبون = %d — يُنتظر ١", n)
	}

	// **قائمةُ السائق تقول «مطلوب» ولا تقول الكود.**
	list := h.GET("/api/v1/driver/orders", drv.Token)
	if list.Code != http.StatusOK || !strings.Contains(string(list.Body), `"delivery_code_required":true`) {
		t.Fatalf("DCODE-3 قائمةُ السائق بلا delivery_code_required=true: %s", list)
	}
	if strings.Contains(string(list.Body), `"delivery_code":`) {
		t.Fatalf("DCODE-3 قائمةُ السائق تحمل الكودَ نفسَه: %s", list)
	}

	if got := dcodeDeliver(h, drv, oid, nil); got.Code != http.StatusUnprocessableEntity || got.Err() != "delivery_code_wrong" {
		t.Fatalf("DCODE-4 تسليمٌ بلا كود: %s — يُنتظر 422 delivery_code_wrong", got)
	}
	if got := dcodeDeliver(h, drv, oid, wrongOf(*code)); got.Code != http.StatusUnprocessableEntity || got.Err() != "delivery_code_wrong" {
		t.Fatalf("DCODE-4 تسليمٌ بكودٍ خاطئ: %s — يُنتظر 422 delivery_code_wrong", got)
	}
	if _, a := dcodeOf(t, h, oid); a != 2 {
		t.Fatalf("DCODE-4 المحاولاتُ = %d — يُنتظر ٢", a)
	}
	if h.statusOf(oid) != "at_dropoff" {
		t.Fatal("DCODE-4 سُلّم الطلبُ بكودٍ خاطئ")
	}
	if got := dcodeDeliver(h, drv, oid, *code); got.Code >= 400 {
		t.Fatalf("DCODE-5 الكودُ الصحيحُ رُدّ: %s", got)
	}
	if s := h.statusOf(oid); s != "delivered" {
		t.Fatalf("DCODE-5 الحالُ %q بعد الكود الصحيح", s)
	}
}

// TestDCODE_FiveWrongLocks **خمسٌ خاطئةٌ تُقفل — والعمليّاتُ تقرأ الكودَ وتفكّه.**
func TestDCODE_FiveWrongLocks(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	oid, _, drv := dcodeOrder(t, h)
	code, _ := dcodeOf(t, h, oid)
	if code == nil {
		t.Fatal("DCODE لم يولد كود")
	}
	wrong := wrongOf(*code)
	for i := 1; i <= 4; i++ {
		if got := dcodeDeliver(h, drv, oid, wrong); got.Code != http.StatusUnprocessableEntity {
			t.Fatalf("DCODE-6 المحاولةُ %d: %s — يُنتظر 422", i, got)
		}
	}
	if got := dcodeDeliver(h, drv, oid, wrong); got.Code != http.StatusLocked || got.Err() != "delivery_code_locked" {
		t.Fatalf("DCODE-6 الخامسةُ: %s — يُنتظر 423 delivery_code_locked", got)
	}
	// **والصحيحُ بعد القفل لا يمرّ.**
	if got := dcodeDeliver(h, drv, oid, *code); got.Code != http.StatusLocked {
		t.Fatalf("DCODE-6 الكودُ الصحيحُ بعد القفل: %s — يُنتظر 423", got)
	}
	if _, a := dcodeOf(t, h, oid); a != 5 {
		t.Fatalf("DCODE-6 المحاولاتُ = %d بعد القفل — يُنتظر ٥", a)
	}

	// **قارئٌ بلا تدخّلٍ لا يرى الكود · والمُخوَّلُ يراه.**
	capRole(t, h, "qa_dcode_reader", authz.OrdersRead)
	_, readTok := capUser(t, h, "qa_dcode_reader")
	if got := h.GET("/api/v1/admin/orders/"+oid, readTok); got.Code != http.StatusOK ||
		strings.Contains(string(got.Body), `"delivery_code":`) {
		t.Fatalf("DCODE-7 قارئٌ بلا orders.intervene: %s — يُنتظر 200 بلا الكود", got)
	}
	capRole(t, h, "qa_dcode_ops", authz.OrdersRead, authz.OrdersIntervene)
	_, opsTok := capUser(t, h, "qa_dcode_ops")
	if got := h.GET("/api/v1/admin/orders/"+oid, opsTok); got.Code != http.StatusOK ||
		!strings.Contains(string(got.Body), `"delivery_code":"`+*code+`"`) {
		t.Fatalf("DCODE-7 المُخوَّلُ لا يرى الكود: %s", got)
	}

	// **وإذنُ الاستثناء يفكّه.**
	if got := h.POST("/api/v1/admin/orders/"+oid+"/proof-exception", opsTok,
		map[string]any{"reason": "تحقّقنا من الزبون بالهاتف"}); got.Code >= 400 {
		t.Fatalf("DCODE-8 إذنُ الاستثناء رُدّ: %s", got)
	}
	if c, a := dcodeOf(t, h, oid); c != nil || a != 0 {
		t.Fatalf("DCODE-8 الكودُ باقٍ بعد الإذن (%v · %d)", c, a)
	}
	if got := dcodeDeliver(h, drv, oid, nil); got.Code >= 400 {
		t.Fatalf("DCODE-8 التسليمُ رُدّ بعد الإذن: %s", got)
	}
	if s := h.statusOf(oid); s != "delivered" {
		t.Fatalf("DCODE-8 الحالُ %q", s)
	}
}

// TestDCODE_OffNoCode **مطفأٌ ⇒ لا كود، والتسليمُ كما كان.**
func TestDCODE_OffNoCode(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "false")
	oid, _, drv := dcodeOrder(t, h)
	if c, _ := dcodeOf(t, h, oid); c != nil {
		t.Fatalf("DCODE-9 وُلد كودٌ والمفتاحُ مطفأ: %v", *c)
	}
	list := h.GET("/api/v1/driver/orders", drv.Token)
	if !strings.Contains(string(list.Body), `"delivery_code_required":false`) {
		t.Fatalf("DCODE-9 القائمةُ بلا delivery_code_required=false: %s", list)
	}
	if got := dcodeDeliver(h, drv, oid, nil); got.Code >= 400 {
		t.Fatalf("DCODE-9 التسليمُ رُدّ والمفتاحُ مطفأ: %s", got)
	}
}

// TestDCODE_NoChannelNoCode **واتسابٌ وحدَه بلا بوت ⇒ لا كود** — فلا يقف تسليمٌ على رسالةٍ لم تُرسَل.
func TestDCODE_NoChannelNoCode(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	h.Setting("delivery.code_channel", `"whatsapp"`)
	h.API.SetMerchantNotifier(nil)
	oid, _, drv := dcodeOrder(t, h)
	if c, _ := dcodeOf(t, h, oid); c != nil {
		t.Fatalf("DCODE-10 وُلد كودٌ بلا قناة: %v", *c)
	}
	if got := dcodeDeliver(h, drv, oid, nil); got.Code >= 400 {
		t.Fatalf("DCODE-10 التسليمُ رُدّ بلا كود: %s", got)
	}
}

// TestDCODE_WhatsAppFailureClears **واتسابٌ وحدَه تعثّر ⇒ يُمحى الكود** · ونجح ⇒ يبقى.
func TestDCODE_WhatsAppFailureClears(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	h.Setting("delivery.code_channel", `"whatsapp"`)
	t.Cleanup(func() { h.API.SetMerchantNotifier(nil) })

	ok := &dcodeFakeBot{}
	oid, _, _ := dcodeOrderWith(t, h, func() { h.API.SetMerchantNotifier(ok) })
	waitCalls(t, &ok.calls)
	if c, _ := dcodeOf(t, h, oid); c == nil {
		t.Fatal("DCODE-11 الواتسابُ وصل والكودُ مُحي")
	}

	bad := &dcodeFakeBot{fail: true}
	h.API.SetMerchantNotifier(nil)
	oid2, _, _ := dcodeOrderWith(t, h, func() { h.API.SetMerchantNotifier(bad) })
	waitCalls(t, &bad.calls)
	deadline := time.Now().Add(5 * time.Second)
	for {
		c, _ := dcodeOf(t, h, oid2)
		if c == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("DCODE-11 الواتسابُ تعثّر والكودُ باقٍ — يقف التسليمُ على رسالةٍ لم تصل")
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func waitCalls(t *testing.T, c *atomic.Int32) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for c.Load() == 0 {
		if time.Now().After(deadline) {
			t.Fatal("DCODE الواتسابُ لم يُنادَ")
		}
		time.Sleep(20 * time.Millisecond)
	}
}
