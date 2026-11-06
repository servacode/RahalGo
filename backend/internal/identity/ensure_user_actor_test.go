package identity

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestEnsureUserWithRole_EmptyActorOnExistingUser **منحُ دورٍ لحسابٍ قائمٍ بلا فاعل**
// (نداءُ النظام أو جسرُ QA) — كان يمرّر نصّاً فارغاً إلى granted_by فيرفضه بوستغرس (22P02)
// ويصير الجوابُ 404. والفاعلُ الفارغ يُكتب NULL.
func TestEnsureUserWithRole_EmptyActorOnExistingUser(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	s := &Service{repo: NewRepo(pool)}

	id := testdb.NewUser(t, pool, "customer")
	var phone string
	if err := pool.QueryRow(ctx, `SELECT phone FROM users WHERE id = $1`, id).Scan(&phone); err != nil {
		t.Fatalf("read phone: %v", err)
	}
	u, err := s.EnsureUserWithRole(ctx, "", phone, "merchant", "", "", "127.0.0.1")
	if err != nil {
		t.Fatalf("granting a role with an empty actor failed: %v", err)
	}
	if u.ID != id {
		t.Fatalf("got user %s, want %s", u.ID, id)
	}
	var grantedByNull bool
	if err := pool.QueryRow(ctx,
		`SELECT granted_by IS NULL FROM user_roles WHERE user_id = $1 AND role_code = 'merchant'`,
		id).Scan(&grantedByNull); err != nil {
		t.Fatalf("role row missing: %v", err)
	}
	if !grantedByNull {
		t.Fatalf("granted_by should be NULL for an empty actor")
	}
}
