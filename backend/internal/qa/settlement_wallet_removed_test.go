package qa

// ══════════════════════════════════════════════════════════════════════
// **شيلت «المحفظة» من طريقة تسوية المتجر** (قرارُ المالك ٢٠٢٦-١١-١٠، القرار ٩ — خطّةُ المال B2)
// ══════════════════════════════════════════════════════════════════════
//
//  1. البابان (الأدمن وصاحبُ المتجر) يردّان «محفظة» برمزٍ واضح `settlement_wallet_removed`،
//     **و«نقد» يمرّ** (تأكيدُ متجرٍ لم يُسأل بعد).
//  2. **الهجرةُ ٠٥٣٣ تحوّل كلَّ متجرٍ «محفظة» إلى «نقد»** — ورصيدُ محفظته القديمُ يبقى كما هو
//     **ويُطلب سحبُه ويُوافَق عليه.**
//  3. **والطلبُ الجديدُ من متجرٍ كان «محفظة» يُسوّى نقداً عند الاستلام** — لا قيدَ مستحقٍّ في محفظته.

import (
	"net/http"
	"os"
	"testing"
)

func TestSWR01_WalletRejectedAtBothDoors(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	owner := h.TokenFor(m.Owner.ID, "merchant")
	admin := h.NewUser("admin")
	own := "/api/v1/merchant/stores/" + m.ID + "/settlement-method"
	adm := "/api/v1/admin/merchants/" + m.ID + "/settlement-method"

	// **«نقد» يمرّ من البابين** — وهو ما بقي.
	if r := h.Call("PATCH", own, owner, map[string]any{"method": "cash"}, nil); r.Code != http.StatusOK ||
		r.JSON()["settlement_method"] != "cash" {
		t.Fatalf("صاحبُ المتجر يؤكّد «نقد»: %d / %s", r.Code, r.Err())
	}
	if r := h.Call("PATCH", adm, admin.Token, map[string]any{"method": "cash"}, nil); r.Code != http.StatusOK {
		t.Fatalf("الأدمن يضبط «نقد»: %d / %s", r.Code, r.Err())
	}

	// **و«محفظة» تُردّ برمزها من البابين** — والمتجرُ يبقى نقداً.
	for _, c := range []struct{ who, path, tok string }{
		{"صاحبُ المتجر", own, owner}, {"الأدمن", adm, admin.Token},
	} {
		r := h.Call("PATCH", c.path, c.tok, map[string]any{"method": "wallet"}, nil)
		if r.Code != http.StatusBadRequest || r.Err() != "settlement_wallet_removed" {
			t.Errorf("**%s اختار «محفظة» فمرّ أو رُدّ بغير رمزها**: %d / %s", c.who, r.Code, r.Err())
		}
	}
	var method string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT settlement_method FROM merchants WHERE id = $1`, m.ID).
		Scan(&method); err != nil || method != "cash" {
		t.Fatalf("**المتجرُ صار %q بعد ردّ «محفظة»** (%v)", method, err)
	}
}

func TestSWR02_MigrationFlipsWalletStoresBalanceStaysAndNewOrderIsCash(t *testing.T) {
	h := New(t)
	h.Setting("customers.require_whatsapp", "false")
	treasury(t, h)
	fin := h.NewUser("admin")

	item := h.NewItem(6000) // متجرُ المِسنَد «محفظة»
	place := func() string {
		made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("swr"), orderBody(item, 1))
		if made.Code != http.StatusCreated {
			t.Fatalf("الطلب: %s", made)
		}
		id, _ := made.JSON()["id"].(string)
		return id
	}
	var ownerID string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT owner_user_id::text FROM merchants WHERE id = $1`,
		item.MerchantID).Scan(&ownerID); err != nil {
		t.Fatal(err)
	}

	// ── قبل: طلبٌ قديمٌ من متجرٍ «محفظة» ⇒ مستحقٌّ في محفظته ──────────────
	old := place()
	deliverOrder(t, h, old, h.driverOf(old))
	before := walletBalance(t, h, ownerID)
	if got := ledgerByRef(t, h, old, "merchant_earning"); got <= 0 || before < got {
		t.Fatalf("التجهيز لم يقع: مستحقّ %d ورصيد %d", got, before)
	}

	// ── الهجرة ─────────────────────────────────────────────────────────
	raw, err := os.ReadFile("../migrate/migrations/0533_settlement_cash_only.sql")
	if err != nil {
		t.Fatalf("الهجرة: %v", err)
	}
	if _, err := h.Pool.Exec(ctxBG(), string(raw)); err != nil {
		t.Fatalf("تشغيلُ الهجرة: %v", err)
	}
	var method string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT settlement_method FROM merchants WHERE id = $1`, item.MerchantID).Scan(&method)
	if method != "cash" {
		t.Fatalf("**الهجرةُ أبقت المتجرَ %q**", method)
	}
	if got := walletBalance(t, h, ownerID); got != before {
		t.Fatalf("**الهجرةُ مسّت رصيدَ المتجر القديم**: %d ⇒ %d", before, got)
	}

	// ── بعد: طلبٌ جديدٌ من المتجر نفسِه ⇒ نقدٌ عند الاستلام، ولا قيدَ في محفظته ──
	fresh := place()
	deliverOrder(t, h, fresh, h.driverOf(fresh))
	var smethod, state string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT method, state FROM merchant_settlements
		WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, fresh, item.MerchantID).Scan(&smethod, &state); err != nil {
		t.Fatalf("تسويةُ الطلب الجديد: %v", err)
	}
	if smethod != "cash" || state != "cash_paid" {
		t.Errorf("**الطلبُ الجديدُ سُوّي %s/%s — يُنتظر cash/cash_paid**", smethod, state)
	}
	if got := ledgerByRef(t, h, fresh, "merchant_earning"); got != 0 {
		t.Errorf("**بضاعةُ طلبٍ جديدٍ نزلت في محفظة المتجر**: %d", got)
	}
	if got := walletBalance(t, h, ownerID); got != before {
		t.Errorf("**رصيدُ المتجر تحرّك بطلبٍ جديد**: %d ⇒ %d", before, got)
	}

	// ── والرصيدُ القديمُ يُسحب ─────────────────────────────────────────
	ownerTok := h.TokenFor(ownerID, "merchant")
	res := h.POST("/api/v1/me/payouts", ownerTok, map[string]any{"amount": before})
	if res.Code != http.StatusCreated {
		t.Fatalf("**طلبُ سحب الرصيد القديم رُدّ**: %s", res)
	}
	id, _ := res.JSON()["id"].(string)
	if r := decidePayout(h, fin, id, "paid"); r.Code >= 400 {
		t.Fatalf("**الموافقةُ على سحب الرصيد القديم رُدّت**: %s", r)
	}
	if got := walletBalance(t, h, ownerID); got != 0 {
		t.Errorf("بعد السحب الرصيد %d — يُنتظر صفر", got)
	}
}
