package orders_test

// حسمُ بضاعةِ طلبٍ فشل — **بالدفتر لا بالنيّة.**
//
// # ما يُثبت هنا
//
//	رُدّت وله رصيد    ←  يُسترجع كاملاً، ولا يبقى له منه شيء
//	رُدّت ولا رصيد     ←  يُخصم ما وُجد، والباقي دَينٌ عليه
//	مستحقٌّ لاحقٌ      ←  يُقتطع منه الدَّينُ حتى يُوفّى
//	دعمٌ منصوصٌ عليه   ←  يُدفع من الخزينة لا من أحد
//	إلى المكتب        ←  لا قيد؛ الخسارةُ مقيَّدةٌ منذ الاستلام
//	ضغطتان            ←  الثانيةُ تُردّ
//
// **وكلُّ رقمٍ يُقرأ من `wallet_transactions` و`wallets`** — لا من قيمةٍ
// تُرجعها الدالّة عن نفسها. **ودالّةٌ تشهد لنفسها لا تشهد.**
//
// # ولماذا في حزمة الطلبات لا الخادم
//
// **الخزينةُ واحدةٌ في القاعدة** — فهرسٌ فريدٌ يمنع ثانيةً. وحزمُ الاختبار
// تُشغَّل **متوازيةً**، فحزمتان تسمان خزينةً في اللحظة نفسها تصطدمان.
// **والخزينةُ تُوسَم في هذه الحزمة وحدَها** (`armTreasury`) — فتبقى المطالبةُ
// بيدٍ واحدة. (وقع فعلاً حين كُتبت هذه الاختبارات في `server`.)

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// goodsCase طلبٌ مشى مسارَه كاملاً حتى الفشل — **والمتجرُ قبض عند خروج
// البضاعة من يده**، وهو الحالُ الذي يُسترجع منه.
//
// **ولا يُقفز إلى `failed` مباشرةً**: القفزُ يترك الدفترَ فارغاً من مستحقّ
// المتجر، **فيمرّ اختبارُ استرجاعٍ لا يسترجع شيئاً.**
func goodsCase(t *testing.T) (f *fixture, owner, treasury string) {
	t.Helper()
	f = setup(t, "at_pickup", 100_000, 10_000, 0)
	owner, treasury = f.armTreasury(t)
	ctx := context.Background()
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, to, ""); err != nil {
			t.Fatalf("تعذّر الانتقالُ إلى %s: %v", to, err)
		}
	}
	if _, err := f.svc.TransitionWithReason(ctx, f.driver, []string{"driver"},
		f.orderID, "failed", "", "customer_absent"); err != nil {
		t.Fatalf("تعذّر الإفشال: %v", err)
	}
	// **٩٠٬٠٠٠ شراءً ناقصَ عمولةِ ١٠٪** — والاسترجاعُ يُقاس عليه.
	if got := f.merchantPosted(t); got != 81_000 {
		t.Fatalf("قُيّد للمتجر %d والمتوقّع 81000", got)
	}
	return f, owner, treasury
}

// setSetting يضبط مفتاحاً ويعيده بعد الاختبار — **والقاعدةُ مشتركة**، فمفتاحٌ
// يُترك على قيمةٍ يقرؤه اختبارٌ آخرُ فيسقط بسببٍ لا يخصّه.
func (f *fixture) setSetting(t *testing.T, key string, v any) {
	t.Helper()
	ctx := context.Background()
	var before *string
	_ = f.pool.QueryRow(ctx, `SELECT value::text FROM app_settings WHERE key = $1`, key).Scan(&before)
	if err := settings.NewStore(f.pool).SetInternal(ctx, key, v); err != nil {
		t.Fatalf("تعذّر ضبطُ %s: %v", key, err)
	}
	t.Cleanup(func() {
		c := context.Background()
		if before == nil {
			_, _ = f.pool.Exec(c, `DELETE FROM app_settings WHERE key = $1`, key)
			return
		}
		_, _ = f.pool.Exec(c,
			`UPDATE app_settings SET value = $2::jsonb WHERE key = $1`, key, *before)
	})
}

// merchantPosted صافي ما قُيّد للمتجر عن هذا الطلب — **موجبُه وسالبُه معاً.**
func (f *fixture) merchantPosted(t *testing.T) int64 {
	t.Helper()
	var sum int64
	if err := f.pool.QueryRow(context.Background(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'merchant_earning'`, f.orderID).Scan(&sum); err != nil {
		t.Fatalf("تعذّرت قراءةُ الدفتر: %v", err)
	}
	return sum
}

// merchantDebt ما بقي على المتجر من بضاعةٍ رُدّت ولم تحتملها محفظتُه.
func (f *fixture) merchantDebt(t *testing.T) int64 {
	t.Helper()
	var d int64
	if err := f.pool.QueryRow(context.Background(),
		`SELECT debt FROM merchants WHERE id = $1`, f.merchantID).Scan(&d); err != nil {
		t.Fatalf("تعذّرت قراءةُ الدَّين: %v", err)
	}
	return d
}

// TestGoods_ReturnedClawsBackWhatWasPaid **رُدّت البضاعةُ فعاد ثمنُها.**
//
// **وليس «قريباً منه»**: يُسترجع ما قُيّد بالضبط لا ما تحسبه المعادلةُ اليوم —
// **فنسبةٌ تُغيَّر بين الاستلام والحسم تجعل المسترجَعَ غيرَ المدفوع.**
func TestGoods_ReturnedClawsBackWhatWasPaid(t *testing.T) {
	f, owner, treasury := goodsCase(t)
	f.setSetting(t, "merchants.return_support_percent", 0)

	if err := f.svc.SettleGoods(context.Background(), f.orderID,
		orders.GoodsToMerchant, treasury); err != nil {
		t.Fatalf("تعذّر الحسم: %v", err)
	}

	if got := f.merchantPosted(t); got != 0 {
		t.Fatalf("بقي له %d من ثمن بضاعةٍ أخذها — **أخذ بضاعتَه وثمنَها معاً**", got)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("رصيدُ المتجر %d والمتوقّع صفرٌ بعد الاسترجاع", got)
	}
	if d := f.merchantDebt(t); d != 0 {
		t.Fatalf("قُيّد دَينٌ %d ورصيدُه كان يكفي", d)
	}

	var settledTo string
	var returned bool
	if err := f.pool.QueryRow(context.Background(),
		`SELECT goods_settled_to, returned_at IS NOT NULL FROM orders WHERE id = $1`,
		f.orderID).Scan(&settledTo, &returned); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if settledTo != orders.GoodsToMerchant || !returned {
		t.Fatalf("الأثرُ ناقص: settled=%q returned=%v", settledTo, returned)
	}
}

// TestGoods_ShortBalanceBecomesDebt **ما عجزت عنه المحفظةُ يُقيَّد دَيناً.**
//
// # ولماذا لا يُرفض الزرّ
//
// قيدُ القاعدة `balance >= 0` يرفض السالبَ **كلَّه لا جزأه**. فمتجرٌ سحب
// أموالَه **يُرفض الاسترجاعُ منه برمّته** — فتبقى البضاعةُ عنده وثمنُها في
// جيبه، **والموظّفُ يرى خطأً لا حيلةَ له فيه.**
//
// (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «يُخصم ما وُجد، والباقي دَينٌ يُستوفى من أوّل
// مستحقٍّ قادم».)
func TestGoods_ShortBalanceBecomesDebt(t *testing.T) {
	f, owner, treasury := goodsCase(t)
	f.setSetting(t, "merchants.return_support_percent", 0)
	ctx := context.Background()

	// **سحب مالَه قبل أن تُردّ البضاعة** — وهو الحالُ الواقع لا المفتعل.
	if _, err := f.pool.Exec(ctx,
		`UPDATE wallets SET balance = 1_000 WHERE user_id = $1`, owner); err != nil {
		t.Fatalf("تعذّر تفريغُ المحفظة: %v", err)
	}

	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury); err != nil {
		t.Fatalf("تعذّر الحسمُ ورصيدُه ناقص: %v — **والزرُّ لا يُرفض**", err)
	}

	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("بقي في محفظته %d — **ويُؤخذ ما وُجد كلُّه**", got)
	}
	if d := f.merchantDebt(t); d != 80_000 { // ٨١٬٠٠٠ ناقصَ ١٬٠٠٠ وُجدت
		t.Fatalf("الدَّينُ %d والمتوقّع 80000", d)
	}
}

// TestGoods_DebtIsTakenFromNextEarning **ويُستوفى من أوّل مستحقٍّ قادم.**
func TestGoods_DebtIsTakenFromNextEarning(t *testing.T) {
	f, owner, treasury := goodsCase(t)
	f.setSetting(t, "merchants.return_support_percent", 0)
	ctx := context.Background()

	if _, err := f.pool.Exec(ctx,
		`UPDATE wallets SET balance = 0 WHERE user_id = $1`, owner); err != nil {
		t.Fatalf("تعذّر تفريغُ المحفظة: %v", err)
	}
	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury); err != nil {
		t.Fatalf("تعذّر الحسم: %v", err)
	}
	if d := f.merchantDebt(t); d != 81_000 {
		t.Fatalf("الدَّينُ %d والمتوقّع 81000", d)
	}

	// **طلبٌ ثانٍ للمتجر نفسِه** — مستحقُّه ٩٬٠٠٠ (١٠٬٠٠٠ شراءً ناقصَ ١٠٪).
	second := f.anotherOrder(t, 10_000)
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"},
		second, "picked_up", ""); err != nil {
		t.Fatalf("تعذّر استلامُ الطلب الثاني: %v", err)
	}

	if d := f.merchantDebt(t); d != 72_000 {
		t.Fatalf("بقي الدَّينُ %d والمتوقّع 72000 بعد اقتطاع 9000", d)
	}
	if got := f.balance(t, owner); got != 0 {
		t.Fatalf("رصيدُه %d وكلُّ مستحقّه ذهب في الدَّين", got)
	}
	// **وسطران لا رقمٌ منقوص** — يرى ما أُعطي وما اقتُطع.
	var lines int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'merchant_earning'`, second).Scan(&lines); err != nil {
		t.Fatalf("تعذّرت قراءةُ الدفتر: %v", err)
	}
	if lines != 2 {
		t.Fatalf("قيدٌ واحدٌ (%d) — **ومتجرٌ يرى رقماً أصغرَ بلا سطرٍ يقول لماذا يظنّ أنّه غُبن**", lines)
	}
}

// anotherOrder طلبٌ ثانٍ للمتجر نفسِه عند باب المتجر — بسعرِ شراءٍ مُعطًى.
func (f *fixture) anotherOrder(t *testing.T, merchantPrice int64) string {
	t.Helper()
	ctx := context.Background()
	sale := merchantPrice + 2_000
	var id string
	if err := f.pool.QueryRow(ctx, `
		INSERT INTO orders (customer_id, merchant_id, driver_id, status, address_text, dropoff,
			payment_method, subtotal, delivery_fee, total, wallet_paid, cash_due)
		VALUES ($1, $2, $3, 'at_pickup', 'عنوان اختبار',
			ST_SetSRID(ST_MakePoint(39.0079, 35.9528), 4326)::geography,
			'cash', $4, 0, $4, 0, $4)
		RETURNING id`, f.customer, f.merchantID, f.driver, sale).Scan(&id); err != nil {
		t.Fatalf("تعذّر إنشاءُ طلبٍ ثانٍ: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, qty, options)
		VALUES ($1, 'صنفُ اختبار', $2, $3, 1, '[]'::jsonb)`, id, sale, merchantPrice); err != nil {
		t.Fatalf("تعذّر إنشاءُ بندٍ ثانٍ: %v", err)
	}
	return id
}

// TestGoods_SupportIsPaidFromTreasury **والدعمُ من الخزينة لا من أحد.**
func TestGoods_SupportIsPaidFromTreasury(t *testing.T) {
	f, owner, treasury := goodsCase(t)
	f.setSetting(t, "merchants.return_support_percent", 20)
	ctx := context.Background()

	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury); err != nil {
		t.Fatalf("تعذّر الحسم: %v", err)
	}

	var support int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'compensation' AND user_id = $2`,
		f.orderID, owner).Scan(&support); err != nil {
		t.Fatalf("تعذّرت قراءةُ الدعم: %v", err)
	}
	if support != 16_200 { // ٢٠٪ من ٨١٬٠٠٠
		t.Fatalf("الدعمُ %d والمتوقّع 16200 — **نسبةٌ ممّا استُرجع لا من سعر البضاعة**", support)
	}
	// **ولا يُعطى أكثرَ من بيعةٍ ناجحة**: أخذ بضاعتَه وبقي معه الدعمُ وحدَه.
	if got := f.balance(t, owner); got != support {
		t.Fatalf("بقي معه %d والمتوقّع %d — الدعمُ وحدَه", got, support)
	}
	// **ونفقةٌ تخرج من الخزينة** — دعمٌ يُقيَّد للمتجر وحدَه يجعل المنصةَ
	// تظهر رابحةً وهي تدفع.
	var expense int64
	if err := f.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		WHERE ref = $1 AND kind = 'platform_expense' AND note LIKE 'دعمُ متجر%'`,
		f.orderID).Scan(&expense); err != nil {
		t.Fatalf("تعذّرت قراءةُ نفقة الخزينة: %v", err)
	}
	if expense != -support {
		t.Fatalf("نفقةُ الخزينة %d والمتوقّع %d", expense, -support)
	}
}

// TestGoods_ToOfficeChangesNoMoney **إلى المكتب: لا قيد.**
//
// المالُ عند المتجر منذ خروج بضاعته والخزينةُ خصمته حينها — **والخسارةُ
// مقيَّدةٌ قبل أن يُضغط الزرّ.** وما يفعله الزرُّ أن يقول «انتهى أمرُ هذه
// البضاعة» فلا تبقى معلّقةً في السجلّ.
func TestGoods_ToOfficeChangesNoMoney(t *testing.T) {
	f, owner, treasury := goodsCase(t)
	ctx := context.Background()
	before := f.balance(t, owner)

	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToOffice, treasury); err != nil {
		t.Fatalf("تعذّر الحسم: %v", err)
	}

	if got := f.balance(t, owner); got != before {
		t.Fatalf("تحرّك رصيدُ المتجر من %d إلى %d — **وماله ثبت منذ خروج بضاعته**", before, got)
	}
	if got := f.merchantPosted(t); got != 81_000 {
		t.Fatalf("قيدُ المتجر صار %d — **ولا يُسترجع منه شيء**", got)
	}
	var settledTo string
	var returned *string
	if err := f.pool.QueryRow(ctx,
		`SELECT goods_settled_to, returned_at::text FROM orders WHERE id = $1`,
		f.orderID).Scan(&settledTo, &returned); err != nil {
		t.Fatalf("تعذّرت قراءةُ الطلب: %v", err)
	}
	if settledTo != orders.GoodsToOffice {
		t.Fatalf("الحسمُ %q لا %q", settledTo, orders.GoodsToOffice)
	}
	// **ولا تاريخَ إرجاعٍ لما لم يُردّ** — تاريخٌ يُكتب بلا واقعةٍ يُقرأ واقعة.
	if returned != nil {
		t.Fatalf("كُتب تاريخُ إرجاعٍ %q ولم تُردّ البضاعة", *returned)
	}
}

// TestGoods_SettledTwiceIsRejected **وضغطتان تسترجعان الثمنَ مرّتين.**
func TestGoods_SettledTwiceIsRejected(t *testing.T) {
	f, _, treasury := goodsCase(t)
	f.setSetting(t, "merchants.return_support_percent", 0)
	ctx := context.Background()

	if err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury); err != nil {
		t.Fatalf("تعذّر الحسمُ الأوّل: %v", err)
	}
	err := f.svc.SettleGoods(ctx, f.orderID, orders.GoodsToMerchant, treasury)
	if !errors.Is(err, orders.ErrGoodsAlreadySettled) {
		t.Fatalf("مرّ الحسمُ ثانيةً (%v) — **والثمنُ يُسترجع مرّتين**", err)
	}
}

// TestGoods_NotFailedIsRejected **ولا حسمَ لبضاعةِ طلبٍ لم يفشل.**
func TestGoods_NotFailedIsRejected(t *testing.T) {
	f := setup(t, "at_dropoff", 100_000, 10_000, 0)
	_, treasury := f.armTreasury(t)

	err := f.svc.SettleGoods(context.Background(), f.orderID,
		orders.GoodsToMerchant, treasury)
	if !errors.Is(err, orders.ErrGoodsNotFailed) {
		t.Fatalf("حُسمت بضاعةُ طلبٍ قائم (%v)", err)
	}
}
