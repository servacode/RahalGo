package identity

// **ولا دورين — من أيّ بابٍ جاء المنح.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-١٦: «ممنوعٌ منعاً باتاً أن يأخذ أكثرَ من دور» —
//  تأكيدُ قراره ٢٠٢٦-٠٨-١٠.)
//
// # الثغرةُ التي كانت
//
// **الفحصُ كان في `AdminGrantRole` وحدَه** — **و`EnsureUserWithRole` تمنح
// مباشرةً بلا مرورٍ به.** وهي طريقُ فتح المتجر: **يُكتب هاتفُ سائقٍ في
// خانة صاحب المتجر فيصير سائقاً وصاحبَ متجرٍ معاً.**
//
// **وهو عينُ التعارض الذي كُتبت القاعدةُ لمنعه**: صاحبُ متجرٍ هو سائقٌ
// **يُسند لنفسه طلبَ متجره**. **ولا يُكشف في مراجعةٍ لأنّ كلَّ فعلٍ منه
// مشروعٌ وحدَه.**
//
// **وازداد البابُ استعمالاً** حين صار هاتفُ المالك إلزاميّاً في نموذج
// المتجر (٢٠٢٦-٠٨-١٥) — **فبابٌ كان ضيّقاً صار البابَ.**
//
// # ولماذا الحارسُ في `GrantRole` لا عند نداءاته
//
// **من بلغ المنحَ فقد وجب عليه** — **ولا نداءَ جديدٌ ينساه غدا.**

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestGrantRole_RefusesSecondPrimaryFromAnyPath
// **سائقٌ لا يصير صاحبَ متجرٍ ولو من نافذة المتجر.**
func TestGrantRole_RefusesSecondPrimaryFromAnyPath(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	driver := testdb.NewUser(t, pool, "driver")

	// **والزبونُ يجتمع مع كلّ شيء** — هو بابُ التصفّح لا سلطةٌ تُمنح.
	if err := repo.GrantRole(ctx, driver, "customer", nil); err != nil {
		t.Fatalf("رُدّ دورُ الزبون: %v — **وكلُّ إنسانٍ في المنصّة زبونٌ بالضرورة**", err)
	}
	// **ومنحُ ما هو ممنوحٌ لا شيء.**
	if err := repo.GrantRole(ctx, driver, "driver", nil); err != nil {
		t.Fatalf("رُدّ دورُه نفسُه: %v", err)
	}

	// ══════════════════════════════════════════════════════════════════
	// **والثاني يُردّ — وهذا هو البابُ الذي كان مفتوحا**
	// ══════════════════════════════════════════════════════════════════
	for _, role := range []string{"merchant", "sales", "ops"} {
		if err := repo.GrantRole(ctx, driver, role, nil); !errors.Is(err, ErrRoleConflict) {
			t.Fatalf("مُنح %q لسائقٍ: %v — **وتضاربُ المصالح لا يُكشف في مراجعة**",
				role, err)
		}
	}

	// **ولا يُكتب شيءٌ في القاعدة** — الفحصُ قبل الكتابة لا بعدها.
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM user_roles WHERE user_id = $1 AND role_code NOT IN ('driver','customer')`,
		driver).Scan(&n); err != nil {
		t.Fatalf("تعذّر العدّ: %v", err)
	}
	if n != 0 {
		t.Fatalf("كُتب %d دورٍ رغم الردّ — **ومنحٌ يقع ثمّ يُسحب يترك أثراً في السجلّ**", n)
	}
}
