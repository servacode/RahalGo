package qa

import (
	"net/http"
	"net/url"
	"testing"
)

// TestADG3_RoleRevokeWithReasonInQuery **سحبُ الدور بسببٍ في الرابط يمضي بالتأكيد** (بلاغُ
// المالك ٢٠٢٦-١٠-٢٠: «مشكلة سحب الدور») — كان الإثباتُ يُبصَم على الرابط مع «?reason=» وعند
// الاستعمال على المسار وحدَه، **فلا يطابقان أبداً** ويُردّ كلُّ سحبٍ «انتهت مهلةُ التأكيد».
func TestADG3_RoleRevokeWithReasonInQuery(t *testing.T) {
	hh := New(t)
	admin := stepUser(t, hh, "admin")
	victim := hh.NewUser("driver")
	path := "/api/v1/admin/users/" + victim.ID + "/roles/driver?reason=" + url.QueryEscape("لم يعد يعمل")

	g, code := askStepUp(t, hh, admin.Token, "DELETE", path, nil, stepPass)
	if g == "" {
		t.Fatalf("لم يصدر إثباتٌ لسحب الدور: %d", code)
	}
	if r := withGrant(hh, admin.Token, "DELETE", path, nil, g); r.Code >= 400 {
		t.Fatalf("سحبُ الدور المؤكَّد رُدّ: %s — «انتهت مهلةُ التأكيد» عائدة", r)
	}
	var has bool
	_ = hh.Pool.QueryRow(ctxBG(), `SELECT EXISTS (SELECT 1 FROM user_roles WHERE user_id = $1::uuid AND role_code = 'driver')`,
		victim.ID).Scan(&has)
	if has {
		t.Fatal("الدورُ باقٍ بعد سحبٍ مؤكَّد")
	}
	// **والإثباتُ لا يُستعمل مرّتين** — ولو تبدّل ما بعد «؟».
	if r := withGrant(hh, admin.Token, "DELETE", path+"x", nil, g); r.Code != http.StatusForbidden {
		t.Fatalf("إثباتٌ استُعمل مرّتين: %d", r.Code)
	}
}
