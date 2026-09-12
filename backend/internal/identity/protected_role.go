package identity

// ══════════════════════════════════════════════════════════════════════
//  **حارسُ الدور المحميّ — الحدُّ في المحرّك لا في اللوحة**
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع
//
// **دورةُ ٢٠٢٦-٠٩-١٢ أخفت `owner_super_admin` من نافذة الإسناد** —
// **وذاك لطفٌ بالعين لا حراسة.** **فقِيس المسارُ الحقيقيُّ على
// التجهيز**: أدمنٌ يملك `roles.manage` ولا يملك دورَ المالك ⇒
//
//	منحُه لحسابٍ آخر   ⇒ **200 granted**
//	منحُه لنفسِه       ⇒ **200 granted**   ← ترقيةٌ ذاتيّةٌ كاملة
//	نزعُه              ⇒ **200 revoked**
//
// **وحارسُ الدور الواحد لم يمنع الترقيةَ الذاتيّة** — **فهو للميدان
// وحدَه** (سائقٌ ومتجرٌ ومندوب)، **وأدوارُ المكتب تتّحد بقرار
// ٢٠٢٦-٠٩-٠٧.** **وظنّي أنّه يمنعها كان خطأً، والقياسُ صحّحه.**
//
// # والقاعدةُ المُختارة — وهي الأضيق
//
//	**من ملك دورَ المالك وحدَه يمنحه أو ينزعه**
//	وبتأكيدٍ بكلمة المرور وقيدِ تدقيقٍ كما كانا
//
// **ولا قدرةَ جديدةٌ تُخترَع** (`owner.manage` ونحوُها): **قدرةٌ جديدةٌ
// تحتاج صفّاً في المصفوفة، وصفٌّ يُمنَح بالخطأ يُبطل الحارسَ كلَّه.**
// **والدورُ نفسُه هويّةُ صاحبه** — انظر `authz.RoleOwnerSuperAdmin`.
//
// # وآخرُ مالكٍ لا يُنزَع
//
// **والقاعدةُ أعلاه تُنتج قفلاً**: **لو نزع المالكُ الأوحدُ دورَه
// لَما بقي أحدٌ يستطيع منحَه** — **فلا يُستعاد إلّا من الخادم.**
// **وهي بالضرورة نزعٌ من النفس**: من يملك الحقَّ في النزع مالكٌ،
// **فإن كان المالكُ واحداً فهو هو.**
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
	// ErrOwnerRoleProtected **لا يُمنَح دورُ المالك إلّا من مالك.**
	//
	// **والمفتاحُ قائمٌ في المعجم** («لا تملك صلاحية لهذا الإجراء») —
	// **وهو صادقٌ حرفيّاً**، فلا نصَّ جديدٌ في الويب ولا أثرٌ جديدٌ له.
	ErrOwnerRoleProtected = httpx.NewError(http.StatusForbidden,
		"owner_role_protected", "errors.forbidden")

	// ErrLastOwner **ولا تُفرِّغ المنصّةَ من مالكها.**
	//
	// **وهي نزعٌ من النفس بالضرورة** — فمفتاحُ «لا يمكنك تنفيذ هذا
	// الإجراء على حسابك» يصفها، **ورمزُها مستقلٌّ يُقرأ في العقد.**
	ErrLastOwner = httpx.NewError(http.StatusConflict,
		"last_owner", "errors.self_action")
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

// guardProtectedGrant **منحُ دورٍ محميٍّ — من مالكٍ إلى غيره.**
func guardProtectedGrant(ctx context.Context, q dbtx.Querier, actorID, role string) error {
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
	return nil
}

// guardProtectedRevoke **نزعُه** — من مالكٍ، وما دام يبقى مالك.
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

// guardProtectedRoles **بابُ إنشاء الحساب** — وهو يأخذ الأدوارَ من مدخله.
//
// **ولا يُتَّكل على `AllRoles`**: **قائمةٌ مُصرَّفةٌ تحجب اليومَ بالعرَض**،
// **ومن أضاف رمزاً إليها غداً فتح باباً لا يعرف أنّه فتحه.**
func (s *Service) guardProtectedRoles(ctx context.Context, actorID string, roles []string) error {
	for _, r := range roles {
		if err := guardProtectedGrant(ctx, s.repo.pool(), actorID, r); err != nil {
			return err
		}
	}
	return nil
}
