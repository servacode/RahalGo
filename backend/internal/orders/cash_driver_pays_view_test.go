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

// TestDRVPAY_CommissionShownToDriver **ولماذا يدفع أقلَّ من الفاتورة** (قرارُ المالك ٢٠٢٦-١١-١٢) —
// الفاتورةُ والعمولةُ ونسبتُها ودَينُ المتجر المقتطَع، **من حسبة الدفعة نفسِها.**
func TestDRVPAY_CommissionShownToDriver(t *testing.T) {
	f := newCashFixture(t)
	ctx := context.Background()
	m, _ := f.merchant(t, "cash", 10)
	f.mkDebt(t, m, 5_000)
	oid := f.order(t, m, 50_000, 10_000, "cash")
	got, err := f.svc.PayMerchants(ctx, f.pool, []string{oid})
	if err != nil {
		t.Fatal(err)
	}
	p := got[oid]
	if len(p) != 1 {
		t.Fatalf("لا دفعة: %+v", p)
	}
	if p[0].Gross != 50_000 || p[0].Cut != 5_000 || p[0].DebtCut != 5_000 ||
		p[0].CutPercent == nil || p[0].CutPercent.String() != "10" {
		t.Fatalf("**العمولةُ لا تُقال للسائق**: %+v — يُنتظر فاتورة 50000 · عمولة 5000 (10٪) · دَين 5000", p[0])
	}
	if p[0].Gross-p[0].Cut-p[0].DebtCut != p[0].Amount {
		t.Fatalf("**الأرقامُ لا تجتمع**: %d − %d − %d ≠ %d", p[0].Gross, p[0].Cut, p[0].DebtCut, p[0].Amount)
	}
}
