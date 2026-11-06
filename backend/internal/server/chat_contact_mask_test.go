package server

// ══════════════════════════════════════════════════════════════════════
// **رقمُ الجوال والواتساب يُخفيان في حديث الطلب — بالاتّجاهين** (قرارُ المالك ٢٠٢٦-١١-٠٥، البند ١)
// ══════════════════════════════════════════════════════════════════════
//
// **«رقمي 0991234567 كلمني واتس» كانت تصل كما هي** — والحديثُ هو الباب الوحيد بين
// السائق والزبون. **فيُخفى «***» ويُوسَم السطرُ للإدارة كالشتيمة**، والرقمُ الأصليُّ
// في `flag_word` لمن يراجع. **والعدديُّ العاديّ لا يُمسّ**: سعرٌ ورقمُ بناية.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
	"github.com/servacode/rahalgo/backend/internal/testdb"
)

func TestChatContactMask_BothDirectionsFlaggedAndOpsAlerted(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.comms = comms.New(f.pool)
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	ops := testdb.NewUser(t, f.pool, "operations")
	orderID := f.problemOrderAt(t, "on_the_way", d)
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`,
		orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	send := func(uid, role, text string) {
		t.Helper()
		// **وحدُّ المعدّل** — رسالتان من المرسِل نفسِه في ثانيةٍ تُردّ الثانية؛ فيُزاح الزمن.
		if _, err := f.pool.Exec(ctx,
			`UPDATE order_messages SET created_at = created_at - interval '1 minute' WHERE order_id = $1`, orderID); err != nil {
			t.Fatal(err)
		}
		w := f.call(f.srv.handleSendOrderMessage, http.MethodPost, "/x", orderID, uid, []string{role},
			js(map[string]any{"body": text}))
		if w.Code != http.StatusCreated {
			t.Fatalf("الحديثُ رُفض: %d %s", w.Code, w.Body.String())
		}
	}
	type row struct {
		body, word    string
		flagged, leak bool
	}
	last := func() row {
		t.Helper()
		var r row
		if err := f.pool.QueryRow(ctx, `
			SELECT body, COALESCE(flag_word, ''), flagged, contact_leak FROM order_messages
			 WHERE order_id = $1 AND NOT auto ORDER BY created_at DESC LIMIT 1`, orderID).
			Scan(&r.body, &r.word, &r.flagged, &r.leak); err != nil {
			t.Fatal(err)
		}
		return r
	}
	alerts := func() int {
		t.Helper()
		var n int
		if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND title = $2`,
			ops, notifTitles.contactLeak).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// (١) الزبونُ يكتب رقمَه وكلمةَ الواتس.
	send(customer, "customer", "رقمي 0991234567 كلمني واتس")
	got := last()
	if got.body != "رقمي *** كلمني ***" || !got.flagged || !got.leak || !strings.Contains(got.word, "0991234567") {
		t.Errorf("**رسالةُ الزبون لم تُخفَ وتُوسَم**: %+v", got)
	}
	if n := alerts(); n != 1 {
		t.Errorf("**الإدارةُ نُبّهت %d مرّة** — والمتوقّع مرّة", n)
	}
	// **والسائقُ لا يصله الرقمُ ولا في إشعاره.**
	var leaked int
	if err := f.pool.QueryRow(ctx, `SELECT count(*) FROM notifications WHERE user_id = $1 AND body LIKE '%0991234567%'`,
		d).Scan(&leaked); err != nil {
		t.Fatal(err)
	}
	if leaked != 0 {
		t.Errorf("**الرقمُ وصل السائقَ في إشعار الرسالة**")
	}
	// **ولا رقمَ في نصّ تنبيه الإدارة** — يُقرأ على شاشةٍ مقفلة.
	var opsBody string
	if err := f.pool.QueryRow(ctx, `SELECT body FROM notifications WHERE user_id = $1 AND title = $2 LIMIT 1`,
		ops, notifTitles.contactLeak).Scan(&opsBody); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(opsBody, "0991234567") {
		t.Errorf("**الرقمُ في نصّ تنبيه الإدارة**: %q", opsBody)
	}

	// (٢) والسائقُ يكتب رقمَه — الاتّجاهُ الآخر.
	send(d, "driver", "+963 991 234 567")
	got = last()
	if got.body != "***" || !got.flagged || !got.leak {
		t.Errorf("**رسالةُ السائق لم تُخفَ**: %+v", got)
	}
	if n := alerts(); n != 2 {
		t.Errorf("**الإدارةُ نُبّهت %d مرّة** — والمتوقّع مرّتان", n)
	}

	// (٣) والعدديُّ العاديّ يصل كما كُتب، بلا وسمٍ ولا تنبيه.
	send(customer, "customer", "السعر 25000 والبناية 12")
	got = last()
	if got.body != "السعر 25000 والبناية 12" || got.flagged || got.leak {
		t.Errorf("**البريءُ مُسّ**: %+v", got)
	}
	if n := alerts(); n != 2 {
		t.Errorf("**نُبّهت الإدارةُ على بريء**: %d", n)
	}

	// (٤) وسجلُّ الإدارة يقول «رقم/واتساب مخفي».
	thread, err := f.srv.comms.Audit(ctx, orderID)
	if err != nil {
		t.Fatal(err)
	}
	leaks := 0
	for _, l := range thread.Lines {
		if l.ContactLeak {
			leaks++
		}
	}
	if leaks != 2 {
		t.Errorf("**سجلُّ الإدارة يعدّ %d سطراً مخفيّاً** — والمتوقّع ٢", leaks)
	}
}

// TestDriverOrders_MasksContactsInCustomerText **وما كتبه الزبونُ في طلبه لا يحمل رقمَه
// إلى السائق** (البند ١ج) — العنوانُ ونصُّ الطلب الخاصّ في «طلباتي»، والأصلُ في القاعدة
// للإدارة.
func TestDriverOrders_MasksContactsInCustomerText(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "on_the_way", d)
	if _, err := f.pool.Exec(ctx, `
		UPDATE orders SET address_text = 'بناية الأمل — اتصل 0991234567', custom_request = 'كلمني عالواتس'
		 WHERE id = $1`, orderID); err != nil {
		t.Fatal(err)
	}
	w := f.call(f.srv.handleDriverOrders, http.MethodGet, "/driver/orders", "", d, []string{"driver"}, "")
	if w.Code != http.StatusOK {
		t.Fatalf("طلباتي: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "0991234567") || strings.Contains(w.Body.String(), "عالواتس") {
		t.Errorf("**رقمُ الزبون في «طلباتي»**: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "بناية الأمل") {
		t.Errorf("**العنوانُ نفسُه غاب**: %s", w.Body.String())
	}
	var stored string
	if err := f.pool.QueryRow(ctx, `SELECT address_text FROM orders WHERE id = $1`, orderID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stored, "0991234567") {
		t.Errorf("**الأصلُ تغيّر في القاعدة**: %q", stored)
	}
}

// TestChatContactMask_SocialAndEmail **تلغرام وفيسبوك وانستغرام والبريد تُخفى في الحديث
// كالرقم** (قرارُ المالك ٢٠٢٦-١١-٠٦) — والأصلُ في `flag_word` للإدارة.
func TestChatContactMask_SocialAndEmail(t *testing.T) {
	f := newDriverFixture(t, 1)
	d := f.drivers[0]
	f.srv.comms = comms.New(f.pool)
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	ctx := context.Background()
	orderID := f.problemOrderAt(t, "on_the_way", d)
	var customer string
	if err := f.pool.QueryRow(ctx, `SELECT customer_id::text FROM orders WHERE id = $1`,
		orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	js := func(v any) string { b, _ := json.Marshal(v); return string(b) }
	for _, c := range []struct{ uid, role, in, want, orig string }{
		{d, "driver", "ضيفني تلغرام @ahmad_99", "ضيفني *** ***", "@ahmad_99"},
		{customer, "customer", "هاد حسابي facebook.com/ahmad", "هاد حسابي ***", "facebook.com/ahmad"},
		{d, "driver", "ايميلي ahmad@gmail.com", "ايميلي ***", "ahmad@gmail.com"},
	} {
		if _, err := f.pool.Exec(ctx,
			`UPDATE order_messages SET created_at = created_at - interval '1 minute' WHERE order_id = $1`, orderID); err != nil {
			t.Fatal(err)
		}
		w := f.call(f.srv.handleSendOrderMessage, http.MethodPost, "/x", orderID, c.uid, []string{c.role},
			js(map[string]any{"body": c.in}))
		if w.Code != http.StatusCreated {
			t.Fatalf("الحديثُ رُفض: %d %s", w.Code, w.Body.String())
		}
		var body, word string
		var leak bool
		if err := f.pool.QueryRow(ctx, `
			SELECT body, COALESCE(flag_word, ''), contact_leak FROM order_messages
			 WHERE order_id = $1 AND NOT auto ORDER BY created_at DESC LIMIT 1`, orderID).
			Scan(&body, &word, &leak); err != nil {
			t.Fatal(err)
		}
		if body != c.want || !leak || !strings.Contains(word, c.orig) {
			t.Errorf("%q ⇒ نصّ=%q وسم=%v أصل=%q", c.in, body, leak, word)
		}
	}
}
