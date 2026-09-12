package identity

// ══════════════════════════════════════════════════════════════════════
//  **حدُّ إسناد الأدوار — في المحرّك لا في اللوحة**
// ══════════════════════════════════════════════════════════════════════
//
// # ثغرتان مقيستان (٢٠٢٦-٠٩-١٢، بالمسار الحقيقيّ على التجهيز)
//
// **الأولى**: أدمنٌ يملك `roles.manage` ولا يملك دورَ المالك ⇒
//
//	منحُه لحسابٍ آخر ⇒ **200 granted**
//	منحُه لنفسِه     ⇒ **200 granted**   ← ترقيةٌ ذاتيّةٌ كاملة
//	نزعُه            ⇒ **200 revoked**
//
// **وحارسُ الدور الواحد لم يمنع الترقيةَ الذاتيّة** — **فهو للميدان
// وحدَه**، **وأدوارُ المكتب تتّحد بقرار ٢٠٢٦-٠٩-٠٧.**
//
// **والثانية أوسع** — **ولا تحتاج `roles.manage` إطلاقاً**:
//
//	trust_safety:  GET /admin/roles ⇒ 403  ·  POST .../roles ⇒ 403
//	ثمّ POST /admin/users {roles:["admin"], password: يختارها} ⇒ **201**
//	ثمّ يدخل به ⇒ GET /admin/roles ⇒ **200**
//
// **فبابُ الإنشاء كان بابَ منحٍ بقدرةٍ أخرى** — `users.status.manage`.
//
// # والقواعدُ الثلاث (قرارُ المالك ٢٠٢٦-٠٩-١٢)
//
//	١ · **إنشاءُ حسابٍ لا يتجاوز سياسةَ المنح** (بندُ د)
//	    **فلا يُخلَق حسابٌ إلّا بصفةِ حساب**، وتخويلُ العمل بخطوةٍ
//	    ثانيةٍ مخوَّلة (بندُ هـ)
//	٢ · **والمرتفعُ `admin` للمالك وحدَه** (بندُ و) — **يبلغ
//	    `roles.manage` فمنحُه منحُ سلطةِ السلطات**
//	٣ · **والمحميُّ `owner_super_admin` للمالك** ولا يُنزَع آخرُه
//	    (بندُ ز)
//
// # وموضعُ الفحص — داخلَ المعاملة قبل الكتابة
//
// **وفحصٌ خارجَ المعاملة يسابقها**: نزعان متزامنان يقرأ كلٌّ منهما
// «اثنان» فينزعان معاً ⇒ **صفرُ مالكين.** **فالصفوفُ تُقفل ثمّ تُعَدّ.**

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

var (
	// ErrOwnerRoleProtected **دورٌ لا يُمنَح إلّا من مالك.**
	//
	// **والمفتاحُ قائمٌ في المعجم** («لا تملك صلاحية لهذا الإجراء») —
	// **وهو صادقٌ حرفيّاً**، فلا نصَّ جديدٌ في الويب لأجله.
	ErrOwnerRoleProtected = httpx.NewError(http.StatusForbidden,
		"owner_role_protected", "errors.forbidden")

	// ErrLastOwner **ولا تُفرِّغ المنصّةَ من مالكها.**
	//
	// **وهي نزعٌ من النفس بالضرورة** — من يملك حقَّ النزع مالكٌ،
	// فإن كان المالكُ واحداً فهو هو.
	ErrLastOwner = httpx.NewError(http.StatusConflict,
		"last_owner", "errors.self_action")

	// ErrRoleNotCreatable **دورٌ لا يُخلَق به حسابٌ مباشرةً.**
	//
	// **وتخويلُ العمل خطوتان**: حسابٌ يُخلَق ثمّ دورٌ يُمنَح بمساره
	// المخوَّل — **فبابُ الإنشاء قدرتُه `users.status.manage` لا
	// `roles.manage`.**
	ErrRoleNotCreatable = httpx.NewError(http.StatusForbidden,
		"role_not_creatable", "errors.forbidden")

	// ErrRoleGrantRetired **دورُ إرثٍ يُقرأ ولا يُمنَح جديداً** (`OPS-5`).
	//
	// **ومن يحمله يبقى** — **والدمجُ هجرةٌ بقرارٍ مستقلّ.**
	ErrRoleGrantRetired = httpx.NewError(http.StatusForbidden,
		"role_grant_retired", "errors.forbidden")
)

// actorHoldsRole **أيملك الفاعلُ هذا الدورَ فعلاً؟**
//
// **ويُقرأ من `user_roles` لا من مُدّعى الإثبات** — **ورمزُ وصولٍ
// قديمٌ يحمل أدواراً نُزعت.**
func actorHoldsRole(ctx context.Context, q dbtx.Querier, actorID, role string) (bool, error) {
	if actorID == "" {
		return false, nil
	}
	var has bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_roles
		                WHERE user_id = $1::uuid AND role_code = $2)`,
		actorID, role).Scan(&has); err != nil {
		return false, err
	}
	return has, nil
}

// guardGrantAuthority **سلطةُ منحِ هذا الدور** — القاعدةُ الواحدة.
//
// **ويقرؤها المنحُ والإنشاءُ معاً** — **ومن كتب الشرطَ في واحدٍ منهما
// ترك البابَ الآخر**، وذاك ما وقع.
func guardGrantAuthority(ctx context.Context, q dbtx.Querier, actorID, role string) error {
	switch authz.AuthorityToGrant(role) {
	case authz.GrantNever:
		return ErrRoleGrantRetired
	case authz.GrantByOwner:
		has, err := actorHoldsRole(ctx, q, actorID, authz.RoleOwnerSuperAdmin)
		if err != nil {
			return err
		}
		if !has {
			return ErrOwnerRoleProtected
		}
		return nil
	default:
		// **و`roles.manage` يقيسها الوسيطُ قبل أن يصل النداءُ هنا** —
		// **ولا تُقاس مرّتين بمصدرين.**
		return nil
	}
}

// guardProtectedGrant **حارسُ مسار المنح.**
func guardProtectedGrant(ctx context.Context, q dbtx.Querier, actorID, role string) error {
	return guardGrantAuthority(ctx, q, actorID, role)
}

// guardProtectedRevoke **حارسُ مسار النزع.**
//
// **والنزعُ ليس المنحَ** (وهذا فرقٌ مقصود):
//
//	**المحميُّ**  ⇒ من مالكٍ، وما دام يبقى مالكٌ بعده
//	**المرتفعُ**  ⇒ **يُنزَع بـ`roles.manage`** — **فنزعُه خفضٌ لا
//	              رفع**، **ولو لزمه مالكٌ وليس في الإنتاج مالكٌ لَصار
//	              كلُّ أدمنٍ دائماً** ولا يُسحب منه شيء
//	**الإرثُ**    ⇒ يُنزَع ليُفرَّغ تدريجاً (`OPS-5`)
func guardProtectedRevoke(ctx context.Context, q dbtx.Querier, actorID, role string) error {
	if !authz.IsProtectedRole(role) {
		return nil
	}
	has, err := actorHoldsRole(ctx, q, actorID, role)
	if err != nil {
		return err
	}
	if !has {
		return ErrOwnerRoleProtected
	}
	// **وتُقفل الصفوفُ ثمّ تُعَدّ** — **ولا عدٌّ بلا قفلٍ في نزعٍ
	// متزامن.**
	rows, err := q.Query(ctx,
		`SELECT user_id FROM user_roles WHERE role_code = $1 FOR UPDATE`, role)
	if err != nil {
		return err
	}
	n := 0
	for rows.Next() {
		n++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	if n <= 1 {
		return ErrLastOwner
	}
	return nil
}

// guardCreatableRoles **بابُ إنشاء الحساب** — **وهو أضيقُ من باب المنح.**
//
// **ولا يُتَّكل على `AllRoles`**: **قائمةٌ مُصرَّفةٌ حجبت دورَ المالك
// بالعرَض لا بقصد** (رمزُه ليس فيها)، **ولم تحجب `admin`** — **وذاك
// هو الذي فُتح منه الباب.**
func (s *Service) guardCreatableRoles(ctx context.Context, actorID string, roles []string) error {
	for _, r := range roles {
		// **والسلطةُ تُقاس قبل الشكل** — **والترتيبُ مقصود**:
		//
		//	موظّفٌ يُنشئ أدمناً  ⇒ `owner_role_protected` — **جوابُ أمنٍ**
		//	مالكٌ يُنشئ أدمناً   ⇒ `role_not_creatable`  — **جوابُ تدفّق**
		//
		// **ولو عُكس لَقرأ المتسلّلُ «لا يُخلَق من هنا»** فظنّ البابَ
		// شكليّاً، **ولَقرأ المالكُ «لا تملك صلاحية» وهو يملكها.**
		if err := guardGrantAuthority(ctx, s.repo.pool(), actorID, r); err != nil {
			return err
		}
		if !authz.CreatableAtSignup(r) {
			return ErrRoleNotCreatable
		}
	}
	return nil
}
