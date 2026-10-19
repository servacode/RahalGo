package identity

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// TestSessionReason_LatestMeaningfulReasonWins **السببُ هو آخرُ إنهاءٍ لا التدوير** (قرارُ
// المالك ٢٠٢٦-١٠-١٩): جلسةٌ دُوّرت مرّاتٍ ثمّ أُعيدت كلمتُها ⇒ «انغيّرت كلمة السرّ».
// كانت `max(revoked_reason)` تختار أبجديّاً فتردّ «rotated» أو «superseded» لغير سببها.
func TestSessionReason_LatestMeaningfulReasonWins(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	sid, err := repo.StoreRefresh(ctx, uid, "r-1", testSessionTTL, "هاتف", "2.2.2.2", "", "android-driver")
	if err != nil {
		t.Fatalf("جلسة: %v", err)
	}
	if _, _, err := repo.RevokeRefresh(ctx, "r-1"); err != nil {
		t.Fatalf("تدوير: %v", err)
	}
	if _, err := repo.StoreRefresh(ctx, uid, "r-2", testSessionTTL, "هاتف", "2.2.2.2", sid, "android-driver"); err != nil {
		t.Fatalf("خلف: %v", err)
	}
	if got, _ := repo.RevokedReasonOfSession(ctx, sid); got != "" {
		t.Fatalf("جلسةٌ حيّةٌ دُوّرت لها سببُ إنهاء %q", got)
	}
	if _, err := repo.RevokeAllTokens(ctx, uid, ReasonPasswordReset); err != nil {
		t.Fatalf("إبطال: %v", err)
	}
	if got, _ := repo.RevokedReasonOfSession(ctx, sid); got != ReasonPasswordReset {
		t.Fatalf("السبب = %q، والمتوقَّع %q", got, ReasonPasswordReset)
	}
	if e := SessionEndError(ReasonPasswordReset); e != ErrSessionPasswordChanged {
		t.Fatal("إعادةُ الكلمة لا تُسمّى للجهاز")
	}
	if e := SessionEndError(ReasonLogout); e != nil {
		t.Fatal("خروجٌ يدويٌّ سُمّي سبباً")
	}
}

// TestSessionReason_RecoverLostRotation **انقطع النتُّ بعد التدوير** — الجهازُ عاد بالرمز
// القديم والجلسةُ حيّة ⇒ يُسترَدّ ولا يُخرَج. **وجلسةٌ أُنهيت لا تُسترَدّ.**
func TestSessionReason_RecoverLostRotation(t *testing.T) {
	pool := testdb.Pool(t)
	repo := NewRepo(pool)
	ctx := context.Background()
	uid := testdb.NewUser(t, pool, "driver")

	sid, err := repo.StoreRefresh(ctx, uid, "g-1", testSessionTTL, "هاتف", "2.2.2.2", "", "android-driver")
	if err != nil {
		t.Fatalf("جلسة: %v", err)
	}
	if _, _, err := repo.RevokeRefresh(ctx, "g-1"); err != nil {
		t.Fatalf("تدوير: %v", err)
	}
	if _, err := repo.StoreRefresh(ctx, uid, "g-2", testSessionTTL, "هاتف", "2.2.2.2", sid, "android-driver"); err != nil {
		t.Fatalf("خلف: %v", err)
	}

	gotUID, gotSID, ok, err := repo.RecoverRotated(ctx, "g-1")
	if err != nil || !ok || gotUID != uid || gotSID != sid {
		t.Fatalf("تدويرٌ ضاع جوابُه لم يُسترَدّ: ok=%v err=%v", ok, err)
	}
	// **والخلفُ الضائعُ أُبطل** — فلا يعيش رمزان.
	if _, _, err := repo.RevokeRefresh(ctx, "g-2"); err == nil {
		t.Fatal("الخلفُ الضائعُ بقي حيّاً بعد الاسترداد")
	}
	// **ورمزٌ أقدمُ من الجلسة الحيّة نفسِها يُسترَدّ أيضاً** — الجهازُ حفظ رمزاً فوق أحدث منه.
	if _, err := repo.StoreRefresh(ctx, uid, "g-3", testSessionTTL, "هاتف", "2.2.2.2", sid, "android-driver"); err != nil {
		t.Fatalf("خلف: %v", err)
	}
	if _, _, ok, _ := repo.RecoverRotated(ctx, "g-1"); !ok {
		t.Fatal("رمزٌ أقدمُ من جلسةٍ حيّةٍ طُرد")
	}
	// **وما لا يُعرف لا يُسترَدّ.**
	if _, _, ok, _ := repo.RecoverRotated(ctx, "no-such"); ok {
		t.Fatal("رمزٌ مجهولٌ استُرِدّ")
	}

	// **وجلسةٌ أُنهيت لا تُسترَدّ بالرمز القديم.**
	sid2, _ := repo.StoreRefresh(ctx, uid, "h-1", testSessionTTL, "هاتف", "2.2.2.2", "", "android-customer")
	_, _, _ = repo.RevokeRefresh(ctx, "h-1")
	_, _ = repo.StoreRefresh(ctx, uid, "h-2", testSessionTTL, "هاتف", "2.2.2.2", sid2, "android-customer")
	if _, err := repo.RevokeAllTokens(ctx, uid, ReasonAdminLogout); err != nil {
		t.Fatalf("إبطال: %v", err)
	}
	if _, _, ok, _ := repo.RecoverRotated(ctx, "h-1"); ok {
		t.Fatal("جلسةٌ أخرجتها الإدارةُ عادت بالرمز القديم")
	}
	if got, _ := repo.RevokedReasonOfToken(ctx, "h-2"); got != ReasonAdminLogout {
		t.Fatalf("سببُ الخلف = %q", got)
	}
}
