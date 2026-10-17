package push

import (
	"bytes"
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
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ══════════════════════════════════════════════════════════════════════
// **إشعاراتُ المتصفّح — Web Push** (قرارُ المالك ٢٠٢٦-١٠-١٧)
// ══════════════════════════════════════════════════════════════════════
//
// الزبونُ وصاحبُ المتجر على الويب (تطبيقُ الشاشة الرئيسيّة) **كانا لا
// يصلهما شيءٌ والصفحةُ مغلقة** — البثُّ الحيُّ يحتاج صفحةً مفتوحة.
//
// **والمعيارُ هو المعيار**: RFC 8030 للإرسال، وRFC 8292 (VAPID) لتوقيع
// المرسِل، وRFC 8291 (aes128gcm) لتشفير الحمولة. **وبلا حزمةٍ جديدة** —
// كلُّها في مكتبة Go القياسيّة، كما بُني ناقلُ FCM.
//
// **وبلا مفاتيح يصمت ولا يسقط**: من لم يضبط المتغيّرات الثلاثة لم يُبنَ
// له ناقل، والاشتراكُ يُرفض، وكلُّ ما عداه يعمل كما كان.
//
// # والمتغيّرات
//
//	WEBPUSH_VAPID_PUBLIC_KEY   المفتاحُ العامّ (base64url، ٦٥ بايت)
//	WEBPUSH_VAPID_PRIVATE_KEY  المفتاحُ الخاصّ (base64url، ٣٢ بايت) — سرّ
//	WEBPUSH_SUBJECT            mailto: أو https:// — لمن يتواصل معه مزوّدُ الدفع
//
// وتُولَّد مرّةً واحدة: `go run ./cmd/vapidkeys`.

// PlatformWeb **منصّةُ المتصفّح** — ورمزُها في صفوف النقل هو عنوانُ الاشتراك.
const PlatformWeb = "web"

// تطبيقا الويب اللذان يشتركان — والقائمةُ مغلقة.
const (
	WebAppCustomer = "customer"
	WebAppMerchant = "merchant"
)

// maxWebSubsPerUser **سقفُ اشتراكات الحساب الواحد** — متصفّحٌ يُمسح ويُعاد
// يترك اشتراكاً ميّتاً، والسقفُ يمنع تراكمَها إلى ما لا نهاية.
const maxWebSubsPerUser = 10

// webPushRecordSize حجمُ السجلّ في aes128gcm — وحمولتُنا سجلٌّ واحد.
const webPushRecordSize = 4096

// maxWebPayload **أقصى حمولةٍ قبل التشفير** — السجلُّ ٤٠٩٦ ناقصَ الوسم
// وبايتِ الفاصل وترويسةِ الملفّ. ونقصّ النصَّ قبل أن نبلغه.
const maxWebPayload = 3000

// ErrWebPushDisabled **الاشتراكُ والناقلُ غيرُ مهيّأ.**
var ErrWebPushDisabled = errors.New("webpush: غيرُ مهيّأ")

// ErrWebPushBadSubscription اشتراكٌ لا يصلح — عنوانٌ أو مفاتيحُ معطوبة.
var ErrWebPushBadSubscription = errors.New("webpush: اشتراكٌ غيرُ صالح")

// WebPushConfig إعدادُ الناقل.
type WebPushConfig struct {
	PublicKey  string
	PrivateKey string
	Subject    string
	// AllowHost **أيُّ مزوّدٍ يُقبل عنوانُه** — والافتراضُ قائمةُ المزوّدين
	// المعروفين. **فالخادمُ ينادي عنواناً كتبه المستخدم**، ومن قبل أيَّ عنوانٍ
	// جعل المحرّكَ ينادي شبكتَه الداخليّة.
	AllowHost func(scheme, host string) bool
	HTTP      *http.Client
}

// WebPush ناقلُ المتصفّح.
type WebPush struct {
	pub       []byte // ٦٥ بايت، غيرُ مضغوط
	pubB64    string
	key       *ecdsa.PrivateKey
	subject   string
	allowHost func(scheme, host string) bool
	http      *http.Client
	db        *pgxpool.Pool
	logger    *slog.Logger
	now       func() time.Time
}

// NewWebPushFromEnv **من المتغيّرات أو لا شيء.**
//
// غيابُ الثلاثة حالٌ عاديّة (جهازُ المطوّر) فيردّ لا شيء بلا خطأ. **وواحدٌ
// موجودٌ والآخرُ ناقصٌ أو معطوبٌ خطأٌ يُصرَّح به** — سكوتٌ عليه يعني منصّةً
// تظنّ أنّها تُرسل.
func NewWebPushFromEnv(logger *slog.Logger) (*WebPush, error) {
	pub := strings.TrimSpace(os.Getenv("WEBPUSH_VAPID_PUBLIC_KEY"))
	priv := strings.TrimSpace(os.Getenv("WEBPUSH_VAPID_PRIVATE_KEY"))
	subj := strings.TrimSpace(os.Getenv("WEBPUSH_SUBJECT"))
	if pub == "" && priv == "" {
		return nil, nil
	}
	w, err := NewWebPush(WebPushConfig{PublicKey: pub, PrivateKey: priv, Subject: subj})
	if err != nil {
		return nil, err
	}
	w.logger = logger
	if logger != nil {
		logger.Info("الدفع: إشعاراتُ المتصفّح مهيّأة")
	}
	return w, nil
}

// NewWebPush يبني الناقلَ من مفاتيحه — ويتحقّق أنّ العامَّ يطابق الخاصّ.
func NewWebPush(cfg WebPushConfig) (*WebPush, error) {
	if cfg.PublicKey == "" || cfg.PrivateKey == "" {
		return nil, errors.New("webpush: يلزم WEBPUSH_VAPID_PUBLIC_KEY وWEBPUSH_VAPID_PRIVATE_KEY معاً")
	}
	if !strings.HasPrefix(cfg.Subject, "mailto:") && !strings.HasPrefix(cfg.Subject, "https://") {
		return nil, errors.New("webpush: WEBPUSH_SUBJECT يبدأ بـ mailto: أو https://")
	}
	rawPriv, err := decodeB64(cfg.PrivateKey)
	if err != nil || len(rawPriv) != 32 {
		return nil, errors.New("webpush: المفتاحُ الخاصّ ليس ٣٢ بايتاً بـ base64url")
	}
	key, err := ecdsa.ParseRawPrivateKey(elliptic.P256(), rawPriv)
	if err != nil {
		return nil, fmt.Errorf("webpush: المفتاحُ الخاصّ معطوب: %w", err)
	}
	pub, err := key.PublicKey.Bytes()
	if err != nil {
		return nil, err
	}
	given, err := decodeB64(cfg.PublicKey)
	if err != nil || !bytes.Equal(given, pub) {
		return nil, errors.New("webpush: المفتاحُ العامّ لا يطابق الخاصّ")
	}
	allow := cfg.AllowHost
	if allow == nil {
		allow = KnownPushHost
	}
	hc := cfg.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second}
	}
	return &WebPush{
		pub: pub, pubB64: base64.RawURLEncoding.EncodeToString(pub),
		key: key, subject: cfg.Subject, allowHost: allow, http: hc,
		now: time.Now,
	}, nil
}

// KnownPushHost **مزوّدو الدفع المعروفون** — كروم وفايرفوكس وسفاري وإيدج.
//
// وعنوانٌ لغيرهم يُرفض عند الاشتراك. **ولا نداءَ إلى عنوانٍ داخليّ.**
func KnownPushHost(scheme, host string) bool {
	if scheme != "https" {
		return false
	}
	host = strings.ToLower(host)
	switch host {
	case "fcm.googleapis.com", "android.googleapis.com",
		"updates.push.services.mozilla.com":
		return true
	}
	for _, suffix := range []string{".push.apple.com", ".notify.windows.com",
		".push.services.mozilla.com"} {
		if strings.HasSuffix(host, suffix) {
			return true
		}
	}
	return false
}

// Platform المنصّةُ التي يخدمها.
func (w *WebPush) Platform() string { return PlatformWeb }

// PublicKey **المفتاحُ العامّ** — يقرؤه المتصفّحُ ليشترك.
func (w *WebPush) PublicKey() string {
	if w == nil {
		return ""
	}
	return w.pubB64
}

// webSub صفُّ اشتراكٍ كما يُقرأ للإرسال.
type webSub struct {
	App    string
	P256dh []byte
	Auth   []byte
}

// Send **يرسل إلى اشتراكاتٍ بعناوينها** — ويردّ ما قال المزوّدُ إنّه انتهى.
func (w *WebPush) Send(ctx context.Context, endpoints []string, msg Message) ([]string, error) {
	var dead []string
	var firstErr error
	for _, ep := range endpoints {
		sub, ok, err := w.lookup(ctx, ep)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		if !ok {
			// **حُذف الاشتراكُ قبل أن يُرسَل إليه** — خروجٌ أو إلغاء. لا يُعاد.
			dead = append(dead, ep)
			continue
		}
		gone, err := w.sendOne(ctx, ep, sub, msg)
		if gone {
			dead = append(dead, ep)
			continue
		}
		w.markResult(ctx, ep, err == nil)
		if err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return dead, firstErr
}

func (w *WebPush) lookup(ctx context.Context, endpoint string) (webSub, bool, error) {
	var s webSub
	if w.db == nil {
		return s, false, errors.New("webpush: لا قاعدة")
	}
	var p, a string
	err := w.db.QueryRow(ctx, `
		SELECT app, p256dh, auth FROM web_push_subscriptions WHERE endpoint = $1`,
		endpoint).Scan(&s.App, &p, &a)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return s, false, nil
		}
		return s, false, err
	}
	s.P256dh, _ = decodeB64(p)
	s.Auth, _ = decodeB64(a)
	return s, true, nil
}

func (w *WebPush) markResult(ctx context.Context, endpoint string, ok bool) {
	if w.db == nil {
		return
	}
	q := `UPDATE web_push_subscriptions SET failures = failures + 1 WHERE endpoint = $1`
	if ok {
		q = `UPDATE web_push_subscriptions SET last_ok_at = now(), failures = 0 WHERE endpoint = $1`
	}
	_, _ = w.db.Exec(ctx, q, endpoint)
}

// WebPayload **ما يقرؤه عاملُ الخدمة في المتصفّح** (`public/sw.js`).
type WebPayload struct {
	Title    string `json:"title"`
	Body     string `json:"body"`
	URL      string `json:"url"`
	Tag      string `json:"tag"`
	Kind     string `json:"kind,omitempty"`
	EntityID string `json:"entity_id,omitempty"`
	Urgent   bool   `json:"urgent,omitempty"`
}

// BuildWebPayload **يبني الحمولةَ لتطبيقٍ بعينه** — والوجهةُ تختلف بينهما.
func BuildWebPayload(app string, msg Message) WebPayload {
	kind, entity, id := msg.Data["kind"], msg.Data["entity"], msg.Data["entity_id"]
	tag := msg.Collapse
	if tag == "" {
		tag = kind
		if id != "" {
			tag = kind + "-" + id
		}
	}
	p := WebPayload{
		Title: cut(msg.Title, 200), Body: cut(msg.Body, 1000),
		URL: WebURL(app, kind, entity, id), Tag: cut(tag, 100),
		Kind: kind, EntityID: id, Urgent: msg.Urgent,
	}
	return p
}

// WebURL **أين يفتح الضغطُ على الإشعار.**
//
//	المتجر: طلبٌ ⇒ /store/orders · وغيرُه ⇒ /store/notifications
//	الزبون: طلبٌ ⇒ /orders/{id} · شكوى ⇒ /complaints/{id} · وغيرُه ⇒ /notifications
func WebURL(app, kind, entity, id string) string {
	safeID := url.PathEscape(id)
	if app == WebAppMerchant {
		if entity == "order" || kind == kindOrder {
			return "/store/orders"
		}
		return "/store/notifications"
	}
	switch {
	case (entity == "order" || entity == "order_chat") && id != "":
		return "/orders/" + safeID
	case entity == "ticket" && id != "":
		return "/complaints/" + safeID
	}
	return "/notifications"
}

func cut(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// webTTL **عمرُ الخبر عند المزوّد** — العاجلُ قصير، والعاديُّ يوم.
//
// **وطلبٌ جديدٌ للمتجر عاجل**: بعد عشر دقائقَ يكون قد ذهب أو صُعّد، **ورنّةٌ
// عنه بعدها تُربك.**
func webTTL(msg Message) int {
	if msg.TTL > 0 {
		sec := int((msg.TTL + time.Second - 1) / time.Second)
		if sec < 1 {
			sec = 1
		}
		return sec
	}
	if msg.Urgent {
		return 600
	}
	return 86400
}

func (w *WebPush) sendOne(ctx context.Context, endpoint string, sub webSub, msg Message) (gone bool, err error) {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || !w.allowHost(u.Scheme, u.Host) {
		// **عنوانٌ لا يُقبل اليوم** (قائمةٌ ضاقت) — يُحذف ولا يُنادى.
		return true, nil
	}
	payload := BuildWebPayload(sub.App, msg)
	plain, _ := json.Marshal(payload)
	if len(plain) > maxWebPayload {
		payload.Body = cut(payload.Body, 200)
		plain, _ = json.Marshal(payload)
	}
	asPriv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return false, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return false, err
	}
	body, err := encryptAES128GCM(plain, sub.P256dh, sub.Auth, asPriv, salt)
	if err != nil {
		// **مفاتيحُ الاشتراك معطوبة** — لن تصلح بالإعادة.
		return true, nil
	}
	jwtTok, err := w.vapidJWT(u.Scheme + "://" + u.Host)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", strconv.Itoa(webTTL(msg)))
	urgency := "normal"
	if msg.Urgent {
		urgency = "high"
	}
	req.Header.Set("Urgency", urgency)
	if t := webTopic(payload.Tag); t != "" {
		req.Header.Set("Topic", t)
	}
	req.Header.Set("Authorization", "vapid t="+jwtTok+", k="+w.pubB64)

	res, err := w.http.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<14))
	switch {
	case res.StatusCode < 300:
		return false, nil
	// **٤٠٤ و٤١٠: الاشتراكُ انتهى** — المتصفّحُ أُلغي أو مُسحت بياناته.
	case res.StatusCode == http.StatusNotFound || res.StatusCode == http.StatusGone:
		return true, nil
	default:
		return false, fmt.Errorf("مزوّدُ الدفع ردّ %d: %s", res.StatusCode, truncate(raw))
	}
}

// webTopic **مفتاحُ الطيّ عند المزوّد** — ٣٢ حرفاً من أبجديّة base64url.
func webTopic(tag string) string {
	if tag == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(tag))
	return base64.RawURLEncoding.EncodeToString(sum[:])[:32]
}

// vapidJWT **توقيعُ المرسِل** (RFC 8292) — ES256، عمرُه اثنتا عشرة ساعة.
func (w *WebPush) vapidJWT(audience string) (string, error) {
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": audience,
		"exp": w.now().Add(12 * time.Hour).Unix(),
		"sub": w.subject,
	})
	return tok.SignedString(w.key)
}

// encryptAES128GCM **تشفيرُ الحمولة** — RFC 8291 بحرفه، سجلٌّ واحد.
//
//	ecdh   = ECDH(as_private, ua_public)
//	ikm    = HKDF(salt=auth, ecdh, "WebPush: info"‖0‖ua_public‖as_public, 32)
//	cek    = HKDF(salt, ikm, "Content-Encoding: aes128gcm"‖0, 16)
//	nonce  = HKDF(salt, ikm, "Content-Encoding: nonce"‖0, 12)
//	الناتج = salt‖rs‖idlen‖as_public‖AES-GCM(plain‖0x02)
func encryptAES128GCM(plain, uaPublic, authSecret []byte, asPriv *ecdh.PrivateKey, salt []byte) ([]byte, error) {
	if len(authSecret) != 16 || len(salt) != 16 {
		return nil, ErrWebPushBadSubscription
	}
	uaPub, err := ecdh.P256().NewPublicKey(uaPublic)
	if err != nil {
		return nil, ErrWebPushBadSubscription
	}
	shared, err := asPriv.ECDH(uaPub)
	if err != nil {
		return nil, err
	}
	asPublic := asPriv.PublicKey().Bytes()

	info := make([]byte, 0, 14+65+65)
	info = append(info, "WebPush: info\x00"...)
	info = append(info, uaPublic...)
	info = append(info, asPublic...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(info), 32)
	if err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	padded := append(append([]byte{}, plain...), 0x02)
	sealed := gcm.Seal(nil, nonce, padded, nil)

	out := make([]byte, 0, 16+4+1+len(asPublic)+len(sealed))
	out = append(out, salt...)
	rs := make([]byte, 4)
	binary.BigEndian.PutUint32(rs, webPushRecordSize)
	out = append(out, rs...)
	out = append(out, byte(len(asPublic)))
	out = append(out, asPublic...)
	out = append(out, sealed...)
	return out, nil
}

// decodeB64 **يقبل base64url وbase64 العاديّ، بحشوٍ أو بلا حشو** — والمتصفّحاتُ
// لا تتّفق.
func decodeB64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	s = strings.NewReplacer("+", "-", "/", "_").Replace(s)
	s = strings.TrimRight(s, "=")
	return base64.RawURLEncoding.DecodeString(s)
}

// ValidWebSubscription **يفحص الاشتراكَ قبل حفظه** — عنوانٌ عند مزوّدٍ معروف،
// ومفتاحٌ على المنحنى، وسرٌّ بطوله.
func (w *WebPush) ValidWebSubscription(endpoint, p256dh, auth string) bool {
	if w == nil || len(endpoint) > 2048 {
		return false
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || !w.allowHost(u.Scheme, u.Host) {
		return false
	}
	pk, err := decodeB64(p256dh)
	if err != nil {
		return false
	}
	if _, err := ecdh.P256().NewPublicKey(pk); err != nil {
		return false
	}
	a, err := decodeB64(auth)
	return err == nil && len(a) == 16
}

// ══════════════════════════════════════════════════════════════════════
// **الاشتراكُ في الخدمة**
// ══════════════════════════════════════════════════════════════════════

// webTransport ناقلُ المتصفّح إن رُكّب.
func (s *Service) webTransport() *WebPush {
	if s == nil {
		return nil
	}
	w, _ := s.transports[PlatformWeb].(*WebPush)
	return w
}

// WebPushPublicKey **المفتاحُ العامّ — وفارغٌ يعني «غيرُ مهيّأ».**
func (s *Service) WebPushPublicKey() string { return s.webTransport().PublicKey() }

// SubscribeWeb **يحفظ اشتراكَ متصفّحٍ لصاحبه** — ومربوطٌ بعائلة جلسته (D12).
//
// والعنوانُ ينتقل إلى آخرِ من سجّله، كرمز الجهاز: متصفّحٌ واحدٌ دخله حسابٌ
// ثانٍ لا يبقى يستقبل إشعاراتِ الأوّل.
func (s *Service) SubscribeWeb(ctx context.Context, userID, sessionID, app, endpoint, p256dh, auth, userAgent string) error {
	w := s.webTransport()
	if w == nil {
		return ErrWebPushDisabled
	}
	if app != WebAppCustomer && app != WebAppMerchant {
		return ErrWebPushBadSubscription
	}
	if !w.ValidWebSubscription(endpoint, p256dh, auth) {
		return ErrWebPushBadSubscription
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if sessionID != "" {
		var alive int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM (
				SELECT 1 FROM refresh_tokens
				 WHERE session_id = $1::uuid AND revoked_at IS NULL AND expires_at > now()
				 FOR UPDATE
			) t`, sessionID).Scan(&alive); err != nil {
			return err
		}
		if alive == 0 {
			// **جلسةٌ أُبطلت** — لا يُكتب لها هدف.
			return tx.Commit(ctx)
		}
	}
	if len(userAgent) > 300 {
		userAgent = userAgent[:300]
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO web_push_subscriptions
		       (endpoint, user_id, session_id, app, p256dh, auth, user_agent)
		VALUES ($1, $2::uuid, nullif($3, '')::uuid, $4, $5, $6, $7)
		ON CONFLICT (endpoint) DO UPDATE
		SET user_id = EXCLUDED.user_id, session_id = EXCLUDED.session_id,
		    app = EXCLUDED.app, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth,
		    user_agent = EXCLUDED.user_agent, failures = 0`,
		endpoint, userID, sessionID, app, p256dh, auth, userAgent); err != nil {
		return err
	}
	// **والسقف**: الأحدثُ يبقى.
	if _, err := tx.Exec(ctx, `
		DELETE FROM web_push_subscriptions
		 WHERE user_id = $1::uuid AND endpoint NOT IN (
		       SELECT endpoint FROM web_push_subscriptions
		        WHERE user_id = $1::uuid ORDER BY created_at DESC LIMIT $2)`,
		userID, maxWebSubsPerUser); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UnsubscribeWeb يحذف اشتراكاً — **مقيَّداً بصاحبه.**
func (s *Service) UnsubscribeWeb(ctx context.Context, userID, endpoint string) error {
	if s == nil || endpoint == "" {
		return nil
	}
	_, err := s.db.Exec(ctx,
		`DELETE FROM web_push_subscriptions WHERE endpoint = $1 AND user_id = $2::uuid`,
		endpoint, userID)
	return err
}
