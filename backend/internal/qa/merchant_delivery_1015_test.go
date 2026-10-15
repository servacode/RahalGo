package qa

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» — قراراتُ المالك ٢٠٢٦-١٠-١٥**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · لا دينَ أبداً — المتاحُ لا يكفي ⇒ يُردّ الطلبُ والقبول
//	٢ · «أنا نقداً» يُقيَّد في الصندوق عند الاستلام لا التسليم
//	٣ · الفشلُ بعد الاستلام بالذنب الحقيقيّ لا بذنب «المنصّة» المفروض
//	٤ · إلغاءُ المتجر والسائقُ عنده ⇒ إنذارٌ وأثرٌ ولا تعويضَ آليّ
//	٥ · الأجرةُ: أساسٌ + كيلومتراتُ المتجر ← المستلِم (والإحضارُ ذهاباً وإياباً)
//	٦ · «اقبض من المستلم» عرضٌ لا يدخل الدفترَ ولا الصندوق
//	٧ · «إحضار»: الاستلامُ من العنوان والتسليمُ للمتجر
//
// **وكلُّ فحصٍ هنا يسقط على الشيفرة السابقة** — والمالُ يُقاس من الصفوف لا من الردود،
// **وثوابتُ الدفتر تُصوَّر قبل السيناريو ويُطلب ألّا يستجدّ فيها خرق.**

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"testing"
)

// mdList **ردٌّ مصفوفةً** — قائمةُ طلبات السائق.
func mdList(r Res) []any {
	var env struct {
		Data []any `json:"data"`
	}
	_ = json.Unmarshal(r.Body, &env)
	return env.Data
}

// mdDriveTo **يقود السائقُ التوصيلةَ إلى حالٍ بعينها** — بانتقالات المجال الحقيقيّة.
func mdDriveTo(t *testing.T, h *Harness, id string, drv *User, last string) {
	t.Helper()
	steps := []string{"at_pickup", "picked_up", "on_the_way", "at_dropoff"}
	// **ويُكمل من حيث هو** — لا يعيد خطوةً مضت.
	now := h.statusOf(id)
	for i, st := range steps {
		if st == now {
			steps = steps[i+1:]
			break
		}
	}
	for _, to := range steps {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token,
			map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقالُ إلى %s رُدّ: %s", to, r)
		}
		if to == last {
			return
		}
	}
}

// mdDoorFail **المكتبُ يُنهي عند الباب بذنبٍ يكتبه** — ومعه اختيارُ من يدفع.
func mdDoorFail(t *testing.T, h *Harness, m mdFx, id string, body map[string]any) {
	t.Helper()
	body["action"] = "return_to_office"
	body["note"] = "قرارُ المكتب بعد الاتّصال"
	if r := h.POST("/api/v1/admin/orders/"+id+"/door-resolution", m.admin.Token, body); r.Code >= 400 {
		t.Fatalf("إنهاءُ المكتب رُدّ: %d / %s", r.Code, r)
	}
	if st := h.statusOf(id); st != "failed" {
		t.Fatalf("الحال %s والمنتظَر failed", st)
	}
}

// mdCollected **ما قُيّد تحصيلاً في الصناديق عن التوصيلة.**
func mdCollected(t *testing.T, h *Harness, id string) (sum int64, rows int) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0), count(*) FROM driver_cash_entries
		 WHERE ref = $1 AND kind = 'order_collection'`, id).Scan(&sum, &rows); err != nil {
		t.Fatalf("الصندوق: %v", err)
	}
	return sum, rows
}

// mdCompRequests **طلباتُ تعويض السائق المكتوبةُ على التوصيلة** — وذنبُ آخرها.
func mdCompRequests(t *testing.T, h *Harness, id string) (n int, fault string) {
	t.Helper()
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*), COALESCE(max(fault), '') FROM driver_compensation_requests
		 WHERE order_id = $1::uuid`, id).Scan(&n, &fault); err != nil {
		t.Fatalf("طلباتُ التعويض: %v", err)
	}
	return n, fault
}

// mdObligations **التزاماتُ طرفٍ بسببٍ على التوصيلة.**
func mdObligations(t *testing.T, h *Harness, party, partyID, cause, id string) int64 {
	t.Helper()
	var n int64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(sum(amount), 0) FROM financial_obligations
		 WHERE party_kind = $1 AND party_id = $2::uuid AND cause = $3 AND order_id = $4::uuid`,
		party, partyID, cause, id).Scan(&n); err != nil {
		t.Fatalf("الالتزامات: %v", err)
	}
	return n
}

// TestMD1015_01_WalletShortRefusedNoDebt **لا دينَ أبداً — عند الطلب وعند القبول.**
//
// **وعلى الشيفرة السابقة يسقط**: «الدينُ المفتوح» مشتعلٌ افتراضاً فيُنشأ الطلبُ بلا رصيد،
// **والقبولُ يقرأ الرصيدَ الخامَ لا المتاح** فيخصم مالاً محجوزاً لسحبٍ معلَّق.
func TestMD1015_01_WalletShortRefusedNoDebt(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-01")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)

	// ── عند الطلب: المحفظةُ فارغة ─────────────────────────────────────
	q := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
	if q.Code != http.StatusOK {
		t.Fatalf("عرضُ السعر: %d / %s", q.Code, q.Err())
	}
	if can, _ := q.JSON()["merchant_can_pay"].(bool); can {
		t.Fatalf("**محفظةٌ فارغةٌ وعرضُ السعر يقول «يقدر»** — والدينُ ممنوع: %s", q)
	}
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md1015-01a"), m.body("merchant"))
	if r.Code != http.StatusPaymentRequired || r.Err() != "delivery_credit_exhausted" {
		t.Fatalf("**بلا رصيدٍ أُنشئت توصيلةٌ يدفعها المتجر**: %d / %s", r.Code, r.Err())
	}

	// ── عند القبول: الرصيدُ يكفي والمتاحُ لا ─────────────────────────
	fundWallet(t, h, m.admin, owner, 600)
	id := m.create(t, h, "merchant", uniq("md1015-01b"))
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE wallets SET reserved = 200 WHERE user_id = $1::uuid`, owner); err != nil {
		t.Fatal(err)
	}
	acc := h.POST("/api/v1/admin/orders/"+id+"/transition", m.admin.Token, map[string]any{"to": "accepted"})
	if acc.Code != http.StatusPaymentRequired || acc.Err() != "merchant_delivery_unpaid" {
		t.Fatalf("**قُبلت والمتاحُ ٤٠٠ والأجرةُ ٥٠٠**: %d / %s", acc.Code, acc.Err())
	}
	if st := h.statusOf(id); st != "pending" {
		t.Fatalf("الحال %s والمنتظَر pending — تبقى معلَّقةً بيد المكتب", st)
	}
	if got := walletOf(t, h, owner); got != 600 {
		t.Fatalf("**مُسّت المحفظة**: %d والمنتظَر 600", got)
	}
	var debts int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM financial_obligations
		WHERE party_kind = 'merchant' AND party_id = $1::uuid`, m.fx.M.ID).Scan(&debts); err != nil {
		t.Fatal(err)
	}
	if debts != 0 {
		t.Fatalf("**كُتب على المتجر دينٌ (%d)** — والقرار: لا دينَ أبداً", debts)
	}
	// **والحجزُ المكتوبُ بيدٍ يُرفع** — لا طلبَ سحبٍ وراءه فيُقرأ خرقاً في FI-11.e.
	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE wallets SET reserved = 0 WHERE user_id = $1::uuid`, owner); err != nil {
		t.Fatal(err)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMD1015_02_MerchantCashCollectedAtPickup **«أنا نقداً» يُقيَّد عند الاستلام — مرّةً واحدة.**
func TestMD1015_02_MerchantCashCollectedAtPickup(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-02")
	base := mdaBaseline(t, h)
	id := m.create(t, h, "merchant_cash", uniq("md1015-02"))
	m.accept(t, h, id)
	drv := h.driverOf(id)

	mdDriveTo(t, h, id, drv, "picked_up")
	if sum, _ := mdCollected(t, h, id); sum != 500 {
		t.Fatalf("**بعد الاستلام الصندوقُ قيّد %d والمنتظَر 500** — النقدُ بيده ولا أثرَ له", sum)
	}
	mdDriveTo(t, h, id, drv, "at_dropoff")
	h.authorizeProofExempt(id)
	if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token,
		map[string]any{"to": "delivered"}); r.Code >= 400 {
		t.Fatalf("التسليم: %s", r)
	}
	if sum, rows := mdCollected(t, h, id); sum != 500 || rows != 1 {
		t.Fatalf("**بعد التسليم الصندوقُ %d في %d قيد** — والمنتظَرُ 500 في قيدٍ واحد", sum, rows)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
	mdaNoNewViolations(t, h, base) // FI-06.a · FI-10.b
}

// TestMD1015_03_RecipientFaultWalletNoRefund **ذنبُ المستلِم والمتجرُ دفع من محفظته ⇒ لا ردّ،
// والسائقُ يقبض أجرَه، ولا طلبَ تعويض.**
func TestMD1015_03_RecipientFaultWalletNoRefund(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-03")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	base := mdaBaseline(t, h)
	id := m.create(t, h, "merchant", uniq("md1015-03"))
	m.accept(t, h, id)
	drv := h.driverOf(id)
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "customer", "customer_action": "none"})

	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("**محفظةُ المتجر %d والمنتظَر 1500** — رُدّت أجرةٌ والذنبُ ذنبُ المستلِم", got)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("**أجرُ السائق %d والمنتظَر 450** — كأنّها سُلّمت", got)
	}
	if n, _ := mdCompRequests(t, h, id); n != 0 {
		t.Fatalf("**طلبُ تعويضٍ (%d) والمتجرُ يدفع**", n)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMD1015_04_RecipientFaultRecipientPayer **المستلِمُ دافعٌ ولم يُقبض منه شيء ⇒ المتجرُ يدفع:
// من محفظته إن كفت، وإلّا دينٌ بسبب `delivery_failed_recipient`.**
func TestMD1015_04_RecipientFaultRecipientPayer(t *testing.T) {
	t.Run("wallet", func(t *testing.T) {
		h := New(t)
		m := newMDFx(t, h, "منطقةُ MD1015-04a")
		owner := m.fx.M.Owner.ID
		fundWallet(t, h, m.admin, owner, 2000)
		base := mdaBaseline(t, h)
		id := m.create(t, h, "recipient", uniq("md1015-04a"))
		m.accept(t, h, id)
		drv := h.driverOf(id)
		mdDriveTo(t, h, id, drv, "at_dropoff")
		mdDoorFail(t, h, m, id, map[string]any{"fault": "customer", "customer_action": "none"})
		if got := walletOf(t, h, owner); got != 1500 {
			t.Fatalf("**محفظةُ المتجر %d والمنتظَر 1500** — لم يحمل الأجرة", got)
		}
		if got := walletOf(t, h, drv.ID); got != 450 {
			t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
		}
		if d := mdObligations(t, h, "merchant", m.fx.M.ID, "delivery_failed_recipient", id); d != 0 {
			t.Fatalf("**دينٌ %d والمحفظةُ تكفي**", d)
		}
		if n, _ := mdCompRequests(t, h, id); n != 0 {
			t.Fatalf("طلبُ تعويضٍ (%d) والمتجرُ يدفع", n)
		}
		mdaNoNewViolations(t, h, base)
	})
	t.Run("debt", func(t *testing.T) {
		h := New(t)
		m := newMDFx(t, h, "منطقةُ MD1015-04b")
		base := mdaBaseline(t, h)
		id := m.create(t, h, "recipient", uniq("md1015-04b"))
		m.accept(t, h, id)
		drv := h.driverOf(id)
		mdDriveTo(t, h, id, drv, "at_dropoff")
		mdDoorFail(t, h, m, id, map[string]any{"fault": "customer", "customer_action": "none"})
		if d := mdObligations(t, h, "merchant", m.fx.M.ID, "delivery_failed_recipient", id); d != 500 {
			t.Fatalf("**دينُ المتجر %d والمنتظَر 500** بسبب delivery_failed_recipient", d)
		}
		if got := walletOf(t, h, drv.ID); got != 450 {
			t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
		}
		mdaNoNewViolations(t, h, base)
	})
}

// TestMD1015_05_DriverFaultDriverFull **ذنبُ السائق والمكتبُ يحمّله كلَّها ⇒ يُردّ للمتجر،
// ولا أجرَ للسائق، ودينٌ عليه.**
//
// **وعلى الشيفرة السابقة يسقط**: الخسارةُ تُقرأ من ربح المنصّة — وهو صفرٌ في التوصيلة —
// **فخيارُ «على السائق» لا يكتب شيئاً.**
func TestMD1015_05_DriverFaultDriverFull(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-05")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	base := mdaBaseline(t, h)
	id := m.create(t, h, "merchant", uniq("md1015-05"))
	m.accept(t, h, id)
	drv := h.driverOf(id)
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "driver", "driver_share": "full"})

	if got := walletOf(t, h, owner); got != 2000 {
		t.Fatalf("محفظةُ المتجر %d والمنتظَر 2000 — يُردّ له كاملاً", got)
	}
	if got := walletOf(t, h, drv.ID); got != 0 {
		t.Fatalf("**السائقُ قبض %d وذنبُه** — والمنتظَر 0", got)
	}
	if d := mdObligations(t, h, "driver", drv.ID, "failed_order_driver", id); d != 500 {
		t.Fatalf("**دينُ السائق %d والمنتظَر 500** — المكتبُ حمّله الكلفةَ كلَّها", d)
	}
	if n, _ := mdCompRequests(t, h, id); n != 0 {
		t.Fatalf("طلبُ تعويضٍ (%d) بذنب السائق", n)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMD1015_06_PlatformFaultRefundsCashAndRequestsComp **ذنبُ المنصّة ⇒ يُردّ للمتجر — ولو
// دفع نقداً عند الاستلام — وطلبُ تعويضٍ للسائق بذنب المنصّة.**
func TestMD1015_06_PlatformFaultRefundsCashAndRequestsComp(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-06")
	owner := m.fx.M.Owner.ID
	base := mdaBaseline(t, h)
	id := m.create(t, h, "merchant_cash", uniq("md1015-06"))
	m.accept(t, h, id)
	drv := h.driverOf(id)
	mdDriveTo(t, h, id, drv, "at_dropoff")
	mdDoorFail(t, h, m, id, map[string]any{"fault": "platform"})

	if got := walletOf(t, h, owner); got != 500 {
		t.Fatalf("**المتجرُ دفع ٥٠٠ نقداً عند الاستلام ولم يُردّ له شيء**: محفظتُه %d", got)
	}
	if got := walletOf(t, h, drv.ID); got != 0 {
		t.Fatalf("السائقُ قبض %d والمنتظَر 0 — تعويضُه بطلبٍ معلَّق", got)
	}
	if n, fault := mdCompRequests(t, h, id); n != 1 || fault != "platform" {
		t.Fatalf("طلباتُ تعويض %d بذنب %q — والمنتظَرُ واحدٌ بذنب platform", n, fault)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMD1015_07_MerchantCancelAtPickupWarns **المتجرُ يُلغي والسائقُ عنده ⇒ إنذارٌ وأثرٌ،
// ولا تعويضَ آليّ.**
func TestMD1015_07_MerchantCancelAtPickupWarns(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-07")
	owner := m.fx.M.Owner.ID
	id := m.create(t, h, "recipient", uniq("md1015-07"))
	m.accept(t, h, id)
	drv := h.driverOf(id)
	mdDriveTo(t, h, id, drv, "at_pickup")
	if c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	var warned, events int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications
		 WHERE user_id = $1::uuid AND entity_id = $2 AND title LIKE '%بعد وصول السائق%'`,
		owner, id).Scan(&warned); err != nil {
		t.Fatal(err)
	}
	if warned != 1 {
		t.Fatalf("**إنذاراتُ المتجر %d والمنتظَرُ واحد**", warned)
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM order_events WHERE order_id = $1::uuid AND note LIKE '%بعد وصول السائق%'`,
		id).Scan(&events); err != nil {
		t.Fatal(err)
	}
	if events != 1 {
		t.Fatalf("**أثرُ المكتب في السجلّ %d والمنتظَرُ واحد**", events)
	}
	if n, _ := mdCompRequests(t, h, id); n != 0 {
		t.Fatalf("**تعويضٌ آليٌّ (%d)** — والمكتبُ يعوّض بيده إن رأى", n)
	}
}

// TestMD1015_08_FeeFromStoreToRecipient **الأجرةُ = الأساسُ + كيلومتراتُ المتجر ← المستلِم.**
//
// **وعلى الشيفرة السابقة يسقط**: المسافةُ كانت من الدبّوس إلى مركز المنطقة.
func TestMD1015_08_FeeFromStoreToRecipient(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-08")
	h.Setting("delivery.by_distance", "true")
	h.Setting("delivery.per_km", "100")
	h.Setting("delivery.max_fee", "0")
	// **المتجرُ شمالَ المركز والمستلِمُ جنوبَه** — فالمسافتان تختلفان.
	storeLat, storeLng := m.z.Lat+0.010, m.z.Lng
	dropLat, dropLng := m.z.Lat-0.005, m.z.Lng
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET location =
		ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography WHERE id = $3::uuid`,
		storeLat, storeLng, m.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	var d float64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT ST_Distance(ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography,
		                   ST_SetSRID(ST_MakePoint($4, $3), 4326)::geography)`,
		storeLat, storeLng, dropLat, dropLng).Scan(&d); err != nil {
		t.Fatal(err)
	}
	want := 500 + 100*int64(math.Ceil(d/1000))
	wantFetch := 500 + 100*int64(math.Ceil(2*d/1000))
	quote := func(q string) map[string]any {
		r := h.GET(m.url("/delivery-quote")+q, m.fx.Tok)
		if r.Code != http.StatusOK {
			t.Fatalf("عرضُ السعر %s: %d / %s", q, r.Code, r.Err())
		}
		return r.JSON()
	}
	pt := fmt.Sprintf("?lat=%f&lng=%f", dropLat, dropLng)
	if j := quote(pt); j["fee"] != float64(want) || j["fee_basis"] != "distance" {
		t.Fatalf("**الأجرةُ %v (%v) والمنتظَر %d بمسافة المتجر ← المستلِم %.0fم**", j["fee"], j["fee_basis"], want, d)
	}
	if j := quote(""); j["fee"] != float64(500) || j["fee_basis"] != "no_point" {
		t.Fatalf("**بلا نقطةٍ الأجرةُ %v (%v) والمنتظَر الأساسُ 500**", j["fee"], j["fee_basis"])
	}
	if j := quote(pt + "&mode=fetch"); j["fee"] != float64(wantFetch) {
		t.Fatalf("**الإحضارُ %v والمنتظَر %d — ذهابٌ وإياب**", j["fee"], wantFetch)
	}
	body := m.body("recipient")
	body["lat"], body["lng"], body["delivery_mode"] = dropLat, dropLng, "fetch"
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md1015-08"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	var fee int64
	var mode string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT delivery_fee, delivery_mode FROM orders WHERE id = $1::uuid`, id).Scan(&fee, &mode)
	if fee != wantFetch || mode != "fetch" {
		t.Fatalf("الإنشاء: أجرةٌ %d · وضعٌ %q — والمنتظَرُ %d و fetch", fee, mode, wantFetch)
	}
}

// TestMD1015_09_CollectAmountDisplayOnly **«اقبض من المستلم» يصل السائقَ والمتجر — ولا يدخل
// الدفترَ ولا الصندوق.**
func TestMD1015_09_CollectAmountDisplayOnly(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-09")
	base := mdaBaseline(t, h)
	body := m.body("recipient")
	body["collect_amount"] = 25000
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md1015-09"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	m.accept(t, h, id)
	drv := h.driverOf(id)

	dl := h.GET("/api/v1/driver/orders", drv.Token)
	found := false
	for _, x := range mdList(dl) {
		if o, _ := x.(map[string]any); o["id"] == id {
			found = true
			if o["collect_amount"] != float64(25000) {
				t.Fatalf("**السائقُ لا يرى المبلغ**: %v", o["collect_amount"])
			}
		}
	}
	if !found {
		t.Fatalf("السائقُ لا يرى التوصيلة: %s", dl)
	}
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok)
	if one.JSON()["collect_amount"] != float64(25000) {
		t.Fatalf("**المتجرُ لا يرى المبلغَ في توصيلته**: %s", one)
	}
	deliverOrder(t, h, id, drv)
	var inLedger, inBox int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND abs(amount) >= 25000`, id).Scan(&inLedger)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM driver_cash_entries
		WHERE ref = $1 AND abs(amount) >= 25000`, id).Scan(&inBox)
	if inLedger != 0 || inBox != 0 {
		t.Fatalf("**المبلغُ دخل الدفتر (%d) أو الصندوق (%d)**", inLedger, inBox)
	}
	if sum, _ := mdCollected(t, h, id); sum != 500 {
		t.Fatalf("الصندوقُ %d والمنتظَرُ الأجرةُ وحدَها 500", sum)
	}
	mdaNoNewViolations(t, h, base)
}

// TestMD1015_10_FetchStopsReversed **«إحضار»: يستلم من العنوان ويسلّم للمتجر.**
func TestMD1015_10_FetchStopsReversed(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD1015-10")
	body := m.body("merchant_cash")
	body["delivery_mode"] = "fetch"
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md1015-10"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	m.accept(t, h, id)
	drv := h.driverOf(id)
	var sLat, sLng float64
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(location::geometry), ST_X(location::geometry)
		FROM merchants WHERE id = $1::uuid`, m.fx.M.ID).Scan(&sLat, &sLng); err != nil {
		t.Fatal(err)
	}
	dl := h.GET("/api/v1/driver/orders", drv.Token)
	for _, x := range mdList(dl) {
		o, _ := x.(map[string]any)
		if o["id"] != id {
			continue
		}
		near := func(a any, b float64) bool { f, _ := a.(float64); return math.Abs(f-b) < 1e-6 }
		if o["delivery_mode"] != "fetch" {
			t.Fatalf("الوضعُ لا يصل السائق: %v", o["delivery_mode"])
		}
		if !near(o["nav_lat"], m.dropLat) || !near(o["nav_lng"], m.dropLng) {
			t.Fatalf("**الاستلامُ ليس من العنوان**: %v,%v والمنتظَر %f,%f", o["nav_lat"], o["nav_lng"], m.dropLat, m.dropLng)
		}
		if !near(o["lat"], sLat) || !near(o["lng"], sLng) {
			t.Fatalf("**التسليمُ ليس للمتجر**: %v,%v والمنتظَر %f,%f", o["lat"], o["lng"], sLat, sLng)
		}
		if o["customer_name"] != m.fx.M.Name || o["merchant_name"] != "أبو خالد" {
			t.Fatalf("الوقفتان بأسمائهما: استلامٌ %v · تسليمٌ %v", o["merchant_name"], o["customer_name"])
		}
		return
	}
	t.Fatalf("السائقُ لا يرى التوصيلة: %s", dl)
}
