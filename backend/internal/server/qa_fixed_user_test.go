package server

// **شهادةُ العطب الحيّ (Batch 2, staging QA)**: تدخّلُ الأدمن وإسنادُ سائق QA
// كانا يسقطان بـ٤٠٤ بعد أوّل نداء في الشهادة الحيّة، وسجلُّ الخادم صامت.
//
// # العلّة
//
// `qaFixedUser` ينادي `EnsureUserWithRole(ctx, actorID="", …)` بلا شرط. على
// **حسابٍ قائم** يمرّ ذلك بـ`GrantRole(uid, role, &"")` — **سلسلةٌ خاويةٌ لعمودِ
// `granted_by` من نوع `uuid`** ⇒ خطأُ بوستغرس `22P02` (نصٌّ ليس معرّفاً). و`respondErr`
// **يصنّف `22P02` «غير موجود» (٤٠٤) بلا تسجيل** (auth_handlers.go) — فيبدو كأنّ
// الطلبَ غيرُ موجودٍ والسجلُّ نظيف.
//
// **وأوّلُ نداءٍ ينجح** لأنّ الحسابَ لم يوجد بعد فيُنشأ عبر `CreateUserWithRole`
// (لا `granted_by` فيه). فكلُّ نداءٍ تالٍ يسقط — وهو ما رُئي: تدخّلٌ أوّلُ نجح، ثمّ
// كلُّ تدخّلٍ وإسنادٍ بعده ٤٠٤.
//
// **الإصلاحُ ضيّقٌ في أداة QA وحدَها**: يُبحَث عن الحساب بهاتفه أوّلاً ولا يُنشأ
// إلّا مرّة — كما يفعل `handleQAStagingSession` أصلاً. **ولا يُمَسّ عقدُ الإنتاج.**

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestQAFixedUserIdempotent(t *testing.T) {
	pool := testdb.Pool(t)
	ctx := context.Background()
	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	ident := identity.NewService(identity.NewRepo(pool), nil, nil, nil, "", quiet)
	srv := &Server{pg: pool, logger: quiet, identity: ident}

	const phone = "+963900555556" // رقمُ اختبارٍ معزولٌ بعيدٌ عن أرقام QA الثابتة
	norm, _ := identity.NormalizePhone(phone)
	t.Cleanup(func() {
		bg := context.Background()
		_, _ = pool.Exec(bg, `DELETE FROM user_roles WHERE user_id IN (SELECT id FROM users WHERE phone=$1)`, norm)
		_, _ = pool.Exec(bg, `DELETE FROM users WHERE phone=$1`, norm)
	})

	// أوّلُ نداءٍ يُنشئ الحساب (CreateUserWithRole — لا granted_by).
	uid1, err := srv.qaFixedUser(ctx, phone, "driver", "سائق اختبار qaFixedUser", "127.0.0.1")
	if err != nil {
		t.Fatalf("النداءُ الأوّل (إنشاء) يجب أن ينجح: %v", err)
	}
	if uid1 == "" {
		t.Fatal("النداءُ الأوّل ردّ معرّفاً فارغاً")
	}

	// النداءُ الثاني على الحساب القائم: العطبُ يجعله 22P02 (⇒ ٤٠٤ في المعالِج).
	uid2, err := srv.qaFixedUser(ctx, phone, "driver", "سائق اختبار qaFixedUser", "127.0.0.1")
	if err != nil {
		t.Fatalf("النداءُ الثاني (حسابٌ قائم) يجب أن ينجح — العطبُ يجعله 22P02: %v", err)
	}
	if uid2 != uid1 {
		t.Fatalf("المعرّفُ يجب أن يكون نفسَه في النداءين: %s != %s", uid1, uid2)
	}

	// وثالثُ نداءٍ كذلك — التكرارُ لا يكسر.
	uid3, err := srv.qaFixedUser(ctx, phone, "driver", "سائق اختبار qaFixedUser", "127.0.0.1")
	if err != nil {
		t.Fatalf("النداءُ الثالث يجب أن ينجح: %v", err)
	}
	if uid3 != uid1 {
		t.Fatalf("المعرّفُ يجب أن يبقى ثابتاً: %s != %s", uid3, uid1)
	}
}
