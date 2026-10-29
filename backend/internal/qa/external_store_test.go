package qa

// ══════════════════════════════════════════════════════════════════════
// **«من متجر غير تابع لنا»** — قرارُ المالك ٢٠٢٦-١٠-٢٩
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ عاديٌّ نقديٌّ يُقلب طلبَ مشترياتٍ (`orders/external_store.go`)، ومن بعدها يمشي في
// مسار الخاصّ القائم: الأجرةُ تثبّتها المنصّة، والسائقُ يوثّق الثمن، والزبونُ يؤكّد.
// **ولا منطقَ ماليَّ جديد** — فالفحصُ الماليُّ هنا أنّ المسارَ القائمَ يحسبه صحيحاً.

import (
	"strings"
	"testing"
)

// extStoreSetup **إعداداتٌ معلومةٌ** — أجرةٌ ٥٠٠ وعمولةٌ ١٠٪ ولا هامشَ ولا عرضَ توصيلٍ مجانيّ.
func extStoreSetup(h *Harness) {
	waiverSettings(h)
	h.Setting("delivery.share_purchase", "0")
}

// extStoreOrder **طلبٌ عاديٌّ نقديٌّ من متجرٍ بصنفٍ واحد** — في الحال المطلوبة.
// يردّ الطلبَ ومتجرَه وسائقَه (إن كان).
func extStoreOrder(t *testing.T, h *Harness, status string, body map[string]any, m *Merchant) (string, *User) {
	t.Helper()
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("extstore"), body)
	if made.Code >= 400 {
		t.Fatalf("إنشاءُ الطلب: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	var drv *User
	switch status {
	case "pending":
	case "accepted", "preparing":
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = $2, accepted_at = now()
			WHERE id = $1::uuid`, oid, status); err != nil {
			t.Fatal(err)
		}
	case "dispatching":
		if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'dispatching', accepted_at = now(),
			dispatched_at = now() WHERE id = $1::uuid`, oid); err != nil {
			t.Fatal(err)
		}
	case "assigned", "at_pickup", "picked_up":
		drv = h.driverOf(oid)
		if status != "assigned" {
			if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = $2 WHERE id = $1::uuid`,
				oid, status); err != nil {
				t.Fatal(err)
			}
		}
	default:
		t.Fatalf("حالٌ غيرُ معروفة %q", status)
	}
	return oid, drv
}

func extStorePost(h *Harness, tok, oid, reason string) Res {
	return h.POST("/api/v1/admin/orders/"+oid+"/external-store", tok, map[string]any{"reason": reason})
}

// TestEXTSTORE_001_EachAllowedStatusConverts **كلُّ حالٍ مسموحةٍ تُقلب بأعمدتها الصحيحة.**
func TestEXTSTORE_001_EachAllowedStatusConverts(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	want := map[string]string{
		"accepted": "dispatching", "preparing": "dispatching", "dispatching": "dispatching",
		"assigned": "assigned", "at_pickup": "assigned",
	}
	f := h.Factory()
	for from, to := range want {
		t.Run(from, func(t *testing.T) {
			m := f.Merchant(Commission(10))
			item := h.NewItemFor(m, 1000)
			oid, drv := extStoreOrder(t, h, from, orderBody(item, 2), m)

			res := extStorePost(h, ops.Token, oid, "  المتجر مسكّر  ")
			if res.Code != 200 {
				t.Fatalf("القلبُ رُدّ: %s", res)
			}
			if view, _ := res.JSON()["order"].(map[string]any); view == nil || view["kind"] != "custom" ||
				view["ext_store_merchant_name"] != m.Name {
				t.Errorf("ردُّ المكتب لا يحمل النوعَ والمتجرَ الأوّل: %v", res.JSON()["order"])
			}

			var kind, mode, status, request, source, reason string
			var merchant, extMerchant, extBy *string
			var snapshot *int64
			var mayChange, dispatched bool
			var subtotal, fee, total, cashDue, commission, waived int64
			if err := h.Pool.QueryRow(ctxBG(), `
				SELECT kind, custom_mode, status, custom_request, custom_fee_source, ext_store_reason,
				       merchant_id::text, ext_store_merchant_id::text, ext_store_by::text,
				       custom_fee_snapshot, custom_driver_may_change_fee, dispatched_at IS NOT NULL,
				       subtotal, delivery_fee, total, cash_due, platform_commission, promo_delivery_waived
				  FROM orders WHERE id = $1::uuid`, oid).Scan(&kind, &mode, &status, &request, &source,
				&reason, &merchant, &extMerchant, &extBy, &snapshot, &mayChange, &dispatched,
				&subtotal, &fee, &total, &cashDue, &commission, &waived); err != nil {
				t.Fatal(err)
			}
			if kind != "custom" || mode != "purchase" || status != to {
				t.Errorf("النوع %s/%s والحال %s — يُنتظر custom/purchase و%s", kind, mode, status, to)
			}
			if merchant != nil || extMerchant == nil || *extMerchant != m.ID || extBy == nil || *extBy != ops.ID {
				t.Errorf("المتجر %v · الأوّل %v · الفاعل %v", merchant, extMerchant, extBy)
			}
			if reason != "المتجر مسكّر" {
				t.Errorf("السببُ %q", reason)
			}
			if !strings.Contains(request, "2 × "+item.Name) {
				t.Errorf("نصُّ الطلب لا يحمل الصنفَ وكمّيتَه: %q", request)
			}
			if source != "admin_defined" || mayChange || snapshot == nil || *snapshot != 500 {
				t.Errorf("الأجرة: %s %v %v — يُنتظر admin_defined ٥٠٠ بلا إذنٍ للسائق", source, snapshot, mayChange)
			}
			if commission != 0 || waived != 0 {
				t.Errorf("عمولة %d · إعفاء %d — يُنتظر صفران", commission, waived)
			}
			if to == "dispatching" && !dispatched {
				t.Error("نزل الطابورَ بلا dispatched_at")
			}
			var itemStores int
			_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_items
				WHERE order_id = $1::uuid AND merchant_id IS NOT NULL`, oid).Scan(&itemStores)
			if itemStores != 0 {
				t.Errorf("%d صنفاً بقي على متجره", itemStores)
			}
			var notes int
			_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_events
				WHERE order_id = $1::uuid AND note LIKE 'من متجرٍ غير تابع لنا — %'`, oid).Scan(&notes)
			if notes != 1 {
				t.Errorf("حدثُ القلب %d — يُنتظر واحد", notes)
			}
			var audits int
			_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM audit_log
				WHERE action = 'ops.order_external_store' AND entity_id = $1`, oid).Scan(&audits)
			if audits != 1 {
				t.Errorf("أثرُ التدقيق %d — يُنتظر واحد", audits)
			}

			if drv == nil {
				// **قبل السائق: لا شيءَ يُؤكَّد بعد** — الأجرةُ تُثبَّت لحظةَ الإسناد.
				if subtotal != 0 || fee != 0 || total != 0 || cashDue != 0 {
					t.Errorf("الأرقام %d/%d/%d/%d — يُنتظر أصفار", subtotal, fee, total, cashDue)
				}
				return
			}
			// **ومع السائق: الأجرةُ ثُبّتت وأُكّدت، وثمنُ البضاعة ينتظر.**
			var customFee *int64
			var pending, confirmed bool
			if err := h.Pool.QueryRow(ctxBG(), `
				SELECT custom_fee, custom_goods_pending, quote_confirmed_at IS NOT NULL, total
				  FROM orders WHERE id = $1::uuid`, oid).Scan(&customFee, &pending, &confirmed, &total); err != nil {
				t.Fatal(err)
			}
			if customFee == nil || *customFee != 500 || !pending || !confirmed || total != 500 {
				t.Errorf("الأجرةُ الثابتة: %v ينتظر %v مؤكَّد %v مجموع %d", customFee, pending, confirmed, total)
			}
		})
	}
}

// TestEXTSTORE_002_Refusals **ما لا يُقلب يُردّ بخطئه.**
func TestEXTSTORE_002_Refusals(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	f := h.Factory()
	mk := func(status string) string {
		m := f.Merchant(Commission(10))
		oid, _ := extStoreOrder(t, h, status, orderBody(h.NewItemFor(m, 1000), 1), m)
		return oid
	}
	expect := func(name string, res Res, code int, errCode string) {
		t.Helper()
		if res.Code != code || (errCode != "" && res.Err() != errCode) {
			t.Errorf("%s: المنتظَر %d %s — جاء %s", name, code, errCode, res)
		}
	}

	expect("طلبٌ معلَّق", extStorePost(h, ops.Token, mk("pending"), "سبب"), 409, "ext_store_bad_status")
	expect("بعد الاستلام", extStorePost(h, ops.Token, mk("picked_up"), "سبب"), 409, "ext_store_bad_status")

	wallet := mk("accepted")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET payment_method = 'wallet' WHERE id = $1::uuid`, wallet); err != nil {
		t.Fatal(err)
	}
	expect("محفظة", extStorePost(h, ops.Token, wallet, "سبب"), 409, "ext_store_wallet")

	twice := mk("accepted")
	expect("القلبُ الأوّل", extStorePost(h, ops.Token, twice, "سبب"), 200, "")
	expect("قلبٌ ثانٍ", extStorePost(h, ops.Token, twice, "سبب"), 409, "ext_store_not_standard")

	expect("بلا سبب", extStorePost(h, ops.Token, mk("accepted"), "   "), 400, "")
	expect("معرّفٌ تالف", extStorePost(h, ops.Token, "not-a-uuid", "سبب"), 404, "")

	moved := mk("assigned")
	f.CreditRef(h.Customer().ID, 100, "adjustment", moved)
	expect("مالٌ تحرّك", extStorePost(h, ops.Token, moved, "سبب"), 409, "ext_store_money_moved")

	fin := h.NewUser("finance")
	expect("بلا قدرة", extStorePost(h, fin.Token, mk("accepted"), "سبب"), 403, "")

	// **والمرفوضُ لم يُمسّ.**
	var kind string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT kind FROM orders WHERE id = $1::uuid`, moved).Scan(&kind)
	if kind != "standard" {
		t.Errorf("طلبٌ رُدّ قلبُه صار %s", kind)
	}
}

// TestEXTSTORE_003_CouponReleased **الكودُ يعود للزبون — ولا شيءَ في محفظته.**
func TestEXTSTORE_003_CouponReleased(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	code := mpsCode(t, h, "fixed", 100, false)
	m := h.Factory().Merchant(Commission(10))
	body := orderBody(h.NewItemFor(m, 1000), 1)
	body["promo_code"] = code
	oid, _ := extStoreOrder(t, h, "accepted", body, m)

	var used, redemptions int
	var discount int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT used_count FROM promo_codes WHERE code = $1`, code).Scan(&used)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT discount FROM orders WHERE id = $1::uuid`, oid).Scan(&discount)
	if used != 1 || discount != 100 {
		t.Fatalf("الكودُ لم يُستعمل عند الإنشاء: used=%d discount=%d", used, discount)
	}
	res := extStorePost(h, ops.Token, oid, "المتجر ما عنده")
	if res.Code != 200 {
		t.Fatalf("القلبُ رُدّ: %s", res)
	}
	var promo *string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT used_count FROM promo_codes WHERE code = $1`, code).Scan(&used)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM promo_redemptions WHERE order_id = $1::uuid`, oid).Scan(&redemptions)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT promo_code::text, discount FROM orders WHERE id = $1::uuid`, oid).Scan(&promo, &discount)
	if used != 0 || redemptions != 0 || promo != nil || discount != 0 {
		t.Errorf("الكودُ لم يُحرَّر: used=%d redemptions=%d promo=%v discount=%d", used, redemptions, promo, discount)
	}
	if n := orderLedgerCount(t, h, oid); n != 0 {
		t.Errorf("**القلبُ قيّد %d سطراً في الدفتر**", n)
	}
}

// TestEXTSTORE_004_FullFlowMoney **بعد القلب: الأجرةُ ثابتة، السائقُ يوثّق، الزبونُ يؤكّد، ثمّ التسليم.**
//
//	الأجرة       ٥٠٠ (أجرةُ الطلب الأصليّة) — ثُبّتت وأُكّدت لحظةَ القلب
//	البضاعة      ٨٠٠ يوثّقها السائق ⇒ المجموعُ ١٣٠٠ يؤكّده الزبون
//	صندوقُ السائق ١٣٠٠ − ٨٠٠ = ٥٠٠ · ولا مستحقَّ متجرٍ ولا عمولة
func TestEXTSTORE_004_FullFlowMoney(t *testing.T) {
	h := New(t)
	h.oldScaleMoney()
	treasury(t, h)
	extStoreSetup(h)
	base := financialBaseline(t, h)
	ops := h.NewUser("operations")
	m := h.Factory().Merchant(Commission(10))
	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("extstore"), orderBody(h.NewItemFor(m, 1000), 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	drv := h.driverOf(oid)

	if res := extStorePost(h, ops.Token, oid, "المتجر مسكّر"); res.Code != 200 {
		t.Fatalf("القلبُ رُدّ: %s", res)
	}
	// **والسائقُ لا يملك الأجرة** — يرسل غيرَها فتُفرض لقطةُ المنصّة.
	if r := h.POST("/api/v1/driver/orders/"+oid+"/agree", drv.Token,
		map[string]any{"step": "goods", "goods_amount": 800, "fee": 9999}); r.Code >= 400 {
		t.Fatalf("توثيقُ الثمن رُدّ: %s", r)
	}
	var total, version, fee int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT total, quote_version, custom_fee FROM orders WHERE id = $1::uuid`,
		oid).Scan(&total, &version, &fee); err != nil {
		t.Fatal(err)
	}
	if total != 1300 || fee != 500 {
		t.Fatalf("بعد التوثيق: مجموع %d أجرة %d — يُنتظر ١٣٠٠ و٥٠٠", total, fee)
	}
	if r := h.POST("/api/v1/orders/"+oid+"/confirm-quote", cust.Token, map[string]any{
		"payment_method": "cash", "expected_total": total, "expected_quote_version": version,
	}); r.Code >= 400 {
		t.Fatalf("تأكيدُ الزبون رُدّ: %s", r)
	}
	for _, to := range []string{"picked_up", "on_the_way", "at_dropoff"} {
		if r := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, r)
		}
	}
	h.authorizeProofExempt(oid)
	if r := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليمُ رُدّ: %s", r)
	}

	var cashDue, storePaid, commission int64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT cash_due, COALESCE(custom_store_paid, -1), platform_commission
		FROM orders WHERE id = $1::uuid`, oid).Scan(&cashDue, &storePaid, &commission); err != nil {
		t.Fatal(err)
	}
	if cashDue != 1300 || storePaid != 800 || commission != 0 {
		t.Errorf("نقد %d · للمتجر %d · عمولة %d — يُنتظر ١٣٠٠ و٨٠٠ وصفر", cashDue, storePaid, commission)
	}
	if got := cashHeld(t, h, drv.ID); got != cashDue-storePaid {
		t.Errorf("**صندوقُ السائق %d — يُنتظر %d** (المقبوضُ ناقصَ ما دُفع للمتجر)", got, cashDue-storePaid)
	}
	if got := ledgerByRef(t, h, oid, "merchant_earning"); got != 0 {
		t.Errorf("**مستحقُّ متجرٍ في طلبٍ بلا متجر**: %d", got)
	}
	var settlements int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM merchant_settlements WHERE order_id = $1::uuid`, oid).Scan(&settlements)
	if settlements != 0 {
		t.Errorf("تسويةُ متجرٍ لطلبٍ بلا متجر: %d", settlements)
	}
	if got := walletBalance(t, h, m.Owner.ID); got != 0 {
		t.Errorf("محفظةُ المتجر الأوّل %d", got)
	}
	assertNewViolations(t, h, base)
}

// TestEXTSTORE_005_StoreNoLongerSeesOrder **شاشاتُ المتجر تُسقطه — القائمةُ والطلبُ الواحد.**
func TestEXTSTORE_005_StoreNoLongerSeesOrder(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	h.Setting("platform.orders_mode", `"merchants"`)
	ops := h.NewUser("operations")
	m := h.Factory().Merchant(Commission(10))
	oid, _ := extStoreOrder(t, h, "preparing", orderBody(h.NewItemFor(m, 1000), 1), m)

	listed := func() bool {
		t.Helper()
		res := h.GET("/api/v1/merchant/stores/"+m.ID+"/orders", m.Owner.Token)
		if res.Code != 200 {
			t.Fatalf("قائمةُ المتجر: %s", res)
		}
		return strings.Contains(string(res.Body), oid)
	}
	if !listed() || h.GET("/api/v1/merchant/orders/"+oid, m.Owner.Token).Code != 200 {
		t.Fatal("المتجرُ لا يرى طلبَه قبل القلب — الفحصُ لا يقيس شيئاً")
	}
	if res := extStorePost(h, ops.Token, oid, "سبب"); res.Code != 200 {
		t.Fatalf("القلبُ رُدّ: %s", res)
	}
	if listed() {
		t.Error("الطلبُ باقٍ في قائمة المتجر بعد القلب")
	}
	if got := h.GET("/api/v1/merchant/orders/"+oid, m.Owner.Token); got.Code != 404 {
		t.Errorf("الطلبُ الواحدُ يُقرأ للمتجر بعد القلب: %s", got)
	}
}

// TestEXTSTORE_006_StoreNotifiedOnlyWhenSelfManaged **المتجرُ يُخبَر في وضع «المتاجر تدير» وحدَه.**
func TestEXTSTORE_006_StoreNotifiedOnlyWhenSelfManaged(t *testing.T) {
	h := New(t)
	extStoreSetup(h)
	ops := h.NewUser("operations")
	f := h.Factory()
	notified := func(mode string) int {
		t.Helper()
		h.Setting("platform.orders_mode", `"`+mode+`"`)
		m := f.Merchant(Commission(10))
		oid, _ := extStoreOrder(t, h, "accepted", orderBody(h.NewItemFor(m, 1000), 1), m)
		if res := extStorePost(h, ops.Token, oid, "سبب"); res.Code != 200 {
			t.Fatalf("القلبُ رُدّ: %s", res)
		}
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
			WHERE user_id = $1::uuid AND entity_id = $2 AND body LIKE '%لم يعد من متجرك'`,
			m.Owner.ID, oid).Scan(&n)
		return n
	}
	if n := notified("merchants"); n != 1 {
		t.Errorf("متجرٌ يدير طلباته أُخبر %d مرّة — يُنتظر واحدة", n)
	}
	if n := notified("platform"); n != 0 {
		t.Errorf("متجرٌ تديره المنصّة أُخبر %d مرّة — لم يستلم الطلبَ أصلاً", n)
	}
}
