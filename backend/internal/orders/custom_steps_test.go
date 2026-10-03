package orders_test

// ══════════════════════════════════════════════════════════════════════
// **الطلبُ الخاصّ: أمانةٌ أم مشتريات — والتوثيقُ على خطوتين** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
// ══════════════════════════════════════════════════════════════════════
//
// «أوّلَ شي لازم يوثّق أجرةَ التوصيل… لأنّ السائقَ أوّلَ شي ما يعرف شقد سعرُ البضاعة».
// **وكلُّ اختبارٍ هنا يسقط على الشيفرة القديمة**: كان التوثيقُ خطوةً واحدةً بالخانتين.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func setMode(t *testing.T, orderID, mode string) {
	t.Helper()
	if _, err := testdb.Pool(t).Exec(context.Background(),
		`UPDATE orders SET custom_mode = $2 WHERE id = $1`, orderID, mode); err != nil {
		t.Fatal(err)
	}
}

func chatLines(t *testing.T, orderID string) []string {
	t.Helper()
	rows, err := testdb.Pool(t).Query(context.Background(),
		`SELECT body FROM order_messages WHERE order_id = $1 ORDER BY created_at`, orderID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var b string
		_ = rows.Scan(&b)
		out = append(out, b)
	}
	return out
}

// TestCustomSteps_PurchaseFeeThenGoods **المشتريات: الأجرةُ أوّلاً، ولا شراءَ قبل الثمن.**
func TestCustomSteps_PurchaseFeeThenGoods(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)

	if err := svc.AgreeCustomStep(ctx, orderID, driver, orders.AgreeStepFee, 0, 3_000); err != nil {
		t.Fatalf("توثيقُ الأجرة: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 3_000, 1); err != nil {
		t.Fatalf("موافقةُ الأجرة: %v", err)
	}
	// **ولا «تم شراء المطلوب» قبل توثيق الثمن.**
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); !errors.Is(err, orders.ErrGoodsPending) {
		t.Fatalf("اشترى قبل توثيق الثمن: %v", err)
	}
	if err := svc.AgreeCustomStep(ctx, orderID, driver, orders.AgreeStepGoods, 12_000, 0); err != nil {
		t.Fatalf("توثيقُ الثمن: %v", err)
	}
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.CustomGoodsPending || o.Total != 15_000 || o.QuoteVersion != 2 {
		t.Fatalf("(معلَّق=%v · مجموع=%d · نسخة=%d) — والمنتظَرُ (false · 15000 · 2)",
			o.CustomGoodsPending, o.Total, o.QuoteVersion)
	}
	// **والزيادةُ تُعيد التأكيد** — فلا شراءَ قبل موافقة الثمن أيضاً.
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); !errors.Is(err, orders.ErrQuoteNotConfirmed) {
		t.Fatalf("اشترى قبل موافقة الثمن: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 15_000, 2); err != nil {
		t.Fatalf("موافقةُ الثمن: %v", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("الشراءُ بعد الخطوتين: %v", err)
	}
	// **وفي الحديث ما وُثّق وما وُوفق عليه** — بمبالغه.
	lines := chatLines(t, orderID)
	want := []string{"أجرة التوصيل: 3٬000 ل.س — بانتظار موافقتك.", "وافقتُ على أجرة التوصيل: 3٬000 ل.س.",
		"ثمن البضاعة: 12٬000 ل.س — المجموع مع التوصيل 15٬000 ل.س. بانتظار موافقتك.", "وافقتُ على المجموع: 15٬000 ل.س."}
	for _, w := range want {
		found := false
		for _, l := range lines {
			if l == w {
				found = true
			}
		}
		if !found {
			t.Errorf("غاب من الحديث %q — وفيه %q", w, lines)
		}
	}
}

// TestCustomSteps_AmanahHasNoGoods **الأمانةُ بلا ثمنِ بضاعة** — الأجرةُ تكفي.
func TestCustomSteps_AmanahHasNoGoods(t *testing.T) {
	svc, w, customer, driver, _ := cqSetup(t)
	ctx := context.Background()
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	setMode(t, orderID, orders.CustomModeAmanah)

	if err := svc.AgreeCustomStep(ctx, orderID, driver, orders.AgreeStepFee, 0, 2_000); err != nil {
		t.Fatalf("توثيقُ الأجرة: %v", err)
	}
	if err := svc.AgreeCustomStep(ctx, orderID, driver, orders.AgreeStepGoods, 5_000, 0); !errors.Is(err, orders.ErrAgreeStep) {
		t.Fatalf("ثمنُ بضاعةٍ في أمانة: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 2_000, 1); err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("«استلمت الأمانة»: %v", err)
	}
	// **والأمانةُ تُستلم لا تُشترى** — والموافقةُ على الأجرة لا «المجموع» (دورةُ المحاكي ٢٠٢٦-١٠-٠٣).
	lines := chatLines(t, orderID)
	has := func(want string) bool {
		for _, l := range lines {
			if l == want {
				return true
			}
		}
		return false
	}
	if !has("استلمتُ الأمانة — في طريقي إليك.") || !has("وافقتُ على أجرة التوصيل: 2٬000 ل.س.") {
		t.Fatalf("سطورُ الأمانة في الحديث: %q", lines)
	}
}

// TestCustomSteps_CustomerSeesBuying **والزبونُ يرى «في طريقه للشراء» لا «نبحث عن سائق».**
func TestCustomSteps_CustomerSeesBuying(t *testing.T) {
	if got := orders.CustomStageOf(orders.StAssigned); got != orders.StageBuying {
		t.Fatalf("«أُسند» عند الزبون %q — والمنتظَر %q", got, orders.StageBuying)
	}
}

// TestStoreProblem_CustomerStaysPreparing **وتبديلُ المتجر لا يُرى رجوعاً** (٢٠٢٦-١٠-٠٣).
func TestStoreProblem_CustomerStaysPreparing(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	f.armTreasury(t)
	if _, err := officeStoreBlock(t, f.svc, f.pool, f.orderID, "merchant_closed"); err != nil {
		t.Fatalf("قرارُ المكتب: %v", err)
	}
	o, err := f.svc.GetByID(context.Background(), f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.Status != "accepted" || o.Stage != orders.StagePreparing {
		t.Fatalf("(%s · %s) — والزبونُ يبقى على «قيد التحضير»", o.Status, o.Stage)
	}
}

// TestStoreProblem_OfficeSaysCollect **«استلم الطلب» — الطلبُ يبقى معه عند المتجر.**
func TestStoreProblem_OfficeSaysCollect(t *testing.T) {
	f := setup(t, "at_pickup", 100_000, 10_000, 0)
	ops := testdb.NewUser(t, f.pool, "ops")
	if _, err := f.svc.ResolveDoor(context.Background(), ops, []string{"ops"}, f.orderID,
		orders.DoorResolution{Action: orders.DoorDeliverNow, Note: "المتجرُ فتح"}, nil); err != nil {
		t.Fatalf("«استلم الطلب»: %v", err)
	}
	if st := f.statusOf(t); st != "at_pickup" {
		t.Fatalf("الحالُ %q — **والطلبُ معه عند المتجر**", st)
	}
	if _, err := f.svc.Transition(context.Background(), f.driver, []string{"driver"}, f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلامُ بعد الأمر: %v", err)
	}
}

// TestMerchantDelivery_NoChatLines **ولا رسائلَ تلقائيّةً في «لدي توصيلة»** (٢٠٢٦-١٠-٠٣) —
// المستلمُ ليس على التطبيق. **قِيس**: ثلاثُ رسائلَ لزبونٍ لا وجودَ له (#1342).
func TestMerchantDelivery_NoChatLines(t *testing.T) {
	f := setup(t, "at_pickup", 0, 10_000, 0)
	ctx := context.Background()
	f.armTreasury(t)
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET kind = 'merchant_delivery', recipient_name = 'مستلم',
		       recipient_phone = '0999000000', fee_payer = 'merchant'
		WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, f.orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلام: %v", err)
	}
	if lines := chatLines(t, f.orderID); len(lines) != 0 {
		t.Fatalf("رسائلُ في توصيلة: %q", lines)
	}
}
