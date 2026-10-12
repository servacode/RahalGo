package orders_test

// ══════════════════════════════════════════════════════════════════════
// **أجرةُ المنصّة لا تُوثَّق ولا تُؤكَّد** (قرارُ المالك ٢٠٢٦-١٠-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// «وقت تكون المنصّة محدّدة السعر أصلاً ما في داعي لتوثيق سعر التوصيل… خطوة على
// الزبون والسائق». قِيس على #1028: أجرةٌ مفروضةٌ ١٥٠ وطُلب «وثّق» و«موافق».

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func fixedFeeOrder(t *testing.T, mode string, source string, mayChange bool) (*orders.Service, string, string, string) {
	t.Helper()
	svc, w, customer, driver, _ := cqSetup(t)
	var snap *int64
	if source == "admin_defined" {
		v := int64(150)
		snap = &v
	}
	orderID := cqSeedOrder(t, w, customer, driver, source, snap, mayChange)
	setMode(t, orderID, mode)
	if _, err := testdb.Pool(t).Exec(context.Background(),
		`UPDATE orders SET custom_request = 'ظرف أوراق من مكتبة النور' WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	svc.OpenCustomChatForTest(ctx, orderID, customer, driver)
	svc.AutoAgreeFixedFeeForTest(ctx, orderID, customer, driver)
	return svc, orderID, customer, driver
}

// TestFixedFee_AmanahReadyToPickUp **الأمانةُ بأجرة المنصّة: تُستلم فوراً بلا توثيقٍ ولا موافقة.**
func TestFixedFee_AmanahReadyToPickUp(t *testing.T) {
	svc, orderID, _, driver := fixedFeeOrder(t, orders.CustomModeAmanah, "admin_defined", false)
	ctx := context.Background()
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.CustomFee == nil || *o.CustomFee != 150 || o.Total != 150 || o.CashDue != 150 {
		t.Fatalf("الأجرةُ لم تُثبَّت: fee=%v total=%d cash_due=%d", o.CustomFee, o.Total, o.CashDue)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("الاستلامُ رُدّ والأجرةُ مفروضة: %v", err)
	}
	lines := strings.Join(chatLines(t, orderID), "\n")
	if strings.Contains(lines, "بانتظار موافقتك") || strings.Contains(lines, "وافقتُ على") ||
		strings.Contains(lines, "سوف تصلك") {
		t.Fatalf("رسائلُ توثيقٍ والأجرةُ مفروضة:\n%s", lines)
	}
	if !strings.Contains(lines, "في طريقي لاستلام أمانتك") {
		t.Fatalf("تحيّةُ الأمانة غائبة:\n%s", lines)
	}
}

// TestFixedFee_PurchaseStillDocumentsGoods **المشترياتُ بأجرة المنصّة: الأجرةُ ثابتة، وثمنُ البضاعة يبقى.**
func TestFixedFee_PurchaseStillDocumentsGoods(t *testing.T) {
	svc, orderID, customer, driver := fixedFeeOrder(t, orders.CustomModePurchase, "admin_defined", false)
	ctx := context.Background()
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); !errors.Is(err, orders.ErrGoodsPending) {
		t.Fatalf("اشترى قبل توثيق الثمن: %v", err)
	}
	if err := svc.AgreeCustomStep(ctx, orderID, driver, orders.AgreeStepGoods, 350, 0); err != nil {
		t.Fatalf("توثيقُ الثمن: %v", err)
	}
	lines := chatLines(t, orderID)
	last := lines[len(lines)-1]
	if last != "ثمن البضاعة: 350 ل.س\nأجرة التوصيل: 150 ل.س\nالمجموع: 500 ل.س — بانتظار موافقتك." {
		t.Fatalf("سطرُ الثمن: %q", last)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 500, 2); err != nil {
		t.Fatalf("موافقةُ الثمن: %v", err)
	}
	if _, err := svc.Transition(ctx, driver, []string{"driver"}, orderID, "picked_up", ""); err != nil {
		t.Fatalf("الشراءُ رُدّ بعد الموافقة: %v", err)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "في طريقي لشراء طلبك") {
		t.Fatalf("تحيّةُ المشتريات غائبة:\n%s", strings.Join(lines, "\n"))
	}
}

// TestFixedFee_DriverDefinedUnchanged **وأجرةُ السائق تبقى كما كانت** — توثيقٌ وموافقة.
func TestFixedFee_DriverDefinedUnchanged(t *testing.T) {
	svc, orderID, _, _ := fixedFeeOrder(t, orders.CustomModeAmanah, "driver_defined", true)
	o, err := svc.GetByID(context.Background(), orderID)
	if err != nil {
		t.Fatal(err)
	}
	if o.CustomFee != nil {
		t.Fatalf("أجرةُ السائق ثُبّتت وحدَها: %v", *o.CustomFee)
	}
	if !strings.Contains(strings.Join(chatLines(t, orderID), "\n"), "سوف تصلك تفاصيل السعر") {
		t.Fatalf("تحيّةُ أجرة السائق تغيّرت")
	}
}
