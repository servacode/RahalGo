package server

// ══════════════════════════════════════════════════════════════════════
// **طلباتُ الواتساب — أبوابُ اللوحة** (قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «عندي طلبات خارجية على الواتس اب، بدي خيار باللوحة… وينزل طلب نظامي ع التطبيق».
// **والأبوابُ رقيقةٌ والمنطقُ في المحرّك** (`orders/whatsapp_order.go`): الحرّاسُ والأجرةُ
// والإنشاءُ في الطابور — **والتدقيقُ في معاملة الإنشاء نفسِها.** وكلُّها بقدرة `orders.whatsapp`
// (جدولُ السياسة).
//
//	GET  /whatsapp-orders                 آخرُ طلبات الواتساب (و`?open=1&q=<رقم>` للتلميح)
//	GET  /whatsapp-orders/stores?q=       بحثٌ في متاجرنا الفعّالة ذات الموقع
//	GET  /whatsapp-orders/quote           الأجرةُ قبل الإرسال
//	POST /whatsapp-orders/resolve-location رابطُ موقعٍ ← نقطة
//	POST /whatsapp-orders                 الإنشاء — مرّةً واحدةً بمفتاح المحاولة
//	POST /whatsapp-orders/{id}/cancel     الإلغاءُ قبل الاستلام بسببٍ إلزاميّ
//	POST /whatsapp-orders/{id}/address    تصحيحُ العنوان المكتوب قبل التسليم (ملحق ٢٠٢٦-١١-٠٥، د)
//	POST /whatsapp-orders/location-requests      «اطلب الموقع» من رقم المستلِم (ملحق ٢٠٢٦-١١-٠٥، أ)
//	GET  /whatsapp-orders/location-requests/{id} أوصل الموقع؟ — النموذجُ يسأل كلَّ ثوانٍ

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/dbtx"
	"github.com/servacode/rahalgo/backend/internal/geolink"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/textguard"
)

// رموزُ طلب الواتساب — **للوحة وحدَها** (`SERVER_ADMIN` في `web/scripts/check-app-error-codes.mjs`).
var waErrors = []struct {
	sentinel error
	reply    error
}{
	{orders.ErrWAStoreRequired, httpx.NewError(http.StatusBadRequest,
		"wa_store_required", "errors.wa_store_required")},
	{orders.ErrWAStoreConflict, httpx.NewError(http.StatusBadRequest,
		"wa_store_conflict", "errors.wa_store_conflict")},
	{orders.ErrWAStoreUnavailable, httpx.NewError(http.StatusUnprocessableEntity,
		"wa_store_unavailable", "errors.wa_store_unavailable")},
	{orders.ErrWAFeeBelowShare, httpx.NewError(http.StatusUnprocessableEntity,
		"wa_fee_below_share", "errors.wa_fee_below_share")},
	{orders.ErrWANotWhatsApp, httpx.NewError(http.StatusConflict,
		"wa_not_whatsapp", "errors.wa_not_whatsapp")},
	{orders.ErrWACancelAfterPickup, httpx.NewError(http.StatusConflict,
		"wa_cancel_after_pickup", "errors.wa_cancel_after_pickup")},
	{orders.ErrWAFeeRequired, httpx.NewError(http.StatusUnprocessableEntity,
		"wa_fee_required", "errors.wa_fee_required")},
	{orders.ErrWAAddressClosed, httpx.NewError(http.StatusConflict,
		"wa_address_closed", "errors.wa_address_closed")},
}

// waReply **علامةُ المحرّك ← ردُّ اللوحة** — وحدّا الأجرة في التفصيل نصّاً.
func waReply(err error) error {
	var rng *orders.WAFeeRangeError
	if errors.As(err, &rng) {
		e := httpx.NewError(http.StatusUnprocessableEntity, "wa_fee_out_of_range", "errors.wa_fee_out_of_range")
		e.Details = map[string]any{"min": strconv.FormatInt(rng.Min, 10), "max": strconv.FormatInt(rng.Max, 10)}
		return e
	}
	for _, e := range waErrors {
		if errors.Is(err, e.sentinel) {
			return e.reply
		}
	}
	return err
}

// handleListWhatsAppOrders `GET /admin/whatsapp-orders` — آخرُ عشرين، **و`?open=1&q=<رقم>`
// يحصرها في الجارية لرقم المستلِم** (تلميحُ النموذج).
func (s *Server) handleListWhatsAppOrders(w http.ResponseWriter, r *http.Request) {
	phone := ""
	if raw := strings.TrimSpace(r.URL.Query().Get("q")); raw != "" {
		p, ok := identity.NormalizePhone(raw)
		if !ok {
			httpx.JSON(w, http.StatusOK, map[string]any{"orders": []any{}})
			return
		}
		phone = p
	}
	list, err := s.orders.WhatsAppOrders(r.Context(), phone, r.URL.Query().Get("open") == "1", 20)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	views, err := orderViews(orders.AudienceOps, list)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": views})
}

// handleWhatsAppStores `GET /admin/whatsapp-orders/stores?q=` — متاجرُنا الفعّالةُ ذاتُ الموقع.
func (s *Server) handleWhatsAppStores(w http.ResponseWriter, r *http.Request) {
	list, err := s.orders.WhatsAppStores(r.Context(), r.URL.Query().Get("q"), 20)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stores": list})
}

// waFloat **إحداثيّةٌ اختياريّة من الرابط** — فارغُها nil، وفاسدُها خطأ.
func waFloat(r *http.Request, key string) (*float64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, errValidation
	}
	return &v, nil
}

// handleWhatsAppQuote `GET /admin/whatsapp-orders/quote` — `store_id` أو `pickup_lat/pickup_lng`،
// مع `lat/lng` المستلِم. **بدالّة الإنشاء نفسِها** — فلا يُعرض رقمٌ ويُكتب غيرُه.
//
// **والمنصّةُ في دوامها وغيرُ موقوفة** — بوّابةُ كلِّ طلب (سؤالُ المالك ٤، الافتراض).
func (s *Server) handleWhatsAppQuote(w http.ResponseWriter, r *http.Request) {
	if !s.requireOrdering(w, r) {
		return
	}
	lat, err1 := waFloat(r, "lat")
	lng, err2 := waFloat(r, "lng")
	slat, err3 := waFloat(r, "pickup_lat")
	slng, err4 := waFloat(r, "pickup_lng")
	// **وبلا `lat/lng` للمستلِم عرضٌ بلا موقع** (ملحقُ المالك ٢٠٢٦-١١-٠٥، ب) — الأساسُ اقتراحاً
	// والحدّان، **والأجرةُ يكتبها الموظّف** (`fee_required`).
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || (lat == nil) != (lng == nil) {
		s.respondErr(w, errValidation)
		return
	}
	q, err := s.orders.QuoteWhatsAppOrder(r.Context(), r.URL.Query().Get("store_id"), slat, slng, lat, lng)
	if err != nil {
		s.respondErr(w, waReply(err))
		return
	}
	httpx.JSON(w, http.StatusOK, q)
}

// handleWhatsAppResolveLocation `POST /admin/whatsapp-orders/resolve-location` — رابطُ موقعٍ من
// الواتساب أو خرائط غوغل ← نقطة (`geolink`). **ردٌّ صريحٌ لا خطأ** إن لم يكن فيه موقع.
func (s *Server) handleWhatsAppResolveLocation(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		URL string `json:"url"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if len(req.URL) > 2000 {
		s.respondErr(w, errValidation)
		return
	}
	lat, lng, err := geolink.Resolve(r.Context(), nil, req.URL)
	if err != nil {
		if !errors.Is(err, geolink.ErrNoLocation) {
			s.logger.Warn("طلب واتساب: رابطُ الموقع لم يُقرأ", "error", err)
		}
		httpx.JSON(w, http.StatusOK, map[string]any{"found": false})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"found": true, "lat": lat, "lng": lng})
}

// waStorePhone **رقمُ المتجر كما كُتب** — رقمٌ صالحٌ يُطبَّع، **وهاتفٌ أرضيٌّ يبقى أرقاماً**
// (المتجرُ قد لا يملك جوّالاً). وما سوى الأرقام يُردّ.
func waStorePhone(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", true
	}
	if p, ok := identity.NormalizePhone(raw); ok {
		return p, true
	}
	digits := 0
	for _, c := range raw {
		switch {
		case c >= '0' && c <= '9':
			digits++
		case c == '+' || c == ' ' || c == '-':
		default:
			return "", false
		}
	}
	return raw, digits >= 6 && digits <= 15
}

// handleCreateWhatsAppOrder `POST /admin/whatsapp-orders` — **مرّةً واحدةً** (`s.idempotent`).
//
// **والإنشاءُ والتدقيقُ وعلامةُ منع التكرار في معاملةٍ واحدة** (`WithIdempotentTx`): طلبٌ أُنشئ
// وضاع ردُّه فأُعيد لا يُنشأ ثانيةً، **ولا طلبٌ بلا أثرِ من كتبه** (الأجرةُ المحسوبةُ والمكتوبة).
// **والعرضُ على السائقين بعد التثبيت** (`AfterWhatsAppOrder`).
func (s *Server) handleCreateWhatsAppOrder(w http.ResponseWriter, r *http.Request) {
	if !s.requireOrdering(w, r) {
		return
	}
	req, err := decode[struct {
		// StoreID **متجرٌ من متاجرنا** — أو الأربعةُ التالية لمتجرٍ غير تابع.
		StoreID      string   `json:"store_id"`
		StoreName    string   `json:"store_name"`
		StorePhone   string   `json:"store_phone"`
		StoreAddress string   `json:"store_address"`
		StoreLat     *float64 `json:"store_lat"`
		StoreLng     *float64 `json:"store_lng"`
		Details      string   `json:"details"`
		// GoodsAmount **ثمنُ البضاعة إن عُرف** — رقمٌ عشريٌّ للدولار (١٢٫٥ ⇒ ١٢٥٠ سنتاً).
		GoodsAmount    *float64 `json:"goods_amount"`
		Currency       string   `json:"amount_currency"`
		RecipientName  string   `json:"recipient_name"`
		RecipientPhone string   `json:"recipient_phone"`
		AddressText    string   `json:"address_text"`
		Lat            *float64 `json:"lat"`
		Lng            *float64 `json:"lng"`
		DriverNote     string   `json:"driver_note"`
		// Fee **أجرةٌ عدّلها الموظّف** — وغيابُها: المحسوبة. **وبلا `lat/lng` إلزاميّة.**
		Fee *int64 `json:"fee"`
		// LocationRequestID **«اطلب الموقع» من النموذج** (ملحقُ المالك ٢٠٢٦-١١-٠٥، أ) — يُربط بالطلب:
		// وصل الموقعُ قبلها؟ نقطتُه نقطةُ التسليم إن لم تُحدَّد. **وإن وصل بعدها كُتب حين يصل.**
		LocationRequestID string `json:"location_request_id"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ونصوصُه تمرّ بالحارس** — انظر `text_limits.go`.
	if _, err := s.guardText(r.Context(),
		tf("store_name", &req.StoreName, maxStoreName, textguard.Name),
		tf("store_address", &req.StoreAddress, maxAddressText, textguard.Address),
		tf("details", &req.Details, orders.MaxWhatsAppDetails, textguard.Notes),
		tf("recipient_name", &req.RecipientName, maxPersonName, textguard.Name),
		tf("address_text", &req.AddressText, maxAddressText, textguard.Address),
		tf("driver_note", &req.DriverNote, maxOrderNotes, textguard.Notes),
	); err != nil {
		s.respondErr(w, err)
		return
	}
	// **ورقمُ المستلِم يُطبَّع كرقم أيّ حساب** — يتّصل به السائقُ ويصله كودُ التسليم على واتسابه.
	phone, ok := identity.NormalizePhone(req.RecipientPhone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	storePhone, ok := waStorePhone(req.StorePhone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	in := orders.WhatsAppOrderInput{
		StoreID: req.StoreID, StoreName: req.StoreName, StorePhone: storePhone,
		StoreAddress: req.StoreAddress, StoreLat: req.StoreLat, StoreLng: req.StoreLng,
		Details: req.Details, Currency: req.Currency,
		RecipientName: req.RecipientName, RecipientPhone: phone, AddressText: req.AddressText,
		Lat: req.Lat, Lng: req.Lng, DriverNote: req.DriverNote, Fee: req.Fee,
	}
	if req.GoodsAmount != nil {
		v, ok := orders.AmountUnits(*req.GoodsAmount, req.Currency)
		if !ok {
			s.respondErr(w, errValidation)
			return
		}
		in.GoodsAmount = &v
	}
	actor := userIDFrom(r)
	locReqID := strings.TrimSpace(req.LocationRequestID)
	s.WithIdempotentTx(w, r, func(ctx context.Context, q dbtx.Querier) (IdempotentBody, error) {
		in := in
		// **طلبُ الموقع يُقفَل قبل الإنشاء** — فموقعٌ يصل في اللحظة نفسِها ينتظر الربطَ ثمّ يُكتب
		// في الطلب (`ApplyWhatsAppShared`)، **ولا يضيع بينهما.**
		if locReqID != "" {
			lat, lng, err := lockWhatsAppLocReqTx(ctx, q, locReqID)
			if err != nil {
				return IdempotentBody{}, err
			}
			if in.Lat == nil && in.Lng == nil && lat != nil && lng != nil {
				in.Lat, in.Lng = lat, lng
			}
		}
		c, err := s.orders.CreateWhatsAppOrderIn(ctx, q, actor, in)
		if err != nil {
			return IdempotentBody{}, waReply(err)
		}
		if locReqID != "" {
			if _, err := q.Exec(ctx,
				`UPDATE recipient_location_requests SET order_id = $2 WHERE id = $1`, locReqID, c.ID); err != nil {
				return IdempotentBody{}, err
			}
		}
		if err := s.auditTx(ctx, q, r, "ops.whatsapp_order_create", "order", c.ID, map[string]any{
			"number":             c.Number,
			"store_name":         c.StoreName,
			"pickup_merchant_id": c.PickupMerchantID,
			"computed_fee":       c.ComputedFee,
			"fee":                c.Fee,
			"driver_fee":         c.DriverFee,
			"goods_amount":       in.GoodsAmount,
			"amount_currency":    in.Currency,
			"has_point":          c.HasPoint,
			"location_request":   locReqID,
		}); err != nil {
			return IdempotentBody{}, err
		}
		id := c.ID
		return IdempotentBody{
			Status:  http.StatusCreated,
			Payload: map[string]any{"id": id, "number": c.Number},
			AfterCommit: func() {
				if _, err := s.orders.AfterWhatsAppOrder(context.WithoutCancel(r.Context()), id); err != nil {
					s.logger.Error("طلب واتساب: بعد الإنشاء", "order", id, "error", err)
				}
				s.touch("orders", "ops")
			},
		}, nil
	})
}

// handleCancelWhatsAppOrder `POST /admin/whatsapp-orders/{id}/cancel` — `{"reason": "…"}`.
//
// **لمن يملك `orders.whatsapp`** — والآلةُ تحكم كما في بابها العامّ (قبل الاستلام وحدَه)، **والتدقيقُ
// في معاملة الانتقال.**
func (s *Server) handleCancelWhatsAppOrder(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Reason string `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.guardText(r.Context(), tf("reason", &req.Reason, maxOrderNotes, textguard.Notes)); err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	o, err := s.orders.CancelWhatsAppOrder(r.Context(), userIDFrom(r), orderID, req.Reason,
		func(ctx context.Context, q dbtx.Querier) error {
			return s.auditTx(ctx, q, r, "ops.whatsapp_order_cancel", "order", orderID, map[string]any{
				"reason": strings.TrimSpace(req.Reason),
			})
		})
	if err != nil {
		s.respondErr(w, waReply(err))
		return
	}
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}

// ══════════════════════════════════════════════════════════════════════
// **«اطلب الموقع» من نموذج الواتساب، وتصحيحُ العنوان** (ملحقُ المالك ٢٠٢٦-١١-٠٥)
// ══════════════════════════════════════════════════════════════════════
//
// «من لوحة التحكم مثل توصيلة متجر وامانة نحط الرقم صاحب الطلب ويشارك موقعه ويجي لعندنا».
// **الصفُّ نفسُه** (`recipient_location_requests`، هجرة ٠٤٨٤ ثمّ ٠٤٩٠ ثمّ ٠٥٠٧) **بنوع `whatsapp`**:
// صاحبُه المكتبُ (`requested_by`) لا متجرٌ ولا زبون، **والرسالةُ الواحدةُ نفسُها** (`locReqText`)،
// **والردُّ بالواتساب والرابطُ يملآنه كما يملآن غيرَه** (`locReqReceived`).

// locReqModeWhatsApp **طلبُ موقعٍ من المكتب لطلب واتساب.**
const locReqModeWhatsApp = "whatsapp"

// waLocReqPerHour **حدُّ الموظّف في الساعة** — المكتبُ يكتب طلباتٍ أكثرَ من متجر، **ورسائلُ البوت
// ليست مجّانيّةً على الرقم.**
const waLocReqPerHour = 60

// lockWhatsAppLocReqTx **يقفل طلبَ موقعٍ للمكتب غيرَ مربوطٍ بطلب** — ويردّ نقطتَه إن وصلت.
func lockWhatsAppLocReqTx(ctx context.Context, q dbtx.Querier, id string) (lat, lng *float64, err error) {
	if !isUUID(id) {
		return nil, nil, errAmanahLocReq
	}
	err = q.QueryRow(ctx, `
		SELECT lat, lng FROM recipient_location_requests
		 WHERE id = $1 AND mode = 'whatsapp' AND order_id IS NULL
		 FOR UPDATE`, id).Scan(&lat, &lng)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, errAmanahLocReq
	}
	return lat, lng, err
}

// handleWhatsAppRequestLocation `POST /admin/whatsapp-orders/location-requests` —
// `{"phone": "…", "store_name": "…"}`. **والرسالةُ الواحدةُ نفسُها** لكلّ طلب موقع.
//
// **والحدّان قبل الكتابة** — دقيقةٌ بين طلبين للرقم نفسِه من المكتب، وستّون في الساعة للموظّف.
func (s *Server) handleWhatsAppRequestLocation(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Phone     string `json:"phone"`
		StoreName string `json:"store_name"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.guardText(r.Context(), tf("store_name", &req.StoreName, maxStoreName, textguard.Name)); err != nil {
		s.respondErr(w, err)
		return
	}
	phone, ok := identity.NormalizePhone(req.Phone)
	if !ok {
		s.respondErr(w, identity.ErrInvalidPhone)
		return
	}
	ctx := r.Context()
	actor := userIDFrom(r)
	var recent, hourly int
	if err := s.pg.QueryRow(ctx, `
		SELECT count(*) FILTER (WHERE phone = $2 AND created_at > now() - make_interval(secs => $3)),
		       count(*) FILTER (WHERE requested_by = NULLIF($1, '')::uuid)
		  FROM recipient_location_requests
		 WHERE mode = 'whatsapp' AND created_at > now() - interval '1 hour'`,
		actor, phone, int(locReqGap.Seconds())).Scan(&recent, &hourly); err != nil {
		s.respondErr(w, err)
		return
	}
	if recent > 0 {
		e := *errLocReqTooSoon
		e.Details = map[string]any{"retry_after_sec": int(locReqGap.Seconds())}
		s.respondErr(w, &e)
		return
	}
	if hourly >= waLocReqPerHour {
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
		       (requested_by, phone, mode, store_label, token_hash, expires_at)
		VALUES (NULLIF($1, '')::uuid, $2, 'whatsapp', NULLIF($3, ''), $4, now() + make_interval(secs => $5))
		RETURNING id::text, expires_at`,
		actor, phone, strings.TrimSpace(req.StoreName), locReqHash(token), int(locReqTTL.Seconds())).
		Scan(&id, &expires); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, s.locReqSend(ctx, id, phone, token, expires))
}

// handleWhatsAppLocationRequest `GET /admin/whatsapp-orders/location-requests/{id}` — **أوصل الموقع؟**
// والنموذجُ يسأله كلَّ ثوانٍ ما دام ينتظر. **وطلباتُ المكتب وحدَها** (`mode='whatsapp'`).
func (s *Server) handleWhatsAppLocationRequest(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !isUUID(id) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	var lat, lng, acc *float64
	var received, followUp *time.Time
	var expires time.Time
	var sent bool
	var orderID string
	err := s.pg.QueryRow(r.Context(), `
		SELECT lat, lng, accuracy_m, received_at, expires_at, whatsapp_sent, followup_sent_at,
		       COALESCE(order_id::text, '')
		  FROM recipient_location_requests WHERE id = $1 AND mode = 'whatsapp'`, id).
		Scan(&lat, &lng, &acc, &received, &expires, &sent, &followUp, &orderID)
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
		// followup_sent **أبعث البوتُ «اكتبلنا عنوانك بالتفصيل»** (ملحقُ المالك ٢٠٢٦-١١-٠٥، د).
		"followup_sent": followUp != nil,
		"order_id":      orderID,
	}
	if received != nil {
		out["received_at"] = received.Format(time.RFC3339)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleWhatsAppOrderAddress `POST /admin/whatsapp-orders/{id}/address` — `{"address_text": "…"}`.
//
// **المكتبُ يكتب العنوانَ التفصيليّ** الذي ردّ به المستلِمُ على الواتساب (ملحقُ المالك ٢٠٢٦-١١-٠٥،
// د: «نحن رح نعبيه يدوي») — قبل التسليم، **بتدقيقٍ فيه القديمُ والجديد**، ويصل السائقَ في لحظته
// **بلا رقم المستلِم.**
func (s *Server) handleWhatsAppOrderAddress(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		AddressText string `json:"address_text"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if _, err := s.guardText(r.Context(), tf("address_text", &req.AddressText, maxAddressText, textguard.Address)); err != nil {
		s.respondErr(w, err)
		return
	}
	orderID := chi.URLParam(r, "id")
	address := strings.TrimSpace(req.AddressText)
	o, err := s.orders.UpdateWhatsAppAddress(r.Context(), orderID, address,
		func(ctx context.Context, q dbtx.Querier, old string) error {
			return s.auditTx(ctx, q, r, "ops.whatsapp_order_address", "order", orderID, map[string]any{
				"old": old, "new": address,
			})
		})
	if err != nil {
		s.respondErr(w, waReply(err))
		return
	}
	s.touch("order", "ops")
	view, err := orderView(orders.AudienceOps, o)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"order": view})
}
