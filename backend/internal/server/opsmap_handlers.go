package server

// ══════════════════════════════════════════════════════════════════════
// **خريطةُ العمليات — أبوابُها**
// ══════════════════════════════════════════════════════════════════════
//
// # ولا بابٌ عملاقٌ يردّ كلَّ شيء (البند ٤٢)
//
// **بابٌ لكلّ طبقة** — فمن أطفأ طبقةً لم يحمّلها، **ومن فتح الخريطةَ
// أوّلَ مرّةٍ لم ينتظر تحليلاتٍ لا يريدها.**
//
// # وكلُّها تقبل `bbox` (البند ٤٣)
//
// **ولا تُحمَّل سوريا كلُّها بكلّ كائناتها في أوّل فتحة.**

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/opsmap"
)

// requirePerm حارسُ صلاحيّةٍ مسمّاة.
//
// **ولا تُفتح الصفحةُ بتوكن إدارةٍ وحدَه** (البند ٣٢) — **والدورُ
// يُترجَم إلى صلاحيّةٍ في موضعٍ واحد**، فحين يُبنى `AQ-1` يُبدَّل
// المُترجِمُ ولا تُمسّ المعالجات.
func (s *Server) requirePerm(p opsmap.Perm, h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !opsmap.Allows(rolesFrom(r), p) {
			httpx.Error(w, errForbidden)
			return
		}
		h(w, r)
	}
}

// bboxFrom مستطيلُ المشهد من الاستعلام — `bbox=minLng,minLat,maxLng,maxLat`.
//
// **وفارغٌ يعني «كلَّ شيء»** — وتبقى الحدودُ العُليا تحرسه.
func bboxFrom(r *http.Request) (*opsmap.BBox, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("bbox"))
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	if len(parts) != 4 {
		return nil, httpx.NewError(http.StatusBadRequest, "bad_bbox", "errors.bad_request")
	}
	var v [4]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return nil, httpx.NewError(http.StatusBadRequest, "bad_bbox", "errors.bad_request")
		}
		v[i] = f
	}
	b := opsmap.BBox{MinLng: v[0], MinLat: v[1], MaxLng: v[2], MaxLat: v[3]}
	if !b.Valid() {
		return nil, httpx.NewError(http.StatusBadRequest, "bad_bbox", "errors.bad_request")
	}
	return &b, nil
}

// boolParam ثلاثيّةٌ: غيابٌ يعني «لا ترشّح».
func boolParam(r *http.Request, key string) *bool {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return nil
	}
	v := raw == "true" || raw == "1"
	return &v
}

// handleOpsMapMeta ما يملكه من يفتح الخريطة، وأين يبدأ المشهد.
//
// **والواجهةُ لا ترسم طبقةً لا يملكها صاحبُها** — **ورؤيةُ طبقةٍ تردّ
// `403` أسوأُ من غيابها.**
func (s *Server) handleOpsMapMeta(w http.ResponseWriter, r *http.Request) {
	perms := opsmap.Granted(rolesFrom(r))
	out := map[string]any{
		"permissions": perms,
		// **ونبضةُ الموضع تُرسَل** — الواجهةُ تشرح بها معنى «حيّ».
		"location_ping_sec": s.settings.GetNum(r.Context(), "drivers.location_ping_sec", 60),
		"assignable_sec":    int(opsmap.AssignableWindow.Seconds()),
	}
	httpx.JSON(w, http.StatusOK, out)
}

// handleOpsMapDrivers طبقةُ السائقين.
func (s *Server) handleOpsMapDrivers(w http.ResponseWriter, r *http.Request) {
	box, err := bboxFrom(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	q := r.URL.Query()
	f := opsmap.DriverFilter{
		CityID:    q.Get("city_id"),
		Status:    q.Get("status"),
		Search:    strings.TrimSpace(q.Get("q")),
		OnShift:   boolParam(r, "on_shift"),
		HasActive: boolParam(r, "has_active"),
		Stale:     boolParam(r, "stale"),
	}
	limit, _ := strconv.Atoi(q.Get("limit"))

	ping := s.settings.GetNum(r.Context(), "drivers.location_ping_sec", 60)
	money := opsmap.Allows(rolesFrom(r), opsmap.PermViewMoney)

	out, err := opsmap.Drivers(r.Context(), s.pg, box, f, ping, money, limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"drivers": out,
		"count":   len(out),
	})
}

// handleOpsMapMerchants طبقةُ المتاجر.
func (s *Server) handleOpsMapMerchants(w http.ResponseWriter, r *http.Request) {
	box, err := bboxFrom(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	q := r.URL.Query()
	f := opsmap.MerchantFilter{
		CityID:    q.Get("city_id"),
		Status:    q.Get("status"),
		Search:    strings.TrimSpace(q.Get("q")),
		OpenNow:   boolParam(r, "open_now"),
		HasActive: boolParam(r, "has_active"),
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	// **واسمُ المندوب لمن يراقب نشاطَ المندوبين وحدَه** (البند ٩).
	withRep := opsmap.Allows(rolesFrom(r), opsmap.PermViewRepActivity)

	out, err := opsmap.Merchants(r.Context(), s.pg, box, f, withRep, limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"merchants": out, "count": len(out)})
}

// handleOpsMapOrders طبقةُ الطلبات النشطة.
func (s *Server) handleOpsMapOrders(w http.ResponseWriter, r *http.Request) {
	box, err := bboxFrom(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	q := r.URL.Query()
	f := opsmap.OrderFilter{
		Status:     q.Get("status"),
		MerchantID: q.Get("merchant_id"),
		DriverID:   q.Get("driver_id"),
		Search:     strings.TrimSpace(q.Get("q")),
		Unassigned: boolParam(r, "unassigned"),
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	money := opsmap.Allows(rolesFrom(r), opsmap.PermViewMoney)

	out, err := opsmap.ActiveOrders(r.Context(), s.pg, box, f, money, limit)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"orders": out, "count": len(out)})
}

// ══════════════════════════════════════════════════════════════════════
// **التغطيةُ وطلباتُها — `MAP-3`**
// ══════════════════════════════════════════════════════════════════════
//
// # والتدقيقُ على كلّ كتابةٍ حسّاسة (البند ٣٤)
//
// **`s.audit` هو نمطُ المنصّة القائم** — **وهو `best-effort` خارجَ
// المعاملة، وذلك `XG-20` مسجَّلةٌ ومجمَّدة.** **ولا يُصلَح نظامُ التدقيق
// كلُّه هنا** (خارجَ النطاق) — **ويبقى النقصُ معلَناً في الحقيقة
// والبوّابة**، لا مخبوءاً.

func (s *Server) handleOpsMapCoverage(w http.ResponseWriter, r *http.Request) {
	box, err := bboxFrom(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	onlyActive := r.URL.Query().Get("active") == "true"
	out, err := opsmap.Zones(r.Context(), s.pg, box, onlyActive)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"zones": out, "count": len(out)})
}

func (s *Server) handleOpsMapCoverageSave(w http.ResponseWriter, r *http.Request) {
	in, err := decode[opsmap.ZoneInput](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	newID, err := opsmap.SavePolygonZone(r.Context(), s.pg, id, *in)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	action := "coverage.create"
	if id != "" {
		action = "coverage.update"
	}
	s.audit(r, action, "delivery_zone", newID, map[string]any{
		"name": in.Name, "points": len(in.Ring),
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"id": newID})
}

func (s *Server) handleOpsMapCoverageActive(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Active bool `json:"active"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	if err := opsmap.SetZoneActive(r.Context(), s.pg, id, in.Active); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "coverage.set_active", "delivery_zone", id,
		map[string]any{"active": in.Active})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleOpsMapRequests(w http.ResponseWriter, r *http.Request) {
	box, err := bboxFrom(r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	q := r.URL.Query()
	f := opsmap.RequestFilter{Status: q.Get("status"), CityID: q.Get("city_id")}
	if from, to, ok := rangeFrom(r); ok {
		f.Since, f.Until = &from, &to
	}
	out, err := opsmap.CoverageRequests(r.Context(), s.pg, box, f, 0)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"requests": out, "count": len(out), "states": opsmap.RequestStates(),
	})
}

func (s *Server) handleOpsMapRequestUpdate(w http.ResponseWriter, r *http.Request) {
	in, err := decode[struct {
		Status string `json:"status"`
		Note   string `json:"note"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id := chi.URLParam(r, "id")
	if err := opsmap.UpdateRequest(r.Context(), s.pg, id, in.Status, in.Note,
		userIDFrom(r)); err != nil {
		s.respondErr(w, err)
		return
	}
	s.audit(r, "coverage_request.update", "coverage_request", id,
		map[string]any{"status": in.Status})
	httpx.JSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleCoverageRequestCreate **زرُّ «اطلب تغطية منطقتي».**
//
// # ولماذا يقبل من لا حسابَ له
//
// **من فتح التطبيقَ فوجد أنّنا لا نصله قد لا يكون سجّل بعد** —
// **ورفضُ طلبه لأنّه بلا حسابٍ يمحو أصدقَ إشارةٍ عندنا.**
func (s *Server) handleCoverageRequestCreate(w http.ResponseWriter, r *http.Request) {
	in, err := decode[opsmap.NewRequest](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	id, err := opsmap.CreateRequest(r.Context(), s.pg, userIDFrom(r), *in)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, map[string]any{"id": id})
}

// rangeFrom مدىً زمنيٌّ من الاستعلام (البند ٢٩).
//
// **و`range=today|7d|30d` أو `from`/`to` صريحان.**
func rangeFrom(r *http.Request) (from, to time.Time, ok bool) {
	q := r.URL.Query()
	now := time.Now()
	switch q.Get("range") {
	case "today":
		return now.Truncate(24 * time.Hour), now, true
	case "7d":
		return now.AddDate(0, 0, -7), now, true
	case "30d":
		return now.AddDate(0, 0, -30), now, true
	}
	f, e1 := time.Parse(time.RFC3339, q.Get("from"))
	t, e2 := time.Parse(time.RFC3339, q.Get("to"))
	if e1 == nil && e2 == nil && t.After(f) {
		return f, t, true
	}
	return time.Time{}, time.Time{}, false
}
