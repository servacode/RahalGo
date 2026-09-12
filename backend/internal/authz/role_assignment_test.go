package authz

// ══════════════════════════════════════════════════════════════════════
//  **مسارُ إسناد الأدوار — ضماناتُه لا تُنزع بإصلاحِ واجهة**
// ══════════════════════════════════════════════════════════════════════
//
// # ولماذا هنا
//
// **دورةُ ٢٠٢٦-٠٩-١٢ أصلحت الواجهةَ**: نافذةُ أدوارِ الحساب كانت تقصّ
// ثمانيةَ عشرَ دوراً في ٣٩٨ بكسل **فلم يجد المالكُ `observability`**،
// **وكانت تعرض `owner_super_admin` حبّةً كبقيّتها.**
//
// **والإصلاحُ في الواجهة لا يمسّ التخويل** — **وهذا هو ما يجب أن
// يبقى مقيساً**: **من حرس بالواجهة وحدَها حرس بابَ بيتٍ بستارة.**
//
// # وما يُحرَس
//
//	١ · **منحُ دورٍ ونزعُه يطلبان `roles.manage`** — لا اسمَ دورٍ
//	٢ · **وكلٌّ منهما فعلٌ حسّاسٌ يطلب تأكيداً** بكلمة المرور
//	٣ · **وإنشاءُ الدور كذلك** — وهو الفعلُ الذي مشاه المالكُ
//
// **ونزعُ صفٍّ من `sensitiveActions` يُسقط هذا** — **وإخفاءُ حبّةٍ في
// الواجهة لا يعوّضه.**

import "testing"

// TestRoleAssignmentNeedsRolesManage **والسلطةُ قدرةٌ لا اسمُ دور.**
func TestRoleAssignmentNeedsRolesManage(t *testing.T) {
	for _, c := range []struct {
		method string
		path   string
	}{
		{"POST", "/users/11111111-1111-1111-1111-111111111111/roles"},
		{"DELETE", "/users/11111111-1111-1111-1111-111111111111/roles/observability"},
		{"POST", "/roles"},
		{"GET", "/roles"},
	} {
		// **والنمطُ يُشتقّ من مسارٍ حقيقيٍّ** — لا يُكتب هنا بيدٍ،
		// **فلو تبدّل شكلُ الاشتقاق ظهر هنا.**
		pattern := AdminPattern(c.path)
		got, ok := LookupAdmin(c.method, pattern)
		if !ok {
			t.Errorf("**مسارُ أدوارٍ بلا صفٍّ في جدول السياسة**: %s %s — "+
				"**فيمرّ بحارسٍ عامٍّ لا بقدرةٍ مسمّاة**", c.method, c.path)
			continue
		}
		if got != RolesManage {
			t.Errorf("%s %s يطلب %q والمنتظَرُ %q — "+
				"**وإسنادُ الأدوار سلطةُ السلطات**", c.method, c.path, got, RolesManage)
		}
	}
}

// TestRoleAssignmentIsSensitive **وكلُّ إسنادٍ يطلب تأكيداً بكلمة المرور.**
//
// **وهذا الشرطُ هو الذي كشف عطبَي ٢٠٢٦-٠٩-١٢** (ظرفُ التحدّي، ثمّ
// ترويسةُ `X-Step-Up`) — **فنزعُ صفٍّ منه يُخفي التدفّقَ كلَّه.**
func TestRoleAssignmentIsSensitive(t *testing.T) {
	want := map[string]string{
		"POST /users/{id}/roles":          "admin.role_grant",
		"DELETE /users/{id}/roles/{role}": "admin.role_revoke",
		"POST /roles":                     "admin.role_create",
	}
	seen := map[string]string{}
	for _, a := range SensitiveActions() {
		seen[a.Method+" "+a.Pattern] = a.Action
	}
	for k, action := range want {
		got, ok := seen[k]
		if !ok {
			t.Errorf("**فعلُ إسنادٍ بلا تأكيد**: %s — "+
				"**فدورٌ يُمنح بنقرةٍ واحدةٍ بلا كلمةِ مرور**", k)
			continue
		}
		if got != action {
			t.Errorf("%s اسمُه %q والمنتظَرُ %q — **واسمُ الفعل هو ما يُقرأ في سجلّ التدقيق**", k, got, action)
		}
	}
}
