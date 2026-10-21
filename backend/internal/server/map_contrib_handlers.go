package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/notifications"
	"github.com/servacode/rahalgo/backend/internal/routing"
)

// ══════════════════════════════════════════════════════════════════════
// **ما يضيفه السائقون للخريطة** (قرارات المالك ٢٠٢٦-١٠-٢١، هجرة ٠٤٧٠)
// ══════════════════════════════════════════════════════════════════════
//
// مدخلُ البناية · «موقعك الصح هون؟» · طريقٌ مسكّر · «أضف معلم» · متاجرُنا على الخريطة.

const (
	entranceMaxFromPinM   = 150.0 // **أبعدُ من هذا ليس بابَ هذا العنوان**
	entranceMatchAddressM = 30.0  // **العنوانُ الذي نُسخت منه نقطةُ الطلب**
	dropoffSuggestMinM    = 100.0 // **أبعدُ من هذا يُقترَح على الزبون تعديلُ نقطته**
	closureMergeM         = 40.0
	closureFirstTTL       = 24 * time.Hour
	closureConfirmedTTL   = 72 * time.Hour
)

var (
	errEntranceFar       = httpx.NewError(http.StatusUnprocessableEntity, "entrance_too_far", "errors.validation")
	errEntranceNotYet    = httpx.NewError(http.StatusConflict, "entrance_not_delivered", "errors.conflict")
	errNoSuggestion      = httpx.NewError(http.StatusConflict, "no_dropoff_suggestion", "errors.conflict")
	errClosureReason     = httpx.NewError(http.StatusUnprocessableEntity, "closure_reason", "errors.validation")
	errLandmarkName      = httpx.NewError(http.StatusUnprocessableEntity, "landmark_name", "errors.validation")
	errLandmarkKind      = httpx.NewError(http.StatusUnprocessableEntity, "landmark_kind", "errors.validation")
	errLandmarkDecided   = httpx.NewError(http.StatusConflict, "landmark_decided", "errors.conflict")
	errMapContribNotOurs = httpx.NewError(http.StatusForbidden, "not_your_order", "errors.not_your_order")
)

// validPoint **نقطةٌ على الأرض** — لا صفرٌ ولا خارجَ الحدود.
func validPoint(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180 && !(lat == 0 && lng == 0)
}

// ── ١ · مدخلُ البناية ────────────────────────────────────────────────

// handleDriverEntrance **«أنا عند باب البناية»** بعد التسليم — يُحفظ مع عنوان الزبون،
// **وإن بعُد عن نقطته** اقتُرح عليه تعديلُها. POST /driver/orders/{id}/entrance
func (s *Server) handleDriverEntrance(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !validPoint(req.Lat, req.Lng) {
		s.respondErr(w, httpx.NewError(http.StatusUnprocessableEntity, "bad_point", "errors.validation"))
		return
	}
	ctx := r.Context()
	orderID := chi.URLParam(r, "id")
	uid := userIDFrom(r)
	var driver *string
	var status string
	var customer *string
	var dist float64
	err = s.pg.QueryRow(ctx, `
		SELECT driver_id::text, status, customer_id::text,
		       ST_Distance(dropoff, ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography)
		  FROM orders WHERE id = $1`, orderID, req.Lng, req.Lat).Scan(&driver, &status, &customer, &dist)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	switch {
	case driver == nil || *driver != uid:
		s.respondErr(w, errMapContribNotOurs)
		return
	case status != "delivered":
		s.respondErr(w, errEntranceNotYet)
		return
	case dist > entranceMaxFromPinM:
		s.respondErr(w, errEntranceFar)
		return
	}
	saved := int64(0)
	if customer != nil {
		tag, err := s.pg.Exec(ctx, `
			UPDATE user_addresses a
			   SET entrance = ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, entrance_set_at = now()
			  FROM orders o
			 WHERE o.id = $1 AND a.user_id = $2::uuid
			   AND ST_DWithin(a.location, o.dropoff, $5)`,
			orderID, *customer, req.Lng, req.Lat, entranceMatchAddressM)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		saved = tag.RowsAffected()
	}
	// **«موقعك الصح هون؟»** — بعيدٌ عن نقطته ⇒ اقتراحٌ وإشعار، **ولا تعديلَ بلا موافقته.**
	suggested := false
	if customer != nil && saved > 0 && dist > dropoffSuggestMinM {
		if _, err := s.pg.Exec(ctx, `UPDATE orders SET dropoff_suggest = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
			WHERE id = $1`, orderID, req.Lng, req.Lat); err == nil {
			suggested = true
			s.notify.Notify(ctx, notifications.Input{
				UserID: *customer, Kind: notifications.KindOrder,
				Title:  "موقعك الصح هون؟ — نقطة عنوانك بعيدة عن باب بنايتك",
				Body:   "اضغط لتعدّل نقطة عنوانك فيوصلك السائق المرة الجاية بالضبط",
				Entity: "order", EntityID: orderID, Href: "/portal/orders",
				Apps: []string{notifications.AppCustomer},
			})
			s.touch("order", "customer:"+*customer)
		}
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"saved": saved > 0, "suggested": suggested})
}

// suggestFromDelivery **«موقعك الصح هون؟» من مكان التسليم** (قرارُ المالك ٢٠٢٦-١٠-٢١: «نسأل
// الزبون إذا هذا عنوان بيتك الصحيح») — شُهد على الجهاز طلبٌ سُلّم على بُعد ٦٣٣ م من دبّوسه
// فلم يُسأل أحد، لأنّ «باب البناية» لا يُقبل أبعدَ من ١٥٠ م.
//
// **مكانُ التسليم موقعُ السائق لحظتَه** (قرارُ المالك: «السائق والزبون اجتمعو وسلم — هي أصحّ
// نقطة») — آخرُ نقطةٍ له في الدقيقتين، وإلّا موقعُ الزبون المباشر في الدقائق الثلاث؛
// **وكلاهما بدقّةٍ لا تتجاوز ٥٠ م**. وأبعدُ من ١٠٠ م عن الدبّوس وأقربُ من
// ٥ كم ⇒ اقتراحٌ وإشعار. **ولا يُغيَّر شيءٌ بلا موافقته**: ربّما استلم في شغله.
func (s *Server) suggestFromDelivery(ctx context.Context, orderID string) {
	var customer string
	var lat, lng float64
	err := s.pg.QueryRow(ctx, `
		WITH o AS (
			SELECT id, customer_id, driver_id, dropoff FROM orders
			 WHERE id = $1 AND status = 'delivered' AND dropoff IS NOT NULL
			   AND customer_id IS NOT NULL AND dropoff_suggest IS NULL
		), here AS (
			SELECT point, 2 AS pri FROM order_live_locations l, o
			 WHERE l.order_id = o.id AND l.at > now() - interval '3 minutes'
			   AND COALESCE(l.accuracy_m, 0) <= 50
			UNION ALL
			SELECT * FROM (SELECT t.at AS point, 1 AS pri FROM driver_track t, o
			 WHERE t.driver_id = o.driver_id AND t.created_at > now() - interval '2 minutes'
			   AND COALESCE(t.accuracy_m, 0) <= 50
			 ORDER BY t.created_at DESC LIMIT 1) d
		)
		SELECT o.customer_id::text, ST_Y(h.point::geometry), ST_X(h.point::geometry)
		  FROM o, (SELECT point FROM here ORDER BY pri LIMIT 1) h
		 WHERE ST_Distance(h.point, o.dropoff) BETWEEN $2 AND 5000
		   AND EXISTS (SELECT 1 FROM user_addresses a
		                WHERE a.user_id = o.customer_id AND ST_DWithin(a.location, o.dropoff, $3))`,
		orderID, dropoffSuggestMinM, entranceMatchAddressM).Scan(&customer, &lat, &lng)
	if err != nil {
		return
	}
	if _, err := s.pg.Exec(ctx, `UPDATE orders SET dropoff_suggest = ST_SetSRID(ST_MakePoint($2, $3), 4326)::geography
		WHERE id = $1 AND dropoff_suggest IS NULL`, orderID, lng, lat); err != nil {
		return
	}
	s.notify.Notify(ctx, notifications.Input{
		UserID: customer, Kind: notifications.KindOrder,
		Title:  "هون عنوانك الصح؟ — استلمت طلبك بعيد عن نقطة عنوانك",
		Body:   "اضغط لتعدّل نقطة عنوانك فيوصلك السائق المرة الجاية بالضبط",
		Entity: "order", EntityID: orderID, Href: "/portal/orders",
		Apps: []string{notifications.AppCustomer},
	})
	s.touch("order", "customer:"+customer)
}

// handleAcceptDropoffSuggestion **الزبونُ وافق: نقطةُ عنوانه تصير البابَ.**
// POST /orders/{id}/accept-dropoff-fix
func (s *Server) handleAcceptDropoffSuggestion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orderID := chi.URLParam(r, "id")
	uid := userIDFrom(r)
	tag, err := s.pg.Exec(ctx, `
		WITH o AS (
			SELECT id, dropoff, dropoff_suggest FROM orders
			 WHERE id = $1 AND customer_id = $2::uuid AND dropoff_suggest IS NOT NULL
		)
		UPDATE user_addresses a SET location = o.dropoff_suggest, updated_at = now()
		  FROM o WHERE a.user_id = $2::uuid AND ST_DWithin(a.location, o.dropoff, $3)`,
		orderID, uid, entranceMatchAddressM)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, errNoSuggestion)
		return
	}
	_, _ = s.pg.Exec(ctx, `UPDATE orders SET dropoff_suggest = NULL WHERE id = $1`, orderID)
	httpx.JSON(w, http.StatusOK, map[string]any{"updated": true})
}

// handleDismissDropoffSuggestion **«لا، نقطتي صحيحة»** — فلا يُسأل ثانيةً.
// DELETE /orders/{id}/accept-dropoff-fix
func (s *Server) handleDismissDropoffSuggestion(w http.ResponseWriter, r *http.Request) {
	if _, err := s.pg.Exec(r.Context(), `UPDATE orders SET dropoff_suggest = NULL
		WHERE id = $1 AND customer_id = $2::uuid`, chi.URLParam(r, "id"), userIDFrom(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"dismissed": true})
}

// ── ٢ · طريقٌ مسكّر ──────────────────────────────────────────────────

var closureReasons = map[string]bool{"works": true, "barrier": true, "market": true, "traffic": true, "other": true}

// handleDriverRoadClosure **«الطريق مسكّر هون»** — من موقعه. وبلاغٌ ثانٍ قريبٌ يُمدّده.
// POST /driver/road-closures
func (s *Server) handleDriverRoadClosure(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Lat    float64 `json:"lat"`
		Lng    float64 `json:"lng"`
		Reason string  `json:"reason"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if !validPoint(req.Lat, req.Lng) {
		s.respondErr(w, httpx.NewError(http.StatusUnprocessableEntity, "bad_point", "errors.validation"))
		return
	}
	if !closureReasons[req.Reason] {
		s.respondErr(w, errClosureReason)
		return
	}
	ctx := r.Context()
	uid := userIDFrom(r)
	// **بلاغٌ قريبٌ من بلاغٍ حيٍّ من غيره يؤكّده** — ثلاثةُ أيّام.
	var id int64
	err = s.pg.QueryRow(ctx, `
		UPDATE road_closures SET reports = reports + 1, expires_at = GREATEST(expires_at, now() + make_interval(hours => $4))
		 WHERE id = (SELECT id FROM road_closures
		              WHERE cleared_at IS NULL AND expires_at > now()
		                AND ST_DWithin(point, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3)
		                AND reported_by IS DISTINCT FROM $5::uuid
		              ORDER BY created_at DESC LIMIT 1)
		RETURNING id`, req.Lng, req.Lat, closureMergeM, int(closureConfirmedTTL.Hours()), uid).Scan(&id)
	merged := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		err = s.pg.QueryRow(ctx, `
			INSERT INTO road_closures (point, reason, reported_by, expires_at)
			VALUES (ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $3, $4::uuid, now() + make_interval(hours => $5))
			RETURNING id`, req.Lng, req.Lat, req.Reason, uid, int(closureFirstTTL.Hours())).Scan(&id)
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("road_closure", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "confirmed": merged})
}

// activeClosures **المسكَّرُ الحيُّ قربَ الرحلة** — يتجنّبه التوجيه (`routing.SetExcluder`).
func (s *Server) activeClosures(ctx context.Context, a, b routing.Point) []routing.Point {
	minLat, maxLat := a.Lat, b.Lat
	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}
	minLng, maxLng := a.Lng, b.Lng
	if minLng > maxLng {
		minLng, maxLng = maxLng, minLng
	}
	const pad = 0.02 // ~٢ كم
	rows, err := s.pg.Query(ctx, `
		SELECT ST_Y(point::geometry), ST_X(point::geometry) FROM road_closures
		 WHERE cleared_at IS NULL AND expires_at > now()
		   AND ST_Intersects(point, ST_MakeEnvelope($1, $2, $3, $4, 4326)::geography)
		   -- **ولا يُتجنَّب شارعٌ يقف عليه السائقُ أو الزبون** (٢٠٢٦-١٠-٢١) — منه يبدأ وإليه يصل.
		   AND NOT ST_DWithin(point, ST_SetSRID(ST_MakePoint($6, $5), 4326)::geography, 60)
		   AND NOT ST_DWithin(point, ST_SetSRID(ST_MakePoint($8, $7), 4326)::geography, 60)
		 LIMIT 50`, minLng-pad, minLat-pad, maxLng+pad, maxLat+pad, a.Lat, a.Lng, b.Lat, b.Lng)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []routing.Point
	for rows.Next() {
		var p routing.Point
		if rows.Scan(&p.Lat, &p.Lng) == nil {
			out = append(out, p)
		}
	}
	return out
}

// handleAdminRoadClosures **المسكَّرُ الآن** — لخريطة العمليات. GET /admin/road-closures
func (s *Server) handleAdminRoadClosures(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT c.id, ST_Y(c.point::geometry), ST_X(c.point::geometry), c.reason, c.reports,
		       c.created_at, c.expires_at, COALESCE(u.full_name, '')
		  FROM road_closures c LEFT JOIN users u ON u.id = c.reported_by
		 WHERE c.cleared_at IS NULL AND c.expires_at > now()
		 ORDER BY c.created_at DESC LIMIT 500`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type closure struct {
		ID        int64     `json:"id"`
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		Reason    string    `json:"reason"`
		Reports   int       `json:"reports"`
		CreatedAt time.Time `json:"created_at"`
		ExpiresAt time.Time `json:"expires_at"`
		By        string    `json:"reported_by_name"`
	}
	out := []closure{}
	for rows.Next() {
		var c closure
		if err := rows.Scan(&c.ID, &c.Lat, &c.Lng, &c.Reason, &c.Reports, &c.CreatedAt, &c.ExpiresAt, &c.By); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, c)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"closures": out, "count": len(out)})
}

// handleAdminClearRoadClosure **انفتح الطريق.** DELETE /admin/road-closures/{id}
func (s *Server) handleAdminClearRoadClosure(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	tag, err := s.pg.Exec(r.Context(), `UPDATE road_closures SET cleared_at = now(), cleared_by = $2::uuid
		WHERE id = $1 AND cleared_at IS NULL`, id, userIDFrom(r))
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if tag.RowsAffected() == 0 {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	s.touch("road_closure", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"cleared": true})
}

// ── ٣ · «أضف معلم» ───────────────────────────────────────────────────

// landmarkKinds **الأنواعُ ووسمُها في جدول المعالم** — كما تُقرأ في «قرب المعلم».
var landmarkKinds = map[string]string{
	"mosque": "amenity=place_of_worship", "pharmacy": "amenity=pharmacy", "school": "amenity=school",
	"hospital": "amenity=hospital", "shop": "shop=yes", "roundabout": "place=square", "other": "tourism=attraction",
}

// handleDriverLandmark **«أضف معلم»** — من موقعه، وبانتظار الإدارة. POST /driver/landmarks
func (s *Server) handleDriverLandmark(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Name string  `json:"name"`
		Kind string  `json:"kind"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if n := len([]rune(name)); n < 2 || n > 80 {
		s.respondErr(w, errLandmarkName)
		return
	}
	if _, ok := landmarkKinds[req.Kind]; !ok {
		s.respondErr(w, errLandmarkKind)
		return
	}
	if !validPoint(req.Lat, req.Lng) {
		s.respondErr(w, httpx.NewError(http.StatusUnprocessableEntity, "bad_point", "errors.validation"))
		return
	}
	var id int64
	if err := s.pg.QueryRow(r.Context(), `
		INSERT INTO landmark_suggestions (name, kind, point, suggested_by)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography, $5::uuid) RETURNING id`,
		name, req.Kind, req.Lng, req.Lat, userIDFrom(r)).Scan(&id); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("landmark", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"id": id, "status": "pending"})
}

// handleAdminLandmarkSuggestions **ما ينتظر الموافقة.** GET /admin/landmark-suggestions
func (s *Server) handleAdminLandmarkSuggestions(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT l.id, l.name, l.kind, ST_Y(l.point::geometry), ST_X(l.point::geometry), l.created_at,
		       COALESCE(u.full_name, '')
		  FROM landmark_suggestions l LEFT JOIN users u ON u.id = l.suggested_by
		 WHERE l.status = 'pending' ORDER BY l.created_at LIMIT 500`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type sugg struct {
		ID        int64     `json:"id"`
		Name      string    `json:"name"`
		Kind      string    `json:"kind"`
		Lat       float64   `json:"lat"`
		Lng       float64   `json:"lng"`
		CreatedAt time.Time `json:"created_at"`
		By        string    `json:"suggested_by_name"`
	}
	out := []sugg{}
	for rows.Next() {
		var x sugg
		if err := rows.Scan(&x.ID, &x.Name, &x.Kind, &x.Lat, &x.Lng, &x.CreatedAt, &x.By); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"suggestions": out, "count": len(out)})
}

// handleAdminDecideLandmark **موافقةٌ أو رفض** — والموافَقُ يدخل «قرب المعلم» فوراً.
// POST /admin/landmark-suggestions/{id}/decide
func (s *Server) handleAdminDecideLandmark(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Approve bool `json:"approve"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer func() { _ = tx.Rollback(ctx) }()
	status := "rejected"
	if req.Approve {
		status = "approved"
	}
	var name, kind string
	err = tx.QueryRow(ctx, `
		UPDATE landmark_suggestions SET status = $2, decided_by = $3::uuid, decided_at = now()
		 WHERE id = $1 AND status = 'pending' RETURNING name, kind`, id, status, userIDFrom(r)).Scan(&name, &kind)
	if errors.Is(err, pgx.ErrNoRows) {
		s.respondErr(w, errLandmarkDecided)
		return
	}
	if err != nil {
		s.respondErr(w, err)
		return
	}
	if req.Approve {
		if _, err := tx.Exec(ctx, `
			INSERT INTO map_landmarks (name, kind, point, source)
			SELECT name, $2, point, 'driver' FROM landmark_suggestions WHERE id = $1`,
			id, landmarkKinds[kind]); err != nil {
			s.respondErr(w, err)
			return
		}
	}
	if err := tx.Commit(ctx); err != nil {
		s.respondErr(w, err)
		return
	}
	s.touch("landmark", "ops")
	httpx.JSON(w, http.StatusOK, map[string]any{"status": status})
}

// ── ٤ · متاجرُنا على الخريطة ─────────────────────────────────────────

// handlePublicMapStores **كلُّ متجرٍ نشطٍ نقطةٌ على الخريطة** — فيظهر الجديدُ بلا بناءِ
// خريطة. GET /public/map/stores
func (s *Server) handlePublicMapStores(w http.ResponseWriter, r *http.Request) {
	rows, err := s.pg.Query(r.Context(), `
		SELECT m.id, m.name, ST_Y(m.location::geometry), ST_X(m.location::geometry)
		  FROM merchants m
		 WHERE m.status = 'active' AND m.location IS NOT NULL
		   AND EXISTS (SELECT 1 FROM menu_items i WHERE i.merchant_id = m.id AND i.available)
		 LIMIT 5000`)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	defer rows.Close()
	type store struct {
		ID   string  `json:"id"`
		Name string  `json:"name"`
		Lat  float64 `json:"lat"`
		Lng  float64 `json:"lng"`
	}
	out := []store{}
	for rows.Next() {
		var x store
		if err := rows.Scan(&x.ID, &x.Name, &x.Lat, &x.Lng); err != nil {
			s.respondErr(w, err)
			return
		}
		out = append(out, x)
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	// **وبصيغة الخريطة لمن طلبها** — تقرؤها طبقةُ المتاجر في التطبيقات مباشرةً (MapStores).
	if r.URL.Query().Get("format") == "geojson" {
		feats := make([]map[string]any, 0, len(out))
		for _, x := range out {
			feats = append(feats, map[string]any{
				"type":       "Feature",
				"geometry":   map[string]any{"type": "Point", "coordinates": []float64{x.Lng, x.Lat}},
				"properties": map[string]any{"id": x.ID, "name": x.Name},
			})
		}
		w.Header().Set("Content-Type", "application/geo+json")
		_ = json.NewEncoder(w).Encode(map[string]any{"type": "FeatureCollection", "features": feats})
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"stores": out})
}

// ── ٥ · التطبيقُ يُبلغ قبل الخروج ─────────────────────────────────────

// handleClientLogoutReport **التطبيقُ يقول لماذا أخرج صاحبَه** (قرارُ المالك ٢٠٢٦-١٠-٢١) —
// قبل أن يمحو جلستَه. **بلا توثيق** (الجلسةُ ماتت)، **وصفٌّ في سجلّ الخروج** بنوع
// `client`. POST /auth/client-report
func (s *Server) handleClientLogoutReport(w http.ResponseWriter, r *http.Request) {
	// **وبلا توثيقٍ فبحدٍّ للعنوان** — عشرون في الساعة تكفي جهازاً يخرج، ولا تكفي من يُغرق السجلّ.
	ip := clientIP(r)
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	key := "authrep:rl:" + ip
	if n, err := s.rdb.Incr(r.Context(), key).Result(); err == nil {
		if n == 1 {
			s.rdb.Expire(r.Context(), key, time.Hour)
		}
		if n > 20 {
			s.respondErr(w, httpx.NewError(http.StatusTooManyRequests, "rate_limited", "errors.rate_limited"))
			return
		}
	}
	req, err := decode[struct {
		Code   string `json:"code"`
		Path   string `json:"path"`
		Detail string `json:"detail"`
		// RefreshToken **رمزُ الجلسة التي ماتت** — يُعرف به صاحبُها في السجلّ، ولا يُحفظ.
		RefreshToken string `json:"refresh_token"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	clip := func(v string, n int) string {
		if rs := []rune(v); len(rs) > n {
			return string(rs[:n])
		}
		return v
	}
	code := clip(strings.TrimSpace(req.Code), 60)
	if code == "" {
		code = "unknown"
	}
	detail := clip(strings.TrimSpace(req.Path)+" "+strings.TrimSpace(req.Detail), 200)
	uid, sid, _ := s.identity.RefreshTokenOwner(r.Context(), req.RefreshToken)
	s.recordAuthFailure(r, "client", code, uid, sid, strings.TrimSpace(detail))
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}
