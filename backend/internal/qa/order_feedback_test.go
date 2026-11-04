package qa

// ══════════════════════════════════════════════════════════════════════
// **رأيُ الزبون بعد التسليم بالواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٤، هجرة ٠٤٩٨)
// ══════════════════════════════════════════════════════════════════════

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// recNotifier **بوتٌ يسجّل ما أُرسل** — مع جاهزيّةٍ يُتحكَّم بها.
type recNotifier struct {
	mu    sync.Mutex
	ready bool
	sent  []recMsg
}

type recMsg struct{ phone, text string }

func (n *recNotifier) Ready() bool { return n.ready }
func (n *recNotifier) SendText(_ context.Context, phone, text string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, recMsg{phone, text})
	return nil
}

// to **ما وصل رقماً بعينه.**
func (n *recNotifier) to(phone string) []recMsg {
	n.mu.Lock()
	defer n.mu.Unlock()
	var out []recMsg
	for _, m := range n.sent {
		if strings.HasSuffix(m.phone, strings.TrimPrefix(phone, "+")) || m.phone == phone {
			out = append(out, m)
		}
	}
	return out
}

// seedOldLink **رابطٌ قديمٌ أُرسل قبل قرار «بلا رابط»** — الصفحةُ العامّةُ تبقى له حتّى ينتهي.
func seedOldLink(t *testing.T, h *Harness, orderID, token string) {
	t.Helper()
	sum := sha256.Sum256([]byte(token))
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_feedback_requests (order_id, token_hash, sent_at, expires_at)
		VALUES ($1::uuid, $2, now(), now() + interval '7 days')`, orderID, hex.EncodeToString(sum[:])); err != nil {
		t.Fatal(err)
	}
}

// feedbackFixture **طلبٌ مُسلَّمٌ قبل `ago` دقيقة**، وكلُّ ما سواه مسدودٌ عن الكانس
// (القاعدةُ مشتركةٌ بين الفحوص والكانسُ عامّ).
func feedbackFixture(t *testing.T, h *Harness, agoMin int) (orderID string, cust *User, number int64) {
	t.Helper()
	orderID, _ = activeOrderFor(t, h)
	var custID string
	if err := h.Pool.QueryRow(ctxBG(), `
		UPDATE orders SET status = 'delivered',
		       delivered_at = now() - make_interval(mins => $2)
		 WHERE id = $1::uuid RETURNING customer_id::text, number`, orderID, agoMin).Scan(&custID, &number); err != nil {
		t.Fatal(err)
	}
	var phone string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT COALESCE(NULLIF(whatsapp_phone, ''), phone::text) FROM users WHERE id = $1::uuid`, custID).Scan(&phone); err != nil {
		t.Fatal(err)
	}
	cust = &User{ID: custID, Phone: phone}
	return
}

// blockOthers **يسدّ كلَّ طلبٍ مُسلَّمٍ غيرِ المذكورين عن الكانس.**
func blockOthers(t *testing.T, h *Harness, keep ...string) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_feedback_requests (order_id, token_hash, expires_at)
		SELECT id, 'blocked-' || id::text, now() FROM orders
		 WHERE status = 'delivered' AND NOT (id::text = ANY($1))
		ON CONFLICT DO NOTHING`, keep); err != nil {
		t.Fatal(err)
	}
}

func armFeedback(t *testing.T, h *Harness, ready bool) *recNotifier {
	t.Helper()
	n := &recNotifier{ready: ready}
	h.API.SetMerchantNotifier(n)
	t.Cleanup(func() { h.API.SetMerchantNotifier(nil) })
	return n
}

// تُرسَل مرّةً واحدة، وتحمل رقمَ الطلب ورابطَ /rate/.
func TestOrderFeedback_SweepSendsOnce(t *testing.T) {
	h := New(t)
	oid, cust, number := feedbackFixture(t, h, 15)
	blockOthers(t, h, oid)
	n := armFeedback(t, h, true)

	if got := h.API.SweepOrderFeedback(ctxBG()); got != 1 {
		t.Fatalf("الجولةُ الأولى أرسلت %d — والمنتظَر 1", got)
	}
	msgs := n.to(cust.Phone)
	if len(msgs) != 1 {
		t.Fatalf("وصل الزبونَ %d رسالة — والمنتظَر 1", len(msgs))
	}
	want := "وصلك طلبك رقم " + strconv.FormatInt(number, 10) + "؟\n" +
		"كيف كانت الخدمة؟ رأيك بيفرق معنا 🌷\n" +
		"مع تحيات إدارة تطبيق رحّال غو"
	if msgs[0].text != want {
		t.Fatalf("نصُّ الرسالة: %q — والمنتظَر %q", msgs[0].text, want)
	}
	if strings.Contains(msgs[0].text, "http") || strings.Contains(msgs[0].text, "/rate/") {
		t.Fatalf("في الرسالة رابط: %q", msgs[0].text)
	}
	if got := h.API.SweepOrderFeedback(ctxBG()); got != 0 {
		t.Fatalf("الجولةُ الثانية أرسلت %d — مرّةٌ لكلّ طلب", got)
	}
	var sentAt *string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT sent_at::text FROM order_feedback_requests WHERE order_id = $1::uuid`, oid).Scan(&sentAt); err != nil || sentAt == nil {
		t.Fatalf("sent_at لم يُسجَّل: %v / %v", sentAt, err)
	}
}

// لا يُسأل: من قيَّم، ولا غيرُ المُسلَّم، ولا المبكّر (<١٠ د)، ولا القديم (>يوم)، ولا والبوتُ غيرُ جاهز.
func TestOrderFeedback_SweepSkips(t *testing.T) {
	h := New(t)
	rated, ratedCust, _ := feedbackFixture(t, h, 15)
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_ratings (order_id, customer_id, platform_stars) VALUES ($1::uuid, $2::uuid, 5)`,
		rated, ratedCust.ID); err != nil {
		t.Fatal(err)
	}
	cancelled, cancelledCust, _ := feedbackFixture(t, h, 15)
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE orders SET status = 'cancelled' WHERE id = $1::uuid`, cancelled); err != nil {
		t.Fatal(err)
	}
	_, earlyCust, _ := feedbackFixture(t, h, 3)
	_, oldCust, _ := feedbackFixture(t, h, 3*24*60)
	notReady, notReadyCust, _ := feedbackFixture(t, h, 15)
	blockOthers(t, h, rated, cancelled, notReady)

	n := armFeedback(t, h, false)
	if got := h.API.SweepOrderFeedback(ctxBG()); got != 0 {
		t.Fatalf("البوتُ غيرُ جاهزٍ وأُرسل %d", got)
	}
	var rows int
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM order_feedback_requests WHERE order_id = $1::uuid`, notReady).Scan(&rows); err != nil || rows != 0 {
		t.Fatalf("حُجز صفٌّ والبوتُ غيرُ جاهز: %d / %v", rows, err)
	}

	n.ready = true
	if got := h.API.SweepOrderFeedback(ctxBG()); got != 1 {
		t.Fatalf("بعد عودة البوت أُرسل %d — والمنتظَر 1 (الجاهز وحدَه)", got)
	}
	for name, c := range map[string]*User{"rated": ratedCust, "cancelled": cancelledCust, "early": earlyCust, "old": oldCust} {
		if m := n.to(c.Phone); len(m) != 0 {
			t.Errorf("وصلت رسالةٌ لحالة %s: %q", name, m[0].text)
		}
	}
	if m := n.to(notReadyCust.Phone); len(m) != 1 {
		t.Fatalf("الطلبُ المؤجَّل لم يصل بعد عودة البوت: %d", len(m))
	}
}

// الرابط: يفتح، يكتب order_ratings بمسار التطبيق مع الكلمة، ويُغلق بعد أوّل إرسال، ويظهر في اللوحة.
func TestOrderFeedback_SubmitWritesRatingOnceAndShowsInAdmin(t *testing.T) {
	h := New(t)
	oid, cust, _ := feedbackFixture(t, h, 20)
	_ = cust
	seedOldLink(t, h, oid, "old-link-token-aaaaaaaaaaaa")
	path := "/api/v1/public/feedback-requests/old-link-token-aaaaaaaaaaaa"

	info := h.GET(path, "")
	if info.Code != http.StatusOK || info.JSON()["status"] != "pending" || info.JSON()["has_driver"] != true {
		t.Fatalf("صفحةُ الرأي: %s", info)
	}
	// نجومٌ خاطئة لا تحرق الرابط.
	if r := h.POST(path, "", map[string]any{"service_stars": 0}); r.Code != http.StatusBadRequest {
		t.Fatalf("نجومٌ صفر: %s", r)
	}
	if r := h.POST(path, "", map[string]any{"service_stars": 4, "driver_stars": 5, "feedback": "  التطبيق ممتاز  "}); r.Code != http.StatusOK {
		t.Fatalf("الإرسال: %s", r)
	}
	var ps int
	var ds *int
	var fb string
	if err := h.Pool.QueryRow(ctxBG(), `
		SELECT platform_stars, driver_stars, app_feedback FROM order_ratings WHERE order_id = $1::uuid`, oid).
		Scan(&ps, &ds, &fb); err != nil {
		t.Fatalf("لا صفَّ في order_ratings: %v", err)
	}
	if ps != 4 || ds == nil || *ds != 5 || fb != "التطبيق ممتاز" {
		t.Fatalf("القيمُ المكتوبة: %d / %v / %q", ps, ds, fb)
	}
	// الثانية مرفوضة، والأولى لم تتغيّر.
	if r := h.POST(path, "", map[string]any{"service_stars": 1}); r.Code != http.StatusGone {
		t.Fatalf("الإرسالُ الثاني: %s", r)
	}
	if err := h.Pool.QueryRow(ctxBG(), `SELECT platform_stars FROM order_ratings WHERE order_id = $1::uuid`, oid).Scan(&ps); err != nil || ps != 4 {
		t.Fatalf("تبدّل التقييمُ بعد الإغلاق: %d / %v", ps, err)
	}
	if info := h.GET(path, ""); info.JSON()["status"] != "answered" {
		t.Fatalf("الحالُ بعد الإرسال: %s", info)
	}

	// وتراه اللوحة.
	admin := h.NewUser("admin")
	r := h.GET("/api/v1/admin/ratings/list", admin.Token)
	if r.Code != http.StatusOK {
		t.Fatalf("قائمةُ التقييمات: %s", r)
	}
	found := false
	list, _ := r.JSON()["ratings"].([]any)
	for _, x := range list {
		if row, _ := x.(map[string]any); row["order_id"] == oid {
			found = row["app_feedback"] == "التطبيق ممتاز"
		}
	}
	if !found {
		t.Fatal("كلمةُ الزبون لا تظهر في قائمة التقييمات")
	}
}

// منتهي الصلاحية ⇒ 410، ورمزٌ مجهول ⇒ 404، ومن قيَّم في التطبيق أولاً ⇒ 410 بلا كتابةٍ ثانية.
func TestOrderFeedback_ExpiredAndAlreadyRated(t *testing.T) {
	h := New(t)
	oid, cust, _ := feedbackFixture(t, h, 20)
	oid2, cust2, _ := feedbackFixture(t, h, 20)
	seedOldLink(t, h, oid, "old-link-token-bbbbbbbbbbbb")
	seedOldLink(t, h, oid2, "old-link-token-cccccccccccc")
	tok := func(c *User) string {
		if c == cust {
			return "old-link-token-bbbbbbbbbbbb"
		}
		return "old-link-token-cccccccccccc"
	}

	if _, err := h.Pool.Exec(ctxBG(),
		`UPDATE order_feedback_requests SET expires_at = now() - interval '1 minute' WHERE order_id = $1::uuid`, oid); err != nil {
		t.Fatal(err)
	}
	if r := h.POST("/api/v1/public/feedback-requests/"+tok(cust), "", map[string]any{"service_stars": 5}); r.Code != http.StatusGone {
		t.Fatalf("منتهي الصلاحية: %s", r)
	}
	if r := h.POST("/api/v1/public/feedback-requests/"+strings.Repeat("x", 24), "", map[string]any{"service_stars": 5}); r.Code != http.StatusNotFound {
		t.Fatalf("رمزٌ مجهول: %s", r)
	}

	// قيَّم من التطبيق قبل أن يفتح الرابط.
	if _, err := h.Pool.Exec(ctxBG(), `
		INSERT INTO order_ratings (order_id, customer_id, platform_stars) VALUES ($1::uuid, $2::uuid, 2)`, oid2, cust2.ID); err != nil {
		t.Fatal(err)
	}
	if r := h.POST("/api/v1/public/feedback-requests/"+tok(cust2), "", map[string]any{"service_stars": 5}); r.Code != http.StatusGone {
		t.Fatalf("قُيِّم من التطبيق: %s", r)
	}
	var ps int
	if err := h.Pool.QueryRow(ctxBG(), `SELECT platform_stars FROM order_ratings WHERE order_id = $1::uuid`, oid2).Scan(&ps); err != nil || ps != 2 {
		t.Fatalf("تقييمُ التطبيق تبدّل: %d / %v", ps, err)
	}
}
