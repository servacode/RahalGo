package qa

// ══════════════════════════════════════════════════════════════════════
// **تقريرُ «أرصدة من بضاعة» — قراءةٌ فقط** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٥، الدفعة B1)
// ══════════════════════════════════════════════════════════════════════
//
// القاعدةُ الجديدة: البضاعةُ لا تدخل حساباً. **والأرصدةُ القديمةُ تبقى كما هي** — فيلزم المالكَ
// تقريرٌ يقول مَن يحمل مالاً أصلُه بضاعة: محافظُ المتاجر «محفظة» · ديونُ المتاجر من بضاعة ·
// ديونُ السائقين من خسارة بضاعة · مستحقّاتُ المتاجر النقديّة غيرُ المدفوعة · ما دخل محافظَ
// الزبائن من بضاعةٍ باسترجاع · وسلفُ السائقين القائمة. **صفٌّ من كلّ مصدرٍ يظهر في قسمه.**

import (
	"net/http"
	"testing"
)

// gbSection **صفوفُ قسمٍ من التقرير** — بمعرّف الطرف.
func gbSection(t *testing.T, body map[string]any, key string) (map[string]map[string]any, float64) {
	t.Helper()
	sec, ok := body[key].(map[string]any)
	if !ok {
		t.Fatalf("**التقريرُ بلا قسم %q**: %v", key, body)
	}
	out := map[string]map[string]any{}
	rows, _ := sec["rows"].([]any)
	for _, x := range rows {
		r, _ := x.(map[string]any)
		id, _ := r["id"].(string)
		out[id] = r
	}
	total, _ := sec["total"].(float64)
	return out, total
}

func gbExpect(t *testing.T, body map[string]any, key, id string, amount int64) {
	t.Helper()
	rows, total := gbSection(t, body, key)
	r, ok := rows[id]
	if !ok {
		t.Errorf("**قسم %s بلا الصفّ المزروع** %s", key, id)
		return
	}
	if got, _ := r["amount"].(float64); int64(got) != amount {
		t.Errorf("قسم %s: المبلغ %v — يُنتظر %d", key, r["amount"], amount)
	}
	if total < float64(amount) {
		t.Errorf("قسم %s: المجموعُ %v أقلّ من صفٍّ فيه %d", key, total, amount)
	}
	if n, _ := r["count"].(float64); n < 1 {
		t.Errorf("قسم %s: عددُ الطلبات %v", key, r["count"])
	}
}

func TestGoodsBalances_ReportListsEachSource(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)
	fin := h.NewUser("finance")
	admin := h.NewUser("admin")

	place := func(item *Item, cust *User) string {
		made := h.POSTKey("/api/v1/orders", cust.Token, uniq("gb"), orderBody(item, 1))
		if made.Code != http.StatusCreated {
			t.Fatalf("الطلب: %s", made)
		}
		id, _ := made.JSON()["id"].(string)
		return id
	}
	settlementAmount := func(oid, merchantID string) int64 {
		var a int64
		if err := h.Pool.QueryRow(ctxBG(), `SELECT amount - reversed_amount FROM merchant_settlements
			WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, merchantID).Scan(&a); err != nil {
			t.Fatalf("صفُّ التسوية: %v", err)
		}
		return a
	}

	// ── ١ · متجرٌ «محفظة» سلّم بضاعة ⇒ مستحقٌّ في محفظته ──────────────
	wItem := h.NewItem(3000)
	wOrder := place(wItem, h.Customer())
	deliverOrder(t, h, wOrder, h.driverOf(wOrder))
	wGoods := settlementAmount(wOrder, wItem.MerchantID)

	// ── ٢ · متجرٌ «نقداً»: سائقٌ استلم ولم يسلّم ⇒ سلفةٌ قائمة ───────────
	cItem := h.NewItem(4000)
	coSetCash(t, h, cItem.MerchantID)
	aOrder := place(cItem, h.Customer())
	aDrv := h.driverOf(aOrder)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := h.POST("/api/v1/driver/orders/"+aOrder+"/transition", aDrv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقال إلى %s: %s", to, r)
		}
	}
	advance := settlementAmount(aOrder, cItem.MerchantID)

	// ── ٣ · طلبٌ نقديٌّ سُلّم ثمّ استُرجع ⇒ بضاعةٌ في محفظة الزبون + دينٌ على المتجر ──
	rItem := h.NewItem(5000)
	coSetCash(t, h, rItem.MerchantID)
	rCust := h.Customer()
	rOrder := place(rItem, rCust)
	deliverOrder(t, h, rOrder, h.driverOf(rOrder))
	if r := h.POST("/api/v1/admin/orders/"+rOrder+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "تقرير البضاعة"}); r.Code >= 400 {
		t.Fatalf("الاسترجاع: %s", r)
	}
	var refunded, feePaid, storeDebt int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT (SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		         WHERE kind = 'refund' AND ref = $1 AND user_id = o.customer_id),
		       o.delivery_fee - o.promo_delivery_waived,
		       (SELECT COALESCE(sum(amount - settled), 0) FROM financial_obligations
		         WHERE party_kind = 'merchant' AND order_id = o.id AND closed_at IS NULL)
		FROM orders o WHERE o.id = $1::uuid`, rOrder).Scan(&refunded, &feePaid, &storeDebt); err != nil {
		t.Fatal(err)
	}
	if refunded <= feePaid || storeDebt <= 0 {
		t.Fatalf("التجهيز لم يقع: استرجاع %d أجرة %d دين المتجر %d", refunded, feePaid, storeDebt)
	}

	// ── ٤ · مستحقٌّ نقديٌّ قديمٌ لم يُدفع ⇒ في الاحتباس ─────────────────
	hItem := h.NewItem(2000)
	hOrder := place(hItem, h.Customer())
	seedLegacyCashDue(t, h, hOrder, hItem.MerchantID, 1_700)

	// ── ٥ · دينُ سائقٍ من خسارة بضاعة ────────────────────────────────
	dDrv := h.Factory().Driver()
	var obID string
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO financial_obligations (party_kind, party_id, amount, cause, order_id)
		VALUES ('driver', $1::uuid, 900, 'failed_order_driver', $2::uuid) RETURNING id::text`,
		dDrv.ID, hOrder).Scan(&obID); err != nil {
		t.Fatalf("زرعُ دين السائق: %v", err)
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM financial_obligations WHERE id = $1::uuid`, obID)
	})

	// ── التقرير ──────────────────────────────────────────────────────
	r := h.GET("/api/v1/admin/finance/goods-balances", fin.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("**التقريرُ لا يُفتح للماليّة**: %s", r)
	}
	body := r.JSON()
	gbExpect(t, body, "store_wallets", wItem.MerchantID, wGoods)
	gbExpect(t, body, "driver_advances", aDrv.ID, advance)
	gbExpect(t, body, "customer_refunds", rCust.ID, refunded-feePaid)
	gbExpect(t, body, "store_debts", rItem.MerchantID, storeDebt)
	gbExpect(t, body, "cash_holding", hItem.MerchantID, 1_700)
	gbExpect(t, body, "driver_debts", dDrv.ID, 900)

	// ── قراءةٌ فقط: من لا يملك قراءةَ المال يُردّ ──────────────────────
	if r := h.GET("/api/v1/admin/finance/goods-balances", h.NewUser("operations").Token); !capDenied(r) {
		t.Errorf("**العمليّاتُ قرأت تقريرَ المال**: %d", r.Code)
	}
}
