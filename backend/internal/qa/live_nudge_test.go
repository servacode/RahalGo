package qa

// ══════════════════════════════════════════════════════════════════════
// **«لازم يوصلنا موقع مباشر» — التذكيرُ التلقائيّ وردُّ الواتساب** (قرارُ المالك ٢٠٢٦-١١-١٨)
// ══════════════════════════════════════════════════════════════════════
//
// طلبٌ بالطريق بلا موقع: بعد دقيقة إشعارٌ للزبون وطلبُ موقعٍ يتجدّد (ولا ينحسب من مرّات الكابتن)،
// وبعد دقيقتين رسالةُ واتساب واحدة بلا رابط. **وردُّ الزبون بموقعه من الواتساب يصير موقعاً مباشراً لطلبه.**

import (
	"context"
	"strings"
	"sync"
	"testing"
)

// liveNudgeBot **بوتٌ مقيس** — يحفظ ما انبعت.
type liveNudgeBot struct {
	mu   sync.Mutex
	sent []string
}

func (b *liveNudgeBot) Ready() bool { return true }
func (b *liveNudgeBot) SendText(_ context.Context, phone, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.sent = append(b.sent, phone+"|"+text)
	return nil
}

func (b *liveNudgeBot) count() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.sent)
}

// liveNudgeOrder **طلبٌ عاديٌّ بالطريق** — و«بالطريق» قبل `ago` ثانية.
func liveNudgeOrder(t *testing.T, h *Harness) (oid string, cust, drv *User) {
	t.Helper()
	cust = h.Customer()
	item := h.NewItem(1000)
	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 400 {
		t.Fatalf("الطلب: %s", made)
	}
	oid, _ = made.JSON()["id"].(string)
	drv = h.driverOf(oid)
	for _, to := range []string{"at_pickup", "picked_up", "on_the_way"} {
		if r := h.POST("/api/v1/driver/orders/"+oid+"/transition", drv.Token, map[string]any{"to": to}); r.Code >= 400 {
			t.Fatalf("الانتقال %s: %s", to, r)
		}
	}
	return oid, cust, drv
}

func onTheWayAgo(t *testing.T, h *Harness, oid string, sec int) {
	t.Helper()
	if _, err := h.Pool.Exec(ctxBG(), `UPDATE order_events SET created_at = now() - make_interval(secs => $2)
		WHERE order_id = $1 AND to_status = 'on_the_way'`, oid, sec); err != nil {
		t.Fatal(err)
	}
}

// TestLIVENUDGE01_PushThenWhatsAppOnce **إشعارٌ بعد دقيقة، وواتسابٌ بعد دقيقتين، ومرّةً وحدة لكلٍّ.**
func TestLIVENUDGE01_PushThenWhatsAppOnce(t *testing.T) {
	h := New(t)
	bot := &liveNudgeBot{}
	h.API.SetMerchantNotifier(bot)
	oid, cust, _ := liveNudgeOrder(t, h)

	// **قبل الدقيقة لا شي.**
	onTheWayAgo(t, h, oid, 20)
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	var pushAt, waAt, reqAt *string
	var asks int
	read := func() {
		_ = h.Pool.QueryRow(ctxBG(), `SELECT live_nudge_push_at::text, live_nudge_wa_at::text, live_requested_at::text, live_request_count
			FROM orders WHERE id = $1`, oid).Scan(&pushAt, &waAt, &reqAt, &asks)
	}
	read()
	if pushAt != nil || waAt != nil {
		t.Fatalf("**ذكّر قبل وقته**: %v %v", pushAt, waAt)
	}

	// **بعد دقيقة: إشعارٌ وطلبُ موقعٍ — ولا ينحسب من مرّات الكابتن.**
	onTheWayAgo(t, h, oid, 70)
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	read()
	if pushAt == nil || reqAt == nil || waAt != nil || asks != 0 {
		t.Fatalf("بعد دقيقة: إشعار %v · طلب %v · واتساب %v · مرّات %d", pushAt, reqAt, waAt, asks)
	}
	var told int
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity_id = $1 AND user_id = $2::uuid
		AND title LIKE '%شارك موقعك المباشر%'`, oid, cust.ID).Scan(&told)
	if told != 1 {
		t.Fatalf("**إشعار الزبون** %d", told)
	}
	if bot.count() != 0 {
		t.Fatalf("**واتساب قبل الدقيقتين**")
	}

	// **بعد دقيقتين: واتسابٌ واحدٌ بلا رابط.**
	onTheWayAgo(t, h, oid, 130)
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	if bot.count() != 1 {
		t.Fatalf("رسائل الواتساب %d والمنتظَر 1", bot.count())
	}
	msg := bot.sent[0]
	if !strings.HasPrefix(msg, cust.Phone+"|") || strings.Contains(msg, "http") || !strings.Contains(msg, "شارك الموقع المباشر") {
		t.Fatalf("الرسالة: %q", msg)
	}
	_ = h.Pool.QueryRow(ctxBG(), `SELECT count(*) FROM notifications WHERE entity_id = $1 AND user_id = $2::uuid
		AND title LIKE '%شارك موقعك المباشر%'`, oid, cust.ID).Scan(&told)
	if told != 1 {
		t.Fatalf("**الإشعار تكرّر** %d", told)
	}

	// **والزبونُ يردّ بموقعه المباشر من الواتساب ⇒ موقعٌ لطلبه، والجوابُ مرّة.**
	reply := h.API.HandleWALocationKind(ctxBG(), cust.Phone, 35.951, 39.012, 8, true)
	if !strings.Contains(reply, "وصلنا موقعك المباشر") {
		t.Fatalf("الجواب: %q", reply)
	}
	if again := h.API.HandleWALocationKind(ctxBG(), cust.Phone, 35.952, 39.013, 8, true); again != "" {
		t.Fatalf("**جوابٌ لكلّ تحديث**: %q", again)
	}
	var lat float64
	var src *string
	if err := h.Pool.QueryRow(ctxBG(), `SELECT ST_Y(l.point::geometry), o.live_first_source FROM order_live_locations l
		JOIN orders o ON o.id = l.order_id WHERE l.order_id = $1`, oid).Scan(&lat, &src); err != nil {
		t.Fatalf("**ما انحفظ الموقع**: %v", err)
	}
	if lat < 35.9515 || src == nil || *src != "whatsapp" {
		t.Fatalf("الموقع %v · المصدر %v", lat, src)
	}

	// **واللوحةُ تعدّه.**
	st, err := h.Orders.LiveStatsToday(ctxBG())
	if err != nil || st.OnTheWay < 1 || st.FromWhatsApp < 1 || st.NudgedPush < 1 || st.NudgedWA < 1 {
		t.Fatalf("الأرقام: %+v %v", st, err)
	}
}

// TestLIVENUDGE02_NoNudgeWhileSharing **من عم يشارك ما بيتذكّر.**
func TestLIVENUDGE02_NoNudgeWhileSharing(t *testing.T) {
	h := New(t)
	bot := &liveNudgeBot{}
	h.API.SetMerchantNotifier(bot)
	oid, cust, _ := liveNudgeOrder(t, h)
	if r := h.POST("/api/v1/orders/"+oid+"/live-location", cust.Token,
		map[string]any{"lat": 35.95, "lng": 39.01, "accuracy_m": 6}); r.Code >= 400 {
		t.Fatalf("المشاركة: %s", r)
	}
	onTheWayAgo(t, h, oid, 300)
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	var pushAt *string
	var src *string
	_ = h.Pool.QueryRow(ctxBG(), `SELECT live_nudge_push_at::text, live_first_source FROM orders WHERE id = $1`, oid).Scan(&pushAt, &src)
	if pushAt != nil || bot.count() != 0 {
		t.Fatalf("**ذكّر واحد عم يشارك**: %v · %d", pushAt, bot.count())
	}
	if src == nil || *src != "app" {
		t.Fatalf("المصدر %v", src)
	}
	// **والمفتاحُ يطفي.**
	h.Setting("orders.live_nudge_enabled", "false")
	if _, err := h.Pool.Exec(ctxBG(), `DELETE FROM order_live_locations WHERE order_id = $1`, oid); err != nil {
		t.Fatal(err)
	}
	h.Orders.SweepLiveNudgeOnce(ctxBG())
	_ = h.Pool.QueryRow(ctxBG(), `SELECT live_nudge_push_at::text FROM orders WHERE id = $1`, oid).Scan(&pushAt)
	if pushAt != nil {
		t.Fatalf("**ذكّر والمفتاح طافي**")
	}
}

// TestLIVENUDGE03_WhatsAppFromStrangerIgnored **موقعٌ من رقمٍ ما إلو طلب ماشي — ما بيعمل شي.**
func TestLIVENUDGE03_WhatsAppFromStrangerIgnored(t *testing.T) {
	h := New(t)
	if r := h.API.HandleWALocationKind(ctxBG(), "+963999123456", 35.95, 39.01, 5, true); r != "" {
		t.Fatalf("ردّ على غريب: %q", r)
	}
}
