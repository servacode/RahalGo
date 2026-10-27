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

// TestAPR_ProposerApprovesOwnFlagged **قرارُ المالك ٢٠٢٦-١٠-٢٢: المقترحُ يوافق على ما
// اقترحه ولو وُجد غيرُه — وتُعلَّم في السجلّ.**
func TestAPR_ProposerApprovesOwnFlagged(t *testing.T) {
	pool := testdb.Pool(t)
	admin := testdb.NewUser(t, pool, "admin")
	fin := testdb.NewUser(t, pool, "finance")
	ctx, tx := isolated(t, admin, fin)
	v, err := Check(ctx, tx, req(admin, admin))
	if err != nil || !v.SelfApproved {
		t.Fatalf("**المقترحُ لم يوافق على اقتراحه**: %v · %v", v, err)
	}
	if f := v.AuditFields(); f["self_approved"] != true || f["self_approval_reason"] != SelfApprovalByProposer {
		t.Fatalf("**الموافقةُ الذاتيّةُ بلا علامةٍ في السجلّ**: %v", f)
	}
}

// TestAPR_OwnFundsRefusedWhenAnotherExists **مالُ الفاعل نفسِه (سحبُه) ⇒ لا يقرّر فيه
// ويوجد غيرُه — ولو كان المالك.**
func TestAPR_OwnFundsRefusedWhenAnotherExists(t *testing.T) {
	pool := testdb.Pool(t)
	owner := testdb.NewUser(t, pool, authz.RoleOwnerSuperAdmin)
	fin := testdb.NewUser(t, pool, "finance")
	ctx, tx := isolated(t, owner, fin)
	r := req(owner, owner)
	r.OwnFunds = true
	if _, err := Check(ctx, tx, r); !errors.Is(err, ErrSelfApprove) {
		t.Fatalf("**قرّر في سحبه وفي المنصّة غيرُه**: %v", err)
	}
}

// TestAPR_OwnerAloneSelfApprovesFlagged **المالكُ وحدَه بلا بديل ⇒ يوافق، وتُعلَّم في السجلّ.**
func TestAPR_OwnerAloneSelfApprovesFlagged(t *testing.T) {
	pool := testdb.Pool(t)
	owner := testdb.NewUser(t, pool, authz.RoleOwnerSuperAdmin)
	ctx, tx := isolated(t, owner)
	r := req(owner, owner)
	r.OwnFunds = true
	v, err := Check(ctx, tx, r)
	if err != nil || !v.SelfApproved {
		t.Fatalf("**المالكُ وحدَه لم يوافق على نفسه**: %v · %v", v, err)
	}
	f := v.AuditFields()
	if f["self_approved"] != true || f["self_approval_reason"] != SelfApprovalReason {
		t.Fatalf("**الموافقةُ الذاتيّةُ بلا علامةٍ صريحةٍ في السجلّ**: %v", f)
	}
}

// TestAPR_AutoAppliesForOwnerAndAdminOnly **قرارُ المالك ٢٠٢٦-١٠-٢٧: حركةُ المالك أو المدير
// تنفذ فوراً، وطلبُ غيرِهما ينتظر** — والحكمُ من الأدوار في القاعدة لا من الشاشة.
func TestAPR_AutoAppliesForOwnerAndAdminOnly(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	for role, want := range map[string]bool{
		authz.RoleOwnerSuperAdmin: true,
		authz.RoleAdmin:           true,
		"finance":                 false,
		"operations":              false,
	} {
		u := testdb.NewUser(t, pool, role)
		got, err := AutoApplies(ctx, pool, u)
		if err != nil || got != want {
			t.Errorf("الدور %s: تنفيذٌ فوريّ=%v (%v) — والمنتظَر %v", role, got, err, want)
		}
	}
	if got, err := AutoApplies(ctx, pool, ""); err != nil || got {
		t.Errorf("فاعلٌ بلا معرّف نفذ فوراً: %v %v", got, err)
	}
}
