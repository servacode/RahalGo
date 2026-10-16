package server

// ══════════════════════════════════════════════════════════════════════
// **الزبونُ يشارك موقعَه المباشرَ مع الكابتن** (قرارُ المالك ٢٠٢٦-١٠-١٦)
// ══════════════════════════════════════════════════════════════════════
//
//	POST   /api/v1/orders/{id}/live-location   {lat, lng, accuracy_m}
//	DELETE /api/v1/orders/{id}/live-location
//
// **صاحبُ الطلب وحدَه يكتب** — وطلبُ غيره «غيرُ موجود» (٤٠٤) لا «ممنوع»: **من
// جرّب معرّفاتٍ لا يعرف أيُّها حقيقيّ.** **والطلبُ في طريقه وحدَه** (`LiveStatuses`).
//
// # والسائقُ يُنبَّه بلا حمولة
//
// **إشارةُ «تغيّر شيء» إلى غرفته** (`customer_live`) — **وموقعُ الزبون لا يُبثّ**:
// السائقُ يقرؤه من بابه هو (`/driver/orders`) حيث يُشترط أنّ الطلبَ طلبُه.
// **وبثٌّ يحمل الموقعَ قناةٌ ثانيةٌ يلزمها عقدٌ ثانٍ.**

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/settings"
)

// liveMaxPerMinute **سقفُ ما يُقبل من زبونٍ لطلبٍ في الدقيقة.**
//
// **والتطبيقُ يرسل كلَّ عشر ثوانٍ** — ستٌّ في الدقيقة. **وعشرون تتّسع لإعادةِ
// ما سقط على شبكةٍ سيّئة**، **وما فوقها عميلٌ معطوبٌ أو معدَّلٌ يُتعب القاعدة.**
const liveMaxPerMinute = 20

// liveAllowed **حدُّ المعدّل بالزبون والطلب** — **وعطلُ الكاش لا يعطّل الميزة.**
func (s *Server) liveAllowed(ctx context.Context, uid, orderID string) bool {
	key := "live:rl:" + uid + ":" + orderID + ":" + time.Now().UTC().Format("200601021504")
	n, err := s.incr(ctx, key)
	if err != nil {
		return true
	}
	if n == 1 && s.rdb != nil {
		s.rdb.Expire(ctx, key, 2*time.Minute)
	}
	return n <= liveMaxPerMinute
}

// liveSetting **رقمٌ من اللوحة** — وافتراضُه إن غاب المخزن.
func (s *Server) liveSetting(ctx context.Context, key string) int64 {
	if s.settings == nil {
		return settings.Default(key)
	}
	return s.settings.GetInt(ctx, key)
}

// liveRule **حدّا تحويل الطريق إلى الزبون** — البُعدُ عن الدبّوس وأسوأُ دقّةٍ مقبولة.
func (s *Server) liveRule(ctx context.Context) orders.LiveRule {
	return orders.LiveRule{
		SwitchM:      s.liveSetting(ctx, orders.SettingLiveSwitchM),
		MaxAccuracyM: s.liveSetting(ctx, orders.SettingLiveMaxAccuracyM),
	}
}

// notifyCustomerLive **إشارةٌ إلى غرفة السائق** — بلا موقعٍ ولا حمولة طلب.
func (s *Server) notifyCustomerLive(driverID, orderID string) {
	if driverID == "" || s.hub == nil {
		return
	}
	s.hub.Publish("driver:"+driverID, map[string]any{"type": "customer_live", "order_id": orderID})
}

// handleCustomerLiveLocation **نقطةٌ من هاتف الزبون** — آخرُها وحدَه يُحفظ.
func (s *Server) handleCustomerLiveLocation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	uid := userIDFrom(r)
	req, err := decode[struct {
		Lat       float64  `json:"lat"`
		Lng       float64  `json:"lng"`
		AccuracyM *float64 `json:"accuracy_m"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **وصفرٌ صفرٌ ليس موضعاً** — جهازٌ بلا إشارة (`driver_location.go`).
	if (req.Lat == 0 && req.Lng == 0) || req.Lat < -90 || req.Lat > 90 ||
		req.Lng < -180 || req.Lng > 180 {
		s.respondErr(w, errValidation)
		return
	}
	// **ودقّةٌ سالبةٌ أو غيرُ معقولةٍ لا تُكتب رقماً** — تُحذف ويبقى الموضع.
	if req.AccuracyM != nil && (*req.AccuracyM < 0 || *req.AccuracyM > 100000) {
		req.AccuracyM = nil
	}
	if !s.liveAllowed(r.Context(), uid, orderID) {
		s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
		return
	}
	driverID, err := s.orders.SaveLive(r.Context(), orderID, uid, req.Lat, req.Lng, req.AccuracyM)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notifyCustomerLive(driverID, orderID)
	// **والحدُّ يعود إلى هاتف الزبون** (قرارُ المالك ٢٠٢٦-١٠-١٦) — به يرشّح قراءاتِه
	// (`LiveSendFilter`)، **فلا يُكتب رقمٌ في التطبيق ورقمٌ في اللوحة.**
	httpx.JSON(w, http.StatusOK, map[string]any{
		"saved":          true,
		"max_accuracy_m": s.liveSetting(r.Context(), orders.SettingLiveMaxAccuracyM),
	})
}

// handleStopCustomerLiveLocation **«وقّف المشاركة»** — تُمحى النقطةُ فيعود السائقُ إلى الدبّوس.
func (s *Server) handleStopCustomerLiveLocation(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	driverID, err := s.orders.StopLive(r.Context(), orderID, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.notifyCustomerLive(driverID, orderID)
	httpx.JSON(w, http.StatusOK, map[string]any{"stopped": true})
}

// attachDriverLive **يُلحق موقعَ الزبون الحديثَ بطلبات السائق** — طلباتِه هو وحدَها.
//
// **وتعثّرُه لا يُسقط القائمة**: الموقعُ ترفٌ فوق الدبّوس، **وقائمةٌ تسقط لأنّ
// نقطةً لم تُقرأ تُعمي السائقَ عن طلبه كلِّه.**
func (s *Server) attachDriverLive(ctx context.Context, driverID string, list []driverOrder) {
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].ReturnTo == "" {
			ids = append(ids, list[i].ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	got, err := s.orders.LiveFor(ctx, ids, driverID, s.liveRule(ctx))
	if err != nil {
		s.logger.Warn("موقعُ الزبون: تعذّرت القراءة", "driver", driverID, "error", err)
		return
	}
	for i := range list {
		if p, ok := got[list[i].ID]; ok && list[i].ReturnTo == "" {
			list[i].CustomerLive = p
		}
	}
}

// liveRouteTarget **أيُقاد السائقُ إلى الزبون لا إلى الدبّوس؟** — ويردّ النقطةَ إن نعم.
//
// **والحكمُ واحد** (`LivePoint.Switch`): بعدُه عن الدبّوس فوق الحدّ، **ودقيقٌ، وحديثٌ لا شائخ.**
func (s *Server) liveRouteTarget(ctx context.Context, orderID, driverID string) *orders.LivePoint {
	got, err := s.orders.LiveFor(ctx, []string{orderID}, driverID, s.liveRule(ctx))
	if err != nil {
		s.logger.Warn("موقعُ الزبون: تعذّرت قراءةُ الوجهة", "order", orderID, "error", err)
		return nil
	}
	if p := got[orderID]; p.Switch() {
		return p
	}
	return nil
}
