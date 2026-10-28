package server

// ══════════════════════════════════════════════════════════════════════
// **«اطلب الموقع من المستلم»** (قرارُ المالك ٢٠٢٦-١٠-٢٩، هجرة ٠٤٨٤)
// ══════════════════════════════════════════════════════════════════════
//
// كان المتجرُ يطلب من المستلِم رابطَ موقعٍ بالواتساب ثمّ ينسخه ويلصقه —
// **متعِبٌ للطرفين.** والآن:
//
//	١ · المتجرُ يكتب رقمَ المستلِم ويضغط «اطلب الموقع من المستلم».
//	٢ · يصل المستلِمَ بالواتساب رابطٌ لصفحةٍ عامّةٍ صغيرة (`/loc/<رمز>`)
//	    فيها زرٌّ واحد «أرسل موقعي» (موقعُ الجهاز بدقّةٍ عالية، والخريطةُ احتياط).
//	٣ · تصل النقطةُ ودقّتُها إلى نموذج التوصيلة — **والتطبيقُ يسأل كلَّ ثوانٍ.**
//
// # طلبٌ مستقلٌّ قبل التوصيلة — لا حقلٌ عليها
//
// **اختيرت المسودّة على «توصيلة بانتظار موقع المستلم»** لأنّ الأجرةَ والتغطيةَ
// تُحسبان من النقطة لحظةَ الإنشاء (`CreateMerchantDeliveryIn`) **ومالُها يُخصم
// ويُلقَط عندها** — **ونقطةٌ تصل بعد الإنشاء تعني أجرةً تُعاد حسابُها بعد الخصم.**
// فالطلبُ هنا لا يمسّ طلباً ولا مالاً: **نقطةٌ تملأ النموذجَ كما يملؤه الرابط.**
//
// # والصفحةُ العامّة لا تكشف شيئاً
//
// **اسمُ المتجر وحدَه** — لا رقمَ ولا عنوانَ ولا اسمَ مستلِم. **والرمزُ لا يُخزَّن**
// (بصمتُه وحدَها)، **وينتهي بعد ٢٤ ساعة أو عند أوّل إرسال.** وحدُّ معدّلٍ بالعنوان.

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

const (
	// locReqTTL **عمرُ الرابط** — يومٌ يكفي مستلِماً رأى الرسالةَ متأخّراً.
	locReqTTL = 24 * time.Hour
	// locReqGap **بين طلبين للرقم نفسِه** — ضغطتان متتاليتان لا تُزعجان المستلِم.
	locReqGap = time.Minute
	// locReqPerHour **حدُّ المتجر في الساعة** — رسائلُ البوت ليست مجّانيّةً على الرقم.
	locReqPerHour = 20
	// locPublicPerHour **حدُّ الصفحة العامّة للعنوان** — تخمينُ الرموز لا يُجرَّب مجّاناً.
	locPublicPerHour = 60
)

var (
	errLocReqTooSoon = httpx.NewError(http.StatusTooManyRequests,
		"location_request_too_soon", "errors.rate_limited")
	errLocReqLimit = httpx.NewError(http.StatusTooManyRequests,
		"location_request_limit", "errors.rate_limited")
	// errLocReqClosed **الرابطُ انتهى أو استُعمل** — والصفحةُ تقول ذلك بلفظها.
	errLocReqClosed = httpx.NewError(http.StatusGone,
		"location_request_closed", "errors.location_request_closed")
)

// locReqHash **بصمةُ الرمز** — والرمزُ نفسُه لا يُخزَّن.
func locReqHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// locReqLink **رابطُ الصفحة العامّة** — من أصل الموقع المسموح، وفي التطوير موقعُ الزبون.
func (s *Server) locReqLink(token string) string {
	base := s.siteBase()
	if base == "" {
		base = s.siteURL()
	}
	return base + "/loc/" + token
}

// locReqText **رسالةُ الواتساب** — ولا اسمَ منصّةٍ فيها (قاعدةُ المالك).
func locReqText(store, mode, link string) string {
	why := "لتوصيل طلبك إليك"
	if mode == "fetch" {
		why = "ليصل إليك السائق ويستلم الغرض"
	}
	return "مرحباً، يطلب منك متجر «" + store + "» موقعك " + why + ".\n" +
		"اضغط الرابط ثمّ «أرسل موقعي»:\n" + link
}

// handleMerchantRequestLocation **يطلب موقعَ المستلِم برسالة واتساب** —
// `POST /merchant/stores/{id}/location-requests`.
func (s *Server) handleMerchantRequestLocation(w http.ResponseWriter, r *http.Request) {
	merchantID := chi.URLParam(r, "id")
	if !s.ownsMerchant(r, merchantID) {
		s.respondErr(w, errForbidden)
		return
	}
	req, err := decode[struct {
		Phone string `json:"phone"`
		Mode  string `json:"mode"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	mode := "send"
	if req.Mode == "fetch" {
		mode = "fetch"
	}
	ctx := r.Context()

	// **الحدّان قبل الكتابة** — ضغطتان للرقم نفسِه في دقيقة، وعشرون في الساعة للمتجر.
	var recent, hourly int
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE phone = $2 AND created_at > now() - make_interval(secs => $3)),
		       count(*) FILTER (WHERE created_at > now() - interval '1 hour')
		  FROM recipient_location_requests WHERE merchant_id = $1`,
		merchantID, phone, int(locReqGap.Seconds())).Scan(&recent, &hourly); err != nil {
		s.respondErr(w, err)
		return
	}
	if recent > 0 {
		e := *errLocReqTooSoon
		e.Details = map[string]any{"retry_after_sec": int(locReqGap.Seconds())}
		s.respondErr(w, &e)
		return
	}
	if hourly >= locReqPerHour {
		s.respondErr(w, errLocReqLimit)
		return
	}

	raw := make([]byte, 18)
	if _, err := rand.Read(raw); err != nil {
		s.respondErr(w, err)
		return
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	var id, store string
	var expires time.Time
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO recipient_location_requests
		       (merchant_id, requested_by, phone, mode, token_hash, expires_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, now() + make_interval(secs => $6))
		RETURNING id::text, expires_at,
		          (SELECT name FROM merchants WHERE id = $1)`,
		merchantID, userIDFrom(r), phone, mode, locReqHash(token),
		int(locReqTTL.Seconds())).Scan(&id, &expires, &store); err != nil {
		s.respondErr(w, err)
		return
	}
	link := s.locReqLink(token)

	// **والإرسالُ بانتظاره** — الشاشةُ تقول «وصلت الرسالة» أو تعرض زرَّ المشاركة.
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	sent := s.sendText(sendCtx, phone, locReqText(store, mode, link))
	if sent {
		if _, err := s.pg.Exec(ctx,
			`UPDATE recipient_location_requests SET whatsapp_sent = true WHERE id = $1`, id); err != nil {
			s.logger.Warn("طلبُ الموقع: تعليمُ الإرسال سقط", "error", err)
		}
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":            id,
		"status":        "pending",
		"whatsapp_sent": sent,
		// **والرابطُ له** — إن لم تصل الرسالةُ يشاركه بنفسه.
		"link":       link,
		"share_text": locReqText(store, mode, link),
		// **ورابطُ واتساب جاهزٌ برقمه الدوليّ** — المكتوبُ في النموذج قد يكون محلّيّاً.
		"wa_url": "https://wa.me/" + strings.TrimPrefix(phone, "+") +
			"?text=" + url.QueryEscape(locReqText(store, mode, link)),
		"expires_at": expires.Format(time.RFC3339),
	})
}

// locReqStatus **حالُ الطلب** — `pending` · `received` · `expired`.
func locReqStatus(received *time.Time, expires time.Time) string {
	switch {
	case received != nil:
		return "received"
	case time.Now().After(expires):
		return "expired"
	default:
		return "pending"
	}
}

// handleMerchantLocationRequest **أوصل موقعُ المستلِم؟** — `GET /merchant/location-requests/{id}`.
// **والتطبيقُ يسأله كلَّ ثوانٍ** ما دام النموذجُ مفتوحاً.
func (s *Server) handleMerchantLocationRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var merchantID string
	var lat, lng, acc *float64
	var received *time.Time
	var expires time.Time
	var sent bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT merchant_id::text, lat, lng, accuracy_m, received_at, expires_at, whatsapp_sent
		  FROM recipient_location_requests WHERE id = $1`, id).
		Scan(&merchantID, &lat, &lng, &acc, &received, &expires, &sent)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !s.ownsMerchant(r, merchantID)) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	out := map[string]any{
		"id":            id,
		"status":        locReqStatus(received, expires),
		"whatsapp_sent": sent,
		"lat":           lat,
		"lng":           lng,
		"accuracy_m":    acc,
		"received_at":   nil,
	}
	if received != nil {
		out["received_at"] = received.Format(time.RFC3339)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// locPublicAllowed **حدُّ الصفحة العامّة بالعنوان** — كحارس حجز الدعوة.
func (s *Server) locPublicAllowed(r *http.Request) bool {
	key := "locreq:ip:" + clientIP(r)
	n, err := s.rdb.Incr(r.Context(), key).Result()
	if err != nil {
		return true // عطلُ الكاش لا يعطّل الصفحة
	}
	if n == 1 {
		s.rdb.Expire(r.Context(), key, time.Hour)
	}
	return n <= locPublicPerHour
}

// handlePublicLocationRequest **ما تعرضه صفحةُ المستلِم** — `GET /public/location-requests/{token}`.
// **اسمُ المتجر والنوعُ والحال لا غير.**
func (s *Server) handlePublicLocationRequest(w http.ResponseWriter, r *http.Request) {
	if !s.locPublicAllowed(r) {
		s.respondErr(w, ErrInviteClaimRateLimited)
		return
	}
	token := chi.URLParam(r, "token")
	if len(token) < 16 || len(token) > 64 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var store, mode string
	var received *time.Time
	var expires time.Time
	err := s.pg.QueryRow(r.Context(), `
		SELECT m.name, l.mode, l.received_at, l.expires_at
		  FROM recipient_location_requests l JOIN merchants m ON m.id = l.merchant_id
		 WHERE l.token_hash = $1`, locReqHash(token)).Scan(&store, &mode, &received, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"store_name": store,
		"mode":       mode,
		"status":     locReqStatus(received, expires),
	})
}

// handlePublicSubmitLocation **المستلِمُ يرسل موقعَه** — `POST /public/location-requests/{token}`.
// **مرّةً واحدة** — وما بعدها `location_request_closed`.
func (s *Server) handlePublicSubmitLocation(w http.ResponseWriter, r *http.Request) {
	if !s.locPublicAllowed(r) {
		s.respondErr(w, ErrInviteClaimRateLimited)
		return
	}
	token := chi.URLParam(r, "token")
	if len(token) < 16 || len(token) > 64 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		Lat       *float64 `json:"lat"`
		Lng       *float64 `json:"lng"`
		AccuracyM *float64 `json:"accuracy_m"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Lat == nil || req.Lng == nil || !orders.ValidPoint(*req.Lat, *req.Lng) {
		s.respondErr(w, orders.ErrBadPoint)
		return
	}
	// **والدقّةُ اختياريّة** — ومن الخريطة لا دقّة. وما خرج عن المعقول يُهمَل.
	var acc *float64
	if req.AccuracyM != nil && *req.AccuracyM > 0 && *req.AccuracyM < 100000 &&
		!math.IsNaN(*req.AccuracyM) {
		v := math.Round(*req.AccuracyM)
		acc = &v
	}
	hash := locReqHash(token)
	var ownerID string
	err = s.pg.QueryRow(r.Context(), `
		UPDATE recipient_location_requests l
		   SET lat = $2, lng = $3, accuracy_m = $4, received_at = now()
		  FROM merchants m
		 WHERE l.token_hash = $1 AND m.id = l.merchant_id
		   AND l.received_at IS NULL AND l.expires_at > now()
		RETURNING COALESCE(m.owner_user_id::text, '')`,
		hash, *req.Lat, *req.Lng, acc).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = s.pg.QueryRow(r.Context(),
			`SELECT EXISTS(SELECT 1 FROM recipient_location_requests WHERE token_hash = $1)`,
			hash).Scan(&exists)
		if !exists {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errLocReqClosed)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ونبضةٌ لصاحب المتجر** — شاشتُه تقرأ من جديد ولا تنتظر دورةَ السؤال.
	if strings.TrimSpace(ownerID) != "" {
		s.touchUser(ownerID, "deliveries")
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
