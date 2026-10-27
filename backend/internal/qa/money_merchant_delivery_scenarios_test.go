package qa

// ══════════════════════════════════════════════════════════════════════
// **مالُ «لدي توصيلة» — السيناريوهاتُ التي لم يحرسها أحد**
// ══════════════════════════════════════════════════════════════════════
//
// **حرّاسُ العائلة القائمة** (`merchant_delivery_test.go` · `merchant_delivery_1015_test.go` ·
// `merchant_delivery_approval_test.go` · `merchant_delivery_fetch_amount_test.go`) **تقيس
// المتجرَ والسائقَ ولا تقيس الخزينة**، **ولا تمشي «إحضار» بمالٍ**، ولا الحصّةَ الثابتة،
// ولا التوصيلَ المجانيَّ حتّى التسليم، ولا إسقاطَ الدين القديم `merchant_delivery_fee`.
//
// **والأرقامُ كلُّها من `newMDFx`**: الأجرةُ ٥٠٠ ليرة جديدة، وحصّةُ المنصّة ١٠٪ ⇒ السائقُ
// ٤٥٠ والمنصّةُ ٥٠. **والمالُ يُقاس من الصفوف، وثوابتُ الدفتر تُصوَّر قبل السيناريو.**

import (
	"net/http"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// mdmProfit **ما قُيّد للخزينة ربحاً عن التوصيلة** — `platform_profit` بمرجعها.
func mdmProfit(t *testing.T, h *Harness, id string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		 WHERE ref = $1 AND kind = 'platform_profit'`, id).Scan(&n); err != nil {
		t.Fatalf("ربحُ المنصّة: %v", err)
	}
	return n
}

// mdmCreate **توصيلةٌ بوضعٍ ودافع** — مقبولةٌ من المكتب، وسائقُها.
func mdmCreate(t *testing.T, h *Harness, m mdFx, payer, mode, key string) (string, *User) {
	t.Helper()
	body := m.body(payer)
	if mode != "" {
		body["delivery_mode"] = mode
	}
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq(key), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	m.accept(t, h, id)
	return id, h.driverOf(id)
}

// mdmDeliver **من «عند الباب» إلى «سُلّمت»** — بإذن استثناء الصورة.
func mdmDeliver(t *testing.T, h *Harness, id string, drv *User) {
	t.Helper()
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
}

// TestMDM01_MerchantWalletTreasuryGetsShare **المتجرُ يدفع من محفظته ⇒ السائقُ ٤٥٠ والخزينةُ ٥٠.**
//
// **والقائمُ يقيس السائقَ والمتجرَ وحدَهما** — وخزينةٌ لا يقيسها أحدٌ تخطئ بصمت.
func TestMDM01_MerchantWalletTreasuryGetsShare(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-01")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "merchant", "", "mdm01")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)

	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("محفظةُ المتجر %d والمنتظَر 1500", got)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	if got := mdmProfit(t, h, id); got != 50 {
		t.Fatalf("**ربحُ المنصّة %d والمنتظَر 50** (٥٠٠ − ٤٥٠)", got)
	}
	if sum, rows := mdCollected(t, h, id); sum != 0 || rows != 0 {
		t.Fatalf("**نقدٌ في الصندوق (%d في %d قيد) والمتجرُ دفع من محفظته**", sum, rows)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM02_FetchRecipientPaysAtPickup **«إحضار» والمستلِمُ يدفع ⇒ يُقبض منه عند الاستلام**
// (هو الواقفُ عند نقطة الاستلام — `cashAtPickup`) **ولا يُقبض ثانيةً عند التسليم.**
func TestMDM02_FetchRecipientPaysAtPickup(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-02")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "fetch", "mdm02")

	mdDriveTo(t, h, id, drv, "at_pickup")
	if sum, _ := mdCollected(t, h, id); sum != 0 {
		t.Fatalf("قُبض %d قبل الاستلام", sum)
	}
	mdDriveTo(t, h, id, drv, "picked_up")
	if sum, rows := mdCollected(t, h, id); sum != 500 || rows != 1 {
		t.Fatalf("**بعد الاستلام الصندوقُ %d في %d قيد** — والمنتظَرُ 500 في قيد", sum, rows)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if sum, rows := mdCollected(t, h, id); sum != 500 || rows != 1 {
		t.Fatalf("**بعد التسليم الصندوقُ %d في %d قيد** — قُبض مرّتين", sum, rows)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	if got := mdmProfit(t, h, id); got != 50 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 50", got)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("مُسّت محفظةُ المتجر: %d", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM03_FetchMerchantCashPaysAtDelivery **«إحضار» و«أنا نقداً» ⇒ المتجرُ يدفع عند التسليم**
// (الغرضُ يصله هو — فالوقفتان مقلوبتان) **ولا شيءَ يُقبض عند الاستلام.**
func TestMDM03_FetchMerchantCashPaysAtDelivery(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-03")
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "merchant_cash", "fetch", "mdm03")

	mdDriveTo(t, h, id, drv, "picked_up")
	if sum, _ := mdCollected(t, h, id); sum != 0 {
		t.Fatalf("**قُبض %d من المتجر عند الاستلام والمتجرُ ليس هناك**", sum)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if sum, rows := mdCollected(t, h, id); sum != 500 || rows != 1 {
		t.Fatalf("بعد التسليم الصندوقُ %d في %d قيد — والمنتظَرُ 500 في قيد", sum, rows)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	if got := mdmProfit(t, h, id); got != 50 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 50", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM04_FixedPlatformShare **حصّةٌ ثابتةٌ ١٠٠ ليرة** (قرارُ المالك ٢٠٢٦-١٠-١٣) ⇒ السائقُ ٤٠٠
// والخزينةُ ١٠٠ — **لا ١٠٪.**
func TestMDM04_FixedPlatformShare(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-04")
	h.Setting("delivery.share_merchant_mode", `"fixed"`)
	h.Setting(mdPct, "100")
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "", "mdm04")
	var driverFee, fixed int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT driver_fee, snap_platform_delivery_fixed
		FROM orders WHERE id = $1::uuid`, id).Scan(&driverFee, &fixed); err != nil {
		t.Fatal(err)
	}
	if driverFee != 400 || fixed != 100 {
		t.Fatalf("أجرُ السائق %d · الحصّةُ الملتقَطة %d — والمنتظَر 400 و100", driverFee, fixed)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if got := walletOf(t, h, drv.ID); got != 400 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 400", got)
	}
	if got := mdmProfit(t, h, id); got != 100 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 100", got)
	}
	if sum, _ := mdCollected(t, h, id); sum != 500 {
		t.Fatalf("الصندوقُ %d والمنتظَر 500 من المستلِم", sum)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM05_FreeDeliveryTreasuryPaysDriver **توصيلٌ مجانيٌّ للمتجر** (قرارُ المالك ٢٠٢٦-١٠-٢٣)
// ⇒ لا نقدَ يُقبض، والسائقُ بأجره كاملاً ٤٥٠، **والخزينةُ تتحمّله: −٤٥٠.**
func TestMDM05_FreeDeliveryTreasuryPaysDriver(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-05")
	t.Cleanup(func() { _, _ = h.Pool.Exec(ctxBG(), `DELETE FROM free_delivery_campaigns`) })
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`)
	now := time.Now().UTC()
	if r := h.POST("/api/v1/admin/free-delivery", m.admin.Token, map[string]any{
		"active": true, "starts_at": now.Add(-time.Hour), "ends_at": now.Add(24 * time.Hour),
		"kinds": []string{"merchant"},
	}); r.Code != http.StatusCreated {
		t.Fatalf("إنشاءُ العرض: %d / %s", r.Code, r.Err())
	}
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "", "mdm05")
	var fee, cash, waived int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, cash_due, promo_delivery_waived
		FROM orders WHERE id = $1::uuid`, id).Scan(&fee, &cash, &waived); err != nil {
		t.Fatal(err)
	}
	if fee != 0 || cash != 0 || waived != 500 {
		t.Fatalf("أجرة %d · نقد %d · معفى %d — والمنتظَر 0 · 0 · 500", fee, cash, waived)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdmDeliver(t, h, id, drv)
	if sum, rows := mdCollected(t, h, id); sum != 0 || rows != 0 {
		t.Fatalf("**قُبض %d نقداً على توصيلةٍ مجانيّة**", sum)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450 — بأجره كاملاً", got)
	}
	if got := mdmProfit(t, h, id); got != -450 {
		t.Fatalf("**الخزينةُ %d والمنتظَر −450** — هي من تحمّل العرض", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM06_LegacyDeliveryDebtVoidedOnCancel **دينُ `merchant_delivery_fee` القديمُ يُسقَط**
// إن أُلغيت توصيلتُه قبل الاستلام — **دينٌ على توصيلةٍ لم تقع باطل.** (لا دينَ جديداً منه منذ
// ٢٠٢٦-١٠-١٥؛ فيُكتب هنا بيدٍ كما كتبته الشيفرةُ القديمة: المتجرُ دافعٌ ولم تُخصم محفظتُه.)
func TestMDM06_LegacyDeliveryDebtVoidedOnCancel(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-06")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id, _ := mdmCreate(t, h, m, "recipient", "", "mdm06")
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE orders SET fee_payer = 'merchant', cash_due = 0 WHERE id = $1::uuid`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := obligations.Create(ctxBG(), h.Pool, obligations.PartyMerchant, m.fx.M.ID, 500,
		obligations.CauseDeliveryFee, id, nil); err != nil {
		t.Fatal(err)
	}
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1::uuid`, m.fx.M.ID).Scan(&debt)
	if debt != 500 {
		t.Fatalf("الدينُ القديمُ %d والمنتظَر 500", debt)
	}
	if c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1::uuid`, m.fx.M.ID).Scan(&debt)
	if debt != 0 {
		t.Fatalf("**الدينُ %d بعد الإلغاء والمنتظَر 0** — لم يُسقَط", debt)
	}
	var open int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM financial_obligations
		WHERE order_id = $1::uuid AND cause = 'merchant_delivery_fee' AND closed_at IS NULL`, id).Scan(&open); err != nil {
		t.Fatal(err)
	}
	if open != 0 {
		t.Fatalf("**التزامٌ مفتوحٌ (%d) على توصيلةٍ أُلغيت**", open)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("**رُدّ للمتجر %d ولم يدفع شيئاً**", got)
	}
	if got := mdmProfit(t, h, id); got != 0 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 0", got)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM07_MerchantCashRecipientFaultKeepsCash **«أنا نقداً» وفشلٌ بذنب المستلِم ⇒ نقدُ الاستلام
// باقٍ، ولا ردَّ ولا دين، والسائقُ ٤٥٠ والخزينةُ ٥٠** (`merchantBearsFailedDelivery`).
func TestMDM07_MerchantCashRecipientFaultKeepsCash(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-07")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "merchant_cash", "", "mdm07")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "customer", "customer_action": "none"})

	if sum, rows := mdCollected(t, h, id); sum != 500 || rows != 1 {
		t.Fatalf("الصندوقُ %d في %d قيد — والمنتظَرُ 500 في قيد", sum, rows)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("**رُدّ للمتجر %d والذنبُ ذنبُ المستلِم**", got)
	}
	if d := mdObligations(t, h, "merchant", m.fx.M.ID, "delivery_failed_recipient", id); d != 0 {
		t.Fatalf("**دينٌ %d والمتجرُ دفع نقداً**", d)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	if got := mdmProfit(t, h, id); got != 50 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 50", got)
	}
	if n, _ := mdCompRequests(t, h, id); n != 0 {
		t.Fatalf("طلبُ تعويضٍ (%d) والمتجرُ حمل الأجرة", n)
	}
	mdaNoNewViolations(t, h, base)
}

// mdmReturned **ما ردّه سائقٌ للمستلِم من صندوقه عن التوصيلة** — موجباً — وعددُ سطوره.
func mdmReturned(t *testing.T, h *Harness, id string) (sum int64, rows int, driverID string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT -COALESCE(sum(amount), 0), count(*), COALESCE(max(driver_id::text), '')
		  FROM driver_cash_entries WHERE ref = $1 AND kind = 'recipient_cash_returned'`, id).
		Scan(&sum, &rows, &driverID); err != nil {
		t.Fatalf("ردُّ نقد المستلِم: %v", err)
	}
	return sum, rows, driverID
}

// mdmHeldFor **صافي ما بذمّة السائق عن التوصيلة** — كلُّ سطور صندوقه بمرجعها.
func mdmHeldFor(t *testing.T, h *Harness, id string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM driver_cash_entries WHERE ref = $1`, id).Scan(&n); err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	return n
}

// TestMDM08_FetchRecipientCashPlatformFault **«إحضار» والمستلِمُ دفع عند الاستلام ثمّ فشلت
// بذنب المنصّة ⇒ يُردّ نقدُه له** (قرارُ المالك ٢٠٢٦-١٠-٢٧).
//
// **كان نقدُ المستلِم يبقى في الصندوق وتقرؤه الخزينةُ ربحاً (٥٠٠)** — والمستلِمُ لم يُخطئ.
// وصار: سطرٌ سالبٌ بـ٥٠٠ في صندوق السائق نفسِه، **وربحُ المنصّة صفر**، والسائقُ بطلب تعويض.
func TestMDM08_FetchRecipientCashPlatformFault(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-08")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "fetch", "mdm08")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "platform"})

	if sum, _ := mdCollected(t, h, id); sum != 500 {
		t.Fatalf("الصندوقُ %d والمنتظَر 500 قُبضت من المستلِم عند الاستلام", sum)
	}
	if sum, rows, who := mdmReturned(t, h, id); sum != 500 || rows != 1 || who != drv.ID {
		t.Fatalf("**ردُّ نقد المستلِم** %d في %d سطر من %q — والمنتظَرُ 500 في سطرٍ واحدٍ من السائق", sum, rows, who)
	}
	if got := mdmHeldFor(t, h, id); got != 0 {
		t.Fatalf("بقي بذمّة السائق %d عن توصيلةٍ ردّ نقدَها", got)
	}
	if got := mdmProfit(t, h, id); got != 0 {
		t.Fatalf("**ربحُ المنصّة %d من نقد مستلِمٍ لم يُخطئ** — والمنتظَر 0", got)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("مُسّت محفظةُ المتجر: %d", got)
	}
	if got := walletOf(t, h, drv.ID); got != 0 {
		t.Fatalf("السائقُ قبض %d والمنتظَر 0 — تعويضُه بطلب", got)
	}
	if n, fault := mdCompRequests(t, h, id); n != 1 || fault != "platform" {
		t.Fatalf("طلباتُ تعويض %d بذنب %q — والمنتظَرُ واحدٌ بذنب platform", n, fault)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM08b_FetchRecipientCashDriverFault **وبذنب السائق كذلك يُردّ نقدُ المستلِم** —
// ولا أجرَ ولا تعويضَ ولا دينَ على السائق (لم تخسر المنصّةُ شيئاً: ما قبضته ردّته).
func TestMDM08b_FetchRecipientCashDriverFault(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-08b")
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "fetch", "mdm08b")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "driver", "driver_share": "full"})

	if sum, rows, who := mdmReturned(t, h, id); sum != 500 || rows != 1 || who != drv.ID {
		t.Fatalf("ردُّ نقد المستلِم %d في %d سطر من %q — والمنتظَرُ 500 من السائق", sum, rows, who)
	}
	if got := mdmHeldFor(t, h, id); got != 0 {
		t.Fatalf("بقي بذمّة السائق %d", got)
	}
	if got := mdmProfit(t, h, id); got != 0 {
		t.Fatalf("ربحُ المنصّة %d — والمنتظَر 0", got)
	}
	if got := walletOf(t, h, drv.ID); got != 0 {
		t.Fatalf("السائقُ قبض %d وذنبُه", got)
	}
	if n, _ := mdCompRequests(t, h, id); n != 0 {
		t.Fatalf("طلبُ تعويضٍ (%d) بذنب السائق", n)
	}
	if d := mdObligations(t, h, "driver", drv.ID, "failed_order_driver", id); d != 0 {
		t.Fatalf("دينٌ على السائق %d ولا خسارةَ للمنصّة", d)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMDM08c_FetchRecipientCashRecipientFault **وبذنب المستلِم يبقى كما كان**: نقدُه للسائق
// (٤٥٠) والمنصّة (٥٠) كأنّها سُلّمت، **ولا ردّ.**
func TestMDM08c_FetchRecipientCashRecipientFault(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MDM-08c")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id, drv := mdmCreate(t, h, m, "recipient", "fetch", "mdm08c")
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "customer", "customer_action": "none"})

	if _, rows, _ := mdmReturned(t, h, id); rows != 0 {
		t.Fatalf("رُدّ نقدُ المستلِم (%d سطر) والذنبُ ذنبُه", rows)
	}
	if got := mdmHeldFor(t, h, id); got != 500 {
		t.Fatalf("بذمّة السائق %d والمنتظَر 500", got)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	if got := mdmProfit(t, h, id); got != 50 {
		t.Fatalf("ربحُ المنصّة %d والمنتظَر 50", got)
	}
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("مُسّت محفظةُ المتجر: %d — المستلِمُ دفع", got)
	}
	mdaNoNewViolations(t, h, base)
}
