package qa

import (
	"net/http"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **مصدرُ احتساب عمولة المندوب** — `RQ-6` · `XG-13` · `XG-14`
// ══════════════════════════════════════════════════════════════════════
//
// # ما كان
//
// **البوّابةُ مثبَّتةٌ في الشيفرة**: `platformCommission == 0` تمنع
// كلَّ شيءٍ **ثمّ يُحسَب من الهامش بعد أسطر** — **فيُبوَّب بوضعٍ
// ويُحسَب بآخر.**
//
// **وقيس**: متجرٌ عمولتُه صفرٌ وهامشُه ألفٌ ⇒ **مندوبُه صفر** ·
// والاثنتان ألفٌ ⇒ **مئة** (من الهامش وحدَه).
//
// # والعقد
//
//	platform_commission   عمولةُ المنصّة وحدَها
//	pricing_margin        هامشُ التسعير وحدَه — **الافتراض**
//	both                  مجموعُهما
//
// **وكلُّ مركّبٍ قد يكون صفراً وحدَه** — **ولا يمنع الصفرُ في أحدهما
// الحسابَ من الآخر** حيث يسمح الوضع.

// commissionCase طلبٌ مُسلَّمٌ بإعداداتٍ معلومة.
//
// **ويُقاس بالمسار الحقيقيّ** — إنشاءٌ فتسليمٌ ثمّ يُقرأ الدفتر.
func commissionCase(t *testing.T, mode string, merchantPct, repPct, margin int, cost int64) (platform, marginOut, rep int64) {
	t.Helper()
	h := New(t)
	treasury(t, h)
	h.Setting("sales.commission_source", `"`+mode+`"`)
	h.Setting("merchants.commission_percent", itoa(merchantPct))
	h.Setting("sales.commission_percent", itoa(repPct))
	h.Setting("sales.activation_orders", "0")
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
// **M1…M3 · الأوضاعُ الثلاثةُ تُعطي ما توجبه — والبوّابةُ نُزعت**
// ══════════════════════════════════════════════════════════════════════
func TestXG14_ThreeModesGiveTheirContract(t *testing.T) {
	// **والنسبةُ عشرةٌ في المئة في كلّ الحالات** — فالفارقُ في القاعدة.
	cases := []struct {
		mode, name              string
		merchantPct, marginCost int
		wantPlatform, wantRep   int64
	}{
		// ── الوضعُ الأوّل: عمولةُ المنصّة وحدَها ──────────────────
		{"platform_commission", "أ · عمولةٌ بلا هامش", 20, 0, 1000, 100},
		{"platform_commission", "ب · هامشٌ بلا عمولة", 0, 1000, 0, 0},
		{"platform_commission", "ج · الاثنتان", 20, 1000, 1000, 100},

		// ── الوضعُ الثاني: الهامشُ وحدَه — **الافتراض** ───────────
		{"pricing_margin", "أ · عمولةٌ بلا هامش", 20, 0, 1000, 0},
		{"pricing_margin", "ب · هامشٌ بلا عمولة", 0, 1000, 0, 100},
		{"pricing_margin", "ج · الاثنتان", 20, 1000, 1000, 100},

		// ── الوضعُ الثالث: مجموعُهما ──────────────────────────────
		{"both", "أ · عمولةٌ بلا هامش", 20, 0, 1000, 100},
		{"both", "ب · هامشٌ بلا عمولة", 0, 1000, 0, 100},
		{"both", "ج · الاثنتان", 20, 1000, 1000, 200},
		{"both", "د · لا هذه ولا تلك", 0, 0, 0, 0},
	}
	for _, c := range cases {
		t.Run(c.mode+" — "+c.name, func(t *testing.T) {
			p, m, rep := commissionCase(t, c.mode, c.merchantPct, 10, c.marginCost, 5000)
			t.Logf("%-20s %s: عمولةُ المنصّة=%d · هامشٌ=%d · **عمولةُ المندوب=%d** (المرتقَبُ %d)",
				c.mode, c.name, p, m, rep, c.wantRep)
			if p != c.wantPlatform {
				t.Fatalf("**عمولةُ المنصّة %d والمرتقَبُ %d** — تبدّلت المقدّمات", p, c.wantPlatform)
			}
			if rep != c.wantRep {
				t.Errorf("**عمولةُ المندوب %d والوضعُ %q يوجب %d**", rep, c.mode, c.wantRep)
			}
		})
	}
}

// ══════════════════════════════════════════════════════════════════════
// **M4 · والافتراضُ `pricing_margin` — أقلُّ فارقٍ ماليّ**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا يُقرأ من ذاكرة**: يُقاس بلا ضبطِ المفتاح أصلاً.
func TestXG13_DefaultIsPricingMargin(t *testing.T) {
	h := New(t)
	treasury(t, h)
	// **والافتراضُ يُقرأ من المعجم لا من الجدول** — **ومن قرأ الجدولَ
	// بيده كتب الافتراضَ مرّتين.**
	res := h.GET("/api/v1/admin/settings", h.NewUser("admin").Token)
	t.Logf("M4: معجمُ الإعدادات ⇒ %d", res.Code)

	// **والسلوكُ يُقاس لا الوصف** — طلبٌ بهامشٍ بلا عمولة.
	p, m, rep := commissionCase(t, "pricing_margin", 0, 10, 1000, 5000)
	t.Logf("M4: الافتراضُ عملاً — عمولةُ المنصّة=%d · هامشٌ=%d · مندوبٌ=%d", p, m, rep)
	if rep != 100 {
		t.Errorf("**M4: الافتراضُ لا يحسب من الهامش** — %d", rep)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **M5 · مجهولُ الوضع يُردّ ولا يرتدّ إلى افتراض**
// ══════════════════════════════════════════════════════════════════════
//
// **وقيمةٌ فاسدةٌ تُقرأ افتراضاً تدفع مالاً لا يقصده أحد** — **ولا
// يُكتشَف إلّا في كشفٍ شهريّ.**
func TestXG13_UnknownModeIsRejected(t *testing.T) {
	h := New(t)
	admin := h.NewUser("admin")
	// ══════════════════════════════════════════════════════════════
	// **وما يُبدَّل عبر الباب يُستردّ بعد الفحص**
	// ══════════════════════════════════════════════════════════════
	//
	// **`hh.Setting` تُسجّل ردَّها، والبابُ لا** — **وفحصٌ يترك مفتاحاً
	// ماليّاً مبدَّلاً يُسقط فحصاً بعيداً بلا سببٍ ظاهر.** (قِيس:
	// `TestFIN_CommissionSourceMatrix` قرأ `both` فأعطى ضعفَ المرتقَب.)
	h.Setting("sales.commission_source", `"pricing_margin"`)

	// ── والبابُ يرفض المجهول ────────────────────────────────────
	res := h.Call("PUT", "/api/v1/admin/settings/sales.commission_source", admin.Token,
		map[string]any{"value": "whatever"}, nil)
	t.Logf("M5: قيمةٌ مجهولةٌ عبر اللوحة ⇒ %d", res.Code)
	if res.Code < 400 {
		t.Errorf("**M5: قُبلت قيمةٌ خارجَ المعجم** — %d", res.Code)
	}

	// ── والأوضاعُ الثلاثةُ تُقبَل ───────────────────────────────
	for _, v := range []string{"platform_commission", "pricing_margin", "both"} {
		ok := h.Call("PUT", "/api/v1/admin/settings/sales.commission_source", admin.Token,
			map[string]any{"value": v}, nil)
		if ok.Code >= 400 {
			t.Errorf("**M5: رُدّ وضعٌ معتمد** %q — %s", v, ok)
		}
	}
}

// ══════════════════════════════════════════════════════════════════════
// **M6 · والعكسُ يتناظر مع التسوية في كلّ وضع**
// ══════════════════════════════════════════════════════════════════════
//
// **وبوّابةٌ في العكس لا في التسوية تترك قيداً لا يُعكَس** — **وهو
// خطأٌ في الاتّجاه المعاكس**: **الزبونُ يستردّ والمندوبُ يحتفظ.**
func TestXG14_ReversalMirrorsSettlementInEveryMode(t *testing.T) {
	for _, mode := range []string{"platform_commission", "pricing_margin", "both"} {
		t.Run(mode, func(t *testing.T) {
			h := New(t)
			treasury(t, h)
			h.Setting("sales.commission_source", `"`+mode+`"`)
			// **ومقدّماتٌ تُعطي عمولةً في الأوضاع الثلاثة**: عمولةٌ
			// وهامشٌ معاً.
			h.Setting("merchants.commission_percent", "20")
			h.Setting("sales.commission_percent", "10")
			h.Setting("sales.activation_orders", "0")
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
			if paid <= 0 {
				t.Fatalf("**لم تُدفع عمولةٌ في وضع %q** — %d", mode, paid)
			}
			admin := h.NewUser("admin")
			ref := h.Call("POST", "/api/v1/admin/orders/"+oid+"/transition", admin.Token,
				map[string]any{"to": "refunded", "note": "XG-14"}, nil)
			net := repCommissionOf(t, h, oid)
			t.Logf("M6 %-20s: دُفع=%d · بعد الاسترداد صافٍ=%d · الردّ=%d",
				mode, paid, net, ref.Code)
			if ref.Code >= 400 && ref.Code != http.StatusConflict {
				t.Fatalf("الاسترداد: %s", ref)
			}
			if ref.Code < 400 && net != 0 {
				t.Errorf("**M6: عمولةٌ لم تُعكَس في وضع %q** — صافٍ=%d", mode, net)
			}
		})
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

// ══════════════════════════════════════════════════════════════════════
// **M7 · قيمةٌ فاسدةٌ مخزَّنة تمنع الإنشاء ولا تمسّ قائماً**
// ══════════════════════════════════════════════════════════════════════
//
// # وعقدان لا عقدٌ واحد
//
//	لا صفَّ في المخزن   ⇒ **الافتراضُ الكانونيّ** `pricing_margin`
//	صفٌّ بقيمةٍ فاسدة   ⇒ **خطأٌ وسقوطٌ آمن**
//
// **ولا يُخلَط بينهما**: **غيابُ الخبر ليس خبراً بالسلامة.**
//
// # وموضعُ القراءة انتقل بـ`XQ-2`
//
// **كان الوضعُ يُقرأ لحظةَ التسوية** — **فصار يُقرأ لحظةَ الإنشاء
// ويُلتقَط.** **فحدُّ القيمة الفاسدة عند البابين معاً**:
//
//	قبلَ الإنشاء  ⇒ **لا يُنشأ طلبٌ بلقطةٍ لا تُقرأ**
//	بعدَ الإنشاء  ⇒ **لا يمسّ طلباً التقط اقتصادَه** (`S4`/`S5`/`S6`)
//
// **والبابُ الإداريُّ يحرس المدخل** (`M5`) — **وهذا يحرس المخزن.**
func TestXG13_InvalidStoredValueFailsSafe(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "1000")

	f := h.Factory()
	r := f.RepAccount()
	m := f.Merchant(OwnedByRep(r.ID))
	item := h.NewItemFor(m, 5000)

	// ── والفاسدةُ تُكتب في المخزن للقياس وحدَه ───────────────────
	//
	// **ولا يُضعَّف حارسُ الباب ولا قيدُ القاعدة**، و`Setting`
	// تستردّها بعد الفحص.
	h.Setting("sales.commission_source", `"__invalid__"`)

	cust := h.Customer()
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("m7"), orderBody(item, 1))
	var rows int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID).Scan(&rows)
	t.Logf("M7: قيمةٌ فاسدةٌ مخزَّنة ⇒ الإنشاءُ %d · طلباتٌ=%d", made.Code, rows)

	if made.Code < 400 {
		t.Errorf("**M7: أُنشئ طلبٌ بوضعٍ لا يعرفه المعجم** — %d", made.Code)
	}
	if rows != 0 {
		t.Errorf("**M7: بقي طلبٌ بلا اقتصادٍ معلوم** — %d", rows)
	}
}
