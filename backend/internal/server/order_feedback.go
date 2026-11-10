package server

// ══════════════════════════════════════════════════════════════════════
// **طلبُ رأيِ الزبون بعد التسليم** (قرارُ المالك ٢٠٢٦-١١-٠٤، هجرة ٠٤٩٨)
// ══════════════════════════════════════════════════════════════════════
//
// بعد عشر دقائق من تسليم الطلب يصل الزبونَ بالواتساب رسالةٌ ودّيّةٌ ورابطٌ لصفحةٍ
// عامّةٍ صغيرة (`/rate/<رمز>`) بلا حساب: نجومُ الخدمة (لازمة)، ونجومُ السائق (إن كان
// للطلب سائق)، وكلمةٌ عن التطبيق. وهو **مرآةُ «اطلب الموقع من المستلم»**
// (`recipient_location.go`): رمزٌ عشوائيّ لا تُخزَّن منه إلّا بصمتُه، وحدُّ معدّلٍ بالعنوان،
// وينتهي بعد ٧ أيّام أو عند أوّل إرسال.
//
// # النجومُ تُكتب بمسار التطبيق نفسِه
//
// `orders.RateOrder` — فالمتوسّطُ والسمعةُ والتجميعاتُ لا تعرف من أين جاء التقييم.
// **والكلمةُ في `order_ratings.app_feedback` وتظهر على شاشة التقييمات في اللوحة**
// (شرطُ المالك: لا تضيع كما ضاعت التعليقاتُ قديماً — هجرة ٠١٢٥).
//
// # ومن لا يُسأل
//
//	· طلبٌ غيرُ مُسلَّم (ملغى · فشل · مُسترَدّ · ما بينهما).
//	· `merchant_delivery` — المستلِمُ ليس زبوننا. **إلّا طلبَ الواتساب** (٢٠٢٦-١١-١٠): المستلمُ زبوننا.
//	· من قيَّم في التطبيق (صفٌّ في `order_ratings`).
//	· من سُئل مرّةً (`order_feedback_requests` مفتاحُها الطلب).
//	· ما مضى على تسليمه أكثرُ من يوم — **كي لا يُغرَق زبائنُ الأمس عند أوّل نشر.**
//
// # الإرسال
//
// الكانسُ يحجز الصفَّ **قبل** الإرسال (`ON CONFLICT DO NOTHING`) فلا يتسابق كانسان، **ولا
// إعادةَ إرسال**: من سقط إرسالُه يبقى صفُّه بلا `sent_at` ويُسجَّل. والبوتُ غيرُ جاهزٍ ⇒ لا
// حجزَ ولا إرسال، وسطرُ سجلٍّ يتيم، وتُلتقَط الطلباتُ حين يعود ما دامت داخل اليوم.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
)

const (
	// feedbackAfter **متى يُسأل الزبون** — بعد التسليم بعشر دقائق.
	feedbackAfter = 10 * time.Minute
	// feedbackWindow **أقدمُ تسليمٍ يُسأل عنه** — يومٌ، فلا تُغرَق الطلباتُ القديمة.
	feedbackWindow = 24 * time.Hour
	// feedbackTTL **عمرُ الطلب**: عمرُ الروابط القديمة (٧ أيّام).
	feedbackTTL = 7 * 24 * time.Hour
	// feedbackBatch **حدُّ الجولة** — رسائلُ البوت ليست مجّانيّةً على الرقم.
	feedbackBatch = 20
	// feedbackMaxRunes **أقصى طولٍ للكلمة.**
	feedbackMaxRunes = 1000
	// feedbackPublicPerHour **حدُّ الصفحة العامّة للعنوان.**
	feedbackPublicPerHour = 60
)

// feedbackSendGap **الفاصلُ بين رسالتين في الجولة** — متغيّرٌ ليُصفَّر في الفحص.
var feedbackSendGap = 3 * time.Second

// errFeedbackClosed **الرابطُ انتهى أو استُعمل أو قُيّم الطلبُ بغيره.**
var errFeedbackClosed = httpx.NewError(http.StatusGone,
	"feedback_request_closed", "errors.feedback_request_closed")

// feedbackText **رسالةُ الواتساب** — **بلا أيّ رابط** (قرارُ المالك ٢٠٢٦-١١-٠٤: «الناس تخاف الروابط»):
// الزبونُ يردّ عليها بكلامه في الواتساب نفسِه ويقرؤه المالك.
//
// **ونصُّها بألفاظ المالك** (٢٠٢٦-١١-١٠) — وشِيل منها سطرُ الحماية: «مزعجة وتخوف الزبائن».
func feedbackText(number int64) string {
	return "طلبك رقم " + strconv.FormatInt(number, 10) + "\n" +
		"كيف كانت الخدمة؟ رأيك يهمنا 🌷\n" +
		"مع تحيات إدارة تطبيق رحّال غو"
}

// RunOrderFeedbackSweeper **يسأل الزبائنَ عن رأيهم** — دورةٌ كلَّ دقيقة تكفي.
func (s *Server) RunOrderFeedbackSweeper(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.SweepOrderFeedback(ctx)
		}
	}
}

// SweepOrderFeedback جولةٌ واحدة — ويردّ عددَ الرسائل التي وصلت. **يُنادى من الفحص مباشرةً.**
func (s *Server) SweepOrderFeedback(ctx context.Context) int {
	type cand struct {
		orderID, phone string
		number         int64
	}
	rows, err := s.pg.Query(ctx, `
		SELECT o.id::text, o.number,
		       COALESCE(NULLIF(u.whatsapp_phone, ''), u.phone::text, NULLIF(o.recipient_phone, ''))
		  FROM orders o
		  LEFT JOIN users u ON u.id = o.customer_id
		 WHERE o.status = 'delivered'
		   -- **وطلبُ الواتساب يُسأل كباقي الطلبات** (قرارُ المالك ٢٠٢٦-١١-١٠) — على رقم المستلم.
		   AND (o.kind <> 'merchant_delivery' OR o.origin = 'whatsapp')
		   AND COALESCE(NULLIF(u.whatsapp_phone, ''), u.phone::text, NULLIF(o.recipient_phone, '')) IS NOT NULL
		   AND o.delivered_at <= now() - make_interval(secs => $1)
		   AND o.delivered_at >  now() - make_interval(secs => $2)
		   AND NOT EXISTS (SELECT 1 FROM order_ratings r WHERE r.order_id = o.id)
		   AND NOT EXISTS (SELECT 1 FROM order_feedback_requests f WHERE f.order_id = o.id)
		 ORDER BY o.delivered_at
		 LIMIT $3`,
		int(feedbackAfter.Seconds()), int(feedbackWindow.Seconds()), feedbackBatch)
	if err != nil {
		s.logger.Warn("طلبُ الرأي: قراءةُ المرشَّحين سقطت", "error", err)
		return 0
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.orderID, &c.number, &c.phone); err != nil {
			rows.Close()
			s.logger.Warn("طلبُ الرأي: قراءةُ صفٍّ سقطت", "error", err)
			return 0
		}
		cands = append(cands, c)
	}
	rows.Close()
	if len(cands) == 0 {
		return 0
	}
	// **البوتُ غيرُ جاهز ⇒ لا حجزَ ولا إرسال** — أثرٌ في السجلّ ولا إعادةَ إلحاح.
	if !s.merchantReady() {
		s.logger.Info("طلبُ الرأي: البوتُ غيرُ جاهز — تأجّل", "pending", len(cands))
		return 0
	}

	sent := 0
	for _, c := range cands {
		if ctx.Err() != nil {
			break
		}
		// **الحجزُ قبل الإرسال** — من خسر السباقَ لا يصله شيء. **ولا رمزَ ولا رابط**: العمودُ
		// `token_hash` فريدٌ فيُملأ بقيمةٍ مشتقّةٍ من الطلب (الصفحةُ العامّةُ للروابط القديمة تبقى).
		tag, err := s.pg.Exec(ctx, `
			INSERT INTO order_feedback_requests (order_id, token_hash, expires_at)
			VALUES ($1, $2, now() + make_interval(secs => $3))
			ON CONFLICT DO NOTHING`,
			c.orderID, "nolink-"+c.orderID, int(feedbackTTL.Seconds()))
		if err != nil {
			s.logger.Warn("طلبُ الرأي: الحجزُ سقط", "order", c.orderID, "error", err)
			continue
		}
		if tag.RowsAffected() == 0 {
			continue
		}
		sendCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		ok := s.sendText(sendCtx, c.phone, feedbackText(c.number))
		cancel()
		if !ok {
			s.logger.Warn("طلبُ الرأي: الرسالةُ لم تصل — لا إعادة", "order", c.orderID)
			continue
		}
		sent++
		if _, err := s.pg.Exec(ctx,
			`UPDATE order_feedback_requests SET sent_at = now() WHERE order_id = $1`, c.orderID); err != nil {
			s.logger.Warn("طلبُ الرأي: تعليمُ الإرسال سقط", "order", c.orderID, "error", err)
		}
		if feedbackSendGap > 0 {
			select {
			case <-ctx.Done():
			case <-time.After(feedbackSendGap):
			}
		}
	}
	return sent
}

// feedbackPublicAllowed **حدُّ الصفحة العامّة بالعنوان** — تخمينُ الرموز لا يُجرَّب مجّاناً.
func (s *Server) feedbackPublicAllowed(r *http.Request) bool {
	key := "feedbackreq:ip:" + clientIP(r)
	n, err := s.rdb.Incr(r.Context(), key).Result()
	if err != nil {
		return true // عطلُ الكاش لا يعطّل الصفحة
	}
	if n == 1 {
		s.rdb.Expire(r.Context(), key, time.Hour)
	}
	return n <= feedbackPublicPerHour
}

// handlePublicFeedbackRequest **ما تعرضه صفحةُ الرأي** — `GET /public/feedback-requests/{token}`.
// **رقمُ الطلب وهل له سائقٌ والحال لا غير.**
func (s *Server) handlePublicFeedbackRequest(w http.ResponseWriter, r *http.Request) {
	if !s.feedbackPublicAllowed(r) {
		s.respondErr(w, ErrInviteClaimRateLimited)
		return
	}
	token := chi.URLParam(r, "token")
	if len(token) < 16 || len(token) > 64 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var number int64
	var hasDriver, rated bool
	var answered *time.Time
	var expires time.Time
	err := s.pg.QueryRow(r.Context(), `
		SELECT o.number, o.driver_id IS NOT NULL,
		       EXISTS (SELECT 1 FROM order_ratings rt WHERE rt.order_id = o.id),
		       f.answered_at, f.expires_at
		  FROM order_feedback_requests f
		  JOIN orders o ON o.id = f.order_id
		 WHERE f.token_hash = $1`, locReqHash(token)).
		Scan(&number, &hasDriver, &rated, &answered, &expires)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	status := "pending"
	switch {
	case answered != nil || rated:
		status = "answered"
	case time.Now().After(expires):
		status = "expired"
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"order_number": number,
		"has_driver":   hasDriver,
		"status":       status,
	})
}

// handlePublicSubmitFeedback **الزبونُ يرسل رأيَه** — `POST /public/feedback-requests/{token}`.
// **مرّةً واحدة** — وما بعدها `feedback_request_closed`.
func (s *Server) handlePublicSubmitFeedback(w http.ResponseWriter, r *http.Request) {
	if !s.feedbackPublicAllowed(r) {
		s.respondErr(w, ErrInviteClaimRateLimited)
		return
	}
	token := chi.URLParam(r, "token")
	if len(token) < 16 || len(token) > 64 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	req, err := decode[struct {
		ServiceStars int    `json:"service_stars"`
		DriverStars  *int   `json:"driver_stars"`
		Feedback     string `json:"feedback"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **التحقّقُ قبل الحجز** — نجومٌ خاطئة لا تحرق الرابط.
	if req.ServiceStars < 1 || req.ServiceStars > 5 ||
		(req.DriverStars != nil && (*req.DriverStars < 1 || *req.DriverStars > 5)) {
		s.respondErr(w, orders.ErrBadStars)
		return
	}
	text := strings.TrimSpace(req.Feedback)
	if utf8.RuneCountInString(text) > feedbackMaxRunes {
		text = string([]rune(text)[:feedbackMaxRunes])
	}

	ctx := r.Context()
	hash := locReqHash(token)
	var orderID, customerID string
	err = s.pg.QueryRow(ctx, `
		UPDATE order_feedback_requests f
		   SET answered_at = now(), feedback = $2
		  FROM orders o
		 WHERE f.token_hash = $1 AND o.id = f.order_id
		   AND f.answered_at IS NULL AND f.expires_at > now()
		RETURNING f.order_id::text, o.customer_id::text`, hash, text).Scan(&orderID, &customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		var exists bool
		_ = s.pg.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM order_feedback_requests WHERE token_hash = $1)`, hash).Scan(&exists)
		if !exists {
			s.respondErr(w, httpx.ErrNotFound)
			return
		}
		s.respondErr(w, errFeedbackClosed)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}

	// **بمسار التطبيق نفسِه** — فلا يختلف ما يُحسب.
	if err := s.orders.RateOrder(ctx, customerID, nil, orderID, req.ServiceStars, req.DriverStars); err != nil {
		if errors.Is(err, orders.ErrAlreadyRated) {
			s.respondErr(w, errFeedbackClosed) // قيَّم من التطبيق قبله
			return
		}
		// **وما سواه يُعيد الرابطَ مفتوحاً** — لا يُحرق رابطٌ لم يُسجَّل تقييمُه.
		if _, e := s.pg.Exec(ctx, `UPDATE order_feedback_requests SET answered_at = NULL, feedback = ''
			WHERE order_id = $1`, orderID); e != nil {
			s.logger.Warn("طلبُ الرأي: فتحُ الرابط من جديد سقط", "order", orderID, "error", e)
		}
		s.respondErr(w, err)
		return
	}
	if text != "" {
		if _, err := s.pg.Exec(ctx,
			`UPDATE order_ratings SET app_feedback = $2 WHERE order_id = $1`, orderID, text); err != nil {
			// **النسخةُ في صفّ الطلب باقية** — لا تضيع الكلمة.
			s.logger.Warn("طلبُ الرأي: كتابةُ الكلمة في التقييم سقطت", "order", orderID, "error", err)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
