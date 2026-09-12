package authz

// ══════════════════════════════════════════════════════════════════════
//  **أصنافُ الأدوار — سياسةُ إسنادٍ واحدةٌ يقرؤها كلُّ مسلك**
// ══════════════════════════════════════════════════════════════════════
//
// # ما قِيس (٢٠٢٦-٠٩-١٢، على التجهيز بالمسار الحقيقيّ)
//
// **سلسلةُ تصعيدٍ كاملةٌ لا تحتاج `roles.manage` إطلاقاً**:
//
//	trust_safety:  GET /admin/roles ⇒ 403   ·   POST .../roles ⇒ 403
//	               **ولا يملك `roles.manage` بحال**
//	ثمّ:  POST /admin/users {roles:["admin"], password: يختارها} ⇒ **201**
//	      **وبلا تأكيدٍ** — فالمسلكُ ليس فعلاً حسّاساً
//	ثمّ:  يدخل بالحساب المُنشَأ ⇒ GET /admin/roles ⇒ **200**
//	      **فصار يملك `roles.manage`**
//
// **فبابُ إنشاء الحساب كان بابَ منحِ أدوارٍ بقدرةٍ أخرى** —
// `users.status.manage` لا `roles.manage` — **ومن ملك الأولى ملك
// الثانية بخطوتين.**
//
// # والقاعدةُ الجامعة (قرارُ المالك ٢٠٢٦-٠٩-١٢، بندُ د)
//
//	**إنشاءُ حسابٍ لا يتجاوز سياسةَ منح الأدوار**
//
// **فمن لا يملك منحَ دورٍ لا يكسبه بأن يضعه في جسم الإنشاء.**
//
// # وأصنافٌ خمسةٌ ومجهول
//
//	account_type  صفةُ الحساب — تُنشأ بقواعد المنتج القائمة
//	staff         تخويلُ عملٍ — **يمرّ بمسار المنح المخوَّل وحدَه**
//	elevated      `admin` — **يبلغ `roles.manage` فيصير سلطةَ السلطات**
//	protected     `owner_super_admin` — **للمالك وحدَه**
//	legacy        `ops` — **قائمٌ يعمل، ولا منحَ جديد** (`OPS-5`)
//	custom        ما أنشأه الأدمنُ من اللوحة — **لا أهليّةً مرتفعةً صامتة**

// RoleClass **صنفُ الدور في السياسة** — لا في وجوده.
//
// **والوجودُ من القاعدة دائماً** (`ADG-2`): **جدولُ `roles` هو الحقيقة**،
// **وهذا تصنيفُ ما وُجد لا قائمةُ ما يوجد.**
type RoleClass string

const (
	ClassAccountType RoleClass = "account_type"
	ClassStaff       RoleClass = "staff"
	ClassElevated    RoleClass = "elevated"
	ClassProtected   RoleClass = "protected"
	ClassLegacy      RoleClass = "legacy"
	ClassCustom      RoleClass = "custom"
)

// RoleOwnerSuperAdmin **رمزُ دور المالك الأعلى** كما بُذر في `0138`.
const RoleOwnerSuperAdmin = "owner_super_admin"

// RoleAdmin **الدورُ المرتفع** — يملك `roles.manage` في المصفوفة.
const RoleAdmin = "admin"

// roleClasses **التصنيفُ الصريح.**
//
// **والمجهولُ لا يُصنَّف هنا** — يصير `custom`: **يُعرَض ويُسند بمسار
// المنح المخوَّل، ولا يكسب أهليّةَ إنشاءٍ ولا سلطةً مرتفعةً صامتة.**
var roleClasses = map[string]RoleClass{
	// **المالكُ الأعلى** (بندُ ز) — للمالك وحدَه، منحاً ونزعاً.
	RoleOwnerSuperAdmin: ClassProtected,

	// **والمرتفعُ** (بندُ و) — **يبلغ `roles.manage`**، فمنحُه للمالك.
	RoleAdmin: ClassElevated,

	// **أدوارُ العمل** (بندُ ج) — تخويلٌ يمرّ بمسار المنح المخوَّل.
	"operations":            ClassStaff,
	"finance":               ClassStaff,
	"customer_support":      ClassStaff,
	"analytics":             ClassStaff,
	"driver_verification":   ClassStaff,
	"merchant_verification": ClassStaff,
	"marketing_content":     ClassStaff,
	"trust_safety":          ClassStaff,
	"observability":         ClassStaff,

	// **وصفةُ الحساب ليست وظيفة** — والمحرّكُ يرفض أكثرَها بيدٍ
	// (`ErrMerchantNeedsStore` · `ErrRoleConflict`).
	"customer": ClassAccountType,
	"driver":   ClassAccountType,
	"merchant": ClassAccountType,
	"sales":    ClassAccountType,

	// **و`ops` إرثٌ يُقرأ ولا يُمنَح** (`OPS-5`، قرارُ المالك بندَ ج):
	// **اسمُه العربيُّ «العمليات» كاسم `operations` وقدراتُهما مختلفة**،
	// **ودمجُهما هجرةٌ بقرارٍ مستقلّ.** **فمن يحمله يبقى، ولا يُحمَّل
	// أحدٌ جديد.**
	"ops": ClassLegacy,
}

// ClassOf **صنفُ رمزٍ** — والمجهولُ `custom`.
func ClassOf(code string) RoleClass {
	if c, ok := roleClasses[code]; ok {
		return c
	}
	return ClassCustom
}

// RolesInClass **رموزُ صنفٍ** — للتقرير والحرّاس، لا للقرار.
func RolesInClass(c RoleClass) []string {
	out := []string{}
	for code, cls := range roleClasses {
		if cls == c {
			out = append(out, code)
		}
	}
	return out
}

// IsProtectedRole **أهذا دورٌ محميّ؟** — `owner_super_admin`.
func IsProtectedRole(code string) bool { return ClassOf(code) == ClassProtected }

// ProtectedRoles **القائمةُ للتقرير والحرّاس.**
func ProtectedRoles() []string { return RolesInClass(ClassProtected) }

// ══════════════════════════════════════════════════════════════════════
//  **ومن يملك المنحَ؟ — ثلاثةُ أجوبةٍ لا أكثر**
// ══════════════════════════════════════════════════════════════════════

// GrantAuthority **ما يلزم لمنح دورٍ من هذا الصنف.**
type GrantAuthority string

const (
	// GrantByRolesManage **يكفيه `roles.manage` وتأكيدٌ وقيدُ تدقيق.**
	GrantByRolesManage GrantAuthority = "roles.manage"
	// GrantByOwner **لا يمنحه إلّا من يملك دورَ المالك.**
	GrantByOwner GrantAuthority = "owner"
	// GrantNever **لا يُمنَح جديداً** — قائمُه يعمل ويُنزَع.
	GrantNever GrantAuthority = "never"
)

// AuthorityToGrant **سلطةُ منحِ رمزٍ بعينه.**
//
// **والمرتفعُ والمحميُّ للمالك** (بندَا و · ز): **و`admin` يبلغ
// `roles.manage`** — **فمن منحه منح سلطةَ السلطات، ولا يفعلها من
// يملكها هو.**
func AuthorityToGrant(code string) GrantAuthority {
	switch ClassOf(code) {
	case ClassProtected, ClassElevated:
		return GrantByOwner
	case ClassLegacy:
		return GrantNever
	default:
		return GrantByRolesManage
	}
}

// CreatableAtSignup **أيُخلَق حسابٌ بهذا الدور مباشرةً؟**
//
// **وصفةُ الحساب وحدَها** (بندُ د): **بابُ الإنشاء قدرتُه
// `users.status.manage`**، **فلو حمل دورَ عملٍ صار منحاً بقدرةٍ أخرى.**
//
// **وتخويلُ العمل يمرّ بخطوتين** (بندُ هـ): **حسابٌ يُخلَق ثمّ دورٌ
// يُمنَح بمساره المخوَّل** — بـ`roles.manage` وتأكيدٍ وقيدِ تدقيق.
func CreatableAtSignup(code string) bool {
	return ClassOf(code) == ClassAccountType
}
