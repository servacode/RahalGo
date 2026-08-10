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
	other, err := r.hasOtherPrimary(ctx, userID, role)
	if err != nil {
		return err
	}
	if other != "" {
		return ErrRoleConflict
	}
	return nil
}

// FieldRoleSet **الأدوارُ الميدانيّة** — يُمنح صاحبُها دورَ الزبون تلقائياً.
var FieldRoleSet = []string{"merchant", "driver", "sales"}

// isField **أدورٌ ميدانيّ؟**
func isField(role string) bool { return slices.Contains(FieldRoleSet, role) }
