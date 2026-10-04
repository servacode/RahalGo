package main

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestSeedRolesExist **كلُّ دورٍ تزرعه الزراعةُ موجودٌ في جدول الأدوار.**
//
// حُذف الدورُ `ops` في الهجرة 0270 ونُقل حاملوه إلى `operations`، وبقيت
// الزراعةُ تكتب `ops` — فسقطت على قاعدةٍ جديدةٍ عند أوّل حساب طاقم
// (`user_roles_role_code_fkey`) ولم يُزرع شيءٌ بعده.
func TestSeedRolesExist(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	for _, a := range accounts {
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM roles WHERE code = $1)`, a.Role).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("الزراعةُ تمنح %s الدورَ %q وهو غيرُ موجودٍ في roles", a.Phone, a.Role)
		}
	}
}

// TestSeedStorePlatformSectionsExist **كلُّ قسمِ سوقٍ يُسنَد إليه صنفُ المطعم موجود.**
//
// صار «حلويات» «حلويّات» في جدول أقسام السوق، وبقيت الزراعةُ تطلب القديم —
// فسقط `seed -store` قبل أن يُنشئ قسمَ الحلويات وأصنافَه.
func TestSeedStorePlatformSectionsExist(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	for _, sec := range restaurant.Sections {
		if sec.Platform == "" {
			continue
		}
		var ok bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM platform_sections WHERE name = $1)`, sec.Platform).Scan(&ok); err != nil {
			t.Fatal(err)
		}
		if !ok {
			t.Errorf("قسمُ «%s» يُسنَد إلى قسمِ سوقٍ «%s» غيرِ موجود", sec.Name, sec.Platform)
		}
	}
}
