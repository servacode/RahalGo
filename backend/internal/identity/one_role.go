package identity

/*
**دورٌ واحدٌ ومعه الزبون — ولا ثالث.**

(قرارُ المالك ٢٠٢٦-٠٨-١٠: «السائقُ يمكن أن يكون زبوناً، وكذلك المتجرُ والمندوبُ
 وصاحبُ المنصّة والموظّفون. يعني أيُّ دورٍ لا يمكن أن يظهر له إلّا كزبون — ما
 يصير سائقاً ومندوباً وصاحبَ متجرٍ معاً. يعني دورين فقط».)

# ولماذا الزبونُ لا يُعدّ دوراً ثانياً

**كلُّ إنسانٍ في المنصّة زبونٌ بالضرورة** — السائقُ يطلب عشاءه، وصاحبُ المتجر
يطلب من متجرٍ آخر. **ودورُ الزبون بابُ التصفّح والطلب والمحفظة**، لا سلطةٌ
تُمنح.

# وما الذي يمنعه هذا

**الجمعُ بين دورين ميدانيّين أو إداريّين** — سائقٌ ومندوب، أو صاحبُ متجرٍ
وسائق، أو عملياتٌ وماليّة.

**والضررُ ليس في الشاشة**: مندوبٌ هو صاحبُ متجرٍ **يمنح متجرَه عمولةَ نفسِه**؛
وسائقٌ هو مندوبٌ **يُسند لنفسه**. **وتضاربُ المصالح لا يُكشف في مراجعةٍ لأنّ
كلَّ فعلٍ منه مشروعٌ وحدَه.**

# والحكمُ في المحرّك لا في الشاشة

**قائمةُ اختيارٍ تمنع لا تمنع شيئاً** — من نادى الواجهةَ البرمجيّة مباشرةً
تجاوزها. **وحجبٌ في العرض وحدَه وعدٌ بحجب.**
*/

import (
	"context"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"net/http"
	"slices"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// RoleCustomer **دورُ الزبون — لا يُعدّ في السقف.**
const RoleCustomer = "customer"

// ErrRoleConflict **له دورٌ غيرُ الزبون بالفعل.**
var ErrRoleConflict = httpx.NewError(http.StatusConflict,
	"role_conflict", "errors.role_conflict")

// primaryRoles **الأدوارُ التي لا تُجمع** — كلُّ ما عدا الزبون.
func primaryRoles(roles []string) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if r != RoleCustomer {
			out = append(out, r)
		}
	}
	return out
}

// checkOnePrimary **يمنع أن يحمل الحسابُ دورين غيرَ الزبون.**
//
// **ويُنادى قبل الكتابة لا بعدها** — ومنحٌ يقع ثمّ يُسحب يترك أثراً في السجلّ
// وإشعاراً وصل صاحبَه.
func checkOnePrimary(roles []string) error {
	if len(primaryRoles(roles)) > 1 {
		return ErrRoleConflict
	}
	return nil
}

// hasOtherPrimary **أله دورٌ أساسيٌّ غيرُ هذا؟**
// GrantRoleTx كـ`GrantRole` **في معاملةٍ مُمرَّرة** — بحارس الدور الواحد.
//
// **والحارسُ محفوظٌ** (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ممنوعٌ منعاً باتاً أن
// يأخذ أكثرَ من دور»): **ومن نسخ المنحَ بلا حارسِه فتح البابَ الذي
// أُغلق.**
func GrantRoleTx(ctx context.Context, q dbtx.Querier, userID, role string,
	grantedBy *string) error {
	if role != RoleCustomer {
		rows, err := q.Query(ctx,
			`SELECT role_code FROM user_roles WHERE user_id = $1`, userID)
		if err != nil {
			return err
		}
		var other string
		for rows.Next() {
			var code string
			if err := rows.Scan(&code); err != nil {
				rows.Close()
				return err
			}
			if code != RoleCustomer && code != role {
				other = code
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
		if other != "" {
			return ErrRoleConflict
		}
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_code, granted_by) VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING`, userID, role, grantedBy); err != nil {
		return err
	}
	if grantsCustomer(role) {
		if _, err := q.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1, 'customer')
			ON CONFLICT DO NOTHING`, userID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) hasOtherPrimary(ctx context.Context, userID, role string) (string, error) {
	rows, err := r.db.Query(ctx,
		`SELECT role_code FROM user_roles WHERE user_id = $1`, userID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return "", err
		}
		// **والدورُ نفسُه لا يُعدّ تعارضاً** — منحُ ما هو ممنوحٌ لا شيء.
		if code != RoleCustomer && code != role {
			return code, nil
		}
	}
	return "", rows.Err()
}

// ensureOnePrimary **يردّ خطأً إن كان له دورٌ أساسيٌّ آخر.**
func (r *Repo) ensureOnePrimary(ctx context.Context, userID, role string) error {
	if role == RoleCustomer {
		// **والزبونُ يُضاف لأيّ أحد** — هو الدورُ الذي يجتمع مع كلّ شيء.
		return nil
	}
	// ══════════════════════════════════════════════════════════════
	// **والقاعدةُ للميدان لا للمكتب** — مصالحةُ دورةِ ٢٥
	// ══════════════════════════════════════════════════════════════
	//
	// **علّتُها المكتوبةُ مشاركةُ الحسابات**: «سائقٌ يعطي رقمَه ورمزَه
	// لآخرَ فيعملان معاً **فيقبضان على اسمٍ واحدٍ ولا تعرف المنصّةُ
	// من سلّم**» (قرارُ المالك ٢٠٢٦-٠٨-١٦).
	//
	// **وأدوارُ المكتب لا تقبض ولا تُسلّم** — **وقرارُ ٢٠٢٦-٠٩-٠٧
	// يقول نصّاً**: «MULTIPLE ROLES: effective capability set =
	// UNION». **فموظّفٌ يجمع الماليّةَ والتحليلَ عملٌ مشروع.**
	//
	// **والمنعُ حيث علّتُه**: **ميدانٌ لا يجتمع بمكتبٍ ولا بميدان.**
	//
	//	سائقٌ + `ops`      ⇒ **تضاربُ مصالح** — يُمنَع
	//	سائقٌ + متجرٌ      ⇒ حسابٌ مشترَك — يُمنَع
	//	ماليّةٌ + تحليلٌ    ⇒ **اتّحادٌ مشروع** — يُسمَح
	//
	// **ويُفحَص الطرفان**: **الجديدُ والقائم** — **فمنعٌ يفحص أحدَهما
	// يمرّ من الجهة الأخرى.**
	other, err := r.hasOtherPrimary(ctx, userID, role)
	if err != nil {
		return err
	}
	if other == "" {
		return nil
	}
	if isField(role) || isField(other) {
		return ErrRoleConflict
	}
	// **دورا مكتبٍ يجتمعان** — وهو عقدُ الاتّحاد.
	return nil
}

// FieldRoleSet **الأدوارُ الميدانيّة** — يُمنح صاحبُها دورَ الزبون تلقائياً.
var FieldRoleSet = []string{"merchant", "driver", "sales"}

// isField **أدورٌ ميدانيّ؟**
func isField(role string) bool { return slices.Contains(FieldRoleSet, role) }

// ErrMerchantNeedsStore **دورُ المتجر لا يُمنح بيد.**
var ErrMerchantNeedsStore = httpx.NewError(http.StatusConflict,
	"merchant_needs_store", "errors.merchant_needs_store")

// checkGrantable **يمنع منحَ دورٍ لا معنى له وحدَه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٠: «أصلاً لا يمكن إنشاءُ حسابٍ بدون متجر».)
//
// # ولماذا يُمنع بدل أن تُجمَّل شاشتُه
//
// **صاحبُ متجرٍ بلا متجرٍ يقع على شاشةٍ ميّتة**: رسالةٌ وحدَها بلا شريطٍ ولا
// زرِّ خروج — **لا يخرج ولا يذهب إلى حسابه.**
//
// **والحالُ تُبلَغ بضغطةٍ واحدة**: يُمنح أحدٌ دورَ «متجر» من لوحة الحسابات
// فيدخل ويعلق. **وإصلاحُ الشاشة يُجمّل حالاً لا ينبغي أن تقع.**
//
// # والمسارُ الشرعيُّ لا يمرّ من هنا
//
// **دورُ المتجر يُمنح مع إنشاء المتجر نفسِه** (`EnsureUserWithRole` في
// `catalog`) عند تحويل طلب اشتراك. **فالمنعُ هنا يغلق البابَ اليدويَّ
// وحدَه** — ولا يمسّ التحويل.
func checkGrantable(role string) error {
	if role == RoleMerchant {
		return ErrMerchantNeedsStore
	}
	return nil
}

// RoleMerchant **دورُ صاحب المتجر.**
const RoleMerchant = "merchant"
