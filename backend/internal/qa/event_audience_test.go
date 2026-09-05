package qa

// عقودُ الجمهور والدفع — **`P-7` البنود ٨…١٥ و١٩ و٢١.**

import (
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/server"
)

// TestEV_OrderAudience **البند ٨ — من يجب أن يستقبل ومن يجب ألّا يستقبل.**
//
// **والطرفان يُختبَران** — **واختبارٌ يتحقّق من الوصول وحدَه لا يكشف
// تسريباً.**
func TestEV_OrderAudience(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 2500)
	cust := h.Customer()
	other := h.Customer()
	otherM := f.Merchant()
	drv := f.Driver(OnShift())

	caps := []struct {
		name string
		cap  *Capture
		must bool
	}{
		{"صاحبُ الطلب", h.Listen("customer:" + cust.ID), true},
		{"متجرُ الطلب", h.Listen("merchant:" + m.ID), true},
		{"زبونٌ آخر", h.Listen("customer:" + other.ID), false},
		{"متجرٌ آخر", h.Listen("merchant:" + otherM.ID), false},
		{"سائق", h.Listen("driver:" + drv.ID), false},
	}

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}

	ok, bad := 0, 0
	for _, c := range caps {
		got := c.cap.Next(2*time.Second) != nil
		t.Logf("%-14s استقبل: %v (يجب: %v)", c.name, got, c.must)
		switch {
		case c.must && got:
			ok++
		case c.must && !got:
			bad++
			t.Errorf("MUST RECEIVE خُرق: %s لم يستقبل", c.name)
		case !c.must && got:
			bad++
			t.Errorf("MUST NOT RECEIVE خُرق: %s استقبل", c.name)
		default:
			ok++
		}
	}
	t.Logf("ORDER AUDIENCE = %d/%d", ok, ok+bad)
}

// TestEV_WalletAudience **البند ٨ — حركةُ محفظةٍ لصاحبها وحدَه.**
func TestEV_WalletAudience(t *testing.T) {
	h := New(t)
	f := h.Factory()
	admin := h.NewUser("admin")
	owner := f.NewUserWith("customer")
	other := f.NewUserWith("customer")

	before := evNotifCount(t, h, owner.ID)
	beforeOther := evNotifCount(t, h, other.ID)

	got := h.POST("/api/v1/admin/users/"+owner.ID+"/wallet", admin.Token,
		map[string]any{"amount": 7000, "kind": "topup", "note": "P-7"})
	if got.Code >= 400 {
		t.Fatalf("القيد: %s", got)
	}
	waitNotif(h, owner.ID, before+1, 3*time.Second)

	after := evNotifCount(t, h, owner.ID)
	afterOther := evNotifCount(t, h, other.ID)
	t.Logf("صاحبُ المحفظة %d ⇒ %d · وغيرُه %d ⇒ %d", before, after, beforeOther, afterOther)
	if after <= before {
		t.Errorf("MUST RECEIVE خُرق: صاحبُ المحفظة لم يُشعَر")
	}
	if afterOther != beforeOther {
		t.Errorf("MUST NOT RECEIVE خُرق: غيرُ صاحبها أُشعِر")
	} else if after > before {
		t.Logf("WALLET AUDIENCE = PASS")
	}
}

func evNotifCount(t *testing.T, h *Harness, uid string) int {
	t.Helper()
	var n int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM notifications WHERE user_id = $1::uuid`, uid).Scan(&n)
	return n
}

// waitNotif ينتظر بلوغَ عددٍ — **والإشعارُ يُكتب في خيطٍ أحياناً.**
func waitNotif(h *Harness, uid string, want int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		var n int
		_ = h.Pool.QueryRow(ctxBG(),
			`SELECT count(*) FROM notifications WHERE user_id = $1::uuid`, uid).Scan(&n)
		if n >= want {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(3 * time.Millisecond)
	}
}

// TestEV_MerchantDriverAssignment **البند ١١ — يعلم ولا يرى هويّةَ السائق.**
func TestEV_MerchantDriverAssignment(t *testing.T) {
	h := New(t)
	f := h.Factory()
	treasury(t, h)
	h.Setting("drivers.assignment_mode", `"queue"`)
	h.Setting("drivers.max_active_orders", "5")
	h.Setting("drivers.cash_limit", "9000000")
	m := f.Merchant()
	item := h.NewItemFor(m, 2000)

	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	oid, _ := made.JSON()["id"].(string)
	_, _ = h.Pool.Exec(ctxBG(), `UPDATE orders SET status='dispatching' WHERE id=$1::uuid`, oid)

	cap := h.Listen("merchant:" + m.ID)
	drv := f.Driver(OnShift())
	if got := h.POST("/api/v1/driver/orders/"+oid+"/accept", drv.Token, nil); got.Code >= 400 {
		t.Fatalf("القبول: %s", got)
	}

	msgs := cap.Drain(2 * time.Second)
	t.Logf("وصل المتجرَ %d حمولة عند الإسناد", len(msgs))
	if len(msgs) == 0 {
		t.Logf("MERCHANT DRIVER ASSIGNMENT = EXPECTED FAIL — لا حمولةَ وصلت (MD-2)")
		return
	}
	leaks := 0
	for _, msg := range msgs {
		p := orderPayload(msg)
		for _, v := range CheckPayload("merchant", "realtime", p) {
			switch v.Field {
			case "driver_phone", "driver_id", "driver_name":
				leaks++
				t.Logf("  DRIVER IDENTITY LEAK — %s", v)
			}
		}
	}
	t.Logf("MERCHANT DRIVER ASSIGNMENT — أُعلِم=true · تسريبُ هويّةِ سائقٍ=%d", leaks)
	if leaks > 0 {
		t.Logf("  EXPECTED FAIL — هويّةُ السائق تصل المتجرَ (D20)")
	} else {
		t.Logf("  PASS — يعلم ولا يرى هويّةَ السائق")
	}
}

// TestEV_AutoAcceptMerchantAwareness **البند ١٣ — ولا يُغيَّر إعدادٌ عامّ.**
//
// **والإعدادُ يُفعَّل في نطاق هذا السيناريو وحدَه** — `h.Setting` يُرجَع
// بانتهاء الاختبار.
func TestEV_AutoAcceptMerchantAwareness(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 1800)
	h.Setting("orders.auto_accept_sec", "1")

	before := evNotifCount(t, h, m.Owner.ID)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	// **والزمنُ يُزاح في البيانة** — لا يُنتظَر.
	_, _ = h.Pool.Exec(ctxBG(),
		`UPDATE orders SET created_at = now() - interval '10 minutes' WHERE id=$1::uuid`, oid)

	var status string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM orders WHERE id=$1::uuid`, oid).Scan(&status)
	waitNotif(h, m.Owner.ID, before+1, 2*time.Second)
	after := evNotifCount(t, h, m.Owner.ID)
	t.Logf("حالُ الطلب=%q · إشعاراتُ المتجر %d ⇒ %d", status, before, after)

	// **ولا يُحكَم إلّا إن وقع القبولُ التلقائيّ.**
	//
	// **وأوّلُ صيغةٍ قالت `PASS` لأنّ المتجرَ أُشعِر** — **والإشعارُ كان
	// إشعارَ الطلب الجديد لا القبولَ التلقائيّ**، والحالُ `pending`.
	// **ونجاحٌ يُقاس بحدثٍ لم يقع ادّعاء.**
	if status != "accepted" {
		t.Logf("AUTO-ACCEPT = PARTIAL — لم يقع القبولُ التلقائيُّ هنا (الحالُ %q)", status)
		t.Logf("  والراصدُ يُنفّذه في دورته (sweepAutoAccept) — **ولا بابَ يُناديه**")
		t.Logf("  TESTABILITY SEAM REQUIRED — لإثبات وعيِ المتجر بالقبول التلقائيّ")
		return
	}
	if after > before {
		t.Logf("AUTO-ACCEPT MERCHANT AWARENESS = PASS — قُبل تلقائيّاً والمتجرُ أُشعِر")
	} else {
		t.Logf("AUTO-ACCEPT MERCHANT AWARENESS = EXPECTED FAIL — قُبل تلقائيّاً بلا إشعار (MD-5)")
	}
}

// TestEV_PushTokenTargeting **البند ٢١ — اختيارُ الرموز في الخادم.**
func TestEV_PushTokenTargeting(t *testing.T) {
	fake := NewFakePush()
	h := NewWith(t, server.WithPushTransport(fake))
	f := h.Factory()
	admin := h.NewUser("admin")
	owner := f.NewUserWith("customer")
	other := f.NewUserWith("customer")

	if !addToken(t, h, owner.ID, "tok-owner-1") {
		t.Skip("جدولُ رموز الدفع غيرُ متاحٍ بهذا الشكل")
	}
	addToken(t, h, owner.ID, "tok-owner-2")
	addToken(t, h, other.ID, "tok-other")

	got := h.POST("/api/v1/admin/users/"+owner.ID+"/wallet", admin.Token,
		map[string]any{"amount": 5000, "kind": "topup", "note": "P-7"})
	if got.Code >= 400 {
		t.Fatalf("القيد: %s", got)
	}
	if !fake.WaitCalls(1, 3*time.Second) {
		t.Skip("لم يُنادَ الدفعُ في هذه النافذة")
	}
	calls := fake.Calls()
	t.Logf("نداءاتُ الدفع = %d", len(calls))
	bad := 0
	for _, c := range calls {
		t.Logf("  الرموز=%v · العنوان=%q · البيانات=%v", c.Tokens, c.Title, c.Data)
		for _, tok := range c.Tokens {
			if tok == "tok-other" {
				bad++
				t.Errorf("PUSH TOKEN TARGETING خُرق: رمزُ مستخدمٍ آخرَ استُهدف")
			}
		}
		if c.Data["entity"] == "" && c.Data["entity_id"] == "" {
			t.Logf("  ENTITY/DEEP-LINK — لا entity ولا entity_id في المغلّف")
		}
		// **والخصوصيّةُ تُطبَّق على المغلّف** (البند ٣٢).
		//
		// **ومفاتيحُ المغلّف نفسِها تُستثنى** — `kind` و`entity` و`entity_id`
		// و`href` **أسماءُ المغلّف لا حقولُ طلب.**
		//
		// **و`kind` تصادمُ اسمٍ مقيسٌ في `P-1`**: في المغلّف نوعُ إشعار،
		// وفي الطلب نوعُ طلب. **والسؤالُ هو: أتركب حقولُ طلبٍ المغلَّف؟**
		env := map[string]any{}
		for k, v := range c.Data {
			switch k {
			case "kind", "entity", "entity_id", "href":
				continue
			}
			env[k] = v
		}
		if vs := CheckPayload("customer", "push", env); len(vs) > 0 {
			bad++
			t.Errorf("PUSH PRIVACY خُرق: %d حقلَ طلبٍ ركب المغلّف", len(vs))
			for _, v := range vs {
				t.Errorf("  %s", v)
			}
		}
	}
	if bad == 0 {
		t.Logf("PUSH TOKEN TARGETING = PASS — لم يُستهدَف رمزُ غيرِ صاحبه ولا حقلَ محظور")
	}
}

func addToken(t *testing.T, h *Harness, uid, token string) bool {
	t.Helper()
	_, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO device_tokens (token, user_id, platform, app)
		VALUES ($1, $2::uuid, 'android', 'customer') ON CONFLICT DO NOTHING`, token, uid)
	if err != nil {
		t.Logf("رموزُ الدفع: %v", err)
		return false
	}
	t.Cleanup(func() {
		_, _ = h.Pool.Exec(ctxBG(), `DELETE FROM device_tokens WHERE token = $1`, token)
	})
	return true
}

// TestEV_R23PushFailureIsLost **البند ١٥ — سقوطُ الدفع.**
//
// **وبالمِعراض المعتمَد** — ناقلٌ مبرمَجٌ يردّ `500` ثمّ مهلةً ثمّ قطعَ
// اتّصال.
func TestEV_R23PushFailureIsLost(t *testing.T) {
	fake := NewFakePush().Script(PushHTTP500, PushTimeout, PushConnReset)
	h := NewWith(t, server.WithPushTransport(fake))
	f := h.Factory()
	admin := h.NewUser("admin")
	u := f.NewUserWith("customer")
	if !addToken(t, h, u.ID, "tok-r23") {
		t.Skip("جدولُ رموز الدفع غيرُ متاح")
	}

	for i, label := range []string{"HTTP 500", "TIMEOUT", "CONNECTION RESET"} {
		got := h.POST("/api/v1/admin/users/"+u.ID+"/wallet", admin.Token,
			map[string]any{"amount": 1000, "kind": "topup", "note": "P-7"})
		fake.WaitCalls(i+1, 2*time.Second)
		t.Logf("%-18s ← الفعلُ ردّ %d", label, got.Code)
		// **البند ٢٩**: سقوطُ الدفع ليس سقوطَ العمل.
		if got.Code >= 400 {
			t.Errorf("PUSH FAILURE != BUSINESS FAILURE خُرق: الفعلُ سقط بسقوط الدفع")
		}
	}
	calls := fake.Calls()
	t.Logf("نداءاتُ الدفع = %d لثلاث عمليّات", len(calls))
	if len(calls) == 0 {
		t.Skip("لم يُنادَ الدفعُ — لا شيءَ يُقاس")
	}
	for _, c := range calls {
		t.Logf("  %s ← %v", c.Outcome, c.Err)
	}
	if len(calls) <= 3 {
		t.Logf("R23 FCM DELIVERY = RISK CONFIRMED")
		t.Logf("  الدفعُ يُحاوَل مرّةً واحدةً · **ولا إعادةَ ولا صفَّ انتظارٍ ولا حالٌ معلَّقة**")
		t.Logf("  ولا ظهورَ للإدارة — DEFECT CANDIDATE يُعرَض على المالك")
	} else {
		t.Logf("R23 = PASS — ثمّةَ إعادةٌ مقيسة (%d نداءً)", len(calls))
	}
}

// TestEV_R21AutoTransferAwareness **البند ١٩.**
func TestEV_R21AutoTransferAwareness(t *testing.T) {
	h := New(t)
	f := h.Factory()
	m := f.Merchant()
	item := h.NewItemFor(m, 1500)
	h.Setting("orders.auto_transfer_amount", "1")

	before := evNotifCount(t, h, m.Owner.ID)
	cap := h.Listen("merchant:" + m.ID)
	made := h.POSTKey("/api/v1/orders", h.Customer().Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("إنشاء: %s", made)
	}
	oid, _ := made.JSON()["id"].(string)
	msgs := cap.Drain(2 * time.Second)
	waitNotif(h, m.Owner.ID, before+1, 2*time.Second)
	after := evNotifCount(t, h, m.Owner.ID)

	var status string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT status FROM orders WHERE id=$1::uuid`, oid).Scan(&status)
	t.Logf("حالُ الطلب=%q · بثٌّ للمتجر=%d · إشعاراتُه %d ⇒ %d",
		status, len(msgs), before, after)

	// **ولا يُحكَم إلّا إن وقع التحويلُ فعلاً.**
	//
	// **وأوّلُ صيغةٍ قالت `PASS`** — **والحالُ `pending` فلا تحويلَ وقع**،
	// وما وصل المتجرَ إشعارُ الطلب الجديد. **ونجاحٌ يُقاس بحدثٍ لم يقع
	// ادّعاء.**
	if status == "pending" {
		t.Logf("R21 AUTO-TRANSFER COMMUNICATION = PARTIAL")
		t.Logf("  لم يقع تحويلٌ تلقائيٌّ في هذه النافذة — **والخيطُ لا يُنتظَر** (XOB-7)")
		t.Logf("  وما وصل المتجرَ إشعارُ الطلب الجديد لا خبرَ تحويل")
		return
	}
	if len(msgs) == 0 && after == before {
		t.Logf("R21 AUTO-TRANSFER COMMUNICATION = RISK CONFIRMED")
		t.Logf("  التحويلُ وقع (%q) **ولا بثَّ ولا إشعارَ يبلغ المتجرَ**", status)
	} else {
		t.Logf("R21 = PASS — التحويلُ وقع (%q) والمتجرُ يعلم عبر %d بثٍّ و%d إشعاراً",
			status, len(msgs), after-before)
	}
}
