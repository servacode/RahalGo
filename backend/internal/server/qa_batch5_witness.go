package server

import (
	"net/http"
	"sync"
	"time"

	"github.com/servacode/rahalgo/backend/internal/httpx"
	"github.com/servacode/rahalgo/backend/internal/identity"
	"github.com/servacode/rahalgo/backend/internal/platform"
	"github.com/servacode/rahalgo/backend/internal/realtime"
)

// ══════════════════════════════════════════════════════════════════════
// **شاهدُ حدِّ الوقت الحيّ — جدولٌ يُغلَق بعد دقيقتين** (Batch 5، FINAL-TIME)
// ══════════════════════════════════════════════════════════════════════
//
// (قرارُ المالك ٢٠٢٦-٠٩-٢٥: «Staging QA لا يملك مصادقةَ أدمن، فأنشئ قدرةً
//  ضيّقةً تمرّ بخدمةِ الأدمن الحقيقيّة (`SetSchedule`/`SetZoneSchedule`)
//  لتضبطَ حدَّ إغلاقٍ مستقبليّاً، ثمّ يُشهَد انقلابُ الجهازِ عند الحدّ».)
//
// # لماذا هذه القدرةُ آمنة
//
// **لا توكن، ولا صلاحيّةَ أدمن، ولا جدولَ من الطلب**: القدرةُ تحسب نافذةً
// تُغلَق بعد `close_in_min` دقيقةً (افتراضُه ٢) وتُعيد الفتحَ بعدها بدقيقتين،
// **وتمرّ بنفسِ `SetSchedule`/`SetZoneSchedule` الحقيقيّين** — لا حقنَ SQL.
// **وتحفظ الجدولَ السابقَ لتستعيدَه** (`boundary_restore`). **وتسقط مغلقةً
// في الإنتاج** (حارسٌ في المنادي وهنا). **ومنطقةُ الشاهدِ هي منطقةُ عنوانِ
// زبونِ QA الافتراضيِّ نفسِها** — لا معرّفَ من الطلب.
//
// **والبثُّ يُطلَق عند التسليح** (`TopicCatalog`) — كما يفعل بابُ الأدمن
// الحقيقيّ عبر `announceWrites` — **ليجدّد التطبيقُ المفتوحُ نفسَه مرّةً
// فيُسلّح مُوقِّتَ الحدّ.** **والانقلابُ عند الحدِّ نفسِه بالمؤقّتِ لا بحدثِ
// خادم** — وهو المشهود.

// qaBoundarySaved **الحالُ السابقةُ لتُستعاد** — لا يُترَك جدولٌ اختباريٌّ نافذاً.
var qaBoundarySaved struct {
	mu sync.Mutex

	platformArmed    bool
	platformSchedule platform.Schedule
	platformEnforced bool

	zoneArmed    bool
	zoneID       string
	zoneSchedule platform.Schedule
	zoneEnforced bool
}

// qaBoundaryWindows **يبني نافذتين لليوم**: مفتوحةٌ الآن تُغلَق بعد
// `closeInMin`، ثمّ مفتوحةٌ ثانيةً بعد الإغلاقِ بدقيقتين (حدُّ إعادةِ الفتح).
// يُرجع النوافذَ ولحظتَي الإغلاقِ والفتحِ المطلقتين، أو خطأً إن قَرُبَ منتصفُ الليل.
func qaBoundaryWindows(now time.Time, closeInMin int) ([]platform.Window, time.Time, time.Time, bool) {
	day := int(now.Weekday())
	mod := now.Hour()*60 + now.Minute()
	closeMOD := mod + closeInMin
	reopenStart := closeMOD + 2
	reopenEnd := reopenStart + 120
	// **قربَ منتصفِ الليل يُرفض** — نوافذُ اليومِ لا تعبر، والشاهدُ نهاريّ.
	if reopenEnd >= 1440 || closeMOD >= 1439 || mod < 2 {
		return nil, time.Time{}, time.Time{}, false
	}
	openStart := closeMOD - 120
	if openStart < 0 {
		openStart = 0
	}
	ws := []platform.Window{
		{Day: day, Start: platform.Minutes(openStart), End: platform.Minutes(closeMOD)},
		{Day: day, Start: platform.Minutes(reopenStart), End: platform.Minutes(reopenEnd)},
	}
	loc := platform.Location()
	closeAt := time.Date(now.Year(), now.Month(), now.Day(), closeMOD/60, closeMOD%60, 0, 0, loc)
	reopenAt := time.Date(now.Year(), now.Month(), now.Day(), reopenStart/60, reopenStart%60, 0, 0, loc)
	return ws, closeAt, reopenAt, true
}

// qaBoundaryArm يضبط حدَّ إغلاقٍ مستقبليّاً للمنصّةِ أو المنطقةِ عبر خدمةِ
// الأدمن الحقيقيّة، ويحفظ السابقَ، ويبثّ إشارةَ التغيّر.
func (s *Server) qaBoundaryArm(w http.ResponseWriter, r *http.Request, target string, closeInMin int) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	if closeInMin < 1 || closeInMin > 30 {
		closeInMin = 2
	}
	ctx := r.Context()
	now := time.Now().In(platform.Location())
	ws, closeAt, reopenAt, ok := qaBoundaryWindows(now, closeInMin)
	if !ok {
		s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_boundary_near_midnight", "errors.conflict"))
		return
	}
	actor := s.qaCoverageActor(ctx)

	qaBoundarySaved.mu.Lock()
	defer qaBoundarySaved.mu.Unlock()

	out := map[string]any{
		"kind":      "boundary_arm",
		"target":    target,
		"close_at":  closeAt.Format(time.RFC3339),
		"reopen_at": reopenAt.Format(time.RFC3339),
	}

	switch target {
	case "platform":
		prev, err := s.platform.Schedule(ctx, s.pg)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		prevEnf := s.settings.GetBool(ctx, platform.EnforcedKey)
		if _, err := s.platform.SetSchedule(ctx, actor, ws); err != nil {
			s.respondErr(w, err)
			return
		}
		if err := s.settings.Set(ctx, platform.EnforcedKey, true, nil); err != nil {
			s.respondErr(w, err)
			return
		}
		qaBoundarySaved.platformSchedule = prev
		qaBoundarySaved.platformEnforced = prevEnf
		qaBoundarySaved.platformArmed = true
	case "zone":
		phone, _ := identity.NormalizePhone(qaStagingPhone)
		var lat, lng float64
		if err := s.pg.QueryRow(ctx, `
			SELECT ST_Y(a.location::geometry), ST_X(a.location::geometry)
			FROM addresses a JOIN users u ON u.id = a.user_id
			WHERE u.phone = $1 AND a.is_default
			LIMIT 1`, phone).Scan(&lat, &lng); err != nil {
			s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_default_address", "errors.conflict"))
			return
		}
		zc, err := s.orders.ZoneAt(ctx, s.pg, lat, lng)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		if zc.ID == "" {
			s.respondErr(w, httpx.NewError(http.StatusConflict, "qa_no_zone_for_address", "errors.conflict"))
			return
		}
		prev, err := s.platform.ZoneWindows(ctx, s.pg, zc.ID)
		if err != nil {
			s.respondErr(w, err)
			return
		}
		_, prevEnf, _ := s.platform.ZoneSchedule(ctx, s.pg, zc.ID)
		if _, err := s.platform.SetZoneSchedule(ctx, zc.ID, true, ws); err != nil {
			s.respondErr(w, err)
			return
		}
		qaBoundarySaved.zoneID = zc.ID
		qaBoundarySaved.zoneSchedule = prev
		qaBoundarySaved.zoneEnforced = prevEnf
		qaBoundarySaved.zoneArmed = true
		out["zone_id"] = zc.ID
	default:
		s.respondErr(w, errValidation)
		return
	}

	// **إشارةُ «تغيّر شيء»** — كما بابُ الأدمن، ليجدّد التطبيقُ فيُسلّح مؤقّتَه.
	s.hub.Publish(realtime.TopicCatalog, map[string]any{"type": "catalog"})
	s.logger.Warn("QA boundary armed (staging-only)", "target", target, "close_at", closeAt.Format(time.RFC3339))
	httpx.JSON(w, http.StatusOK, out)
}

// qaBoundaryRestore يُعيد الجدولَ السابقَ (المنصّةِ أو المنطقة) ويبثّ التغيّر.
func (s *Server) qaBoundaryRestore(w http.ResponseWriter, r *http.Request, target string) {
	if !s.qaStagingEnabled() {
		s.respondErr(w, httpx.ErrNotFound)
		return
	}
	ctx := r.Context()
	actor := s.qaCoverageActor(ctx)

	qaBoundarySaved.mu.Lock()
	defer qaBoundarySaved.mu.Unlock()

	switch target {
	case "platform":
		if qaBoundarySaved.platformArmed {
			if _, err := s.platform.SetSchedule(ctx, actor, qaBoundarySaved.platformSchedule); err != nil {
				s.respondErr(w, err)
				return
			}
			_ = s.settings.Set(ctx, platform.EnforcedKey, qaBoundarySaved.platformEnforced, nil)
			qaBoundarySaved.platformArmed = false
		}
	case "zone":
		if qaBoundarySaved.zoneArmed {
			if _, err := s.platform.SetZoneSchedule(ctx, qaBoundarySaved.zoneID, qaBoundarySaved.zoneEnforced, qaBoundarySaved.zoneSchedule); err != nil {
				s.respondErr(w, err)
				return
			}
			qaBoundarySaved.zoneArmed = false
		}
	default:
		s.respondErr(w, errValidation)
		return
	}

	s.hub.Publish(realtime.TopicCatalog, map[string]any{"type": "catalog"})
	s.logger.Warn("QA boundary restored (staging-only)", "target", target)
	httpx.JSON(w, http.StatusOK, map[string]any{"restored": target, "kind": "boundary_restore"})
}
