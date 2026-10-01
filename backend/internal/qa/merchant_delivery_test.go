package qa

// ══════════════════════════════════════════════════════════════════════
// **«لدي توصيلة» — من الباب إلى الباب** (الخطوة ١٨، ٢٠٢٦-١٠-٠١)
// ══════════════════════════════════════════════════════════════════════
//
// **حرّاسُ الميزة كانت حسبةَ النسبة وحدَها** (`TestMerchantDeliveryShare`) —
// **ولم تُنشأ توصيلةٌ واحدةٌ على القاعدة قطّ.** فأربعةُ أعطابٍ مانعةٍ عاشت
// صامتة: ضمٌّ صلبٌ على الزبون يُخفي التوصيلةَ من كلّ قراءة، و`customer_id`
// فارغٌ يُسقط كلَّ انتقال، وردُّ الإلغاء إلى زبونٍ لا وجودَ له، وسببُ دينٍ
// لا يعرفه قيدُ `0130`.
//
// **وهنا تُمشى كما يمشيها صاحبُ المتجر**: عرضُ السعر · الإنشاءُ بمفتاح ·
// سائقٌ يستلم ويسلّم · **والمالُ يُقاس في كلّ خطوة من الصفوف لا من الردود.**

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

const mdPct = "delivery.merchant_delivery_platform_percent"

type mdFx struct {
	f        *Factory
	fx       offerFx
	z        zoneFx
	admin    *User
	dropLat  float64
	dropLng  float64
	storeURL string
}

func newMDFx(t *testing.T, h *Harness, name string) mdFx {
	t.Helper()
	m := newMDFxBare(t, h, name)
	// **وسائقٌ بالدوام عند المتجر** — لا توصيلةَ بلا سائقٍ قريب (قرارُ المالك
	// ٢٠٢٦-١٠-٠١). وحرّاسُ غيابه في `merchant_delivery_gate_test.go`.
	mdDriverAtStore(t, h, m.f, m.fx.M.ID)
	return m
}

// newMDFxBare **المتجرُ ومنطقتُه بلا سائق** — لحرّاس «لا سائقَ قريب».
func newMDFxBare(t *testing.T, h *Harness, name string) mdFx {
	t.Helper()
	treasury(t, h)
	z := zoneForDemand(t, h, name)
	h.Setting(mdPct, "10")
	f := h.Factory()
	m := mdFx{
		f:  f,
		fx: newOfferFx(t, h, f, 1000), z: z, admin: h.NewUser("admin"),
		dropLat: z.Lat + 0.002, dropLng: z.Lng + 0.002,
	}
	return m
}

// mdDriverAtStore سائقٌ بالدوام موقعُه الآن عند المتجر.
func mdDriverAtStore(t *testing.T, h *Harness, f *Factory, merchantID string) *User {
	t.Helper()
	var lat, lng float64
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT COALESCE(ST_Y(location::geometry), 0), COALESCE(ST_X(location::geometry), 0)
		FROM merchants WHERE id = $1`, merchantID).Scan(&lat, &lng); err != nil {
		t.Fatal(err)
	}
	return f.Driver(OnShift(), LocationAt(lat, lng, time.Now()))
}

func (m mdFx) url(p string) string { return "/api/v1/merchant/stores/" + m.fx.M.ID + p }

func (m mdFx) body(payer string) map[string]any {
	return map[string]any{
		"recipient_name": "أبو خالد", "recipient_phone": "0935111222",
		"address_text": "شارع المنصور جانب الفرن", "lat": m.dropLat, "lng": m.dropLng,
		"parcel_note": "كيس طعام ساخن", "fee_payer": payer,
	}
}

func (m mdFx) create(t *testing.T, h *Harness, payer, key string) string {
	t.Helper()
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, key, m.body(payer))
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	if id == "" {
		t.Fatalf("الإنشاءُ بلا معرّف: %s", r)
	}
	return id
}

// TestMD01_MerchantPaysEndToEnd **المتجرُ يدفع من محفظته — والدورةُ كاملة.**
func TestMD01_MerchantPaysEndToEnd(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-01")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)

	// ── عرضُ السعر قبل الطلب ──────────────────────────────────────────
	q := h.GET(fmt.Sprintf("%s?lat=%f&lng=%f", m.url("/delivery-quote"), m.dropLat, m.dropLng), m.fx.Tok)
	if q.Code != http.StatusOK {
		t.Fatalf("عرضُ السعر: %d / %s", q.Code, q.Err())
	}
	if fee, _ := q.JSON()["fee"].(float64); fee != 500 {
		t.Fatalf("الأجرة %v والمنتظَر 500", q.JSON()["fee"])
	}
	if can, _ := q.JSON()["merchant_can_pay"].(bool); !can {
		t.Fatalf("محفظةٌ ٢٠٠٠ وأجرةٌ ٥٠٠ ⇒ يقدر — والردّ: %s", q)
	}

	// ── الإنشاء مرّتين بالمفتاح نفسِه ─────────────────────────────────
	key := uniq("md01")
	id := m.create(t, h, "merchant", key)
	again := m.create(t, h, "merchant", key)
	if again != id {
		t.Fatalf("**الإعادةُ بالمفتاح نفسِه أنشأت توصيلةً ثانية**: %s ثمّ %s", id, again)
	}
	var n int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM orders WHERE merchant_id = $1 AND kind = 'merchant_delivery'`, m.fx.M.ID).Scan(&n)
	if n != 1 {
		t.Fatalf("توصيلاتٌ %d والمنتظَرُ واحدة", n)
	}
	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("**محفظةُ المتجر %d والمنتظَر 1500** — خُصمت مرّتين أو لم تُخصم", got)
	}
	if st := h.statusOf(id); st != "dispatching" {
		t.Fatalf("الحال %s والمنتظَر dispatching — تولد في الطابور", st)
	}

	// ── تُقرأ — ولا ضمَّ صلباً يُخفيها ────────────────────────────────
	list := h.GET(m.url("/deliveries"), m.fx.Tok)
	if list.Code != http.StatusOK {
		t.Fatalf("القائمة: %d / %s", list.Code, list.Err())
	}
	ds, _ := list.JSON()["deliveries"].([]any)
	if len(ds) != 1 {
		t.Fatalf("القائمةُ %d والمنتظَرُ واحدة — **ضمٌّ صلبٌ على الزبون أخفاها؟**", len(ds))
	}
	d0, _ := ds[0].(map[string]any)
	if d0["customer_name"] != "أبو خالد" || d0["parcel_note"] != "كيس طعام ساخن" || d0["fee_payer"] != "merchant" {
		t.Fatalf("المستلِمُ لا يُقرأ في موضع الزبون: %v", d0)
	}
	// ── وسجلُّ المبيعات بلاها ─────────────────────────────────────────
	sales := h.GET(m.url("/orders"), m.fx.Tok)
	if sales.Code != http.StatusOK {
		t.Fatalf("سجلُّ المبيعات: %d", sales.Code)
	}
	if items, _ := sales.JSON()["items"].([]any); len(items) != 0 {
		t.Fatalf("**التوصيلةُ ظهرت بين المبيعات**: %d سطر", len(items))
	}

	// **ومن بوّابة الحقول لا خاماً** — لا هاتفَ سائقٍ ولا معرّفَ زبونٍ لدى المتجر.
	for _, k := range []string{"driver_phone", "driver_id", "customer_id"} {
		if _, has := d0[k]; has {
			t.Fatalf("**تسرّب «%s» إلى المتجر**: %v", k, d0[k])
		}
	}

	// ── السائقُ يراها ويمشيها إلى التسليم ─────────────────────────────
	drv := h.driverOf(id)
	// **وشاشةُ المراقبة تقرأ حالَها** — (نصُّ المالك: «ليعرف المتجرُ حالةَ توصيلته»).
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok)
	if one.Code != http.StatusOK || one.JSON()["status"] != "assigned" {
		t.Fatalf("المراقبة: %d / %s", one.Code, one)
	}
	if _, has := one.JSON()["driver_phone"]; has {
		t.Fatalf("**تسرّب هاتفُ السائق في المراقبة**")
	}
	dl := h.GET("/api/v1/driver/orders", drv.Token)
	if dl.Code != http.StatusOK || !containsID(dl, id) {
		t.Fatalf("**السائقُ لا يرى التوصيلة**: %d / %s", dl.Code, dl)
	}
	deliverOrder(t, h, id, drv)
	if st := h.statusOf(id); st != "delivered" {
		t.Fatalf("الحال %s والمنتظَر delivered", st)
	}
	// **وأجرُ السائق ٩٠٪ من ٥٠٠** — والباقي للخزينة.
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("**أجرُ السائق %d والمنتظَر 450**", got)
	}
	// **ولا مستحقَّ متجرٍ ولا عمولةَ مندوب** — لا بيعَ في التوصيلة.
	var merchantEarn, repComm int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount),0) FROM wallet_transactions
		WHERE order_id = $1 AND kind = 'merchant_earning'`, id).Scan(&merchantEarn)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT COALESCE(sum(amount),0) FROM wallet_transactions
		WHERE order_id = $1 AND kind = 'commission'`, id).Scan(&repComm)
	if merchantEarn != 0 || repComm != 0 {
		t.Fatalf("**مالُ بيعٍ على توصيلة**: مستحقُّ متجر %d · عمولةُ مندوب %d", merchantEarn, repComm)
	}
}

// TestMD02_CancelBeforePickupRefundsWallet **الإلغاءُ قبل الاستلام يُعيد المال لمن دفعه.**
func TestMD02_CancelBeforePickupRefundsWallet(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-02")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	id := m.create(t, h, "merchant", uniq("md02"))
	if got := walletOf(t, h, owner); got != 1500 {
		t.Fatalf("بعد الإنشاء %d والمنتظَر 1500", got)
	}
	c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{})
	if c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	if st := h.statusOf(id); st != "cancelled" {
		t.Fatalf("الحال %s", st)
	}
	if got := walletOf(t, h, owner); got != 2000 {
		t.Fatalf("**بعد الإلغاء %d والمنتظَر 2000** — لم يُعَد المالُ إلى المتجر", got)
	}
}

// TestMD03_DebtPathAndVoid **لا تكفي المحفظة ⇒ دينٌ تحت السقف · والإلغاءُ يُسقطه.**
func TestMD03_DebtPathAndVoid(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-03")
	// **بلا سقفٍ لا دين** — يُردّ بلفظه لا بـ٥٠٠.
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md03a"), m.body("merchant"))
	if r.Code != http.StatusPaymentRequired {
		t.Fatalf("بلا رصيدٍ ولا سقف ⇒ 402 — والردّ %d / %s", r.Code, r.Err())
	}
	// **والسقفُ من باب الإدارة** — ولا يرفعه المتجرُ لنفسه.
	credit := "/api/v1/admin/merchants/" + m.fx.M.ID + "/delivery-credit"
	if r := h.PATCH(credit, m.fx.Tok, map[string]any{"limit": 1000}); r.Code < 400 {
		t.Fatalf("**المتجرُ رفع سقفَ دينه بنفسه**: %d", r.Code)
	}
	if r := h.PATCH(credit, m.admin.Token, map[string]any{"limit": 1000}); r.Code != http.StatusOK {
		t.Fatalf("ضبطُ السقف: %d / %s", r.Code, r.Err())
	}
	if r := h.GET(credit, m.admin.Token); r.Code != http.StatusOK || r.JSON()["limit"] != float64(1000) {
		t.Fatalf("قراءةُ السقف: %d / %s", r.Code, r)
	}
	id := m.create(t, h, "merchant", uniq("md03b"))
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1`, m.fx.M.ID).Scan(&debt)
	if debt != 500 {
		t.Fatalf("**دينُ المتجر %d والمنتظَر 500**", debt)
	}
	if c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{}); c.Code != http.StatusOK {
		t.Fatalf("الإلغاء: %d / %s", c.Code, c.Err())
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1`, m.fx.M.ID).Scan(&debt)
	var open int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM financial_obligations
		WHERE order_id = $1 AND closed_at IS NULL`, id).Scan(&open)
	if debt != 0 || open != 0 {
		t.Fatalf("**بعد الإلغاء دينٌ %d والتزاماتٌ مفتوحة %d** — والمنتظَرُ صفران", debt, open)
	}
}

// TestMD04_RecipientPaysCash **المستلِمُ يدفع نقداً — والمتجرُ لا يُمَسّ.**
func TestMD04_RecipientPaysCash(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-04")
	owner := m.fx.M.Owner.ID
	id := m.create(t, h, "recipient", uniq("md04"))
	var cashDue int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT cash_due FROM orders WHERE id = $1`, id).Scan(&cashDue)
	if cashDue != 500 {
		t.Fatalf("النقدُ المستحقّ %d والمنتظَر 500", cashDue)
	}
	drv := h.driverOf(id)
	deliverOrder(t, h, id, drv)
	if got := walletOf(t, h, owner); got != 0 {
		t.Fatalf("**مُسّت محفظةُ المتجر والمستلِمُ هو الدافع**: %d", got)
	}
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
}

// TestMD05_NoCancelAfterPickup **بعد الاستلام لا يُلغي المتجر** — الغرضُ في الطريق.
func TestMD05_NoCancelAfterPickup(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-05")
	id := m.create(t, h, "recipient", uniq("md05"))
	drv := h.driverOf(id)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("%s: %s", to, r)
		}
	}
	c := h.POST("/api/v1/merchant/deliveries/"+id+"/cancel", m.fx.Tok, map[string]any{})
	if c.Code < 400 {
		t.Fatalf("**أُلغيت بعد الاستلام**: %d", c.Code)
	}
	if st := h.statusOf(id); st != "picked_up" {
		t.Fatalf("الحال %s والمنتظَر picked_up", st)
	}
}

func containsID(r Res, id string) bool {
	return len(id) > 0 && contains(string(r.Body), id)
}

// TestMD06_MerchantPaysCash **«أنا نقداً»** — (نصُّ المالك ٢٠٢٦-١٠-٠١).
//
// **المتجرُ يدفع الأجرةَ نقداً بيد السائق** — لا محفظةَ تُخصم ولا دينَ يُكتب،
// **والنقدُ يُقيَّد في صندوق السائق** كنقد المستلِم.
func TestMD06_MerchantPaysCash(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-06")
	owner := m.fx.M.Owner.ID
	fundWallet(t, h, m.admin, owner, 2000)
	id := m.create(t, h, "merchant_cash", uniq("md06"))
	var cashDue, walletPaid int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT cash_due, wallet_paid FROM orders WHERE id = $1`, id).Scan(&cashDue, &walletPaid)
	if cashDue != 500 || walletPaid != 0 {
		t.Fatalf("نقدٌ %d · محفظةٌ %d — والمنتظَرُ 500 و0", cashDue, walletPaid)
	}
	if got := walletOf(t, h, owner); got != 2000 {
		t.Fatalf("**خُصمت محفظةُ المتجر وهو يدفع نقداً**: %d", got)
	}
	var debt int64
	_ = h.Pool.QueryRow(ctxBG(), `SELECT debt FROM merchants WHERE id = $1`, m.fx.M.ID).Scan(&debt)
	if debt != 0 {
		t.Fatalf("**كُتب دينٌ وهو يدفع نقداً**: %d", debt)
	}
	drv := h.driverOf(id)
	deliverOrder(t, h, id, drv)
	if got := walletOf(t, h, drv.ID); got != 450 {
		t.Fatalf("أجرُ السائق %d والمنتظَر 450", got)
	}
}

// TestMD07_NoPointUsesStoreZone **بلا نقطةٍ على الخريطة** — (نصُّ المالك:
// «نقطة التسليم غير إجباريّة»). **الأجرةُ من منطقة المتجر، والنقطةُ «غيرُ معروفة».**
func TestMD07_NoPointUsesStoreZone(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ MD-07")
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE merchants SET location =
		ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography WHERE id = $3`, m.z.Lat, m.z.Lng, m.fx.M.ID); err != nil {
		t.Fatal(err)
	}
	q := h.GET(m.url("/delivery-quote"), m.fx.Tok)
	if q.Code != http.StatusOK || q.JSON()["fee"] != float64(500) {
		t.Fatalf("عرضُ السعر بلا نقطة: %d / %s", q.Code, q)
	}
	body := m.body("recipient")
	delete(body, "lat")
	delete(body, "lng")
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq("md07"), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاءُ بلا نقطة: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	var known bool
	_ = h.Pool.QueryRow(ctxBG(), `SELECT dropoff_known FROM orders WHERE id = $1`, id).Scan(&known)
	if known {
		t.Fatalf("**نقطةٌ لم يحدّدها أحدٌ عُلّمت معروفة** — يُوجَّه السائقُ إلى المتجر")
	}
	list := h.GET(m.url("/deliveries"), m.fx.Tok)
	ds, _ := list.JSON()["deliveries"].([]any)
	if len(ds) != 1 {
		t.Fatalf("القائمة %d", len(ds))
	}
	if d0, _ := ds[0].(map[string]any); d0["dropoff_known"] != false {
		t.Fatalf("`dropoff_known` لا يصل التطبيق: %v", d0["dropoff_known"])
	}
}
