package orders_test

import (
	"context"
	"testing"
)

// TestDRVPAY_PayMerchantShownToDriver **ما يدفعه السائقُ لكلّ متجرٍ قبل الاستلام**
// — الصافي ناقصَ دَينِ المتجر، **وبعد الاستلام ما دفعه فعلاً.**
func TestDRVPAY_PayMerchantShownToDriver(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	f.mkDebt(t, m, 5_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")

	got, err := f.svc.PayMerchants(ctx, f.pool, []string{oid})
	if err != nil {
		t.Fatal(err)
	}
	if p := got[oid]; len(p) != 1 || p[0].Amount != 40_000 || p[0].MerchantID != m {
		t.Fatalf("قبل الاستلام: %+v — يُنتظر 40000 (45000 − دَينُ 5000)", p)
	}
	f.pickup(t, oid)
	got, err = f.svc.PayMerchants(ctx, f.pool, []string{oid})
	if err != nil {
		t.Fatal(err)
	}
	if p := got[oid]; len(p) != 1 || p[0].Amount != 40_000 {
		t.Fatalf("بعد الاستلام: %+v — يُنتظر 40000", p)
	}
	o, err := f.svc.GetByID(ctx, oid)
	if err != nil {
		t.Fatal(err)
	}
	if o.PayMerchant != 40_000 {
		t.Fatalf("pay_merchant في الطلب %d لا 40000", o.PayMerchant)
	}

	// **والمتجرُ المحفظيُّ لا يُدفع له شيء.**
	mw, _ := f.merchant(t, "wallet", 10)
	ow := f.order(t, mw, 50_000, 10_000, "wallet")
	got, err = f.svc.PayMerchants(ctx, f.pool, []string{ow})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[ow]) != 0 {
		t.Fatalf("متجرٌ محفظيٌّ يُدفع له: %+v", got[ow])
	}
}
