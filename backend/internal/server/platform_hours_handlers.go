package server

// ══════════════════════════════════════════════════════════════════════
// **لوحةُ دوام المنصّة وإيقافِها** (`PH`، ٢٠٢٦-٠٩-١٤)
// ══════════════════════════════════════════════════════════════════════
//
// **ولا قدرةَ جديدةٌ تُخترَع**: **`settings.general.manage` هي التي
// تحكم مناطقَ التوصيل** (`/zones`) **وأبوابَ الإطلاق** (عبر
// `settingCapability`) — **وهذا من بابها نفسِه.** **وقدرةٌ تُضاف لبابٍ
// واحدٍ تُمنَح ثمّ تُنسى.**

import (
	"net/http"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/platform"
)

// windowInput **فترةٌ كما تُرسلها اللوحةُ** — `HH:MM` نصّاً.
//
// **ولا دقائقُ خامّةٌ في عقدٍ عامّ**: **رقمٌ ٦٢٠ يُقرأ خطأً**،
// **و`10:20` تُقرأ كما هي.**
type windowInput struct {
	Day   int    `json:"day_of_week"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// handleGetPlatformHours **جدولُ الأسبوع كما هو محفوظ.**
func (s *Server) handleGetPlatformHours(w http.ResponseWriter, r *http.Request) {
	sch, err := s.platform.Schedule(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"windows": sch,
		// **وسريانُ الجدول يُقرأ معه** — **وجدولٌ مكتوبٌ لا يسري
		// يُقرأ سارياً إن لم يُقَل**، **فيظنّ المالكُ أنّه أغلق الليلَ
		// وهو مفتوح.**
		"enforced": s.settings.GetBool(r.Context(), platform.EnforcedKey),
		"timezone": platform.TZ,
	})
}

// handleSetPlatformHours **يستبدل الجدولَ كلَّه.**
//
// **والتحقّقُ في المحرّك** — **ونداءٌ مباشرٌ يتجاوز كلَّ تحقّقٍ في
// متصفّح.**
func (s *Server) handleSetPlatformHours(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Windows []windowInput `json:"windows"`
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
	sch, err := s.platform.SetSchedule(r.Context(), userIDFrom(r), ws)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"windows": sch})
}

// handleGetServiceClosure **حالُ الإيقاف المؤقّت.**
func (s *Server) handleGetServiceClosure(w http.ResponseWriter, r *http.Request) {
	c, err := s.platform.Closure(r.Context(), s.pg)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, c)
}

// handleSetServiceClosure **يضبط الإيقافَ المؤقّت.**
//
// **وهو ليس وضعَ إطلاق** — **ولا يُطفئ `launch.customer_orders`
// ولا يقرؤه.** **الطبقتان مستقلّتان بقصد.**
func (s *Server) handleSetServiceClosure(w http.ResponseWriter, r *http.Request) {
	req, err := decode[struct {
		Active  bool   `json:"active"`
		Message string `json:"message"`
		// EndsAt **موعدُ العودة** — RFC 3339، وفارغٌ يعني «لا موعد».
		EndsAt string `json:"ends_at"`
	}](r)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	c := platform.Closure{Active: req.Active, Message: req.Message}
	if req.EndsAt != "" {
		t, err := time.Parse(time.RFC3339, req.EndsAt)
		if err != nil {
			s.respondErr(w, platform.ErrClosureRange)
			return
		}
		c.EndsAt = &t
	}
	out, err := s.platform.SetClosure(r.Context(), userIDFrom(r), c)
	if err != nil {
		s.respondErr(w, err)
		return
	}
	// ══════════════════════════════════════════════════════════════════
	// **ويُسجَّل من أوقف المنصّةَ ومن أعادها** (٢٠٢٦-٠٩-١٧)
	// ══════════════════════════════════════════════════════════════════
	//
	// **و`service_closure` صفٌّ واحدٌ يُكتب فوقه** — **فـ`updated_by`
	// يحفظ آخرَ فاعلٍ لا تاريخَ الأفعال.** **ومن أوقف المنصّةَ أمسِ
	// ثمّ أعادها غيرُه اليومَ ذهب أوّلُهما بلا أثر.**
	//
	// **وهذا أخطرُ ما يُسأل عنه**: **إيقافُ المنصّة يمنع الطلبَ عن
	// البلد كلِّه** — **وأخوهُ `admin.zone_hours_set` مسجَّلٌ منذ
	// زمن**، فكان غيابُه سهواً لا قصدا. (قِيس في قبول لوحة التجهيز.)
	//
	// **وبالسجلّ القائم لا بسجلٍّ ثانٍ** — **والنصُّ إعلانٌ للناس
	// لا سرّ.**
	meta := map[string]any{"active": c.Active, "message": c.Message}
	if c.EndsAt != nil {
		meta["ends_at"] = c.EndsAt.Format(time.RFC3339)
	}
	s.audit(r, "admin.platform_closure", "platform", "", meta)
	httpx.JSON(w, http.StatusOK, out)
}
