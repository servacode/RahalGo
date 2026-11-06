package qa

// ══════════════════════════════════════════════════════════════════════
// **عمولةُ المندوب في طلبٍ بعدّة محطّات — كلُّ مندوبٍ من ربح متجره** (الدفعة P6)
// ══════════════════════════════════════════════════════════════════════
//
// قرارُ المالك ٢٠٢٦-١١-٠٦ (٢أ): «كلّ مندوب ياخذ عمولته من ربح المنصّة من متجره هو» · الكوبونُ
// يُقسَم على المتاجر بقيمة البضاعة (والباقي لأكبرها) · **والأرضُ لكلّ متجر** (يدفع أكثرَ قليلاً
// من أرضٍ على الطلب كلّه، ولا يدفع أقلّ). **وطلبُ المتجر الواحد على مساره اليوم.**

import (
	"net/http"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/obligations"
)

// msRepFamilies **حرّاسُ المال كلُّها لطلبٍ بعدّة محطّات** (الخطّة §7.4).
var msRepFamilies = []string{"FI-03", "FI-04", "FI-05", "FI-06", "FI-09", "FI-10", "FI-11", "FI-14"}

// msRepShop **متجران لمندوبين ونسبةُ المندوب ١٠٪ من عمولة المنصّة.**
//
// أ: كلفةٌ ١٠٠٠ بعمولة ١٠٪ ⇒ ربحُ المنصّة ١٠٠ ⇒ مندوبُه ١٠.
// ب: كلفةٌ ٢٠٠٠ بعمولة ٢٠٪ ⇒ ربحُ المنصّة ٤٠٠ ⇒ مندوبُه ٤٠.
type msRepShop struct {
	*msShop
	r1, r2 *Rep
}

func newMSRepShop(t *testing.T) *msRepShop {
	t.Helper()
	s := newMSShop(t, "merchants")
	s.h.Setting("sales.commission_source", `"platform_commission"`)
	s.h.Setting("sales.commission_percent", "10")
	s.h.Setting("sales.activation_orders", "0")
	s.h.Setting("pricing.margin_fixed", "0")
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE free_delivery_campaigns SET active = false`); err != nil {
		t.Fatal(err)
	}
	rs := &msRepShop{msShop: s, r1: s.f.RepAccount(), r2: s.f.RepAccount()}
	rs.setRep(t, s.mA, rs.r1)
	rs.setRep(t, s.mB, rs.r2)
	s.a = s.h.NewItemPriced(s.mA, 1000, 1000)
	s.b = s.h.NewItemPriced(s.mB, 2000, 2000)
	return rs
}

func (s *msRepShop) setRep(t *testing.T, m *Merchant, r *Rep) {
	t.Helper()
	if _, err := s.h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET sales_rep_user_id = $2::uuid WHERE id = $1::uuid`, m.ID, r.ID); err != nil {
		t.Fatal(err)
	}
	m.RepID = r.ID
}

// placeAccepted **طلبٌ من المتجرين قبله كلاهما** — وبكودٍ إن مُرّر.
func (s *msRepShop) placeAccepted(t *testing.T, code string) string {
	t.Helper()
	body := twoStoreBody(s.a, s.b)
	if code != "" {
		body["promo_code"] = code
	}
	r := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("msrep"), body)
	if r.Code >= 400 {
		t.Fatalf("إنشاءُ طلبٍ من متجرين: %s", r)
	}
	oid, _ := r.JSON()["id"].(string)
	s.mustAnswer(t, "قبولُ الأوّل", s.tokA, oid, map[string]any{"to": "accepted"})
	s.mustAnswer(t, "قبولُ الثاني", s.tokB, oid, map[string]any{"to": "accepted"})
	return oid
}

// deliverMultiStopRep **يقود طلباً بعدّة محطّات إلى التسليم** — استلامٌ لكلّ محطّةٍ بترتيبها
// (`stop`)، ثمّ الطريقُ والباب والتسليم.
//
// **ويعمل قبل أن يصير الاستلامُ لكلّ محطّةٍ إلزاميّاً وبعده**: إن كان الطلبُ قد صار «استُلم»
// بنداءٍ بلا محطّة، **فـ٤٠٩ للمحطّة التالية متوقَّعٌ ويُتجاوز.**
func deliverMultiStopRep(t *testing.T, h *Harness, oid string) *User {
	t.Helper()
	drv := h.driverOf(oid)
	rows, err := h.Pool.Query(ctxBG(),
		`SELECT merchant_id::text FROM order_parts
		  WHERE order_id = $1::uuid AND status NOT IN ('rejected', 'removed') ORDER BY seq`, oid)
	if err != nil {
		t.Fatal(err)
	}
	var stops []string
	for rows.Next() {
		var m string
		if err := rows.Scan(&m); err != nil {
			t.Fatal(err)
		}
		stops = append(stops, m)
	}
	rows.Close()
	for _, stop := range stops {
		for _, to := range []string{"at_pickup", "picked_up"} {
			got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
				map[string]any{"to": to, "stop": stop})
			if got.Code < 400 {
				continue
			}
			if got.Code == http.StatusConflict && h.statusOf(oid) == "picked_up" {
				continue
			}
			t.Fatalf("المحطّة %s — الانتقالُ إلى %s رُدّ: %s", stop, to, got)
		}
	}
	if st := h.statusOf(oid); st != "picked_up" {
		t.Fatalf("بعد كلّ المحطّات الحالُ %s — يُنتظر picked_up", st)
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
			map[string]any{"to": to}); got.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, got)
		}
	}
	h.authorizeProofExempt(oid)
	if got := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); got.Code >= 400 {
		t.Fatalf("التسليمُ رُدّ: %s", got)
	}
	return drv
}

// repLedger **سطورُ عمولة مندوبٍ بمرجع الطلب** — موجبُها وعددُها وصافيها.
func msRepLedger(t *testing.T, h *Harness, oid, repID string) (positive int64, rows int, net int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount) FILTER (WHERE amount > 0), 0)::bigint,
		       count(*) FILTER (WHERE amount > 0)::int,
		       COALESCE(sum(amount), 0)::bigint
		  FROM wallet_transactions WHERE ref = $1 AND kind = 'commission' AND user_id = $2::uuid`,
		oid, repID).Scan(&positive, &rows, &net); err != nil {
		t.Fatal(err)
	}
	return positive, rows, net
}

// partRep **تفصيلُ عمولة المندوب على قسم متجر.**
func partRep(t *testing.T, h *Harness, oid, mid string) (rep string, amount int64) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(rep_id::text, ''), rep_commission FROM order_parts
		 WHERE order_id = $1::uuid AND merchant_id = $2::uuid`, oid, mid).Scan(&rep, &amount); err != nil {
		t.Fatal(err)
	}
	return rep, amount
}

func heldOf(t *testing.T, h *Harness, oid, repID string) (amount int64, status string) {
	t.Helper()
	err := h.Pool.QueryRow(ctxBG(), `
		SELECT amount, status FROM rep_held_commissions WHERE order_id = $1::uuid AND rep_id = $2::uuid`,
		oid, repID).Scan(&amount, &status)
	if err != nil {
		return 0, ""
	}
	return amount, status
}

// TestMultiStop_SettleRepPaysEachStoresRep **كلُّ مندوبٍ من ربح متجره** — `T17`.
//
// **كان مندوبُ المتجر الأوّل يأخذ ١٠٪ من ربح الطلب كلّه (٥٠) ومندوبُ الثاني لا شيء.**
func TestMultiStop_SettleRepPaysEachStoresRep(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)

	// ١ · مندوبان ⇒ سطرٌ لكلٍّ بربح متجره.
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	if p, n, _ := msRepLedger(t, s.h, oid, s.r1.ID); p != 10 || n != 1 {
		t.Errorf("**مندوبُ المتجر الأوّل** قُيّد له %d في %d سطر — يُنتظر 10 في سطرٍ واحد", p, n)
	}
	if p, n, _ := msRepLedger(t, s.h, oid, s.r2.ID); p != 40 || n != 1 {
		t.Errorf("**مندوبُ المتجر الثاني** قُيّد له %d في %d سطر — يُنتظر 40 في سطرٍ واحد", p, n)
	}
	if r, a := partRep(t, s.h, oid, s.mA.ID); r != s.r1.ID || a != 10 {
		t.Errorf("تفصيلُ القسم الأوّل %s/%d — يُنتظر مندوبُه و10", r, a)
	}
	if r, a := partRep(t, s.h, oid, s.mB.ID); r != s.r2.ID || a != 40 {
		t.Errorf("تفصيلُ القسم الثاني %s/%d — يُنتظر مندوبُه و40", r, a)
	}

	// ٢ · مندوبٌ واحدٌ للمتجرين ⇒ سطرٌ واحدٌ بمجموعهما.
	s.setRep(t, s.mB, s.r1)
	same := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, same)
	if p, n, _ := msRepLedger(t, s.h, same, s.r1.ID); p != 50 || n != 1 {
		t.Errorf("**مندوبٌ واحدٌ للمتجرين** قُيّد له %d في %d سطر — يُنتظر 50 في سطرٍ واحد", p, n)
	}
	if p, _, _ := msRepLedger(t, s.h, same, s.r2.ID); p != 0 {
		t.Errorf("مندوبٌ لا متجرَ له في الطلب قُيّد له %d", p)
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}

// TestMultiStop_RepClampPerStore **الأرضُ لكلّ متجر** — كوبونٌ أكبرُ من ربح متجرٍ واحد (`T17`، حالُ الأرض).
//
// أ: كلفةٌ ١٠٠٠ بعمولة ٥٪ ⇒ ٥٠ · ب: كلفةٌ ٥٠٠٠ بعمولة ٢٠٪ ⇒ ١٠٠٠ · كوبونٌ ٦٠٠ يُقسَم بالبضاعة:
// أ ١٠٠ · ب ٥٠٠. ⇒ أ: max(٥٠ − ١٠٠، ٠) = ٠ · ب: ١٠٠٠ − ٥٠٠ = ٥٠٠ ⇒ مندوبُ ب ٥٠.
// **وأرضٌ على الطلب كلّه كانت تعطي ١٠٪ من (١٠٥٠ − ٦٠٠) = ٤٥** — والقرارُ: أكثرُ قليلاً لا أقلّ.
func TestMultiStop_RepClampPerStore(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)
	if _, err := s.h.Pool.Exec(ctxBG(),
		`UPDATE merchants SET commission_percent = 5 WHERE id = $1::uuid`, s.mA.ID); err != nil {
		t.Fatal(err)
	}
	s.b = s.h.NewItemPriced(s.mB, 5000, 5000)
	code := mpsCode(t, s.h, "fixed", 600, false)
	oid := s.placeAccepted(t, code)
	var discount int64
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT discount FROM orders WHERE id = $1::uuid`, oid).
		Scan(&discount); err != nil || discount != 600 {
		t.Fatalf("الكوبونُ لم يُطبَّق: خصمٌ %d (%v)", discount, err)
	}
	deliverMultiStopRep(t, s.h, oid)
	if p, _, _ := msRepLedger(t, s.h, oid, s.r1.ID); p != 0 {
		t.Errorf("**مندوبُ المتجر الذي أكل الكوبونُ ربحَه** قُيّد له %d — يُنتظر 0", p)
	}
	if p, _, _ := msRepLedger(t, s.h, oid, s.r2.ID); p != 50 {
		t.Errorf("**مندوبُ المتجر الثاني** قُيّد له %d — يُنتظر 50 (الأرضُ لكلّ متجر)", p)
	}
	if r, a := partRep(t, s.h, oid, s.mA.ID); r != "" || a != 0 {
		t.Errorf("قسمٌ بلا عمولةٍ يحمل مندوباً %q ومبلغاً %d", r, a)
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}

// TestMultiStop_HeldCommissionPerRep **الموقوفُ تُحجز عمولتُه وحدَه** — `T18` (الهجرة 0518).
//
// **كان الحجزُ واحداً لكلّ طلب** — فمندوبان موقوفان على الطلب نفسِه: الثاني يُبتلع.
func TestMultiStop_HeldCommissionPerRep(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)
	for _, r := range []*Rep{s.r1, s.r2} {
		if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, r.ID); err != nil {
			t.Fatal(err)
		}
	}
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	if a, st := heldOf(t, s.h, oid, s.r1.ID); a != 10 || st != "held" {
		t.Errorf("**محجوزُ المندوب الأوّل** %d/%q — يُنتظر 10 محجوزة", a, st)
	}
	if a, st := heldOf(t, s.h, oid, s.r2.ID); a != 40 || st != "held" {
		t.Errorf("**محجوزُ المندوب الثاني** %d/%q — يُنتظر 40 محجوزة", a, st)
	}
	for _, r := range []*Rep{s.r1, s.r2} {
		if p, _, _ := msRepLedger(t, s.h, oid, r.ID); p != 0 {
			t.Errorf("مندوبٌ موقوفٌ قبض %d", p)
		}
	}

	// **والعائدُ يُصرف له محجوزُه وحدَه** — والآخرُ يبقى محجوزاً.
	if r := s.h.PATCH("/api/v1/admin/users/"+s.r2.ID, s.admin.Token, map[string]any{"status": "active"}); r.Code != 200 {
		t.Fatalf("التفعيل: %s", r)
	}
	if p, _, _ := msRepLedger(t, s.h, oid, s.r2.ID); p != 40 {
		t.Errorf("المحجوزُ لم يُصرف للمندوب العائد (%d)", p)
	}
	if _, st := heldOf(t, s.h, oid, s.r1.ID); st != "held" {
		t.Errorf("محجوزُ المندوب الآخر صار %q — يُنتظر held", st)
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}

// TestMultiStop_RefundRepShareSplitFromLedger **نصيبُ المندوبين على متاجرهم من الدفتر** — `T28`.
//
// مندوبُ أ عليه دَينٌ ٤ ⇒ يبقى له ٦ ⇒ دينٌ ٦ على أ. مندوبُ ب موقوفٌ ⇒ لم يقبض ⇒ لا دينَ على ب.
// ثمّ مندوبٌ واحدٌ للمتجرين بلا دَين ⇒ ٥٠ تُقسَم ١٠ على أ و٤٠ على ب.
func TestMultiStop_RefundRepShareSplitFromLedger(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)
	if _, err := obligations.Create(ctxBG(), s.h.Pool, obligations.PartyRep, s.r1.ID, 4,
		obligations.CauseLegacy, "", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE users SET status = 'suspended' WHERE id = $1::uuid`, s.r2.ID); err != nil {
		t.Fatal(err)
	}
	refund := func(oid string) {
		t.Helper()
		if got := s.h.POST("/api/v1/admin/orders/"+oid+"/transition", s.admin.Token,
			map[string]any{"to": "refunded", "note": "على المتجر", "rep_share": "merchant"}); got.Code >= 400 {
			t.Fatalf("الاسترداد: %s", got)
		}
	}

	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	if _, _, net := msRepLedger(t, s.h, oid, s.r1.ID); net != 6 {
		t.Fatalf("صافي المندوب الأوّل %d — يُنتظر 6 (١٠ ناقصَ دَين ٤)", net)
	}
	refund(oid)
	if got := xg10ShareDebt(t, s.h, oid, "merchant", s.mA.ID); got != 6 {
		t.Errorf("**دينُ المتجر الأوّل** %d — يُنتظر 6 (ما بقي لمندوبه فعلاً)", got)
	}
	if got := xg10ShareDebt(t, s.h, oid, "merchant", s.mB.ID); got != 0 {
		t.Errorf("**دينُ المتجر الثاني** %d — مندوبُه محجوزٌ لم يقبض، يُنتظر 0", got)
	}

	// مندوبٌ واحدٌ للمتجرين.
	s.setRep(t, s.mB, s.r1)
	two := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, two)
	refund(two)
	if a, b := xg10ShareDebt(t, s.h, two, "merchant", s.mA.ID), xg10ShareDebt(t, s.h, two, "merchant", s.mB.ID); a != 10 || b != 40 {
		t.Errorf("**نصيبُ مندوبٍ واحدٍ على متجريه** أ %d · ب %d — يُنتظر 10 و40", a, b)
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}

// TestMultiStop_WaiverReversesEachRep **محوُ العمولة يعكس كلَّ مندوبٍ من متاجره** (§7.2).
//
// **كان المحوُ يعكس مندوبَ المتجر الأوّل وحدَه** — ومندوبُ الثاني يبقى بعمولةٍ على ربحٍ لم يقع.
func TestMultiStop_WaiverReversesEachRep(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)
	for _, m := range []*Merchant{s.mA, s.mB} {
		if _, err := s.h.Pool.Exec(ctxBG(), `UPDATE merchants SET settlement_method = 'cash' WHERE id = $1::uuid`,
			m.ID); err != nil {
			t.Fatal(err)
		}
	}
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)
	fin := s.h.NewUser("finance")
	if r := s.h.POST("/api/v1/admin/orders/"+oid+"/waive-commission", fin.Token,
		map[string]any{"reason": "المتجر رفض يخصم العمولة"}); r.Code != http.StatusOK {
		t.Fatalf("المحو: %s", r)
	}
	for _, c := range []struct {
		label string
		rep   *Rep
		paid  int64
	}{{"الأوّل", s.r1, 10}, {"الثاني", s.r2, 40}} {
		p, _, net := msRepLedger(t, s.h, oid, c.rep.ID)
		if p != c.paid || net != 0 {
			t.Errorf("**مندوبُ المتجر %s بعد المحو** قبض %d وصافيه %d — يُنتظر %d ثمّ صفر", c.label, p, net, c.paid)
		}
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}

// TestSingleStore_RepPayUnchangedAfterBeingASecondaryStop **متجرٌ كان محطّةً ثانيةً يُعطي مندوبَه
// على طلبه الوحيد كما كان** — `T23b`.
func TestSingleStore_RepPayUnchangedAfterBeingASecondaryStop(t *testing.T) {
	s := newMSRepShop(t)
	base := financialBaseline(t, s.h, msRepFamilies...)
	oid := s.placeAccepted(t, "")
	deliverMultiStopRep(t, s.h, oid)

	// طلبٌ من ب وحدَه: ٢٠٠٠ بعمولة ٢٠٪ ⇒ ٤٠٠ ⇒ مندوبُه ٤٠ — بلا أقسام.
	made := s.h.POSTKey("/api/v1/orders", s.cust.Token, uniq("ss-after"), orderBody(s.b, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	single, _ := made.JSON()["id"].(string)
	deliverOrder(t, s.h, single, s.h.driverOf(single))
	if p, n, _ := msRepLedger(t, s.h, single, s.r2.ID); p != 40 || n != 1 {
		t.Errorf("**مندوبُ المتجر على طلبه الوحيد** %d في %d سطر — يُنتظر 40 كما كان", p, n)
	}
	var parts int
	if err := s.h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM order_parts WHERE order_id = $1::uuid`, single).
		Scan(&parts); err != nil || parts != 0 {
		t.Errorf("طلبُ متجرٍ واحدٍ له %d قسم (%v)", parts, err)
	}
	assertNewViolations(t, s.h, base, msRepFamilies...)
}
