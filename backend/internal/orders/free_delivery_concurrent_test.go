package orders_test

// سؤالُ المالك (٢٠٢٦-١٠-٢٦): زبونٌ طلب طلبين بنفس اللحظة والعرضُ شغّال —
// الأوّلُ مجّانيٌّ والثاني بأجرته؟ وتسليمُ الأوّل لا يغيّر شيئاً في الثاني؟

import (
	"context"
	"sync"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
)

// TestFreeCampaign_TwoOrdersAtOnce **طلبان معاً: واحدٌ مجّانيّ فقط، والآخرُ بأجرته.**
func TestFreeCampaign_TwoOrdersAtOnce(t *testing.T) {
	f := setup(t, "pending", 100_000, 0, 0)
	ctx := context.Background()
	f.verifyWhatsApp(t)
	f.armTreasury(t)
	itemID := f.armItem(t)
	setDeliveryFee(t, f, "5000")
	armFreeCampaign(t, []string{orders.FreeKindStandard}, nil)

	var wg sync.WaitGroup
	ids := make([]string, 2)
	errs := make([]error, 2)
	start := make(chan struct{})
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			o, err := f.svc.Create(ctx, f.customer, []string{"customer"}, orders.CreateInput{
				CustomerID: f.customer, MerchantID: f.merchantID,
				Items:       []orders.ItemInput{{MenuItemID: itemID, Qty: 1}},
				AddressText: "طلبان معاً", Lat: 35.9528, Lng: 39.0079, PaymentMethod: "cash",
			}, "127.0.0.1")
			errs[i] = err
			if o != nil {
				ids[i] = o.ID
			}
		}(i)
	}
	close(start)
	wg.Wait()
	for i, id := range ids {
		if errs[i] != nil {
			t.Fatalf("الطلب %d: %v", i+1, errs[i])
		}
		id := id
		t.Cleanup(func() { _, _ = f.pool.Exec(context.Background(), `DELETE FROM orders WHERE id = $1`, id) })
	}

	freeCount, paidCount := 0, 0
	var freeID, paidID string
	for _, id := range ids {
		d, _, w, _ := f.fees(t, id)
		switch {
		case d == 0 && w == 5000:
			freeCount++
			freeID = id
		case d == 5000 && w == 0:
			paidCount++
			paidID = id
		default:
			t.Fatalf("طلبٌ بأجرة %d ومعفى %d — لا مجّانيٌّ ولا مدفوع", d, w)
		}
	}
	if freeCount != 1 || paidCount != 1 {
		t.Fatalf("مجّانيّ %d ومدفوع %d — والمنتظَر واحدٌ وواحد", freeCount, paidCount)
	}

	// **يُسلَّم المجّانيّ: الثاني لا يتغيّر فيه شيء.**
	var cashBefore, totalBefore, feeBefore int64
	if err := f.pool.QueryRow(ctx, `SELECT cash_due, total, delivery_fee FROM orders WHERE id = $1`, paidID).
		Scan(&cashBefore, &totalBefore, &feeBefore); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(ctx,
		`UPDATE orders SET driver_id = $2, status = 'at_dropoff' WHERE id = $1`, freeID, f.driver); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.Transition(ctx, f.driver, []string{"driver"}, freeID, "delivered", ""); err != nil {
		t.Fatalf("التسليم: %v", err)
	}
	var cashAfter, totalAfter, feeAfter int64
	if err := f.pool.QueryRow(ctx, `SELECT cash_due, total, delivery_fee FROM orders WHERE id = $1`, paidID).
		Scan(&cashAfter, &totalAfter, &feeAfter); err != nil {
		t.Fatal(err)
	}
	if cashAfter != cashBefore || totalAfter != totalBefore || feeAfter != feeBefore {
		t.Fatalf("تسليمُ الأوّل غيّر الثاني: النقد %d→%d · الإجمالي %d→%d · الأجرة %d→%d",
			cashBefore, cashAfter, totalBefore, totalAfter, feeBefore, feeAfter)
	}
	if cashAfter == 0 {
		t.Fatalf("الثاني صار «مدفوعاً» (نقدُه صفر) — والمنتظَر أن يدفع الزبونُ نقداً")
	}
}
