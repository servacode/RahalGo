package qa

// ══════════════════════════════════════════════════════════════════════
// **«إحضار» — قراراتُ المالك ٢٠٢٦-١٠-١٦**
// ══════════════════════════════════════════════════════════════════════
//
//	١ · السائقُ يكتب ما دفعه بمكان الاستلام — وتقديرُ المتجر يبقى تقديراً
//	٢ · المتجرُ يوافق أو يرفض بنسخة — ولا «استلمت» قبل الموافقة
//	٣ · كودُ الاستلام في «إحضار» للمتجر — يراه في توصيلته، والسائقُ يُدخله ليُسلّم
//
// **وكلُّ فحصٍ هنا يسقط على الشيفرة السابقة**: لا بابَ للمبلغ ولا حارسَ عند «استلمت»،
// والكودُ كان يذهب لرقم المستلِم ولا يراه المتجر.

import (
	"net/http"
	"strings"
	"testing"
)

// fetchCreate **توصيلةُ «إحضار» مقبولةٌ ومُسندة** — و`estimate` تقديرُ المتجر (صفرٌ: بلا مبلغ).
func fetchCreate(t *testing.T, h *Harness, m mdFx, key string, estimate int64) (string, *User) {
	t.Helper()
	body := m.body("recipient")
	body["delivery_mode"] = "fetch"
	if estimate > 0 {
		body["collect_amount"] = estimate
	}
	r := h.POSTKey(m.url("/deliveries"), m.fx.Tok, uniq(key), body)
	if r.Code != http.StatusCreated {
		t.Fatalf("الإنشاء: %d / %s", r.Code, r.Err())
	}
	id, _ := r.JSON()["id"].(string)
	m.accept(t, h, id)
	return id, h.driverOf(id)
}

func fetchStep(h *Harness, drv *User, id, to string) Res {
	return h.POST("/api/v1/driver/orders/"+id+"/transition", drv.Token, map[string]any{"to": to})
}

func fetchPropose(h *Harness, drv *User, id string, amount int64) Res {
	return h.POST("/api/v1/driver/orders/"+id+"/fetch-amount", drv.Token, map[string]any{"amount": amount})
}

func fetchDecide(h *Harness, m mdFx, id, verb string, version int64) Res {
	return h.POST("/api/v1/merchant/deliveries/"+id+"/fetch-amount/"+verb, m.fx.Tok,
		map[string]any{"expected_version": version})
}

// TestFETCHAMT_01_PickupWaitsForMerchantApproval **السائقُ يكتب · المتجرُ يرفض · يكتب ثانيةً ·
// الموافقةُ القديمةُ تُردّ · والجديدةُ تفتح «استلمت».** والمالُ لا يدخل الدفترَ ولا الصندوق.
func TestFETCHAMT_01_PickupWaitsForMerchantApproval(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FETCHAMT-01")
	base := mdaBaseline(t, h)
	owner := m.fx.M.Owner.ID
	id, drv := fetchCreate(t, h, m, "fetchamt-01", 20000)

	// **قبل الوصول لا مبلغ** — ليس وقتَه.
	if r := fetchPropose(h, drv, id, 21000); r.Code != http.StatusConflict || r.Err() != "fetch_amount_stage" {
		t.Fatalf("FETCHAMT-1 مبلغٌ قبل الوصول: %s — يُنتظر 409 fetch_amount_stage", r)
	}
	if r := fetchStep(h, drv, id, "at_pickup"); r.Code >= 400 {
		t.Fatalf("الوصول: %s", r)
	}
	// **ولا «استلمت» بلا مبلغٍ موافَقٍ عليه** — والمتجرُ طلب مبلغاً.
	if r := fetchStep(h, drv, id, "picked_up"); r.Code != http.StatusConflict || r.Err() != "fetch_amount_not_approved" {
		t.Fatalf("FETCHAMT-2 «استلمت» بلا مبلغ: %s — يُنتظر 409 fetch_amount_not_approved", r)
	}
	if r := fetchPropose(h, drv, id, -5); r.Code != http.StatusBadRequest {
		t.Fatalf("FETCHAMT-3 مبلغٌ سالب: %s — يُنتظر 400", r)
	}
	if r := fetchPropose(h, drv, id, 21000); r.Code != http.StatusOK {
		t.Fatalf("FETCHAMT-4 كتابةُ المبلغ: %s", r)
	}
	// **والمتجرُ يُخبَر.**
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications
		 WHERE user_id = $1::uuid AND entity_id = $2 AND title LIKE '%بانتظار موافقتك%'`,
		owner, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("FETCHAMT-5 إشعارُ المتجر = %d — يُنتظر ١", n)
	}
	// **والمتجرُ يرى المبلغَ ونسختَه وحالَه.**
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok).JSON()
	if one["fetch_amount"] != float64(21000) || one["fetch_amount_version"] != float64(1) ||
		one["fetch_amount_status"] != "pending" || one["collect_amount"] != float64(20000) {
		t.Fatalf("FETCHAMT-6 ما يراه المتجر: %v · %v · %v · تقديرٌ %v", one["fetch_amount"],
			one["fetch_amount_version"], one["fetch_amount_status"], one["collect_amount"])
	}
	if r := fetchStep(h, drv, id, "picked_up"); r.Err() != "fetch_amount_not_approved" {
		t.Fatalf("FETCHAMT-7 «استلمت» والمبلغُ ينتظر: %s", r)
	}
	// **الرفضُ ثمّ مبلغٌ جديد.**
	if r := fetchDecide(h, m, id, "reject", 1); r.Code != http.StatusOK || r.JSON()["fetch_amount_status"] != "rejected" {
		t.Fatalf("FETCHAMT-8 الرفض: %s", r)
	}
	if r := fetchStep(h, drv, id, "picked_up"); r.Err() != "fetch_amount_not_approved" {
		t.Fatalf("FETCHAMT-9 «استلمت» بعد الرفض: %s", r)
	}
	if r := fetchDecide(h, m, id, "approve", 1); r.Code != http.StatusConflict || r.Err() != "fetch_amount_stale" {
		t.Fatalf("FETCHAMT-10 موافقةٌ بعد الرفض: %s — يُنتظر 409 fetch_amount_stale", r)
	}
	if r := fetchPropose(h, drv, id, 20500); r.Code != http.StatusOK {
		t.Fatalf("FETCHAMT-11 مبلغٌ جديد: %s", r)
	}
	// **وموافقةٌ بنسخةٍ قديمةٍ لا تُقفل الجديد.**
	if r := fetchDecide(h, m, id, "approve", 1); r.Code != http.StatusConflict || r.Err() != "fetch_amount_stale" {
		t.Fatalf("FETCHAMT-12 موافقةٌ بنسخةٍ قديمة: %s — يُنتظر 409 fetch_amount_stale", r)
	}
	if r := fetchDecide(h, m, id, "approve", 2); r.Code != http.StatusOK || r.JSON()["fetch_amount_status"] != "approved" {
		t.Fatalf("FETCHAMT-13 الموافقة: %s", r)
	}
	// **وبعد الموافقة لا يُغيَّر.**
	if r := fetchPropose(h, drv, id, 30000); r.Code != http.StatusConflict || r.Err() != "fetch_amount_stage" {
		t.Fatalf("FETCHAMT-14 تغييرٌ بعد الموافقة: %s — يُنتظر 409 fetch_amount_stage", r)
	}
	if r := fetchStep(h, drv, id, "picked_up"); r.Code >= 400 {
		t.Fatalf("FETCHAMT-15 «استلمت» بعد الموافقة رُدّ: %s", r)
	}
	// **والمكتبُ يرى المبلغَ وحالَه.**
	adm := h.GET("/api/v1/admin/orders/"+id, m.admin.Token).JSON()
	if adm["fetch_amount"] != float64(20500) || adm["fetch_amount_status"] != "approved" {
		t.Fatalf("FETCHAMT-16 المكتب: %v · %v", adm["fetch_amount"], adm["fetch_amount_status"])
	}
	// **وخارجَ الدفتر والصندوق.**
	var inLedger, inBox int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM wallet_transactions
		WHERE ref = $1 AND abs(amount) >= 20000`, id).Scan(&inLedger)
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM driver_cash_entries
		WHERE ref = $1 AND abs(amount) >= 20000`, id).Scan(&inBox)
	if inLedger != 0 || inBox != 0 {
		t.Fatalf("FETCHAMT-17 المبلغُ دخل الدفتر (%d) أو الصندوق (%d)", inLedger, inBox)
	}
	mdaNoNewViolations(t, h, base)
}

// TestFETCHAMT_02_NoEstimateNoStep **بلا تقديرٍ من المتجر لا خطوةَ تُفرض** — «استلمت» يمرّ.
func TestFETCHAMT_02_NoEstimateNoStep(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FETCHAMT-02")
	id, drv := fetchCreate(t, h, m, "fetchamt-02", 0)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := fetchStep(h, drv, id, to); r.Code >= 400 {
			t.Fatalf("FETCHAMT-20 %s بلا تقدير رُدّ: %s", to, r)
		}
	}
}

// TestFETCHAMT_03_SendModeAndStrangers **«توصيل» لا مبلغَ فيه · ومتجرٌ غريبٌ لا يقرّر.**
func TestFETCHAMT_03_SendModeAndStrangers(t *testing.T) {
	h := New(t)
	m := newMDFx(t, h, "منطقةُ FETCHAMT-03")
	send := m.create(t, h, "recipient", uniq("fetchamt-03s"))
	m.accept(t, h, send)
	sdrv := h.driverOf(send)
	if r := fetchStep(h, sdrv, send, "at_pickup"); r.Code >= 400 {
		t.Fatalf("الوصول: %s", r)
	}
	if r := fetchPropose(h, sdrv, send, 1000); r.Code != http.StatusConflict || r.Err() != "fetch_amount_stage" {
		t.Fatalf("FETCHAMT-30 مبلغٌ في «توصيل»: %s — يُنتظر 409 fetch_amount_stage", r)
	}

	id, drv := fetchCreate(t, h, m, "fetchamt-03f", 5000)
	if r := fetchStep(h, drv, id, "at_pickup"); r.Code >= 400 {
		t.Fatalf("الوصول: %s", r)
	}
	// **وسائقٌ غيرُ سائقها لا يكتب.**
	other := h.NewUser("driver")
	if r := fetchPropose(h, other, id, 5000); r.Code < 400 {
		t.Fatalf("FETCHAMT-31 سائقٌ غريبٌ كتب المبلغ: %s", r)
	}
	if r := fetchPropose(h, drv, id, 5000); r.Code != http.StatusOK {
		t.Fatalf("كتابةُ المبلغ: %s", r)
	}
	// **ومتجرٌ غيرُ صاحبها لا يوافق.**
	m2 := newMDFx(t, h, "منطقةُ FETCHAMT-03ب")
	if r := fetchDecide(h, m2, id, "approve", 1); r.Code != http.StatusNotFound {
		t.Fatalf("FETCHAMT-32 متجرٌ غريبٌ وافق: %s — يُنتظر 404", r)
	}
	if r := h.POST("/api/v1/merchant/deliveries/"+id+"/fetch-amount/approve", m.fx.Tok,
		map[string]any{}); r.Code != http.StatusBadRequest {
		t.Fatalf("FETCHAMT-33 موافقةٌ بلا نسخة: %s — يُنتظر 400", r)
	}
	if st := h.statusOf(id); st != "at_pickup" {
		t.Fatalf("الحال %s", st)
	}
}

// TestFETCHAMT_04_MerchantHoldsDeliveryCode **كودُ «إحضار» للمتجر** — يراه في توصيلته ويصله
// إشعار، **والسائقُ لا يراه** ويُدخله ليُسلّم. **وفي «توصيل» لا يراه المتجرُ أبداً.**
func TestFETCHAMT_04_MerchantHoldsDeliveryCode(t *testing.T) {
	h := New(t)
	h.Setting("delivery.code_required", "true")
	m := newMDFx(t, h, "منطقةُ FETCHAMT-04")
	owner := m.fx.M.Owner.ID
	id, drv := fetchCreate(t, h, m, "fetchamt-04", 0)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := fetchStep(h, drv, id, to); r.Code >= 400 {
			t.Fatalf("%s رُدّ: %s", to, r)
		}
	}
	code, _ := dcodeOf(t, h, id)
	if code == nil || len(*code) != 4 {
		t.Fatalf("FETCHAMT-40 لم يولد كودٌ عند الاستلام (%v)", code)
	}
	one := h.GET("/api/v1/merchant/deliveries/"+id, m.fx.Tok)
	if one.JSON()["delivery_code"] != *code {
		t.Fatalf("FETCHAMT-41 المتجرُ لا يرى الكود: %s", one)
	}
	list := h.GET(m.url("/deliveries"), m.fx.Tok)
	if !strings.Contains(string(list.Body), `"delivery_code":"`+*code+`"`) {
		t.Fatalf("FETCHAMT-42 قائمةُ المتجر بلا الكود: %s", list)
	}
	var n int
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notifications
		 WHERE user_id = $1::uuid AND entity_id = $2 AND body LIKE '%' || $3 || '%'`,
		owner, id, *code).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("FETCHAMT-43 إشعارُ الكود للمتجر = %d — يُنتظر ١", n)
	}
	dl := h.GET("/api/v1/driver/orders", drv.Token)
	if strings.Contains(string(dl.Body), `"delivery_code":`) ||
		!strings.Contains(string(dl.Body), `"delivery_code_required":true`) {
		t.Fatalf("FETCHAMT-44 السائقُ يرى الكود أو لا يعرف أنّه مطلوب: %s", dl)
	}
	for _, to := range []string{"on_the_way", "at_dropoff"} {
		if r := fetchStep(h, drv, id, to); r.Code >= 400 {
			t.Fatalf("%s رُدّ: %s", to, r)
		}
	}
	h.authorizeProofExempt(id)
	if r := dcodeDeliver(h, drv, id, wrongOf(*code)); r.Code != http.StatusUnprocessableEntity {
		t.Fatalf("FETCHAMT-45 كودٌ خاطئ: %s — يُنتظر 422", r)
	}
	if r := dcodeDeliver(h, drv, id, *code); r.Code >= 400 {
		t.Fatalf("FETCHAMT-46 كودُ المتجر رُدّ: %s", r)
	}
	if st := h.statusOf(id); st != "delivered" {
		t.Fatalf("FETCHAMT-46 الحال %s", st)
	}

	// **وفي «توصيل» الكودُ للمستلِم — لا يصل المتجر.**
	send := m.create(t, h, "recipient", uniq("fetchamt-04s"))
	m.accept(t, h, send)
	sdrv := h.driverOf(send)
	for _, to := range []string{"at_pickup", "picked_up"} {
		if r := fetchStep(h, sdrv, send, to); r.Code >= 400 {
			t.Fatalf("%s رُدّ: %s", to, r)
		}
	}
	if s := h.GET("/api/v1/merchant/deliveries/"+send, m.fx.Tok); strings.Contains(string(s.Body), `"delivery_code":`) {
		t.Fatalf("FETCHAMT-47 المتجرُ يرى كودَ المستلِم في «توصيل»: %s", s)
	}
}
