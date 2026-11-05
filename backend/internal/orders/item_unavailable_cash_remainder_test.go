package orders_test

// ══════════════════════════════════════════════════════════════════════
// **صنفٌ غيرُ متوفّر في طلب محفظةٍ له باقٍ نقديّ** — خلل PD-2 (٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// **تعديلُ الطلب يترك نقداً على طلب المحفظة** حين لا يكفي الرصيدُ الزيادة (`order_edit.go`).
// **وكان حذفُ الصنف يكتب «كلُّه من المحفظة»** لكلّ طلب محفظة (`wallet_paid = total`، `cash_due = 0`)
// — فيضيع الباقي النقديّ: لا يقبضه السائق، **والمحفظةُ تقول دُفع ما لم يُدفع.**
//
// **وصار بقاعدة تعديل الطلب نفسِها**: المحفظةُ لا تزيد على ما دُفع منها، ويُردّ ما زاد على المجموع
// الجديد، **والباقي نقد.**

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestItemUnavailableKeepsCashRemainder **T0b — طلبُ محفظةٍ بـ١٠٠٠ زِيد عليه ٥٠٠ والرصيدُ ٢٠٠
// (المحفظةُ ١٢٠٠ والنقدُ ٣٠٠)، ثمّ صنفٌ بـ٢٠٠ غيرُ متوفّر ⇒ المجموعُ ١٣٠٠: المحفظةُ ١٢٠٠ كما
// دُفعت والنقدُ ١٠٠ ولا استرجاع** — لا ١٣٠٠ من المحفظة وصفرُ نقد. **ثمّ يُسلَّم والدفترُ أخضر.**
func TestItemUnavailableKeepsCashRemainder(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	base, err := fininv.Capture(ctx, f.pool)
	if err != nil {
		t.Fatalf("تصويرُ الأساس: %v", err)
	}
	m, _ := f.merchant(t, "cash", 10)
	big := editMenuItem(t, f, m, "إضافة كبيرة", 300)
	small := editMenuItem(t, f, m, "إضافة صغيرة", 200)
	oid := f.orderWalletPaid(t, m, 800, 200) // ١٠٠٠ كلُّها من المحفظة
	if _, err := f.wallet.Apply(ctx, f.customer, 200, "topup", "seed2-"+oid, "رصيد", nil); err != nil {
		t.Fatal(err)
	}

	// **قبل الاستلام**: +٥٠٠ والرصيدُ ٢٠٠ ⇒ يُخصم ٢٠٠ والباقي ٣٠٠ نقد.
	if _, res, err := f.svc.EditOrder(ctx, f.treasury, oid, orders.EditRequest{Reason: "زيادة",
		Add: []orders.EditAdd{{MenuItemID: big, Qty: 1}, {MenuItemID: small, Qty: 1}}}, false, nil); err != nil {
		t.Fatalf("التعديل: %v", err)
	} else if res.Charged != 200 || res.CashDue != 300 || res.NewTotal != 1_500 {
		t.Fatalf("التعديل: %+v — يُنتظر خصم 200 ونقد 300 ومجموع 1500", res)
	}

	line := f.exec(t, `SELECT id::text FROM order_items WHERE order_id = $1 AND menu_item_id = $2`, oid, small)
	_, res, err := f.svc.MarkItemUnavailable(ctx, f.treasury, oid, line, 0, nil)
	if err != nil {
		t.Fatalf("الحذف: %v", err)
	}
	if res.NewTotal != 1_300 || res.Refunded != 0 {
		t.Fatalf("الحذف: %+v — يُنتظر مجموع 1300 بلا استرجاع", res)
	}
	if total, cash, wp := f.orderRow(t, oid); total != 1_300 || wp != 1_200 || cash != 100 {
		t.Fatalf("الطلب: مجموع %d محفظة %d نقد %d — يُنتظر 1300 و1200 و100 (الباقي النقديّ لا يضيع)",
			total, wp, cash)
	}
	if got := f.sumKind(t, oid, "refund"); got != 0 {
		t.Fatalf("استرجاعٌ %d — والمحفظةُ لم تدفع أكثرَ من المجموع الجديد", got)
	}
	fresh := func(when string) {
		t.Helper()
		after, err := fininv.Run(ctx, f.pool)
		if err != nil {
			t.Fatalf("%s: تشغيلُ الثوابت: %v", when, err)
		}
		for _, v := range base.New(after) {
			t.Errorf("%s: خرقُ ثابتٍ ماليّ: %s", when, v)
		}
	}
	fresh("بعد الحذف") // FI-06.d: الدفترُ −١٢٠٠ = −wallet_paid

	f.deliver(t, oid)
	fresh("بعد التسليم")
}
