package push

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestPush_UnclaimedSkipsSamePhone **هاتفٌ عاد فسجّل رمزَه لا يُحذَّر من «جهازٍ آخر»** —
// ورمزٌ لم يعد (هاتفٌ آخرُ أُزيح حقّاً) يُحذَّر (٢٠٢٦-١٠-١٩).
func TestPush_UnclaimedSkipsSamePhone(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	svc := New(pool, quietLogger())
	uid := testdb.NewUser(t, pool, "sales")

	if err := svc.Register(ctx, uid, "tok-same-phone", PlatformAndroid, "rep", "1.0"); err != nil {
		t.Fatalf("تسجيل: %v", err)
	}
	got := svc.Unclaimed(ctx, []string{"tok-same-phone", "tok-other-phone"})
	if len(got) != 1 || got[0] != "tok-other-phone" {
		t.Fatalf("المُحذَّرون = %v — والمتوقَّع الهاتفُ الآخرُ وحدَه", got)
	}
	if svc.Unclaimed(ctx, nil) != nil {
		t.Fatal("لا رموزَ ⇒ لا تحذير")
	}
}
