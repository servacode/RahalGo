package referrals_test

// **هديّةُ الحساب الجديد تصل المحفظة فعلاً.**
//
// (شكوى المالك ٢٠٢٦-٠٨-١٨: «مسحنا جميع الحسابات، وبالرغم من ذلك عند
//
//	تسجيل حسابٍ جديدٍ لنفس الرقم لم يحصل على المكافأة».)
//
// # ولماذا لم تُصرف قطّ
//
// **الوسيطُ نفسُه لعمودين من نوعين** في فحص التكرار: `user_id` من نوع
// uuid و`ref` نصّ — **فتردّ بوستغرس «لا مُعامِلَ بين نصٍّ وuuid»**،
// **وكان الخطأُ يُخلط بجواب «صُرفت سلفاً» في سطرٍ واحد** فتخرج الدالّة
// صامتة.
//
// # وما يُقاس
//
// **الرصيدُ بعدها** — لا أنّها «لم تسقط»: **دالّةٌ تخرج صامتةً لا
// تسقط**، وهو بعينه ما وقع شهرا.
//
// **ويُقاس الطرفان**: يُقيَّد للزبون ويُخصم من الخزينة. **ودفترٌ يأخذ
// من طرفٍ ولا يعطي آخرَ لا يتوازن.**

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/servacode/rahalgo/backend/internal/referrals"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

func TestGrantSignupBonus_Credits(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL غير مضبوط")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatalf("تعذّر الاتّصال: %v", err)
	}
	t.Cleanup(db.Close)

	const phone = "+963900888111"
	const treasuryPhone = "+963900888112"
	var customer, treasury string

	clean := func() {
		for _, p := range []string{phone, treasuryPhone} {
			_, _ = db.Exec(ctx, `DELETE FROM phone_claims WHERE phone_hash =
				encode(sha256(((SELECT value FROM app_secrets WHERE key='phone_pepper')
				               || $1::text)::bytea),'hex')`, p)
			_, _ = db.Exec(ctx, `DELETE FROM users WHERE phone = $1`, p)
		}
	}
	clean()
	t.Cleanup(clean)

	mk := func(p string) string {
		var id string
		if err := db.QueryRow(ctx, `
			INSERT INTO users (phone, full_name, status)
			VALUES ($1, 'فحصُ الهديّة', 'active') RETURNING id::text`, p).Scan(&id); err != nil {
			t.Skipf("تعذّر إنشاءُ حساب: %v", err)
		}
		return id
	}
	customer = mk(phone)
	treasury = mk(treasuryPhone)

	// **والخزينةُ تُوسَم** — **والقيدُ عليها سالبٌ دائماً**، والقاعدةُ
	// تمنع السالبَ إلّا لها (`balance >= 0 OR is_treasury`).
	//
	// **وبلا وسمٍ يُردّ القيدُ بـ`insufficient_balance`** — وهو ما وقع
	// في أوّل تشغيلةٍ لهذا الاختبار، **وهو خطرٌ حقيقيٌّ في منصّةٍ
	// جديدةٍ لم تُوسَم خزينتُها بعد.**
	if _, err := db.Exec(ctx, `
		INSERT INTO wallets (user_id, is_treasury) VALUES ($1::uuid, true)
		ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, treasury); err != nil {
		t.Skipf("تعذّر وسمُ الخزينة: %v", err)
	}

	const amount = 15
	svc := referrals.New(
		db, wallet.NewService(db), fakeSettings{perKey: map[string]int64{"customers.signup_bonus": amount}},
		func(context.Context) string { return treasury },
		nil, slog.New(slog.NewTextHandler(os.Stderr, nil)),
	)

	svc.GrantSignupBonus(ctx, customer, customer)

	var got, treasuryBal int64
	_ = db.QueryRow(ctx, `SELECT COALESCE(balance,0) FROM wallets WHERE user_id = $1::uuid`,
		customer).Scan(&got)
	_ = db.QueryRow(ctx, `SELECT COALESCE(balance,0) FROM wallets WHERE user_id = $1::uuid`,
		treasury).Scan(&treasuryBal)

	if got != amount {
		t.Fatalf("رصيدُ الزبون %d والمنتظَر %d — **وهذا ما شكا منه المالك**", got, amount)
	}
	if treasuryBal != -amount {
		t.Errorf("الخزينة %d والمنتظَر %d — **ودفترٌ لا يتوازن**", treasuryBal, -amount)
	}

	// ── ولا تُصرف مرّتين ────────────────────────────────────────────
	svc.GrantSignupBonus(ctx, customer, customer)
	_ = db.QueryRow(ctx, `SELECT balance FROM wallets WHERE user_id = $1::uuid`,
		customer).Scan(&got)
	if got != amount {
		t.Errorf("صُرفت مرّتين: %d — **ومن نادى الدالّةَ ثانيةً ضاعفها**", got)
	}
}
