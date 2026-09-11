package qa

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/authz"
)

// ══════════════════════════════════════════════════════════════════════
// **إنشاءُ دورٍ من اللوحة — الفعلُ الذي كان ناقصاً** (دورةُ ٧٠ب-و١)
// ══════════════════════════════════════════════════════════════════════
//
// **وجدولُ السياسة يحجز `/roles` لأيّ فعلٍ منذ `ADG-2`، والمُوجِّهُ لم
// يسجّل إلّا القراءة** — **فبابٌ محجوزٌ لم يُفتَح.**
//
// **وأثرُه عمليّ**: **قدرةٌ تُضاف في الشيفرة لا تجد دوراً ضيّقاً
// يحملها**، **فتُمنَح لدورٍ عريضٍ أو يُكتب صفٌّ بيدٍ في القاعدة** —
// وكلاهما نقضٌ لعقد «لا تعديلَ يدويٌّ للتشغيل».
//
// **ولا توازيَ في هذه الحزمة** — `XG-41C`.

const rolesPath = "/api/v1/admin/roles"

func rolesOf(t *testing.T, hh *Harness, tok string) []map[string]any {
	t.Helper()
	r := hh.GET(rolesPath, tok)
	if r.Code != http.StatusOK {
		t.Fatalf("**قراءةُ الأدوار** — %d: %s", r.Code, r.Body)
	}
	var env struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(r.Body, &env); err != nil {
		t.Fatalf("**ليس JSON**: %v", err)
	}
	return env.Data
}

// TestU1_A1_CreateRoleNeedsRolesManage **البابُ بقدرته لا بدورٍ عريض.**
func TestU1_A1_CreateRoleNeedsRolesManage(t *testing.T) {
	hh := New(t)

	if r := hh.POST(rolesPath, "", map[string]any{
		"code": "u1_anon", "name": "بلا جلسة",
	}); r.Code != http.StatusUnauthorized {
		t.Fatalf("**بلا جلسةٍ يجب ٤٠١** — %d", r.Code)
	}

	// **قدرةٌ إداريّةٌ أخرى لا تفتحه** — وهي أقربُ ما يُخلَط به.
	capRole(t, hh, "u1_weak", authz.UsersRead, authz.OrdersRead)
	_, weak := capUser(t, hh, "u1_weak")
	if r := hh.POST(rolesPath, weak, map[string]any{
		"code": "u1_weak_try", "name": "محاولة",
	}); r.Code != http.StatusForbidden {
		t.Fatalf("**قدرةٌ أخرى لا تُنشئ دوراً** — %d: %s", r.Code, r.Body)
	}

	// **ولا يُنشَأ شيءٌ حين يُمنَع** — **ومنعٌ يترك أثراً ليس منعاً.**
	capRole(t, hh, "u1_mgr", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr")
	for _, r := range rolesOf(t, hh, mgr) {
		if r["code"] == "u1_weak_try" {
			t.Fatal("**الدورُ أُنشئ رغم المنع**")
		}
	}
	t.Logf("A1: بلا جلسة=٤٠١ · بقدرةٍ أخرى=٤٠٣ · ولا أثرَ للممنوع")
}

// TestU1_A2_CreateRoleAppearsInBackendList **ويظهر في مصدر الحقيقة.**
//
// **وهو عقدُ الواجهة**: **دورٌ يُنشَأ يظهر بلا بناءِ واجهةٍ جديد** —
// **فالقائمةُ تُقرأ من هنا لا من معجمِ نصوصٍ في العميل.**
func TestU1_A2_CreateRoleAppearsInBackendList(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "u1_mgr2", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr2")

	before := len(rolesOf(t, hh, mgr))
	r := hh.POST(rolesPath, mgr, map[string]any{
		"code": "u1_observability", "name": "مراقبة التشغيل",
	})
	if r.Code != http.StatusCreated {
		t.Fatalf("**الإنشاءُ يجب ٢٠١** — %d: %s", r.Code, r.Body)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = 'u1_observability'`)
	})

	after := rolesOf(t, hh, mgr)
	if len(after) != before+1 {
		t.Fatalf("**القائمةُ لم تنمُ** — %d ← %d", before, len(after))
	}
	var found map[string]any
	for _, x := range after {
		if x["code"] == "u1_observability" {
			found = x
		}
	}
	if found == nil {
		t.Fatal("**الدورُ الجديدُ غائبٌ عن قائمة المحرّك**")
	}
	// **ودورٌ يُنشَأ اليومَ لا يملك شيئاً** — `ADG-1`.
	caps, _ := found["capabilities"].([]any)
	if len(caps) != 0 {
		t.Errorf("**دورٌ جديدٌ وُلد بقدرات** — %v", caps)
	}
	// **واسمُه نصٌّ يُقرأ لا مفتاحُ ترجمة.**
	if found["name_key"] != "مراقبة التشغيل" {
		t.Errorf("**الاسمُ لم يُحفَظ كما كُتب** — %v", found["name_key"])
	}
	t.Logf("A2: %d ← %d دوراً · الجديدُ بلا قدرةٍ واسمُه نصّ", before, len(after))
}

// TestU1_A3_DuplicateCodeIsRejected **ولا يُكتَب فوق دورٍ قائم.**
//
// **وإنشاءٌ صامتٌ فوق موجودٍ يمنح قدراتِ غيرِه لمن أنشأه** — **وهو
// تصعيدُ امتيازٍ بلا سطرِ شيفرةٍ خبيث.**
func TestU1_A3_DuplicateCodeIsRejected(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "u1_mgr3", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr3")

	// **و`admin` قائمٌ ويملك خمساً وعشرين قدرة.**
	r := hh.POST(rolesPath, mgr, map[string]any{
		"code": "admin", "name": "محاولةُ اختطاف",
	})
	if r.Code != http.StatusConflict {
		t.Fatalf("**الرمزُ المأخوذُ يجب ٤٠٩** — %d: %s", r.Code, r.Body)
	}
	// **ولم يُبدَّل اسمُه.**
	for _, x := range rolesOf(t, hh, mgr) {
		if x["code"] == "admin" && x["name_key"] == "محاولةُ اختطاف" {
			t.Fatal("**اسمُ دورٍ قائمٍ أُعيدت كتابتُه**")
		}
	}
	t.Logf("A3: ٤٠٩ · ودورُ admin لم يُمَسّ")
}

// TestU1_A4_RoleCodeIsNarrow **ورمزُ الدور يدخل كلَّ جدولٍ وسجلّ.**
func TestU1_A4_RoleCodeIsNarrow(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "u1_mgr4", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr4")

	for _, bad := range []map[string]any{
		{"code": "Bad_Upper", "name": "كبيرة"},
		{"code": "has space", "name": "فراغ"},
		{"code": "ar", "name": "قصير"},
		{"code": "9starts", "name": "يبدأ برقم"},
		{"code": "good_code", "name": "   "},
		{"code": "", "name": "بلا رمز"},
	} {
		if r := hh.POST(rolesPath, mgr, bad); r.Code != http.StatusBadRequest {
			t.Errorf("**قُبل رمزٌ لا يصلح** %q ⇒ %d", bad["code"], r.Code)
		}
	}
	// **والجدولُ سليم.**
	if len(rolesOf(t, hh, mgr)) == 0 {
		t.Fatal("**جدولُ الأدوار تضرّر**")
	}
	t.Logf("A4: ستُّ مدخلاتٍ رُدّت · والجدولُ سليم")
}

// TestU1_A5_NewRoleIsAssignableToUser **والدائرةُ تُغلَق من اللوحة.**
//
// **وهو ما كانت تعجز عنه الواجهة**: **دورٌ يُنشَأ لا يظهر في مُختار
// الأدوار لأنّه كان مثبَّتاً في العميل.**
func TestU1_A5_NewRoleIsAssignableToUser(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "u1_mgr5", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr5")

	if r := hh.POST(rolesPath, mgr, map[string]any{
		"code": "u1_watcher", "name": "مراقب",
	}); r.Code != http.StatusCreated {
		t.Fatalf("**الإنشاء** — %d: %s", r.Code, r.Body)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM user_roles WHERE role_code = 'u1_watcher'`)
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM role_capabilities WHERE role_code = 'u1_watcher'`)
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = 'u1_watcher'`)
	})

	// **وتُمنَح قدرةٌ واحدةٌ لا غير.**
	if r := hh.POST(rolesPath+"/u1_watcher/capabilities", mgr, map[string]any{
		"capability": string(authz.ObservabilityRead),
	}); r.Code != http.StatusOK {
		t.Fatalf("**منحُ القدرة** — %d: %s", r.Code, r.Body)
	}

	// **ثمّ يُسنَد الدورُ لحساب.**
	u := hh.NewUser("customer")
	if r := hh.POST("/api/v1/admin/users/"+u.ID+"/roles", mgr, map[string]any{
		"role": "u1_watcher", "reason": "٧٠ب-و١",
	}); r.Code != http.StatusOK {
		t.Fatalf("**إسنادُ الدور** — %d: %s", r.Code, r.Body)
	}

	seen := false
	for _, x := range rolesOf(t, hh, mgr) {
		if x["code"] != "u1_watcher" {
			continue
		}
		seen = true
		caps, _ := x["capabilities"].([]any)
		if len(caps) != 1 || caps[0] != string(authz.ObservabilityRead) {
			t.Errorf("**القدراتُ ليست واحدةً بعينها** — %v", caps)
		}
		if n, _ := x["members"].(float64); n != 1 {
			t.Errorf("**عددُ الأصحاب ليس واحداً** — %v", x["members"])
		}
	}
	if !seen {
		t.Fatal("**الدورُ غائبٌ عن القائمة بعد الإسناد**")
	}
	t.Logf("A5: أُنشئ · مُنح قدرةً واحدةً · أُسنِد — والدائرةُ مُغلَقة")
}

// TestU1_A6_CreateRoleIsAudited **ولا تبديلَ سياسةٍ بلا أثر.**
func TestU1_A6_CreateRoleIsAudited(t *testing.T) {
	hh := New(t)
	capRole(t, hh, "u1_mgr6", authz.RolesManage)
	_, mgr := capUser(t, hh, "u1_mgr6")

	if r := hh.POST(rolesPath, mgr, map[string]any{
		"code": "u1_audited", "name": "مُقيَّد",
	}); r.Code != http.StatusCreated {
		t.Fatalf("**الإنشاء** — %d: %s", r.Code, r.Body)
	}
	t.Cleanup(func() {
		_, _ = hh.Pool.Exec(ctxBG(), `DELETE FROM roles WHERE code = 'u1_audited'`)
	})

	var n int
	if err := hh.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM audit_log
		 WHERE action = 'admin.role_create'
		   AND entity = 'role' AND entity_id = 'u1_audited'`).Scan(&n); err != nil {
		t.Fatalf("قراءةُ السجلّ: %v", err)
	}
	if n != 1 {
		t.Fatalf("**صفُّ تدقيقٍ واحدٌ منتظَر** — %d", n)
	}
	t.Logf("A6: أثرٌ واحدٌ في سجلّ التدقيق")
}
