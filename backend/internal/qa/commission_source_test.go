package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المندوب من ربح المنصّة كلِّه** — قرارُ المالك ٢٠٢٦-١٠-٠٤
// ══════════════════════════════════════════════════════════════════════
//
// **كان للمصدر زرٌّ بثلاثة أوضاع** (`RQ-6` · `XG-13` · `XG-14`) — عمولةُ
// المنصّة أو الهامشُ أو مجموعُهما. **وحسم المالكُ القاعدة** (الإعدادات،
// البند ٤): **عشرةٌ بالمئة من ربح المنصّة = الهامش + عمولة المتجر**، والزرُّ
// حُذف لأنّه يُرجع الخطأَ بلمسة. فهذه الحرّاسُ تقيس القاعدةَ الثابتة.

// commissionCase طلبٌ مُسلَّمٌ بإعداداتٍ معلومة.
//
// **ويُقاس بالمسار الحقيقيّ** — إنشاءٌ فتسليمٌ ثمّ يُقرأ الدفتر.
func commissionCase(t *testing.T, merchantPct, repPct, margin int, cost int64) (platform, marginOut, rep int64) {
	t.Helper()
	h := New(t)
	treasury(t, h)
	// **وصفٌّ قديمٌ للمصدر يبقى في قاعدةٍ لم تُهاجَر بعدُ لا أثرَ له** —
	// يُكتب «الهامش وحده» عمداً فيُثبت أنّه لا يُقرأ.
	h.Setting("sales.commission_source", `"pricing_margin"`)
	h.Setting("merchants.commission_percent", itoa(merchantPct))
	h.Setting("sales.commission_percent", itoa(repPct))
	h.Setting("pricing.margin_fixed", itoa(margin))

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, cost)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))

	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.platform_commission,
		       COALESCE((SELECT sum((oi.unit_price - oi.merchant_price) * oi.qty)
		                 FROM order_items oi WHERE oi.order_id = o.id), 0),
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = o.id::text AND kind = 'commission'), 0)
		  FROM orders o WHERE o.id = $1::uuid`, oid).
		Scan(&platform, &marginOut, &rep); err != nil {
		t.Fatalf("القراءة: %v", err)
	}
	return platform, marginOut, rep
}

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المندوب = ١٠٪ من (الهامش + عمولة المتجر)** — مهما كان المصدرُ المخزَّن
// ══════════════════════════════════════════════════════════════════════
func TestSETTINGS_RepCommissionFromWholePlatformProfit(t *testing.T) {
	cases := []struct {
		name                    string
		merchantPct, marginCost int
		wantPlatform, wantRep   int64
	}{
		{"أ · عمولةٌ بلا هامش", 20, 0, 1000, 100},
		{"ب · هامشٌ بلا عمولة", 0, 1000, 0, 100},
		{"ج · الاثنتان", 20, 1000, 1000, 200},
		{"د · لا هذه ولا تلك", 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, m, rep := commissionCase(t, c.merchantPct, 10, c.marginCost, 5000)
			t.Logf("%s: عمولةُ المنصّة=%d · هامشٌ=%d · **عمولةُ المندوب=%d** (المرتقَبُ %d)",
				c.name, p, m, rep, c.wantRep)
			if p != c.wantPlatform {
				t.Fatalf("**عمولةُ المنصّة %d والمرتقَبُ %d**", p, c.wantPlatform)
			}
			if rep != c.wantRep {
				t.Errorf("**عمولةُ المندوب %d والقاعدةُ توجب %d**", rep, c.wantRep)
			}
		})
	}
}

// **ومفتاحُ المصدر لا يُكتب من اللوحة** — حُذف من الفهرس فيُردّ كأيّ مجهول.
func TestSETTINGS_CommissionSourceToggleRemoved(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	for _, key := range []string{"sales.commission_source", "sales.activation_orders",
		"platform.invite_code", "platform.app_url", "delivery.merchant_delivery_platform_percent"} {
		res := h.Call("PUT", "/api/v1/admin/settings/"+key, admin.Token,
			map[string]any{"value": "both"}, nil)
		if res.Code != http.StatusBadRequest {
			t.Errorf("**المفتاحُ المحذوفُ %q قُبل** — %d", key, res.Code)
		}
	}
}

// **والعكسُ يتناظر مع التسوية** — الاسترداد يُرجع ما دُفع للمندوب.
func TestSETTINGS_RepCommissionReversalMirrorsSettlement(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("pricing.margin_fixed", "1000")

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, 5000)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	deliverOrder(t, h, oid, h.driverOf(oid))

	paid := repCommissionOf(t, h, oid)
	if paid != 200 {
		t.Fatalf("**عمولةُ المندوب %d والقاعدةُ توجب 200** (١٠٪ من ١٠٠٠ هامش + ١٠٠٠ عمولة)", paid)
	}
	admin := h.NewUser("admin")
	ref := h.Call("POST", "/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "settings-1004"}, nil)
	net := repCommissionOf(t, h, oid)
	if ref.Code >= 400 && ref.Code != http.StatusConflict {
		t.Fatalf("الاسترداد: %s", ref)
	}
	if ref.Code < 400 && net != 0 {
		t.Errorf("**عمولةٌ لم تُعكَس** — صافٍ=%d", net)
	}
}

// repCommissionOf صافي عمولة المندوب المقيَّدة لهذا الطلب.
func repCommissionOf(t *testing.T, h *Harness, oid string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(sum(amount), 0) FROM wallet_transactions
		  WHERE ref = $1 AND kind = 'commission'`, oid).Scan(&n); err != nil {
		t.Fatalf("قراءةُ العمولة: %v", err)
	}
	return n
}
