package approval

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/authz"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ══════════════════════════════════════════════════════════════════════
// **«المقترحُ غيرُ الموافق» — قرارُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ الأدوار، البند ٦)**
// ══════════════════════════════════════════════════════════════════════
//
// **وقاعدةُ الاختبار مشتركةٌ متراكمة** — ففيها دائماً من يملك `finance.manage`.
// **فتُقاس «لا أحدَ غيرُه» داخل معاملةٍ تُرجَع**: تُنزَع القدرةُ من كلّ الأدوار
// ثمّ تُمنح لدورٍ مؤقّتٍ يحمله من نريد — ولا يبقى في القاعدة أثر.

func isolated(t *testing.T, holders ...string) (context.Context, pgx.Tx) {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatalf("بدءُ المعاملة: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if _, err := tx.Exec(ctx, `DELETE FROM role_capabilities WHERE capability_code = $1`,
		string(authz.FinanceManage)); err != nil {
		t.Fatalf("عزلُ القدرة: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO roles (code, name_key) VALUES ('zz_apr_fin', 'zz_apr_fin');
	`); err != nil {
		t.Fatalf("دورٌ مؤقّت: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO role_capabilities (role_code, capability_code) VALUES ('zz_apr_fin', $1)`,
		string(authz.FinanceManage)); err != nil {
		t.Fatalf("قدرةُ الدور المؤقّت: %v", err)
	}
	for _, h := range holders {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_code) VALUES ($1::uuid, 'zz_apr_fin')`, h); err != nil {
			t.Fatalf("حاملُ الدور المؤقّت: %v", err)
		}
	}
	return ctx, tx
}

func req(proposer, actor string) Request {
	return Request{ProposedBy: proposer, Actor: actor, Capability: authz.FinanceManage}
}

// TestAPR_OtherApproverPasses **غيرُ صاحب الاقتراح يوافق بلا علامة.**
func TestAPR_OtherApproverPasses(t *testing.T) {
	pool := testdb.Pool(t)
	a := testdb.NewUser(t, pool, "finance")
	b := testdb.NewUser(t, pool, "admin")
	ctx, tx := isolated(t, a, b)
	v, err := Check(ctx, tx, req(a, b))
	if err != nil || v.SelfApproved {
		t.Fatalf("**موافقةُ غيره رُدّت أو عُلّمت ذاتيّة**: %v · %v", v, err)
	}
}

// TestAPR_SelfApproveRefusedWhenAnotherExists **يوجد غيرُه ⇒ لا يوافق على نفسه ولو كان المالك.**
func TestAPR_SelfApproveRefusedWhenAnotherExists(t *testing.T) {
	pool := testdb.Pool(t)
	owner := testdb.NewUser(t, pool, authz.RoleOwnerSuperAdmin)
	fin := testdb.NewUser(t, pool, "finance")
	ctx, tx := isolated(t, owner, fin)
	if _, err := Check(ctx, tx, req(owner, owner)); !errors.Is(err, ErrSelfApprove) {
		t.Fatalf("**المالكُ وافق على نفسه وفي المنصّة غيرُه**: %v", err)
	}
}

// TestAPR_AdminAloneCannotSelfApprove **مديرُ المنصّة وحدَه لا يوافق على نفسه — المالكُ وحدَه.**
//
// **وكان `canSelfApprove` يجيزها لمدير المنصّة** — وقرارُ المالك: للمالك وحدَه.
func TestAPR_AdminAloneCannotSelfApprove(t *testing.T) {
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	ctx, tx := isolated(t, admin)
	if _, err := Check(ctx, tx, req(admin, admin)); !errors.Is(err, ErrSelfApprove) {
		t.Fatalf("**مديرُ المنصّة وافق على نفسه**: %v", err)
	}
}

// TestAPR_OwnerAloneSelfApprovesFlagged **المالكُ وحدَه بلا بديل ⇒ يوافق، وتُعلَّم في السجلّ.**
func TestAPR_OwnerAloneSelfApprovesFlagged(t *testing.T) {
	pool := testdb.Pool(t)
	owner := testdb.NewUser(t, pool, authz.RoleOwnerSuperAdmin)
	ctx, tx := isolated(t, owner)
	v, err := Check(ctx, tx, req(owner, owner))
	if err != nil || !v.SelfApproved {
		t.Fatalf("**المالكُ وحدَه لم يوافق على نفسه**: %v · %v", v, err)
	}
	f := v.AuditFields()
	if f["self_approved"] != true || f["self_approval_reason"] != SelfApprovalReason {
		t.Fatalf("**الموافقةُ الذاتيّةُ بلا علامةٍ صريحةٍ في السجلّ**: %v", f)
	}
}
