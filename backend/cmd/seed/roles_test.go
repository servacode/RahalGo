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

// TestSeedCustomerGetsBalance **زبونُ الزراعة يُودَع رصيدُه فعلاً.**
//
// كان الإيداعُ مشروطاً بـ«لا محفظةَ بعد»، والمحفظةُ يُنشئها مشغّلُ
// `users_wallet_trigger` مع المستخدم — فبقي الرصيدُ صفراً والزراعةُ تطبع غيره.
func TestSeedCustomerGetsBalance(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	seedCustomer(ctx, tx)
	var bal, sum int64
	if err := tx.QueryRow(ctx, `
		SELECT w.balance, COALESCE((SELECT sum(amount) FROM wallet_transactions t WHERE t.user_id = u.id), 0)
		FROM users u JOIN wallets w ON w.user_id = u.id WHERE u.phone = $1`, customer.Phone).Scan(&bal, &sum); err != nil {
		t.Fatal(err)
	}
	if bal != customer.Balance || sum != customer.Balance {
		t.Fatalf("رصيدُ زبون الزراعة %d ومجموعُ قيوده %d — والمطلوب %d", bal, sum, customer.Balance)
	}
	// **والإعادةُ لا تودع ثانيةً.**
	seedCustomer(ctx, tx)
	if err := tx.QueryRow(ctx, `SELECT w.balance FROM users u JOIN wallets w ON w.user_id = u.id WHERE u.phone = $1`,
		customer.Phone).Scan(&bal); err != nil {
		t.Fatal(err)
	}
	if bal != customer.Balance {
		t.Fatalf("إعادةُ الزراعة أودعت ثانيةً: %d", bal)
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
