package orders_test

// ══════════════════════════════════════════════════════════════════════
// **تعديلُ الطلب العاديّ — قبل الاستلام** (قرارُ المالك ٢٠٢٦-١١-٠٣، المرحلة ١)
// ══════════════════════════════════════════════════════════════════════
//
// **يُقاس ما يقع**: البنودُ والمجموعُ والنقدُ وردُّ المحفظة وخصمُها وما لا يكفيه
// رصيدُها، والحدثُ والإشعار، **والمعاينةُ التي لا تترك أثراً**، ثمّ استلامٌ وتسليمٌ
// على طلبٍ عُدّل وثوابتُ الدفتر خضراء.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/fininv"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

// editMenuItem **صنفٌ في قائمة متجر** — بهامشٍ صفريٍّ صريحٍ فيُقرأ سعرُه بلا افتراض.
func editMenuItem(t *testing.T, f interface {
	exec(t *testing.T, sql string, args ...any) string
}, merchantID, name string, price int64) string {
	t.Helper()
	sec := f.exec(t, `INSERT INTO menu_sections (merchant_id, name) VALUES ($1, 'الرئيسية') RETURNING id::text`, merchantID)
	return f.exec(t, `
		INSERT INTO menu_items (merchant_id, section_id, platform_section_id, name, merchant_price, price,
		                        available, margin_override)
		VALUES ($1, $2, (SELECT id FROM platform_sections ORDER BY sort_order LIMIT 1), $3, $4, $4, true, 0)
		RETURNING id::text`, merchantID, sec, name, price)
}

func (f *iuFixture) exec(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("زرع: %v", err)
	}
	return id
}

func (f *cashFixture) exec(t *testing.T, sql string, args ...any) string {
	t.Helper()
	var id string
	if err := f.pool.QueryRow(context.Background(), sql, args...).Scan(&id); err != nil {
		t.Fatalf("زرع: %v", err)
	}
	return id
}

func i64(v int64) *int64 { return &v }

// ── نقداً قبل الاستلام: كمّيّةٌ تزيد، وسعرُ متجرٍ تغيّر، وصنفٌ يُضاف ─────────
func TestOrderEdit_BeforePickup_Cash(t *testing.T) {
	f := iuSetup(t, "accepted", "cash") // أ ×٢ (١٠٠٠/٩٠٠) · ب ×١ (٥٠٠/٤٥٠) · توصيل ١٠٠ = ٢٦٠٠
	ctx := context.Background()
	item := editMenuItem(t, f, f.merchantID, "ج", 300)

	o, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "طلب الزبون بالهاتف",
		Lines: []orders.EditLine{
			{ItemID: f.lineA, Qty: 3},
			{ItemID: f.lineB, Qty: 1, MerchantPrice: i64(600)}, // ٥٠٠ ← ٦٥٠ (الهامشُ ٥٠ كما كان)
		},
		Add: []orders.EditAdd{{MenuItemID: item, Qty: 2}},
	}, false, nil)
	if err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	var addUnit int64
	if err := f.pool.QueryRow(ctx, `SELECT unit_price FROM order_items WHERE order_id = $1 AND name = 'ج'`,
		f.orderID).Scan(&addUnit); err != nil {
		t.Fatalf("الصنفُ المضاف: %v", err)
	}
	wantSub := int64(3*1000+650) + 2*addUnit
	st := f.read(t)
	if st.subtotal != wantSub || st.total != wantSub+100 || st.cashDue != wantSub+100 || st.walletPaid != 0 {
		t.Fatalf("بعد التعديل: %+v — يُنتظر بضاعة %d ومجموعاً ونقداً %d", st, wantSub, wantSub+100)
	}
	if res.OldTotal != 2_600 || res.NewTotal != wantSub+100 || len(res.Changes) != 3 || o.Total != wantSub+100 {
		t.Fatalf("النتيجة: %+v", res)
	}
	var unit, cost int64
	if err := f.pool.QueryRow(ctx, `SELECT unit_price, merchant_price FROM order_items WHERE id = $1`, f.lineB).
		Scan(&unit, &cost); err != nil || unit != 650 || cost != 600 {
		t.Fatalf("سعرُ «ب»: %d/%d (%v) — يُنتظر 650/600", unit, cost, err)
	}
	var events, notes int
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM order_events WHERE order_id = $1 AND from_status = 'accepted'
		   AND to_status = 'accepted' AND note LIKE '%تعديل الطلب%' AND note LIKE '%طلب الزبون بالهاتف%'`,
		f.orderID).Scan(&events); err != nil || events != 1 {
		t.Fatalf("حدثُ الطلب=%d (%v)", events, err)
	}
	if err := f.pool.QueryRow(ctx, `
		SELECT count(*) FROM notifications WHERE user_id = $1 AND entity_id = $2 AND body LIKE '%طلب الزبون بالهاتف%'`,
		f.customer, f.orderID).Scan(&notes); err != nil || notes != 1 {
		t.Fatalf("إشعارُ الزبون=%d (%v)", notes, err)
	}
	// **وثوابتُ مجاميع الطلب.**
	v, err := fininv.Run(ctx, f.pool, "FI-03.b")
	if err != nil || len(v) != 0 {
		t.Fatalf("FI-03.b: %v %+v", err, v)
	}
}

// ── المعاينةُ تحسب ولا تترك أثراً ──────────────────────────────────────
func TestOrderEdit_DryRunLeavesNoTrace(t *testing.T) {
	f := iuSetup(t, "preparing", "cash")
	ctx := context.Background()
	before := f.read(t)
	o, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "معاينة", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 1}}, DryRun: true,
	}, false, nil)
	if err != nil || o != nil {
		t.Fatalf("المعاينة: %v (%v)", err, o)
	}
	if res.NewTotal != 1_600 || !res.DryRun {
		t.Fatalf("المعاينةُ قالت %+v — يُنتظر 1600", res)
	}
	if after := f.read(t); after != before {
		t.Fatalf("المعاينةُ غيّرت الطلب: %+v ← %+v", before, after)
	}
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM order_events WHERE order_id = $1 AND note LIKE '%تعديل الطلب%'`,
		f.orderID).Scan(&n)
	if n != 0 {
		t.Fatalf("المعاينةُ كتبت %d حدثاً", n)
	}
}

// ── بالمحفظة: النقصُ يُردّ، والزيادةُ تُخصم، وما لا يكفيه الرصيدُ نقدٌ ───────
func TestOrderEdit_BeforePickup_Wallet(t *testing.T) {
	f := iuSetup(t, "accepted", "wallet") // دُفع ٢٦٠٠ من المحفظة
	ctx := context.Background()

	// نقص: «ب» تُحذف ⇒ ٢١٠٠، يُردّ ٥٠٠.
	if _, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "نقص", Lines: []orders.EditLine{{ItemID: f.lineB, Qty: 0}},
	}, false, nil); err != nil || res.Refunded != 500 {
		t.Fatalf("النقص: %v %+v", err, res)
	}
	if st := f.read(t); st.total != 2_100 || st.walletPaid != 2_100 || st.cashDue != 0 {
		t.Fatalf("بعد النقص: %+v", st)
	}
	if got := f.balance(t, f.customer); got != 500 {
		t.Fatalf("المحفظةُ %d لا 500", got)
	}

	// زيادة: «أ» ×٢ ← ×٣ (+١٠٠٠) والرصيدُ ٥٠٠ ⇒ يُخصم ٥٠٠ والباقي ٥٠٠ نقد.
	_, res, err := f.svc.EditOrder(ctx, f.customer, f.orderID, orders.EditRequest{
		Reason: "زيادة", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 3}},
	}, false, nil)
	if err != nil || res.Charged != 500 || res.CashDue != 500 {
		t.Fatalf("الزيادة: %v %+v", err, res)
	}
	if st := f.read(t); st.total != 3_100 || st.walletPaid != 2_600 || st.cashDue != 500 {
		t.Fatalf("بعد الزيادة: %+v", st)
	}
	if got := f.balance(t, f.customer); got != 0 {
		t.Fatalf("المحفظةُ %d لا 0", got)
	}
	v, err := fininv.Run(ctx, f.pool, "FI-03.b")
	if err != nil || len(v) != 0 {
		t.Fatalf("FI-03.b: %v %+v", err, v)
	}
}

// ── الحدود: آخرُ صنف، ومتجرٌ ليس في الطلب، وبلا سبب، وبعد التسليم، والخاصّ ──
func TestOrderEdit_Guards(t *testing.T) {
	f := iuSetup(t, "accepted", "cash")
	ctx := context.Background()
	edit := func(req orders.EditRequest) error {
		_, _, err := f.svc.EditOrder(ctx, f.customer, f.orderID, req, false, nil)
		return err
	}
	if err := edit(orders.EditRequest{Reason: "كلّه", Lines: []orders.EditLine{
		{ItemID: f.lineA, Qty: 0}, {ItemID: f.lineB, Qty: 0}}}); !errors.Is(err, orders.ErrOrderEditLast) {
		t.Fatalf("حذفُ الكلّ: %v — يُنتظر order_edit_last_item", err)
	}
	other, _ := (&cashFixture{pool: f.pool, categoryID: f.categoryOf(t)}).merchant(t, "cash", 10)
	foreign := editMenuItem(t, f, other, "غريب", 100)
	if err := edit(orders.EditRequest{Reason: "غريب", Add: []orders.EditAdd{{MenuItemID: foreign, Qty: 1}}}); !errors.Is(err, orders.ErrOrderEditStore) {
		t.Fatalf("صنفُ متجرٍ آخر: %v", err)
	}
	if err := edit(orders.EditRequest{Reason: "  ", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 1}}}); !errors.Is(err, orders.ErrOrderEditReason) {
		t.Fatalf("بلا سبب: %v", err)
	}
	if err := edit(orders.EditRequest{Reason: "لا شيء", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 2}}}); !errors.Is(err, orders.ErrOrderEditNoChange) {
		t.Fatalf("بلا تغيير: %v", err)
	}
	// **والطلبُ كما كان بعد كلّ ما رُدّ.**
	if st := f.read(t); st.total != 2_600 {
		t.Fatalf("تغيّر الطلبُ بعد ردودٍ: %+v", st)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'delivered' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if err := edit(orders.EditRequest{Reason: "بعد", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 1}}}); !errors.Is(err, orders.ErrOrderEditClosed) {
		t.Fatalf("بعد التسليم: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = 'accepted', kind = 'custom' WHERE id = $1`, f.orderID); err != nil {
		t.Fatal(err)
	}
	if err := edit(orders.EditRequest{Reason: "خاص", Lines: []orders.EditLine{{ItemID: f.lineA, Qty: 1}}}); !errors.Is(err, orders.ErrOrderEditKind) {
		t.Fatalf("الخاصّ: %v", err)
	}
}

func (f *iuFixture) categoryOf(t *testing.T) string {
	t.Helper()
	return f.exec(t, `SELECT category_id::text FROM merchants WHERE id = $1`, f.merchantID)
}

// ── طلبٌ عُدّل قبل الاستلام يُستلم ويُسلَّم والدفترُ أخضر ─────────────────────
func TestOrderEdit_BeforePickup_ThenDeliverGreen(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash") // at_pickup، بندٌ ٥٠٬٠٠٠
	ctx := context.Background()
	add := editMenuItem(t, f, m, "إضافة", 5_000)
	var line string
	if err := f.pool.QueryRow(ctx, `SELECT id::text FROM order_items WHERE order_id = $1`, oid).Scan(&line); err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.svc.EditOrder(ctx, f.driver, oid, orders.EditRequest{
		Reason: "قبل الاستلام", Lines: []orders.EditLine{{ItemID: line, Qty: 2}},
		Add: []orders.EditAdd{{MenuItemID: add, Qty: 1}},
	}, false, nil); err != nil {
		t.Fatalf("التعديل: %v", err)
	}
	for _, st := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	// **مستحقُّ المتجر تبع البنود الجديدة**: (١٠٠٬٠٠٠ + ٥٬٠٠٠) − ١٠٪.
	if s := f.settlement(t, oid, m); s.amount != 94_500 || s.state != "cash_paid" {
		t.Fatalf("التسوية: %+v — يُنتظر 94500 مدفوعة", s)
	}
	f.assertDriverPayGreen(t)
	v, err := fininv.Run(ctx, f.pool, "FI-03.b", "FI-03.c", "FI-06.b", "FI-05.c")
	if err != nil || len(v) != 0 {
		t.Fatalf("الثوابت: %v %+v", err, v)
	}
}

// ── دليلُ المحرّر: سطورُ الطلب بسعريها وأصنافُ متاجره وحدَها ─────────────────
func TestOrderEdit_Catalog(t *testing.T) {
	f := iuSetup(t, "accepted", "cash")
	item := editMenuItem(t, f, f.merchantID, "ج", 300)
	cat, err := f.svc.EditCatalog(context.Background(), f.orderID)
	if err != nil {
		t.Fatalf("الدليل: %v", err)
	}
	if len(cat.Lines) != 2 || len(cat.Stores) != 1 || cat.Stores[0].MerchantID != f.merchantID {
		t.Fatalf("الدليل: %+v", cat)
	}
	var costA int64
	for _, l := range cat.Lines {
		if l.ID == f.lineA {
			costA = l.MerchantPrice
		}
	}
	if costA != 900 {
		t.Fatalf("سعرُ متجر «أ» %d لا 900", costA)
	}
	found := false
	for _, it := range cat.Stores[0].Items {
		found = found || it.ID == item
	}
	if !found {
		t.Fatal("الصنفُ «ج» غائبٌ عن الدليل")
	}
}
