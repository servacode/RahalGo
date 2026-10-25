package qa

// **البحثُ العامُّ في اللوحة** — تصميمُ اللوحة ٢٠٢٦-١٠-٢٥.
//
// خانةٌ أعلى كلّ صفحة تجد المتجرَ باسمه، والحسابَ بهاتفه، والطلبَ برقمه —
// **وكلُّ قسمٍ بقدرته**: من لا يملك اللوحةَ أصلاً يُردّ.

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestAdminSearch_FindsStoreAndAccount(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	store := f.Merchant()
	var name string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT name FROM merchants WHERE id = $1`, store.ID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	cust := h.Customer()

	// ── المتجرُ باسمه ─────────────────────────────────────────────────
	r := h.GET("/api/v1/admin/search?q="+url.QueryEscape(name), admin.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("البحثُ باسم المتجر: %d / %s", r.Code, r.Err())
	}
	ms, _ := r.JSON()["merchants"].([]any)
	found := false
	for _, x := range ms {
		if m, _ := x.(map[string]any); m["id"] == store.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("المتجرُ %q لم يُوجد باسمه: %s", name, r)
	}
	if _, ok := r.JSON()["orders"]; !ok {
		t.Fatalf("قسمُ الطلبات غائبٌ عن الأدمن: %s", r)
	}

	// ── والحسابُ بهاتفه ───────────────────────────────────────────────
	digits := strings.TrimPrefix(cust.Phone, "+")
	r = h.GET("/api/v1/admin/search?q="+url.QueryEscape(digits[len(digits)-7:]), admin.Token)
	us, _ := r.JSON()["users"].([]any)
	found = false
	for _, x := range us {
		if u, _ := x.(map[string]any); u["id"] == cust.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("الزبونُ لم يُوجد بهاتفه: %s", r)
	}

	// ── وحرفٌ واحدٌ لا يمسح جدولاً ─────────────────────────────────────
	r = h.GET("/api/v1/admin/search?q="+url.QueryEscape("ا"), admin.Token)
	if r.Code != http.StatusOK || len(r.JSON()) != 0 {
		t.Fatalf("حرفٌ واحد: %d / %s — والمنتظَر ردٌّ فارغ", r.Code, r)
	}

	// ── ومن لا يملك اللوحةَ يُردّ ──────────────────────────────────────
	if got := h.GET("/api/v1/admin/search?q="+url.QueryEscape(name), cust.Token).Code; got != http.StatusForbidden {
		t.Fatalf("الزبونُ بلغ البحثَ الإداريّ: %d — والمنتظَر 403", got)
	}
}
