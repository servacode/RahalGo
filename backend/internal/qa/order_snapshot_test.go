package qa

import (
	"context"
	"testing"
)

// ══════════════════════════════════════════════════════════════════════
// **لقطةُ اقتصادِ الطلب** — `XQ-2` · `XG-25`…`XG-28`
// ══════════════════════════════════════════════════════════════════════
//
//	EXISTING ORDER USES ITS FINANCIAL SNAPSHOT
//
// **والنسبُ تُثبَّت لحظةَ نشوء الالتزام الماليّ — إنشاءِ الطلب** —
// **ولا يبدّل إعدادٌ لاحقٌ اقتصادَ طلبٍ قائم.**
//
// **والأثرُ الذي قيس قبل**: طلبٌ أُنشئ بوضعٍ وسُلّم بآخر ⇒ **مئتان بدل
// مئة**، **وطلبٌ نسبتُه خمسةٌ يُسلَّم بخمسةَ عشرَ إن بُدّل الإعداد.**

// snapshotOf لقطةُ طلبٍ كما استقرّت في القاعدة.
type orderSnap struct {
	MerchantPct, RepPct, Activation int64
	Source                          string
}

func readSnap(t *testing.T, h *Harness, oid string) orderSnap {
	t.Helper()
	var s orderSnap
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(snap_merchant_commission_percent, -1),
		       COALESCE(snap_rep_commission_percent, -1),
		       COALESCE(snap_commission_source, ''),
		       COALESCE(snap_activation_orders, -1)
		  FROM orders WHERE id = $1::uuid`, oid).
		Scan(&s.MerchantPct, &s.RepPct, &s.Source, &s.Activation); err != nil {
		t.Fatalf("قراءةُ اللقطة: %v", err)
	}
	return s
}

// snapOrder ينشئ طلباً بإعداداتٍ معلومة ويردّ معرّفَه ولقطتَه.
func snapOrder(t *testing.T, h *Harness, m *Merchant, cost int64) (string, orderSnap) {
	t.Helper()
	item := h.NewItemFor(m, cost)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("xq2"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	return oid, readSnap(t, h, oid)
}

// ══════════════════════════════════════════════════════════════════════
// **S1 · S2 · القديمُ باقتصاده والجديدُ بالجديد**
// ══════════════════════════════════════════════════════════════════════
//
// **وهو دليلُ `XQ-2` الأساسيّ** — والأعضاءُ الأربعةُ فيه معاً.
func TestXQ2_S1S2_OldOrderKeepsItsEconomicsNewOrderTakesTheNew(t *testing.T) {
	h := New(t)
	treasury(t, h)
	// ── T0 · الإعداداتُ «أ» ──────────────────────────────────────
	h.Setting("merchants.commission_percent", "20")          // XG-25
	h.Setting("sales.commission_percent", "10")              // XG-26
	h.Setting("sales.commission_source", `"pricing_margin"`) // XG-27
	h.Setting("sales.activation_orders", "0")                // XG-28
	h.Setting("pricing.margin_fixed", "1000")

	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))

	// ── T1 · طلبٌ يُنشأ ─────────────────────────────────────────
	oid, snap := snapOrder(t, h, m, 5000)
	t.Logf("S1: لقطةُ الطلب — عمولةٌ=%d%% · مندوبٌ=%d%% · مصدرٌ=%q · عتبةٌ=%d",
		snap.MerchantPct, snap.RepPct, snap.Source, snap.Activation)
	if snap.MerchantPct != 20 || snap.RepPct != 10 ||
		snap.Source != "pricing_margin" || snap.Activation != 0 {
		t.Fatalf("**S1: اللقطةُ لا تطابق إعداداتِ الإنشاء** — %+v", snap)
	}

	// ── T2 · تُبدَّل الإعداداتُ إلى «ب» والطلبُ قائم ────────────
	h.Setting("merchants.commission_percent", "40")
	h.Setting("sales.commission_percent", "20")
	h.Setting("sales.commission_source", `"both"`)
	h.Setting("sales.activation_orders", "99")

	// ── T3 · يُسوّى الطلبُ القديم ───────────────────────────────
	deliverOrder(t, h, oid, h.driverOf(oid))
	oldPlatform, oldRep := economicsOf(t, h, oid)
	t.Logf("S1: الطلبُ القديمُ سُوّي — عمولةُ المنصّة=%d · عمولةُ المندوب=%d",
		oldPlatform, oldRep)

	// **بإعدادات «أ»**: عمولةُ المنصّة = ٢٠٪ × ٥٠٠٠ = ١٠٠٠ ·
	// والمندوبُ ١٠٪ من الهامش (١٠٠٠) = ١٠٠.
	if oldPlatform != 1000 {
		t.Errorf("**XG-25: عمولةُ المنصّة %d — واللقطةُ توجب 1000**", oldPlatform)
	}
	if oldRep != 100 {
		t.Errorf("**XG-26/XG-27: عمولةُ المندوب %d — واللقطةُ توجب 100**", oldRep)
	}

	// ── T4 · طلبٌ جديدٌ يأخذ «ب» ────────────────────────────────
	n := f.Merchant(OwnedByRep(rep.ID))
	nid, nsnap := snapOrder(t, h, n, 5000)
	t.Logf("S2: لقطةُ الجديد — عمولةٌ=%d%% · مندوبٌ=%d%% · مصدرٌ=%q · عتبةٌ=%d",
		nsnap.MerchantPct, nsnap.RepPct, nsnap.Source, nsnap.Activation)
	if nsnap.MerchantPct != 40 || nsnap.RepPct != 20 ||
		nsnap.Source != "both" || nsnap.Activation != 99 {
		t.Errorf("**S2: الطلبُ الجديدُ لم يأخذ الإعداداتِ الجديدة** — %+v", nsnap)
	}
	_ = nid
}

// economicsOf عمولةُ المنصّة المقيَّدةُ وعمولةُ المندوب لهذا الطلب.
func economicsOf(t *testing.T, h *Harness, oid string) (platform, rep int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT o.platform_commission,
		       COALESCE((SELECT sum(amount) FROM wallet_transactions
		                 WHERE ref = o.id::text AND kind = 'commission'), 0)
		  FROM orders o WHERE o.id = $1::uuid`, oid).Scan(&platform, &rep); err != nil {
		t.Fatalf("قراءةُ الاقتصاد: %v", err)
	}
	return platform, rep
}

// ══════════════════════════════════════════════════════════════════════
// **S3 · والاستردادُ باقتصاد الطلب لا باقتصاد اليوم**
// ══════════════════════════════════════════════════════════════════════
func TestXQ2_S3_RefundUsesOriginalEconomics(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.commission_source", `"both"`)
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "1000")

	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	oid, _ := snapOrder(t, h, m, 5000)
	deliverOrder(t, h, oid, h.driverOf(oid))
	_, paid := economicsOf(t, h, oid)
	if paid <= 0 {
		t.Fatalf("لم تُدفع عمولة: %d", paid)
	}

	// **ثمّ تُبدَّل الإعداداتُ إلى «ج» قبل الاسترداد.**
	h.Setting("merchants.commission_percent", "50")
	h.Setting("sales.commission_percent", "50")
	h.Setting("sales.commission_source", `"platform_commission"`)

	admin := h.NewUser("admin")
	ref := h.Call("POST", "/api/v1/admin/orders/"+oid+"/transition", admin.Token,
		map[string]any{"to": "refunded", "note": "XQ-2"}, nil)
	_, net := economicsOf(t, h, oid)
	t.Logf("S3: دُفع=%d · الاستردادُ ⇒ %d · الصافي=%d", paid, ref.Code, net)
	if ref.Code >= 400 {
		t.Fatalf("الاسترداد: %s", ref)
	}
	if net != 0 {
		t.Errorf("**S3: الاستردادُ لم يعكس ما دُفع فعلاً** — صافٍ=%d", net)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **S4 · S5 · S6 · اللقطةُ تبقى ولا تعتمد على المخزن**
// ══════════════════════════════════════════════════════════════════════
//
// **S4** إعادةُ الاتّصال (عمليّةٌ جديدةٌ تقرأ من القاعدة) ·
// **S5** محوُ صفوف الإعدادات · **S6** إفسادُ قيمتها.
func TestXQ2_S4S5S6_SnapshotSurvivesSettingLoss(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.commission_source", `"pricing_margin"`)
	h.Setting("sales.activation_orders", "0")
	h.Setting("pricing.margin_fixed", "1000")

	f := h.Factory()
	rep := f.RepAccount()
	m := f.Merchant(OwnedByRep(rep.ID))
	oid, snap := snapOrder(t, h, m, 5000)

	// ── S5 · تُمحى صفوفُ الإعدادات بعد الإنشاء ──────────────────
	if _, err := h.Pool.Exec(ctxBG(), `
		DELETE FROM app_settings
		 WHERE key IN ('merchants.commission_percent','sales.commission_percent',
		               'sales.activation_orders')`); err != nil {
		t.Fatalf("محوُ الإعدادات: %v", err)
	}
	// ── S6 · وتُفسَد قيمةُ الوضع ────────────────────────────────
	h.Setting("sales.commission_source", `"__invalid__"`)

	// ── S4 · واللقطةُ باقيةٌ في القاعدة تُقرأ في نداءٍ جديد ────
	again := readSnap(t, h, oid)
	t.Logf("S4: اللقطةُ بعد محو الإعدادات وإفسادها — %+v", again)
	if again != snap {
		t.Fatalf("**S4: تبدّلت اللقطةُ بتبديل المخزن** — %+v ← %+v", snap, again)
	}

	deliverOrder(t, h, oid, h.driverOf(oid))
	platform, repCom := economicsOf(t, h, oid)
	t.Logf("S5/S6: سُوّي بلا إعداداتٍ صالحة — عمولةُ المنصّة=%d · مندوبٌ=%d",
		platform, repCom)
	if platform != 1000 || repCom != 100 {
		t.Errorf("**S5/S6: التسويةُ اعتمدت على المخزن لا على اللقطة** — %d · %d",
			platform, repCom)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F1 · F2 · الطلبُ ولقطتُه فعلٌ واحد**
// ══════════════════════════════════════════════════════════════════════
//
// **ولا طلبٌ بلا لقطة** — يُسوّى باقتصادٍ لم يُوافَق عليه · **ولا
// لقطةٌ بلا طلب.**
func TestXQ2_F1F2_SnapshotIsAtomicWithTheOrder(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("sales.commission_source", `"pricing_margin"`)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 5000)

	// **والعدُّ لزبون هذا الفحص وحدَه** — والقاعدةُ مشترَكة.
	cust := h.Customer()
	fp := h.ArmAny("XQ2/order-insert", "orders", "INSERT")
	res := h.POSTKey("/api/v1/orders", cust.Token, uniq("xq2f"), orderBody(item, 1))
	fp.Disarm()
	var rows, halves int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID).Scan(&rows)
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM orders
		 WHERE num_nulls(snap_merchant_commission_percent, snap_rep_commission_percent,
		                 snap_commission_source, snap_activation_orders) NOT IN (0, 4)`).
		Scan(&halves)
	t.Logf("F1/F2: سقوطُ كتابة الطلب ⇒ %d · طلباتٌ=%d · أنصافُ لقطةٍ=%d",
		res.Code, rows, halves)
	if res.Code < 400 {
		t.Errorf("**F1: مضى الإنشاءُ وكتابتُه ساقطة** — %d", res.Code)
	}
	if rows != 0 {
		t.Errorf("**F2: بقي طلبٌ بعد سقوطٍ** — %d", rows)
	}
	if halves != 0 {
		t.Errorf("**نصفُ لقطةٍ** — %d", halves)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **F3 · وضعٌ لا يُقرأ ⇒ لا لقطةَ ولا طلب**
// ══════════════════════════════════════════════════════════════════════
func TestXQ2_F3_UnreadableSettingBlocksCreation(t *testing.T) {
	h := New(t)
	treasury(t, h)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 5000)
	h.Setting("sales.commission_source", `"__invalid__"`)

	cust := h.Customer()
	res := h.POSTKey("/api/v1/orders", cust.Token, uniq("xq2g"), orderBody(item, 1))
	var rows int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM orders WHERE customer_id = $1::uuid`, cust.ID).Scan(&rows)
	t.Logf("F3: وضعٌ مجهولٌ ⇒ الإنشاءُ %d · طلباتٌ=%d", res.Code, rows)
	if res.Code < 400 {
		t.Errorf("**F3: أُنشئ طلبٌ بلقطةٍ لا تُقرأ** — %d", res.Code)
	}
	if rows != 0 {
		t.Errorf("**F3: بقي طلبٌ بلا اقتصادٍ معلوم** — %d", rows)
	}
}

// ══════════════════════════════════════════════════════════════════════
// **C1 · تبديلُ الإعدادات يسابق الإنشاء — ولا لقطةَ هجينة**
// ══════════════════════════════════════════════════════════════════════
//
// **ونقطةُ الحسم قراءةُ الأربع** — **تقع كلُّها قبل كتابة الطلب**،
// **فتخرج اللقطةُ من عقدٍ واحدٍ لا من نصفَين.**
func TestXQ2_C1_ConcurrentSettingChangeGivesNoHybridSnapshot(t *testing.T) {
	h := New(t)
	treasury(t, h)
	h.Setting("merchants.commission_percent", "20")
	h.Setting("sales.commission_percent", "10")
	h.Setting("sales.commission_source", `"pricing_margin"`)
	h.Setting("sales.activation_orders", "0")

	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 5000)

	r := Race(t, 0,
		Actor{Name: "إنشاء", Do: func(context.Context) any {
			return h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("xq2c"),
				orderBody(item, 1))
		}},
		Actor{Name: "تبديل", Do: func(context.Context) any {
			admin := h.NewUser("admin")
			for _, kv := range [][2]string{
				{"merchants.commission_percent", "40"},
				{"sales.commission_percent", "20"},
			} {
				h.Call("PUT", "/api/v1/admin/settings/"+kv[0], admin.Token,
					map[string]any{"value": atoiOr(kv[1])}, nil)
			}
			return nil
		}},
	)

	// ══════════════════════════════════════════════════════════════
	// **وما هو التمزّق وما ليس تمزّقاً**
	// ══════════════════════════════════════════════════════════════
	//
	// **الأدمنُ يبدّل مفتاحين في نداءين** — **فحالُ `40/10` عقدٌ
	// ثُبِّت فعلاً بين النداءين**، ولقطةٌ تأخذه صادقة.
	//
	// **والتمزّقُ هو `20/20`**: **نسبةُ المنصّة القديمةُ مع نسبةِ
	// المندوب الجديدة** — **تركيبٌ لم يوجد في القاعدة قطّ**، ولا
	// يخرج إلّا من قراءتين تتخطّيان كتابةً بينهما.
	var hybrids int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM orders
		 WHERE snap_merchant_commission_percent = 20
		   AND snap_rep_commission_percent = 20`).
		Scan(&hybrids)
	var snaps string
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(string_agg(snap_merchant_commission_percent || '/' ||
		                           snap_rep_commission_percent, ' · '), '—')
		  FROM orders`).Scan(&snaps)
	t.Logf("C1: تداخلٌ=%d · لقطاتٌ=%s · هجينةٌ=%d", r.Probe.Max(), snaps, hybrids)
	if hybrids != 0 {
		t.Errorf("**C1: تركيبٌ لم يوجد في القاعدة قطّ** — %d · "+
			"**وذاك قراءتان تتخطّيان كتابةً بينهما.**", hybrids)
	}
}

// atoiOr رقمٌ من نصٍّ — لقيم الإعدادات العدديّة.
func atoiOr(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}
