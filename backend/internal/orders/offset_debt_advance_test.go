package orders_test

// **سائقٌ عليه سلفةٌ ودينٌ معاً لا يُمنع من التسليم** (بلاغُ الإنتاج ٢٠٢٦-١١-٠٧، الطلب ١٢٤٤).
//
// كانت الأجرةُ تُقيَّد فيُنقص المُطلِقُ `advance_owed` بقدرها، ثمّ يُقتطع الدينُ منها
// فينزل الرصيدُ تحت الحدّ الجديد ويسقط قيدُ `wallets_balance_check` — فيُردّ التسليمُ
// كلُّه بـ`insufficient_balance`. **والآن لا يُقتطع إلّا ما يحتمله الرصيدُ الموجب.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

func TestOffsetDebt_DriverWithAdvanceCanDeliver(t *testing.T) {
	for _, tc := range []struct {
		name    string
		advance int64
	}{
		{"advance larger than fee", 1_000_000},
		{"fee covers advance", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			f := setup(t, "at_pickup", 100_000, 10_000, 0)
			t.Cleanup(func() {
				_, _ = f.pool.Exec(context.Background(),
					`DELETE FROM financial_obligations WHERE party_kind = 'driver' AND party_id = $1`, f.driver)
			})
			if _, err := f.pool.Exec(ctx, `
				INSERT INTO wallets (user_id, balance, advance_owed) VALUES ($1, $2, $3)
				ON CONFLICT (user_id) DO UPDATE SET balance = $2, advance_owed = $3, reserved = 0`,
				f.driver, -tc.advance, tc.advance); err != nil {
				t.Fatalf("زرعُ السلفة: %v", err)
			}
			const debt = 50_000_000
			if _, err := obligations.Create(ctx, f.pool, obligations.PartyDriver, f.driver, debt,
				obligations.CauseLegacy, "", nil); err != nil {
				t.Fatalf("زرعُ الدين: %v", err)
			}
			for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
				if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
					t.Fatalf("الانتقالُ إلى %s رُدّ: %v", to, err)
				}
			}
			bal := f.balance(t, f.driver)
			left, err := obligations.Balance(ctx, f.pool, obligations.PartyDriver, f.driver)
			if err != nil {
				t.Fatal(err)
			}
			if tc.advance > 1 {
				// الأجرةُ كلُّها تسدّ السلفة، والدينُ باقٍ كما هو.
				if bal >= 0 || bal <= -tc.advance {
					t.Fatalf("الرصيد %d — يُنتظر بين −%d والصفر", bal, tc.advance)
				}
				if left != debt {
					t.Fatalf("الدين صار %d — لا يُقتطع منه والرصيدُ سالب", left)
				}
				return
			}
			// الأجرةُ تسدّ ليرةَ السلفة، والباقي كلُّه للدين.
			if bal != 0 {
				t.Fatalf("الرصيد %d — يُنتظر صفر", bal)
			}
			if left >= debt {
				t.Fatalf("الدين %d لم ينقص", left)
			}
		})
	}
}
