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
// **اسمُ المنصّة وحدَه** (لا اسمَ متجرٍ منذ ٢٠٢٦-١١-٠٦) — لا رقمَ ولا عنوانَ ولا اسمَ مستلِم. **والرمزُ لا يُخزَّن**
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

	"github.com/servacode/rahalgo/backend/internal/dbtx"
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

// locReqPlatformName **ما يراه المستلِمُ اسماً** — لا اسمَ متجر (قرارُ المالك ٢٠٢٦-١١-٠٦).
const locReqPlatformName = "رحّال غو"

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

// locReqText **رسالةُ الواتساب — واحدةٌ لكلّ طلب موقع** (قرارُ المالك ٢٠٢٦-١١-٠٥، نصُّه):
// توصيلةُ المتجر و«إحضار» والأمانةُ وطلبُ الواتساب من المكتب. **ويُلغي قاعدةَ «لا اسمَ منصّةٍ
// فيها»** — المالكُ كتبها باسم التطبيق: «طبعا هي الرسالة تصبح للكل وقت الامانة والاحضار والتوصيل».
//
// **ولا رابطَ ولا اسمَ متجر** (قرارُ المالك ٢٠٢٦-١١-٠٦): «الناس بسوريا تخاف من روابط الهكر»
// و«دائما لديك امانة او طلب من تطبيق رحال غو … مشان اي شخص جديد يسال شو هو رحال غو». فالمستلِمُ
// يردّ على المحادثة بموقعه الحاليّ أو المباشر من زرّ المشبك، **والبوتُ يلتقطه** (`HandleWALocation`).
// **وصفحةُ `/loc` باقيةٌ لرسائلَ وصلت قبل القرار** — لكنّ رابطَها لا يُكتب في رسالةٍ جديدة.
func locReqText(mode string) string {
	what := "لديك طلب"
	if mode == "amanah" {
		what = "لديك أمانة"
	}
	text := "السلام عليكم ورحمة الله وبركاته\n" +
		what + " من تطبيق رحّال غو للتسوق وطلبات التوصيل بالرقة\n" +
		"ياريت تشارك موقعك أو ترسله عبر الواتساب لنقدر نوصل إليك بسهولة:\n" +
		"ردّ على هالرسالة بموقعك: اضغط 📎 ← الموقع ← أرسل موقعك الحالي أو المباشر"
	// **وطلبُ الواتساب يطلب العنوانَ المكتوبَ معه** (قرارُ المالك ٢٠٢٦-١١-٠٦) — والبوتُ يلتقط الردَّ
	// النصّيَّ عنواناً للطلب (`HandleWAAddressText`). وغيرُه لا يقرأ أحدٌ ردَّه النصّيّ.
	if mode == locReqModeWhatsApp {
		text += "\n" + waAddressAskLine
	}
	return text
}

// waAddressAskLine **سطرُ العنوان في رسالة طلب الواتساب** (قرارُ المالك ٢٠٢٦-١١-٠٦).
const waAddressAskLine = "وإذا بتحب اكتبلنا عنوانك بالتفصيل: الحي، الشارع، وأقرب معلم"

// locReqAcc **تُنظّف الدقّة** — اختياريّة، وما خرج عن المعقول يُهمَل.
func locReqAcc(v *float64) *float64 {
	if v == nil || *v <= 0 || *v >= 100000 || math.IsNaN(*v) {
		return nil
	}
	r := math.Round(*v)
	return &r
}

// locReqReceived **ما يلي وصولَ الموقع** — واحدٌ للرابط وللردّ بالواتساب فلا يختلفان:
// نبضةٌ لصاحب المتجر، وكتابةُ النقطة في الطلب المربوط وبثُّها للسائق، ونبضةٌ للزبون وللمكتب.
//
//	amanah    نقطةُ الاستلام في الأمانة (٢٠٢٦-١٠-٣٠)
//	whatsapp  نقطةُ التسليم في طلب الواتساب (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب)
func (s *Server) locReqReceived(ctx context.Context, ownerID, customerID, orderID, mode string, lat, lng float64) {
	// **ونبضةٌ لصاحب المتجر** — شاشتُه تقرأ من جديد ولا تنتظر دورةَ السؤال.
	if strings.TrimSpace(ownerID) != "" {
		s.touchUser(ownerID, "deliveries")
	}
	// **الطلبُ أُنشئ قبل أن يصل الموقع** — فتُكتب النقطةُ فيه وتُبثّ للسائق. **ونقطةٌ خارجَ
	// التغطية لا تُسقط الصفحة**: في الأمانة يتّصل به السائق، وفي طلب الواتساب تُكتب ويُخبَر المكتب.
	if orderID != "" {
		switch mode {
		case locReqModeWhatsApp:
			if _, err := s.orders.ApplyWhatsAppShared(ctx, orderID, lat, lng); err != nil {
				s.logger.Warn("موقعُ مستلِم طلب الواتساب: تعذّرت كتابتُه في الطلب", "order", orderID, "error", err)
			}
		default:
			if _, err := s.orders.ApplyAmanahShared(ctx, orderID, lat, lng); err != nil {
				s.logger.Warn("موقعُ الأمانة: تعذّرت كتابتُه في الطلب", "order", orderID, "error", err)
			}
		}
	}
	if customerID != "" {
		s.touchUser(customerID, "location_request", "orders")
	}
	// **ونموذجُ الواتساب في اللوحة يقرأ من جديد** — لا ينتظر دورةَ السؤال.
	if mode == locReqModeWhatsApp {
		s.touch("orders", "ops")
	}
}

// HandleWALocation **المستلِمُ ردّ على رسالة البوت بموقعه** (قرارُ المالك ٢٠٢٦-١١-٠٣).
// يملأ **أحدثَ** طلبٍ مفتوحٍ لرقمه (لم يصل ولم ينتهِ) بالمسار نفسِه الذي يملأ به الرابط.
// **ولا طلبَ مفتوحاً ⇒ سلسلةٌ فارغة** (لا ردَّ، والرسالةُ تمرّ كما كانت).
//
// **والموقعُ المباشر كالعاديّ** (٢٠٢٦-١١-٠٦) — البوتُ يقرأ الاثنين (`notify.LocationFromMessage`).
// **أوّلُ نقطةٍ تملأ الطلب**، وتحديثاتُه اللاحقة لا تجد طلباً مفتوحاً (`received_at IS NULL`)
// فلا ردَّ ولا نبضةَ ولا نقطةً جديدةً للسائق.
func (s *Server) HandleWALocation(ctx context.Context, from string, lat, lng, accuracy float64) string {
	phone, ok := identity.NormalizePhone(from)
	if !ok || !orders.ValidPoint(lat, lng) {
		return ""
	}
	var ownerID, customerID, orderID, mode string
	err := s.pg.QueryRow(ctx, `
		UPDATE recipient_location_requests l
		   SET lat = $2, lng = $3, accuracy_m = $4, received_at = now()
		 WHERE l.id = (SELECT id FROM recipient_location_requests
		                WHERE phone = $1 AND received_at IS NULL AND expires_at > now()
		                ORDER BY created_at DESC LIMIT 1)
		   AND l.received_at IS NULL AND l.expires_at > now()
		RETURNING COALESCE((SELECT m.owner_user_id::text FROM merchants m WHERE m.id = l.merchant_id), ''),
		          COALESCE(l.customer_id::text, ''), COALESCE(l.order_id::text, ''), l.mode`,
		phone, lat, lng, locReqAcc(&accuracy)).Scan(&ownerID, &customerID, &orderID, &mode)
	if errors.Is(err, pgx.ErrNoRows) {
		return ""
	}
	if err != nil {
		s.logger.Warn("ردُّ الموقع بالواتساب: التحديثُ سقط", "error", err)
		return ""
	}
	s.locReqReceived(ctx, ownerID, customerID, orderID, mode, lat, lng)
	return "وصلنا موقعك ✅ شكراً، السائق رح يوصل لعندك."
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
	var id string
	var expires time.Time
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO recipient_location_requests
		       (merchant_id, requested_by, phone, mode, token_hash, expires_at)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, now() + make_interval(secs => $6))
		RETURNING id::text, expires_at`,
		merchantID, userIDFrom(r), phone, mode, locReqHash(token),
		int(locReqTTL.Seconds())).Scan(&id, &expires); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, s.locReqSend(ctx, id, phone, mode, token, expires))
}

// locReqSend **يرسل الرسالةَ الواحدة ويعلّم وصولَها** — ويردّ ما تعرضه الشاشة. **واحدٌ للمتجر
// وللأمانة ولطلب الواتساب من المكتب** فلا تفترق رسالتان (قرارُ المالك ٢٠٢٦-١١-٠٥).
//
// **والإرسالُ بانتظاره** — الشاشةُ تقول «وصلت الرسالة» أو تعرض زرَّ المشاركة.
func (s *Server) locReqSend(ctx context.Context, id, phone, mode, token string, expires time.Time) map[string]any {
	link := s.locReqLink(token)
	text := locReqText(mode)
	sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	sent := s.sendText(sendCtx, phone, text)
	if sent {
		if _, err := s.pg.Exec(ctx,
			`UPDATE recipient_location_requests SET whatsapp_sent = true WHERE id = $1`, id); err != nil {
			s.logger.Warn("طلبُ الموقع: تعليمُ الإرسال سقط", "error", err)
		}
	}
	return map[string]any{
		"id":            id,
		"status":        "pending",
		"whatsapp_sent": sent,
		// **والرابطُ للموظّف وحدَه** (نسخُه يدوياً) — لا يُكتب في الرسالة (قرارُ المالك ٢٠٢٦-١١-٠٦).
		"link":       link,
		"share_text": text,
		// **ورابطُ واتساب جاهزٌ برقمه الدوليّ** — المكتوبُ في النموذج قد يكون محلّيّاً.
		"wa_url": "https://wa.me/" + strings.TrimPrefix(phone, "+") +
			"?text=" + url.QueryEscape(text),
		"expires_at": expires.Format(time.RFC3339),
	}
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
// **اسمُ المنصّة والنوعُ والحال لا غير.**
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
	var mode string
	var received *time.Time
	var expires time.Time
	err := s.pg.QueryRow(r.Context(), `
		SELECT l.mode, l.received_at, l.expires_at
		  FROM recipient_location_requests l
		 WHERE l.token_hash = $1`, locReqHash(token)).Scan(&mode, &received, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **اسمُ المنصّة وحدَه — لا اسمُ متجرٍ ولا زبونٍ ولا ما كتبه المكتب** (قرارُ المالك ٢٠٢٦-١١-٠٦:
	// «الافضل مانذكر اسم المتجر»). والحقلُ باسمه القديم (`store_name`) فلا تنكسر الصفحة.
	store := locReqPlatformName
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
	acc := locReqAcc(req.AccuracyM)
	hash := locReqHash(token)
	var ownerID, customerID, orderID, mode string
	err = s.pg.QueryRow(r.Context(), `
		UPDATE recipient_location_requests l
		   SET lat = $2, lng = $3, accuracy_m = $4, received_at = now()
		 WHERE l.token_hash = $1
		   AND l.received_at IS NULL AND l.expires_at > now()
		RETURNING COALESCE((SELECT m.owner_user_id::text FROM merchants m WHERE m.id = l.merchant_id), ''),
		          COALESCE(l.customer_id::text, ''), COALESCE(l.order_id::text, ''), l.mode`,
		hash, *req.Lat, *req.Lng, acc).Scan(&ownerID, &customerID, &orderID, &mode)
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
	s.locReqReceived(r.Context(), ownerID, customerID, orderID, mode, *req.Lat, *req.Lng)
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// ══════════════════════════════════════════════════════════════════════
// **«اطلب الموقع منه» في الأمانة** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
// ══════════════════════════════════════════════════════════════════════
//
// **الزبونُ لا يعرف أين الأمانة** — يكتب اسمَ صاحبها ورقمَه ويضغط الزرّ، فيصل صاحبَها
// الرابطُ نفسُه الذي يصل مستلِمَ «إحضار» المتجر. **والتدفّقُ مرآةُ المتجر**:
//
//	١ · `POST /my/location-requests` قبل الإرسال — يردّ `id` ورابطَ واتساب احتياطاً.
//	٢ · التطبيقُ يسأل `GET /my/location-requests/{id}` ويقول «وصلنا موقعه ✓».
//	٣ · الطلبُ يُرسَل ومعه `location_request_id` — فيُربط الطلبُ به (`order_id`):
//	    وصل الموقعُ قبلها؟ يُكتب نقطةَ استلامٍ في معاملة الإنشاء. **وإن وصل بعدها**
//	    كُتب حين يصل (`handlePublicSubmitLocation` ← `ApplyAmanahShared`) وبُثّ للسائق.
//
// **والحدّان كحدَّي المتجر** — دقيقةٌ بين طلبين للرقم نفسِه، وعشرون في الساعة للزبون.

// handleCustomerRequestLocation — `POST /my/location-requests`.
func (s *Server) handleCustomerRequestLocation(w http.ResponseWriter, r *http.Request) {
	customerID := userIDFrom(r)
	req, err := decode[struct {
		Phone string `json:"phone"`
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
	ctx := r.Context()
	var recent, hourly int
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE phone = $2 AND created_at > now() - make_interval(secs => $3)),
		       count(*) FILTER (WHERE created_at > now() - interval '1 hour')
		  FROM recipient_location_requests WHERE customer_id = $1`,
		customerID, phone, int(locReqGap.Seconds())).Scan(&recent, &hourly); err != nil {
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
	var id string
	var expires time.Time
	if err := s.pg.QueryRow(ctx, `
		INSERT INTO recipient_location_requests
		       (customer_id, requested_by, phone, mode, token_hash, expires_at)
		VALUES ($1, $1, $2, 'amanah', $3, now() + make_interval(secs => $4))
		RETURNING id::text, expires_at`,
		customerID, phone, locReqHash(token), int(locReqTTL.Seconds())).
		Scan(&id, &expires); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, s.locReqSend(ctx, id, phone, "amanah", token, expires))
}

// handleCustomerLocationRequest **أوصل موقعُ صاحب الأمانة؟** — `GET /my/location-requests/{id}`.
func (s *Server) handleCustomerLocationRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var lat, lng, acc *float64
	var received *time.Time
	var expires time.Time
	var sent bool
	err := s.pg.QueryRow(r.Context(), `
		SELECT lat, lng, accuracy_m, received_at, expires_at, whatsapp_sent
		  FROM recipient_location_requests WHERE id = $1 AND customer_id = $2`,
		id, userIDFrom(r)).Scan(&lat, &lng, &acc, &received, &expires, &sent)
	if errors.Is(err, pgx.ErrNoRows) {
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

// errAmanahLocReq **طلبُ الموقع ليس له أو رُبط بطلبٍ آخر.**
var errAmanahLocReq = httpx.NewError(http.StatusUnprocessableEntity,
	"location_request_invalid", "errors.location_request_invalid")

// errAmanahHolder **الأمانةُ بلا اسم صاحبها أو رقمه** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
var errAmanahHolder = httpx.NewError(http.StatusUnprocessableEntity,
	"amanah_holder_required", "errors.amanah_holder_required")

// linkAmanahLocationTx **يربط طلبَ الموقع بالأمانة في معاملة الإنشاء** — ويردّ النقطةَ إن
// وصلت قبلها. **طلبُ الزبون نفسِه وحدَه، وغيرُ مربوطٍ بطلبٍ آخر.**
func linkAmanahLocationTx(ctx context.Context, q dbtx.Querier, reqID, customerID, orderID string) (lat, lng *float64, err error) {
	if !isUUID(reqID) {
		return nil, nil, errAmanahLocReq
	}
	err = q.QueryRow(ctx, `
		UPDATE recipient_location_requests
		   SET order_id = $3
		 WHERE id = $1 AND customer_id = $2 AND mode = 'amanah' AND order_id IS NULL
		RETURNING lat, lng`, reqID, customerID, orderID).Scan(&lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errAmanahLocReq
	}
	return lat, lng, err
}
