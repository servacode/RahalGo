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
