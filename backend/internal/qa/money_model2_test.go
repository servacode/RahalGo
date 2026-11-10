package qa

// ══════════════════════════════════════════════════════════════════════
// **النموذج ٢: البضاعةُ خارجَ كلّ حساب** (خطّةُ المال ٢٠٢٦-١١-١٠، الدفعات B4…B9)
// ══════════════════════════════════════════════════════════════════════
//
// المفتاحُ `finance.goods_outside` مشغَّلٌ هنا وحدَه (مطفأٌ افتراضاً في كلّ مكان). وكلُّ سيناريو
// يُختم بثوابت المال كلِّها (النموذج ١ والنموذج ٢ معاً) بلا خرقٍ جديد.

import (
	"testing"
)

// m2Checks **العائلاتُ التي يُختم بها كلُّ سيناريو** — القديمةُ (التي تستثني النموذج ٢ حيث يلزم)
// والجديدةُ FI-17.
var m2Checks = []string{"FI-03", "FI-04", "FI-05", "FI-06", "FI-09", "FI-10", "FI-13", "FI-14", "FI-15", "FI-17"}

// m2Prep **منصّةٌ بخزينةٍ والنموذج ٢ مشغَّل** — ومتجرٌ بعمولة ١٠٪ وصنفٌ بهامش ١٠٠٠.
func m2Prep(t *testing.T, h *Harness) (item *Item, tid string) {
	t.Helper()
	wsPrep(t, h)
	tid = treasury(t, h)
	h.Setting("finance.goods_outside", "true")
	m := h.Factory().Merchant(Commission(10))
	return h.NewItemPriced(m, 10_000, 11_000), tid
}

type m2Facts struct {
	goodsRows, advance, profitRows int64
	profit, box, rowAmount         int64
	rowState, rowVia               string
	offLedger                      bool
	cashDue, walletPaid, driverFee int64
	commission                     int64
}

func m2Read(t *testing.T, h *Harness, oid string) m2Facts {
	t.Helper()
	var f m2Facts
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT
		  (SELECT count(*) FROM wallet_transactions WHERE ref = $1
		     AND kind IN ('merchant_earning','merchant_cash_accrued','merchant_cash_paid','custom_goods_return')),
		  (SELECT count(*) FROM office_cash_entries WHERE ref = $1
		     AND source IN ('advance_return','undelivered_return')),
		  (SELECT count(*) FROM wallet_transactions WHERE ref = $1 AND kind = 'platform_profit'),
		  COALESCE((SELECT sum(amount) FROM wallet_transactions WHERE ref = $1 AND kind = 'platform_profit'), 0),
		  COALESCE((SELECT sum(amount) FROM driver_cash_entries WHERE ref = $1 AND kind = 'order_collection'), 0),
		  COALESCE((SELECT sum(amount - reversed_amount) FROM merchant_settlements WHERE order_id = $1::uuid), 0),
		  COALESCE((SELECT max(state) FROM merchant_settlements WHERE order_id = $1::uuid), ''),
		  COALESCE((SELECT max(paid_via) FROM merchant_settlements WHERE order_id = $1::uuid), ''),
		  -- **بـ to_jsonb لا بالعمود** — فيُقرأ على قاعدةٍ قبل الهجرة ٠٥٣٥ ويسقط الاختبارُ بسببه الحقيقيّ.
		  COALESCE((SELECT bool_and(COALESCE((to_jsonb(ms)->>'off_ledger')::bool, false))
		              FROM merchant_settlements ms WHERE ms.order_id = $1::uuid), false),
		  o.cash_due, o.wallet_paid, o.driver_fee, o.platform_commission
		FROM orders o WHERE o.id = $1::uuid`, oid).
		Scan(&f.goodsRows, &f.advance, &f.profitRows, &f.profit, &f.box, &f.rowAmount,
			&f.rowState, &f.rowVia, &f.offLedger, &f.cashDue, &f.walletPaid, &f.driverFee, &f.commission); err != nil {
		t.Fatalf("قراءةُ وقائع الطلب: %v", err)
	}
	return f
}

// ── B4 · طلبٌ عاديٌّ نقداً وبأجرةٍ من المحفظة: البضاعةُ لا تدخل حساباً ──────────
func TestModel2_CashOrderGoodsOffLedger(t *testing.T) {
	h := New(t)
	item, tid := m2Prep(t, h)
	base := financialBaseline(t, h, m2Checks...)

	for _, c := range []struct{ name, method, scope string }{
		{"cash", "cash", ""},
		{"fee-from-wallet", "", "delivery"},
	} {
		cust := h.Customer()
		oid := wsMake(t, h, cust, wsBody(item, 2, c.method, c.scope), 5_000)
		if m := moneyModelOf(t, h, oid); m != 2 {
			t.Fatalf("%s: نموذجُ الطلب %d — يُنتظر ٢", c.name, m)
		}
		drv := h.driverOf(oid)
		before := treasuryBalance(t, h, tid)
		wsDrive(t, h, oid, drv, "at_pickup", "picked_up")

		// ── بعد الاستلام: صفُّ سجلٍّ دفعه الكابتن، ولا قيدَ ولا خزينة ──
		f := m2Read(t, h, oid)
		if !f.offLedger || f.rowState != "cash_paid" || f.rowVia != "driver" || f.rowAmount != 18_000 {
			t.Errorf("%s بعد الاستلام: الصفّ %+v — يُنتظر سجلٌّ بلا دفتر، دفعه الكابتن 18000 (كلفةُ 20000 ناقصَ ١٠٪)", c.name, f)
		}
		if f.goodsRows != 0 || f.profitRows != 0 {
			t.Errorf("%s بعد الاستلام: قيودُ بضاعة %d وقيودُ ربح %d — يُنتظر صفر", c.name, f.goodsRows, f.profitRows)
		}
		if after := treasuryBalance(t, h, tid); after != before {
			t.Errorf("%s: الخزينةُ تحرّكت عند الاستلام %d ← %d — البضاعةُ ليست عليها", c.name, before, after)
		}

		wsDrive(t, h, oid, drv, "on_the_way", "at_dropoff", "delivered")
		f = m2Read(t, h, oid)
		// الصندوقُ: نقدُ الطلب ناقصَ ما دفعه للمتجر = الأجرة (إن كاش) + العمولة ٢٠٠٠ + الهامش ٢٠٠٠.
		if f.box != f.cashDue-18_000 || f.advance != 0 || f.goodsRows != 0 {
			t.Errorf("%s بعد التسليم: صندوق %d (نقد %d) · سلفة %d · قيودُ بضاعة %d — يُنتظر صندوق = النقد − 18000 بلا سلفةٍ ولا قيد",
				c.name, f.box, f.cashDue, f.advance, f.goodsRows)
		}
		if want := f.walletPaid + f.box - f.driverFee; f.profit != want {
			t.Errorf("%s: ربحُ المنصّة %d — يُنتظر المحفظة %d + الصندوق %d − أجرُ الكابتن %d = %d",
				c.name, f.profit, f.walletPaid, f.box, f.driverFee, want)
		}
		if f.commission != 2_000 {
			t.Errorf("%s: عمولةُ الطلب %d — يُنتظر 2000", c.name, f.commission)
		}
	}
	assertNewViolations(t, h, base, m2Checks...)
}
