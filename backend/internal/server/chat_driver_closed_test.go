package server

// ══════════════════════════════════════════════════════════════════════
// **حديثُ الطلب يُقفل في وجه السائق لحظةَ ينتهي الطلب** (قرارُ المالك ٢٠٢٦-١١-٠٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كان السائقُ يقرأ أحاديثَ طلباته المنتهية إلى الأبد** — وفيها اسمُ الزبون وعنوانُه
// وما قاله، **فيعود إليه برّا المنصّة.** فصار: انتهى الطلبُ (سُلّم أو أُلغي أو فشل)
// ⇒ لا قراءةَ ولا كتابةَ للسائق، ولا يظهر في «محادثاتي» عنده. **والزبونُ يبقى له حديثُه،
// والإدارةُ تقرأ كلَّ شيءٍ أبداً** ومعه الأصلُ المخفيّ في `flag_word`.

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/servacode/rahalgo/backend/internal/comms"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/support"
)

func TestDriverChat_ClosedForDriverWhenOrderEnds(t *testing.T) {
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
	send := func(uid, role, text string) int {
		t.Helper()
		if _, err := f.pool.Exec(ctx,
			`UPDATE order_messages SET created_at = created_at - interval '1 minute' WHERE order_id = $1`, orderID); err != nil {
			t.Fatal(err)
		}
		return f.call(f.srv.handleSendOrderMessage, http.MethodPost, "/x", orderID, uid, []string{role},
			js(map[string]any{"body": text})).Code
	}
	read := func(uid, role string) (int, string) {
		t.Helper()
		w := f.call(f.srv.handleOrderMessages, http.MethodGet, "/x", orderID, uid, []string{role}, "")
		return w.Code, w.Body.String()
	}
	chats := func(uid, role string) string {
		t.Helper()
		w := f.call(f.srv.handleMyChats, http.MethodGet, "/x", "", uid, []string{role}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("محادثاتي: %d %s", w.Code, w.Body.String())
		}
		return w.Body.String()
	}

	// **والطلبُ في الطريق: الطرفان يكتبان ويقرآن.**
	if c := send(customer, "customer", "رقمي 0991234567 أنا نازل"); c != http.StatusCreated {
		t.Fatalf("الزبونُ لم يكتب: %d", c)
	}
	if c := send(d, "driver", "تمام أنا تحت"); c != http.StatusCreated {
		t.Fatalf("السائقُ لم يكتب: %d", c)
	}
	if c, body := read(d, "driver"); c != http.StatusOK || !strings.Contains(body, "أنا نازل") {
		t.Fatalf("السائقُ لا يقرأ حديثَ طلبٍ قائم: %d %s", c, body)
	}
	if !strings.Contains(chats(d, "driver"), orderID) {
		t.Fatalf("الطلبُ القائمُ غائبٌ عن محادثات السائق")
	}

	for _, end := range []string{"delivered", "cancelled", "failed"} {
		if _, err := f.pool.Exec(ctx, `UPDATE orders SET status = $2 WHERE id = $1`, orderID, end); err != nil {
			t.Fatal(err)
		}
		// **السائقُ: لا قراءةَ ولا كتابة** — ٤٠٣ برمز «المحادثة مغلقة».
		c, body := read(d, "driver")
		if c != http.StatusForbidden || !strings.Contains(body, "comms_closed") {
			t.Errorf("[%s] **السائقُ ما زال يقرأ حديثَ طلبٍ منتهٍ**: %d %s", end, c, body)
		}
		if strings.Contains(body, "أنا نازل") || strings.Contains(body, "أنا تحت") {
			t.Errorf("[%s] **نصُّ الحديث خرج للسائق**: %s", end, body)
		}
		if c := send(d, "driver", "كلمة بعد الانتهاء"); c != http.StatusForbidden {
			t.Errorf("[%s] **السائقُ كتب بعد الانتهاء**: %d", end, c)
		}
		if strings.Contains(chats(d, "driver"), orderID) {
			t.Errorf("[%s] **الطلبُ المنتهي ما زال في محادثات السائق**", end)
		}
		// **والزبونُ يقرأ حديثَه كما كان** — مغلقاً للكتابة.
		c, body = read(customer, "customer")
		if c != http.StatusOK || !strings.Contains(body, "أنا تحت") {
			t.Errorf("[%s] **الزبونُ فقد حديثَه**: %d %s", end, c, body)
		}
		if !strings.Contains(chats(customer, "customer"), orderID) {
			t.Errorf("[%s] **الطلبُ غاب عن محادثات الزبون**", end)
		}
		// **والإدارةُ تقرأ كلَّ شيء ومعه الأصلُ المخفيّ.**
		th, err := f.srv.comms.Audit(ctx, orderID)
		if err != nil {
			t.Fatal(err)
		}
		var lines, orig int
		for _, l := range th.Lines {
			lines++
			if strings.Contains(l.FlagWord, "0991234567") {
				orig++
			}
		}
		if lines < 2 || orig != 1 {
			t.Errorf("[%s] **سجلُّ الإدارة ناقص**: أسطر=%d أصل=%d", end, lines, orig)
		}
	}
}
