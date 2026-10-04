// Package rolesguard **حارسُ «آخرُ من يدير الأدوار لا يُنزَع»** — يقرؤه بابُ نزع
// القدرة من دور (`server`) وبابُ نزع الدور من حساب (`identity`). **وخطؤه لوحةٌ
// لا هاتف** — فلا يسكن حزمةً يبلغها تطبيقٌ.
package rolesguard

// ══════════════════════════════════════════════════════════════════════
// **آخرُ `roles.manage` في المنصّة لا يُنزَع** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **قِيس على التجهيز**: دورُ المالك الأعلى بلا حاملٍ، ومديرُ المنصّة حسابٌ
// واحد. **فلو نزع المديرُ `roles.manage` من دوره لما بقي أحدٌ يُعيدها** —
// واللوحةُ مقفلةٌ على صاحبها، **ولا مخرجَ إلّا سطرٌ يدويٌّ في القاعدة.**
//
// **ويُقاس من بابين**: نزعُ القدرة من دور، ونزعُ الدور من حساب. **ويُقفَل
// صفُّ القدرة في الجدول قبل العدّ** — فنزعان متزامنان لا يمرّان معاً.

import (
	"context"
	"net/http"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// ErrLastRolesManage **لا يُنزَع آخرُ من يدير الأدوار.**
var ErrLastRolesManage = httpx.NewError(http.StatusConflict,
	"last_roles_manage", "errors.last_roles_manage")

// RolesManageLoss **ما سيُفقَد** — قدرةٌ من دور، أو دورٌ من حساب.
type RolesManageLoss struct {
	// Role **دورٌ سيفقد `roles.manage`** — فارغٌ إن لم يكن.
	Role string
	// UserID و UserRole **حسابٌ سيفقد دوراً** — فارغان إن لم يكن.
	UserID   string
	UserRole string
}

// GuardRolesManageRemains **أيبقى بعد هذا النزع حسابٌ فعّالٌ يملك `roles.manage`؟**
//
// يُنادى داخل معاملة النزع قبل الكتابة.
func GuardRolesManageRemains(ctx context.Context, q dbtx.Querier, loss RolesManageLoss) error {
	// **والقفلُ أوّلاً** — صفوفُ `roles.manage` في جدول القدرات.
	if _, err := q.Exec(ctx, `
		SELECT 1 FROM role_capabilities WHERE capability_code = $1 FOR UPDATE`,
		string(authz.RolesManage)); err != nil {
		return err
	}
	var remains bool
	if err := q.QueryRow(ctx, `
		SELECT EXISTS (
		  SELECT 1 FROM users u
		    JOIN user_roles ur ON ur.user_id = u.id
		    JOIN role_capabilities rc ON rc.role_code = ur.role_code
		   WHERE rc.capability_code = $1
		     AND u.status = 'active'
		     AND rc.role_code <> $2
		     AND NOT (ur.user_id::text = $3 AND ur.role_code = $4))`,
		string(authz.RolesManage), loss.Role, loss.UserID, loss.UserRole).
		Scan(&remains); err != nil {
		return err
	}
	if !remains {
		return ErrLastRolesManage
	}
	return nil
}
