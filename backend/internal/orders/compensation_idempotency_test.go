package orders_test

// **تعويضُ السائق التلقائيُّ مرّةً واحدةً لكلّ (طلب، سائق)** — دورةُ ٢٠٢٦-٠٩-٢٧.
//
// **العلّة**: حظرُ المتجر (`merchant_blocked.go`) يعيد الطلبَ إلى `accepted`
// (حالةٌ غيرُ منتهية) ويُعوّض السائق، **فإعادةُ توزيعِه ثمّ حظرُه ثانيةً على
// المتجر نفسِه كانت تُعوّض السائقَ نفسَه مرّتين** والخزينةُ تُخصَم مرّتين —
// **والمسارُ التلقائيُّ بلا حارس** خلافاً للمسار اليدويّ. أُضيف حارسُ
// `EXISTS(ref,kind='compensation',user)` في `compensateDriverOnFail`.
//
// **والوحدةُ (طلب، سائق) لا (طلب)**: سائقان مختلفان قادا مشوارَهما وحُظرا
// يستحقّان تعويضَين — فصار ثابتُ `FI-05.a` يجمع بـ(ref,user_id).

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func compRows(t *testing.T, f *fixture, orderID, userID string) int64 {
	t.Helper()
	var n int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'compensation' AND user_id = $2`,
		orderID, userID).Scan(&n); err != nil {
		t.Fatalf("عدُّ التعويضات: %v", err)
	}
	return n
}

func assertFI05aGreen(t *testing.T, f *fixture) {
	t.Helper()
	v, err := fininv.Run(context.Background(), f.pool, "FI-05.a")
	if err != nil {
		t.Fatalf("تعذّر تشغيلُ FI-05.a: %v", err)
	}
	if len(v) != 0 {
		t.Fatalf("خُرق FI-05.a: %+v", v)
	}
}

// TestCompensation_MerchantBlockedTwiceSameDriver_CompensatesOnce
// **حظرٌ مرّتين للسائق نفسِه ⇒ تعويضٌ واحد** (الحارسُ الجديد).
func TestCompensation_MerchantBlockedTwiceSameDriver_CompensatesOnce(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)

	// ── الحظرُ الأوّل ──────────────────────────────────────────────
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("الحظرُ الأوّل فشل: %v", err)
	}
	comp1 := f.balance(t, f.driver)
	if comp1 != 5000 { // نصفُ رسم التوصيل ١٠٬٠٠٠
		t.Fatalf("التعويضُ الأوّل = %d، والمتوقّع 5000", comp1)
	}
	if n := compRows(t, f, f.orderID, f.driver); n != 1 {
		t.Fatalf("صفوفُ التعويض بعد الحظر الأوّل = %d، والمتوقّع 1", n)
	}

	// ── يُعاد توزيعُ الطلب إلى السائق نفسِه ويُحظَر ثانيةً ──────────
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET status = 'at_pickup', driver_id = $2 WHERE id = $1`,
		f.orderID, f.driver); err != nil {
		t.Fatalf("إعادةُ التوزيع: %v", err)
	}
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("الحظرُ الثاني فشل: %v", err)
	}

	// **لا تعويضَ ثانٍ** — الرصيدُ كما هو، وصفٌّ واحد.
	if comp2 := f.balance(t, f.driver); comp2 != comp1 {
		t.Fatalf("عُوِّض السائقُ مرّتين: %d ← %d", comp1, comp2)
	}
	if n := compRows(t, f, f.orderID, f.driver); n != 1 {
		t.Fatalf("تعويضٌ مكرَّرٌ للسائق نفسِه: %d صفّاً", n)
	}
	assertFI05aGreen(t, f)
}

// TestCompensation_MerchantBlockedDifferentDrivers_EachCompensatedOnce
// **سائقان مختلفان ⇒ تعويضان مشروعان، وFI-05.a أخضر** (حبيبةُ (طلب،سائق)).
func TestCompensation_MerchantBlockedDifferentDrivers_EachCompensatedOnce(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	driverB := testdb.NewUser(t, f.pool, "driver")

	// السائقُ أ يُحظَر
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("حظرُ أ: %v", err)
	}
	// يُعاد توزيعُ الطلب إلى السائق ب
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET status = 'at_pickup', driver_id = $2 WHERE id = $1`,
		f.orderID, driverB); err != nil {
		t.Fatalf("إعادةُ التوزيع إلى ب: %v", err)
	}
	if _, err := f.svc.TransitionWithReason(ctx, driverB, []string{"driver"},
		f.orderID, "failed", "", "merchant_refused"); err != nil {
		t.Fatalf("حظرُ ب: %v", err)
	}

	// **كلٌّ عُوِّض عن مشوارِه مرّةً** — صفٌّ لكلّ سائق.
	if a := f.balance(t, f.driver); a != 5000 {
		t.Fatalf("تعويضُ أ = %d، والمتوقّع 5000", a)
	}
	if b := f.balance(t, driverB); b != 5000 {
		t.Fatalf("تعويضُ ب = %d، والمتوقّع 5000", b)
	}
	if n := compRows(t, f, f.orderID, f.driver); n != 1 {
		t.Fatalf("تعويضُ أ = %d صفّاً", n)
	}
	if n := compRows(t, f, f.orderID, driverB); n != 1 {
		t.Fatalf("تعويضُ ب = %d صفّاً", n)
	}
	// **وFI-05.a أخضرُ رغمَ صفَّي تعويضٍ لطلبٍ واحد** — الحبيبةُ (ref,user).
	assertFI05aGreen(t, f)
}
