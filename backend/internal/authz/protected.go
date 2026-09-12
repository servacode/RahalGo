package authz

// ══════════════════════════════════════════════════════════════════════
//  **الأدوارُ المحميّة — ما لا يكفي له `roles.manage`**
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس (٢٠٢٦-٠٩-١٢، على التجهيز بالمسار الحقيقيّ)
//
// **أدمنٌ يملك `roles.manage` ولا يملك دورَ المالك**:
//
//	POST /admin/users/{آخر}/roles  role=owner_super_admin  ⇒ **200 granted**
//	POST /admin/users/{نفسه}/roles role=owner_super_admin  ⇒ **200 granted**
//	DELETE .../roles/owner_super_admin                     ⇒ **200 revoked**
//
// **فإخفاءُ الحبّةِ في اللوحة لم يكن حدّاً** — **والحدُّ يُكتب هنا.**
//
// # ولا مفهومَ مالكٍ ثانٍ يُخترَع
//
// **الدورُ نفسُه هو الهويّة**: `owner_super_admin` — **وهو الاسمُ
// الكانونيُّ المبذورُ في هجرة `0138`**، **وهو وحدَه يملك `roles.manage`
// في المصفوفة الكانونيّة** بقرار المالك المكتوب فيها. **فلا قدرةَ
// جديدةٌ ولا جدولَ جديدٌ ولا علَمَ «مالكٍ» موازٍ.**
//
// # ولماذا الحقيقةُ هنا لا في خدمة الهويّة
//
// **ثلاثةُ مسالكَ تكتب في `user_roles`** — منحٌ ونزعٌ وإنشاءُ حساب —
// **ومن كتب الشرطَ في واحدٍ منها ترك البابين.** **فالحقيقةُ في مكانٍ
// واحدٍ يقرؤه الثلاثةُ وحارسُهم.**

// RoleOwnerSuperAdmin **رمزُ دور المالك الأعلى** كما بُذر في `0138`.
const RoleOwnerSuperAdmin = "owner_super_admin"

// protectedRoles **ما لا يُمنَح ولا يُنزَع بـ`roles.manage` وحدَها.**
//
// **وواحدٌ اليومَ بقصد** — **وقائمةٌ تكبر بلا قرارٍ تكبر بلا علمِ أحد.**
var protectedRoles = map[string]bool{
	RoleOwnerSuperAdmin: true,
}

// IsProtectedRole **أهذا دورٌ محميّ؟**
func IsProtectedRole(code string) bool { return protectedRoles[code] }

// ProtectedRoles **القائمةُ للتقرير والحرّاس** — لا للقرار.
func ProtectedRoles() []string {
	out := make([]string, 0, len(protectedRoles))
	for code := range protectedRoles {
		out = append(out, code)
	}
	return out
}
