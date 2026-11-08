package qa

// ══════════════════════════════════════════════════════════════════════
// **الدفع بشام كاش** (قرار المالك ٢٠٢٦-١١-٠٩) — على خادم شام كاش مزيّف بنفس البروتوكول المشفّر
// ══════════════════════════════════════════════════════════════════════
//
// فاتورة ← دفع ← إشعار ← تأكيد بـgetBill ← إيداع مرّة وحدة · إشعار مكرّر · مزوّر · إشعار
// كذّاب · إشعار ضاع (الفاحص) · انتهاء · إرجاع جزئيّ وزائد · مطفي · دفع طلب ونجاحه وفشله
// وانتهاؤه — **وثوابت الدفتر (ومنها FI-16 وFI-15.g) ما بيستجدّ فيها خرق.**

import (
	"encoding/json"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/server"
	"github.com/servacode/rahalgo/backend/internal/shamcash"
)

const shamHook = "/api/v1/public/shamcash/webhook"

func newShamHarness(t *testing.T) (*Harness, *shamcash.Fake) {
	t.Helper()
	f := shamcash.NewFake()
	t.Cleanup(f.Close)
	c, err := shamcash.New(f.Config("https://api.example.test"+shamHook, "https://shop.example.test"))
	if err != nil {
		t.Fatal(err)
	}
	h := NewWith(t, server.WithShamCash(c))
	h.Setting("customers.shamcash_enabled", "true")
	h.Setting("customers.shamcash_min", "10")
	h.Setting("customers.shamcash_max", "100000")
	treasury(t, h)
	return h, f
}

func shamBalance(t *testing.T, h *Harness, uid string) int64 {
	t.Helper()
	var b int64
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE((SELECT balance FROM wallets WHERE user_id = $1::uuid), 0)`, uid).Scan(&b); err != nil {
		t.Fatal(err)
	}
	return b
}

// shamStart فاتورة شحن — ويرجّع المعرّف ورقم شام كاش.
func shamStart(t *testing.T, h *Harness, u *User, amount int64) (string, string) {
	t.Helper()
	r := h.POST("/api/v1/my/shamcash/topups", u.Token, map[string]any{"amount": amount, "return": "web"})
	if r.Code != 201 {
		t.Fatalf("فاتورة الشحن: %s", r)
	}
	bill, _ := r.JSON()["bill"].(map[string]any)
	if r.JSON()["payment_url"] == "" || bill["status"] != "pending" {
		t.Fatalf("فاتورة بلا رابط أو بغير حالة: %s", r)
	}
	return bill["id"].(string), bill["number"].(string)
}

func shamStatus(t *testing.T, h *Harness, u *User, id string) map[string]any {
	t.Helper()
	r := h.GET("/api/v1/my/shamcash/bills/"+id, u.Token)
	if r.Code != 200 {
		t.Fatalf("قراءة الفاتورة: %s", r)
	}
	b, _ := r.JSON()["bill"].(map[string]any)
	return b
}

func shamWebhook(h *Harness, body []byte) Res {
	return h.POST(shamHook, "", json.RawMessage(body))
}

func shamCount(t *testing.T, h *Harness, sql string, args ...any) int {
	t.Helper()
	var n int
	if err := h.Pool.QueryRow(ctxBG(), sql, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// TestSHAM_SwitchOffIsUnavailable مطفي أو بلا مفاتيح ⇒ «مو متاح» والخيار مخفي.
func TestSHAM_SwitchOffIsUnavailable(t *testing.T) {
	// بلا مفاتيح (الإنتاج اليوم)
	h := New(t)
	h.Setting("customers.shamcash_enabled", "true")
	u := h.Customer()
	if r := h.GET("/api/v1/my/shamcash", u.Token); r.Code != 200 || r.JSON()["enabled"] != false {
		t.Fatalf("بلا مفاتيح والخيار ظاهر: %s", r)
	}
	if r := h.POST("/api/v1/my/shamcash/topups", u.Token, map[string]any{"amount": 100}); r.Err() != "shamcash_unavailable" {
		t.Fatalf("بلا مفاتيح: %s", r)
	}
	if r := shamWebhook(h, []byte(`{"encData":"x"}`)); r.Code != 503 {
		t.Fatalf("إشعار بلا مفاتيح: %s", r)
	}

	// مفاتيح موجودة والإعداد مطفي
	h2, _ := newShamHarness(t)
	h2.Setting("customers.shamcash_enabled", "false")
	u2 := h2.Customer()
	if r := h2.GET("/api/v1/my/shamcash", u2.Token); r.JSON()["enabled"] != false {
		t.Fatalf("مطفي والخيار ظاهر: %s", r)
	}
	if r := h2.POST("/api/v1/my/shamcash/topups", u2.Token, map[string]any{"amount": 100}); r.Err() != "shamcash_unavailable" {
		t.Fatalf("مطفي: %s", r)
	}
	if r := h2.POST("/api/v1/my/shamcash/orders", u2.Token, map[string]any{"order": map[string]any{}}); r.Err() != "shamcash_unavailable" {
		t.Fatalf("مطفي (طلب): %s", r)
	}
}

// TestSHAM_TopupPaidOnceAndDrawerUntouched فاتورة ← دفع ← إشعار ← إيداع مرّة وحدة.
func TestSHAM_TopupPaidOnceAndDrawerUntouched(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()

	if r := h.POST("/api/v1/my/shamcash/topups", u.Token, map[string]any{"amount": 5}); r.Err() != "shamcash_amount_out_of_range" {
		t.Fatalf("مبلغ تحت الحدّ: %s", r)
	}
	id, no := shamStart(t, h, u, 500)
	if b := shamBalance(t, h, u.ID); b != 0 {
		t.Fatalf("انودع قبل الدفع: %d", b)
	}
	body := f.Pay(no)
	if r := shamWebhook(h, body); r.Code != 200 {
		t.Fatalf("الإشعار: %s", r)
	}
	if b := shamBalance(t, h, u.ID); b != 500 {
		t.Fatalf("بعد الدفع الرصيد %d والمنتظر ٥٠٠", b)
	}
	// **الإشعار بيتعاد** (شام كاش بتعيد لحدّ ٢٠٠) — ما بينودع تاني
	for i := 0; i < 3; i++ {
		if r := shamWebhook(h, body); r.Code != 200 {
			t.Fatalf("الإشعار المكرّر: %s", r)
		}
	}
	if b := shamBalance(t, h, u.ID); b != 500 {
		t.Fatalf("إيداع مكرّر: الرصيد %d", b)
	}
	if st := shamStatus(t, h, u, id); st["status"] != "paid" || st["paid_at"] == nil {
		t.Fatalf("الحالة: %v", st)
	}
	if n := shamCount(t, h, `SELECT count(*) FROM office_cash_entries WHERE ref = $1`, id); n != 0 {
		t.Fatalf("مال شام كاش دخل درج المكتب: %d سطر", n)
	}
	if n := shamCount(t, h, `SELECT count(*) FROM shamcash_account_entries WHERE ref = $1::uuid AND source = 'bill_paid' AND amount = 500`, id); n != 1 {
		t.Fatalf("دفتر شام كاش: %d", n)
	}
	// والفاتورة لغير صاحبها ٤٠٤
	if r := h.GET("/api/v1/my/shamcash/bills/"+id, h.Customer().Token); r.Code != 404 {
		t.Fatalf("فاتورة غيره: %s", r)
	}
	assertNewViolations(t, h, base)
}

// TestSHAM_ForgedAndLyingWebhooks مزوّر ⇒ ٤٠٠ · «مدفوع» وgetBill بيقول لأ ⇒ ولا شي.
func TestSHAM_ForgedAndLyingWebhooks(t *testing.T) {
	h, f := newShamHarness(t)
	u := h.Customer()
	id, no := shamStart(t, h, u, 300)

	other := shamcash.NewFake() // مفتاح تاني تماماً
	defer other.Close()
	if r := shamWebhook(h, other.Webhook(no, nil, shamcash.StatusPaid)); r.Code != 400 {
		t.Fatalf("إشعار مزوّر انقبل: %s", r)
	}
	if r := shamWebhook(h, []byte(`{"encData":"not.a.token.at.all"}`)); r.Code != 400 {
		t.Fatalf("إشعار ما بينفتح انقبل: %s", r)
	}
	tran := int64(42)
	if r := shamWebhook(h, f.Webhook(no, &tran, shamcash.StatusPaid)); r.Code != 200 {
		t.Fatalf("إشعار كذّاب: %s", r)
	}
	if b := shamBalance(t, h, u.ID); b != 0 {
		t.Fatalf("انودع على كلمة الإشعار بلا تأكيد: %d", b)
	}
	if st := shamStatus(t, h, u, id); st["status"] != "pending" {
		t.Fatalf("الحالة تغيّرت: %v", st)
	}
	// فاتورة مو إلنا ⇒ ٢٠٠ ولا شي
	if r := shamWebhook(h, f.Webhook("RG-NOT-OURS", nil, shamcash.StatusPaid)); r.Code != 200 {
		t.Fatalf("فاتورة غريبة: %s", r)
	}
}

// TestSHAM_MissedWebhookPollerCredits الإشعار ضاع ⇒ الفاحص بيسأل بعد انتهاء الفاتورة ويودع.
func TestSHAM_MissedWebhookPollerCredits(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()
	id, no := shamStart(t, h, u, 700)
	f.Pay(no) // بلا إشعار

	h.API.SweepShamCash(ctxBG()) // لسّا ما خلصت العشر دقايق ⇒ ما بيسأل
	if b := shamBalance(t, h, u.ID); b != 0 {
		t.Fatalf("الفاحص سأل قبل وقته: %d", b)
	}
	before := f.Calls["getBillInfo"]
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE shamcash_bills SET created_at = now() - interval '12 minutes' WHERE id = $1`, id); err != nil {
		t.Fatal(err)
	}
	h.API.SweepShamCash(ctxBG())
	if b := shamBalance(t, h, u.ID); b != 700 {
		t.Fatalf("الفاحص ما أودع: %d", b)
	}
	h.API.SweepShamCash(ctxBG())
	if f.Calls["getBillInfo"]-before != 1 {
		t.Fatalf("الفاحص سأل %d مرّة — المحسومة ما بتنسأل", f.Calls["getBillInfo"]-before)
	}
	assertNewViolations(t, h, base)
}

// TestSHAM_Expiry انتهاء عند شام كاش، وانتهاء عندنا بعد مهلة طويلة.
func TestSHAM_Expiry(t *testing.T) {
	h, f := newShamHarness(t)
	u := h.Customer()
	id, no := shamStart(t, h, u, 200)
	if r := shamWebhook(h, f.Expire(no)); r.Code != 200 {
		t.Fatalf("إشعار الانتهاء: %s", r)
	}
	if st := shamStatus(t, h, u, id); st["status"] != "expired" {
		t.Fatalf("الحالة: %v", st)
	}

	id2, _ := shamStart(t, h, u, 200) // معلّقة عند شام كاش للأبد
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE shamcash_bills SET created_at = now() - interval '31 minutes' WHERE id = $1`, id2); err != nil {
		t.Fatal(err)
	}
	h.API.SweepShamCash(ctxBG())
	if st := shamStatus(t, h, u, id2); st["status"] != "expired" {
		t.Fatalf("مهلة محلّية: %v", st)
	}
	if b := shamBalance(t, h, u.ID); b != 0 {
		t.Fatalf("رصيد بلا دفع: %d", b)
	}
}

// TestSHAM_RefundPartialAndOver إرجاع جزئيّ ⇒ خصم من المحفظة ودفتر طالع · زائد ⇒ مرفوض.
func TestSHAM_RefundPartialAndOver(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()
	admin := h.NewUser("admin")
	id, no := shamStart(t, h, u, 1000)
	shamWebhook(h, f.Pay(no))

	refund := func(amount int64) Res {
		return h.POSTKey("/api/v1/admin/shamcash/bills/"+id+"/refund", admin.Token, uniq("rf"),
			map[string]any{"amount": amount, "note": "test"})
	}
	if r := refund(400); r.Code != 201 || r.JSON()["refund"].(map[string]any)["status"] != "done" {
		t.Fatalf("إرجاع جزئيّ: %s", r)
	}
	if b := shamBalance(t, h, u.ID); b != 600 {
		t.Fatalf("بعد الإرجاع الرصيد %d والمنتظر ٦٠٠", b)
	}
	if f.Status(no) != shamcash.StatusPartlyRefunded {
		t.Fatalf("حالة شام كاش %d", f.Status(no))
	}
	if r := refund(700); r.Err() != "shamcash_refund_exceeds" {
		t.Fatalf("إرجاع زائد انقبل: %s", r)
	}
	if b := shamBalance(t, h, u.ID); b != 600 {
		t.Fatalf("الإرجاع الزائد مسّ الرصيد: %d", b)
	}
	// الزبون صرف رصيده ⇒ ما في شي يُخصم
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE wallets SET balance = balance - 550 WHERE user_id = $1`, u.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.Pool.Exec(ctxBG(), `INSERT INTO wallet_transactions (user_id, amount, kind, note) VALUES ($1, -550, 'adjustment', 'test spend')`, u.ID); err != nil {
		t.Fatal(err)
	}
	if r := refund(100); r.Err() != "insufficient_balance" {
		t.Fatalf("إرجاع أكبر من الرصيد: %s", r)
	}
	// موظّف عمليّات ما بيقدر
	if r := h.POSTKey("/api/v1/admin/shamcash/bills/"+id+"/refund", h.NewUser("operations").Token, uniq("rf"),
		map[string]any{"amount": 10}); r.Code != 403 {
		t.Fatalf("العمليّات رجّعت مال: %s", r)
	}
	// والقائمة والمطابقة
	if r := h.GET("/api/v1/admin/shamcash/bills?q="+no, admin.Token); r.Code != 200 || r.JSON()["total"].(float64) != 1 ||
		r.JSON()["book_balance"].(float64) < 600 {
		t.Fatalf("القائمة: %s", r)
	}
	assertNewViolations(t, h, base, "FI-16", "FI-15", "FI-02.a")
}

// TestSHAM_RefundRejectedRestoresWallet شام كاش رفضت الإرجاع ⇒ الخصم بيرجع للمحفظة.
func TestSHAM_RefundRejectedRestoresWallet(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()
	admin := h.NewUser("admin")
	id, no := shamStart(t, h, u, 500)
	shamWebhook(h, f.Pay(no))
	// عطل نقل ⇒ يبقى معلّق، والفاحص بيعيد بنفس المفتاح
	f.FailNext("refundBill", 3)
	r := h.POSTKey("/api/v1/admin/shamcash/bills/"+id+"/refund", admin.Token, uniq("rf"), map[string]any{"amount": 200})
	if r.Code != 201 || r.JSON()["refund"].(map[string]any)["status"] != "pending" {
		t.Fatalf("إرجاع بعطل نقل: %s", r)
	}
	if b := shamBalance(t, h, u.ID); b != 300 {
		t.Fatalf("الخصم المعلّق: %d", b)
	}
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE shamcash_refunds SET created_at = now() - interval '2 minutes' WHERE bill_id = $1`, id); err != nil {
		t.Fatal(err)
	}
	h.API.SweepShamCash(ctxBG())
	var rst, rerr string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status, error FROM shamcash_refunds WHERE bill_id = $1`, id).Scan(&rst, &rerr)
	t.Logf("refund after sweep: %s %s calls=%d", rst, rerr, f.Calls["refundBill"])
	if n := shamCount(t, h, `SELECT count(*) FROM shamcash_refunds r JOIN shamcash_bills b ON b.id = r.bill_id WHERE b.id = $1 AND r.status = 'done'`, id); n != 1 {
		t.Fatalf("الفاحص ما حسم الإرجاع: %d", n)
	}
	assertNewViolations(t, h, base, "FI-16", "FI-02.a")
}

// TestSHAM_OrderPaidThenCreated دفع طلب ⇒ إيداع ثمّ الطلب من المحفظة بنفس الطريق.
func TestSHAM_OrderPaidThenCreated(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()
	item := h.NewItem(1000)
	body := orderBody(item, 2)
	delete(body, "payment_method")
	before := h.CountOrders(u.ID)

	r := h.POST("/api/v1/my/shamcash/orders", u.Token, map[string]any{"order": body, "return": "app"})
	if r.Code != 201 {
		t.Fatalf("فاتورة الطلب: %s", r)
	}
	bill := r.JSON()["bill"].(map[string]any)
	id, no, amount := bill["id"].(string), bill["number"].(string), int64(bill["amount"].(float64))
	if h.CountOrders(u.ID) != before {
		t.Fatal("انعمل طلب قبل الدفع")
	}
	if amount <= 0 {
		t.Fatalf("مبلغ الفاتورة %d", amount)
	}
	shamWebhook(h, f.Pay(no))
	st := shamStatus(t, h, u, id)
	if st["status"] != "paid" || st["order_id"] == nil || st["order_error"] != "" {
		t.Fatalf("بعد الدفع: %v", st)
	}
	var total, walletPaid int64
	var method string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT total, wallet_paid, payment_method FROM orders WHERE id = $1`, st["order_id"]).
		Scan(&total, &walletPaid, &method); err != nil {
		t.Fatal(err)
	}
	if method != "wallet" || walletPaid != total || total != amount {
		t.Fatalf("الطلب: %s مدفوع %d من %d والفاتورة %d", method, walletPaid, total, amount)
	}
	if b := shamBalance(t, h, u.ID); b != 0 {
		t.Fatalf("الرصيد بعد الطلب %d", b)
	}
	shamWebhook(h, f.Webhook(no, nil, shamcash.StatusPaid)) // مكرّر
	if h.CountOrders(u.ID) != before+1 {
		t.Fatalf("طلبات: %d", h.CountOrders(u.ID)-before)
	}
	assertNewViolations(t, h, base)
}

// TestSHAM_OrderFailsAfterPaymentKeepsMoney اندفعت والصنف خلص ⇒ المبلغ بالمحفظة ولا طلب.
func TestSHAM_OrderFailsAfterPaymentKeepsMoney(t *testing.T) {
	h, f := newShamHarness(t)
	base := financialBaseline(t, h)
	u := h.Customer()
	item := h.NewItem(800)
	body := orderBody(item, 1)
	delete(body, "payment_method")
	before := h.CountOrders(u.ID)
	r := h.POST("/api/v1/my/shamcash/orders", u.Token, map[string]any{"order": body})
	if r.Code != 201 {
		t.Fatalf("فاتورة الطلب: %s", r)
	}
	bill := r.JSON()["bill"].(map[string]any)
	id, no, amount := bill["id"].(string), bill["number"].(string), int64(bill["amount"].(float64))
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE menu_items SET available = false WHERE id = $1`, item.ID); err != nil {
		t.Fatal(err)
	}
	shamWebhook(h, f.Pay(no))
	st := shamStatus(t, h, u, id)
	if st["status"] != "paid" || st["order_id"] != nil || st["order_error"] == "" {
		t.Fatalf("بعد الدفع: %v", st)
	}
	if b := shamBalance(t, h, u.ID); b != amount {
		t.Fatalf("المبلغ لازم يضلّ بالمحفظة: %d من %d", b, amount)
	}
	if h.CountOrders(u.ID) != before {
		t.Fatal("انعمل طلب")
	}
	assertNewViolations(t, h, base)
}

// TestSHAM_OrderBillExpiresNoOrder انتهت فاتورة الطلب ⇒ لا طلب ولا مال.
func TestSHAM_OrderBillExpiresNoOrder(t *testing.T) {
	h, f := newShamHarness(t)
	u := h.Customer()
	item := h.NewItem(500)
	body := orderBody(item, 1)
	delete(body, "payment_method")
	before := h.CountOrders(u.ID)
	r := h.POST("/api/v1/my/shamcash/orders", u.Token, map[string]any{"order": body})
	if r.Code != 201 {
		t.Fatalf("فاتورة الطلب: %s", r)
	}
	bill := r.JSON()["bill"].(map[string]any)
	shamWebhook(h, f.Expire(bill["number"].(string)))
	if st := shamStatus(t, h, u, bill["id"].(string)); st["status"] != "expired" || st["order_id"] != nil {
		t.Fatalf("الحالة: %v", st)
	}
	if h.CountOrders(u.ID) != before || shamBalance(t, h, u.ID) != 0 {
		t.Fatal("طلب أو مال بلا دفع")
	}
	// سلّة غلط ⇒ ما في فاتورة أصلاً
	bad := orderBody(item, 0)
	delete(bad, "payment_method")
	if r := h.POST("/api/v1/my/shamcash/orders", u.Token, map[string]any{"order": bad}); r.Code < 400 {
		t.Fatalf("سلّة غلط عملت فاتورة: %s", r)
	}
}
