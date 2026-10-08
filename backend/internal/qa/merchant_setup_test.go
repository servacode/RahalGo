package qa

// **إعدادُ المتاجر — مَن ضبط ومَن لم يضبط** (طلبُ المالك ٢٠٢٦-١٠-٠٨): الدوامُ
// والعنوانُ والموقعُ، **وتأكيدُ طريقة المستحقّات والاسترداد** — من التطبيق أو اللوحة.

import (
	"net/http"
	"strings"
	"testing"
)

func setupRow(t *testing.T, h *Harness, tok, id string) map[string]any {
	t.Helper()
	r := h.Call("GET", "/api/v1/admin/merchants-setup", tok, nil, nil)
	if r.Code != http.StatusOK {
		t.Fatalf("صفحةُ الإعداد: %d / %s", r.Code, r.Err())
	}
	for _, s := range r.JSON()["stores"].([]any) {
		row := s.(map[string]any)
		if row["id"] == id {
			return row
		}
	}
	t.Fatalf("المتجر غائبٌ عن الصفحة")
	return nil
}

func missingOf(row map[string]any) string {
	var out []string
	for _, x := range row["missing"].([]any) {
		out = append(out, x.(string))
	}
	return strings.Join(out, ",")
}

func TestMSU01_SetupStatusAndConfirmations(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	rep := h.NewUser("sales")
	m := f.Merchant(OwnedByRep(rep.ID))
	tok := h.TokenFor(m.Owner.ID, "merchant")
	for _, q := range []string{`UPDATE merchants SET address_text = '', location = NULL WHERE id = $1`,
		`DELETE FROM merchant_hours WHERE merchant_id = $1`} {
		if _, err := h.Pool.Exec(ctxBG(), q, m.ID); err != nil {
			t.Fatal(err)
		}
	}

	row := setupRow(t, h, admin.Token, m.ID)
	if got := missingOf(row); got != "hours,address,location,settlement,returns" {
		t.Fatalf("الناقص: %q", got)
	}
	msg, _ := row["message"].(string)
	if !strings.Contains(msg, "ساعات الدوام") || !strings.Contains(msg, "نقدي") || !strings.Contains(msg, "ترجيع") {
		t.Fatalf("الرسالةُ لا تذكر الناقص:\n%s", msg)
	}
	if u, _ := row["whatsapp_url"].(string); !strings.HasPrefix(u, "https://wa.me/") {
		t.Fatalf("رابطُ الواتساب: %q", u)
	}

	// **صاحبُ المتجر يختار من تطبيقه** — الاسترداد والطريقة والعنوان.
	if r := h.Call("PATCH", "/api/v1/merchant/stores/"+m.ID+"/settings", tok,
		map[string]any{"accepts_returns": true, "address_text": "الرقة — شارع تل أبيض", "lat": 35.95, "lng": 39.01}, nil); r.Code != http.StatusOK {
		t.Fatalf("إعداداتُه: %d / %s", r.Code, r.Err())
	}
	if r := h.Call("PATCH", "/api/v1/merchant/stores/"+m.ID+"/settlement-method", tok,
		map[string]any{"method": "cash"}, nil); r.Code != http.StatusOK {
		t.Fatalf("طريقتُه: %d / %s", r.Code, r.Err())
	}
	row = setupRow(t, h, admin.Token, m.ID)
	if got := missingOf(row); got != "hours" {
		t.Fatalf("بعد اختياره، الناقص: %q", got)
	}
	if row["returns_confirmed_by"] != "merchant" || row["settlement_confirmed_by"] != "merchant" || row["accepts_returns"] != true {
		t.Fatalf("مَن أكّد: %v / %v / %v", row["returns_confirmed_by"], row["settlement_confirmed_by"], row["accepts_returns"])
	}

	// **وتطبيقُه يعرف أنّه اختار.**
	r := h.Call("GET", "/api/v1/merchant/stores", tok, nil, nil)
	st := r.JSON()["stores"].([]any)[0].(map[string]any)
	if st["settlement_confirmed"] != true || st["returns_confirmed"] != true || st["accepts_returns"] != true {
		t.Fatalf("متاجرُه: %v", st)
	}

	// **والتذكيرُ يصل صندوقَه.**
	if r := h.Call("POST", "/api/v1/admin/merchants/"+m.ID+"/setup-reminder", admin.Token, nil, nil); r.Code != http.StatusOK || r.JSON()["sent"] != true {
		t.Fatalf("التذكير: %d / %s", r.Code, r.Err())
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE user_id = $1 AND body LIKE '%ساعات الدوام%'`, m.Owner.ID).Scan(&n)
	if n == 0 {
		t.Fatalf("التذكيرُ لم يُحفظ في صندوقه")
	}

	// **واختيارُ اللوحة يُكتب «admin».**
	other := f.Merchant(OwnedByRep(rep.ID))
	if r := h.Call("PATCH", "/api/v1/admin/merchants/"+other.ID, admin.Token, map[string]any{"accepts_returns": false}, nil); r.Code != http.StatusOK {
		t.Fatalf("اللوحة: %d / %s", r.Code, r.Err())
	}
	row = setupRow(t, h, admin.Token, other.ID)
	if row["returns_confirmed_by"] != "admin" || row["settlement_confirmed_by"] != nil {
		t.Fatalf("اللوحة: %v / %v", row["returns_confirmed_by"], row["settlement_confirmed_by"])
	}
}
