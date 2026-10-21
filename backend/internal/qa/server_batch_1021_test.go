package qa

import (
	"strings"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// TestQuoteReject_DriverIsNotified **رفضُ الزبون للسعر يصل السائقَ** (قرارُ المالك
// ٢٠٢٦-١٠-٢١) — كان المكتبُ وحدَه يعلم، والسائقُ ينتظر تأكيداً لن يأتي.
func TestQuoteReject_DriverIsNotified(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	id := newCustomOrder(t, h, cust)
	drv := h.NewUser("driver")
	assignDriver(t, h, id, drv.ID)
	if res := h.POST("/api/v1/driver/orders/"+id+"/agree", drv.Token, map[string]any{
		"goods_amount": 300, "fee": 100,
	}); res.Code >= 300 {
		t.Fatalf("توثيقُ السائق: %s", res)
	}
	if res := h.POST("/api/v1/orders/"+id+"/reject-quote", cust.Token, map[string]any{}); res.Code >= 300 {
		t.Fatalf("رفضُ الزبون: %s", res)
	}
	want := "رفض الزبون السعر — تواصل معه بالدردشة وعدّل السعر"
	deadline := time.Now().Add(5 * time.Second)
	for {
		var n int
		if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*)::int FROM notifications
			WHERE user_id = $1::uuid AND title = $2`, drv.ID, want).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n == 1 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("لم يصل السائقَ إشعارُ الرفض (عدد=%d)", n)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// TestSearch_EmptyStoreIsHidden **متجرٌ بلا صنفٍ لا يظهر في البحث باسمه**
// (قرارُ المالك ٢٠٢٦-١٠-٢١) — والمتجرُ نفسُه يظهر حين يُضاف له صنف.
func TestSearch_EmptyStoreIsHidden(t *testing.T) {
	h := New(t)
	m := h.Factory().Merchant()
	name := uniq("متجرفارغ")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET name = $2 WHERE id = $1::uuid`, m.ID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM menu_items WHERE merchant_id = $1::uuid`, m.ID); err != nil {
		t.Fatal(err)
	}
	cust := h.Customer()
	if res := h.GET("/api/v1/public/search?q="+name, cust.Token); strings.Contains(string(res.Body), m.ID) {
		t.Fatalf("**متجرٌ بلا أصنافٍ ظهر في البحث**: %s", res)
	}
	h.NewItemFor(m, 100)
	if res := h.GET("/api/v1/public/search?q="+name, cust.Token); !strings.Contains(string(res.Body), m.ID) {
		t.Fatalf("متجرٌ له صنفٌ لم يظهر في البحث باسمه: %s", res)
	}
}

// TestAdmin_TwoWebSessionsInEveryEnv **حسابُ الأدمن على جهازين** (قرارُ المالك
// ٢٠٢٦-١٠-٢١) — الدخولُ الثاني من المتصفّح لا يُخرج الأوّل، والثالثُ يُخرج الأقدم.
func TestAdmin_TwoWebSessionsInEveryEnv(t *testing.T) {
	h := New(t)
	phone := aprUser(t, h, "admin")
	// **ورمزُ الأدمن مضبوطٌ** — الدخولُ كلمةٌ ثمّ رمزٌ بالتحدّي.
	pinHash, _ := auth.HashPassword("739164")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE users SET admin_pin_hash = $2, admin_pin_set_at = now()
		WHERE phone = $1`, phone, pinHash); err != nil {
		t.Fatalf("رمزُ الأدمن: %v", err)
	}
	web := map[string]string{"X-RahalGo-Client": "web"}
	login := func() string {
		res := h.Call("POST", "/api/v1/auth/login", "", map[string]any{"phone": phone, "password": aprPassword}, web)
		if res.Code != 200 {
			t.Fatalf("دخولُ الأدمن: %s", res)
		}
		ch, _ := res.JSON()["challenge"].(string)
		if ch == "" {
			t.Fatalf("لا تحدّيَ لرمز الأدمن: %s", res)
		}
		res = h.Call("POST", "/api/v1/auth/pin", "", map[string]any{"challenge": ch, "pin": "739164"}, web)
		if res.Code != 200 {
			t.Fatalf("رمزُ الأدمن: %s", res)
		}
		tok, _ := res.JSON()["tokens"].(map[string]any)
		a, _ := tok["access_token"].(string)
		if a == "" {
			t.Fatalf("دخولُ الأدمن بلا رمز: %s", res)
		}
		return a
	}
	first := login()
	second := login()
	if res := h.GET("/api/v1/auth/me", first); res.Code != 200 {
		t.Fatalf("**الجهازُ الثاني أخرج الأوّل**: %s", res)
	}
	if res := h.GET("/api/v1/auth/me", second); res.Code != 200 {
		t.Fatalf("الجهازُ الثاني: %s", res)
	}
	login()
	if res := h.GET("/api/v1/auth/me", first); res.Code != 401 {
		t.Fatalf("الدخولُ الثالث يجب أن يُخرج الأقدم: %s", res)
	}
}
