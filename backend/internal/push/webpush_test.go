package push

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعاراتُ المتصفّح — التشفيرُ والتوقيعُ والحمولة** (٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════

func b64(t *testing.T, s string) []byte {
	t.Helper()
	b, err := decodeB64(s)
	if err != nil {
		t.Fatalf("فكّ %q: %v", s, err)
	}
	return b
}

// TestWebPush_RFC8291Vector **مثالُ الملحق A في RFC 8291 بحرفه.**
//
// مفاتيحُ ثابتةٌ وملحٌ ثابت ⇒ ناتجٌ واحدٌ لا غير. **ومن أخطأ بايتاً في
// ترتيب `info` أو في الفاصل خرج ناتجٌ آخر** — والمتصفّحُ يرميه صامتاً.
func TestWebPush_RFC8291Vector(t *testing.T) {
	asPriv, err := ecdh.P256().NewPrivateKey(b64(t, "yfWPiYE-n46HLnH0KqZOF1fJJU3MYrct3AELtAQ-oRw"))
	if err != nil {
		t.Fatal(err)
	}
	uaPublic := b64(t, "BCVxsr7N_eNgVRqvHtD0zTZsEc6-VV-JvLexhqUzORcxaOzi6-AYWXvTBHm4bjyPjs7Vd8pZGH6SRpkNtoIAiw4")
	auth := b64(t, "BTBZMqHH6r4Tts7J_aSIgg")
	salt := b64(t, "DGv6ra1nlYgDCS1FRnbzlw")
	got, err := encryptAES128GCM([]byte("When I grow up, I want to be a watermelon"),
		uaPublic, auth, asPriv, salt)
	if err != nil {
		t.Fatal(err)
	}
	want := "DGv6ra1nlYgDCS1FRnbzlwAAEABBBP4z9KsN6nGRTbVYI_c7VJSPQTBtkgcy27mlmlMoZIIgDll6e3vCYLocInmYWAmS6TlzAC8wEqKK6PBru3jl7A_yl95bQpu6cVPTpK4Mqgkf1CXztLVBSt2Ks3oZwbuwXPXLWyouBWLVWGNWQexSgSxsj_Qulcy4a-fN"
	if g := base64.RawURLEncoding.EncodeToString(got); g != want {
		t.Fatalf("**الناتجُ لا يطابق RFC 8291**\n got=%s\nwant=%s", g, want)
	}
}

// decryptForTest **فكُّ ما يفكّه المتصفّح** — بمفتاح الوكيل الخاصّ.
func decryptForTest(t *testing.T, body []byte, uaPriv *ecdh.PrivateKey, auth []byte) []byte {
	t.Helper()
	salt := body[:16]
	rs := binary.BigEndian.Uint32(body[16:20])
	idlen := int(body[20])
	asPublic := body[21 : 21+idlen]
	ct := body[21+idlen:]
	if rs != webPushRecordSize || idlen != 65 {
		t.Fatalf("ترويسةٌ غريبة: rs=%d idlen=%d", rs, idlen)
	}
	asPub, err := ecdh.P256().NewPublicKey(asPublic)
	if err != nil {
		t.Fatal(err)
	}
	shared, err := uaPriv.ECDH(asPub)
	if err != nil {
		t.Fatal(err)
	}
	uaPublic := uaPriv.PublicKey().Bytes()
	info := append(append([]byte("WebPush: info\x00"), uaPublic...), asPublic...)
	ikm, _ := hkdf.Key(sha256.New, shared, auth, string(info), 32)
	cek, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	nonce, _ := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	block, _ := aes.NewCipher(cek)
	gcm, _ := cipher.NewGCM(block)
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		t.Fatalf("**المتصفّحُ لا يفكّه**: %v", err)
	}
	if plain[len(plain)-1] != 0x02 {
		t.Fatalf("**بايتُ الفاصل الأخير ليس 0x02**")
	}
	return plain[:len(plain)-1]
}

type fakeBrowser struct {
	priv *ecdh.PrivateKey
	auth []byte
}

func newFakeBrowser(t *testing.T) fakeBrowser {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	auth := make([]byte, 16)
	_, _ = rand.Read(auth)
	return fakeBrowser{priv: priv, auth: auth}
}

func (b fakeBrowser) sub(app string) webSub {
	return webSub{App: app, P256dh: b.priv.PublicKey().Bytes(), Auth: b.auth}
}

func testVAPID(t *testing.T, allowAll bool) *WebPush {
	t.Helper()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	priv, _ := key.Bytes()
	pub, _ := key.PublicKey.Bytes()
	cfg := WebPushConfig{
		PublicKey:  base64.RawURLEncoding.EncodeToString(pub),
		PrivateKey: base64.RawURLEncoding.EncodeToString(priv),
		Subject:    "mailto:ops@example.com",
	}
	if allowAll {
		cfg.AllowHost = func(string, string) bool { return true }
	}
	w, err := NewWebPush(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

// TestWebPush_SendHeadersAndPayload **طلبٌ جديدٌ للمتجر — عاجلٌ قصيرُ العمر،
// وحمولةٌ يفكّها المتصفّح وفيها وجهةُ الفتح.**
func TestWebPush_SendHeadersAndPayload(t *testing.T) {
	var got *http.Request
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	wp := testVAPID(t, true)
	br := newFakeBrowser(t)
	msg := Message{
		Title: "طلب جديد", Body: "#12 — صنفان",
		Data:   map[string]string{"kind": "order", "entity": "order", "entity_id": "abc"},
		Urgent: true,
	}
	gone, err := wp.sendOne(context.Background(), srv.URL+"/push/xyz", br.sub(WebAppMerchant), msg)
	if err != nil || gone {
		t.Fatalf("الإرسال: gone=%v err=%v", gone, err)
	}
	h := got.Header
	if h.Get("Urgency") != "high" {
		t.Errorf("**العجلة**: %q — والمنتظَر high", h.Get("Urgency"))
	}
	if h.Get("TTL") != "600" {
		t.Errorf("**العمر**: %q — والمنتظَر 600", h.Get("TTL"))
	}
	if h.Get("Content-Encoding") != "aes128gcm" {
		t.Errorf("الترميز: %q", h.Get("Content-Encoding"))
	}
	if len(h.Get("Topic")) != 32 {
		t.Errorf("مفتاحُ الطيّ: %q", h.Get("Topic"))
	}

	// **والتوقيعُ يُقرأ بالمفتاح العامّ نفسِه، وجمهورُه أصلُ المزوّد.**
	auth := h.Get("Authorization")
	if !strings.HasPrefix(auth, "vapid t=") || !strings.Contains(auth, ", k="+wp.PublicKey()) {
		t.Fatalf("ترويسةُ VAPID: %q", auth)
	}
	tok := strings.TrimSuffix(strings.TrimPrefix(auth, "vapid t="), ", k="+wp.PublicKey())
	parsed, err := jwt.Parse(tok, func(*jwt.Token) (any, error) { return &wp.key.PublicKey, nil },
		jwt.WithValidMethods([]string{"ES256"}))
	if err != nil || !parsed.Valid {
		t.Fatalf("**التوقيعُ لا يُقرأ**: %v", err)
	}
	claims := parsed.Claims.(jwt.MapClaims)
	if claims["aud"] != srv.URL || claims["sub"] != "mailto:ops@example.com" {
		t.Errorf("المطالبات: %v", claims)
	}

	var p WebPayload
	if err := json.Unmarshal(decryptForTest(t, body, br.priv, br.auth), &p); err != nil {
		t.Fatal(err)
	}
	if p.Title != "طلب جديد" || p.URL != "/store/orders" || p.Tag != "order-abc" || !p.Urgent {
		t.Errorf("**الحمولة**: %+v", p)
	}
}

// TestWebPush_GoneIsDead **٤١٠ و٤٠٤ موتُ اشتراك — و٥٠٠ ليست.**
func TestWebPush_GoneIsDead(t *testing.T) {
	for _, tc := range []struct {
		code int
		gone bool
		err  bool
	}{{410, true, false}, {404, true, false}, {500, false, true}, {429, false, true}, {201, false, false}} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(tc.code)
		}))
		wp := testVAPID(t, true)
		gone, err := wp.sendOne(context.Background(), srv.URL, newFakeBrowser(t).sub(WebAppCustomer),
			Message{Title: "x", Data: map[string]string{}})
		srv.Close()
		if gone != tc.gone || (err != nil) != tc.err {
			t.Errorf("ردّ %d: gone=%v err=%v", tc.code, gone, err)
		}
	}
}

// TestWebPush_URLsAndHosts **وجهاتُ الفتح، وقائمةُ المزوّدين.**
func TestWebPush_URLsAndHosts(t *testing.T) {
	cases := map[string]string{
		WebURL(WebAppCustomer, "order", "order", "o1"):     "/orders/o1",
		WebURL(WebAppCustomer, "chat", "order_chat", "o2"): "/orders/o2",
		WebURL(WebAppCustomer, "ticket", "ticket", "t1"):   "/complaints/t1",
		WebURL(WebAppCustomer, "wallet", "wallet", ""):     "/notifications",
		WebURL(WebAppMerchant, "order", "order", "o3"):     "/store/orders",
		WebURL(WebAppMerchant, "account", "merchant", "m"): "/store/notifications",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("وجهة: %q والمنتظَر %q", got, want)
		}
	}
	for host, ok := range map[string]bool{
		"fcm.googleapis.com": true, "web.push.apple.com": true,
		"updates.push.services.mozilla.com": true, "wns2-db5p.notify.windows.com": true,
		"127.0.0.1": false, "evil.example.com": false, "push.apple.com.evil.com": false,
	} {
		if KnownPushHost("https", host) != ok {
			t.Errorf("المزوّد %s: المنتظَر %v", host, ok)
		}
	}
	if KnownPushHost("http", "fcm.googleapis.com") {
		t.Error("**http بلا تشفير قُبل**")
	}
}

// TestWebPush_ConfigRejectsMismatch **مفتاحٌ عامٌّ لا يطابق الخاصّ يُرفض عند الإقلاع.**
func TestWebPush_ConfigRejectsMismatch(t *testing.T) {
	a, b := testVAPID(t, false), testVAPID(t, false)
	priv, _ := a.key.Bytes()
	_, err := NewWebPush(WebPushConfig{
		PublicKey:  b.PublicKey(),
		PrivateKey: base64.RawURLEncoding.EncodeToString(priv),
		Subject:    "mailto:x@example.com",
	})
	if err == nil {
		t.Fatal("**قُبل زوجٌ لا يتطابق**")
	}
	if _, err := NewWebPush(WebPushConfig{PublicKey: a.PublicKey(),
		PrivateKey: base64.RawURLEncoding.EncodeToString(priv), Subject: "ops"}); err == nil {
		t.Fatal("**قُبل موضوعٌ بلا mailto: ولا https://**")
	}
}
