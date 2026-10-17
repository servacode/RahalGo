package qa

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/servacode/rahalgo/backend/internal/push"
	"github.com/servacode/rahalgo/backend/internal/server"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعاراتُ المتصفّح من الباب إلى المزوّد** (قرارُ المالك ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// اشتراكٌ عبر الباب الحقيقيّ ⇒ طلبٌ حقيقيّ ⇒ جولةُ نقل ⇒ **مزوّدٌ مزيّفٌ
// يستقبل نداءً بترويساته.** ثمّ يردّ المزوّدُ ٤١٠ فيُحذف الاشتراك.

// fakePushService **مزوّدُ دفعٍ مزيّف** — يسجّل ما يصله، ويردّ ما يُبرمَج.
type fakePushService struct {
	mu   sync.Mutex
	hits map[string][]http.Header // المسارُ ⇒ ترويساتُ كلِّ نداء
	code int
	srv  *httptest.Server
}

func newFakePushService(t *testing.T) *fakePushService {
	f := &fakePushService{hits: map[string][]http.Header{}, code: http.StatusCreated}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.hits[r.URL.Path] = append(f.hits[r.URL.Path], r.Header.Clone())
		code := f.code
		f.mu.Unlock()
		w.WriteHeader(code)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakePushService) count(path string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.hits[path])
}

func (f *fakePushService) last(path string) http.Header {
	f.mu.Lock()
	defer f.mu.Unlock()
	h := f.hits[path]
	if len(h) == 0 {
		return nil
	}
	return h[len(h)-1]
}

func (f *fakePushService) respond(code int) {
	f.mu.Lock()
	f.code = code
	f.mu.Unlock()
}

// webPushHarness **مِسنَدٌ بناقل متصفّحٍ يقبل المزوّدَ المحلّيّ.**
func webPushHarness(t *testing.T) *Harness {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	priv, _ := key.Bytes()
	pub, _ := key.PublicKey.Bytes()
	wp, err := push.NewWebPush(push.WebPushConfig{
		PublicKey:  base64.RawURLEncoding.EncodeToString(pub),
		PrivateKey: base64.RawURLEncoding.EncodeToString(priv),
		Subject:    "mailto:qa@example.com",
		AllowHost:  func(string, string) bool { return true },
	})
	if err != nil {
		t.Fatal(err)
	}
	return NewWith(t, server.WithWebPush(wp))
}

// browserKeys **مفاتيحُ متصفّحٍ حقيقيّة** — فالتشفيرُ يجري عليها فعلاً.
func browserKeys(t *testing.T) map[string]any {
	t.Helper()
	priv, _ := ecdh.P256().GenerateKey(rand.Reader)
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	return map[string]any{
		"p256dh": base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()),
		"auth":   base64.RawURLEncoding.EncodeToString(auth),
	}
}

// merchantOwnerToken **توكنُ صاحب المتجر بجلسةٍ دائمة.**
func merchantOwnerToken(t *testing.T, h *Harness, merchantID string) (string, string) {
	t.Helper()
	var owner, sid string
	if err := h.Pool.QueryRow(ctxBG(),
		`SELECT owner_user_id::text FROM merchants WHERE id = $1::uuid`, merchantID).Scan(&owner); err != nil {
		t.Fatal(err)
	}
	if err := h.Pool.QueryRow(ctxBG(), `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at, client)
		VALUES ($1::uuid, $2, now() + interval '30 days', 'web')
		RETURNING session_id::text`, owner, "qa-wp-"+owner).Scan(&sid); err != nil {
		t.Fatal(err)
	}
	return owner, h.TokenWithSession(owner, sid, "merchant")
}

func subCount(t *testing.T, h *Harness, endpoint string) int {
	t.Helper()
	var n int
	_ = h.Pool.QueryRow(ctxBG(),
		`SELECT count(*) FROM web_push_subscriptions WHERE endpoint = $1`, endpoint).Scan(&n)
	return n
}

// deliverUntil **جولاتُ نقلٍ حتّى يتحقّق الشرط** — والنبضةُ قد تسبق.
func deliverUntil(h *Harness, cond func() bool) bool {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		h.API.DeliverPushOnce(ctxBG())
		if cond() {
			return true
		}
		time.Sleep(50 * time.Millisecond)
	}
	return cond()
}

// TestWebPush_NewOrderReachesMerchantBrowser **اشتراك ⇒ طلب ⇒ نداءٌ عاجلٌ إلى المزوّد.**
func TestWebPush_NewOrderReachesMerchantBrowser(t *testing.T) {
	h := webPushHarness(t)
	fake := newFakePushService(t)

	if got := h.GET("/api/v1/public/webpush-key", ""); got.Code != 200 ||
		got.JSON()["enabled"] != true || got.JSON()["public_key"] == "" {
		t.Fatalf("**المفتاحُ العامّ**: %s", got)
	}

	item := h.NewItem(5000)
	_, merchTok := merchantOwnerToken(t, h, item.MerchantID)
	cust := h.Customer()

	merchEP := fake.srv.URL + "/m/" + uniq("ep")
	custEP := fake.srv.URL + "/c/" + uniq("ep")
	if got := h.POST("/api/v1/me/webpush", merchTok, map[string]any{
		"endpoint": merchEP, "keys": browserKeys(t), "app": "merchant"}); got.Code != 200 {
		t.Fatalf("اشتراكُ المتجر: %s", got)
	}
	// **والزبونُ لا يشترك بلوحة متجر** — لا دورَ له.
	if got := h.POST("/api/v1/me/webpush", cust.Token, map[string]any{
		"endpoint": custEP, "keys": browserKeys(t), "app": "merchant"}); got.Code != 403 {
		t.Fatalf("**زبونٌ اشترك بلوحة متجر**: %s", got)
	}
	if got := h.POST("/api/v1/me/webpush", cust.Token, map[string]any{
		"endpoint": custEP, "keys": browserKeys(t), "app": "customer"}); got.Code != 200 {
		t.Fatalf("اشتراكُ الزبون: %s", got)
	}
	// **ومفاتيحُ معطوبةٌ تُرفض قبل الحفظ.**
	if got := h.POST("/api/v1/me/webpush", cust.Token, map[string]any{
		"endpoint": custEP + "x", "keys": map[string]any{"p256dh": "AAAA", "auth": "BBBB"},
		"app": "customer"}); got.Code != 400 {
		t.Fatalf("**قُبلت مفاتيحُ معطوبة**: %s", got)
	}

	made := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made.Code >= 300 {
		t.Fatalf("الطلب: %s", made)
	}
	mPath := strings.TrimPrefix(merchEP, fake.srv.URL)
	if !deliverUntil(h, func() bool { return fake.count(mPath) > 0 }) {
		t.Fatalf("**لم يصل المزوّدَ نداءٌ لمتصفّح المتجر**")
	}
	hd := fake.last(mPath)
	if hd.Get("Urgency") != "high" || hd.Get("Content-Encoding") != "aes128gcm" ||
		!strings.HasPrefix(hd.Get("Authorization"), "vapid t=") || hd.Get("TTL") == "" {
		t.Fatalf("**ترويساتُ النداء**: %v", hd)
	}
	accepted := func() bool {
		var n int
		_ = h.Pool.QueryRow(ctxBG(), `
			SELECT count(*) FROM notification_deliveries
			 WHERE token = $1 AND platform = 'web' AND state = 'accepted'`, merchEP).Scan(&n)
		return n > 0
	}
	if !deliverUntil(h, accepted) {
		t.Fatalf("**صفُّ النقل لم يُقيَّد مقبولاً**")
	}
	t.Logf("المتجر: نداءٌ عاجلٌ وصل المزوّد · TTL=%s", hd.Get("TTL"))

	// **وخبرُ «طلبٌ جديدٌ للمتجر» لا يرنّ في متصفّح الزبون** — لا صفَّ نقلٍ له.
	var wrong int
	_ = h.Pool.QueryRow(ctxBG(), `
		SELECT count(*) FROM notification_deliveries d
		  JOIN notifications n ON n.id = d.notification_id
		 WHERE d.token = $1 AND 'merchant' = ANY(n.push_apps)`, custEP).Scan(&wrong)
	if wrong != 0 {
		t.Fatalf("**خبرُ المتجر فُرّع لمتصفّح الزبون**: %d", wrong)
	}

	// ── ٤١٠ ⇒ يُحذف الاشتراك ─────────────────────────────────────
	fake.respond(http.StatusGone)
	before := fake.count(mPath)
	made2 := h.POSTKey("/api/v1/orders", cust.Token, uniq("k"), orderBody(item, 1))
	if made2.Code >= 300 {
		t.Fatalf("الطلب الثاني: %s", made2)
	}
	if !deliverUntil(h, func() bool { return subCount(t, h, merchEP) == 0 }) {
		t.Fatalf("**ردّ المزوّدُ ٤١٠ وبقي الاشتراك** (نداءات: %d ⇒ %d)", before, fake.count(mPath))
	}
	t.Logf("٤١٠ ⇒ حُذف الاشتراك")

	// ── والإلغاءُ مقيَّدٌ بصاحبه ──────────────────────────────────
	if got := h.Call("DELETE", "/api/v1/me/webpush", merchTok,
		map[string]any{"endpoint": custEP}, nil); got.Code != 200 {
		t.Fatalf("الإلغاء: %s", got)
	}
	if subCount(t, h, custEP) != 1 {
		t.Fatalf("**ألغى المتجرُ اشتراكَ غيره**")
	}
	if got := h.Call("DELETE", "/api/v1/me/webpush", cust.Token,
		map[string]any{"endpoint": custEP}, nil); got.Code != 200 || subCount(t, h, custEP) != 0 {
		t.Fatalf("**إلغاءُ صاحبه لم يحذف**: %s", got)
	}
}

// TestWebPush_DisabledWithoutKeys **بلا مفاتيح: المفتاحُ فارغ والاشتراكُ مرفوض — وما عداه يعمل.**
func TestWebPush_DisabledWithoutKeys(t *testing.T) {
	t.Setenv("WEBPUSH_VAPID_PUBLIC_KEY", "")
	t.Setenv("WEBPUSH_VAPID_PRIVATE_KEY", "")
	h := New(t)
	got := h.GET("/api/v1/public/webpush-key", "")
	if got.Code != 200 || got.JSON()["enabled"] != false {
		t.Fatalf("**المفتاحُ بلا إعداد**: %s", got)
	}
	cust := h.Customer()
	res := h.POST("/api/v1/me/webpush", cust.Token, map[string]any{
		"endpoint": "https://fcm.googleapis.com/fcm/send/x", "keys": browserKeys(t), "app": "customer"})
	if res.Code != 503 || res.Err() != "webpush_disabled" {
		t.Fatalf("**اشتراكٌ بلا مفاتيح**: %s", res)
	}
}

// TestWebPush_LogoutRemovesSubscription **الخروجُ يُنهي اشتراكَ المتصفّح كرمز الهاتف** (`D12`).
func TestWebPush_LogoutRemovesSubscription(t *testing.T) {
	h := webPushHarness(t)
	fake := newFakePushService(t)
	_, drv := activeOrderFor(t, h)
	access, refresh := loginPair(t, h, drv.ID)
	ep := fake.srv.URL + "/l/" + uniq("ep")
	if got := h.POST("/api/v1/me/webpush", access, map[string]any{
		"endpoint": ep, "keys": browserKeys(t), "app": "customer"}); got.Code != 200 {
		t.Fatalf("الاشتراك: %s", got)
	}
	if subCount(t, h, ep) != 1 {
		t.Fatal("لم يُحفظ الاشتراك")
	}
	if code := logout(t, h, refresh, ""); code >= 400 {
		t.Fatalf("الخروج: %d", code)
	}
	if subCount(t, h, ep) != 0 {
		t.Fatalf("**بقي اشتراكُ المتصفّح بعد الخروج**")
	}
}
