package referrals_test

// متى تُصرف مكافأةُ الدعوة — **وضعان لا واحد.**
//
// # القرار
//
// قرّر المالكُ (٢٠٢٦-٠٨-٠٥): «المكافأةُ برابط الدعوة عند التسجيل فقط وتوثيق
// واتساب، **وليس عند أوّل طلب، لأنّ هيك يحسّ الزبونُ أنّنا عم نضحك عليه**...
// ولكن رح نحطّ خيارين، زرٌّ ذكيّ».
//
// # ولماذا يُحرَس الوضعان
//
// **الخطرُ ليس في أيّهما أصحّ بل في أن يصرفا معاً.** حسبتان لصرفٍ واحدٍ
// تفترقان يوماً: **يُصرف عند التسجيل ثمّ يُصرف ثانيةً عند الطلب**، ولا يظهر
// في أيّ خطأ — **مالٌ يخرج مرّتين والدفترُ متوازن.**

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/servacode/rahalgo/backend/internal/referrals"
	"github.com/servacode/rahalgo/backend/internal/testdb"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// fakeSettings إعداداتٌ في الذاكرة — **والوضعُ هو المتغيّر الوحيد.**
type fakeSettings struct {
	mode   string
	reward int64
}

func (f fakeSettings) GetInt(context.Context, string) int64 { return f.reward }
func (f fakeSettings) GetString(_ context.Context, k string) string {
	if k == "referral.reward_on" {
		return f.mode
	}
	return ""
}

type quietNotifier struct{ bodies []string }

func (q *quietNotifier) NotifyWallet(_ context.Context, _, _, body, _ string) {
	q.bodies = append(q.bodies, body)
}

// refFixture داعٍ ومدعوٌّ ودعوةٌ معلّقة.
type refFixture struct {
	pool     *pgxpool.Pool
	svc      *referrals.Service
	inviter  string
	invitee  string
	treasury string
	notif    *quietNotifier
}

func arm(t *testing.T, mode string) *refFixture {
	t.Helper()
	ctx := context.Background()
	pool := testdb.Pool(t)

	f := &refFixture{pool: pool, notif: &quietNotifier{}}
	f.inviter = testdb.NewUser(t, pool, "customer")
	f.invitee = testdb.NewUser(t, pool, "customer")
	f.treasury = testdb.NewUser(t, pool, "admin")

	if _, err := pool.Exec(ctx, `
		INSERT INTO wallets (user_id, is_treasury) VALUES ($1, true)
		ON CONFLICT (user_id) DO UPDATE SET is_treasury = true`, f.treasury); err != nil {
		t.Fatalf("تعذّرت الخزينة: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`UPDATE wallets SET is_treasury = false, balance = 0 WHERE user_id = $1`, f.treasury)
	})

	// **دعوةٌ معلّقة** — نُسبت ولم تُصرف.
	if _, err := pool.Exec(ctx, `
		INSERT INTO referrals (inviter_id, invitee_id, rank)
		VALUES ($1, $2, 1)`, f.inviter, f.invitee); err != nil {
		t.Fatalf("تعذّرت الدعوة: %v", err)
	}

	f.svc = referrals.New(pool, wallet.NewService(pool),
		fakeSettings{mode: mode, reward: 5_000},
		func(context.Context) string { return f.treasury }, f.notif)
	return f
}

func (f *refFixture) balance(t *testing.T) int64 {
	t.Helper()
	var b int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT COALESCE(balance, 0) FROM wallets WHERE user_id = $1`, f.inviter).Scan(&b); err != nil {
		return 0
	}
	return b
}

// TestRewardOnSignup_PaysAtSignupNotAtFirstOrder **وضعُ التسجيل يصرف عنده.**
func TestRewardOnSignup_PaysAtSignupNotAtFirstOrder(t *testing.T) {
	f := arm(t, referrals.OnSignup)
	ctx := context.Background()

	f.svc.SettleOnSignup(ctx, f.invitee, f.inviter)
	if got := f.balance(t); got != 5_000 {
		t.Fatalf("رصيدُ الداعي %d والمتوقّع 5000 — **صرفٌ عند التسجيل لم يقع**", got)
	}

	// **ولا يُصرف ثانيةً عند الطلب** — والختمُ هو الحارس.
	f.svc.SettleFirstOrder(ctx, f.invitee, "", f.inviter)
	if got := f.balance(t); got != 5_000 {
		t.Fatalf("رصيدُ الداعي صار %d — **صُرفت المكافأةُ مرّتين والدفترُ متوازن**", got)
	}
}

// TestRewardOnFirstOrder_HoldsUntilTheOrder **ووضعُ الطلب لا يصرف عند التسجيل.**
func TestRewardOnFirstOrder_HoldsUntilTheOrder(t *testing.T) {
	f := arm(t, referrals.OnFirstOrder)
	ctx := context.Background()

	f.svc.SettleOnSignup(ctx, f.invitee, f.inviter)
	if got := f.balance(t); got != 0 {
		t.Fatalf("صُرف %d عند التسجيل والوضعُ «عند أوّل طلب» — **والوضعُ زينة**", got)
	}

	f.svc.SettleFirstOrder(ctx, f.invitee, "", f.inviter)
	if got := f.balance(t); got != 5_000 {
		t.Fatalf("رصيدُ الداعي %d والمتوقّع 5000 — **طلبٌ وقع ولا مكافأة**", got)
	}
}

// TestRewardOn_DefaultsToSignup **وإعدادٌ فارغٌ يعني التسجيل.**
//
// **ومنصةٌ جديدةٌ تحتاج الثقةَ أكثرَ ممّا تحتاج الحذر** — وافتراضٌ يمسك المالَ
// حتّى الطلب يجعل أوّلَ الداعين ينتظرون ما لا يملكون أن يفعلوه.
func TestRewardOn_DefaultsToSignup(t *testing.T) {
	f := arm(t, "")
	f.svc.SettleOnSignup(context.Background(), f.invitee, f.inviter)
	if got := f.balance(t); got != 5_000 {
		t.Fatalf("رصيدُ الداعي %d والمتوقّع 5000 — **والافتراضُ التسجيل**", got)
	}
}

// TestRewardOnSignup_BodyDoesNotClaimAnOrder **ولا يُقال له «طلب» ولم يطلب.**
//
// يقرؤها الداعي فيسأل صديقَه عن طلبٍ لم يقع، **ويظنّ أنّ في الحساب خللاً.**
func TestRewardOnSignup_BodyDoesNotClaimAnOrder(t *testing.T) {
	f := arm(t, referrals.OnSignup)
	f.svc.SettleOnSignup(context.Background(), f.invitee, f.inviter)

	if len(f.notif.bodies) == 0 {
		t.Fatal("لم يُبلَّغ الداعي — **ومكافأةٌ لا يراها صاحبُها لم تُصرف في نظره**")
	}
	if got := f.notif.bodies[0]; got != "صديقٌ دعوتَه أكمل تسجيلَه" {
		t.Fatalf("النصُّ %q — **وهو لم يطلب بعد**", got)
	}
}
