package qa

// **الطبقةُ الأولى — التخويل: من يدخل ومن يُردّ.**
//
// المعرّفات: `AUTH-*` · الوسم: `@api @security @critical @release`
//
// # ما تقيسه
//
// **كلُّ بابٍ خاصٍّ يُردّ ٤٠١ بلا توكن**، وبتوكنٍ مزوَّر، وبتوكنٍ منتهٍ.
// **والدورُ يُفرَض**: زبونٌ لا يفتح بابَ سائقٍ ولا بابَ إدارة.
//
// # وما لا تقيسه
//
// **لا تقيس تجديدَ التوكن ولا دورةَ الرمز على واتساب** — تلك في
// `SESSION-*` و`OTP-*`. **وحارسٌ يقيس شيئين يسقط فلا يُعرف أيُّهما.**

import (
	"net/http"
	"testing"
)

// privateRoutes **أبوابٌ لا تُفتح بلا توكن** — والقائمةُ صريحةٌ لا
// مُخمَّنةٌ من الموجّه: **بابٌ عامٌّ يدخل القائمةَ يُنذر على الصواب.**
var privateRoutes = []struct{ ID, Method, Path string }{
	{"AUTH-001", "GET", "/api/v1/auth/me"},
	{"AUTH-002", "GET", "/api/v1/my/orders"},
	{"AUTH-003", "GET", "/api/v1/my/wallet"},
	{"AUTH-004", "GET", "/api/v1/my/addresses"},
	{"AUTH-005", "GET", "/api/v1/my/favorites"},
	{"AUTH-006", "GET", "/api/v1/me/summary"},
	{"AUTH-007", "GET", "/api/v1/my/ratings"},
	{"AUTH-008", "POST", "/api/v1/orders"},
	{"AUTH-009", "POST", "/api/v1/orders/custom"},
	{"AUTH-010", "GET", "/api/v1/me/notifications"},
}

// TestAUTH_NoToken **بلا توكنٍ لا شيء.**
func TestAUTH_NoToken(t *testing.T) {
	h := New(t)
	for _, r := range privateRoutes {
		t.Run(r.ID, func(t *testing.T) {
			got := h.Call(r.Method, r.Path, "", nil, nil)
			if got.Code != http.StatusUnauthorized {
				t.Errorf("%s %s بلا توكن: يُنتظر 401 ووقع %s", r.Method, r.Path, got)
			}
		})
	}
}

// TestAUTH_GarbageToken **وتوكنٌ مزوَّرٌ كلا توكن.**
func TestAUTH_GarbageToken(t *testing.T) {
	h := New(t)
	for _, tok := range []string{
		"not-a-token",
		"eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJhIn0.zzzz", // توقيعٌ خاطئ
		"Bearer Bearer",
	} {
		got := h.GET("/api/v1/auth/me", tok)
		if got.Code != http.StatusUnauthorized {
			t.Errorf("توكنٌ مزوَّرٌ %q: يُنتظر 401 ووقع %s", tok, got)
		}
	}
}

// TestAUTH_ExpiredToken **والمنتهي مرفوض.**
//
// **وهذا ما لا يقيسه اختبارُ توقيعٍ صحيح**: التوقيعُ سليمٌ والمهلةُ
// مضت — **ومن لا يفحص `exp` يقبل توكنَ العامِ الماضي.**
func TestAUTH_ExpiredToken(t *testing.T) {
	h := New(t)
	u := h.Customer()
	got := h.GET("/api/v1/auth/me", h.ExpiredToken(u.ID, "customer"))
	if got.Code != http.StatusUnauthorized {
		t.Errorf("AUTH-011 توكنٌ منتهٍ: يُنتظر 401 ووقع %s", got)
	}
}

// TestAUTH_ValidToken **والصحيحُ يمرّ** — وإلّا كان ما فوقه بلا معنى.
//
// **وحارسٌ يردّ الجميعَ ينجح في كلّ ما سبق** — فيُقاس القبولُ كما يُقاس
// الردّ.
func TestAUTH_ValidToken(t *testing.T) {
	h := New(t)
	u := h.Customer()
	got := h.GET("/api/v1/auth/me", u.Token)
	if got.Code != http.StatusOK {
		t.Fatalf("AUTH-012 توكنٌ صحيح: يُنتظر 200 ووقع %s", got)
	}
	if id, _ := got.JSON()["id"].(string); id != u.ID {
		t.Errorf("AUTH-012 /auth/me أعاد حساباً آخر: %v ≠ %s", got.JSON()["id"], u.ID)
	}
}

// roleWalls **أبوابٌ لدورٍ لا يفتحها غيرُه.**
var roleWalls = []struct{ ID, Method, Path, Need string }{
	{"AUTH-020", "GET", "/api/v1/driver/queue", "driver"},
	{"AUTH-021", "GET", "/api/v1/driver/orders", "driver"},
	{"AUTH-022", "GET", "/api/v1/rep/customers", "rep"},
	{"AUTH-023", "GET", "/api/v1/admin/settings", "admin"},
	{"AUTH-024", "GET", "/api/v1/admin/users", "admin"},
	{"AUTH-025", "GET", "/api/v1/admin/banners", "admin"},
	{"AUTH-026", "GET", "/api/v1/merchant/orders", "merchant"},
}

// TestAUTH_RoleEnforced **والزبونُ لا يفتح بابَ غيرِه.**
func TestAUTH_RoleEnforced(t *testing.T) {
	h := New(t)
	cust := h.Customer()
	for _, r := range roleWalls {
		t.Run(r.ID, func(t *testing.T) {
			got := h.Call(r.Method, r.Path, cust.Token, nil, nil)
			if got.Code != http.StatusUnauthorized && got.Code != http.StatusForbidden {
				t.Errorf("%s بدور customer حيث يلزم %s: يُنتظر 401/403 ووقع %s",
					r.Path, r.Need, got)
			}
		})
	}
}

// TestAUTH_ForgedRoleInToken **ودورٌ يُكتب في توكنٍ موقَّعٍ بسرٍّ آخر
// لا يمرّ.**
//
// **وهذا أخطرُ من توكنٍ مزوَّرٍ كلِّه**: من عرف بنيةَ الحمولة يكتب
// `roles: ["admin"]` — **والتوقيعُ وحدَه هو ما يمنعه.**
func TestAUTH_ForgedRoleInToken(t *testing.T) {
	h := New(t)
	u := h.Customer()
	// **موقَّعٌ بسرٍّ غيرِ سرِّ الخادم** — أقربُ ما يملكه مهاجم.
	forged := signWith("another-secret-entirely", u.ID, []string{"admin"})
	got := h.GET("/api/v1/admin/settings", forged)
	if got.Code != http.StatusUnauthorized && got.Code != http.StatusForbidden {
		t.Errorf("AUTH-030 توكنٌ موقَّعٌ بسرٍّ آخرَ يحمل دورَ admin: "+
			"يُنتظر 401/403 ووقع %s", got)
	}
}
