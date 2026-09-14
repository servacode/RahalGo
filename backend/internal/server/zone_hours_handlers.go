package server

// ══════════════════════════════════════════════════════════════════════
// **أوقاتُ منطقةِ توصيلٍ — من باب المنطقة نفسِها** (`ZH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا قسمٌ جديدٌ في اللوحة** — **وجدولُ المنطقة صفةٌ من صفاتها كالرسم
// ونصفِ القطر والحدِّ الأدنى**، **ومن فصله في شاشةٍ ثانيةٍ جعل من يضبط
// منطقةً يبحث عن وقتها في مكانٍ آخر.**
//
// **والقدرةُ قدرةُ المناطق عينُها** (`settings.general.manage`) — **ولا
// قدرةَ جديدةٌ تُخترَع لبابٍ واحد.**

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// handleGetZoneHours **جدولُ منطقةٍ ورايةُ سريانه.**
//
// **ويُقرأ الجدولُ ولو كان غيرَ سارٍ** — **فمن كتب جدولاً ثمّ أطفأ
// السريانَ يجب أن يجد ما كتبه حين يعود**، **لا صفحةً بيضاء.**
func (s *Server) handleGetZoneHours(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var enforced bool
	if err := s.pg.QueryRow(r.Context(),
		`SELECT hours_enforced FROM delivery_zones WHERE id = $1::uuid`, id).
		Scan(&enforced); err != nil {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	sch, err := s.platform.ZoneWindows(r.Context(), s.pg, id)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"windows":  sch,
		"enforced": enforced,
		"timezone": platform.TZ,
	})
}

// handleSetZoneHours **يستبدل جدولَ منطقةٍ ويضبط سريانَه.**
func (s *Server) handleSetZoneHours(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Enforced bool          `json:"enforced"`
		Windows  []windowInput `json:"windows"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	ws := make([]platform.Window, 0, len(req.Windows))
	for _, in := range req.Windows {
		start, err := platform.ParseClock(in.Start)
		if err != nil {
			s.respondErr(w, platform.ErrBadSchedule)
			return
		}
		end, err := platform.ParseClock(in.End)
		if err != nil {
			s.respondErr(w, platform.ErrBadSchedule)
			return
		}
		ws = append(ws, platform.Window{Day: in.Day, Start: start, End: end})
	}
	id := chi.URLParam(r, "id")
	sch, err := s.platform.SetZoneSchedule(r.Context(), id, req.Enforced, ws)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// **ويُسجَّل من بدّل أيَّ منطقةٍ ومتى** — **بالسجلّ القائم لا بسجلٍّ
	// ثانٍ**، **وتبديلُ أوقاتِ التوصيل قرارٌ تشغيليٌّ يُسأل عنه.**
	s.audit(r, "admin.zone_hours_set", "zone", id, map[string]any{
		"enforced": req.Enforced,
		"windows":  sch,
	})
	httpx.JSON(w, http.StatusOK, map[string]any{
		"windows": sch, "enforced": req.Enforced,
	})
}
