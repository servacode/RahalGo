package identity

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/auth"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestStaff_LoginAsStoreWithoutKickingOwner **موظّفُ المتجر يدخل برقمه باسم المتجر**
// (قرارُ المالك ٢٠٢٦-١٠-٢٠) — ولا تُزيح جلستُه جلسةَ صاحبه، ولا جلسةُ صاحبه جلستَه؛
// ودخولُه ثانيةً يُزيح جلستَه هو؛ ولا يدخل تطبيقَ الزبون؛ وحذفُه يُخرجه بسببه.
func TestStaff_LoginAsStoreWithoutKickingOwner(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError + 1}))
	tokens := auth.NewTokenIssuer("staff-test-secret", 15*time.Minute)
	svc := NewService(NewRepo(pool), testRedis(t), tokens, nil, "staff-test", quiet)

	owner := testdb.NewUser(t, pool, "merchant")
	hash, err := auth.HashPassword("Owner@12345")
	if err != nil {
		t.Fatal(err)
	}
	var ownerPhone string
	if err := pool.QueryRow(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1 RETURNING phone`,
		owner, hash).Scan(&ownerPhone); err != nil {
		t.Fatal(err)
	}
	var cat, store string
	if err := pool.QueryRow(ctx, `INSERT INTO categories (name, icon, active) VALUES ('فحص الموظّفين', 'other', true)
		RETURNING id`).Scan(&cat); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO merchants (name, owner_user_id, status, category_id)
		VALUES ('متجر الموظّفين', $1, 'active', $2) RETURNING id`, owner, cat).Scan(&store); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, store)
		_, _ = pool.Exec(context.Background(), `DELETE FROM categories WHERE id = $1`, cat)
	})

	staffPhone := fmt.Sprintf("+96394%07d", time.Now().UnixNano()%10_000_000)
	st, plain, err := svc.AddStaff(ctx, owner, owner, staffPhone, "المسائي", "1.1.1.1")
	if err != nil {
		t.Fatalf("إضافةُ الموظّف: %v", err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(context.Background(), `DELETE FROM merchant_staff WHERE id = $1`, st.ID) })

	merchantApp := WithClient(ctx, "android-merchant")
	sessionOf := func(res *AuthResult) (sid, staff, sub string) {
		t.Helper()
		c, err := tokens.VerifyAccess(res.Tokens.AccessToken)
		if err != nil {
			t.Fatal(err)
		}
		return c.SID, c.Staff, c.Subject
	}
	alive := func(sid string) bool {
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM refresh_tokens WHERE session_id = $1::uuid
			AND revoked_at IS NULL`, sid).Scan(&n)
		return n > 0
	}

	ownerRes, err := svc.LoginPassword(merchantApp, ownerPhone, "Owner@12345", "ua", "1.1.1.1")
	if err != nil {
		t.Fatalf("دخولُ صاحب المتجر: %v", err)
	}
	ownerSID, ownerStaff, _ := sessionOf(ownerRes)
	if ownerStaff != "" {
		t.Fatal("جلسةُ صاحب المتجر وُسمت موظّفاً")
	}

	staffRes, err := svc.LoginPassword(merchantApp, staffPhone, plain, "ua", "2.2.2.2")
	if err != nil {
		t.Fatalf("دخولُ الموظّف: %v", err)
	}
	staffSID, staffID, sub := sessionOf(staffRes)
	if staffID != st.ID || sub != owner {
		t.Fatalf("الموظّفُ لم يدخل باسم المتجر: staff=%q sub=%q", staffID, sub)
	}
	if !alive(ownerSID) {
		t.Fatal("دخولُ الموظّف أخرج صاحبَ المتجر")
	}

	// **وصاحبُ المتجر يدخل ثانيةً — فلا يُخرج موظّفَه.**
	if _, err := svc.LoginPassword(merchantApp, ownerPhone, "Owner@12345", "ua", "1.1.1.1"); err != nil {
		t.Fatal(err)
	}
	if !alive(staffSID) {
		t.Fatal("دخولُ صاحب المتجر أخرج الموظّف")
	}

	// **والتجديدُ يُبقي الموظّفَ موظّفاً.**
	ref, err := svc.Refresh(merchantApp, staffRes.Tokens.RefreshToken, "ua", "2.2.2.2")
	if err != nil {
		t.Fatalf("تجديدُ جلسة الموظّف: %v", err)
	}
	if _, s2, _ := sessionOf(ref); s2 != st.ID {
		t.Fatal("التجديدُ أسقط وسمَ الموظّف — فتنفتح له أبوابُ صاحب الحساب")
	}

	// **ودخولُه من جوالٍ ثانٍ يُزيح جلستَه الأولى وحدَها.**
	again, err := svc.LoginPassword(merchantApp, staffPhone, plain, "ua2", "3.3.3.3")
	if err != nil {
		t.Fatal(err)
	}
	if alive(staffSID) {
		t.Fatal("دخولُ الموظّف ثانيةً لم يُزح جلستَه الأولى")
	}
	againSID, _, _ := sessionOf(again)

	// **ولا يدخل تطبيقَ الزبون باسم المتجر.**
	if _, err := svc.LoginPassword(WithClient(ctx, "android-customer"), staffPhone, plain, "ua", "4.4.4.4"); err == nil {
		t.Fatal("الموظّفُ دخل تطبيقَ الزبون باسم المتجر")
	}

	// **وحذفُه يُخرجه بسببه.**
	if err := svc.RemoveStaff(ctx, owner, owner, st.ID, "1.1.1.1"); err != nil {
		t.Fatalf("حذفُ الموظّف: %v", err)
	}
	if alive(againSID) {
		t.Fatal("حُذف الموظّفُ وبقيت جلستُه حيّة")
	}
	if got := svc.RevokeReason(ctx, againSID); got != ReasonStaffRemoved {
		t.Fatalf("سببُ الخروج = %q", got)
	}
	if SessionEndError(ReasonStaffRemoved) != ErrSessionStaffRemoved {
		t.Fatal("خروجُ الموظّف المحذوف بلا سببٍ مسمّى")
	}
	if _, err := svc.LoginPassword(merchantApp, staffPhone, plain, "ua", "5.5.5.5"); err == nil {
		t.Fatal("موظّفٌ محذوفٌ دخل")
	}

	// **ورقمُ صاحبِ متجرٍ لا يصير موظّفاً.**
	if _, _, err := svc.AddStaff(ctx, owner, owner, ownerPhone, "", "1.1.1.1"); err != ErrStaffPhoneTaken {
		t.Fatalf("رقمُ صاحب متجرٍ قُبل موظّفاً: %v", err)
	}
}
