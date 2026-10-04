package rolesguard

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **آخرُ `roles.manage` لا يُنزَع** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
// ══════════════════════════════════════════════════════════════════════
//
// **والقاعدةُ مشتركةٌ متراكمة** — فيُعزَل داخل معاملةٍ تُرجَع: تُنزَع
// `roles.manage` من كلّ الأدوار ثمّ تُمنح لدورٍ مؤقّتٍ يحمله من نريد.

func rolesManageIsolated(t *testing.T, holders ...string) (context.Context, pgx.Tx) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("بدءُ المعاملة: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	for _, q := range []string{
		`DELETE FROM role_capabilities WHERE capability_code = 'roles.manage'`,
		`INSERT INTO roles (code, name_key) VALUES ('zz_rm_last', 'zz_rm_last')`,
		`INSERT INTO role_capabilities (role_code, capability_code) VALUES ('zz_rm_last', 'roles.manage')`,
	} {
		if _, err := tx.Exec(ctx, q); err != nil {
			t.Fatalf("العزل: %v", err)
		}
	}
	for _, h := range holders {
		if _, err := tx.Exec(ctx,
			`INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'zz_rm_last')`, h); err != nil {
			t.Fatalf("حاملٌ: %v", err)
		}
	}
	return ctx, tx
}

// TestLastRolesManage_CapabilityRevokeRefused **نزعُ `roles.manage` من الدور الوحيد الحامل ⇒ يُردّ.**
func TestLastRolesManage_CapabilityRevokeRefused(t *testing.T) {
	pool := testdb.Pool(t)
	u := testdb.NewUser(t, pool, "customer")
	ctx, tx := rolesManageIsolated(t, u)
	err := GuardRolesManageRemains(ctx, tx, RolesManageLoss{Role: "zz_rm_last"})
	if !errors.Is(err, ErrLastRolesManage) {
		t.Fatalf("**نُزعت آخرُ `roles.manage` في المنصّة**: %v", err)
	}
}

// TestLastRolesManage_UserRoleRevokeRefused **نزعُ الدور من آخر حاملٍ ⇒ يُردّ، ومع حاملٍ ثانٍ يمرّ.**
func TestLastRolesManage_UserRoleRevokeRefused(t *testing.T) {
	pool := testdb.Pool(t)
	a := testdb.NewUser(t, pool, "customer")
	b := testdb.NewUser(t, pool, "customer")
	ctx, tx := rolesManageIsolated(t, a)
	loss := RolesManageLoss{UserID: a, UserRole: "zz_rm_last"}
	if err := GuardRolesManageRemains(ctx, tx, loss); !errors.Is(err, ErrLastRolesManage) {
		t.Fatalf("**نُزع الدورُ من آخر من يدير الأدوار**: %v", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'zz_rm_last')`, b); err != nil {
		t.Fatalf("حاملٌ ثانٍ: %v", err)
	}
	if err := GuardRolesManageRemains(ctx, tx, loss); err != nil {
		t.Fatalf("**رُدّ النزعُ وفي المنصّة غيرُه**: %v", err)
	}
	// **والموقوفُ لا يُعَدّ** — حسابٌ لا يدخل لا يُدير شيئاً.
	if _, err := tx.Exec(ctx, `UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, b); err != nil {
		t.Fatalf("إيقاف: %v", err)
	}
	if err := GuardRolesManageRemains(ctx, tx, loss); !errors.Is(err, ErrLastRolesManage) {
		t.Fatalf("**عُدّ حسابٌ موقوفٌ مديراً للأدوار**: %v", err)
	}
}
