package qa

// ══════════════════════════════════════════════════════════════════════
// **قراراتُ المالك في الديون** (٢٠٢٦-١٠-١٥)
// ══════════════════════════════════════════════════════════════════════
//
//	١ · السائقُ طرفُ دين — وينقطع دينُه كاملاً من أجره الجاي
//	٢ · المكافأةُ والتعويضُ يسدّان الدينَ أوّلاً
//	٣ · عقوبةٌ أكبرُ من الرصيد: يُؤخذ المتاحُ والباقي دين
//	٥ · عكسُ مستحقّ المتجر يقع على متجر الطلب لا أوّلِ متاجر مالكه
//	· · العمولةُ المحجوزةُ حين تُصرف تسدّ دينَ المندوب
//	٦ · صاحبُ الدين يُخبَر حين ينشأ وحين ينقطع
//
// **وكلُّ فحصٍ هنا يقرأ الدفترَ والثوابتَ بعده** — دينٌ يُسدّ ولا تتوازن به
// الخزينةُ أسوأُ من دينٍ باقٍ.

import (
	"context"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/incentives"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/wallet"
)

// debtsFI **الثوابتُ التي يمسّها الدين** — الحفظُ والأجرُ والمكافآتُ والصورُ.
var debtsFI = []string{"FI-02.a", "FI-02.b", "FI-03.a", "FI-06.a", "FI-06.c", "FI-12.a",
	"FI-13.a", "FI-13.b", "FI-13.c", "FI-13.d", "FI-13.e"}

// debtsIncentives خدمةُ الحوافز على قاعدة الفحص — **وخزينتُها خزينةُ المحرّك.**
func debtsIncentives(h *Harness) *incentives.Service {
	return incentives.New(h.Pool, wallet.NewService(h.Pool), h.Settings,
		func(ctx context.Context) string { return h.Orders.TreasuryID(ctx) })
}

// openDebt مجموعُ المفتوح على طرف — **من الوقائع.**
func openDebt(t *testing.T, h *Harness, kind, id string) int64 {
	t.Helper()
	var v int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2::uuid AND closed_at IS NULL`, kind, id).
		Scan(&v); err != nil {
		t.Fatalf("قراءةُ الدين: %v", err)
	}
	return v
}

// noticeCount عددُ إشعارات المحفظة لشخصٍ بعنوانٍ بعينه.
func noticeCount(t *testing.T, h *Harness, userID, title string) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications WHERE user_id = $1::uuid AND title = $2`,
		userID, title).Scan(&n); err != nil {
		t.Fatalf("قراءةُ الإشعارات: %v", err)
	}
	return n
}

func driverFeeOf(t *testing.T, h *Harness, oid string) int64 {
	t.Helper()
	var fee int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT driver_fee FROM orders WHERE id = $1::uuid`, oid).
		Scan(&fee); err != nil {
		t.Fatalf("أجرُ السائق: %v", err)
	}
	return fee
}

func newOrderFor(t *testing.T, h *Harness, item *Item) string {
	t.Helper()
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 2))
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	return oid
}

// ══════════════════════════════════════════════════════════════════════
// **١ و٣ و٦ · عقوبةٌ أكبرُ من رصيد السائق تصير ديناً ينقطع من أجره الجاي**
// ══════════════════════════════════════════════════════════════════════
func TestDEBT_DriverPenaltyDebtDeductedFromNextDelivery(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, debtsFI...)
	admin := h.NewUser("admin")
	h.Setting("delivery.fee", "1000")

	oid := newOrderFor(t, h, h.NewItem(1000))
	drv := h.driverOf(oid)
	fee := driverFeeOf(t, h, oid)
	if fee < 2 {
		t.Skip("أجرُ السائق صفر")
	}
	penalty := fee / 2

	// **رصيدُه صفر** — فالعقوبةُ كلُّها دين.
	e, err := debtsIncentives(h).Grant(ctxBG(), admin.ID, drv.ID, incentives.KindPenalty,
		penalty, "تأخّر عن الاستلام")
	if err != nil {
		t.Fatalf("**عقوبةٌ أكبرُ من الرصيد رُدّت**: %v — والقرارُ أن يصير الباقي ديناً", err)
	}
	if got := openDebt(t, h, "driver", drv.ID); got != penalty {
		t.Fatalf("دينُ السائق %d والمنتظَر %d", got, penalty)
	}
	var cause, src string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT cause, COALESCE(source_ref, '') FROM financial_obligations
		 WHERE party_kind = 'driver' AND party_id = $1::uuid`, drv.ID).Scan(&cause, &src); err != nil {
		t.Fatal(err)
	}
	if cause != "penalty_remainder" || src != e.ID {
		t.Errorf("السببُ %q والمصدرُ %q — والمنتظَر باقي عقوبة من %s", cause, src, e.ID)
	}
	if n := noticeCount(t, h, drv.ID, "صار عليك دين"); n != 1 {
		t.Errorf("إشعاراتُ نشأة الدين %d والمنتظَر واحد", n)
	}

	deliverOrder(t, h, oid, drv)

	if got := openDebt(t, h, "driver", drv.ID); got != 0 {
		t.Errorf("**بقي على السائق %d بعد أجرٍ قدرُه %d** — والدينُ ينقطع كاملاً", got, fee)
	}
	var net int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'driver_earning' AND user_id = $2::uuid`, oid, drv.ID).
		Scan(&net); err != nil {
		t.Fatal(err)
	}
	if net != fee-penalty {
		t.Errorf("صافي أجره على الطلب %d والمنتظَر %d", net, fee-penalty)
	}
	if n := noticeCount(t, h, drv.ID, "انقطع من دينك"); n != 1 {
		t.Errorf("إشعاراتُ الاقتطاع %d والمنتظَر واحد", n)
	}
	assertNewViolations(t, h, base, debtsFI...)
}

// ══════════════════════════════════════════════════════════════════════
// **٢ · تعويضٌ لسائقٍ عليه دين يسدّ الدينَ أوّلاً**
// ══════════════════════════════════════════════════════════════════════
func TestDEBT_CompensationPaysDriverDebtFirst(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, debtsFI...)
	admin := h.NewUser("admin")

	oid := newOrderFor(t, h, h.NewItem(1000))
	drv := h.driverOf(oid)
	deliverOrder(t, h, oid, drv)
	before, _ := wallet.NewService(h.Pool).Balance(ctxBG(), drv.ID)

	legacyDebt(t, h, "driver", drv.ID, 700)

	ctx := ctxBG()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id, err := orders.ProposeCompensationTx(ctx, tx, orders.CompensationProposal{
		Kind: orders.CompKindDriver, OrderID: oid, BeneficiaryID: drv.ID,
		Fault: orders.FaultPlatform, Reason: "other", Amount: 2000,
	})
	if err != nil {
		t.Fatalf("اقتراحُ التعويض: %v", err)
	}
	c, err := h.Orders.CompensationByIDTx(ctx, tx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := h.Orders.ApproveCompensationTx(ctx, tx, c, 2000, admin.ID, "", false); err != nil {
		t.Fatalf("الموافقة: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	if got := openDebt(t, h, "driver", drv.ID); got != 0 {
		t.Errorf("**بقي على السائق %d بعد تعويضٍ بألفين** — والتعويضُ يسدّ الدينَ أوّلاً", got)
	}
	after, _ := wallet.NewService(h.Pool).Balance(ctxBG(), drv.ID)
	if after-before != 1300 {
		t.Errorf("زادت محفظتُه %d والمنتظَر ١٣٠٠ (ألفان ناقص دينٍ ٧٠٠)", after-before)
	}
	var back int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(t.amount), 0) FROM wallet_transactions t
		  JOIN wallets w ON w.user_id = t.user_id
		 WHERE w.is_treasury AND t.kind = 'platform_profit' AND t.ref = $1`, id).Scan(&back)
	if back != 700 {
		t.Errorf("عاد إلى الخزينة %d بمرجع التعويض والمنتظَر ٧٠٠", back)
	}
	if n := noticeCount(t, h, drv.ID, "انقطع من دينك"); n != 1 {
		t.Errorf("إشعاراتُ الاقتطاع %d والمنتظَر واحد", n)
	}
	assertNewViolations(t, h, base, debtsFI...)
}

// ══════════════════════════════════════════════════════════════════════
// **٥ · مالكُ متجرين: عكسُ المستحقّ يقع على متجر الطلب**
// ══════════════════════════════════════════════════════════════════════
func TestDEBT_ReverseCommissionsDebtOnOrderStore(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	owner := f.NewUserWith("merchant")
	first := f.Merchant(OwnedBy(owner))
	second := f.Merchant(OwnedBy(owner))
	item := h.NewItemFor(second, 5000)

	oid := newOrderFor(t, h, item)
	deliverOrder(t, h, oid, h.driverOf(oid))
	earned := merchantEarned(t, h, oid)
	if earned <= 0 {
		t.Skip("لم يُقيَّد مستحقٌّ للمتجر")
	}
	f.Credit(owner.ID, -earned, "payout")
	if code := refundOrder(t, h, oid); code >= 400 {
		t.Fatalf("الاسترداد: %d", code)
	}
	if got := openDebt(t, h, "merchant", first.ID); got != 0 {
		t.Errorf("**الدينُ وقع على المتجر الأوّل (%d) والطلبُ من الثاني**", got)
	}
	if got := openDebt(t, h, "merchant", second.ID); got != earned {
		t.Errorf("دينُ متجر الطلب %d والمنتظَر %d", got, earned)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **· العمولةُ المحجوزةُ حين تُصرف تسدّ دينَ المندوب**
// ══════════════════════════════════════════════════════════════════════
func TestDEBT_ReleaseHeldOffsetsRepDebt(t *testing.T) {
	h := New(t)
	treasury(t, h)
	base := financialBaseline(t, h, debtsFI...)
	admin := h.NewUser("admin")
	rep, item := repFixture(t, h)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, rep.ID); err != nil {
		t.Fatal(err)
	}
	oid := placeAndDeliver(t, h, item)
	var held int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM rep_held_commissions
		 WHERE rep_id = $1::uuid AND order_id = $2::uuid AND status = 'held'`, rep.ID, oid).Scan(&held)
	if held < 2 {
		t.Skip("لم تُحجز عمولة")
	}
	debt := held / 2
	legacyDebt(t, h, "rep", rep.ID, debt)
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE users SET status = 'active' WHERE id = $1::uuid`, rep.ID); err != nil {
		t.Fatal(err)
	}

	ctx := ctxBG()
	tx, err := h.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, _, err := h.Orders.ReleaseHeldCommissionsTx(ctx, tx, rep.ID, admin.ID); err != nil {
		t.Fatalf("صرفُ المحجوز: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if got := openDebt(t, h, "rep", rep.ID); got != 0 {
		t.Errorf("**بقي على المندوب %d بعد صرف عمولةٍ محجوزةٍ قدرُها %d**", got, held)
	}
	var net int64
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'commission' AND user_id = $2::uuid`, oid, rep.ID).Scan(&net)
	if net != held-debt {
		t.Errorf("صافي عمولته %d والمنتظَر %d", net, held-debt)
	}
	assertNewViolations(t, h, base, debtsFI...)
}
