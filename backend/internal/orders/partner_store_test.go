package orders_test

// ══════════════════════════════════════════════════════════════════════
// **مشترياتٌ من متجرٍ تابعٍ إلنا** — قرارُ المالك ٢٠٢٦-١٠-١٥
// ══════════════════════════════════════════════════════════════════════
//
// مثالُ المالك: ثمنٌ ٥٠٠ ونسبةُ المتجر ١٠٪ وأجرةٌ ١٠٠ كلُّها للسائق، ومندوبٌ ١٠٪ من العمولة.
//
//	الزبونُ يدفع ٦٠٠ · السائقُ يدفع للمتجر ٤٥٠ · صندوقُه ١٥٠ (عمولة ٥٠ + أجرة ١٠٠)
//	محفظتُه ١٠٠ · المندوبُ ٥ · الخزينةُ ٤٥

import (
	"context"
	"errors"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

// ptMerchant **متجرٌ تابعٌ فعّالٌ بنسبته ومندوبه.**
func ptMerchant(t *testing.T, pct int64, repID string) string {
	t.Helper()
	pool := testdb.Pool(t)
	ctx := context.Background()
	var cat, id string
	if err := pool.QueryRow(ctx, `SELECT id::text FROM categories LIMIT 1`).Scan(&cat); err != nil {
		if err := pool.QueryRow(ctx, `INSERT INTO categories (name, active) VALUES ('تصنيفُ التابع', true)
			RETURNING id::text`).Scan(&cat); err != nil {
			t.Fatalf("تصنيف: %v", err)
		}
	}
	var rep *string
	if repID != "" {
		rep = &repID
	}
	if err := pool.QueryRow(ctx, `
		INSERT INTO merchants (name, category_id, commission_percent, status, sales_rep_user_id)
		VALUES ('متجرٌ تابع', $1, $2, 'active', $3) RETURNING id::text`, cat, pct, rep).Scan(&id); err != nil {
		t.Fatalf("متجر: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `UPDATE orders SET partner_merchant_id = NULL,
			partner_commission_percent = NULL WHERE partner_merchant_id = $1`, id)
		_, _ = pool.Exec(context.Background(), `DELETE FROM merchants WHERE id = $1`, id)
	})
	return id
}

// ptDeliver **يمرّ بالطلب من الاستلام إلى التسليم.**
func ptDeliver(t *testing.T, svc *orders.Service, driver, orderID string) {
	t.Helper()
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff", "delivered"} {
		if _, err := svc.Transition(context.Background(), driver, []string{"driver"}, orderID, to, ""); err != nil {
			t.Fatalf("الانتقال إلى %s: %v", to, err)
		}
	}
}

// ptAdvanceReturn **سطرُ «سلفة رجعت للمكتب» للطلب.**
func ptAdvanceReturn(t *testing.T, orderID string) int64 {
	t.Helper()
	var v int64
	if err := testdb.Pool(t).QueryRow(context.Background(), `
		SELECT COALESCE(sum(amount), 0) FROM office_cash_entries
		 WHERE ref = $1 AND source = 'advance_return' AND direction = 'out'`, orderID).Scan(&v); err != nil {
		t.Fatalf("السلفة: %v", err)
	}
	return v
}

// ptFixture **الإعدادُ المشترك**: أجرةٌ كلُّها للسائق، ومندوبٌ ١٠٪ في لقطة الطلب.
func ptFixture(t *testing.T, payment string) (svc *orders.Service, balance func(string) int64,
	customer, driver, rep, orderID string) {
	t.Helper()
	svc, w, customer, driver, exec := cqSetup(t)
	ctx := context.Background()
	cqArmTreasury(t)
	exec(ctx, `UPDATE app_settings SET value = '0'::jsonb WHERE key = 'delivery.share_purchase'`)
	ptFeeBounds(t, exec)
	rep = testdb.NewUser(t, testdb.Pool(t), "sales")
	orderID = cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	exec(ctx, `UPDATE orders SET snap_rep_commission_percent = 10 WHERE id = $1`, orderID)
	if payment == "wallet" {
		cqDeposit(t, customer, 10_000)
	}
	balance = func(u string) int64 {
		b, _ := w.Balance(ctx, u)
		return b
	}
	return svc, balance, customer, driver, rep, orderID
}

// ── أ · نقداً: مثالُ المالك حرفاً ──────────────────────────────────────
func TestPartnerPurchase_Cash_OwnerExample(t *testing.T) {
	svc, balance, customer, driver, rep, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	merchant := ptMerchant(t, 10, rep)

	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); err != nil {
		t.Fatalf("الربط: %v", err)
	}
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 600, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	// **والسائقُ يرى ما يدفعه للمتجر** — الثمنُ ناقصَ العمولة.
	o, err := svc.GetByID(ctx, orderID)
	if err != nil {
		t.Fatalf("القراءة: %v", err)
	}
	if o.PayMerchant != 450 {
		t.Fatalf("ادفع للمتجر=%d — المنتظَر 450", o.PayMerchant)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if got := cqHeld(t, driver) - heldBefore; got != 150 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 150 (عمولة ٥٠ + أجرة ١٠٠)", got)
	}
	if got := balance(driver); got != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100", got)
	}
	if got := balance(rep); got != 5 {
		t.Fatalf("المندوب=%d — المنتظَر 5 (١٠٪ من عمولة ٥٠)", got)
	}
	if got := cqTreasury(t); got != 45 {
		t.Fatalf("الخزينة=%d — المنتظَر 45 (العمولةُ ٥٠ ناقصَ المندوب)", got)
	}
	var commission, storePaid int64
	if err := testdb.Pool(t).QueryRow(ctx, `SELECT platform_commission, custom_store_paid
		FROM orders WHERE id = $1`, orderID).Scan(&commission, &storePaid); err != nil {
		t.Fatalf("الطلب: %v", err)
	}
	if commission != 50 || storePaid != 450 {
		t.Fatalf("العمولة=%d والمدفوعُ للمتجر=%d — المنتظَر 50 و450", commission, storePaid)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 0 {
		t.Fatalf("سلفةٌ رجعت=%d — والزبونُ دفع نقداً يغطّيها", adv)
	}
}

// ── ب · زبونُ المحفظة: السلفةُ تُردّ من الدرج ─────────────────────────────
func TestPartnerPurchase_Wallet_AdvanceReturn(t *testing.T) {
	svc, balance, customer, driver, rep, orderID := ptFixture(t, "wallet")
	ctx := context.Background()
	merchant := ptMerchant(t, 10, rep)

	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); err != nil {
		t.Fatalf("الربط: %v", err)
	}
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "wallet", 600, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if got := cqHeld(t, driver) - heldBefore; got != 0 {
		t.Fatalf("صندوقُ السائق تحرّك %d — ولم يقبض شيئاً", got)
	}
	if adv := ptAdvanceReturn(t, orderID); adv != 450 {
		t.Fatalf("سلفةٌ رجعت=%d — المنتظَر 450", adv)
	}
	if got := balance(driver); got != 100 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 100", got)
	}
	if got := balance(rep); got != 5 {
		t.Fatalf("المندوب=%d — المنتظَر 5", got)
	}
	if got := cqTreasury(t); got != 45 {
		t.Fatalf("الخزينة=%d — المنتظَر 45", got)
	}
}

// ── ج · مشترياتٌ بلا متجرٍ تابع: الصندوقُ الأجرةُ وحدَها، والربحُ كما كان ──
func TestPurchaseNoPartner_CashCollectsFeeOnly(t *testing.T) {
	svc, w, customer, driver, exec := cqSetup(t) // حصّةُ المنصّة ١٠٪ من الأجرة
	ctx := context.Background()
	ptFeeBounds(t, exec)
	cqArmTreasury(t)
	orderID := cqSeedOrder(t, w, customer, driver, "driver_defined", nil, true)
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	if _, err := svc.ConfirmQuote(ctx, orderID, customer, "cash", 600, 1); err != nil {
		t.Fatalf("التأكيد: %v", err)
	}
	heldBefore := cqHeld(t, driver)
	ptDeliver(t, svc, driver, orderID)

	if got := cqHeld(t, driver) - heldBefore; got != 100 {
		t.Fatalf("صندوقُ السائق=%d — المنتظَر 100 (الأجرةُ وحدَها؛ الثمنُ دفعه للمتجر)", got)
	}
	// **والربحُ كما كان**: ٦٠٠ − ٥٠٠ بضاعة − ٩٠ للسائق = ١٠.
	if got := cqTreasury(t); got != 10 {
		t.Fatalf("الخزينة=%d — المنتظَر 10 (كما كانت)", got)
	}
	if b, _ := w.Balance(ctx, driver); b != 90 {
		t.Fatalf("محفظةُ السائق=%d — المنتظَر 90", b)
	}
}

// ── د · لا ربطَ بعد توثيق الثمن ─────────────────────────────────────────
func TestPartnerLink_RefusedAfterGoods(t *testing.T) {
	svc, _, customer, driver, rep, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	merchant := ptMerchant(t, 10, rep)
	if err := svc.AgreeCustom(ctx, orderID, driver, 500, 100); err != nil {
		t.Fatalf("الاتّفاق: %v", err)
	}
	_, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil)
	if !errors.Is(err, orders.ErrPartnerTooLate) {
		t.Fatalf("الربطُ بعد التوثيق=%v — المنتظَر partner_goods_documented", err)
	}
	// **والفكُّ كذلك.**
	if _, _, err := svc.LinkPartner(ctx, customer, orderID, "", nil); !errors.Is(err, orders.ErrPartnerTooLate) {
		t.Fatalf("الفكُّ بعد التوثيق=%v — المنتظَر partner_goods_documented", err)
	}
}

// ── هـ · والمتجرُ غيرُ الفعّال لا يُربط ──────────────────────────────────
func TestPartnerLink_InactiveRefused(t *testing.T) {
	svc, _, customer, _, rep, orderID := ptFixture(t, "cash")
	ctx := context.Background()
	merchant := ptMerchant(t, 10, rep)
	if _, err := testdb.Pool(t).Exec(ctx, `UPDATE merchants SET status = 'inactive' WHERE id = $1`, merchant); err != nil {
		t.Fatalf("تعطيل: %v", err)
	}
	if _, _, err := svc.LinkPartner(ctx, customer, orderID, merchant, nil); !errors.Is(err, orders.ErrPartnerInactive) {
		t.Fatalf("ربطُ متجرٍ غيرِ فعّال=%v — المنتظَر partner_merchant_inactive", err)
	}
}

// ptFeeBounds **حدّا الأجرة يتّسعان لمثال المالك** (أجرةٌ ١٠٠) — ويُعادان بعده.
func ptFeeBounds(t *testing.T, exec func(ctx context.Context, sql string, args ...any)) {
	t.Helper()
	exec(context.Background(), `INSERT INTO app_settings (key, value) VALUES
		('delivery.custom_fee_min', '0'::jsonb), ('delivery.custom_fee_max', '1000000'::jsonb)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value`)
	t.Cleanup(func() {
		_, _ = testdb.Pool(t).Exec(context.Background(), `DELETE FROM app_settings
			WHERE key IN ('delivery.custom_fee_min', 'delivery.custom_fee_max')`)
	})
}
