package server

// ══════════════════════════════════════════════════════════════════════
// **لوحةُ معالم السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٥) — قسمٌ في «الخريطة من السائقين»
// ══════════════════════════════════════════════════════════════════════
//
// «السائقين عم يقولو عم يبعثو معالم بس انا ماعم يوصلني شي»: كانت تُحفظ ولا إشعارَ بها،
// **وطبقتُها في خريطة العمليّات مطفأةٌ من أوّل يوم.** فصار لها:
//
//   - `GET /admin/ops-map/landmarks/daily?day=` — مربّعاتُ يوم المنصّة وكلُّ سائقٍ فيه.
//   - `GET /admin/ops-map/landmark-suggestions/count` — شارةُ القائمة الجانبيّة.
//   - `GET /admin/ops-map/landmark-suggestions` — **افتراضُه «بانتظار الموافقة» كما كان**
//     (خريطةُ العمليّات تقرؤه بلا مُعامِل)، **ويقبل الحالةَ واليومَ والسائقَ والصفحات.**

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
)

// landmarkPageMax **أطولُ صفحة** — والطابورُ يكبر بخمسةٍ لكلّ سائقٍ كلَّ يوم.
const landmarkPageMax = 500

// handleAdminLandmarksDaily **يومُ المعالم** — افتراضُه يومُ المنصّة الآن. GET /admin/ops-map/landmarks/daily
func (s *Server) handleAdminLandmarksDaily(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	startHour := s.dayStartHour(ctx)
	today := siteDay(time.Now(), startHour)
	day := strings.TrimSpace(r.URL.Query().Get("day"))
	if day == "" {
		day = today
	} else if _, err := time.Parse("2006-01-02", day); err != nil {
		s.respondErr(w, errValidation)
		return
	}
	quota := s.landmarkQuota(ctx)
	rows, tot, err := s.landmarkDaily(ctx, day, startHour, quota)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"day": day, "today": today, "quota": quota, "totals": tot, "drivers": rows,
	})
}

// handleAdminLandmarkCount **كم ينتظر الموافقة** — لشارة «الخريطة من السائقين».
// GET /admin/ops-map/landmark-suggestions/count
func (s *Server) handleAdminLandmarkCount(w http.ResponseWriter, r *http.Request) {
	var n int
	if err := s.pg.QueryRow(r.Context(),
		`SELECT count(*) FROM landmark_suggestions WHERE status = 'pending'`).Scan(&n); err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"pending": n})
}

// landmarkSuggestionsQuery **مرشِّحاتُ القائمة** — والفاسدُ يُردّ لا يُتجاهَل.
type landmarkSuggestionsQuery struct {
	status, day, driver string
	limit, offset       int
}

func parseLandmarkSuggestionsQuery(r *http.Request) (landmarkSuggestionsQuery, bool) {
	q := r.URL.Query()
	out := landmarkSuggestionsQuery{
		status: strings.TrimSpace(q.Get("status")),
		day:    strings.TrimSpace(q.Get("day")),
		driver: strings.TrimSpace(q.Get("driver")),
		limit:  landmarkPageMax,
	}
	switch out.status {
	case "":
		out.status = "pending"
	case "pending", "approved", "rejected", "duplicate", "all":
	default:
		return out, false
	}
	if out.day != "" {
		if _, err := time.Parse("2006-01-02", out.day); err != nil {
			return out, false
		}
	}
	if out.driver != "" && !isUUID(out.driver) {
		return out, false
	}
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return out, false
		}
		out.limit = min(n, landmarkPageMax)
	}
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return out, false
		}
		out.offset = n
	}
	return out, true
}
