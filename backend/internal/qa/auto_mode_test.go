package qa

// ══════════════════════════════════════════════════════════════════════
//  **زرُّ «تلقائي/يدوي» لحساب العمليات** (ملاحظةُ المالك ٢٠٢٦-١٠-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// **كان الزرُّ لا يظهر للعمليات** — يقرأ لوحَ الإعدادات ولا قدرةَ لهم
// عليه. **فالبابُ الضيّق** (`auto_mode.go`): موظّفُ العمليات يقلب زرَّ
// الطلبات، **ولا يصل إلى لوح الإعدادات ولا إلى مفتاحٍ ثالث.**

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

func amState(t *testing.T, h *Harness, kind, tok string) (on, editable bool) {
	t.Helper()
	r := h.GET("/api/v1/admin/auto-mode/"+kind, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("قراءةُ زرّ %s: %d %s", kind, r.Code, r.Body)
	}
	var env struct {
		Data struct {
			On       bool `json:"on"`
			Editable bool `json:"editable"`
		} `json:"data"`
	}
	_ = json.Unmarshal(r.Body, &env)
	return env.Data.On, env.Data.Editable
}

func TestAM01_OperationsFlipsOrdersAutoMode(t *testing.T) {
	h := New(t)
	h.Setting("orders.auto_transfer", "false")
	staff := h.NewUser("operations")

	on, editable := amState(t, h, "orders", staff.Token)
	if on || !editable {
		t.Fatalf("**العمليات يجب أن ترى الزرَّ قابلاً للتبديل** — on=%v editable=%v", on, editable)
	}
	if r := h.Call("PUT", "/api/v1/admin/auto-mode/orders", staff.Token, map[string]any{"on": true}, nil); r.Code != http.StatusOK {
		t.Fatalf("تبديلُ زرّ الطلبات: %d %s", r.Code, r.Body)
	}
	if on, _ := amState(t, h, "orders", staff.Token); !on {
		t.Fatal("**الزرُّ لم يُحفظ**")
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log WHERE action = 'admin.setting_update' AND entity_id = 'orders.auto_transfer'`).Scan(&n)
	if n == 0 {
		t.Fatal("**تبديلٌ بلا قيدٍ في سجلّ التدقيق**")
	}

	// **ولا يكتب في لوح الإعدادات** — يقرؤه (وهو حالُه قبل اليوم) ولا يبدّل
	// فيه مفتاحاً، **ولا هذا المفتاحَ نفسَه من غير بابه.**
	if r := h.Call("PUT", "/api/v1/admin/settings/orders.auto_transfer", staff.Token, map[string]any{"value": false}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**العمليات كتبت مفتاحَ الطلبات من اللوح** — %d", r.Code)
	}
	if r := h.Call("PUT", "/api/v1/admin/settings/leads.auto_approve", staff.Token, map[string]any{"value": true}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**العمليات كتبت إعداداً من اللوح** — %d", r.Code)
	}
}

func TestAM02_EachDoorItsOwnCapability(t *testing.T) {
	h := New(t)
	h.Setting("leads.auto_approve", "false")

	if r := h.GET("/api/v1/admin/auto-mode/orders", ""); r.Code != http.StatusUnauthorized {
		t.Fatalf("**بلا جلسةٍ يجب ٤٠١** — %d", r.Code)
	}

	// **من يقرأ الطلبات ولا يتدخّل** يرى الحالَ ولا يبدّله.
	capRole(t, h, "qa-am-reader", authz.OrdersRead)
	_, reader := capUser(t, h, "qa-am-reader")
	if _, editable := amState(t, h, "orders", reader); editable {
		t.Fatal("**قارئُ الطلبات وُعد بزرٍّ قابلٍ للتبديل**")
	}
	if r := h.Call("PUT", "/api/v1/admin/auto-mode/orders", reader, map[string]any{"on": true}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**قارئُ الطلبات بدّل الزرّ** — %d", r.Code)
	}
	if r := h.Call("PUT", "/api/v1/admin/auto-mode/leads", reader, map[string]any{"on": true}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**قدرةُ الطلبات فتحت بابَ الانضمام** — %d", r.Code)
	}

	// **ومقرِّرُ الانضمام يبدّل زرَّه هو.**
	capRole(t, h, "qa-am-verify", authz.MerchantsVerify)
	_, verifier := capUser(t, h, "qa-am-verify")
	if r := h.Call("PUT", "/api/v1/admin/auto-mode/leads", verifier, map[string]any{"on": true}, nil); r.Code != http.StatusOK {
		t.Fatalf("تبديلُ زرّ الانضمام: %d %s", r.Code, r.Body)
	}
	if on, _ := amState(t, h, "leads", verifier); !on {
		t.Fatal("**زرُّ الانضمام لم يُحفظ**")
	}
	if r := h.Call("PUT", "/api/v1/admin/auto-mode/orders", verifier, map[string]any{"on": true}, nil); r.Code != http.StatusForbidden {
		t.Fatalf("**قدرةُ الانضمام فتحت زرَّ الطلبات** — %d", r.Code)
	}
}
