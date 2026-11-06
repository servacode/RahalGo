package server

// ══════════════════════════════════════════════════════════════════════
// **محطّاتُ الطلب في حمولة السائق وأبوابه** (طلبٌ بعدّة محطّات، الدفعة P4)
// ══════════════════════════════════════════════════════════════════════
//
// `stops[]` و`next_stop` في كلّ حمولةٍ تبلغ السائق (القائمة · الطابور · ردُّ القبول والخطوة) —
// **وغائبةٌ في طلب المتجر الواحد فحمولتُه حرفاً كما كانت.** **والحقولُ المسطّحةُ** (`merchant_name`
// · `merchant_phone` · `pickup_address` · `nav_lat/lng` · `to_pickup_m`) **هي المحطّةُ التالية.**
//
// **و«وصلت»/«استلمت» بلا محطّةٍ في طلبٍ بعدّة محطّات ⇒ ٤٢٦** (G3) — والقبولُ كذلك لنسخةٍ لا
// تعرف المحطّات.

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/orders"
	"github.com/servacode/rahalgo/backend/internal/shifts"
)

// attachDriverStops **يُلحق بكلّ طلبٍ بعدّة محطّات محطّاتِه كما يراها هذا السائق** — وأصنافَ كلّ
// محطّةٍ بشكل `driverItem` من أصناف الطلب المقروءة سلفاً.
func (s *Server) attachDriverStops(ctx context.Context, driverID string, list []driverOrder) error {
	ids := make([]string, 0, len(list))
	for i := range list {
		if list[i].Kind == "standard" && list[i].ReturnTo == "" {
			ids = append(ids, list[i].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	stops, next, err := s.orders.DriverStops(ctx, s.pg, ids, driverID, staleLocationMinutes)
	if err != nil {
		return err
	}
	for i := range list {
		o := &list[i]
		st, ok := stops[o.ID]
		if !ok || len(st) == 0 {
			continue
		}
		byID := make(map[string]driverItem, len(o.Items))
		for _, it := range o.Items {
			byID[it.ID] = it
		}
		for j := range st {
			items := []any{}
			for _, id := range st[j].ItemIDs {
				if it, ok := byID[id]; ok {
					items = append(items, it)
				}
			}
			st[j].Items = items
		}
		o.Stops = st
		// **وبضاعةٌ من متجرين لا تعود لمتجر** (قرارُ المالك ٤أ).
		o.AcceptsReturns = false
		n, ok := next[o.ID]
		if !ok {
			continue
		}
		o.NextStop = &n
		for _, d := range st {
			if !strings.EqualFold(d.MerchantID, n) {
				continue
			}
			o.MerchantName = d.Name
			o.PickupAddress = d.Address
			o.MerchantPhone = nil
			if d.Phone != "" {
				phone := d.Phone
				o.MerchantPhone = &phone
			}
			o.NavLat, o.NavLng = d.Lat, d.Lng
			o.ToPickupM = d.ToM
		}
	}
	return nil
}

// driverReplyView **ردُّ خطوة السائق أو قبوله** — حمولةُ السائق للطلب، **ومعها محطّاتُه** للطلب
// بعدّة محطّات (العقدُ §1). **وطلبُ المتجر الواحد كما كان حرفاً.**
func (s *Server) driverReplyView(r *http.Request, o *orders.Order) map[string]any {
	view := orders.ViewFor(orders.AudienceDriver, o)
	if o == nil || o.Stops <= 1 || view == nil {
		return view
	}
	list, err := s.loadDriverOrders(r, driverOrderSelect+` WHERE o.id = $3::uuid`,
		userIDFrom(r), staleLocationMinutes, o.ID)
	if err != nil || len(list) == 0 || len(list[0].Stops) == 0 {
		return view
	}
	view["stops"] = list[0].Stops
	if list[0].NextStop != nil {
		view["next_stop"] = *list[0].NextStop
	} else {
		view["next_stop"] = nil
	}
	return view
}

// driverBuildBelowStops **نسخةُ السائق دون نسخة المحطّات** (حزامُ القبول، §5.5) — **ومن لا يرسل
// نسخةً** (لوحةٌ أو أداةُ فحص) **لا يُحبس.**
func (s *Server) driverBuildBelowStops(r *http.Request) bool {
	kind, have := requestAppBuild(r)
	if kind != "driver" || s.settings == nil {
		return false
	}
	build := s.settings.GetInt(r.Context(), multiStopBuildKey)
	return build > 0 && int64(have) < build
}

// releaseWithGoods **«لدي مشكلة: حادث» والسائقُ يحمل بضاعةَ محطّة** (§5.3) — **طارئٌ بعد الاستلام
// لا تركٌ إلى الطابور**: يُسجَّل في غرفة الطوارئ بمرحلة «بعد الاستلام»، **ونقطةُ الاستلام البديلة
// موضعُه** (أو كلمتُها إن لم يُعرف)، ويُستثنى من الطلب ويُغلق دوامُه ويُقفل حتّى «السائقُ بخير».
// **ولا انتقال**: الطلبُ يبقى حيث هو والمكتبُ يقرّر — سائقٌ آخرُ يستلم منه، أو شيلُ الباقي.
func (s *Server) releaseWithGoods(w http.ResponseWriter, r *http.Request, orderID string, number int64, note string) {
	ctx := r.Context()
	uid := userIDFrom(r)
	if _, err := shifts.Close(ctx, s.pg, uid, shifts.ReasonEmergency); err != nil {
		s.logger.Error("الترك: تعذّر إغلاق الدوام", "driver", uid, "error", err)
	}
	s.lockDriverAfterAccident(ctx, uid)
	var emergencyID string
	err := s.pg.QueryRow(ctx, `
		INSERT INTO driver_emergencies (driver_id, order_id, at, note, kind, stage)
		SELECT $1, $2, u.last_location, $3, $4, 'after_pickup' FROM users u WHERE u.id = $1
		ON CONFLICT (order_id) WHERE status = 'open' DO NOTHING
		RETURNING id::text`, uid, orderID, clip(note, 500), EmergencyAccident).Scan(&emergencyID)
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.pg.QueryRow(ctx, `SELECT id::text FROM driver_emergencies
			WHERE order_id = $1 AND status = 'open' ORDER BY created_at LIMIT 1`, orderID).Scan(&emergencyID)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **والبضاعةُ معه حيث وقف** — فلا يذهب الثاني إلى متجرٍ سلّم بضاعتَه.
	if _, err := s.pg.Exec(ctx, `
		UPDATE orders o
		   SET pickup_override = COALESCE(u.last_location, o.pickup_override),
		       pickup_override_note = CASE WHEN u.last_location IS NOT NULL
		           THEN 'استلامٌ من موضع طارئ — البضاعةُ مع السائق'
		           ELSE 'البضاعةُ مع سائقٍ سابقٍ وقع له طارئ — لا تذهب إلى المتجر، اتّصل بالعمليات' END
		  FROM (SELECT (SELECT last_location FROM users WHERE id = $2::uuid
		                 AND last_location_at > now() - interval '15 minutes') AS last_location) u
		 WHERE o.id = $1`, orderID, uid); err != nil {
		s.respondErr(w, err)
		return
	}
	if err := s.orders.ExcludeDriverTx(ctx, s.pg, orderID, uid); err != nil {
		s.logger.Error("الترك: تعذّر استثناءُ السائق", "order", orderID, "error", err)
	}
	others := s.emergencyOtherOrders(ctx, uid, orderID)
	body := "#" + strconv.FormatInt(number, 10) + " — " + releaseReasonLabels["accident"] + " · " + note +
		" · البضاعةُ معه من محطّة — لم يُحرَّر"
	for _, o := range others {
		if o.Released {
			body += " · #" + strconv.FormatInt(o.Number, 10) + " حُرّر إلى الطابور"
		} else {
			body += " · #" + strconv.FormatInt(o.Number, 10) + " البضاعةُ معه — لم يُحرَّر، اتّصل به"
		}
	}
	s.notify.NotifyCaps(ctx, orders.EmergencyAlertCaps, notifications.Input{
		Kind: notifications.KindOrder, Title: notifTitles.driverEmergency, Body: body,
		Entity: "order", EntityID: orderID, Href: "/dashboard/emergencies/" + emergencyID,
	})
	s.notifyCustomerEmergency(ctx, orderID, emergencyCustomerFollowing)
	for _, o := range others {
		if o.Released {
			s.notifyCustomerEmergency(ctx, o.ID, emergencyCustomerRedispatch)
		}
	}
	s.audit(r, "driver.emergency", "order", orderID, map[string]any{
		"emergency_id": emergencyID, "released": false, "kind": EmergencyAccident, "via": "release",
		"stops_in_hand": true,
	})
	s.touch("emergency", "ops")
	s.touch("order", "ops")
	s.touch("driver", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{
		"released": false, "shift_ended": true, "emergency_id": emergencyID,
	})
}
