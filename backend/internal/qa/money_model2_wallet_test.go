package qa

// ══════════════════════════════════════════════════════════════════════
// **النموذج ٢: المحفظةُ تدفع التوصيلَ وحدَه** (قرارُ المالك ٢٠٢٦-١١-١٠، الدفعة B3)
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ من النموذج ٢ (المفتاحُ `finance.goods_outside` مشغَّلٌ في الاختبار وحدَه):
//
//   - «الكل من المحفظة» صريحاً (`wallet_scope=full`) يُردّ `wallet_full_removed` — عاديّاً وخاصّاً
//     وتبديلاً بعد الإنشاء.
//   - **التطبيقُ القديمُ** (`payment_method=wallet` بلا نطاق) يصير «التوصيل من المحفظة والباقي كاش»:
//     الأجرةُ من المحفظة، والبضاعةُ `cash_due`، **وإشعارٌ للزبون يشرح.**
//   - **والنموذج ١ كما كان** — `wallet` بلا نطاق = «الكل».

import (
	"net/http"
	"testing"
)

func TestModel2_WalletPaysDeliveryOnly(t *testing.T) {
	h := New(t)
	wsPrep(t, h)
	item := h.NewItem(1500)

	// ── النموذج ١ (المفتاحُ مطفأ): التطبيقُ القديمُ = «الكل» كما كان ──────────
	old := h.Customer()
	oid1 := wsMake(t, h, old, wsBody(item, 1, "wallet", ""), 50_000)
	if o := wsRead(t, h, oid1); o.scope != "full" || o.walletPaid != o.total || o.cashDue != 0 {
		t.Fatalf("النموذج ١ تبدّل: %+v — يُنتظر «الكل» كما كان", o)
	}

	h.Setting("finance.goods_outside", "true")

	// ── ١ · «الكل» صريحاً يُردّ ─────────────────────────────────────────
	cust := h.Customer()
	h.Factory().Credit(cust.ID, 50_000, "topup")
	if r := h.POSTKey("/api/v1/orders", cust.Token, uniq("m2w"), wsBody(item, 1, "", "full")); r.Code != http.StatusBadRequest || r.Err() != "wallet_full_removed" {
		t.Fatalf("النموذج ٢ قبل «الكل من المحفظة»: %s — يُنتظر 400 wallet_full_removed", r)
	}

	// ── ٢ · التطبيقُ القديمُ ⇒ «التوصيل من المحفظة» + إشعار ─────────────────
	oid := wsMake(t, h, cust, wsBody(item, 1, "wallet", ""), 0)
	if m := moneyModelOf(t, h, oid); m != 2 {
		t.Fatalf("الطلبُ نموذجُه %d — يُنتظر ٢", m)
	}
	o := wsRead(t, h, oid)
	if o.scope != "delivery" || o.walletPaid != o.fee || o.cashDue != o.total-o.fee || o.paid != o.fee || o.fee <= 0 {
		t.Fatalf("التطبيقُ القديمُ على النموذج ٢: %+v — يُنتظر الأجرةُ %d وحدَها من المحفظة", o, o.fee)
	}
	var notes int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications
		WHERE user_id = $1::uuid AND entity_id = $2 AND body LIKE '%كاش للكابتن%'`, cust.ID, oid).Scan(&notes)
	if notes != 1 {
		t.Errorf("إشعارُ «المحفظة بتدفع التوصيل بس» للزبون: %d — يُنتظر ١", notes)
	}

	// ── ٣ · التبديلُ إلى «الكل» يُردّ، والكاشُ والتوصيلُ يمشيان ───────────────
	if r := wsChange(h, cust, oid, "full", o.total); r.Code != http.StatusBadRequest || r.Err() != "wallet_full_removed" {
		t.Errorf("التبديلُ إلى «الكل» على النموذج ٢: %s — يُنتظر 400 wallet_full_removed", r)
	}
	if r := wsChange(h, cust, oid, "none", o.total); r.Code >= 400 {
		t.Errorf("التبديلُ إلى كاش: %s", r)
	}
	if r := wsChange(h, cust, oid, "delivery", o.total); r.Code >= 400 {
		t.Errorf("التبديلُ إلى التوصيل: %s", r)
	}

	// ── ٤ · الخاصّ: «الكل» يُردّ، والقديمُ يصير «التوصيل» ──────────────────
	for _, c := range []struct {
		name, method, scope, wantErr, wantScope string
	}{
		{"full", "wallet", "full", "wallet_full_removed", ""},
		{"old-app", "wallet", "", "", "delivery"},
	} {
		cc := h.Customer()
		h.Factory().Credit(cc.ID, 50_000, "topup")
		cx := h.POSTKey("/api/v1/orders/custom", cc.Token, uniq("m2wc"), customBody("أغراضٌ من السوق — نموذج ٢"))
		if cx.Code >= 400 {
			t.Fatalf("إنشاءُ الخاصّ: %s", cx)
		}
		cid, _ := cx.JSON()["id"].(string)
		drv := h.driverOf(cid)
		if r := h.POST("/api/v1/driver/orders/"+cid+"/agree", drv.Token,
			map[string]any{"goods_amount": 5000, "fee": 700}); r.Code >= 400 {
			t.Fatalf("توثيقُ العرض: %s", r)
		}
		var ver int64
		_ = h.Pool.QueryRow(ctxBG(), `SELECT quote_version FROM orders WHERE id = $1::uuid`, cid).Scan(&ver)
		body := map[string]any{"payment_method": c.method, "expected_total": 5700, "expected_quote_version": ver}
		if c.scope != "" {
			body["wallet_scope"] = c.scope
		}
		r := h.POST("/api/v1/orders/"+cid+"/confirm-quote", cc.Token, body)
		if c.wantErr != "" {
			if r.Code != http.StatusBadRequest || r.Err() != c.wantErr {
				t.Errorf("الخاصّ %s: %s — يُنتظر 400 %s", c.name, r, c.wantErr)
			}
			continue
		}
		if r.Code >= 400 {
			t.Fatalf("الخاصّ %s: %s", c.name, r)
		}
		var scope string
		var cash, reserved int64
		_ = h.Pool.QueryRow(ctxBG(), `SELECT wallet_scope, cash_due, custom_reserved_amount
			FROM orders WHERE id = $1::uuid`, cid).Scan(&scope, &cash, &reserved)
		if scope != c.wantScope || cash != 5000 || reserved != 700 {
			t.Errorf("الخاصّ %s: نطاق %s نقد %d حجز %d — يُنتظر delivery و5000 و700", c.name, scope, cash, reserved)
		}
	}
}
