package server

// ══════════════════════════════════════════════════════════════════════
// **حديثُ «لدي توصيلة» بين المتجر وكابتنه** (قرارُ المالك ٢٠٢٦-١١-٢٦)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت `Permit` تضمّ الزبونَ ضمّاً داخليّاً** — والتوصيلةُ لا زبونَ لها، **فيردّ «لستَ طرفاً»
// للمتجر والكابتن معاً** وزرُّ الحديث مخفيٌّ عند الكابتن. **والآن المتجرُ مكانَ الزبون**:
// يُفتح من الإسناد ويُغلق بالتسليم، **والرقمُ يُخفى، والإشعارُ في تطبيق الطرف الآخر.**

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

type storeChatFixture struct {
	*driverFixture
	owner, driver, order string
}

func newStoreChatFixture(t *testing.T, status string) *storeChatFixture {
	t.Helper()
	f := newDriverFixture(t, 2)
	f.srv.comms = comms.New(f.pool)
	f.srv.support = support.NewService(f.pool, f.srv.identity, f.srv.wallet)
	f.srv.notify = notifications.New(f.pool, f.srv.hub, f.srv.logger)
	owner := testdb.NewUser(t, f.pool, "merchant")
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE merchants SET owner_user_id = $2 WHERE id = $1`, f.merchantID, owner); err != nil {
		t.Fatal(err)
	}
	d := f.drivers[0]
	order := f.deliveryOrderAt(t, status, d, true, "merchant")
	return &storeChatFixture{driverFixture: f, owner: owner, driver: d, order: order}
}

func (f *storeChatFixture) send(t *testing.T, uid, role, text string) (int, string) {
	t.Helper()
	// **حدُّ المعدّل** — يُزاح الزمنُ كي لا تُردّ الرسالةُ الثانية.
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE order_messages SET created_at = created_at - interval '1 minute' WHERE order_id = $1`, f.order); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"body": text})
	w := f.call(f.srv.handleSendOrderMessage, http.MethodPost, "/x", f.order, uid, []string{role}, string(b))
	return w.Code, w.Body.String()
}

type chatHead struct {
	Messages []struct {
		Body string `json:"body"`
		Role string `json:"role"`
		Mine bool   `json:"mine"`
	} `json:"messages"`
	PeerName string `json:"peer_name"`
	Open     bool   `json:"open"`
}

func (f *storeChatFixture) read(t *testing.T, uid, role string) (int, chatHead) {
	t.Helper()
	w := f.call(f.srv.handleOrderMessages, http.MethodGet, "/x", f.order, uid, []string{role}, "")
	var env struct {
		Data chatHead `json:"data"`
	}
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
	}
	return w.Code, env.Data
}

// TestStoreCaptainChat_BothPartiesTalk **المتجرُ والكابتنُ يتراسلان** — وكلٌّ يرى الآخرَ باسمه الصحيح.
func TestStoreCaptainChat_BothPartiesTalk(t *testing.T) {
	f := newStoreChatFixture(t, "assigned")

	if code, body := f.send(t, f.owner, "merchant", "الكيس عند الكاشير"); code != http.StatusCreated {
		t.Fatalf("**المتجرُ لم يكتب**: %d %s", code, body)
	}
	code, h := f.read(t, f.driver, "driver")
	if code != http.StatusOK {
		t.Fatalf("**الكابتنُ لم يقرأ**: %d", code)
	}
	if h.PeerName != "متجر اختبار السائق" {
		t.Errorf("**الكابتنُ يرى %q** — والمتوقّع اسمُ المتجر لا «الزبون»", h.PeerName)
	}
	if !h.Open || len(h.Messages) != 1 || h.Messages[0].Role != "merchant" || h.Messages[0].Mine {
		t.Errorf("**حديثُ الكابتن غلط**: %+v", h)
	}

	if code, body := f.send(t, f.driver, "driver", "وصلت، طالع"); code != http.StatusCreated {
		t.Fatalf("**الكابتنُ لم يكتب**: %d %s", code, body)
	}
	code, h = f.read(t, f.owner, "merchant")
	if code != http.StatusOK {
		t.Fatalf("**المتجرُ لم يقرأ**: %d", code)
	}
	if h.PeerName != comms.CustomerDriverLabel {
		t.Errorf("**المتجرُ يرى %q** — والمتوقّع «%s»", h.PeerName, comms.CustomerDriverLabel)
	}
	if len(h.Messages) != 2 || !h.Messages[0].Mine || h.Messages[1].Role != "driver" {
		t.Errorf("**حديثُ المتجر غلط**: %+v", h)
	}

	// **والإشعارُ في تطبيق الطرف الآخر** — الكابتنُ في تطبيقه بعنوان «رسالة من المتجر»،
	// والمتجرُ في تطبيق متجره لا تطبيق الزبون.
	var title string
	var apps []string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT title, push_apps FROM notifications WHERE user_id = $1 AND kind = 'chat'
		 ORDER BY created_at DESC LIMIT 1`, f.driver).Scan(&title, &apps); err != nil {
		t.Fatalf("**لا إشعارَ للكابتن**: %v", err)
	}
	if !strings.HasPrefix(title, notifTitles.messageFromStore) || len(apps) != 1 || apps[0] != notifications.AppDriver {
		t.Errorf("**إشعارُ الكابتن غلط**: %q %v", title, apps)
	}
	if err := f.pool.QueryRow(context.Background(), `
		SELECT push_apps FROM notifications WHERE user_id = $1 AND kind = 'chat'
		 ORDER BY created_at DESC LIMIT 1`, f.owner).Scan(&apps); err != nil {
		t.Fatalf("**لا إشعارَ للمتجر**: %v", err)
	}
	if len(apps) != 1 || apps[0] != notifications.AppMerchant {
		t.Errorf("**إشعارُ المتجر في تطبيقٍ غلط**: %v", apps)
	}

	// **وسجلُّ الإدارة يقول المتجرَ باسمه.**
	th, err := f.srv.comms.Audit(context.Background(), f.order)
	if err != nil {
		t.Fatal(err)
	}
	if th.Store != "متجر اختبار السائق" || len(th.Lines) != 2 || th.Lines[0].Sender != "متجر اختبار السائق" ||
		th.Lines[0].Role != "merchant" {
		t.Errorf("**سجلُّ الإدارة غلط**: %+v", th)
	}

	// **و«محادثاتي» عند الكابتن فيها التوصيلةُ باسم المتجر.**
	threads, err := f.srv.comms.Threads(context.Background(), f.driver)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range threads {
		if x.OrderID == f.order {
			found = true
			if x.Peer != "متجر اختبار السائق" || !x.Open {
				t.Errorf("**صفُّ الكابتن غلط**: %+v", x)
			}
		}
	}
	if !found {
		t.Errorf("**التوصيلةُ ليست في «محادثاتي» عند الكابتن**")
	}
}

// TestStoreCaptainChat_StrangerDenied **لا متجرٌ آخرُ ولا كابتنٌ آخر** — ٤٠٤ لا ٤٠٣.
func TestStoreCaptainChat_StrangerDenied(t *testing.T) {
	f := newStoreChatFixture(t, "on_the_way")
	other := testdb.NewUser(t, f.pool, "merchant")
	for _, c := range []struct{ uid, role string }{{other, "merchant"}, {f.drivers[1], "driver"}} {
		if code, _ := f.read(t, c.uid, c.role); code != http.StatusNotFound {
			t.Errorf("**غريبٌ (%s) قرأ**: %d", c.role, code)
		}
		if code, _ := f.send(t, c.uid, c.role, "مرحبا"); code != http.StatusNotFound {
			t.Errorf("**غريبٌ (%s) كتب**: %d", c.role, code)
		}
	}
}

// TestStoreCaptainChat_ClosesAtDelivery **يُغلق بالتسليم** — المتجرُ يقرأ ولا يكتب، والكابتنُ لا يقرأ.
func TestStoreCaptainChat_ClosesAtDelivery(t *testing.T) {
	f := newStoreChatFixture(t, "at_dropoff")
	if code, body := f.send(t, f.owner, "merchant", "سلّم بإيدو"); code != http.StatusCreated {
		t.Fatalf("%d %s", code, body)
	}
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET status = 'delivered', delivered_at = now() WHERE id = $1`, f.order); err != nil {
		t.Fatal(err)
	}
	code, h := f.read(t, f.owner, "merchant")
	if code != http.StatusOK || h.Open || len(h.Messages) != 1 {
		t.Errorf("**المتجرُ لا يقرأ حديثَه بعد التسليم أو بقي مفتوحاً**: %d %+v", code, h)
	}
	if code, _ := f.send(t, f.owner, "merchant", "شكراً"); code != http.StatusConflict {
		t.Errorf("**المتجرُ كتب بعد التسليم**: %d", code)
	}
	if code, _ := f.read(t, f.driver, "driver"); code != http.StatusForbidden {
		t.Errorf("**الكابتنُ قرأ توصيلةً انتهت**: %d", code)
	}
}

// TestStoreCaptainChat_NotBeforeAssignment **لا كابتنَ بعد ⇒ لا حديث.**
func TestStoreCaptainChat_NotBeforeAssignment(t *testing.T) {
	f := newStoreChatFixture(t, "dispatching")
	if _, err := f.pool.Exec(context.Background(),
		`UPDATE orders SET driver_id = NULL WHERE id = $1`, f.order); err != nil {
		t.Fatal(err)
	}
	if code, _ := f.send(t, f.owner, "merchant", "وينك"); code != http.StatusConflict {
		t.Errorf("**المتجرُ كتب ولا كابتن**: %d", code)
	}
}

// TestStoreCaptainChat_ContactMasked **رقمُ الجوال يُخفى في رسالة المتجر** — ويُوسَم ويُنبَّه المكتب.
func TestStoreCaptainChat_ContactMasked(t *testing.T) {
	f := newStoreChatFixture(t, "assigned")
	ops := testdb.NewUser(t, f.pool, "operations")
	if code, body := f.send(t, f.owner, "merchant", "رقمي 0991234567 كلمني واتس"); code != http.StatusCreated {
		t.Fatalf("%d %s", code, body)
	}
	var body, word string
	var leak bool
	if err := f.pool.QueryRow(context.Background(), `
		SELECT body, COALESCE(flag_word, ''), contact_leak FROM order_messages
		 WHERE order_id = $1 ORDER BY created_at DESC LIMIT 1`, f.order).Scan(&body, &word, &leak); err != nil {
		t.Fatal(err)
	}
	if body != "رقمي *** كلمني ***" || !leak || !strings.Contains(word, "0991234567") {
		t.Errorf("**رسالةُ المتجر لم تُخفَ وتُوسَم**: %q %q %v", body, word, leak)
	}
	var alert string
	if err := f.pool.QueryRow(context.Background(), `
		SELECT body FROM notifications WHERE user_id = $1 AND title = $2 LIMIT 1`,
		ops, notifTitles.contactLeak).Scan(&alert); err != nil {
		t.Fatalf("**المكتبُ لم يُنبَّه**: %v", err)
	}
	if !strings.Contains(alert, notifTitles.contactLeakByStore) {
		t.Errorf("**التنبيهُ لا يقول «من المتجر»**: %q", alert)
	}
}

// TestStoreCaptainChat_UnreadOnDelivery **ما ينتظر المتجرَ يُقرأ مع التوصيلة** — ويسقط بعد القراءة.
func TestStoreCaptainChat_UnreadOnDelivery(t *testing.T) {
	f := newStoreChatFixture(t, "on_the_way")
	if code, body := f.send(t, f.driver, "driver", "بالطريق"); code != http.StatusCreated {
		t.Fatalf("%d %s", code, body)
	}
	got := func() (float64, float64) {
		t.Helper()
		w := f.call(f.srv.handleMerchantDelivery, http.MethodGet, "/x", f.order, f.owner, []string{"merchant"}, "")
		if w.Code != http.StatusOK {
			t.Fatalf("التوصيلة: %d %s", w.Code, w.Body.String())
		}
		var env struct {
			Data map[string]any `json:"data"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
			t.Fatal(err)
		}
		u, _ := env.Data["chat_unread"].(float64)
		n, _ := env.Data["chat_lines"].(float64)
		return u, n
	}
	if u, n := got(); u != 1 || n != 1 {
		t.Errorf("**العدُّ قبل القراءة** = %v/%v — والمتوقّع ١/١", u, n)
	}
	f.read(t, f.owner, "merchant")
	if u, n := got(); u != 0 || n != 1 {
		t.Errorf("**العدُّ بعد القراءة** = %v/%v — والمتوقّع ٠/١", u, n)
	}
}

// TestStoreCaptainChat_CustomerOrdersUnchanged **طلبُ الزبون كما كان** — وصاحبُ متجرٍ ليس طرفاً فيه.
func TestStoreCaptainChat_CustomerOrdersUnchanged(t *testing.T) {
	f := newStoreChatFixture(t, "assigned")
	orderID := f.problemOrderAt(t, "on_the_way", f.driver)
	var customer string
	if err := f.pool.QueryRow(context.Background(),
		`SELECT customer_id::text FROM orders WHERE id = $1`, orderID).Scan(&customer); err != nil {
		t.Fatal(err)
	}
	p, err := f.srv.comms.Permit(context.Background(), orderID, customer)
	if err != nil || p.Me != comms.RoleCustomer || p.PeerName != comms.CustomerDriverLabel || !p.Open ||
		p.PeerRole != comms.RoleDriver {
		t.Errorf("**الزبونُ تبدّل**: %+v %v", p, err)
	}
	p, err = f.srv.comms.Permit(context.Background(), orderID, f.driver)
	if err != nil || p.Me != comms.RoleDriver || p.PeerID != customer || p.PeerRole != comms.RoleCustomer {
		t.Errorf("**السائقُ تبدّل**: %+v %v", p, err)
	}
	if _, err := f.srv.comms.Permit(context.Background(), orderID, f.owner); err != comms.ErrNotParty {
		t.Errorf("**صاحبُ المتجر صار طرفاً في طلب زبون**: %v", err)
	}
}
