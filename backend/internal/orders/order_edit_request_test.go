package orders_test

// ══════════════════════════════════════════════════════════════════════
// **طلب تعديل من الكابتن — والمكتب يوافق أو يرفض** (قرارُ المالك ٢٠٢٦-١١-١٠)
// ══════════════════════════════════════════════════════════════════════
//
// **يُقاس ما يقع**: الطلب والحلول محل المعلّق، والموافقة من باب تعديل المكتب بمالها كلّه (المجموع
// والنقد والمحفظة وحق المتجر والعمولة) قبل الاستلام وبعده، والقدرة بعد الاستلام، والرفض، والطلب
// اللي تغيّر تحته، ثمّ تسليمٌ وثوابتُ الدفتر خضراء.

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// addLine **سطرٌ زائدٌ على طلب العُدّة** — والمجموع والنقد بيتبعوه.
func (f *cashFixture) addLine(t *testing.T, orderID, merchantID, name string, price int64, qty int) string {
	t.Helper()
	id := f.exec(t, `
		INSERT INTO order_items (order_id, name, unit_price, merchant_price, merchant_id, qty, options,
		                         merchant_settlement_method)
		VALUES ($1, $2, $3, $3, $4, $5, '[]'::jsonb,
		        (SELECT merchant_settlement_method FROM order_items WHERE order_id = $1 LIMIT 1))
		RETURNING id::text`, orderID, name, price, merchantID, qty)
	extra := price * int64(qty)
	if _, err := f.pool.Exec(context.Background(), `
		UPDATE orders SET subtotal = subtotal + $2, total = total + $2,
		       cash_due = CASE WHEN payment_method = 'cash' THEN cash_due + $2 ELSE cash_due END
		 WHERE id = $1`, orderID, extra); err != nil {
		t.Fatal(err)
	}
	return id
}

func (f *cashFixture) requestStatus(t *testing.T, id string) string {
	t.Helper()
	return f.exec(t, `SELECT status FROM order_edit_requests WHERE id = $1`, id)
}

func (f *cashFixture) askEdit(t *testing.T, orderID string, in orders.DriverEditInput) *orders.EditRequestView {
	t.Helper()
	v, err := f.svc.RequestOrderEdit(context.Background(), orderID, f.driver, in)
	if err != nil {
		t.Fatalf("طلب التعديل: %v", err)
	}
	return v
}

// ── قبل الاستلام: طلبٌ يحلّ محل طلب، وموافقةٌ من المكتب بلا قدرة ما بعد الاستلام ─────
func TestEditRequest_BeforePickup_ReplaceThenApprove(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash") // at_pickup
	b := f.addLine(t, oid, m, "ب", 5_000, 2)     // المجموع ٧٠٬٠٠٠

	first := f.askEdit(t, oid, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: b, Action: orders.EditRequestMissing, Qty: 1}},
		Adds:  []orders.EditRequestAdd{{Name: "خبز", Qty: 2, Price: 1_000}},
	})
	if first.Status != "pending" || first.OldTotal != 70_000 || first.NewTotal != 67_000 {
		t.Fatalf("الطلب الأول: %+v — يُنتظر 70000←67000", first)
	}
	// **والطلب ما بيغيّر شي قبل الموافقة.**
	if total, _, _ := f.orderRow(t, oid); total != 70_000 {
		t.Fatalf("الطلب تغيّر قبل الموافقة: %d", total)
	}
	second := f.askEdit(t, oid, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: b, Action: orders.EditRequestCancelled}},
		Adds:  []orders.EditRequestAdd{{Name: "خبز", Qty: 1, Price: 1_500}},
		Note:  "الزبون اتصل",
	})
	if got := f.requestStatus(t, first.ID); got != "superseded" {
		t.Fatalf("الأول صار %q — يُنتظر superseded", got)
	}
	if second.NewTotal != 61_500 {
		t.Fatalf("الثاني: %+v — يُنتظر 61500", second)
	}
	// **ولوحة المكتب بتشوفه** — بالبطاقة وبـ«بانتظار قرارك».
	info, err := f.svc.BoardInfoOf(ctx, []string{oid})
	if err != nil || info[oid] == nil || info[oid].EditRequestAt == nil {
		t.Fatalf("اللوحة ما بتشوف الطلب: %+v %v", info[oid], err)
	}
	pend, err := f.svc.PendingEditRequest(ctx, oid, f.treasury)
	if err != nil || pend == nil || pend.Preview == nil || pend.Preview.NewTotal != 61_500 || pend.Stale || pend.AfterPickup {
		t.Fatalf("معاينة المكتب: %+v %v", pend, err)
	}

	_, res, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, second.ID, false, nil)
	if err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	if res.AfterPickup || res.NewTotal != 61_500 || res.CashDue != 61_500 {
		t.Fatalf("النتيجة: %+v", res)
	}
	if total, cash, _ := f.orderRow(t, oid); total != 61_500 || cash != 61_500 {
		t.Fatalf("المجموع %d والنقد %d — يُنتظر 61500", total, cash)
	}
	if got := f.requestStatus(t, second.ID); got != "approved" {
		t.Fatalf("الحالة %q", got)
	}
	// **والزبون بيوصله المجموع الحقيقي.**
	var notes int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND entity_id = $2
		AND body LIKE '%61,500%'`, f.customer, oid).Scan(&notes)
	if notes != 1 {
		t.Fatalf("إشعار الزبون بالمجموع الجديد: %d", notes)
	}
	if info, _ := f.svc.BoardInfoOf(ctx, []string{oid}); info[oid].EditRequestAt != nil {
		t.Fatal("الطلب لسا ظاهر معلّق باللوحة")
	}
	// **والاستلام والتسليم على البضاعة الحقيقية** — العمولة ١٠٪ من ٥١٬٥٠٠.
	f.deliver(t, oid)
	var commission int64
	_ = f.pool.QueryRow(ctx, `SELECT platform_commission FROM orders WHERE id = $1`, oid).Scan(&commission)
	if commission != 5_150 {
		t.Fatalf("العمولة %d لا 5150", commission)
	}
	f.green(t, "بعد التسليم")
	// **والموافقة مرّة وحدة.**
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, second.ID, true, nil); !errors.Is(err, orders.ErrEditRequestNotPending) {
		t.Fatalf("موافقة تانية: %v", err)
	}
}

// ── بعد الاستلام: «ناقص» ينقص حق المتجر و«الزبون لغاه» يبقيه، وبقدرة الأدمن بس ──────
func TestEditRequest_AfterPickup_MissingAndCancelled(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	b := f.addLine(t, oid, m, "ب", 5_000, 2)
	c := f.addLine(t, oid, m, "ج", 3_000, 1) // المجموع ٧٣٬٠٠٠ والبضاعة ٦٣٬٠٠٠
	f.pickup(t, oid)                         // دفع السائق ٥٦٬٧٠٠

	req := f.askEdit(t, oid, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{
			{ItemID: b, Action: orders.EditRequestMissing, Qty: 1},
			{ItemID: c, Action: orders.EditRequestCancelled},
		},
	})
	if req.NewTotal != 65_000 {
		t.Fatalf("المتوقّع: %+v — يُنتظر 65000", req)
	}
	pend, err := f.svc.PendingEditRequest(ctx, oid, f.treasury)
	if err != nil || pend == nil || !pend.AfterPickup {
		t.Fatalf("المعاينة لازم تقول بعد الاستلام: %+v %v", pend, err)
	}
	// **بلا قدرة ما بعد الاستلام بينرد ولا بيتغيّر شي** (البند ٤).
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.driver, oid, req.ID, false, nil); !errors.Is(err, orders.ErrOrderEditAfterPickup) {
		t.Fatalf("بلا قدرة: %v", err)
	}
	if got := f.requestStatus(t, req.ID); got != "pending" {
		t.Fatalf("الطلب صار %q بعد رفض القدرة", got)
	}
	_, res, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, true, nil)
	if err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	// ب ×١ ناقصة: المتجر بيرجّع ٤٬٥٠٠. ج لغاها الزبون: حقّ المتجر كامل والمنصة بتتحمّل.
	if len(res.Stores) != 1 || res.Stores[0].DriverPays != -4_500 || res.Stores[0].NewDue != 52_200 {
		t.Fatalf("حق المتجر: %+v — يُنتظر -4500 و52200", res.Stores)
	}
	if total, cash, _ := f.orderRow(t, oid); total != 65_000 || cash != 65_000 {
		t.Fatalf("المجموع %d والنقد %d — يُنتظر 65000", total, cash)
	}
	var returned int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM order_items WHERE order_id = $1 AND returned`, oid).Scan(&returned)
	if returned != 1 {
		t.Fatalf("سطر «الزبون لغاه» لازم يضل مرتجع: %d", returned)
	}
	f.green(t, "بعد الموافقة")
	for _, st := range []string{"on_the_way", "at_dropoff", "delivered"} {
		f.transition(t, oid, st)
	}
	f.green(t, "بعد التسليم")
}

// ── محفظة: النقص بيرجع للمحفظة فوراً ─────────────────────────────────────
func TestEditRequest_WalletRefund(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.orderWalletPaid(t, m, 50_000, 10_000)
	b := f.addLine(t, oid, m, "ب", 5_000, 2)
	if _, err := f.pool.Exec(ctx, `UPDATE orders SET wallet_paid = total WHERE id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, 10_000, "topup", "seed2-"+oid, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := f.wallet.Apply(ctx, f.customer, -10_000, "order_payment", oid, "", nil); err != nil {
		t.Fatal(err)
	}
	before := f.balance(t, f.customer)
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: b, Action: orders.EditRequestMissing}},
	})
	_, res, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, false, nil)
	if err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	if res.Refunded != 10_000 || res.CashDue != 0 {
		t.Fatalf("النتيجة: %+v — يُنتظر رد 10000 ونقد صفر", res)
	}
	if got := f.balance(t, f.customer) - before; got != 10_000 {
		t.Fatalf("المحفظة زادت %d لا 10000", got)
	}
	f.deliver(t, oid)
	f.green(t, "بعد التسليم")
}

// ── الطلب تغيّر تحت الطلب ⇒ الموافقة بتنرد وبيصير «قديم» ───────────────────
func TestEditRequest_StaleAfterOrderChanged(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	b := f.addLine(t, oid, m, "ب", 5_000, 3)
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: b, Action: orders.EditRequestMissing, Qty: 1}},
	})
	f.edit(t, oid, orders.EditRequest{Lines: []orders.EditLine{{ItemID: b, Qty: 2}}})
	pend, err := f.svc.PendingEditRequest(ctx, oid, f.treasury)
	if err != nil || pend == nil || !pend.Stale {
		t.Fatalf("المعاينة لازم تقول قديم: %+v %v", pend, err)
	}
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, true, nil); !errors.Is(err, orders.ErrEditRequestStale) {
		t.Fatalf("موافقة على طلب قديم: %v", err)
	}
	if got := f.requestStatus(t, req.ID); got != "stale" {
		t.Fatalf("الحالة %q لا stale", got)
	}
	if total, _, _ := f.orderRow(t, oid); total != 70_000 {
		t.Fatalf("المجموع %d — التعديل القديم ما لازم يمشي", total)
	}
}

// ── الرفض بسبب، والكابتن بيعرف، والطلب ما بيتغيّر ─────────────────────────
func TestEditRequest_Reject(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	rec := &recorder{}
	f.svc.SetNotifier(rec)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "ماء", Qty: 1, Price: 500}},
	})
	if rec.ops != 1 {
		t.Fatalf("المكتب ما انخبر بالطلب: %d", rec.ops)
	}
	if err := f.svc.RejectEditRequest(ctx, f.treasury, oid, req.ID, " ", nil); !errors.Is(err, orders.ErrEditRequestRejectReason) {
		t.Fatalf("رفض بلا سبب: %v", err)
	}
	if err := f.svc.RejectEditRequest(ctx, f.treasury, oid, req.ID, "الزبون ما طلب", nil); err != nil {
		t.Fatalf("الرفض: %v", err)
	}
	if got := f.requestStatus(t, req.ID); got != "rejected" {
		t.Fatalf("الحالة %q", got)
	}
	if !rec.told(f.driver) {
		t.Fatal("الكابتن ما انخبر بالرفض")
	}
	v, err := f.svc.DriverEditRequest(ctx, oid, f.driver)
	if err != nil || v == nil || v.Status != "rejected" || v.RejectReason != "الزبون ما طلب" {
		t.Fatalf("الكابتن بيشوف: %+v %v", v, err)
	}
	if total, _, _ := f.orderRow(t, oid); total != 60_000 {
		t.Fatalf("المجموع تغيّر: %d", total)
	}
	if _, _, err := f.svc.ApproveEditRequest(ctx, f.treasury, oid, req.ID, true, nil); !errors.Is(err, orders.ErrEditRequestNotPending) {
		t.Fatalf("موافقة بعد الرفض: %v", err)
	}
}

// ── حرّاس الطلب: كابتن غيره، فاضي، أكتر من الكمّية، آخر صنف، بعد التسليم ─────────
func TestEditRequest_Guards(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	line := f.line(t, oid)
	code := func(err error) string {
		var ae *httpx.AppError
		if errors.As(err, &ae) {
			return ae.Code
		}
		return ""
	}
	other := testdb.NewUser(t, f.pool, "driver")
	if _, err := f.svc.RequestOrderEdit(ctx, oid, other, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "س", Qty: 1, Price: 100}}}); code(err) != "not_your_order" {
		t.Fatalf("كابتن غيره: %v", err)
	}
	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{}); !errors.Is(err, orders.ErrEditRequestEmpty) {
		t.Fatalf("فاضي: %v", err)
	}
	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: line, Action: orders.EditRequestMissing, Qty: 5}}}); code(err) != "validation" {
		t.Fatalf("أكتر من الكمّية: %v", err)
	}
	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: line, Action: "x"}}}); code(err) != "validation" {
		t.Fatalf("فعل غلط: %v", err)
	}
	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{
		Lines: []orders.EditRequestLine{{ItemID: line, Action: orders.EditRequestCancelled}}}); !errors.Is(err, orders.ErrOrderEditLast) {
		t.Fatalf("آخر صنف: %v", err)
	}
	var n int
	_ = f.pool.QueryRow(ctx, `SELECT count(*) FROM order_edit_requests WHERE order_id = $1`, oid).Scan(&n)
	if n != 0 {
		t.Fatalf("انحفظ طلب مرفوض: %d", n)
	}
	f.deliver(t, oid)
	if _, err := f.svc.RequestOrderEdit(ctx, oid, f.driver, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "س", Qty: 1, Price: 100}}}); !errors.Is(err, orders.ErrEditRequestStage) {
		t.Fatalf("بعد التسليم: %v", err)
	}
}

// ── طلب معلّق على طلب انتهى بيتسكّر بجولة الراصد ─────────────────────────
func TestEditRequest_ClosedWhenOrderEnds(t *testing.T) {
	f := newCashFixture(t)
	m, _ := f.merchant(t, "cash", 10)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	req := f.askEdit(t, oid, orders.DriverEditInput{
		Adds: []orders.EditRequestAdd{{Name: "ماء", Qty: 1, Price: 500}},
	})
	f.deliver(t, oid)
	f.svc.SweepPriceRequestsOnce(context.Background())
	if got := f.requestStatus(t, req.ID); got != "closed" {
		t.Fatalf("الحالة %q لا closed", got)
	}
}
