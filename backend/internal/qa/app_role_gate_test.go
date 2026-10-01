package qa

import (
	"testing"

	"github.com/servacode/rahalgo/backend/internal/auth"
)

// ══════════════════════════════════════════════════════════════════════
// **كلُّ تطبيقِ دورٍ لا يفتح جلسةً إلّا لصاحب صفته** — `APR`
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-١٠-٠١: «تمّ تسجيل الدخول بحساب المتجر بالرغم انه
//  مندوب… المفروض مايدخل يقول هذا الحساب ليس حساب متجر».)
//
// **قِيس على التجهيز قبل الإصلاح**: اثنتا عشرةَ تركيبةً (أربعةُ حساباتٍ ×
// ثلاثةِ تطبيقات) — **كلُّها دخلت بـ٢٠٠**، والرفضُ يأتي بعدها من كلّ شاشة.

const aprPassword = "Qa!AppRole-2026"

func aprUser(t *testing.T, h *Harness, role string) string {
	t.Helper()
	u := h.NewUser(role)
	admin := h.NewUser("admin")
	if set := h.POST("/api/v1/admin/users/"+u.ID+"/password", admin.Token,
		map[string]any{"password": aprPassword}); set.Code >= 400 {
		t.Fatalf("وضعُ الكلمة: %s", set)
	}
	// **وكلمةُ الإدارة تُلزم بالتغيير** — وليس ذاك ما يُقاس هنا.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET must_change_password = false WHERE id = $1::uuid`, u.ID); err != nil {
		t.Fatalf("رفعُ إلزام التغيير: %v", err)
	}
	return u.Phone
}

func aprLogin(h *Harness, phone, app string) Res {
	return h.Call("POST", "/api/v1/auth/login", "", map[string]any{
		"phone": phone, "password": aprPassword,
	}, map[string]string{"X-RahalGo-Client": "android-" + app})
}

// APR01 — **المصفوفةُ كاملة**: الصفةُ تفتح تطبيقَها وحدَه، والزبونُ للجميع.
func TestAPR01_EachAppAdmitsOnlyItsRole(t *testing.T) {
	h := New(t)
	phones := map[string]string{
		"sales":    aprUser(t, h, "sales"),
		"driver":   aprUser(t, h, "driver"),
		"merchant": aprUser(t, h, "merchant"),
		"customer": aprUser(t, h, "customer"),
	}
	apps := map[string]struct{ role, refusal string }{
		"merchant": {"merchant", "not_merchant_account"},
		"rep":      {"sales", "not_rep_account"},
		"driver":   {"driver", "not_driver_account"},
		"customer": {"", ""},
	}
	for role, phone := range phones {
		for app, want := range apps {
			got := aprLogin(h, phone, app)
			if want.role == "" || want.role == role {
				if got.Code != 200 {
					t.Errorf("%s في تطبيق %s: يجب أن يدخل — %s", role, app, got)
				}
				continue
			}
			if got.Code != 403 || got.Err() != want.refusal {
				t.Errorf("%s في تطبيق %s: يجب الرفضُ بـ%s — %s", role, app, want.refusal, got)
			}
		}
	}
}

// APR02 — **وجلسةٌ فُتحت قبل القفل لا تُجدَّد**: فيخرج التطبيقُ برسالته.
func TestAPR02_OldWrongAppSessionDoesNotRefresh(t *testing.T) {
	h := New(t)
	rep := h.NewUser("sales")
	raw, hash, err := auth.NewOpaqueToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'android-merchant')`,
		rep.ID, hash); err != nil {
		t.Fatalf("جلسةٌ قديمة: %v", err)
	}
	got := h.Call("POST", "/api/v1/auth/refresh", "", map[string]any{"refresh_token": raw},
		map[string]string{"X-RahalGo-Client": "android-merchant"})
	if got.Code != 403 || got.Err() != "not_merchant_account" {
		t.Fatalf("تجديدُ جلسةِ مندوبٍ في تطبيق المتجر يجب أن يُرفض: %s", got)
	}
}

// APR03 — **ودخولٌ مرفوضٌ لا يزيح جلسةً قائمة**: صاحبُ الصفتين لا يُمسّ،
// **وصاحبُ الصفة الواحدة لا جلسةَ له في التطبيق الآخر أصلاً.** فالمقيسُ هنا
// أنّ الرفضَ يقع قبل الإبطال: جلسةُ المندوب في تطبيقه تبقى حيّةً بعد محاولته
// دخولَ تطبيق المتجر.
func TestAPR03_RefusedLoginKeepsOwnAppSession(t *testing.T) {
	h := New(t)
	phone := aprUser(t, h, "sales")
	in := aprLogin(h, phone, "rep")
	if in.Code != 200 {
		t.Fatalf("دخولُ المندوب تطبيقَه: %s", in)
	}
	access := tokenField(in, "access_token")
	if got := aprLogin(h, phone, "merchant"); got.Code != 403 {
		t.Fatalf("يجب الرفض: %s", got)
	}
	if got := h.Call("GET", "/api/v1/rep/merchants", access, nil,
		map[string]string{"X-RahalGo-Client": "android-rep"}); got.Code != 200 {
		t.Fatalf("جلسةُ المندوب في تطبيقه سقطت بعد محاولةٍ مرفوضة: %s", got)
	}
}
